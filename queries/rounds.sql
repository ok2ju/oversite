-- name: CreateRound :one
INSERT INTO rounds (demo_id, round_number, start_tick, end_tick, winner_side, win_reason, ct_score, t_score)
VALUES (@demo_id, @round_number, @start_tick, @end_tick, @winner_side, @win_reason, @ct_score, @t_score)
RETURNING *;

-- name: GetRoundsByDemoID :many
SELECT * FROM rounds WHERE demo_id = @demo_id ORDER BY round_number;

-- name: GetRoundByDemoAndNumber :one
SELECT * FROM rounds WHERE demo_id = @demo_id AND round_number = @round_number;

-- name: DeleteRoundsByDemoID :exec
DELETE FROM rounds WHERE demo_id = @demo_id;
