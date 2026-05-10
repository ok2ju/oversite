-- name: CreateAnalysisMistake :exec
INSERT INTO analysis_mistakes (demo_id, steam_id, round_number, round_id, tick, kind, category, severity, extras_json)
VALUES (@demo_id, @steam_id, @round_number, @round_id, @tick, @kind, @category, @severity, @extras_json);

-- name: DeleteAnalysisMistakesByDemoID :exec
DELETE FROM analysis_mistakes
WHERE demo_id = @demo_id;

-- name: ListAnalysisMistakesByDemoIDAndSteamID :many
-- Returns every mistake row for (demo, player), ordered chronologically so the
-- viewer side panel can render them as-is.
SELECT id, demo_id, steam_id, round_number, round_id, tick, kind, category, severity, extras_json, created_at
FROM analysis_mistakes
WHERE demo_id = @demo_id
  AND steam_id = @steam_id
ORDER BY tick ASC, id ASC;

-- name: GetAnalysisMistakeByID :one
SELECT id, demo_id, steam_id, round_number, round_id, tick, kind, category, severity, extras_json, created_at
FROM analysis_mistakes
WHERE id = @id;

-- name: CountAnalysisMistakesByCategory :many
-- Returns one row per (demo, category) with the mistake count. Used by the
-- match-insights summary so the team-level view can render category badges
-- without re-walking every row on the frontend.
SELECT category, count(*) AS total
FROM analysis_mistakes
WHERE demo_id = @demo_id
GROUP BY category
ORDER BY category ASC;

-- name: CountAnalysisMistakesByKindForPlayer :one
-- Counts mistakes of a specific kind for one (demo, player). Used by the
-- HabitReport builder to populate count-based habits (e.g. untraded deaths)
-- without round-tripping the full timeline.
SELECT count(*) AS total
FROM analysis_mistakes
WHERE demo_id = @demo_id
  AND steam_id = @steam_id
  AND kind = @kind;

