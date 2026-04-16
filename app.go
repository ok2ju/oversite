package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/ok2ju/oversite/internal/auth"
	"github.com/ok2ju/oversite/internal/database"
	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/migrations"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct holds application state and is bound to the frontend.
type App struct {
	ctx           context.Context
	db            *sql.DB
	queries       *store.Queries
	authService   *auth.AuthService
	importService *demo.ImportService
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
// Stub bindings — implemented in later phases
// ---------------------------------------------------------------------------

var errNotImplemented = errors.New("not implemented")

// GetFaceitProfile returns the current user's Faceit profile.
func (a *App) GetFaceitProfile() (*FaceitProfile, error) {
	return nil, errNotImplemented
}

// GetEloHistory returns elo history for the given number of days.
func (a *App) GetEloHistory(days int) ([]EloHistoryPoint, error) {
	return nil, errNotImplemented
}

// GetFaceitMatches returns a paginated, filtered list of Faceit matches.
func (a *App) GetFaceitMatches(page, perPage int, mapName, result string) (*FaceitMatchListResult, error) {
	return nil, errNotImplemented
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
		IsOvertime:  r.RoundNumber > 24,
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
