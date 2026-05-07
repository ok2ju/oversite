-- Remove ct_team_name / t_team_name columns from rounds.
-- SQLite supports DROP COLUMN since 3.35.
ALTER TABLE rounds DROP COLUMN ct_team_name;
ALTER TABLE rounds DROP COLUMN t_team_name;
