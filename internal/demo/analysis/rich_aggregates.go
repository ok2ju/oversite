package analysis

import (
	"math"

	"github.com/ok2ju/oversite/internal/demo"
)

// MatchAggregates is the per-(demo, player) rollup that backs the persisted
// columns on player_match_analysis. Each field maps 1:1 to a column.
// Computing them lives here rather than scattered across each rule file
// because every aggregate is a different roll-up over the same event / tick
// streams; isolating the math keeps RunMatchSummary readable.
//
// All fields default to zero — players who didn't fire / didn't throw / didn't
// die in the relevant scenarios produce zero counts, which the persistence
// layer happily writes (the column defaults are 0, so an absent player ends
// up with the same row a zero-event player would).
type MatchAggregates struct {
	// Aim.
	TimeToFireMsSum   float64 // running sum of reaction_ms for slow_reaction flags.
	TimeToFireMsCount int
	FlickFires        int // total flick-class fires (yaw delta past threshold).
	FlickHits         int // flick-class fires that landed (HitVictimSteamID populated).
	// Spray.
	FirstShotFires int
	FirstShotHits  int
	SprayDecaySum  float64 // sum of (8 / shot_index) used as a proxy for the decay slope per flagged spray; light-weight.
	SprayDecayCnt  int
	// Movement.
	CounterStrafeFires int
	CounterStrafeHits  int // counter-strafe SUCCESSES — fires where the player did stop after running.
	StandingShotFires  int
	StandingShotHits   int
	// Utility.
	SmokesThrown     int
	SmokesKillAssist int
	FlashAssists     int
	HeDamage         int
	// Positioning.
	IsolatedPeekDeaths int
	RepeatedDeathZones int
	// Economy.
	FullBuyDamage int
	FullBuyRounds int
	EcoKills      int
}

// computeMatchAggregates walks the events / mistakes / round data once and
// produces the per-player aggregate map used by RunMatchSummary. Mistakes
// must be the slice returned by Run — most counts come straight off it.
//
// idx may be empty (legacy fixtures with no AnalysisTick fanout); aim /
// movement counts that depend on tick samples degenerate to zero in that
// case — callers shouldn't fail on missing data.
func computeMatchAggregates(events []demo.GameEvent, rounds []demo.RoundData, idx PerPlayerTickIndex, mistakes []Mistake) map[string]*MatchAggregates {
	if len(events) == 0 && len(rounds) == 0 {
		return nil
	}
	out := make(map[string]*MatchAggregates, 16)
	ensure := func(steam string) *MatchAggregates {
		a, ok := out[steam]
		if !ok {
			a = &MatchAggregates{}
			out[steam] = a
		}
		return a
	}

	// 1) Mistake-driven counts (rules already did the heavy lifting; we just
	//    bucket them).
	for _, m := range mistakes {
		switch m.Kind {
		case string(MistakeKindSlowReaction):
			a := ensure(m.SteamID)
			a.TimeToFireMsSum += extractFloat(m.Extras, "reaction_ms")
			a.TimeToFireMsCount++
		case string(MistakeKindSprayDecay):
			a := ensure(m.SteamID)
			a.SprayDecaySum += extractFloat(m.Extras, "burst_hit_pct")
			a.SprayDecayCnt++
		case string(MistakeKindFlashAssist):
			ensure(m.SteamID).FlashAssists++
		case string(MistakeKindHeDamage):
			a := ensure(m.SteamID)
			a.HeDamage += int(extractFloat(m.Extras, "damage"))
		case string(MistakeKindIsolatedPeek):
			ensure(m.SteamID).IsolatedPeekDeaths++
		case string(MistakeKindRepeatedDeathZone):
			ensure(m.SteamID).RepeatedDeathZones++
		}
	}

	// 2) Direct event walks — fires, hits, smoke throws, eco kills.
	lastFireTickByShooter := make(map[string]int, 16)
	for _, ev := range events {
		switch ev.Type {
		case "weapon_fire":
			if ev.AttackerSteamID == "" {
				continue
			}
			a := ensure(ev.AttackerSteamID)
			extra, _ := ev.ExtraData.(*demo.WeaponFireExtra)
			// First-shot accuracy: first fire after firstShotIdleTicks idleness
			// for the same shooter.
			prev, hadPrev := lastFireTickByShooter[ev.AttackerSteamID]
			lastFireTickByShooter[ev.AttackerSteamID] = ev.Tick
			isFirstShot := !hadPrev || ev.Tick-prev >= firstShotIdleTicks
			if isFirstShot && !isNonShotWeapon(ev.Weapon) {
				a.FirstShotFires++
				if extra != nil && extra.HitVictimSteamID != "" {
					a.FirstShotHits++
				}
			}
			// Standing-shot stats — needs idx; we count fires that happened at
			// low speed and call those "standing".
			if !isNonShotWeapon(ev.Weapon) && len(idx.Rows) > 0 {
				a.StandingShotFires++
				if row, ok := nearestTick(idx, ev.AttackerSteamID, ev.Tick); ok {
					speed := math.Sqrt(float64(row.Vx)*float64(row.Vx) + float64(row.Vy)*float64(row.Vy))
					if speed <= standingShotMaxSpeed {
						a.StandingShotHits++
					}
					a.CounterStrafeFires++
					if _, _, didStrafe := counterStrafeAtFire(idx, ev.AttackerSteamID, ev.Tick); didStrafe {
						a.CounterStrafeHits++
					}
				}
			}
			// Flick hit/total — every fire whose lookback yaw delta exceeds the
			// flick threshold counts as a flick attempt; landing flicks
			// (HitVictimSteamID populated) increment hits. Hit pct is
			// FlickHits / FlickFires.
			if !isNonShotWeapon(ev.Weapon) && extra != nil && len(idx.Rows) > 0 {
				if delta, ok := flickDeltaDeg(idx, ev.AttackerSteamID, ev.Tick, extra.Yaw); ok {
					if math.Abs(delta) > flickHalfAngleDeg {
						a.FlickFires++
						if extra.HitVictimSteamID != "" {
							a.FlickHits++
						}
					}
				}
			}
		case "grenade_throw":
			if ev.AttackerSteamID == "" {
				continue
			}
			a := ensure(ev.AttackerSteamID)
			tok := normalizeUtilToken(ev.Weapon)
			if tok == "smokegrenade" {
				a.SmokesThrown++
			}
		}
	}

	// 3) Smoke kill-assist count — for each smoke that produced a teammate
	//    kill (the inverse of the unused_smoke mistake), credit the thrower.
	smokeAssists := computeSmokeAssistsByThrower(events, 64.0)
	for steam, n := range smokeAssists {
		ensure(steam).SmokesKillAssist += n
	}

	// 4) Round-driven aggregates (full-buy ADR, eco kills).
	roundDamage := computePerRoundDamage(events)
	for _, r := range rounds {
		// Classify each side's average buy.
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
		ctAvg := safeDiv(ctTotal, ctCount)
		tAvg := safeDiv(tTotal, tCount)
		for _, rp := range r.Roster {
			if rp.SteamID == "" {
				continue
			}
			val := demo.SumLoadoutValue(rp.Inventory)
			a := ensure(rp.SteamID)
			// Full-buy band per the plan: ≥ 3500.
			if val >= buyForceMax {
				a.FullBuyRounds++
				a.FullBuyDamage += roundDamage[rp.SteamID][r.Number]
			}
			// Eco kill: enemy was on eco AND this player got a kill in this
			// round with a strong loadout. Pre-compute the enemy avg for
			// the player's side.
			oppAvg := tAvg
			if rp.TeamSide == "T" {
				oppAvg = ctAvg
			}
			if oppAvg < buyEcoMax && val >= buyForceMax {
				a.EcoKills += roundKillsForPlayer(events, rp.SteamID, r.Number)
			}
		}
	}

	return out
}

// safeDiv returns total / count or 0 when count == 0. Avoids the per-call
// "if count == 0" check at every aggregate site.
func safeDiv(total, count int) int {
	if count == 0 {
		return 0
	}
	return total / count
}

// extractFloat reads a float from an extras map. Returns 0 when the key is
// absent or the value is not a number — analyzers use sentinel-zero semantics
// (no separate "missing" state).
func extractFloat(extras map[string]any, key string) float64 {
	if extras == nil {
		return 0
	}
	v, ok := extras[key]
	if !ok || v == nil {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	}
	return 0
}

// computePerRoundDamage walks player_hurt events and returns
// damage[attacker][round] = total HP damage dealt.
func computePerRoundDamage(events []demo.GameEvent) map[string]map[int]int {
	out := make(map[string]map[int]int, 16)
	for _, ev := range events {
		if ev.Type != "player_hurt" {
			continue
		}
		if ev.AttackerSteamID == "" {
			continue
		}
		h, _ := ev.ExtraData.(*demo.PlayerHurtExtra)
		if h == nil {
			continue
		}
		dmg := h.HealthDamage
		if dmg <= 0 {
			continue
		}
		byRound, ok := out[ev.AttackerSteamID]
		if !ok {
			byRound = make(map[int]int, 4)
			out[ev.AttackerSteamID] = byRound
		}
		byRound[ev.RoundNumber] += dmg
	}
	return out
}

// roundKillsForPlayer returns the count of kills the supplied player landed
// in the given round. World / self / FF kills are skipped (mirrors trades).
func roundKillsForPlayer(events []demo.GameEvent, steamID string, round int) int {
	n := 0
	for _, ev := range events {
		if ev.RoundNumber != round || ev.Type != "kill" {
			continue
		}
		if ev.AttackerSteamID != steamID {
			continue
		}
		if ev.AttackerSteamID == ev.VictimSteamID {
			continue
		}
		k, _ := ev.ExtraData.(*demo.KillExtra)
		if k == nil {
			continue
		}
		if k.AttackerTeam != "" && k.AttackerTeam == k.VictimTeam {
			continue
		}
		n++
	}
	return n
}

// computeSmokeAssistsByThrower walks the same smoke detonations
// smokeEffectiveness considers but flips the predicate — count the smokes
// that DID produce a teammate kill within the radius/window. Returns
// thrower-keyed counts.
func computeSmokeAssistsByThrower(events []demo.GameEvent, tickRate float64) map[string]int {
	if tickRate <= 0 {
		tickRate = 64
	}
	windowTicks := int(smokeAssistWindowSecs * tickRate)
	out := make(map[string]int, 8)
	for i, ev := range events {
		if ev.Type != "smoke_start" {
			continue
		}
		if ev.AttackerSteamID == "" {
			continue
		}
		s := smokeRec{
			idx:     i,
			tick:    ev.Tick,
			round:   ev.RoundNumber,
			thrower: ev.AttackerSteamID,
			x:       ev.X,
			y:       ev.Y,
		}
		if smokeProducedAssist(events, s, windowTicks) {
			out[s.thrower]++
		}
	}
	return out
}

// computeRoundEconomy is the per-(player, round) economy roll-up — buy
// classification, money spent, nade usage. Returns a nested map keyed by
// [steamID][roundNumber]. Players absent from a round (substitutions) are
// absent from that round's inner map.
func computeRoundEconomy(events []demo.GameEvent, rounds []demo.RoundData) map[string]map[int]RoundEconomy {
	out := make(map[string]map[int]RoundEconomy, 16)
	thrownByPlayerRound := make(map[string]map[int]int, 16)
	for _, ev := range events {
		if ev.Type != "grenade_throw" || ev.AttackerSteamID == "" {
			continue
		}
		if !isUtilToken(normalizeUtilToken(ev.Weapon)) {
			continue
		}
		byRound, ok := thrownByPlayerRound[ev.AttackerSteamID]
		if !ok {
			byRound = make(map[int]int, 4)
			thrownByPlayerRound[ev.AttackerSteamID] = byRound
		}
		byRound[ev.RoundNumber]++
	}
	for _, r := range rounds {
		for _, rp := range r.Roster {
			if rp.SteamID == "" {
				continue
			}
			val := demo.SumLoadoutValue(rp.Inventory)
			eco := RoundEconomy{
				BuyType:    classifyBuy(val, r.Number),
				MoneySpent: val,
				Spawned:    countUtilFromInventory(rp.Inventory),
				Used:       thrownByPlayerRound[rp.SteamID][r.Number],
			}
			eco.Unused = eco.Spawned - eco.Used
			if eco.Unused < 0 {
				eco.Unused = 0
			}
			byRound, ok := out[rp.SteamID]
			if !ok {
				byRound = make(map[int]RoundEconomy, 4)
				out[rp.SteamID] = byRound
			}
			byRound[r.Number] = eco
		}
	}
	return out
}

// RoundEconomy is the per-(player, round) economy snapshot used by
// RunPlayerRoundAnalysis. Spawned / Used / Unused are nade counts.
type RoundEconomy struct {
	BuyType    string
	MoneySpent int
	Spawned    int
	Used       int
	Unused     int
}

// classifyBuy maps a loadout value (sum of equipment) to one of the canonical
// CS2 buy bands. Round numbers 1 and 13 always classify as pistol regardless
// of loadout — the implicit "you start with $800 + USP/Glock" makes the
// loadout-value comparison meaningless on those rounds.
func classifyBuy(value, roundNumber int) string {
	if roundNumber == pistolRoundFirstHalf || roundNumber == pistolRoundSecondHalf {
		return "pistol"
	}
	switch {
	case value < buyEcoMax:
		return "eco"
	case value < buyForceMax:
		return "force"
	default:
		return "full_buy"
	}
}

// countUtilFromInventory returns the number of grenade-type tokens in a
// freeze-end inventory string. Mirrors parseUtilFromInventory's filter (only
// counts true grenades, not kits / kevlar / weapons).
func countUtilFromInventory(inv string) int {
	tokens := parseUtilFromInventory(inv)
	return len(tokens)
}

// computeRoundShots returns the per-(player, round) fire and hit counts used
// by the analysis page's per-round bar chart. shotsHit reuses the same
// HitVictimSteamID predicate as firstShotAccuracy / sprayDecay.
func computeRoundShots(events []demo.GameEvent) map[string]map[int][2]int {
	out := make(map[string]map[int][2]int, 16)
	for _, ev := range events {
		if ev.Type != "weapon_fire" || ev.AttackerSteamID == "" {
			continue
		}
		if isNonShotWeapon(ev.Weapon) {
			continue
		}
		extra, _ := ev.ExtraData.(*demo.WeaponFireExtra)
		byRound, ok := out[ev.AttackerSteamID]
		if !ok {
			byRound = make(map[int][2]int, 4)
			out[ev.AttackerSteamID] = byRound
		}
		entry := byRound[ev.RoundNumber]
		entry[0]++
		if extra != nil && extra.HitVictimSteamID != "" {
			entry[1]++
		}
		byRound[ev.RoundNumber] = entry
	}
	return out
}

// matchAggregateAvgs maps a *MatchAggregates' running sums into the column-
// shaped float64s the persistence layer writes. Centralizing the divisions
// here keeps the persistence path declarative — it just reads the populated
// MatchSummaryRow.
func (a *MatchAggregates) AvgTimeToFireMs() float64 {
	if a.TimeToFireMsCount == 0 {
		return 0
	}
	return a.TimeToFireMsSum / float64(a.TimeToFireMsCount)
}

func (a *MatchAggregates) FirstShotPct() float64 {
	if a.FirstShotFires == 0 {
		return 0
	}
	return float64(a.FirstShotHits) / float64(a.FirstShotFires)
}

func (a *MatchAggregates) DecaySlope() float64 {
	if a.SprayDecayCnt == 0 {
		return 0
	}
	return a.SprayDecaySum / float64(a.SprayDecayCnt)
}

func (a *MatchAggregates) StandingPct() float64 {
	if a.StandingShotFires == 0 {
		return 0
	}
	return float64(a.StandingShotHits) / float64(a.StandingShotFires)
}

func (a *MatchAggregates) CounterStrafePctValue() float64 {
	if a.CounterStrafeFires == 0 {
		return 0
	}
	return float64(a.CounterStrafeHits) / float64(a.CounterStrafeFires)
}

func (a *MatchAggregates) FullBuyADR() float64 {
	if a.FullBuyRounds == 0 {
		return 0
	}
	return float64(a.FullBuyDamage) / float64(a.FullBuyRounds)
}

func (a *MatchAggregates) FlickHitPctValue() float64 {
	if a.FlickFires == 0 {
		return 0
	}
	return float64(a.FlickHits) / float64(a.FlickFires)
}
