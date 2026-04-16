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
