-- name: CreateDemo :one
INSERT INTO demos (user_id, faceit_match_id, map_name, file_path, file_size, status, match_date)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetDemoByID :one
SELECT * FROM demos WHERE id = $1;

-- name: ListDemosByUserID :many
SELECT * FROM demos
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateDemoStatus :one
UPDATE demos SET status = $2 WHERE id = $1
RETURNING *;

-- name: UpdateDemoAfterParse :one
UPDATE demos SET
    status = 'ready',
    total_ticks = $2,
    tick_rate = $3,
    duration_secs = $4
WHERE id = $1
RETURNING *;

-- name: DeleteDemo :exec
DELETE FROM demos WHERE id = $1;

-- name: CountDemosByUserID :one
SELECT count(*) FROM demos WHERE user_id = $1;
