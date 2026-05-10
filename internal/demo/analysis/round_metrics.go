package analysis

import (
	"github.com/ok2ju/oversite/internal/demo"
)

// computeRoundTrades is the per-round, per-player aggregation of trade-eligible
// deaths. The predicate logic mirrors computeTradesSummary in trades.go exactly
// (same world-kill / friendly-fire / self-kill skips, same forward-walk for
// tradeResponseTick) so the round breakdown sums back to the match aggregate
// row-for-row.
//
// Returns a nested map keyed by [steamID][roundNumber]. Players with zero
// eligible own-deaths in a round are absent from that player's inner map (no
// row is persisted for "this player did not die in round N" — matches the
// slice-5 contract: absent ↔ nothing to report, not "trade_pct = 0").
func computeRoundTrades(events []demo.GameEvent, _ []demo.RoundData, tickRate float64) map[string]map[int]TradesSummary {
	if len(events) == 0 {
		return nil
	}
	if tickRate <= 0 {
		tickRate = 64
	}
	windowTicks := int(TradeWindowSeconds * tickRate)

	type accum struct {
		ownDeaths       int
		tradedDeaths    int
		sumTradeRespDtT int
	}
	// (steamID, roundNumber) → accum.
	byPlayerRound := make(map[string]map[int]*accum, 8)

	for i, ev := range events {
		if ev.Type != "kill" {
			continue
		}
		if ev.VictimSteamID == "" {
			continue
		}
		if ev.AttackerSteamID == "" {
			continue // world kill — not trade-eligible.
		}
		if ev.AttackerSteamID == ev.VictimSteamID {
			continue // self-kill / suicide.
		}
		k, _ := ev.ExtraData.(*demo.KillExtra)
		if k == nil || k.VictimTeam == "" {
			continue
		}
		if k.AttackerTeam != "" && k.AttackerTeam == k.VictimTeam {
			continue // friendly fire — not a trade scenario.
		}

		byRound, ok := byPlayerRound[ev.VictimSteamID]
		if !ok {
			byRound = make(map[int]*accum, 4)
			byPlayerRound[ev.VictimSteamID] = byRound
		}
		a, ok := byRound[ev.RoundNumber]
		if !ok {
			a = &accum{}
			byRound[ev.RoundNumber] = a
		}
		a.ownDeaths++

		respTick, traded := tradeResponseTick(events, i, ev, k, windowTicks)
		if traded {
			a.tradedDeaths++
			a.sumTradeRespDtT += respTick - ev.Tick
		}
	}

	out := make(map[string]map[int]TradesSummary, len(byPlayerRound))
	for steamID, byRound := range byPlayerRound {
		inner := make(map[int]TradesSummary, len(byRound))
		for roundNumber, a := range byRound {
			s := TradesSummary{OwnDeaths: a.ownDeaths, TradedDeaths: a.tradedDeaths}
			if a.ownDeaths > 0 {
				s.TradePct = float64(a.tradedDeaths) / float64(a.ownDeaths)
			}
			if a.tradedDeaths > 0 {
				s.AvgTradeTicks = float64(a.sumTradeRespDtT) / float64(a.tradedDeaths)
			}
			inner[roundNumber] = s
		}
		out[steamID] = inner
	}
	return out
}
