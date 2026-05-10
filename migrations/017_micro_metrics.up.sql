-- P3-2: micro-skill metric columns on player_match_analysis. Each one is a
-- per-player aggregate the slice-11 micro card grid renders directly:
--
-- - time_to_stop_ms_avg:      average ms between "running" (>= 100 u/s) and
--                             "stopped" (<= 40 u/s) before fires that produced
--                             a counter-strafe — quality of the strafe-aim
--                             stop. 0 when no eligible strafe-aim shots.
-- - crouch_before_shot_count: count of fires where the player was crouched at
--                             the moment of fire (Player.IsDucking()).
-- - crouch_instead_of_strafe_count: count of fires where the player crouched
--                             AND was still moving (no counter-strafe stop) —
--                             the failure mode the plan calls out.
-- - flick_overshoot_avg_deg:  mean signed flick error past the target — small
--                             positive number on a balanced player.
-- - flick_undershoot_avg_deg: mean unsigned undershoot error.
-- - flick_balance_pct:        100 * over / (over + under). 50 = balanced.
--
-- All NOT NULL DEFAULT 0; legacy rows pick up zeros until the next analyzer
-- run (RecomputeAnalysis on import or on demand).
ALTER TABLE player_match_analysis ADD COLUMN time_to_stop_ms_avg            REAL    NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN crouch_before_shot_count       INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN crouch_instead_of_strafe_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN flick_overshoot_avg_deg        REAL    NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN flick_undershoot_avg_deg       REAL    NOT NULL DEFAULT 0;
ALTER TABLE player_match_analysis ADD COLUMN flick_balance_pct              REAL    NOT NULL DEFAULT 0;
