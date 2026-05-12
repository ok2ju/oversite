package detectors

import (
	"math"

	"github.com/ok2ju/oversite/internal/demo/contacts"
)

// IsolatedPeekRadius — analysis §4.1: no teammate alive within 800u of
// P at t_first.
const IsolatedPeekRadius = 800.0

// IsolatedPeek emits when no living teammate is within
// IsolatedPeekRadius of the subject at c.TFirst.
//
// Extras: t_first, nearest_teammate (omitted when no living teammate),
// nearest_distance.
func IsolatedPeek(c *contacts.Contact, ctx *DetectorCtx) []ContactMistake {
	subjectRow, ok := nearestTick(ctx.Ticks, c.Subject, int(c.TFirst))
	if !ok || !subjectRow.IsAlive {
		return nil
	}
	nearestDist := math.MaxFloat64
	var nearestTeammate string
	for steam, p := range ctx.Players {
		if steam == c.Subject {
			continue
		}
		if p.TeamSide != ctx.SubjectTeam {
			continue
		}
		teamRow, ok := nearestTick(ctx.Ticks, steam, int(c.TFirst))
		if !ok || !teamRow.IsAlive {
			continue
		}
		dx := float64(teamRow.X - subjectRow.X)
		dy := float64(teamRow.Y - subjectRow.Y)
		d := math.Sqrt(dx*dx + dy*dy)
		if d < nearestDist {
			nearestDist = d
			nearestTeammate = steam
		}
	}
	if nearestDist <= IsolatedPeekRadius {
		return nil
	}
	tick := c.TFirst
	extras := map[string]any{
		"t_first": int(c.TFirst),
	}
	if nearestTeammate != "" {
		extras["nearest_teammate"] = nearestTeammate
		extras["nearest_distance"] = nearestDist
	}
	return []ContactMistake{NewContactMistake("isolated_peek", &tick, extras)}
}
