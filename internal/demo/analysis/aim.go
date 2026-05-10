package analysis

import (
	"math"

	"github.com/ok2ju/oversite/internal/demo"
)

// pitchSlackDeg is the tolerance (degrees) the attacker's pitch is allowed to
// sit below the line-of-sight to a plausible target before we flag the shot
// as "crosshair too low". 5° matches the slice's initial calibration; easy to
// retune once a recent personal demo lights up the rule.
const pitchSlackDeg = 5.0

// eyeHeightOffset is the approximate offset (CS2 world units) from a player's
// origin Z (feet) to their eye Z. Standing stance ~64 u, ducking ~46 u.
// Slice 8 uses a single constant — fine-grained ducking detection is deferred.
const eyeHeightOffset = 64.0

// crosshairTooLow emits one Mistake per weapon_fire event where the attacker's
// pitch (downward-positive in CS2's convention exposed by demoinfocs) sits
// significantly below the line of sight to the most-recent opposing-team
// player in the tick index. Skips world fires, fires without a typed
// *WeaponFireExtra, fires from attackers with no enemy sample available, and
// (when minEngagements > 0) fires from attackers with fewer than that many
// total fires across the match — the fire-count gate drops noisy single-shot
// outliers like one pistol-round half-flick that would otherwise dominate a
// player's category score.
//
// Sign convention for pitch (CS2 / demoinfocs ViewDirectionY): positive is
// looking down. The expected line-of-sight pitch from attacker eye to victim
// head is therefore atan2(attackerEyeZ - victimHeadZ, planar distance), with
// positive values when the attacker is above the victim. We flag fires where
// pitch sits more than pitchSlackDeg below that expected angle — i.e. the
// attacker is aiming significantly lower than the target's head.
func crosshairTooLow(events []demo.GameEvent, rounds []demo.RoundData, idx PerPlayerTickIndex, minEngagements int, _ float64) []Mistake {
	if len(events) == 0 || len(idx) == 0 {
		return nil
	}

	teamsByRound := teamsByRoundFromRosters(rounds)

	if minEngagements > 0 {
		fireCounts := make(map[string]int, 16)
		for _, ev := range events {
			if ev.Type != "weapon_fire" {
				continue
			}
			if ev.AttackerSteamID == "" {
				continue
			}
			fireCounts[ev.AttackerSteamID]++
		}
		out := emitCrosshairTooLow(events, teamsByRound, idx, fireCounts, minEngagements)
		return out
	}
	return emitCrosshairTooLow(events, teamsByRound, idx, nil, 0)
}

func emitCrosshairTooLow(
	events []demo.GameEvent,
	teamsByRound map[int]map[string]string,
	idx PerPlayerTickIndex,
	fireCounts map[string]int,
	minEngagements int,
) []Mistake {
	out := make([]Mistake, 0, 8)
	for _, ev := range events {
		if ev.Type != "weapon_fire" {
			continue
		}
		if ev.AttackerSteamID == "" {
			continue
		}
		extra, _ := ev.ExtraData.(*demo.WeaponFireExtra)
		if extra == nil {
			continue
		}
		if minEngagements > 0 && fireCounts[ev.AttackerSteamID] < minEngagements {
			continue
		}
		teams := teamsByRound[ev.RoundNumber]
		attackerTeam := teams[ev.AttackerSteamID]
		if attackerTeam == "" {
			continue
		}
		// Attacker's own tick row carries Z (feet); add the eye-height offset
		// for the line-of-sight calculation. If we have no row at all for the
		// attacker the rule cannot compare to anything — skip.
		attackerRow, ok := nearestTick(idx, ev.AttackerSteamID, ev.Tick)
		if !ok {
			continue
		}
		victimRow, ok := nearestEnemyTick(idx, teams, attackerTeam, ev.Tick)
		if !ok {
			continue
		}

		eyeZ := float64(attackerRow.Z) + eyeHeightOffset
		headZ := float64(victimRow.Z) + eyeHeightOffset
		dx := ev.X - 0 // attacker X/Y travel on the event itself; victim X/Y aren't on AnalysisTick (memory cut)
		// We don't have victim X/Y on AnalysisTick, so approximate planar
		// distance from the attacker fire event's X/Y to a reasonable
		// reference. Without the victim's planar position the rule degenerates
		// to a pitch-vs-relative-elevation check — enough for slice 8's
		// coarse "crosshair clearly below an enemy's head" signal. Track the
		// vertical delta and require a configurable slack.
		_ = dx
		dz := eyeZ - headZ
		// Use a fixed reference planar distance to convert dz → expected pitch.
		// The choice of distance affects sensitivity; 500 units (~12 m) is a
		// typical CS2 mid-range engagement and keeps the threshold from going
		// extreme on near or far fights. If a future slice ships victim X/Y
		// on AnalysisTick, swap to the true distance.
		const refPlanarDist = 500.0
		expectedPitch := math.Atan2(dz, refPlanarDist) * 180.0 / math.Pi
		if extra.Pitch > expectedPitch+pitchSlackDeg {
			out = append(out, Mistake{
				SteamID:     ev.AttackerSteamID,
				RoundNumber: ev.RoundNumber,
				Tick:        ev.Tick,
				Kind:        string(MistakeKindCrosshairTooLow),
				Extras: map[string]any{
					"pitch":          extra.Pitch,
					"expected_pitch": expectedPitch,
					"weapon":         ev.Weapon,
				},
			})
		}
	}
	return out
}

// teamsByRoundFromRosters builds a per-round (steamID → "CT"/"T") map from
// the parser's RoundParticipant rosters. Players present in multiple rounds
// (the common case) appear once per round, so a halftime side switch is
// represented correctly without a global side map.
func teamsByRoundFromRosters(rounds []demo.RoundData) map[int]map[string]string {
	out := make(map[int]map[string]string, len(rounds))
	for _, r := range rounds {
		if len(r.Roster) == 0 {
			continue
		}
		inner := make(map[string]string, len(r.Roster))
		for _, rp := range r.Roster {
			if rp.SteamID == "" || rp.TeamSide == "" {
				continue
			}
			inner[rp.SteamID] = rp.TeamSide
		}
		out[r.Number] = inner
	}
	return out
}
