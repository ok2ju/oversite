package demo

import (
	"testing"
)

// helper to build a kill event with ExtraData fields.
func killEvent(round, tick int, attackerID, victimID string, extra map[string]interface{}) GameEvent {
	if extra == nil {
		extra = map[string]interface{}{}
	}
	// Set defaults for required fields if not provided.
	if _, ok := extra["headshot"]; !ok {
		extra["headshot"] = false
	}
	return GameEvent{
		Tick:            tick,
		RoundNumber:     round,
		Type:            "kill",
		AttackerSteamID: attackerID,
		VictimSteamID:   victimID,
		ExtraData:       extra,
	}
}

// helper to build a player_hurt event.
func hurtEvent(round, tick int, attackerID, victimID string, damage int) GameEvent {
	return GameEvent{
		Tick:            tick,
		RoundNumber:     round,
		Type:            "player_hurt",
		AttackerSteamID: attackerID,
		VictimSteamID:   victimID,
		ExtraData: map[string]interface{}{
			"health_damage": damage,
		},
	}
}

func findPlayerStats(stats []PlayerRoundStats, steamID string) *PlayerRoundStats {
	for i := range stats {
		if stats[i].SteamID == steamID {
			return &stats[i]
		}
	}
	return nil
}

func TestCalculatePlayerRoundStats_KDA(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
	}
	events := []GameEvent{
		killEvent(1, 100, "1001", "2001", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Alice",
			"attacker_team": "CT",
			"victim_name":   "Bob",
			"victim_team":   "T",
		}),
		killEvent(1, 200, "1001", "2002", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Alice",
			"attacker_team": "CT",
			"victim_name":   "Charlie",
			"victim_team":   "T",
		}),
		killEvent(1, 300, "2003", "1002", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Dave",
			"attacker_team": "T",
			"victim_name":   "Eve",
			"victim_team":   "CT",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)

	if len(result) != 1 {
		t.Fatalf("expected 1 round, got %d", len(result))
	}

	stats := result[1]
	alice := findPlayerStats(stats, "1001")
	if alice == nil {
		t.Fatal("expected stats for Alice (1001)")
	}
	if alice.Kills != 2 {
		t.Errorf("Alice kills: got %d, want 2", alice.Kills)
	}
	if alice.Deaths != 0 {
		t.Errorf("Alice deaths: got %d, want 0", alice.Deaths)
	}

	bob := findPlayerStats(stats, "2001")
	if bob == nil {
		t.Fatal("expected stats for Bob (2001)")
	}
	if bob.Kills != 0 {
		t.Errorf("Bob kills: got %d, want 0", bob.Kills)
	}
	if bob.Deaths != 1 {
		t.Errorf("Bob deaths: got %d, want 1", bob.Deaths)
	}

	dave := findPlayerStats(stats, "2003")
	if dave == nil {
		t.Fatal("expected stats for Dave (2003)")
	}
	if dave.Kills != 1 {
		t.Errorf("Dave kills: got %d, want 1", dave.Kills)
	}
}

func TestCalculatePlayerRoundStats_HeadshotKills(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
	}
	events := []GameEvent{
		killEvent(1, 100, "1001", "2001", map[string]interface{}{
			"headshot":      true,
			"attacker_name": "Alice",
			"attacker_team": "CT",
			"victim_name":   "Bob",
			"victim_team":   "T",
		}),
		killEvent(1, 200, "1001", "2002", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Alice",
			"attacker_team": "CT",
			"victim_name":   "Charlie",
			"victim_team":   "T",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)
	stats := result[1]
	alice := findPlayerStats(stats, "1001")
	if alice == nil {
		t.Fatal("expected stats for Alice")
	}
	if alice.HeadshotKills != 1 {
		t.Errorf("Alice headshot kills: got %d, want 1", alice.HeadshotKills)
	}
}

func TestCalculatePlayerRoundStats_FirstKillFirstDeath(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
	}
	events := []GameEvent{
		killEvent(1, 100, "1001", "2001", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Alice",
			"attacker_team": "CT",
			"victim_name":   "Bob",
			"victim_team":   "T",
		}),
		killEvent(1, 200, "1002", "2002", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Eve",
			"attacker_team": "CT",
			"victim_name":   "Charlie",
			"victim_team":   "T",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)
	stats := result[1]

	alice := findPlayerStats(stats, "1001")
	if alice == nil {
		t.Fatal("expected stats for Alice")
	}
	if !alice.FirstKill {
		t.Error("Alice should have first kill")
	}
	if alice.FirstDeath {
		t.Error("Alice should not have first death")
	}

	bob := findPlayerStats(stats, "2001")
	if bob == nil {
		t.Fatal("expected stats for Bob")
	}
	if !bob.FirstDeath {
		t.Error("Bob should have first death")
	}
	if bob.FirstKill {
		t.Error("Bob should not have first kill")
	}

	eve := findPlayerStats(stats, "1002")
	if eve == nil {
		t.Fatal("expected stats for Eve")
	}
	if eve.FirstKill {
		t.Error("Eve should not have first kill (second kill)")
	}
}

func TestCalculatePlayerRoundStats_Assists(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
	}
	events := []GameEvent{
		killEvent(1, 100, "1001", "2001", map[string]interface{}{
			"headshot":          false,
			"attacker_name":     "Alice",
			"attacker_team":     "CT",
			"victim_name":       "Bob",
			"victim_team":       "T",
			"assister_steam_id": "1002",
		}),
		killEvent(1, 200, "1001", "2002", map[string]interface{}{
			"headshot":          false,
			"attacker_name":     "Alice",
			"attacker_team":     "CT",
			"victim_name":       "Charlie",
			"victim_team":       "T",
			"assister_steam_id": "1002",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)
	stats := result[1]

	eve := findPlayerStats(stats, "1002")
	if eve == nil {
		t.Fatal("expected stats for assister (1002)")
	}
	if eve.Assists != 2 {
		t.Errorf("assister assists: got %d, want 2", eve.Assists)
	}
}

func TestCalculatePlayerRoundStats_Damage(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
	}
	events := []GameEvent{
		hurtEvent(1, 50, "1001", "2001", 40),
		hurtEvent(1, 60, "1001", "2001", 60),
		hurtEvent(1, 70, "1001", "2002", 25),
		killEvent(1, 100, "1001", "2001", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Alice",
			"attacker_team": "CT",
			"victim_name":   "Bob",
			"victim_team":   "T",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)
	stats := result[1]

	alice := findPlayerStats(stats, "1001")
	if alice == nil {
		t.Fatal("expected stats for Alice")
	}
	if alice.Damage != 125 {
		t.Errorf("Alice damage: got %d, want 125", alice.Damage)
	}
}

func TestCalculatePlayerRoundStats_ClutchKills(t *testing.T) {
	// Scenario: 1v2 clutch. Alice (CT) is last alive, kills 2 T's.
	rounds := []RoundData{
		{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
	}
	events := []GameEvent{
		// T kills CT teammates first, leaving Alice alone
		killEvent(1, 100, "2001", "1002", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Bob",
			"attacker_team": "T",
			"victim_name":   "Eve",
			"victim_team":   "CT",
		}),
		killEvent(1, 150, "2002", "1003", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Charlie",
			"attacker_team": "T",
			"victim_name":   "Frank",
			"victim_team":   "CT",
		}),
		// Now Alice (1001) is 1v2 — her kills are clutch kills
		killEvent(1, 200, "1001", "2001", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Alice",
			"attacker_team": "CT",
			"victim_name":   "Bob",
			"victim_team":   "T",
		}),
		killEvent(1, 300, "1001", "2002", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Alice",
			"attacker_team": "CT",
			"victim_name":   "Charlie",
			"victim_team":   "T",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)
	stats := result[1]

	alice := findPlayerStats(stats, "1001")
	if alice == nil {
		t.Fatal("expected stats for Alice")
	}
	if alice.ClutchKills != 2 {
		t.Errorf("Alice clutch kills: got %d, want 2", alice.ClutchKills)
	}
}

func TestCalculatePlayerRoundStats_WorldKill(t *testing.T) {
	// World kill: no attacker (fall damage, etc.)
	rounds := []RoundData{
		{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
	}
	events := []GameEvent{
		killEvent(1, 100, "", "2001", map[string]interface{}{
			"headshot":    false,
			"victim_name": "Bob",
			"victim_team": "T",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)
	stats := result[1]

	bob := findPlayerStats(stats, "2001")
	if bob == nil {
		t.Fatal("expected stats for Bob")
	}
	if bob.Deaths != 1 {
		t.Errorf("Bob deaths: got %d, want 1", bob.Deaths)
	}

	// No entry for empty attacker should exist as a player
	for _, s := range stats {
		if s.SteamID == "" {
			t.Error("should not have stats entry for empty steam ID (world)")
		}
	}
}

func TestCalculatePlayerRoundStats_SelfKill(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
	}
	events := []GameEvent{
		killEvent(1, 100, "2001", "2001", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Bob",
			"attacker_team": "T",
			"victim_name":   "Bob",
			"victim_team":   "T",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)
	stats := result[1]

	bob := findPlayerStats(stats, "2001")
	if bob == nil {
		t.Fatal("expected stats for Bob")
	}
	if bob.Deaths != 1 {
		t.Errorf("Bob deaths: got %d, want 1", bob.Deaths)
	}
	if bob.Kills != 0 {
		t.Errorf("Bob kills on self-kill: got %d, want 0", bob.Kills)
	}
}

func TestCalculatePlayerRoundStats_NoEvents(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
	}
	events := []GameEvent{}

	result := CalculatePlayerRoundStats(rounds, events)

	// Round exists but has no player stats
	stats, ok := result[1]
	if ok && len(stats) != 0 {
		t.Errorf("expected 0 player stats for empty round, got %d", len(stats))
	}
}

func TestCalculatePlayerRoundStats_MultipleRounds(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
		{Number: 2, StartTick: 1001, EndTick: 2000, WinnerSide: "T"},
	}
	events := []GameEvent{
		killEvent(1, 100, "1001", "2001", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Alice",
			"attacker_team": "CT",
			"victim_name":   "Bob",
			"victim_team":   "T",
		}),
		killEvent(2, 1100, "2001", "1001", map[string]interface{}{
			"headshot":      true,
			"attacker_name": "Bob",
			"attacker_team": "T",
			"victim_name":   "Alice",
			"victim_team":   "CT",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)
	if len(result) != 2 {
		t.Fatalf("expected 2 rounds, got %d", len(result))
	}

	// Round 1: Alice 1 kill, Bob 1 death
	r1Alice := findPlayerStats(result[1], "1001")
	if r1Alice == nil || r1Alice.Kills != 1 {
		t.Errorf("Round 1 Alice kills: got %v, want 1", r1Alice)
	}
	r1Bob := findPlayerStats(result[1], "2001")
	if r1Bob == nil || r1Bob.Deaths != 1 {
		t.Errorf("Round 1 Bob deaths: got %v, want 1", r1Bob)
	}

	// Round 2: Bob 1 kill (headshot), Alice 1 death
	r2Bob := findPlayerStats(result[2], "2001")
	if r2Bob == nil || r2Bob.Kills != 1 || r2Bob.HeadshotKills != 1 {
		t.Errorf("Round 2 Bob: got %+v, want 1 kill 1 hs", r2Bob)
	}
	r2Alice := findPlayerStats(result[2], "1001")
	if r2Alice == nil || r2Alice.Deaths != 1 {
		t.Errorf("Round 2 Alice deaths: got %v, want 1", r2Alice)
	}
}

func TestCalculatePlayerRoundStats_OvertimeRounds(t *testing.T) {
	// Overtime round (>24). Should work identically.
	rounds := []RoundData{
		{Number: 25, StartTick: 50000, EndTick: 51000, WinnerSide: "CT", IsOvertime: true},
	}
	events := []GameEvent{
		killEvent(25, 50100, "1001", "2001", map[string]interface{}{
			"headshot":      true,
			"attacker_name": "Alice",
			"attacker_team": "CT",
			"victim_name":   "Bob",
			"victim_team":   "T",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)
	stats := result[25]
	if len(stats) == 0 {
		t.Fatal("expected stats for overtime round 25")
	}

	alice := findPlayerStats(stats, "1001")
	if alice == nil {
		t.Fatal("expected stats for Alice in overtime")
	}
	if alice.Kills != 1 || alice.HeadshotKills != 1 {
		t.Errorf("Alice overtime: got kills=%d hs=%d, want 1/1", alice.Kills, alice.HeadshotKills)
	}
}

func TestCalculatePlayerRoundStats_DamageFromFloat64(t *testing.T) {
	// JSON-decoded ExtraData may have float64 instead of int for health_damage.
	rounds := []RoundData{
		{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
	}
	events := []GameEvent{
		{
			Tick:            50,
			RoundNumber:     1,
			Type:            "player_hurt",
			AttackerSteamID: "1001",
			VictimSteamID:   "2001",
			ExtraData: map[string]interface{}{
				"health_damage": float64(55),
			},
		},
		killEvent(1, 100, "1001", "2001", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Alice",
			"attacker_team": "CT",
			"victim_name":   "Bob",
			"victim_team":   "T",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)
	alice := findPlayerStats(result[1], "1001")
	if alice == nil {
		t.Fatal("expected stats for Alice")
	}
	if alice.Damage != 55 {
		t.Errorf("Alice damage from float64: got %d, want 55", alice.Damage)
	}
}

func TestCalculatePlayerRoundStats_PlayerNameAndTeam(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
	}
	events := []GameEvent{
		killEvent(1, 100, "1001", "2001", map[string]interface{}{
			"headshot":      false,
			"attacker_name": "Alice",
			"attacker_team": "CT",
			"victim_name":   "Bob",
			"victim_team":   "T",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)
	stats := result[1]

	alice := findPlayerStats(stats, "1001")
	if alice == nil {
		t.Fatal("expected stats for Alice")
	}
	if alice.PlayerName != "Alice" {
		t.Errorf("Alice name: got %q, want %q", alice.PlayerName, "Alice")
	}
	if alice.TeamSide != "CT" {
		t.Errorf("Alice team: got %q, want %q", alice.TeamSide, "CT")
	}

	bob := findPlayerStats(stats, "2001")
	if bob == nil {
		t.Fatal("expected stats for Bob")
	}
	if bob.PlayerName != "Bob" {
		t.Errorf("Bob name: got %q, want %q", bob.PlayerName, "Bob")
	}
	if bob.TeamSide != "T" {
		t.Errorf("Bob team: got %q, want %q", bob.TeamSide, "T")
	}
}
