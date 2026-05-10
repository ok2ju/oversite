package analysis

import (
	"context"
	"fmt"
	"sort"

	"github.com/ok2ju/oversite/internal/store"
)

// CoachingHabitRow is one card on the /coaching landing page. Mirrors HabitRow
// (label, status, norm thresholds) but the value is aggregated across the
// player's last N demos and a per-row Trend powers the sparkline.
type CoachingHabitRow struct {
	HabitRow
	// Trend is the (demo, value) sequence newest-first — the same shape
	// HistoryForKey returns. Hidden by the frontend when len < 2 (no
	// sparkline drawn for a single point).
	Trend []HistoryPoint
}

// MistakeKindCount is one row in the coaching errors strip — a fire-related
// mistake kind plus its total count across the lookback window.
type MistakeKindCount struct {
	Kind  string
	Total int
}

// CoachingReport is the response shape of GetCoachingReport. Habits is the
// 6-card grid; Errors is the taxonomy strip; LatestDemoID + LastDemoAt let the
// CTA link to the most recent demo's analysis page.
type CoachingReport struct {
	SteamID      string
	Lookback     int
	Habits       []CoachingHabitRow
	Errors       []MistakeKindCount
	LatestDemoID int64
	LastDemoAt   string
}

// coachingHabitKeys is the deliberate first-six order surfaced on the
// coaching landing card grid (plan §6.1: the "micro" habits). Card-level
// surfaces never render the remaining match-shape habits.
func coachingHabitKeys() []HabitKey {
	return []HabitKey{
		HabitCounterStrafe,
		HabitReaction,
		HabitFirstShotAcc,
		HabitShootingInMotion,
		HabitCrouchBeforeShot,
		HabitFlickBalance,
	}
}

// BuildCoachingReport aggregates a player's history into the coaching landing
// shape. Pure: no DB. The caller resolves history via LoadHabitHistory and
// errors via LoadMistakeKindCounts (or analogous test fixtures).
//
// Aggregation strategy per habit:
//   - LowerIsBetter (ms): median across history (median is robust to a single
//     bad demo skewing the rolling card).
//   - HigherIsBetter / Balanced (%): mean across history (a few low samples
//     average out cleanly; medians on percentages over-quantize).
//   - Counts: mean (rendered as a per-match average on the card).
//
// History is expected sorted newest-first (LoadHabitHistory's contract). An
// empty history produces an empty report (Habits=nil, Errors=nil) so the
// frontend can render an empty state without try/catching the call.
func BuildCoachingReport(steamID string, lookback int, history []HistoryRecord, errors []MistakeKindCount) CoachingReport {
	out := CoachingReport{
		SteamID:  steamID,
		Lookback: lookback,
		Errors:   errors,
	}
	if len(history) == 0 {
		return out
	}
	// Latest demo is the head of newest-first history.
	out.LatestDemoID = history[0].DemoID
	out.LastDemoAt = history[0].MatchDate

	rows := make([]CoachingHabitRow, 0, len(coachingHabitKeys()))
	for _, key := range coachingHabitKeys() {
		norm, ok := LookupNorm(key)
		if !ok {
			continue
		}
		trend := HistoryForKey(history, key)
		if len(trend) == 0 {
			continue
		}
		value := aggregate(norm, trend)
		row := rowFromNorm(norm, value)
		rows = append(rows, CoachingHabitRow{
			HabitRow: row,
			Trend:    trend,
		})
	}
	out.Habits = rows
	return out
}

// aggregate picks the central-tendency function appropriate for the habit's
// direction: median for LowerIsBetter (ms-scale, outlier-robust); mean for
// HigherIsBetter/Balanced/counts (smooths small samples without over-rounding).
func aggregate(norm Norm, trend []HistoryPoint) float64 {
	if len(trend) == 0 {
		return 0
	}
	if norm.Direction == LowerIsBetter && norm.Unit == "ms" {
		return median(trend)
	}
	return mean(trend)
}

func mean(trend []HistoryPoint) float64 {
	if len(trend) == 0 {
		return 0
	}
	sum := 0.0
	for _, p := range trend {
		sum += p.Value
	}
	return sum / float64(len(trend))
}

func median(trend []HistoryPoint) float64 {
	if len(trend) == 0 {
		return 0
	}
	values := make([]float64, len(trend))
	for i, p := range trend {
		values[i] = p.Value
	}
	sort.Float64s(values)
	n := len(values)
	if n%2 == 1 {
		return values[n/2]
	}
	return (values[n/2-1] + values[n/2]) / 2
}

// LoadMistakeKindCounts reads the coaching errors strip data — mistake counts
// per kind for a player across their last `lookback` analyzed demos. Returns
// only kinds with count > 0; the frontend hides zero rows by definition.
func LoadMistakeKindCounts(ctx context.Context, q *store.Queries, steamID string, lookback int) ([]MistakeKindCount, error) {
	if steamID == "" {
		return nil, nil
	}
	if lookback <= 0 {
		lookback = 8
	}
	rows, err := q.ListMistakeKindCountsForPlayerLookback(ctx, store.ListMistakeKindCountsForPlayerLookbackParams{
		SteamID:  steamID,
		LimitVal: int64(lookback),
	})
	if err != nil {
		return nil, fmt.Errorf("list mistake kind counts: %w", err)
	}
	out := make([]MistakeKindCount, len(rows))
	for i, r := range rows {
		out[i] = MistakeKindCount{Kind: r.Kind, Total: int(r.Total)}
	}
	return out, nil
}
