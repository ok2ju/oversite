package analysis_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/analysis"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

// fixtureInput is the disk shape of testdata/analysis/<name>/input.json. It is
// flatter than demo.ParseResult on purpose — golden fixtures should be hand-
// editable, and the demo extras (`*KillExtra` etc.) carry pointers we don't
// want to serialize.
type fixtureInput struct {
	TickRate float64        `json:"tick_rate"`
	Rounds   []fixtureRound `json:"rounds"`
	Events   []fixtureEvent `json:"events"`
}

type fixtureRound struct {
	Number        int             `json:"number"`
	StartTick     int             `json:"start_tick"`
	FreezeEndTick int             `json:"freeze_end_tick"`
	EndTick       int             `json:"end_tick"`
	Roster        []fixtureRoster `json:"roster,omitempty"`
}

// fixtureRoster mirrors demo.RoundParticipant. Only the freeze-end inventory
// is needed by the analyzer rules; the other fields are kept for parity with
// the parser shape so tightening the production struct in the future doesn't
// silently empty the test rosters.
type fixtureRoster struct {
	SteamID    string `json:"steam_id"`
	PlayerName string `json:"player_name"`
	TeamSide   string `json:"team_side"`
	Inventory  string `json:"inventory"`
}

type fixtureEvent struct {
	Tick            int    `json:"tick"`
	RoundNumber     int    `json:"round_number"`
	Type            string `json:"type"`
	AttackerSteamID string `json:"attacker_steam_id"`
	VictimSteamID   string `json:"victim_steam_id"`
	AttackerTeam    string `json:"attacker_team"`
	VictimTeam      string `json:"victim_team"`
	Weapon          string `json:"weapon"`
}

func (fi fixtureInput) toParseResult() *demo.ParseResult {
	rounds := make([]demo.RoundData, len(fi.Rounds))
	for i, r := range fi.Rounds {
		rd := demo.RoundData{
			Number:        r.Number,
			StartTick:     r.StartTick,
			FreezeEndTick: r.FreezeEndTick,
			EndTick:       r.EndTick,
		}
		if len(r.Roster) > 0 {
			rd.Roster = make([]demo.RoundParticipant, len(r.Roster))
			for j, rp := range r.Roster {
				rd.Roster[j] = demo.RoundParticipant{
					SteamID:    rp.SteamID,
					PlayerName: rp.PlayerName,
					TeamSide:   rp.TeamSide,
					Inventory:  rp.Inventory,
				}
			}
		}
		rounds[i] = rd
	}
	events := make([]demo.GameEvent, len(fi.Events))
	for i, e := range fi.Events {
		ev := demo.GameEvent{
			Tick:            e.Tick,
			RoundNumber:     e.RoundNumber,
			Type:            e.Type,
			AttackerSteamID: e.AttackerSteamID,
			VictimSteamID:   e.VictimSteamID,
			Weapon:          e.Weapon,
		}
		if e.Type == "kill" {
			ev.ExtraData = &demo.KillExtra{
				AttackerTeam: e.AttackerTeam,
				VictimTeam:   e.VictimTeam,
			}
		}
		events[i] = ev
	}
	return &demo.ParseResult{
		Header: demo.MatchHeader{TickRate: fi.TickRate},
		Rounds: rounds,
		Events: events,
	}
}

func TestRun_NoTradeDeath_Golden(t *testing.T) {
	var input fixtureInput
	testutil.LoadFixture(t, "analysis/no_trade_death/input.json", &input)

	got, err := analysis.Run(input.toParseResult(), nil)
	if err != nil {
		t.Fatalf("analysis.Run: %v", err)
	}

	encoded, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("marshal mistakes: %v", err)
	}
	encoded = append(encoded, '\n')

	testutil.CompareGolden(t, "analysis/no_trade_death/expected.golden.json", encoded)
}

func TestRun_DiedWithUtilUnused_Golden(t *testing.T) {
	var input fixtureInput
	testutil.LoadFixture(t, "analysis/died_with_util_unused/input.json", &input)

	got, err := analysis.Run(input.toParseResult(), nil)
	if err != nil {
		t.Fatalf("analysis.Run: %v", err)
	}

	encoded, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("marshal mistakes: %v", err)
	}
	encoded = append(encoded, '\n')

	testutil.CompareGolden(t, "analysis/died_with_util_unused/expected.golden.json", encoded)
}

// TestRun_BothRules_OrderedByTick asserts that when both rules emit, the
// combined slice is stably ordered by (Tick, SteamID) regardless of which
// rule produced each entry. The contract matters because additional rules in
// later slices must not silently reshuffle persisted ordering — readers
// (frontend side panel, scoring composites) walk the slice top-to-bottom.
func TestRun_BothRules_OrderedByTick(t *testing.T) {
	// Two T players. alice dies untraded at tick 200 (no_trade_death). bob dies
	// to the world at tick 6000 with a smoke he never threw
	// (died_with_util_unused). The two events live in different rounds so the
	// rules don't collide on the same (player, round) pair.
	rounds := []demo.RoundData{
		{
			Number:        1,
			StartTick:     0,
			FreezeEndTick: 100,
			EndTick:       5000,
			Roster: []demo.RoundParticipant{
				{SteamID: "alice", TeamSide: "T", Inventory: "AK-47"},
				{SteamID: "carol", TeamSide: "CT", Inventory: "M4A1"},
			},
		},
		{
			Number:        2,
			StartTick:     5001,
			FreezeEndTick: 5100,
			EndTick:       10000,
			Roster: []demo.RoundParticipant{
				{SteamID: "bob", TeamSide: "T", Inventory: "AK-47,Smokegrenade"},
				{SteamID: "carol", TeamSide: "CT", Inventory: "M4A1"},
			},
		},
	}
	events := []demo.GameEvent{
		{
			Tick:            200,
			RoundNumber:     1,
			Type:            "kill",
			AttackerSteamID: "carol",
			VictimSteamID:   "alice",
			Weapon:          "m4a1",
			ExtraData: &demo.KillExtra{
				AttackerTeam: "CT",
				VictimTeam:   "T",
			},
		},
		{
			Tick:            6000,
			RoundNumber:     2,
			Type:            "kill",
			AttackerSteamID: "",
			VictimSteamID:   "bob",
			Weapon:          "world",
			ExtraData: &demo.KillExtra{
				AttackerTeam: "",
				VictimTeam:   "T",
			},
		},
	}
	result := &demo.ParseResult{
		Header: demo.MatchHeader{TickRate: 64},
		Rounds: rounds,
		Events: events,
	}

	got, err := analysis.Run(result, nil)
	if err != nil {
		t.Fatalf("analysis.Run: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 mistakes, got %d (%+v)", len(got), got)
	}
	if got[0].Tick != 200 || got[0].Kind != string(analysis.MistakeKindNoTradeDeath) {
		t.Errorf("got[0] = {Tick:%d, Kind:%q}, want {200, no_trade_death}", got[0].Tick, got[0].Kind)
	}
	if got[1].Tick != 6000 || got[1].Kind != string(analysis.MistakeKindDiedWithUtilUnused) {
		t.Errorf("got[1] = {Tick:%d, Kind:%q}, want {6000, died_with_util_unused}", got[1].Tick, got[1].Kind)
	}
}

func TestRun_NilResult(t *testing.T) {
	got, err := analysis.Run(nil, nil)
	if err != nil {
		t.Fatalf("analysis.Run(nil): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty mistakes for nil result, got %d", len(got))
	}
}

func TestPersist_IsIdempotent(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	d, err := q.CreateDemo(ctx, store.CreateDemoParams{
		MapName:  "de_dust2",
		FilePath: "/tmp/persist.dem",
		FileSize: 1,
		Status:   "imported",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}

	first := []analysis.Mistake{
		{SteamID: "alice", RoundNumber: 1, Tick: 100, Kind: "no_trade_death", Extras: map[string]any{"killer_steam_id": "carol"}},
		{SteamID: "alice", RoundNumber: 2, Tick: 6000, Kind: "no_trade_death"},
	}
	if err := analysis.Persist(ctx, db, d.ID, first); err != nil {
		t.Fatalf("Persist (first): %v", err)
	}

	second := []analysis.Mistake{
		{SteamID: "bob", RoundNumber: 3, Tick: 12345, Kind: "no_trade_death", Extras: map[string]any{"killer_steam_id": "dave"}},
	}
	if err := analysis.Persist(ctx, db, d.ID, second); err != nil {
		t.Fatalf("Persist (second): %v", err)
	}

	aliceRows, err := q.ListAnalysisMistakesByDemoIDAndSteamID(ctx, store.ListAnalysisMistakesByDemoIDAndSteamIDParams{
		DemoID:  d.ID,
		SteamID: "alice",
	})
	if err != nil {
		t.Fatalf("ListAnalysisMistakesByDemoIDAndSteamID(alice): %v", err)
	}
	if len(aliceRows) != 0 {
		t.Errorf("expected alice's first-batch rows to be wiped, got %d", len(aliceRows))
	}

	bobRows, err := q.ListAnalysisMistakesByDemoIDAndSteamID(ctx, store.ListAnalysisMistakesByDemoIDAndSteamIDParams{
		DemoID:  d.ID,
		SteamID: "bob",
	})
	if err != nil {
		t.Fatalf("ListAnalysisMistakesByDemoIDAndSteamID(bob): %v", err)
	}
	if len(bobRows) != 1 {
		t.Fatalf("expected 1 bob row after second persist, got %d", len(bobRows))
	}
	if bobRows[0].Tick != 12345 {
		t.Errorf("bob.Tick = %d, want 12345", bobRows[0].Tick)
	}
	if bobRows[0].Kind != "no_trade_death" {
		t.Errorf("bob.Kind = %q, want no_trade_death", bobRows[0].Kind)
	}
}

func TestPersist_EmptyMistakesStillWipes(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	d, err := q.CreateDemo(ctx, store.CreateDemoParams{
		MapName:  "de_inferno",
		FilePath: "/tmp/wipe.dem",
		FileSize: 1,
		Status:   "imported",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}

	// Seed one mistake, then persist an empty batch — the seeded row should be
	// gone so a re-parse that produces zero mistakes converges to the empty
	// state.
	if err := analysis.Persist(ctx, db, d.ID, []analysis.Mistake{
		{SteamID: "alice", RoundNumber: 1, Tick: 100, Kind: "no_trade_death"},
	}); err != nil {
		t.Fatalf("Persist (seed): %v", err)
	}
	if err := analysis.Persist(ctx, db, d.ID, nil); err != nil {
		t.Fatalf("Persist (empty): %v", err)
	}

	rows, err := q.ListAnalysisMistakesByDemoIDAndSteamID(ctx, store.ListAnalysisMistakesByDemoIDAndSteamIDParams{
		DemoID:  d.ID,
		SteamID: "alice",
	})
	if err != nil {
		t.Fatalf("ListAnalysisMistakesByDemoIDAndSteamID: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows after empty persist, got %d", len(rows))
	}
}
