package analysis_test

import (
	"strconv"
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/analysis"
)

// mkTickIdx is a small helper so the per-rule tests can build a synthetic
// PerPlayerTickIndex without round-tripping through the fixture loader. The
// rule files take a *PerPlayerTickIndex; we get one by constructing
// AnalysisTick rows and handing them to BuildTickIndex.
func mkTickIdx(rows ...demo.AnalysisTick) analysis.PerPlayerTickIndex {
	return analysis.BuildTickIndex(rows)
}

func u64(s string) uint64 { v, _ := strconv.ParseUint(s, 10, 64); return v }

func TestRun_TimeToFire_FlagsSlowReaction(t *testing.T) {
	// Attacker (steam 100) drifts past the victim (steam 200) for ~32 ticks
	// before firing. At 64 tickrate that's 500 ms, well above the 300 ms
	// reaction threshold, so the rule should emit a slow_reaction mistake.
	rounds := []demo.RoundData{{
		Number:    1,
		StartTick: 0,
		EndTick:   1000,
		Roster: []demo.RoundParticipant{
			{SteamID: "100", TeamSide: "T", Inventory: "AK-47"},
			{SteamID: "200", TeamSide: "CT", Inventory: "M4A1"},
		},
	}}
	// At tick 0 the attacker faces away (yaw 180°) so the victim sits behind.
	// At ticks 16, 32, 48 they pivot to yaw 0 — victim entered FOV at tick
	// 16. The fire lands at tick 80 (kill event). Reaction time = 64 ticks
	// = 1000 ms — well above the 300 ms threshold.
	idxRows := []demo.AnalysisTick{
		{Tick: 0, SteamID: u64("100"), X: 0, Y: 0, Yaw: 180, IsAlive: true},
		{Tick: 16, SteamID: u64("100"), X: 0, Y: 0, Yaw: 0, IsAlive: true},
		{Tick: 32, SteamID: u64("100"), X: 0, Y: 0, Yaw: 0, IsAlive: true},
		{Tick: 48, SteamID: u64("100"), X: 0, Y: 0, Yaw: 0, IsAlive: true},
		{Tick: 64, SteamID: u64("100"), X: 0, Y: 0, Yaw: 0, IsAlive: true},
		{Tick: 0, SteamID: u64("200"), X: 500, Y: 0, IsAlive: true},
		{Tick: 16, SteamID: u64("200"), X: 500, Y: 0, IsAlive: true},
		{Tick: 32, SteamID: u64("200"), X: 500, Y: 0, IsAlive: true},
		{Tick: 48, SteamID: u64("200"), X: 500, Y: 0, IsAlive: true},
		{Tick: 64, SteamID: u64("200"), X: 500, Y: 0, IsAlive: true},
	}
	events := []demo.GameEvent{{
		Tick:            80,
		RoundNumber:     1,
		Type:            "kill",
		AttackerSteamID: "100",
		VictimSteamID:   "200",
		Weapon:          "ak47",
		ExtraData:       &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"},
	}}
	result := &demo.ParseResult{
		Header:        demo.MatchHeader{TickRate: 64},
		Rounds:        rounds,
		Events:        events,
		AnalysisTicks: idxRows,
	}

	got, _, err := analysis.Run(result, nil, analysis.RunOpts{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, m := range got {
		if m.Kind == string(analysis.MistakeKindSlowReaction) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected slow_reaction mistake, got %+v", got)
	}
	_ = mkTickIdx
}

func TestRun_RepeatedDeathZones_FlagsThirdDeath(t *testing.T) {
	// Three deaths for steam 100 within the same 200u grid cell — the third
	// death (and any further) should be flagged.
	rounds := []demo.RoundData{
		{Number: 1, EndTick: 1000, Roster: []demo.RoundParticipant{{SteamID: "100", TeamSide: "T"}}},
		{Number: 2, EndTick: 2000, Roster: []demo.RoundParticipant{{SteamID: "100", TeamSide: "T"}}},
		{Number: 3, EndTick: 3000, Roster: []demo.RoundParticipant{{SteamID: "100", TeamSide: "T"}}},
	}
	mkKill := func(tick, round int, x, y float64) demo.GameEvent {
		return demo.GameEvent{
			Tick: tick, RoundNumber: round, Type: "kill",
			AttackerSteamID: "999", VictimSteamID: "100",
			X: x, Y: y, Weapon: "ak47",
			ExtraData: &demo.KillExtra{AttackerTeam: "CT", VictimTeam: "T"},
		}
	}
	events := []demo.GameEvent{
		mkKill(100, 1, 50, 50),
		mkKill(1100, 2, 90, 90),   // same 200u cell as (50, 50)
		mkKill(2100, 3, 150, 150), // still same 200u cell
	}
	result := &demo.ParseResult{
		Header: demo.MatchHeader{TickRate: 64},
		Rounds: rounds,
		Events: events,
	}
	got, _, err := analysis.Run(result, nil, analysis.RunOpts{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	zones := 0
	for _, m := range got {
		if m.Kind == string(analysis.MistakeKindRepeatedDeathZone) {
			zones++
		}
	}
	if zones != 1 {
		t.Errorf("expected 1 repeated_death_zone mistake (the 3rd death), got %d (%+v)", zones, got)
	}
}

func TestRun_IsolatedPeek_FlagsLoneDeath(t *testing.T) {
	rounds := []demo.RoundData{{
		Number: 1, EndTick: 1000,
		Roster: []demo.RoundParticipant{
			{SteamID: "100", TeamSide: "T", Inventory: "AK-47"},
			{SteamID: "200", TeamSide: "T", Inventory: "AK-47"},
			{SteamID: "300", TeamSide: "CT", Inventory: "M4A1"},
		},
	}}
	// Victim 100 dies at (0, 0). Teammate 200 is sitting at (5000, 5000) —
	// far outside the 600u radius. CT 300 is the killer.
	idxRows := []demo.AnalysisTick{
		{Tick: 90, SteamID: u64("100"), X: 0, Y: 0, IsAlive: true},
		{Tick: 90, SteamID: u64("200"), X: 5000, Y: 5000, IsAlive: true},
		{Tick: 90, SteamID: u64("300"), X: 100, Y: 100, IsAlive: true},
	}
	events := []demo.GameEvent{{
		Tick: 100, RoundNumber: 1, Type: "kill",
		AttackerSteamID: "300", VictimSteamID: "100",
		X: 0, Y: 0, Weapon: "m4a1",
		ExtraData: &demo.KillExtra{AttackerTeam: "CT", VictimTeam: "T"},
	}}
	result := &demo.ParseResult{
		Header:        demo.MatchHeader{TickRate: 64},
		Rounds:        rounds,
		Events:        events,
		AnalysisTicks: idxRows,
	}
	got, _, err := analysis.Run(result, nil, analysis.RunOpts{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	found := false
	for _, m := range got {
		if m.Kind == string(analysis.MistakeKindIsolatedPeek) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected isolated_peek mistake, got %+v", got)
	}

	// Now place the teammate within 600u — the rule should NOT fire.
	idxRows[1].X = 200
	idxRows[1].Y = 200
	result.AnalysisTicks = idxRows
	got, _, err = analysis.Run(result, nil, analysis.RunOpts{})
	if err != nil {
		t.Fatalf("Run (supported): %v", err)
	}
	for _, m := range got {
		if m.Kind == string(analysis.MistakeKindIsolatedPeek) {
			t.Errorf("did not expect isolated_peek when teammate is in range, got %+v", m)
		}
	}
}

func TestRun_EcoMisbuy_FlagsEcoVsForce(t *testing.T) {
	// Round 5: T side full eco (no money), CT side force (~2.5k each). T
	// players should be flagged.
	rounds := []demo.RoundData{{
		Number: 5, FreezeEndTick: 100, EndTick: 5000,
		Roster: []demo.RoundParticipant{
			{SteamID: "t1", TeamSide: "T", Inventory: ""},
			{SteamID: "t2", TeamSide: "T", Inventory: ""},
			{SteamID: "ct1", TeamSide: "CT", Inventory: "Famas,Kevlar"},
			{SteamID: "ct2", TeamSide: "CT", Inventory: "Famas,Kevlar"},
		},
	}}
	result := &demo.ParseResult{
		Header: demo.MatchHeader{TickRate: 64},
		Rounds: rounds,
	}
	got, _, err := analysis.Run(result, nil, analysis.RunOpts{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	flagged := map[string]bool{}
	for _, m := range got {
		if m.Kind == string(analysis.MistakeKindEcoMisbuy) {
			flagged[m.SteamID] = true
		}
	}
	if !flagged["t1"] || !flagged["t2"] {
		t.Errorf("expected eco_misbuy for both T players, got %v", flagged)
	}
	if flagged["ct1"] || flagged["ct2"] {
		t.Errorf("CT side should not be flagged, got %v", flagged)
	}
}
