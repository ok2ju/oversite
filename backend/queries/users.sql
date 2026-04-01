-- name: CreateUser :one
INSERT INTO users (faceit_id, nickname, avatar_url, faceit_elo, faceit_level, country)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByFaceitID :one
SELECT * FROM users WHERE faceit_id = $1;

-- name: UpdateUser :one
UPDATE users SET
    nickname = $2,
    avatar_url = $3,
    faceit_elo = $4,
    faceit_level = $5,
    country = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
