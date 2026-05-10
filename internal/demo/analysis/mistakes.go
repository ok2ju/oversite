package analysis

import (
	"sort"
	"strings"

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

// diedWithUtilUnused emits one Mistake per (player, round) where the player
// died holding at least one grenade type they did not throw before the death
// tick. The bar is "threw zero of type X": a player who spawns with 2
// flashbangs and throws only one is *not* flagged — multi-grenade nuance is
// deferred. Survivors of a round produce no mistake; the rule fires only on
// the player's death event.
func diedWithUtilUnused(events []demo.GameEvent, rounds []demo.RoundData) []Mistake {
	if len(events) == 0 || len(rounds) == 0 {
		return nil
	}

	// (round, steamID) → set of normalized utility tokens present at freeze-end.
	type key struct {
		round int
		steam string
	}
	spawnUtil := make(map[key]map[string]struct{}, len(rounds)*10)
	for _, r := range rounds {
		for _, rp := range r.Roster {
			if rp.SteamID == "" {
				continue
			}
			util := parseUtilFromInventory(rp.Inventory)
			if len(util) == 0 {
				continue
			}
			set := make(map[string]struct{}, len(util))
			for _, u := range util {
				set[u] = struct{}{}
			}
			spawnUtil[key{round: r.Number, steam: rp.SteamID}] = set
		}
	}
	if len(spawnUtil) == 0 {
		return nil
	}

	thrownSoFar := make(map[key]map[string]struct{}, len(spawnUtil))
	out := make([]Mistake, 0, 8)
	for _, ev := range events {
		switch ev.Type {
		case "grenade_throw":
			if ev.AttackerSteamID == "" {
				continue
			}
			tok := normalizeUtilToken(ev.Weapon)
			if !isUtilToken(tok) {
				continue
			}
			k := key{round: ev.RoundNumber, steam: ev.AttackerSteamID}
			set, ok := thrownSoFar[k]
			if !ok {
				set = make(map[string]struct{}, 4)
				thrownSoFar[k] = set
			}
			set[tok] = struct{}{}
		case "kill":
			if ev.VictimSteamID == "" {
				continue
			}
			k := key{round: ev.RoundNumber, steam: ev.VictimSteamID}
			spawn, ok := spawnUtil[k]
			if !ok || len(spawn) == 0 {
				continue
			}
			thrown := thrownSoFar[k]
			unused := make([]string, 0, len(spawn))
			for u := range spawn {
				if _, t := thrown[u]; t {
					continue
				}
				unused = append(unused, u)
			}
			if len(unused) == 0 {
				continue
			}
			sort.Strings(unused)
			out = append(out, Mistake{
				SteamID:     ev.VictimSteamID,
				RoundNumber: ev.RoundNumber,
				Tick:        ev.Tick,
				Kind:        string(MistakeKindDiedWithUtilUnused),
				Extras: map[string]any{
					"unused": unused,
				},
			})
		}
	}
	return out
}

// parseUtilFromInventory returns the deduplicated set of normalized utility
// tokens present in a freeze-end inventory string. Inventory tokens come from
// encodeInventory (Equipment.String() joined by ","); we normalize via
// lowercase + stripping whitespace and hyphens before comparing against the
// same normalization applied to grenade_throw.Weapon. Non-utility entries
// (rifles, kits, kevlar, …) are dropped.
func parseUtilFromInventory(inv string) []string {
	if inv == "" {
		return nil
	}
	seen := make(map[string]struct{}, 4)
	out := make([]string, 0, 4)
	for _, raw := range strings.Split(inv, ",") {
		tok := normalizeUtilToken(raw)
		if !isUtilToken(tok) {
			continue
		}
		if _, dup := seen[tok]; dup {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
	}
	return out
}

// normalizeUtilToken collapses Equipment.String()-style names to a stable key
// shared between inventory entries and grenade_throw.Weapon. Both surfaces use
// the same demoinfocs Equipment.String() output, so the only divergence we
// need to absorb is whitespace and hyphenation drift across versions.
func normalizeUtilToken(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}

// utilTokenSet enumerates the normalized grenade types we consider for the
// rule. Aliases ("incgrenade" / "incendiarygrenade", "decoy" / "decoygrenade")
// cover Equipment.String() drift across demoinfocs versions.
var utilTokenSet = map[string]struct{}{
	"hegrenade":         {},
	"flashbang":         {},
	"smokegrenade":      {},
	"molotov":           {},
	"incgrenade":        {},
	"incendiarygrenade": {},
	"decoy":             {},
	"decoygrenade":      {},
}

func isUtilToken(tok string) bool {
	if tok == "" {
		return false
	}
	_, ok := utilTokenSet[tok]
	return ok
}
