-- name: InsertContactMoment :one
INSERT INTO contact_moments (
    demo_id,
    round_id,
    subject_steam,
    t_first,
    t_last,
    t_pre,
    t_post,
    enemies_json,
    outcome,
    signal_count,
    extras_json,
    builder_version
) VALUES (
    @demo_id, @round_id, @subject_steam,
    @t_first, @t_last, @t_pre, @t_post,
    @enemies_json, @outcome, @signal_count,
    @extras_json, @builder_version
)
RETURNING id;

-- name: DeleteContactMomentsByDemoID :exec
DELETE FROM contact_moments WHERE demo_id = @demo_id;

-- name: ListContactsByDemoSubject :many
-- Phase 4 read path: every contact for (demo, subject), ordered chronologically.
SELECT id, demo_id, round_id, subject_steam,
       t_first, t_last, t_pre, t_post,
       enemies_json, outcome, signal_count,
       extras_json, builder_version, created_at
FROM contact_moments
WHERE demo_id = @demo_id
  AND subject_steam = @subject_steam
ORDER BY t_first ASC, id ASC;

-- name: ListContactsByDemoRoundSubject :many
-- Phase 3/4 read path: every contact in one round for one subject.
SELECT id, demo_id, round_id, subject_steam,
       t_first, t_last, t_pre, t_post,
       enemies_json, outcome, signal_count,
       extras_json, builder_version, created_at
FROM contact_moments
WHERE demo_id = @demo_id
  AND round_id = @round_id
  AND subject_steam = @subject_steam
ORDER BY t_first ASC, id ASC;

-- name: MaxBuilderVersionForDemo :one
-- Phase 3 rebuild gate: if MAX < compiled builder_version, rebuild for demo.
SELECT COALESCE(MAX(builder_version), 0) AS version
FROM contact_moments
WHERE demo_id = @demo_id;
