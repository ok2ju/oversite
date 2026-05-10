package analysis

import (
	"math"

	"github.com/ok2ju/oversite/internal/demo"
)

// slowReactionMs is the time-to-fire ceiling: a kill where the attacker took
// longer than this to fire after the victim entered their FOV is flagged. The
// plan calls for 300 ms — set as a constant rather than a magic number so the
// rule docs can refer to it.
const slowReactionMs = 300.0

// fovEntryHalfAngleDeg is the half-angle used to decide "victim is in
// shooter's FOV". 30° matches the plan's spec and approximates the screen
// area where a target sits within the crosshair's effective vision cone.
const fovEntryHalfAngleDeg = 30.0

// flickHalfAngleDeg is the per-fire yaw delta (degrees) above which the shot
// counts as a flick. Same 30° as the FOV gate by design — a flick is a snap
// from outside the FOV to on-target.
const flickHalfAngleDeg = 30.0

// flickLookbackSamples is the number of sampled ticks before the fire we
// scan to compute the largest yaw delta. 3 mirrors the counter-strafe
// lookback so a flick covers the same ~187 ms window the plan describes
// ("yaw delta in last 3 ticks before fire").
const flickLookbackSamples = 3

// fovLookbackSamples bounds how far back fovEntryTick walks before giving up
// on finding an off-target sample. 64 samples × 4-tick sample interval ≈
// 256 game ticks ≈ 4 s at 64 tickrate — comfortably larger than any
// reasonable reaction-time threshold (slowReactionMs = 300 ms ≈ 5 samples).
const fovLookbackSamples = 64

// timeToFire emits one Mistake per kill where the attacker took longer than
// slowReactionMs to fire after the victim first entered their FOV. The rule
// walks back from the kill tick and finds the most recent sampled tick where
// the angle from the attacker's facing direction to the victim was already
// inside fovEntryHalfAngleDeg — that's the FOV-entry tick. The reaction time
// is (kill_tick - fov_entry_tick) / tickRate.
//
// Skips world / friendly-fire / self kills (mirrors trades.go), kills with no
// sampled rows for the attacker or victim, and kills where the attacker was
// already on-target at the start of the lookback window (we cannot bound the
// reaction time in that case — flagging would be a false positive).
//
// Why this is not perfect: we don't have line-of-sight from the parsed demo,
// so a kill where the victim was around a corner for 800 ms before entering
// FOV at the last moment will not be flagged here. The plan accepts this
// limitation; it's a reaction-speed signal, not a wallhack detector.
func timeToFire(events []demo.GameEvent, idx PerPlayerTickIndex, tickRate float64) []Mistake {
	if len(events) == 0 || len(idx.Rows) == 0 {
		return nil
	}
	if tickRate <= 0 {
		tickRate = 64
	}
	thresholdTicks := int(slowReactionMs / 1000.0 * tickRate)

	out := make([]Mistake, 0, 8)
	for _, ev := range events {
		if ev.Type != "kill" {
			continue
		}
		if ev.AttackerSteamID == "" || ev.VictimSteamID == "" {
			continue
		}
		if ev.AttackerSteamID == ev.VictimSteamID {
			continue
		}
		k, _ := ev.ExtraData.(*demo.KillExtra)
		if k == nil || k.VictimTeam == "" {
			continue
		}
		if k.AttackerTeam != "" && k.AttackerTeam == k.VictimTeam {
			continue
		}

		fovTick, ok := fovEntryTick(idx, ev.AttackerSteamID, ev.VictimSteamID, ev.Tick)
		if !ok {
			continue
		}
		dt := ev.Tick - fovTick
		if dt <= thresholdTicks {
			continue
		}
		out = append(out, Mistake{
			SteamID:     ev.AttackerSteamID,
			RoundNumber: ev.RoundNumber,
			Tick:        ev.Tick,
			Kind:        string(MistakeKindSlowReaction),
			Extras: map[string]any{
				"reaction_ms":    float64(dt) / tickRate * 1000.0,
				"victim_steam":   ev.VictimSteamID,
				"weapon":         ev.Weapon,
				"fov_entry_tick": fovTick,
			},
		})
	}
	return out
}

// fovEntryTick walks the attacker's sampled ticks at or before killTick and
// returns the most recent tick where the angle to the victim was already
// inside fovEntryHalfAngleDeg. Returns ok=false when no sample is available
// or when the very first sample we look at was already on-target — the
// reaction-time bound is undefined in that case.
//
// Iteration is bounded by previousTickRows (most-recent-first). The previous
// implementation collected every attacker tick into a slice and bubble-sorted
// it on every kill; on long matches that turned the analyzer pass into the
// hot path that stalled "parsing" at ~80%.
func fovEntryTick(idx PerPlayerTickIndex, attackerSteam, victimSteam string, killTick int) (int, bool) {
	if _, ok := idx.Rows[victimSteam]; !ok {
		return 0, false
	}
	// We only need to walk back far enough to find the first off-target
	// sample — once seen, allOnTarget is false and the loop breaks. Capping
	// at fovLookbackSamples keeps the worst case bounded if the attacker was
	// already aimed at the victim for the entire pre-fire history (e.g. a
	// long hold from a watch position); allOnTarget stays true and the rule
	// returns ok=false, which is the original semantic.
	rows := previousTickRows(idx, attackerSteam, killTick+1, fovLookbackSamples)
	if len(rows) == 0 {
		return 0, false
	}
	bestEntry := -1
	allOnTarget := true
	for _, aRow := range rows {
		vRow, ok := nearestTick(idx, victimSteam, int(aRow.Tick))
		if !ok {
			continue
		}
		dx := float64(vRow.X - aRow.X)
		dy := float64(vRow.Y - aRow.Y)
		expected := math.Atan2(dy, dx) * 180.0 / math.Pi
		yaw := float64(aRow.Yaw)
		delta := normalizeYawDeltaDeg(yaw - expected)
		if math.Abs(delta) <= fovEntryHalfAngleDeg {
			bestEntry = int(aRow.Tick)
			continue
		}
		// First off-target sample (working backward) — stop.
		allOnTarget = false
		break
	}
	if bestEntry < 0 || allOnTarget {
		return 0, false
	}
	return bestEntry, true
}

// normalizeYawDeltaDeg wraps a yaw delta into the (-180, 180] range so a
// crossing from 359° to 1° reads as +2° rather than -358°.
func normalizeYawDeltaDeg(deg float64) float64 {
	for deg > 180 {
		deg -= 360
	}
	for deg <= -180 {
		deg += 360
	}
	return deg
}

// missedFlick emits one Mistake per weapon_fire event where the attacker's
// yaw delta over flickLookbackSamples sampled ticks before the fire exceeds
// flickHalfAngleDeg AND the shot did not land on a player (no HitVictimSteamID
// on the WeaponFireExtra). The aggregate "flick hit rate" is computed
// alongside in mechanical_aggregates.go.
//
// Skips fires with no WeaponFireExtra (defensive — every weapon_fire ought to
// have one), no sampled tick row for the attacker, or fewer than two samples
// in the lookback window (we can't compute a delta with one point).
func missedFlick(events []demo.GameEvent, idx PerPlayerTickIndex) []Mistake {
	if len(events) == 0 || len(idx.Rows) == 0 {
		return nil
	}
	out := make([]Mistake, 0, 8)
	for _, ev := range events {
		if ev.Type != "weapon_fire" {
			continue
		}
		if ev.AttackerSteamID == "" {
			continue
		}
		if isNonShotWeapon(ev.Weapon) {
			continue
		}
		extra, _ := ev.ExtraData.(*demo.WeaponFireExtra)
		if extra == nil {
			continue
		}
		delta, ok := flickDeltaDeg(idx, ev.AttackerSteamID, ev.Tick, extra.Yaw)
		if !ok {
			continue
		}
		if math.Abs(delta) <= flickHalfAngleDeg {
			continue
		}
		// A flick that connected isn't a mistake — it's good aim under
		// pressure. We only flag misses.
		if extra.HitVictimSteamID != "" {
			continue
		}
		out = append(out, Mistake{
			SteamID:     ev.AttackerSteamID,
			RoundNumber: ev.RoundNumber,
			Tick:        ev.Tick,
			Kind:        string(MistakeKindMissedFlick),
			Extras: map[string]any{
				"yaw_delta": math.Abs(delta),
				"weapon":    ev.Weapon,
			},
		})
	}
	return out
}

// flickDeltaDeg returns the (signed, normalized) yaw delta between the fire
// yaw and the attacker's facing direction flickLookbackSamples ticks back.
// Returns ok=false when no pre-fire sample is available.
func flickDeltaDeg(idx PerPlayerTickIndex, steamID string, fireTick int, fireYaw float64) (float64, bool) {
	rows := previousTickRows(idx, steamID, fireTick, flickLookbackSamples)
	if len(rows) == 0 {
		return 0, false
	}
	// rows is most-recent-first; the Nth-most-recent (or oldest available
	// when there aren't N samples) is the one we want.
	prev := rows[len(rows)-1]
	return normalizeYawDeltaDeg(fireYaw - float64(prev.Yaw)), true
}
