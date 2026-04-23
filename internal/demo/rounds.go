package demo

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/ok2ju/oversite/internal/store"
)

// IngestRounds calculates per-player stats, then inserts rounds + player_rounds
// in a single transaction. Returns roundNumber -> DB roundID map (needed by event ingestion).
// Idempotent: deletes existing rounds first (FK cascade deletes player_rounds).
func IngestRounds(ctx context.Context, db *sql.DB, demoID int64, result *ParseResult) (map[int]int64, error) {
	if len(result.Rounds) == 0 {
		return nil, nil
	}

	slog.Info("starting round ingestion", "demo_id", demoID, "round_count", len(result.Rounds))

	// Calculate player stats BEFORE opening the transaction.
	statsMap := CalculatePlayerRoundStats(result.Rounds, result.Events)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := store.New(tx)

	// Delete existing rounds — FK cascade deletes player_rounds.
	if err := q.DeleteRoundsByDemoID(ctx, demoID); err != nil {
		return nil, fmt.Errorf("delete existing rounds: %w", err)
	}

	roundMap := make(map[int]int64, len(result.Rounds))

	for _, rd := range result.Rounds {
		round, err := q.CreateRound(ctx, store.CreateRoundParams{
			DemoID:        demoID,
			RoundNumber:   int64(rd.Number),
			StartTick:     int64(rd.StartTick),
			FreezeEndTick: int64(rd.FreezeEndTick),
			EndTick:       int64(rd.EndTick),
			WinnerSide:    rd.WinnerSide,
			WinReason:     rd.WinReason,
			CtScore:       int64(rd.CTScore),
			TScore:        int64(rd.TScore),
			IsOvertime:    boolToInt64(rd.IsOvertime),
		})
		if err != nil {
			return nil, fmt.Errorf("insert round %d: %w", rd.Number, err)
		}

		roundMap[rd.Number] = round.ID

		// Insert player round stats for this round.
		for _, ps := range statsMap[rd.Number] {
			if _, err := q.CreatePlayerRound(ctx, store.CreatePlayerRoundParams{
				RoundID:       round.ID,
				SteamID:       ps.SteamID,
				PlayerName:    ps.PlayerName,
				TeamSide:      ps.TeamSide,
				Kills:         int64(ps.Kills),
				Deaths:        int64(ps.Deaths),
				Assists:       int64(ps.Assists),
				Damage:        int64(ps.Damage),
				HeadshotKills: int64(ps.HeadshotKills),
				FirstKill:     boolToInt64(ps.FirstKill),
				FirstDeath:    boolToInt64(ps.FirstDeath),
				ClutchKills:   int64(ps.ClutchKills),
			}); err != nil {
				return nil, fmt.Errorf("insert player round (round %d, steam %s): %w", rd.Number, ps.SteamID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	slog.Info("round ingestion complete", "demo_id", demoID, "rounds_inserted", len(result.Rounds))
	return roundMap, nil
}
