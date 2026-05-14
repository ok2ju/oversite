package demo

// HLTV 2.0 rating (public approximation — Mathieu Garstecki / Dust2 spec):
//
//	rating2 = 0.0073 * KAST
//	        + 0.3591 * KPR
//	        - 0.5329 * DPR
//	        + 0.2372 * Impact
//	        + 0.0032 * ADR
//	        + 0.1587
//
//	Impact  = 2.13 * KPR + 0.42 * APR − 0.41
//
// KAST is a 0–100 percentage; KPR / DPR / APR are per-round rates; ADR is the
// per-round average damage. Returns 0 when roundsPlayed is zero.
const (
	ratingCoefKAST   = 0.0073
	ratingCoefKPR    = 0.3591
	ratingCoefDPR    = -0.5329
	ratingCoefImpact = 0.2372
	ratingCoefADR    = 0.0032
	ratingConstant   = 0.1587
	impactCoefKPR    = 2.13
	impactCoefAPR    = 0.42
	impactConstant   = -0.41
)

// ComputeImpact returns the HLTV 2.0 impact term.
func ComputeImpact(kpr, apr float64) float64 {
	return impactCoefKPR*kpr + impactCoefAPR*apr + impactConstant
}

// ComputeRating2 returns the HLTV 2.0 rating for a player given their match
// totals. roundsPlayed must be > 0 — callers should fall back to zero when the
// player participated in zero rounds.
func ComputeRating2(kills, deaths, assists int, kastPercent, adr float64, roundsPlayed int) float64 {
	if roundsPlayed <= 0 {
		return 0
	}
	rp := float64(roundsPlayed)
	kpr := float64(kills) / rp
	dpr := float64(deaths) / rp
	apr := float64(assists) / rp
	impact := ComputeImpact(kpr, apr)
	return ratingCoefKAST*kastPercent +
		ratingCoefKPR*kpr +
		ratingCoefDPR*dpr +
		ratingCoefImpact*impact +
		ratingCoefADR*adr +
		ratingConstant
}

// ComputeKASTPercent returns the KAST percent (0–100) from per-round bits and
// the rounds-played denominator. kastRounds is the count of rounds where the
// player had a kill, assist, survived, or was traded.
func ComputeKASTPercent(kastRounds, roundsPlayed int) float64 {
	if roundsPlayed <= 0 {
		return 0
	}
	return float64(kastRounds) / float64(roundsPlayed) * 100
}
