# ADR-0004: Use TimescaleDB Hypertable for Tick-Level Player Position Data

**Date:** 2026-03-31
**Status:** Accepted

## Context

The 2D demo viewer needs per-tick player positions for smooth playback. A typical CS2 demo produces ~1.28 million rows of tick data (10 players x 64 ticks/sec x ~2000 seconds). The primary query pattern is range scans: given a `demo_id`, fetch all ticks in a range for playback or heatmap generation.

Standard PostgreSQL can handle this volume, but query performance degrades as demos accumulate without careful partitioning and index management.

### Alternatives considered

| Approach | Why rejected |
|----------|-------------|
| **Plain PostgreSQL (no partitioning)** | Works initially, but sequential scans on a table with hundreds of millions of rows (across many demos) become slow. Manual partitioning is possible but requires custom DDL and maintenance. |
| **InfluxDB / dedicated TSDB** | Purpose-built for time-series, but adds a second database to operate. Query language (Flux/InfluxQL) differs from SQL. Overkill when the data is naturally scoped by `demo_id`, not wall-clock time. |
| **ClickHouse** | Excellent for analytical queries, but column-oriented storage is less suited to the row-oriented range-scan access pattern. Another service to manage. |
| **Client-side storage (IndexedDB)** | Shifts storage burden to the browser. Demo data must be re-downloaded on every session. No server-side heatmap aggregation possible. |

## Decision

Use TimescaleDB (PostgreSQL extension) with a hypertable for the `player_ticks` table, partitioned by a synthetic timestamp derived from tick number. Compression policy compresses chunks older than 7 days. All queries use standard SQL — TimescaleDB is transparent to the application layer.

Coordinate calibration (world-space to pixel-space) is handled client-side using per-map calibration data, not stored in the database.

## Consequences

### Positive

- Transparent partitioning — the application writes standard `INSERT` and `SELECT` statements
- Chunk-level compression reduces storage 10-20x for older demos
- Range queries on `(demo_id, tick_range)` hit only relevant chunks
- Stays within the PostgreSQL ecosystem — same backup, monitoring, and tooling
- `sqlc` works unchanged since TimescaleDB speaks standard SQL

### Negative

- Requires the TimescaleDB extension in PostgreSQL — slightly more complex Docker image and any managed Postgres must support it
- Synthetic timestamps (tick-based, not wall-clock) are a non-standard use of TimescaleDB's time-oriented partitioning
- Compression makes recent-data updates cheap but historical-data mutations expensive (acceptable since parsed data is immutable)
