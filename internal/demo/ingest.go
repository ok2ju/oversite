package demo

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ok2ju/oversite/internal/store"
)

// DefaultBatchSize is the number of rows inserted per multi-row VALUES statement.
//
// 16 columns × 500 rows = 8000 placeholders, well below SQLite's default
// SQLITE_LIMIT_VARIABLE_NUMBER (32766). Batching like this is the single
// biggest ingest speedup available: it cuts SQL parse + WAL frame overhead
// by ~100× compared to per-row INSERTs.
const DefaultBatchSize = 500

// DefaultTickSinkBuffer is the channel buffer size used by the streaming
// parse → ingest pipeline. ~5000 TickSnapshots is roughly 750 KB on the
// heap — large enough that a brief WAL fsync stall doesn't starve the
// parser, small enough that we don't recreate the 100+ MB peak we were
// streaming to avoid in the first place. Sized as ~10× the ingest batch
// so a single full-batch exec drains roughly 1/10th of the channel.
const DefaultTickSinkBuffer = 5000

// tickColumnCount is the number of columns inserted per row in tick_data.
// Must match the column list in tickInsertColumns.
const tickColumnCount = 16

const tickInsertColumns = "demo_id, tick, steam_id, x, y, z, yaw, health, armor, is_alive, weapon, money, has_helmet, has_defuser, ammo_clip, ammo_reserve"

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
// This is a thin wrapper that fans the slice into a buffered channel and
// delegates to IngestStream so the per-row batching/commit logic only lives
// in one place. Callers with a known-bounded slice (tests, small fixtures)
// can keep using this; the streaming pipeline (app.go parseDemo) goes
// through IngestStream directly to overlap with parsing.
func (ti *TickIngester) Ingest(ctx context.Context, demoID int64, ticks []TickSnapshot) (int64, error) {
	if len(ticks) == 0 {
		return 0, nil
	}

	// Buffer sized so the producer never blocks: the consumer drains in one
	// pass, and we don't pay for goroutine scheduling thrash on small inputs.
	ch := make(chan TickSnapshot, len(ticks))
	for _, t := range ticks {
		ch <- t
	}
	close(ch)

	return ti.IngestStream(ctx, demoID, ch)
}

// IngestStream drains TickSnapshots from ticksIn, batching them into multi-row
// INSERTs against tick_data inside a single transaction. It returns the total
// number of rows inserted when the channel closes (parser finished) or
// ctx.Err() when the caller cancels — either way, defer-rollback wipes any
// uncommitted partial work.
//
// The transaction starts before the first read so DeleteTickDataByDemoID and
// the row inserts share atomicity: a failure mid-stream leaves zero rows for
// the demoID rather than a broken half-set. The caller is responsible for
// closing the channel exactly once when the producer exits (Parse does this
// via a top-level defer).
func (ti *TickIngester) IngestStream(ctx context.Context, demoID int64, ticksIn <-chan TickSnapshot) (total int64, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("ingest tick stream: panic: %v", rec)
		}
	}()

	slog.Info("starting tick ingestion (stream)", "demo_id", demoID, "batch_size", ti.batchSize)

	tx, err := ti.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := store.New(tx)
	if err := q.DeleteTickDataByDemoID(ctx, demoID); err != nil {
		return 0, fmt.Errorf("delete existing tick data: %w", err)
	}

	fullStmt, err := tx.PrepareContext(ctx, buildTickBatchInsert(ti.batchSize))
	if err != nil {
		return 0, fmt.Errorf("prepare full batch insert: %w", err)
	}
	defer fullStmt.Close() //nolint:errcheck

	batch := make([]TickSnapshot, 0, ti.batchSize)
	args := make([]any, 0, ti.batchSize*tickColumnCount)

	flush := func(stmt *sql.Stmt, n int) error {
		if n == 0 {
			return nil
		}
		args = appendTickBatchArgs(args[:0], demoID, batch[:n])
		if _, execErr := stmt.ExecContext(ctx, args...); execErr != nil {
			return fmt.Errorf("insert tick batch (%d rows, total so far %d): %w", n, total, execErr)
		}
		total += int64(n)
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		case t, ok := <-ticksIn:
			if !ok {
				// Producer closed the channel; flush any remaining partial batch
				// with a one-off prepared statement of the right shape.
				if len(batch) > 0 {
					partial, perr := tx.PrepareContext(ctx, buildTickBatchInsert(len(batch)))
					if perr != nil {
						return total, fmt.Errorf("prepare partial batch insert (%d rows): %w", len(batch), perr)
					}
					if err := flush(partial, len(batch)); err != nil {
						_ = partial.Close()
						return total, err
					}
					if err := partial.Close(); err != nil {
						return total, fmt.Errorf("close partial stmt: %w", err)
					}
				}
				if err := tx.Commit(); err != nil {
					return total, fmt.Errorf("commit tx: %w", err)
				}
				slog.Info("tick ingestion complete (stream)", "demo_id", demoID, "rows_inserted", total)
				return total, nil
			}
			batch = append(batch, t)
			if len(batch) >= ti.batchSize {
				if err := flush(fullStmt, ti.batchSize); err != nil {
					return total, err
				}
				batch = batch[:0]
			}
		}
	}
}

// buildTickBatchInsert returns an INSERT statement with rowCount value tuples.
func buildTickBatchInsert(rowCount int) string {
	const tuple = "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"
	var b strings.Builder
	b.Grow(64 + len(tickInsertColumns) + rowCount*(len(tuple)+1))
	b.WriteString("INSERT INTO tick_data (")
	b.WriteString(tickInsertColumns)
	b.WriteString(") VALUES ")
	for i := 0; i < rowCount; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(tuple)
	}
	return b.String()
}

// appendTickBatchArgs appends one positional arg per column for each tick in batch.
func appendTickBatchArgs(args []any, demoID int64, batch []TickSnapshot) []any {
	for _, t := range batch {
		args = append(args,
			demoID,
			int64(t.Tick),
			t.SteamID,
			t.X, t.Y, t.Z, t.Yaw,
			int64(t.Health),
			int64(t.Armor),
			boolToInt64(t.IsAlive),
			t.Weapon,
			int64(t.Money),
			boolToInt64(t.HasHelmet),
			boolToInt64(t.HasDefuser),
			int64(t.AmmoClip),
			int64(t.AmmoReserve),
		)
	}
	return args
}
