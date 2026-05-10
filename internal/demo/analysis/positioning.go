package analysis

import (
	"math"

	"github.com/ok2ju/oversite/internal/demo"
)

// isolatedPeekRadius is the planar distance (CS2 world units) within which
// an alive teammate counts as "supporting" the player at their death tick.
// 600 u matches the plan; ~one mid-range engagement distance.
const isolatedPeekRadius = 600.0

// isolatedPeekLookbackTicks is how far back we scan for "any alive teammate
// within radius". 64 game ticks = 1 s at 64 tickrate, matching the plan.
const isolatedPeekLookbackTicks = 64

// repeatedDeathCellSize is the grid edge (world units) used to bucket death
// positions for the "you keep dying in the same spot" rule. 200 u matches
// the plan; small enough to discriminate distinct positions and large enough
// to absorb the noise of where on a doorway the body lands.
const repeatedDeathCellSize = 200.0

// repeatedDeathThreshold is how many deaths in the same grid cell before we
// flag a death zone. 3 matches the plan: two deaths in the same spot is
// coincidence, three is a habit.
const repeatedDeathThreshold = 3

// isolatedPeek emits one Mistake per kill where, at the victim's death tick
// AND across the prior isolatedPeekLookbackTicks game ticks, no living
// teammate of the victim was within isolatedPeekRadius. Skips world / self /
// friendly-fire kills (mirrors trades.go) and kills missing the victim's
// team in KillExtra.
//
// The rule needs the tick index to find teammate positions; when the index
// is empty we return no mistakes (the rule degenerates silently — same
// pattern as crosshairTooLow).
func isolatedPeek(events []demo.GameEvent, idx PerPlayerTickIndex, rounds []demo.RoundData) []Mistake {
	if len(events) == 0 || len(idx.Rows) == 0 {
		return nil
	}
	teamsByRound := teamsByRoundFromRosters(rounds)
	out := make([]Mistake, 0, 8)
	for _, ev := range events {
		if ev.Type != "kill" {
			continue
		}
		if ev.VictimSteamID == "" || ev.AttackerSteamID == "" {
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
		teams := teamsByRound[ev.RoundNumber]
		if len(teams) == 0 {
			continue
		}
		victimTeam := teams[ev.VictimSteamID]
		if victimTeam == "" {
			victimTeam = k.VictimTeam
		}
		if hasNearbyTeammate(idx, teams, victimTeam, ev.VictimSteamID, ev.X, ev.Y, ev.Tick) {
			continue
		}
		out = append(out, Mistake{
			SteamID:     ev.VictimSteamID,
			RoundNumber: ev.RoundNumber,
			Tick:        ev.Tick,
			Kind:        string(MistakeKindIsolatedPeek),
			Extras: map[string]any{
				"x":      ev.X,
				"y":      ev.Y,
				"weapon": ev.Weapon,
			},
		})
	}
	return out
}

// hasNearbyTeammate scans the lookback window for any sample of any alive
// teammate (same team, different steamID) inside isolatedPeekRadius of the
// victim's death position. Uses tickRangeRows so each teammate contributes
// only the samples that land in [deathTick-lookback, deathTick] rather than
// the full per-player history.
func hasNearbyTeammate(idx PerPlayerTickIndex, teams map[string]string, victimTeam, victimSteam string, deathX, deathY float64, deathTick int) bool {
	lo := deathTick - isolatedPeekLookbackTicks
	for steam := range idx.Rows {
		if steam == victimSteam {
			continue
		}
		side, ok := teams[steam]
		if !ok || side == "" || side != victimTeam {
			continue
		}
		rows := tickRangeRows(idx, steam, lo, deathTick)
		for _, row := range rows {
			if !row.IsAlive {
				continue
			}
			dx := float64(row.X) - deathX
			dy := float64(row.Y) - deathY
			if math.Sqrt(dx*dx+dy*dy) <= isolatedPeekRadius {
				return true
			}
		}
	}
	return false
}

// repeatedDeathZones emits one Mistake per kill that pushes the victim's
// per-(player, grid-cell) death count to or past repeatedDeathThreshold.
// We flag every death from the threshold onward, not just the threshold
// itself, so the timeline shows a cluster forming rather than a single
// arbitrary entry.
//
// Grid cells are anchored to (0, 0) and use repeatedDeathCellSize as the
// edge. This bucketing is intentionally coarse — it's a "you keep dying in
// the same area" signal, not "you died on the exact same pixel."
func repeatedDeathZones(events []demo.GameEvent) []Mistake {
	if len(events) == 0 {
		return nil
	}
	type cellKey struct {
		steam string
		gx    int
		gy    int
	}
	counts := make(map[cellKey]int, 16)
	out := make([]Mistake, 0, 8)
	for _, ev := range events {
		if ev.Type != "kill" {
			continue
		}
		if ev.VictimSteamID == "" {
			continue
		}
		if ev.X == 0 && ev.Y == 0 {
			continue // missing position — better to skip than misbucket.
		}
		gx := int(math.Floor(ev.X / repeatedDeathCellSize))
		gy := int(math.Floor(ev.Y / repeatedDeathCellSize))
		key := cellKey{steam: ev.VictimSteamID, gx: gx, gy: gy}
		counts[key]++
		if counts[key] < repeatedDeathThreshold {
			continue
		}
		out = append(out, Mistake{
			SteamID:     ev.VictimSteamID,
			RoundNumber: ev.RoundNumber,
			Tick:        ev.Tick,
			Kind:        string(MistakeKindRepeatedDeathZone),
			Extras: map[string]any{
				"x":     ev.X,
				"y":     ev.Y,
				"count": counts[key],
			},
		})
	}
	return out
}
