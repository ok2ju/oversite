-- Per-(demo, player, round) analysis row computed alongside
-- player_match_analysis. Where player_match_analysis stores the aggregate row
-- driving the viewer's "Overall: NN/100" header, this table breaks the same
-- per-player metrics down to one row per round so the standalone analysis
-- page can render per-round drilldowns (slice 7's per-round trade_pct bar
-- chart).
--
-- Slice 7 only populates the trade column (trade_pct); subsequent slices add
-- other categories (utility, aim, …) without changing the row shape — any
-- category-specific context that doesn't fit a column lives in extras_json,
-- mirroring the analysis_mistakes / player_match_analysis pattern.
--
-- Keyed by (demo_id, steam_id, round_number) so re-running the analyzer for
-- a demo upserts a single row per (player, round) pair; the unique index lets
-- the analyzer's delete-then-insert path resolve to one row.
CREATE TABLE player_round_analysis (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    demo_id         INTEGER NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    steam_id        TEXT    NOT NULL,
    round_number    INTEGER NOT NULL,
    trade_pct       REAL    NOT NULL DEFAULT 0,
    extras_json     TEXT    NOT NULL DEFAULT '{}',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX idx_player_round_analysis_demo_player_round
    ON player_round_analysis(demo_id, steam_id, round_number);
