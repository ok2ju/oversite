# SQLite WAL Mode

**Related:** [[sqlc-workflow]] · [[migrations]] · [ADR-0008](../decisions/0008-sqlite-local-database.md) · [architecture/database](../architecture/database.md)

## Why WAL + single connection

`modernc.org/sqlite` is pure Go (no CGo). We enable WAL mode (`PRAGMA journal_mode=WAL`) so readers don't block writers, which matters during demo parsing (long writer) while the UI is browsing the library (short readers).

But WAL doesn't save us from `SQLITE_BUSY` when multiple Go goroutines write. We use `sql.DB.SetMaxOpenConns(1)` — one writer at a time — which is the **simplest correct** policy for a single-user desktop app. Reads from the same connection serialize too, but SQLite reads are fast enough that it doesn't matter.

## Startup sequence

1. `internal/database/sqlite.go` opens the DB.
2. Set pragmas: `journal_mode=WAL`, `foreign_keys=ON`, `synchronous=NORMAL`.
3. `SetMaxOpenConns(1)`, `SetMaxIdleConns(1)`.
4. Run embedded migrations (`//go:embed migrations/*`) via golang-migrate.
5. Return the `*sql.DB` to be shared by all services.

## Shutdown

Wails `App.Shutdown` should call `PRAGMA wal_checkpoint(TRUNCATE)` before closing the DB, so the `.db-wal` sidecar gets folded into the main file. Users who copy the DB file for backup won't get a half-synced snapshot.

## Don't

- Don't open a second `*sql.DB` to the same file — it defeats `SetMaxOpenConns(1)` and reopens BUSY races.
- Don't use `BEGIN IMMEDIATE` — unnecessary with a single connection.
- Don't delete `.db-wal` / `.db-shm` manually. SQLite manages them.

## Tick data scale

A typical 30-round demo produces ~1.28M rows in `tick_data`. We batch-insert at 10K rows per transaction to keep parse time under 10s. The composite primary key `(demo_id, tick, steam_id)` is the clustered index — no additional indexes on tick_data.
