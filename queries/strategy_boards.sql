-- name: CreateStrategyBoard :one
INSERT INTO strategy_boards (title, map_name)
VALUES (@title, @map_name)
RETURNING *;

-- name: GetStrategyBoardByID :one
SELECT * FROM strategy_boards WHERE id = @id;

-- name: ListStrategyBoards :many
SELECT * FROM strategy_boards ORDER BY updated_at DESC;

-- name: UpdateStrategyBoard :one
UPDATE strategy_boards SET
    title = @title,
    map_name = @map_name,
    updated_at = datetime('now')
WHERE id = @id
RETURNING *;

-- name: UpdateBoardState :one
UPDATE strategy_boards SET
    board_state = @board_state,
    updated_at = datetime('now')
WHERE id = @id
RETURNING *;

-- name: DeleteStrategyBoard :exec
DELETE FROM strategy_boards WHERE id = @id;
