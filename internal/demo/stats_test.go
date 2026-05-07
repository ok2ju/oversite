package demo

import (
	"testing"
)

// -- helper: find a player's stats in the result slice by steamID.
func findStats(stats []PlayerRoundStats, steamID string) (PlayerRoundStats, bool) {
	for _, s := range stats {
		if s.SteamID == steamID {
			return s, true
		}
	}
	return PlayerRoundStats{}, false
}

// -- synthetic event builders --------------------------------------------------

func makeKillEvent(tick, round int, attackerID, victimID, weapon string, headshot bool, extra map[string]interface{}) GameEvent {
	if extra == nil {
		extra = map[string]interface{}{}
	}
	extra["headshot"] = headshot
	if _, ok := extra["attacker_name"]; !ok {
		extra["attacker_name"] = "Attacker"
	}
	if _, ok := extra["attacker_team"]; !ok {
		extra["attacker_team"] = "CT"
	}
	if _, ok := extra["victim_name"]; !ok {
		extra["victim_name"] = "Victim"
	}
	if _, ok := extra["victim_team"]; !ok {
		extra["victim_team"] = "T"
	}
	return GameEvent{
		Tick:            tick,
		RoundNumber:     round,
		Type:            "kill",
		AttackerSteamID: attackerID,
		VictimSteamID:   victimID,
		Weapon:          weapon,
		ExtraData:       extra,
	}
}

func makeHurtEvent(tick, round int, attackerID, victimID string, healthDamage int, attackerTeam, victimTeam string) GameEvent {
	return GameEvent{
		Tick:            tick,
		RoundNumber:     round,
		Type:            "player_hurt",
		AttackerSteamID: attackerID,
		VictimSteamID:   victimID,
		ExtraData: map[string]interface{}{
			"health_damage": healthDamage,
			"attacker_name": "Attacker",
			"attacker_team": attackerTeam,
			"victim_name":   "Victim",
			"victim_team":   victimTeam,
		},
	}
}

// -- TestGetExtraDataString ---------------------------------------------------

func TestGetExtraDataString(t *testing.T) {
	tests := []struct {
		name  string
		extra map[string]interface{}
		key   string
		want  string
	}{
		{
			name:  "existing string key",
			extra: map[string]interface{}{"weapon": "ak47"},
			key:   "weapon",
			want:  "ak47",
		},
		{
			name:  "missing key",
			extra: map[string]interface{}{"weapon": "ak47"},
			key:   "missing",
			want:  "",
		},
		{
			name:  "non-string value (int)",
			extra: map[string]interface{}{"damage": 80},
			key:   "damage",
			want:  "",
		},
		{
			name:  "nil map",
			extra: nil,
			key:   "weapon",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getExtraDataString(tt.extra, tt.key)
			if got != tt.want {
				t.Errorf("getExtraDataString(%v, %q) = %q, want %q", tt.extra, tt.key, got, tt.want)
			}
		})
	}
}

// -- TestGetExtraDataBool -----------------------------------------------------

func TestGetExtraDataBool(t *testing.T) {
	tests := []struct {
		name  string
		extra map[string]interface{}
		key   string
		want  bool
	}{
		{
			name:  "existing bool key true",
			extra: map[string]interface{}{"headshot": true},
			key:   "headshot",
			want:  true,
		},
		{
			name:  "existing bool key false",
			extra: map[string]interface{}{"headshot": false},
			key:   "headshot",
			want:  false,
		},
		{
			name:  "missing key",
			extra: map[string]interface{}{"headshot": true},
			key:   "missing",
			want:  false,
		},
		{
			name:  "non-bool value (string \"true\")",
			extra: map[string]interface{}{"headshot": "true"},
			key:   "headshot",
			want:  false,
		},
		{
			name:  "nil map",
			extra: nil,
			key:   "headshot",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getExtraDataBool(tt.extra, tt.key)
			if got != tt.want {
				t.Errorf("getExtraDataBool(%v, %q) = %v, want %v", tt.extra, tt.key, got, tt.want)
			}
		})
	}
}

// -- TestGetExtraDataInt ------------------------------------------------------

func TestGetExtraDataInt(t *testing.T) {
	tests := []struct {
		name  string
		extra map[string]interface{}
		key   string
		want  int
	}{
		{
			name:  "native int",
			extra: map[string]interface{}{"damage": 80},
			key:   "damage",
			want:  80,
		},
		{
			name:  "float64 (JSON decoded)",
			extra: map[string]interface{}{"damage": float64(45)},
			key:   "damage",
			want:  45,
		},
		{
			name:  "int64",
			extra: map[string]interface{}{"damage": int64(100)},
			key:   "damage",
			want:  100,
		},
		{
			name:  "missing key",
			extra: map[string]interface{}{"damage": 80},
			key:   "missing",
			want:  0,
		},
		{
			name:  "non-numeric value (string)",
			extra: map[string]interface{}{"damage": "80"},
			key:   "damage",
			want:  0,
		},
		{
			name:  "nil map",
			extra: nil,
			key:   "damage",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getExtraDataInt(tt.extra, tt.key)
			if got != tt.want {
				t.Errorf("getExtraDataInt(%v, %q) = %d, want %d", tt.extra, tt.key, got, tt.want)
			}
		})
	}
}

// -- TestIsClutching ----------------------------------------------------------

func TestIsClutching(t *testing.T) {
	tests := []struct {
		name      string
		steamID   string
		team      string
		teamAlive map[string]map[string]bool
		want      bool
	}{
		{
			name:    "solo alive, enemies alive → clutching",
			steamID: "player1",
			team:    "CT",
			teamAlive: map[string]map[string]bool{
				"CT": {"player1": true},
				"T":  {"enemy1": true, "enemy2": true},
			},
			want: true,
		},
		{
			name:    "solo alive, no enemies → not clutching",
			steamID: "player1",
			team:    "CT",
			teamAlive: map[string]map[string]bool{
				"CT": {"player1": true},
				"T":  {},
			},
			want: false,
		},
		{
			name:    "two alive on team → not clutching",
			steamID: "player1",
			team:    "CT",
			teamAlive: map[string]map[string]bool{
				"CT": {"player1": true, "player2": true},
				"T":  {"enemy1": true},
			},
			want: false,
		},
		{
			name:    "player not in alive set",
			steamID: "player1",
			team:    "CT",
			teamAlive: map[string]map[string]bool{
				"CT": {"player2": true},
				"T":  {"enemy1": true},
			},
			want: false,
		},
		{
			name:      "nil team map",
			steamID:   "player1",
			team:      "CT",
			teamAlive: nil,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isClutching(tt.steamID, tt.team, tt.teamAlive)
			if got != tt.want {
				t.Errorf("isClutching(%q, %q, ...) = %v, want %v", tt.steamID, tt.team, got, tt.want)
			}
		})
	}
}

// -- TestCalculatePlayerRoundStats --------------------------------------------

func TestCalculatePlayerRoundStats_BasicKillDeath(t *testing.T) {
	rounds := []RoundData{{Number: 1}}
	events := []GameEvent{
		makeKillEvent(100, 1, "steamA", "steamB", "ak47", true, nil),
		makeKillEvent(200, 1, "steamA", "steamC", "ak47", false, map[string]interface{}{
			"attacker_name": "PlayerA",
			"attacker_team": "CT",
			"victim_name":   "PlayerC",
			"victim_team":   "T",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)

	roundStats, ok := result[1]
	if !ok {
		t.Fatal("expected stats for round 1")
	}

	a, ok := findStats(roundStats, "steamA")
	if !ok {
		t.Fatal("expected stats for steamA")
	}
	if a.Kills != 2 {
		t.Errorf("steamA kills = %d, want 2", a.Kills)
	}
	if a.HeadshotKills != 1 {
		t.Errorf("steamA headshot kills = %d, want 1", a.HeadshotKills)
	}

	b, ok := findStats(roundStats, "steamB")
	if !ok {
		t.Fatal("expected stats for steamB")
	}
	if b.Deaths != 1 {
		t.Errorf("steamB deaths = %d, want 1", b.Deaths)
	}

	c, ok := findStats(roundStats, "steamC")
	if !ok {
		t.Fatal("expected stats for steamC")
	}
	if c.Deaths != 1 {
		t.Errorf("steamC deaths = %d, want 1", c.Deaths)
	}
}

func TestCalculatePlayerRoundStats_DamageTracking(t *testing.T) {
	rounds := []RoundData{{Number: 1}}
	events := []GameEvent{
		makeHurtEvent(100, 1, "steamA", "steamB", 80, "CT", "T"),
		makeHurtEvent(200, 1, "steamA", "steamC", 45, "CT", "T"),
	}

	result := CalculatePlayerRoundStats(rounds, events)

	roundStats, ok := result[1]
	if !ok {
		t.Fatal("expected stats for round 1")
	}

	a, ok := findStats(roundStats, "steamA")
	if !ok {
		t.Fatal("expected stats for steamA")
	}
	if a.Damage != 125 {
		t.Errorf("steamA damage = %d, want 125", a.Damage)
	}
}

func TestCalculatePlayerRoundStats_Assists(t *testing.T) {
	rounds := []RoundData{{Number: 1}}
	events := []GameEvent{
		makeKillEvent(100, 1, "steamA", "steamB", "ak47", false, map[string]interface{}{
			"attacker_name":     "PlayerA",
			"attacker_team":     "CT",
			"victim_name":       "PlayerB",
			"victim_team":       "T",
			"assister_steam_id": "steamC",
			"assister_name":     "PlayerC",
			"assister_team":     "CT",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)

	roundStats, ok := result[1]
	if !ok {
		t.Fatal("expected stats for round 1")
	}

	c, ok := findStats(roundStats, "steamC")
	if !ok {
		t.Fatal("expected stats for steamC (assister)")
	}
	if c.Assists != 1 {
		t.Errorf("steamC assists = %d, want 1", c.Assists)
	}
}

func TestCalculatePlayerRoundStats_FirstKillDeath(t *testing.T) {
	rounds := []RoundData{{Number: 1}}
	events := []GameEvent{
		makeKillEvent(100, 1, "steamA", "steamB", "ak47", false, map[string]interface{}{
			"attacker_name": "PlayerA",
			"attacker_team": "CT",
			"victim_name":   "PlayerB",
			"victim_team":   "T",
		}),
		makeKillEvent(200, 1, "steamC", "steamD", "m4a1", false, map[string]interface{}{
			"attacker_name": "PlayerC",
			"attacker_team": "CT",
			"victim_name":   "PlayerD",
			"victim_team":   "T",
		}),
		makeKillEvent(300, 1, "steamE", "steamF", "awp", false, map[string]interface{}{
			"attacker_name": "PlayerE",
			"attacker_team": "CT",
			"victim_name":   "PlayerF",
			"victim_team":   "T",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)

	roundStats, ok := result[1]
	if !ok {
		t.Fatal("expected stats for round 1")
	}

	// First kill / first death should only be on the first kill event.
	a, ok := findStats(roundStats, "steamA")
	if !ok {
		t.Fatal("expected stats for steamA")
	}
	if !a.FirstKill {
		t.Error("steamA should have firstKill=true (first kill in round)")
	}

	b, ok := findStats(roundStats, "steamB")
	if !ok {
		t.Fatal("expected stats for steamB")
	}
	if !b.FirstDeath {
		t.Error("steamB should have firstDeath=true (first death in round)")
	}

	// Later kills should not get first* flags.
	c, ok := findStats(roundStats, "steamC")
	if !ok {
		t.Fatal("expected stats for steamC")
	}
	if c.FirstKill {
		t.Error("steamC should NOT have firstKill=true (not first kill in round)")
	}

	d, ok := findStats(roundStats, "steamD")
	if !ok {
		t.Fatal("expected stats for steamD")
	}
	if d.FirstDeath {
		t.Error("steamD should NOT have firstDeath=true (not first death in round)")
	}
}

func TestCalculatePlayerRoundStats_SelfKill(t *testing.T) {
	rounds := []RoundData{{Number: 1}}
	events := []GameEvent{
		makeKillEvent(100, 1, "steamA", "steamA", "world", false, map[string]interface{}{
			"attacker_name": "PlayerA",
			"attacker_team": "CT",
			"victim_name":   "PlayerA",
			"victim_team":   "CT",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)

	roundStats, ok := result[1]
	if !ok {
		t.Fatal("expected stats for round 1")
	}

	a, ok := findStats(roundStats, "steamA")
	if !ok {
		t.Fatal("expected stats for steamA")
	}
	if a.Kills != 0 {
		t.Errorf("steamA kills = %d, want 0 (self-kill should not count)", a.Kills)
	}
	if a.Deaths != 1 {
		t.Errorf("steamA deaths = %d, want 1 (victim still gets death)", a.Deaths)
	}
}

func TestCalculatePlayerRoundStats_ClutchKills(t *testing.T) {
	// 5v5: register all players through hurt events first so teamAlive is populated.
	// Then 4 CT teammates die, leaving steamCT1 as the last CT alive.
	// steamCT1 then kills an enemy → clutchKill.
	rounds := []RoundData{{Number: 1}}

	// Register all 10 players via hurt events (no kills yet).
	events := []GameEvent{
		// CT team: steamCT1 .. steamCT5
		makeHurtEvent(10, 1, "steamT1", "steamCT2", 100, "T", "CT"),
		makeHurtEvent(20, 1, "steamT2", "steamCT3", 100, "T", "CT"),
		makeHurtEvent(30, 1, "steamT3", "steamCT4", 100, "T", "CT"),
		makeHurtEvent(40, 1, "steamT4", "steamCT5", 100, "T", "CT"),
		// T team: steamT1 .. steamT5 (register steamCT1 and all T players)
		makeHurtEvent(50, 1, "steamT5", "steamCT1", 50, "T", "CT"),
	}

	// Kill 4 CT players (CT1 survives).
	events = append(events,
		makeKillEvent(60, 1, "steamT1", "steamCT2", "ak47", false, map[string]interface{}{
			"attacker_name": "T1",
			"attacker_team": "T",
			"victim_name":   "CT2",
			"victim_team":   "CT",
		}),
		makeKillEvent(70, 1, "steamT2", "steamCT3", "ak47", false, map[string]interface{}{
			"attacker_name": "T2",
			"attacker_team": "T",
			"victim_name":   "CT3",
			"victim_team":   "CT",
		}),
		makeKillEvent(80, 1, "steamT3", "steamCT4", "ak47", false, map[string]interface{}{
			"attacker_name": "T3",
			"attacker_team": "T",
			"victim_name":   "CT4",
			"victim_team":   "CT",
		}),
		makeKillEvent(90, 1, "steamT4", "steamCT5", "ak47", false, map[string]interface{}{
			"attacker_name": "T4",
			"attacker_team": "T",
			"victim_name":   "CT5",
			"victim_team":   "CT",
		}),
		// Now CT1 is the last CT alive; kills T1 → clutch kill.
		makeKillEvent(100, 1, "steamCT1", "steamT1", "deagle", false, map[string]interface{}{
			"attacker_name": "CT1",
			"attacker_team": "CT",
			"victim_name":   "T1",
			"victim_team":   "T",
		}),
	)

	result := CalculatePlayerRoundStats(rounds, events)

	roundStats, ok := result[1]
	if !ok {
		t.Fatal("expected stats for round 1")
	}

	ct1, ok := findStats(roundStats, "steamCT1")
	if !ok {
		t.Fatal("expected stats for steamCT1")
	}
	if ct1.ClutchKills != 1 {
		t.Errorf("steamCT1 clutchKills = %d, want 1", ct1.ClutchKills)
	}
}

func TestCalculatePlayerRoundStats_EmptyEvents(t *testing.T) {
	rounds := []RoundData{{Number: 1}}
	events := []GameEvent{}

	result := CalculatePlayerRoundStats(rounds, events)

	if len(result) != 0 {
		t.Errorf("expected empty result for round with no events, got %d round(s)", len(result))
	}
}

// TestCalculatePlayerRoundStats_RosterSeed verifies that every alive
// participant gets a player_rounds row with correct name + team + zero stats
// even when no events fire — fixing the numeric-nickname bug surfaced in the
// viewer for passive players.
func TestCalculatePlayerRoundStats_RosterSeed(t *testing.T) {
	roster := []RoundParticipant{
		{SteamID: "steam_ct1", PlayerName: "CT1", TeamSide: "CT"},
		{SteamID: "steam_ct2", PlayerName: "CT2", TeamSide: "CT"},
		{SteamID: "steam_ct3", PlayerName: "CT3", TeamSide: "CT"},
		{SteamID: "steam_ct4", PlayerName: "CT4", TeamSide: "CT"},
		{SteamID: "steam_ct5", PlayerName: "CT5", TeamSide: "CT"},
		{SteamID: "steam_t1", PlayerName: "T1", TeamSide: "T"},
		{SteamID: "steam_t2", PlayerName: "T2", TeamSide: "T"},
		{SteamID: "steam_t3", PlayerName: "T3", TeamSide: "T"},
		{SteamID: "steam_t4", PlayerName: "T4", TeamSide: "T"},
		{SteamID: "steam_t5", PlayerName: "T5", TeamSide: "T"},
	}
	rounds := []RoundData{{Number: 1, Roster: roster}}

	result := CalculatePlayerRoundStats(rounds, nil)

	stats, ok := result[1]
	if !ok {
		t.Fatal("expected stats for round 1")
	}
	if len(stats) != 10 {
		t.Fatalf("len(stats) = %d, want 10 (one row per roster entry)", len(stats))
	}
	for _, rp := range roster {
		ps, ok := findStats(stats, rp.SteamID)
		if !ok {
			t.Errorf("roster member %q not present in stats", rp.SteamID)
			continue
		}
		if ps.PlayerName != rp.PlayerName {
			t.Errorf("%s name = %q, want %q", rp.SteamID, ps.PlayerName, rp.PlayerName)
		}
		if ps.TeamSide != rp.TeamSide {
			t.Errorf("%s team = %q, want %q", rp.SteamID, ps.TeamSide, rp.TeamSide)
		}
		if ps.Kills != 0 || ps.Deaths != 0 || ps.Damage != 0 {
			t.Errorf("%s expected zero stats, got K=%d D=%d Dmg=%d",
				rp.SteamID, ps.Kills, ps.Deaths, ps.Damage)
		}
	}
}

// TestCalculatePlayerRoundStats_RosterSeedPlusKills verifies kill/death events
// layer correctly on top of the seeded roster.
func TestCalculatePlayerRoundStats_RosterSeedPlusKills(t *testing.T) {
	roster := []RoundParticipant{
		{SteamID: "steam_ct1", PlayerName: "CT1", TeamSide: "CT"},
		{SteamID: "steam_ct2", PlayerName: "CT2", TeamSide: "CT"},
		{SteamID: "steam_t1", PlayerName: "T1", TeamSide: "T"},
		{SteamID: "steam_t2", PlayerName: "T2", TeamSide: "T"},
	}
	rounds := []RoundData{{Number: 1, Roster: roster}}
	events := []GameEvent{
		makeKillEvent(100, 1, "steam_ct1", "steam_t1", "ak47", false, map[string]interface{}{
			"attacker_name": "CT1",
			"attacker_team": "CT",
			"victim_name":   "T1",
			"victim_team":   "T",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)

	stats, ok := result[1]
	if !ok {
		t.Fatal("expected stats for round 1")
	}
	if len(stats) != 4 {
		t.Fatalf("len(stats) = %d, want 4", len(stats))
	}

	attacker, _ := findStats(stats, "steam_ct1")
	if attacker.Kills != 1 {
		t.Errorf("steam_ct1 kills = %d, want 1", attacker.Kills)
	}
	victim, _ := findStats(stats, "steam_t1")
	if victim.Deaths != 1 {
		t.Errorf("steam_t1 deaths = %d, want 1", victim.Deaths)
	}
	for _, sid := range []string{"steam_ct2", "steam_t2"} {
		ps, _ := findStats(stats, sid)
		if ps.Kills != 0 || ps.Deaths != 0 {
			t.Errorf("%s should have zero stats, got K=%d D=%d", sid, ps.Kills, ps.Deaths)
		}
	}
}

// TestCalculatePlayerRoundStats_LateJoinerNotInRoster verifies the getPlayer
// fallback still registers players who appear in events but were not in the
// freeze-end roster (e.g. mid-round reconnects).
func TestCalculatePlayerRoundStats_LateJoinerNotInRoster(t *testing.T) {
	roster := []RoundParticipant{
		{SteamID: "steam_ct1", PlayerName: "CT1", TeamSide: "CT"},
	}
	rounds := []RoundData{{Number: 1, Roster: roster}}
	events := []GameEvent{
		// late joiner steam_t_late was not in the freeze-end snapshot.
		makeKillEvent(100, 1, "steam_t_late", "steam_ct1", "ak47", false, map[string]interface{}{
			"attacker_name": "TLate",
			"attacker_team": "T",
			"victim_name":   "CT1",
			"victim_team":   "CT",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)

	stats, ok := result[1]
	if !ok {
		t.Fatal("expected stats for round 1")
	}
	if len(stats) != 2 {
		t.Fatalf("len(stats) = %d, want 2 (1 roster + 1 late joiner)", len(stats))
	}
	late, ok := findStats(stats, "steam_t_late")
	if !ok {
		t.Fatal("late joiner not registered via getPlayer fallback")
	}
	if late.Kills != 1 {
		t.Errorf("late joiner kills = %d, want 1", late.Kills)
	}
	if late.PlayerName != "TLate" {
		t.Errorf("late joiner name = %q, want %q", late.PlayerName, "TLate")
	}
}

func TestCalculatePlayerRoundStats_MultiRound(t *testing.T) {
	rounds := []RoundData{{Number: 1}, {Number: 2}}
	events := []GameEvent{
		makeKillEvent(100, 1, "steamA", "steamB", "ak47", false, map[string]interface{}{
			"attacker_name": "PlayerA",
			"attacker_team": "CT",
			"victim_name":   "PlayerB",
			"victim_team":   "T",
		}),
		makeKillEvent(200, 2, "steamC", "steamD", "m4a1", false, map[string]interface{}{
			"attacker_name": "PlayerC",
			"attacker_team": "T",
			"victim_name":   "PlayerD",
			"victim_team":   "CT",
		}),
	}

	result := CalculatePlayerRoundStats(rounds, events)

	// Round 1: steamA kills steamB.
	r1, ok := result[1]
	if !ok {
		t.Fatal("expected stats for round 1")
	}
	a, ok := findStats(r1, "steamA")
	if !ok {
		t.Fatal("expected stats for steamA in round 1")
	}
	if a.Kills != 1 {
		t.Errorf("round 1 steamA kills = %d, want 1", a.Kills)
	}
	if _, inRound1 := findStats(r1, "steamC"); inRound1 {
		t.Error("steamC should NOT appear in round 1 stats")
	}

	// Round 2: steamC kills steamD.
	r2, ok := result[2]
	if !ok {
		t.Fatal("expected stats for round 2")
	}
	c, ok := findStats(r2, "steamC")
	if !ok {
		t.Fatal("expected stats for steamC in round 2")
	}
	if c.Kills != 1 {
		t.Errorf("round 2 steamC kills = %d, want 1", c.Kills)
	}
	if _, inRound2 := findStats(r2, "steamA"); inRound2 {
		t.Error("steamA should NOT appear in round 2 stats")
	}
}
