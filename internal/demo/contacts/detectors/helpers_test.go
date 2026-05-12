package detectors

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/analysis"
)

// mkTick is a one-line constructor used across detector tests to build
// AnalysisTick fixtures inline.
func mkTick(tick int32, steam uint64, x, y, z, yaw, pitch, vx, vy float32, alive bool, ammo int16) demo.AnalysisTick {
	return demo.AnalysisTick{
		Tick: tick, SteamID: steam,
		X: x, Y: y, Z: z, Yaw: yaw, Pitch: pitch,
		Vx: vx, Vy: vy, IsAlive: alive, AmmoClip: ammo,
	}
}

// mkTickIndex wraps analysis.BuildTickIndex so detector tests don't
// have to import analysis directly for the helper.
func mkTickIndex(rows []demo.AnalysisTick) analysis.PerPlayerTickIndex {
	return analysis.BuildTickIndex(rows)
}

// assertKinds verifies the per-contact mistake list emitted by a
// detector matches the expected kinds, in order.
func assertKinds(t *testing.T, got []ContactMistake, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("kind count: got=%d (%v), want=%d (%v)", len(got), kindsOf(got), len(want), want)
	}
	for i, m := range got {
		if m.Kind != want[i] {
			t.Errorf("kind[%d]: got %q, want %q", i, m.Kind, want[i])
		}
	}
}

// kindsOf returns the kinds in order, used in error messages and
// scenario goldens.
func kindsOf(ms []ContactMistake) []string {
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.Kind)
	}
	return out
}

// analysisKindsOf is the analysis.Mistake equivalent.
func analysisKindsOf(ms []analysis.Mistake) []string {
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.Kind)
	}
	return out
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func containsAll(haystack []string, needles []string) bool {
	for _, n := range needles {
		if !contains(haystack, n) {
			return false
		}
	}
	return true
}

// sameStringSets compares two []string slices ignoring order. Used in
// catalog_test.go to assert V1() returns the expected ten kinds.
func sameStringSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int, len(a))
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
	}
	for _, v := range seen {
		if v != 0 {
			return false
		}
	}
	return true
}

func intPtr(v int32) *int32 {
	return &v
}
