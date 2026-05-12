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
	"github.com/ok2ju/oversite/internal/demo/analysis"
	"github.com/ok2ju/oversite/internal/demo/contacts"
	"github.com/ok2ju/oversite/internal/demo/contacts/detectors"
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
	// Shared by both runParseSerialized (import flow) and RecomputeAnalysis
	// (legacy-demo backfill); a recompute issued during an import waits behind
	// the in-flight parse instead of racing it.
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

	if _, err := demo.IngestPlayerVisibility(a.ctx, a.db, demoID, result.Visibility, roundMap); err != nil {
		slog.Error("parseDemo: ingest visibility", append(logCtx, "err", err)...)
		a.failDemo(demoID, fmt.Sprintf("ingest visibility: %v", err), emitProgress)
		return
	}

	// Mechanical-analysis pass: runs over the in-memory events (still
	// available — IngestGameEvents only reads them) and persists per-player
	// findings into analysis_mistakes. Bracketed by progress events so the
	// UI's parse-progress bar covers the previously silent gap between
	// "ingest events" and "complete".
	emitProgress("analyzing", 80)
	analysisOpts := analysis.RunOpts{}
	mistakes, duels, analysisErr := analysis.Run(result, roundMap, analysisOpts)
	if analysisErr != nil {
		slog.Error("parseDemo: run analyzer", append(logCtx, "err", analysisErr)...)
		a.failDemo(demoID, fmt.Sprintf("run analyzer: %v", analysisErr), emitProgress)
		return
	}
	// Phase 2 of timeline-contact-moments: build per-(player, signal-cluster)
	// contact moments from the in-memory ParseResult and persist. Phase 3
	// layers mistake detectors on top, both reading from the persisted
	// contact set and emitting an aggregate slice that joins the analyzer
	// mistakes before they hit analysis_mistakes.
	slog.Info("starting contact build", "demo_id", demoID, "round_count", len(result.Rounds))
	contactsList, contactsErr := contacts.Run(result, roundMap, contacts.RunOpts{})
	if contactsErr != nil {
		slog.Error("parseDemo: run contact builder", append(logCtx, "err", contactsErr)...)
		a.failDemo(demoID, fmt.Sprintf("run contact builder: %v", contactsErr), emitProgress)
		return
	}
	if err := contacts.Persist(a.ctx, a.db, demoID, contactsList); err != nil {
		slog.Error("parseDemo: persist contact moments", append(logCtx, "err", err)...)
		a.failDemo(demoID, fmt.Sprintf("persist contacts: %v", err), emitProgress)
		return
	}
	slog.Info("contact build complete", "demo_id", demoID, "contact_count", len(contactsList))
	// Phase 3: mistake detectors run over the persisted contact set.
	// boundMistakes are written to contact_mistakes; aggregateMistakes
	// merge into the analyzer mistake slice so the four WriteAggregate
	// kinds (slow_reaction / missed_first_shot / isolated_peek /
	// shot_while_moving) appear in analysis_mistakes alongside the
	// existing round-level rules.
	boundMistakes, aggregateMistakes, detectorsSkipped, detectorsErr := detectors.Run(
		a.ctx, a.db, demoID, result, contactsList, detectors.RunOpts{},
	)
	if detectorsErr != nil {
		slog.Error("parseDemo: run mistake detectors", append(logCtx, "err", detectorsErr)...)
		a.failDemo(demoID, fmt.Sprintf("run detectors: %v", detectorsErr), emitProgress)
		return
	}
	if !detectorsSkipped {
		mistakes = append(mistakes, aggregateMistakes...)
	}
	if err := analysis.PersistWithRoundMap(a.ctx, a.db, demoID, mistakes, duels, roundMap); err != nil {
		slog.Error("parseDemo: persist analyzer mistakes", append(logCtx, "err", err)...)
		a.failDemo(demoID, fmt.Sprintf("persist analysis: %v", err), emitProgress)
		return
	}
	if !detectorsSkipped {
		if err := detectors.Persist(a.ctx, a.db, demoID, boundMistakes); err != nil {
			slog.Error("parseDemo: persist contact mistakes", append(logCtx, "err", err)...)
			a.failDemo(demoID, fmt.Sprintf("persist contact mistakes: %v", err), emitProgress)
			return
		}
	}
	slog.Info("mistake detectors complete",
		"demo_id", demoID,
		"contact_mistakes", len(boundMistakes),
		"aggregate_mistakes", len(aggregateMistakes),
		"skipped", detectorsSkipped,
	)
	// Per-(demo, player) summary row alongside the timeline. Each persist is
	// its own transaction; the divergence window between mistakes-on-disk
	// and summary-on-disk is small (single-tenant, single-process) and slice 7
	// will revisit consolidating both into one tx when a third table lands.
	summaryRows, summaryErr := analysis.RunMatchSummary(result, roundMap, analysisOpts)
	if summaryErr != nil {
		slog.Error("parseDemo: run match summary", append(logCtx, "err", summaryErr)...)
		a.failDemo(demoID, fmt.Sprintf("run match summary: %v", summaryErr), emitProgress)
		return
	}
	if err := analysis.PersistMatchSummary(a.ctx, a.db, demoID, summaryRows); err != nil {
		slog.Error("parseDemo: persist match summary", append(logCtx, "err", err)...)
		a.failDemo(demoID, fmt.Sprintf("persist match summary: %v", err), emitProgress)
		return
	}
	// Per-(demo, player, round) breakdown alongside the match-level summary.
	// Backs the standalone analysis page's per-round bar chart (slice 7).
	roundRows, roundErr := analysis.RunPlayerRoundAnalysis(result, roundMap, analysisOpts)
	if roundErr != nil {
		slog.Error("parseDemo: run player round analysis", append(logCtx, "err", roundErr)...)
		a.failDemo(demoID, fmt.Sprintf("run player round analysis: %v", roundErr), emitProgress)
		return
	}
	if err := analysis.PersistPlayerRoundAnalysis(a.ctx, a.db, demoID, roundRows); err != nil {
		slog.Error("parseDemo: persist player round analysis", append(logCtx, "err", err)...)
		a.failDemo(demoID, fmt.Sprintf("persist player round analysis: %v", err), emitProgress)
		return
	}
	emitProgress("analyzing", 95)

	// Drop our reference to the (potentially 100+ MB) events slice so the
	// next GC cycle can reclaim it before we move on to the metadata update
	// and emit "complete". The post-parse aggregation (lineups, kill→hurt
	// pairing) is already done by the time IngestGameEvents returns, and
	// analysis.Run finished consuming the slice above.
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
		demo.WithTickFanout(true),
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

// roundImportantEventTypes are the event types the round-mode timeline lane
// considers. Excludes kill/player_hurt/player_flashed — those collapse into
// the per-player contact lane in player mode and don't render in round mode
// at all (per analysis §4.5).
var roundImportantEventTypes = []string{
	"grenade_throw",
	"grenade_detonate",
	"bomb_plant",
	"bomb_defuse",
	"bomb_explode",
}

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

// GetMistakeTimeline returns the chronologically ordered list of analyzer
// mistakes for a (demo, player). Unknown demos / unknown players return an
// empty slice rather than an error — the side panel renders an empty state
// for the latter and treats the former as "no analysis yet".
func (a *App) GetMistakeTimeline(demoID, steamID string) ([]MistakeEntry, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}
	if steamID == "" {
		return []MistakeEntry{}, nil
	}

	rows, err := a.queries.ListAnalysisMistakesByDemoIDAndSteamID(a.ctx, store.ListAnalysisMistakesByDemoIDAndSteamIDParams{
		DemoID:  id,
		SteamID: steamID,
	})
	if err != nil {
		return nil, fmt.Errorf("getting mistake timeline: %w", err)
	}

	result := make([]MistakeEntry, len(rows))
	for i, r := range rows {
		tpl := analysis.TemplateForKind(r.Kind)
		category := r.Category
		if category == "" {
			category = string(tpl.Category)
		}
		severity := int(r.Severity)
		if severity == 0 {
			severity = int(tpl.Severity)
		}
		entry := MistakeEntry{
			ID:          r.ID,
			Kind:        r.Kind,
			Category:    category,
			Severity:    severity,
			Title:       tpl.Title,
			Suggestion:  tpl.Suggestion,
			WhyItHurts:  tpl.WhyItHurts,
			RoundNumber: int(r.RoundNumber),
			Tick:        r.Tick,
			SteamID:     r.SteamID,
		}
		if r.DuelID.Valid {
			v := r.DuelID.Int64
			entry.DuelID = &v
		}
		// extras_json is stored as a TEXT blob; decode lazily so the frontend
		// can read individual rule fields without a second round-trip. A
		// malformed blob (shouldn't happen — Persist always marshals) is
		// surfaced as an empty map so the panel still renders the row.
		if r.ExtrasJson != "" && r.ExtrasJson != "{}" {
			extras := map[string]any{}
			if err := json.Unmarshal([]byte(r.ExtrasJson), &extras); err == nil {
				entry.Extras = extras
			}
		}
		result[i] = entry
	}
	return result, nil
}

// GetMistakeContext returns one analyzer mistake by ID, enriched with the
// surrounding round window so the analysis-detail card can render the play
// without an extra GetDemoRounds round-trip. Returns sql.ErrNoRows-shaped
// errors as "not found" rather than propagating — the panel renders an empty
// detail when a stale ID is clicked.
func (a *App) GetMistakeContext(mistakeID int64) (*MistakeContext, error) {
	row, err := a.queries.GetAnalysisMistakeByID(a.ctx, mistakeID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting mistake by id: %w", err)
	}
	tpl := analysis.TemplateForKind(row.Kind)
	category := row.Category
	if category == "" {
		category = string(tpl.Category)
	}
	severity := int(row.Severity)
	if severity == 0 {
		severity = int(tpl.Severity)
	}
	entry := MistakeEntry{
		ID:          row.ID,
		Kind:        row.Kind,
		Category:    category,
		Severity:    severity,
		Title:       tpl.Title,
		Suggestion:  tpl.Suggestion,
		WhyItHurts:  tpl.WhyItHurts,
		RoundNumber: int(row.RoundNumber),
		Tick:        row.Tick,
		SteamID:     row.SteamID,
	}
	if row.DuelID.Valid {
		v := row.DuelID.Int64
		entry.DuelID = &v
	}
	if row.ExtrasJson != "" && row.ExtrasJson != "{}" {
		extras := map[string]any{}
		if err := json.Unmarshal([]byte(row.ExtrasJson), &extras); err == nil {
			entry.Extras = extras
		}
	}
	out := &MistakeContext{Entry: entry}
	// Round window — best-effort. A row whose round_id is NULL (legacy
	// pre-slice-10 data) carries empty round window fields; the frontend
	// falls back to scanning the rounds collection in that case.
	if row.RoundID.Valid {
		r, err := a.queries.GetRoundByID(a.ctx, row.RoundID.Int64)
		if err == nil {
			out.RoundStartTck = r.StartTick
			out.RoundEndTick = r.EndTick
			out.FreezeEndTick = r.FreezeEndTick
		}
	}
	// Co-occurring mistakes — same (demo, player), within ±coOccurringWindowTicks
	// of the pinned tick, excluding self. Reuses the existing list query so we
	// don't have to add a tick-range SQL — match-level mistake counts are small
	// enough (hundreds, not thousands) that an in-memory filter is cheaper than
	// a new index.
	if siblings, err := a.queries.ListAnalysisMistakesByDemoIDAndSteamID(a.ctx, store.ListAnalysisMistakesByDemoIDAndSteamIDParams{
		DemoID:  row.DemoID,
		SteamID: row.SteamID,
	}); err == nil {
		out.CoOccurring = collectCoOccurring(row.ID, row.Tick, siblings)
	}
	return out, nil
}

// GetContactMoments returns the contact list for one (demo, round, player)
// with each contact's mistakes embedded. Unknown demos / unknown rounds /
// unknown players return an empty slice rather than an error — the frontend
// renders an empty lane in that case, mirroring GetMistakeTimeline's contract.
//
// Mistakes are populated from contact_mistakes; the slice ordering is
// (phase ASC, severity DESC, tick ASC), matching the
// ListContactMistakesByContact SQL ORDER BY.
func (a *App) GetContactMoments(demoID string, roundNumber int, subjectSteam string) ([]ContactMoment, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}
	if subjectSteam == "" {
		return []ContactMoment{}, nil
	}

	round, err := a.queries.GetRoundByDemoAndNumber(a.ctx, store.GetRoundByDemoAndNumberParams{
		DemoID:      id,
		RoundNumber: int64(roundNumber),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return []ContactMoment{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting round: %w", err)
	}

	rows, err := a.queries.ListContactsByDemoRoundSubject(a.ctx, store.ListContactsByDemoRoundSubjectParams{
		DemoID:       id,
		RoundID:      round.ID,
		SubjectSteam: subjectSteam,
	})
	if err != nil {
		return nil, fmt.Errorf("listing contacts: %w", err)
	}

	out := make([]ContactMoment, 0, len(rows))
	for _, r := range rows {
		cm, err := storeContactToBinding(r, int(round.RoundNumber))
		if err != nil {
			return nil, fmt.Errorf("decoding contact id=%d: %w", r.ID, err)
		}
		mistakeRows, err := a.queries.ListContactMistakesByContact(a.ctx, r.ID)
		if err != nil {
			return nil, fmt.Errorf("listing mistakes for contact %d: %w", r.ID, err)
		}
		cm.Mistakes = make([]ContactMistake, 0, len(mistakeRows))
		for _, m := range mistakeRows {
			cm.Mistakes = append(cm.Mistakes, storeContactMistakeToBinding(m))
		}
		out = append(out, cm)
	}
	return out, nil
}

// GetRoundImportantMoments returns only the events the round-mode timeline
// lane considers important: grenade lifecycle + bomb lifecycle. Phase 4 uses
// this to drop kill/player_hurt/player_flashed from the round-mode lane
// (analysis §4.5: round-mode shows what the team did, not who individually
// died). Equivalent to GetEventsByTypes(demoID, roundImportantEventTypes),
// filtered in-Go to the supplied round. Returns an empty slice for unknown
// demos / unknown rounds.
func (a *App) GetRoundImportantMoments(demoID string, roundNumber int) ([]GameEvent, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}

	round, err := a.queries.GetRoundByDemoAndNumber(a.ctx, store.GetRoundByDemoAndNumberParams{
		DemoID:      id,
		RoundNumber: int64(roundNumber),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return []GameEvent{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting round: %w", err)
	}

	typesJSON, err := json.Marshal(roundImportantEventTypes)
	if err != nil {
		return nil, fmt.Errorf("marshaling event types: %w", err)
	}

	events, err := a.queries.GetGameEventsByTypes(a.ctx, id, string(typesJSON))
	if err != nil {
		return nil, fmt.Errorf("getting events by types: %w", err)
	}

	out := make([]GameEvent, 0, len(events))
	for _, e := range events {
		if e.Tick < round.StartTick || e.Tick > round.EndTick {
			continue
		}
		out = append(out, storeGameEventToBinding(e))
	}
	return out, nil
}

// coOccurringWindowTicks bounds the per-mistake co-occurrence chip row. Half a
// second at 64 tickrate is wide enough to capture a multi-fault duel but
// narrow enough that the chips refer to the same fight, not the next one.
const coOccurringWindowTicks = 32

// collectCoOccurring filters the player's full mistake list to those within
// ±coOccurringWindowTicks of pinnedTick and not equal to pinnedID. Title /
// kind come from analysis.TemplateForKind so the chip row can render without
// extra round-trips. Capped at 4 entries — more than that overflows the chip
// row visually and tends to indicate a noisy detection.
func collectCoOccurring(pinnedID, pinnedTick int64, siblings []store.ListAnalysisMistakesByDemoIDAndSteamIDRow) []MistakeCoOccurrence {
	const maxChips = 4
	out := make([]MistakeCoOccurrence, 0, maxChips)
	for _, s := range siblings {
		if s.ID == pinnedID {
			continue
		}
		dt := s.Tick - pinnedTick
		if dt < 0 {
			dt = -dt
		}
		if dt > coOccurringWindowTicks {
			continue
		}
		tpl := analysis.TemplateForKind(s.Kind)
		title := tpl.Title
		if title == "" {
			title = s.Kind
		}
		out = append(out, MistakeCoOccurrence{
			ID:    s.ID,
			Kind:  s.Kind,
			Title: title,
			Tick:  s.Tick,
		})
		if len(out) >= maxChips {
			break
		}
	}
	return out
}

// ListDuelsForPlayer returns the directed duels (attacker→victim or
// victim→attacker) where the supplied player is one side of the
// engagement, ordered by (round, start_tick). Powers the duels lane on
// the round-timeline. Unknown demos / unknown players return an empty
// slice.
func (a *App) ListDuelsForPlayer(demoID, steamID string) ([]DuelEntry, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}
	if steamID == "" {
		return []DuelEntry{}, nil
	}
	rows, err := a.queries.ListAnalysisDuelsByDemoIDAndSteamID(a.ctx, store.ListAnalysisDuelsByDemoIDAndSteamIDParams{
		DemoID:  id,
		SteamID: steamID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing duels for player: %w", err)
	}
	out := make([]DuelEntry, len(rows))
	for i, r := range rows {
		out[i] = duelRowToEntry(r)
	}
	return out, nil
}

// GetDuelContext returns one duel and the mistakes attached to it.
// Returns a nil pointer for unknown ids (mirrors GetMistakeContext).
func (a *App) GetDuelContext(duelID int64) (*DuelContext, error) {
	row, err := a.queries.GetAnalysisDuelByID(a.ctx, duelID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting duel by id: %w", err)
	}
	ctxOut := &DuelContext{Duel: duelRowToEntry(row)}
	mistakeRows, err := a.queries.ListAnalysisMistakesByDuelID(a.ctx, sql.NullInt64{Int64: duelID, Valid: true})
	if err != nil {
		return ctxOut, nil
	}
	ctxOut.Mistakes = make([]MistakeEntry, len(mistakeRows))
	for i, m := range mistakeRows {
		tpl := analysis.TemplateForKind(m.Kind)
		category := m.Category
		if category == "" {
			category = string(tpl.Category)
		}
		severity := int(m.Severity)
		if severity == 0 {
			severity = int(tpl.Severity)
		}
		entry := MistakeEntry{
			ID:          m.ID,
			Kind:        m.Kind,
			Category:    category,
			Severity:    severity,
			Title:       tpl.Title,
			Suggestion:  tpl.Suggestion,
			WhyItHurts:  tpl.WhyItHurts,
			RoundNumber: int(m.RoundNumber),
			Tick:        m.Tick,
			SteamID:     m.SteamID,
		}
		if m.DuelID.Valid {
			v := m.DuelID.Int64
			entry.DuelID = &v
		}
		if m.ExtrasJson != "" && m.ExtrasJson != "{}" {
			extras := map[string]any{}
			if err := json.Unmarshal([]byte(m.ExtrasJson), &extras); err == nil {
				entry.Extras = extras
			}
		}
		ctxOut.Mistakes[i] = entry
	}
	return ctxOut, nil
}

// duelRowToEntry converts the sqlc-generated AnalysisDuel row into the
// wire shape consumed by the frontend. HitConfirmed is stored as an
// INTEGER (0/1) in SQLite; the boolean conversion happens here.
func duelRowToEntry(r store.AnalysisDuel) DuelEntry {
	entry := DuelEntry{
		ID:            r.ID,
		RoundNumber:   int(r.RoundNumber),
		AttackerSteam: r.AttackerSteam,
		VictimSteam:   r.VictimSteam,
		StartTick:     r.StartTick,
		EndTick:       r.EndTick,
		Outcome:       r.Outcome,
		EndReason:     r.EndReason,
		HitConfirmed:  r.HitConfirmed != 0,
		HurtCount:     int(r.HurtCount),
		ShotCount:     int(r.ShotCount),
	}
	if r.MutualDuelID.Valid {
		v := r.MutualDuelID.Int64
		entry.MutualDuelID = &v
	}
	return entry
}

// GetPlayerAnalysis returns the per-(demo, player) summary row written by the
// match-summary analyzer. Unknown demos / unknown players return a zero-valued
// PlayerAnalysis (not an error) so the viewer's gauge / category card can
// render an empty state, mirroring GetMistakeTimeline's contract.
func (a *App) GetPlayerAnalysis(demoID, steamID string) (PlayerAnalysis, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return PlayerAnalysis{}, fmt.Errorf("invalid demo id: %w", err)
	}
	if steamID == "" {
		return PlayerAnalysis{}, nil
	}

	row, err := a.queries.GetPlayerMatchAnalysis(a.ctx, store.GetPlayerMatchAnalysisParams{
		DemoID:  id,
		SteamID: steamID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return PlayerAnalysis{}, nil
	}
	if err != nil {
		return PlayerAnalysis{}, fmt.Errorf("getting player analysis: %w", err)
	}

	out := PlayerAnalysis{
		SteamID:               row.SteamID,
		OverallScore:          int(row.OverallScore),
		Version:               int(row.Version),
		TradePct:              row.TradePct,
		AvgTradeTicks:         row.AvgTradeTicks,
		CrosshairHeightAvgOff: row.CrosshairHeightAvgOff,
		TimeToFireMsAvg:       row.TimeToFireMsAvg,
		FlickCount:            int(row.FlickCount),
		FlickHitPct:           row.FlickHitPct,
		FirstShotAccPct:       row.FirstShotAccPct,
		SprayDecaySlope:       row.SprayDecaySlope,
		StandingShotPct:       row.StandingShotPct,
		CounterStrafePct:      row.CounterStrafePct,
		SmokesThrown:          int(row.SmokesThrown),
		SmokesKillAssist:      int(row.SmokesKillAssist),
		FlashAssists:          int(row.FlashAssists),
		HeDamage:              int(row.HeDamage),
		NadesUnused:           int(row.NadesUnused),
		IsolatedPeekDeaths:    int(row.IsolatedPeekDeaths),
		RepeatedDeathZones:    int(row.RepeatedDeathZones),
		FullBuyADR:            row.FullBuyAdr,
		EcoKills:              int(row.EcoKills),
	}
	if row.ExtrasJson != "" && row.ExtrasJson != "{}" {
		extras := map[string]any{}
		if err := json.Unmarshal([]byte(row.ExtrasJson), &extras); err == nil {
			out.Extras = extras
		}
	}
	return out, nil
}

// GetHabitReport returns the per-(demo, player) habit checklist — the rows
// the analysis page's "habits" surface renders. Each row carries its own
// status (good/warn/bad) classified server-side from the norm catalog
// (analysis/norms.go) so the frontend never re-implements thresholds.
//
// Empty contracts mirror GetPlayerAnalysis: an empty steamID returns a zero
// HabitReport with no habits (not an error) so the analysis page can render
// an empty state. Unknown (demo, player) — i.e. no player_match_analysis row
// — also returns the empty report. This keeps the consumer contract uniform
// regardless of whether the demo is pre-analysis or simply doesn't have the
// player.
func (a *App) GetHabitReport(demoID, steamID string) (HabitReport, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return HabitReport{}, fmt.Errorf("invalid demo id: %w", err)
	}
	out := HabitReport{DemoID: demoID, SteamID: steamID, Habits: []HabitRow{}}
	if steamID == "" {
		return out, nil
	}

	if d, derr := a.queries.GetDemoByID(a.ctx, id); derr == nil {
		out.AsOf = d.MatchDate
	}

	in, ok, err := analysis.LoadHabitInputs(a.ctx, a.queries, id, steamID)
	if err != nil {
		return HabitReport{}, fmt.Errorf("loading habit inputs: %w", err)
	}
	if !ok {
		return out, nil
	}

	rows := analysis.BuildHabitReport(in)

	// History powers the per-row delta line. 9 rows is the current demo +
	// 8 prior — the sparkline window the coaching page renders. A failure
	// to load history is non-fatal for the report; we just ship the rows
	// without deltas and log so the slow path is visible in errors.txt.
	history, herr := analysis.LoadHabitHistory(a.ctx, a.queries, steamID, 9)
	if herr != nil {
		slog.Warn("GetHabitReport: load habit history",
			"demo_id", id, "steam_id", steamID, "err", herr)
	} else {
		analysis.AttachDeltas(rows, history, id)
	}

	out.Habits = make([]HabitRow, len(rows))
	for i, r := range rows {
		out.Habits[i] = habitRowToBinding(r)
	}
	return out, nil
}

// GetHabitHistory returns the player's last `limit` (demo, value) pairs for a
// single habit, sorted newest-first. Powers the analysis-page trend column
// and the coaching-landing-page sparklines (the latter calls it once per
// card).
//
// Empty contracts: empty steamID → empty list; unknown habit key → empty
// list; limit ≤ 0 → defaults to 8. Returns an error only on actual SQL
// failure so callers can render an empty trend without try/catching.
func (a *App) GetHabitHistory(steamID, habitKey string, limit int) ([]HistoryPoint, error) {
	if steamID == "" || habitKey == "" {
		return []HistoryPoint{}, nil
	}
	if _, ok := analysis.LookupNorm(analysis.HabitKey(habitKey)); !ok {
		return []HistoryPoint{}, nil
	}
	if limit <= 0 {
		limit = 8
	}

	records, err := analysis.LoadHabitHistory(a.ctx, a.queries, steamID, limit)
	if err != nil {
		return nil, fmt.Errorf("loading habit history: %w", err)
	}
	points := analysis.HistoryForKey(records, analysis.HabitKey(habitKey))

	out := make([]HistoryPoint, len(points))
	for i, p := range points {
		out[i] = HistoryPoint{
			DemoID:    strconv.FormatInt(p.DemoID, 10),
			MatchDate: p.MatchDate,
			Value:     p.Value,
		}
	}
	return out, nil
}

// GetNextDrill returns the single drill prescribed for the player's worst
// habit (priority: bad > warn, ties broken by impact rank — see
// analysis.PickNextDrill / docs §6.2). Empty steamID, unknown
// (demo, player), and "all good" all return the maintenance fallback so
// the frontend can render the card without try/catching.
func (a *App) GetNextDrill(demoID, steamID string) (NextDrill, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return NextDrill{}, fmt.Errorf("invalid demo id: %w", err)
	}
	if steamID == "" {
		return drillToBinding(analysis.MaintenanceDrill), nil
	}

	in, ok, err := analysis.LoadHabitInputs(a.ctx, a.queries, id, steamID)
	if err != nil {
		return NextDrill{}, fmt.Errorf("loading habit inputs: %w", err)
	}
	if !ok {
		return drillToBinding(analysis.MaintenanceDrill), nil
	}

	rows := analysis.BuildHabitReport(in)
	return drillToBinding(analysis.PickNextDrill(rows)), nil
}

// drillToBinding flattens analysis.Drill to the wire-shape NextDrill. Lives
// here so types.go stays free of analysis-package imports (mirrors
// habitRowToBinding's pattern).
func drillToBinding(d analysis.Drill) NextDrill {
	chips := d.Chips
	if chips == nil {
		chips = []string{}
	}
	return NextDrill{
		Key:      string(d.Key),
		Title:    d.Title,
		Why:      d.Why,
		Duration: d.Duration,
		Chips:    chips,
	}
}

// GetCoachingReport returns the player's coaching landing surface — six
// micro-habit cards aggregated across the last `lookback` analyzed demos plus
// the per-kind mistake taxonomy strip. Empty steamID and "no demos for this
// player" both return the empty report so the route can render an empty state
// without try/catching the call.
func (a *App) GetCoachingReport(steamID string, lookback int) (CoachingReport, error) {
	if lookback <= 0 {
		lookback = 8
	}
	out := CoachingReport{
		SteamID:  steamID,
		Lookback: lookback,
		Habits:   []CoachingHabitRow{},
		Errors:   []MistakeKindCount{},
	}
	if steamID == "" {
		return out, nil
	}

	history, err := analysis.LoadHabitHistory(a.ctx, a.queries, steamID, lookback)
	if err != nil {
		return CoachingReport{}, fmt.Errorf("loading habit history: %w", err)
	}

	errors, err := analysis.LoadMistakeKindCounts(a.ctx, a.queries, steamID, lookback)
	if err != nil {
		return CoachingReport{}, fmt.Errorf("loading mistake kind counts: %w", err)
	}

	report := analysis.BuildCoachingReport(steamID, lookback, history, errors)

	out.Habits = make([]CoachingHabitRow, len(report.Habits))
	for i, r := range report.Habits {
		out.Habits[i] = coachingHabitRowToBinding(r)
	}
	out.Errors = make([]MistakeKindCount, len(report.Errors))
	for i, e := range report.Errors {
		out.Errors[i] = MistakeKindCount{Kind: e.Kind, Total: e.Total}
	}
	if report.LatestDemoID != 0 {
		out.LatestDemoID = strconv.FormatInt(report.LatestDemoID, 10)
	}
	out.LastDemoAt = report.LastDemoAt
	return out, nil
}

// coachingHabitRowToBinding flattens analysis.CoachingHabitRow to the
// wire-shape CoachingHabitRow, mapping the trend points to wire HistoryPoint.
func coachingHabitRowToBinding(r analysis.CoachingHabitRow) CoachingHabitRow {
	trend := make([]HistoryPoint, len(r.Trend))
	for i, p := range r.Trend {
		trend[i] = HistoryPoint{
			DemoID:    strconv.FormatInt(p.DemoID, 10),
			MatchDate: p.MatchDate,
			Value:     p.Value,
		}
	}
	return CoachingHabitRow{
		HabitRow: habitRowToBinding(r.HabitRow),
		Trend:    trend,
	}
}

// habitRowToBinding flattens the typed enums to wire-friendly strings. Living
// here (not in types.go) keeps types.go free of analysis-package imports.
func habitRowToBinding(r analysis.HabitRow) HabitRow {
	return HabitRow{
		Key:           string(r.Key),
		Label:         r.Label,
		Description:   r.Description,
		Unit:          r.Unit,
		Direction:     string(r.Direction),
		Value:         r.Value,
		Status:        string(r.Status),
		GoodThreshold: r.GoodThreshold,
		WarnThreshold: r.WarnThreshold,
		GoodMin:       r.GoodMin,
		GoodMax:       r.GoodMax,
		WarnMin:       r.WarnMin,
		WarnMax:       r.WarnMax,
		PreviousValue: r.PreviousValue,
		Delta:         r.Delta,
	}
}

// GetPlayerRoundAnalysis returns the per-(demo, player) round breakdown
// written by the round-level analyzer, ordered by round_number ASC. Unknown
// demos return an error; an empty steamID returns an empty slice (mirrors
// GetMistakeTimeline's contract — the analysis page's bar chart renders an
// empty state for the empty case rather than treating it as a fetch error).
func (a *App) GetPlayerRoundAnalysis(demoID, steamID string) ([]PlayerRoundEntry, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}
	if steamID == "" {
		return []PlayerRoundEntry{}, nil
	}

	rows, err := a.queries.GetPlayerRoundAnalysisByDemoAndPlayer(a.ctx, store.GetPlayerRoundAnalysisByDemoAndPlayerParams{
		DemoID:  id,
		SteamID: steamID,
	})
	if err != nil {
		return nil, fmt.Errorf("getting player round analysis: %w", err)
	}

	result := make([]PlayerRoundEntry, len(rows))
	for i, r := range rows {
		entry := PlayerRoundEntry{
			SteamID:     r.SteamID,
			RoundNumber: int(r.RoundNumber),
			TradePct:    r.TradePct,
			BuyType:     r.BuyType,
			MoneySpent:  int(r.MoneySpent),
			NadesUsed:   int(r.NadesUsed),
			NadesUnused: int(r.NadesUnused),
			ShotsFired:  int(r.ShotsFired),
			ShotsHit:    int(r.ShotsHit),
		}
		if r.ExtrasJson != "" && r.ExtrasJson != "{}" {
			extras := map[string]any{}
			if err := json.Unmarshal([]byte(r.ExtrasJson), &extras); err == nil {
				entry.Extras = extras
			}
		}
		result[i] = entry
	}
	return result, nil
}

// GetAnalysisStatus reports whether mechanical-analysis rows exist for the
// given demo. The viewer's mistake-list panel uses the result to decide
// between rendering the populated slice-5 surface, a shimmer placeholder, or
// nothing at all. See types.AnalysisStatus for the enum.
//
// Logic: if the demo's lifecycle status is "imported", "parsing", or "failed",
// surface that status verbatim — the demo isn't analyzable yet. If the demo
// is "ready" but no player_match_analysis rows exist for it, return "missing"
// (the legacy-import case). Otherwise return "ready".
func (a *App) GetAnalysisStatus(demoID string) (AnalysisStatus, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return AnalysisStatus{}, fmt.Errorf("invalid demo id: %w", err)
	}

	d, err := a.queries.GetDemoByID(a.ctx, id)
	if err != nil {
		return AnalysisStatus{}, fmt.Errorf("getting demo: %w", err)
	}

	if d.Status != "ready" {
		return AnalysisStatus{DemoID: demoID, Status: d.Status}, nil
	}

	// Slice 5 writes one row per rostered player on every successful parse, so
	// 0 ↔ analyzer never ran for this demo. If a future analyzer ever skips
	// rostered players, revisit this sentinel.
	count, err := a.queries.CountPlayerMatchAnalysisByDemoID(a.ctx, id)
	if err != nil {
		return AnalysisStatus{}, fmt.Errorf("counting analysis rows: %w", err)
	}
	if count == 0 {
		return AnalysisStatus{DemoID: demoID, Status: "missing"}, nil
	}
	return AnalysisStatus{DemoID: demoID, Status: "ready"}, nil
}

// RecomputeAnalysis re-runs the full parse-and-analyze pipeline for an already-
// imported demo. Used by the viewer panel to backfill mechanical-analysis rows
// for demos imported before slice 1 landed.
//
// We re-run the full parseDemo (not a "skip ingest, only analyzer" optimization)
// because parseDemo is already idempotent (delete-then-insert in the ingester
// and analysis.Persist), the surface area is one binding, and the existing
// demo:parse:progress event stream covers the deferred-progress UX. The cost
// (~10–30 s on a 100 MB demo to re-ingest tick data) is acceptable for a
// one-shot legacy backfill.
//
// Synchronous: returns after parseDemo finishes so the frontend mutation's
// isPending reflects the actual recompute. Concurrent calls block on parseMu.
func (a *App) RecomputeAnalysis(demoID string) error {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid demo id: %w", err)
	}
	d, err := a.queries.GetDemoByID(a.ctx, id)
	if err != nil {
		return fmt.Errorf("getting demo: %w", err)
	}
	if d.FilePath == "" {
		return fmt.Errorf("demo %d has no file path", id)
	}
	if _, err := os.Stat(d.FilePath); err != nil {
		return fmt.Errorf("demo file unavailable: %w", err)
	}
	a.runParseSerialized(d.ID, d.FilePath)
	return nil
}

// GetMatchInsights returns the team-level summary surfaced on the standalone
// analysis page. Aggregates every player_match_analysis row for the demo into
// per-side averages plus a small list of standout players. Unknown demos
// return a zero-valued MatchInsights (not an error) so the page renders an
// empty-state surface, mirroring the read-only-binding convention in this
// file.
func (a *App) GetMatchInsights(demoID string) (*MatchInsights, error) {
	id, err := strconv.ParseInt(demoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid demo id: %w", err)
	}
	rows, err := a.queries.ListPlayerMatchAnalysisByDemoID(a.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("listing match analysis: %w", err)
	}
	out := &MatchInsights{
		DemoID:    demoID,
		CTSummary: TeamSummary{Side: "CT"},
		TSummary:  TeamSummary{Side: "T"},
	}
	if len(rows) == 0 {
		return out, nil
	}

	// Resolve each player's side via the most recent round's roster — the
	// demo's "team" assignment can flip at halftime; we report the side they
	// played most rounds on (good enough for the standout list).
	sideBySteam := mostFrequentSideByPlayer(a.ctx, a.queries, id)

	type accum struct {
		players         int
		overallScore    float64
		tradePct        float64
		standingShot    float64
		counterStrafe   float64
		firstShot       float64
		flashAssists    int
		smokesKill      int
		heDamage        int
		isolatedDeaths  int
		ecoKills        int
		fullBuyADRSum   float64
		fullBuyADRCount int
	}
	ct := &accum{}
	t := &accum{}

	for _, r := range rows {
		side := sideBySteam[r.SteamID]
		var a *accum
		switch side {
		case "CT":
			a = ct
		case "T":
			a = t
		default:
			continue
		}
		a.players++
		a.overallScore += float64(r.OverallScore)
		a.tradePct += r.TradePct
		a.standingShot += r.StandingShotPct
		a.counterStrafe += r.CounterStrafePct
		a.firstShot += r.FirstShotAccPct
		a.flashAssists += int(r.FlashAssists)
		a.smokesKill += int(r.SmokesKillAssist)
		a.heDamage += int(r.HeDamage)
		a.isolatedDeaths += int(r.IsolatedPeekDeaths)
		a.ecoKills += int(r.EcoKills)
		if r.FullBuyAdr > 0 {
			a.fullBuyADRSum += r.FullBuyAdr
			a.fullBuyADRCount++
		}
	}
	finalize := func(side string, a *accum) TeamSummary {
		s := TeamSummary{Side: side, Players: a.players}
		if a.players > 0 {
			s.AvgOverallScore = a.overallScore / float64(a.players)
			s.AvgTradePct = a.tradePct / float64(a.players)
			s.AvgStandingShot = a.standingShot / float64(a.players)
			s.AvgCounterStrafe = a.counterStrafe / float64(a.players)
			s.AvgFirstShot = a.firstShot / float64(a.players)
		}
		if a.fullBuyADRCount > 0 {
			s.AvgFullBuyADR = a.fullBuyADRSum / float64(a.fullBuyADRCount)
		}
		s.TotalFlashAssist = a.flashAssists
		s.TotalSmokesKA = a.smokesKill
		s.TotalHeDamage = a.heDamage
		s.TotalIsolated = a.isolatedDeaths
		s.TotalEcoKills = a.ecoKills
		return s
	}
	out.CTSummary = finalize("CT", ct)
	out.TSummary = finalize("T", t)
	out.Standouts = pickStandouts(rows)
	return out, nil
}

// mostFrequentSideByPlayer returns a (steamID → "CT" | "T") map by counting
// each player's appearances across every round_loadouts row for the demo.
// Halftime swaps land each player on whichever side has more appearances —
// fine for a coarse "this player belongs to side X" classifier.
func mostFrequentSideByPlayer(ctx context.Context, q *store.Queries, demoID int64) map[string]string {
	out := make(map[string]string, 16)
	rows, err := q.GetRostersByDemoID(ctx, demoID)
	if err != nil {
		return out
	}
	type counts struct{ ct, t int }
	byPlayer := make(map[string]*counts, 16)
	for _, r := range rows {
		c, ok := byPlayer[r.SteamID]
		if !ok {
			c = &counts{}
			byPlayer[r.SteamID] = c
		}
		switch r.TeamSide {
		case "CT":
			c.ct++
		case "T":
			c.t++
		}
	}
	for steam, c := range byPlayer {
		if c.ct >= c.t {
			out[steam] = "CT"
		} else {
			out[steam] = "T"
		}
	}
	return out
}

// pickStandouts selects the highest-value player per category from the
// supplied analysis rows. Ties resolve to the row with the lower steam_id —
// stable across runs.
func pickStandouts(rows []store.PlayerMatchAnalysis) []PlayerHighlight {
	if len(rows) == 0 {
		return nil
	}
	type pick struct {
		category   string
		metricName string
		val        float64
		steam      string
	}
	max := func(category, metric string, val float64, steam string, cur *pick) {
		if cur.steam == "" || val > cur.val || (val == cur.val && steam < cur.steam) {
			*cur = pick{category, metric, val, steam}
		}
	}
	overall := pick{category: "overall"}
	trade := pick{category: "trade"}
	aim := pick{category: "aim"}
	utility := pick{category: "utility"}
	movement := pick{category: "movement"}
	for _, r := range rows {
		max("overall", "overall_score", float64(r.OverallScore), r.SteamID, &overall)
		max("trade", "trade_pct", r.TradePct, r.SteamID, &trade)
		max("aim", "first_shot_acc_pct", r.FirstShotAccPct, r.SteamID, &aim)
		utilityScore := float64(r.FlashAssists) + float64(r.SmokesKillAssist)*2
		max("utility", "utility_kill_score", utilityScore, r.SteamID, &utility)
		max("movement", "counter_strafe_pct", r.CounterStrafePct, r.SteamID, &movement)
	}
	out := make([]PlayerHighlight, 0, 5)
	for _, p := range []pick{overall, trade, aim, utility, movement} {
		if p.steam == "" {
			continue
		}
		out = append(out, PlayerHighlight{
			SteamID:    p.steam,
			Category:   p.category,
			MetricName: p.metricName,
			MetricVal:  p.val,
		})
	}
	return out
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

func storeContactToBinding(r store.ContactMoment, roundNumber int) (ContactMoment, error) {
	cm := ContactMoment{
		ID:           r.ID,
		DemoID:       r.DemoID,
		RoundID:      r.RoundID,
		RoundNumber:  roundNumber,
		SubjectSteam: r.SubjectSteam,
		TFirst:       int32(r.TFirst),
		TLast:        int32(r.TLast),
		TPre:         int32(r.TPre),
		TPost:        int32(r.TPost),
		Outcome:      ContactOutcome(r.Outcome),
		SignalCount:  int(r.SignalCount),
	}
	if r.EnemiesJson != "" && r.EnemiesJson != "[]" {
		if err := json.Unmarshal([]byte(r.EnemiesJson), &cm.Enemies); err != nil {
			return ContactMoment{}, fmt.Errorf("decoding enemies: %w", err)
		}
	}
	if r.ExtrasJson != "" && r.ExtrasJson != "{}" {
		cm.Extras = map[string]any{}
		if err := json.Unmarshal([]byte(r.ExtrasJson), &cm.Extras); err != nil {
			// Mirror GetMistakeTimeline's lenient extras decode — a malformed
			// blob yields empty extras instead of failing the row.
			cm.Extras = map[string]any{}
		}
	}
	return cm, nil
}

func storeContactMistakeToBinding(r store.ContactMistake) ContactMistake {
	cm := ContactMistake{
		Kind:     r.Kind,
		Category: r.Category,
		Severity: int(r.Severity),
		Phase:    r.Phase,
	}
	if r.Tick.Valid {
		v := int32(r.Tick.Int64)
		cm.Tick = &v
	}
	if r.ExtrasJson != "" && r.ExtrasJson != "{}" {
		extras := map[string]any{}
		if err := json.Unmarshal([]byte(r.ExtrasJson), &extras); err == nil {
			cm.Extras = extras
		}
	}
	return cm
}

func storeTickDatumToBinding(d store.TickDatum) TickData {
	td := TickData{
		Tick:        int(d.Tick),
		SteamID:     d.SteamID,
		X:           roundToInt16(d.X),
		Y:           roundToInt16(d.Y),
		Z:           roundToInt16(d.Z),
		Yaw:         roundToInt16(d.Yaw),
		Pitch:       roundToInt16(d.Pitch),
		Crouch:      d.Crouch != 0,
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
