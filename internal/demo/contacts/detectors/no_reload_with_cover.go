package detectors

import (
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

// NoReloadClipFraction — analysis §4.3: ammo_clip < 30% of mag is the
// "should have reloaded" threshold.
const NoReloadClipFraction = 0.30

// NoReloadWithCover emits when the subject:
//   - survived the contact,
//   - finished with a low magazine (< 30% of max for the last-held
//     weapon),
//   - was not contested in the next 1.5s post-window (no enemy fired
//     in the window — see helpers.enemySpottedBetween for the v1
//     proxy), and
//   - did NOT reload during that window.
//
// Extras: ammo_clip, weapon_max, weapon, post_end.
func NoReloadWithCover(c *contacts.Contact, ctx *DetectorCtx) []ContactMistake {
	if subjectDiedInContact(c) {
		return nil
	}
	lastShot, ok := lastSubjectShot(c, ctx)
	if !ok {
		return nil
	}
	weapon, ok := ctx.Weapons.Lookup(lastShot.Weapon)
	if !ok || weapon.MaxClip == 0 {
		return nil
	}
	postEnd := c.TLast + contacts.PostWindowTicks
	if postEnd > c.TPost {
		postEnd = c.TPost
	}
	if postEnd <= c.TLast {
		return nil
	}
	if enemySpottedBetween(c, ctx, c.TLast, postEnd) {
		return nil
	}
	if reloadCompletedBetween(ctx, c.Subject, c.TLast, postEnd, weapon.MaxClip) {
		return nil
	}
	threshold := int(float64(weapon.MaxClip) * NoReloadClipFraction)
	endRow, ok := nearestTick(ctx.Ticks, c.Subject, int(postEnd))
	if !ok {
		return nil
	}
	if int(endRow.AmmoClip) >= threshold {
		return nil
	}
	tick := postEnd
	extras := map[string]any{
		"ammo_clip":  int(endRow.AmmoClip),
		"weapon_max": weapon.MaxClip,
		"weapon":     lastShot.Weapon,
		"post_end":   int(postEnd),
	}
	return []ContactMistake{NewContactMistake("no_reload_with_cover", &tick, extras)}
}
