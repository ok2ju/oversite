package detectors

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ok2ju/oversite/internal/store"
)

// Persist replaces the contact_mistakes rows for demoID with the
// supplied list. Wraps delete + inserts in a single transaction so a
// failure mid-way rolls back to the prior state. Idempotent: re-
// running with the same input converges on the same rows.
func Persist(ctx context.Context, db *sql.DB, demoID int64, rows []BoundContactMistake) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := store.New(tx)
	if err := q.DeleteContactMistakesByDemoID(ctx, demoID); err != nil {
		return fmt.Errorf("delete existing contact mistakes: %w", err)
	}
	for _, r := range rows {
		extrasJSON, err := marshalMistakeExtras(r.Mistake.Extras)
		if err != nil {
			return fmt.Errorf("marshal extras (kind=%s): %w", r.Mistake.Kind, err)
		}
		var tickArg sql.NullInt64
		if r.Mistake.Tick != nil {
			tickArg = sql.NullInt64{Int64: int64(*r.Mistake.Tick), Valid: true}
		}
		if err := q.InsertContactMistake(ctx, store.InsertContactMistakeParams{
			ContactID:       r.ContactID,
			Kind:            r.Mistake.Kind,
			Category:        r.Mistake.Category,
			Severity:        int64(r.Mistake.Severity),
			Phase:           r.Mistake.Phase,
			Tick:            tickArg,
			ExtrasJson:      extrasJSON,
			DetectorVersion: int64(DetectorVersion),
		}); err != nil {
			return fmt.Errorf("insert contact mistake (kind=%s, contact_id=%d): %w",
				r.Mistake.Kind, r.ContactID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// marshalMistakeExtras serializes the per-mistake extras blob to a
// stable JSON string. "{}" for nil/empty so the column is never empty.
func marshalMistakeExtras(e map[string]any) (string, error) {
	if len(e) == 0 {
		return "{}", nil
	}
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
