-- name: InsertPlayerVisibility :exec
INSERT INTO player_visibility (
    demo_id,
    round_id,
    tick,
    spotted_steam,
    spotter_steam,
    state
) VALUES (?, ?, ?, ?, ?, ?);

-- name: DeleteVisibilityByDemoID :exec
DELETE FROM player_visibility WHERE demo_id = ?;

-- name: ListVisibilityForRound :many
SELECT *
FROM player_visibility
WHERE round_id = ?
ORDER BY tick ASC, spotted_steam ASC, spotter_steam ASC;
