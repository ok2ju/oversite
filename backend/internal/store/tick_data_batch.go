package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

// CopyTickData performs a bulk insert of tick data rows using PostgreSQL's COPY protocol.
// This is orders of magnitude faster than individual INSERTs for large batches (1M+ rows).
func CopyTickData(ctx context.Context, db *sql.DB, rows []InsertTickDataParams) (int64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

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

	// Flush the COPY stream
	if _, err := stmt.ExecContext(ctx); err != nil {
		_ = stmt.Close()
		return 0, fmt.Errorf("flush copy: %w", err)
	}
	_ = stmt.Close()

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	return int64(len(rows)), nil
}
