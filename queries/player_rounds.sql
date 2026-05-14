-- name: CreatePlayerRound :one
INSERT INTO player_rounds (round_id, steam_id, player_name, team_side, kills, deaths, assists, damage, headshot_kills, first_kill, first_death, clutch_kills, survived, equip_value, money_spent, kast_round)
VALUES (@round_id, @steam_id, @player_name, @team_side, @kills, @deaths, @assists, @damage, @headshot_kills, @first_kill, @first_death, @clutch_kills, @survived, @equip_value, @money_spent, @kast_round)
RETURNING *;

-- name: GetPlayerRoundsByRoundID :many
SELECT * FROM player_rounds WHERE round_id = @round_id ORDER BY steam_id;

-- name: GetRostersByDemoID :many
SELECT r.round_number, pr.steam_id, pr.player_name, pr.team_side
FROM player_rounds pr
JOIN rounds r ON pr.round_id = r.id
WHERE r.demo_id = @demo_id
ORDER BY r.round_number, pr.steam_id;

-- name: GetPlayerRoundsBySteamID :many
SELECT * FROM player_rounds WHERE steam_id = @steam_id
ORDER BY round_id;

-- name: DeletePlayerRoundsByRoundID :exec
DELETE FROM player_rounds WHERE round_id = @round_id;

-- name: GetPlayerStatsByDemoID :many
WITH ranked AS (
    SELECT pr.steam_id, pr.player_name, pr.kills, pr.deaths, pr.assists,
           pr.damage, pr.headshot_kills,
           FIRST_VALUE(pr.team_side) OVER (
               PARTITION BY pr.steam_id ORDER BY r.round_number
           ) AS first_team_side
    FROM player_rounds pr
    JOIN rounds r ON pr.round_id = r.id
    WHERE r.demo_id = @demo_id
)
SELECT steam_id, player_name,
       CAST(first_team_side AS TEXT) as team_side,
       SUM(kills) as total_kills, SUM(deaths) as total_deaths,
       SUM(assists) as total_assists, SUM(damage) as total_damage,
       SUM(headshot_kills) as total_headshot_kills, COUNT(*) as rounds_played
FROM ranked
GROUP BY steam_id, player_name, first_team_side
ORDER BY team_side, total_kills DESC;

-- name: GetPlayerRoundsForOverview :many
SELECT r.round_number, r.is_overtime,
       pr.steam_id, pr.player_name, pr.team_side,
       pr.kills, pr.deaths, pr.assists, pr.damage, pr.headshot_kills,
       pr.first_kill, pr.first_death, pr.clutch_kills,
       pr.survived, pr.equip_value, pr.money_spent, pr.kast_round
FROM player_rounds pr
JOIN rounds r ON pr.round_id = r.id
WHERE r.demo_id = @demo_id
ORDER BY r.round_number, pr.steam_id;
