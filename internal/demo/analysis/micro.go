package analysis

import (
	"math"

	"github.com/ok2ju/oversite/internal/demo"
)

// MicroAgg is the per-player rollup of the slice-11 micro-skill metrics.
// Each field maps 1:1 to a column on player_match_analysis. Metrics that
// cannot be computed (e.g. crouch on a demo imported before the slice-11
// parser change) stay at zero — the migration's NOT NULL DEFAULT 0 means a
// missing field is indistinguishable from a true zero, which the frontend
// renders as "no data" using the engagement gate the same as the slice-10
// aim/standing-shot percentages.
type MicroAgg struct {
	// TimeToStopMsAvg is the mean time (ms) between leaving the
	// counterStrafeArmedSpeed band and reaching the standingShotMaxSpeed
	// band, measured over fires that produced a counter-strafe success.
	// Zero when no eligible strafe-aim shots were sampled.
	TimeToStopMsAvg float64
	// CrouchBeforeShotCount is the count of fires where the attacker was
	// crouched at the moment of fire.
	CrouchBeforeShotCount int
	// CrouchInsteadOfStrafeCount is the count of fires where the attacker was
	// crouched AND moving above standingShotMaxSpeed — the failure mode.
	CrouchInsteadOfStrafeCount int
	// FlickOvershootAvgDeg / UndershootAvgDeg are the magnitudes of the
	// signed flick error past / before the target across all flick-class
	// fires. Mean over the eligible set; 0 when no flicks sampled.
	FlickOvershootAvgDeg  float64
	FlickUndershootAvgDeg float64
	// FlickBalancePct is 100 * over / (over + under), where over / under
	// are counts of overshooting / undershooting flicks. 50 = balanced;
	// > 50 = leans overshoot (high sens / over-correct); < 50 = leans
	// undershoot (low sens / hesitant).
	FlickBalancePct float64
}

// computeMicroAggregates walks weapon_fire events and produces per-player
// rollups for the six micro metrics. idx must be the same PerPlayerTickIndex
// the slice-9 rules use; with no tick fanout the function returns nil so
// callers can skip persistence entirely.
//
// teamsByRound is the per-round (steamID → "CT"/"T") map the slice-8 aim rule
// already builds via teamsByRoundFromRosters. The flick over/under classifier
// needs it so it can compare the fire angle to the angular distance to the
// most recent opposing-team sample. May be nil — flick metrics then degrade
// to zero, the other four metrics still compute.
//
// All math is bounded by the per-fire lookback windows the existing rules
// already use (counterStrafeLookbackSamples for stop-time, flickLookbackSamples
// for flick error). The function does not re-walk the full match, so the
// added cost is roughly equal to one extra rule pass.
func computeMicroAggregates(
	events []demo.GameEvent,
	idx PerPlayerTickIndex,
	teamsByRound map[int]map[string]string,
	tickRate float64,
) map[string]MicroAgg {
	if len(events) == 0 || len(idx.Rows) == 0 {
		return nil
	}
	if tickRate <= 0 {
		tickRate = 64
	}
	type accum struct {
		stopMsSum   float64
		stopMsCount int
		crouchFires int
		crouchMove  int
		flickOver   int
		flickUnder  int
		overSum     float64
		underSum    float64
	}
	byPlayer := make(map[string]*accum, 16)
	ensure := func(steam string) *accum {
		a, ok := byPlayer[steam]
		if !ok {
			a = &accum{}
			byPlayer[steam] = a
		}
		return a
	}

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
		row, ok := nearestTick(idx, ev.AttackerSteamID, ev.Tick)
		if !ok {
			continue
		}
		curSpeed := math.Sqrt(float64(row.Vx)*float64(row.Vx) + float64(row.Vy)*float64(row.Vy))
		a := ensure(ev.AttackerSteamID)

		// Crouch counters. Crouch is false on demos parsed before P3-1, so
		// these counters degrade to zero on legacy data.
		if row.Crouch {
			a.crouchFires++
			if curSpeed > standingShotMaxSpeed {
				a.crouchMove++
			}
		}

		// Time-to-stop. Walk back through the counter-strafe lookback window
		// to find the most recent sample at or above counterStrafeArmedSpeed
		// — the moment the player began their stop. The interval between
		// that sample and the fire (in ms) is the strafe-aim stop time. We
		// only measure the duration when the fire itself happened in the
		// standing band, otherwise we'd record incomplete stops.
		if curSpeed <= standingShotMaxSpeed {
			prev := previousTickRows(idx, ev.AttackerSteamID, ev.Tick, counterStrafeLookbackSamples)
			for _, p := range prev {
				ps := math.Sqrt(float64(p.Vx)*float64(p.Vx) + float64(p.Vy)*float64(p.Vy))
				if ps >= counterStrafeArmedSpeed {
					stopTicks := ev.Tick - int(p.Tick)
					if stopTicks > 0 {
						a.stopMsSum += float64(stopTicks) / tickRate * 1000.0
						a.stopMsCount++
					}
					break
				}
			}
		}

		// Flick over/under. Use the existing flickDeltaDeg helper so the
		// lookback window matches missedFlick exactly. The classification
		// requires line-of-sight to the most recent enemy sample so we know
		// the angular distance to target; without that, we cannot decide
		// whether the player overshot or undershot.
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
		if teamsByRound == nil {
			continue
		}
		teams := teamsByRound[ev.RoundNumber]
		attackerTeam := teams[ev.AttackerSteamID]
		if attackerTeam == "" {
			continue
		}
		victim, ok := nearestEnemyTick(idx, teams, attackerTeam, ev.Tick)
		if !ok {
			continue
		}
		dx := float64(victim.X - row.X)
		dy := float64(victim.Y - row.Y)
		expectedYaw := math.Atan2(dy, dx) * 180.0 / math.Pi
		// Signed yaw error of the fire angle relative to the target. Positive
		// values mean the fire angle "led" the rotation by more degrees than
		// the angular distance to the enemy — overshoot. Negative values
		// mean the player rotated less than needed — undershoot.
		toTarget := normalizeYawDeltaDeg(expectedYaw - float64(row.Yaw))
		err := normalizeYawDeltaDeg(extra.Yaw - float64(row.Yaw) - toTarget)
		if err > 0 {
			a.flickOver++
			a.overSum += err
		} else if err < 0 {
			a.flickUnder++
			a.underSum += -err
		}
	}

	out := make(map[string]MicroAgg, len(byPlayer))
	for steamID, a := range byPlayer {
		ag := MicroAgg{
			CrouchBeforeShotCount:      a.crouchFires,
			CrouchInsteadOfStrafeCount: a.crouchMove,
		}
		if a.stopMsCount > 0 {
			ag.TimeToStopMsAvg = a.stopMsSum / float64(a.stopMsCount)
		}
		if a.flickOver > 0 {
			ag.FlickOvershootAvgDeg = a.overSum / float64(a.flickOver)
		}
		if a.flickUnder > 0 {
			ag.FlickUndershootAvgDeg = a.underSum / float64(a.flickUnder)
		}
		if total := a.flickOver + a.flickUnder; total > 0 {
			ag.FlickBalancePct = 100.0 * float64(a.flickOver) / float64(total)
		}
		out[steamID] = ag
	}
	return out
}
