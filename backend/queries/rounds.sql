-- name: CreateRound :one
INSERT INTO rounds (demo_id, round_number, start_tick, end_tick, winner_side, win_reason, ct_score, t_score)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetRoundsByDemoID :many
SELECT * FROM rounds WHERE demo_id = $1 ORDER BY round_number;

-- name: GetRoundByDemoAndNumber :one
SELECT * FROM rounds WHERE demo_id = $1 AND round_number = $2;

-- name: DeleteRoundsByDemoID :exec
DELETE FROM rounds WHERE demo_id = $1;
