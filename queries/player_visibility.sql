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

-- name: ListVisibilityForDemo :many
-- Phase 2 contact builder: one demo-scoped fetch, then in-Go partition
-- by round_id. Cheaper than round_id-by-round_id queries for the
-- per-(player, round) builder loop.
SELECT demo_id, round_id, tick, spotted_steam, spotter_steam, state
FROM player_visibility
WHERE demo_id = @demo_id
ORDER BY round_id ASC, tick ASC, spotted_steam ASC, spotter_steam ASC;
