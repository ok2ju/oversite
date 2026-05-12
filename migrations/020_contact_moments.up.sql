-- Phase 2 of the timeline contact-moments feature
-- (.claude/plans/timeline-contact-moments/). One row per (player,
-- signal-cluster) -- the atomic timeline marker for player-mode in the
-- 2D viewer. Built post-import by internal/demo/contacts/builder.go
-- from in-memory game_events + player_visibility (migration 019).
--
-- builder_version gates rebuilds when the grouping/outcome logic
-- changes: on demo open, if MAX(builder_version) for the demo is below
-- the compiled builder's version, the orchestrator wipes and rebuilds
-- contacts for that demo. The default of 1 is the first shipping
-- version.

CREATE TABLE contact_moments (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    demo_id         INTEGER NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    round_id        INTEGER NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
    subject_steam   TEXT    NOT NULL,
    t_first         INTEGER NOT NULL,
    t_last          INTEGER NOT NULL,
    t_pre           INTEGER NOT NULL,
    t_post          INTEGER NOT NULL,
    enemies_json    TEXT    NOT NULL,                -- JSON array of enemy steam_ids, ordered by first signal
    outcome         TEXT    NOT NULL,                -- see internal/demo/contacts.ContactOutcome
    signal_count    INTEGER NOT NULL,
    extras_json     TEXT    NOT NULL DEFAULT '{}',   -- truncated_round_end, flash_only, teammate_flashed_during, wallbang_taken
    builder_version INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE (demo_id, subject_steam, t_first)
);

CREATE INDEX idx_cm_demo_subject
    ON contact_moments(demo_id, subject_steam, t_first);
CREATE INDEX idx_cm_round_subject
    ON contact_moments(round_id, subject_steam, t_first);
