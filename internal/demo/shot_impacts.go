package demo

// shotImpactPairWindow is the maximum tick gap between a weapon_fire and the
// player_hurt that resulted from it. Bullet flight + damage registration in CS2
// is well under 250ms (~16 ticks at 64Hz) for any in-map distance.
const shotImpactPairWindow = 16

// pairShotsWithImpacts walks events in tick order and writes hit_x / hit_y into
// each weapon_fire's extra_data when it can be matched to a subsequent
// player_hurt from the same attacker. The frontend uses these to draw the
// tracer line ending exactly at the victim's position; unmatched shots fall
// back to a fixed-length directional ray (a miss or a wall hit, which the demo
// format does not expose).
//
// Pairing strategy: for each attacker, remember the most recent unpaired
// weapon_fire; when a player_hurt from that attacker appears within the
// window, consume the shot and record the impact. One shot pairs with at most
// one hurt event — wallbangs through multiple players still report only the
// first impact, but the visual approximation is acceptable.
func pairShotsWithImpacts(events []GameEvent) {
	lastShotIdx := make(map[string]int)
	for i := range events {
		ev := &events[i]
		switch ev.Type {
		case "weapon_fire":
			if ev.AttackerSteamID == "" {
				continue
			}
			lastShotIdx[ev.AttackerSteamID] = i
		case "player_hurt":
			if ev.AttackerSteamID == "" {
				continue
			}
			shotIdx, ok := lastShotIdx[ev.AttackerSteamID]
			if !ok {
				continue
			}
			shot := &events[shotIdx]
			if ev.Tick-shot.Tick > shotImpactPairWindow {
				delete(lastShotIdx, ev.AttackerSteamID)
				continue
			}
			extra, _ := shot.ExtraData.(*WeaponFireExtra)
			if extra == nil {
				extra = &WeaponFireExtra{}
				shot.ExtraData = extra
			}
			hx, hy := ev.X, ev.Y
			extra.HitX = &hx
			extra.HitY = &hy
			if ev.VictimSteamID != "" {
				extra.HitVictimSteamID = ev.VictimSteamID
			}
			delete(lastShotIdx, ev.AttackerSteamID)
		}
	}
}
