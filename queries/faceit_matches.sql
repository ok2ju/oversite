-- name: CreateFaceitMatch :one
INSERT INTO faceit_matches (user_id, faceit_match_id, map_name, score_team, score_opponent, result, elo_before, elo_after, kills, deaths, assists, demo_url, demo_id, played_at)
VALUES (@user_id, @faceit_match_id, @map_name, @score_team, @score_opponent, @result, @elo_before, @elo_after, @kills, @deaths, @assists, @demo_url, @demo_id, @played_at)
RETURNING *;

-- name: GetFaceitMatchesByUserID :many
SELECT * FROM faceit_matches
WHERE user_id = @user_id
ORDER BY played_at DESC
LIMIT @limit_val OFFSET @offset_val;

-- name: GetFaceitMatchByID :one
SELECT * FROM faceit_matches WHERE id = @id;

-- name: LinkFaceitMatchToDemo :one
UPDATE faceit_matches SET demo_id = @demo_id WHERE id = @id
RETURNING *;

-- name: GetEloHistory :many
SELECT id, faceit_match_id, map_name, elo_after, played_at
FROM faceit_matches
WHERE user_id = @user_id AND played_at >= @since
ORDER BY played_at ASC;

-- name: GetExistingFaceitMatchIDs :many
SELECT faceit_match_id FROM faceit_matches WHERE user_id = @user_id;

-- name: UpsertFaceitMatch :one
INSERT INTO faceit_matches (user_id, faceit_match_id, map_name, score_team, score_opponent, result, elo_before, elo_after, kills, deaths, assists, demo_url, demo_id, played_at)
VALUES (@user_id, @faceit_match_id, @map_name, @score_team, @score_opponent, @result, @elo_before, @elo_after, @kills, @deaths, @assists, @demo_url, @demo_id, @played_at)
ON CONFLICT (user_id, faceit_match_id) DO NOTHING
RETURNING *;

-- name: CountFaceitMatchesByUserID :one
SELECT COUNT(*) FROM faceit_matches WHERE user_id = @user_id;

-- name: GetCurrentStreak :many
SELECT result FROM faceit_matches
WHERE user_id = @user_id
ORDER BY played_at DESC
LIMIT 30;

-- name: DeleteFaceitMatchesByUserID :exec
DELETE FROM faceit_matches WHERE user_id = @user_id;

-- name: CountFaceitMatchesFiltered :one
SELECT COUNT(*) FROM faceit_matches
WHERE user_id = @user_id
  AND (sqlc.narg('map_name') IS NULL OR map_name = sqlc.narg('map_name'))
  AND (sqlc.narg('result') IS NULL OR result = sqlc.narg('result'));

-- name: GetFaceitMatchesFiltered :many
SELECT * FROM faceit_matches
WHERE user_id = @user_id
  AND (sqlc.narg('map_name') IS NULL OR map_name = sqlc.narg('map_name'))
  AND (sqlc.narg('result') IS NULL OR result = sqlc.narg('result'))
ORDER BY played_at DESC
LIMIT @limit_val OFFSET @offset_val;
