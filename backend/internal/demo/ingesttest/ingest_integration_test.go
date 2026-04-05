//go:build integration

package ingesttest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"

	"github.com/ok2ju/oversite/backend/internal/demo"
	"github.com/ok2ju/oversite/backend/internal/store"
	"github.com/ok2ju/oversite/backend/internal/testutil"
)

var (
	pgContainer testcontainers.Container
	testDB      *sql.DB
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var err error
	var connURL string
	pgContainer, connURL, err = testutil.PostgresContainer(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "starting postgres container: %v\n", err)
		os.Exit(1)
	}

	if err := testutil.RunMigrations(connURL); err != nil {
		fmt.Fprintf(os.Stderr, "running migrations: %v\n", err)
		pgContainer.Terminate(ctx)
		os.Exit(1)
	}

	testDB, err = sql.Open("postgres", connURL)
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

// createTestDemo creates a user and demo for testing, returning the demo ID
// and a cleanup function.
func createTestDemo(t *testing.T) (uuid.UUID, func()) {
	t.Helper()
	ctx := context.Background()
	q := store.New(testDB)

	user, err := q.CreateUser(ctx, store.CreateUserParams{
		FaceitID: fmt.Sprintf("ingest-test-%s", t.Name()),
		Nickname: t.Name(),
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	demoRow, err := q.CreateDemo(ctx, store.CreateDemoParams{
		UserID:   user.ID,
		FilePath: "/demos/ingest-test.dem",
		FileSize: 1000000,
		Status:   "parsing",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}

	return demoRow.ID, func() {
		q.DeleteTickDataByDemoID(ctx, demoRow.ID)
		q.DeleteDemo(ctx, demoRow.ID)
		q.DeleteUser(ctx, user.ID)
	}
}

func makeTicks(n int, players []string) []demo.TickSnapshot {
	ticks := make([]demo.TickSnapshot, 0, n)
	for i := 0; i < n; i++ {
		ticks = append(ticks, demo.TickSnapshot{
			Tick:    (i / len(players)) * 4,
			SteamID: players[i%len(players)],
			X:       float64(100 + i%500),
			Y:       float64(200 + i%300),
			Z:       0,
			Yaw:     float64(i % 360),
			Health:  100,
			Armor:   100,
			IsAlive: true,
			Weapon:  "ak47",
		})
	}
	return ticks
}

var testPlayers = []string{
	"76561198000001", "76561198000002", "76561198000003",
	"76561198000004", "76561198000005", "76561198000006",
	"76561198000007", "76561198000008", "76561198000009",
	"76561198000010",
}

func countTickRows(t *testing.T, demoID uuid.UUID) int {
	t.Helper()
	var count int
	err := testDB.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM tick_data WHERE demo_id = $1", demoID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("counting tick rows: %v", err)
	}
	return count
}

func TestTickIngester_BatchInsert(t *testing.T) {
	demoID, cleanup := createTestDemo(t)
	defer cleanup()

	ctx := context.Background()
	ingester := demo.NewTickIngester(testDB, 10_000)
	baseTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	// 100k rows: 10 players * 10000 sampled ticks
	const numRows = 100_000
	ticks := makeTicks(numRows, testPlayers)

	inserted, err := ingester.Ingest(ctx, demoID, ticks, baseTime, 64)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if inserted != int64(numRows) {
		t.Errorf("inserted = %d, want %d", inserted, numRows)
	}

	// Verify count via direct query
	got := countTickRows(t, demoID)
	if got != numRows {
		t.Errorf("row count = %d, want %d", got, numRows)
	}

	// Spot-check: query back a small range and verify data
	q := store.New(testDB)
	rows, err := q.GetTickDataByRange(ctx, store.GetTickDataByRangeParams{
		DemoID: demoID,
		Tick:   0,
		Tick_2: 0,
	})
	if err != nil {
		t.Fatalf("GetTickDataByRange: %v", err)
	}
	// Tick 0 should have 10 players
	if len(rows) != 10 {
		t.Errorf("tick 0 rows = %d, want 10", len(rows))
	}
}

func TestTickIngester_LargeInsert_Performance(t *testing.T) {
	demoID, cleanup := createTestDemo(t)
	defer cleanup()

	ctx := context.Background()
	ingester := demo.NewTickIngester(testDB, 50_000)
	baseTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	const numRows = 500_000
	ticks := makeTicks(numRows, testPlayers)

	start := time.Now()
	inserted, err := ingester.Ingest(ctx, demoID, ticks, baseTime, 64)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if inserted != int64(numRows) {
		t.Errorf("inserted = %d, want %d", inserted, numRows)
	}
	if elapsed > 30*time.Second {
		t.Errorf("insert took %v, want < 30s", elapsed)
	}
	t.Logf("inserted %d rows in %v", numRows, elapsed)
}

func TestTickIngester_Idempotency(t *testing.T) {
	demoID, cleanup := createTestDemo(t)
	defer cleanup()

	ctx := context.Background()
	ingester := demo.NewTickIngester(testDB, 5_000)
	baseTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	// First ingest: 1000 rows
	ticks1 := makeTicks(1000, testPlayers)
	_, err := ingester.Ingest(ctx, demoID, ticks1, baseTime, 64)
	if err != nil {
		t.Fatalf("first Ingest: %v", err)
	}
	if got := countTickRows(t, demoID); got != 1000 {
		t.Fatalf("after first ingest: count = %d, want 1000", got)
	}

	// Re-ingest: 2000 rows — should replace the 1000
	ticks2 := makeTicks(2000, testPlayers)
	inserted, err := ingester.Ingest(ctx, demoID, ticks2, baseTime, 64)
	if err != nil {
		t.Fatalf("second Ingest: %v", err)
	}
	if inserted != 2000 {
		t.Errorf("inserted = %d, want 2000", inserted)
	}
	if got := countTickRows(t, demoID); got != 2000 {
		t.Errorf("after re-ingest: count = %d, want 2000", got)
	}
}

func TestTickIngester_IdempotencyIsolation(t *testing.T) {
	demoA, cleanupA := createTestDemo(t)
	defer cleanupA()

	// Need a second demo with a distinct user
	ctx := context.Background()
	q := store.New(testDB)
	userB, err := q.CreateUser(ctx, store.CreateUserParams{
		FaceitID: "ingest-isolation-b",
		Nickname: "IsolationB",
	})
	if err != nil {
		t.Fatalf("CreateUser B: %v", err)
	}
	demoB, err := q.CreateDemo(ctx, store.CreateDemoParams{
		UserID:   userB.ID,
		FilePath: "/demos/isolation-b.dem",
		FileSize: 1000000,
		Status:   "parsing",
	})
	if err != nil {
		t.Fatalf("CreateDemo B: %v", err)
	}
	defer func() {
		q.DeleteTickDataByDemoID(ctx, demoB.ID)
		q.DeleteDemo(ctx, demoB.ID)
		q.DeleteUser(ctx, userB.ID)
	}()

	ingester := demo.NewTickIngester(testDB, 5_000)
	baseTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	// Insert 500 rows into demo A
	ticksA := makeTicks(500, testPlayers)
	if _, err := ingester.Ingest(ctx, demoA, ticksA, baseTime, 64); err != nil {
		t.Fatalf("Ingest A: %v", err)
	}

	// Insert 300 rows into demo B
	ticksB := makeTicks(300, testPlayers)
	if _, err := ingester.Ingest(ctx, demoB.ID, ticksB, baseTime, 64); err != nil {
		t.Fatalf("Ingest B: %v", err)
	}

	// Re-ingest demo A with 200 rows — should NOT affect demo B
	ticksA2 := makeTicks(200, testPlayers)
	if _, err := ingester.Ingest(ctx, demoA, ticksA2, baseTime, 64); err != nil {
		t.Fatalf("Re-Ingest A: %v", err)
	}

	if got := countTickRows(t, demoA); got != 200 {
		t.Errorf("demo A count = %d, want 200", got)
	}
	if got := countTickRows(t, demoB.ID); got != 300 {
		t.Errorf("demo B count = %d, want 300 (should be unaffected)", got)
	}
}

func TestTickIngester_EmptyTicks(t *testing.T) {
	demoID, cleanup := createTestDemo(t)
	defer cleanup()

	ctx := context.Background()
	ingester := demo.NewTickIngester(testDB, 10_000)
	baseTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	// First: ingest with no prior data
	inserted, err := ingester.Ingest(ctx, demoID, nil, baseTime, 64)
	if err != nil {
		t.Fatalf("Ingest empty (no prior data): %v", err)
	}
	if inserted != 0 {
		t.Errorf("inserted = %d, want 0", inserted)
	}
	if got := countTickRows(t, demoID); got != 0 {
		t.Errorf("row count = %d, want 0", got)
	}

	// Second: ingest real data, then re-ingest with empty ticks — old data must be deleted
	ticks := makeTicks(100, testPlayers)
	if _, err := ingester.Ingest(ctx, demoID, ticks, baseTime, 64); err != nil {
		t.Fatalf("Ingest 100 rows: %v", err)
	}
	if got := countTickRows(t, demoID); got != 100 {
		t.Fatalf("after ingest: count = %d, want 100", got)
	}

	inserted, err = ingester.Ingest(ctx, demoID, nil, baseTime, 64)
	if err != nil {
		t.Fatalf("Ingest empty (with prior data): %v", err)
	}
	if inserted != 0 {
		t.Errorf("inserted = %d, want 0", inserted)
	}
	if got := countTickRows(t, demoID); got != 0 {
		t.Errorf("after empty re-ingest: count = %d, want 0 (old data should be deleted)", got)
	}
}

func TestTickIngester_SyntheticTimeValues(t *testing.T) {
	demoID, cleanup := createTestDemo(t)
	defer cleanup()

	ctx := context.Background()
	ingester := demo.NewTickIngester(testDB, 10_000)
	baseTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	ticks := []demo.TickSnapshot{
		{Tick: 0, SteamID: "player1", X: 1, Y: 2, Z: 3, Health: 100, IsAlive: true},
		{Tick: 64, SteamID: "player1", X: 4, Y: 5, Z: 6, Health: 100, IsAlive: true},
		{Tick: 128, SteamID: "player1", X: 7, Y: 8, Z: 9, Health: 100, IsAlive: true},
	}

	_, err := ingester.Ingest(ctx, demoID, ticks, baseTime, 64)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	q := store.New(testDB)
	rows, err := q.GetTickDataByRange(ctx, store.GetTickDataByRangeParams{
		DemoID: demoID,
		Tick:   0,
		Tick_2: 128,
	})
	if err != nil {
		t.Fatalf("GetTickDataByRange: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	wantTimes := []time.Time{
		baseTime,
		baseTime.Add(1 * time.Second),
		baseTime.Add(2 * time.Second),
	}
	for i, r := range rows {
		if !r.Time.Equal(wantTimes[i]) {
			t.Errorf("row[%d].Time = %v, want %v", i, r.Time, wantTimes[i])
		}
	}
}
