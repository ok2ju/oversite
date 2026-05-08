package demo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"time"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

// Defensive limits guarding against runaway slice growth from corrupt or
// pathological demos. With IgnorePacketEntitiesPanic = true the parser keeps
// running past entity-state damage that previously aborted parsing fast,
// which has been observed to drive tick/event accumulation into the millions
// and OOM the host. Hitting these caps fails the parse with a clear error
// rather than letting the OS kill the app (or the whole machine, on swap).
//
// At 200 B/row a 5M-row tick slice is ~1 GB; 100K events at ~250 B is ~25 MB.
// A 90-min match samples roughly 800K tick rows and emits ~30-50K events, so
// these are ~6× and ~2-3× headroom over the worst legitimate case. The
// previous 500K event cap was set when entity-panic recovery let runaway
// demos accumulate millions of events; with the heap watchdog catching
// pathological cases earlier we don't need that headroom and a tighter cap
// fails earlier on corrupt demos.
const (
	maxTickRows  = 5_000_000
	maxGameEvent = 100_000
)

// defaultMaxHeapBytes is the fallback heap ceiling used when the caller didn't
// supply one via WithHeapLimit. The watchdog defends against runaway entity-
// table growth on corrupt demos (the demoinfocs parser buffers per-frame
// protobuf messages and accumulates entity state outside our own slice caps)
// — tripping it is fatal because the partial output may reference dropped
// entities. Production callers should size this from host RAM via
// internal/sysinfo so a 16 GB Windows machine fails fast instead of paging
// the OS into a freeze; this constant exists only so direct DemoParser users
// (tests, CLI tools) don't need to do that wiring themselves.
const defaultMaxHeapBytes uint64 = 4 << 30

// heartbeatFrameInterval is how often the FrameDone handler logs a
// progress/memory line and checks the heap ceiling. ~10K frames is roughly
// every 2-3 seconds of demo time at 64 tick.
const heartbeatFrameInterval = 10_000

// ErrTickLimitExceeded indicates the parser's tick accumulator hit maxTickRows
// before the demo finished. The demo is likely corrupt or non-standard.
var ErrTickLimitExceeded = fmt.Errorf("tick row limit exceeded (%d): demo may be corrupt", maxTickRows)

// ErrEventLimitExceeded indicates the parser's event accumulator hit
// maxGameEvent before the demo finished.
var ErrEventLimitExceeded = fmt.Errorf("game event limit exceeded (%d): demo may be corrupt", maxGameEvent)

// ErrHeapLimitExceeded indicates the parser tripped its configured heap
// ceiling. Returned to spare the host OS from paging itself into a freeze.
// The configured ceiling (in MiB) is included in the message so users can tell
// from a stale errors.txt which budget was in effect.
var ErrHeapLimitExceeded = errors.New("heap allocation limit exceeded: demo may be corrupt")

// ErrCorruptEntityTable indicates the underlying demoinfocs library hit an
// "unable to find existing entity" panic — i.e. the demo's entity table is
// damaged. Surfaced only when the parser was constructed without
// WithIgnoreEntityPanics; with that option set, demoinfocs swallows the panic
// and continues parsing, which has been observed to drive runaway memory
// growth on pathological demos. See parser.go for the trade-off.
//
// Two emission paths converge here: a Go panic that escapes the dispatcher
// (caught by Parse's deferred recover) and an error returned from ParseToEnd
// after demoinfocs's own PanicHandler swallowed the panic — both wrap with
// %w so the caller's errors.Is(...) auto-retry check matches.
var ErrCorruptEntityTable = errors.New("demo has a corrupt entity table; parsing was stopped to avoid running out of memory")

// ParseResult is the complete output of parsing a demo file.
//
// Ticks is nil when WithTickSink was passed to NewDemoParser — in the
// streaming pipeline the snapshots are flushed through the channel during
// parsing and never accumulated in this slice.
type ParseResult struct {
	Header  MatchHeader
	Rounds  []RoundData
	Ticks   []TickSnapshot
	Events  []GameEvent
	Lineups []GrenadeLineup
}

// MatchHeader contains match-level metadata.
type MatchHeader struct {
	MapName      string
	TickRate     float64
	TotalTicks   int
	DurationSecs int
}

// RoundData contains data for a single round.
type RoundData struct {
	Number        int
	StartTick     int
	FreezeEndTick int // tick at which freeze time ends and the round goes live; 0 if unknown
	EndTick       int
	WinnerSide    string // "CT" or "T"
	WinReason     string
	CTScore       int
	TScore        int
	IsOvertime    bool
	CTTeamName    string // clan tag for the CT team (e.g. "Astralis"); empty for matchmaking demos
	TTeamName     string // clan tag for the T team (e.g. "NaVi"); empty for matchmaking demos
	Roster        []RoundParticipant
}

// RoundParticipant is one alive player captured at the end of freeze time.
// Used to seed per-round stats so passive players (no kills, no damage, no
// deaths) still receive a player_rounds row instead of falling back to the
// numeric SteamID slice on the frontend.
//
// Inventory is the comma-separated weapon list at freeze-end (encodeInventory
// output). Persisted into round_loadouts (migration 011) so the team bars can
// render the player's purchased loadout without paying for tick-rate inventory
// snapshots — mid-round pickups/drops are intentionally not tracked.
type RoundParticipant struct {
	SteamID    string
	PlayerName string
	TeamSide   string // "CT" or "T"
	Inventory  string
}

// TickSnapshot is one player's state at a sampled tick.
//
// Inventory is intentionally not part of this struct: the freeze-end loadout
// covers the team-bars use case at ~5000× fewer rows. See RoundParticipant
// and migration 011.
type TickSnapshot struct {
	Tick       int
	SteamID    string // Steam64ID as string
	X, Y, Z    float64
	Yaw        float64
	Health     int
	Armor      int
	IsAlive    bool
	Weapon     string
	Money      int
	HasHelmet  bool
	HasDefuser bool
	// AmmoClip / AmmoReserve are the active weapon's clip and reserve counts.
	// Both 0 when the active item has no ammo (e.g. knife) or no active weapon.
	AmmoClip    int
	AmmoReserve int
}

// GameEvent represents a parsed game event (kill, grenade, bomb, round boundary).
//
// ExtraData holds a typed pointer-to-struct from extras.go (e.g. *KillExtra,
// *WeaponFireExtra). Using `any` rather than the EventExtra marker interface
// keeps json.Marshal happy without forcing every reader to call a getter.
// nil indicates "no extras" — JSON marshals it as null, same as the old empty
// map case.
type GameEvent struct {
	Tick            int
	RoundNumber     int
	Type            string // "kill", "weapon_fire", "player_hurt", "grenade_throw", "grenade_bounce", "grenade_detonate", "smoke_start", "smoke_expired", "decoy_start", "fire_start", "bomb_plant", "bomb_defuse", "bomb_explode"
	AttackerSteamID string
	VictimSteamID   string
	Weapon          string
	X, Y, Z         float64
	ExtraData       any
}

// ProgressFunc is called during parsing to report progress.
// stage is a human-readable label (e.g. "parsing"), percent is 0-100.
type ProgressFunc func(stage string, percent float64)

// Option configures the DemoParser.
type Option func(*DemoParser)

// WithTickInterval sets the tick sampling interval (default: 4).
// Values <= 0 are ignored.
func WithTickInterval(n int) Option {
	return func(dp *DemoParser) {
		if n > 0 {
			dp.tickInterval = n
		}
	}
}

// WithSkipWarmup controls whether warmup rounds are skipped (default: true).
func WithSkipWarmup(skip bool) Option {
	return func(dp *DemoParser) {
		dp.skipWarmup = skip
	}
}

// WithIncludeBots controls whether bot players are included (default: false).
func WithIncludeBots(include bool) Option {
	return func(dp *DemoParser) {
		dp.includeBots = include
	}
}

// WithProgressFunc sets a callback for parsing progress updates.
func WithProgressFunc(fn ProgressFunc) Option {
	return func(dp *DemoParser) {
		dp.progressFunc = fn
	}
}

// WithTickSink configures the parser to push each captured TickSnapshot to the
// supplied channel instead of accumulating them into ParseResult.Ticks. The
// caller owns reading from the channel; Parse closes it exactly once when it
// returns (success or error). When a sink is set, ParseResult.Ticks is nil.
//
// This enables overlapping parse (CPU-bound on protobuf decode) with ingest
// (I/O-bound on SQLite WAL writes) and caps peak heap by removing the
// 100 MB+ tick slice that would otherwise live until ingestion finishes.
func WithTickSink(sink chan<- TickSnapshot) Option {
	return func(dp *DemoParser) {
		dp.tickSink = sink
	}
}

// WithHeapLimit overrides the heartbeat watchdog's heap ceiling (bytes). When
// HeapAlloc exceeds this value the parse aborts with ErrHeapLimitExceeded.
// Values <= 0 are ignored, so callers can pass a computed sysinfo budget
// without an extra branch.
func WithHeapLimit(bytes uint64) Option {
	return func(dp *DemoParser) {
		if bytes > 0 {
			dp.maxHeapBytes = bytes
		}
	}
}

// WithIgnoreEntityPanics controls whether the underlying demoinfocs parser
// swallows "unable to find existing entity" panics and continues. Default
// is false: such panics are surfaced as ErrCorruptEntityTable so the import
// fails fast instead of accumulating runaway state. Set to true to attempt
// a partial parse on demos with damaged entity tables — at the risk of
// driving the parser past its heap ceiling.
func WithIgnoreEntityPanics(ignore bool) Option {
	return func(dp *DemoParser) {
		dp.ignoreEntityPanics = ignore
	}
}

// WithProfilesDir sets the directory where the heap watchdog writes pprof
// dumps when it trips. Empty disables pprof output (the watchdog still trips
// and aborts the parse, just without a dump for triage).
func WithProfilesDir(dir string) Option {
	return func(dp *DemoParser) {
		dp.profilesDir = dir
	}
}

// DemoParser extracts structured data from CS2 .dem files.
type DemoParser struct {
	tickInterval       int
	skipWarmup         bool
	includeBots        bool
	progressFunc       ProgressFunc
	tickSink           chan<- TickSnapshot
	maxHeapBytes       uint64
	ignoreEntityPanics bool
	profilesDir        string
}

// NewDemoParser creates a parser with the given options.
func NewDemoParser(opts ...Option) *DemoParser {
	dp := &DemoParser{
		tickInterval: 4,
		skipWarmup:   true,
		includeBots:  false,
		maxHeapBytes: defaultMaxHeapBytes,
	}
	for _, opt := range opts {
		opt(dp)
	}
	return dp
}

// parseState holds mutable state tracked during parsing.
type parseState struct {
	ctx                 context.Context // cancelled by Parse's caller; checked at natural boundaries
	mapName             string
	matchStarted        bool
	matchStartCount     int
	currentRound        int
	roundStart          int
	freezeEndTick       int
	ctScore             int
	tScore              int
	maxRegulationRounds int // mp_maxrounds; 0 until the convar is observed.
	currentRoster       []RoundParticipant
	rounds              []RoundData
	ticks               []TickSnapshot      // populated only when tickSink is nil
	tickSink            chan<- TickSnapshot // when non-nil, snapshots are pushed here instead of appended to ticks
	tickCount           int                 // number of TickSnapshots produced (sink + slice combined); enforces maxTickRows
	events              []GameEvent
	lastSampledTick     int
	knifeRoundNumbers   map[int]bool
	limitExceeded       error             // set when ticks/events exceed maxTickRows/maxGameEvent
	frameCount          int               // total FrameDone events seen; drives the heartbeat
	steamIDs            map[uint64]string // SteamID64 → decimal string cache; saves millions of strconv allocs across all event handlers
}

// steamID returns the decimal string form of p.SteamID64, cached per player.
// Returns "" if p is nil. Safe to call from any handler since the demoinfocs
// dispatcher is single-threaded.
func (s *parseState) steamID(p *common.Player) string {
	if p == nil {
		return ""
	}
	if v, ok := s.steamIDs[p.SteamID64]; ok {
		return v
	}
	if s.steamIDs == nil {
		s.steamIDs = make(map[uint64]string, 16)
	}
	v := strconv.FormatUint(p.SteamID64, 10)
	s.steamIDs[p.SteamID64] = v
	return v
}

// shouldStopAppending returns true once a limit has tripped. Handlers check
// this before each append so we don't keep growing slices after Cancel() has
// been called (Cancel takes effect at the next dispatcher pop, not synchronously).
func (s *parseState) shouldStopAppending() bool {
	return s.limitExceeded != nil
}

// addEvent appends an event subject to the maxGameEvent cap. Returns true on
// success; on cap-trip it sets s.limitExceeded and returns false so the
// caller can ask the parser to Cancel.
func (s *parseState) addEvent(ev GameEvent) bool {
	if s.limitExceeded != nil {
		return false
	}
	if len(s.events) >= maxGameEvent {
		s.limitExceeded = ErrEventLimitExceeded
		return false
	}
	s.events = append(s.events, ev)
	return true
}

// pushTick routes a TickSnapshot to either the streaming sink (when set) or
// the in-memory slice. Returns true on success; on cap-trip or ctx-cancel it
// sets s.limitExceeded and returns false so the caller can ask the parser to
// Cancel.
//
// The select in the streaming branch protects against a stalled ingester:
// if the consumer goroutine errored and stopped draining, the errgroup's
// shared ctx will already be cancelled, so we don't deadlock here.
func (s *parseState) pushTick(t TickSnapshot) bool {
	if s.limitExceeded != nil {
		return false
	}
	if s.tickCount >= maxTickRows {
		s.limitExceeded = ErrTickLimitExceeded
		return false
	}
	if s.tickSink != nil {
		select {
		case s.tickSink <- t:
		case <-s.ctx.Done():
			s.limitExceeded = s.ctx.Err()
			return false
		}
	} else {
		s.ticks = append(s.ticks, t)
	}
	s.tickCount++
	return true
}

// ensureFormat populates maxRegulationRounds from the live convar map. ConVars
// are streamed as the demo plays out and may be empty if read too early, so
// this is called lazily — once the value is known we keep it.
func (s *parseState) ensureFormat(p demoinfocs.Parser) {
	if s.maxRegulationRounds != 0 {
		return
	}
	rules := p.GameState().Rules()
	if rules == nil {
		return
	}
	cv := rules.ConVars()
	if cv == nil {
		return
	}
	v, ok := cv["mp_maxrounds"]
	if !ok {
		return
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return
	}
	s.maxRegulationRounds = n
}

// resetForPreMatchRestart discards all captured data from the pre-match phase.
// Called when MatchStartedChanged(true) re-fires with the match score still at
// 0-0, which signals a Faceit-style knife-round → live-match transition.
//
// When streaming via tickSink, snapshots already pushed to the channel cannot
// be pulled back — the ingester may have committed them. tickCount is reset
// to 0 so the maxTickRows cap counts only post-restart frames, but the
// DB-side cleanup is the responsibility of the next re-import (the ingester's
// DeleteTickDataByDemoID runs first on every Ingest call). In practice the
// pre-restart phase emits only a handful of warmup ticks, so the impact is
// negligible.
func (s *parseState) resetForPreMatchRestart() {
	s.currentRound = 0
	s.roundStart = 0
	s.freezeEndTick = 0
	s.lastSampledTick = 0
	s.currentRoster = nil
	s.rounds = nil
	s.ticks = nil
	s.tickCount = 0
	s.events = nil
	s.knifeRoundNumbers = nil
	// Keep steamIDs across restart — same players, same SteamID64s.
}

// Parse reads a CS2 demo from r and returns all extracted data.
//
// ctx is honored at natural boundaries (FrameDone handler entry); when ctx
// is cancelled mid-parse, the underlying parser is asked to stop and Parse
// returns ctx.Err(). When WithTickSink was set, the sink channel is closed
// exactly once before Parse returns (success, error, panic), and
// ParseResult.Ticks is left nil; the caller is responsible for draining the
// channel concurrently.
func (dp *DemoParser) Parse(ctx context.Context, r io.Reader) (result *ParseResult, err error) {
	if dp.tickSink != nil {
		defer close(dp.tickSink)
	}
	defer func() {
		if rec := recover(); rec != nil {
			// In-handler panics (including "unable to find existing entity")
			// are caught by demoinfocs's own dispatcher PanicHandler and
			// surfaced as ParseToEnd errors — see the corresponding wrap
			// path below — so this branch only fires for panics outside the
			// dispatcher loop. The string match remains as a belt-and-braces
			// fallback for any path that does escape: better to return
			// ErrCorruptEntityTable and trigger the caller's retry than to
			// surface a raw panic message.
			recMsg := fmt.Sprintf("%v", rec)
			if strings.Contains(recMsg, "unable to find existing entity") {
				err = fmt.Errorf("%w: %s", ErrCorruptEntityTable, recMsg)
				return
			}
			err = fmt.Errorf("parsing demo: panic: %v", rec)
		}
	}()

	// IgnorePacketEntitiesPanic recovers from "unable to find existing entity"
	// panics that fire on some POV demos when an entity update references an
	// index missing from p.entities. With it on, demoinfocs swallows the
	// panic and keeps parsing — which on a pathological demo means an
	// unbounded internal accumulation loop that drove a 13 GB Windows
	// working-set blow-up before the heap watchdog could cancel the parse.
	// Default off: surface the panic as ErrCorruptEntityTable so the import
	// fails fast. Callers that need partial-parse tolerance can opt in via
	// WithIgnoreEntityPanics(true).
	//
	// IgnoreErrBombsiteIndexNotFound is the analogous flag for game events
	// that reference an unknown bombsite index — left on unconditionally
	// because it just skips a malformed bomb event and doesn't accumulate
	// state.
	config := demoinfocs.DefaultParserConfig
	config.IgnorePacketEntitiesPanic = dp.ignoreEntityPanics
	config.IgnoreErrBombsiteIndexNotFound = true
	p := demoinfocs.NewParserWithConfig(r, config)
	defer func() {
		if closeErr := p.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing parser: %w", closeErr)
		}
	}()

	state := &parseState{
		ctx:      ctx,
		tickSink: dp.tickSink,
	}

	// Independent goroutine watchdog: polls runtime.ReadMemStats every 500 ms
	// regardless of whether the FrameDone handler has yielded. Catches the
	// pre-frame phase (string tables, entity baselines, DataTable decode)
	// where the in-handler heartbeat at parser.go:797 cannot fire.
	wd := newHeapWatchdog(dp.maxHeapBytes, 500*time.Millisecond, dp.profilesDir, 0,
		func(trip error) {
			state.limitExceeded = trip
			p.Cancel()
		})
	go wd.Run()
	defer wd.Stop()

	dp.registerHandlers(p, state)

	dp.reportProgress("parsing", 0)

	if parseErr := p.ParseToEnd(); parseErr != nil {
		// "unable to find existing entity" panics from sendtablescs2 are
		// caught by demoinfocs's own dispatcher PanicHandler (parser.go:551
		// in v5.1.2) when IgnorePacketEntitiesPanic is false and surfaced as
		// an error here, NOT as a Go panic — so the recover() above never
		// fires for them. Wrap as ErrCorruptEntityTable so the caller's
		// auto-retry path (app.go runParsePipeline) matches via errors.Is
		// and re-runs with entity-panic tolerance enabled.
		if strings.Contains(parseErr.Error(), "unable to find existing entity") {
			return nil, fmt.Errorf("%w: %v", ErrCorruptEntityTable, parseErr)
		}
		// demoinfocs returns ErrUnexpectedEndOfDemo for truncated demos;
		// treat as fatal only if we got zero data.
		if len(state.rounds) == 0 && state.tickCount == 0 && len(state.events) == 0 {
			return nil, fmt.Errorf("parsing demo: %w", parseErr)
		}
	}

	// A tripped tick/event cap (or ctx cancel) is fatal even if we collected
	// partial data — the demo is corrupt or non-standard, and the partial
	// slices may reference dropped entities. Better to surface a clear error
	// than show the user a half-broken viewer.
	if state.limitExceeded != nil {
		return nil, state.limitExceeded
	}

	dp.reportProgress("parsing", 90)

	tickRate := p.TickRate()
	totalTicks := p.GameState().IngameTick()

	var durationSecs int
	if tickRate > 0 {
		durationSecs = int(float64(totalTicks) / tickRate)
	}

	rounds, events := dropKnifeRounds(state.rounds, state.events, state.knifeRoundNumbers)

	pairShotsWithImpacts(events)

	lineups := ExtractGrenadeLineups(state.mapName, events)

	slog.Info("parser: completed",
		"rounds", len(rounds),
		"ticks", state.tickCount,
		"events", len(events),
		"map", state.mapName,
		"total_ticks", totalTicks)

	dp.reportProgress("parsing", 100)

	return &ParseResult{
		Header: MatchHeader{
			MapName:      state.mapName,
			TickRate:     tickRate,
			TotalTicks:   totalTicks,
			DurationSecs: durationSecs,
		},
		Rounds:  rounds,
		Ticks:   state.ticks,
		Events:  events,
		Lineups: lineups,
	}, nil
}

// dropKnifeRounds removes rounds flagged as knife rounds from the parsed data.
// Rounds are flagged by two complementary signals (see registerHandlers):
//
//  1. A post-start MatchStartedChanged(true) event with scores still 0-0
//     triggers a full state reset — those rounds never reach here.
//  2. At each freeze-time end, if every live player's inventory is a knife,
//     the round number is recorded in the flagged map. This is the fallback
//     when the restart event does not fire.
//
// Remaining rounds are renumbered contiguously from 1, events are renumbered to
// match, and scores are adjusted to cancel out the dropped knife-round wins.
// The IsOvertime flag captured at RoundEnd survives renumbering unchanged —
// it's sourced from the live game state, not derived from the round number.
func dropKnifeRounds(rounds []RoundData, events []GameEvent, flagged map[int]bool) ([]RoundData, []GameEvent) {
	if len(rounds) == 0 || len(flagged) == 0 {
		return rounds, events
	}

	renumber := make(map[int]int, len(rounds))
	ctOffset, tOffset := 0, 0
	newNumber := 0
	filteredRounds := make([]RoundData, 0, len(rounds))
	for _, rd := range rounds {
		if flagged[rd.Number] {
			switch rd.WinnerSide {
			case "CT":
				ctOffset++
			case "T":
				tOffset++
			}
			continue
		}
		newNumber++
		renumber[rd.Number] = newNumber
		rd.Number = newNumber
		rd.CTScore -= ctOffset
		rd.TScore -= tOffset
		if rd.CTScore < 0 {
			rd.CTScore = 0
		}
		if rd.TScore < 0 {
			rd.TScore = 0
		}
		filteredRounds = append(filteredRounds, rd)
	}

	filteredEvents := make([]GameEvent, 0, len(events))
	for _, ev := range events {
		if flagged[ev.RoundNumber] {
			continue
		}
		if n, ok := renumber[ev.RoundNumber]; ok {
			ev.RoundNumber = n
		}
		filteredEvents = append(filteredEvents, ev)
	}

	return filteredRounds, filteredEvents
}

// isKnifeRoundByInventory returns true if every provided inventory is exactly
// {EqKnife}. Callers must pre-filter to alive participants — a dead player has
// an empty Weapons() slice that would otherwise produce false negatives.
//
// Faceit knife rounds force knife-only loadouts via mp_startmoney=0 + empty
// default secondaries; the inventory check at freeze-end catches them even
// when the MatchStartedChanged restart signal is absent. The minimum-sample
// guard (>= 8) avoids high-variance flagging on the rare frames where only
// 1–2 players are in Playing() during reconnects.
func isKnifeRoundByInventory(inventories [][]common.EquipmentType) bool {
	if len(inventories) < 8 {
		return false
	}
	for _, inv := range inventories {
		if len(inv) != 1 || inv[0] != common.EqKnife {
			return false
		}
	}
	return true
}

func (dp *DemoParser) reportProgress(stage string, percent float64) {
	if dp.progressFunc != nil {
		dp.progressFunc(stage, percent)
	}
}

func (dp *DemoParser) registerHandlers(p demoinfocs.Parser, state *parseState) {
	// Map name from demo file header.
	p.RegisterNetMessageHandler(func(m *msg.CDemoFileHeader) {
		state.mapName = m.GetMapName()
	})

	// Match start tracking. A post-start MatchStartedChanged(true) re-fire with
	// the match score still at 0-0 is the canonical Faceit signal that the knife
	// round (and any warmup that ran after it) was pre-match noise — discard the
	// captured data and let the live match start populating state from scratch.
	// A re-fire with non-zero scores is an admin mp_restartgame mid-match; we
	// keep the captured data and log for manual inspection.
	p.RegisterEventHandler(func(e events.MatchStartedChanged) {
		state.matchStarted = e.NewIsStarted
		if !e.NewIsStarted {
			return
		}
		state.matchStartCount++
		if state.matchStartCount == 1 {
			return
		}
		tick := p.GameState().IngameTick()
		if state.ctScore == 0 && state.tScore == 0 {
			slog.Info("pre-match restart detected; discarding captured data",
				"tick", tick, "count", state.matchStartCount)
			state.resetForPreMatchRestart()
			return
		}
		slog.Warn("mid-match restart detected; keeping captured data",
			"tick", tick, "count", state.matchStartCount,
			"ct", state.ctScore, "t", state.tScore)
	})

	// Round start.
	p.RegisterEventHandler(func(e events.RoundStart) {
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}
		state.currentRound++
		state.roundStart = p.GameState().IngameTick()
		state.freezeEndTick = 0
		state.currentRoster = nil
	})

	// Freeze time end — the round goes live. Capture the tick once per round
	// (ignore duplicate/late events that would push it past the true end), then
	// snapshot the alive player roster (used to seed per-round stats so passive
	// players still get a player_rounds row) and inventories (used to flag knife
	// rounds for later drop in dropKnifeRounds).
	captureFreezeEnd := func() {
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}
		if state.freezeEndTick != 0 {
			return
		}
		state.freezeEndTick = p.GameState().IngameTick()

		inventories := make([][]common.EquipmentType, 0, 10)
		roster := make([]RoundParticipant, 0, 10)
		for _, player := range p.GameState().Participants().Playing() {
			if shouldSkipPlayer(player, dp.includeBots) {
				continue
			}
			if !player.IsAlive() {
				continue
			}
			weapons := player.Weapons()
			roster = append(roster, RoundParticipant{
				SteamID:    state.steamID(player),
				PlayerName: player.Name,
				TeamSide:   teamSideString(player.Team),
				Inventory:  encodeInventory(weapons),
			})
			inv := make([]common.EquipmentType, 0, len(weapons))
			for _, w := range weapons {
				if w == nil {
					continue
				}
				inv = append(inv, w.Type)
			}
			inventories = append(inventories, inv)
		}
		state.currentRoster = roster
		if isKnifeRoundByInventory(inventories) {
			if state.knifeRoundNumbers == nil {
				state.knifeRoundNumbers = make(map[int]bool)
			}
			state.knifeRoundNumbers[state.currentRound] = true
		}
	}

	// Classical Source1 game event path (round_freeze_end).
	p.RegisterEventHandler(func(e events.RoundFreezetimeEnd) {
		captureFreezeEnd()
	})

	// Property-based path (m_bFreezePeriod going false). Fires for CS2 demos
	// where the legacy game event may be absent or mis-ordered.
	p.RegisterEventHandler(func(e events.RoundFreezetimeChanged) {
		if !e.NewIsFreezetime {
			captureFreezeEnd()
		}
	})

	// Round end.
	p.RegisterEventHandler(func(e events.RoundEnd) {
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}
		if state.currentRound == 0 {
			return
		}

		// ScoreUpdated fires before RoundEnd in v5 and is the only authoritative
		// per-team score path; e.WinnerState.Score() is documented as not-up-to-
		// date here. The library docs and v5 behavior contradict each other on
		// the WinnerState read, so we don't read it. If ScoreUpdated has not yet
		// fired (rare; malformed demos), increment from the prior captured state
		// to keep the score monotonic and log a warn so the demo surfaces.
		gs := p.GameState()
		state.ensureFormat(p)
		expectedTotal := state.currentRound
		actualTotal := state.ctScore + state.tScore
		if actualTotal < expectedTotal && actualTotal == expectedTotal-1 {
			switch e.Winner {
			case common.TeamCounterTerrorists:
				state.ctScore++
			case common.TeamTerrorists:
				state.tScore++
			default:
				slog.Warn("round end without preceding ScoreUpdated; cannot increment",
					"round", state.currentRound, "ct", state.ctScore, "t", state.tScore)
			}
			slog.Warn("round end with stale score; incremented from winner",
				"round", state.currentRound, "ct", state.ctScore, "t", state.tScore)
		}

		ctClan := ""
		tClan := ""
		if cts := gs.TeamCounterTerrorists(); cts != nil {
			ctClan = cts.ClanName()
		}
		if ts := gs.TeamTerrorists(); ts != nil {
			tClan = ts.ClanName()
		}

		isOT := gs.OvertimeCount() > 0

		// Cross-check format invariant: when the format is known and we're
		// supposedly in regulation, the round number must not exceed
		// mp_maxrounds. Log only — don't crash on malformed demos.
		if !isOT && state.maxRegulationRounds > 0 && state.currentRound > state.maxRegulationRounds {
			slog.Warn("round number exceeds mp_maxrounds but OvertimeCount is 0",
				"round", state.currentRound, "max_rounds", state.maxRegulationRounds)
		}

		state.rounds = append(state.rounds, RoundData{
			Number:        state.currentRound,
			StartTick:     state.roundStart,
			FreezeEndTick: state.freezeEndTick,
			EndTick:       gs.IngameTick(),
			WinnerSide:    teamSideString(e.Winner),
			WinReason:     roundEndReasonString(e.Reason),
			CTScore:       state.ctScore,
			TScore:        state.tScore,
			IsOvertime:    isOT,
			CTTeamName:    ctClan,
			TTeamName:     tClan,
			Roster:        state.currentRoster,
		})

		// Report progress proportional to rounds parsed.
		// Rough estimate: most Faceit demos have 24-30 rounds.
		pct := float64(state.currentRound) / 30.0 * 80.0
		if pct > 80 {
			pct = 80
		}
		dp.reportProgress("parsing", pct)
	})

	// Score tracking for accuracy. In demoinfocs v5, ScoreUpdated fires
	// before RoundEnd, so this keeps state.ctScore/tScore in sync with the
	// authoritative per-team score as each round closes.
	p.RegisterEventHandler(func(e events.ScoreUpdated) {
		if e.TeamState == nil {
			return
		}
		switch e.TeamState.Team() {
		case common.TeamCounterTerrorists:
			state.ctScore = e.NewScore
		case common.TeamTerrorists:
			state.tScore = e.NewScore
		}
	})

	// Player position sampling.
	p.RegisterEventHandler(func(_ events.FrameDone) {
		if state.shouldStopAppending() {
			return
		}
		// Honor caller's ctx at this natural boundary. The ingester goroutine
		// (when streaming) cancels the shared errgroup ctx if it errors; we
		// notice here on the next FrameDone and stop ourselves so demoinfocs's
		// dispatch loop can drain.
		if state.ctx != nil {
			if err := state.ctx.Err(); err != nil {
				state.limitExceeded = err
				p.Cancel()
				return
			}
		}
		gs := p.GameState()
		tick := gs.IngameTick()

		// Heartbeat — runs unconditionally (warmup included) so a long pre-match
		// phase doesn't look like a hang in errors.txt and the UI doesn't sit at
		// 0% the whole way. Also enforces the heap ceiling so a runaway parse
		// fails loudly instead of paging the whole OS into a freeze.
		state.frameCount++
		if state.frameCount%heartbeatFrameInterval == 0 {
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			slog.Info("parser: heartbeat",
				"frames", state.frameCount,
				"ingame_tick", tick,
				"ticks_captured", state.tickCount,
				"events_captured", len(state.events),
				"rounds_captured", len(state.rounds),
				"heap_alloc_mb", mem.HeapAlloc>>20,
				"heap_sys_mb", mem.HeapSys>>20,
			)
			if mem.HeapAlloc > dp.maxHeapBytes {
				state.limitExceeded = fmt.Errorf("%w (limit %d MiB, observed %d MiB)",
					ErrHeapLimitExceeded, dp.maxHeapBytes>>20, mem.HeapAlloc>>20)
				slog.Warn("parser: heap limit exceeded; cancelling parse",
					"heap_alloc_mb", mem.HeapAlloc>>20,
					"max_heap_mb", dp.maxHeapBytes>>20,
					"frames", state.frameCount,
					"ingame_tick", tick,
				)
				p.Cancel()
				return
			}
			// Emit a slowly-climbing "alive" progress so the UI moves while we
			// wait for the first RoundEnd (which is when real % progress kicks
			// in). Capped at 5 so the round-based progress can take over freely.
			alivePct := float64(state.frameCount) / 200_000.0 * 5.0
			if alivePct > 5 {
				alivePct = 5
			}
			dp.reportProgress("parsing", alivePct)
		}

		if tick <= 0 {
			return
		}
		if dp.skipWarmup && gs.IsWarmupPeriod() {
			return
		}
		if !shouldSampleTick(tick, dp.tickInterval) {
			return
		}
		// FrameDone can fire multiple times per ingame tick during pauses or
		// warmup/live transitions, which would emit duplicate (tick, steam_id)
		// rows and fail the tick_data PK constraint on insert.
		if tick == state.lastSampledTick {
			return
		}
		state.lastSampledTick = tick

		for _, player := range gs.Participants().Playing() {
			if shouldSkipPlayer(player, dp.includeBots) {
				continue
			}

			pos := player.Position()
			weapon := ""
			ammoClip := 0
			ammoReserve := 0
			if w := player.ActiveWeapon(); w != nil {
				weapon = w.String()
				ammoClip = w.AmmoInMagazine()
				ammoReserve = w.AmmoReserve()
			}

			if !state.pushTick(TickSnapshot{
				Tick:        tick,
				SteamID:     state.steamID(player),
				X:           pos.X,
				Y:           pos.Y,
				Z:           pos.Z,
				Yaw:         float64(player.ViewDirectionX()),
				Health:      player.Health(),
				Armor:       player.Armor(),
				IsAlive:     player.IsAlive(),
				Weapon:      weapon,
				Money:       player.Money(),
				HasHelmet:   player.HasHelmet(),
				HasDefuser:  player.HasDefuseKit(),
				AmmoClip:    ammoClip,
				AmmoReserve: ammoReserve,
			}) {
				slog.Warn("parser: tick push failed; cancelling parse",
					"tick", tick, "ticks_captured", state.tickCount,
					"err", state.limitExceeded)
				p.Cancel()
				return
			}
		}
	})

	// Kill events.
	p.RegisterEventHandler(func(e events.Kill) {
		if state.shouldStopAppending() {
			return
		}
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}

		var attackerID, victimID string
		var x, y, z float64
		var weaponName string

		if e.Killer != nil {
			attackerID = state.steamID(e.Killer)
		}
		if e.Victim != nil {
			victimID = state.steamID(e.Victim)
			pos := e.Victim.Position()
			x, y, z = pos.X, pos.Y, pos.Z
		}
		if e.Weapon != nil {
			weaponName = e.Weapon.String()
		}

		extra := &KillExtra{
			Headshot:      e.IsHeadshot,
			Penetrated:    e.PenetratedObjects,
			FlashAssist:   e.AssistedFlash,
			ThroughSmoke:  e.ThroughSmoke,
			NoScope:       e.NoScope,
			AttackerBlind: e.AttackerBlind,
			Wallbang:      e.IsWallBang(),
		}

		if e.Assister != nil && e.Assister.SteamID64 != 0 {
			extra.AssisterSteamID = state.steamID(e.Assister)
			extra.AssisterName = e.Assister.Name
			extra.AssisterTeam = teamSideString(e.Assister.Team)
		}
		if e.Killer != nil {
			extra.AttackerName = e.Killer.Name
			extra.AttackerTeam = teamSideString(e.Killer.Team)
			killerPos := e.Killer.Position()
			ax, ay, az := killerPos.X, killerPos.Y, killerPos.Z
			extra.AttackerX = &ax
			extra.AttackerY = &ay
			extra.AttackerZ = &az
		}
		if e.Victim != nil {
			extra.VictimName = e.Victim.Name
			extra.VictimTeam = teamSideString(e.Victim.Team)
		}

		if !state.addEvent(GameEvent{
			Tick:            p.GameState().IngameTick(),
			RoundNumber:     state.currentRound,
			Type:            "kill",
			AttackerSteamID: attackerID,
			VictimSteamID:   victimID,
			Weapon:          weaponName,
			X:               x,
			Y:               y,
			Z:               z,
			ExtraData:       extra,
		}) {
			p.Cancel()
		}
	})

	// Weapon fire events — every shot, used to render shot tracers in the
	// 2D viewer. WeaponFire fires for grenades and knife slashes too, so
	// filter to firearm classes only.
	p.RegisterEventHandler(func(e events.WeaponFire) {
		if state.shouldStopAppending() {
			return
		}
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}
		if e.Shooter == nil || e.Weapon == nil {
			return
		}
		switch e.Weapon.Class() {
		case common.EqClassPistols, common.EqClassSMG, common.EqClassHeavy, common.EqClassRifle:
		default:
			return
		}

		pos := e.Shooter.Position()
		extra := &WeaponFireExtra{
			Yaw:   float64(e.Shooter.ViewDirectionX()),
			Pitch: float64(e.Shooter.ViewDirectionY()),
		}

		if !state.addEvent(GameEvent{
			Tick:            p.GameState().IngameTick(),
			RoundNumber:     state.currentRound,
			Type:            "weapon_fire",
			AttackerSteamID: state.steamID(e.Shooter),
			Weapon:          e.Weapon.String(),
			X:               pos.X,
			Y:               pos.Y,
			Z:               pos.Z,
			ExtraData:       extra,
		}) {
			p.Cancel()
		}
	})

	// Player hurt events (for damage tracking).
	p.RegisterEventHandler(func(e events.PlayerHurt) {
		if state.shouldStopAppending() {
			return
		}
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}

		var attackerID, victimID string
		extra := &PlayerHurtExtra{
			HealthDamage: e.HealthDamage,
			ArmorDamage:  e.ArmorDamage,
		}

		if e.Attacker != nil {
			attackerID = state.steamID(e.Attacker)
			extra.AttackerName = e.Attacker.Name
			extra.AttackerTeam = teamSideString(e.Attacker.Team)
		}
		var x, y, z float64
		if e.Player != nil {
			victimID = state.steamID(e.Player)
			extra.VictimName = e.Player.Name
			extra.VictimTeam = teamSideString(e.Player.Team)
			pos := e.Player.Position()
			x, y, z = pos.X, pos.Y, pos.Z
		}

		weaponName := ""
		if e.Weapon != nil {
			weaponName = e.Weapon.String()
		}

		if !state.addEvent(GameEvent{
			Tick:            p.GameState().IngameTick(),
			RoundNumber:     state.currentRound,
			Type:            "player_hurt",
			AttackerSteamID: attackerID,
			VictimSteamID:   victimID,
			Weapon:          weaponName,
			X:               x,
			Y:               y,
			Z:               z,
			ExtraData:       extra,
		}) {
			p.Cancel()
		}
	})

	// Grenade throw.
	p.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
		if state.shouldStopAppending() {
			return
		}
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}
		if e.Projectile == nil {
			return
		}

		var throwerID string
		if e.Projectile.Thrower != nil {
			throwerID = state.steamID(e.Projectile.Thrower)
		}

		pos := e.Projectile.Position()
		grenadeType := ""
		if e.Projectile.WeaponInstance != nil {
			grenadeType = e.Projectile.WeaponInstance.String()
		}

		extra := &GrenadeThrowExtra{}
		if e.Projectile.Entity != nil {
			extra.EntityID = e.Projectile.Entity.ID()
		}
		if e.Projectile.Thrower != nil {
			extra.ThrowYaw = float64(e.Projectile.Thrower.ViewDirectionX())
			extra.ThrowPitch = float64(e.Projectile.Thrower.ViewDirectionY())
		}

		if !state.addEvent(GameEvent{
			Tick:            p.GameState().IngameTick(),
			RoundNumber:     state.currentRound,
			Type:            "grenade_throw",
			AttackerSteamID: throwerID,
			Weapon:          grenadeType,
			X:               pos.X,
			Y:               pos.Y,
			Z:               pos.Z,
			ExtraData:       extra,
		}) {
			p.Cancel()
		}
	})

	// Grenade bounce — intermediate trajectory points between throw and
	// detonation. Without these, in-flight rendering would teleport between
	// the throw and detonation positions instead of curving along the actual
	// path (off walls, floors, props).
	p.RegisterEventHandler(func(e events.GrenadeProjectileBounce) {
		if state.shouldStopAppending() {
			return
		}
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}
		if e.Projectile == nil {
			return
		}

		var throwerID string
		if e.Projectile.Thrower != nil {
			throwerID = state.steamID(e.Projectile.Thrower)
		}

		pos := e.Projectile.Position()
		grenadeType := ""
		if e.Projectile.WeaponInstance != nil {
			grenadeType = e.Projectile.WeaponInstance.String()
		}

		extra := &GrenadeBounceExtra{
			BounceNr: e.BounceNr,
		}
		if e.Projectile.Entity != nil {
			extra.EntityID = e.Projectile.Entity.ID()
		}

		if !state.addEvent(GameEvent{
			Tick:            p.GameState().IngameTick(),
			RoundNumber:     state.currentRound,
			Type:            "grenade_bounce",
			AttackerSteamID: throwerID,
			Weapon:          grenadeType,
			X:               pos.X,
			Y:               pos.Y,
			Z:               pos.Z,
			ExtraData:       extra,
		}) {
			p.Cancel()
		}
	})

	// Grenade detonations (HE, flash, smoke, decoy).
	registerGrenadeDetonate := func(eventType string) func(events.GrenadeEvent) {
		return func(e events.GrenadeEvent) {
			if state.shouldStopAppending() {
				return
			}
			if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
				return
			}
			var throwerID string
			if e.Thrower != nil {
				throwerID = state.steamID(e.Thrower)
			}

			extra := &GrenadeDetonateExtra{
				EntityID: e.GrenadeEntityID,
			}

			if !state.addEvent(GameEvent{
				Tick:            p.GameState().IngameTick(),
				RoundNumber:     state.currentRound,
				Type:            eventType,
				AttackerSteamID: throwerID,
				Weapon:          e.GrenadeType.String(),
				X:               e.Position.X,
				Y:               e.Position.Y,
				Z:               e.Position.Z,
				ExtraData:       extra,
			}) {
				p.Cancel()
			}
		}
	}

	p.RegisterEventHandler(func(e events.HeExplode) {
		registerGrenadeDetonate("grenade_detonate")(e.GrenadeEvent)
	})
	p.RegisterEventHandler(func(e events.FlashExplode) {
		registerGrenadeDetonate("grenade_detonate")(e.GrenadeEvent)
	})
	p.RegisterEventHandler(func(e events.SmokeStart) {
		registerGrenadeDetonate("smoke_start")(e.GrenadeEvent)
	})
	p.RegisterEventHandler(func(e events.SmokeExpired) {
		registerGrenadeDetonate("smoke_expired")(e.GrenadeEvent)
	})
	p.RegisterEventHandler(func(e events.DecoyStart) {
		registerGrenadeDetonate("decoy_start")(e.GrenadeEvent)
	})

	// Incendiary/Molotov detonation — missed in web-era parser.
	// Spike identified ~25% orphaned grenade throws due to this gap.
	p.RegisterEventHandler(func(e events.FireGrenadeStart) {
		registerGrenadeDetonate("fire_start")(e.GrenadeEvent)
	})

	// Bomb events.
	// All bomb events use GameState().Bomb().Position() for the bomb's world-space
	// coordinates (planted location), since the planter may have moved or died.
	p.RegisterEventHandler(func(e events.BombPlanted) {
		if state.shouldStopAppending() {
			return
		}
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}
		var playerID string
		if e.Player != nil {
			playerID = state.steamID(e.Player)
		}

		bombPos := p.GameState().Bomb().Position()
		if !state.addEvent(GameEvent{
			Tick:            p.GameState().IngameTick(),
			RoundNumber:     state.currentRound,
			Type:            "bomb_plant",
			AttackerSteamID: playerID,
			X:               bombPos.X,
			Y:               bombPos.Y,
			Z:               bombPos.Z,
			ExtraData: &BombPlantExtra{
				Site: bombsiteString(e.Site),
			},
		}) {
			p.Cancel()
		}
	})

	p.RegisterEventHandler(func(e events.BombDefused) {
		if state.shouldStopAppending() {
			return
		}
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}
		var playerID string
		if e.Player != nil {
			playerID = state.steamID(e.Player)
		}

		hasKit := false
		if e.Player != nil {
			hasKit = e.Player.HasDefuseKit()
		}

		bombPos := p.GameState().Bomb().Position()
		if !state.addEvent(GameEvent{
			Tick:            p.GameState().IngameTick(),
			RoundNumber:     state.currentRound,
			Type:            "bomb_defuse",
			AttackerSteamID: playerID,
			X:               bombPos.X,
			Y:               bombPos.Y,
			Z:               bombPos.Z,
			ExtraData: &BombDefuseExtra{
				Site:   bombsiteString(e.Site),
				HasKit: hasKit,
			},
		}) {
			p.Cancel()
		}
	})

	p.RegisterEventHandler(func(e events.BombExplode) {
		if state.shouldStopAppending() {
			return
		}
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}

		bombPos := p.GameState().Bomb().Position()
		if !state.addEvent(GameEvent{
			Tick:        p.GameState().IngameTick(),
			RoundNumber: state.currentRound,
			Type:        "bomb_explode",
			X:           bombPos.X,
			Y:           bombPos.Y,
			Z:           bombPos.Z,
			ExtraData: &BombExplodeExtra{
				Site: bombsiteString(e.Site),
			},
		}) {
			p.Cancel()
		}
	})
}

// shouldSampleTick returns true if tick should be sampled at the given interval.
func shouldSampleTick(tick, interval int) bool {
	if interval <= 0 {
		return false
	}
	return tick%interval == 0
}

// encodeInventory serializes a player's weapons as a comma-separated list of
// canonical names so we can decode them on the frontend without committing to
// a JSON shape. Empty/knife-only entries are kept since the bar UI sorts and
// filters them itself.
func encodeInventory(weapons []*common.Equipment) string {
	if len(weapons) == 0 {
		return ""
	}
	names := make([]string, 0, len(weapons))
	for _, w := range weapons {
		if w == nil {
			continue
		}
		s := w.String()
		if s == "" || s == "UNKNOWN" {
			continue
		}
		names = append(names, s)
	}
	return strings.Join(names, ",")
}

// shouldSkipPlayer returns true if the player should be excluded from tick snapshots.
func shouldSkipPlayer(player *common.Player, includeBots bool) bool {
	if player == nil {
		return true
	}
	if includeBots {
		return false
	}
	return player.SteamID64 == 0 || player.IsBot
}

// teamSideString converts a common.Team to "CT" or "T" string.
func teamSideString(team common.Team) string {
	switch team {
	case common.TeamCounterTerrorists:
		return "CT"
	case common.TeamTerrorists:
		return "T"
	default:
		return ""
	}
}

// bombsiteString converts an events.Bombsite to a string.
func bombsiteString(site events.Bombsite) string {
	switch site {
	case events.BombsiteA:
		return "A"
	case events.BombsiteB:
		return "B"
	default:
		return ""
	}
}

// roundEndReasonString converts a RoundEndReason to a human-readable string.
func roundEndReasonString(r events.RoundEndReason) string {
	switch r {
	case events.RoundEndReasonTargetBombed:
		return "target_bombed"
	case events.RoundEndReasonBombDefused:
		return "bomb_defused"
	case events.RoundEndReasonCTWin:
		return "ct_win"
	case events.RoundEndReasonTerroristsWin:
		return "t_win"
	case events.RoundEndReasonDraw:
		return "draw"
	case events.RoundEndReasonTargetSaved:
		return "target_saved"
	case events.RoundEndReasonTerroristsSurrender:
		return "t_surrender"
	case events.RoundEndReasonCTSurrender:
		return "ct_surrender"
	default:
		return fmt.Sprintf("reason_%d", r)
	}
}
