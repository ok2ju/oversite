package contacts

import (
	"github.com/ok2ju/oversite/internal/demo"
)

// ContactOutcome enumerates the eight outcome labels from analysis §4.4.
// String values are the source of truth for both the Go layer and the
// contact_moments.outcome SQL column. Cross-referenced by the root
// types.go ContactOutcome — keep both lists in sync.
type ContactOutcome string

const (
	OutcomeWonClean           ContactOutcome = "won_clean"
	OutcomeWonDamaged         ContactOutcome = "won_damaged"
	OutcomeTradedWin          ContactOutcome = "traded_win"
	OutcomeTradedDeath        ContactOutcome = "traded_death"
	OutcomeUntradedDeath      ContactOutcome = "untraded_death"
	OutcomeDisengaged         ContactOutcome = "disengaged"
	OutcomePartialWin         ContactOutcome = "partial_win"
	OutcomeMutualDamageNoKill ContactOutcome = "mutual_damage_no_kill"
)

// ClassifyInputs is the per-contact input to Classify.
type ClassifyInputs struct {
	Contact     *Contact
	Subject     string
	SubjectTeam string

	// PostWindowKills are kills in (TLast, TLast + TradeWindowTicks] sorted
	// ascending by tick.
	PostWindowKills []demo.GameEvent

	// AliveAtTLast is the set of players alive at TLast keyed by SteamID.
	AliveAtTLast map[string]bool

	// EnemyTeam maps enemy steam_id to their team side ("CT"/"T").
	EnemyTeam map[string]string
}

// Classify returns the outcome label for a Contact.
func Classify(in ClassifyInputs) ContactOutcome {
	if in.Contact == nil {
		return OutcomeDisengaged
	}

	if in.Contact.Extras.FlashOnly {
		return OutcomeDisengaged
	}

	sum := summarize(in.Contact, in.Subject)
	enemyCount := len(in.Contact.Enemies)

	switch {
	case sum.subjectDied:
		if subjectKillerTraded(in, sum.subjectKillerSteam) {
			if len(sum.subjectKills) > 0 {
				return OutcomeTradedWin
			}
			return OutcomeTradedDeath
		}
		return OutcomeUntradedDeath

	case len(sum.subjectKills) >= enemyCount && enemyCount > 0:
		if sum.subjectDamageTaken <= 25 {
			return OutcomeWonClean
		}
		return OutcomeWonDamaged

	case len(sum.subjectKills) > 0:
		return OutcomePartialWin

	case anyDamageExchanged(in.Contact, in.Subject):
		return OutcomeMutualDamageNoKill

	default:
		return OutcomeDisengaged
	}
}

type killSummary struct {
	subjectKills       map[string]bool
	subjectDied        bool
	subjectKillerSteam string
	subjectDamageTaken int
}

func summarize(c *Contact, subject string) killSummary {
	s := killSummary{subjectKills: map[string]bool{}}
	_ = subject // signals already carry SubjectRole tags
	for i := range c.Signals {
		sig := c.Signals[i]
		switch sig.Kind {
		case SignalKill:
			if sig.Subject == SubjectAggressor {
				s.subjectKills[sig.EnemySteam] = true
			} else {
				s.subjectDied = true
				s.subjectKillerSteam = sig.EnemySteam
			}
		case SignalPlayerHurt, SignalUtilityDamage:
			if sig.Subject == SubjectVictim {
				s.subjectDamageTaken += sig.Extras.HealthDamage
			}
		}
	}
	return s
}

// subjectKillerTraded reports whether a teammate of the subject killed
// the player who killed the subject, within TradeWindowTicks after the
// subject's death tick.
func subjectKillerTraded(in ClassifyInputs, killerSteam string) bool {
	if killerSteam == "" {
		return false
	}
	subjectTeam := in.SubjectTeam
	if subjectTeam == "" {
		subjectTeam = subjectTeamFromEnemyTeam(in)
	}
	if subjectTeam == "" {
		return false
	}

	deathTick := subjectDeathTick(in.Contact, in.Subject)
	cutoff := deathTick + TradeWindowTicks

	for _, evt := range in.PostWindowKills {
		if evt.Type != "kill" {
			continue
		}
		if int32(evt.Tick) > cutoff {
			continue
		}
		if evt.VictimSteamID != killerSteam {
			continue
		}
		if evt.AttackerSteamID == "" {
			continue
		}
		if evt.AttackerSteamID == in.Subject {
			// Subject can't trade their own killer.
			continue
		}
		if attackerTeam, ok := in.EnemyTeam[evt.AttackerSteamID]; ok {
			if attackerTeam != subjectTeam {
				// Attacker is on the enemy team — not a trade.
				continue
			}
		}
		// Attacker is the subject's teammate (or unknown but on the subject's side).
		return true
	}
	return false
}

func subjectTeamFromEnemyTeam(in ClassifyInputs) string {
	for _, enemy := range in.Contact.Enemies {
		if team, ok := in.EnemyTeam[enemy]; ok {
			switch team {
			case "CT":
				return "T"
			case "T":
				return "CT"
			}
		}
	}
	return ""
}

func subjectDeathTick(c *Contact, subject string) int32 {
	_ = subject
	for i := range c.Signals {
		s := c.Signals[i]
		if s.Kind == SignalKill && s.Subject == SubjectVictim {
			return s.Tick
		}
	}
	return c.TLast
}

// anyDamageExchanged returns true if both sides took damage inside the
// contact (subject took damage AND subject dealt damage).
func anyDamageExchanged(c *Contact, subject string) bool {
	_ = subject
	gotHit, dealtDamage := false, false
	for i := range c.Signals {
		s := c.Signals[i]
		if s.Kind != SignalPlayerHurt && s.Kind != SignalUtilityDamage {
			continue
		}
		if s.Subject == SubjectVictim {
			gotHit = true
		} else {
			dealtDamage = true
		}
		if gotHit && dealtDamage {
			return true
		}
	}
	return false
}
