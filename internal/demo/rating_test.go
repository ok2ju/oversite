package demo

import (
	"math"
	"testing"
)

// approx asserts got is within tol of want — HLTV 2.0 figures published in
// public references round to 2-3 decimal places, so a 1e-3 tolerance is
// strict enough to catch coefficient swaps without choking on rounding.
func approx(t *testing.T, label string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Fatalf("%s: got %.6f, want %.6f (±%g)", label, got, want, tol)
	}
}

func TestComputeImpact(t *testing.T) {
	// Reference: KPR=0.75, APR=0.15 → Impact = 2.13*0.75 + 0.42*0.15 − 0.41
	//   = 1.5975 + 0.063 − 0.41 = 1.2505
	got := ComputeImpact(0.75, 0.15)
	approx(t, "Impact", got, 1.2505, 1e-4)

	// Zero rates → just the constant.
	approx(t, "Impact zero", ComputeImpact(0, 0), -0.41, 1e-9)
}

func TestComputeRating2(t *testing.T) {
	tests := []struct {
		name     string
		k, d, a  int
		kast     float64
		adr      float64
		rounds   int
		expected float64
	}{
		{
			name:   "average player",
			k:      18,
			d:      18,
			a:      4,
			kast:   70,
			adr:    75,
			rounds: 24,
			// kpr=0.75, dpr=0.75, apr=0.1667
			// impact = 2.13*0.75 + 0.42*0.1667 - 0.41 = 1.5975 + 0.07 - 0.41 = 1.2575
			// rating = 0.0073*70 + 0.3591*0.75 - 0.5329*0.75 + 0.2372*1.2575 + 0.0032*75 + 0.1587
			//        = 0.511 + 0.2693 - 0.3997 + 0.2982 + 0.24 + 0.1587
			//        ≈ 1.0776
			expected: 1.0776,
		},
		{
			name:     "zero rounds returns zero",
			k:        5,
			d:        2,
			a:        1,
			kast:     100,
			adr:      100,
			rounds:   0,
			expected: 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeRating2(tc.k, tc.d, tc.a, tc.kast, tc.adr, tc.rounds)
			approx(t, tc.name, got, tc.expected, 0.01)
		})
	}
}

func TestComputeKASTPercent(t *testing.T) {
	approx(t, "20/24", ComputeKASTPercent(20, 24), 83.333, 1e-2)
	approx(t, "0/24", ComputeKASTPercent(0, 24), 0, 1e-9)
	approx(t, "zero rounds", ComputeKASTPercent(5, 0), 0, 1e-9)
	approx(t, "100% full", ComputeKASTPercent(24, 24), 100, 1e-9)
}
