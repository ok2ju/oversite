-- name: CreatePlayerRound :one
INSERT INTO player_rounds (round_id, steam_id, player_name, team_side, kills, deaths, assists, damage, headshot_kills, first_kill, first_death, clutch_kills)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetPlayerRoundsByRoundID :many
SELECT * FROM player_rounds WHERE round_id = $1;

-- name: GetPlayerRoundsBySteamID :many
SELECT * FROM player_rounds WHERE steam_id = $1
ORDER BY round_id;

-- name: DeletePlayerRoundsByRoundID :exec
DELETE FROM player_rounds WHERE round_id = $1;
