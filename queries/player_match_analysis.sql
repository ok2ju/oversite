-- name: UpsertPlayerMatchAnalysis :exec
-- Insert or replace the (demo, player) summary row. The unique index on
-- (demo_id, steam_id) gives the ON CONFLICT clause a single target row.
INSERT INTO player_match_analysis (
    demo_id, steam_id,
    overall_score, trade_pct, avg_trade_ticks,
    version,
    crosshair_height_avg_off, time_to_fire_ms_avg, flick_count, flick_hit_pct,
    first_shot_acc_pct, spray_decay_slope,
    standing_shot_pct, counter_strafe_pct,
    smokes_thrown, smokes_kill_assist, flash_assists, he_damage, nades_unused,
    isolated_peek_deaths, repeated_death_zones,
    full_buy_adr, eco_kills,
    extras_json
) VALUES (
    @demo_id, @steam_id,
    @overall_score, @trade_pct, @avg_trade_ticks,
    @version,
    @crosshair_height_avg_off, @time_to_fire_ms_avg, @flick_count, @flick_hit_pct,
    @first_shot_acc_pct, @spray_decay_slope,
    @standing_shot_pct, @counter_strafe_pct,
    @smokes_thrown, @smokes_kill_assist, @flash_assists, @he_damage, @nades_unused,
    @isolated_peek_deaths, @repeated_death_zones,
    @full_buy_adr, @eco_kills,
    @extras_json
)
ON CONFLICT(demo_id, steam_id) DO UPDATE SET
    overall_score            = excluded.overall_score,
    trade_pct                = excluded.trade_pct,
    avg_trade_ticks          = excluded.avg_trade_ticks,
    version                  = excluded.version,
    crosshair_height_avg_off = excluded.crosshair_height_avg_off,
    time_to_fire_ms_avg      = excluded.time_to_fire_ms_avg,
    flick_count              = excluded.flick_count,
    flick_hit_pct            = excluded.flick_hit_pct,
    first_shot_acc_pct       = excluded.first_shot_acc_pct,
    spray_decay_slope        = excluded.spray_decay_slope,
    standing_shot_pct        = excluded.standing_shot_pct,
    counter_strafe_pct       = excluded.counter_strafe_pct,
    smokes_thrown            = excluded.smokes_thrown,
    smokes_kill_assist       = excluded.smokes_kill_assist,
    flash_assists            = excluded.flash_assists,
    he_damage                = excluded.he_damage,
    nades_unused             = excluded.nades_unused,
    isolated_peek_deaths     = excluded.isolated_peek_deaths,
    repeated_death_zones     = excluded.repeated_death_zones,
    full_buy_adr             = excluded.full_buy_adr,
    eco_kills                = excluded.eco_kills,
    extras_json              = excluded.extras_json;

-- name: GetPlayerMatchAnalysis :one
-- Returns a single (demo, player) summary row or sql.ErrNoRows if none exists.
SELECT *
FROM player_match_analysis
WHERE demo_id = @demo_id
  AND steam_id = @steam_id;

-- name: ListPlayerMatchAnalysisByDemoID :many
-- Returns every row persisted for a demo, ordered for stable rendering. Used
-- by GetMatchInsights to compute team-level aggregates without N round trips.
SELECT *
FROM player_match_analysis
WHERE demo_id = @demo_id
ORDER BY steam_id ASC;

-- name: DeletePlayerMatchAnalysisByDemoID :exec
DELETE FROM player_match_analysis
WHERE demo_id = @demo_id;

-- name: CountPlayerMatchAnalysisByDemoID :one
-- Returns the number of summary rows persisted for a demo. The slice-5
-- analyzer writes one row per rostered player on every successful parse, so
-- 0 is the missing-analysis sentinel for legacy demos imported before slice 1.
SELECT count(*) FROM player_match_analysis WHERE demo_id = @demo_id;
