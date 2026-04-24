# ADR-0010: sqlc for Type-Safe SQL Generation

**Date:** 2026-04-12
**Status:** Accepted

## Context

The desktop app uses SQLite ([ADR-0008](0008-sqlite-local-database.md)) and needs a Go data access layer. The web version used sqlc with PostgreSQL, which worked well. For the desktop pivot, we need to choose how to interact with SQLite from Go.

### Alternatives considered

| Approach | Why rejected |
|----------|-------------|
| **GORM** | Heavy ORM with implicit behavior (lazy loading, auto-migrations). Harder to reason about generated SQL. SQLite support exists but GORM's query builder can generate inefficient queries for batch inserts and complex joins. Magic struct tags obscure the actual SQL. |
| **Raw `database/sql`** | Maximum control but no compile-time type safety. Easy to introduce runtime errors from typos in column names or mismatched scan targets. Tedious to maintain as schema evolves. |
| **sqlx** | Better than raw `database/sql` (struct scanning, named parameters) but still no compile-time verification of SQL against schema. Errors surface at runtime. |
| **Ent** | Facebook's entity framework for Go. Code-gen from schema, but opinionated graph-based API doesn't match our relational query patterns (range scans on tick_data, aggregations for heatmaps). |

## Decision

Use **sqlc** with the SQLite dialect. SQL queries are written in `.sql` files under `queries/`, and sqlc generates type-safe Go code in `internal/store/`.

Key constraints of sqlc's SQLite dialect vs PostgreSQL:
- No `ANY()` — use `IN` with explicit parameter lists or `json_each()` for dynamic lists
- No `cardinality()` — use `json_array_length()` or subquery counts
- No `sqlc.narg()` for nullable parameters — use `CASE WHEN` or `COALESCE` patterns
- `RETURNING *` supported (SQLite 3.35+, available in modernc.org/sqlite)

### Workflow

1. Write SQL in `queries/*.sql` with sqlc annotations
2. Run `make sqlc` to generate Go code
3. Generated `*.sql.go` files are committed but protected by PreToolUse hook (no manual edits)

## Consequences

### Positive

- SQL is the source of truth — no ORM abstraction hiding the actual queries
- Compile-time type safety: schema changes break the build immediately if queries are stale
- Generated code is straightforward, readable, and debuggable
- Same tool used in web version — team familiarity carries over
- Query performance is predictable (you write the SQL, sqlc just wraps it)

### Negative

- SQLite dialect has fewer features than PostgreSQL — some queries need rewriting from web version
- Dynamic queries (variable WHERE clauses) are awkward — may need multiple query variants or `json_each()` workarounds
- sqlc config and query annotations add a learning curve for contributors unfamiliar with the tool
- Generated code must be regenerated after any `.sql` file change (enforced by Makefile target)
