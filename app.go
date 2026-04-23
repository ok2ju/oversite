package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ok2ju/oversite/internal/auth"
	"github.com/ok2ju/oversite/internal/database"
	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/faceit"
	"github.com/ok2ju/oversite/internal/logging"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/migrations"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/natefinch/lumberjack.v2"
)

// App struct holds application state and is bound to the frontend.
type App struct {
	ctx             context.Context
	db              *sql.DB
	queries         *store.Queries
	authService     *auth.AuthService
	faceitClient    faceit.FaceitClient
	importService   *demo.ImportService
	syncService     *faceit.SyncService
	downloadService *faceit.DownloadService

	// logDir is the directory containing errors.txt / network.txt.
	// Set by main.go before Startup runs.
	logDir string
	// networkLog is non-nil only in dev builds; closed in Shutdown.
	networkLog *lumberjack.Logger
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{}
}

// Startup is called when the app starts. The context is saved
// so it can be used by other methods.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	dbPath, err := database.DefaultDBPath()
	if err != nil {
		log.Fatalf("failed to get database path: %v", err)
	}

	db, err := database.OpenWithMigrations(dbPath, migrations.FS)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	a.db = db
	a.queries = store.New(db)

	// Demo import service.
	a.importService = demo.NewImportService(a.queries, a.db)

	// Build the shared HTTP transport. In dev builds, HTTP traffic is dumped
	// to network.txt; in production builds no transport wrapping is applied.
	var httpTransport http.RoundTripper
	var httpClient *http.Client
	if wailsRuntime.Environment(ctx).BuildType == "dev" {
		netLog, nlErr := logging.OpenNetworkLog(a.logDir)
		if nlErr != nil {
			slog.Warn("failed to open network log", "err", nlErr)
		} else {
			a.networkLog = netLog
			httpTransport = logging.NewTransport(http.DefaultTransport, netLog)
			httpClient = &http.Client{Transport: httpTransport}
		}
	}

	// Auth service with OS keychain and real Faceit client.
	kr := &auth.RealKeyring{}
	tokens := auth.NewTokenStore(kr)
	faceitClient := auth.NewHTTPFaceitClient(os.Getenv("FACEIT_API_KEY"), httpTransport)

	oauthCfg := auth.OAuthConfig{
		ClientID:     os.Getenv("FACEIT_CLIENT_ID"),
		ClientSecret: os.Getenv("FACEIT_CLIENT_SECRET"),
		AuthURL:      os.Getenv("FACEIT_AUTH_URL"),
		TokenURL:     os.Getenv("FACEIT_TOKEN_URL"),
		RelayURL:     os.Getenv("FACEIT_RELAY_URL"),
		HTTPClient:   httpClient,
	}

	a.authService = auth.NewAuthService(
		oauthCfg,
		tokens,
		faceitClient,
		a.queries,
		func(url string) error {
			wailsRuntime.BrowserOpenURL(a.ctx, url)
			return nil
		},
	)

	a.faceitClient = faceitClient
	a.syncService = faceit.NewSyncService(faceitClient, a.queries)

	// Try to restore a previous session (refresh access token from keychain).
	// Non-fatal: if refresh fails the user simply needs to log in again.
	if err := a.authService.TryRestoreSession(ctx); err != nil {
		slog.Warn("session restore failed", "err", err)
	}

	downloadClient := httpClient
	if downloadClient == nil {
		downloadClient = &http.Client{}
	}
	downloadDir := filepath.Join(filepath.Dir(dbPath), "demos")
	a.downloadService = faceit.NewDownloadService(
		downloadClient,
		a.importService,
		a.queries,
		downloadDir,
	)
}

// Shutdown is called when the app is closing.
func (a *App) Shutdown(_ context.Context) {
	if a.db != nil {
		_ = a.db.Close()
	}
	if a.networkLog != nil {
		_ = a.networkLog.Close()
	}
	_ = logging.Close()
}

// Greet returns a greeting for the given name.
// This is a placeholder binding to verify frontend-to-Go communication.
func (a *App) Greet(name string) string {
	return "Hello " + name + ", welcome to Oversite!"
}

// ---------------------------------------------------------------------------
// Auth bindings
// ---------------------------------------------------------------------------

// GetCurrentUser returns the currently authenticated user.
func (a *App) GetCurrentUser() (*User, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, nil
	}
	return &User{
		UserID:   strconv.FormatInt(u.ID, 10),
		FaceitID: u.FaceitID,
		Nickname: u.Nickname,
	}, nil
}

// LoginWithFaceit initiates the Faceit OAuth login flow.
func (a *App) LoginWithFaceit() error {
	_, err := a.authService.Login(a.ctx)
	return err
}

// Logout clears the authentication state and keychain.
func (a *App) Logout() error {
	return a.authService.Logout()
}

// ---------------------------------------------------------------------------
// Demo bindings
// ---------------------------------------------------------------------------

// ListDemos returns a paginated list of imported demos.
func (a *App) ListDemos(page, perPage int) (*DemoListResult, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return &DemoListResult{
			Data: []Demo{},
			Meta: PaginationMeta{Total: 0, Page: page, PerPage: perPage},
		}, nil
	}

	offset := int64((page - 1) * perPage)
	demos, err := a.queries.ListDemosByUserID(a.ctx, store.ListDemosByUserIDParams{
		UserID:    u.ID,
		OffsetVal: offset,
		LimitVal:  int64(perPage),
	})
	if err != nil {
		return nil, fmt.Errorf("listing demos: %w", err)
	}

	total, err := a.queries.CountDemosByUserID(a.ctx, u.ID)
	if err != nil {
		return nil, fmt.Errorf("counting demos: %w", err)
	}

	data := make([]Demo, len(demos))
	for i, d := range demos {
		data[i] = storeDemoToBinding(d)
	}

	return &DemoListResult{
		Data: data,
		Meta: PaginationMeta{
			Total:   int(total),
			Page:    page,
			PerPage: perPage,
		},
	}, nil
}

// ImportDemoFile opens a native file dialog and imports the selected .dem file.
func (a *App) ImportDemoFile() error {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("not logged in")
	}

	filePath, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select CS2 Demo File",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "CS2 Demo Files (*.dem, *.dem.zst)", Pattern: "*.dem;*.zst"},
		},
	})
	if err != nil {
		return fmt.Errorf("file dialog: %w", err)
	}
	if filePath == "" {
		return nil // User cancelled.
	}

	d, err := a.importService.ImportFile(a.ctx, filePath, u.ID)
	if err != nil {
		return err
	}
	go a.parseDemo(d.ID, d.FilePath)
	return nil
}

// ImportDemoFolder opens a native directory dialog and imports all .dem files.
func (a *App) ImportDemoFolder() (*FolderImportResult, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("not logged in")
	}

	dirPath, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Demo Folder",
	})
	if err != nil {
		return nil, fmt.Errorf("directory dialog: %w", err)
	}
	if dirPath == "" {
		return nil, nil // User cancelled.
	}

	result, err := a.importService.ImportFolder(a.ctx, dirPath, u.ID, func(current, total int, fileName string) {
		wailsRuntime.EventsEmit(a.ctx, "demo:folder:progress", map[string]interface{}{
			"current":  current,
			"total":    total,
			"fileName": fileName,
		})
	})
	if err != nil {
		return nil, err
	}

	imported := make([]Demo, len(result.Imported))
	for i, d := range result.Imported {
		imported[i] = storeDemoToBinding(*d)
		go a.parseDemo(d.ID, d.FilePath)
	}

	importErrors := make([]string, len(result.Errors))
	for i, e := range result.Errors {
		importErrors[i] = e.Error()
	}

	return &FolderImportResult{
		Imported: imported,
		Errors:   importErrors,
	}, nil
}

// ImportDemoByPath imports a .dem file at the given path (used for drag-and-drop).
func (a *App) ImportDemoByPath(filePath string) error {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("not logged in")
	}

	d, err := a.importService.ImportFile(a.ctx, filePath, u.ID)
	if err != nil {
		return err
	}
	go a.parseDemo(d.ID, d.FilePath)
	return nil
}

// DeleteDemo removes a demo by ID.
func (a *App) DeleteDemo(id int64) error {
	return a.queries.DeleteDemo(a.ctx, id)
}

// parseDemo runs the full parse-and-ingest pipeline for a demo in the background.
// It transitions the demo through imported → parsing → ready (or failed).
func (a *App) parseDemo(demoID int64, filePath string) {
	fileName := filepath.Base(filePath)

	emitProgress := func(stage string, percent float64, errMsg ...string) {
		payload := map[string]interface{}{
			"demoId":   demoID,
			"fileName": fileName,
			"percent":  percent,
			"stage":    stage,
		}
		if len(errMsg) > 0 && errMsg[0] != "" {
			payload["error"] = errMsg[0]
		}
		wailsRuntime.EventsEmit(a.ctx, "demo:parse:progress", payload)
	}

	// Mark as parsing.
	if _, err := a.queries.UpdateDemoStatus(a.ctx, store.UpdateDemoStatusParams{
		Status: "parsing",
		ID:     demoID,
	}); err != nil {
		slog.Error("parseDemo: update status to parsing", "demo_id", demoID, "err", err)
		return
	}

	emitProgress("parsing", 0)

	f, err := os.Open(filePath)
	if err != nil {
		slog.Error("parseDemo: open file", "demo_id", demoID, "err", err)
		a.failDemo(demoID, fmt.Sprintf("open file: %v", err), emitProgress)
		return
	}
	defer f.Close() //nolint:errcheck

	parser := demo.NewDemoParser(demo.WithProgressFunc(func(stage string, percent float64) {
		emitProgress(stage, percent)
	}))

	result, err := parser.Parse(f)
	if err != nil {
		slog.Error("parseDemo: parse", "demo_id", demoID, "err", err)
		a.failDemo(demoID, fmt.Sprintf("parse failed: %v", err), emitProgress)
		return
	}

	// Ingest rounds + player stats.
	roundMap, err := demo.IngestRounds(a.ctx, a.db, demoID, result)
	if err != nil {
		slog.Error("parseDemo: ingest rounds", "demo_id", demoID, "err", err)
		a.failDemo(demoID, fmt.Sprintf("ingest rounds: %v", err), emitProgress)
		return
	}

	// Ingest game events.
	if _, err := demo.IngestGameEvents(a.ctx, a.db, demoID, result.Events, roundMap); err != nil {
		slog.Error("parseDemo: ingest events", "demo_id", demoID, "err", err)
		a.failDemo(demoID, fmt.Sprintf("ingest events: %v", err), emitProgress)
		return
	}

	// Ingest tick data.
	ingester := demo.NewTickIngester(a.db, 0)
	if _, err := ingester.Ingest(a.ctx, demoID, result.Ticks); err != nil {
		slog.Error("parseDemo: ingest ticks", "demo_id", demoID, "err", err)
		a.failDemo(demoID, fmt.Sprintf("ingest ticks: %v", err), emitProgress)
		return
	}

	// Mark demo as ready with parsed metadata.
	if _, err := a.queries.UpdateDemoAfterParse(a.ctx, store.UpdateDemoAfterParseParams{
		MapName:      result.Header.MapName,
		TotalTicks:   int64(result.Header.TotalTicks),
		TickRate:     result.Header.TickRate,
		DurationSecs: int64(result.Header.DurationSecs),
		ID:           demoID,
	}); err != nil {
		slog.Error("parseDemo: update after parse", "demo_id", demoID, "err", err)
		a.failDemo(demoID, fmt.Sprintf("save metadata: %v", err), emitProgress)
		return
	}

	emitProgress("complete", 100)
}

// failDemo marks a demo as failed and emits an error progress event with
// the error message so the frontend can display what went wrong.
func (a *App) failDemo(demoID int64, errMsg string, emitProgress func(string, float64, ...string)) {
	_, _ = a.queries.UpdateDemoStatus(a.ctx, store.UpdateDemoStatusParams{
		Status: "failed",
		ID:     demoID,
	})
	emitProgress("error", 0, errMsg)
}

// ---------------------------------------------------------------------------
// Viewer bindings
// ---------------------------------------------------------------------------

// GetDemoByID returns a single demo by its ID.
func (a *App) GetDemoByID(id string) (*Demo, error) {
	demoID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}

	d, err := a.queries.GetDemoByID(a.ctx, demoID)
	if err != nil {
		return nil, fmt.Errorf("getting demo: %w", err)
	}

	result := storeDemoToBinding(d)
	return &result, nil
}

// GetDemoRounds returns all rounds for a demo.
func (a *App) GetDemoRounds(demoID string) ([]Round, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}

	rounds, err := a.queries.GetRoundsByDemoID(a.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting rounds: %w", err)
	}

	result := make([]Round, len(rounds))
	for i, r := range rounds {
		result[i] = storeRoundToBinding(r)
	}
	return result, nil
}

// GetDemoEvents returns all game events for a demo.
func (a *App) GetDemoEvents(demoID string) ([]GameEvent, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}

	events, err := a.queries.GetGameEventsByDemoID(a.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting events: %w", err)
	}

	result := make([]GameEvent, len(events))
	for i, e := range events {
		result[i] = storeGameEventToBinding(e)
	}
	return result, nil
}

// GetDemoTicks returns player tick data for a range of ticks within a demo.
func (a *App) GetDemoTicks(demoID string, startTick, endTick int) ([]TickData, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}

	ticks, err := a.queries.GetTickDataByRange(a.ctx, store.GetTickDataByRangeParams{
		DemoID:    id,
		StartTick: int64(startTick),
		EndTick:   int64(endTick),
	})
	if err != nil {
		return nil, fmt.Errorf("getting ticks: %w", err)
	}

	result := make([]TickData, len(ticks))
	for i, d := range ticks {
		result[i] = storeTickDatumToBinding(d)
	}
	return result, nil
}

// GetRoundRoster returns the player roster for a specific round.
func (a *App) GetRoundRoster(demoID string, roundNumber int) ([]PlayerRosterEntry, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}

	round, err := a.queries.GetRoundByDemoAndNumber(a.ctx, store.GetRoundByDemoAndNumberParams{
		DemoID:      id,
		RoundNumber: int64(roundNumber),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return []PlayerRosterEntry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting round: %w", err)
	}

	players, err := a.queries.GetPlayerRoundsByRoundID(a.ctx, round.ID)
	if err != nil {
		return nil, fmt.Errorf("getting roster: %w", err)
	}

	result := make([]PlayerRosterEntry, len(players))
	for i, p := range players {
		result[i] = PlayerRosterEntry{
			SteamID:    p.SteamID,
			PlayerName: p.PlayerName,
			TeamSide:   p.TeamSide,
		}
	}
	return result, nil
}

// GetScoreboard returns aggregated player stats for a demo.
func (a *App) GetScoreboard(demoID string) ([]ScoreboardEntry, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}

	rows, err := a.queries.GetPlayerStatsByDemoID(a.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting scoreboard: %w", err)
	}

	result := make([]ScoreboardEntry, len(rows))
	for i, r := range rows {
		kills := int(r.TotalKills.Float64)
		deaths := int(r.TotalDeaths.Float64)
		assists := int(r.TotalAssists.Float64)
		damage := int(r.TotalDamage.Float64)
		hsKills := int(r.TotalHeadshotKills.Float64)
		roundsPlayed := int(r.RoundsPlayed)

		var hsPercent float64
		if kills > 0 {
			hsPercent = float64(hsKills) / float64(kills) * 100
		}
		var adr float64
		if roundsPlayed > 0 {
			adr = float64(damage) / float64(roundsPlayed)
		}
		result[i] = ScoreboardEntry{
			SteamID:      r.SteamID,
			PlayerName:   r.PlayerName,
			TeamSide:     r.TeamSide,
			Kills:        kills,
			Deaths:       deaths,
			Assists:      assists,
			Damage:       damage,
			HSKills:      hsKills,
			RoundsPlayed: roundsPlayed,
			HSPercent:    hsPercent,
			ADR:          adr,
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Faceit bindings
// ---------------------------------------------------------------------------

// GetFaceitProfile returns the current user's Faceit profile.
func (a *App) GetFaceitProfile() (*FaceitProfile, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("not logged in")
	}

	// Fetch live match count from Faceit API; fall back to local DB count.
	token := a.authService.GetAccessToken()
	ctx := auth.WithAccessToken(a.ctx, token)

	var matchesPlayed int
	stats, statsErr := a.faceitClient.GetPlayerLifetimeStats(ctx, u.FaceitID)
	if statsErr == nil && stats != nil && stats.Matches > 0 {
		matchesPlayed = stats.Matches
	} else {
		if statsErr != nil {
			slog.Warn("faceit lifetime stats unavailable, using local count", "err", statsErr)
		}
		localCount, err := a.queries.CountFaceitMatchesByUserID(a.ctx, u.ID)
		if err != nil {
			return nil, fmt.Errorf("counting matches: %w", err)
		}
		matchesPlayed = int(localCount)
	}

	results, err := a.queries.GetCurrentStreak(a.ctx, u.ID)
	if err != nil {
		return nil, fmt.Errorf("getting streak: %w", err)
	}

	profile := &FaceitProfile{
		Nickname:      u.Nickname,
		MatchesPlayed: matchesPlayed,
		CurrentStreak: computeStreak(results),
	}
	if u.AvatarUrl != "" {
		profile.AvatarURL = &u.AvatarUrl
	}
	if u.FaceitElo != 0 {
		elo := int(u.FaceitElo)
		profile.Elo = &elo
	}
	if u.FaceitLevel != 0 {
		level := int(u.FaceitLevel)
		profile.Level = &level
	}
	if u.Country != "" {
		profile.Country = &u.Country
	}
	return profile, nil
}

// GetFaceitMatches returns a paginated, filtered list of Faceit matches from
// the last 60 days.
func (a *App) GetFaceitMatches(page, perPage int, mapName, result string) (*FaceitMatchListResult, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("not logged in")
	}

	offset := int64((page - 1) * perPage)
	since := time.Now().AddDate(0, 0, -60).Format(time.RFC3339)

	var mapFilter, resultFilter interface{}
	if mapName != "" {
		mapFilter = mapName
	}
	if result != "" {
		resultFilter = result
	}

	matches, err := a.queries.GetFaceitMatchesFiltered(a.ctx, store.GetFaceitMatchesFilteredParams{
		UserID:    u.ID,
		SinceDate: since,
		MapName:   mapFilter,
		Result:    resultFilter,
		OffsetVal: offset,
		LimitVal:  int64(perPage),
	})
	if err != nil {
		return nil, fmt.Errorf("listing faceit matches: %w", err)
	}

	total, err := a.queries.CountFaceitMatchesFiltered(a.ctx, store.CountFaceitMatchesFilteredParams{
		UserID:    u.ID,
		SinceDate: since,
		MapName:   mapFilter,
		Result:    resultFilter,
	})
	if err != nil {
		return nil, fmt.Errorf("counting faceit matches: %w", err)
	}

	data := make([]FaceitMatch, len(matches))
	for i, m := range matches {
		data[i] = storeFaceitMatchToBinding(m)
	}

	return &FaceitMatchListResult{
		Data: data,
		Meta: PaginationMeta{
			Total:   int(total),
			Page:    page,
			PerPage: perPage,
		},
	}, nil
}

// SyncFaceitMatches fetches match history from Faceit and stores new matches.
// Returns the number of newly inserted matches. Progress events are emitted
// via the Wails runtime under "faceit:sync:progress".
func (a *App) SyncFaceitMatches() (int, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return 0, err
	}
	if u == nil {
		return 0, errors.New("not logged in")
	}

	token := a.authService.GetAccessToken()
	ctx := auth.WithAccessToken(a.ctx, token)

	inserted, err := a.syncService.SyncMatches(ctx, u.ID, u.FaceitID, func(current, total int) {
		wailsRuntime.EventsEmit(a.ctx, "faceit:sync:progress", map[string]interface{}{
			"current": current,
			"total":   total,
		})
	})
	if err != nil {
		return inserted, fmt.Errorf("syncing matches: %w", err)
	}
	return inserted, nil
}

// ImportMatchDemo downloads and imports a demo from a Faceit match.
// Progress events are emitted via "faceit:demo:download:progress".
func (a *App) ImportMatchDemo(faceitMatchID string) error {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return err
	}
	if u == nil {
		return errors.New("not logged in")
	}

	id, err := strconv.ParseInt(faceitMatchID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid match id: %w", err)
	}

	_, err = a.downloadService.DownloadAndImport(a.ctx, id, u.ID, func(bytesDownloaded, totalBytes int64) {
		wailsRuntime.EventsEmit(a.ctx, "faceit:demo:download:progress", map[string]interface{}{
			"bytesDownloaded": bytesDownloaded,
			"totalBytes":      totalBytes,
		})
	})
	if err != nil {
		slog.Error("ImportMatchDemo failed", "faceit_match_id", faceitMatchID, "err", err)
	}
	return err
}

// ---------------------------------------------------------------------------
// Heatmap bindings
// ---------------------------------------------------------------------------

// GetHeatmapData returns aggregated kill positions for the given demos and filters.
func (a *App) GetHeatmapData(demoIDs []int64, weapons []string, playerSteamID string, side string) ([]HeatmapPoint, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("not logged in")
	}

	demoIDsJSON, err := json.Marshal(demoIDs)
	if err != nil {
		return nil, fmt.Errorf("marshaling demo ids: %w", err)
	}

	// Verify user owns all requested demos.
	demos, err := a.queries.GetDemosByIDs(a.ctx, string(demoIDsJSON))
	if err != nil {
		return nil, fmt.Errorf("getting demos: %w", err)
	}
	for _, d := range demos {
		if d.UserID != u.ID {
			return nil, errors.New("unauthorized: demo does not belong to user")
		}
	}

	weaponsJSON, err := json.Marshal(weapons)
	if err != nil {
		return nil, fmt.Errorf("marshaling weapons: %w", err)
	}

	params := store.GetHeatmapAggregationParams{
		DemoIDs: string(demoIDsJSON),
		Weapons: string(weaponsJSON),
	}
	if playerSteamID != "" {
		params.PlayerSteamID = &playerSteamID
	}
	if side != "" {
		params.Side = &side
	}

	rows, err := a.queries.GetHeatmapAggregation(a.ctx, params)
	if err != nil {
		return nil, fmt.Errorf("getting heatmap data: %w", err)
	}

	result := make([]HeatmapPoint, len(rows))
	for i, r := range rows {
		result[i] = HeatmapPoint{
			X:         r.X,
			Y:         r.Y,
			KillCount: int(r.KillCount),
		}
	}
	return result, nil
}

// GetUniqueWeapons returns distinct weapons from kill events for the given demos.
func (a *App) GetUniqueWeapons(demoIDs []int64) ([]string, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("not logged in")
	}

	demoIDsJSON, err := json.Marshal(demoIDs)
	if err != nil {
		return nil, fmt.Errorf("marshaling demo ids: %w", err)
	}

	// Verify user owns all requested demos.
	demos, err := a.queries.GetDemosByIDs(a.ctx, string(demoIDsJSON))
	if err != nil {
		return nil, fmt.Errorf("getting demos: %w", err)
	}
	for _, d := range demos {
		if d.UserID != u.ID {
			return nil, errors.New("unauthorized: demo does not belong to user")
		}
	}

	weapons, err := a.queries.GetDistinctWeapons(a.ctx, string(demoIDsJSON))
	if err != nil {
		return nil, fmt.Errorf("getting weapons: %w", err)
	}
	if weapons == nil {
		return []string{}, nil
	}
	return weapons, nil
}

// GetUniquePlayers returns distinct players from kill events for the given demos.
func (a *App) GetUniquePlayers(demoIDs []int64) ([]PlayerInfo, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("not logged in")
	}

	demoIDsJSON, err := json.Marshal(demoIDs)
	if err != nil {
		return nil, fmt.Errorf("marshaling demo ids: %w", err)
	}

	// Verify user owns all requested demos.
	demos, err := a.queries.GetDemosByIDs(a.ctx, string(demoIDsJSON))
	if err != nil {
		return nil, fmt.Errorf("getting demos: %w", err)
	}
	for _, d := range demos {
		if d.UserID != u.ID {
			return nil, errors.New("unauthorized: demo does not belong to user")
		}
	}

	rows, err := a.queries.GetDistinctPlayers(a.ctx, string(demoIDsJSON))
	if err != nil {
		return nil, fmt.Errorf("getting players: %w", err)
	}

	result := make([]PlayerInfo, len(rows))
	for i, r := range rows {
		result[i] = PlayerInfo{
			SteamID:    r.SteamID,
			PlayerName: r.PlayerName,
		}
	}
	return result, nil
}

// GetWeaponStats returns weapon kill stats for a demo.
func (a *App) GetWeaponStats(demoID string) ([]WeaponStat, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("not logged in")
	}

	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}

	// Verify user owns the requested demo.
	demoIDsJSON, err := json.Marshal([]int64{id})
	if err != nil {
		return nil, fmt.Errorf("marshaling demo ids: %w", err)
	}
	demos, err := a.queries.GetDemosByIDs(a.ctx, string(demoIDsJSON))
	if err != nil {
		return nil, fmt.Errorf("getting demos: %w", err)
	}
	for _, d := range demos {
		if d.UserID != u.ID {
			return nil, errors.New("unauthorized: demo does not belong to user")
		}
	}

	rows, err := a.queries.GetWeaponStatsByDemoID(a.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting weapon stats: %w", err)
	}

	result := make([]WeaponStat, len(rows))
	for i, r := range rows {
		result[i] = WeaponStat{
			Weapon:    r.Weapon,
			KillCount: int(r.KillCount),
			HSCount:   int(r.HSCount),
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func storeDemoToBinding(d store.Demo) Demo {
	return Demo{
		ID:           d.ID,
		MapName:      d.MapName,
		FilePath:     d.FilePath,
		FileSize:     d.FileSize,
		Status:       d.Status,
		TotalTicks:   int(d.TotalTicks),
		TickRate:     int(d.TickRate),
		DurationSecs: int(d.DurationSecs),
		MatchDate:    d.MatchDate,
		CreatedAt:    d.CreatedAt,
	}
}

func storeRoundToBinding(r store.Round) Round {
	return Round{
		ID:            strconv.FormatInt(r.ID, 10),
		RoundNumber:   int(r.RoundNumber),
		StartTick:     int(r.StartTick),
		FreezeEndTick: int(r.FreezeEndTick),
		EndTick:       int(r.EndTick),
		WinnerSide:    r.WinnerSide,
		WinReason:     r.WinReason,
		CTScore:       int(r.CtScore),
		TScore:        int(r.TScore),
		IsOvertime:    r.IsOvertime != 0,
	}
}

func storeGameEventToBinding(e store.GameEvent) GameEvent {
	x, y, z := e.X, e.Y, e.Z
	ge := GameEvent{
		ID:        strconv.FormatInt(e.ID, 10),
		DemoID:    strconv.FormatInt(e.DemoID, 10),
		Tick:      int(e.Tick),
		EventType: e.EventType,
		X:         &x,
		Y:         &y,
		Z:         &z,
	}
	if e.RoundID != 0 {
		rid := strconv.FormatInt(e.RoundID, 10)
		ge.RoundID = &rid
	}
	if e.AttackerSteamID.Valid {
		ge.AttackerSteamID = &e.AttackerSteamID.String
	}
	if e.VictimSteamID.Valid {
		ge.VictimSteamID = &e.VictimSteamID.String
	}
	if e.Weapon.Valid {
		ge.Weapon = &e.Weapon.String
	}
	if e.ExtraData != "" {
		var m map[string]any
		if err := json.Unmarshal([]byte(e.ExtraData), &m); err == nil {
			ge.ExtraData = m
		}
	}
	return ge
}

func storeFaceitMatchToBinding(m store.FaceitMatch) FaceitMatch {
	fm := FaceitMatch{
		ID:            strconv.FormatInt(m.ID, 10),
		FaceitMatchID: m.FaceitMatchID,
		MapName:       m.MapName,
		ScoreTeam:     int(m.ScoreTeam),
		ScoreOpponent: int(m.ScoreOpponent),
		Result:        m.Result,
		PlayedAt:      m.PlayedAt,
		HasDemo:       m.DemoID.Valid,
	}
	if m.Kills != 0 {
		v := int(m.Kills)
		fm.Kills = &v
	}
	if m.Deaths != 0 {
		v := int(m.Deaths)
		fm.Deaths = &v
	}
	if m.Assists != 0 {
		v := int(m.Assists)
		fm.Assists = &v
	}
	if m.Adr != 0 {
		v := m.Adr
		fm.ADR = &v
	}
	if m.DemoUrl != "" {
		fm.DemoURL = &m.DemoUrl
	}
	if m.DemoID.Valid {
		v := strconv.FormatInt(m.DemoID.Int64, 10)
		fm.DemoID = &v
	}
	return fm
}

func computeStreak(results []string) CurrentStreak {
	if len(results) == 0 {
		return CurrentStreak{Type: "none", Count: 0}
	}
	streakType := results[0]
	count := 1
	for i := 1; i < len(results); i++ {
		if results[i] != streakType {
			break
		}
		count++
	}
	// Map DB values to frontend-expected types.
	typeLabel := "draw"
	switch streakType {
	case "W":
		typeLabel = "win"
	case "L":
		typeLabel = "loss"
	}
	return CurrentStreak{Type: typeLabel, Count: count}
}

func storeTickDatumToBinding(d store.TickDatum) TickData {
	td := TickData{
		Tick:    int(d.Tick),
		SteamID: d.SteamID,
		X:       d.X,
		Y:       d.Y,
		Z:       d.Z,
		Yaw:     d.Yaw,
		Health:  int(d.Health),
		Armor:   int(d.Armor),
		IsAlive: d.IsAlive != 0,
	}
	if d.Weapon != "" {
		td.Weapon = &d.Weapon
	}
	return td
}
