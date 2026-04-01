-- name: InsertTickData :exec
INSERT INTO tick_data (time, demo_id, tick, steam_id, x, y, z, yaw, health, armor, is_alive, weapon)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: GetTickDataByRange :many
SELECT * FROM tick_data
WHERE demo_id = $1 AND tick >= $2 AND tick <= $3
ORDER BY tick, steam_id;

-- name: GetTickDataByRangeAndPlayers :many
SELECT * FROM tick_data
WHERE demo_id = $1 AND tick >= $2 AND tick <= $3 AND steam_id = ANY($4::varchar[])
ORDER BY tick, steam_id;

-- name: DeleteTickDataByDemoID :exec
DELETE FROM tick_data WHERE demo_id = $1;
