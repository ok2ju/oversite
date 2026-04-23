package demo

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func TestShouldSampleTick(t *testing.T) {
	tests := []struct {
		name     string
		tick     int
		interval int
		want     bool
	}{
		{
			name:     "divisible tick",
			tick:     4,
			interval: 4,
			want:     true,
		},
		{
			name:     "non-divisible tick",
			tick:     5,
			interval: 4,
			want:     false,
		},
		{
			name:     "tick zero is divisible by any positive interval",
			tick:     0,
			interval: 4,
			want:     true,
		},
		{
			name:     "multiple of interval",
			tick:     8,
			interval: 4,
			want:     true,
		},
		{
			name:     "every tick (interval=1)",
			tick:     1,
			interval: 1,
			want:     true,
		},
		{
			name:     "zero interval returns false",
			tick:     100,
			interval: 0,
			want:     false,
		},
		{
			name:     "negative interval returns false",
			tick:     100,
			interval: -1,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSampleTick(tt.tick, tt.interval)
			if got != tt.want {
				t.Errorf("shouldSampleTick(%d, %d) = %v, want %v", tt.tick, tt.interval, got, tt.want)
			}
		})
	}
}

func TestIsOvertime(t *testing.T) {
	tests := []struct {
		name     string
		roundNum int
		want     bool
	}{
		{
			name:     "round 1 is regulation",
			roundNum: 1,
			want:     false,
		},
		{
			name:     "round 12 is regulation",
			roundNum: 12,
			want:     false,
		},
		{
			name:     "round 24 is last regulation round",
			roundNum: 24,
			want:     false,
		},
		{
			name:     "round 25 is first overtime round",
			roundNum: 25,
			want:     true,
		},
		{
			name:     "round 30 is overtime",
			roundNum: 30,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOvertime(tt.roundNum)
			if got != tt.want {
				t.Errorf("isOvertime(%d) = %v, want %v", tt.roundNum, got, tt.want)
			}
		})
	}
}

func TestShouldSkipPlayer(t *testing.T) {
	tests := []struct {
		name        string
		player      *common.Player
		includeBots bool
		want        bool
	}{
		{
			name:        "nil player is skipped",
			player:      nil,
			includeBots: false,
			want:        true,
		},
		{
			name:        "bot with includeBots=false is skipped",
			player:      &common.Player{SteamID64: 0, IsBot: true},
			includeBots: false,
			want:        true,
		},
		{
			name:        "bot with includeBots=true is not skipped",
			player:      &common.Player{SteamID64: 0, IsBot: true},
			includeBots: true,
			want:        false,
		},
		{
			name:        "real player is not skipped",
			player:      &common.Player{SteamID64: 76561198012345678, IsBot: false},
			includeBots: false,
			want:        false,
		},
		{
			name:        "player with zero SteamID and not marked as bot is skipped",
			player:      &common.Player{SteamID64: 0, IsBot: false},
			includeBots: false,
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSkipPlayer(tt.player, tt.includeBots)
			if got != tt.want {
				t.Errorf("shouldSkipPlayer(player=%v, includeBots=%v) = %v, want %v",
					tt.player, tt.includeBots, got, tt.want)
			}
		})
	}
}

func TestTeamSideString(t *testing.T) {
	tests := []struct {
		name string
		team common.Team
		want string
	}{
		{
			name: "counter-terrorists",
			team: common.TeamCounterTerrorists,
			want: "CT",
		},
		{
			name: "terrorists",
			team: common.TeamTerrorists,
			want: "T",
		},
		{
			name: "unassigned returns empty",
			team: common.TeamUnassigned,
			want: "",
		},
		{
			name: "spectators returns empty",
			team: common.TeamSpectators,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := teamSideString(tt.team)
			if got != tt.want {
				t.Errorf("teamSideString(%v) = %q, want %q", tt.team, got, tt.want)
			}
		})
	}
}

func TestBombsiteString(t *testing.T) {
	tests := []struct {
		name string
		site events.Bombsite
		want string
	}{
		{
			name: "bombsite A",
			site: events.BombsiteA,
			want: "A",
		},
		{
			name: "bombsite B",
			site: events.BombsiteB,
			want: "B",
		},
		{
			name: "unknown bombsite returns empty",
			site: events.Bombsite(99),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bombsiteString(tt.site)
			if got != tt.want {
				t.Errorf("bombsiteString(%v) = %q, want %q", tt.site, got, tt.want)
			}
		})
	}
}

func TestRoundEndReasonString(t *testing.T) {
	tests := []struct {
		name   string
		reason events.RoundEndReason
		want   string
	}{
		{
			name:   "target bombed",
			reason: events.RoundEndReasonTargetBombed,
			want:   "target_bombed",
		},
		{
			name:   "bomb defused",
			reason: events.RoundEndReasonBombDefused,
			want:   "bomb_defused",
		},
		{
			name:   "CT win",
			reason: events.RoundEndReasonCTWin,
			want:   "ct_win",
		},
		{
			name:   "terrorists win",
			reason: events.RoundEndReasonTerroristsWin,
			want:   "t_win",
		},
		{
			name:   "draw",
			reason: events.RoundEndReasonDraw,
			want:   "draw",
		},
		{
			name:   "target saved",
			reason: events.RoundEndReasonTargetSaved,
			want:   "target_saved",
		},
		{
			name:   "terrorists surrender",
			reason: events.RoundEndReasonTerroristsSurrender,
			want:   "t_surrender",
		},
		{
			name:   "CT surrender",
			reason: events.RoundEndReasonCTSurrender,
			want:   "ct_surrender",
		},
		{
			name:   "unknown reason returns formatted string",
			reason: events.RoundEndReason(200),
			want:   "reason_200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roundEndReasonString(tt.reason)
			if got != tt.want {
				t.Errorf("roundEndReasonString(%v) = %q, want %q", tt.reason, got, tt.want)
			}
		})
	}
}

func TestNewDemoParser_Defaults(t *testing.T) {
	dp := NewDemoParser()

	if dp.tickInterval != 4 {
		t.Errorf("default tickInterval = %d, want 4", dp.tickInterval)
	}
	if !dp.skipWarmup {
		t.Errorf("default skipWarmup = %v, want true", dp.skipWarmup)
	}
	if dp.includeBots {
		t.Errorf("default includeBots = %v, want false", dp.includeBots)
	}
	if dp.progressFunc != nil {
		t.Errorf("default progressFunc = %v, want nil", dp.progressFunc)
	}
}

func TestOptionWithTickInterval(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		wantTick int
	}{
		{
			name:     "valid positive interval is applied",
			input:    8,
			wantTick: 8,
		},
		{
			name:     "zero interval is ignored (stays at default)",
			input:    0,
			wantTick: 4,
		},
		{
			name:     "negative interval is ignored (stays at default)",
			input:    -1,
			wantTick: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dp := NewDemoParser(WithTickInterval(tt.input))
			if dp.tickInterval != tt.wantTick {
				t.Errorf("WithTickInterval(%d): tickInterval = %d, want %d",
					tt.input, dp.tickInterval, tt.wantTick)
			}
		})
	}
}

func TestOptionWithSkipWarmup(t *testing.T) {
	dp := NewDemoParser(WithSkipWarmup(false))
	if dp.skipWarmup {
		t.Errorf("WithSkipWarmup(false): skipWarmup = %v, want false", dp.skipWarmup)
	}
}

func TestOptionWithIncludeBots(t *testing.T) {
	dp := NewDemoParser(WithIncludeBots(true))
	if !dp.includeBots {
		t.Errorf("WithIncludeBots(true): includeBots = %v, want true", dp.includeBots)
	}
}

func TestOptionWithProgressFunc(t *testing.T) {
	var called bool
	fn := func(stage string, percent float64) {
		called = true
	}

	dp := NewDemoParser(WithProgressFunc(fn))
	if dp.progressFunc == nil {
		t.Fatal("WithProgressFunc: progressFunc is nil, want non-nil")
	}

	// Invoke via the stored field to confirm it is the same function.
	dp.progressFunc("test", 50)
	if !called {
		t.Error("progressFunc was not invoked after calling it directly")
	}
}

func TestProgressFunc_Callback(t *testing.T) {
	type call struct {
		stage   string
		percent float64
	}

	var calls []call
	fn := func(stage string, percent float64) {
		calls = append(calls, call{stage, percent})
	}

	dp := NewDemoParser(WithProgressFunc(fn))

	// reportProgress is the internal method that dispatches to progressFunc.
	dp.reportProgress("init", 0)
	dp.reportProgress("parsing", 50)
	dp.reportProgress("done", 100)

	if len(calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(calls))
	}

	wantCalls := []call{
		{"init", 0},
		{"parsing", 50},
		{"done", 100},
	}
	for i, wc := range wantCalls {
		if calls[i] != wc {
			t.Errorf("call[%d] = %+v, want %+v", i, calls[i], wc)
		}
	}
}

func TestIsKnifeRoundByInventory(t *testing.T) {
	tenKnives := func() [][]common.EquipmentType {
		inv := make([][]common.EquipmentType, 10)
		for i := range inv {
			inv[i] = []common.EquipmentType{common.EqKnife}
		}
		return inv
	}

	tests := []struct {
		name        string
		inventories [][]common.EquipmentType
		want        bool
	}{
		{
			name:        "all players knife-only",
			inventories: tenKnives(),
			want:        true,
		},
		{
			name: "all knives plus T-side C4",
			inventories: [][]common.EquipmentType{
				{common.EqKnife},
				{common.EqKnife, common.EqBomb},
				{common.EqKnife},
				{common.EqKnife},
				{common.EqKnife},
			},
			want: true,
		},
		{
			name: "one player holds a pistol",
			inventories: [][]common.EquipmentType{
				{common.EqKnife},
				{common.EqKnife},
				{common.EqGlock, common.EqKnife},
			},
			want: false,
		},
		{
			name: "one player carries an AK",
			inventories: [][]common.EquipmentType{
				{common.EqKnife},
				{common.EqKnife},
				{common.EqKnife, common.EqAK47},
			},
			want: false,
		},
		{
			name:        "no live players — not a knife round",
			inventories: nil,
			want:        false,
		},
		{
			name: "player with only C4 but no knife",
			inventories: [][]common.EquipmentType{
				{common.EqKnife},
				{common.EqBomb},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isKnifeRoundByInventory(tt.inventories)
			if got != tt.want {
				t.Errorf("isKnifeRoundByInventory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDropKnifeRounds_NoFlaggedRounds(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT", CTScore: 1, TScore: 0},
		{Number: 2, StartTick: 1001, EndTick: 2000, WinnerSide: "T", CTScore: 1, TScore: 1},
	}
	events := []GameEvent{
		{Tick: 100, RoundNumber: 1, Type: "kill", Weapon: "AK-47"},
		{Tick: 1100, RoundNumber: 2, Type: "kill", Weapon: "M4A1"},
	}

	gotRounds, gotEvents := dropKnifeRounds(rounds, events, nil)
	if len(gotRounds) != 2 {
		t.Fatalf("rounds length = %d, want 2", len(gotRounds))
	}
	if len(gotEvents) != len(events) {
		t.Fatalf("events length = %d, want %d", len(gotEvents), len(events))
	}
}

func TestDropKnifeRounds_DropsFlaggedRoundAndRenumbers(t *testing.T) {
	rounds := []RoundData{
		// Round 1 is a knife round — CT wins.
		{Number: 1, StartTick: 0, EndTick: 500, WinnerSide: "CT", CTScore: 1, TScore: 0},
		// Real pistol round — T wins; score inflated by +1 CT from knife round.
		{Number: 2, StartTick: 501, EndTick: 2000, WinnerSide: "T", CTScore: 1, TScore: 1},
		// Round 3 — CT wins.
		{Number: 3, StartTick: 2001, EndTick: 3000, WinnerSide: "CT", CTScore: 2, TScore: 1},
	}
	events := []GameEvent{
		{Tick: 50, RoundNumber: 1, Type: "kill", Weapon: "Knife"},
		{Tick: 600, RoundNumber: 2, Type: "kill", Weapon: "Glock-18"},
		{Tick: 650, RoundNumber: 2, Type: "player_hurt", Weapon: "USP-S"},
		{Tick: 2500, RoundNumber: 3, Type: "kill", Weapon: "AK-47"},
	}
	flagged := map[int]bool{1: true}

	gotRounds, gotEvents := dropKnifeRounds(rounds, events, flagged)

	if len(gotRounds) != 2 {
		t.Fatalf("rounds length = %d, want 2", len(gotRounds))
	}
	if gotRounds[0].Number != 1 || gotRounds[1].Number != 2 {
		t.Errorf("renumbered = [%d, %d], want [1, 2]", gotRounds[0].Number, gotRounds[1].Number)
	}
	if gotRounds[0].CTScore != 0 || gotRounds[0].TScore != 1 {
		t.Errorf("round 1 score = (%d, %d), want (0, 1)", gotRounds[0].CTScore, gotRounds[0].TScore)
	}
	if gotRounds[1].CTScore != 1 || gotRounds[1].TScore != 1 {
		t.Errorf("round 2 score = (%d, %d), want (1, 1)", gotRounds[1].CTScore, gotRounds[1].TScore)
	}

	if len(gotEvents) != 3 {
		t.Fatalf("events length = %d, want 3", len(gotEvents))
	}
	for _, ev := range gotEvents {
		if ev.RoundNumber != 1 && ev.RoundNumber != 2 {
			t.Errorf("event RoundNumber = %d, want 1 or 2", ev.RoundNumber)
		}
	}
}

func TestDropKnifeRounds_FlaggedButWinnerSideUnknown(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, StartTick: 0, EndTick: 500, WinnerSide: "", CTScore: 0, TScore: 0},
		{Number: 2, StartTick: 501, EndTick: 2000, WinnerSide: "T", CTScore: 0, TScore: 1},
	}
	flagged := map[int]bool{1: true}

	gotRounds, _ := dropKnifeRounds(rounds, nil, flagged)

	if len(gotRounds) != 1 {
		t.Fatalf("rounds length = %d, want 1", len(gotRounds))
	}
	if gotRounds[0].Number != 1 || gotRounds[0].TScore != 1 {
		t.Errorf("round 1 = (%d, T=%d), want (1, T=1)", gotRounds[0].Number, gotRounds[0].TScore)
	}
}

func TestParseStateResetForPreMatchRestart(t *testing.T) {
	state := &parseState{
		currentRound:      1,
		roundStart:        100,
		freezeEndTick:     200,
		lastSampledTick:   250,
		rounds:            []RoundData{{Number: 1}},
		ticks:             []TickSnapshot{{Tick: 100}},
		events:            []GameEvent{{Tick: 100, RoundNumber: 1, Type: "kill"}},
		knifeRoundNumbers: map[int]bool{1: true},
		matchStartCount:   2,
		// Scores stay zero — the trigger condition for the reset.
	}

	state.resetForPreMatchRestart()

	if state.currentRound != 0 {
		t.Errorf("currentRound = %d, want 0", state.currentRound)
	}
	if state.roundStart != 0 {
		t.Errorf("roundStart = %d, want 0", state.roundStart)
	}
	if state.freezeEndTick != 0 {
		t.Errorf("freezeEndTick = %d, want 0", state.freezeEndTick)
	}
	if state.lastSampledTick != 0 {
		t.Errorf("lastSampledTick = %d, want 0", state.lastSampledTick)
	}
	if state.rounds != nil {
		t.Errorf("rounds = %v, want nil", state.rounds)
	}
	if state.ticks != nil {
		t.Errorf("ticks = %v, want nil", state.ticks)
	}
	if state.events != nil {
		t.Errorf("events = %v, want nil", state.events)
	}
	if state.knifeRoundNumbers != nil {
		t.Errorf("knifeRoundNumbers = %v, want nil", state.knifeRoundNumbers)
	}
	// matchStartCount is intentionally preserved so subsequent restarts are
	// still recognized as "second or later".
	if state.matchStartCount != 2 {
		t.Errorf("matchStartCount = %d, want 2 (preserved)", state.matchStartCount)
	}
}

// captureSlog swaps slog's default logger for one that writes to a buffer
// and returns a restore function. Used to assert on log output.
func captureSlog(t *testing.T, level slog.Level) (*bytes.Buffer, func()) {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: level})))
	return buf, func() { slog.SetDefault(prev) }
}

func TestMidMatchRestart_NoResetAndWarnLogged(t *testing.T) {
	// Exercise the same decision as the MatchStartedChanged handler without
	// needing a real demo parser: non-zero score + post-initial restart must
	// preserve captured data and emit a warn-level log line.
	buf, restore := captureSlog(t, slog.LevelWarn)
	defer restore()

	state := &parseState{
		ctScore:         5,
		tScore:          3,
		currentRound:    9,
		rounds:          []RoundData{{Number: 1}, {Number: 2}},
		events:          []GameEvent{{RoundNumber: 1, Type: "kill"}},
		matchStartCount: 2,
	}

	// This mirrors the second branch of the handler.
	if state.ctScore != 0 || state.tScore != 0 {
		slog.Warn("mid-match restart detected; keeping captured data",
			"tick", 1234, "count", state.matchStartCount,
			"ct", state.ctScore, "t", state.tScore)
	} else {
		state.resetForPreMatchRestart()
	}

	if len(state.rounds) != 2 {
		t.Errorf("rounds length = %d, want 2 (preserved on mid-match restart)", len(state.rounds))
	}
	if len(state.events) != 1 {
		t.Errorf("events length = %d, want 1 (preserved on mid-match restart)", len(state.events))
	}
	if state.currentRound != 9 {
		t.Errorf("currentRound = %d, want 9 (preserved)", state.currentRound)
	}
	if !strings.Contains(buf.String(), "mid-match restart detected") {
		t.Errorf("expected warn log containing 'mid-match restart detected', got: %q", buf.String())
	}
}
