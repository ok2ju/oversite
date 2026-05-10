package analysis

import (
	"math"
	"strings"

	"github.com/ok2ju/oversite/internal/demo"
)

// standingShotMaxSpeed is the planar speed (CS2 world units per second) at
// or below which the attacker is considered "standing still" for the
// shot-while-moving rule. 30 u/s mirrors strafeMinSpeed = 60 in
// internal/demo/player_stats.go: the standing-still threshold sits below the
// strafing threshold because "shot while moving" is a stricter signal than
// "is strafing" — a player can move slowly without strafe-aiming yet still
// suffer accuracy loss at 30+ u/s.
const standingShotMaxSpeed = 30.0

// shotWhileMoving emits one Mistake per weapon_fire event where the shooter's
// most-recent sampled speed exceeds standingShotMaxSpeed. Skips knife and
// grenade weapons (CS2 does not penalize moving with these), fires from
// attackers with no prior tick row available (first sample after a respawn
// or round start — pre-fire velocity is unknown), and fires without a
// non-empty AttackerSteamID.
func shotWhileMoving(events []demo.GameEvent, idx PerPlayerTickIndex, _ float64) []Mistake {
	if len(events) == 0 || len(idx) == 0 {
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
