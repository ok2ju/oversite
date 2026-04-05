package demo

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/ok2ju/oversite/backend/internal/store"
)

// DefaultBatchSize is the default number of rows per COPY batch.
const DefaultBatchSize = 10_000

const defaultTickRate = 64.0

// TickIngester converts parsed tick snapshots into TimescaleDB rows
// using chunked PostgreSQL COPY within a single transaction.
type TickIngester struct {
	db        *sql.DB
	batchSize int
}

// NewTickIngester creates a TickIngester. batchSize controls how many rows
// are sent per COPY batch; values <= 0 use DefaultBatchSize.
func NewTickIngester(db *sql.DB, batchSize int) *TickIngester {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	return &TickIngester{db: db, batchSize: batchSize}
}

// Ingest converts ticks to DB params, deletes any existing rows for demoID
// (idempotent re-ingestion), and bulk-inserts in batches within one transaction.
// Returns the total number of rows inserted.
func (ti *TickIngester) Ingest(ctx context.Context, demoID uuid.UUID, ticks []TickSnapshot, matchDate time.Time, tickRate float64) (int64, error) {
	if len(ticks) == 0 {
		return 0, nil
	}

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
	if _, err := tx.ExecContext(ctx, "DELETE FROM tick_data WHERE demo_id = $1", demoID); err != nil {
		return 0, fmt.Errorf("delete existing tick data: %w", err)
	}

	var total int64
	for i, batch := range batches {
		n, err := copyTickDataTx(ctx, tx, batch)
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
}

// copyTickDataTx performs a COPY insert within an existing transaction.
func copyTickDataTx(ctx context.Context, tx *sql.Tx, rows []store.InsertTickDataParams) (int64, error) {
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn(
		"tick_data",
		"time", "demo_id", "tick", "steam_id",
		"x", "y", "z", "yaw",
		"health", "armor", "is_alive", "weapon",
	))
	if err != nil {
		return 0, fmt.Errorf("prepare copy: %w", err)
	}

	for _, r := range rows {
		if _, err := stmt.ExecContext(ctx,
			r.Time, r.DemoID, r.Tick, r.SteamID,
			r.X, r.Y, r.Z, r.Yaw,
			r.Health, r.Armor, r.IsAlive, r.Weapon,
		); err != nil {
			_ = stmt.Close()
			return 0, fmt.Errorf("copy row: %w", err)
		}
	}

	// Flush the COPY stream.
	if _, err := stmt.ExecContext(ctx); err != nil {
		_ = stmt.Close()
		return 0, fmt.Errorf("flush copy: %w", err)
	}
	_ = stmt.Close()

	return int64(len(rows)), nil
}
