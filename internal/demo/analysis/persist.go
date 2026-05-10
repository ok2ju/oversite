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
//
// Category and Severity columns are filled from templates.go at insert time
// when the Mistake's own fields are zero (the rules in mistakes.go don't
// populate them; rule authors don't have to remember).
func Persist(ctx context.Context, db *sql.DB, demoID int64, mistakes []Mistake) error {
	return PersistWithRoundMap(ctx, db, demoID, mistakes, nil)
}

// PersistWithRoundMap is the round-aware variant. roundMap (round_number →
// rounds.id) lets each row carry the canonical round_id back-reference so the
// analysis_mistakes.round_id FK CASCADE deletes when a round is dropped.
// Callers without a round map can pass nil — round_id is left NULL.
func PersistWithRoundMap(ctx context.Context, db *sql.DB, demoID int64, mistakes []Mistake, roundMap map[int]int64) error {
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
		category := m.Category
		severity := m.Severity
		if category == "" || severity == 0 {
			tpl := TemplateForKind(m.Kind)
			if category == "" {
				category = string(tpl.Category)
			}
			if severity == 0 {
				severity = int(tpl.Severity)
			}
		}
		var roundID sql.NullInt64
		if roundMap != nil {
			if id, ok := roundMap[m.RoundNumber]; ok {
				roundID = sql.NullInt64{Int64: id, Valid: true}
			}
		}
		if err := q.CreateAnalysisMistake(ctx, store.CreateAnalysisMistakeParams{
			DemoID:      demoID,
			SteamID:     m.SteamID,
			RoundNumber: int64(m.RoundNumber),
			RoundID:     roundID,
			Tick:        int64(m.Tick),
			Kind:        m.Kind,
			Category:    category,
			Severity:    int64(severity),
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

// PersistMatchSummary replaces the player_match_analysis rows for demoID with
// the supplied list. Wraps the delete + upserts in a single transaction so a
// failure mid-way rolls back to the prior state. Idempotent: re-running with
// the same input converges on the same rows; running with an empty slice
// wipes any prior rows.
func PersistMatchSummary(ctx context.Context, db *sql.DB, demoID int64, rows []MatchSummaryRow) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := store.New(tx)
	if err := q.DeletePlayerMatchAnalysisByDemoID(ctx, demoID); err != nil {
		return fmt.Errorf("delete existing player match analysis: %w", err)
	}

	for _, r := range rows {
		extras, err := marshalExtras(r.Extras)
		if err != nil {
			return fmt.Errorf("marshal extras (steam=%s): %w", r.SteamID, err)
		}
		version := r.Version
		if version == 0 {
			version = AnalysisVersion
		}
		if err := q.UpsertPlayerMatchAnalysis(ctx, store.UpsertPlayerMatchAnalysisParams{
			DemoID:                     demoID,
			SteamID:                    r.SteamID,
			OverallScore:               int64(r.OverallScore),
			TradePct:                   r.TradePct,
			AvgTradeTicks:              r.AvgTradeTicks,
			Version:                    int64(version),
			CrosshairHeightAvgOff:      r.CrosshairHeightAvgOff,
			TimeToFireMsAvg:            r.TimeToFireMsAvg,
			FlickCount:                 int64(r.FlickCount),
			FlickHitPct:                r.FlickHitPct,
			FirstShotAccPct:            r.FirstShotAccPct,
			SprayDecaySlope:            r.SprayDecaySlope,
			StandingShotPct:            r.StandingShotPct,
			CounterStrafePct:           r.CounterStrafePct,
			SmokesThrown:               int64(r.SmokesThrown),
			SmokesKillAssist:           int64(r.SmokesKillAssist),
			FlashAssists:               int64(r.FlashAssists),
			HeDamage:                   int64(r.HeDamage),
			NadesUnused:                int64(r.NadesUnused),
			IsolatedPeekDeaths:         int64(r.IsolatedPeekDeaths),
			RepeatedDeathZones:         int64(r.RepeatedDeathZones),
			FullBuyAdr:                 r.FullBuyADR,
			EcoKills:                   int64(r.EcoKills),
			TimeToStopMsAvg:            r.TimeToStopMsAvg,
			CrouchBeforeShotCount:      int64(r.CrouchBeforeShotCount),
			CrouchInsteadOfStrafeCount: int64(r.CrouchInsteadOfStrafeCount),
			FlickOvershootAvgDeg:       r.FlickOvershootAvgDeg,
			FlickUndershootAvgDeg:      r.FlickUndershootAvgDeg,
			FlickBalancePct:            r.FlickBalancePct,
			ExtrasJson:                 extras,
		}); err != nil {
			return fmt.Errorf("upsert player match analysis (steam=%s): %w", r.SteamID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// PersistPlayerRoundAnalysis replaces the player_round_analysis rows for
// demoID with the supplied list. Wraps the delete + upserts in a single
// transaction so a failure mid-way rolls back to the prior state. Idempotent:
// re-running with the same input converges on the same rows; running with an
// empty slice wipes any prior rows.
func PersistPlayerRoundAnalysis(ctx context.Context, db *sql.DB, demoID int64, rows []PlayerRoundRow) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := store.New(tx)
	if err := q.DeletePlayerRoundAnalysisByDemoID(ctx, demoID); err != nil {
		return fmt.Errorf("delete existing player round analysis: %w", err)
	}

	for _, r := range rows {
		extras, err := marshalExtras(r.Extras)
		if err != nil {
			return fmt.Errorf("marshal extras (steam=%s, round=%d): %w", r.SteamID, r.RoundNumber, err)
		}
		if err := q.UpsertPlayerRoundAnalysis(ctx, store.UpsertPlayerRoundAnalysisParams{
			DemoID:      demoID,
			SteamID:     r.SteamID,
			RoundNumber: int64(r.RoundNumber),
			TradePct:    r.TradePct,
			BuyType:     r.BuyType,
			MoneySpent:  int64(r.MoneySpent),
			NadesUsed:   int64(r.NadesUsed),
			NadesUnused: int64(r.NadesUnused),
			ShotsFired:  int64(r.ShotsFired),
			ShotsHit:    int64(r.ShotsHit),
			ExtrasJson:  extras,
		}); err != nil {
			return fmt.Errorf("upsert player round analysis (steam=%s, round=%d): %w", r.SteamID, r.RoundNumber, err)
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
