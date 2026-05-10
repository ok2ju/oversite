-- P3-1: pitch (Player.ViewDirectionY) + crouch (Player.IsDucking) per tick.
-- Pitch powers the mouse-spiral viz and the over/under-flick classifier;
-- crouch powers the crouch_before_shot habit and cause-tag (P2-1).
--
-- Both are NOT NULL with neutral defaults so old demos imported before this
-- migration read 0 / false until they are re-imported. Analyzer rules that
-- depend on these columns gate on the new player_match_analysis micro
-- columns (slice 11 P3-2), not on a per-row "is unknown" flag — so a
-- demo missing the data simply produces zero counts.
ALTER TABLE tick_data ADD COLUMN pitch  REAL    NOT NULL DEFAULT 0;
ALTER TABLE tick_data ADD COLUMN crouch INTEGER NOT NULL DEFAULT 0;
