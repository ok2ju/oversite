-- name: CreateGrenadeLineup :one
INSERT INTO grenade_lineups (demo_id, tick, map_name, grenade_type, throw_x, throw_y, throw_z, throw_yaw, throw_pitch, land_x, land_y, land_z, title, description, tags, is_favorite)
VALUES (@demo_id, @tick, @map_name, @grenade_type, @throw_x, @throw_y, @throw_z, @throw_yaw, @throw_pitch, @land_x, @land_y, @land_z, @title, @description, @tags, @is_favorite)
RETURNING *;

-- name: GetGrenadeLineupByID :one
SELECT * FROM grenade_lineups WHERE id = @id;

-- name: ListGrenadeLineups :many
SELECT * FROM grenade_lineups
ORDER BY created_at DESC
LIMIT @limit_val OFFSET @offset_val;

-- name: ListGrenadeLineupsByMapAndType :many
SELECT * FROM grenade_lineups
WHERE map_name = @map_name AND grenade_type = @grenade_type
ORDER BY created_at DESC;

-- name: ToggleLineupFavorite :one
UPDATE grenade_lineups SET is_favorite = 1 - is_favorite WHERE id = @id
RETURNING *;

-- name: DeleteGrenadeLineup :exec
DELETE FROM grenade_lineups WHERE id = @id;
