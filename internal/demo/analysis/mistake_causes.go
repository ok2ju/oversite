package analysis

import (
	"math"

	"github.com/ok2ju/oversite/internal/demo"
)

// CauseTag enumerates the heuristic-derived "why this shot missed" labels the
// analyzer attaches to fire-related mistakes. The frontend reads them off
// extras_json.cause_tag to drive the mistake-detail headline + speed bar
// coloring. Priority order (when multiple match) follows §6.3 of the
// analysis-overhaul plan: crouch > shot_before_stop > no_counter_strafe >
// over/under_flick > late_reaction.
type CauseTag string

const (
	CauseTagCrouchBeforeShot CauseTag = "crouch_before_shot"
	CauseTagShotBeforeStop   CauseTag = "shot_before_stop"
	CauseTagNoCounterStrafe  CauseTag = "no_counter_strafe"
	CauseTagOverFlick        CauseTag = "over_flick"
	CauseTagUnderFlick       CauseTag = "under_flick"
	CauseTagLateReaction     CauseTag = "late_reaction"
)

// fireCauseMistakeKinds is the set of mistake kinds the cause-classifier walks.
// Only fire-related kinds get the per-tick window forensics: an untraded death
// or eco misbuy has no firing window to forensically replay.
var fireCauseMistakeKinds = map[string]struct{}{
	string(MistakeKindMissedFirstShot): {},
	string(MistakeKindShotWhileMoving): {},
	string(MistakeKindMissedFlick):     {},
	string(MistakeKindSlowReaction):    {},
	string(MistakeKindNoCounterStrafe): {},
}

// causeFireWindowSamples is how many sampled ticks (most recent at or before
// the fire) we snapshot into extras_json. Four matches the plan's example
// payload and aligns with the existing flick / counter-strafe lookback.
const causeFireWindowSamples = 4

// defaultWeaponSpeedCap is the planar-speed cap (CS2 world u/s) above which
// first-bullet accuracy is degraded. Slice 12 ships one global value matching
// standingShotMaxSpeed; a per-weapon table is deferred to a later slice.
const defaultWeaponSpeedCap = standingShotMaxSpeed

// flickCauseTolDeg is the half-window (degrees) inside which a flick error is
// considered "centered" — outside this band the classifier picks over- or
// under-flick. 7.5° (= flickHalfAngleDeg / 4) keeps the classifier from
// flipping on tiny rounding noise and matches the §6.3 "+ tolerance" wording.
const flickCauseTolDeg = flickHalfAngleDeg / 4.0

// IsFireRelatedMistake reports whether kind is one the cause-classifier
// enriches with the per-tick speed/yaw/pitch window. Exported for tests and
// for the frontend-facing context binding (P2-3) so co-occurring chips can
// filter to the same set without duplicating the constant.
func IsFireRelatedMistake(kind string) bool {
	_, ok := fireCauseMistakeKinds[kind]
	return ok
}

// EnrichFireMistakes augments fire-related mistakes' Extras with the tick
// speed window, yaw / pitch paths, weapon speed cap, and a single derived
// cause_tag. Non-fire mistakes are returned unchanged. Mistakes whose attacker
// has no sampled tick row are returned unchanged — the rule degrades to "no
// forensics available" rather than guessing.
//
// The function mutates in place AND returns the slice header for chainability
// — the caller's slice is unchanged, only its element fields. This avoids
// reallocating mistakes on every match (~hundreds of fire-related rows).
func EnrichFireMistakes(
	mistakes []Mistake,
	idx PerPlayerTickIndex,
	teamsByRound map[int]map[string]string,
) []Mistake {
	if len(mistakes) == 0 || len(idx.Rows) == 0 {
		return mistakes
	}
	for i := range mistakes {
		if !IsFireRelatedMistake(mistakes[i].Kind) {
			continue
		}
		enrichOneFireMistake(&mistakes[i], idx, teamsByRound)
	}
	return mistakes
}

// enrichOneFireMistake attaches the tick-window forensics + cause tag to a
// single mistake. Skips silently when no tick rows are available — extras
// stays at whatever the rule emitted so the panel still renders the row.
func enrichOneFireMistake(
	m *Mistake,
	idx PerPlayerTickIndex,
	teamsByRound map[int]map[string]string,
) {
	rows := previousTickRows(idx, m.SteamID, m.Tick+1, causeFireWindowSamples)
	if len(rows) == 0 {
		return
	}
	// previousTickRows returns most-recent-first; reverse to oldest-first so
	// ticks_window / speeds / paths read left-to-right in time order.
	ordered := make([]demo.AnalysisTick, len(rows))
	for j, r := range rows {
		ordered[len(rows)-1-j] = r
	}
	ticksWindow := make([]int, len(ordered))
	speeds := make([]float64, len(ordered))
	yawPath := make([]float64, len(ordered))
	pitchPath := make([]float64, len(ordered))
	for j, r := range ordered {
		ticksWindow[j] = int(r.Tick)
		speeds[j] = math.Sqrt(float64(r.Vx)*float64(r.Vx) + float64(r.Vy)*float64(r.Vy))
		yawPath[j] = float64(r.Yaw)
		pitchPath[j] = float64(r.Pitch)
	}
	speedAtFire := speeds[len(speeds)-1]
	cause := classifyCause(*m, ordered, idx, teamsByRound)

	if m.Extras == nil {
		m.Extras = map[string]any{}
	}
	m.Extras["fire_tick"] = m.Tick
	m.Extras["speed_at_fire"] = speedAtFire
	m.Extras["weapon_speed_cap"] = defaultWeaponSpeedCap
	m.Extras["ticks_window"] = ticksWindow
	m.Extras["speeds"] = speeds
	m.Extras["yaw_path"] = yawPath
	m.Extras["pitch_path"] = pitchPath
	if cause != "" {
		m.Extras["cause_tag"] = string(cause)
	}
}

// classifyCause picks the highest-priority cause tag for the fire window per
// §6.3. Returns "" when no heuristic matches — the frontend renders the
// mistake's Title without a cause subtitle in that case.
func classifyCause(
	m Mistake,
	window []demo.AnalysisTick,
	idx PerPlayerTickIndex,
	teamsByRound map[int]map[string]string,
) CauseTag {
	if len(window) == 0 {
		return ""
	}
	fireRow := window[len(window)-1]
	speedAtFire := math.Sqrt(float64(fireRow.Vx)*float64(fireRow.Vx) + float64(fireRow.Vy)*float64(fireRow.Vy))

	// 1) crouch_before_shot — most damaging and easiest to spot. Crouch is
	// false on demos parsed before P3-1; the branch is therefore inert on
	// legacy data without a feature gate.
	if fireRow.Crouch {
		return CauseTagCrouchBeforeShot
	}

	// 2 / 3) Speed-related causes. Distinguish "tried to stop, mistimed"
	// from "never tried" by inspecting whether the previous sample showed
	// meaningful deceleration. A ≥ 5 u/s drop in one sample-interval is the
	// floor — anything smaller is within sampling noise on a stationary
	// player and we'd flag a non-attempt.
	if speedAtFire > defaultWeaponSpeedCap {
		if len(window) >= 2 {
			prev := window[len(window)-2]
			prevSpeed := math.Sqrt(float64(prev.Vx)*float64(prev.Vx) + float64(prev.Vy)*float64(prev.Vy))
			if prevSpeed > speedAtFire+5.0 {
				return CauseTagShotBeforeStop
			}
		}
		return CauseTagNoCounterStrafe
	}

	// 4) over/under_flick — classify only when the rule itself fired on a
	// flick or we can confirm a flick-class yaw delta. Other kinds (e.g. a
	// stationary missed_first_shot) won't pick this up.
	if m.Kind == string(MistakeKindMissedFlick) || m.Kind == string(MistakeKindMissedFirstShot) {
		if cause := flickCause(m, window, idx, teamsByRound); cause != "" {
			return cause
		}
	}

	// 5) late_reaction — the slow_reaction rule already confirms the timing,
	// so the cause tag is just the kind's surface label here.
	if m.Kind == string(MistakeKindSlowReaction) {
		return CauseTagLateReaction
	}

	return ""
}

// flickCause classifies a flick as over- or under-shoot using the most recent
// enemy sample as the angular reference. Returns "" when the angular distance
// can't be computed (no enemy in the index, attacker has no team metadata, or
// the yaw delta is below the flick threshold).
func flickCause(
	m Mistake,
	window []demo.AnalysisTick,
	idx PerPlayerTickIndex,
	teamsByRound map[int]map[string]string,
) CauseTag {
	if len(window) < 2 {
		return ""
	}
	teams := teamsByRound[m.RoundNumber]
	if teams == nil {
		return ""
	}
	attackerTeam := teams[m.SteamID]
	if attackerTeam == "" {
		return ""
	}
	fireRow := window[len(window)-1]
	victim, ok := nearestEnemyTick(idx, teams, attackerTeam, m.Tick)
	if !ok {
		return ""
	}
	prev := window[0] // oldest sample in window — pre-flick orientation
	dx := float64(victim.X - fireRow.X)
	dy := float64(victim.Y - fireRow.Y)
	expectedYaw := math.Atan2(dy, dx) * 180.0 / math.Pi
	yawDelta := normalizeYawDeltaDeg(float64(fireRow.Yaw) - float64(prev.Yaw))
	if math.Abs(yawDelta) < flickHalfAngleDeg {
		return ""
	}
	toTarget := normalizeYawDeltaDeg(expectedYaw - float64(prev.Yaw))
	err := yawDelta - toTarget
	if err > flickCauseTolDeg {
		return CauseTagOverFlick
	}
	if err < -flickCauseTolDeg {
		return CauseTagUnderFlick
	}
	return ""
}
