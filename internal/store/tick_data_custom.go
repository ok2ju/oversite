package store

// Manual tick_data queries. sqlc generates GetTickDataByRangeAndPlayers but
// drops the `@steam_ids` json_each binding (the SQLite engine cannot bind a
// parameter inside a table-valued function call). The single-player variant
// for player-stats aggregation is straightforward without json_each so we
// keep it here next to its sibling custom queries.

import (
	"context"
)

const getTickDataByDemoAndPlayer = `
SELECT demo_id, tick, steam_id, x, y, z, yaw, pitch, crouch, health, armor, is_alive, weapon, money, has_helmet, has_defuser, ammo_clip, ammo_reserve
FROM tick_data
WHERE demo_id = ? AND steam_id = ?
ORDER BY tick
`

// GetTickDataByDemoAndPlayer returns every sampled tick row for a single
// player in a demo, ordered by tick. Used by the player-stats aggregator to
// compute movement/timing breakdowns without fetching the full ~100K-row
// match payload.
func (q *Queries) GetTickDataByDemoAndPlayer(ctx context.Context, demoID int64, steamID string) ([]TickDatum, error) {
	rows, err := q.db.QueryContext(ctx, getTickDataByDemoAndPlayer, demoID, steamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	var items []TickDatum
	for rows.Next() {
		var i TickDatum
		if err := rows.Scan(
			&i.DemoID,
			&i.Tick,
			&i.SteamID,
			&i.X,
			&i.Y,
			&i.Z,
			&i.Yaw,
			&i.Health,
			&i.Armor,
			&i.IsAlive,
			&i.Weapon,
			&i.Money,
			&i.HasHelmet,
			&i.HasDefuser,
			&i.AmmoClip,
			&i.AmmoReserve,
		); err != nil {
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
