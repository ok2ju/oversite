DROP INDEX IF EXISTS idx_analysis_mistakes_duel;
ALTER TABLE analysis_mistakes DROP COLUMN duel_id;

DROP INDEX IF EXISTS idx_analysis_duels_demo_victim;
DROP INDEX IF EXISTS idx_analysis_duels_demo_attacker;
DROP INDEX IF EXISTS idx_analysis_duels_demo_round;
DROP TABLE IF EXISTS analysis_duels;
