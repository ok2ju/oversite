package demo

import (
	"strings"
	"testing"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func TestShouldSampleTick(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tick     int
		interval int
		want     bool
	}{
		{"tick 0 interval 4", 0, 4, true},
		{"tick 1 interval 4", 1, 4, false},
		{"tick 4 interval 4", 4, 4, true},
		{"tick 8 interval 4", 8, 4, true},
		{"tick 3 interval 4", 3, 4, false},
		{"tick 0 interval 1", 0, 1, true},
		{"tick 7 interval 1", 7, 1, true},
		{"tick 64 interval 64", 64, 64, true},
		{"tick 63 interval 64", 63, 64, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldSampleTick(tt.tick, tt.interval); got != tt.want {
				t.Errorf("shouldSampleTick(%d, %d) = %v, want %v", tt.tick, tt.interval, got, tt.want)
			}
		})
	}
}

func TestIsOvertime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		roundNum int
		want     bool
	}{
		{"round 1", 1, false},
		{"round 12", 12, false},
		{"round 24", 24, false},
		{"round 25 (OT start)", 25, true},
		{"round 30", 30, true},
		{"round 48", 48, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isOvertime(tt.roundNum); got != tt.want {
				t.Errorf("isOvertime(%d) = %v, want %v", tt.roundNum, got, tt.want)
			}
		})
	}
}

func TestShouldSkipPlayer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		steamID64   uint64
		isBot       bool
		includeBots bool
		want        bool
	}{
		{"human player, no bots", 76561198012345678, false, false, false},
		{"human player, include bots", 76561198012345678, false, true, false},
		{"bot, no bots", 0, true, false, true},
		{"bot, include bots", 0, true, true, false},
		{"steamID 0 not bot flag, no bots", 0, false, false, true},
		{"steamID 0 not bot flag, include bots", 0, false, true, false},
		{"bot with steamID, no bots", 76561198012345678, true, false, true},
		{"bot with steamID, include bots", 76561198012345678, true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			player := &common.Player{
				SteamID64: tt.steamID64,
				IsBot:     tt.isBot,
			}
			if got := shouldSkipPlayer(player, tt.includeBots); got != tt.want {
				t.Errorf("shouldSkipPlayer(steamID=%d, isBot=%v, include=%v) = %v, want %v",
					tt.steamID64, tt.isBot, tt.includeBots, got, tt.want)
			}
		})
	}
}

func TestShouldSkipPlayer_Nil(t *testing.T) {
	t.Parallel()
	if got := shouldSkipPlayer(nil, false); got != true {
		t.Errorf("shouldSkipPlayer(nil, false) = %v, want true", got)
	}
	if got := shouldSkipPlayer(nil, true); got != true {
		t.Errorf("shouldSkipPlayer(nil, true) = %v, want true", got)
	}
}

func TestNewDemoParser_Defaults(t *testing.T) {
	t.Parallel()

	dp := NewDemoParser()

	if dp.tickInterval != 4 {
		t.Errorf("default tickInterval = %d, want 4", dp.tickInterval)
	}
	if !dp.skipWarmup {
		t.Error("default skipWarmup = false, want true")
	}
	if dp.includeBots {
		t.Error("default includeBots = true, want false")
	}
}

func TestNewDemoParser_Options(t *testing.T) {
	t.Parallel()

	dp := NewDemoParser(
		WithTickInterval(8),
		WithSkipWarmup(false),
		WithIncludeBots(true),
	)

	if dp.tickInterval != 8 {
		t.Errorf("tickInterval = %d, want 8", dp.tickInterval)
	}
	if dp.skipWarmup {
		t.Error("skipWarmup = true, want false")
	}
	if !dp.includeBots {
		t.Error("includeBots = false, want true")
	}
}

func TestParse_EmptyReader(t *testing.T) {
	t.Parallel()

	dp := NewDemoParser()
	_, err := dp.Parse(strings.NewReader(""))

	if err == nil {
		t.Fatal("Parse(empty reader) should return error")
	}
}

func TestParse_InvalidData(t *testing.T) {
	t.Parallel()

	dp := NewDemoParser()
	_, err := dp.Parse(strings.NewReader("not a demo file"))

	if err == nil {
		t.Fatal("Parse(invalid data) should return error")
	}
}

func TestTeamSideString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		team common.Team
		want string
	}{
		{"CT", common.TeamCounterTerrorists, "CT"},
		{"T", common.TeamTerrorists, "T"},
		{"spectators", common.TeamSpectators, ""},
		{"unassigned", common.TeamUnassigned, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := teamSideString(tt.team); got != tt.want {
				t.Errorf("teamSideString(%v) = %q, want %q", tt.team, got, tt.want)
			}
		})
	}
}

func TestBombsiteString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		site events.Bombsite
		want string
	}{
		{"site A", events.BombsiteA, "A"},
		{"site B", events.BombsiteB, "B"},
		{"unknown", events.BomsiteUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := bombsiteString(tt.site); got != tt.want {
				t.Errorf("bombsiteString(%v) = %q, want %q", tt.site, got, tt.want)
			}
		})
	}
}
