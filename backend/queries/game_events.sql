-- name: CreateGameEvent :one
INSERT INTO game_events (demo_id, round_id, tick, event_type, attacker_steam_id, victim_steam_id, weapon, x, y, z, extra_data)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetGameEventsByDemoID :many
SELECT * FROM game_events WHERE demo_id = $1 ORDER BY tick;

-- name: GetGameEventsByDemoAndRound :many
SELECT * FROM game_events WHERE demo_id = $1 AND round_id = $2 ORDER BY tick;

-- name: GetGameEventsByType :many
SELECT * FROM game_events WHERE demo_id = $1 AND event_type = $2 ORDER BY tick;

-- name: DeleteGameEventsByDemoID :exec
DELETE FROM game_events WHERE demo_id = $1;
