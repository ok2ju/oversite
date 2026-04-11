-- name: GetDemosByIDs :many
SELECT id, user_id, map_name FROM demos
WHERE id = ANY(@demo_ids::uuid[]);

-- name: GetHeatmapAggregation :many
SELECT ge.x, ge.y, count(*)::bigint AS kill_count
FROM game_events ge
LEFT JOIN player_rounds pr
    ON pr.round_id = ge.round_id AND pr.steam_id = ge.attacker_steam_id
WHERE ge.demo_id = ANY(@demo_ids::uuid[])
  AND ge.event_type = 'kill'
  AND ge.x IS NOT NULL
  AND ge.y IS NOT NULL
  AND (sqlc.narg('player_steam_id')::text IS NULL OR ge.attacker_steam_id = sqlc.narg('player_steam_id'))
  AND (sqlc.narg('side')::text IS NULL OR (pr.team_side IS NOT NULL AND pr.team_side = sqlc.narg('side')))
  AND (cardinality(@weapons::text[]) = 0 OR ge.weapon = ANY(@weapons::text[]))
GROUP BY ge.x, ge.y;
