package contacts

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
)

func TestBuildEnemyTeam_TwoSides(t *testing.T) {
	roster := []demo.RoundParticipant{
		{SteamID: "1", TeamSide: "CT"},
		{SteamID: "2", TeamSide: "CT"},
		{SteamID: "3", TeamSide: "T"},
		{SteamID: "4", TeamSide: "T"},
	}
	got := buildEnemyTeam(roster)
	if len(got) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(got))
	}
	if got["1"] != "CT" || got["3"] != "T" {
		t.Errorf("team mapping wrong: %v", got)
	}
}

func TestBuildEnemyTeam_SkipsEmptySteamID(t *testing.T) {
	roster := []demo.RoundParticipant{
		{SteamID: "", TeamSide: "T"},
		{SteamID: "5", TeamSide: "T"},
	}
	got := buildEnemyTeam(roster)
	if len(got) != 1 {
		t.Errorf("empty steam IDs should be skipped, got %v", got)
	}
}

func TestPartitionEventsByRound(t *testing.T) {
	events := []demo.GameEvent{
		{Tick: 1, RoundNumber: 1, Type: "kill"},
		{Tick: 2, RoundNumber: 1, Type: "kill"},
		{Tick: 3, RoundNumber: 2, Type: "kill"},
		{Tick: 4, RoundNumber: 0, Type: "kill"}, // warmup
	}
	got := partitionEventsByRound(events)
	if len(got[1]) != 2 || len(got[2]) != 1 {
		t.Errorf("partition wrong: %v", got)
	}
	if _, exists := got[0]; exists {
		t.Errorf("round 0 should not be partitioned")
	}
}

func TestPartitionVisibilityByRound(t *testing.T) {
	vis := []demo.VisibilityChange{
		{RoundNumber: 1, Tick: 100, State: 1},
		{RoundNumber: 1, Tick: 200, State: 0},
		{RoundNumber: 2, Tick: 300, State: 1},
	}
	got := partitionVisibilityByRound(vis)
	if len(got[1]) != 2 || len(got[2]) != 1 {
		t.Errorf("partition wrong: %v", got)
	}
}

func TestDeriveAliveRange_DiesMidRound(t *testing.T) {
	round := demo.RoundData{Number: 1, FreezeEndTick: 100, EndTick: 5000}
	events := []demo.GameEvent{
		{Tick: 500, RoundNumber: 1, Type: "kill", AttackerSteamID: "E1", VictimSteamID: "S_P"},
	}
	got := deriveAliveRange("S_P", round, events)
	if got.SpawnTick != 100 || got.DeathTick != 500 {
		t.Errorf("AliveRange = %+v, want {100, 500}", got)
	}
}

func TestDeriveAliveRange_SurvivesRound(t *testing.T) {
	round := demo.RoundData{Number: 1, FreezeEndTick: 100, EndTick: 5000}
	events := []demo.GameEvent{
		{Tick: 500, RoundNumber: 1, Type: "kill", AttackerSteamID: "S_P", VictimSteamID: "E1"},
	}
	got := deriveAliveRange("S_P", round, events)
	if got.SpawnTick != 100 || got.DeathTick != 0 {
		t.Errorf("AliveRange = %+v, want {100, 0}", got)
	}
}

func TestPostWindowKills_RespectsRoundEnd(t *testing.T) {
	events := []demo.GameEvent{
		{Tick: 100, Type: "kill"},
		{Tick: 200, Type: "kill"},
		{Tick: 350, Type: "kill"},
		{Tick: 500, Type: "kill"},
	}
	// tLast=100, roundEnd=300: window is (100, min(100+320, 300)] = (100, 300].
	got := postWindowKills(events, 100, 300)
	if len(got) != 1 {
		t.Fatalf("expected 1 kill in clamped window, got %d", len(got))
	}
	if got[0].Tick != 200 {
		t.Errorf("expected tick 200, got %d", got[0].Tick)
	}
}

func TestPostWindowKills_TradeWindowClamps(t *testing.T) {
	events := []demo.GameEvent{
		{Tick: 100, Type: "kill"},
		{Tick: 200, Type: "kill"},
		{Tick: 500, Type: "kill"}, // outside TradeWindowTicks=320
	}
	got := postWindowKills(events, 100, 10000)
	if len(got) != 1 || got[0].Tick != 200 {
		t.Errorf("expected only tick 200 within trade window, got %+v", got)
	}
}

func TestIsHumanSubject_BotName(t *testing.T) {
	if isHumanSubject(demo.RoundParticipant{SteamID: "0", PlayerName: "BOT Cooper", TeamSide: "T"}) {
		t.Errorf("steam 0 should be filtered")
	}
	if isHumanSubject(demo.RoundParticipant{SteamID: "12345", PlayerName: "BOT Cooper", TeamSide: "T"}) {
		t.Errorf("BOT prefix should be filtered")
	}
}

func TestIsHumanSubject_RegularPlayer(t *testing.T) {
	if !isHumanSubject(demo.RoundParticipant{SteamID: "12345", PlayerName: "Alice", TeamSide: "T"}) {
		t.Errorf("regular player should pass")
	}
}

func TestRoundForTick(t *testing.T) {
	rounds := []demo.RoundData{
		{Number: 1, StartTick: 0, EndTick: 1000},
		{Number: 2, StartTick: 1100, EndTick: 2000},
	}
	if got := roundForTick(500, rounds); got != 1 {
		t.Errorf("expected round 1, got %d", got)
	}
	if got := roundForTick(1500, rounds); got != 2 {
		t.Errorf("expected round 2, got %d", got)
	}
	if got := roundForTick(1050, rounds); got != 0 {
		t.Errorf("expected round 0 for gap, got %d", got)
	}
}

func TestAliveAtTick_Nil(t *testing.T) {
	got := aliveAtTick(nil, 100)
	if len(got) != 0 {
		t.Errorf("nil perRound should yield empty set, got %v", got)
	}
}
