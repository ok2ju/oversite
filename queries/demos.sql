-- name: CreateDemo :one
INSERT INTO demos (map_name, file_path, file_size, status, match_date)
VALUES (@map_name, @file_path, @file_size, @status, @match_date)
RETURNING *;

-- name: GetDemoByID :one
SELECT * FROM demos WHERE id = @id;

-- name: ListDemos :many
SELECT
    d.*,
    CAST(COALESCE(s.ct_score, 0) AS INTEGER) AS final_ct_score,
    CAST(COALESCE(s.t_score, 0) AS INTEGER) AS final_t_score
FROM demos d
LEFT JOIN (
    SELECT demo_id, MAX(ct_score) AS ct_score, MAX(t_score) AS t_score
    FROM rounds
    GROUP BY demo_id
) s ON s.demo_id = d.id
ORDER BY d.created_at DESC
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
