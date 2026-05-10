package analysis

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ok2ju/oversite/internal/store"
)

// Persist replaces the analysis_mistakes rows for demoID with the supplied
// list. Wraps delete + inserts in a single transaction so a failure mid-way
// rolls back to the prior state instead of leaving a partial set. Idempotent:
// re-running with the same input converges on the same rows.
//
// Mirrors the IngestGameEvents pattern in internal/demo/ingest.go (begin tx,
// delete by demo, insert each row, commit).
func Persist(ctx context.Context, db *sql.DB, demoID int64, mistakes []Mistake) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := store.New(tx)
	if err := q.DeleteAnalysisMistakesByDemoID(ctx, demoID); err != nil {
		return fmt.Errorf("delete existing analysis mistakes: %w", err)
	}

	for _, m := range mistakes {
		extras, err := marshalExtras(m.Extras)
		if err != nil {
			return fmt.Errorf("marshal extras (kind=%s, steam=%s): %w", m.Kind, m.SteamID, err)
		}
		if err := q.CreateAnalysisMistake(ctx, store.CreateAnalysisMistakeParams{
			DemoID:      demoID,
			SteamID:     m.SteamID,
			RoundNumber: int64(m.RoundNumber),
			Tick:        int64(m.Tick),
			Kind:        m.Kind,
			ExtrasJson:  extras,
		}); err != nil {
			return fmt.Errorf("insert analysis mistake (kind=%s, steam=%s): %w", m.Kind, m.SteamID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// marshalExtras serializes the rule's extras blob to a stable JSON string.
// Returns "{}" for nil/empty so the column is never empty (the frontend reads
// it as JSON).
func marshalExtras(extras map[string]any) (string, error) {
	if len(extras) == 0 {
		return "{}", nil
	}
	data, err := json.Marshal(extras)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
