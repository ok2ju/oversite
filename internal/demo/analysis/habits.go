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

	// "standing_shot_pct" from extras_json — the mech-aggregate value
	// (fraction of fires while standing). Present-or-absent flag controls
	// whether the shooting-in-motion habit appears in the report; the
	// PlayerMatchAnalysis column of the same name is *accuracy* of standing
	// shots and does not power this habit.
	StandingShotMechRatio   float64
	HasStandingShotMechData bool

	// Slice 11 micro metrics (P3-2). HasMicroData gates the rows on having
	// at least one fire sampled — a player who never fired produces
	// zero-everywhere rows that the UI would otherwise render as "perfect".
	TimeToStopMsAvg            float64
	CrouchBeforeShotCount      int
	CrouchInsteadOfStrafeCount int
	FlickBalancePct            float64
	HasMicroData               bool
	// FireCount is the engagement gate. We re-use the slice-10 mechanical
	// aggregate "engagements" from extras_json — the analyzer writes it on
	// every successful pass; absent means "no fires this match".
	FireCount int
}

// HabitRow is the in-Go shape of one row in the HabitReport. Mirrors the
// HabitRow struct in types.go (which is the wire-level type tagged for JSON).
// Keeping the in-Go struct typed (HabitKey, Direction, Status enums) catches
// builder-side bugs at compile time; types.go converts to plain strings for
// the binding.
//
// PreviousValue / Delta are filled by AttachDeltas when history is available;
// nil means "no prior demo with data for this habit" and the UI hides the
// delta line.
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
	PreviousValue *float64
	Delta         *float64
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

	if in.HasMicroData && in.TimeToStopMsAvg > 0 {
		add(HabitCounterStrafe, in.TimeToStopMsAvg)
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
	if in.HasMicroData && in.FireCount > 0 {
		// Crouch-before-shot rate: fires where the player was crouched at
		// the moment of fire, normalised by total fires sampled.
		crouchPct := 100.0 * float64(in.CrouchBeforeShotCount) / float64(in.FireCount)
		add(HabitCrouchBeforeShot, crouchPct)
	}
	if in.HasMicroData && in.FlickBalancePct > 0 {
		add(HabitFlickBalance, in.FlickBalancePct)
	}
	add(HabitTradeTiming, in.TradePctRatio*100)
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
		SteamID:                    row.SteamID,
		TimeToFireMsAvg:            row.TimeToFireMsAvg,
		FirstShotAccRatio:          row.FirstShotAccPct,
		TradePctRatio:              row.TradePct,
		IsolatedPeekDeaths:         int(row.IsolatedPeekDeaths),
		RepeatedDeathZones:         int(row.RepeatedDeathZones),
		TimeToStopMsAvg:            row.TimeToStopMsAvg,
		CrouchBeforeShotCount:      int(row.CrouchBeforeShotCount),
		CrouchInsteadOfStrafeCount: int(row.CrouchInsteadOfStrafeCount),
		FlickBalancePct:            row.FlickBalancePct,
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
	// Engagements rides in the same blob; we use it as the "any fires sampled"
	// gate for the slice-11 micro metrics.
	if row.ExtrasJson != "" && row.ExtrasJson != "{}" {
		var extras map[string]any
		if jsonErr := json.Unmarshal([]byte(row.ExtrasJson), &extras); jsonErr == nil {
			if v, ok := extras["standing_shot_pct"]; ok {
				if f, ok := toFloat64(v); ok {
					in.StandingShotMechRatio = f
					in.HasStandingShotMechData = true
				}
			}
			if v, ok := extras["engagements"]; ok {
				if f, ok := toFloat64(v); ok {
					in.FireCount = int(f)
					in.HasMicroData = f > 0
				}
			}
		}
	}

	return in, true, nil
}

// HistoryRecord is one entry in a player's analysis history — the data needed
// to derive any habit's value at the time of that demo. Sorted match_date
// DESC by LoadHabitHistory; consumers should not re-sort.
type HistoryRecord struct {
	DemoID    int64
	MatchDate string

	TimeToFireMsAvg    float64
	FirstShotAccRatio  float64
	HasFirstShotData   bool
	TradePctRatio      float64
	IsolatedPeekDeaths int
	RepeatedDeathZones int

	StandingShotMechRatio   float64
	HasStandingShotMechData bool

	// Slice 11 micro metrics (P3-2). HasMicroData mirrors HabitInputs and
	// gates the rows on having at least one fire sampled in this demo.
	TimeToStopMsAvg       float64
	CrouchBeforeShotCount int
	FlickBalancePct       float64
	FireCount             int
	HasMicroData          bool
}

// HistoryPoint is one (demo, value) pair returned by HistoryForKey — the wire
// shape for GetHabitHistory. MatchDate is passed through verbatim from the
// demos.match_date column so the frontend renders the sparkline x-axis with
// the same string the rest of the app uses.
type HistoryPoint struct {
	DemoID    int64
	MatchDate string
	Value     float64
}

// LoadHabitHistory loads the player's last `limit` analysis rows, newest
// first, with the per-row aggregates BuildHabitReport / AttachDeltas need.
// Reads from player_match_analysis JOIN demos, so legacy demos without an
// analysis row simply don't appear.
func LoadHabitHistory(ctx context.Context, q *store.Queries, steamID string, limit int) ([]HistoryRecord, error) {
	if steamID == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 8
	}
	rows, err := q.ListHabitHistoryForPlayer(ctx, store.ListHabitHistoryForPlayerParams{
		SteamID:  steamID,
		LimitVal: int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list habit history: %w", err)
	}

	out := make([]HistoryRecord, len(rows))
	for i, r := range rows {
		rec := HistoryRecord{
			DemoID:             r.DemoID,
			MatchDate:          r.MatchDate,
			TimeToFireMsAvg:    r.TimeToFireMsAvg,
			FirstShotAccRatio:  r.FirstShotAccPct,
			HasFirstShotData:   r.FirstShotAccPct > 0,
			TradePctRatio:      r.TradePct,
			IsolatedPeekDeaths: int(r.IsolatedPeekDeaths),
			RepeatedDeathZones: int(r.RepeatedDeathZones),

			TimeToStopMsAvg:       r.TimeToStopMsAvg,
			CrouchBeforeShotCount: int(r.CrouchBeforeShotCount),
			FlickBalancePct:       r.FlickBalancePct,
		}
		if r.ExtrasJson != "" && r.ExtrasJson != "{}" {
			var extras map[string]any
			if jsonErr := json.Unmarshal([]byte(r.ExtrasJson), &extras); jsonErr == nil {
				if v, ok := extras["standing_shot_pct"]; ok {
					if f, ok := toFloat64(v); ok {
						rec.StandingShotMechRatio = f
						rec.HasStandingShotMechData = true
					}
				}
				if v, ok := extras["engagements"]; ok {
					if f, ok := toFloat64(v); ok {
						rec.FireCount = int(f)
						rec.HasMicroData = f > 0
					}
				}
			}
		}
		out[i] = rec
	}
	return out, nil
}

// HabitValueFromRecord returns the habit's value at the given history record
// in the same display unit BuildHabitReport produces (% on a 0..100 scale,
// counts as integers, ms as ms). ok=false means the record carries no data
// for this habit (e.g. no first-shot fires sampled) and the caller should
// treat it as "not present" rather than zero.
func HabitValueFromRecord(key HabitKey, rec HistoryRecord) (float64, bool) {
	switch key {
	case HabitCounterStrafe:
		if !rec.HasMicroData || rec.TimeToStopMsAvg <= 0 {
			return 0, false
		}
		return rec.TimeToStopMsAvg, true
	case HabitReaction:
		if rec.TimeToFireMsAvg <= 0 {
			return 0, false
		}
		return rec.TimeToFireMsAvg, true
	case HabitFirstShotAcc:
		if !rec.HasFirstShotData {
			return 0, false
		}
		return rec.FirstShotAccRatio * 100, true
	case HabitShootingInMotion:
		if !rec.HasStandingShotMechData {
			return 0, false
		}
		motion := (1 - rec.StandingShotMechRatio) * 100
		if motion < 0 {
			motion = 0
		}
		if motion > 100 {
			motion = 100
		}
		return motion, true
	case HabitCrouchBeforeShot:
		if !rec.HasMicroData || rec.FireCount <= 0 {
			return 0, false
		}
		return 100.0 * float64(rec.CrouchBeforeShotCount) / float64(rec.FireCount), true
	case HabitFlickBalance:
		if !rec.HasMicroData || rec.FlickBalancePct <= 0 {
			return 0, false
		}
		return rec.FlickBalancePct, true
	case HabitTradeTiming:
		return rec.TradePctRatio * 100, true
	case HabitIsolatedPeekDeaths:
		return float64(rec.IsolatedPeekDeaths), true
	case HabitRepeatedDeathZone:
		return float64(rec.RepeatedDeathZones), true
	}
	return 0, false
}

// HistoryForKey filters records to (demo, value) points where the habit has
// data, preserving the input order (newest first). Used by GetHabitHistory.
func HistoryForKey(records []HistoryRecord, key HabitKey) []HistoryPoint {
	out := make([]HistoryPoint, 0, len(records))
	for _, r := range records {
		v, ok := HabitValueFromRecord(key, r)
		if !ok {
			continue
		}
		out = append(out, HistoryPoint{
			DemoID:    r.DemoID,
			MatchDate: r.MatchDate,
			Value:     v,
		})
	}
	return out
}

// AttachDeltas mutates rows so each row's PreviousValue/Delta point to the
// most recent prior demo that had data for that habit. currentDemoID is the
// demo the rows describe; "previous" is the next-older record in history.
//
// History is expected sorted match_date DESC (LoadHabitHistory's contract).
// If currentDemoID is not in history (e.g. analysis row exists but the demo
// is not yet in the history result for some reason) the rows are left with
// nil deltas — first-demo behaviour, which is the conservative default.
func AttachDeltas(rows []HabitRow, history []HistoryRecord, currentDemoID int64) {
	currentIdx := -1
	for i, r := range history {
		if r.DemoID == currentDemoID {
			currentIdx = i
			break
		}
	}
	if currentIdx == -1 || currentIdx >= len(history)-1 {
		return // first demo or current demo not in history → no previous to compare
	}
	for i := range rows {
		row := &rows[i]
		// Walk forward (older) until we find a demo with data for this habit.
		for j := currentIdx + 1; j < len(history); j++ {
			if v, ok := HabitValueFromRecord(row.Key, history[j]); ok {
				prev := v
				delta := row.Value - v
				row.PreviousValue = &prev
				row.Delta = &delta
				break
			}
		}
	}
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
