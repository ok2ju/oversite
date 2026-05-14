-- Add survival + economy + KAST columns to player_rounds so the match-overview
-- page can render real per-round damage, equipment value, and HLTV 2.0 rating
-- (which depends on KAST). Pre-migration demos persist these columns as zero
-- until RecomputeMatchOverview re-runs the parser pass for them.

ALTER TABLE player_rounds ADD COLUMN survived    INTEGER NOT NULL DEFAULT 0; -- 1 = alive at round_end_tick
ALTER TABLE player_rounds ADD COLUMN equip_value INTEGER NOT NULL DEFAULT 0; -- $ value at freeze_end_tick
ALTER TABLE player_rounds ADD COLUMN money_spent INTEGER NOT NULL DEFAULT 0; -- money_at_round_start - money_at_freeze_end
ALTER TABLE player_rounds ADD COLUMN kast_round  INTEGER NOT NULL DEFAULT 0; -- 1 if K, A, S, or traded this round
