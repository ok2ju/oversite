package detectors

import (
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

// PeekWhileReloadingClipFraction — analysis §4.1: ammo_clip at t_first
// < 30% of max for the held weapon.
const PeekWhileReloadingClipFraction = 0.30

// PeekWhileReloading emits when the subject opens a contact (t_first)
// with a clip below 30% of the held weapon's max. The held weapon is
// inferred from the subject's most recent weapon_fire — AnalysisTick
// only carries AmmoClip, not the active-weapon class.
//
// Extras: ammo_clip, weapon_max, weapon.
func PeekWhileReloading(c *contacts.Contact, ctx *DetectorCtx) []ContactMistake {
	subjectRow, ok := nearestTick(ctx.Ticks, c.Subject, int(c.TFirst))
	if !ok || !subjectRow.IsAlive {
		return nil
	}
	weapon, ok := subjectWeaponAtTick(c, ctx, c.TFirst)
	if !ok {
		return nil
	}
	info, ok := ctx.Weapons.Lookup(weapon)
	if !ok || info.MaxClip == 0 {
		return nil
	}
	threshold := int(float64(info.MaxClip) * PeekWhileReloadingClipFraction)
	if int(subjectRow.AmmoClip) >= threshold {
		return nil
	}
	tick := c.TFirst
	extras := map[string]any{
		"ammo_clip":  int(subjectRow.AmmoClip),
		"weapon_max": info.MaxClip,
		"weapon":     weapon,
	}
	return []ContactMistake{NewContactMistake("peek_while_reloading", &tick, extras)}
}
