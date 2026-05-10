package demo

import (
	"math"
	"testing"
)

// floatClose is a small abs-tolerance comparator used by the Phase 2 movement
// tests below; the floating-point math involves multiple multiplies so a
// strict equality check is too brittle.
func floatClose(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

// findRound returns the per-round detail for round n, or zero-value + false.
func findRound(rounds []PlayerRoundDetail, n int) (PlayerRoundDetail, bool) {
	for _, r := range rounds {
		if r.RoundNumber == n {
			return r, true
		}
	}
	return PlayerRoundDetail{}, false
}

// findOpponent returns the damage-by-opponent row for steamID, or zero-value + false.
func findOpponent(rows []DamageByOpponent, steamID string) (DamageByOpponent, bool) {
	for _, r := range rows {
		if r.SteamID == steamID {
			return r, true
		}
	}
	return DamageByOpponent{}, false
}

// findWeapon returns the damage-by-weapon row for weapon (lowercased).
func findWeapon(rows []DamageByWeapon, weapon string) (DamageByWeapon, bool) {
	for _, r := range rows {
		if r.Weapon == weapon {
			return r, true
		}
	}
	return DamageByWeapon{}, false
}

func twoRoundRoster() []RoundParticipant {
	return []RoundParticipant{
		{SteamID: "steamA", PlayerName: "PlayerA", TeamSide: "CT"},
		{SteamID: "steamB", PlayerName: "PlayerB", TeamSide: "T"},
		{SteamID: "steamC", PlayerName: "PlayerC", TeamSide: "T"},
	}
}

func TestComputePlayerMatchStats_AggregatesKillsAndDeaths(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, Roster: twoRoundRoster()},
		{Number: 2, Roster: twoRoundRoster()},
	}
	events := []GameEvent{
		makeKillEvent(100, 1, "steamA", "steamB", "ak47", true, &KillExtra{
			AttackerName: "PlayerA", AttackerTeam: "CT",
			VictimName: "PlayerB", VictimTeam: "T",
		}),
		makeKillEvent(150, 1, "steamA", "steamC", "ak47", false, &KillExtra{
			AttackerName: "PlayerA", AttackerTeam: "CT",
			VictimName: "PlayerC", VictimTeam: "T",
		}),
		makeKillEvent(220, 2, "steamB", "steamA", "deagle", false, &KillExtra{
			AttackerName: "PlayerB", AttackerTeam: "T",
			VictimName: "PlayerA", VictimTeam: "CT",
		}),
	}

	stats := ComputePlayerMatchStats(rounds, events, nil, nil, nil, "steamA", 64)

	if stats.RoundsPlayed != 2 {
		t.Errorf("RoundsPlayed = %d, want 2", stats.RoundsPlayed)
	}
	if stats.Kills != 2 {
		t.Errorf("Kills = %d, want 2", stats.Kills)
	}
	if stats.Deaths != 1 {
		t.Errorf("Deaths = %d, want 1", stats.Deaths)
	}
	if stats.HeadshotKills != 1 {
		t.Errorf("HeadshotKills = %d, want 1", stats.HeadshotKills)
	}
	if stats.HSPercent != 50 {
		t.Errorf("HSPercent = %v, want 50", stats.HSPercent)
	}
	if stats.TeamSide != "CT" {
		t.Errorf("TeamSide = %q, want CT", stats.TeamSide)
	}
	if stats.PlayerName != "PlayerA" {
		t.Errorf("PlayerName = %q, want PlayerA", stats.PlayerName)
	}
}

func TestComputePlayerMatchStats_DamageBreakdowns(t *testing.T) {
	rounds := []RoundData{{Number: 1, Roster: twoRoundRoster()}}
	events := []GameEvent{
		// 50 dmg to B with ak, 30 dmg to B with deagle, 75 dmg to C with ak.
		{
			Tick: 100, RoundNumber: 1, Type: "player_hurt",
			AttackerSteamID: "steamA", VictimSteamID: "steamB", Weapon: "AK-47",
			ExtraData: &PlayerHurtExtra{HealthDamage: 50, AttackerName: "PlayerA", AttackerTeam: "CT", VictimName: "PlayerB", VictimTeam: "T"},
		},
		{
			Tick: 110, RoundNumber: 1, Type: "player_hurt",
			AttackerSteamID: "steamA", VictimSteamID: "steamB", Weapon: "Deagle",
			ExtraData: &PlayerHurtExtra{HealthDamage: 30, AttackerName: "PlayerA", AttackerTeam: "CT", VictimName: "PlayerB", VictimTeam: "T"},
		},
		{
			Tick: 120, RoundNumber: 1, Type: "player_hurt",
			AttackerSteamID: "steamA", VictimSteamID: "steamC", Weapon: "AK-47",
			ExtraData: &PlayerHurtExtra{HealthDamage: 75, AttackerName: "PlayerA", AttackerTeam: "CT", VictimName: "PlayerC", VictimTeam: "T"},
		},
		// Damage taken — must not be added to A's damage total.
		{
			Tick: 130, RoundNumber: 1, Type: "player_hurt",
			AttackerSteamID: "steamB", VictimSteamID: "steamA", Weapon: "AK-47",
			ExtraData: &PlayerHurtExtra{HealthDamage: 40, AttackerName: "PlayerB", AttackerTeam: "T", VictimName: "PlayerA", VictimTeam: "CT"},
		},
	}

	stats := ComputePlayerMatchStats(rounds, events, nil, nil, nil, "steamA", 64)

	if stats.Damage != 155 {
		t.Errorf("total damage = %d, want 155 (50+30+75)", stats.Damage)
	}
	if stats.ADR != 155 {
		t.Errorf("ADR = %v, want 155 (1 round)", stats.ADR)
	}

	ak, ok := findWeapon(stats.DamageByWeapon, "ak-47")
	if !ok {
		t.Fatalf("expected ak-47 in damage-by-weapon, got %+v", stats.DamageByWeapon)
	}
	if ak.Damage != 125 {
		t.Errorf("ak-47 damage = %d, want 125 (50+75)", ak.Damage)
	}
	deagle, ok := findWeapon(stats.DamageByWeapon, "deagle")
	if !ok {
		t.Fatalf("expected deagle in damage-by-weapon")
	}
	if deagle.Damage != 30 {
		t.Errorf("deagle damage = %d, want 30", deagle.Damage)
	}
	// Sort: highest first.
	if stats.DamageByWeapon[0].Damage < stats.DamageByWeapon[1].Damage {
		t.Errorf("damage-by-weapon not sorted desc: %+v", stats.DamageByWeapon)
	}

	bRow, ok := findOpponent(stats.DamageByOpponent, "steamB")
	if !ok {
		t.Fatalf("expected steamB in damage-by-opponent")
	}
	if bRow.Damage != 80 {
		t.Errorf("steamB damage = %d, want 80 (50+30)", bRow.Damage)
	}
	if bRow.PlayerName != "PlayerB" {
		t.Errorf("steamB name = %q, want PlayerB", bRow.PlayerName)
	}
	cRow, ok := findOpponent(stats.DamageByOpponent, "steamC")
	if !ok {
		t.Fatalf("expected steamC in damage-by-opponent")
	}
	if cRow.Damage != 75 {
		t.Errorf("steamC damage = %d, want 75", cRow.Damage)
	}
}

func TestComputePlayerMatchStats_FirstKillFirstDeathOpening(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, Roster: twoRoundRoster()},
		{Number: 2, Roster: twoRoundRoster()},
	}
	events := []GameEvent{
		// Round 1: A opens with the first kill on B.
		makeKillEvent(100, 1, "steamA", "steamB", "ak47", false, &KillExtra{
			AttackerName: "PlayerA", AttackerTeam: "CT",
			VictimName: "PlayerB", VictimTeam: "T",
		}),
		// Round 2: A dies first.
		makeKillEvent(200, 2, "steamB", "steamA", "deagle", false, &KillExtra{
			AttackerName: "PlayerB", AttackerTeam: "T",
			VictimName: "PlayerA", VictimTeam: "CT",
		}),
	}

	stats := ComputePlayerMatchStats(rounds, events, nil, nil, nil, "steamA", 64)
	if stats.FirstKills != 1 || stats.OpeningWins != 1 {
		t.Errorf("FirstKills = %d (want 1), OpeningWins = %d (want 1)", stats.FirstKills, stats.OpeningWins)
	}
	if stats.FirstDeaths != 1 || stats.OpeningLosses != 1 {
		t.Errorf("FirstDeaths = %d (want 1), OpeningLosses = %d (want 1)", stats.FirstDeaths, stats.OpeningLosses)
	}

	r1, ok := findRound(stats.Rounds, 1)
	if !ok {
		t.Fatal("expected round 1 detail")
	}
	if !r1.FirstKill {
		t.Error("round 1: FirstKill should be true")
	}
	r2, ok := findRound(stats.Rounds, 2)
	if !ok {
		t.Fatal("expected round 2 detail")
	}
	if !r2.FirstDeath {
		t.Error("round 2: FirstDeath should be true")
	}
}

func TestComputePlayerMatchStats_TradeKill(t *testing.T) {
	roster := []RoundParticipant{
		{SteamID: "steamA", PlayerName: "PlayerA", TeamSide: "CT"},
		{SteamID: "steamMate", PlayerName: "PlayerMate", TeamSide: "CT"},
		{SteamID: "steamB", PlayerName: "PlayerB", TeamSide: "T"},
	}
	rounds := []RoundData{{Number: 1, Roster: roster}}
	events := []GameEvent{
		// Mate dies at tick 100.
		makeKillEvent(100, 1, "steamB", "steamMate", "ak47", false, &KillExtra{
			AttackerName: "PlayerB", AttackerTeam: "T",
			VictimName: "PlayerMate", VictimTeam: "CT",
		}),
		// A trades within 5s (320 ticks at 64tps, so tick 200 = ~1.5s — within window).
		makeKillEvent(200, 1, "steamA", "steamB", "deagle", false, &KillExtra{
			AttackerName: "PlayerA", AttackerTeam: "CT",
			VictimName: "PlayerB", VictimTeam: "T",
		}),
	}

	stats := ComputePlayerMatchStats(rounds, events, nil, nil, nil, "steamA", 64)
	if stats.TradeKills != 1 {
		t.Errorf("TradeKills = %d, want 1", stats.TradeKills)
	}
	r1, ok := findRound(stats.Rounds, 1)
	if !ok {
		t.Fatal("expected round 1 detail")
	}
	if !r1.TradeKill {
		t.Error("round 1: TradeKill should be true")
	}
}

func TestComputePlayerMatchStats_TradeKillOutsideWindow(t *testing.T) {
	roster := []RoundParticipant{
		{SteamID: "steamA", PlayerName: "PlayerA", TeamSide: "CT"},
		{SteamID: "steamMate", PlayerName: "PlayerMate", TeamSide: "CT"},
		{SteamID: "steamB", PlayerName: "PlayerB", TeamSide: "T"},
	}
	rounds := []RoundData{{Number: 1, Roster: roster}}
	events := []GameEvent{
		// Mate dies at tick 100.
		makeKillEvent(100, 1, "steamB", "steamMate", "ak47", false, &KillExtra{
			AttackerName: "PlayerB", AttackerTeam: "T",
			VictimName: "PlayerMate", VictimTeam: "CT",
		}),
		// A retaliates 6 seconds later — outside the 5s trade window.
		makeKillEvent(100+6*64+1, 1, "steamA", "steamB", "deagle", false, &KillExtra{
			AttackerName: "PlayerA", AttackerTeam: "CT",
			VictimName: "PlayerB", VictimTeam: "T",
		}),
	}

	stats := ComputePlayerMatchStats(rounds, events, nil, nil, nil, "steamA", 64)
	if stats.TradeKills != 0 {
		t.Errorf("TradeKills = %d, want 0 (outside trade window)", stats.TradeKills)
	}
}

func TestComputePlayerMatchStats_LoadoutValueFromInventory(t *testing.T) {
	rounds := []RoundData{{Number: 1, Roster: twoRoundRoster()}}
	loadouts := map[int][]RoundLoadoutEntry{
		1: {
			{SteamID: "steamA", Inventory: "AK-47,Kevlar+Helmet,HEgrenade,Smokegrenade"},
			{SteamID: "steamB", Inventory: "Deagle"},
		},
	}

	stats := ComputePlayerMatchStats(rounds, nil, loadouts, nil, nil, "steamA", 64)
	r1, ok := findRound(stats.Rounds, 1)
	if !ok {
		t.Fatal("expected round 1 detail")
	}
	// AK 2700 + Kevlar+Helmet 1000 + HE 300 + Smoke 300 = 4300
	if r1.LoadoutValue != 4300 {
		t.Errorf("LoadoutValue = %d, want 4300", r1.LoadoutValue)
	}
}

func TestComputePlayerMatchStats_PlayerNotInRoster(t *testing.T) {
	rounds := []RoundData{{Number: 1, Roster: twoRoundRoster()}}
	stats := ComputePlayerMatchStats(rounds, nil, nil, nil, nil, "steamGhost", 64)
	if stats.RoundsPlayed != 0 {
		t.Errorf("RoundsPlayed = %d, want 0 for absent player", stats.RoundsPlayed)
	}
	if len(stats.Rounds) != 0 {
		t.Errorf("Rounds = %d, want 0 for absent player", len(stats.Rounds))
	}
}

func TestComputePlayerMatchStats_EmptyInputs(t *testing.T) {
	stats := ComputePlayerMatchStats(nil, nil, nil, nil, nil, "steamA", 64)
	if stats.RoundsPlayed != 0 {
		t.Errorf("RoundsPlayed = %d, want 0", stats.RoundsPlayed)
	}
	stats = ComputePlayerMatchStats([]RoundData{{Number: 1}}, nil, nil, nil, nil, "", 64)
	if stats.RoundsPlayed != 0 {
		t.Errorf("empty steamID: RoundsPlayed = %d, want 0", stats.RoundsPlayed)
	}
}

func TestSumLoadoutValue(t *testing.T) {
	tests := []struct {
		name      string
		inventory string
		want      int
	}{
		{"empty", "", 0},
		{"single ak", "AK-47", 2700},
		{"awp + kevlar+helmet", "awp,Kevlar+Helmet", 4750 + 1000},
		{"unknown items skipped", "knife,Deagle,unknown_item", 700},
		{"whitespace tolerated", "  AK-47 ,  Smokegrenade  ", 2700 + 300},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sumLoadoutValue(tt.inventory)
			if got != tt.want {
				t.Errorf("sumLoadoutValue(%q) = %d, want %d", tt.inventory, got, tt.want)
			}
		})
	}
}

// Phase 2 — movement / timing tests -----------------------------------------

func TestComputePlayerMatchStats_TimeToFirstContactFromFreezeEnd(t *testing.T) {
	rounds := []RoundData{
		{
			Number:        1,
			StartTick:     0,
			FreezeEndTick: 320, // 5s into round at 64 tps
			EndTick:       6400,
			Roster:        twoRoundRoster(),
		},
	}
	events := []GameEvent{
		// First hurt event with steamA as victim, 1.5s after freeze end.
		{
			Tick: 416, RoundNumber: 1, Type: "player_hurt",
			AttackerSteamID: "steamB", VictimSteamID: "steamA", Weapon: "AK-47",
			ExtraData: &PlayerHurtExtra{
				HealthDamage: 30,
				AttackerName: "PlayerB", AttackerTeam: "T",
				VictimName: "PlayerA", VictimTeam: "CT",
			},
		},
	}

	stats := ComputePlayerMatchStats(rounds, events, nil, nil, nil, "steamA", 64)
	if len(stats.Rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(stats.Rounds))
	}
	r1 := stats.Rounds[0]
	if r1.TimeToFirstContactSec == nil {
		t.Fatalf("expected non-nil TimeToFirstContactSec")
	}
	got := *r1.TimeToFirstContactSec
	want := float64(416-320) / 64.0 // 1.5s
	if !floatClose(got, want, 1e-6) {
		t.Errorf("TimeToFirstContactSec = %v, want %v", got, want)
	}
	if !floatClose(stats.Timings.AvgTimeToFirstContactSecs, want, 1e-6) {
		t.Errorf("AvgTimeToFirstContactSecs = %v, want %v", stats.Timings.AvgTimeToFirstContactSecs, want)
	}
}

func TestComputePlayerMatchStats_TimeToFirstContactNilWhenNoEvent(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, StartTick: 0, FreezeEndTick: 320, EndTick: 6400, Roster: twoRoundRoster()},
	}
	stats := ComputePlayerMatchStats(rounds, nil, nil, nil, nil, "steamA", 64)
	if len(stats.Rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(stats.Rounds))
	}
	if stats.Rounds[0].TimeToFirstContactSec != nil {
		t.Errorf("TimeToFirstContactSec = %v, want nil", *stats.Rounds[0].TimeToFirstContactSec)
	}
}

func TestComputePlayerMatchStats_DistanceAndAvgSpeedFromSamples(t *testing.T) {
	// 64 tps → sampleHz = 16. Three samples 4 ticks apart, moving 100 units
	// along +X each step. Step speed = 100 / (1/16) = 1600 u/s — beyond the
	// teleport clamp, so all deltas are dropped. Use a step size that stays
	// within the run cap: 8 units per 4-tick step → 128 u/s, kept.
	rounds := []RoundData{
		{Number: 1, StartTick: 0, FreezeEndTick: 0, EndTick: 1000, Roster: twoRoundRoster()},
	}
	samples := []PlayerTickSample{
		{Tick: 0, X: 0, Y: 0, IsAlive: true},
		{Tick: 4, X: 8, Y: 0, IsAlive: true},
		{Tick: 8, X: 16, Y: 0, IsAlive: true},
	}
	stats := ComputePlayerMatchStats(rounds, nil, nil, samples, nil, "steamA", 64)

	if stats.Movement.DistanceUnits != 16 {
		t.Errorf("DistanceUnits = %d, want 16", stats.Movement.DistanceUnits)
	}
	wantAvg := 16.0 / (3.0 / 16.0)
	if !floatClose(stats.Movement.AvgSpeedUps, wantAvg, 1e-3) {
		t.Errorf("AvgSpeedUps = %v, want %v", stats.Movement.AvgSpeedUps, wantAvg)
	}
	if len(stats.Rounds) != 1 {
		t.Fatalf("expected 1 round detail, got %d", len(stats.Rounds))
	}
	if stats.Rounds[0].DistanceUnits != 16 {
		t.Errorf("round 1 DistanceUnits = %d, want 16", stats.Rounds[0].DistanceUnits)
	}
	wantAlive := 3.0 / 16.0
	if !floatClose(stats.Rounds[0].AliveDurationSecs, wantAlive, 1e-6) {
		t.Errorf("round 1 AliveDurationSecs = %v, want %v", stats.Rounds[0].AliveDurationSecs, wantAlive)
	}
}

func TestComputePlayerMatchStats_MaxSpeedClampDropsTeleports(t *testing.T) {
	// 100u in 4 ticks at 64tps → 1600 u/s, far above the 260 u/s clamp.
	// The aggregator must drop the delta entirely so the teleport doesn't
	// pollute distance or max-speed.
	rounds := []RoundData{
		{Number: 1, StartTick: 0, FreezeEndTick: 0, EndTick: 1000, Roster: twoRoundRoster()},
	}
	samples := []PlayerTickSample{
		{Tick: 0, X: 0, Y: 0, IsAlive: true},
		{Tick: 4, X: 100, Y: 0, IsAlive: true},  // 1600 u/s, dropped
		{Tick: 8, X: 110, Y: 0, IsAlive: true},  // 160 u/s, kept
		{Tick: 12, X: 120, Y: 0, IsAlive: true}, // 160 u/s, kept
	}
	stats := ComputePlayerMatchStats(rounds, nil, nil, samples, nil, "steamA", 64)
	if stats.Movement.MaxSpeedUps > movementMaxSpeed {
		t.Errorf("MaxSpeedUps = %v, want ≤ %v (clamp)", stats.Movement.MaxSpeedUps, movementMaxSpeed)
	}
	// Distance should reflect only the kept samples (10 + 10 = 20).
	if stats.Movement.DistanceUnits != 20 {
		t.Errorf("DistanceUnits = %d, want 20 (teleport dropped)", stats.Movement.DistanceUnits)
	}
}

func TestComputePlayerMatchStats_StrafePercentDetectsLateralMovement(t *testing.T) {
	// Yaw = 0° (forward = +X). Player moves +Y → 90° off forward → strafe.
	// Step size 8u/4ticks = 128 u/s — above strafe minimum (60), kept.
	rounds := []RoundData{
		{Number: 1, StartTick: 0, FreezeEndTick: 0, EndTick: 1000, Roster: twoRoundRoster()},
	}
	samples := []PlayerTickSample{
		{Tick: 0, X: 0, Y: 0, Yaw: 0, IsAlive: true},
		{Tick: 4, X: 0, Y: 8, Yaw: 0, IsAlive: true},
		{Tick: 8, X: 0, Y: 16, Yaw: 0, IsAlive: true},
		{Tick: 12, X: 0, Y: 24, Yaw: 0, IsAlive: true},
	}
	stats := ComputePlayerMatchStats(rounds, nil, nil, samples, nil, "steamA", 64)
	if stats.Movement.StrafePercent < 99 || stats.Movement.StrafePercent > 100.001 {
		t.Errorf("StrafePercent = %v, want ~100", stats.Movement.StrafePercent)
	}
}

func TestComputePlayerMatchStats_StrafePercentForwardMovementIsZero(t *testing.T) {
	// Yaw = 0°, moving +X (forward) → 0° angle → not strafe.
	rounds := []RoundData{
		{Number: 1, StartTick: 0, FreezeEndTick: 0, EndTick: 1000, Roster: twoRoundRoster()},
	}
	samples := []PlayerTickSample{
		{Tick: 0, X: 0, Y: 0, Yaw: 0, IsAlive: true},
		{Tick: 4, X: 8, Y: 0, Yaw: 0, IsAlive: true},
		{Tick: 8, X: 16, Y: 0, Yaw: 0, IsAlive: true},
	}
	stats := ComputePlayerMatchStats(rounds, nil, nil, samples, nil, "steamA", 64)
	if stats.Movement.StrafePercent != 0 {
		t.Errorf("StrafePercent = %v, want 0 for pure forward motion", stats.Movement.StrafePercent)
	}
}

func TestComputePlayerMatchStats_SpeedBuckets(t *testing.T) {
	// Sample interval = 4 ticks at 64 tps → dt = 1/16 s.
	// Speed in u/s = dist * 16. Place each transition in a different bucket.
	// Stationary: dist 0.5 → speed 8.
	// Walking: dist 4 → speed 64.
	// Running: dist 12.5 → speed 200.
	rounds := []RoundData{
		{Number: 1, StartTick: 0, FreezeEndTick: 0, EndTick: 1000, Roster: twoRoundRoster()},
	}
	samples := []PlayerTickSample{
		{Tick: 0, X: 0, Y: 0, IsAlive: true},
		{Tick: 4, X: 0.5, Y: 0, IsAlive: true}, // stationary
		{Tick: 8, X: 4.5, Y: 0, IsAlive: true}, // walking
		{Tick: 12, X: 17, Y: 0, IsAlive: true}, // running
	}
	stats := ComputePlayerMatchStats(rounds, nil, nil, samples, nil, "steamA", 64)

	got := []float64{stats.Movement.StationaryRatio, stats.Movement.WalkingRatio, stats.Movement.RunningRatio}
	want := []float64{1.0 / 3.0, 1.0 / 3.0, 1.0 / 3.0}
	for i, g := range got {
		if !floatClose(g, want[i], 1e-6) {
			t.Errorf("bucket[%d] = %v, want %v", i, g, want[i])
		}
	}
}

func TestComputePlayerMatchStats_RoundBoundaryDoesNotCountInterRoundJump(t *testing.T) {
	// Two rounds with a teleport-sized gap between R1 last sample and R2
	// first sample. Distance must not include the cross-round jump.
	rounds := []RoundData{
		{Number: 1, StartTick: 0, FreezeEndTick: 0, EndTick: 100, Roster: twoRoundRoster()},
		{Number: 2, StartTick: 200, FreezeEndTick: 200, EndTick: 1000, Roster: twoRoundRoster()},
	}
	samples := []PlayerTickSample{
		{Tick: 0, X: 0, Y: 0, IsAlive: true},
		{Tick: 4, X: 10, Y: 0, IsAlive: true},
		// R1 EndTick = 100; first sample of R2 = 200.
		{Tick: 200, X: 9000, Y: 0, IsAlive: true}, // huge jump (different spawn)
		{Tick: 204, X: 9010, Y: 0, IsAlive: true},
	}
	stats := ComputePlayerMatchStats(rounds, nil, nil, samples, nil, "steamA", 64)

	// Only same-round consecutive deltas count: R1 +10 and R2 +10 = 20.
	if stats.Movement.DistanceUnits != 20 {
		t.Errorf("DistanceUnits = %d, want 20 (cross-round jump excluded)", stats.Movement.DistanceUnits)
	}
}

func TestComputePlayerMatchStats_TimeOnSiteProxy(t *testing.T) {
	// Three alive samples all inside A site; sampleHz = 16 → 3/16 s on site.
	rounds := []RoundData{
		{Number: 1, StartTick: 0, FreezeEndTick: 0, EndTick: 1000, Roster: twoRoundRoster()},
	}
	samples := []PlayerTickSample{
		{Tick: 0, X: 100, Y: 100, IsAlive: true},
		{Tick: 4, X: 110, Y: 100, IsAlive: true},
		{Tick: 8, X: 120, Y: 100, IsAlive: true},
	}
	bombsites := []BombsiteCentroid{{Site: "A", X: 100, Y: 100}}

	stats := ComputePlayerMatchStats(rounds, nil, nil, samples, bombsites, "steamA", 64)
	want := 3.0 / 16.0
	if !floatClose(stats.Timings.TimeOnSiteASecs, want, 1e-6) {
		t.Errorf("TimeOnSiteASecs = %v, want %v", stats.Timings.TimeOnSiteASecs, want)
	}
	if stats.Timings.TimeOnSiteBSecs != 0 {
		t.Errorf("TimeOnSiteBSecs = %v, want 0 (no B centroid)", stats.Timings.TimeOnSiteBSecs)
	}
}

func TestBombsiteCentroidsFromEvents(t *testing.T) {
	events := []GameEvent{
		{Tick: 1, Type: "bomb_plant", X: 100, Y: 200, ExtraData: &BombPlantExtra{Site: "A"}},
		{Tick: 2, Type: "bomb_plant", X: 200, Y: 300, ExtraData: &BombPlantExtra{Site: "A"}},
		{Tick: 3, Type: "bomb_plant", X: -500, Y: 400, ExtraData: &BombPlantExtra{Site: "B"}},
		{Tick: 4, Type: "kill", X: 9999, Y: 9999}, // ignored
	}
	got := BombsiteCentroidsFromEvents(events)
	if len(got) != 2 {
		t.Fatalf("len(centroids) = %d, want 2", len(got))
	}
	if got[0].Site != "A" || got[0].X != 150 || got[0].Y != 250 {
		t.Errorf("A centroid = %+v, want {A 150 250}", got[0])
	}
	if got[1].Site != "B" || got[1].X != -500 || got[1].Y != 400 {
		t.Errorf("B centroid = %+v, want {B -500 400}", got[1])
	}
}

// Phase 3 — utility / hit-group / polygon tests -----------------------------

func TestComputePlayerMatchStats_UtilityCountsByGrenadeType(t *testing.T) {
	rounds := []RoundData{{Number: 1, Roster: twoRoundRoster()}}
	mk := func(tick int, weapon string) GameEvent {
		return GameEvent{
			Tick: tick, RoundNumber: 1, Type: "grenade_throw",
			AttackerSteamID: "steamA", Weapon: weapon,
			ExtraData: &GrenadeThrowExtra{},
		}
	}
	events := []GameEvent{
		mk(10, "Smoke Grenade"),
		mk(11, "smokegrenade"),
		mk(12, "Flashbang"),
		mk(13, "HE Grenade"),
		mk(14, "Molotov"),
		mk(15, "Incendiary Grenade"),
		mk(16, "Decoy Grenade"),
		// Throws by another player must not count.
		{
			Tick: 17, RoundNumber: 1, Type: "grenade_throw",
			AttackerSteamID: "steamB", Weapon: "Smoke Grenade",
			ExtraData: &GrenadeThrowExtra{},
		},
	}

	stats := ComputePlayerMatchStats(rounds, events, nil, nil, nil, "steamA", 64)
	if stats.Utility.SmokesThrown != 2 {
		t.Errorf("SmokesThrown = %d, want 2", stats.Utility.SmokesThrown)
	}
	if stats.Utility.FlashesThrown != 1 {
		t.Errorf("FlashesThrown = %d, want 1", stats.Utility.FlashesThrown)
	}
	if stats.Utility.HEsThrown != 1 {
		t.Errorf("HEsThrown = %d, want 1", stats.Utility.HEsThrown)
	}
	if stats.Utility.MolotovsThrown != 2 {
		t.Errorf("MolotovsThrown = %d, want 2 (molotov+incgrenade)", stats.Utility.MolotovsThrown)
	}
	if stats.Utility.DecoysThrown != 1 {
		t.Errorf("DecoysThrown = %d, want 1", stats.Utility.DecoysThrown)
	}
}

func TestComputePlayerMatchStats_BlindTimeAndFlashAssists(t *testing.T) {
	rounds := []RoundData{{Number: 1, Roster: twoRoundRoster()}}
	events := []GameEvent{
		// Enemy flashed for 2.5s by us.
		{
			Tick: 10, RoundNumber: 1, Type: "player_flashed",
			AttackerSteamID: "steamA", VictimSteamID: "steamB",
			ExtraData: &PlayerFlashedExtra{
				DurationSecs: 2.5,
				AttackerName: "PlayerA", AttackerTeam: "CT",
				VictimName: "PlayerB", VictimTeam: "T",
			},
		},
		// Self-flash — must not count.
		{
			Tick: 12, RoundNumber: 1, Type: "player_flashed",
			AttackerSteamID: "steamA", VictimSteamID: "steamA",
			ExtraData: &PlayerFlashedExtra{
				DurationSecs: 1.0,
				AttackerName: "PlayerA", AttackerTeam: "CT",
				VictimName: "PlayerA", VictimTeam: "CT",
			},
		},
		// Kill credited as a flash assist.
		makeKillEvent(20, 1, "steamMate", "steamB", "ak47", false, &KillExtra{
			AttackerName: "PlayerMate", AttackerTeam: "CT",
			VictimName: "PlayerB", VictimTeam: "T",
			AssisterSteamID: "steamA",
			FlashAssist:     true,
		}),
		// Our own kill where FlashAssist is set — also counts (we are
		// attacker, the parser flag travels on KillExtra regardless).
		makeKillEvent(25, 1, "steamA", "steamC", "ak47", false, &KillExtra{
			AttackerName: "PlayerA", AttackerTeam: "CT",
			VictimName: "PlayerC", VictimTeam: "T",
			FlashAssist: true,
		}),
	}

	stats := ComputePlayerMatchStats(rounds, events, nil, nil, nil, "steamA", 64)
	if stats.Utility.EnemiesFlashed != 1 {
		t.Errorf("EnemiesFlashed = %d, want 1", stats.Utility.EnemiesFlashed)
	}
	if !floatClose(stats.Utility.BlindTimeInflictedSecs, 2.5, 1e-6) {
		t.Errorf("BlindTimeInflictedSecs = %v, want 2.5", stats.Utility.BlindTimeInflictedSecs)
	}
	if stats.Utility.FlashAssists != 1 {
		t.Errorf("FlashAssists = %d, want 1 (only our own kill counts)", stats.Utility.FlashAssists)
	}
}

func TestComputePlayerMatchStats_HitGroupBreakdown(t *testing.T) {
	rounds := []RoundData{{Number: 1, Roster: twoRoundRoster()}}
	mk := func(tick, dmg, hg int, victimTeam string) GameEvent {
		return GameEvent{
			Tick: tick, RoundNumber: 1, Type: "player_hurt",
			AttackerSteamID: "steamA", VictimSteamID: "steamB", Weapon: "AK-47",
			ExtraData: &PlayerHurtExtra{
				HealthDamage: dmg, HitGroup: hg,
				AttackerName: "PlayerA", AttackerTeam: "CT",
				VictimName: "PlayerB", VictimTeam: victimTeam,
			},
		}
	}
	events := []GameEvent{
		mk(1, 50, 1, "T"),  // head
		mk(2, 20, 2, "T"),  // chest
		mk(3, 30, 2, "T"),  // chest
		mk(4, 40, 1, "CT"), // friendly fire — must NOT count
	}
	stats := ComputePlayerMatchStats(rounds, events, nil, nil, nil, "steamA", 64)

	if len(stats.HitGroups) != 2 {
		t.Fatalf("len(HitGroups) = %d, want 2", len(stats.HitGroups))
	}
	// Sorted by damage desc → chest (50) first, head (50) second by tie-break
	// on hit-group asc → head (1) before chest (2).
	if stats.HitGroups[0].HitGroup != 1 || stats.HitGroups[0].Damage != 50 {
		t.Errorf("HitGroups[0] = %+v, want head/50", stats.HitGroups[0])
	}
	if stats.HitGroups[0].Label != "Head" {
		t.Errorf("HitGroups[0].Label = %q, want Head", stats.HitGroups[0].Label)
	}
	if stats.HitGroups[1].HitGroup != 2 || stats.HitGroups[1].Damage != 50 {
		t.Errorf("HitGroups[1] = %+v, want chest/50", stats.HitGroups[1])
	}
	if stats.HitGroups[1].Hits != 2 {
		t.Errorf("HitGroups[1].Hits = %d, want 2", stats.HitGroups[1].Hits)
	}
}

func TestComputePlayerMatchStats_BombsitePolygonOverridesCircle(t *testing.T) {
	// Sample at (0, 0) is OUTSIDE the polygon (which is the rect 100..200 x
	// 100..200) but INSIDE the bounding circle centered at (150, 150) with
	// radius 800. The polygon must take priority and the sample must be
	// excluded from time-on-site.
	rounds := []RoundData{
		{Number: 1, StartTick: 0, FreezeEndTick: 0, EndTick: 1000, Roster: twoRoundRoster()},
	}
	samples := []PlayerTickSample{
		{Tick: 0, X: 0, Y: 0, IsAlive: true},
		{Tick: 4, X: 0, Y: 0, IsAlive: true},
	}
	bombsites := []BombsiteCentroid{{
		Site: "A", X: 150, Y: 150,
		MinX: 100, MaxX: 200,
		MinY: 100, MaxY: 200,
	}}
	stats := ComputePlayerMatchStats(rounds, nil, nil, samples, bombsites, "steamA", 64)
	if stats.Timings.TimeOnSiteASecs != 0 {
		t.Errorf("TimeOnSiteASecs = %v, want 0 (polygon excludes (0,0))", stats.Timings.TimeOnSiteASecs)
	}

	// Now move the sample inside the polygon — should count.
	samples = []PlayerTickSample{
		{Tick: 0, X: 150, Y: 150, IsAlive: true},
		{Tick: 4, X: 160, Y: 160, IsAlive: true},
	}
	stats = ComputePlayerMatchStats(rounds, nil, nil, samples, bombsites, "steamA", 64)
	if stats.Timings.TimeOnSiteASecs == 0 {
		t.Errorf("TimeOnSiteASecs = 0, want > 0 (polygon includes inside points)")
	}
}

func TestBombsitePolygonsForMap(t *testing.T) {
	got := BombsitePolygonsForMap("de_dust2")
	if len(got) != 2 {
		t.Fatalf("len(BombsitePolygonsForMap(de_dust2)) = %d, want 2", len(got))
	}
	if got[0].Site != "A" {
		t.Errorf("[0].Site = %q, want A", got[0].Site)
	}
	if got[1].Site != "B" {
		t.Errorf("[1].Site = %q, want B", got[1].Site)
	}
	if BombsitePolygonsForMap("unknown_map") != nil {
		t.Error("expected nil for unknown map")
	}
}

func TestComputePlayerMatchStats_AvgAliveDuration(t *testing.T) {
	rounds := []RoundData{
		{Number: 1, StartTick: 0, FreezeEndTick: 0, EndTick: 100, Roster: twoRoundRoster()},
		{Number: 2, StartTick: 200, FreezeEndTick: 200, EndTick: 1000, Roster: twoRoundRoster()},
	}
	samples := []PlayerTickSample{
		{Tick: 0, X: 0, Y: 0, IsAlive: true},
		{Tick: 4, X: 0, Y: 0, IsAlive: true}, // round 1: 2 alive samples = 2/16
		{Tick: 200, X: 0, Y: 0, IsAlive: true},
		{Tick: 204, X: 0, Y: 0, IsAlive: true},
		{Tick: 208, X: 0, Y: 0, IsAlive: false}, // dead — does not count
	}
	stats := ComputePlayerMatchStats(rounds, nil, nil, samples, nil, "steamA", 64)
	want := ((2.0 / 16.0) + (2.0 / 16.0)) / 2.0
	if !floatClose(stats.Timings.AvgAliveDurationSecs, want, 1e-6) {
		t.Errorf("AvgAliveDurationSecs = %v, want %v", stats.Timings.AvgAliveDurationSecs, want)
	}
}
