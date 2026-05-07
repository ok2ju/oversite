package demo_test

import (
	"context"
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

// createDemoForRounds creates a demo record for round ingestion tests.
func createDemoForRounds(t *testing.T, q *store.Queries) store.Demo {
	t.Helper()
	d, err := q.CreateDemo(context.Background(), store.CreateDemoParams{
		FilePath: "/test-rounds.dem",
		FileSize: 1000,
		Status:   "imported",
		MapName:  "de_dust2",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}
	return d
}

// syntheticParseResult builds a ParseResult with 3 rounds and events that
// produce player round stats via CalculatePlayerRoundStats.
func syntheticParseResult() *demo.ParseResult {
	return &demo.ParseResult{
		Rounds: []demo.RoundData{
			{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT", WinReason: "CTWin", CTScore: 1, TScore: 0},
			{Number: 2, StartTick: 1001, EndTick: 2000, WinnerSide: "T", WinReason: "TWin", CTScore: 1, TScore: 1},
			{Number: 3, StartTick: 2001, EndTick: 3000, WinnerSide: "CT", WinReason: "BombDefused", CTScore: 2, TScore: 1},
		},
		Events: []demo.GameEvent{
			// Round 1: hurt + kill so CalculatePlayerRoundStats produces data.
			{Tick: 100, RoundNumber: 1, Type: "player_hurt", AttackerSteamID: "steam1", VictimSteamID: "steam2", ExtraData: map[string]interface{}{"health_damage": 80, "attacker_name": "Player1", "attacker_team": "CT", "victim_name": "Player2", "victim_team": "T"}},
			{Tick: 200, RoundNumber: 1, Type: "kill", AttackerSteamID: "steam1", VictimSteamID: "steam2", Weapon: "ak47", ExtraData: map[string]interface{}{"headshot": true, "attacker_name": "Player1", "attacker_team": "CT", "victim_name": "Player2", "victim_team": "T"}},
			// Round 2: another kill.
			{Tick: 1100, RoundNumber: 2, Type: "kill", AttackerSteamID: "steam2", VictimSteamID: "steam1", Weapon: "awp", ExtraData: map[string]interface{}{"headshot": false, "attacker_name": "Player2", "attacker_team": "T", "victim_name": "Player1", "victim_team": "CT"}},
		},
	}
}

func TestIngestRounds_Basic(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	d := createDemoForRounds(t, q)

	result := syntheticParseResult()
	roundMap, err := demo.IngestRounds(ctx, db, d.ID, result)
	if err != nil {
		t.Fatalf("IngestRounds: %v", err)
	}

	// Verify roundMap has 3 entries.
	if len(roundMap) != 3 {
		t.Fatalf("roundMap length = %d, want 3", len(roundMap))
	}

	// Verify rounds persisted in DB.
	rounds, err := q.GetRoundsByDemoID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetRoundsByDemoID: %v", err)
	}
	if len(rounds) != 3 {
		t.Fatalf("DB round count = %d, want 3", len(rounds))
	}

	// Check field values for round 1.
	r1 := rounds[0]
	if r1.RoundNumber != 1 {
		t.Errorf("RoundNumber = %d, want 1", r1.RoundNumber)
	}
	if r1.StartTick != 0 {
		t.Errorf("StartTick = %d, want 0", r1.StartTick)
	}
	if r1.EndTick != 1000 {
		t.Errorf("EndTick = %d, want 1000", r1.EndTick)
	}
	if r1.WinnerSide != "CT" {
		t.Errorf("WinnerSide = %q, want %q", r1.WinnerSide, "CT")
	}
	if r1.WinReason != "CTWin" {
		t.Errorf("WinReason = %q, want %q", r1.WinReason, "CTWin")
	}
	if r1.CtScore != 1 {
		t.Errorf("CtScore = %d, want 1", r1.CtScore)
	}
	if r1.TScore != 0 {
		t.Errorf("TScore = %d, want 0", r1.TScore)
	}

	// Check round 2 winner side.
	r2 := rounds[1]
	if r2.WinnerSide != "T" {
		t.Errorf("Round 2 WinnerSide = %q, want %q", r2.WinnerSide, "T")
	}

	// Check round 3 win reason.
	r3 := rounds[2]
	if r3.WinReason != "BombDefused" {
		t.Errorf("Round 3 WinReason = %q, want %q", r3.WinReason, "BombDefused")
	}
}

func TestIngestRounds_PlayerRounds(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	d := createDemoForRounds(t, q)

	result := syntheticParseResult()
	roundMap, err := demo.IngestRounds(ctx, db, d.ID, result)
	if err != nil {
		t.Fatalf("IngestRounds: %v", err)
	}

	// Verify player rounds for round 1.
	round1ID := roundMap[1]
	playerRounds, err := q.GetPlayerRoundsByRoundID(ctx, round1ID)
	if err != nil {
		t.Fatalf("GetPlayerRoundsByRoundID: %v", err)
	}
	if len(playerRounds) != 2 {
		t.Fatalf("player round count for round 1 = %d, want 2", len(playerRounds))
	}

	// Find steam1 (attacker in round 1: 1 kill, 0 deaths, 80 damage, 1 headshot_kill, first_kill=1).
	var steam1PR *store.PlayerRound
	for i := range playerRounds {
		if playerRounds[i].SteamID == "steam1" {
			steam1PR = &playerRounds[i]
			break
		}
	}
	if steam1PR == nil {
		t.Fatal("steam1 player round not found")
	}
	if steam1PR.Kills != 1 {
		t.Errorf("steam1 Kills = %d, want 1", steam1PR.Kills)
	}
	if steam1PR.Deaths != 0 {
		t.Errorf("steam1 Deaths = %d, want 0", steam1PR.Deaths)
	}
	if steam1PR.Damage != 80 {
		t.Errorf("steam1 Damage = %d, want 80", steam1PR.Damage)
	}
	if steam1PR.HeadshotKills != 1 {
		t.Errorf("steam1 HeadshotKills = %d, want 1", steam1PR.HeadshotKills)
	}
	if steam1PR.FirstKill != 1 {
		t.Errorf("steam1 FirstKill = %d, want 1 (true)", steam1PR.FirstKill)
	}
	if steam1PR.FirstDeath != 0 {
		t.Errorf("steam1 FirstDeath = %d, want 0 (false)", steam1PR.FirstDeath)
	}

	// Find steam2 (victim in round 1: 0 kills, 1 death, first_death=1).
	var steam2PR *store.PlayerRound
	for i := range playerRounds {
		if playerRounds[i].SteamID == "steam2" {
			steam2PR = &playerRounds[i]
			break
		}
	}
	if steam2PR == nil {
		t.Fatal("steam2 player round not found")
	}
	if steam2PR.Kills != 0 {
		t.Errorf("steam2 Kills = %d, want 0", steam2PR.Kills)
	}
	if steam2PR.Deaths != 1 {
		t.Errorf("steam2 Deaths = %d, want 1", steam2PR.Deaths)
	}
	if steam2PR.FirstDeath != 1 {
		t.Errorf("steam2 FirstDeath = %d, want 1 (true)", steam2PR.FirstDeath)
	}
}

func TestIngestRounds_Idempotent(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	d := createDemoForRounds(t, q)

	result := syntheticParseResult()

	// First ingest.
	_, err := demo.IngestRounds(ctx, db, d.ID, result)
	if err != nil {
		t.Fatalf("first IngestRounds: %v", err)
	}

	// Second ingest — should replace, not duplicate.
	_, err = demo.IngestRounds(ctx, db, d.ID, result)
	if err != nil {
		t.Fatalf("second IngestRounds: %v", err)
	}

	rounds, err := q.GetRoundsByDemoID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetRoundsByDemoID: %v", err)
	}
	if len(rounds) != 3 {
		t.Errorf("DB round count after two ingests = %d, want 3", len(rounds))
	}
}

func TestIngestRounds_EmptyRounds(t *testing.T) {
	_, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	result := &demo.ParseResult{
		Rounds: nil,
	}

	roundMap, err := demo.IngestRounds(ctx, db, 999, result)
	if err != nil {
		t.Fatalf("IngestRounds empty: %v", err)
	}
	if roundMap != nil {
		t.Errorf("roundMap = %v, want nil", roundMap)
	}
}

// TestIngestRounds_TeamNames covers the parser → schema → binding flow for
// per-round clan names captured at RoundEnd.
func TestIngestRounds_TeamNames(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	d := createDemoForRounds(t, q)

	result := &demo.ParseResult{
		Rounds: []demo.RoundData{
			{
				Number:     1,
				StartTick:  0,
				EndTick:    1000,
				WinnerSide: "CT",
				WinReason:  "CTWin",
				CTScore:    1,
				TScore:     0,
				CTTeamName: "Astralis",
				TTeamName:  "NaVi",
			},
		},
	}

	if _, err := demo.IngestRounds(ctx, db, d.ID, result); err != nil {
		t.Fatalf("IngestRounds: %v", err)
	}

	rounds, err := q.GetRoundsByDemoID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetRoundsByDemoID: %v", err)
	}
	if len(rounds) != 1 {
		t.Fatalf("rounds length = %d, want 1", len(rounds))
	}
	if rounds[0].CtTeamName != "Astralis" {
		t.Errorf("ct_team_name = %q, want %q", rounds[0].CtTeamName, "Astralis")
	}
	if rounds[0].TTeamName != "NaVi" {
		t.Errorf("t_team_name = %q, want %q", rounds[0].TTeamName, "NaVi")
	}
}

func TestIngestRounds_RoundMapKeys(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	d := createDemoForRounds(t, q)

	result := syntheticParseResult()
	roundMap, err := demo.IngestRounds(ctx, db, d.ID, result)
	if err != nil {
		t.Fatalf("IngestRounds: %v", err)
	}

	// Verify keys match round numbers.
	for _, rd := range result.Rounds {
		id, ok := roundMap[rd.Number]
		if !ok {
			t.Errorf("roundMap missing key %d", rd.Number)
			continue
		}
		if id <= 0 {
			t.Errorf("roundMap[%d] = %d, want positive DB ID", rd.Number, id)
		}
	}

	// Verify all IDs are unique.
	seen := make(map[int64]bool)
	for num, id := range roundMap {
		if seen[id] {
			t.Errorf("duplicate DB ID %d for round %d", id, num)
		}
		seen[id] = true
	}
}
