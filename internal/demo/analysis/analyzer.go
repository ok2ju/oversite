// Package analysis runs post-ingest mechanical-analysis rules over a parsed
// demo and persists the resulting findings ("mistakes") to the
// analysis_mistakes table. Each rule lives in mistakes.go; the aggregator in
// this file wires them together.
//
// Slice 1 (this file) ships the no_trade_death rule only. Subsequent slices
// add rules without changing the public surface — Run keeps returning the
// combined []Mistake list.
package analysis

import (
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
	MistakeKindNoTradeDeath MistakeKind = "no_trade_death"
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

// Run executes every analyzer rule against the parse result and returns the
// combined list of mistakes. result must be the same struct produced by the
// streaming parser; roundMap (round number → DB round ID) is currently unused
// but accepted so future rules that need DB-side correlations don't have to
// change the signature again. Returns (nil, nil) on a nil result.
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
	return out, nil
}
