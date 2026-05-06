-- Best-effort rollback for migration 005.
-- This is a one-way change in practice; deleted rows are not recoverable.

CREATE TABLE IF NOT EXISTS users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    faceit_id       TEXT    NOT NULL UNIQUE,
    nickname        TEXT    NOT NULL,
    avatar_url      TEXT    NOT NULL DEFAULT '',
    faceit_elo      INTEGER NOT NULL DEFAULT 0,
    faceit_level    INTEGER NOT NULL DEFAULT 0,
    country         TEXT    NOT NULL DEFAULT '',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

DROP INDEX IF EXISTS idx_demos_status;

CREATE TABLE demos_old (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    faceit_match_id TEXT,
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

INSERT INTO demos_old (id, user_id, map_name, file_path, file_size, status, total_ticks, tick_rate, duration_secs, match_date, created_at)
SELECT id, 0, map_name, file_path, file_size, status, total_ticks, tick_rate, duration_secs, match_date, created_at
FROM demos;

DROP TABLE demos;

ALTER TABLE demos_old RENAME TO demos;

CREATE INDEX idx_demos_user_id ON demos(user_id);
CREATE INDEX idx_demos_status ON demos(status);

CREATE TABLE IF NOT EXISTS faceit_matches (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    faceit_match_id TEXT    NOT NULL,
    map_name        TEXT    NOT NULL,
    score_team      INTEGER NOT NULL,
    score_opponent  INTEGER NOT NULL,
    result          TEXT    NOT NULL,
    elo_before      INTEGER NOT NULL DEFAULT 0,
    elo_after       INTEGER NOT NULL DEFAULT 0,
    kills           INTEGER NOT NULL DEFAULT 0,
    deaths          INTEGER NOT NULL DEFAULT 0,
    assists         INTEGER NOT NULL DEFAULT 0,
    adr             REAL    NOT NULL DEFAULT 0,
    demo_url        TEXT    NOT NULL DEFAULT '',
    demo_id         INTEGER REFERENCES demos(id) ON DELETE SET NULL,
    played_at       TEXT    NOT NULL,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, faceit_match_id)
);

CREATE INDEX idx_faceit_matches_user_id ON faceit_matches(user_id);
CREATE INDEX idx_faceit_matches_played_at ON faceit_matches(user_id, played_at);
