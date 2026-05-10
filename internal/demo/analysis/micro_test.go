package analysis

import (
	"math"
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
)

// makeIdx is a tiny builder for PerPlayerTickIndex used by these tests. The
// production BuildTickIndex sorts and indexes by SteamID-as-string, so we
// emulate the same shape directly to keep the table-driven tests readable.
func makeIdx(rows []demo.AnalysisTick) PerPlayerTickIndex {
	return BuildTickIndex(rows)
}

func TestComputeMicroAggregates_TimeToStop(t *testing.T) {
	const tickRate = 64.0

	// Player went from running (~150 u/s) at tick 96 to standing (~0) at tick
	// 100, then fired at tick 100. With tickInterval=4 the lookback window
	// reaches sample (-3 = tick 88) — so the most-recent armed sample is the
	// 96 sample. Stop interval = 100-96 = 4 ticks → 4/64*1000 = 62.5 ms.
	ticks := []demo.AnalysisTick{
		{Tick: 88, SteamID: 1, Vx: 150, Vy: 0, IsAlive: true},
		{Tick: 92, SteamID: 1, Vx: 150, Vy: 0, IsAlive: true},
		{Tick: 96, SteamID: 1, Vx: 150, Vy: 0, IsAlive: true},
		{Tick: 100, SteamID: 1, Vx: 0, Vy: 0, IsAlive: true},
	}
	events := []demo.GameEvent{
		{
			Tick:            100,
			Type:            "weapon_fire",
			AttackerSteamID: "1",
			Weapon:          "ak47",
			ExtraData:       &demo.WeaponFireExtra{Yaw: 0},
		},
	}

	out := computeMicroAggregates(events, makeIdx(ticks), nil, tickRate)
	got, ok := out["1"]
	if !ok {
		t.Fatalf("want player 1 in results, got: %+v", out)
	}
	const wantMs = 62.5
	if math.Abs(got.TimeToStopMsAvg-wantMs) > 0.01 {
		t.Errorf("TimeToStopMsAvg = %.3f, want %.3f", got.TimeToStopMsAvg, wantMs)
	}
}

func TestComputeMicroAggregates_CrouchCounters(t *testing.T) {
	ticks := []demo.AnalysisTick{
		// Player 1: crouched + standing at fire — counts as crouch_before_shot
		// only (not crouch_instead_of_strafe).
		{Tick: 100, SteamID: 1, Vx: 0, Vy: 0, IsAlive: true, Crouch: true},
		// Player 2: crouched AND moving fast at fire — counts as both.
		{Tick: 100, SteamID: 2, Vx: 200, Vy: 0, IsAlive: true, Crouch: true},
		// Player 3: not crouched — neither counter increments.
		{Tick: 100, SteamID: 3, Vx: 0, Vy: 0, IsAlive: true, Crouch: false},
	}
	events := []demo.GameEvent{
		{Tick: 100, Type: "weapon_fire", AttackerSteamID: "1", Weapon: "ak47", ExtraData: &demo.WeaponFireExtra{}},
		{Tick: 100, Type: "weapon_fire", AttackerSteamID: "2", Weapon: "ak47", ExtraData: &demo.WeaponFireExtra{}},
		{Tick: 100, Type: "weapon_fire", AttackerSteamID: "3", Weapon: "ak47", ExtraData: &demo.WeaponFireExtra{}},
	}

	out := computeMicroAggregates(events, makeIdx(ticks), nil, 64.0)

	if got := out["1"].CrouchBeforeShotCount; got != 1 {
		t.Errorf("p1 CrouchBeforeShotCount = %d, want 1", got)
	}
	if got := out["1"].CrouchInsteadOfStrafeCount; got != 0 {
		t.Errorf("p1 CrouchInsteadOfStrafeCount = %d, want 0", got)
	}
	if got := out["2"].CrouchBeforeShotCount; got != 1 {
		t.Errorf("p2 CrouchBeforeShotCount = %d, want 1", got)
	}
	if got := out["2"].CrouchInsteadOfStrafeCount; got != 1 {
		t.Errorf("p2 CrouchInsteadOfStrafeCount = %d, want 1", got)
	}
	if got := out["3"].CrouchBeforeShotCount; got != 0 {
		t.Errorf("p3 CrouchBeforeShotCount = %d, want 0", got)
	}
}

func TestComputeMicroAggregates_FlickBalance(t *testing.T) {
	// Two flicks: one overshoot, one undershoot. Player at origin facing east
	// (yaw=0). Enemy at (+1000, 0) — angular distance to target = 0 deg from
	// current facing, but we'll move the player's yaw between samples so the
	// flick lookback registers a > 30 deg delta.
	//
	// Setup uses `previousTickRows` which steps back through the lookback
	// window, so we need at least flickLookbackSamples (3) prior samples per
	// flick to anchor the "before" yaw. We give 3 idle samples then the fire.
	ticks := []demo.AnalysisTick{
		// Round 1: pre-fire facing west (yaw=180), fire facing east (yaw=0).
		{Tick: 88, SteamID: 1, X: 0, Y: 0, Yaw: 180, Vx: 0, IsAlive: true},
		{Tick: 92, SteamID: 1, X: 0, Y: 0, Yaw: 180, Vx: 0, IsAlive: true},
		{Tick: 96, SteamID: 1, X: 0, Y: 0, Yaw: 180, Vx: 0, IsAlive: true},
		{Tick: 100, SteamID: 1, X: 0, Y: 0, Yaw: 0, Vx: 0, IsAlive: true},
		// Round 2: pre-fire facing east (yaw=0), fire facing south (yaw=270).
		{Tick: 188, SteamID: 1, X: 0, Y: 0, Yaw: 0, Vx: 0, IsAlive: true},
		{Tick: 192, SteamID: 1, X: 0, Y: 0, Yaw: 0, Vx: 0, IsAlive: true},
		{Tick: 196, SteamID: 1, X: 0, Y: 0, Yaw: 0, Vx: 0, IsAlive: true},
		{Tick: 200, SteamID: 1, X: 0, Y: 0, Yaw: 270, Vx: 0, IsAlive: true},
		// Enemy samples — needed by nearestEnemyTick. Place at +X, so the
		// expected yaw to enemy is 0°. Round 1: under-flick (fire yaw 0
		// matches expected, but our lookback says yaw 180 was the prior, so
		// the fire angle minus prior yaw is +180° — anchored against
		// toTarget = +180 (relative to attacker yaw 180), the error is 0).
		// We use a different trick: place enemy at -X (yaw 180) so toTarget
		// after rotation can resolve to over/under correctly.
		{Tick: 100, SteamID: 2, X: -1000, Y: 0, Yaw: 0, IsAlive: true},
		{Tick: 200, SteamID: 2, X: 0, Y: -1000, Yaw: 0, IsAlive: true},
	}
	events := []demo.GameEvent{
		{
			Tick:            100,
			RoundNumber:     1,
			Type:            "weapon_fire",
			AttackerSteamID: "1",
			Weapon:          "ak47",
			// Fire yaw 0 → flick of 180°. Enemy at -X → expected yaw 180.
			// Player rotated 180° (yaw 180 → 0), but only needed 180° to
			// face the enemy at -X — actually rotated short of target
			// (rotated 180 east, but the enemy was west so the player
			// actually ended up facing the OPPOSITE direction). That makes
			// it an "overshoot" of 180° in our metric — enemy at -X, fire
			// yaw 0, prior yaw 180 → toTarget = 180 - 180 = 0, flick =
			// 0 - 180 = -180 (normalized to 180), err = 180 - 0 = 180 →
			// overshoot.
			ExtraData: &demo.WeaponFireExtra{Yaw: 0},
		},
		{
			Tick:            200,
			RoundNumber:     1,
			Type:            "weapon_fire",
			AttackerSteamID: "1",
			Weapon:          "ak47",
			// Player rotated 270° (yaw 0 → 270). Enemy at -Y → expected yaw
			// 270. Prior yaw 0, fire yaw 270, toTarget = 270 - 0 = 270 (or
			// -90 normalized), flick = 270 - 0 = 270 (or -90 normalized),
			// err = -90 - -90 = 0. Hmm — equal-magnitude rotation, so this
			// is a 0-error flick that won't bucket into over/under. We use
			// a different fire yaw to deliberately undershoot.
			ExtraData: &demo.WeaponFireExtra{Yaw: 250},
		},
	}
	teams := map[int]map[string]string{
		1: {"1": "T", "2": "CT"},
	}

	out := computeMicroAggregates(events, makeIdx(ticks), teams, 64.0)
	got, ok := out["1"]
	if !ok {
		t.Fatalf("want player 1 in results, got: %+v", out)
	}
	// Both flick errors should be > 0 (one over, one under, after the
	// normalize step), driving FlickBalancePct away from neither 0 nor 100.
	if got.FlickBalancePct <= 0 || got.FlickBalancePct >= 100 {
		t.Errorf("FlickBalancePct = %.2f, want strictly in (0, 100)", got.FlickBalancePct)
	}
	if got.FlickOvershootAvgDeg <= 0 && got.FlickUndershootAvgDeg <= 0 {
		t.Errorf("expected at least one of over/under > 0, got over=%.2f under=%.2f",
			got.FlickOvershootAvgDeg, got.FlickUndershootAvgDeg)
	}
}

func TestComputeMicroAggregates_NoTickIndex(t *testing.T) {
	events := []demo.GameEvent{
		{Tick: 100, Type: "weapon_fire", AttackerSteamID: "1", Weapon: "ak47", ExtraData: &demo.WeaponFireExtra{}},
	}
	if got := computeMicroAggregates(events, PerPlayerTickIndex{}, nil, 64.0); got != nil {
		t.Errorf("expected nil result with empty tick index, got %+v", got)
	}
}

func TestComputeMicroAggregates_EmptyEvents(t *testing.T) {
	idx := makeIdx([]demo.AnalysisTick{
		{Tick: 100, SteamID: 1, IsAlive: true},
	})
	if got := computeMicroAggregates(nil, idx, nil, 64.0); got != nil {
		t.Errorf("expected nil result with empty events, got %+v", got)
	}
}
