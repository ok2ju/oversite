package detectors

import (
	"math"

	"github.com/ok2ju/oversite/internal/demo/contacts"
)

const (
	// BadCrosshairPitchThresholdDeg — analysis §4.1: pitch off
	// head-height by >5° toward the known enemy position.
	BadCrosshairPitchThresholdDeg = 5.0

	// HeadHeightOffsetZ — CS2 standing eye height above feet. A
	// crouched model loses ~25u; v1 accepts the resulting overshoot.
	HeadHeightOffsetZ = 64.0
)

// BadCrosshairHeight implements analysis §4.1 "Yaw/pitch sample at
// t_first: pitch off head-height by >5° toward known enemy position".
// Reads the subject's pitch from the AnalysisTick row at c.TFirst and
// compares against the expected pitch toward the nearest enemy.
//
// Pitch=0 is treated as "not measured" (parser leaves zero on old
// demos) — no finding.
//
// Extras: pitch_actual, pitch_expected, pitch_delta, enemy_steam.
func BadCrosshairHeight(c *contacts.Contact, ctx *DetectorCtx) []ContactMistake {
	subjectRow, ok := nearestTick(ctx.Ticks, c.Subject, int(c.TFirst))
	if !ok || !subjectRow.IsAlive {
		return nil
	}
	if subjectRow.Pitch == 0 {
		return nil
	}
	enemySteam, enemyRow, ok := nearestEnemyAtTick(c, ctx, c.TFirst)
	if !ok {
		return nil
	}
	dz := float64(enemyRow.Z+HeadHeightOffsetZ) - float64(subjectRow.Z+HeadHeightOffsetZ)
	horiz := math.Hypot(float64(enemyRow.X-subjectRow.X), float64(enemyRow.Y-subjectRow.Y))
	if horiz == 0 {
		return nil
	}
	// CS2 pitch is downward-positive — a target *below* the eye level
	// (dz < 0) needs a positive pitch.
	expectedPitch := -math.Atan2(dz, horiz) * 180.0 / math.Pi
	delta := math.Abs(float64(subjectRow.Pitch) - expectedPitch)
	if delta <= BadCrosshairPitchThresholdDeg {
		return nil
	}
	tick := c.TFirst
	extras := map[string]any{
		"pitch_actual":   float64(subjectRow.Pitch),
		"pitch_expected": expectedPitch,
		"pitch_delta":    delta,
		"enemy_steam":    enemySteam,
	}
	return []ContactMistake{NewContactMistake("bad_crosshair_height", &tick, extras)}
}
