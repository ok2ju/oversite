package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ok2ju/oversite/internal/database"
	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/logging"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/migrations"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct holds application state and is bound to the frontend.
type App struct {
	ctx           context.Context
	db            *sql.DB
	queries       *store.Queries
	importService *demo.ImportService
	demosDir      string
}

// NewApp opens the database, runs migrations, and returns an App ready to be
// bound to Wails. DB init must happen here (not in Startup) because Wails on
// macOS launches OnStartup in a goroutine, which races with binding calls from
// the WebView. See internal/frontend/desktop/darwin/frontend.go in wails v2.
func NewApp() (*App, error) {
	dbPath, err := database.DefaultDBPath()
	if err != nil {
		return nil, fmt.Errorf("resolve db path: %w", err)
	}

	demosDir, err := database.DemosDir()
	if err != nil {
		return nil, fmt.Errorf("resolve demos dir: %w", err)
	}

	db, err := database.OpenWithMigrations(dbPath, migrations.FS)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	queries := store.New(db)
	return &App{
		ctx:           context.Background(),
		db:            db,
		queries:       queries,
		importService: demo.NewImportService(queries, db, demosDir),
		demosDir:      demosDir,
	}, nil
}

// Startup is called by Wails after the window is created. It replaces the
// background context with the Wails-aware one needed by runtime.EventsEmit.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// Shutdown is called when the app is closing.
func (a *App) Shutdown(_ context.Context) {
	if a.db != nil {
		_ = a.db.Close()
	}
	_ = logging.Close()
}

// Greet returns a greeting for the given name.
func (a *App) Greet(name string) string {
	return "Hello " + name + ", welcome to Oversite!"
}

// ---------------------------------------------------------------------------
// Diagnostics bindings
// ---------------------------------------------------------------------------

// LogsDir returns the absolute path to the directory holding errors.txt.
// Used by the Settings UI so users can locate logs without knowing the
// platform-specific app data path.
func (a *App) LogsDir() string {
	return logging.Dir()
}

// OpenLogsFolder opens the logs directory in the OS file manager.
func (a *App) OpenLogsFolder() error {
	dir := logging.Dir()
	if dir == "" {
		return fmt.Errorf("logs directory not initialized")
	}
	return logging.Reveal(dir)
}

// ---------------------------------------------------------------------------
// Demo bindings
// ---------------------------------------------------------------------------

// ListDemos returns a paginated list of imported demos.
func (a *App) ListDemos(page, perPage int) (*DemoListResult, error) {
	offset := int64((page - 1) * perPage)
	demos, err := a.queries.ListDemos(a.ctx, store.ListDemosParams{
		OffsetVal: offset,
		LimitVal:  int64(perPage),
	})
	if err != nil {
		return nil, fmt.Errorf("listing demos: %w", err)
	}

	total, err := a.queries.CountDemos(a.ctx)
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

	d, err := a.importService.ImportFile(a.ctx, filePath)
	if err != nil {
		return err
	}
	go a.parseDemo(d.ID, d.FilePath)
	return nil
}

// ImportDemoByPath imports a .dem file at the given path (used for drag-and-drop).
func (a *App) ImportDemoByPath(filePath string) error {
	d, err := a.importService.ImportFile(a.ctx, filePath)
	if err != nil {
		return err
	}
	go a.parseDemo(d.ID, d.FilePath)
	return nil
}

// DeleteDemo removes a demo by ID and also removes its file copy from the
// app-managed demos folder. Files outside that folder (legacy imports) are
// left untouched.
func (a *App) DeleteDemo(id int64) error {
	if d, err := a.queries.GetDemoByID(a.ctx, id); err == nil {
		if a.demosDir != "" && isWithinDir(a.demosDir, d.FilePath) {
			_ = os.Remove(d.FilePath)
		}
	}
	return a.queries.DeleteDemo(a.ctx, id)
}

// isWithinDir reports whether path is contained inside dir.
func isWithinDir(dir, path string) bool {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// parseDemo runs the full parse-and-ingest pipeline for a demo in the background.
// It transitions the demo through imported → parsing → ready (or failed).
func (a *App) parseDemo(demoID int64, filePath string) {
	fileName := filepath.Base(filePath)

	var fileSize int64 = -1
	if info, err := os.Stat(filePath); err == nil {
		fileSize = info.Size()
	}
	logCtx := []any{
		"demo_id", demoID,
		"file_path", filePath,
		"file_size", fileSize,
	}

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

	if _, err := a.queries.UpdateDemoStatus(a.ctx, store.UpdateDemoStatusParams{
		Status: "parsing",
		ID:     demoID,
	}); err != nil {
		slog.Error("parseDemo: update status to parsing", append(logCtx, "err", err)...)
		return
	}

	// Always-on breadcrumb so errors.txt records which file kicked off a parse,
	// even if the failure later in the pipeline doesn't reach the user.
	slog.Warn("parseDemo: start", logCtx...)

	emitProgress("parsing", 0)

	f, err := os.Open(filePath)
	if err != nil {
		slog.Error("parseDemo: open file", append(logCtx, "err", err)...)
		a.failDemo(demoID, fmt.Sprintf("open file: %v", err), emitProgress)
		return
	}
	defer f.Close() //nolint:errcheck

	parser := demo.NewDemoParser(demo.WithProgressFunc(func(stage string, percent float64) {
		emitProgress(stage, percent)
	}))

	result, err := parser.Parse(f)
	if err != nil {
		slog.Error("parseDemo: parse", append(logCtx, "err", err)...)
		a.failDemo(demoID, fmt.Sprintf("parse failed: %v", err), emitProgress)
		return
	}

	roundMap, err := demo.IngestRounds(a.ctx, a.db, demoID, result)
	if err != nil {
		slog.Error("parseDemo: ingest rounds", append(logCtx, "err", err)...)
		a.failDemo(demoID, fmt.Sprintf("ingest rounds: %v", err), emitProgress)
		return
	}

	if _, err := demo.IngestGameEvents(a.ctx, a.db, demoID, result.Events, roundMap); err != nil {
		slog.Error("parseDemo: ingest events", append(logCtx, "err", err)...)
		a.failDemo(demoID, fmt.Sprintf("ingest events: %v", err), emitProgress)
		return
	}

	ingester := demo.NewTickIngester(a.db, 0)
	if _, err := ingester.Ingest(a.ctx, demoID, result.Ticks); err != nil {
		slog.Error("parseDemo: ingest ticks", append(logCtx, "err", err)...)
		a.failDemo(demoID, fmt.Sprintf("ingest ticks: %v", err), emitProgress)
		return
	}

	if _, err := a.queries.UpdateDemoAfterParse(a.ctx, store.UpdateDemoAfterParseParams{
		MapName:      result.Header.MapName,
		TotalTicks:   int64(result.Header.TotalTicks),
		TickRate:     result.Header.TickRate,
		DurationSecs: int64(result.Header.DurationSecs),
		ID:           demoID,
	}); err != nil {
		slog.Error("parseDemo: update after parse", append(logCtx, "err", err)...)
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
// Heatmap bindings
// ---------------------------------------------------------------------------

// GetHeatmapData returns aggregated kill positions for the given demos and filters.
func (a *App) GetHeatmapData(demoIDs []int64, weapons []string, playerSteamID string, side string) ([]HeatmapPoint, error) {
	demoIDsJSON, err := json.Marshal(demoIDs)
	if err != nil {
		return nil, fmt.Errorf("marshaling demo ids: %w", err)
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
		CTTeamName:    r.CtTeamName,
		TTeamName:     r.TTeamName,
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
		Tick:       int(d.Tick),
		SteamID:    d.SteamID,
		X:          d.X,
		Y:          d.Y,
		Z:          d.Z,
		Yaw:        d.Yaw,
		Health:     int(d.Health),
		Armor:      int(d.Armor),
		IsAlive:    d.IsAlive != 0,
		Money:      int(d.Money),
		HasHelmet:  d.HasHelmet != 0,
		HasDefuser: d.HasDefuser != 0,
		Inventory:  splitInventory(d.Inventory),
	}
	if d.Weapon != "" {
		td.Weapon = &d.Weapon
	}
	return td
}

// splitInventory parses the comma-separated inventory string written by the
// parser. Returns an empty slice when the column is empty so the JSON encoder
// emits `[]` rather than `null` and the frontend can iterate without nil checks.
func splitInventory(s string) []string {
	if s == "" {
		return []string{}
	}
	out := make([]string, 0, 8)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			if i > start {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
