package detectors

import (
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

// AimWhileFlashedMinDurationSecs mirrors the contact-opener threshold
// (../phase-2/02-signals.md §5.5): only "serious" flashes count.
const AimWhileFlashedMinDurationSecs = 0.7

// AimWhileFlashed walks every subject-aggressor weapon_fire inside the
// contact and emits one mistake per shot fired during an active
// subject-victim flash window. Teammate-attacker flashes are filtered
// out (the teammate is at fault, not the subject).
//
// Multi-emit: one row per offending shot.
//
// Extras (per shot): shot_tick, weapon, flash_overlap.
func AimWhileFlashed(c *contacts.Contact, ctx *DetectorCtx) []ContactMistake {
	flashes := subjectFlashWindows(c, ctx, AimWhileFlashedMinDurationSecs)
	if len(flashes) == 0 {
		return nil
	}
	var out []ContactMistake
	for _, evt := range ctx.Events {
		if int32(evt.Tick) < c.TFirst || int32(evt.Tick) > c.TLast {
			continue
		}
		if evt.Type != "weapon_fire" || evt.AttackerSteamID != c.Subject {
			continue
		}
		if !insideAnyFlash(int32(evt.Tick), flashes) {
			continue
		}
		tick := int32(evt.Tick)
		extras := map[string]any{
			"shot_tick":     evt.Tick,
			"weapon":        evt.Weapon,
			"flash_overlap": true,
		}
		out = append(out, NewContactMistake("aim_while_flashed", &tick, extras))
	}
	return out
}
