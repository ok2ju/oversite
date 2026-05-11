package analysis_test

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/analysis"
)

// asFloat64Slice converts an extras_json slice (any) back to []float64. The
// extras blob is built as map[string]any so number slices arrive as []float64
// directly when constructed by the analyzer (no JSON round-trip in unit
// tests). The helper guards against accidental shape drift — a future change
// to int64 here would surface a clear test failure rather than a silent
// frontend regression.
func asFloat64Slice(t *testing.T, v any) []float64 {
	t.Helper()
	s, ok := v.([]float64)
	if !ok {
		t.Fatalf("extras value type = %T, want []float64", v)
	}
	return s
}

func asIntSlice(t *testing.T, v any) []int {
	t.Helper()
	s, ok := v.([]int)
	if !ok {
		t.Fatalf("extras value type = %T, want []int", v)
	}
	return s
}

// TestEnrichFireMistakes_ShotBeforeStop covers the canonical "tried to stop
// but mistimed it" case: prior sample at full running speed, fire sample
// decelerated but still above the cap.
func TestEnrichFireMistakes_ShotBeforeStop(t *testing.T) {
	// Attacker decelerates from 200 → 80 → 60 over three sampled ticks, fires
	// at tick 100 still moving above the 40 u/s cap.
	idx := analysis.BuildTickIndex([]demo.AnalysisTick{
		{Tick: 88, SteamID: u64("100"), Vx: 200, Vy: 0, IsAlive: true},
		{Tick: 92, SteamID: u64("100"), Vx: 80, Vy: 0, IsAlive: true},
		{Tick: 96, SteamID: u64("100"), Vx: 60, Vy: 0, IsAlive: true},
	})
	mistakes := []analysis.Mistake{{
		SteamID:     "100",
		RoundNumber: 1,
		Tick:        100,
		Kind:        string(analysis.MistakeKindShotWhileMoving),
		Extras:      map[string]any{"weapon": "ak47", "speed": 60.0},
	}}

	got := analysis.EnrichFireMistakes(mistakes, idx, nil)

	if got[0].Extras["cause_tag"] != string(analysis.CauseTagShotBeforeStop) {
		t.Errorf("cause_tag = %v, want shot_before_stop", got[0].Extras["cause_tag"])
	}
	speeds := asFloat64Slice(t, got[0].Extras["speeds"])
	if len(speeds) != 3 {
		t.Fatalf("speeds len = %d, want 3", len(speeds))
	}
	if speeds[0] != 200 || speeds[1] != 80 || speeds[2] != 60 {
		t.Errorf("speeds = %v, want [200 80 60]", speeds)
	}
	if got[0].Extras["weapon_speed_cap"] != 40.0 {
		t.Errorf("weapon_speed_cap = %v, want 40", got[0].Extras["weapon_speed_cap"])
	}
	ticks := asIntSlice(t, got[0].Extras["ticks_window"])
	if len(ticks) != 3 || ticks[0] != 88 || ticks[2] != 96 {
		t.Errorf("ticks_window = %v, want [88 92 96]", ticks)
	}
}

// TestEnrichFireMistakes_NoCounterStrafe covers "fired at full speed without
// any deceleration in the window".
func TestEnrichFireMistakes_NoCounterStrafe(t *testing.T) {
	idx := analysis.BuildTickIndex([]demo.AnalysisTick{
		{Tick: 88, SteamID: u64("100"), Vx: 150, Vy: 0, IsAlive: true},
		{Tick: 92, SteamID: u64("100"), Vx: 150, Vy: 0, IsAlive: true},
		{Tick: 96, SteamID: u64("100"), Vx: 150, Vy: 0, IsAlive: true},
	})
	mistakes := []analysis.Mistake{{
		SteamID:     "100",
		RoundNumber: 1,
		Tick:        100,
		Kind:        string(analysis.MistakeKindNoCounterStrafe),
		Extras:      map[string]any{"weapon": "ak47"},
	}}

	got := analysis.EnrichFireMistakes(mistakes, idx, nil)

	if got[0].Extras["cause_tag"] != string(analysis.CauseTagNoCounterStrafe) {
		t.Errorf("cause_tag = %v, want no_counter_strafe", got[0].Extras["cause_tag"])
	}
}

// TestEnrichFireMistakes_CrouchBeforeShot covers the crouch-flag branch. The
// crouch field defaults to false on demos parsed before P3-1, so this case
// only fires when the parser actively reports the crouched state.
func TestEnrichFireMistakes_CrouchBeforeShot(t *testing.T) {
	idx := analysis.BuildTickIndex([]demo.AnalysisTick{
		{Tick: 96, SteamID: u64("100"), Vx: 0, Vy: 0, Crouch: true, IsAlive: true},
	})
	mistakes := []analysis.Mistake{{
		SteamID:     "100",
		RoundNumber: 1,
		Tick:        100,
		Kind:        string(analysis.MistakeKindMissedFirstShot),
		Extras:      map[string]any{"weapon": "ak47"},
	}}

	got := analysis.EnrichFireMistakes(mistakes, idx, nil)

	if got[0].Extras["cause_tag"] != string(analysis.CauseTagCrouchBeforeShot) {
		t.Errorf("cause_tag = %v, want crouch_before_shot", got[0].Extras["cause_tag"])
	}
}

// TestEnrichFireMistakes_LateReaction confirms the slow_reaction kind always
// receives the late_reaction tag when it survives the speed-cap check.
func TestEnrichFireMistakes_LateReaction(t *testing.T) {
	idx := analysis.BuildTickIndex([]demo.AnalysisTick{
		{Tick: 96, SteamID: u64("100"), Vx: 0, Vy: 0, IsAlive: true},
	})
	mistakes := []analysis.Mistake{{
		SteamID:     "100",
		RoundNumber: 1,
		Tick:        100,
		Kind:        string(analysis.MistakeKindSlowReaction),
		Extras:      map[string]any{"reaction_ms": 500.0},
	}}

	got := analysis.EnrichFireMistakes(mistakes, idx, nil)

	if got[0].Extras["cause_tag"] != string(analysis.CauseTagLateReaction) {
		t.Errorf("cause_tag = %v, want late_reaction", got[0].Extras["cause_tag"])
	}
}

// TestEnrichFireMistakes_NonFireKindUnchanged confirms the enrichment leaves
// non-fire mistakes alone — an eco misbuy has no firing window.
func TestEnrichFireMistakes_NonFireKindUnchanged(t *testing.T) {
	idx := analysis.BuildTickIndex([]demo.AnalysisTick{
		{Tick: 96, SteamID: u64("100"), Vx: 100, Vy: 0, IsAlive: true},
	})
	mistakes := []analysis.Mistake{{
		SteamID:     "100",
		RoundNumber: 1,
		Tick:        100,
		Kind:        string(analysis.MistakeKindEcoMisbuy),
		Extras:      map[string]any{},
	}}

	got := analysis.EnrichFireMistakes(mistakes, idx, nil)

	if _, set := got[0].Extras["cause_tag"]; set {
		t.Errorf("cause_tag should not be set on eco_misbuy, got %v", got[0].Extras)
	}
	if _, set := got[0].Extras["speeds"]; set {
		t.Errorf("speeds should not be set on eco_misbuy")
	}
}

// TestEnrichFireMistakes_NoTickRowsLeavesExtras covers the degrade-gracefully
// case: a mistake whose attacker has no sample in the index keeps its rule-
// emitted extras unchanged so the panel still renders.
func TestEnrichFireMistakes_NoTickRowsLeavesExtras(t *testing.T) {
	idx := analysis.BuildTickIndex([]demo.AnalysisTick{
		{Tick: 96, SteamID: u64("999"), Vx: 0, Vy: 0, IsAlive: true},
	})
	mistakes := []analysis.Mistake{{
		SteamID:     "100",
		RoundNumber: 1,
		Tick:        100,
		Kind:        string(analysis.MistakeKindMissedFirstShot),
		Extras:      map[string]any{"weapon": "ak47"},
	}}

	got := analysis.EnrichFireMistakes(mistakes, idx, nil)

	if _, set := got[0].Extras["speeds"]; set {
		t.Errorf("speeds should not be set when attacker has no sample")
	}
	if got[0].Extras["weapon"] != "ak47" {
		t.Errorf("rule-emitted weapon clobbered: %v", got[0].Extras["weapon"])
	}
}

func TestIsFireRelatedMistake(t *testing.T) {
	cases := []struct {
		kind string
		want bool
	}{
		{string(analysis.MistakeKindMissedFirstShot), true},
		{string(analysis.MistakeKindShotWhileMoving), true},
		{string(analysis.MistakeKindSlowReaction), true},
		{string(analysis.MistakeKindNoCounterStrafe), true},
		{string(analysis.MistakeKindEcoMisbuy), false},
		{"unknown_kind", false},
	}
	for _, tc := range cases {
		if got := analysis.IsFireRelatedMistake(tc.kind); got != tc.want {
			t.Errorf("IsFireRelatedMistake(%q) = %v, want %v", tc.kind, got, tc.want)
		}
	}
}
