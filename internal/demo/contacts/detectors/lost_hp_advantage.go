package detectors

import (
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

// LostHPAdvantageThreshold — analysis §4.2: at t_first, P.hp − E.hp >
// 25 but P dies in the contact.
const LostHPAdvantageThreshold = 25

// LostHPAdvantage emits when the subject opens the contact with an HP
// advantage over at least one enemy and still dies inside the contact.
// "HP advantage" is the worst-case (largest) lead vs. any enemy in
// c.Enemies — a 100hp subject vs. (100, 50) enemy stack reports 50 as
// the advantage.
//
// Health is derived from cumulative player_hurt events (AnalysisTick
// doesn't carry health in v1 — see 04 §4.4 for the deferred fix).
//
// Extras: subject_hp, worst_enemy, hp_advantage.
func LostHPAdvantage(c *contacts.Contact, ctx *DetectorCtx) []ContactMistake {
	if !subjectDiedInContact(c) {
		return nil
	}
	subjectHP := subjectHealthAt(c, ctx, c.TFirst)
	worstAdvantage := 0
	worstEnemy := ""
	for _, e := range c.Enemies {
		enemyHP := enemyHealthAt(ctx, e, c.TFirst)
		adv := subjectHP - enemyHP
		if adv > worstAdvantage {
			worstAdvantage = adv
			worstEnemy = e
		}
	}
	if worstAdvantage <= LostHPAdvantageThreshold || worstEnemy == "" {
		return nil
	}
	tick := c.TFirst
	extras := map[string]any{
		"subject_hp":   subjectHP,
		"worst_enemy":  worstEnemy,
		"hp_advantage": worstAdvantage,
	}
	return []ContactMistake{NewContactMistake("lost_hp_advantage", &tick, extras)}
}
