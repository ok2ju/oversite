package demo

// MatchFormat describes the rule set the demo was played under.
// Mirrors the wire-level types.MatchFormat shape — the binding copies fields
// into the root-package type. Keeping the type duplicated here lets the
// aggregator tests stay package-local without importing main.
type MatchFormat struct {
	RegulationRounds   int
	HalftimeRound      int
	OvertimeHalfLen    int
	HasOvertime        bool
	TotalRounds        int
	PistolRoundNumbers []int
}

// RoundSummary is the per-round input shape the format / winner helpers
// expect. Mirrors the round_number + is_overtime columns the aggregator query
// joins through.
type RoundSummary struct {
	Number     int
	IsOvertime bool
}

// DetectMatchFormat infers regulation length, halftime round, and overtime
// half length from the round list. Returns a zero-valued format when rounds
// is empty.
//
// Rules:
//   - Regulation rounds = count of rounds where is_overtime == false. If that
//     exceeds 24 we assume MR15 (30 regulation rounds), otherwise MR12 (24).
//   - Halftime is regulation/2 (12 for MR12, 15 for MR15).
//   - Overtime half length: 3 for MR12, 5 for MR15.
//   - Pistol rounds: round 1 (CT/T opening), round halftime+1 (side-swap opening),
//     then the first round of each overtime half (regulation+1, +otHalf, +otHalf*2, …).
func DetectMatchFormat(rounds []RoundSummary) MatchFormat {
	var f MatchFormat
	if len(rounds) == 0 {
		return f
	}
	regCount := 0
	hasOT := false
	for _, r := range rounds {
		if r.IsOvertime {
			hasOT = true
		} else {
			regCount++
		}
	}
	// Cap regulation by the format ceiling. If the observed count overshoots,
	// fall back to the higher MR15 ceiling so over-counting (rare; malformed
	// demos) does not yield an absurd halftime round.
	switch {
	case regCount >= 30:
		f.RegulationRounds = 30
	case regCount >= 24:
		f.RegulationRounds = 24
	case regCount > 0:
		// Short demos (e.g. surrender / disconnect). Treat as MR12 so the
		// "of N" footer reads sensibly.
		f.RegulationRounds = 24
	default:
		f.RegulationRounds = 24
	}
	f.HalftimeRound = f.RegulationRounds / 2
	if f.RegulationRounds == 30 {
		f.OvertimeHalfLen = 5
	} else {
		f.OvertimeHalfLen = 3
	}
	f.HasOvertime = hasOT
	f.TotalRounds = len(rounds)

	// Pistol-round numbers: round 1 and halftime+1 in regulation, plus the
	// first round of each overtime half thereafter, walking until we cover
	// every played round.
	f.PistolRoundNumbers = []int{1}
	if f.HalftimeRound+1 <= f.TotalRounds {
		f.PistolRoundNumbers = append(f.PistolRoundNumbers, f.HalftimeRound+1)
	}
	otStart := f.RegulationRounds + 1
	for n := otStart; n <= f.TotalRounds; n += f.OvertimeHalfLen {
		f.PistolRoundNumbers = append(f.PistolRoundNumbers, n)
	}
	return f
}

// IsPistolRound returns true when n is a pistol round in the given format.
func (f MatchFormat) IsPistolRound(n int) bool {
	for _, p := range f.PistolRoundNumbers {
		if p == n {
			return true
		}
	}
	return false
}

// WinnerForRound returns "a" if Team A (the T-side starter) won the round,
// otherwise "b". Decides via the per-round t_team_name when it's populated,
// falling back to the round's overtime-aware half map otherwise — never the
// "R<=12 = first half" heuristic.
//
//   - teamAName is the canonical Team A name (typically rounds[0].t_team_name).
//   - halfTeamASide is "T" or "CT" — the side Team A is playing during this
//     round. Computed by HalfTeamASide.
func WinnerForRound(winnerSide, tTeamName, ctTeamName, teamAName, halfTeamASide string) string {
	if teamAName != "" {
		if winnerSide == "T" {
			if tTeamName == teamAName {
				return "a"
			}
			return "b"
		}
		if winnerSide == "CT" {
			if ctTeamName == teamAName {
				return "a"
			}
			return "b"
		}
	}
	if winnerSide == halfTeamASide {
		return "a"
	}
	return "b"
}

// HalfTeamASide returns the side ("T" or "CT") that Team A is playing for the
// given (1-indexed) round number under the supplied format. Team A starts on
// T and swaps each half — regulation halftime + each overtime half flips
// sides.
func HalfTeamASide(roundNumber int, f MatchFormat) string {
	if roundNumber <= 0 {
		return "T"
	}
	if roundNumber <= f.RegulationRounds {
		if roundNumber <= f.HalftimeRound {
			return "T"
		}
		return "CT"
	}
	// Overtime: each half of overtime flips the side relative to the previous.
	// In CS2, the first half of OT1 has Team A starting on T (mirroring the
	// regulation opening), and they swap each overtime half thereafter.
	otRound := roundNumber - f.RegulationRounds - 1 // 0-indexed within OT
	if f.OvertimeHalfLen <= 0 {
		return "T"
	}
	halfIndex := otRound / f.OvertimeHalfLen
	if halfIndex%2 == 0 {
		return "T"
	}
	return "CT"
}
