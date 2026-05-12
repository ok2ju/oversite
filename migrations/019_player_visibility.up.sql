-- Slice 11: capture server-side visibility transitions (PlayerSpottersChanged).
-- Foundation for the timeline "contact moments" feature
-- (.claude/plans/timeline-contact-moments/). Stored as transition rows
-- (state=1 spotted_on, state=0 spotted_off) per (spotted, spotter) pair,
-- already debounced (4-tick window) by the parser. Volume target is well
-- under 50k rows per demo on typical MM/Faceit input; above that the
-- builder falls back to run-length-window storage (deferred — see
-- ../phase-1-visibility-capture.md "Pre-merge spike").

CREATE TABLE player_visibility (
    demo_id        INTEGER NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    round_id       INTEGER NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
    tick           INTEGER NOT NULL,
    spotted_steam  TEXT NOT NULL,   -- player who is visible
    spotter_steam  TEXT NOT NULL,   -- player who can see them
    state          INTEGER NOT NULL, -- 1 = on, 0 = off
    PRIMARY KEY (demo_id, tick, spotted_steam, spotter_steam)
);

CREATE INDEX idx_pv_round_pair
    ON player_visibility(round_id, spotted_steam, spotter_steam, tick);
