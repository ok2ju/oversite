package demo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ok2ju/oversite/internal/store"
)

// IngestGameEvents deletes existing game events for the demo, then inserts
// all parsed events within a single transaction. It is idempotent.
// roundMap maps round numbers to DB round IDs (from IngestRounds).
// Returns the number of inserted events.
func IngestGameEvents(ctx context.Context, db *sql.DB, demoID int64, events []GameEvent, roundMap map[int]int64) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}

	slog.Info("starting game event ingestion", "demo_id", demoID, "event_count", len(events))

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := store.New(tx)

	if err := q.DeleteGameEventsByDemoID(ctx, demoID); err != nil {
		return 0, fmt.Errorf("delete existing game events: %w", err)
	}

	for _, evt := range events {
		params, err := toEventParams(demoID, evt, roundMap)
		if err != nil {
			return 0, fmt.Errorf("convert event (tick %d, type %s): %w", evt.Tick, evt.Type, err)
		}
		if err := q.CreateGameEvent(ctx, params); err != nil {
			return 0, fmt.Errorf("insert game event (tick %d, type %s): %w", evt.Tick, evt.Type, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	slog.Info("game event ingestion complete", "demo_id", demoID, "events_inserted", len(events))
	return len(events), nil
}

// toEventParams converts a parsed GameEvent to sqlc CreateGameEventParams.
//
// Promoted fields (headshot, assister_steam_id, health_damage, attacker/victim
// name+team) are pulled out of the typed extras and into top-level columns.
// The remaining cold fields stay in the JSON extra_data blob (their
// `json:"-"`-tagged siblings are skipped by encoding/json).
func toEventParams(demoID int64, evt GameEvent, roundMap map[int]int64) (store.CreateGameEventParams, error) {
	extra, err := marshalExtraData(evt.ExtraData)
	if err != nil {
		return store.CreateGameEventParams{}, fmt.Errorf("marshal extra data: %w", err)
	}

	params := store.CreateGameEventParams{
		DemoID:          demoID,
		RoundID:         resolveRoundID(evt.RoundNumber, roundMap),
		Tick:            int64(evt.Tick),
		EventType:       evt.Type,
		AttackerSteamID: nullString(evt.AttackerSteamID),
		VictimSteamID:   nullString(evt.VictimSteamID),
		Weapon:          nullString(evt.Weapon),
		X:               evt.X,
		Y:               evt.Y,
		Z:               evt.Z,
		ExtraData:       extra,
	}

	switch e := evt.ExtraData.(type) {
	case *KillExtra:
		if e != nil {
			params.Headshot = boolToInt64(e.Headshot)
			params.AssisterSteamID = nullString(e.AssisterSteamID)
			params.AttackerName = e.AttackerName
			params.VictimName = e.VictimName
			params.AttackerTeam = e.AttackerTeam
			params.VictimTeam = e.VictimTeam
		}
	case *PlayerHurtExtra:
		if e != nil {
			params.HealthDamage = int64(e.HealthDamage)
			params.AttackerName = e.AttackerName
			params.VictimName = e.VictimName
			params.AttackerTeam = e.AttackerTeam
			params.VictimTeam = e.VictimTeam
		}
	case *PlayerFlashedExtra:
		if e != nil {
			params.AttackerName = e.AttackerName
			params.VictimName = e.VictimName
			params.AttackerTeam = e.AttackerTeam
			params.VictimTeam = e.VictimTeam
		}
	}

	return params, nil
}

// resolveRoundID looks up the DB round ID for a given round number.
// Returns 0 if roundNumber is 0, roundMap is nil, or the round is not found.
func resolveRoundID(roundNumber int, roundMap map[int]int64) int64 {
	if roundNumber == 0 || roundMap == nil {
		return 0
	}
	return roundMap[roundNumber]
}

// marshalExtraData serializes the typed extras (see extras.go) to a JSON
// string for storage. Returns "{}" for nil so the DB column is never empty
// — frontend readers are happy with `{}` and the heatmap query's
// json_extract calls are happy too. Empty/zero-value structs still marshal
// to e.g. `{"site":""}`, which is the same shape the previous map path
// produced.
func marshalExtraData(extra any) (string, error) {
	if extra == nil {
		return "{}", nil
	}
	data, err := json.Marshal(extra)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
