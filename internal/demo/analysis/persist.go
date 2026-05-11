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
	return PersistWithRoundMap(ctx, db, demoID, mistakes, nil, nil)
}

// PersistWithRoundMap is the round-aware variant. roundMap (round_number →
// rounds.id) lets each row carry the canonical round_id back-reference so the
// analysis_mistakes.round_id FK CASCADE deletes when a round is dropped.
// Callers without a round map can pass nil — round_id is left NULL.
//
// Slice 13 adds duels: when non-nil, the duels are inserted first (delete-
// then-insert by demo) and the returned rowids are mapped onto each
// mistake's DuelID before the mistake row is written. Mistakes whose
// detector-local DuelID doesn't resolve in the map (e.g. a duel was
// pruned post-detection) fall back to NULL — the row still persists.
func PersistWithRoundMap(ctx context.Context, db *sql.DB, demoID int64, mistakes []Mistake, duels []Duel, roundMap map[int]int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := store.New(tx)
	// Delete duels FIRST so the analysis_mistakes.duel_id FK doesn't dangle
	// during the mistakes delete (SET NULL on cascade is fine, but doing
	// duels first lets the transaction stay narrow and explicit).
	if err := q.DeleteAnalysisDuelsByDemoID(ctx, demoID); err != nil {
		return fmt.Errorf("delete existing analysis duels: %w", err)
	}
	if err := q.DeleteAnalysisMistakesByDemoID(ctx, demoID); err != nil {
		return fmt.Errorf("delete existing analysis mistakes: %w", err)
	}

	// localID (detector) → rowid (persisted). Used to rewrite each
	// mistake's DuelID after duel inserts return.
	localToRowID := make(map[int64]int64, len(duels))
	for _, d := range duels {
		extras, err := marshalExtras(d.Extras)
		if err != nil {
			return fmt.Errorf("marshal duel extras (round=%d, A=%s, V=%s): %w", d.RoundNumber, d.AttackerSteam, d.VictimSteam, err)
		}
		var roundID sql.NullInt64
		if roundMap != nil {
			if id, ok := roundMap[d.RoundNumber]; ok {
				roundID = sql.NullInt64{Int64: id, Valid: true}
			}
		}
		hitConfirmed := int64(0)
		if d.HitConfirmed {
			hitConfirmed = 1
		}
		rowID, err := q.CreateAnalysisDuel(ctx, store.CreateAnalysisDuelParams{
			DemoID:        demoID,
			RoundNumber:   int64(d.RoundNumber),
			RoundID:       roundID,
			AttackerSteam: d.AttackerSteam,
			VictimSteam:   d.VictimSteam,
			StartTick:     int64(d.StartTick),
			EndTick:       int64(d.EndTick),
			Outcome:       d.Outcome,
			EndReason:     d.EndReason,
			HitConfirmed:  hitConfirmed,
			HurtCount:     int64(d.HurtCount),
			ShotCount:     int64(d.ShotCount),
			ExtrasJson:    extras,
		})
		if err != nil {
			return fmt.Errorf("insert analysis duel (round=%d, A=%s, V=%s): %w", d.RoundNumber, d.AttackerSteam, d.VictimSteam, err)
		}
		localToRowID[int64(d.LocalID)] = rowID
	}
	// Second pass: backfill mutual_duel_id now that every duel has a rowid.
	for _, d := range duels {
		if d.MutualLocalID <= 0 {
			continue
		}
		selfID, ok := localToRowID[int64(d.LocalID)]
		if !ok {
			continue
		}
		peerID, ok := localToRowID[int64(d.MutualLocalID)]
		if !ok {
			continue
		}
		if err := q.UpdateAnalysisDuelMutual(ctx, store.UpdateAnalysisDuelMutualParams{
			ID:           selfID,
			MutualDuelID: sql.NullInt64{Int64: peerID, Valid: true},
		}); err != nil {
			return fmt.Errorf("update mutual_duel_id (id=%d): %w", selfID, err)
		}
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
		var duelID sql.NullInt64
		if m.DuelID != nil {
			if rowID, ok := localToRowID[*m.DuelID]; ok {
				duelID = sql.NullInt64{Int64: rowID, Valid: true}
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
			DuelID:      duelID,
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
