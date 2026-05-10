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
)

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
// player_match_analysis. OverallScore in slice 5 is the trade percentage
// rounded to an integer (0–100); slices 7–8 will fold additional categories
// into the composite formula. Downstream readers MUST NOT assume
// OverallScore == round(TradePct * 100) long-term — treat it as an opaque
// composite once the schema settles.
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
func Run(result *demo.ParseResult, roundMap map[int]int64) ([]Mistake, error) {
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
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Tick != out[j].Tick {
			return out[i].Tick < out[j].Tick
		}
		return out[i].SteamID < out[j].SteamID
	})
	return out, nil
}

// RunMatchSummary computes the per-(demo, player) aggregate row written to
// player_match_analysis. Currently only the trade category is populated;
// slice 7+ adds utility / aim / movement / positioning / economy under the
// same shape. Players with zero trade-eligible deaths are absent from the
// returned slice (no row is persisted for spectators / unrostered slots).
//
// Rows are returned sorted by SteamID ASC for stable persistence and
// test-friendly comparisons.
func RunMatchSummary(result *demo.ParseResult, roundMap map[int]int64) ([]MatchSummaryRow, error) {
	_ = roundMap // reserved for rules that look up DB round IDs.
	if result == nil {
		return nil, nil
	}
	tickRate := result.Header.TickRate
	if tickRate <= 0 {
		tickRate = 64
	}

	trades := computeTradesSummary(result.Events, result.Rounds, tickRate)
	if len(trades) == 0 {
		return nil, nil
	}

	out := make([]MatchSummaryRow, 0, len(trades))
	for steamID, ts := range trades {
		out = append(out, MatchSummaryRow{
			SteamID:       steamID,
			OverallScore:  composeOverallScore(ts),
			TradePct:      ts.TradePct,
			AvgTradeTicks: ts.AvgTradeTicks,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].SteamID < out[j].SteamID
	})
	return out, nil
}

// RunPlayerRoundAnalysis computes one row per (player, round) where the
// player had at least one eligible own-death. Slice 7 only fills TradePct;
// subsequent slices fold in additional categories under the same shape.
// Players / rounds with zero eligible deaths are absent from the returned
// slice (no row is persisted — mirrors RunMatchSummary's contract).
//
// Rows are returned sorted by (SteamID ASC, RoundNumber ASC) for stable
// persistence and test-friendly comparisons.
func RunPlayerRoundAnalysis(result *demo.ParseResult, roundMap map[int]int64) ([]PlayerRoundRow, error) {
	_ = roundMap // reserved for rules that look up DB round IDs.
	if result == nil {
		return nil, nil
	}
	tickRate := result.Header.TickRate
	if tickRate <= 0 {
		tickRate = 64
	}

	roundTrades := computeRoundTrades(result.Events, result.Rounds, tickRate)
	if len(roundTrades) == 0 {
		return nil, nil
	}

	out := make([]PlayerRoundRow, 0, len(roundTrades))
	for steamID, byRound := range roundTrades {
		for roundNumber, ts := range byRound {
			out = append(out, PlayerRoundRow{
				SteamID:     steamID,
				RoundNumber: roundNumber,
				TradePct:    ts.TradePct,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SteamID != out[j].SteamID {
			return out[i].SteamID < out[j].SteamID
		}
		return out[i].RoundNumber < out[j].RoundNumber
	})
	return out, nil
}

// composeOverallScore maps the per-player TradesSummary into a 0–100 integer
// score. Slice-5 stub: round(TradePct * 100). Slices 7–8 will rebalance this
// once additional categories report; downstream readers must treat the value
// as an opaque composite, not the raw trade percentage.
func composeOverallScore(s TradesSummary) int {
	score := int(math.Round(s.TradePct * 100))
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}
