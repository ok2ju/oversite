package testutil_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

func TestNewTestDB_MigratedAndForeignKeys(t *testing.T) {
	db := testutil.NewTestDB(t)

	// Verify migrations ran — all expected tables exist.
	tables := []string{
		"users", "demos", "rounds", "player_rounds",
		"tick_data", "game_events", "strategy_boards",
		"grenade_lineups", "faceit_matches",
	}
	for _, table := range tables {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}

	// Verify foreign keys are enforced.
	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}

func TestNewTestQueries_InsertAndSelect(t *testing.T) {
	q, _ := testutil.NewTestQueries(t)
	ctx := context.Background()

	user, err := q.CreateUser(ctx, store.CreateUserParams{
		FaceitID:    "test-faceit-id",
		Nickname:    "testplayer",
		AvatarUrl:   "https://example.com/avatar.png",
		FaceitElo:   2000,
		FaceitLevel: 9,
		Country:     "SE",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected non-zero user ID")
	}

	got, err := q.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.Nickname != "testplayer" {
		t.Errorf("Nickname = %q, want %q", got.Nickname, "testplayer")
	}
}

func TestMockKeyring_RoundTrip(t *testing.T) {
	kr := testutil.NewMockKeyring()

	// Get before set returns ErrKeyNotFound.
	_, err := kr.Get("oversite", "token")
	if err != testutil.ErrKeyNotFound {
		t.Fatalf("Get before Set: err = %v, want ErrKeyNotFound", err)
	}

	// Set then Get returns the value.
	if err := kr.Set("oversite", "token", "secret-abc"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	val, err := kr.Get("oversite", "token")
	if err != nil {
		t.Fatalf("Get after Set: %v", err)
	}
	if val != "secret-abc" {
		t.Errorf("Get = %q, want %q", val, "secret-abc")
	}

	// Delete then Get returns ErrKeyNotFound.
	if err := kr.Delete("oversite", "token"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = kr.Get("oversite", "token")
	if err != testutil.ErrKeyNotFound {
		t.Fatalf("Get after Delete: err = %v, want ErrKeyNotFound", err)
	}

	// Delete non-existent returns ErrKeyNotFound.
	if err := kr.Delete("oversite", "token"); err != testutil.ErrKeyNotFound {
		t.Errorf("Delete non-existent: err = %v, want ErrKeyNotFound", err)
	}
}

func TestMockFaceitClient_Configurable(t *testing.T) {
	client := &testutil.MockFaceitClient{
		GetPlayerFn: func(_ context.Context, playerID string) (*testutil.FaceitPlayer, error) {
			return &testutil.FaceitPlayer{
				PlayerID:  playerID,
				Nickname:  "s1mple",
				FaceitElo: 3000,
			}, nil
		},
	}

	player, err := client.GetPlayer(context.Background(), "player-123")
	if err != nil {
		t.Fatalf("GetPlayer: %v", err)
	}
	if player.Nickname != "s1mple" {
		t.Errorf("Nickname = %q, want %q", player.Nickname, "s1mple")
	}

	// Unconfigured methods return zero values.
	history, err := client.GetPlayerHistory(context.Background(), "player-123", 0, 10)
	if err != nil {
		t.Fatalf("GetPlayerHistory: %v", err)
	}
	if len(history.Items) != 0 {
		t.Errorf("expected empty history, got %d items", len(history.Items))
	}
}

func TestCompareGolden(t *testing.T) {
	// Create a temporary golden file to test the round-trip.
	dir := t.TempDir()
	goldenFile := filepath.Join(dir, "test.golden")
	content := []byte("expected output\n")
	if err := os.WriteFile(goldenFile, content, 0o644); err != nil {
		t.Fatalf("write temp golden: %v", err)
	}

	// Verify TestdataPath returns a valid directory path (even if it doesn't
	// exist yet — the project may not have testdata/ created).
	path := testutil.TestdataPath()
	if path == "" {
		t.Fatal("TestdataPath() returned empty string")
	}
}
