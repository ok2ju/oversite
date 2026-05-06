-- name: CreateDemo :one
INSERT INTO demos (map_name, file_path, file_size, status, match_date)
VALUES (@map_name, @file_path, @file_size, @status, @match_date)
RETURNING *;

-- name: GetDemoByID :one
SELECT * FROM demos WHERE id = @id;

-- name: ListDemos :many
SELECT * FROM demos
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

-- name: CountDemos :one
SELECT count(*) FROM demos;
