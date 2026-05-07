-- name: InsertTickData :exec
INSERT INTO tick_data (demo_id, tick, steam_id, x, y, z, yaw, health, armor, is_alive, weapon, money, has_helmet, has_defuser, inventory)
VALUES (@demo_id, @tick, @steam_id, @x, @y, @z, @yaw, @health, @armor, @is_alive, @weapon, @money, @has_helmet, @has_defuser, @inventory);

-- name: GetTickDataByRange :many
SELECT * FROM tick_data
WHERE demo_id = @demo_id AND tick >= @start_tick AND tick <= @end_tick
ORDER BY tick, steam_id;

-- name: GetTickDataByRangeAndPlayers :many
SELECT * FROM tick_data
WHERE demo_id = @demo_id AND tick >= @start_tick AND tick <= @end_tick
  AND steam_id IN (SELECT value FROM json_each(@steam_ids))
ORDER BY tick, steam_id;

-- name: DeleteTickDataByDemoID :exec
DELETE FROM tick_data WHERE demo_id = @demo_id;
