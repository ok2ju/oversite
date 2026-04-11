-- name: CreateFaceitMatch :one
INSERT INTO faceit_matches (user_id, faceit_match_id, map_name, score_team, score_opponent, result, elo_before, elo_after, kills, deaths, assists, demo_url, demo_id, played_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: GetFaceitMatchesByUserID :many
SELECT * FROM faceit_matches
WHERE user_id = $1
ORDER BY played_at DESC
LIMIT $2 OFFSET $3;

-- name: GetFaceitMatchByID :one
SELECT * FROM faceit_matches WHERE id = $1;

-- name: LinkFaceitMatchToDemo :one
UPDATE faceit_matches SET demo_id = $2 WHERE id = $1
RETURNING *;

-- name: GetEloHistory :many
SELECT id, faceit_match_id, map_name, elo_after, played_at
FROM faceit_matches
WHERE user_id = $1 AND played_at >= $2
ORDER BY played_at ASC;

-- name: GetExistingFaceitMatchIDs :many
SELECT faceit_match_id FROM faceit_matches WHERE user_id = $1;

-- name: UpsertFaceitMatch :one
INSERT INTO faceit_matches (user_id, faceit_match_id, map_name, score_team, score_opponent, result, elo_before, elo_after, kills, deaths, assists, demo_url, demo_id, played_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (user_id, faceit_match_id) DO NOTHING
RETURNING *;

-- name: CountFaceitMatchesByUserID :one
SELECT COUNT(*) FROM faceit_matches WHERE user_id = $1;

-- name: GetCurrentStreak :many
SELECT result FROM faceit_matches
WHERE user_id = $1
ORDER BY played_at DESC
LIMIT 30;

-- name: DeleteFaceitMatchesByUserID :exec
DELETE FROM faceit_matches WHERE user_id = $1;

-- name: CountFaceitMatchesFiltered :one
SELECT COUNT(*) FROM faceit_matches
WHERE user_id = $1
  AND (sqlc.narg('map_name')::varchar IS NULL OR map_name = sqlc.narg('map_name'))
  AND (sqlc.narg('result')::varchar IS NULL OR result = sqlc.narg('result'));

-- name: GetFaceitMatchesFiltered :many
SELECT * FROM faceit_matches
WHERE user_id = $1
  AND (sqlc.narg('map_name')::varchar IS NULL OR map_name = sqlc.narg('map_name'))
  AND (sqlc.narg('result')::varchar IS NULL OR result = sqlc.narg('result'))
ORDER BY played_at DESC
LIMIT $2 OFFSET $3;
