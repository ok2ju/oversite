-- name: CreatePlayerRound :one
INSERT INTO player_rounds (round_id, steam_id, player_name, team_side, kills, deaths, assists, damage, headshot_kills, first_kill, first_death, clutch_kills)
VALUES (@round_id, @steam_id, @player_name, @team_side, @kills, @deaths, @assists, @damage, @headshot_kills, @first_kill, @first_death, @clutch_kills)
RETURNING *;

-- name: GetPlayerRoundsByRoundID :many
SELECT * FROM player_rounds WHERE round_id = @round_id;

-- name: GetPlayerRoundsBySteamID :many
SELECT * FROM player_rounds WHERE steam_id = @steam_id
ORDER BY round_id;

-- name: DeletePlayerRoundsByRoundID :exec
DELETE FROM player_rounds WHERE round_id = @round_id;
