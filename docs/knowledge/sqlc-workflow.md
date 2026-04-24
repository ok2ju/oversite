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

## Query file organization

One `.sql` file per domain (demos, rounds, tick_data, events, lineups, users, faceit_matches, boards). Each file opens with schema comments describing the table's shape for readers who don't have the full DDL in mind.
