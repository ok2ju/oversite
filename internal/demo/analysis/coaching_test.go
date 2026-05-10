package analysis_test

import (
	"math"
	"testing"

	"github.com/ok2ju/oversite/internal/demo/analysis"
)

// TestBuildCoachingReport_HandComputed verifies the aggregation strategy
// against three demos with hand-rolled values. Reaction (ms, lower-is-better)
// uses median; percentages use mean. Trend ordering is preserved newest-first.
func TestBuildCoachingReport_HandComputed(t *testing.T) {
	t.Parallel()

	history := []analysis.HistoryRecord{
		{
			DemoID:                  3,
			MatchDate:               "2026-05-01T00:00:00Z",
			TimeToFireMsAvg:         180,
			FirstShotAccRatio:       0.50,
			HasFirstShotData:        true,
			StandingShotMechRatio:   0.90,
			HasStandingShotMechData: true,
			TimeToStopMsAvg:         120,
			CrouchBeforeShotCount:   0,
			FlickBalancePct:         50,
			FireCount:               40,
			HasMicroData:            true,
		},
		{
			DemoID:                  2,
			MatchDate:               "2026-04-30T00:00:00Z",
			TimeToFireMsAvg:         220,
			FirstShotAccRatio:       0.40,
			HasFirstShotData:        true,
			StandingShotMechRatio:   0.85,
			HasStandingShotMechData: true,
			TimeToStopMsAvg:         150,
			CrouchBeforeShotCount:   2,
			FlickBalancePct:         55,
			FireCount:               20,
			HasMicroData:            true,
		},
		{
			DemoID:                  1,
			MatchDate:               "2026-04-29T00:00:00Z",
			TimeToFireMsAvg:         260,
			FirstShotAccRatio:       0.45,
			HasFirstShotData:        true,
			StandingShotMechRatio:   0.80,
			HasStandingShotMechData: true,
			TimeToStopMsAvg:         180,
			CrouchBeforeShotCount:   1,
			FlickBalancePct:         60,
			FireCount:               10,
			HasMicroData:            true,
		},
	}

	report := analysis.BuildCoachingReport(
		"alice",
		10,
		history,
		[]analysis.MistakeKindCount{{Kind: "missed_first_shot", Total: 5}},
	)

	if report.SteamID != "alice" {
		t.Errorf("SteamID = %q, want alice", report.SteamID)
	}
	if report.LatestDemoID != 3 {
		t.Errorf("LatestDemoID = %d, want 3 (newest)", report.LatestDemoID)
	}
	if report.LastDemoAt != "2026-05-01T00:00:00Z" {
		t.Errorf("LastDemoAt = %q, want newest match_date", report.LastDemoAt)
	}
	if len(report.Habits) != 6 {
		t.Fatalf("Habits = %d, want 6 micro cards", len(report.Habits))
	}
	if len(report.Errors) != 1 || report.Errors[0].Kind != "missed_first_shot" {
		t.Errorf("Errors not propagated: %+v", report.Errors)
	}

	idx := make(map[analysis.HabitKey]analysis.CoachingHabitRow, len(report.Habits))
	for _, r := range report.Habits {
		idx[r.Key] = r
	}

	// Reaction is ms / lower-is-better → median(180, 220, 260) = 220.
	if got := idx[analysis.HabitReaction].Value; !approxEq(got, 220) {
		t.Errorf("Reaction value = %v, want 220 (median)", got)
	}

	// Counter-strafe is also ms / lower-is-better → median(120, 150, 180) = 150.
	if got := idx[analysis.HabitCounterStrafe].Value; !approxEq(got, 150) {
		t.Errorf("CounterStrafe value = %v, want 150 (median)", got)
	}

	// First-shot acc is % / higher-is-better → mean(50, 40, 45) = 45.
	if got := idx[analysis.HabitFirstShotAcc].Value; !approxEq(got, 45) {
		t.Errorf("FirstShotAcc value = %v, want 45 (mean)", got)
	}

	// Shooting-in-motion is (1 - standing) * 100 → mean(10, 15, 20) = 15.
	if got := idx[analysis.HabitShootingInMotion].Value; !approxEq(got, 15) {
		t.Errorf("ShootingInMotion value = %v, want 15 (mean)", got)
	}

	// Crouch-before-shot rate per demo: 0/40, 2/20, 1/10 → 0%, 10%, 10% →
	// mean = 6.667%.
	if got := idx[analysis.HabitCrouchBeforeShot].Value; !approxEq(got, 20.0/3.0) {
		t.Errorf("CrouchBeforeShot value = %v, want 6.667 (mean)", got)
	}

	// Flick balance is % / balanced → mean(50, 55, 60) = 55.
	if got := idx[analysis.HabitFlickBalance].Value; !approxEq(got, 55) {
		t.Errorf("FlickBalance value = %v, want 55 (mean)", got)
	}

	// Trend should preserve newest-first and have 3 points for each habit.
	for _, r := range report.Habits {
		if len(r.Trend) != 3 {
			t.Errorf("%s trend len = %d, want 3", r.Key, len(r.Trend))
		}
		if r.Trend[0].DemoID != 3 {
			t.Errorf("%s trend not newest-first: head DemoID = %d", r.Key, r.Trend[0].DemoID)
		}
	}
}

// TestBuildCoachingReport_EmptyHistory ensures the empty-data path returns a
// zero-shaped report rather than erroring — the landing page renders an empty
// state when this happens.
func TestBuildCoachingReport_EmptyHistory(t *testing.T) {
	t.Parallel()

	report := analysis.BuildCoachingReport("alice", 10, nil, nil)
	if len(report.Habits) != 0 {
		t.Errorf("Habits = %d, want 0 for empty history", len(report.Habits))
	}
	if report.LatestDemoID != 0 {
		t.Errorf("LatestDemoID = %d, want 0 sentinel", report.LatestDemoID)
	}
	if report.LastDemoAt != "" {
		t.Errorf("LastDemoAt = %q, want empty", report.LastDemoAt)
	}
}

// TestBuildCoachingReport_OmitsHabitWithNoData verifies a card is omitted
// (rather than rendered as a "perfect" 0) when no demo carries data for it.
func TestBuildCoachingReport_OmitsHabitWithNoData(t *testing.T) {
	t.Parallel()

	history := []analysis.HistoryRecord{
		{
			DemoID:           1,
			MatchDate:        "2026-04-29T00:00:00Z",
			TimeToFireMsAvg:  220,
			HasFirstShotData: false, // no first-shot data
		},
	}
	report := analysis.BuildCoachingReport("alice", 5, history, nil)
	for _, r := range report.Habits {
		if r.Key == analysis.HabitFirstShotAcc {
			t.Errorf("FirstShotAcc card present despite no data: %+v", r)
		}
	}
}

func approxEq(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}
