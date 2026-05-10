package analysis

import (
	"github.com/ok2ju/oversite/internal/demo"
)

// TradesSummary is the per-player aggregate of trade-eligible deaths in a
// match. Reported only for players who actually died (the world / friendly-
// fire / self-kill skips that mistakes.noTradeDeath applies are mirrored
// here so mistakes_for_player + traded_for_player == own_deaths_for_player).
//
// TradePct is the fraction of the player's eligible own-deaths that a
// teammate traded within TradeWindowSeconds, in [0, 1]. AvgTradeTicks is the
// mean (trade_response_tick - own_death_tick) across only the deaths that
// were traded; zero traded deaths → 0 (not NaN).
type TradesSummary struct {
	OwnDeaths     int     `json:"own_deaths"`
	TradedDeaths  int     `json:"traded_deaths"`
	TradePct      float64 `json:"trade_pct"`
	AvgTradeTicks float64 `json:"avg_trade_ticks"`
}

// computeTradesSummary walks every kill event in tick order and aggregates,
// per victim, how often the death was traded by a teammate within the trade
// window. Players with zero eligible own-deaths are absent from the returned
// map (no row is persisted for spectators / unrostered slots).
//
// The "was this death traded?" predicate mirrors mistakes.isTradedForward
// exactly so the trade-aware analyzer rules and this aggregate agree
// row-for-row on what counts as a trade.
func computeTradesSummary(events []demo.GameEvent, _ []demo.RoundData, tickRate float64) map[string]TradesSummary {
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
	byPlayer := make(map[string]*accum, 8)

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

		a, ok := byPlayer[ev.VictimSteamID]
		if !ok {
			a = &accum{}
			byPlayer[ev.VictimSteamID] = a
		}
		a.ownDeaths++

		respTick, traded := tradeResponseTick(events, i, ev, k, windowTicks)
		if traded {
			a.tradedDeaths++
			a.sumTradeRespDtT += respTick - ev.Tick
		}
	}

	out := make(map[string]TradesSummary, len(byPlayer))
	for steamID, a := range byPlayer {
		s := TradesSummary{OwnDeaths: a.ownDeaths, TradedDeaths: a.tradedDeaths}
		if a.ownDeaths > 0 {
			s.TradePct = float64(a.tradedDeaths) / float64(a.ownDeaths)
		}
		if a.tradedDeaths > 0 {
			s.AvgTradeTicks = float64(a.sumTradeRespDtT) / float64(a.tradedDeaths)
		}
		out[steamID] = s
	}
	return out
}

// tradeResponseTick reports whether (and at what tick) some teammate of the
// victim killed the original attacker within windowTicks. The predicate is the
// same forward-walk as mistakes.isTradedForward — when this returns
// (respTick, true), mistakes.noTradeDeath would have skipped the death; when
// it returns (_, false), noTradeDeath would have emitted a no_trade_death.
func tradeResponseTick(events []demo.GameEvent, deathIdx int, death demo.GameEvent, k *demo.KillExtra, windowTicks int) (int, bool) {
	limit := death.Tick + windowTicks
	for j := deathIdx + 1; j < len(events); j++ {
		next := events[j]
		if next.Tick > limit {
			return 0, false
		}
		if next.Type != "kill" {
			continue
		}
		if next.AttackerSteamID == "" {
			continue
		}
		if next.AttackerSteamID == death.VictimSteamID {
			continue // a victim cannot trade their own death.
		}
		if next.VictimSteamID != death.AttackerSteamID {
			continue
		}
		nk, _ := next.ExtraData.(*demo.KillExtra)
		if nk == nil {
			continue
		}
		if nk.AttackerTeam != k.VictimTeam {
			continue
		}
		return next.Tick, true
	}
	return 0, false
}
