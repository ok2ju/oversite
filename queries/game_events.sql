-- name: CreateGameEvent :exec
INSERT INTO game_events (
    demo_id, round_id, tick, event_type,
    attacker_steam_id, victim_steam_id, weapon, x, y, z,
    headshot, assister_steam_id, health_damage,
    attacker_name, victim_name, attacker_team, victim_team,
    extra_data
) VALUES (
    @demo_id, @round_id, @tick, @event_type,
    @attacker_steam_id, @victim_steam_id, @weapon, @x, @y, @z,
    @headshot, @assister_steam_id, @health_damage,
    @attacker_name, @victim_name, @attacker_team, @victim_team,
    @extra_data
);

-- name: GetGameEventsByDemoID :many
SELECT * FROM game_events WHERE demo_id = @demo_id ORDER BY tick;

-- name: GetGameEventsByDemoAndRound :many
SELECT * FROM game_events WHERE demo_id = @demo_id AND round_id = @round_id ORDER BY tick;

-- name: GetGameEventsByType :many
SELECT * FROM game_events WHERE demo_id = @demo_id AND event_type = @event_type ORDER BY tick;

-- GetGameEventsByTypes is implemented manually in
-- internal/store/game_events_custom.go because sqlc's SQLite engine cannot
-- bind parameters inside json_each() table-valued function calls. See
-- internal/store/heatmaps_custom.go for the same pattern. Callers pass a
-- JSON array of event_type strings so unused rows are never loaded and
-- their extra_data JSON is never decoded.

-- name: DeleteGameEventsByDemoID :exec
DELETE FROM game_events WHERE demo_id = @demo_id;
