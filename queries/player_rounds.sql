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

-- name: GetPlayerStatsByDemoID :many
SELECT pr.steam_id, pr.player_name, pr.team_side,
       SUM(pr.kills) as total_kills, SUM(pr.deaths) as total_deaths,
       SUM(pr.assists) as total_assists, SUM(pr.damage) as total_damage,
       SUM(pr.headshot_kills) as total_headshot_kills, COUNT(*) as rounds_played
FROM player_rounds pr
JOIN rounds r ON pr.round_id = r.id
WHERE r.demo_id = @demo_id
GROUP BY pr.steam_id ORDER BY pr.team_side, total_kills DESC;
