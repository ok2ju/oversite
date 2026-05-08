-- Promote hot extra_data fields out of the JSON blob into real columns.
-- Eliminates the triple JSON serialization (SQLite TEXT → Go decode → Wails →
-- JS) for kill-log rendering and lets the weapon-stats query filter on a
-- column instead of paying for json_extract on every row.
--
-- Affected fields (kill / player_hurt only — other event types had no
-- promotable hot fields):
--   headshot          : kill (was bool in JSON, now 0/1 int column)
--   assister_steam_id : kill
--   attacker_name     : kill, player_hurt
--   victim_name       : kill, player_hurt
--   attacker_team     : kill, player_hurt
--   victim_team       : kill, player_hurt
--   health_damage     : player_hurt
--
-- Older demos parsed before this migration get backfilled below using
-- json_extract; new ingests write directly to the columns and omit these
-- fields from the JSON blob, halving extra_data size on kill rows.
ALTER TABLE game_events ADD COLUMN headshot          INTEGER NOT NULL DEFAULT 0;
ALTER TABLE game_events ADD COLUMN assister_steam_id TEXT;
ALTER TABLE game_events ADD COLUMN health_damage     INTEGER NOT NULL DEFAULT 0;
ALTER TABLE game_events ADD COLUMN attacker_name     TEXT    NOT NULL DEFAULT '';
ALTER TABLE game_events ADD COLUMN victim_name       TEXT    NOT NULL DEFAULT '';
ALTER TABLE game_events ADD COLUMN attacker_team     TEXT    NOT NULL DEFAULT '';
ALTER TABLE game_events ADD COLUMN victim_team       TEXT    NOT NULL DEFAULT '';

-- Backfill from existing extra_data so already-imported demos don't lose info.
-- json_extract returns NULL for missing keys; coalesce to the column default
-- so the NOT NULL constraint holds.
UPDATE game_events SET
    headshot      = CASE WHEN json_extract(extra_data, '$.headshot') IN (1, true) THEN 1 ELSE 0 END,
    attacker_name = COALESCE(json_extract(extra_data, '$.attacker_name'), ''),
    victim_name   = COALESCE(json_extract(extra_data, '$.victim_name'),   ''),
    attacker_team = COALESCE(json_extract(extra_data, '$.attacker_team'), ''),
    victim_team   = COALESCE(json_extract(extra_data, '$.victim_team'),   ''),
    assister_steam_id = json_extract(extra_data, '$.assister_steam_id')
WHERE event_type = 'kill';

UPDATE game_events SET
    health_damage = COALESCE(json_extract(extra_data, '$.health_damage'), 0),
    attacker_name = COALESCE(json_extract(extra_data, '$.attacker_name'), ''),
    victim_name   = COALESCE(json_extract(extra_data, '$.victim_name'),   ''),
    attacker_team = COALESCE(json_extract(extra_data, '$.attacker_team'), ''),
    victim_team   = COALESCE(json_extract(extra_data, '$.victim_team'),   '')
WHERE event_type = 'player_hurt';
