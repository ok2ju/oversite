//go:build integration

package testutil_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"

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

// --- Migration tests ---

func TestMigrations_TablesExist(t *testing.T) {
	ctx := context.Background()

	expectedTables := []string{
		"users", "demos", "rounds", "player_rounds",
		"tick_data", "game_events", "strategy_boards",
		"grenade_lineups", "faceit_matches",
	}

	for _, table := range expectedTables {
		var exists bool
		err := testDB.QueryRowContext(ctx,
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)",
			table,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("checking table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("expected table %s to exist", table)
		}
	}
}

func TestMigrations_TickDataIsHypertable(t *testing.T) {
	ctx := context.Background()

	var isHypertable bool
	err := testDB.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT FROM timescaledb_information.hypertables WHERE hypertable_name = 'tick_data')",
	).Scan(&isHypertable)
	if err != nil {
		t.Fatalf("checking hypertable: %v", err)
	}
	if !isHypertable {
		t.Error("expected tick_data to be a hypertable")
	}
}

func TestMigrations_DownAndUp(t *testing.T) {
	// Use a separate container for the destructive down-migration test
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	container, connURL, err := testutil.PostgresContainer(ctx)
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	// Run up
	if err := testutil.RunMigrations(connURL); err != nil {
		t.Fatalf("running migrations up: %v", err)
	}

	// Verify tables exist
	db, err := sql.Open("postgres", connURL)
	if err != nil {
		t.Fatalf("opening db: %v", err)
	}
	defer db.Close()

	tables := []string{"users", "demos", "rounds", "tick_data", "game_events"}
	for _, table := range tables {
		var exists bool
		err := db.QueryRowContext(ctx,
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)",
			table,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("checking table %s: %v", table, err)
		}
		if !exists {
			t.Fatalf("expected table %s to exist after migrate up", table)
		}
	}

	// Run down
	if err := testutil.RunMigrationsDown(connURL); err != nil {
		t.Fatalf("running migrations down: %v", err)
	}

	// Verify tables are gone
	for _, table := range tables {
		var exists bool
		err := db.QueryRowContext(ctx,
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)",
			table,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("checking table %s after down: %v", table, err)
		}
		if exists {
			t.Errorf("expected table %s to NOT exist after migrate down", table)
		}
	}
}

// --- Redis container test ---

func TestRedisContainer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	container, redisURL, err := testutil.RedisContainer(ctx)
	if err != nil {
		t.Fatalf("starting redis container: %v", err)
	}
	defer container.Terminate(ctx)

	if redisURL == "" {
		t.Fatal("expected non-empty redis URL")
	}

	// Verify we can connect (basic TCP check via the container's state)
	state, err := container.State(ctx)
	if err != nil {
		t.Fatalf("getting container state: %v", err)
	}
	if !state.Running {
		t.Fatal("expected redis container to be running")
	}
}

// --- MinIO container test ---

func TestMinIOContainer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	container, endpoint, err := testutil.MinIOContainer(ctx)
	if err != nil {
		t.Fatalf("starting minio container: %v", err)
	}
	defer container.Terminate(ctx)

	if endpoint == "" {
		t.Fatal("expected non-empty minio endpoint")
	}

	state, err := container.State(ctx)
	if err != nil {
		t.Fatalf("getting container state: %v", err)
	}
	if !state.Running {
		t.Fatal("expected minio container to be running")
	}
}

// --- sqlc CRUD integration tests ---

func TestCRUD_Users(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)

	// Create
	user, err := q.CreateUser(ctx, store.CreateUserParams{
		FaceitID:  "crud-test-faceit-id",
		Nickname:  "CRUDTestPlayer",
		AvatarUrl: sql.NullString{String: "https://example.com/avatar.png", Valid: true},
		Country:   sql.NullString{String: "US", Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.Nickname != "CRUDTestPlayer" {
		t.Errorf("expected nickname 'CRUDTestPlayer', got %q", user.Nickname)
	}

	// Read by ID
	fetched, err := q.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if fetched.FaceitID != "crud-test-faceit-id" {
		t.Errorf("expected faceit_id 'crud-test-faceit-id', got %q", fetched.FaceitID)
	}

	// Read by Faceit ID
	byFaceit, err := q.GetUserByFaceitID(ctx, "crud-test-faceit-id")
	if err != nil {
		t.Fatalf("GetUserByFaceitID: %v", err)
	}
	if byFaceit.ID != user.ID {
		t.Errorf("expected same user ID, got %s vs %s", byFaceit.ID, user.ID)
	}

	// Update
	updated, err := q.UpdateUser(ctx, store.UpdateUserParams{
		ID:        user.ID,
		Nickname:  "UpdatedNick",
		AvatarUrl: sql.NullString{String: "https://example.com/new.png", Valid: true},
		Country:   sql.NullString{String: "DE", Valid: true},
	})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if updated.Nickname != "UpdatedNick" {
		t.Errorf("expected updated nickname 'UpdatedNick', got %q", updated.Nickname)
	}

	// Delete
	if err := q.DeleteUser(ctx, user.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	_, err = q.GetUserByID(ctx, user.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows after delete, got %v", err)
	}
}

func TestCRUD_Demos(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)

	// Create a user first (FK dependency)
	user, err := q.CreateUser(ctx, store.CreateUserParams{
		FaceitID: "demo-test-user",
		Nickname: "DemoTestPlayer",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	defer q.DeleteUser(ctx, user.ID)

	// Create demo
	demo, err := q.CreateDemo(ctx, store.CreateDemoParams{
		UserID:   user.ID,
		MapName:  "de_dust2",
		FilePath: "/demos/test.dem",
		FileSize: 1024000,
		Status:   "uploading",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}
	if demo.MapName != "de_dust2" {
		t.Errorf("expected map 'de_dust2', got %q", demo.MapName)
	}

	// Read
	fetched, err := q.GetDemoByID(ctx, demo.ID)
	if err != nil {
		t.Fatalf("GetDemoByID: %v", err)
	}
	if fetched.Status != "uploading" {
		t.Errorf("expected status 'uploading', got %q", fetched.Status)
	}

	// Update status
	updated, err := q.UpdateDemoStatus(ctx, store.UpdateDemoStatusParams{
		ID:     demo.ID,
		Status: "parsing",
	})
	if err != nil {
		t.Fatalf("UpdateDemoStatus: %v", err)
	}
	if updated.Status != "parsing" {
		t.Errorf("expected status 'parsing', got %q", updated.Status)
	}

	// List by user
	demos, err := q.ListDemosByUserID(ctx, store.ListDemosByUserIDParams{
		UserID: user.ID,
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListDemosByUserID: %v", err)
	}
	if len(demos) != 1 {
		t.Errorf("expected 1 demo, got %d", len(demos))
	}

	// Delete
	if err := q.DeleteDemo(ctx, demo.ID); err != nil {
		t.Fatalf("DeleteDemo: %v", err)
	}
}

func TestCRUD_RoundsAndGameEvents(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)

	user, err := q.CreateUser(ctx, store.CreateUserParams{
		FaceitID: "round-test-user",
		Nickname: "RoundTestPlayer",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	defer q.DeleteUser(ctx, user.ID)

	demo, err := q.CreateDemo(ctx, store.CreateDemoParams{
		UserID:   user.ID,
		MapName:  "de_inferno",
		FilePath: "/demos/round-test.dem",
		FileSize: 2048000,
		Status:   "ready",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}
	defer q.DeleteDemo(ctx, demo.ID)

	// Create round
	round, err := q.CreateRound(ctx, store.CreateRoundParams{
		DemoID:      demo.ID,
		RoundNumber: 1,
		StartTick:   0,
		EndTick:     3200,
		WinnerSide:  "CT",
		WinReason:   "TargetBombed",
		CtScore:     1,
		TScore:      0,
	})
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	// Create game event
	evt, err := q.CreateGameEvent(ctx, store.CreateGameEventParams{
		DemoID:    demo.ID,
		Tick:      1500,
		EventType: "kill",
		AttackerSteamID: sql.NullString{String: "STEAM_0:1:12345", Valid: true},
		VictimSteamID:   sql.NullString{String: "STEAM_0:1:67890", Valid: true},
		Weapon:          sql.NullString{String: "ak47", Valid: true},
	})
	if err != nil {
		t.Fatalf("CreateGameEvent: %v", err)
	}

	// Query rounds by demo
	rounds, err := q.GetRoundsByDemoID(ctx, demo.ID)
	if err != nil {
		t.Fatalf("GetRoundsByDemoID: %v", err)
	}
	if len(rounds) != 1 {
		t.Errorf("expected 1 round, got %d", len(rounds))
	}
	if rounds[0].WinnerSide != "CT" {
		t.Errorf("expected winner_side 'CT', got %q", rounds[0].WinnerSide)
	}

	// Query events by demo
	events, err := q.GetGameEventsByDemoID(ctx, demo.ID)
	if err != nil {
		t.Fatalf("GetGameEventsByDemoID: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	// Cleanup
	_ = q.DeleteGameEventsByDemoID(ctx, demo.ID)
	_ = q.DeleteRoundsByDemoID(ctx, demo.ID)
	_ = evt
	_ = round
}

func TestBatchInsert_TickData(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)

	user, err := q.CreateUser(ctx, store.CreateUserParams{
		FaceitID: "tick-batch-user",
		Nickname: "TickBatchPlayer",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	defer q.DeleteUser(ctx, user.ID)

	demo, err := q.CreateDemo(ctx, store.CreateDemoParams{
		UserID:   user.ID,
		MapName:  "de_mirage",
		FilePath: "/demos/tick-test.dem",
		FileSize: 3000000,
		Status:   "ready",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}
	defer func() {
		q.DeleteTickDataByDemoID(ctx, demo.ID)
		q.DeleteDemo(ctx, demo.ID)
	}()

	// Generate 10k+ rows
	const numRows = 12800 // 10 players * 1280 ticks
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := make([]store.InsertTickDataParams, numRows)

	players := []string{
		"STEAM_0:1:00001", "STEAM_0:1:00002", "STEAM_0:1:00003",
		"STEAM_0:1:00004", "STEAM_0:1:00005", "STEAM_0:1:00006",
		"STEAM_0:1:00007", "STEAM_0:1:00008", "STEAM_0:1:00009",
		"STEAM_0:1:00010",
	}

	for i := range rows {
		tick := int32(i / 10)
		playerIdx := i % 10
		rows[i] = store.InsertTickDataParams{
			Time:    baseTime.Add(time.Duration(tick) * time.Millisecond * 16), // 64 tick
			DemoID:  demo.ID,
			Tick:    tick,
			SteamID: players[playerIdx],
			X:       float32(100 + i%500),
			Y:       float32(200 + i%300),
			Z:       float32(0),
			Yaw:     float32(i % 360),
			Health:  100,
			Armor:   100,
			IsAlive: true,
			Weapon:  sql.NullString{String: "ak47", Valid: true},
		}
	}

	// Batch insert
	inserted, err := store.CopyTickData(ctx, testDB, rows)
	if err != nil {
		t.Fatalf("CopyTickData: %v", err)
	}
	if inserted != int64(numRows) {
		t.Errorf("expected %d inserted, got %d", numRows, inserted)
	}

	// Verify with range query
	result, err := q.GetTickDataByRange(ctx, store.GetTickDataByRangeParams{
		DemoID: demo.ID,
		Tick:   0,
		Tick_2: 10,
	})
	if err != nil {
		t.Fatalf("GetTickDataByRange: %v", err)
	}
	// Ticks 0-10 = 11 ticks * 10 players = 110 rows
	if len(result) != 110 {
		t.Errorf("expected 110 rows for tick range [0,10], got %d", len(result))
	}
}
