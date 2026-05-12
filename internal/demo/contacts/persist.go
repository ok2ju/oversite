package contacts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ok2ju/oversite/internal/store"
)

// Persist replaces the contact_moments rows for demoID with the supplied
// list. Wraps delete + insert in a single transaction so a failure
// mid-way rolls back. Idempotent: re-running with the same input
// converges on the same rows. contact_mistakes is wiped first via
// CASCADE through the delete; the explicit DeleteContactMistakesByDemoID
// keeps the transaction auditable.
//
// Phase 2 never inserts contact_mistakes rows — Phase 3's detector pass
// owns that table.
func Persist(ctx context.Context, db *sql.DB, demoID int64, contacts []ContactMoment) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := store.New(tx)

	if err := q.DeleteContactMistakesByDemoID(ctx, demoID); err != nil {
		return fmt.Errorf("delete existing contact mistakes: %w", err)
	}
	if err := q.DeleteContactMomentsByDemoID(ctx, demoID); err != nil {
		return fmt.Errorf("delete existing contact moments: %w", err)
	}

	for i := range contacts {
		c := &contacts[i]
		enemiesJSON, err := json.Marshal(c.Enemies)
		if err != nil {
			return fmt.Errorf("marshal enemies (subject=%s, t_first=%d): %w",
				c.SubjectSteam, c.TFirst, err)
		}
		extrasJSON, err := marshalContactExtras(c.Extras)
		if err != nil {
			return fmt.Errorf("marshal extras (subject=%s, t_first=%d): %w",
				c.SubjectSteam, c.TFirst, err)
		}

		id, err := q.InsertContactMoment(ctx, store.InsertContactMomentParams{
			DemoID:         demoID,
			RoundID:        c.RoundID,
			SubjectSteam:   c.SubjectSteam,
			TFirst:         int64(c.TFirst),
			TLast:          int64(c.TLast),
			TPre:           int64(c.TPre),
			TPost:          int64(c.TPost),
			EnemiesJson:    string(enemiesJSON),
			Outcome:        string(c.Outcome),
			SignalCount:    int64(c.SignalCount),
			ExtrasJson:     extrasJSON,
			BuilderVersion: int64(BuilderVersion),
		})
		if err != nil {
			return fmt.Errorf("insert contact (subject=%s, t_first=%d): %w",
				c.SubjectSteam, c.TFirst, err)
		}
		c.ID = id
		c.DemoID = demoID
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// marshalContactExtras serializes ContactExtras to JSON, returning "{}"
// when no flag is set. Same shape as analysis/persist.go marshalExtras.
func marshalContactExtras(e ContactExtras) (string, error) {
	if e == (ContactExtras{}) {
		return "{}", nil
	}
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
