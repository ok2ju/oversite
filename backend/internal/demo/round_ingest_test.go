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

// hurtEventWithNames builds a player_hurt event with attacker/victim names and teams.
func hurtEventWithNames(round, tick int, attackerID, victimID string, damage int, extra map[string]interface{}) GameEvent {
	ev := hurtEvent(round, tick, attackerID, victimID, damage)
	for k, v := range extra {
		ev.ExtraData[k] = v
	}
	return ev
}

func findPlayerStats(stats []PlayerRoundStats, steamID string) *PlayerRoundStats {
	for i := range stats {
		if stats[i].SteamID == steamID {
			return &stats[i]
		}
	}
	return nil
}

func TestCalculatePlayerRoundStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		rounds []RoundData
		events []GameEvent
		check  func(t *testing.T, result map[int][]PlayerRoundStats)
	}{
		{
			name: "basic KDA",
			rounds: []RoundData{
				{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
			},
			events: []GameEvent{
				killEvent(1, 100, "1001", "2001", map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Bob", "victim_team": "T",
				}),
				killEvent(1, 200, "1001", "2002", map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Charlie", "victim_team": "T",
				}),
				killEvent(1, 300, "2003", "1002", map[string]interface{}{
					"headshot": false, "attacker_name": "Dave", "attacker_team": "T",
					"victim_name": "Eve", "victim_team": "CT",
				}),
			},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
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
			},
		},
		{
			name: "headshot kills",
			rounds: []RoundData{
				{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
			},
			events: []GameEvent{
				killEvent(1, 100, "1001", "2001", map[string]interface{}{
					"headshot": true, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Bob", "victim_team": "T",
				}),
				killEvent(1, 200, "1001", "2002", map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Charlie", "victim_team": "T",
				}),
			},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
				alice := findPlayerStats(result[1], "1001")
				if alice == nil {
					t.Fatal("expected stats for Alice")
				}
				if alice.HeadshotKills != 1 {
					t.Errorf("Alice headshot kills: got %d, want 1", alice.HeadshotKills)
				}
			},
		},
		{
			name: "first kill and first death",
			rounds: []RoundData{
				{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
			},
			events: []GameEvent{
				killEvent(1, 100, "1001", "2001", map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Bob", "victim_team": "T",
				}),
				killEvent(1, 200, "1002", "2002", map[string]interface{}{
					"headshot": false, "attacker_name": "Eve", "attacker_team": "CT",
					"victim_name": "Charlie", "victim_team": "T",
				}),
			},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
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
			},
		},
		{
			name: "assists",
			rounds: []RoundData{
				{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
			},
			events: []GameEvent{
				killEvent(1, 100, "1001", "2001", map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Bob", "victim_team": "T",
					"assister_steam_id": "1002", "assister_name": "Eve", "assister_team": "CT",
				}),
				killEvent(1, 200, "1001", "2002", map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Charlie", "victim_team": "T",
					"assister_steam_id": "1002", "assister_name": "Eve", "assister_team": "CT",
				}),
			},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
				eve := findPlayerStats(result[1], "1002")
				if eve == nil {
					t.Fatal("expected stats for assister (1002)")
				}
				if eve.Assists != 2 {
					t.Errorf("assister assists: got %d, want 2", eve.Assists)
				}
			},
		},
		{
			name: "damage accumulation",
			rounds: []RoundData{
				{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
			},
			events: []GameEvent{
				hurtEvent(1, 50, "1001", "2001", 40),
				hurtEvent(1, 60, "1001", "2001", 60),
				hurtEvent(1, 70, "1001", "2002", 25),
				killEvent(1, 100, "1001", "2001", map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Bob", "victim_team": "T",
				}),
			},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
				alice := findPlayerStats(result[1], "1001")
				if alice == nil {
					t.Fatal("expected stats for Alice")
				}
				if alice.Damage != 125 {
					t.Errorf("Alice damage: got %d, want 125", alice.Damage)
				}
			},
		},
		{
			name: "clutch kills 1v2",
			rounds: []RoundData{
				{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
			},
			events: []GameEvent{
				// T kills CT teammates first, leaving Alice alone
				killEvent(1, 100, "2001", "1002", map[string]interface{}{
					"headshot": false, "attacker_name": "Bob", "attacker_team": "T",
					"victim_name": "Eve", "victim_team": "CT",
				}),
				killEvent(1, 150, "2002", "1003", map[string]interface{}{
					"headshot": false, "attacker_name": "Charlie", "attacker_team": "T",
					"victim_name": "Frank", "victim_team": "CT",
				}),
				// Now Alice (1001) is 1v2 — her kills are clutch kills
				killEvent(1, 200, "1001", "2001", map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Bob", "victim_team": "T",
				}),
				killEvent(1, 300, "1001", "2002", map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Charlie", "victim_team": "T",
				}),
			},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
				alice := findPlayerStats(result[1], "1001")
				if alice == nil {
					t.Fatal("expected stats for Alice")
				}
				if alice.ClutchKills != 2 {
					t.Errorf("Alice clutch kills: got %d, want 2", alice.ClutchKills)
				}
			},
		},
		{
			name: "no false clutch when teammate alive via hurt events",
			rounds: []RoundData{
				{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
			},
			events: []GameEvent{
				// Eve (1002) takes damage but is alive — registered via hurt event.
				hurtEventWithNames(1, 50, "2001", "1002", 30, map[string]interface{}{
					"attacker_name": "Bob", "attacker_team": "T",
					"victim_name": "Eve", "victim_team": "CT",
				}),
				// Alice kills Bob — should NOT be a clutch because Eve is alive.
				killEvent(1, 100, "1001", "2001", map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Bob", "victim_team": "T",
				}),
			},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
				alice := findPlayerStats(result[1], "1001")
				if alice == nil {
					t.Fatal("expected stats for Alice")
				}
				if alice.ClutchKills != 0 {
					t.Errorf("Alice clutch kills: got %d, want 0 (Eve is alive)", alice.ClutchKills)
				}
			},
		},
		{
			name: "world kill no attacker",
			rounds: []RoundData{
				{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
			},
			events: []GameEvent{
				killEvent(1, 100, "", "2001", map[string]interface{}{
					"headshot": false, "victim_name": "Bob", "victim_team": "T",
				}),
			},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
				bob := findPlayerStats(result[1], "2001")
				if bob == nil {
					t.Fatal("expected stats for Bob")
				}
				if bob.Deaths != 1 {
					t.Errorf("Bob deaths: got %d, want 1", bob.Deaths)
				}
				for _, s := range result[1] {
					if s.SteamID == "" {
						t.Error("should not have stats entry for empty steam ID (world)")
					}
				}
			},
		},
		{
			name: "self kill not credited",
			rounds: []RoundData{
				{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
			},
			events: []GameEvent{
				killEvent(1, 100, "2001", "2001", map[string]interface{}{
					"headshot": false, "attacker_name": "Bob", "attacker_team": "T",
					"victim_name": "Bob", "victim_team": "T",
				}),
			},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
				bob := findPlayerStats(result[1], "2001")
				if bob == nil {
					t.Fatal("expected stats for Bob")
				}
				if bob.Deaths != 1 {
					t.Errorf("Bob deaths: got %d, want 1", bob.Deaths)
				}
				if bob.Kills != 0 {
					t.Errorf("Bob kills on self-kill: got %d, want 0", bob.Kills)
				}
			},
		},
		{
			name: "no events empty round",
			rounds: []RoundData{
				{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
			},
			events: []GameEvent{},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
				stats, ok := result[1]
				if ok && len(stats) != 0 {
					t.Errorf("expected 0 player stats for empty round, got %d", len(stats))
				}
			},
		},
		{
			name: "multiple rounds",
			rounds: []RoundData{
				{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
				{Number: 2, StartTick: 1001, EndTick: 2000, WinnerSide: "T"},
			},
			events: []GameEvent{
				killEvent(1, 100, "1001", "2001", map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Bob", "victim_team": "T",
				}),
				killEvent(2, 1100, "2001", "1001", map[string]interface{}{
					"headshot": true, "attacker_name": "Bob", "attacker_team": "T",
					"victim_name": "Alice", "victim_team": "CT",
				}),
			},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
				if len(result) != 2 {
					t.Fatalf("expected 2 rounds, got %d", len(result))
				}
				r1Alice := findPlayerStats(result[1], "1001")
				if r1Alice == nil || r1Alice.Kills != 1 {
					t.Errorf("Round 1 Alice kills: got %v, want 1", r1Alice)
				}
				r1Bob := findPlayerStats(result[1], "2001")
				if r1Bob == nil || r1Bob.Deaths != 1 {
					t.Errorf("Round 1 Bob deaths: got %v, want 1", r1Bob)
				}
				r2Bob := findPlayerStats(result[2], "2001")
				if r2Bob == nil || r2Bob.Kills != 1 || r2Bob.HeadshotKills != 1 {
					t.Errorf("Round 2 Bob: got %+v, want 1 kill 1 hs", r2Bob)
				}
				r2Alice := findPlayerStats(result[2], "1001")
				if r2Alice == nil || r2Alice.Deaths != 1 {
					t.Errorf("Round 2 Alice deaths: got %v, want 1", r2Alice)
				}
			},
		},
		{
			name: "overtime rounds",
			rounds: []RoundData{
				{Number: 25, StartTick: 50000, EndTick: 51000, WinnerSide: "CT", IsOvertime: true},
			},
			events: []GameEvent{
				killEvent(25, 50100, "1001", "2001", map[string]interface{}{
					"headshot": true, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Bob", "victim_team": "T",
				}),
			},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
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
			},
		},
		{
			name: "damage from float64 JSON",
			rounds: []RoundData{
				{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
			},
			events: []GameEvent{
				{
					Tick: 50, RoundNumber: 1, Type: "player_hurt",
					AttackerSteamID: "1001", VictimSteamID: "2001",
					ExtraData: map[string]interface{}{
						"health_damage": float64(55),
					},
				},
				killEvent(1, 100, "1001", "2001", map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Bob", "victim_team": "T",
				}),
			},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
				alice := findPlayerStats(result[1], "1001")
				if alice == nil {
					t.Fatal("expected stats for Alice")
				}
				if alice.Damage != 55 {
					t.Errorf("Alice damage from float64: got %d, want 55", alice.Damage)
				}
			},
		},
		{
			name: "player name and team propagation",
			rounds: []RoundData{
				{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
			},
			events: []GameEvent{
				killEvent(1, 100, "1001", "2001", map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Bob", "victim_team": "T",
				}),
			},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
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
			},
		},
		{
			name: "flash assister gets name and team",
			rounds: []RoundData{
				{Number: 1, StartTick: 0, EndTick: 1000, WinnerSide: "CT"},
			},
			events: []GameEvent{
				killEvent(1, 100, "1001", "2001", map[string]interface{}{
					"headshot": false, "attacker_name": "Alice", "attacker_team": "CT",
					"victim_name": "Bob", "victim_team": "T",
					"assister_steam_id": "1002", "assister_name": "Eve", "assister_team": "CT",
					"flash_assist": true,
				}),
			},
			check: func(t *testing.T, result map[int][]PlayerRoundStats) {
				eve := findPlayerStats(result[1], "1002")
				if eve == nil {
					t.Fatal("expected stats for flash assister Eve (1002)")
				}
				if eve.PlayerName != "Eve" {
					t.Errorf("assister name: got %q, want %q", eve.PlayerName, "Eve")
				}
				if eve.TeamSide != "CT" {
					t.Errorf("assister team: got %q, want %q", eve.TeamSide, "CT")
				}
				if eve.Assists != 1 {
					t.Errorf("assister assists: got %d, want 1", eve.Assists)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := CalculatePlayerRoundStats(tt.rounds, tt.events)
			tt.check(t, result)
		})
	}
}
