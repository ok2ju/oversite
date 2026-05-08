-- Restores tick_data.inventory but cannot recover the per-tick history that
-- was discarded when migration 011 ran. New ingests after rollback will
-- repopulate it from the parser.
ALTER TABLE tick_data ADD COLUMN inventory TEXT NOT NULL DEFAULT '';

DROP TABLE round_loadouts;
