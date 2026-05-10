package analysis

import (
	"github.com/ok2ju/oversite/internal/demo"
)

// caughtReloadingClipThreshold is the AmmoClip value at or below which the
// victim's most-recent sample counts as "caught reloading". Zero is the
// unambiguous case (no rounds left in the chamber on a kill); a future tuning
// pass could expand to clip < weapon-specific reload threshold once a per-
// weapon max-clip table lives in the parser.
const caughtReloadingClipThreshold = 0

// caughtReloading emits one Mistake per kill where the victim's nearest
// AnalysisTick at or before the death has AmmoClip == 0. The signal is
// "you died holding an empty gun" — the canonical stop-firing-and-reload-
// behind-cover lesson. Skips world / friendly-fire / self kills (mirrors
// trades.go), kills with no sampled rows for the victim, and any kill where
// the victim's clip carries a positive count (the rule's premise doesn't
// hold; the death wasn't caused by being out of ammo).
//
// The rule needs the tick index to read AmmoClip; when the index is empty we
// return no mistakes (degenerate quietly — same convention as crosshairTooLow
// and isolatedPeek).
func caughtReloading(events []demo.GameEvent, idx PerPlayerTickIndex) []Mistake {
	if len(events) == 0 || len(idx.Rows) == 0 {
		return nil
	}
	out := make([]Mistake, 0, 8)
	for _, ev := range events {
		if ev.Type != "kill" {
			continue
		}
		if ev.AttackerSteamID == "" || ev.VictimSteamID == "" {
			continue
		}
		if ev.AttackerSteamID == ev.VictimSteamID {
			continue
		}
		k, _ := ev.ExtraData.(*demo.KillExtra)
		if k == nil || k.VictimTeam == "" {
			continue
		}
		if k.AttackerTeam != "" && k.AttackerTeam == k.VictimTeam {
			continue
		}
		row, ok := nearestTick(idx, ev.VictimSteamID, ev.Tick)
		if !ok {
			continue
		}
		if int(row.AmmoClip) > caughtReloadingClipThreshold {
			continue
		}
		out = append(out, Mistake{
			SteamID:     ev.VictimSteamID,
			RoundNumber: ev.RoundNumber,
			Tick:        ev.Tick,
			Kind:        string(MistakeKindCaughtReloading),
			Extras: map[string]any{
				"clip":   int(row.AmmoClip),
				"weapon": ev.Weapon,
			},
		})
	}
	return out
}

// flashAssistMinBlindSecs is the minimum blind duration that qualifies a
// teammate-assist credit as a "good flash" highlight. 1.0 s is what
// demoinfocs treats as the assist threshold internally; we mirror it here.
const flashAssistMinBlindSecs = 1.0

// flashAssistKillWindowTicks is the forward-walk window after the flash for
// crediting the kill. Mirrors the trade window so a flash that helps a
// teammate get a kill within 5 s counts; anything later is coincidence.
const flashAssistKillWindowTicks = 5 * 64

// flashAssistHighlight emits one Mistake (positive — surfaced as a low-
// severity highlight) per flashed-enemy event where the flasher's teammate
// killed that enemy within flashAssistKillWindowTicks while the enemy was
// still blind. Reuses the demoinfocs PlayerFlashedExtra durationsecs to gate
// "actually blinded" vs "flash chip", and the existing kill-event walk to
// pair the flash with a kill.
//
// "Why is this a Mistake?" — Mistake is the analyzer's generic "thing worth
// surfacing in the timeline" carrier. Severity-low (templates.go) and the
// flash_assist category let the frontend render these in green / muted to
// distinguish from genuine mistakes; the data model stays uniform.
func flashAssistHighlight(events []demo.GameEvent) []Mistake {
	if len(events) == 0 {
		return nil
	}
	out := make([]Mistake, 0, 8)
	type flash struct {
		idx       int
		tick      int
		round     int
		flasher   string
		victim    string
		duration  float64
		consumed  bool
		flashTeam string
	}
	flashes := make([]flash, 0, 16)
	flasherTeams := teamsByRoundFromKills(events)
	for i, ev := range events {
		if ev.Type != "player_flashed" {
			continue
		}
		if ev.AttackerSteamID == "" || ev.VictimSteamID == "" {
			continue
		}
		if ev.AttackerSteamID == ev.VictimSteamID {
			continue
		}
		extra, _ := ev.ExtraData.(*demo.PlayerFlashedExtra)
		if extra == nil || extra.DurationSecs < flashAssistMinBlindSecs {
			continue
		}
		ft := flasherTeams[ev.RoundNumber][ev.AttackerSteamID]
		if ft == "" {
			continue
		}
		flashes = append(flashes, flash{
			idx:       i,
			tick:      ev.Tick,
			round:     ev.RoundNumber,
			flasher:   ev.AttackerSteamID,
			victim:    ev.VictimSteamID,
			duration:  extra.DurationSecs,
			flashTeam: ft,
		})
	}
	if len(flashes) == 0 {
		return nil
	}
	for fi := range flashes {
		f := &flashes[fi]
		if f.consumed {
			continue
		}
		// Forward-walk for a kill landed by a teammate of the flasher against
		// the blinded victim while still in the blind window.
		blindLimit := f.tick + int(f.duration*64)
		walkLimit := f.tick + flashAssistKillWindowTicks
		for j := f.idx + 1; j < len(events); j++ {
			next := events[j]
			if next.Tick > walkLimit {
				break
			}
			if next.Tick > blindLimit {
				break
			}
			if next.Type != "kill" {
				continue
			}
			if next.VictimSteamID != f.victim {
				continue
			}
			nk, _ := next.ExtraData.(*demo.KillExtra)
			if nk == nil {
				continue
			}
			if nk.AttackerTeam != f.flashTeam {
				continue
			}
			f.consumed = true
			out = append(out, Mistake{
				SteamID:     f.flasher,
				RoundNumber: f.round,
				Tick:        f.tick,
				Kind:        string(MistakeKindFlashAssist),
				Extras: map[string]any{
					"victim_steam": f.victim,
					"duration_s":   f.duration,
					"kill_tick":    next.Tick,
				},
			})
			break
		}
	}
	return out
}

// heDamageHighlightThreshold is the HE damage in a single round (per thrower)
// that earns a highlight. 80 = roughly half a player's HP across two enemies,
// or one chunky single-target hit.
const heDamageHighlightThreshold = 80

// heDamageHighlight emits one Mistake (positive — same low-severity highlight
// pattern as flashAssistHighlight) per (round, thrower) where the thrower's
// HE-grenade damage in that round meets heDamageHighlightThreshold.
// Aggregates player_hurt events whose weapon is "hegrenade" by attacker.
func heDamageHighlight(events []demo.GameEvent) []Mistake {
	if len(events) == 0 {
		return nil
	}
	type key struct {
		round int
		steam string
	}
	type acc struct {
		damage int
		tick   int
	}
	totals := make(map[key]*acc, 16)
	for _, ev := range events {
		if ev.Type != "player_hurt" {
			continue
		}
		if ev.AttackerSteamID == "" || ev.VictimSteamID == "" {
			continue
		}
		if normalizeUtilToken(ev.Weapon) != "hegrenade" {
			continue
		}
		hurt, _ := ev.ExtraData.(*demo.PlayerHurtExtra)
		if hurt == nil {
			continue
		}
		dmg := hurt.HealthDamage
		if dmg <= 0 {
			continue
		}
		k := key{round: ev.RoundNumber, steam: ev.AttackerSteamID}
		a, ok := totals[k]
		if !ok {
			a = &acc{tick: ev.Tick}
			totals[k] = a
		}
		a.damage += dmg
	}
	out := make([]Mistake, 0, 8)
	for k, a := range totals {
		if a.damage < heDamageHighlightThreshold {
			continue
		}
		out = append(out, Mistake{
			SteamID:     k.steam,
			RoundNumber: k.round,
			Tick:        a.tick,
			Kind:        string(MistakeKindHeDamage),
			Extras: map[string]any{
				"damage": a.damage,
			},
		})
	}
	return out
}

// teamsByRoundFromKills mirrors teamsByRoundFromRosters' shape but rebuilds
// the (steamID → team) map from kill events. Used by rules that don't have
// the rounds slice — the flash-assist rule walks the event stream directly
// and never receives RoundData. The map is per-round so a halftime side
// switch flips cleanly.
func teamsByRoundFromKills(events []demo.GameEvent) map[int]map[string]string {
	out := make(map[int]map[string]string, 8)
	ensure := func(round int) map[string]string {
		inner, ok := out[round]
		if !ok {
			inner = make(map[string]string, 12)
			out[round] = inner
		}
		return inner
	}
	for _, ev := range events {
		if ev.Type != "kill" {
			continue
		}
		k, _ := ev.ExtraData.(*demo.KillExtra)
		if k == nil {
			continue
		}
		inner := ensure(ev.RoundNumber)
		if ev.AttackerSteamID != "" && k.AttackerTeam != "" {
			inner[ev.AttackerSteamID] = k.AttackerTeam
		}
		if ev.VictimSteamID != "" && k.VictimTeam != "" {
			inner[ev.VictimSteamID] = k.VictimTeam
		}
	}
	return out
}
