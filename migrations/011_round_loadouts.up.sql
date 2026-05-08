-- Move per-tick player inventory into a per-round table.
--
-- tick_data.inventory was the largest TEXT column on the largest table
-- (~1.28M rows × ~30 B = 40 MB/demo, plus per-row index/page overhead).
-- The team bars only render the freeze-end loadout, so the per-tick
-- granularity was wasted: ~250 (round_id, steam_id) pairs cover the same
-- UI with ~5000× fewer rows.
--
-- New table holds the encoded comma-separated weapon list captured at
-- freeze-end (parser.go captureFreezeEnd). Cascades on round delete so
-- re-importing a demo wipes its loadouts via the existing rounds cascade.
CREATE TABLE round_loadouts (
    round_id  INTEGER NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
    steam_id  TEXT    NOT NULL,
    inventory TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (round_id, steam_id)
);

ALTER TABLE tick_data DROP COLUMN inventory;
