package detectors

import (
	"math"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/analysis"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

// nearestTick is a thin wrapper around analysis.NearestTick — kept here
// so detector bodies refer to the in-package name (matches the
// "internal helper" idiom in the analysis rules).
func nearestTick(idx analysis.PerPlayerTickIndex, steamID string, tick int) (demo.AnalysisTick, bool) {
	return analysis.NearestTick(idx, steamID, tick)
}

// nearestEnemyAtTick picks the enemy in c.Enemies with the smallest 2D
// distance to the subject at the given tick. Returns the SteamID and
// the enemy's AnalysisTick row. ok=false when the subject has no
// sample at the tick or no enemy in c.Enemies has one.
func nearestEnemyAtTick(c *contacts.Contact, ctx *DetectorCtx, tick int32) (string, demo.AnalysisTick, bool) {
	subjectRow, ok := nearestTick(ctx.Ticks, c.Subject, int(tick))
	if !ok {
		return "", demo.AnalysisTick{}, false
	}
	var bestSteam string
	var bestRow demo.AnalysisTick
	bestDist := math.MaxFloat64
	for _, e := range c.Enemies {
		row, ok := nearestTick(ctx.Ticks, e, int(tick))
		if !ok {
			continue
		}
		d := math.Hypot(float64(row.X-subjectRow.X), float64(row.Y-subjectRow.Y))
		if d < bestDist {
			bestDist = d
			bestSteam = e
			bestRow = row
		}
	}
	if bestSteam == "" {
		return "", demo.AnalysisTick{}, false
	}
	return bestSteam, bestRow, true
}

// subjectWeaponAtTick returns the weapon name the subject was holding
// most recently via a weapon_fire event in the [ClampPreLookback,
// atTick] window. Used by peek_while_reloading; the AnalysisTick row
// only carries AmmoClip, not the active-weapon class, so we read it
// from the most recent shot the subject fired.
func subjectWeaponAtTick(c *contacts.Contact, ctx *DetectorCtx, atTick int32) (string, bool) {
	lower := ClampPreLookback(c, ctx)
	for i := len(ctx.Events) - 1; i >= 0; i-- {
		evt := ctx.Events[i]
		if int32(evt.Tick) > atTick {
			continue
		}
		if int32(evt.Tick) < lower {
			return "", false
		}
		if evt.Type == "weapon_fire" && evt.AttackerSteamID == c.Subject {
			return evt.Weapon, true
		}
	}
	return "", false
}

// subjectDiedInContact returns true when any kill signal inside the
// contact has the subject as victim.
func subjectDiedInContact(c *contacts.Contact) bool {
	for _, s := range c.Signals {
		if s.Kind == contacts.SignalKill && s.Subject == contacts.SubjectVictim {
			return true
		}
	}
	return false
}

// subjectGotAKill returns true when any kill signal inside the contact
// has the subject as aggressor.
func subjectGotAKill(c *contacts.Contact) bool {
	for _, s := range c.Signals {
		if s.Kind == contacts.SignalKill && s.Subject == contacts.SubjectAggressor {
			return true
		}
	}
	return false
}

// subjectLastKillTick returns the highest Tick on a kill signal where
// the subject is aggressor. Returns -1 when the subject got no kill.
func subjectLastKillTick(c *contacts.Contact) int32 {
	var last int32 = -1
	for _, s := range c.Signals {
		if s.Kind == contacts.SignalKill && s.Subject == contacts.SubjectAggressor {
			if s.Tick > last {
				last = s.Tick
			}
		}
	}
	return last
}

// lastSubjectShot returns the last weapon_fire event the subject fired
// inside [c.TFirst, c.TLast]. found=false when the subject never fired
// in the contact window.
func lastSubjectShot(c *contacts.Contact, ctx *DetectorCtx) (demo.GameEvent, bool) {
	var last demo.GameEvent
	found := false
	for _, evt := range ctx.Events {
		if int32(evt.Tick) > c.TLast {
			break
		}
		if int32(evt.Tick) < c.TFirst {
			continue
		}
		if evt.Type == "weapon_fire" && evt.AttackerSteamID == c.Subject {
			last = evt
			found = true
		}
	}
	return last, found
}

// subjectHealthAt walks ctx.Events backward from atTick for player_hurt
// rows targeting the subject, subtracting health_damage from the round-
// start baseline (100). Stops at round start.
func subjectHealthAt(c *contacts.Contact, ctx *DetectorCtx, atTick int32) int {
	hp := 100
	for _, evt := range ctx.Events {
		if int32(evt.Tick) >= atTick {
			break
		}
		if evt.Type != "player_hurt" || evt.VictimSteamID != c.Subject {
			continue
		}
		he, _ := evt.ExtraData.(*demo.PlayerHurtExtra)
		if he == nil {
			continue
		}
		hp -= he.HealthDamage
		if hp < 0 {
			hp = 0
		}
	}
	return hp
}

// enemyHealthAt is subjectHealthAt with the victim field set to the
// supplied steam ID.
func enemyHealthAt(ctx *DetectorCtx, enemy string, atTick int32) int {
	hp := 100
	for _, evt := range ctx.Events {
		if int32(evt.Tick) >= atTick {
			break
		}
		if evt.Type != "player_hurt" || evt.VictimSteamID != enemy {
			continue
		}
		he, _ := evt.ExtraData.(*demo.PlayerHurtExtra)
		if he == nil {
			continue
		}
		hp -= he.HealthDamage
		if hp < 0 {
			hp = 0
		}
	}
	return hp
}

// anotherEnemyAliveNearby returns true when ANY enemy in the round (not
// just c.Enemies) is alive and within radius of the subject at c.TLast.
// Used by no_reposition_after_kill — the rule fires when a contact-
// external enemy still threatens the spot.
func anotherEnemyAliveNearby(c *contacts.Contact, ctx *DetectorCtx, radius float64) bool {
	subjectRow, ok := nearestTick(ctx.Ticks, c.Subject, int(c.TLast))
	if !ok {
		return false
	}
	for steam, p := range ctx.Players {
		if steam == c.Subject {
			continue
		}
		if p.TeamSide == ctx.SubjectTeam {
			continue
		}
		row, ok := nearestTick(ctx.Ticks, steam, int(c.TLast))
		if !ok || !row.IsAlive {
			continue
		}
		d := math.Hypot(float64(row.X-subjectRow.X), float64(row.Y-subjectRow.Y))
		if d <= radius {
			return true
		}
	}
	return false
}

// maxMovement returns the maximum 2D distance the steam-player ever
// traveled from (anchor.X, anchor.Y) inside [lo, hi]. Used by
// no_reposition_after_kill.
func maxMovement(idx analysis.PerPlayerTickIndex, steam string, lo, hi int32, anchor demo.AnalysisTick) float64 {
	rows := analysis.TickRangeRows(idx, steam, int(lo), int(hi))
	maxD := 0.0
	for _, r := range rows {
		d := math.Hypot(float64(r.X-anchor.X), float64(r.Y-anchor.Y))
		if d > maxD {
			maxD = d
		}
	}
	return maxD
}

// enemySpottedBetween returns true when any enemy weapon_fire happens
// in (lo, hi]. v1 approximates "enemy spotted" via shots — see
// 04-detectors-during-post.md §6.4.
func enemySpottedBetween(c *contacts.Contact, ctx *DetectorCtx, lo, hi int32) bool {
	for _, evt := range ctx.Events {
		if int32(evt.Tick) <= lo || int32(evt.Tick) > hi {
			continue
		}
		if evt.Type != "weapon_fire" {
			continue
		}
		if evt.AttackerSteamID == c.Subject {
			continue
		}
		attacker, ok := ctx.Players[evt.AttackerSteamID]
		if !ok || attacker.TeamSide == ctx.SubjectTeam {
			continue
		}
		return true
	}
	return false
}

// reloadCompletedBetween returns true when the subject's AmmoClip ever
// rises between two consecutive samples in [lo, hi]. Detects a reload
// completion without modeling the per-weapon reload duration.
func reloadCompletedBetween(ctx *DetectorCtx, steam string, lo, hi int32, _ int) bool {
	rows := analysis.TickRangeRows(ctx.Ticks, steam, int(lo), int(hi))
	if len(rows) < 2 {
		return false
	}
	for i := 1; i < len(rows); i++ {
		if int(rows[i].AmmoClip) > int(rows[i-1].AmmoClip) {
			return true
		}
	}
	return false
}

// flashWindow is a half-open [Start, End) tick range during which the
// subject is still affected by a flash. Built by subjectFlashWindows.
type flashWindow struct {
	Start int32
	End   int32
}

// subjectFlashWindows scans ctx.Events for player_flashed rows where
// the subject is the victim and the flash duration is ≥ the contact-
// opening threshold (0.7s). Teammate-attacker flashes are filtered
// out — the rule only flags enemy-induced flashes (the subject's own
// teammates flashing them is the teammate's fault, not the subject's).
func subjectFlashWindows(c *contacts.Contact, ctx *DetectorCtx, minDurationSecs float64) []flashWindow {
	var out []flashWindow
	for _, evt := range ctx.Events {
		if evt.Type != "player_flashed" || evt.VictimSteamID != c.Subject {
			continue
		}
		fe, _ := evt.ExtraData.(*demo.PlayerFlashedExtra)
		if fe == nil || fe.DurationSecs < minDurationSecs {
			continue
		}
		if evt.AttackerSteamID != "" {
			if attacker, ok := ctx.Players[evt.AttackerSteamID]; ok {
				if attacker.TeamSide == ctx.SubjectTeam {
					continue
				}
			}
		}
		end := int32(evt.Tick) + int32(fe.DurationSecs*ctx.TickRate)
		out = append(out, flashWindow{Start: int32(evt.Tick), End: end})
	}
	return out
}

// insideAnyFlash returns true when tick falls inside any flashWindow.
func insideAnyFlash(tick int32, flashes []flashWindow) bool {
	for _, f := range flashes {
		if tick >= f.Start && tick < f.End {
			return true
		}
	}
	return false
}
