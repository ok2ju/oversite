//go:build integration

package demo_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"

	"github.com/ok2ju/oversite/backend/internal/demo"
	"github.com/ok2ju/oversite/backend/internal/store"
	"github.com/ok2ju/oversite/backend/internal/testutil"
)

var (
	pgContainer testcontainers.Container
	pgConnURL   string
	testDB      *sql.DB
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var err error
	pgContainer, pgConnURL, err = testutil.PostgresContainer(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "starting postgres container: %v\n", err)
		os.Exit(1)
	}

	if err := testutil.RunMigrations(pgConnURL); err != nil {
		fmt.Fprintf(os.Stderr, "running migrations: %v\n", err)
		pgContainer.Terminate(ctx)
		os.Exit(1)
	}

	testDB, err = sql.Open("postgres", pgConnURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "opening db: %v\n", err)
		pgContainer.Terminate(ctx)
		os.Exit(1)
	}

	code := m.Run()

	testDB.Close()
	pgContainer.Terminate(context.Background())
	os.Exit(code)
}

func createTestUserAndDemo(t *testing.T, q *store.Queries) (store.User, store.Demo) {
	t.Helper()
	ctx := context.Background()

	user, err := q.CreateUser(ctx, store.CreateUserParams{
		FaceitID: fmt.Sprintf("ingest-test-%d", time.Now().UnixNano()),
		Nickname: "IngestTestPlayer",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	d, err := q.CreateDemo(ctx, store.CreateDemoParams{
		UserID:   user.ID,
		FilePath: "/demos/ingest-test.dem",
		FileSize: 1024000,
		Status:   "ready",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}

	return user, d
}

func TestIngestRounds_InsertAndQuery(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)

	user, d := createTestUserAndDemo(t, q)
	defer func() {
		q.DeleteRoundsByDemoID(ctx, d.ID)
		q.DeleteDemo(ctx, d.ID)
		q.DeleteUser(ctx, user.ID)
	}()

	result := &demo.ParseResult{
		Rounds: []demo.RoundData{
			{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT", WinReason: "ct_win", CTScore: 1, TScore: 0},
			{Number: 2, StartTick: 1001, EndTick: 2000, WinnerSide: "T", WinReason: "t_win", CTScore: 1, TScore: 1},
		},
		Events: []demo.GameEvent{
			{
				Tick: 100, RoundNumber: 1, Type: "kill",
				AttackerSteamID: "76561198001", VictimSteamID: "76561198002",
				ExtraData: map[string]interface{}{
					"headshot": true, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Bob", "victim_team": "T",
				},
			},
			{
				Tick: 50, RoundNumber: 1, Type: "player_hurt",
				AttackerSteamID: "76561198001", VictimSteamID: "76561198002",
				ExtraData: map[string]interface{}{
					"health_damage": 85,
				},
			},
			{
				Tick: 1100, RoundNumber: 2, Type: "kill",
				AttackerSteamID: "76561198002", VictimSteamID: "76561198001",
				ExtraData: map[string]interface{}{
					"headshot": false, "attacker_name": "Bob", "attacker_team": "T",
					"victim_name": "Alice", "victim_team": "CT",
				},
			},
		},
	}

	if err := demo.IngestRounds(ctx, testDB, d.ID, result); err != nil {
		t.Fatalf("IngestRounds: %v", err)
	}

	// Verify rounds.
	rounds, err := q.GetRoundsByDemoID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetRoundsByDemoID: %v", err)
	}
	if len(rounds) != 2 {
		t.Fatalf("expected 2 rounds, got %d", len(rounds))
	}
	if rounds[0].RoundNumber != 1 || rounds[0].WinnerSide != "CT" {
		t.Errorf("round 1: got number=%d side=%s", rounds[0].RoundNumber, rounds[0].WinnerSide)
	}
	if rounds[1].RoundNumber != 2 || rounds[1].WinnerSide != "T" {
		t.Errorf("round 2: got number=%d side=%s", rounds[1].RoundNumber, rounds[1].WinnerSide)
	}

	// Verify player_rounds for round 1.
	pr1, err := q.GetPlayerRoundsByRoundID(ctx, rounds[0].ID)
	if err != nil {
		t.Fatalf("GetPlayerRoundsByRoundID round 1: %v", err)
	}
	if len(pr1) != 2 {
		t.Fatalf("expected 2 player_rounds in round 1, got %d", len(pr1))
	}

	// Find Alice in round 1.
	var alice *store.PlayerRound
	for i := range pr1 {
		if pr1[i].SteamID == "76561198001" {
			alice = &pr1[i]
		}
	}
	if alice == nil {
		t.Fatal("expected player_round for Alice in round 1")
	}
	if alice.Kills != 1 || alice.Deaths != 0 || alice.HeadshotKills != 1 || alice.Damage != 85 {
		t.Errorf("Alice round 1: kills=%d deaths=%d hs=%d dmg=%d", alice.Kills, alice.Deaths, alice.HeadshotKills, alice.Damage)
	}
	if !alice.FirstKill {
		t.Error("Alice should have first_kill in round 1")
	}
}

func TestIngestRounds_Idempotent(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)

	user, d := createTestUserAndDemo(t, q)
	defer func() {
		q.DeleteRoundsByDemoID(ctx, d.ID)
		q.DeleteDemo(ctx, d.ID)
		q.DeleteUser(ctx, user.ID)
	}()

	result := &demo.ParseResult{
		Rounds: []demo.RoundData{
			{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT", WinReason: "ct_win", CTScore: 1, TScore: 0},
		},
		Events: []demo.GameEvent{
			{
				Tick: 100, RoundNumber: 1, Type: "kill",
				AttackerSteamID: "76561198001", VictimSteamID: "76561198002",
				ExtraData: map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Bob", "victim_team": "T",
				},
			},
		},
	}

	// Ingest twice.
	if err := demo.IngestRounds(ctx, testDB, d.ID, result); err != nil {
		t.Fatalf("IngestRounds first call: %v", err)
	}
	if err := demo.IngestRounds(ctx, testDB, d.ID, result); err != nil {
		t.Fatalf("IngestRounds second call: %v", err)
	}

	// Should still have exactly 1 round, not 2.
	rounds, err := q.GetRoundsByDemoID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetRoundsByDemoID: %v", err)
	}
	if len(rounds) != 1 {
		t.Errorf("expected 1 round after double ingest, got %d", len(rounds))
	}

	pr, err := q.GetPlayerRoundsByRoundID(ctx, rounds[0].ID)
	if err != nil {
		t.Fatalf("GetPlayerRoundsByRoundID: %v", err)
	}
	if len(pr) != 2 {
		t.Errorf("expected 2 player_rounds after double ingest, got %d", len(pr))
	}
}

func TestIngestRounds_EmptyRounds(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)

	user, d := createTestUserAndDemo(t, q)
	defer func() {
		q.DeleteDemo(ctx, d.ID)
		q.DeleteUser(ctx, user.ID)
	}()

	result := &demo.ParseResult{
		Rounds: []demo.RoundData{},
		Events: []demo.GameEvent{},
	}

	if err := demo.IngestRounds(ctx, testDB, d.ID, result); err != nil {
		t.Fatalf("IngestRounds with empty rounds: %v", err)
	}

	rounds, err := q.GetRoundsByDemoID(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetRoundsByDemoID: %v", err)
	}
	if len(rounds) != 0 {
		t.Errorf("expected 0 rounds, got %d", len(rounds))
	}
}
