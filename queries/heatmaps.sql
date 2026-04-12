-- name: GetDemosByIDs :many
SELECT id, user_id, map_name FROM demos
WHERE id IN (SELECT value FROM json_each(@demo_ids));

-- name: GetHeatmapAggregation :many
SELECT ge.x, ge.y, count(*) AS kill_count
FROM game_events ge
LEFT JOIN player_rounds pr
    ON pr.round_id = ge.round_id AND pr.steam_id = ge.attacker_steam_id
WHERE ge.demo_id IN (SELECT value FROM json_each(@demo_ids))
  AND ge.event_type = 'kill'
  AND ge.x IS NOT NULL
  AND ge.y IS NOT NULL
  AND (sqlc.narg('player_steam_id') IS NULL OR ge.attacker_steam_id = sqlc.narg('player_steam_id'))
  AND (sqlc.narg('side') IS NULL OR (pr.team_side IS NOT NULL AND pr.team_side = sqlc.narg('side')))
  AND (json_array_length(@weapons) = 0 OR ge.weapon IN (SELECT value FROM json_each(@weapons)))
GROUP BY ge.x, ge.y;
