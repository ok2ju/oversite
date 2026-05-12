package detectors

import (
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

// SlowReactionMs is the analysis §4.2 threshold: a reaction window
// longer than this from "spotted enemy" to "first shot" trips the
// rule. The contact-aware reading uses the visibility-derived spotted
// tick directly (the round-level rule used a FOV-entry approximation).
const SlowReactionMs = 250.0

// SlowReaction implements the analysis §4.2 "first weapon_fire by P
// after spotted_on(E) > 250 ms" rule scoped to the contact. The first
// SignalVisibility opens the reaction stopwatch; the first
// SignalWeaponFireHit with Subject=SubjectAggressor stops it.
//
// Reads c.Signals only. Tick samples not needed.
// Emits when: firedTick - spottedTick > threshold.
// Extras: reaction_ms, spotted_tick, fire_tick.
func SlowReaction(c *contacts.Contact, ctx *DetectorCtx) []ContactMistake {
	var spottedTick int32 = -1
	var firedTick int32 = -1
	for i := range c.Signals {
		s := c.Signals[i]
		switch s.Kind {
		case contacts.SignalVisibility:
			if spottedTick < 0 {
				spottedTick = s.Tick
			}
		case contacts.SignalWeaponFireHit:
			if s.Subject == contacts.SubjectAggressor && firedTick < 0 {
				firedTick = s.Tick
			}
		}
		if spottedTick >= 0 && firedTick >= 0 {
			break
		}
	}
	if spottedTick < 0 || firedTick < 0 || firedTick <= spottedTick {
		return nil
	}
	dt := firedTick - spottedTick
	tickRate := ctx.TickRate
	if tickRate <= 0 {
		tickRate = contacts.TickRate64
	}
	thresholdTicks := int32(SlowReactionMs / 1000.0 * tickRate)
	if dt <= thresholdTicks {
		return nil
	}
	tick := firedTick
	extras := map[string]any{
		"reaction_ms":  float64(dt) / tickRate * 1000.0,
		"spotted_tick": int(spottedTick),
		"fire_tick":    int(firedTick),
	}
	return []ContactMistake{NewContactMistake("slow_reaction", &tick, extras)}
}
