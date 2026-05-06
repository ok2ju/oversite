-- Remove Faceit integration and the users table.
-- The app pivots to a single-tenant local tool — no auth, no Faceit sync.

-- Drop dependent table first (faceit_matches references both users and demos).
DROP INDEX IF EXISTS idx_faceit_matches_user_id;
DROP INDEX IF EXISTS idx_faceit_matches_played_at;
DROP TABLE IF EXISTS faceit_matches;

-- Recreate demos without user_id and faceit_match_id columns.
DROP INDEX IF EXISTS idx_demos_user_id;
DROP INDEX IF EXISTS idx_demos_status;

CREATE TABLE demos_new (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    map_name        TEXT    NOT NULL DEFAULT '',
    file_path       TEXT    NOT NULL,
    file_size       INTEGER NOT NULL,
    status          TEXT    NOT NULL DEFAULT 'imported',
    total_ticks     INTEGER NOT NULL DEFAULT 0,
    tick_rate       REAL    NOT NULL DEFAULT 0,
    duration_secs   INTEGER NOT NULL DEFAULT 0,
    match_date      TEXT    NOT NULL DEFAULT '',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO demos_new (id, map_name, file_path, file_size, status, total_ticks, tick_rate, duration_secs, match_date, created_at)
SELECT id, map_name, file_path, file_size, status, total_ticks, tick_rate, duration_secs, match_date, created_at
FROM demos;

DROP TABLE demos;

ALTER TABLE demos_new RENAME TO demos;

CREATE INDEX idx_demos_status ON demos(status);

-- Drop the users table last (nothing references it now).
DROP TABLE IF EXISTS users;
