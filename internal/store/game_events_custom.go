package store

// Manual implementations for game_events queries that use json_each() with
// dynamic parameters. sqlc's SQLite engine cannot bind parameters inside
// table-valued function calls, so this is written by hand. See
// heatmaps_custom.go for the same pattern.

import (
	"context"
)

const getGameEventsByTypes = `
SELECT id, demo_id, round_id, tick, event_type, attacker_steam_id,
       victim_steam_id, weapon, x, y, z, extra_data,
       headshot, assister_steam_id, health_damage,
       attacker_name, victim_name, attacker_team, victim_team
FROM game_events
WHERE demo_id = ?
  AND event_type IN (SELECT value FROM json_each(?))
ORDER BY tick
`

// GetGameEventsByTypes returns game events for a demo filtered to the given
// event types. eventTypes is a JSON array string, e.g. `["kill","bomb_plant"]`.
// Callers should marshal a []string with encoding/json and pass the result.
//
// Used by the kill-log and event-layer paths that only render a small subset
// of event types — fetching the full GetGameEventsByDemoID result and then
// filtering client-side would force a JSON decode of every row's extra_data.
func (q *Queries) GetGameEventsByTypes(ctx context.Context, demoID int64, eventTypes string) ([]GameEvent, error) {
	rows, err := q.db.QueryContext(ctx, getGameEventsByTypes, demoID, eventTypes)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	var items []GameEvent
	for rows.Next() {
		var i GameEvent
		if err := rows.Scan(
			&i.ID,
			&i.DemoID,
			&i.RoundID,
			&i.Tick,
			&i.EventType,
			&i.AttackerSteamID,
			&i.VictimSteamID,
			&i.Weapon,
			&i.X,
			&i.Y,
			&i.Z,
			&i.ExtraData,
			&i.Headshot,
			&i.AssisterSteamID,
			&i.HealthDamage,
			&i.AttackerName,
			&i.VictimName,
			&i.AttackerTeam,
			&i.VictimTeam,
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
