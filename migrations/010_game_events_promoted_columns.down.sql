-- Drops the promoted columns; loses any data written to them after the
-- promotion since the JSON blob no longer carries those keys.
ALTER TABLE game_events DROP COLUMN headshot;
ALTER TABLE game_events DROP COLUMN assister_steam_id;
ALTER TABLE game_events DROP COLUMN health_damage;
ALTER TABLE game_events DROP COLUMN attacker_name;
ALTER TABLE game_events DROP COLUMN victim_name;
ALTER TABLE game_events DROP COLUMN attacker_team;
ALTER TABLE game_events DROP COLUMN victim_team;
