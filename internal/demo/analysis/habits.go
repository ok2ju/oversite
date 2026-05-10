package analysis

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ok2ju/oversite/internal/store"
)

// HabitInputs is the data BuildHabitReport needs in order to build the habit
// checklist. Lifted to its own struct so app.go does the SQL fetch and the
// builder stays pure / testable without a database. Ratios stay in their
// native 0..1 scale; the builder converts them to the display unit (% / ms /
// count) for each row.
type HabitInputs struct {
	SteamID string

	// Reaction (ms). Zero means "no fires sampled" — habit omitted.
	TimeToFireMsAvg float64

	// First-shot accuracy as a 0..1 ratio. The PlayerMatchAnalysis column
	// stores a fraction; the row presents it as a percentage.
	FirstShotAccRatio float64
	// Set to true when the source row had any first-shot fires sampled. We
	// use this rather than `>0` because a perfectly-bad shooter with 0/N
	// would otherwise be treated identically to "no data".
	HasFirstShotData bool

	// Trade timing as a 0..1 ratio (traded_deaths / own_deaths).
	TradePctRatio float64

	// Counts straight off player_match_analysis.
	IsolatedPeekDeaths int
	RepeatedDeathZones int

	// Untraded deaths count from analysis_mistakes (kind = no_trade_death).
	UntradedDeathsCount int

	// "standing_shot_pct" from extras_json — the mech-aggregate value
	// (fraction of fires while standing). Present-or-absent flag controls
	// whether the shooting-in-motion habit appears in the report; the
	// PlayerMatchAnalysis column of the same name is *accuracy* of standing
	// shots and does not power this habit.
	StandingShotMechRatio   float64
	HasStandingShotMechData bool
}

// HabitRow is the in-Go shape of one row in the HabitReport. Mirrors the
// HabitRow struct in types.go (which is the wire-level type tagged for JSON).
// Keeping the in-Go struct typed (HabitKey, Direction, Status enums) catches
// builder-side bugs at compile time; types.go converts to plain strings for
// the binding.
type HabitRow struct {
	Key           HabitKey
	Label         string
	Description   string
	Unit          string
	Direction     Direction
	Value         float64
	Status        Status
	GoodThreshold float64
	WarnThreshold float64
	GoodMin       float64
	GoodMax       float64
	WarnMin       float64
	WarnMax       float64
}

// BuildHabitReport assembles the habit checklist from inputs. Habits whose
// underlying metric is not yet computed (counter_strafe ms, crouch %, flick
// balance — pending P3-2) are intentionally omitted, not surfaced as zero
// rows: the frontend renders only what we ship so unfilled metrics don't
// masquerade as "perfect" or "broken".
//
// Order matches AllHabitKeys() so the in-app checklist always paints rows in
// the same sequence regardless of which ones were skipped.
func BuildHabitReport(in HabitInputs) []HabitRow {
	rows := make([]HabitRow, 0, 8)
	add := func(k HabitKey, value float64) {
		n, ok := LookupNorm(k)
		if !ok {
			return
		}
		rows = append(rows, rowFromNorm(n, value))
	}

	if in.TimeToFireMsAvg > 0 {
		add(HabitReaction, in.TimeToFireMsAvg)
	}
	if in.HasFirstShotData {
		add(HabitFirstShotAcc, in.FirstShotAccRatio*100)
	}
	if in.HasStandingShotMechData {
		motion := (1 - in.StandingShotMechRatio) * 100
		if motion < 0 {
			motion = 0
		}
		if motion > 100 {
			motion = 100
		}
		add(HabitShootingInMotion, motion)
	}
	add(HabitTradeTiming, in.TradePctRatio*100)
	add(HabitUntradedDeaths, float64(in.UntradedDeathsCount))
	add(HabitIsolatedPeekDeaths, float64(in.IsolatedPeekDeaths))
	add(HabitRepeatedDeathZone, float64(in.RepeatedDeathZones))
	return rows
}

func rowFromNorm(norm Norm, value float64) HabitRow {
	return HabitRow{
		Key:           norm.Key,
		Label:         norm.Label,
		Description:   norm.Description,
		Unit:          norm.Unit,
		Direction:     norm.Direction,
		Value:         value,
		Status:        ClassifyHabit(value, norm),
		GoodThreshold: norm.GoodThreshold,
		WarnThreshold: norm.WarnThreshold,
		GoodMin:       norm.GoodMin,
		GoodMax:       norm.GoodMax,
		WarnMin:       norm.WarnMin,
		WarnMax:       norm.WarnMax,
	}
}

// LoadHabitInputs reads the per-(demo, player) row plus the untraded-deaths
// count and packs them into a HabitInputs. Returns (zero, false, nil) when
// no player_match_analysis row exists for the (demo, player) — callers should
// surface that as an empty report rather than an error so the analysis page
// renders an "analysis missing" state.
func LoadHabitInputs(ctx context.Context, q *store.Queries, demoID int64, steamID string) (HabitInputs, bool, error) {
	row, err := q.GetPlayerMatchAnalysis(ctx, store.GetPlayerMatchAnalysisParams{
		DemoID:  demoID,
		SteamID: steamID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return HabitInputs{}, false, nil
	}
	if err != nil {
		return HabitInputs{}, false, fmt.Errorf("get player match analysis: %w", err)
	}

	in := HabitInputs{
		SteamID:            row.SteamID,
		TimeToFireMsAvg:    row.TimeToFireMsAvg,
		FirstShotAccRatio:  row.FirstShotAccPct,
		TradePctRatio:      row.TradePct,
		IsolatedPeekDeaths: int(row.IsolatedPeekDeaths),
		RepeatedDeathZones: int(row.RepeatedDeathZones),
	}

	// First-shot accuracy: a 0% column with no extras hint is indistinguishable
	// from "never sampled". We treat any non-default cell as data; the analyzer
	// only writes this column when it actually computed FirstShotPct, so a true
	// 0 is rare. extras_json.first_shot_fires would be a cleaner signal but is
	// not currently persisted.
	in.HasFirstShotData = row.FirstShotAccPct > 0

	// Pull mech "standing_shot_pct" out of extras_json — that's the
	// fraction-of-standing-fires we need to surface shooting-in-motion. The
	// column of the same name on player_match_analysis is *accuracy*, not motion.
	if row.ExtrasJson != "" && row.ExtrasJson != "{}" {
		var extras map[string]any
		if jsonErr := json.Unmarshal([]byte(row.ExtrasJson), &extras); jsonErr == nil {
			if v, ok := extras["standing_shot_pct"]; ok {
				if f, ok := toFloat64(v); ok {
					in.StandingShotMechRatio = f
					in.HasStandingShotMechData = true
				}
			}
		}
	}

	count, err := q.CountAnalysisMistakesByKindForPlayer(ctx, store.CountAnalysisMistakesByKindForPlayerParams{
		DemoID:  demoID,
		SteamID: steamID,
		Kind:    string(MistakeKindNoTradeDeath),
	})
	if err != nil {
		return HabitInputs{}, false, fmt.Errorf("count untraded deaths: %w", err)
	}
	in.UntradedDeathsCount = int(count)

	return in, true, nil
}

func toFloat64(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}
