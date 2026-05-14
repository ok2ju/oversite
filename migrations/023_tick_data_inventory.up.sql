-- Re-introduce per-tick inventory on tick_data.
--
-- Migration 011 moved inventory to round_loadouts on the assumption that the
-- team-bars only needed the freeze-end loadout. That assumption no longer
-- holds: throws, drops, and pickups during a round must reflect on the
-- player cards, so inventory has to be sampled at the same cadence as the
-- rest of the tick state (every tickInterval ticks; ~16 Hz at the 4-tick
-- default).
--
-- round_loadouts stays in place — it's still the source for freeze-end
-- equip-value calculations (player_stats.sumLoadoutValue) and the
-- knife-round detection in parser.captureFreezeEnd.
--
-- Pre-migration tick rows backfill to '' and the team-bars fall back to the
-- round-scoped loadout for those demos.

ALTER TABLE tick_data ADD COLUMN inventory TEXT NOT NULL DEFAULT '';
