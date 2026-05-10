package analysis

import "testing"

func TestClassifyHabit(t *testing.T) {
	t.Parallel()

	lowerNorm := Norm{Direction: LowerIsBetter, GoodThreshold: 100, WarnThreshold: 200}
	higherNorm := Norm{Direction: HigherIsBetter, GoodThreshold: 50, WarnThreshold: 35}
	balancedNorm := Norm{Direction: Balanced, GoodMin: 45, GoodMax: 55, WarnMin: 40, WarnMax: 60}

	tests := []struct {
		name  string
		value float64
		norm  Norm
		want  Status
	}{
		// LowerIsBetter — boundary + interior cases for each band.
		{"lower good interior", 50, lowerNorm, StatusGood},
		{"lower good boundary", 100, lowerNorm, StatusGood},
		{"lower warn interior", 150, lowerNorm, StatusWarn},
		{"lower warn boundary", 200, lowerNorm, StatusWarn},
		{"lower bad", 250, lowerNorm, StatusBad},

		// HigherIsBetter.
		{"higher good interior", 80, higherNorm, StatusGood},
		{"higher good boundary", 50, higherNorm, StatusGood},
		{"higher warn interior", 40, higherNorm, StatusWarn},
		{"higher warn boundary", 35, higherNorm, StatusWarn},
		{"higher bad", 10, higherNorm, StatusBad},

		// Balanced.
		{"balanced good centre", 50, balancedNorm, StatusGood},
		{"balanced good lower bound", 45, balancedNorm, StatusGood},
		{"balanced good upper bound", 55, balancedNorm, StatusGood},
		{"balanced warn below good", 42, balancedNorm, StatusWarn},
		{"balanced warn above good", 58, balancedNorm, StatusWarn},
		{"balanced warn lower bound", 40, balancedNorm, StatusWarn},
		{"balanced warn upper bound", 60, balancedNorm, StatusWarn},
		{"balanced bad below", 30, balancedNorm, StatusBad},
		{"balanced bad above", 80, balancedNorm, StatusBad},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyHabit(tt.value, tt.norm); got != tt.want {
				t.Errorf("ClassifyHabit(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestClassifyHabit_UnknownDirection(t *testing.T) {
	t.Parallel()

	got := ClassifyHabit(123, Norm{Direction: Direction("nonsense")})
	if got != StatusBad {
		t.Errorf("unknown direction: got %v, want %v", got, StatusBad)
	}
}

func TestNormCatalog_Coverage(t *testing.T) {
	t.Parallel()

	keys := AllHabitKeys()
	if len(keys) != 11 {
		t.Fatalf("AllHabitKeys: got %d, want 11 (per §6.1)", len(keys))
	}

	for _, k := range keys {
		n, ok := LookupNorm(k)
		if !ok {
			t.Errorf("LookupNorm(%q): missing", k)
			continue
		}
		if n.Label == "" || n.Description == "" {
			t.Errorf("LookupNorm(%q): empty label or description", k)
		}
		switch n.Direction {
		case LowerIsBetter, HigherIsBetter, Balanced:
			// ok
		default:
			t.Errorf("LookupNorm(%q): invalid direction %q", k, n.Direction)
		}
	}
}

func TestNormCatalog_RealWorldValues(t *testing.T) {
	t.Parallel()

	// Smoke test the seeded catalog with values from the §6.1 table —
	// each row should classify the way the plan describes.
	tests := []struct {
		key   HabitKey
		value float64
		want  Status
	}{
		{HabitCounterStrafe, 80, StatusGood},
		{HabitCounterStrafe, 150, StatusWarn},
		{HabitCounterStrafe, 300, StatusBad},
		{HabitReaction, 180, StatusGood},
		{HabitReaction, 250, StatusWarn},
		{HabitReaction, 350, StatusBad},
		{HabitFirstShotAcc, 60, StatusGood},
		{HabitFirstShotAcc, 40, StatusWarn},
		{HabitFirstShotAcc, 20, StatusBad},
		{HabitFlickBalance, 50, StatusGood},
		{HabitFlickBalance, 42, StatusWarn},
		{HabitFlickBalance, 70, StatusBad},
		{HabitUtilityUsed, 80, StatusGood},
		{HabitUtilityUsed, 60, StatusWarn},
		{HabitUtilityUsed, 30, StatusBad},
	}
	for _, tt := range tests {
		t.Run(string(tt.key), func(t *testing.T) {
			n, ok := LookupNorm(tt.key)
			if !ok {
				t.Fatalf("missing norm %q", tt.key)
			}
			if got := ClassifyHabit(tt.value, n); got != tt.want {
				t.Errorf("%s @ %v: got %v, want %v", tt.key, tt.value, got, tt.want)
			}
		})
	}
}
