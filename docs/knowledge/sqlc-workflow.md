# sqlc Workflow

**Related:** [[sqlite-wal]] · [[migrations]] · [ADR-0010](../decisions/0010-sqlc-type-safe-sql.md)

## The three-directory roundtrip

```
queries/   ← human-written SQL (source of truth)
   │
   │  make sqlc   (runs `go tool sqlc generate`)
   ▼
internal/store/   ← generated Go: types, Queries struct, method per query
```

`queries/*.sql` contains `-- name: ImportDemo :one` annotations. `make sqlc` (or `go tool sqlc generate`) reads them and regenerates `internal/store/*.sql.go`. The generated files are committed to git.

## Editing rules

- **Never hand-edit `internal/store/*.sql.go`.** A PreToolUse hook blocks these edits. If you need a query to behave differently, change the SQL in `queries/` and re-run `make sqlc`.
- **Always re-run sqlc after migration changes.** New columns/tables that the queries reference must exist in the schema the generator sees.
- **Stage both files together** — the `.sql` and the regenerated `.sql.go`.

## sqlc dialect

Configured for SQLite in `sqlc.yaml`. Note that sqlc's SQLite support is newer than its PostgreSQL support — a few things (like `RETURNING` with expressions) have rough edges. If a query gets weird, run the same SQL in `sqlite3` CLI first to make sure it's valid.

### `:one` with `RETURNING id` returns a bare scalar

When a `-- name: CreateFoo :one` query ends `RETURNING id` (no other columns), sqlc generates `func ... CreateFoo(...) (int64, error)` — not a row struct. Other `:one` queries (`SELECT id, ...`) return a generated struct. Mind the shape when reading the rowid out:

```go
rowID, err := q.CreateAnalysisDuel(ctx, params)   // bare int64
```

Worked example: `queries/analysis_duels.sql` → `internal/store/analysis_duels.sql.go`.

### `COALESCE(MAX(col), 0)` in a `:one` returns `interface{}`

sqlc's SQLite generator can't infer the column type of `SELECT COALESCE(MAX(builder_version), 0) AS version` and produces `func (q *Queries) MaxBuilderVersionForDemo(ctx, demoID int64) (interface{}, error)`. Same shape for the Phase 3 `MaxDetectorVersionForDemo` query. Callers must type-assert; the contacts/detectors runner uses a small helper:

```go
func coerceInt64(v interface{}) int64 {
    switch n := v.(type) {
    case int64: return n
    case int:   return int64(n)
    case nil:   return 0
    default:    return 0
    }
}
```

SQLite returns the column as `int64` in practice, but the `nil` branch matters for an empty table (the wrapping COALESCE means it shouldn't fire, but defense in depth costs nothing).

### Two-pass insert for self-referential FKs

`analysis_duels.mutual_duel_id` references `analysis_duels.id`. Both peers need rowids before they can link, so the persistence layer in `internal/demo/analysis/persist.go` does two passes inside the same tx: insert every duel and collect `localID → rowid` in a map, then `UPDATE analysis_duels SET mutual_duel_id = ?` for each linked pair. Avoids deferred-constraint gymnastics SQLite doesn't support cleanly.

## Query file organization

One `.sql` file per domain (demos, rounds, tick_data, events, lineups, boards). Each file opens with schema comments describing the table's shape for readers who don't have the full DDL in mind.

## When sqlc isn't enough — `*_custom.go` files

Some queries don't fit sqlc's SQLite generator cleanly. We write hand-rolled stores alongside the generated package, named `*_custom.go`:

- `internal/store/heatmaps_custom.go` — heatmap aggregation with dynamic filters.
- `internal/store/game_events_custom.go` — `GetGameEventsByTypes` uses `WHERE event_type IN (SELECT value FROM json_each(?))` to take a `[]string` parameter; sqlc can't bind a slice to an `IN (...)` list.

Custom stores use the same `*Queries` receiver as the generated code, so callers don't see the difference. Keep them small and use the generated types (`store.GameEvent`, etc.) as result rows so consumers stay symmetric.

## Prepared statements within an ingest tx

For ingest hot loops (300K+ inserts), bare `tx.ExecContext(ctx, query, ...)` re-parses SQL on every call. Prepare once per ingest tx with `tx.PrepareContext(ctx, fullBatchSQL)` + a partial-batch statement for the remainder, then `stmt.ExecContext` per batch. `internal/demo/ingest.go` is the worked example. Don't apply this universally — only where re-parsing dominates.
