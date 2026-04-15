package demo

import (
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
