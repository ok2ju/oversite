-- name: InsertContactMistake :exec
INSERT OR REPLACE INTO contact_mistakes (
    contact_id,
    kind,
    category,
    severity,
    phase,
    tick,
    extras_json,
    detector_version
) VALUES (
    @contact_id, @kind, @category, @severity,
    @phase, @tick, @extras_json, @detector_version
);

-- name: DeleteContactMistakesByDemoID :exec
DELETE FROM contact_mistakes
WHERE contact_id IN (
    SELECT id FROM contact_moments WHERE demo_id = @demo_id
);

-- name: ListContactMistakesByContact :many
SELECT contact_id, kind, category, severity, phase, tick, extras_json, detector_version
FROM contact_mistakes
WHERE contact_id = @contact_id
ORDER BY phase ASC, severity DESC, tick ASC;

-- name: MaxDetectorVersionForDemo :one
-- Phase 3 rebuild gate: if MAX < compiled DetectorVersion, rewrite for demo.
SELECT COALESCE(MAX(detector_version), 0) AS version
FROM contact_mistakes
WHERE contact_id IN (SELECT id FROM contact_moments WHERE demo_id = @demo_id);
