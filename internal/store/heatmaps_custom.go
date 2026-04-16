package store

// Manual implementations for heatmap queries that use json_each() with dynamic
// parameters. sqlc's SQLite engine cannot bind parameters inside table-valued
// function calls, so these are written by hand.

import (
	"context"
)

const getDemosByIDs = `
SELECT d.id, d.user_id, d.map_name
FROM demos d
INNER JOIN json_each(?) AS je ON d.id = je.value
`

type GetDemosByIDsRow struct {
	ID      int64
	UserID  int64
	MapName string
}

func (q *Queries) GetDemosByIDs(ctx context.Context, demoIDs string) ([]GetDemosByIDsRow, error) {
	rows, err := q.db.QueryContext(ctx, getDemosByIDs, demoIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	var items []GetDemosByIDsRow
	for rows.Next() {
		var i GetDemosByIDsRow
		if err := rows.Scan(&i.ID, &i.UserID, &i.MapName); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getHeatmapAggregation = `
SELECT ge.x, ge.y, count(*) AS kill_count
FROM game_events ge
INNER JOIN json_each(?) AS jd ON ge.demo_id = jd.value
LEFT JOIN player_rounds pr
    ON pr.round_id = ge.round_id AND pr.steam_id = ge.attacker_steam_id
LEFT JOIN json_each(?) AS jw ON ge.weapon = jw.value
WHERE ge.event_type = 'kill'
  AND ge.x IS NOT NULL
  AND ge.y IS NOT NULL
  AND (? IS NULL OR ge.attacker_steam_id = ?)
  AND (? IS NULL OR (pr.team_side IS NOT NULL AND pr.team_side = ?))
  AND (json_array_length(?) = 0 OR jw.value IS NOT NULL)
GROUP BY ge.x, ge.y
`

type GetHeatmapAggregationParams struct {
	DemoIDs       string
	Weapons       string
	PlayerSteamID *string
	Side          *string
}

type GetHeatmapAggregationRow struct {
	X         float64
	Y         float64
	KillCount int64
}

func (q *Queries) GetHeatmapAggregation(ctx context.Context, arg GetHeatmapAggregationParams) ([]GetHeatmapAggregationRow, error) {
	rows, err := q.db.QueryContext(ctx, getHeatmapAggregation,
		arg.DemoIDs,
		arg.Weapons,
		arg.PlayerSteamID, arg.PlayerSteamID,
		arg.Side, arg.Side,
		arg.Weapons,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	var items []GetHeatmapAggregationRow
	for rows.Next() {
		var i GetHeatmapAggregationRow
		if err := rows.Scan(&i.X, &i.Y, &i.KillCount); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getDistinctWeapons = `
SELECT DISTINCT ge.weapon
FROM game_events ge
INNER JOIN json_each(?) AS jd ON ge.demo_id = jd.value
WHERE ge.event_type = 'kill'
  AND ge.weapon IS NOT NULL
  AND ge.weapon != ''
ORDER BY ge.weapon
`

func (q *Queries) GetDistinctWeapons(ctx context.Context, demoIDs string) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, getDistinctWeapons, demoIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	var items []string
	for rows.Next() {
		var weapon string
		if err := rows.Scan(&weapon); err != nil {
			return nil, err
		}
		items = append(items, weapon)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

type GetDistinctPlayersRow struct {
	SteamID    string
	PlayerName string
}

const getDistinctPlayers = `
SELECT DISTINCT ge.attacker_steam_id, pr.player_name
FROM game_events ge
INNER JOIN json_each(?) AS jd ON ge.demo_id = jd.value
INNER JOIN player_rounds pr
    ON pr.round_id = ge.round_id AND pr.steam_id = ge.attacker_steam_id
WHERE ge.event_type = 'kill'
  AND ge.attacker_steam_id IS NOT NULL
ORDER BY pr.player_name
`

func (q *Queries) GetDistinctPlayers(ctx context.Context, demoIDs string) ([]GetDistinctPlayersRow, error) {
	rows, err := q.db.QueryContext(ctx, getDistinctPlayers, demoIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	var items []GetDistinctPlayersRow
	for rows.Next() {
		var i GetDistinctPlayersRow
		if err := rows.Scan(&i.SteamID, &i.PlayerName); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

type GetWeaponStatsByDemoIDRow struct {
	Weapon    string
	KillCount int64
	HSCount   int64
}

const getWeaponStatsByDemoID = `
SELECT ge.weapon, COUNT(*) AS kill_count,
       SUM(CASE WHEN json_extract(ge.extra_data, '$.headshot') = 1 THEN 1 ELSE 0 END) AS hs_count
FROM game_events ge
WHERE ge.demo_id = ?
  AND ge.event_type = 'kill'
  AND ge.weapon IS NOT NULL
  AND ge.weapon != ''
GROUP BY ge.weapon
ORDER BY kill_count DESC
`

func (q *Queries) GetWeaponStatsByDemoID(ctx context.Context, demoID int64) ([]GetWeaponStatsByDemoIDRow, error) {
	rows, err := q.db.QueryContext(ctx, getWeaponStatsByDemoID, demoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	var items []GetWeaponStatsByDemoIDRow
	for rows.Next() {
		var i GetWeaponStatsByDemoIDRow
		if err := rows.Scan(&i.Weapon, &i.KillCount, &i.HSCount); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
