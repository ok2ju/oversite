package detectors

import (
	"math"

	"github.com/ok2ju/oversite/internal/demo/contacts"
)

// ShotWhileMovingSpeedThreshold — analysis §4.2: any weapon_fire by P
// with planar velocity > 110 u/s correlates strongly with "didn't
// counter-strafe."
const ShotWhileMovingSpeedThreshold = 110.0

// ShotWhileMoving walks every subject-aggressor weapon_fire inside
// [c.TFirst, c.TLast] and emits one mistake per shot whose speed
// exceeds ShotWhileMovingSpeedThreshold. Multi-emit: the contact_id +
// kind + tick triple key on contact_mistakes accommodates one row per
// offending shot.
//
// Extras (per shot): shot_tick, weapon, speed.
func ShotWhileMoving(c *contacts.Contact, ctx *DetectorCtx) []ContactMistake {
	var out []ContactMistake
	for _, evt := range ctx.Events {
		if int32(evt.Tick) < c.TFirst || int32(evt.Tick) > c.TLast {
			continue
		}
		if evt.Type != "weapon_fire" || evt.AttackerSteamID != c.Subject {
			continue
		}
		row, ok := nearestTick(ctx.Ticks, c.Subject, evt.Tick)
		if !ok {
			continue
		}
		speed := math.Hypot(float64(row.Vx), float64(row.Vy))
		if speed <= ShotWhileMovingSpeedThreshold {
			continue
		}
		tick := int32(evt.Tick)
		extras := map[string]any{
			"shot_tick": evt.Tick,
			"weapon":    evt.Weapon,
			"speed":     speed,
		}
		out = append(out, NewContactMistake("shot_while_moving", &tick, extras))
	}
	return out
}
