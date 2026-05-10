-- name: UpsertPlayerMatchAnalysis :exec
-- Insert or replace the (demo, player) summary row. The unique index on
-- (demo_id, steam_id) gives the ON CONFLICT clause a single target row.
INSERT INTO player_match_analysis (demo_id, steam_id, overall_score, trade_pct, avg_trade_ticks, extras_json)
VALUES (@demo_id, @steam_id, @overall_score, @trade_pct, @avg_trade_ticks, @extras_json)
ON CONFLICT(demo_id, steam_id) DO UPDATE SET
    overall_score = excluded.overall_score,
    trade_pct = excluded.trade_pct,
    avg_trade_ticks = excluded.avg_trade_ticks,
    extras_json = excluded.extras_json;

-- name: GetPlayerMatchAnalysis :one
-- Returns a single (demo, player) summary row or sql.ErrNoRows if none exists.
SELECT id, demo_id, steam_id, overall_score, trade_pct, avg_trade_ticks, extras_json, created_at
FROM player_match_analysis
WHERE demo_id = @demo_id
  AND steam_id = @steam_id;

-- name: DeletePlayerMatchAnalysisByDemoID :exec
DELETE FROM player_match_analysis
WHERE demo_id = @demo_id;

-- name: CountPlayerMatchAnalysisByDemoID :one
-- Returns the number of summary rows persisted for a demo. The slice-5
-- analyzer writes one row per rostered player on every successful parse, so
-- 0 is the missing-analysis sentinel for legacy demos imported before slice 1.
SELECT count(*) FROM player_match_analysis WHERE demo_id = @demo_id;
