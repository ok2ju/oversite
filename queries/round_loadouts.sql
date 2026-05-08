-- name: CreateRoundLoadout :exec
INSERT INTO round_loadouts (round_id, steam_id, inventory)
VALUES (@round_id, @steam_id, @inventory);

-- name: GetRoundLoadoutsByDemoID :many
-- Returns every (round_number, steam_id, inventory) triple for a demo so the
-- viewer can preload all loadouts in a single Wails round-trip and hand the
-- team bars a per-round lookup. ~250 rows total per match.
SELECT r.round_number, rl.steam_id, rl.inventory
FROM round_loadouts rl
JOIN rounds r ON rl.round_id = r.id
WHERE r.demo_id = @demo_id
ORDER BY r.round_number, rl.steam_id;
