// Package contacts builds per-(player, signal-cluster) contact moments
// from a parsed CS2 demo and persists them to contact_moments. Each
// contact carries an outcome label (analysis §4.4) and the boundaries
// (t_pre, t_first, t_last, t_post) the Phase 3 mistake detectors and
// Phase 4 timeline tooltips read.
//
// The Run entry point mirrors analysis.Run's shape: it takes the
// in-memory ParseResult + roundMap and returns the list of contacts.
// Persistence is a separate Persist call so callers (including tests)
// can inspect the build output before it hits SQLite.
//
// Phase 2 ships only the build + outcome classifier. The mistake
// detector pipeline lives in Phase 3 and writes to contact_mistakes.
// Phase 2's contacts surface with Mistakes == nil.
//
// Output ordering: Run returns contacts in (round_number, subject,
// t_first) order. Phase 3 detectors rely on the per-(round, subject)
// chronological order to read the previous contact's t_last as a
// pre-window cutoff hint.
package contacts

import (
	"fmt"

	"github.com/ok2ju/oversite/internal/demo"
)

// BuilderVersion is the compiled grouper+outcome version. Bumped when
// merge windows, boundary clamping, or outcome rules change.
const BuilderVersion = 1

// RunOpts mirrors analysis.RunOpts's empty-struct shape.
type RunOpts struct{}

// ContactMoment is the persisted shape produced by Run. The orchestrator
// fills RoundID, Outcome, and SignalCount before returning. ID and
// CreatedAt remain zero until Persist writes the row.
type ContactMoment struct {
	ID           int64          `json:"id"`
	DemoID       int64          `json:"demo_id"`
	RoundID      int64          `json:"round_id"`
	RoundNumber  int            `json:"round_number"`
	SubjectSteam string         `json:"subject_steam"`
	TFirst       int32          `json:"t_first"`
	TLast        int32          `json:"t_last"`
	TPre         int32          `json:"t_pre"`
	TPost        int32          `json:"t_post"`
	Enemies      []string       `json:"enemies"`
	Outcome      ContactOutcome `json:"outcome"`
	SignalCount  int            `json:"signal_count"`
	Extras       ContactExtras  `json:"extras"`
	Signals      []Signal       `json:"-"`
}

// Run builds all contact moments for every non-bot subject in every
// round of a parsed demo. Returns a flat slice ordered first by
// round_number, then by subject, then by t_first.
func Run(result *demo.ParseResult, roundMap map[int]int64, opts RunOpts) ([]ContactMoment, error) {
	_ = opts
	if result == nil {
		return nil, nil
	}
	tickRate := result.Header.TickRate
	if tickRate != 0 && (tickRate < 63.9 || tickRate > 64.1) {
		return nil, fmt.Errorf("contacts: unsupported tick rate %.2f (only 64-tick is supported in v1)", tickRate)
	}

	aliveByRound := buildAlivePerTick(result)
	eventsByRound := partitionEventsByRound(result.Events)
	visByRound := partitionVisibilityByRound(result.Visibility)
	teammateFlashes := partitionTeammateFlashesByRound(result.Events, result.Rounds)

	out := make([]ContactMoment, 0, len(result.Rounds)*10)

	for _, round := range result.Rounds {
		if round.Number == 0 {
			continue
		}
		if roundMap != nil {
			if _, ok := roundMap[round.Number]; !ok {
				continue
			}
		}

		roster := round.Roster
		enemyTeam := buildEnemyTeam(roster)
		events := eventsByRound[round.Number]
		vis := visByRound[round.Number]

		for _, subject := range roster {
			if !isHumanSubject(subject) {
				continue
			}
			subjectTeam := subject.TeamSide
			alive := deriveAliveRange(subject.SteamID, round, events)

			collectIn := CollectInputs{
				Subject:      subject.SteamID,
				SubjectTeam:  subjectTeam,
				Round:        round,
				Events:       events,
				Visibility:   vis,
				EnemyTeam:    enemyTeam,
				AlivePerTick: aliveByRound[round.Number],
				SubjectAlive: alive,
			}
			signals := CollectSignals(collectIn)
			if len(signals) == 0 {
				continue
			}

			built := Build(BuildInputs{
				Subject:       subject.SteamID,
				Round:         round,
				SubjectAlive:  alive,
				Signals:       signals,
				TeammateFlash: filterTeammateFlashesForSubject(teammateFlashes, round.Number, subject.SteamID),
			})

			var roundID int64
			if roundMap != nil {
				roundID = roundMap[round.Number]
			}

			for i := range built {
				c := &built[i]
				postKills := postWindowKills(events, c.TLast, round.EndTick)
				outcome := Classify(ClassifyInputs{
					Contact:         c,
					Subject:         subject.SteamID,
					SubjectTeam:     subjectTeam,
					PostWindowKills: postKills,
					AliveAtTLast:    aliveAtTick(aliveByRound[round.Number], c.TLast),
					EnemyTeam:       enemyTeam,
				})
				c.RoundID = roundID
				out = append(out, materialize(c, roundID, outcome))
			}
		}
	}

	return out, nil
}

// materialize converts a builder Contact + resolved round_id + outcome
// label into a persistable ContactMoment. clearTransient drops builder-
// internal state before serialization.
func materialize(c *Contact, roundID int64, outcome ContactOutcome) ContactMoment {
	c.clearTransient()
	enemies := append([]string(nil), c.Enemies...)
	return ContactMoment{
		RoundID:      roundID,
		RoundNumber:  c.RoundNumber,
		SubjectSteam: c.Subject,
		TFirst:       c.TFirst,
		TLast:        c.TLast,
		TPre:         c.TPre,
		TPost:        c.TPost,
		Enemies:      enemies,
		Outcome:      outcome,
		SignalCount:  len(c.Signals),
		Extras:       c.Extras,
		Signals:      c.Signals,
	}
}
