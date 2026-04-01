-- name: CreateStrategyBoard :one
INSERT INTO strategy_boards (user_id, title, map_name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetStrategyBoardByID :one
SELECT * FROM strategy_boards WHERE id = $1;

-- name: ListStrategyBoardsByUserID :many
SELECT * FROM strategy_boards WHERE user_id = $1 ORDER BY updated_at DESC;

-- name: UpdateStrategyBoard :one
UPDATE strategy_boards SET
    title = $2,
    map_name = $3,
    share_mode = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateStrategyBoardYjsState :exec
UPDATE strategy_boards SET yjs_state = $2, updated_at = NOW() WHERE id = $1;

-- name: GetStrategyBoardByShareToken :one
SELECT * FROM strategy_boards WHERE share_token = $1;

-- name: SetStrategyBoardShareToken :one
UPDATE strategy_boards SET share_token = $2, updated_at = NOW() WHERE id = $1
RETURNING *;

-- name: DeleteStrategyBoard :exec
DELETE FROM strategy_boards WHERE id = $1;
