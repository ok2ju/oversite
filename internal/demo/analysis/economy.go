package analysis

import (
	"github.com/ok2ju/oversite/internal/demo"
)

// Buy thresholds used by ecoMisbuy. Bands mirror community consensus: pistol
// round = round 1 + halftime restart; eco < 1500; force 1500–3499;
// full ≥ 3500.
const (
	buyEcoMax             = 1500
	buyForceMax           = 3500
	pistolRoundFirstHalf  = 1
	pistolRoundSecondHalf = 13
)

// ecoMisbuy emits one Mistake per (player, round) where the player's team
// went eco while the opposing team also went eco — i.e. both teams are
// broke and no one is set up to win the round. The signal "you should have
// force-bought" is admittedly judgmental; we tag the mistake on the
// freeze-end tick (round start) so the timeline cursor seeks to the buy
// menu, and let the suggestion explain.
func ecoMisbuy(rounds []demo.RoundData) []Mistake {
	if len(rounds) == 0 {
		return nil
	}
	out := make([]Mistake, 0, 8)
	for _, r := range rounds {
		if r.Number == pistolRoundFirstHalf || r.Number == pistolRoundSecondHalf {
			continue
		}
		var ctTotal, tTotal, ctCount, tCount int
		for _, rp := range r.Roster {
			val := demo.SumLoadoutValue(rp.Inventory)
			switch rp.TeamSide {
			case "CT":
				ctTotal += val
				ctCount++
			case "T":
				tTotal += val
				tCount++
			}
		}
		if ctCount == 0 || tCount == 0 {
			continue
		}
		ctAvg := ctTotal / ctCount
		tAvg := tTotal / tCount
		if ctAvg >= buyEcoMax && tAvg >= buyEcoMax {
			continue
		}
		// One side is full-eco. Flag every player on that side with the
		// "team eco'd while opposition could have been pressured" signal.
		for _, rp := range r.Roster {
			if rp.SteamID == "" {
				continue
			}
			val := demo.SumLoadoutValue(rp.Inventory)
			if val >= buyEcoMax {
				continue
			}
			oppAvg := tAvg
			if rp.TeamSide == "T" {
				oppAvg = ctAvg
			}
			if oppAvg >= buyForceMax {
				continue // they out-bought us anyway — a force wouldn't have helped.
			}
			out = append(out, Mistake{
				SteamID:     rp.SteamID,
				RoundNumber: r.Number,
				Tick:        r.FreezeEndTick,
				Kind:        string(MistakeKindEcoMisbuy),
				Extras: map[string]any{
					"loadout_value": val,
					"opp_avg_value": oppAvg,
				},
			})
		}
	}
	return out
}
