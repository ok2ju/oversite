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
		StandingShotMechRatio:   0.95,
		HasStandingShotMechData: true,
	}

	rows := analysis.BuildHabitReport(in)
	if len(rows) != 6 {
		t.Fatalf("want 6 habits with full inputs, got %d", len(rows))
	}

	idx := indexByKey(rows)
	wantStatus := map[analysis.HabitKey]analysis.Status{
		analysis.HabitReaction:           analysis.StatusGood,
		analysis.HabitFirstShotAcc:       analysis.StatusGood,
		analysis.HabitShootingInMotion:   analysis.StatusGood, // 5% motion
		analysis.HabitTradeTiming:        analysis.StatusWarn, // 55%
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
		IsolatedPeekDeaths: 0,
		RepeatedDeathZones: 0,
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

	// The 3 always-on rows should still be present.
	for _, k := range []analysis.HabitKey{
		analysis.HabitTradeTiming,
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
	if in.IsolatedPeekDeaths != 1 {
		t.Errorf("IsolatedPeekDeaths = %d, want 1", in.IsolatedPeekDeaths)
	}

	rows := analysis.BuildHabitReport(in)
	if len(rows) < 5 {
		t.Errorf("BuildHabitReport returned %d rows, want >= 5 for a fixture demo", len(rows))
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

func TestLoadHabitHistory_ThreeDemos(t *testing.T) {
	t.Parallel()

	q, _ := testutil.NewTestQueries(t)
	ctx := context.Background()

	// Three demos for the same player, ordered chronologically. Insert
	// out-of-order to verify the loader sorts by match_date DESC, not insert
	// order.
	specs := []struct {
		matchDate string
		fileName  string
		fireMs    float64
		firstAcc  float64
	}{
		{"2026-04-10T20:00:00Z", "demo_april.dem", 260, 0.40},
		{"2026-05-01T20:00:00Z", "demo_may.dem", 240, 0.45},
		{"2026-05-09T20:00:00Z", "demo_may_late.dem", 220, 0.50},
	}
	demoIDs := make([]int64, len(specs))
	insertOrder := []int{1, 0, 2} // not chronological
	for _, idx := range insertOrder {
		s := specs[idx]
		d, err := q.CreateDemo(ctx, store.CreateDemoParams{
			MapName:   "de_inferno",
			FilePath:  "/tmp/" + s.fileName,
			FileSize:  1,
			Status:    "ready",
			MatchDate: s.matchDate,
		})
		if err != nil {
			t.Fatalf("CreateDemo: %v", err)
		}
		demoIDs[idx] = d.ID
		if err := q.UpsertPlayerMatchAnalysis(ctx, store.UpsertPlayerMatchAnalysisParams{
			DemoID:          d.ID,
			SteamID:         "alice",
			OverallScore:    50,
			TradePct:        0.6,
			Version:         int64(analysis.AnalysisVersion),
			TimeToFireMsAvg: s.fireMs,
			FirstShotAccPct: s.firstAcc,
			ExtrasJson:      "{}",
		}); err != nil {
			t.Fatalf("UpsertPlayerMatchAnalysis: %v", err)
		}
	}

	history, err := analysis.LoadHabitHistory(ctx, q, "alice", 8)
	if err != nil {
		t.Fatalf("LoadHabitHistory: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("history length = %d, want 3", len(history))
	}
	wantOrderMs := []float64{220, 240, 260}
	for i, want := range wantOrderMs {
		if history[i].TimeToFireMsAvg != want {
			t.Errorf("history[%d].TimeToFireMsAvg = %v, want %v", i, history[i].TimeToFireMsAvg, want)
		}
	}

	// HistoryForKey should preserve order and convert ratios to %.
	pts := analysis.HistoryForKey(history, analysis.HabitFirstShotAcc)
	if len(pts) != 3 {
		t.Fatalf("HistoryForKey: len = %d, want 3", len(pts))
	}
	if pts[0].Value < 49.99 || pts[0].Value > 50.01 {
		t.Errorf("HistoryForKey[0].Value = %v, want 50", pts[0].Value)
	}
	if pts[2].Value < 39.99 || pts[2].Value > 40.01 {
		t.Errorf("HistoryForKey[2].Value = %v, want 40", pts[2].Value)
	}
}

func TestAttachDeltas_SignedAcrossDemos(t *testing.T) {
	t.Parallel()

	// History sorted newest first: D3 (current) → D2 → D1.
	history := []analysis.HistoryRecord{
		{
			DemoID: 3, MatchDate: "2026-05-09T20:00:00Z",
			TimeToFireMsAvg: 220, FirstShotAccRatio: 0.50, HasFirstShotData: true,
		},
		{
			DemoID: 2, MatchDate: "2026-05-01T20:00:00Z",
			TimeToFireMsAvg: 240, FirstShotAccRatio: 0.45, HasFirstShotData: true,
		},
		{
			DemoID: 1, MatchDate: "2026-04-10T20:00:00Z",
			TimeToFireMsAvg: 260, FirstShotAccRatio: 0.40, HasFirstShotData: true,
		},
	}

	rows := analysis.BuildHabitReport(analysis.HabitInputs{
		TimeToFireMsAvg:   220,
		FirstShotAccRatio: 0.50,
		HasFirstShotData:  true,
	})
	analysis.AttachDeltas(rows, history, 3)

	idx := indexByKey(rows)
	reaction, ok := idx[analysis.HabitReaction]
	if !ok {
		t.Fatal("missing reaction row")
	}
	if reaction.PreviousValue == nil || *reaction.PreviousValue != 240 {
		t.Errorf("reaction.PreviousValue = %v, want 240", reaction.PreviousValue)
	}
	if reaction.Delta == nil || *reaction.Delta != -20 {
		t.Errorf("reaction.Delta = %v, want -20 (improved by 20 ms)", reaction.Delta)
	}

	first, ok := idx[analysis.HabitFirstShotAcc]
	if !ok {
		t.Fatal("missing first-shot row")
	}
	if first.PreviousValue == nil {
		t.Fatal("first.PreviousValue should be set")
	}
	if got := *first.PreviousValue; got < 44.99 || got > 45.01 {
		t.Errorf("first.PreviousValue = %v, want 45", got)
	}
	if first.Delta == nil || *first.Delta < 4.99 || *first.Delta > 5.01 {
		t.Errorf("first.Delta = %v, want 5", first.Delta)
	}
}

func TestAttachDeltas_FirstDemoNoPrevious(t *testing.T) {
	t.Parallel()

	// History contains only the current demo — no prior data to compare.
	history := []analysis.HistoryRecord{
		{
			DemoID: 1, MatchDate: "2026-05-09T20:00:00Z",
			TimeToFireMsAvg: 220, FirstShotAccRatio: 0.50, HasFirstShotData: true,
		},
	}
	rows := analysis.BuildHabitReport(analysis.HabitInputs{
		TimeToFireMsAvg:   220,
		FirstShotAccRatio: 0.50,
		HasFirstShotData:  true,
	})
	analysis.AttachDeltas(rows, history, 1)

	for _, r := range rows {
		if r.PreviousValue != nil {
			t.Errorf("%s: PreviousValue should be nil for the first demo, got %v", r.Key, *r.PreviousValue)
		}
		if r.Delta != nil {
			t.Errorf("%s: Delta should be nil for the first demo, got %v", r.Key, *r.Delta)
		}
	}
}

func TestAttachDeltas_SkipsHabitsWithNoPriorData(t *testing.T) {
	t.Parallel()

	// Demo D3 (current) has first-shot data; D2 (prior) does not.
	history := []analysis.HistoryRecord{
		{
			DemoID: 3, FirstShotAccRatio: 0.50, HasFirstShotData: true,
		},
		{
			DemoID: 2, FirstShotAccRatio: 0, HasFirstShotData: false,
		},
	}
	rows := analysis.BuildHabitReport(analysis.HabitInputs{
		FirstShotAccRatio: 0.50,
		HasFirstShotData:  true,
		TradePctRatio:     0.60,
	})
	analysis.AttachDeltas(rows, history, 3)

	idx := indexByKey(rows)
	first := idx[analysis.HabitFirstShotAcc]
	if first.PreviousValue != nil {
		t.Errorf("first.PreviousValue should be nil when prior demo has no data, got %v", *first.PreviousValue)
	}
}

func indexByKey(rows []analysis.HabitRow) map[analysis.HabitKey]analysis.HabitRow {
	out := make(map[analysis.HabitKey]analysis.HabitRow, len(rows))
	for _, r := range rows {
		out[r.Key] = r
	}
	return out
}
