package demo

import (
	"fmt"
	"io"
	"log/slog"
	"strconv"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

// ParseResult is the complete output of parsing a demo file.
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
}

// TickSnapshot is one player's state at a sampled tick.
type TickSnapshot struct {
	Tick    int
	SteamID string // Steam64ID as string
	X, Y, Z float64
	Yaw     float64
	Health  int
	Armor   int
	IsAlive bool
	Weapon  string
}

// GameEvent represents a parsed game event (kill, grenade, bomb, round boundary).
type GameEvent struct {
	Tick            int
	RoundNumber     int
	Type            string // "kill", "player_hurt", "grenade_throw", "grenade_detonate", "smoke_start", "smoke_expired", "decoy_start", "fire_start", "bomb_plant", "bomb_defuse", "bomb_explode"
	AttackerSteamID string
	VictimSteamID   string
	Weapon          string
	X, Y, Z         float64
	ExtraData       map[string]interface{}
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

// DemoParser extracts structured data from CS2 .dem files.
type DemoParser struct {
	tickInterval int
	skipWarmup   bool
	includeBots  bool
	progressFunc ProgressFunc
}

// NewDemoParser creates a parser with the given options.
func NewDemoParser(opts ...Option) *DemoParser {
	dp := &DemoParser{
		tickInterval: 4,
		skipWarmup:   true,
		includeBots:  false,
	}
	for _, opt := range opts {
		opt(dp)
	}
	return dp
}

// parseState holds mutable state tracked during parsing.
type parseState struct {
	mapName           string
	inWarmup          bool
	matchStarted      bool
	matchStartCount   int
	currentRound      int
	roundStart        int
	freezeEndTick     int
	ctScore           int
	tScore            int
	rounds            []RoundData
	ticks             []TickSnapshot
	events            []GameEvent
	lastSampledTick   int
	knifeRoundNumbers map[int]bool
}

// resetForPreMatchRestart discards all captured data from the pre-match phase.
// Called when MatchStartedChanged(true) re-fires with the match score still at
// 0-0, which signals a Faceit-style knife-round → live-match transition.
func (s *parseState) resetForPreMatchRestart() {
	s.currentRound = 0
	s.roundStart = 0
	s.freezeEndTick = 0
	s.lastSampledTick = 0
	s.rounds = nil
	s.ticks = nil
	s.events = nil
	s.knifeRoundNumbers = nil
}

// Parse reads a CS2 demo from r and returns all extracted data.
func (dp *DemoParser) Parse(r io.Reader) (result *ParseResult, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("parsing demo: panic: %v", rec)
		}
	}()

	p := demoinfocs.NewParser(r)
	defer func() {
		if closeErr := p.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing parser: %w", closeErr)
		}
	}()

	state := &parseState{}

	dp.registerHandlers(p, state)

	dp.reportProgress("parsing", 0)

	if parseErr := p.ParseToEnd(); parseErr != nil {
		// demoinfocs returns ErrUnexpectedEndOfDemo for truncated demos;
		// treat as fatal only if we got zero data.
		if len(state.rounds) == 0 && len(state.ticks) == 0 && len(state.events) == 0 {
			return nil, fmt.Errorf("parsing demo: %w", parseErr)
		}
	}

	dp.reportProgress("parsing", 90)

	tickRate := p.TickRate()
	totalTicks := p.GameState().IngameTick()

	var durationSecs int
	if tickRate > 0 {
		durationSecs = int(float64(totalTicks) / tickRate)
	}

	rounds, events := dropKnifeRounds(state.rounds, state.events, state.knifeRoundNumbers)

	lineups := ExtractGrenadeLineups(state.mapName, events)

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
//  2. At each freeze-time end, if every live player's inventory is a knife
//     (optionally plus the C4), the round number is recorded in the flagged
//     map. This is the fallback when the restart event does not fire.
//
// Remaining rounds are renumbered contiguously from 1, events are renumbered to
// match, and scores are adjusted to cancel out the dropped knife-round wins.
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
		rd.IsOvertime = isOvertime(newNumber)
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

// isKnifeRoundByInventory returns true if every provided inventory contains only
// a knife (and optionally the C4). Faceit knife rounds force knife-only loadouts
// via mp_startmoney=0 + empty default secondaries; the inventory check at freeze
// time catches them even if the restart signal is absent. Returns false for an
// empty input to avoid flagging transient zero-player frames.
func isKnifeRoundByInventory(inventories [][]common.EquipmentType) bool {
	if len(inventories) == 0 {
		return false
	}
	for _, inv := range inventories {
		hasKnife := false
		for _, t := range inv {
			switch t {
			case common.EqKnife:
				hasKnife = true
			case common.EqBomb:
				// C4 is allowed — some Faceit configs still assign it on knife rounds.
			default:
				return false
			}
		}
		if !hasKnife {
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

	// Warmup tracking.
	p.RegisterEventHandler(func(e events.IsWarmupPeriodChanged) {
		state.inWarmup = e.NewIsWarmupPeriod
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
		if dp.skipWarmup && state.inWarmup {
			return
		}
		state.currentRound++
		state.roundStart = p.GameState().IngameTick()
		state.freezeEndTick = 0
	})

	// Freeze time end — the round goes live. Capture the tick once per round
	// (ignore duplicate/late events that would push it past the true end), then
	// snapshot player inventories. If every live player holds only a knife
	// (optionally plus the C4), flag the round for later drop in dropKnifeRounds.
	captureFreezeEnd := func() {
		if dp.skipWarmup && state.inWarmup {
			return
		}
		if state.freezeEndTick != 0 {
			return
		}
		state.freezeEndTick = p.GameState().IngameTick()

		var inventories [][]common.EquipmentType
		for _, player := range p.GameState().Participants().Playing() {
			if shouldSkipPlayer(player, dp.includeBots) {
				continue
			}
			weapons := player.Weapons()
			inv := make([]common.EquipmentType, 0, len(weapons))
			for _, w := range weapons {
				if w == nil {
					continue
				}
				inv = append(inv, w.Type)
			}
			inventories = append(inventories, inv)
		}
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
		if dp.skipWarmup && state.inWarmup {
			return
		}
		if state.currentRound == 0 {
			return
		}

		winnerSide := teamSideString(e.Winner)

		// In demoinfocs v5, ScoreUpdated fires BEFORE RoundEnd, so by the time
		// this handler runs e.WinnerState.Score() already reflects the win and
		// state.ctScore/tScore are already post-round. Read them directly.
		// Fall back to an increment-from-state path if the event is missing
		// authoritative team state (defensive against malformed demos).
		ctScore := state.ctScore
		tScore := state.tScore
		if e.WinnerState != nil {
			switch e.Winner {
			case common.TeamCounterTerrorists:
				ctScore = e.WinnerState.Score()
				if e.LoserState != nil {
					tScore = e.LoserState.Score()
				}
			case common.TeamTerrorists:
				tScore = e.WinnerState.Score()
				if e.LoserState != nil {
					ctScore = e.LoserState.Score()
				}
			}
		}
		state.ctScore = ctScore
		state.tScore = tScore

		state.rounds = append(state.rounds, RoundData{
			Number:        state.currentRound,
			StartTick:     state.roundStart,
			FreezeEndTick: state.freezeEndTick,
			EndTick:       p.GameState().IngameTick(),
			WinnerSide:    winnerSide,
			WinReason:     roundEndReasonString(e.Reason),
			CTScore:       ctScore,
			TScore:        tScore,
			IsOvertime:    isOvertime(state.currentRound),
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
		gs := p.GameState()
		tick := gs.IngameTick()

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
			if w := player.ActiveWeapon(); w != nil {
				weapon = w.String()
			}

			state.ticks = append(state.ticks, TickSnapshot{
				Tick:    tick,
				SteamID: strconv.FormatUint(player.SteamID64, 10),
				X:       pos.X,
				Y:       pos.Y,
				Z:       pos.Z,
				Yaw:     float64(player.ViewDirectionX()),
				Health:  player.Health(),
				Armor:   player.Armor(),
				IsAlive: player.IsAlive(),
				Weapon:  weapon,
			})
		}
	})

	// Kill events.
	p.RegisterEventHandler(func(e events.Kill) {
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}

		var attackerID, victimID string
		var x, y, z float64
		var weaponName string

		if e.Killer != nil {
			attackerID = strconv.FormatUint(e.Killer.SteamID64, 10)
		}
		if e.Victim != nil {
			victimID = strconv.FormatUint(e.Victim.SteamID64, 10)
			pos := e.Victim.Position()
			x, y, z = pos.X, pos.Y, pos.Z
		}
		if e.Weapon != nil {
			weaponName = e.Weapon.String()
		}

		extra := map[string]interface{}{
			"headshot":       e.IsHeadshot,
			"penetrated":     e.PenetratedObjects,
			"flash_assist":   e.AssistedFlash,
			"through_smoke":  e.ThroughSmoke,
			"no_scope":       e.NoScope,
			"attacker_blind": e.AttackerBlind,
			"wallbang":       e.IsWallBang(),
		}

		if e.Assister != nil && e.Assister.SteamID64 != 0 {
			extra["assister_steam_id"] = strconv.FormatUint(e.Assister.SteamID64, 10)
			extra["assister_name"] = e.Assister.Name
			extra["assister_team"] = teamSideString(e.Assister.Team)
		}
		if e.Killer != nil {
			extra["attacker_name"] = e.Killer.Name
			extra["attacker_team"] = teamSideString(e.Killer.Team)
			killerPos := e.Killer.Position()
			extra["attacker_x"] = killerPos.X
			extra["attacker_y"] = killerPos.Y
			extra["attacker_z"] = killerPos.Z
		}
		if e.Victim != nil {
			extra["victim_name"] = e.Victim.Name
			extra["victim_team"] = teamSideString(e.Victim.Team)
		}

		state.events = append(state.events, GameEvent{
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
		})
	})

	// Player hurt events (for damage tracking).
	p.RegisterEventHandler(func(e events.PlayerHurt) {
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}

		var attackerID, victimID string
		extra := map[string]interface{}{
			"health_damage": e.HealthDamage,
			"armor_damage":  e.ArmorDamage,
		}

		if e.Attacker != nil {
			attackerID = strconv.FormatUint(e.Attacker.SteamID64, 10)
			extra["attacker_name"] = e.Attacker.Name
			extra["attacker_team"] = teamSideString(e.Attacker.Team)
		}
		if e.Player != nil {
			victimID = strconv.FormatUint(e.Player.SteamID64, 10)
			extra["victim_name"] = e.Player.Name
			extra["victim_team"] = teamSideString(e.Player.Team)
		}

		weaponName := ""
		if e.Weapon != nil {
			weaponName = e.Weapon.String()
		}

		state.events = append(state.events, GameEvent{
			Tick:            p.GameState().IngameTick(),
			RoundNumber:     state.currentRound,
			Type:            "player_hurt",
			AttackerSteamID: attackerID,
			VictimSteamID:   victimID,
			Weapon:          weaponName,
			ExtraData:       extra,
		})
	})

	// Grenade throw.
	p.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}
		if e.Projectile == nil {
			return
		}

		var throwerID string
		if e.Projectile.Thrower != nil {
			throwerID = strconv.FormatUint(e.Projectile.Thrower.SteamID64, 10)
		}

		pos := e.Projectile.Position()
		grenadeType := ""
		if e.Projectile.WeaponInstance != nil {
			grenadeType = e.Projectile.WeaponInstance.String()
		}

		extra := map[string]interface{}{}
		if e.Projectile.Entity != nil {
			extra["entity_id"] = e.Projectile.Entity.ID()
		}
		if e.Projectile.Thrower != nil {
			extra["throw_yaw"] = float64(e.Projectile.Thrower.ViewDirectionX())
			extra["throw_pitch"] = float64(e.Projectile.Thrower.ViewDirectionY())
		}

		state.events = append(state.events, GameEvent{
			Tick:            p.GameState().IngameTick(),
			RoundNumber:     state.currentRound,
			Type:            "grenade_throw",
			AttackerSteamID: throwerID,
			Weapon:          grenadeType,
			X:               pos.X,
			Y:               pos.Y,
			Z:               pos.Z,
			ExtraData:       extra,
		})
	})

	// Grenade detonations (HE, flash, smoke, decoy).
	registerGrenadeDetonate := func(eventType string) func(events.GrenadeEvent) {
		return func(e events.GrenadeEvent) {
			if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
				return
			}
			var throwerID string
			if e.Thrower != nil {
				throwerID = strconv.FormatUint(e.Thrower.SteamID64, 10)
			}

			extra := map[string]interface{}{
				"entity_id": e.GrenadeEntityID,
			}

			state.events = append(state.events, GameEvent{
				Tick:            p.GameState().IngameTick(),
				RoundNumber:     state.currentRound,
				Type:            eventType,
				AttackerSteamID: throwerID,
				Weapon:          e.GrenadeType.String(),
				X:               e.Position.X,
				Y:               e.Position.Y,
				Z:               e.Position.Z,
				ExtraData:       extra,
			})
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
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}
		var playerID string
		if e.Player != nil {
			playerID = strconv.FormatUint(e.Player.SteamID64, 10)
		}

		bombPos := p.GameState().Bomb().Position()
		state.events = append(state.events, GameEvent{
			Tick:            p.GameState().IngameTick(),
			RoundNumber:     state.currentRound,
			Type:            "bomb_plant",
			AttackerSteamID: playerID,
			X:               bombPos.X,
			Y:               bombPos.Y,
			Z:               bombPos.Z,
			ExtraData: map[string]interface{}{
				"site": bombsiteString(e.Site),
			},
		})
	})

	p.RegisterEventHandler(func(e events.BombDefused) {
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}
		var playerID string
		if e.Player != nil {
			playerID = strconv.FormatUint(e.Player.SteamID64, 10)
		}

		hasKit := false
		if e.Player != nil {
			hasKit = e.Player.HasDefuseKit()
		}

		bombPos := p.GameState().Bomb().Position()
		state.events = append(state.events, GameEvent{
			Tick:            p.GameState().IngameTick(),
			RoundNumber:     state.currentRound,
			Type:            "bomb_defuse",
			AttackerSteamID: playerID,
			X:               bombPos.X,
			Y:               bombPos.Y,
			Z:               bombPos.Z,
			ExtraData: map[string]interface{}{
				"site":    bombsiteString(e.Site),
				"has_kit": hasKit,
			},
		})
	})

	p.RegisterEventHandler(func(e events.BombExplode) {
		if dp.skipWarmup && p.GameState().IsWarmupPeriod() {
			return
		}

		bombPos := p.GameState().Bomb().Position()
		state.events = append(state.events, GameEvent{
			Tick:        p.GameState().IngameTick(),
			RoundNumber: state.currentRound,
			Type:        "bomb_explode",
			X:           bombPos.X,
			Y:           bombPos.Y,
			Z:           bombPos.Z,
			ExtraData: map[string]interface{}{
				"site": bombsiteString(e.Site),
			},
		})
	})
}

// shouldSampleTick returns true if tick should be sampled at the given interval.
func shouldSampleTick(tick, interval int) bool {
	if interval <= 0 {
		return false
	}
	return tick%interval == 0
}

// isOvertime returns true if the round number is in overtime.
// Assumes MR12 format (24 regulation rounds) as used in Faceit competitive CS2.
func isOvertime(roundNum int) bool {
	return roundNum > 24
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
