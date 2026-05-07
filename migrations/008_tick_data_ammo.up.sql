-- Active-weapon ammo for the in-game HUD overlay (text labels under each
-- player and in the team bars). Older demos imported before this migration
-- read 0/0 until they are re-imported.
ALTER TABLE tick_data ADD COLUMN ammo_clip    INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tick_data ADD COLUMN ammo_reserve INTEGER NOT NULL DEFAULT 0;
