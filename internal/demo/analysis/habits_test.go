package analysis_test

import (
	"context"
	"testing"

	"github.com/ok2ju/oversite/internal/demo/analysis"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

func TestBuildHabitReport_FixtureValues(t *testing.T) {
	t.Parallel()

	in := analysis.HabitInputs{
		SteamID:                 "alice",
		TimeToFireMsAvg:         180,
		FirstShotAccRatio:       0.6,
		HasFirstShotData:        true,
		TradePctRatio:           0.55,
		IsolatedPeekDeaths:      1,
		RepeatedDeathZones:      0,
		UntradedDeathsCount:     3,
		StandingShotMechRatio:   0.95,
		HasStandingShotMechData: true,
	}

	rows := analysis.BuildHabitReport(in)
	if len(rows) != 7 {
		t.Fatalf("want 7 habits with full inputs, got %d", len(rows))
	}

	idx := indexByKey(rows)
	wantStatus := map[analysis.HabitKey]analysis.Status{
		analysis.HabitReaction:           analysis.StatusGood,
		analysis.HabitFirstShotAcc:       analysis.StatusGood,
		analysis.HabitShootingInMotion:   analysis.StatusGood, // 5% motion
		analysis.HabitTradeTiming:        analysis.StatusWarn, // 55%
		analysis.HabitUntradedDeaths:     analysis.StatusWarn, // 3
		analysis.HabitIsolatedPeekDeaths: analysis.StatusWarn, // 1
		analysis.HabitRepeatedDeathZone:  analysis.StatusGood, // 0
	}

	for k, want := range wantStatus {
		row, ok := idx[k]
		if !ok {
			t.Errorf("missing habit %q", k)
			continue
		}
		if row.Status != want {
			t.Errorf("%s: status = %s, want %s (value=%v)", k, row.Status, want, row.Value)
		}
	}
}

func TestBuildHabitReport_OmitsMissingMetrics(t *testing.T) {
	t.Parallel()

	// Minimal row: only counts populated. The reaction / first-shot /
	// motion habits should be omitted because we have no data for them.
	rows := analysis.BuildHabitReport(analysis.HabitInputs{
		IsolatedPeekDeaths:  0,
		RepeatedDeathZones:  0,
		UntradedDeathsCount: 0,
	})

	idx := indexByKey(rows)
	if _, ok := idx[analysis.HabitReaction]; ok {
		t.Errorf("reaction habit should be omitted when TimeToFireMsAvg == 0")
	}
	if _, ok := idx[analysis.HabitFirstShotAcc]; ok {
		t.Errorf("first-shot accuracy habit should be omitted without HasFirstShotData")
	}
	if _, ok := idx[analysis.HabitShootingInMotion]; ok {
		t.Errorf("shooting-in-motion habit should be omitted without mech extras")
	}

	// The 4 always-on rows should still be present.
	for _, k := range []analysis.HabitKey{
		analysis.HabitTradeTiming,
		analysis.HabitUntradedDeaths,
		analysis.HabitIsolatedPeekDeaths,
		analysis.HabitRepeatedDeathZone,
	} {
		if _, ok := idx[k]; !ok {
			t.Errorf("habit %q should be always-on", k)
		}
	}
}

func TestBuildHabitReport_RatioConversion(t *testing.T) {
	t.Parallel()

	rows := analysis.BuildHabitReport(analysis.HabitInputs{
		FirstShotAccRatio: 0.42,
		HasFirstShotData:  true,
		TradePctRatio:     0.7,
	})
	idx := indexByKey(rows)

	if got := idx[analysis.HabitFirstShotAcc].Value; got < 41.99 || got > 42.01 {
		t.Errorf("first_shot_acc value = %v, want 42 (0.42 * 100)", got)
	}
	if got := idx[analysis.HabitTradeTiming].Value; got < 69.99 || got > 70.01 {
		t.Errorf("trade_timing value = %v, want 70 (0.7 * 100)", got)
	}
}

func TestLoadHabitInputs_RoundTrip(t *testing.T) {
	t.Parallel()

	q, _ := testutil.NewTestQueries(t)
	ctx := context.Background()

	d, err := q.CreateDemo(ctx, store.CreateDemoParams{
		MapName:  "de_inferno",
		FilePath: "/tmp/habit_inputs.dem",
		FileSize: 1,
		Status:   "ready",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}

	if err := q.UpsertPlayerMatchAnalysis(ctx, store.UpsertPlayerMatchAnalysisParams{
		DemoID:             d.ID,
		SteamID:            "alice",
		OverallScore:       60,
		TradePct:           0.55,
		Version:            int64(analysis.AnalysisVersion),
		TimeToFireMsAvg:    220,
		FirstShotAccPct:    0.42,
		IsolatedPeekDeaths: 1,
		RepeatedDeathZones: 0,
		ExtrasJson:         `{"standing_shot_pct":0.92}`,
	}); err != nil {
		t.Fatalf("UpsertPlayerMatchAnalysis: %v", err)
	}

	// Drop a couple of untraded-death mistakes to verify the count flows
	// through.
	for i := 0; i < 2; i++ {
		if err := q.CreateAnalysisMistake(ctx, store.CreateAnalysisMistakeParams{
			DemoID:      d.ID,
			SteamID:     "alice",
			RoundNumber: int64(i + 1),
			Tick:        int64(1000 + i),
			Kind:        string(analysis.MistakeKindNoTradeDeath),
			Category:    "trade",
			Severity:    2,
			ExtrasJson:  "{}",
		}); err != nil {
			t.Fatalf("CreateAnalysisMistake: %v", err)
		}
	}

	in, ok, err := analysis.LoadHabitInputs(ctx, q, d.ID, "alice")
	if err != nil {
		t.Fatalf("LoadHabitInputs: %v", err)
	}
	if !ok {
		t.Fatalf("LoadHabitInputs: ok = false, want true")
	}

	if in.TimeToFireMsAvg != 220 {
		t.Errorf("TimeToFireMsAvg = %v, want 220", in.TimeToFireMsAvg)
	}
	if in.FirstShotAccRatio != 0.42 {
		t.Errorf("FirstShotAccRatio = %v, want 0.42", in.FirstShotAccRatio)
	}
	if !in.HasFirstShotData {
		t.Errorf("HasFirstShotData = false, want true")
	}
	if !in.HasStandingShotMechData {
		t.Errorf("HasStandingShotMechData = false, want true")
	}
	if in.StandingShotMechRatio < 0.91 || in.StandingShotMechRatio > 0.93 {
		t.Errorf("StandingShotMechRatio = %v, want ~0.92", in.StandingShotMechRatio)
	}
	if in.UntradedDeathsCount != 2 {
		t.Errorf("UntradedDeathsCount = %d, want 2", in.UntradedDeathsCount)
	}
	if in.IsolatedPeekDeaths != 1 {
		t.Errorf("IsolatedPeekDeaths = %d, want 1", in.IsolatedPeekDeaths)
	}

	rows := analysis.BuildHabitReport(in)
	if len(rows) < 6 {
		t.Errorf("BuildHabitReport returned %d rows, want >= 6 for a fixture demo", len(rows))
	}
}

func TestLoadHabitInputs_MissingRow(t *testing.T) {
	t.Parallel()

	q, _ := testutil.NewTestQueries(t)
	ctx := context.Background()

	_, ok, err := analysis.LoadHabitInputs(ctx, q, 9999, "alice")
	if err != nil {
		t.Fatalf("LoadHabitInputs on missing row: %v", err)
	}
	if ok {
		t.Errorf("LoadHabitInputs missing row: ok = true, want false")
	}
}

func indexByKey(rows []analysis.HabitRow) map[analysis.HabitKey]analysis.HabitRow {
	out := make(map[analysis.HabitKey]analysis.HabitRow, len(rows))
	for _, r := range rows {
		out[r.Key] = r
	}
	return out
}
