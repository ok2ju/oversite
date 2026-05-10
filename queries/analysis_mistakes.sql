-- name: CreateAnalysisMistake :exec
INSERT INTO analysis_mistakes (demo_id, steam_id, round_number, tick, kind, extras_json)
VALUES (@demo_id, @steam_id, @round_number, @tick, @kind, @extras_json);

-- name: DeleteAnalysisMistakesByDemoID :exec
DELETE FROM analysis_mistakes
WHERE demo_id = @demo_id;

-- name: ListAnalysisMistakesByDemoIDAndSteamID :many
-- Returns every mistake row for (demo, player), ordered chronologically so the
-- viewer side panel can render them as-is.
SELECT id, demo_id, steam_id, round_number, tick, kind, extras_json, created_at
FROM analysis_mistakes
WHERE demo_id = @demo_id
  AND steam_id = @steam_id
ORDER BY tick ASC, id ASC;
