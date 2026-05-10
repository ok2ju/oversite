-- Per-player mechanical-analysis findings discovered post-ingest.
--
-- Each row is a single instance of a named "mistake" attributed to a player
-- in a specific round (e.g. an untraded death). The analyzer runs after
-- IngestGameEvents in parseDemo and persists the rows in a transaction that
-- first deletes the demo's existing rows, so re-importing or re-parsing a
-- demo always converges on a single deterministic set.
--
-- extras_json carries rule-specific context (offending opponent, weapon,
-- delta-tick, …) — the schema stays narrow so adding a new rule does not
-- require a migration.
CREATE TABLE analysis_mistakes (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    demo_id      INTEGER NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    steam_id     TEXT    NOT NULL,
    round_number INTEGER NOT NULL,
    tick         INTEGER NOT NULL,
    kind         TEXT    NOT NULL,
    extras_json  TEXT    NOT NULL DEFAULT '{}',
    created_at   TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_analysis_mistakes_demo_player_tick
    ON analysis_mistakes(demo_id, steam_id, tick);
