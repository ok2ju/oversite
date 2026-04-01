//go:build integration

package testutil_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/ok2ju/oversite/backend/internal/testutil"
)

func TestPostgresContainerAndMigrations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start container
	container, connURL, err := testutil.PostgresContainer(ctx)
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	// Run migrations
	if err := testutil.RunMigrations(connURL); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	// Verify tables exist
	db, err := sql.Open("postgres", connURL)
	if err != nil {
		t.Fatalf("opening db connection: %v", err)
	}
	defer db.Close()

	expectedTables := []string{
		"users", "demos", "rounds", "player_rounds",
		"tick_data", "game_events", "strategy_boards",
		"grenade_lineups", "faceit_matches",
	}

	for _, table := range expectedTables {
		var exists bool
		err := db.QueryRowContext(ctx,
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

	// Verify tick_data is a hypertable
	var isHypertable bool
	err = db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT FROM timescaledb_information.hypertables WHERE hypertable_name = 'tick_data')",
	).Scan(&isHypertable)
	if err != nil {
		t.Fatalf("checking hypertable: %v", err)
	}
	if !isHypertable {
		t.Error("expected tick_data to be a hypertable")
	}

	// Verify we can insert and query a user
	var userID string
	err = db.QueryRowContext(ctx,
		`INSERT INTO users (faceit_id, nickname) VALUES ($1, $2) RETURNING id`,
		"test-faceit-id", "TestPlayer",
	).Scan(&userID)
	if err != nil {
		t.Fatalf("inserting test user: %v", err)
	}
	if userID == "" {
		t.Error("expected non-empty user ID")
	}
}
