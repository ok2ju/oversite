-- Add freeze_end_tick column to rounds table.
-- Stores the in-game tick at which freeze time ends and the round goes live.
-- Defaults to 0 (unknown) for historical rounds parsed before this column existed.
ALTER TABLE rounds ADD COLUMN freeze_end_tick INTEGER NOT NULL DEFAULT 0;
