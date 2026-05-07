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
//
// Conversion to InsertTickDataParams happens row-by-row inside the insert
// loop rather than up-front: pre-building the full params slice doubles peak
// memory during ingestion of large demos (300K+ rows can be hundreds of MB).
func (ti *TickIngester) Ingest(ctx context.Context, demoID int64, ticks []TickSnapshot) (int64, error) {
	if len(ticks) == 0 {
		return 0, nil
	}

	slog.Info("starting tick ingestion", "demo_id", demoID, "tick_count", len(ticks))

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
	for i := 0; i < len(ticks); i += ti.batchSize {
		end := i + ti.batchSize
		if end > len(ticks) {
			end = len(ticks)
		}
		slog.Info("tick ingestion batch", "demo_id", demoID,
			"start", i, "end", end, "of", len(ticks))
		for j := i; j < end; j++ {
			if err := q.InsertTickData(ctx, tickToParams(demoID, ticks[j])); err != nil {
				return 0, fmt.Errorf("insert tick data (row %d): %w", j, err)
			}
		}
		total += int64(end - i)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	slog.Info("tick ingestion complete", "demo_id", demoID, "rows_inserted", total)
	return total, nil
}

// tickToParams converts a single TickSnapshot to InsertTickDataParams without
// allocating an intermediate slice. Used by the streaming insert path.
func tickToParams(demoID int64, t TickSnapshot) store.InsertTickDataParams {
	return store.InsertTickDataParams{
		DemoID:      demoID,
		Tick:        int64(t.Tick),
		SteamID:     t.SteamID,
		X:           t.X,
		Y:           t.Y,
		Z:           t.Z,
		Yaw:         t.Yaw,
		Health:      int64(t.Health),
		Armor:       int64(t.Armor),
		IsAlive:     boolToInt64(t.IsAlive),
		Weapon:      t.Weapon,
		Money:       int64(t.Money),
		HasHelmet:   boolToInt64(t.HasHelmet),
		HasDefuser:  boolToInt64(t.HasDefuser),
		Inventory:   t.Inventory,
		AmmoClip:    int64(t.AmmoClip),
		AmmoReserve: int64(t.AmmoReserve),
	}
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
