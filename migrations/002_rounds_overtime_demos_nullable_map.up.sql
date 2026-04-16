-- Add is_overtime column to rounds table.
ALTER TABLE rounds ADD COLUMN is_overtime INTEGER NOT NULL DEFAULT 0;

-- Recreate demos table with nullable map_name.
-- SQLite doesn't support ALTER COLUMN, so we recreate the table.
CREATE TABLE demos_new (
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

INSERT INTO demos_new SELECT * FROM demos;

DROP TABLE demos;

ALTER TABLE demos_new RENAME TO demos;

CREATE INDEX idx_demos_user_id ON demos(user_id);
CREATE INDEX idx_demos_status ON demos(status);
