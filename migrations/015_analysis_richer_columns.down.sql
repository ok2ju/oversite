-- Drop the slice-10 columns. SQLite supports ALTER TABLE DROP COLUMN since
-- 3.35; modernc.org/sqlite ships a recent enough build. Indexes drop with the
-- table reference; we name them explicitly for safety.

DROP INDEX IF EXISTS idx_analysis_mistakes_round;
DROP INDEX IF EXISTS idx_analysis_mistakes_category;

ALTER TABLE analysis_mistakes DROP COLUMN round_id;
ALTER TABLE analysis_mistakes DROP COLUMN severity;
ALTER TABLE analysis_mistakes DROP COLUMN category;

ALTER TABLE player_match_analysis DROP COLUMN eco_kills;
ALTER TABLE player_match_analysis DROP COLUMN full_buy_adr;
ALTER TABLE player_match_analysis DROP COLUMN repeated_death_zones;
ALTER TABLE player_match_analysis DROP COLUMN isolated_peek_deaths;
ALTER TABLE player_match_analysis DROP COLUMN nades_unused;
ALTER TABLE player_match_analysis DROP COLUMN he_damage;
ALTER TABLE player_match_analysis DROP COLUMN flash_assists;
ALTER TABLE player_match_analysis DROP COLUMN smokes_kill_assist;
ALTER TABLE player_match_analysis DROP COLUMN smokes_thrown;
ALTER TABLE player_match_analysis DROP COLUMN counter_strafe_pct;
ALTER TABLE player_match_analysis DROP COLUMN standing_shot_pct;
ALTER TABLE player_match_analysis DROP COLUMN spray_decay_slope;
ALTER TABLE player_match_analysis DROP COLUMN first_shot_acc_pct;
ALTER TABLE player_match_analysis DROP COLUMN flick_hit_pct;
ALTER TABLE player_match_analysis DROP COLUMN flick_count;
ALTER TABLE player_match_analysis DROP COLUMN time_to_fire_ms_avg;
ALTER TABLE player_match_analysis DROP COLUMN crosshair_height_avg_off;
ALTER TABLE player_match_analysis DROP COLUMN version;

ALTER TABLE player_round_analysis DROP COLUMN shots_hit;
ALTER TABLE player_round_analysis DROP COLUMN shots_fired;
ALTER TABLE player_round_analysis DROP COLUMN nades_unused;
ALTER TABLE player_round_analysis DROP COLUMN nades_used;
ALTER TABLE player_round_analysis DROP COLUMN money_spent;
ALTER TABLE player_round_analysis DROP COLUMN buy_type;
