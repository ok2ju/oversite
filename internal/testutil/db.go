package testutil

import (
	"database/sql"
	"testing"

	"github.com/ok2ju/oversite/internal/database"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/migrations"
)

// NewTestDB opens an in-memory SQLite database with foreign keys enabled and
// all migrations applied. The database is closed automatically when the test
// completes.
//
// WAL mode is not used for in-memory databases (SQLite limitation), but
// foreign_keys=ON is enforced so FK constraints behave as in production.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("testutil.NewTestDB: sql.Open: %v", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		t.Fatalf("testutil.NewTestDB: PRAGMA foreign_keys=ON: %v", err)
	}

	if err := database.RunMigrations(db, migrations.FS); err != nil {
		db.Close()
		t.Fatalf("testutil.NewTestDB: RunMigrations: %v", err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

// NewTestQueries returns a sqlc Queries instance backed by an in-memory SQLite
// database with all migrations applied. Both the Queries and the underlying DB
// are returned so callers can use raw SQL when needed.
func NewTestQueries(t *testing.T) (*store.Queries, *sql.DB) {
	t.Helper()
	db := NewTestDB(t)
	return store.New(db), db
}
