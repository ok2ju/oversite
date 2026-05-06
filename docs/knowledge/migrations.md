# Migrations

**Related:** [[sqlite-wal]] · [[sqlc-workflow]] · slash command: `/create-migration <name>`

## Tooling

`golang-migrate` with the SQLite driver. Migrations are embedded into the binary via `//go:embed migrations/*` in `internal/database/`. There is **no** runtime dependency on the `migrate` CLI — everything ships inside the single Wails binary.

## Numbering

Numeric prefix, zero-padded: `0001_create_users.up.sql`, `0001_create_users.down.sql`, `0002_...`, etc. Each up file has a matching down. The `/create-migration <name>` slash command generates the pair with the next number.

## Up/down discipline

- Up files are **additive**. Down files undo the up by dropping tables/columns or reverting indexes.
- Don't edit a migration after it has been merged. Fixes go into a new numbered pair.
- For destructive schema changes (dropping columns with data), write a data-preserving down if recovery matters.

## SQLite gotchas

- **Dropping a column requires a table rewrite** on SQLite < 3.35. Even on newer versions, foreign-key references to the dropped column block `ALTER TABLE … DROP COLUMN`. Use the create-new → copy → swap pattern:
  1. `CREATE TABLE foo_new (...)` without the dropped columns.
  2. `INSERT INTO foo_new (...) SELECT ... FROM foo`.
  3. `DROP TABLE foo; ALTER TABLE foo_new RENAME TO foo`.
  4. Recreate any indexes that lived on the original table.
  See `migrations/005_remove_faceit_and_users.up.sql` for a worked example.
- Drop dependent tables and indexes **before** the table they reference, since SQLite enforces FKs at migration time when `PRAGMA foreign_keys=ON`.

## Running

Migrations run automatically at startup in `internal/database/sqlite.go` → `RunMigrations(db)`. A corruption check (`PRAGMA integrity_check`) runs first; a pre-migration backup (`oversite.db.bak`) is made.

## Testing migrations

`testutil.NewTestDB(t)` from `internal/testutil/db.go` creates a fresh in-memory SQLite with all migrations applied. Use this for any test that touches the store. Never open a test DB manually.
