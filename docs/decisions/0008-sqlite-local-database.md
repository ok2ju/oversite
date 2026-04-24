# ADR-0008: Use SQLite as the Local Database

**Date:** 2026-04-12
**Status:** Accepted

Supersedes: [ADR-0004](0004-timescaledb-tick-data.md) (TimescaleDB hypertable for tick data)

## Context

The desktop pivot ([ADR-0006](0006-desktop-app-pivot.md)) eliminates the need for a database server. All data is local to one user on one machine. The database must:

- Store tick-level player position data (~1.28M rows per demo) with acceptable query performance
- Support the same query patterns as the web version: range scans by `(demo_id, tick)` for viewer playback, aggregation for heatmaps
- Be embeddable -- no separate database process
- Cross-compile cleanly for macOS, Windows, and Linux (ideally without CGo)

### Alternatives considered

| Approach | Why rejected |
|----------|-------------|
| **Embedded PostgreSQL** | No mature embeddable PostgreSQL for Go. Would require bundling a full PostgreSQL server, defeating the single-binary goal. |
| **DuckDB** | Excellent for analytics, but Go bindings require CGo. Optimized for OLAP, not the row-oriented range scans needed for tick-by-tick playback. |
| **BoltDB / bbolt** | Key-value store; no SQL. Would require implementing query logic manually. No sqlc support. |
| **SQLite via mattn/go-sqlite3** | Mature and fast, but requires CGo. Cross-compilation with CGo is painful -- needs C toolchains for each target platform. |

## Decision

Use **SQLite** via `modernc.org/sqlite` (pure Go, CGo-free SQLite implementation). Key design decisions:

- **Database file location**: `{OS app data dir}/oversite/oversite.db` (e.g., `~/Library/Application Support/oversite/` on macOS, `%APPDATA%\oversite\` on Windows)
- **Tick data storage**: Regular SQLite table with a composite index on `(demo_id, tick)`. No hypertable -- SQLite's B-tree index handles the range scan pattern efficiently at single-user scale.
- **WAL mode**: Enable Write-Ahead Logging for concurrent read/write (frontend reads tick data while parser writes)
- **SQL generation**: Continue using sqlc, configured for SQLite dialect
- **Type adaptations**: `INTEGER PRIMARY KEY` (not UUID), `TEXT` (not TIMESTAMPTZ/JSONB), `REAL` (not REAL -- same), no `BYTEA` (use `BLOB`), no array types (use JSON text or normalized tables)
- **Transaction batching**: Batch tick data inserts in transactions of ~10,000 rows for write performance

## Consequences

### Positive

- Zero infrastructure -- database is a single file, no server process
- Pure Go cross-compilation -- `modernc.org/sqlite` requires no CGo, no C toolchain
- Single-file backup -- users can copy/move their database trivially
- sqlc reuse -- same code generation workflow, just different SQL dialect
- WAL mode provides good concurrent read/write performance for single-user workloads
- Familiar SQL -- all existing query patterns translate directly

### Negative

- No built-in compression -- unlike TimescaleDB chunk compression, SQLite stores data uncompressed (mitigated: ~1.28M rows x ~100 bytes/row = ~128 MB per demo, acceptable for local disk)
- `modernc.org/sqlite` is ~20-30% slower than CGo `mattn/go-sqlite3` for write-heavy workloads (mitigated by transaction batching)
- No server-side aggregation scaling -- heatmap queries across many demos run on the user's CPU (acceptable for single-user desktop app)
- SQLite's type system is more permissive than PostgreSQL -- less strict type enforcement at the database level
- Maximum practical database size depends on user's disk space; no server-side retention policies
