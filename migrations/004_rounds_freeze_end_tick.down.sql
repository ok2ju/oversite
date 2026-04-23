-- Remove freeze_end_tick column from rounds.
-- SQLite supports DROP COLUMN since 3.35.
ALTER TABLE rounds DROP COLUMN freeze_end_tick;
