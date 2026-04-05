package demo

// PlayerRoundStats holds per-player statistics for a single round.
type PlayerRoundStats struct {
	SteamID      string
	PlayerName   string
	TeamSide     string
	Kills        int
	Deaths       int
	Assists      int
	Damage       int
	HeadshotKills int
	ClutchKills  int
	FirstKill    bool
	FirstDeath   bool
}

// CalculatePlayerRoundStats computes per-player stats for each round from
// parsed game events. Returns a map of round number -> player stats slice.
func CalculatePlayerRoundStats(rounds []RoundData, events []GameEvent) map[int][]PlayerRoundStats {
	// Group events by round number.
	eventsByRound := make(map[int][]GameEvent)
	for i := range events {
		rn := events[i].RoundNumber
		eventsByRound[rn] = append(eventsByRound[rn], events[i])
	}

	result := make(map[int][]PlayerRoundStats, len(rounds))
	for _, rd := range rounds {
		roundEvents := eventsByRound[rd.Number]
		stats := calculateRound(roundEvents)
		if len(stats) > 0 {
			result[rd.Number] = stats
		}
	}
	return result
}

// playerAccum accumulates stats for a single player within a round.
type playerAccum struct {
	steamID      string
	playerName   string
	teamSide     string
	kills        int
	deaths       int
	assists      int
	damage       int
	headshotKills int
	clutchKills  int
	firstKill    bool
	firstDeath   bool
}

func calculateRound(events []GameEvent) []PlayerRoundStats {
	players := make(map[string]*playerAccum)

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

	// Separate kill events and hurt events.
	var killEvents []GameEvent
	for _, ev := range events {
		switch ev.Type {
		case "kill":
			killEvents = append(killEvents, ev)
		case "player_hurt":
			attackerName := getExtraDataString(ev.ExtraData, "attacker_name")
			attackerTeam := getExtraDataString(ev.ExtraData, "attacker_team")
			if p := getPlayer(ev.AttackerSteamID, attackerName, attackerTeam); p != nil {
				p.damage += getExtraDataInt(ev.ExtraData, "health_damage")
			}
			// Register victim from hurt events too (for name/team).
			victimName := getExtraDataString(ev.ExtraData, "victim_name")
			victimTeam := getExtraDataString(ev.ExtraData, "victim_team")
			getPlayer(ev.VictimSteamID, victimName, victimTeam)
		}
	}

	// Track alive players per team for clutch detection.
	// Build initial alive set from all players seen in kill events.
	teamAlive := make(map[string]map[string]bool) // team -> set of steamIDs
	for _, ev := range killEvents {
		attackerTeam := getExtraDataString(ev.ExtraData, "attacker_team")
		victimTeam := getExtraDataString(ev.ExtraData, "victim_team")
		if ev.AttackerSteamID != "" && attackerTeam != "" {
			if teamAlive[attackerTeam] == nil {
				teamAlive[attackerTeam] = make(map[string]bool)
			}
			teamAlive[attackerTeam][ev.AttackerSteamID] = true
		}
		if ev.VictimSteamID != "" && victimTeam != "" {
			if teamAlive[victimTeam] == nil {
				teamAlive[victimTeam] = make(map[string]bool)
			}
			teamAlive[victimTeam][ev.VictimSteamID] = true
		}
	}

	// Process kill events in order (already ordered by tick from parser).
	firstKillProcessed := false
	for _, ev := range killEvents {
		attackerName := getExtraDataString(ev.ExtraData, "attacker_name")
		attackerTeam := getExtraDataString(ev.ExtraData, "attacker_team")
		victimName := getExtraDataString(ev.ExtraData, "victim_name")
		victimTeam := getExtraDataString(ev.ExtraData, "victim_team")

		isSelfKill := ev.AttackerSteamID != "" && ev.AttackerSteamID == ev.VictimSteamID

		// Credit kill (not for world kills or self-kills).
		if ev.AttackerSteamID != "" && !isSelfKill {
			attacker := getPlayer(ev.AttackerSteamID, attackerName, attackerTeam)
			if attacker != nil {
				attacker.kills++
				if getExtraDataBool(ev.ExtraData, "headshot") {
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
		assisterID := getExtraDataString(ev.ExtraData, "assister_steam_id")
		if assisterID != "" {
			// Assister inherits team from attacker (same team).
			assister := getPlayer(assisterID, "", attackerTeam)
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

	// Convert accumulators to result slice.
	stats := make([]PlayerRoundStats, 0, len(players))
	for _, p := range players {
		stats = append(stats, PlayerRoundStats{
			SteamID:       p.steamID,
			PlayerName:    p.playerName,
			TeamSide:      p.teamSide,
			Kills:         p.kills,
			Deaths:        p.deaths,
			Assists:       p.assists,
			Damage:        p.damage,
			HeadshotKills: p.headshotKills,
			ClutchKills:   p.clutchKills,
			FirstKill:     p.firstKill,
			FirstDeath:    p.firstDeath,
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

// getExtraDataString safely extracts a string from ExtraData.
func getExtraDataString(extra map[string]interface{}, key string) string {
	if extra == nil {
		return ""
	}
	v, ok := extra[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// getExtraDataBool safely extracts a bool from ExtraData.
func getExtraDataBool(extra map[string]interface{}, key string) bool {
	if extra == nil {
		return false
	}
	v, ok := extra[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	if !ok {
		return false
	}
	return b
}

// getExtraDataInt safely extracts an int from ExtraData.
// Handles both native int and JSON-decoded float64.
func getExtraDataInt(extra map[string]interface{}, key string) int {
	if extra == nil {
		return 0
	}
	v, ok := extra[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	default:
		return 0
	}
}
