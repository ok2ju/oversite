# ADR-0016: Multi-Connection SQLite Pool with `busy_timeout`

**Date:** 2026-05-08
**Status:** Accepted

Refines the operational guidance behind [ADR-0008](0008-sqlite-local-database.md) (does not supersede it — ADR-0008 is the engine choice; this is connection-pool tuning under that choice).

## Context

Through migration 008 the SQLite pool was configured with `SetMaxOpenConns(1)`. The reasoning at the time was simple-correct: `modernc.org/sqlite` + WAL mode + a single connection avoids `SQLITE_BUSY` races by construction, because Go's `database/sql` serializes every call through the one connection.

Two problems surfaced in practice:

- **Reads block during ingest.** A 300K-row tick ingest holds the single connection for several seconds; the library list, heatmap aggregations, and scoreboard all queue behind it. The UI freezes during what should be a background operation.
- **Default `synchronous=FULL`** fsyncs every WAL frame. Inside a single ingest transaction this still produces hundreds of thousands of fsyncs, multiplying ingest time well beyond what the disk should require.

The goals of this change:

- Allow concurrent reads during long writes (the original WAL promise from ADR-0008).
- Keep write correctness — no `SQLITE_BUSY` surfaced to callers.
- Get fsync overhead off the critical path of ingest.

### Alternatives considered

| Approach | Why rejected |
|----------|--------------|
| **Status quo (`MaxOpenConns=1`)** | UI blocks during every ingest. Only viable if ingest stays sub-second, which it doesn't. |
| **Two `*sql.DB` pools — one read, one write** | Hard to enforce in Go's `database/sql`: nothing stops a "read pool" connection from running an `INSERT`. Doubles the number of file handles. The shared cache_size also gets harder to reason about. |
| **Larger pool, no `busy_timeout`** | SQLite still serializes writes internally — second writer hits `SQLITE_BUSY`, which `database/sql` surfaces directly to the caller. Forces every call site to retry. |
| **`BEGIN IMMEDIATE` for writes** | Acquires the write lock at transaction start instead of at first write, but doesn't solve the contention surfacing problem on its own. `busy_timeout` does the same job with less code. |
| **Switch to a CGo SQLite driver to use `sqlite3_busy_handler` directly** | ADR-0008 explicitly chose pure-Go to avoid C toolchains in cross-compilation. `busy_timeout` is portable across drivers. |

## Decision

Configure the pool and per-connection pragmas as follows in `internal/database/sqlite.go`:

- **`SetMaxOpenConns(4)`.** Empirically chosen — enough to absorb a write + 2–3 concurrent UI reads, not so high that it inflates page cache duplication.
- **Per-connection pragmas applied on every connection** (not just at startup):
  - `journal_mode=WAL` — reader/writer concurrency (unchanged from ADR-0008).
  - `foreign_keys=ON` — required for FK enforcement (unchanged).
  - `synchronous=NORMAL` — avoids per-WAL-frame fsyncs. Crash safety is still strong inside WAL — only the last not-yet-committed transaction can be lost on a hard power loss; committed transactions are durable.
  - `busy_timeout=5000` — 5 s wait on lock contention before erroring. SQLite serializes writes (one writer holds the WAL lock at a time) and `busy_timeout` absorbs that contention without surfacing `SQLITE_BUSY` to callers.
  - `cache_size=-64000` (~64 MB) — hot pages stay resident.
  - `mmap_size=268435456` (256 MB) — read path skips the page cache for large demos.
  - `temp_store=MEMORY` — sort/group spills don't hit disk.
  - `journal_size_limit` — bounded so the WAL file doesn't grow unbounded across long sessions.

The contract is: SQLite handles write serialization (WAL lock + `busy_timeout`); the Go pool serves reads in parallel.

## Consequences

### Positive

- Library browse, heatmap aggregation, and scoreboard reads run in parallel with ingest — measurable UI responsiveness improvement during imports.
- `synchronous=NORMAL` alone gives a 3–10× ingest speedup on typical disks.
- `mmap_size` + larger `cache_size` reduce read latency for the viewer's tick range scans.
- No call-site changes — all queries continue to use the shared `*sql.DB`.
- ADR-0008's "one writer at a time" invariant is preserved — it just moves from the Go pool to SQLite's WAL lock.

### Negative

- The architecture/wiki guidance "don't open a second connection — `MaxOpenConns(1)`" had to be retracted; future contributors must not assume single-connection semantics. `[[knowledge/sqlite-wal]]` was rewritten to reflect the new pool model.
- Under heavy contention, a stuck writer can block other writers for up to 5 s before erroring. In a single-user desktop app this is acceptable — there's no scenario where a writer is genuinely stuck for that long without it also being a bug worth surfacing.
- `synchronous=NORMAL` trades a tiny crash-safety window (the last in-flight transaction may not survive a hard power loss) for the ingest throughput win. Acceptable for a desktop app where the user can re-import a demo if their machine power-fails mid-parse; would not be acceptable for a transactional system of record.
- More resource use: 4 connections + 256 MB mmap + 64 MB cache. On a modern desktop this is rounding error; on a constrained machine it's the largest non-WebView memory consumer in the process.
