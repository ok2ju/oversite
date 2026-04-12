-- name: CreateGameEvent :one
INSERT INTO game_events (demo_id, round_id, tick, event_type, attacker_steam_id, victim_steam_id, weapon, x, y, z, extra_data)
VALUES (@demo_id, @round_id, @tick, @event_type, @attacker_steam_id, @victim_steam_id, @weapon, @x, @y, @z, @extra_data)
RETURNING *;

-- name: GetGameEventsByDemoID :many
SELECT * FROM game_events WHERE demo_id = @demo_id ORDER BY tick;

-- name: GetGameEventsByDemoAndRound :many
SELECT * FROM game_events WHERE demo_id = @demo_id AND round_id = @round_id ORDER BY tick;

-- name: GetGameEventsByType :many
SELECT * FROM game_events WHERE demo_id = @demo_id AND event_type = @event_type ORDER BY tick;

-- name: DeleteGameEventsByDemoID :exec
DELETE FROM game_events WHERE demo_id = @demo_id;
