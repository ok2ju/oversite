-- Reverse 009_index_optimizations.up.sql: drop the heatmap covering index
-- and restore the original single-column demo_id index.
DROP INDEX IF EXISTS idx_game_events_heatmap;
CREATE INDEX IF NOT EXISTS idx_game_events_demo_id ON game_events(demo_id);
