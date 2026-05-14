package demo

import (
	"reflect"
	"testing"
)

func TestDetectMatchFormat(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := DetectMatchFormat(nil)
		if got.RegulationRounds != 0 || got.TotalRounds != 0 {
			t.Fatalf("empty: got %+v", got)
		}
	})

	t.Run("MR12 regulation", func(t *testing.T) {
		rounds := make([]RoundSummary, 16)
		for i := range rounds {
			rounds[i] = RoundSummary{Number: i + 1, IsOvertime: false}
		}
		got := DetectMatchFormat(rounds)
		if got.RegulationRounds != 24 {
			t.Errorf("regulation: got %d, want 24", got.RegulationRounds)
		}
		if got.HalftimeRound != 12 {
			t.Errorf("halftime: got %d, want 12", got.HalftimeRound)
		}
		if got.OvertimeHalfLen != 3 {
			t.Errorf("otHalfLen: got %d, want 3", got.OvertimeHalfLen)
		}
		if got.HasOvertime {
			t.Errorf("hasOvertime: got true, want false")
		}
		if got.TotalRounds != 16 {
			t.Errorf("totalRounds: got %d, want 16", got.TotalRounds)
		}
		wantPistols := []int{1, 13}
		if !reflect.DeepEqual(got.PistolRoundNumbers, wantPistols) {
			t.Errorf("pistols: got %v, want %v", got.PistolRoundNumbers, wantPistols)
		}
	})

	t.Run("MR12 + OT1 first half", func(t *testing.T) {
		rounds := make([]RoundSummary, 27)
		for i := range rounds {
			rounds[i] = RoundSummary{Number: i + 1, IsOvertime: i >= 24}
		}
		got := DetectMatchFormat(rounds)
		if !got.HasOvertime {
			t.Fatal("hasOvertime: false")
		}
		if got.OvertimeHalfLen != 3 {
			t.Errorf("otHalfLen: got %d, want 3", got.OvertimeHalfLen)
		}
		// Pistols at 1, 13, 25 (first round of OT1 first half).
		want := []int{1, 13, 25}
		if !reflect.DeepEqual(got.PistolRoundNumbers, want) {
			t.Errorf("pistols: got %v, want %v", got.PistolRoundNumbers, want)
		}
	})

	t.Run("MR12 + OT1 both halves", func(t *testing.T) {
		rounds := make([]RoundSummary, 30)
		for i := range rounds {
			rounds[i] = RoundSummary{Number: i + 1, IsOvertime: i >= 24}
		}
		got := DetectMatchFormat(rounds)
		// Pistols at 1, 13, 25, 28.
		want := []int{1, 13, 25, 28}
		if !reflect.DeepEqual(got.PistolRoundNumbers, want) {
			t.Errorf("pistols: got %v, want %v", got.PistolRoundNumbers, want)
		}
	})

	t.Run("MR15 regulation", func(t *testing.T) {
		rounds := make([]RoundSummary, 30)
		for i := range rounds {
			rounds[i] = RoundSummary{Number: i + 1, IsOvertime: false}
		}
		got := DetectMatchFormat(rounds)
		if got.RegulationRounds != 30 {
			t.Errorf("regulation: got %d, want 30", got.RegulationRounds)
		}
		if got.HalftimeRound != 15 {
			t.Errorf("halftime: got %d, want 15", got.HalftimeRound)
		}
		if got.OvertimeHalfLen != 5 {
			t.Errorf("otHalfLen: got %d, want 5", got.OvertimeHalfLen)
		}
		want := []int{1, 16}
		if !reflect.DeepEqual(got.PistolRoundNumbers, want) {
			t.Errorf("pistols: got %v, want %v", got.PistolRoundNumbers, want)
		}
	})
}

func TestHalfTeamASide(t *testing.T) {
	mf := MatchFormat{
		RegulationRounds: 24,
		HalftimeRound:    12,
		OvertimeHalfLen:  3,
		HasOvertime:      true,
		TotalRounds:      30,
	}
	tests := []struct {
		round int
		want  string
	}{
		{1, "T"},
		{12, "T"},
		{13, "CT"},
		{24, "CT"},
		{25, "T"}, // OT1 first half — Team A back on T
		{27, "T"},
		{28, "CT"}, // OT1 second half
		{30, "CT"},
	}
	for _, tc := range tests {
		got := HalfTeamASide(tc.round, mf)
		if got != tc.want {
			t.Errorf("round %d: got %s, want %s", tc.round, got, tc.want)
		}
	}
}

func TestWinnerForRound(t *testing.T) {
	// With a clan name present: round on T won by team with t_team_name=teamAName → "a".
	t.Run("clan names populated", func(t *testing.T) {
		got := WinnerForRound("T", "Astralis", "NaVi", "Astralis", "T")
		if got != "a" {
			t.Errorf("Astralis T-win: got %s, want a", got)
		}
		got = WinnerForRound("CT", "NaVi", "Astralis", "Astralis", "CT")
		if got != "a" {
			t.Errorf("Astralis CT-win after swap: got %s, want a", got)
		}
		got = WinnerForRound("T", "NaVi", "Astralis", "Astralis", "CT")
		if got != "b" {
			t.Errorf("NaVi T-win: got %s, want b", got)
		}
	})

	// No clan names: rely on halfTeamASide.
	t.Run("fallback no clan names", func(t *testing.T) {
		// 1st half, A on T side, A wins on T → "a"
		got := WinnerForRound("T", "", "", "", "T")
		if got != "a" {
			t.Errorf("A wins 1st half T: got %s, want a", got)
		}
		// 2nd half, A on CT side, A wins on CT → "a"
		got = WinnerForRound("CT", "", "", "", "CT")
		if got != "a" {
			t.Errorf("A wins 2nd half CT: got %s, want a", got)
		}
		// 2nd half, A on CT, T wins → "b"
		got = WinnerForRound("T", "", "", "", "CT")
		if got != "b" {
			t.Errorf("B wins 2nd half T: got %s, want b", got)
		}
	})
}
