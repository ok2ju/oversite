package analysis

import (
	"github.com/ok2ju/oversite/internal/demo"
)

// firstShotIdleTicks is the gap (sampled-tick equivalent doesn't apply here —
// these are raw game ticks) since the player's previous fire that qualifies
// the next fire as a "first shot". 30 game ticks ≈ 470 ms at 64 tickrate;
// matches the plan's "≥ 30 idle ticks" wording.
const firstShotIdleTicks = 30

// burstMinShots is the minimum number of fires-in-a-row, with each fire
// within burstMaxGapTicks of the previous, that qualifies the sequence as a
// "burst" eligible for the spray-decay rule.
const burstMinShots = 4

// burstMaxGapTicks bounds the inter-fire gap for shots to belong to the same
// burst. 16 ticks ≈ 250 ms at 64 tickrate — comfortably above the AK's
// fire interval (~100 ms) and below any deliberate trigger reset.
const burstMaxGapTicks = 16

// sprayDecayMinIndex is the shot index (1-based) past which we expect hit
// rate to have collapsed if recoil control is the issue. The plan says
// "past shot 7"; we flag every shot at index ≥ this with the burst's hit
// rate so far.
const sprayDecayMinIndex = 8

// sprayDecayMaxHitPct is the burst hit-rate ceiling under which the late
// shots of the burst are considered wasted. 10% (= 0.10) matches the plan.
const sprayDecayMaxHitPct = 0.10

// firstShotAccuracy emits one Mistake per weapon_fire event that follows at
// least firstShotIdleTicks of idleness for the same shooter and missed (no
// HitVictimSteamID populated by the shot-impact pairer). Skips knife /
// grenade weapons and fires without WeaponFireExtra.
//
// Why "missed" is HitVictimSteamID == "": the shot-impact pairer in
// shot_impacts.go fills HitVictimSteamID when a player_hurt event landed for
// the same attacker within a short window. Wall hits / pure misses leave it
// empty.
func firstShotAccuracy(events []demo.GameEvent) []Mistake {
	if len(events) == 0 {
		return nil
	}
	out := make([]Mistake, 0, 8)
	lastFireTick := make(map[string]int, 16)
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
		extra, _ := ev.ExtraData.(*demo.WeaponFireExtra)
		if extra == nil {
			continue
		}
		prev, hadPrev := lastFireTick[ev.AttackerSteamID]
		lastFireTick[ev.AttackerSteamID] = ev.Tick
		if !hadPrev {
			// The very first fire of the match for this player IS a first
			// shot by definition (there was no prior shot). Treat it the
			// same as the post-idle case.
		} else if ev.Tick-prev < firstShotIdleTicks {
			continue
		}
		if extra.HitVictimSteamID != "" {
			continue
		}
		out = append(out, Mistake{
			SteamID:     ev.AttackerSteamID,
			RoundNumber: ev.RoundNumber,
			Tick:        ev.Tick,
			Kind:        string(MistakeKindMissedFirstShot),
			Extras: map[string]any{
				"weapon": ev.Weapon,
			},
		})
	}
	return out
}

// sprayDecay emits Mistakes for shots inside a burst (≥ burstMinShots
// consecutive fires with gap ≤ burstMaxGapTicks) at index ≥ sprayDecayMinIndex
// when the burst's hit rate so far is below sprayDecayMaxHitPct. We flag the
// individual late shots rather than the whole burst so the side panel can
// seek straight to the wasted bullets.
//
// Skips knife / grenade weapons and fires without WeaponFireExtra. Bursts are
// per (steamID, weapon): switching weapons mid-spray ends the burst.
func sprayDecay(events []demo.GameEvent) []Mistake {
	if len(events) == 0 {
		return nil
	}
	type burstState struct {
		weapon   string
		lastTick int
		shotIdx  int
		hits     int
		// pendingFlags collects mistakes generated while the burst is still
		// running so we can drop them if the burst ends before reaching
		// burstMinShots — short bursts don't qualify for the rule.
		pendingFlags []Mistake
		started      bool
	}
	out := make([]Mistake, 0, 8)
	bursts := make(map[string]*burstState, 16)

	flushBurst := func(b *burstState) {
		if b == nil {
			return
		}
		if b.shotIdx >= burstMinShots {
			out = append(out, b.pendingFlags...)
		}
		b.pendingFlags = nil
		b.started = false
		b.shotIdx = 0
		b.hits = 0
	}

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
		extra, _ := ev.ExtraData.(*demo.WeaponFireExtra)
		if extra == nil {
			continue
		}
		b, ok := bursts[ev.AttackerSteamID]
		if !ok {
			b = &burstState{}
			bursts[ev.AttackerSteamID] = b
		}
		gap := ev.Tick - b.lastTick
		sameWeapon := b.weapon == ev.Weapon
		if !b.started || !sameWeapon || gap > burstMaxGapTicks {
			flushBurst(b)
			b.weapon = ev.Weapon
			b.started = true
		}
		b.shotIdx++
		b.lastTick = ev.Tick
		if extra.HitVictimSteamID != "" {
			b.hits++
		}
		if b.shotIdx >= sprayDecayMinIndex {
			hitPct := float64(b.hits) / float64(b.shotIdx)
			if hitPct < sprayDecayMaxHitPct {
				b.pendingFlags = append(b.pendingFlags, Mistake{
					SteamID:     ev.AttackerSteamID,
					RoundNumber: ev.RoundNumber,
					Tick:        ev.Tick,
					Kind:        string(MistakeKindSprayDecay),
					Extras: map[string]any{
						"shot_index":    b.shotIdx,
						"burst_hit_pct": hitPct,
						"weapon":        ev.Weapon,
					},
				})
			}
		}
	}
	for _, b := range bursts {
		flushBurst(b)
	}
	return out
}
