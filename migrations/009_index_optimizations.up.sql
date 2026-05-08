-- Index optimizations on game_events identified by perf audit.
-- 1. Drop redundant idx_game_events_demo_id: the composite
--    idx_game_events_type on (demo_id, event_type) already serves any
--    WHERE demo_id = ? lookup as a left-prefix scan, so the single-column
--    index only adds insert-time write overhead.
-- 2. Add a covering index for the heatmap aggregation query in
--    internal/store/heatmaps_custom.go which filters by event_type='kill'
--    and demo_id, then GROUPs BY x, y. With (event_type, demo_id, x, y)
--    SQLite can satisfy the query index-only.
DROP INDEX IF EXISTS idx_game_events_demo_id;
CREATE INDEX IF NOT EXISTS idx_game_events_heatmap ON game_events(event_type, demo_id, x, y);
