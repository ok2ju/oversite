package detectors

import (
	"math"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

// MissedFirstShotMaxDistance — analysis §4.2 excludes long-range
// first-shot misses (AWPs / long-line peeks): only contacts under
// 1500u from the nearest contact-enemy fire the rule.
const MissedFirstShotMaxDistance = 1500.0

// MissedFirstShot implements analysis §4.2 "First weapon_fire from P
// has no hit_victim_steam_id AND distance < 1500u". Walks ctx.Events
// (not c.Signals — SignalWeaponFireHit only fires for hits) to find
// the first subject-aggressor weapon_fire inside the contact window.
//
// Extras: shot_tick, weapon, distance.
func MissedFirstShot(c *contacts.Contact, ctx *DetectorCtx) []ContactMistake {
	if c.Extras.WallbangTaken {
		return nil
	}
	firstShot, ok := firstSubjectShot(c, ctx)
	if !ok {
		return nil
	}
	fe, _ := firstShot.ExtraData.(*demo.WeaponFireExtra)
	if fe != nil && fe.HitVictimSteamID != "" {
		return nil
	}
	// Only consider real firearms (knives / grenades return false).
	if _, known := ctx.Weapons.Lookup(firstShot.Weapon); !known {
		return nil
	}
	enemyDist, ok := nearestEnemyDistanceAtTick(c, ctx, firstShot)
	if !ok || enemyDist > MissedFirstShotMaxDistance {
		return nil
	}
	tick := int32(firstShot.Tick)
	extras := map[string]any{
		"shot_tick": firstShot.Tick,
		"weapon":    firstShot.Weapon,
		"distance":  enemyDist,
	}
	return []ContactMistake{NewContactMistake("missed_first_shot", &tick, extras)}
}

// firstSubjectShot returns the first weapon_fire event by c.Subject in
// [ClampPreLookback, c.TLast].
func firstSubjectShot(c *contacts.Contact, ctx *DetectorCtx) (demo.GameEvent, bool) {
	lower := ClampPreLookback(c, ctx)
	for _, evt := range ctx.Events {
		if int32(evt.Tick) < lower {
			continue
		}
		if int32(evt.Tick) > c.TLast {
			break
		}
		if evt.Type == "weapon_fire" && evt.AttackerSteamID == c.Subject {
			return evt, true
		}
	}
	return demo.GameEvent{}, false
}

// nearestEnemyDistanceAtTick walks every enemy in c.Enemies and returns
// the minimum 2D distance from the subject (at the shot tick) to that
// enemy (at the shot tick). ok=false when neither the subject nor any
// enemy has a tick sample at evt.Tick.
func nearestEnemyDistanceAtTick(c *contacts.Contact, ctx *DetectorCtx, evt demo.GameEvent) (float64, bool) {
	subjectRow, ok := nearestTick(ctx.Ticks, c.Subject, evt.Tick)
	if !ok {
		return 0, false
	}
	best := math.MaxFloat64
	any := false
	for _, e := range c.Enemies {
		row, ok := nearestTick(ctx.Ticks, e, evt.Tick)
		if !ok {
			continue
		}
		d := math.Hypot(float64(row.X-subjectRow.X), float64(row.Y-subjectRow.Y))
		if d < best {
			best = d
			any = true
		}
	}
	if !any {
		return 0, false
	}
	return best, true
}
