package detectors

import (
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

const (
	// NoRepositionMaxMovementUnits — analysis §4.3: moved <75u after
	// the kill.
	NoRepositionMaxMovementUnits = 75.0

	// NoRepositionEnemyRadius — another enemy alive within this radius
	// is the "still dangerous" predicate.
	NoRepositionEnemyRadius = 1500.0
)

// NoRepositionAfterKill emits when the subject killed someone in the
// contact, didn't die afterward, AND stood still (<75u from the kill
// spot for 1.5s) while another enemy was alive within 1500u.
//
// Extras: kill_tick, max_movement, subject_x, subject_y.
func NoRepositionAfterKill(c *contacts.Contact, ctx *DetectorCtx) []ContactMistake {
	if !subjectGotAKill(c) {
		return nil
	}
	if subjectDiedInContact(c) {
		return nil
	}
	if !anotherEnemyAliveNearby(c, ctx, NoRepositionEnemyRadius) {
		return nil
	}
	killTick := subjectLastKillTick(c)
	if killTick < 0 {
		return nil
	}
	startRow, ok := nearestTick(ctx.Ticks, c.Subject, int(killTick))
	if !ok {
		return nil
	}
	endTick := killTick + contacts.PostWindowTicks
	if endTick > c.TPost {
		endTick = c.TPost
	}
	moved := maxMovement(ctx.Ticks, c.Subject, killTick, endTick, startRow)
	if moved >= NoRepositionMaxMovementUnits {
		return nil
	}
	tick := killTick
	extras := map[string]any{
		"kill_tick":    int(killTick),
		"max_movement": moved,
		"subject_x":    float64(startRow.X),
		"subject_y":    float64(startRow.Y),
	}
	return []ContactMistake{NewContactMistake("no_reposition_after_kill", &tick, extras)}
}
