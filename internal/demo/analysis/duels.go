package analysis

import (
	"math"
	"sort"

	"github.com/ok2ju/oversite/internal/demo"
)

// Duel is a directed attacker→victim engagement reconstructed from the
// merged event stream (weapon_fire + player_hurt + kill). Fire-rule
// mistakes (shot_while_moving, no_counter_strafe, missed_first_shot,
// spray_decay) and kill-triggered mistakes (slow_reaction,
// caught_reloading, isolated_peek, repeated_death_zone, flash_assist)
// attach to the Duel they occurred inside. Cross-duel patterns
// (eco_misbuy, he_damage) carry a nil DuelID instead.
//
// MutualLocalID is the LocalID of the peer V→A duel when both players
// fired at each other in overlapping windows; -1 when no peer. The
// viewer renders linked mutuals adjacent so both bands are visible.
type Duel struct {
	// ID is populated by the persistence layer after CreateAnalysisDuel
	// returns. Detector output leaves it 0 — attribution uses LocalID.
	ID            int64
	LocalID       int
	RoundNumber   int
	AttackerSteam string
	VictimSteam   string
	StartTick     int
	EndTick       int
	Outcome       string
	EndReason     string
	HitConfirmed  bool
	HurtCount     int
	ShotCount     int
	MutualLocalID int
	Extras        map[string]any
}

const (
	DuelOutcomeWon          = "won"
	DuelOutcomeLost         = "lost"
	DuelOutcomeInconclusive = "inconclusive"
	DuelOutcomeWonTraded    = "won_then_traded"
	DuelOutcomeLostTraded   = "lost_but_traded"

	DuelEndReasonKill       = "kill"
	DuelEndReasonTrade      = "trade"
	DuelEndReasonGap        = "gap"
	DuelEndReasonConeSwitch = "cone_switch"
	DuelEndReasonRoundEnd   = "round_end"
	DuelEndReasonCleanKill  = "clean_kill"
)

const (
	duelConeHalfDeg              = 15.0
	duelGapWindowSeconds         = 3.0
	duelConeSwitchShots          = 3
	duelConeMaxDist              = 2200.0
	duelVictimRecentActivitySecs = 2.0
)

type stateKey struct {
	attacker string
	victim   string
}

type detectState struct {
	active      map[stateKey]*Duel
	closed      []*Duel
	round       int
	nextLocal   int
	switchCount map[string]int
	switchTo    map[string]string
}

// DetectDuels walks the (tick-ordered) event stream and returns directed
// duels. Per-round state: a map[attacker, victim]→*Duel of active
// engagements. weapon_fire either resolves to an existing active duel or
// opens a new one via pickTarget; player_hurt opens/updates one; kill
// resolves the duel. Stale duels (no event within gapTicks) expire
// inconclusive. A post-pass walks closed duels and flips outcome to
// won_then_traded / lost_but_traded when a trade kill landed inside
// TradeWindowSeconds.
//
// Returns duels sorted by (RoundNumber ASC, StartTick ASC).
func DetectDuels(events []demo.GameEvent, idx PerPlayerTickIndex, rounds []demo.RoundData, tickRate float64) []Duel {
	if len(events) == 0 {
		return nil
	}
	if tickRate <= 0 {
		tickRate = 64
	}
	gapTicks := int(duelGapWindowSeconds * tickRate)
	activityTicks := int(duelVictimRecentActivitySecs * tickRate)
	tradeWindowTicks := int(TradeWindowSeconds * tickRate)

	teamsByRound := teamsByRoundFromRosters(rounds)

	st := &detectState{
		active:      make(map[stateKey]*Duel, 16),
		switchCount: make(map[string]int, 16),
		switchTo:    make(map[string]string, 16),
		round:       -1,
		nextLocal:   1,
	}
	var all []*Duel

	// Sort defensively. Within a tick: weapon_fire first (opens duels),
	// player_hurt next (refines), kill last (closes).
	sorted := make([]demo.GameEvent, len(events))
	copy(sorted, events)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Tick != sorted[j].Tick {
			return sorted[i].Tick < sorted[j].Tick
		}
		return eventTypePriority(sorted[i].Type) < eventTypePriority(sorted[j].Type)
	})

	closeAllActive := func(reason, outcome string, atTick int) {
		for k, d := range st.active {
			d.EndTick = atTick
			if d.EndReason == "" {
				d.EndReason = reason
			}
			if d.Outcome == "" {
				d.Outcome = outcome
			}
			st.closed = append(st.closed, d)
			all = append(all, d)
			delete(st.active, k)
		}
		st.switchCount = make(map[string]int, 16)
		st.switchTo = make(map[string]string, 16)
	}

	expireStale := func(now int) {
		for k, d := range st.active {
			if now-d.EndTick > gapTicks {
				if d.EndReason == "" {
					d.EndReason = DuelEndReasonGap
				}
				if d.Outcome == "" {
					d.Outcome = DuelOutcomeInconclusive
				}
				st.closed = append(st.closed, d)
				all = append(all, d)
				delete(st.active, k)
				if st.switchTo[k.attacker] == k.victim {
					delete(st.switchCount, k.attacker)
					delete(st.switchTo, k.attacker)
				}
			}
		}
	}

	openDuel := func(attacker, victim string, tick, round int) *Duel {
		d := &Duel{
			LocalID:       st.nextLocal,
			RoundNumber:   round,
			AttackerSteam: attacker,
			VictimSteam:   victim,
			StartTick:     tick,
			EndTick:       tick,
			MutualLocalID: -1,
		}
		st.nextLocal++
		st.active[stateKey{attacker, victim}] = d
		return d
	}

	for _, ev := range sorted {
		if st.round == -1 {
			st.round = ev.RoundNumber
		}
		if ev.RoundNumber != st.round {
			endTick := ev.Tick - 1
			if endTick < st.round {
				endTick = ev.Tick
			}
			closeAllActive(DuelEndReasonRoundEnd, DuelOutcomeInconclusive, endTick)
			st.round = ev.RoundNumber
		}

		expireStale(ev.Tick)

		switch ev.Type {
		case "weapon_fire":
			handleFire(ev, st, idx, teamsByRound[ev.RoundNumber], activityTicks, openDuel)
		case "player_hurt":
			handleHurt(ev, st, openDuel)
		case "kill":
			d := handleKillEvent(ev, st, openDuel)
			if d != nil {
				st.closed = append(st.closed, d)
				all = append(all, d)
			}
		}
	}

	closeAllActive(DuelEndReasonRoundEnd, DuelOutcomeInconclusive, lastTick(sorted))

	annotateTrades(all, sorted, tradeWindowTicks)
	linkMutualDuels(all)

	out := make([]Duel, 0, len(all))
	for _, d := range all {
		if len(d.Extras) == 0 {
			d.Extras = nil
		}
		out = append(out, *d)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RoundNumber != out[j].RoundNumber {
			return out[i].RoundNumber < out[j].RoundNumber
		}
		if out[i].StartTick != out[j].StartTick {
			return out[i].StartTick < out[j].StartTick
		}
		return out[i].LocalID < out[j].LocalID
	})
	return out
}

func eventTypePriority(t string) int {
	switch t {
	case "weapon_fire":
		return 0
	case "player_hurt":
		return 1
	case "kill":
		return 2
	}
	return 3
}

func lastTick(events []demo.GameEvent) int {
	if len(events) == 0 {
		return 0
	}
	return events[len(events)-1].Tick
}

// handleFire resolves the attacker's shot to a target and opens / updates
// the duel. Unattributed shots (no target in cone, or multiple ambiguous
// candidates) are skipped — no duel, no mistake.
func handleFire(
	ev demo.GameEvent,
	st *detectState,
	idx PerPlayerTickIndex,
	teams map[string]string,
	activityTicks int,
	openDuel func(attacker, victim string, tick, round int) *Duel,
) {
	target := pickTarget(ev, idx, teams, activityTicks)
	if target == "" {
		return
	}
	wf, _ := ev.ExtraData.(*demo.WeaponFireExtra)
	hit := wf != nil && wf.HitVictimSteamID == target

	k := stateKey{ev.AttackerSteamID, target}
	d, ok := st.active[k]
	if !ok {
		d = openDuel(ev.AttackerSteamID, target, ev.Tick, ev.RoundNumber)
	}
	d.ShotCount++
	d.EndTick = ev.Tick
	if hit {
		d.HitConfirmed = true
		d.HurtCount++
	}

	// Update cone-switch tracker for the attacker.
	prev := st.switchTo[ev.AttackerSteamID]
	if prev == target {
		st.switchCount[ev.AttackerSteamID]++
	} else {
		st.switchTo[ev.AttackerSteamID] = target
		st.switchCount[ev.AttackerSteamID] = 1
	}
	if st.switchCount[ev.AttackerSteamID] >= duelConeSwitchShots {
		for kk, peer := range st.active {
			if kk.attacker != ev.AttackerSteamID || kk.victim == target {
				continue
			}
			peer.EndTick = ev.Tick
			if peer.EndReason == "" {
				peer.EndReason = DuelEndReasonConeSwitch
			}
			if peer.Outcome == "" {
				peer.Outcome = DuelOutcomeInconclusive
			}
			delete(st.active, kk)
		}
	}
}

// pickTarget resolves the victim for a weapon_fire event. Returns "" when
// the shot has no plausible target (spam at wall / smoke / empty space).
//
// Resolution order:
//
//  1. WeaponFireExtra.HitVictimSteamID — authoritative when shot-impact
//     pairing matched a player_hurt for the same attacker.
//  2. Cone enumeration — alive enemies within duelConeHalfDeg of the
//     attacker's yaw, within duelConeMaxDist, with a tick sample inside
//     the last activityTicks. Exactly one match returns that victim;
//     zero or multiple return "" (under-attribute, never mis-attribute).
func pickTarget(
	ev demo.GameEvent,
	idx PerPlayerTickIndex,
	teams map[string]string,
	activityTicks int,
) string {
	if ev.AttackerSteamID == "" {
		return ""
	}
	if isNonShotWeapon(ev.Weapon) {
		return ""
	}
	wf, _ := ev.ExtraData.(*demo.WeaponFireExtra)
	if wf != nil && wf.HitVictimSteamID != "" {
		return wf.HitVictimSteamID
	}
	if len(teams) == 0 {
		return ""
	}
	attackerTeam := teams[ev.AttackerSteamID]
	if attackerTeam == "" {
		return ""
	}
	aRow, ok := nearestTick(idx, ev.AttackerSteamID, ev.Tick)
	if !ok {
		return ""
	}
	var yaw float64
	if wf != nil {
		yaw = wf.Yaw
	} else {
		yaw = float64(aRow.Yaw)
	}
	best := ""
	bestCount := 0
	for victim, side := range teams {
		if victim == ev.AttackerSteamID || side == "" || side == attackerTeam {
			continue
		}
		vRow, ok := nearestTick(idx, victim, ev.Tick)
		if !ok || !vRow.IsAlive {
			continue
		}
		dx := float64(vRow.X - aRow.X)
		dy := float64(vRow.Y - aRow.Y)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > duelConeMaxDist {
			continue
		}
		expected := math.Atan2(dy, dx) * 180.0 / math.Pi
		delta := normalizeYawDeltaDeg(yaw - expected)
		if math.Abs(delta) > duelConeHalfDeg {
			continue
		}
		recent := previousTickRows(idx, victim, ev.Tick+1, 1)
		if len(recent) == 0 || ev.Tick-int(recent[0].Tick) > activityTicks {
			continue
		}
		bestCount++
		best = victim
		if bestCount > 1 {
			return ""
		}
	}
	if bestCount == 1 {
		return best
	}
	return ""
}

// handleHurt opens/updates an A→V duel based on a player_hurt event.
// Authoritative — we always know who got hit.
func handleHurt(
	ev demo.GameEvent,
	st *detectState,
	openDuel func(attacker, victim string, tick, round int) *Duel,
) {
	if ev.AttackerSteamID == "" || ev.VictimSteamID == "" {
		return
	}
	if ev.AttackerSteamID == ev.VictimSteamID {
		return
	}
	hurt, _ := ev.ExtraData.(*demo.PlayerHurtExtra)
	if hurt != nil && hurt.AttackerTeam != "" && hurt.AttackerTeam == hurt.VictimTeam {
		return
	}
	k := stateKey{ev.AttackerSteamID, ev.VictimSteamID}
	d, ok := st.active[k]
	if !ok {
		d = openDuel(ev.AttackerSteamID, ev.VictimSteamID, ev.Tick, ev.RoundNumber)
	}
	d.HitConfirmed = true
	d.HurtCount++
	d.EndTick = ev.Tick
}

// handleKillEvent resolves the duel for the kill. A→V kill closes the
// A→V duel as won. V→A active duels are closed inconclusive (the victim
// can't keep firing once dead). Returns the closed duel for appending.
func handleKillEvent(
	ev demo.GameEvent,
	st *detectState,
	openDuel func(attacker, victim string, tick, round int) *Duel,
) *Duel {
	if ev.AttackerSteamID == "" || ev.VictimSteamID == "" {
		return nil
	}
	if ev.AttackerSteamID == ev.VictimSteamID {
		return nil
	}
	kx, _ := ev.ExtraData.(*demo.KillExtra)
	if kx != nil && kx.AttackerTeam != "" && kx.AttackerTeam == kx.VictimTeam {
		return nil
	}
	k := stateKey{ev.AttackerSteamID, ev.VictimSteamID}
	d, ok := st.active[k]
	if !ok {
		// Synthetic clean-kill duel — no prior fires recorded.
		d = openDuel(ev.AttackerSteamID, ev.VictimSteamID, ev.Tick, ev.RoundNumber)
		d.EndReason = DuelEndReasonCleanKill
		d.HitConfirmed = true
	}
	d.Outcome = DuelOutcomeWon
	if d.EndReason == "" {
		if d.ShotCount > 0 {
			d.EndReason = DuelEndReasonKill
		} else {
			d.EndReason = DuelEndReasonCleanKill
			d.HitConfirmed = true
		}
	}
	d.EndTick = ev.Tick
	delete(st.active, k)
	if st.switchTo[ev.AttackerSteamID] == ev.VictimSteamID {
		delete(st.switchCount, ev.AttackerSteamID)
		delete(st.switchTo, ev.AttackerSteamID)
	}
	// Close any V→X duels: the victim is dead.
	for kk, peer := range st.active {
		if kk.attacker != ev.VictimSteamID {
			continue
		}
		peer.EndTick = ev.Tick
		if peer.EndReason == "" {
			peer.EndReason = DuelEndReasonGap
		}
		if peer.Outcome == "" {
			peer.Outcome = DuelOutcomeInconclusive
		}
		delete(st.active, kk)
	}
	return d
}

// annotateTrades flips outcome from won → won_then_traded when the
// killer subsequently died inside TradeWindowSeconds. The plan keeps the
// directional duel's outcome consistent ("won" with a "_traded" suffix)
// rather than introducing a "lost" state retroactively.
func annotateTrades(duels []*Duel, events []demo.GameEvent, tradeWindowTicks int) {
	if tradeWindowTicks <= 0 {
		return
	}
	type killKey struct {
		attacker string
		victim   string
		tick     int
	}
	kills := make(map[killKey]int, 32)
	for i, ev := range events {
		if ev.Type == "kill" {
			kills[killKey{ev.AttackerSteamID, ev.VictimSteamID, ev.Tick}] = i
		}
	}
	for _, d := range duels {
		if d.Outcome != DuelOutcomeWon {
			continue
		}
		idx, ok := kills[killKey{d.AttackerSteam, d.VictimSteam, d.EndTick}]
		if !ok {
			continue
		}
		limit := d.EndTick + tradeWindowTicks
		for j := idx + 1; j < len(events); j++ {
			next := events[j]
			if next.Tick > limit {
				break
			}
			if next.Type != "kill" {
				continue
			}
			if next.VictimSteamID != d.AttackerSteam {
				continue
			}
			if next.AttackerSteamID == "" || next.AttackerSteamID == d.AttackerSteam {
				continue
			}
			d.Outcome = DuelOutcomeWonTraded
			break
		}
	}
}

// linkMutualDuels pairs A→V and V→A duels whose tick windows overlap.
// Symmetric — both peers carry the other's LocalID.
func linkMutualDuels(duels []*Duel) {
	type pair struct {
		round int
		a, b  string
	}
	groups := make(map[pair][]*Duel, len(duels))
	for _, d := range duels {
		a, b := d.AttackerSteam, d.VictimSteam
		if a > b {
			a, b = b, a
		}
		k := pair{d.RoundNumber, a, b}
		groups[k] = append(groups[k], d)
	}
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		for i := 0; i < len(g); i++ {
			for j := i + 1; j < len(g); j++ {
				di, dj := g[i], g[j]
				if di.AttackerSteam == dj.AttackerSteam {
					continue
				}
				if di.MutualLocalID != -1 || dj.MutualLocalID != -1 {
					continue
				}
				if !ticksOverlap(di.StartTick, di.EndTick, dj.StartTick, dj.EndTick) {
					continue
				}
				di.MutualLocalID = dj.LocalID
				dj.MutualLocalID = di.LocalID
			}
		}
	}
}

func ticksOverlap(aStart, aEnd, bStart, bEnd int) bool {
	if aEnd < bStart || bEnd < aStart {
		return false
	}
	return true
}

// AttributeMistakesToDuels resolves each mistake's DuelID based on the
// kind → role mapping in the plan. Mistakes whose target / killer cannot
// be matched to a duel keep DuelID = nil (unattributed bucket).
//
// Mutates in place; returns the slice header for chainability.
func AttributeMistakesToDuels(mistakes []Mistake, duels []Duel, events []demo.GameEvent) []Mistake {
	if len(mistakes) == 0 || len(duels) == 0 {
		return mistakes
	}
	type bucket struct {
		round int
		steam string
	}
	byAttacker := make(map[bucket][]Duel, len(duels))
	byVictim := make(map[bucket][]Duel, len(duels))
	for _, d := range duels {
		byAttacker[bucket{d.RoundNumber, d.AttackerSteam}] = append(byAttacker[bucket{d.RoundNumber, d.AttackerSteam}], d)
		byVictim[bucket{d.RoundNumber, d.VictimSteam}] = append(byVictim[bucket{d.RoundNumber, d.VictimSteam}], d)
	}

	type fireKey struct {
		round    int
		attacker string
		tick     int
	}
	fireTargets := make(map[fireKey]string, 64)
	killTargets := make(map[fireKey]string, 32)
	for _, ev := range events {
		switch ev.Type {
		case "weapon_fire":
			if ev.AttackerSteamID == "" {
				continue
			}
			wf, _ := ev.ExtraData.(*demo.WeaponFireExtra)
			if wf != nil && wf.HitVictimSteamID != "" {
				fireTargets[fireKey{ev.RoundNumber, ev.AttackerSteamID, ev.Tick}] = wf.HitVictimSteamID
			}
		case "kill":
			if ev.AttackerSteamID == "" || ev.VictimSteamID == "" {
				continue
			}
			killTargets[fireKey{ev.RoundNumber, ev.AttackerSteamID, ev.Tick}] = ev.VictimSteamID
		}
	}

	for i := range mistakes {
		m := &mistakes[i]
		switch m.Kind {
		case string(MistakeKindShotWhileMoving),
			string(MistakeKindNoCounterStrafe),
			string(MistakeKindMissedFirstShot),
			string(MistakeKindSprayDecay):
			target := fireTargets[fireKey{m.RoundNumber, m.SteamID, m.Tick}]
			if id := pickAttackerDuel(byAttacker[bucket{m.RoundNumber, m.SteamID}], target, m.Tick); id > 0 {
				v := int64(id)
				m.DuelID = &v
			}
		case string(MistakeKindSlowReaction):
			victim := killTargets[fireKey{m.RoundNumber, m.SteamID, m.Tick}]
			if id := pickAttackerDuel(byAttacker[bucket{m.RoundNumber, m.SteamID}], victim, m.Tick); id > 0 {
				v := int64(id)
				m.DuelID = &v
			}
		case string(MistakeKindCaughtReloading),
			string(MistakeKindIsolatedPeek),
			string(MistakeKindRepeatedDeathZone):
			if id := pickVictimDuel(byVictim[bucket{m.RoundNumber, m.SteamID}], m.Tick); id > 0 {
				v := int64(id)
				m.DuelID = &v
			}
		case string(MistakeKindFlashAssist):
			if id := pickAnyDuelByTick(duels, m.RoundNumber, m.Tick); id > 0 {
				v := int64(id)
				m.DuelID = &v
			}
		case string(MistakeKindEcoMisbuy), string(MistakeKindHeDamage):
			// No duel — pattern signal. Leaves DuelID nil.
		}
	}
	return mistakes
}

func pickAttackerDuel(candidates []Duel, target string, tick int) int {
	best := 0
	bestScore := -1
	for _, d := range candidates {
		if tick < d.StartTick || tick > d.EndTick {
			continue
		}
		score := 0
		if target != "" && d.VictimSteam == target {
			score = 2
		} else if target == "" {
			score = 1
		}
		if score > bestScore {
			bestScore = score
			best = d.LocalID
		}
	}
	return best
}

func pickVictimDuel(candidates []Duel, tick int) int {
	best := 0
	bestScore := -1
	for _, d := range candidates {
		if tick < d.StartTick || tick > d.EndTick {
			continue
		}
		score := 1
		if d.EndTick == tick && (d.Outcome == DuelOutcomeWon || d.Outcome == DuelOutcomeWonTraded) {
			score = 2
		}
		if score > bestScore {
			bestScore = score
			best = d.LocalID
		}
	}
	return best
}

func pickAnyDuelByTick(duels []Duel, round, tick int) int {
	for _, d := range duels {
		if d.RoundNumber != round {
			continue
		}
		if tick >= d.StartTick && tick <= d.EndTick {
			return d.LocalID
		}
	}
	return 0
}
