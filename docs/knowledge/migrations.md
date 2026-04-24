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

## Running

Migrations run automatically at startup in `internal/database/sqlite.go` → `RunMigrations(db)`. A corruption check (`PRAGMA integrity_check`) runs first; a pre-migration backup (`oversite.db.bak`) is made.

## Testing migrations

`testutil.NewTestDB(t)` from `internal/testutil/db.go` creates a fresh in-memory SQLite with all migrations applied. Use this for any test that touches the store. Never open a test DB manually.
