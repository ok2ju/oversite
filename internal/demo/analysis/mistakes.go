package analysis

import (
	"github.com/ok2ju/oversite/internal/demo"
)

// noTradeDeath emits one Mistake per kill where the victim was not traded by a
// teammate within TradeWindowSeconds. World kills, self-kills, friendly fire,
// and kills missing the typed *KillExtra are all skipped — there is no
// meaningful "trade" to enforce in those cases.
//
// events is expected to be ordered by tick (the parser guarantees this); the
// inner forward-walk relies on that ordering to early-out as soon as the
// trade window closes.
func noTradeDeath(events []demo.GameEvent, tickRate float64) []Mistake {
	if len(events) == 0 {
		return nil
	}
	windowTicks := int(TradeWindowSeconds * tickRate)

	out := make([]Mistake, 0, 8)
	for i, ev := range events {
		if ev.Type != "kill" {
			continue
		}
		if ev.VictimSteamID == "" {
			continue
		}
		if ev.AttackerSteamID == "" {
			continue // world kill — nobody to trade.
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

		if isTradedForward(events, i, ev, k, windowTicks) {
			continue
		}

		out = append(out, Mistake{
			SteamID:     ev.VictimSteamID,
			RoundNumber: ev.RoundNumber,
			Tick:        ev.Tick,
			Kind:        string(MistakeKindNoTradeDeath),
			Extras:      noTradeDeathExtras(ev),
		})
	}
	return out
}

// isTradedForward reports whether some teammate of the victim killed the
// original attacker within windowTicks of the death event at events[deathIdx].
func isTradedForward(events []demo.GameEvent, deathIdx int, death demo.GameEvent, k *demo.KillExtra, windowTicks int) bool {
	limit := death.Tick + windowTicks
	for j := deathIdx + 1; j < len(events); j++ {
		next := events[j]
		if next.Tick > limit {
			return false
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
		return true
	}
	return false
}

// noTradeDeathExtras packs the rule-specific context for the side panel. Kept
// minimal in slice 1 (the panel only renders round + clock); future slices add
// the offending opponent's name / weapon details for the click-to-seek UX.
func noTradeDeathExtras(ev demo.GameEvent) map[string]any {
	extras := map[string]any{
		"killer_steam_id": ev.AttackerSteamID,
	}
	if ev.Weapon != "" {
		extras["weapon"] = ev.Weapon
	}
	return extras
}
