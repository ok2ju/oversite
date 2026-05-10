-- Per-(demo, player) summary analysis row computed alongside the existing
-- analysis_mistakes timeline. Where analysis_mistakes records each individual
-- finding, this table stores aggregated metrics that drive the viewer's
-- "Overall: NN/100" header and per-category cards.
--
-- Slice 5 only populates the trade category (trade_pct, avg_trade_ticks);
-- subsequent slices add other categories (utility, aim, …) without changing
-- the row shape — category-specific context that doesn't fit a column lives
-- in extras_json, mirroring the analysis_mistakes pattern.
--
-- The row is keyed by (demo_id, steam_id) so re-running the analyzer for a
-- demo upserts a single row per player; the unique index lets the
-- ON CONFLICT path resolve to one row.
CREATE TABLE player_match_analysis (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    demo_id         INTEGER NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    steam_id        TEXT    NOT NULL,
    overall_score   INTEGER NOT NULL DEFAULT 0,
    trade_pct       REAL    NOT NULL DEFAULT 0,
    avg_trade_ticks REAL    NOT NULL DEFAULT 0,
    extras_json     TEXT    NOT NULL DEFAULT '{}',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX idx_player_match_analysis_demo_player
    ON player_match_analysis(demo_id, steam_id);
