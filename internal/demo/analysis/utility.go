package analysis

import (
	"math"
	"sort"

	"github.com/ok2ju/oversite/internal/demo"
)

// smokeAssistRadius is the planar distance (CS2 world units) from a smoke's
// detonation center within which a teammate kill counts as a smoke-assisted
// kill. 800 u matches the plan; ~roughly the visual diameter of a deployed
// smoke.
const smokeAssistRadius = 800.0

// smokeAssistWindowSecs is how long after a smoke detonates a teammate kill
// can still be attributed to the smoke. Five seconds matches the plan's
// "kill within 5 s of smoke center".
const smokeAssistWindowSecs = 5.0

// smokeRec is the per-smoke working record used by smokeEffectiveness and
// smokeProducedAssist. Hoisted to package scope so the assist helper can
// reference the type — Go forbids referencing a function-local type from a
// sibling function.
type smokeRec struct {
	idx       int
	tick      int
	round     int
	thrower   string
	throwTeam string
	x, y      float64
}

// smokeEffectiveness emits one Mistake per smoke detonation that produced no
// teammate kill within smokeAssistRadius / smokeAssistWindowSecs. The owning
// player (smoke thrower) carries the mistake — the rule's intent is to
// surface "your smoke didn't help anyone" so they can rethink the lineup.
//
// We pair smoke_start events to the thrower via the most recent grenade_throw
// of weapon = smokegrenade for the same round. Detonation events live on the
// same GameEvent stream tagged Type = "smoke_start" with attacker_steam_id
// already filled by the parser, so the lookup is rarely needed; it serves as
// a fallback for demos where the detonation row lost the attacker.
func smokeEffectiveness(events []demo.GameEvent, tickRate float64) []Mistake {
	if len(events) == 0 {
		return nil
	}
	if tickRate <= 0 {
		tickRate = 64
	}
	windowTicks := int(smokeAssistWindowSecs * tickRate)

	smokes := make([]smokeRec, 0, 16)
	lastThrowByPlayer := make(map[string]demo.GameEvent, 8)
	for i, ev := range events {
		if ev.Type == "grenade_throw" && normalizeUtilToken(ev.Weapon) == "smokegrenade" {
			lastThrowByPlayer[ev.AttackerSteamID] = ev
			continue
		}
		if ev.Type != "smoke_start" {
			continue
		}
		thrower := ev.AttackerSteamID
		if thrower == "" {
			// Fallback: latest smoke throw — best-effort.
			var best demo.GameEvent
			for _, t := range lastThrowByPlayer {
				if t.RoundNumber == ev.RoundNumber && t.Tick > best.Tick {
					best = t
				}
			}
			thrower = best.AttackerSteamID
		}
		if thrower == "" {
			continue
		}
		throwTeam := ""
		if k, ok := lastThrowByPlayer[thrower]; ok {
			if extra, ok := k.ExtraData.(*demo.GrenadeThrowExtra); ok && extra != nil {
				_ = extra // reserved for thrower-team override
			}
		}
		smokes = append(smokes, smokeRec{
			idx:       i,
			tick:      ev.Tick,
			round:     ev.RoundNumber,
			thrower:   thrower,
			throwTeam: throwTeam,
			x:         ev.X,
			y:         ev.Y,
		})
	}
	if len(smokes) == 0 {
		return nil
	}

	out := make([]Mistake, 0, 8)
	for _, s := range smokes {
		if smokeProducedAssist(events, s, windowTicks) {
			continue
		}
		out = append(out, Mistake{
			SteamID:     s.thrower,
			RoundNumber: s.round,
			Tick:        s.tick,
			Kind:        string(MistakeKindUnusedSmoke),
			Extras: map[string]any{
				"smoke_x": s.x,
				"smoke_y": s.y,
			},
		})
	}
	return out
}

// smokeProducedAssist reports whether any teammate kill landed inside the
// smoke's radius / window. We classify "teammate" by checking the thrower's
// team on a kill within the window: if the killer's KillExtra.AttackerTeam
// matches the thrower's KillExtra.AttackerTeam (same round), it's a
// teammate. We discover the thrower's team by walking forward from the
// smoke and accepting the first kill where the attacker is the thrower
// themselves OR by reading any kill row referencing the thrower.
func smokeProducedAssist(events []demo.GameEvent, s smokeRec, windowTicks int) bool {
	throwerTeam := teamForPlayerInRound(events, s.round, s.thrower)
	if throwerTeam == "" {
		return false
	}
	limit := s.tick + windowTicks
	for j := s.idx + 1; j < len(events); j++ {
		next := events[j]
		if next.Tick > limit {
			return false
		}
		if next.Type != "kill" {
			continue
		}
		k, _ := next.ExtraData.(*demo.KillExtra)
		if k == nil {
			continue
		}
		if k.AttackerTeam == "" || k.AttackerTeam != throwerTeam {
			continue
		}
		dx := next.X - s.x
		dy := next.Y - s.y
		if math.Sqrt(dx*dx+dy*dy) > smokeAssistRadius {
			continue
		}
		return true
	}
	return false
}

// teamForPlayerInRound returns the side ("CT"/"T") for the given player in
// the supplied round by scanning kill events. Returns "" when no kill in the
// round references the player as attacker or victim — the rule treats this
// as "team unknown" and skips the smoke check.
func teamForPlayerInRound(events []demo.GameEvent, round int, steamID string) string {
	for _, ev := range events {
		if ev.RoundNumber != round {
			continue
		}
		if ev.Type != "kill" {
			continue
		}
		k, _ := ev.ExtraData.(*demo.KillExtra)
		if k == nil {
			continue
		}
		if ev.AttackerSteamID == steamID && k.AttackerTeam != "" {
			return k.AttackerTeam
		}
		if ev.VictimSteamID == steamID && k.VictimTeam != "" {
			return k.VictimTeam
		}
	}
	return ""
}

// survivedWithUtilUnused emits one Mistake per (player, round) where the
// player survived the round (no death event) but ended with at least one
// grenade type they spawned with and never threw. Complements
// diedWithUtilUnused which only fires on death — together they cover every
// "had util, didn't use it" case.
func survivedWithUtilUnused(events []demo.GameEvent, rounds []demo.RoundData) []Mistake {
	if len(rounds) == 0 {
		return nil
	}
	type key struct {
		round int
		steam string
	}
	spawnUtil := make(map[key]map[string]struct{}, len(rounds)*10)
	rosterPresent := make(map[key]struct{}, len(rounds)*10)
	for _, r := range rounds {
		for _, rp := range r.Roster {
			if rp.SteamID == "" {
				continue
			}
			rosterPresent[key{round: r.Number, steam: rp.SteamID}] = struct{}{}
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
	died := make(map[key]int, len(spawnUtil))
	roundEndTick := make(map[int]int, len(rounds))
	for _, r := range rounds {
		roundEndTick[r.Number] = r.EndTick
	}
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
			died[k] = ev.Tick
		}
	}

	out := make([]Mistake, 0, 8)
	for k, spawn := range spawnUtil {
		if _, dead := died[k]; dead {
			continue // diedWithUtilUnused already covers this case.
		}
		if _, ok := rosterPresent[k]; !ok {
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
		// Place the mistake at round end so the timeline cursor seeks to a
		// stable, post-action moment. Falls back to 0 when the parser
		// didn't capture round_end (legacy fixtures).
		tick := roundEndTick[k.round]
		out = append(out, Mistake{
			SteamID:     k.steam,
			RoundNumber: k.round,
			Tick:        tick,
			Kind:        string(MistakeKindSurvivedWithUtil),
			Extras: map[string]any{
				"unused": unused,
			},
		})
	}
	return out
}

// walkedIntoMolotov emits one Mistake per player_hurt event whose attacker's
// weapon classifies as inferno / molotov damage AND the victim's HP went
// below their previous step (we use the HealthDamage hot column). The rule
// surfaces "you took fire damage that you could have avoided" — repeat hits
// from the same molotov in quick succession collapse into a single mistake
// keyed on the first hit's tick.
func walkedIntoMolotov(events []demo.GameEvent) []Mistake {
	if len(events) == 0 {
		return nil
	}
	type key struct {
		round int
		steam string
	}
	const collapseTicks = 64 // ~1 s — molotov ticks once per ~250 ms; this groups consecutive ticks of the same exposure into one mistake.
	lastHitAt := make(map[key]int, 8)
	out := make([]Mistake, 0, 8)
	for _, ev := range events {
		if ev.Type != "player_hurt" {
			continue
		}
		if ev.VictimSteamID == "" {
			continue
		}
		if !isInfernoWeapon(ev.Weapon) {
			continue
		}
		k := key{round: ev.RoundNumber, steam: ev.VictimSteamID}
		if prev, ok := lastHitAt[k]; ok && ev.Tick-prev <= collapseTicks {
			lastHitAt[k] = ev.Tick
			continue
		}
		lastHitAt[k] = ev.Tick
		out = append(out, Mistake{
			SteamID:     ev.VictimSteamID,
			RoundNumber: ev.RoundNumber,
			Tick:        ev.Tick,
			Kind:        string(MistakeKindWalkedIntoMolotov),
			Extras: map[string]any{
				"weapon": ev.Weapon,
			},
		})
	}
	return out
}

func isInfernoWeapon(weapon string) bool {
	switch normalizeUtilToken(weapon) {
	case "molotov", "incgrenade", "incendiarygrenade", "inferno":
		return true
	}
	return false
}
