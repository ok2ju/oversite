-- Phase 2 of the timeline contact-moments feature. One row per
-- detector finding on a contact, surfaced in the tooltip on hover.
-- Phase 2 ships the schema empty -- Phase 3 detectors populate it.
--
-- detector_version is the analogue of contact_moments.builder_version
-- but scoped to detector-only rebuilds: a new detector or a tuning
-- change bumps the version, and the Phase 3 orchestrator wipes +
-- rewrites this table for the demo without touching contact_moments.
-- Default 0 means "no detectors have run" (the Phase 2 ingest state).

CREATE TABLE contact_mistakes (
    contact_id       INTEGER NOT NULL REFERENCES contact_moments(id) ON DELETE CASCADE,
    kind             TEXT    NOT NULL,                  -- e.g. "slow_reaction"
    category         TEXT    NOT NULL,                  -- "aim" | "movement" | "positioning" | "utility" | "trade"
    severity         INTEGER NOT NULL,                  -- 0=info, 1=low, 2=medium, 3=high
    phase            TEXT    NOT NULL,                  -- "pre" | "during" | "post"
    tick             INTEGER,                           -- nullable: point-in-time for the mistake if applicable
    extras_json      TEXT    NOT NULL DEFAULT '{}',
    detector_version INTEGER NOT NULL DEFAULT 0
);

-- Unique constraint expressed as a partial-aware unique index because
-- SQLite forbids expressions in PRIMARY KEY / UNIQUE constraints. The
-- COALESCE collapses NULL ticks into a sentinel so two rows with the
-- same (contact_id, kind) and both NULL ticks still conflict.
CREATE UNIQUE INDEX idx_cmis_pk ON contact_mistakes(contact_id, kind, COALESCE(tick, -1));

CREATE INDEX idx_cmis_contact ON contact_mistakes(contact_id);
CREATE INDEX idx_cmis_kind    ON contact_mistakes(kind);
