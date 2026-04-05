package demo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// IngestDB abstracts the database operations needed by IngestRounds.
// *sql.DB satisfies this interface.
type IngestDB interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// IngestRounds calculates per-player-per-round stats from a ParseResult and
// inserts rounds + player_rounds into the database. The operation is idempotent:
// existing rounds for the demo are deleted before insertion.
func IngestRounds(ctx context.Context, db IngestDB, demoID uuid.UUID, result *ParseResult) error {
	if len(result.Rounds) == 0 {
		return nil
	}

	statsMap := CalculatePlayerRoundStats(result.Rounds, result.Events)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q := store.New(tx)

	// Delete existing rounds (cascades to player_rounds via FK).
	if err := q.DeleteRoundsByDemoID(ctx, demoID); err != nil {
		return fmt.Errorf("delete existing rounds: %w", err)
	}

	for _, rd := range result.Rounds {
		round, err := q.CreateRound(ctx, store.CreateRoundParams{
			DemoID:      demoID,
			RoundNumber: int16(rd.Number),
			StartTick:   int32(rd.StartTick),
			EndTick:     int32(rd.EndTick),
			WinnerSide:  rd.WinnerSide,
			WinReason:   rd.WinReason,
			CtScore:     int16(rd.CTScore),
			TScore:      int16(rd.TScore),
			IsOvertime:  rd.IsOvertime,
		})
		if err != nil {
			return fmt.Errorf("create round %d: %w", rd.Number, err)
		}

		playerStats := statsMap[rd.Number]
		for _, ps := range playerStats {
			_, err := q.CreatePlayerRound(ctx, store.CreatePlayerRoundParams{
				RoundID:       round.ID,
				SteamID:       ps.SteamID,
				PlayerName:    ps.PlayerName,
				TeamSide:      ps.TeamSide,
				Kills:         int16(ps.Kills),
				Deaths:        int16(ps.Deaths),
				Assists:       int16(ps.Assists),
				Damage:        int32(ps.Damage),
				HeadshotKills: int16(ps.HeadshotKills),
				FirstKill:     ps.FirstKill,
				FirstDeath:    ps.FirstDeath,
				ClutchKills:   int16(ps.ClutchKills),
			})
			if err != nil {
				return fmt.Errorf("create player_round (round %d, steam %s): %w", rd.Number, ps.SteamID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
