package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ok2ju/oversite/internal/database"
	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/logging"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/migrations"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sync/errgroup"
)

// App struct holds application state and is bound to the frontend.
type App struct {
	ctx           context.Context
	cancel        context.CancelFunc
	db            *sql.DB
	queries       *store.Queries
	importService *demo.ImportService
	demosDir      string
	// parseMu serializes parseDemo runs. With MaxOpenConns(4) the DB can serve
	// reads during a write tx, but each parser holds hundreds of MB of state —
	// importing N files at once would N× peak memory. One parse at a time keeps
	// RAM bounded and matches the single-writer guarantee SQLite gives us.
	parseMu sync.Mutex
	// fileImportMu serializes the file-copy / zstd-decompress step in
	// ImportService.ImportFile. Each zstd decoder window is tens of MB; on a
	// bulk drag-and-drop of 10 .zst files the parallel windows alone could
	// spike RAM by 500+ MB on top of whatever parse was running. We accept the
	// extra wall-clock latency (each copy is seconds, not minutes) in exchange
	// for a flat memory profile during bulk imports.
	fileImportMu sync.Mutex
	// parserHeapLimit is the heap-watchdog ceiling sized at startup from host
	// RAM via internal/sysinfo. 0 lets the parser pick its built-in default.
	parserHeapLimit uint64
	// profilesDir is where the heap watchdog writes pprof dumps when it trips.
	// Resolved at startup via database.ProfilesDir(); empty if the resolution
	// failed (the watchdog still trips, just without writing a dump).
	profilesDir string
	// tolerateEntityErrors flips the parser's IgnorePacketEntitiesPanic flag.
	// Default false: corrupt entity tables surface as ErrCorruptEntityTable so
	// the import fails fast instead of running away on memory. Users can opt
	// into partial-parse tolerance via SetTolerateEntityErrors.
	tolerateEntityErrors bool
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

	// Best-effort: if the profiles dir can't be resolved, the watchdog still
	// trips and aborts a runaway parse — it just can't dump a heap profile.
	// Logging the failure here keeps the error visible without making it fatal.
	profilesDir, err := database.ProfilesDir()
	if err != nil {
		slog.Warn("resolve profiles dir; pprof dumps disabled", "err", err)
	}

	return &App{
		ctx:             context.Background(),
		db:              db,
		queries:         queries,
		importService:   demo.NewImportService(queries, db, demosDir),
		demosDir:        demosDir,
		parserHeapLimit: heapLimits.KillSwitch,
		profilesDir:     profilesDir,
	}, nil
}

// Startup is called by Wails after the window is created. It wraps the
// Wails-aware context (needed by runtime.EventsEmit) in a cancellable one so
// Shutdown can interrupt in-flight parse/ingest work that's holding the DB.
func (a *App) Startup(ctx context.Context) {
	a.ctx, a.cancel = context.WithCancel(ctx)
}

// Shutdown is called when the app is closing. It cancels the root context
// before closing the DB so any pending parseDemo goroutines bail out of their
// long-running queries instead of fighting a closed connection pool.
func (a *App) Shutdown(_ context.Context) {
	if a.cancel != nil {
		a.cancel()
	}
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

// ProfilesDir returns the absolute path to the directory holding heap pprof
// dumps written by the parser watchdog. Used by the Settings UI so users can
// locate profiles to attach to a bug report.
func (a *App) ProfilesDir() string {
	return a.profilesDir
}

// OpenProfilesFolder opens the profiles directory in the OS file manager.
func (a *App) OpenProfilesFolder() error {
	if a.profilesDir == "" {
		return fmt.Errorf("profiles directory not initialized")
	}
	return logging.Reveal(a.profilesDir)
}

// GetTolerateEntityErrors returns the current setting for entity-error
// tolerance. When true the parser swallows entity-table corruption and tries
// to keep parsing — at the cost of higher peak memory and a higher chance
// of a watchdog trip on pathological demos.
func (a *App) GetTolerateEntityErrors() bool {
	return a.tolerateEntityErrors
}

// SetTolerateEntityErrors flips the entity-error tolerance flag. Takes effect
// on the next demo import; in-flight parses keep their original setting.
func (a *App) SetTolerateEntityErrors(tolerate bool) {
	a.tolerateEntityErrors = tolerate
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

	data := make([]DemoSummary, len(demos))
	for i, d := range demos {
		data[i] = storeDemoToSummary(d)
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

// CountDemos returns the total number of imported demos. Cheap fan-in for
// callers that only need the count (e.g. sidebar badge) — avoids the
// ListDemos round-trip and JSON marshal of an unused row payload.
func (a *App) CountDemos() (int, error) {
	total, err := a.queries.CountDemos(a.ctx)
	if err != nil {
		return 0, fmt.Errorf("counting demos: %w", err)
	}
	return int(total), nil
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

	d, err := a.importFileSerialized(filePath)
	if err != nil {
		return err
	}
	go a.runParseSerialized(d.ID, d.FilePath)
	return nil
}

// ImportDemoByPath imports a .dem file at the given path (used for drag-and-drop).
func (a *App) ImportDemoByPath(filePath string) error {
	d, err := a.importFileSerialized(filePath)
	if err != nil {
		return err
	}
	go a.runParseSerialized(d.ID, d.FilePath)
	return nil
}

// importFileSerialized wraps ImportService.ImportFile under fileImportMu so a
// bulk drop only runs one file-copy / zstd-decompress at a time. The Wails
// caller still gets a synchronous error return — we just queue them.
func (a *App) importFileSerialized(filePath string) (*store.Demo, error) {
	a.fileImportMu.Lock()
	defer a.fileImportMu.Unlock()
	return a.importService.ImportFile(a.ctx, filePath)
}

// runParseSerialized acquires parseMu before running parseDemo so concurrent
// imports are processed one at a time. Callers spawn this in a goroutine so
// the Wails binding returns immediately; queued parses block on the mutex.
func (a *App) runParseSerialized(demoID int64, filePath string) {
	a.parseMu.Lock()
	defer a.parseMu.Unlock()
	a.parseDemo(demoID, filePath)
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

	// Coalesce same-stage progress emits to at most one per coalesceWindow.
	// Always-emit cases: error (errMsg present), stage change, and terminal stages
	// ("complete"/"error"). Without this, a future per-tick caller could generate
	// 64K+ Wails events per match and flood the IPC bridge.
	const coalesceWindow = 100 * time.Millisecond
	var (
		lastEmit    time.Time
		lastStage   string
		lastPercent int
	)
	emitProgress := func(stage string, percent float64, errMsg ...string) {
		hasErr := len(errMsg) > 0 && errMsg[0] != ""
		isTerminal := stage == "complete" || stage == "error"
		stageChanged := stage != lastStage

		// Round at the IPC boundary so the UI receives clean integer percentages
		// (no "2.6666%" display) and so the monotonic check below operates on the
		// same value the user sees.
		rounded := int(math.Round(percent))
		if rounded < 0 {
			rounded = 0
		} else if rounded > 100 {
			rounded = 100
		}

		// Drop emits that would make the progress bar move backwards within a
		// stage. The parser has two concurrent progress sources — round-based
		// (0–80%, jumps with each RoundEnd) and a frame "alive" heartbeat
		// (0–5%, climbs slowly with frame count). Once round 18 has pushed us to
		// 48%, a subsequent heartbeat emit of ~5% would visually rewind the bar.
		// Stage changes and terminal/error emits always pass through.
		if !hasErr && !isTerminal && !stageChanged && rounded < lastPercent {
			return
		}

		if !hasErr && !isTerminal && !stageChanged && time.Since(lastEmit) < coalesceWindow {
			return
		}
		lastEmit = time.Now()
		lastStage = stage
		lastPercent = rounded

		payload := map[string]interface{}{
			"demoId":   demoID,
			"fileName": fileName,
			"percent":  rounded,
			"stage":    stage,
		}
		if hasErr {
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

	result, err := a.runParsePipeline(demoID, f, a.tolerateEntityErrors, emitProgress)
	if err != nil && errors.Is(err, demo.ErrCorruptEntityTable) && !a.tolerateEntityErrors {
		// The first attempt aborted because demoinfocs panicked on a damaged
		// entity table. Retry once with IgnorePacketEntitiesPanic on so the
		// library swallows the panic and keeps parsing — the heap watchdog
		// (parser.go:488) and tick/event caps (parser.go:34-37) backstop the
		// runaway-memory case the flag was originally kept off to avoid. The
		// ingester's transaction was rolled back on the failed attempt and the
		// next IngestStream call wipes any committed rows via
		// DeleteTickDataByDemoID, so a fresh file offset starts cleanly.
		slog.Warn("parseDemo: retrying with entity-panic tolerance",
			append(logCtx, "first_err", err.Error())...)
		if _, seekErr := f.Seek(0, io.SeekStart); seekErr != nil {
			slog.Error("parseDemo: rewind for retry", append(logCtx, "err", seekErr)...)
			a.failDemo(demoID, fmt.Sprintf("parse failed: %v (rewind for retry: %v)", err, seekErr), emitProgress)
			return
		}
		emitProgress("parsing", 0)
		result, err = a.runParsePipeline(demoID, f, true, emitProgress)
	}
	if err != nil {
		slog.Error("parseDemo: parse/ingest pipeline", append(logCtx, "err", err)...)
		a.failDemo(demoID, fmt.Sprintf("parse failed: %v", err), emitProgress)
		return
	}
	if result == nil {
		// Defensive: should be unreachable since runParsePipeline only returns
		// nil on error.
		slog.Error("parseDemo: parse returned nil result without error", logCtx...)
		a.failDemo(demoID, "parse returned no result", emitProgress)
		return
	}

	// Force a GC cycle now that the demoinfocs parser is closed and its
	// internal entity tables / packet buffers are unreferenced. Without this
	// nudge, all that transient state coexists with the events-ingest tx for
	// the seconds it takes to run, doubling peak commit on Windows where the
	// runtime is slower to scavenge unused heap.
	runtime.GC()

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
	// Drop our reference to the (potentially 100+ MB) events slice so the
	// next GC cycle can reclaim it before we move on to the metadata update
	// and emit "complete". The post-parse aggregation (lineups, kill→hurt
	// pairing) is already done by the time IngestGameEvents returns.
	result.Events = nil
	result.Lineups = nil
	result.Rounds = nil

	// debug.FreeOSMemory() forces a full GC + scavenge, signalling the runtime
	// to madvise/decommit unused pages back to the OS. On macOS/Linux this is
	// mostly a no-op (the runtime is already aggressive); on Windows it's the
	// only reliable way to drop the working set after a memory-heavy operation,
	// since the runtime tends to hold onto pages there. Cost is ~50–200 ms once
	// per import — well worth it for a flat memory profile in Task Manager.
	debug.FreeOSMemory()

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

// runParsePipeline runs the streaming parse+ingest pipeline once. The parser
// pushes each TickSnapshot into ticksCh; the ingester drains it and writes
// batched INSERTs in a single transaction. This overlaps protobuf decode
// (CPU-bound) with SQLite WAL writes (I/O-bound) and caps peak heap by
// removing the 100+ MB tick slice that would otherwise live in memory until
// parsing finished.
//
// Events stay in memory because the post-parse pipeline (dropKnifeRounds,
// pairShotsWithImpacts, ExtractGrenadeLineups) needs the full event list for
// forward-lookup correlations — events are only ~12 MB at the cap so the win
// wouldn't pay for the redesign.
//
// On error, the ingester's transaction is rolled back via defer (ingest.go:100)
// so partial tick rows do not survive a failed attempt; the next call's
// DeleteTickDataByDemoID also wipes any rows that did commit before the
// failure. This makes the pipeline safe to retry from a rewound file offset.
func (a *App) runParsePipeline(
	demoID int64,
	f io.Reader,
	tolerateEntityErrors bool,
	emitProgress func(stage string, percent float64, errMsg ...string),
) (*demo.ParseResult, error) {
	ticksCh := make(chan demo.TickSnapshot, demo.DefaultTickSinkBuffer)
	parser := demo.NewDemoParser(
		demo.WithProgressFunc(func(stage string, percent float64) {
			emitProgress(stage, percent)
		}),
		demo.WithTickSink(ticksCh),
		demo.WithHeapLimit(a.parserHeapLimit),
		demo.WithProfilesDir(a.profilesDir),
		demo.WithIgnoreEntityPanics(tolerateEntityErrors),
	)
	ingester := demo.NewTickIngester(a.db, 0)

	g, gctx := errgroup.WithContext(a.ctx)
	var result *demo.ParseResult

	g.Go(func() error {
		// errgroup synchronizes on g.Wait, so writing result here and reading
		// after Wait is happens-before-correct without a mutex.
		res, parseErr := parser.Parse(gctx, f)
		result = res
		return parseErr
	})

	g.Go(func() error {
		_, ingestErr := ingester.IngestStream(gctx, demoID, ticksCh)
		return ingestErr
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return result, nil
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

// GetEventsByTypes returns only the game events whose event_type is in the
// supplied list. Lets callers like the kill-log avoid loading every event
// (and per-row extra_data JSON decode) when they render a small subset.
// Returns an empty slice when eventTypes is empty rather than every event,
// since "no types requested" almost certainly means the caller is in a
// loading state and we shouldn't accidentally fall back to the full payload.
func (a *App) GetEventsByTypes(demoID string, eventTypes []string) ([]GameEvent, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}
	if len(eventTypes) == 0 {
		return []GameEvent{}, nil
	}

	typesJSON, err := json.Marshal(eventTypes)
	if err != nil {
		return nil, fmt.Errorf("marshaling event types: %w", err)
	}

	events, err := a.queries.GetGameEventsByTypes(a.ctx, id, string(typesJSON))
	if err != nil {
		return nil, fmt.Errorf("getting events by types: %w", err)
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

// GetAllRosters returns every round's roster for a demo in a single call,
// keyed by round_number. The viewer uses this on demo open so PixiJS round
// transitions can look the roster up locally instead of issuing a fresh
// Wails round-trip per round (24-30 per match). 150-300 rows total at ~100 B
// each is well under any payload threshold worth paginating.
func (a *App) GetAllRosters(demoID string) (map[int][]PlayerRosterEntry, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}

	rows, err := a.queries.GetRostersByDemoID(a.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting all rosters: %w", err)
	}

	result := make(map[int][]PlayerRosterEntry)
	for _, r := range rows {
		round := int(r.RoundNumber)
		result[round] = append(result[round], PlayerRosterEntry{
			SteamID:    r.SteamID,
			PlayerName: r.PlayerName,
			TeamSide:   r.TeamSide,
		})
	}
	return result, nil
}

// GetRoundLoadouts returns the freeze-end loadout for every (round, player)
// in a demo, keyed by round_number. The viewer fetches this once at demo
// open and the team bars look up the current round's loadout instead of
// reading inventory off each tick (migration 011).
func (a *App) GetRoundLoadouts(demoID string) (map[int][]RoundLoadoutEntry, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}

	rows, err := a.queries.GetRoundLoadoutsByDemoID(a.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting round loadouts: %w", err)
	}

	result := make(map[int][]RoundLoadoutEntry)
	for _, r := range rows {
		round := int(r.RoundNumber)
		result[round] = append(result[round], RoundLoadoutEntry{
			SteamID:   r.SteamID,
			Inventory: r.Inventory,
		})
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

// GetPlayerMatchStats returns the deep-stats payload for a single player in a
// demo. Computed on demand from already-ingested rounds, events, and loadouts
// — no new ingest write path. The frontend caches the result via TanStack
// Query (staleTime: Infinity) since the underlying data is deterministic per
// (demo, steamID).
func (a *App) GetPlayerMatchStats(demoID, steamID string) (*PlayerMatchStats, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}
	if steamID == "" {
		return nil, fmt.Errorf("steam id required")
	}

	d, err := a.queries.GetDemoByID(a.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting demo: %w", err)
	}

	storeRounds, err := a.queries.GetRoundsByDemoID(a.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting rounds: %w", err)
	}

	storeEvents, err := a.queries.GetGameEventsByDemoID(a.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting events: %w", err)
	}

	loadoutRows, err := a.queries.GetRoundLoadoutsByDemoID(a.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting round loadouts: %w", err)
	}

	rosterRows, err := a.queries.GetRostersByDemoID(a.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting rosters: %w", err)
	}

	rounds, events, loadouts := buildPlayerStatsInputs(storeRounds, storeEvents, loadoutRows, rosterRows)

	// Phase 2: pull this player's tick samples for movement / timing
	// aggregation, and derive bombsite centroids from BombPlanted events for
	// the time-on-site proxy.
	tickRows, err := a.queries.GetTickDataByDemoAndPlayer(a.ctx, id, steamID)
	if err != nil {
		return nil, fmt.Errorf("getting tick data: %w", err)
	}
	samples := tickDataToSamples(tickRows)

	// Phase 3: prefer hand-authored bombsite polygons (callouts.go) when the
	// map is known, else fall back to the BombPlanted-derived centroid + a
	// fixed bounding-circle radius.
	bombsites := demo.BombsiteCentroidsFromEvents(events)
	if polys := demo.BombsitePolygonsForMap(d.MapName); len(polys) > 0 {
		bombsites = mergeBombsitePolygons(bombsites, polys)
	}

	stats := demo.ComputePlayerMatchStats(rounds, events, loadouts, samples, bombsites, steamID, d.TickRate)
	out := computedPlayerMatchStatsToBinding(stats)
	return &out, nil
}

// mergeBombsitePolygons attaches polygon bounds to the BombPlanted-derived
// centroids when both are available, and creates synthetic centroid entries
// for sites that have a polygon but never saw a plant in this match (a CT-
// dominant half can leave one site planted-zero).
func mergeBombsitePolygons(centroids []demo.BombsiteCentroid, polys []demo.SitePolygon) []demo.BombsiteCentroid {
	bySite := make(map[string]demo.BombsiteCentroid, len(centroids))
	for _, c := range centroids {
		bySite[c.Site] = c
	}
	for _, p := range polys {
		c := bySite[p.Site]
		c.Site = p.Site
		c.MinX, c.MaxX = p.MinX, p.MaxX
		c.MinY, c.MaxY = p.MinY, p.MaxY
		// Synthesize centroid X/Y from polygon midpoint when no plant is
		// available — keeps the legacy bounding-circle path useful for
		// callers that don't read the polygon fields.
		if c.X == 0 && c.Y == 0 {
			c.X = (p.MinX + p.MaxX) / 2
			c.Y = (p.MinY + p.MaxY) / 2
		}
		bySite[p.Site] = c
	}
	out := make([]demo.BombsiteCentroid, 0, len(bySite))
	for _, site := range []string{"A", "B"} {
		if c, ok := bySite[site]; ok {
			out = append(out, c)
		}
	}
	return out
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

func storeDemoToSummary(d store.Demo) DemoSummary {
	return DemoSummary{
		ID:           d.ID,
		MapName:      d.MapName,
		FileName:     filepath.Base(d.FilePath),
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
		ID:           strconv.FormatInt(e.ID, 10),
		DemoID:       strconv.FormatInt(e.DemoID, 10),
		Tick:         int(e.Tick),
		EventType:    e.EventType,
		X:            &x,
		Y:            &y,
		Z:            &z,
		Headshot:     e.Headshot != 0,
		HealthDamage: int(e.HealthDamage),
		AttackerName: e.AttackerName,
		VictimName:   e.VictimName,
		AttackerTeam: e.AttackerTeam,
		VictimTeam:   e.VictimTeam,
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
	if e.AssisterSteamID.Valid {
		ge.AssisterSteamID = &e.AssisterSteamID.String
	}
	if e.ExtraData != "" {
		// Pass through as raw JSON: Wails will emit these bytes verbatim into the
		// outer payload, skipping a round-trip through map[string]any (per-row
		// map alloc + per-key string allocs + per-value boxing) on a hot path
		// that runs once per game event.
		ge.ExtraData = json.RawMessage(e.ExtraData)
	}
	return ge
}

func storeTickDatumToBinding(d store.TickDatum) TickData {
	td := TickData{
		Tick:        int(d.Tick),
		SteamID:     d.SteamID,
		X:           roundToInt16(d.X),
		Y:           roundToInt16(d.Y),
		Z:           roundToInt16(d.Z),
		Yaw:         roundToInt16(d.Yaw),
		Health:      int(d.Health),
		Armor:       int(d.Armor),
		IsAlive:     d.IsAlive != 0,
		Money:       int(d.Money),
		HasHelmet:   d.HasHelmet != 0,
		HasDefuser:  d.HasDefuser != 0,
		AmmoClip:    int(d.AmmoClip),
		AmmoReserve: int(d.AmmoReserve),
	}
	if d.Weapon != "" {
		td.Weapon = &d.Weapon
	}
	return td
}

// roundToInt16 clamps and rounds a float to the int16 range. CS2 world
// coordinates fit comfortably in ±32k units (a unit ≈ 2.5 cm), and yaw is in
// degrees ±180 — both well inside int16. Out-of-range values (e.g. clipped
// players outside the playable space) are clamped instead of overflowing.
func roundToInt16(v float64) int16 {
	r := math.Round(v)
	if r > math.MaxInt16 {
		return math.MaxInt16
	}
	if r < math.MinInt16 {
		return math.MinInt16
	}
	return int16(r)
}
