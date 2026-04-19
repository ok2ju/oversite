-- Add ADR (average damage per round) column to Faceit matches.
ALTER TABLE faceit_matches ADD COLUMN adr REAL NOT NULL DEFAULT 0;
