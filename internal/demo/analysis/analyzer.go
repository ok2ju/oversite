// Package analysis runs post-ingest mechanical-analysis rules over a parsed
// demo and persists the resulting findings ("mistakes") to the
// analysis_mistakes table, alongside per-(demo, player) and
// per-(demo, player, round) summary rows in player_match_analysis and
// player_round_analysis. Each rule lives in mistakes.go; the aggregators in
// this file wire them together.
//
// Slice 1 shipped the no_trade_death rule. Slice 3 adds
// died_with_util_unused. Slice 5 adds RunMatchSummary (per-(demo, player)
// aggregates). Slice 7 adds RunPlayerRoundAnalysis (per-(demo, player, round)
// breakdowns) so the standalone analysis page can render per-round drilldowns.
// Subsequent slices add rules without changing the public surface — Run keeps
// returning the combined []Mistake list, stably ordered by
// (Tick ASC, SteamID ASC) so the persisted order is independent of
// rule-source order.
package analysis

import (
	"math"
	"sort"

	"github.com/ok2ju/oversite/internal/demo"
)

// TradeWindowSeconds mirrors demo.TradeWindowSeconds so the rule keeps parity
// with the trade-kill detection in internal/demo/player_stats.go without
// either package having to import a constants file. If the upstream value
// moves, update both — there's a cross-reference comment over there.
const TradeWindowSeconds = demo.TradeWindowSeconds

// MistakeKind names a finding emitted by an analyzer rule. New rules add a
// new constant here; the string value is what gets persisted into
// analysis_mistakes.kind.
type MistakeKind string

// Known mistake kinds.
const (
	MistakeKindNoTradeDeath       MistakeKind = "no_trade_death"
	MistakeKindDiedWithUtilUnused MistakeKind = "died_with_util_unused"
	MistakeKindCrosshairTooLow    MistakeKind = "crosshair_too_low"
	MistakeKindShotWhileMoving    MistakeKind = "shot_while_moving"
)

// RunOpts carries optional knobs for the analyzer pass. Zero-valued fields
// mean "default" — see each field's doc comment. Callers that don't care
// about any knob can pass `RunOpts{}`.
type RunOpts struct {
	// MinEngagementsForAimCritique gates the aim rule on a minimum number of
	// fires per attacker across the match. <=0 disables the gate (every fire
	// is eligible). Slice 8 defaults to 0 inside Run when unset; the App
	// surfaces 8 via a binding.
	MinEngagementsForAimCritique int
}

// Mistake is one analyzer finding — a single (player, round, tick, kind)
// tuple plus an opaque extras blob carrying rule-specific context. The
// persistence layer marshals Extras to JSON.
type Mistake struct {
	SteamID     string         `json:"steam_id"`
	RoundNumber int            `json:"round_number"`
	Tick        int            `json:"tick"`
	Kind        string         `json:"kind"`
	Extras      map[string]any `json:"extras,omitempty"`
}

// MatchSummaryRow is the per-(demo, player) aggregate persisted to
// player_match_analysis. OverallScore in slice 8 is the equal-weight average
// of three category percentages (trade / aim / standing-shot) rounded to
// 0–100; subsequent slices may rebalance the weights. Downstream readers
// MUST NOT assume any single-category equivalence — treat OverallScore as an
// opaque composite.
//
// Slice 8 also carries the new aim_pct and standing_shot_pct in Extras (no
// schema migration — they ride alongside trade percentages until a third
// metric per category warrants promoting them to columns).
type MatchSummaryRow struct {
	SteamID       string         `json:"steam_id"`
	OverallScore  int            `json:"overall_score"`
	TradePct      float64        `json:"trade_pct"`
	AvgTradeTicks float64        `json:"avg_trade_ticks"`
	Extras        map[string]any `json:"extras,omitempty"`
}

// PlayerRoundRow is the per-(demo, player, round) aggregate persisted to
// player_round_analysis. Slice 7 only ships the trade column; subsequent
// slices add other categories under the same shape — anything that doesn't
// fit a column lives in Extras (mirrors MatchSummaryRow).
//
// A round in which the player had zero eligible own-deaths produces no row
// (absent ↔ nothing to report); the chart renderer fills the gap with a
// flat zero-height bar so the match cadence still reads top-to-bottom.
type PlayerRoundRow struct {
	SteamID     string         `json:"steam_id"`
	RoundNumber int            `json:"round_number"`
	TradePct    float64        `json:"trade_pct"`
	Extras      map[string]any `json:"extras,omitempty"`
}

// Run executes every analyzer rule against the parse result and returns the
// combined list of mistakes, stably ordered by (Tick ASC, SteamID ASC). Rule
// order is therefore not part of the contract — adding a new rule cannot
// silently reshuffle the persisted sequence. result must be the same struct
// produced by the streaming parser; roundMap (round number → DB round ID) is
// currently unused but accepted so future rules that need DB-side
// correlations don't have to change the signature again. Returns (nil, nil)
// on a nil result.
//
// The two tick-driven rules (crosshair_too_low, shot_while_moving) require
// result.AnalysisTicks to be non-nil — produced by the parser when invoked
// with WithTickFanout(true). When the field is nil (legacy callers / fixtures
// that pre-date slice 8) the tick rules are skipped silently and Run returns
// successfully with only the loadout / event-driven mistakes.
func Run(result *demo.ParseResult, roundMap map[int]int64, opts RunOpts) ([]Mistake, error) {
	_ = roundMap // reserved for rules that look up DB round IDs (slice 2+).
	if result == nil {
		return nil, nil
	}
	tickRate := result.Header.TickRate
	if tickRate <= 0 {
		tickRate = 64
	}

	out := make([]Mistake, 0, 16)
	out = append(out, noTradeDeath(result.Events, tickRate)...)
	out = append(out, diedWithUtilUnused(result.Events, result.Rounds)...)
	if len(result.AnalysisTicks) > 0 {
		idx := BuildTickIndex(result.AnalysisTicks)
		out = append(out, crosshairTooLow(result.Events, result.Rounds, idx, opts.MinEngagementsForAimCritique, tickRate)...)
		out = append(out, shotWhileMoving(result.Events, idx, tickRate)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Tick != out[j].Tick {
			return out[i].Tick < out[j].Tick
		}
		return out[i].SteamID < out[j].SteamID
	})
	return out, nil
}

// RunMatchSummary computes the per-(demo, player) aggregate row written to
// player_match_analysis. Slice 8 ships three categories — trade, aim,
// standing-shot — with the latter two riding in Extras under aim_pct /
// standing_shot_pct (no schema migration; promoted to columns when a third
// metric per category arrives). Players with zero trade-eligible deaths AND
// zero fires are absent from the returned slice (no row is persisted for
// spectators / unrostered slots).
//
// Rows are returned sorted by SteamID ASC for stable persistence and
// test-friendly comparisons.
func RunMatchSummary(result *demo.ParseResult, roundMap map[int]int64, opts RunOpts) ([]MatchSummaryRow, error) {
	_ = roundMap // reserved for rules that look up DB round IDs.
	if result == nil {
		return nil, nil
	}
	tickRate := result.Header.TickRate
	if tickRate <= 0 {
		tickRate = 64
	}

	trades := computeTradesSummary(result.Events, result.Rounds, tickRate)

	var idx PerPlayerTickIndex
	var mechMistakes []Mistake
	if len(result.AnalysisTicks) > 0 {
		idx = BuildTickIndex(result.AnalysisTicks)
		mechMistakes = append(mechMistakes, crosshairTooLow(result.Events, result.Rounds, idx, opts.MinEngagementsForAimCritique, tickRate)...)
		mechMistakes = append(mechMistakes, shotWhileMoving(result.Events, idx, tickRate)...)
	}
	mechAggs := computeMechanicalAggregates(result.Events, idx, mechMistakes)

	// Union of players seen in either trades or fires — a player who only
	// fired (no eligible deaths) still gets a row so their aim/standing shot
	// percentages persist; symmetrically, a player who only died (no fires)
	// keeps their trade row.
	steamIDs := make(map[string]struct{}, len(trades)+len(mechAggs))
	for steamID := range trades {
		steamIDs[steamID] = struct{}{}
	}
	for steamID := range mechAggs {
		steamIDs[steamID] = struct{}{}
	}
	if len(steamIDs) == 0 {
		return nil, nil
	}

	out := make([]MatchSummaryRow, 0, len(steamIDs))
	for steamID := range steamIDs {
		ts := trades[steamID]
		mech := mechAggs[steamID]
		row := MatchSummaryRow{
			SteamID:       steamID,
			OverallScore:  composeOverallScore(ts, mech),
			TradePct:      ts.TradePct,
			AvgTradeTicks: ts.AvgTradeTicks,
		}
		if mech.Engagements > 0 {
			row.Extras = map[string]any{
				"aim_pct":           mech.AimPct,
				"standing_shot_pct": mech.StandingShotPct,
				"engagements":       mech.Engagements,
				"avg_fire_speed":    mech.AvgFireSpeed,
			}
		}
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].SteamID < out[j].SteamID
	})
	return out, nil
}

// RunPlayerRoundAnalysis computes one row per (player, round) where the
// player had at least one eligible own-death OR at least one fire. Slice 8
// fills TradePct as a column and rides aim_pct / standing_shot_pct in Extras.
// Players / rounds with zero of both are absent from the returned slice (no
// row is persisted — mirrors RunMatchSummary's contract).
//
// Rows are returned sorted by (SteamID ASC, RoundNumber ASC) for stable
// persistence and test-friendly comparisons.
func RunPlayerRoundAnalysis(result *demo.ParseResult, roundMap map[int]int64, opts RunOpts) ([]PlayerRoundRow, error) {
	_ = roundMap // reserved for rules that look up DB round IDs.
	if result == nil {
		return nil, nil
	}
	tickRate := result.Header.TickRate
	if tickRate <= 0 {
		tickRate = 64
	}

	roundTrades := computeRoundTrades(result.Events, result.Rounds, tickRate)

	var idx PerPlayerTickIndex
	var mechMistakes []Mistake
	if len(result.AnalysisTicks) > 0 {
		idx = BuildTickIndex(result.AnalysisTicks)
		mechMistakes = append(mechMistakes, crosshairTooLow(result.Events, result.Rounds, idx, opts.MinEngagementsForAimCritique, tickRate)...)
		mechMistakes = append(mechMistakes, shotWhileMoving(result.Events, idx, tickRate)...)
	}
	roundMech := computeRoundMechanicalAggregates(result.Events, idx, mechMistakes)

	// Union of (player, round) keys seen in trades or fires.
	type key struct {
		steam string
		round int
	}
	seen := make(map[key]struct{}, 64)
	for steamID, byRound := range roundTrades {
		for round := range byRound {
			seen[key{steam: steamID, round: round}] = struct{}{}
		}
	}
	for steamID, byRound := range roundMech {
		for round := range byRound {
			seen[key{steam: steamID, round: round}] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil, nil
	}

	out := make([]PlayerRoundRow, 0, len(seen))
	for k := range seen {
		ts := roundTrades[k.steam][k.round]
		mech := roundMech[k.steam][k.round]
		row := PlayerRoundRow{
			SteamID:     k.steam,
			RoundNumber: k.round,
			TradePct:    ts.TradePct,
		}
		if mech.Engagements > 0 {
			row.Extras = map[string]any{
				"aim_pct":           mech.AimPct,
				"standing_shot_pct": mech.StandingShotPct,
				"engagements":       mech.Engagements,
				"avg_fire_speed":    mech.AvgFireSpeed,
			}
		}
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SteamID != out[j].SteamID {
			return out[i].SteamID < out[j].SteamID
		}
		return out[i].RoundNumber < out[j].RoundNumber
	})
	return out, nil
}

// composeOverallScore maps the per-player category percentages into a 0–100
// integer score. Slice 8 weights three categories equally
// (round((tradePct + aimPct + standingShotPct) * 100 / 3)); later slices
// rebalance. When a category has no data (e.g. the player never fired, so
// MechanicalAgg.Engagements is 0), it contributes 0 to the average — better
// than dropping it (which would let a single dominant category swing the
// score wildly when others are absent). Downstream readers must treat the
// value as an opaque composite, not the raw trade percentage.
func composeOverallScore(s TradesSummary, m MechanicalAgg) int {
	tradePct := s.TradePct
	aimPct := m.AimPct
	standPct := m.StandingShotPct
	if m.Engagements == 0 {
		aimPct = 0
		standPct = 0
	}
	score := int(math.Round((tradePct + aimPct + standPct) * 100 / 3))
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}
