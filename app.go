package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ok2ju/oversite/internal/auth"
	"github.com/ok2ju/oversite/internal/database"
	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/faceit"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/migrations"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct holds application state and is bound to the frontend.
type App struct {
	ctx             context.Context
	db              *sql.DB
	queries         *store.Queries
	authService     *auth.AuthService
	importService   *demo.ImportService
	syncService     *faceit.SyncService
	downloadService *faceit.DownloadService
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

	// Auth service with OS keychain and real Faceit client.
	kr := &auth.RealKeyring{}
	tokens := auth.NewTokenStore(kr)
	faceitClient := auth.NewHTTPFaceitClient()

	oauthCfg := auth.OAuthConfig{
		ClientID:     os.Getenv("FACEIT_CLIENT_ID"),
		ClientSecret: os.Getenv("FACEIT_CLIENT_SECRET"),
		AuthURL:      os.Getenv("FACEIT_AUTH_URL"),
		TokenURL:     os.Getenv("FACEIT_TOKEN_URL"),
		RelayURL:     os.Getenv("FACEIT_RELAY_URL"),
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

	a.syncService = faceit.NewSyncService(faceitClient, a.queries)

	downloadDir := filepath.Join(filepath.Dir(dbPath), "demos")
	a.downloadService = faceit.NewDownloadService(
		&http.Client{},
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
			{DisplayName: "CS2 Demo Files (*.dem)", Pattern: "*.dem"},
		},
	})
	if err != nil {
		return fmt.Errorf("file dialog: %w", err)
	}
	if filePath == "" {
		return nil // User cancelled.
	}

	_, err = a.importService.ImportFile(a.ctx, filePath, u.ID)
	return err
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

	_, err = a.importService.ImportFile(a.ctx, filePath, u.ID)
	return err
}

// DeleteDemo removes a demo by ID.
func (a *App) DeleteDemo(id int64) error {
	return a.queries.DeleteDemo(a.ctx, id)
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

	matchesPlayed, err := a.queries.CountFaceitMatchesByUserID(a.ctx, u.ID)
	if err != nil {
		return nil, fmt.Errorf("counting matches: %w", err)
	}

	results, err := a.queries.GetCurrentStreak(a.ctx, u.ID)
	if err != nil {
		return nil, fmt.Errorf("getting streak: %w", err)
	}

	profile := &FaceitProfile{
		Nickname:      u.Nickname,
		MatchesPlayed: int(matchesPlayed),
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

// GetEloHistory returns elo history for the given number of days.
func (a *App) GetEloHistory(days int) ([]EloHistoryPoint, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("not logged in")
	}

	var since string
	if days > 0 {
		since = time.Now().AddDate(0, 0, -days).Format(time.RFC3339)
	} else {
		since = "0001-01-01T00:00:00Z"
	}

	rows, err := a.queries.GetEloHistory(a.ctx, store.GetEloHistoryParams{
		UserID: u.ID,
		Since:  since,
	})
	if err != nil {
		return nil, fmt.Errorf("getting elo history: %w", err)
	}

	result := make([]EloHistoryPoint, len(rows))
	for i, r := range rows {
		elo := int(r.EloAfter)
		result[i] = EloHistoryPoint{
			Elo:      &elo,
			MapName:  r.MapName,
			PlayedAt: r.PlayedAt,
		}
	}
	return result, nil
}

// GetFaceitMatches returns a paginated, filtered list of Faceit matches.
func (a *App) GetFaceitMatches(page, perPage int, mapName, result string) (*FaceitMatchListResult, error) {
	u, err := a.authService.GetCurrentUser(a.ctx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("not logged in")
	}

	offset := int64((page - 1) * perPage)

	var mapFilter, resultFilter interface{}
	if mapName != "" {
		mapFilter = mapName
	}
	if result != "" {
		resultFilter = result
	}

	matches, err := a.queries.GetFaceitMatchesFiltered(a.ctx, store.GetFaceitMatchesFilteredParams{
		UserID:    u.ID,
		MapName:   mapFilter,
		Result:    resultFilter,
		OffsetVal: offset,
		LimitVal:  int64(perPage),
	})
	if err != nil {
		return nil, fmt.Errorf("listing faceit matches: %w", err)
	}

	total, err := a.queries.CountFaceitMatchesFiltered(a.ctx, store.CountFaceitMatchesFilteredParams{
		UserID:  u.ID,
		MapName: mapFilter,
		Result:  resultFilter,
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
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
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
		ID:          strconv.FormatInt(r.ID, 10),
		RoundNumber: int(r.RoundNumber),
		StartTick:   int(r.StartTick),
		EndTick:     int(r.EndTick),
		WinnerSide:  r.WinnerSide,
		WinReason:   r.WinReason,
		CTScore:     int(r.CtScore),
		TScore:      int(r.TScore),
		IsOvertime:  r.IsOvertime != 0,
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
	if m.EloBefore != 0 {
		v := int(m.EloBefore)
		fm.EloBefore = &v
	}
	if m.EloAfter != 0 {
		v := int(m.EloAfter)
		fm.EloAfter = &v
	}
	if m.EloBefore != 0 && m.EloAfter != 0 {
		v := int(m.EloAfter - m.EloBefore)
		fm.EloChange = &v
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
	return CurrentStreak{Type: streakType, Count: count}
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
