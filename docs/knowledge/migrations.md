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
- **Expressions are forbidden in `PRIMARY KEY` / `UNIQUE` table constraints.** SQLite rejects `PRIMARY KEY (a, b, COALESCE(c, -1))` at `CREATE TABLE` time with `"expressions prohibited in PRIMARY KEY and UNIQUE constraints"`. The workaround is to drop the table-level PK and add a separate `CREATE UNIQUE INDEX idx_x ON foo(a, b, COALESCE(c, -1))` — unique indexes *do* support expressions, including `COALESCE` for nullable composite uniqueness (which is the only way to make `UNIQUE` treat two NULL `c` rows as conflicting, since `UNIQUE` itself treats NULLs as distinct). Hit on migration 021 (`contact_mistakes`); see `migrations/021_contact_mistakes.up.sql` for the index form.

## Running

Migrations run automatically at startup in `internal/database/sqlite.go` → `RunMigrations(db)`. A corruption check (`PRAGMA integrity_check`) runs first; a pre-migration backup (`oversite.db.bak`) is made.

## Testing migrations

`testutil.NewTestDB(t)` from `internal/testutil/db.go` creates a fresh in-memory SQLite with all migrations applied. Use this for any test that touches the store. Never open a test DB manually.

## Pattern: promoting hot JSON fields to columns

When an `extra_data` (or similar TEXT-JSON) field becomes a query hotspot — `json_extract(...)` in a `WHERE` or `GROUP BY`, or a frontend hot read — promote it to a real column. Migration 010 (`game_events_promoted_columns`) is the worked example: `headshot`, `assister_steam_id`, `health_damage`, `attacker_name`, `victim_name`, `attacker_team`, `victim_team`, `is_self_kill` moved out of `extra_data`. Eliminates triple JSON serialization (SQLite TEXT → Go decode → Wails → JS), enables an index, and keeps the remaining `extra_data` blob as the JSON catch-all.

## Pattern: per-N storage for rarely-changing fields

Migration 011 (`round_loadouts`) moved `tick_data.inventory` from per-tick to per-round. 1.28M rows × ~30 B → 250 rows × ~30 B per demo. Apply this whenever a per-tick column only changes on round boundaries.

## Pattern: nullable FK as the loose-coupling join

`analysis_mistakes.duel_id INTEGER REFERENCES analysis_duels(id) ON DELETE SET NULL` (migration 018) lets a mistake belong to a duel without making the relationship mandatory — cross-duel patterns (`eco_misbuy`, `he_damage`) keep `duel_id = NULL` and render in a separate UI surface. `ON DELETE SET NULL` (not CASCADE) means a pruned duel doesn't take its mistakes with it; the analyzer can re-run and re-attribute without losing history. Old rows from prior `AnalysisVersion` also keep `duel_id = NULL` until the user clicks Recompute.
