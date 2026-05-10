package analysis

import (
	"math"
	"strings"

	"github.com/ok2ju/oversite/internal/demo"
)

// standingShotMaxSpeed is the planar speed (CS2 world units per second) at
// or below which the attacker is considered "standing still" for the
// shot-while-moving rule. 40 u/s matches the plan threshold and sits below
// the strafing threshold (60 u/s in player_stats.go) because "shot while
// moving" is a stricter signal than "is strafing" — a player can move slowly
// without strafe-aiming yet still suffer accuracy loss at 40+ u/s.
const standingShotMaxSpeed = 40.0

// counterStrafeArmedSpeed is the running-speed minimum we require sometime
// in the last counterStrafeLookbackSamples ticks before the shot to consider
// the attacker "previously moving". The plan specifies 100 u/s — anything
// below that and the attacker was already walking / standing, so the
// counter-strafe predicate is undefined.
const counterStrafeArmedSpeed = 100.0

// counterStrafeLookbackSamples is the number of sampled ticks (each
// tickInterval = 4 game ticks) we look back from the fire to find the
// pre-stop velocity. Three samples ≈ 12 game ticks ≈ 187 ms at 64 tickrate,
// which matches the plan's "velocity(t-3)" intent.
const counterStrafeLookbackSamples = 3

// shotWhileMoving emits one Mistake per weapon_fire event where the shooter's
// most-recent sampled speed exceeds standingShotMaxSpeed. Skips knife and
// grenade weapons (CS2 does not penalize moving with these), fires from
// attackers with no prior tick row available (first sample after a respawn
// or round start — pre-fire velocity is unknown), and fires without a
// non-empty AttackerSteamID.
func shotWhileMoving(events []demo.GameEvent, idx PerPlayerTickIndex, _ float64) []Mistake {
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
		row, ok := nearestTick(idx, ev.AttackerSteamID, ev.Tick)
		if !ok {
			continue
		}
		// First sample after a respawn / round transition has Vx=Vy=0 by
		// construction (the parser resets prevAnalysisPos across pre-match
		// restart and the first tick of any new round implicitly produces a
		// large delta the analyzer would over-flag). Skip rows with both
		// components exactly zero — the analyzer treats this as "no prior
		// velocity available".
		if row.Vx == 0 && row.Vy == 0 {
			continue
		}
		speed := math.Sqrt(float64(row.Vx)*float64(row.Vx) + float64(row.Vy)*float64(row.Vy))
		if speed <= standingShotMaxSpeed {
			continue
		}
		out = append(out, Mistake{
			SteamID:     ev.AttackerSteamID,
			RoundNumber: ev.RoundNumber,
			Tick:        ev.Tick,
			Kind:        string(MistakeKindShotWhileMoving),
			Extras: map[string]any{
				"speed":  speed,
				"weapon": ev.Weapon,
			},
		})
	}
	return out
}

// counterStrafeAtFire reports whether the attacker stopped (current speed
// below standingShotMaxSpeed) after running (some sample within the lookback
// window above counterStrafeArmedSpeed). Returns (curSpeed, prevPeak, true)
// when the predicate fires; (curSpeed, _, false) when current speed is too
// high or no armed sample is available. Reused by the per-fire counter-
// strafe rule and the per-player aggregate.
func counterStrafeAtFire(idx PerPlayerTickIndex, steamID string, fireTick int) (float64, float64, bool) {
	cur, ok := nearestTick(idx, steamID, fireTick)
	if !ok {
		return 0, 0, false
	}
	curSpeed := math.Sqrt(float64(cur.Vx)*float64(cur.Vx) + float64(cur.Vy)*float64(cur.Vy))
	if curSpeed > standingShotMaxSpeed {
		return curSpeed, 0, false
	}
	prev := previousTickRows(idx, steamID, fireTick, counterStrafeLookbackSamples)
	if len(prev) == 0 {
		return curSpeed, 0, false
	}
	peak := 0.0
	for _, row := range prev {
		s := math.Sqrt(float64(row.Vx)*float64(row.Vx) + float64(row.Vy)*float64(row.Vy))
		if s > peak {
			peak = s
		}
	}
	if peak < counterStrafeArmedSpeed {
		return curSpeed, peak, false
	}
	return curSpeed, peak, true
}

// noCounterStrafe emits one Mistake per weapon_fire event where the attacker
// was moving above standingShotMaxSpeed and had no recent counter-strafe stop
// in the lookback window. Skips knife / grenade weapons and fires from
// attackers without a sampled tick row. The signal complements
// shotWhileMoving by giving credit when the player did counter-strafe before
// firing — i.e. a moving player who genuinely stopped right before the shot
// is *not* flagged here even though they would still miss the standing-shot
// threshold by a hair.
func noCounterStrafe(events []demo.GameEvent, idx PerPlayerTickIndex, _ float64) []Mistake {
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
		row, ok := nearestTick(idx, ev.AttackerSteamID, ev.Tick)
		if !ok {
			continue
		}
		curSpeed := math.Sqrt(float64(row.Vx)*float64(row.Vx) + float64(row.Vy)*float64(row.Vy))
		// Only fire the rule on shots above the standing threshold — those
		// are the ones where counter-strafing would have helped. Below the
		// threshold the shot is already accurate; flagging it as
		// "no counter-strafe" would be misleading.
		if curSpeed <= standingShotMaxSpeed {
			continue
		}
		_, _, didStrafe := counterStrafeAtFire(idx, ev.AttackerSteamID, ev.Tick)
		if didStrafe {
			continue
		}
		out = append(out, Mistake{
			SteamID:     ev.AttackerSteamID,
			RoundNumber: ev.RoundNumber,
			Tick:        ev.Tick,
			Kind:        string(MistakeKindNoCounterStrafe),
			Extras: map[string]any{
				"speed":  curSpeed,
				"weapon": ev.Weapon,
			},
		})
	}
	return out
}

// isNonShotWeapon returns true for weapons whose accuracy is unaffected by
// movement (knife slashes and thrown grenades). The parser's WeaponFire
// handler already filters to firearm classes, but a defensive check here
// keeps the rule honest if the upstream filter ever loosens.
func isNonShotWeapon(weapon string) bool {
	w := strings.ToLower(weapon)
	switch w {
	case "knife", "knife_t":
		return true
	}
	if strings.Contains(w, "grenade") || strings.Contains(w, "molotov") || strings.Contains(w, "incendiary") || strings.Contains(w, "decoy") || w == "flashbang" || w == "smokegrenade" {
		return true
	}
	return false
}
