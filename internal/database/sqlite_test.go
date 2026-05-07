package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ok2ju/oversite/migrations"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func openTestDBWithMigrations(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := OpenWithMigrations(dbPath, migrations.FS)
	if err != nil {
		t.Fatalf("OpenWithMigrations: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestOpen_WALMode(t *testing.T) {
	db := openTestDB(t)

	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want %q", mode, "wal")
	}
}

func TestOpen_ForeignKeysEnabled(t *testing.T) {
	db := openTestDB(t)

	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}

func TestMigrations_AllTablesCreated(t *testing.T) {
	db := openTestDBWithMigrations(t)

	want := []string{
		"demos",
		"rounds",
		"player_rounds",
		"tick_data",
		"game_events",
		"strategy_boards",
		"grenade_lineups",
	}

	for _, table := range want {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

// TestMigration006_RoundsTeamNames verifies the migration adds and round-trips
// ct_team_name / t_team_name on the rounds table.
func TestMigration006_RoundsTeamNames(t *testing.T) {
	db := openTestDBWithMigrations(t)

	for _, col := range []string{"ct_team_name", "t_team_name"} {
		var found int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM pragma_table_info('rounds') WHERE name = ?",
			col,
		).Scan(&found)
		if err != nil {
			t.Fatalf("query column %q: %v", col, err)
		}
		if found != 1 {
			t.Errorf("rounds.%s missing after up migration", col)
		}
	}

	if err := DownMigrations(db, migrations.FS); err != nil {
		t.Fatalf("DownMigrations: %v", err)
	}
}

func TestMigrations_DownRemovesTables(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := OpenWithMigrations(dbPath, migrations.FS)
	if err != nil {
		t.Fatalf("OpenWithMigrations: %v", err)
	}

	if err := DownMigrations(db, migrations.FS); err != nil {
		t.Fatalf("DownMigrations: %v", err)
	}

	var count int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations'",
	).Scan(&count)
	if err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if count != 0 {
		t.Errorf("tables remaining after down = %d, want 0", count)
	}
	_ = db.Close()
}

func TestMigrations_IdempotentUp(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := OpenWithMigrations(dbPath, migrations.FS)
	if err != nil {
		t.Fatalf("first OpenWithMigrations: %v", err)
	}
	_ = db.Close()

	db, err = OpenWithMigrations(dbPath, migrations.FS)
	if err != nil {
		t.Fatalf("second OpenWithMigrations: %v", err)
	}
	_ = db.Close()
}

func TestAppDataDir(t *testing.T) {
	// Override HOME/APPDATA to use a temp dir so we don't pollute the real filesystem.
	tmp := t.TempDir()

	switch runtime.GOOS {
	case "darwin":
		t.Setenv("HOME", tmp)
	case "linux":
		t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, ".local", "share"))
	case "windows":
		t.Setenv("APPDATA", tmp)
	}

	dir, err := AppDataDir()
	if err != nil {
		t.Fatalf("AppDataDir: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat %q: %v", dir, err)
	}
	if !info.IsDir() {
		t.Errorf("%q is not a directory", dir)
	}

	// Verify the path ends with "oversite".
	if filepath.Base(dir) != "oversite" {
		t.Errorf("dir base = %q, want %q", filepath.Base(dir), "oversite")
	}
}
