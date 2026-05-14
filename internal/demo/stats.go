package demo

// PlayerRoundStats holds per-player statistics for a single round.
//
// EquipValue / MoneyAtRoundStart / MoneyAtFreezeEnd / Survived / TradeKill /
// KastRound feed the match-overview aggregator. They are seeded from the
// freeze-end roster (which already carries Survived once RoundEnd sets it) and
// TradeKill is computed against the same trade-window pass used by
// ComputePlayerMatchStats.
type PlayerRoundStats struct {
	SteamID           string
	PlayerName        string
	TeamSide          string
	Kills             int
	Deaths            int
	Assists           int
	Damage            int
	HeadshotKills     int
	ClutchKills       int
	FirstKill         bool
	FirstDeath        bool
	Survived          bool
	EquipValue        int
	MoneyAtRoundStart int
	MoneyAtFreezeEnd  int
	TradeKill         bool
	KastRound         bool
}

// CalculatePlayerRoundStats computes per-player stats for each round from
// parsed game events. Returns a map of round number -> player stats slice.
//
// Players are seeded from each round's freeze-end roster so passive participants
// (no kills, no damage, no deaths) still receive a zero-stat row. Without the
// seed the frontend roster lookup falls back to slicing the SteamID, which
// shows a numeric nickname for those players.
func CalculatePlayerRoundStats(rounds []RoundData, events []GameEvent) map[int][]PlayerRoundStats {
	// Events are produced by the parser in tick-monotonic order, and round
	// numbers are monotonic with tick (knife-round renumbering preserves
	// order). We can therefore walk events once with a cursor, slicing the
	// contiguous range belonging to each round — no map, no per-round slice
	// allocation.
	result := make(map[int][]PlayerRoundStats, len(rounds))
	cursor := 0
	for _, rd := range rounds {
		// Skip events that precede this round (e.g. warmup events with a
		// round number not present in the rounds slice).
		for cursor < len(events) && events[cursor].RoundNumber < rd.Number {
			cursor++
		}
		// Collect contiguous range of events for this round.
		start := cursor
		for cursor < len(events) && events[cursor].RoundNumber == rd.Number {
			cursor++
		}
		stats := calculateRound(rd.Roster, events[start:cursor])
		if len(stats) > 0 {
			result[rd.Number] = stats
		}
	}
	return result
}

// playerAccum accumulates stats for a single player within a round.
type playerAccum struct {
	steamID           string
	playerName        string
	teamSide          string
	kills             int
	deaths            int
	assists           int
	damage            int
	headshotKills     int
	clutchKills       int
	firstKill         bool
	firstDeath        bool
	survived          bool
	equipValue        int
	moneyAtRoundStart int
	moneyAtFreezeEnd  int
	tradeKill         bool
}

func calculateRound(roster []RoundParticipant, events []GameEvent) []PlayerRoundStats {
	players := make(map[string]*playerAccum)

	// Seed from the freeze-end roster — every alive participant gets a row,
	// zero-stat or otherwise. Late joiners not in the roster still show up via
	// the getPlayer fallback below when they appear in a kill/hurt event.
	for _, rp := range roster {
		if rp.SteamID == "" {
			continue
		}
		players[rp.SteamID] = &playerAccum{
			steamID:           rp.SteamID,
			playerName:        rp.PlayerName,
			teamSide:          rp.TeamSide,
			survived:          rp.Survived,
			equipValue:        rp.EquipValue,
			moneyAtRoundStart: rp.MoneyAtRoundStart,
			moneyAtFreezeEnd:  rp.MoneyAtFreezeEnd,
		}
	}

	getPlayer := func(steamID, name, team string) *playerAccum {
		if steamID == "" {
			return nil
		}
		p, ok := players[steamID]
		if !ok {
			p = &playerAccum{steamID: steamID}
			players[steamID] = p
		}
		if name != "" {
			p.playerName = name
		}
		if team != "" {
			p.teamSide = team
		}
		return p
	}

	// First pass: separate kill/hurt events, register all players via getPlayer.
	var killEvents []GameEvent
	for _, ev := range events {
		switch ev.Type {
		case "kill":
			killEvents = append(killEvents, ev)
			// Pre-register kill participants so they appear in the players map.
			k, _ := ev.ExtraData.(*KillExtra)
			if k == nil {
				k = &KillExtra{}
			}
			getPlayer(ev.AttackerSteamID, k.AttackerName, k.AttackerTeam)
			getPlayer(ev.VictimSteamID, k.VictimName, k.VictimTeam)
		case "player_hurt":
			h, _ := ev.ExtraData.(*PlayerHurtExtra)
			if h == nil {
				h = &PlayerHurtExtra{}
			}
			if p := getPlayer(ev.AttackerSteamID, h.AttackerName, h.AttackerTeam); p != nil {
				p.damage += h.HealthDamage
			}
			getPlayer(ev.VictimSteamID, h.VictimName, h.VictimTeam)
		}
	}

	// Build initial alive set from ALL registered players (hurt + kill events).
	// This prevents false clutch detection when teammates are alive but not in kill events.
	teamAlive := make(map[string]map[string]bool) // team -> set of steamIDs
	for steamID, p := range players {
		if p.teamSide == "" {
			continue
		}
		if teamAlive[p.teamSide] == nil {
			teamAlive[p.teamSide] = make(map[string]bool)
		}
		teamAlive[p.teamSide][steamID] = true
	}

	// Process kill events in order (already ordered by tick from parser).
	firstKillProcessed := false
	for _, ev := range killEvents {
		k, _ := ev.ExtraData.(*KillExtra)
		if k == nil {
			k = &KillExtra{}
		}
		attackerName := k.AttackerName
		attackerTeam := k.AttackerTeam
		victimName := k.VictimName
		victimTeam := k.VictimTeam

		isSelfKill := ev.AttackerSteamID != "" && ev.AttackerSteamID == ev.VictimSteamID

		// Credit kill (not for world kills or self-kills).
		if ev.AttackerSteamID != "" && !isSelfKill {
			attacker := getPlayer(ev.AttackerSteamID, attackerName, attackerTeam)
			if attacker != nil {
				attacker.kills++
				if k.Headshot {
					attacker.headshotKills++
				}

				// Clutch detection: is attacker the last alive on their team?
				if attackerTeam != "" && isClutching(ev.AttackerSteamID, attackerTeam, teamAlive) {
					attacker.clutchKills++
				}
			}
		}

		// Credit death.
		if ev.VictimSteamID != "" {
			victim := getPlayer(ev.VictimSteamID, victimName, victimTeam)
			if victim != nil {
				victim.deaths++
			}
		}

		// Credit assist.
		assisterID := k.AssisterSteamID
		if assisterID != "" {
			assisterName := k.AssisterName
			assisterTeam := k.AssisterTeam
			if assisterTeam == "" {
				assisterTeam = attackerTeam
			}
			assister := getPlayer(assisterID, assisterName, assisterTeam)
			if assister != nil {
				assister.assists++
			}
		}

		// First kill / first death.
		if !firstKillProcessed {
			if ev.AttackerSteamID != "" && !isSelfKill {
				if a := players[ev.AttackerSteamID]; a != nil {
					a.firstKill = true
				}
			}
			if ev.VictimSteamID != "" {
				if v := players[ev.VictimSteamID]; v != nil {
					v.firstDeath = true
				}
			}
			firstKillProcessed = true
		}

		// Remove victim from alive set (after clutch check).
		if ev.VictimSteamID != "" && victimTeam != "" {
			if teamAlive[victimTeam] != nil {
				delete(teamAlive[victimTeam], ev.VictimSteamID)
			}
		}
	}

	// Trade-kill detection: did the player kill an enemy within
	// tradeWindowSeconds of a teammate's death? Mirrors the per-match pass in
	// ComputePlayerMatchStats but runs once per round so the match-overview
	// aggregator gets a per-round kast bit. A fixed 64-tps window is used
	// (matching the parser default) because CalculatePlayerRoundStats is not
	// threaded with tickRate; the off-by-tick error on 128-tps demos is small
	// against a 5-second window.
	tradeWindowTicks := int(tradeWindowSeconds * 64)
	for _, ev := range killEvents {
		if ev.AttackerSteamID == "" || ev.AttackerSteamID == ev.VictimSteamID {
			continue
		}
		attacker, ok := players[ev.AttackerSteamID]
		if !ok || attacker.tradeKill {
			continue
		}
		k, _ := ev.ExtraData.(*KillExtra)
		if k == nil {
			continue
		}
		attackerTeam := k.AttackerTeam
		if attackerTeam == "" {
			attackerTeam = attacker.teamSide
		}
		for j := len(killEvents) - 1; j >= 0; j-- {
			prev := killEvents[j]
			if prev.Tick > ev.Tick {
				continue
			}
			if ev.Tick-prev.Tick > tradeWindowTicks {
				break
			}
			if prev.VictimSteamID == "" || prev.VictimSteamID == ev.AttackerSteamID {
				continue
			}
			pk, _ := prev.ExtraData.(*KillExtra)
			if pk == nil {
				continue
			}
			if pk.VictimTeam == attackerTeam {
				attacker.tradeKill = true
				break
			}
		}
	}

	// Convert accumulators to result slice.
	stats := make([]PlayerRoundStats, 0, len(players))
	for _, p := range players {
		kast := p.kills > 0 || p.assists > 0 || p.survived || p.tradeKill
		stats = append(stats, PlayerRoundStats{
			SteamID:           p.steamID,
			PlayerName:        p.playerName,
			TeamSide:          p.teamSide,
			Kills:             p.kills,
			Deaths:            p.deaths,
			Assists:           p.assists,
			Damage:            p.damage,
			HeadshotKills:     p.headshotKills,
			ClutchKills:       p.clutchKills,
			FirstKill:         p.firstKill,
			FirstDeath:        p.firstDeath,
			Survived:          p.survived,
			EquipValue:        p.equipValue,
			MoneyAtRoundStart: p.moneyAtRoundStart,
			MoneyAtFreezeEnd:  p.moneyAtFreezeEnd,
			TradeKill:         p.tradeKill,
			KastRound:         kast,
		})
	}
	return stats
}

// isClutching returns true if the player is the only alive member of their team
// and the opposing team has at least one alive member.
func isClutching(steamID, team string, teamAlive map[string]map[string]bool) bool {
	myTeam := teamAlive[team]
	if myTeam == nil || len(myTeam) != 1 || !myTeam[steamID] {
		return false
	}

	// Check if any opposing team has alive players.
	for t, alive := range teamAlive {
		if t != team && len(alive) > 0 {
			return true
		}
	}
	return false
}
