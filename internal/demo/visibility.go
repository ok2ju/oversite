package demo

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/ok2ju/oversite/internal/store"
)

// IngestPlayerVisibility persists the parser's debounced visibility
// transitions for the given demo. The roundMap is the in-demo round
// number → database round_id mapping returned by IngestRounds.
//
// Behavior mirrors IngestGameEvents:
//   - opens its own transaction
//   - deletes existing rows for the demo (re-import safe)
//   - inserts each row through sqlc
//   - skips rows whose round_number didn't map (pre-match / warmup leak)
//   - returns the number of rows inserted
func IngestPlayerVisibility(
	ctx context.Context,
	db *sql.DB,
	demoID int64,
	changes []VisibilityChange,
	roundMap map[int]int64,
) (int, error) {
	if len(changes) == 0 {
		return 0, nil
	}

	slog.Info("starting visibility ingestion", "demo_id", demoID, "row_count", len(changes))

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := store.New(tx)

	if err := q.DeleteVisibilityByDemoID(ctx, demoID); err != nil {
		return 0, fmt.Errorf("delete existing visibility: %w", err)
	}

	inserted := 0
	for _, c := range changes {
		roundID := resolveRoundID(c.RoundNumber, roundMap)
		if roundID == 0 {
			continue // round didn't make it through IngestRounds (warmup / unmapped)
		}
		if err := q.InsertPlayerVisibility(ctx, store.InsertPlayerVisibilityParams{
			DemoID:       demoID,
			RoundID:      roundID,
			Tick:         int64(c.Tick),
			SpottedSteam: c.SpottedSteam,
			SpotterSteam: c.SpotterSteam,
			State:        int64(c.State),
		}); err != nil {
			return inserted, fmt.Errorf(
				"insert visibility (round %d, tick %d, %s→%s): %w",
				c.RoundNumber, c.Tick, c.SpottedSteam, c.SpotterSteam, err,
			)
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return inserted, fmt.Errorf("commit tx: %w", err)
	}

	slog.Info("visibility ingestion complete", "demo_id", demoID, "rows_inserted", inserted)
	return inserted, nil
}
