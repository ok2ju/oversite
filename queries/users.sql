-- name: CreateUser :one
INSERT INTO users (faceit_id, nickname, avatar_url, faceit_elo, faceit_level, country)
VALUES (@faceit_id, @nickname, @avatar_url, @faceit_elo, @faceit_level, @country)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = @id;

-- name: GetUserByFaceitID :one
SELECT * FROM users WHERE faceit_id = @faceit_id;

-- name: UpdateUser :one
UPDATE users SET
    nickname = @nickname,
    avatar_url = @avatar_url,
    faceit_elo = @faceit_elo,
    faceit_level = @faceit_level,
    country = @country,
    updated_at = datetime('now')
WHERE id = @id
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = @id;
