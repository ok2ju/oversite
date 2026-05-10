package analysis

import (
	"math"

	"github.com/ok2ju/oversite/internal/demo"
)

// MechanicalAgg is the per-player rollup of the slice-8 tick-driven rules.
// Counts are kept alongside the percentages so future slices can re-derive
// confidence intervals without re-walking the events.
type MechanicalAgg struct {
	Engagements     int     // total weapon_fire events for this player (firearm filter applied at the parser).
	CrosshairFlags  int     // fires flagged by crosshairTooLow.
	MovingFlags     int     // fires flagged by shotWhileMoving.
	AimPct          float64 // 1 - CrosshairFlags / Engagements (0 when Engagements == 0).
	StandingShotPct float64 // 1 - MovingFlags / Engagements (0 when Engagements == 0).
	AvgFireSpeed    float64 // mean planar speed at the most-recent sampled tick over this player's fires.
}

// computeMechanicalAggregates walks every weapon_fire event and aggregates
// per-player engagement counts plus the two slice-8 percentages. The flagged
// counts are read from the rule output rather than recomputed so this stays
// in lockstep with what's persisted to analysis_mistakes.
//
// AvgFireSpeed reads from the AnalysisTick at fire time — useful as a
// secondary metric on the analysis page's Movement card. Players with no
// fires (spectators) are absent from the returned map; aggregator callers
// should treat absence as "no row to persist".
func computeMechanicalAggregates(events []demo.GameEvent, idx PerPlayerTickIndex, mistakes []Mistake) map[string]MechanicalAgg {
	if len(events) == 0 {
		return nil
	}
	type accum struct {
		engagements    int
		crosshairFlags int
		movingFlags    int
		speedSum       float64
	}
	byPlayer := make(map[string]*accum, 16)
	for _, ev := range events {
		if ev.Type != "weapon_fire" {
			continue
		}
		if ev.AttackerSteamID == "" {
			continue
		}
		a, ok := byPlayer[ev.AttackerSteamID]
		if !ok {
			a = &accum{}
			byPlayer[ev.AttackerSteamID] = a
		}
		a.engagements++
		if len(idx.Rows) > 0 {
			if row, ok := nearestTick(idx, ev.AttackerSteamID, ev.Tick); ok {
				speed := math.Sqrt(float64(row.Vx)*float64(row.Vx) + float64(row.Vy)*float64(row.Vy))
				a.speedSum += speed
			}
		}
	}
	for _, m := range mistakes {
		switch m.Kind {
		case string(MistakeKindCrosshairTooLow):
			if a, ok := byPlayer[m.SteamID]; ok {
				a.crosshairFlags++
			}
		case string(MistakeKindShotWhileMoving):
			if a, ok := byPlayer[m.SteamID]; ok {
				a.movingFlags++
			}
		}
	}
	out := make(map[string]MechanicalAgg, len(byPlayer))
	for steamID, a := range byPlayer {
		ag := MechanicalAgg{
			Engagements:    a.engagements,
			CrosshairFlags: a.crosshairFlags,
			MovingFlags:    a.movingFlags,
		}
		if a.engagements > 0 {
			ag.AimPct = 1 - float64(a.crosshairFlags)/float64(a.engagements)
			ag.StandingShotPct = 1 - float64(a.movingFlags)/float64(a.engagements)
			ag.AvgFireSpeed = a.speedSum / float64(a.engagements)
		}
		out[steamID] = ag
	}
	return out
}

// computeRoundMechanicalAggregates is the per-(player, round) variant. Players
// who fired zero shots in a round are absent from that player's inner map.
func computeRoundMechanicalAggregates(events []demo.GameEvent, idx PerPlayerTickIndex, mistakes []Mistake) map[string]map[int]MechanicalAgg {
	if len(events) == 0 {
		return nil
	}
	type accum struct {
		engagements    int
		crosshairFlags int
		movingFlags    int
		speedSum       float64
	}
	byPlayerRound := make(map[string]map[int]*accum, 16)
	for _, ev := range events {
		if ev.Type != "weapon_fire" {
			continue
		}
		if ev.AttackerSteamID == "" {
			continue
		}
		byRound, ok := byPlayerRound[ev.AttackerSteamID]
		if !ok {
			byRound = make(map[int]*accum, 4)
			byPlayerRound[ev.AttackerSteamID] = byRound
		}
		a, ok := byRound[ev.RoundNumber]
		if !ok {
			a = &accum{}
			byRound[ev.RoundNumber] = a
		}
		a.engagements++
		if len(idx.Rows) > 0 {
			if row, ok := nearestTick(idx, ev.AttackerSteamID, ev.Tick); ok {
				speed := math.Sqrt(float64(row.Vx)*float64(row.Vx) + float64(row.Vy)*float64(row.Vy))
				a.speedSum += speed
			}
		}
	}
	for _, m := range mistakes {
		byRound, ok := byPlayerRound[m.SteamID]
		if !ok {
			continue
		}
		a, ok := byRound[m.RoundNumber]
		if !ok {
			continue
		}
		switch m.Kind {
		case string(MistakeKindCrosshairTooLow):
			a.crosshairFlags++
		case string(MistakeKindShotWhileMoving):
			a.movingFlags++
		}
	}
	out := make(map[string]map[int]MechanicalAgg, len(byPlayerRound))
	for steamID, byRound := range byPlayerRound {
		inner := make(map[int]MechanicalAgg, len(byRound))
		for roundNumber, a := range byRound {
			ag := MechanicalAgg{
				Engagements:    a.engagements,
				CrosshairFlags: a.crosshairFlags,
				MovingFlags:    a.movingFlags,
			}
			if a.engagements > 0 {
				ag.AimPct = 1 - float64(a.crosshairFlags)/float64(a.engagements)
				ag.StandingShotPct = 1 - float64(a.movingFlags)/float64(a.engagements)
				ag.AvgFireSpeed = a.speedSum / float64(a.engagements)
			}
			inner[roundNumber] = ag
		}
		out[steamID] = inner
	}
	return out
}
