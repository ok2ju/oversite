-- name: UpsertPlayerRoundAnalysis :exec
-- Insert or replace the (demo, player, round) row. The unique index on
-- (demo_id, steam_id, round_number) gives the ON CONFLICT clause a single
-- target row.
INSERT INTO player_round_analysis (
    demo_id, steam_id, round_number, trade_pct,
    buy_type, money_spent, nades_used, nades_unused,
    shots_fired, shots_hit,
    extras_json
) VALUES (
    @demo_id, @steam_id, @round_number, @trade_pct,
    @buy_type, @money_spent, @nades_used, @nades_unused,
    @shots_fired, @shots_hit,
    @extras_json
)
ON CONFLICT(demo_id, steam_id, round_number) DO UPDATE SET
    trade_pct    = excluded.trade_pct,
    buy_type     = excluded.buy_type,
    money_spent  = excluded.money_spent,
    nades_used   = excluded.nades_used,
    nades_unused = excluded.nades_unused,
    shots_fired  = excluded.shots_fired,
    shots_hit    = excluded.shots_hit,
    extras_json  = excluded.extras_json;

-- name: GetPlayerRoundAnalysisByDemoAndPlayer :many
-- Returns every (demo, player) round row ordered by round_number ASC so the
-- frontend can render the per-round bar chart in match cadence without an
-- extra sort.
SELECT *
FROM player_round_analysis
WHERE demo_id = @demo_id
  AND steam_id = @steam_id
ORDER BY round_number ASC;

-- name: DeletePlayerRoundAnalysisByDemoID :exec
DELETE FROM player_round_analysis
WHERE demo_id = @demo_id;

-- name: CountPlayerRoundAnalysisByDemoID :one
-- Returns the number of round-level rows persisted for a demo. The slice-7
-- analyzer writes one row per (rostered player, round-with-eligible-deaths),
-- so 0 is the missing-analysis sentinel for demos imported before slice 7.
SELECT count(*) FROM player_round_analysis WHERE demo_id = @demo_id;
