-- Slice 10: extend the three analysis tables with the columns the plan
-- ("Mechanical Analysis Integration") originally specified. All additions are
-- nullable / default-zero so the migration is forward-only and a no-op for
-- demos that haven't been recomputed yet — the next RecomputeAnalysis or
-- import populates the new columns. Existing readers ignore unknown columns.

-- analysis_mistakes: per-rule category/severity tags + a back-reference to the
-- canonical round so the cascade-delete via rounds(id) lines up with the plan.
-- round_id is nullable for legacy rows; new persistence always fills it.
ALTER TABLE analysis_mistakes ADD COLUMN category TEXT NOT NULL DEFAULT '';
ALTER TABLE analysis_mistakes ADD COLUMN severity INTEGER NOT NULL DEFAULT 1;
ALTER TABLE analysis_mistakes ADD COLUMN round_id INTEGER REFERENCES rounds(id) ON DELETE CASCADE;
CREATE INDEX idx_analysis_mistakes_round ON analysis_mistakes(round_id);
CREATE INDEX idx_analysis_mistakes_category ON analysis_mistakes(demo_id, category);

-- player_match_analysis: promote the per-category aggregates the plan calls
-- out so the frontend reads them as columns instead of poking through extras.
-- All NOT NULL DEFAULT 0 — older rows carry the defaults until the next
-- recompute fills them in.
ALTER TABLE player_match_analysis ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
-- aim
ALTER TABLE player_match_analysis ADD COLUMN crosshair_height_avg_off REAL NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN time_to_fire_ms_avg REAL NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN flick_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN flick_hit_pct REAL NOT NULL DEFAULT 0;
-- spray
ALTER TABLE player_match_analysis ADD COLUMN first_shot_acc_pct REAL NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN spray_decay_slope REAL NOT NULL DEFAULT 0;
-- movement
ALTER TABLE player_match_analysis ADD COLUMN standing_shot_pct REAL NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN counter_strafe_pct REAL NOT NULL DEFAULT 0;
-- utility
ALTER TABLE player_match_analysis ADD COLUMN smokes_thrown INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN smokes_kill_assist INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN flash_assists INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN he_damage INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN nades_unused INTEGER NOT NULL DEFAULT 0;
-- positioning
ALTER TABLE player_match_analysis ADD COLUMN isolated_peek_deaths INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN repeated_death_zones INTEGER NOT NULL DEFAULT 0;
-- economy
ALTER TABLE player_match_analysis ADD COLUMN full_buy_adr REAL NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN eco_kills INTEGER NOT NULL DEFAULT 0;

-- player_round_analysis: per-round economy + nade usage so the analysis page's
-- per-round drilldown can render buy classification and nade-spend bars.
ALTER TABLE player_round_analysis ADD COLUMN buy_type TEXT NOT NULL DEFAULT '';
ALTER TABLE player_round_analysis ADD COLUMN money_spent INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_round_analysis ADD COLUMN nades_used INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_round_analysis ADD COLUMN nades_unused INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_round_analysis ADD COLUMN shots_fired INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_round_analysis ADD COLUMN shots_hit INTEGER NOT NULL DEFAULT 0;
