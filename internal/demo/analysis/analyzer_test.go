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

func TestRunMatchSummary_Golden(t *testing.T) {
	var input fixtureInput
	testutil.LoadFixture(t, "analysis/trades_summary/input.json", &input)

	got, err := analysis.RunMatchSummary(input.toParseResult(), nil)
	if err != nil {
		t.Fatalf("analysis.RunMatchSummary: %v", err)
	}

	encoded, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("marshal summary rows: %v", err)
	}
	encoded = append(encoded, '\n')

	testutil.CompareGolden(t, "analysis/trades_summary/expected.golden.json", encoded)
}

func TestPersistMatchSummary_IsIdempotent(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	d, err := q.CreateDemo(ctx, store.CreateDemoParams{
		MapName:  "de_dust2",
		FilePath: "/tmp/persist_match_summary.dem",
		FileSize: 1,
		Status:   "imported",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}

	first := []analysis.MatchSummaryRow{
		{SteamID: "alice", OverallScore: 50, TradePct: 0.5, AvgTradeTicks: 64},
		{SteamID: "bob", OverallScore: 0, TradePct: 0, AvgTradeTicks: 0},
	}
	if err := analysis.PersistMatchSummary(ctx, db, d.ID, first); err != nil {
		t.Fatalf("PersistMatchSummary (first): %v", err)
	}

	// Re-running with the same input must converge on the same rows
	// (idempotent upsert, no duplicate (demo, steam) entries).
	if err := analysis.PersistMatchSummary(ctx, db, d.ID, first); err != nil {
		t.Fatalf("PersistMatchSummary (re-run): %v", err)
	}

	aliceRow, err := q.GetPlayerMatchAnalysis(ctx, store.GetPlayerMatchAnalysisParams{
		DemoID:  d.ID,
		SteamID: "alice",
	})
	if err != nil {
		t.Fatalf("GetPlayerMatchAnalysis(alice): %v", err)
	}
	if aliceRow.OverallScore != 50 {
		t.Errorf("alice.OverallScore = %d, want 50", aliceRow.OverallScore)
	}
	if aliceRow.TradePct != 0.5 {
		t.Errorf("alice.TradePct = %v, want 0.5", aliceRow.TradePct)
	}

	// Replace alice with new metrics; bob should drop out (delete-then-upsert).
	second := []analysis.MatchSummaryRow{
		{SteamID: "alice", OverallScore: 80, TradePct: 0.8, AvgTradeTicks: 32},
	}
	if err := analysis.PersistMatchSummary(ctx, db, d.ID, second); err != nil {
		t.Fatalf("PersistMatchSummary (second): %v", err)
	}

	aliceRow, err = q.GetPlayerMatchAnalysis(ctx, store.GetPlayerMatchAnalysisParams{
		DemoID:  d.ID,
		SteamID: "alice",
	})
	if err != nil {
		t.Fatalf("GetPlayerMatchAnalysis(alice after second): %v", err)
	}
	if aliceRow.OverallScore != 80 {
		t.Errorf("alice.OverallScore after rerun = %d, want 80", aliceRow.OverallScore)
	}
	if aliceRow.TradePct != 0.8 {
		t.Errorf("alice.TradePct after rerun = %v, want 0.8", aliceRow.TradePct)
	}

	if _, err := q.GetPlayerMatchAnalysis(ctx, store.GetPlayerMatchAnalysisParams{
		DemoID:  d.ID,
		SteamID: "bob",
	}); err == nil {
		t.Errorf("expected bob's row to be wiped, but got a row")
	}

	// Empty batch wipes the demo's rows.
	if err := analysis.PersistMatchSummary(ctx, db, d.ID, nil); err != nil {
		t.Fatalf("PersistMatchSummary (empty): %v", err)
	}
	if _, err := q.GetPlayerMatchAnalysis(ctx, store.GetPlayerMatchAnalysisParams{
		DemoID:  d.ID,
		SteamID: "alice",
	}); err == nil {
		t.Errorf("expected alice's row to be wiped after empty persist, but got a row")
	}
}

func TestRunPlayerRoundAnalysis_Golden(t *testing.T) {
	var input fixtureInput
	testutil.LoadFixture(t, "analysis/round_trades/input.json", &input)

	got, err := analysis.RunPlayerRoundAnalysis(input.toParseResult(), nil)
	if err != nil {
		t.Fatalf("analysis.RunPlayerRoundAnalysis: %v", err)
	}

	encoded, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("marshal round rows: %v", err)
	}
	encoded = append(encoded, '\n')

	testutil.CompareGolden(t, "analysis/round_trades/expected.golden.json", encoded)
}

// TestRunPlayerRoundAnalysis_AggregatesToMatchSummary asserts that summing the
// per-round trade counts back to the per-player level reproduces
// RunMatchSummary's TradePct. Catches accidental divergence between
// computeRoundTrades and computeTradesSummary on the predicate boundary
// (friendly fire, self-kill, world).
func TestRunPlayerRoundAnalysis_AggregatesToMatchSummary(t *testing.T) {
	var input fixtureInput
	testutil.LoadFixture(t, "analysis/trades_summary/input.json", &input)
	parse := input.toParseResult()

	matchRows, err := analysis.RunMatchSummary(parse, nil)
	if err != nil {
		t.Fatalf("RunMatchSummary: %v", err)
	}
	roundRows, err := analysis.RunPlayerRoundAnalysis(parse, nil)
	if err != nil {
		t.Fatalf("RunPlayerRoundAnalysis: %v", err)
	}

	// We rebuild trade_pct from the per-round rows by replaying the same
	// events tick-by-tick but bucketed by (player, round). The match-level
	// row's TradePct is traded_deaths / own_deaths across the whole match;
	// the round breakdown collapses back to that ratio iff the predicates
	// agree row-for-row.
	type counts struct{ owns, trades int }
	tickRate := parse.Header.TickRate
	if tickRate <= 0 {
		tickRate = 64
	}
	windowTicks := int(analysis.TradeWindowSeconds * tickRate)
	matchCounts := make(map[string]*counts, len(matchRows))
	for i, ev := range parse.Events {
		if ev.Type != "kill" || ev.VictimSteamID == "" || ev.AttackerSteamID == "" {
			continue
		}
		if ev.AttackerSteamID == ev.VictimSteamID {
			continue
		}
		k, _ := ev.ExtraData.(*demo.KillExtra)
		if k == nil || k.VictimTeam == "" {
			continue
		}
		if k.AttackerTeam != "" && k.AttackerTeam == k.VictimTeam {
			continue
		}
		c, ok := matchCounts[ev.VictimSteamID]
		if !ok {
			c = &counts{}
			matchCounts[ev.VictimSteamID] = c
		}
		c.owns++
		// Was this death traded? Forward walk identical to trades.go.
		limit := ev.Tick + windowTicks
		for j := i + 1; j < len(parse.Events); j++ {
			next := parse.Events[j]
			if next.Tick > limit {
				break
			}
			if next.Type != "kill" || next.AttackerSteamID == "" {
				continue
			}
			if next.AttackerSteamID == ev.VictimSteamID {
				continue
			}
			if next.VictimSteamID != ev.AttackerSteamID {
				continue
			}
			nk, _ := next.ExtraData.(*demo.KillExtra)
			if nk == nil || nk.AttackerTeam != k.VictimTeam {
				continue
			}
			c.trades++
			break
		}
	}

	// Re-derive per-player totals from the round rows by summing
	// (round_pct * 1) across the rows — but PlayerRoundRow only stores
	// trade_pct, not the raw counts. Instead, re-run RunPlayerRoundAnalysis's
	// fixture path: each round contributes 1 own death (the fixture has at
	// most 1 eligible death per (player, round) pair). When that doesn't
	// hold the test would falsely pass — assert the precondition explicitly.
	for _, row := range roundRows {
		// trade_pct in {0, 1} for fixtures with one death per (player, round).
		if row.TradePct != 0 && row.TradePct != 1 {
			t.Fatalf("fixture invariant broken: row %+v has fractional trade_pct; this test assumes one death per (player, round) bucket", row)
		}
	}

	// Now collapse round rows: each row contributes 1 own death; trade_pct == 1
	// indicates a traded death.
	derived := make(map[string]*counts, len(roundRows))
	for _, row := range roundRows {
		c, ok := derived[row.SteamID]
		if !ok {
			c = &counts{}
			derived[row.SteamID] = c
		}
		c.owns++
		if row.TradePct == 1 {
			c.trades++
		}
	}

	for _, mr := range matchRows {
		matchTotal := matchCounts[mr.SteamID]
		if matchTotal == nil {
			t.Errorf("matchRows references %q but raw event walk found no eligible deaths", mr.SteamID)
			continue
		}
		if matchTotal.owns == 0 {
			continue
		}
		want := float64(matchTotal.trades) / float64(matchTotal.owns)
		if mr.TradePct != want {
			t.Errorf("match TradePct mismatch for %q: row=%v, recomputed=%v", mr.SteamID, mr.TradePct, want)
		}

		got := derived[mr.SteamID]
		if got == nil {
			if matchTotal.owns > 0 {
				t.Errorf("round rows missing %q (match has %d eligible deaths)", mr.SteamID, matchTotal.owns)
			}
			continue
		}
		gotPct := float64(got.trades) / float64(got.owns)
		if gotPct != mr.TradePct {
			t.Errorf("round-aggregate TradePct mismatch for %q: rounds=%v, match=%v", mr.SteamID, gotPct, mr.TradePct)
		}
	}
}

func TestPersistPlayerRoundAnalysis_IsIdempotent(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	d, err := q.CreateDemo(ctx, store.CreateDemoParams{
		MapName:  "de_dust2",
		FilePath: "/tmp/persist_round_analysis.dem",
		FileSize: 1,
		Status:   "imported",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}

	first := []analysis.PlayerRoundRow{
		{SteamID: "alice", RoundNumber: 1, TradePct: 1.0},
		{SteamID: "alice", RoundNumber: 2, TradePct: 0.0},
		{SteamID: "bob", RoundNumber: 1, TradePct: 0.5},
	}
	if err := analysis.PersistPlayerRoundAnalysis(ctx, db, d.ID, first); err != nil {
		t.Fatalf("PersistPlayerRoundAnalysis (first): %v", err)
	}

	// Re-running with the same input must converge on the same rows
	// (idempotent upsert, no duplicates on (demo, steam, round)).
	if err := analysis.PersistPlayerRoundAnalysis(ctx, db, d.ID, first); err != nil {
		t.Fatalf("PersistPlayerRoundAnalysis (re-run): %v", err)
	}

	aliceRows, err := q.GetPlayerRoundAnalysisByDemoAndPlayer(ctx, store.GetPlayerRoundAnalysisByDemoAndPlayerParams{
		DemoID:  d.ID,
		SteamID: "alice",
	})
	if err != nil {
		t.Fatalf("GetPlayerRoundAnalysisByDemoAndPlayer(alice): %v", err)
	}
	if len(aliceRows) != 2 {
		t.Fatalf("expected 2 alice rows, got %d", len(aliceRows))
	}
	if aliceRows[0].RoundNumber != 1 || aliceRows[0].TradePct != 1.0 {
		t.Errorf("aliceRows[0] = %+v, want {round=1, trade_pct=1.0}", aliceRows[0])
	}
	if aliceRows[1].RoundNumber != 2 || aliceRows[1].TradePct != 0.0 {
		t.Errorf("aliceRows[1] = %+v, want {round=2, trade_pct=0.0}", aliceRows[1])
	}

	// Replace alice with new metrics; bob should drop out (delete-then-upsert).
	second := []analysis.PlayerRoundRow{
		{SteamID: "alice", RoundNumber: 1, TradePct: 0.25},
	}
	if err := analysis.PersistPlayerRoundAnalysis(ctx, db, d.ID, second); err != nil {
		t.Fatalf("PersistPlayerRoundAnalysis (second): %v", err)
	}

	aliceRows, err = q.GetPlayerRoundAnalysisByDemoAndPlayer(ctx, store.GetPlayerRoundAnalysisByDemoAndPlayerParams{
		DemoID:  d.ID,
		SteamID: "alice",
	})
	if err != nil {
		t.Fatalf("GetPlayerRoundAnalysisByDemoAndPlayer(alice after second): %v", err)
	}
	if len(aliceRows) != 1 {
		t.Fatalf("expected 1 alice row after second persist, got %d", len(aliceRows))
	}
	if aliceRows[0].TradePct != 0.25 {
		t.Errorf("aliceRows[0].TradePct = %v, want 0.25", aliceRows[0].TradePct)
	}

	bobRows, err := q.GetPlayerRoundAnalysisByDemoAndPlayer(ctx, store.GetPlayerRoundAnalysisByDemoAndPlayerParams{
		DemoID:  d.ID,
		SteamID: "bob",
	})
	if err != nil {
		t.Fatalf("GetPlayerRoundAnalysisByDemoAndPlayer(bob after second): %v", err)
	}
	if len(bobRows) != 0 {
		t.Errorf("expected bob's rows wiped, got %d", len(bobRows))
	}

	// Empty batch wipes the demo's rows.
	if err := analysis.PersistPlayerRoundAnalysis(ctx, db, d.ID, nil); err != nil {
		t.Fatalf("PersistPlayerRoundAnalysis (empty): %v", err)
	}
	aliceRows, err = q.GetPlayerRoundAnalysisByDemoAndPlayer(ctx, store.GetPlayerRoundAnalysisByDemoAndPlayerParams{
		DemoID:  d.ID,
		SteamID: "alice",
	})
	if err != nil {
		t.Fatalf("GetPlayerRoundAnalysisByDemoAndPlayer(alice after empty): %v", err)
	}
	if len(aliceRows) != 0 {
		t.Errorf("expected 0 alice rows after empty persist, got %d", len(aliceRows))
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
