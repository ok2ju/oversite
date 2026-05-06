---
name: create-migration
description: Create a new SQL migration file pair for golang-migrate with proper numbering
disable-model-invocation: true
---

# Create Migration

Creates a properly numbered golang-migrate migration file pair (up + down) in `migrations/`.

## Arguments

- `$ARGUMENTS` — the migration name in snake_case (e.g., `add_user_preferences`)

## Workflow

1. Find the highest existing migration number in `migrations/`
2. Increment by 1 and zero-pad to 6 digits
3. Create both files:
   - `migrations/{number}_{name}.up.sql`
   - `migrations/{number}_{name}.down.sql`
4. Add a header comment with the migration purpose
5. For the up migration: include a placeholder `CREATE TABLE` or `ALTER TABLE` statement based on the name
6. For the down migration: include the corresponding `DROP TABLE` or reverse `ALTER TABLE`
7. Remind the user to:
   - Fill in the actual SQL
   - Run `make migrate-up` to apply
   - Run `make sqlc` if the migration adds/changes tables referenced in `queries/*.sql`
   - Test rollback with `make migrate-down`

## Naming Convention

Migration names should be descriptive and use snake_case:
- `add_match_vod_url`
- `create_grenade_lineups_index`
- `add_demos_status_index`

## Example

```
/create-migration add_match_vod_url
```

Creates:
- `migrations/000005_add_match_vod_url.up.sql`
- `migrations/000005_add_match_vod_url.down.sql`
