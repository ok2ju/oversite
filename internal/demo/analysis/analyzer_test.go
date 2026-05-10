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
	Number        int `json:"number"`
	StartTick     int `json:"start_tick"`
	FreezeEndTick int `json:"freeze_end_tick"`
	EndTick       int `json:"end_tick"`
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
		rounds[i] = demo.RoundData{
			Number:        r.Number,
			StartTick:     r.StartTick,
			FreezeEndTick: r.FreezeEndTick,
			EndTick:       r.EndTick,
		}
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
