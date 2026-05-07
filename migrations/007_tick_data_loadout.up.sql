-- Per-tick player economy and inventory used by the in-game team bars.
-- Older demos imported before this migration will read defaults until they
-- are re-imported.
ALTER TABLE tick_data ADD COLUMN money       INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tick_data ADD COLUMN has_helmet  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tick_data ADD COLUMN has_defuser INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tick_data ADD COLUMN inventory   TEXT    NOT NULL DEFAULT '';
