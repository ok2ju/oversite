-- name: CreateDemo :one
INSERT INTO demos (user_id, faceit_match_id, map_name, file_path, file_size, status, match_date)
VALUES (@user_id, @faceit_match_id, @map_name, @file_path, @file_size, @status, @match_date)
RETURNING *;

-- name: GetDemoByID :one
SELECT * FROM demos WHERE id = @id;

-- name: ListDemosByUserID :many
SELECT * FROM demos
WHERE user_id = @user_id
ORDER BY created_at DESC
LIMIT @limit_val OFFSET @offset_val;

-- name: UpdateDemoStatus :one
UPDATE demos SET status = @status WHERE id = @id
RETURNING *;

-- name: UpdateDemoAfterParse :one
UPDATE demos SET
    status = 'ready',
    map_name = @map_name,
    total_ticks = @total_ticks,
    tick_rate = @tick_rate,
    duration_secs = @duration_secs
WHERE id = @id
RETURNING *;

-- name: DeleteDemo :exec
DELETE FROM demos WHERE id = @id;

-- name: CountDemosByUserID :one
SELECT count(*) FROM demos WHERE user_id = @user_id;
