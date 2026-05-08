# SQLite WAL Mode

**Related:** [[sqlc-workflow]] · [[migrations]] · [ADR-0008](../decisions/0008-sqlite-local-database.md) · [ADR-0016](../decisions/0016-sqlite-multi-connection-pool.md) · [architecture/database](../architecture/database.md)

## Why WAL + small writer pool

`modernc.org/sqlite` is pure Go (no CGo). WAL mode lets readers see the last committed snapshot during a long writer (e.g. demo ingest), so the UI can browse the library while parsing.

We run with `SetMaxOpenConns(4)` + `busy_timeout=5000`: SQLite serializes writes (only one writer holds the WAL lock at a time), and a 5 s busy timeout absorbs contention without surfacing `SQLITE_BUSY` to callers. Reads run on the other connections in parallel.

> **History:** through migration 008 we ran `SetMaxOpenConns(1)`. That blocked every read for the duration of a 300K-row ingest tx. The 1→4 bump in 2026-05-08 was paired with `busy_timeout` so write correctness still holds.

## Pragmas applied per connection

`internal/database/sqlite.go` sets these on every connection (not just at startup):

| Pragma | Value | Why |
|--------|-------|-----|
| `journal_mode` | `WAL` | Reader/writer concurrency |
| `foreign_keys` | `ON` | Required for FK checks |
| `synchronous` | `NORMAL` | Avoid per-WAL-frame fsyncs |
| `busy_timeout` | `5000` | Serialize writes safely with `MaxOpenConns>1` |
| `cache_size` | `-64000` (~64 MB) | Hot pages stay resident |
| `mmap_size` | `268435456` (256 MB) | Read path skips the page cache for large demos |
| `temp_store` | `MEMORY` | Sort/group spills don't hit disk |
| `journal_size_limit` | bounded | WAL doesn't grow unbounded across long sessions |

## Startup sequence

1. `internal/database/sqlite.go` opens the DB.
2. Apply per-connection pragmas (see table above).
3. `SetMaxOpenConns(4)`.
4. Run embedded migrations (`//go:embed migrations/*`) via golang-migrate.
5. Return the `*sql.DB` to be shared by all services.

## Shutdown

Wails `App.Shutdown` should call `PRAGMA wal_checkpoint(TRUNCATE)` before closing the DB, so the `.db-wal` sidecar gets folded into the main file. Users who copy the DB file for backup won't get a half-synced snapshot.

## Don't

- Don't drop the `synchronous=NORMAL` pragma. Default `FULL` fsyncs every WAL frame; ingest goes from seconds to minutes.
- Don't open a second `*sql.DB` to the same file. Use the shared `*sql.DB` everywhere.
- Don't use `BEGIN IMMEDIATE` — `busy_timeout` handles writer contention.
- Don't delete `.db-wal` / `.db-shm` manually. SQLite manages them.

## Tick data scale

A typical 30-round demo produces ~1.28M rows in `tick_data`. We batch with multi-row VALUES (~500 rows/INSERT) inside one transaction. Composite primary key `(demo_id, tick, steam_id)` is the clustered index — no additional indexes on `tick_data`.

Inventory does **not** live on `tick_data`. Migration 011 moved it to `round_loadouts(round_id, steam_id, inventory)` — captured once per round at freeze-end. Mid-round pickups/drops are intentionally not tracked. ~1.28M × ~30 B ≈ 40 MB/demo saved.
