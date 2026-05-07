package demo

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/ok2ju/oversite/internal/store"
)

// DefaultBatchSize is the number of rows inserted per transaction batch.
const DefaultBatchSize = 10_000

// TickIngester batch-inserts tick data into SQLite.
type TickIngester struct {
	db        *sql.DB
	batchSize int
}

// NewTickIngester creates a TickIngester. If batchSize <= 0, DefaultBatchSize is used.
func NewTickIngester(db *sql.DB, batchSize int) *TickIngester {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	return &TickIngester{
		db:        db,
		batchSize: batchSize,
	}
}

// Ingest writes tick snapshots for a demo into the database, replacing any existing
// tick data for the given demoID. It returns the total number of rows inserted.
func (ti *TickIngester) Ingest(ctx context.Context, demoID int64, ticks []TickSnapshot) (int64, error) {
	if len(ticks) == 0 {
		return 0, nil
	}

	slog.Info("starting tick ingestion", "demo_id", demoID, "tick_count", len(ticks))

	params := convertTicksToParams(demoID, ticks)
	batches := chunkTickParams(params, ti.batchSize)

	tx, err := ti.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := store.New(tx)

	if err := q.DeleteTickDataByDemoID(ctx, demoID); err != nil {
		return 0, fmt.Errorf("delete existing tick data: %w", err)
	}

	var total int64
	for i, batch := range batches {
		slog.Debug("inserting batch", "batch", i+1, "size", len(batch), "demo_id", demoID)
		for _, p := range batch {
			if err := q.InsertTickData(ctx, p); err != nil {
				return 0, fmt.Errorf("insert tick data (batch %d): %w", i+1, err)
			}
		}
		total += int64(len(batch))
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	slog.Info("tick ingestion complete", "demo_id", demoID, "rows_inserted", total)
	return total, nil
}

// convertTicksToParams maps parser TickSnapshots to sqlc InsertTickDataParams.
func convertTicksToParams(demoID int64, ticks []TickSnapshot) []store.InsertTickDataParams {
	params := make([]store.InsertTickDataParams, len(ticks))
	for i, t := range ticks {
		params[i] = store.InsertTickDataParams{
			DemoID:     demoID,
			Tick:       int64(t.Tick),
			SteamID:    t.SteamID,
			X:          t.X,
			Y:          t.Y,
			Z:          t.Z,
			Yaw:        t.Yaw,
			Health:     int64(t.Health),
			Armor:      int64(t.Armor),
			IsAlive:    boolToInt64(t.IsAlive),
			Weapon:     t.Weapon,
			Money:      int64(t.Money),
			HasHelmet:  boolToInt64(t.HasHelmet),
			HasDefuser: boolToInt64(t.HasDefuser),
			Inventory:  t.Inventory,
		}
	}
	return params
}

// chunkTickParams splits rows into batches of at most n elements.
func chunkTickParams(rows []store.InsertTickDataParams, n int) [][]store.InsertTickDataParams {
	if len(rows) == 0 {
		return nil
	}
	var chunks [][]store.InsertTickDataParams
	for i := 0; i < len(rows); i += n {
		end := i + n
		if end > len(rows) {
			end = len(rows)
		}
		chunks = append(chunks, rows[i:end])
	}
	return chunks
}
