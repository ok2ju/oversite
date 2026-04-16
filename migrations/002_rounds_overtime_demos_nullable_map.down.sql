-- Remove is_overtime column from rounds.
-- SQLite doesn't support DROP COLUMN in older versions, so recreate.
CREATE TABLE rounds_old (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    demo_id         INTEGER NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    round_number    INTEGER NOT NULL,
    start_tick      INTEGER NOT NULL,
    end_tick        INTEGER NOT NULL,
    winner_side     TEXT    NOT NULL,
    win_reason      TEXT    NOT NULL,
    ct_score        INTEGER NOT NULL DEFAULT 0,
    t_score         INTEGER NOT NULL DEFAULT 0
);

INSERT INTO rounds_old SELECT id, demo_id, round_number, start_tick, end_tick, winner_side, win_reason, ct_score, t_score FROM rounds;

DROP TABLE rounds;

ALTER TABLE rounds_old RENAME TO rounds;

CREATE INDEX idx_rounds_demo_id ON rounds(demo_id);

-- Restore demos with NOT NULL map_name (no DEFAULT).
CREATE TABLE demos_old (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    faceit_match_id TEXT,
    map_name        TEXT    NOT NULL,
    file_path       TEXT    NOT NULL,
    file_size       INTEGER NOT NULL,
    status          TEXT    NOT NULL DEFAULT 'imported',
    total_ticks     INTEGER NOT NULL DEFAULT 0,
    tick_rate       REAL    NOT NULL DEFAULT 0,
    duration_secs   INTEGER NOT NULL DEFAULT 0,
    match_date      TEXT    NOT NULL DEFAULT '',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO demos_old SELECT * FROM demos;

DROP TABLE demos;

ALTER TABLE demos_old RENAME TO demos;

CREATE INDEX idx_demos_user_id ON demos(user_id);
CREATE INDEX idx_demos_status ON demos(status);
