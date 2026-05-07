-- Add per-round team clan names (m_szClanTeamname).
-- Captured from CCSTeam state at RoundEnd. Pro / FACEIT / ESEA demos populate
-- this; matchmaking demos leave it empty and the frontend falls back to "CT" /
-- "T" labels. Per-round (not per-demo) so mid-match team-name changes are
-- reflected at the round granularity.
ALTER TABLE rounds ADD COLUMN ct_team_name TEXT NOT NULL DEFAULT '';
ALTER TABLE rounds ADD COLUMN t_team_name  TEXT NOT NULL DEFAULT '';
