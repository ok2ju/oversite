-- name: CreateGrenadeLineup :one
INSERT INTO grenade_lineups (user_id, demo_id, tick, map_name, grenade_type, throw_x, throw_y, throw_z, throw_yaw, throw_pitch, land_x, land_y, land_z, title, description, tags, is_favorite)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
RETURNING *;

-- name: GetGrenadeLineupByID :one
SELECT * FROM grenade_lineups WHERE id = $1;

-- name: ListGrenadeLineupsByUserID :many
SELECT * FROM grenade_lineups
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListGrenadeLineupsByMapAndType :many
SELECT * FROM grenade_lineups
WHERE user_id = $1 AND map_name = $2 AND grenade_type = $3
ORDER BY created_at DESC;

-- name: ToggleLineupFavorite :one
UPDATE grenade_lineups SET is_favorite = NOT is_favorite WHERE id = $1
RETURNING *;

-- name: DeleteGrenadeLineup :exec
DELETE FROM grenade_lineups WHERE id = $1;
