package demo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"github.com/ok2ju/oversite/backend/internal/store"
)

// DefaultBatchSize is the default number of rows per COPY batch.
const DefaultBatchSize = 10_000

const defaultTickRate = 64.0

// IngestDB abstracts the database connection for TickIngester,
// enabling interface-based dependency injection for testing.
type IngestDB interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// TickIngester converts parsed tick snapshots into TimescaleDB rows
// using chunked PostgreSQL COPY within a single transaction.
type TickIngester struct {
	db        IngestDB
	batchSize int
}

// NewTickIngester creates a TickIngester. batchSize controls how many rows
// are sent per COPY batch; values <= 0 use DefaultBatchSize.
func NewTickIngester(db IngestDB, batchSize int) *TickIngester {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	return &TickIngester{db: db, batchSize: batchSize}
}

// Ingest converts ticks to DB params, deletes any existing rows for demoID
// (idempotent re-ingestion), and bulk-inserts in batches within one transaction.
// Returns the total number of rows inserted.
func (ti *TickIngester) Ingest(ctx context.Context, demoID uuid.UUID, ticks []TickSnapshot, matchDate time.Time, tickRate float64) (int64, error) {
	rows := convertTicks(ticks, demoID, matchDate, tickRate)
	batches := chunkTickParams(rows, ti.batchSize)

	slog.Info("starting tick ingestion",
		"demo_id", demoID,
		"total_rows", len(rows),
		"batches", len(batches),
		"batch_size", ti.batchSize,
	)

	tx, err := ti.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Delete existing tick data for idempotent re-ingestion.
	if err := store.New(tx).DeleteTickDataByDemoID(ctx, demoID); err != nil {
		return 0, fmt.Errorf("delete existing tick data: %w", err)
	}

	var total int64
	for i, batch := range batches {
		n, err := store.CopyTickDataTx(ctx, tx, batch)
		if err != nil {
			return 0, fmt.Errorf("copy batch %d/%d: %w", i+1, len(batches), err)
		}
		total += n
		slog.Debug("ingested batch",
			"demo_id", demoID,
			"batch", i+1,
			"of", len(batches),
			"rows", n,
		)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	slog.Info("tick ingestion complete",
		"demo_id", demoID,
		"total_rows", total,
	)

	return total, nil
}

// syntheticTime computes a hypertable partition timestamp from tick offset.
// Formula: baseTime + (tick / tickRate) * second.
func syntheticTime(baseTime time.Time, tick int, tickRate float64) time.Time {
	if tickRate <= 0 {
		tickRate = defaultTickRate
	}
	offsetSecs := float64(tick) / tickRate
	return baseTime.Add(time.Duration(offsetSecs * float64(time.Second)))
}

// chunkTickParams splits rows into sub-slices of at most n elements.
// Returns nil for empty input. Sub-slices share the backing array (no copy).
func chunkTickParams(rows []store.InsertTickDataParams, n int) [][]store.InsertTickDataParams {
	if len(rows) == 0 {
		return nil
	}
	chunks := make([][]store.InsertTickDataParams, 0, (len(rows)+n-1)/n)
	for i := 0; i < len(rows); i += n {
		end := i + n
		if end > len(rows) {
			end = len(rows)
		}
		chunks = append(chunks, rows[i:end])
	}
	return chunks
}

// convertTicks maps parser TickSnapshots to store InsertTickDataParams.
func convertTicks(ticks []TickSnapshot, demoID uuid.UUID, baseTime time.Time, tickRate float64) []store.InsertTickDataParams {
	if len(ticks) == 0 {
		return nil
	}
	rows := make([]store.InsertTickDataParams, len(ticks))
	for i, t := range ticks {
		var weapon sql.NullString
		if t.Weapon != "" {
			weapon = sql.NullString{String: t.Weapon, Valid: true}
		}
		rows[i] = store.InsertTickDataParams{
			Time:    syntheticTime(baseTime, t.Tick, tickRate),
			DemoID:  demoID,
			Tick:    int32(t.Tick),
			SteamID: t.SteamID,
			X:       float32(t.X),
			Y:       float32(t.Y),
			Z:       float32(t.Z),
			Yaw:     float32(t.Yaw),
			Health:  int16(t.Health),
			Armor:   int16(t.Armor),
			IsAlive: t.IsAlive,
			Weapon:  weapon,
		}
	}
	return rows
// GameEventCreator inserts a single game event row.
// GameEventCreator inserts and deletes game event rows.
// Satisfied by *store.Queries (or WithTx variant).
type GameEventCreator interface {
	CreateGameEvent(ctx context.Context, arg store.CreateGameEventParams) (store.GameEvent, error)
	DeleteGameEventsByDemoID(ctx context.Context, demoID uuid.UUID) error
}

// IngestGameEvents deletes existing game events for the demo, then maps parsed
// GameEvents to store params and inserts them. It is idempotent — safe to retry.
// It returns the number of successfully inserted events.
// The caller manages the transaction (pass store.New(tx) as creator).
func IngestGameEvents(
	ctx context.Context,
	creator GameEventCreator,
	demoID uuid.UUID,
	events []GameEvent,
	roundMap map[int]uuid.UUID,
) (int, error) {
	if err := creator.DeleteGameEventsByDemoID(ctx, demoID); err != nil {
		return 0, fmt.Errorf("deleting existing game events for demo %s: %w", demoID, err)
	}

	for i, evt := range events {
		params, err := toCreateGameEventParams(demoID, evt, roundMap)
		if err != nil {
			return i, fmt.Errorf("building params for event %d (tick %d, type %s): %w", i, evt.Tick, evt.Type, err)
		}
		if _, err := creator.CreateGameEvent(ctx, params); err != nil {
			return i, fmt.Errorf("inserting game event %d (tick %d, type %s): %w", i, evt.Tick, evt.Type, err)
		}
	}
	return len(events), nil
}

// toCreateGameEventParams converts a parsed GameEvent to store.CreateGameEventParams.
func toCreateGameEventParams(demoID uuid.UUID, evt GameEvent, roundMap map[int]uuid.UUID) (store.CreateGameEventParams, error) {
	extraData, err := buildExtraData(evt.ExtraData)
	if err != nil {
		return store.CreateGameEventParams{}, err
	}
	return store.CreateGameEventParams{
		DemoID:          demoID,
		RoundID:         resolveRoundID(evt.RoundNumber, roundMap),
		Tick:            int32(evt.Tick),
		EventType:       evt.Type,
		AttackerSteamID: nullString(evt.AttackerSteamID),
		VictimSteamID:   nullString(evt.VictimSteamID),
		Weapon:          nullString(evt.Weapon),
		X:               sql.NullFloat64{Float64: evt.X, Valid: evt.HasPosition},
		Y:               sql.NullFloat64{Float64: evt.Y, Valid: evt.HasPosition},
		Z:               sql.NullFloat64{Float64: evt.Z, Valid: evt.HasPosition},
		ExtraData:       extraData,
	}, nil
}

// resolveRoundID looks up the round DB ID from the roundMap.
// Returns a null UUID if the round number is 0 or not found.
func resolveRoundID(roundNumber int, roundMap map[int]uuid.UUID) uuid.NullUUID {
	if roundNumber == 0 || roundMap == nil {
		return uuid.NullUUID{}
	}
	id, ok := roundMap[roundNumber]
	if !ok {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: id, Valid: true}
}

// buildExtraData marshals a map to JSONB. Returns null for nil/empty maps.
func buildExtraData(extra map[string]interface{}) (pqtype.NullRawMessage, error) {
	if len(extra) == 0 {
		return pqtype.NullRawMessage{}, nil
	}
	data, err := json.Marshal(extra)
	if err != nil {
		return pqtype.NullRawMessage{}, fmt.Errorf("marshalling extra data: %w", err)
	}
	return pqtype.NullRawMessage{RawMessage: data, Valid: true}, nil
}

// nullString converts a string to sql.NullString. Empty strings become NULL.
func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
