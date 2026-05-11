package analysis

import (
	"math"

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

// smokeRec is the per-smoke working record used by smokeProducedAssist (and
// originally by smokeEffectiveness). Hoisted to package scope so the assist
// helper can reference the type — Go forbids referencing a function-local
// type from a sibling function.
type smokeRec struct {
	idx     int
	tick    int
	round   int
	thrower string
	x, y    float64
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
