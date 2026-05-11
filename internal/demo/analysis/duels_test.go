package analysis_test

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/analysis"
)

// TestDetectDuels_HitAnchored covers the happy path: A fires, hit-anchor
// resolves the target authoritatively (HitVictimSteamID), then A kills V.
func TestDetectDuels_HitAnchored(t *testing.T) {
	events := []demo.GameEvent{
		{
			Tick: 100, RoundNumber: 1, Type: "weapon_fire",
			AttackerSteamID: "100", Weapon: "ak47",
			ExtraData: &demo.WeaponFireExtra{Yaw: 0, HitVictimSteamID: "200"},
		},
		{
			Tick: 100, RoundNumber: 1, Type: "player_hurt",
			AttackerSteamID: "100", VictimSteamID: "200",
			ExtraData: &demo.PlayerHurtExtra{AttackerTeam: "T", VictimTeam: "CT"},
		},
		{
			Tick: 110, RoundNumber: 1, Type: "kill",
			AttackerSteamID: "100", VictimSteamID: "200", Weapon: "ak47",
			ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"},
		},
	}
	rounds := []demo.RoundData{{
		Number: 1,
		Roster: []demo.RoundParticipant{
			{SteamID: "100", TeamSide: "T"},
			{SteamID: "200", TeamSide: "CT"},
		},
	}}
	got := analysis.DetectDuels(events, analysis.PerPlayerTickIndex{}, rounds, 64)
	if len(got) != 1 {
		t.Fatalf("expected 1 duel, got %d (%+v)", len(got), got)
	}
	d := got[0]
	if d.AttackerSteam != "100" || d.VictimSteam != "200" {
		t.Errorf("attacker/victim mismatch: %+v", d)
	}
	if d.Outcome != analysis.DuelOutcomeWon {
		t.Errorf("Outcome = %q, want won", d.Outcome)
	}
	if d.EndReason != analysis.DuelEndReasonKill {
		t.Errorf("EndReason = %q, want kill", d.EndReason)
	}
	if !d.HitConfirmed {
		t.Errorf("expected HitConfirmed = true")
	}
	if d.ShotCount != 1 {
		t.Errorf("ShotCount = %d, want 1", d.ShotCount)
	}
}

// TestDetectDuels_CleanKill: A kills V with no prior fires (AWP one-tap).
// Detector synthesises a clean_kill duel at the kill tick.
func TestDetectDuels_CleanKill(t *testing.T) {
	events := []demo.GameEvent{{
		Tick: 200, RoundNumber: 1, Type: "kill",
		AttackerSteamID: "100", VictimSteamID: "200", Weapon: "awp",
		ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"},
	}}
	rounds := []demo.RoundData{{
		Number: 1,
		Roster: []demo.RoundParticipant{
			{SteamID: "100", TeamSide: "T"},
			{SteamID: "200", TeamSide: "CT"},
		},
	}}
	got := analysis.DetectDuels(events, analysis.PerPlayerTickIndex{}, rounds, 64)
	if len(got) != 1 {
		t.Fatalf("expected 1 duel, got %d", len(got))
	}
	d := got[0]
	if d.EndReason != analysis.DuelEndReasonCleanKill {
		t.Errorf("EndReason = %q, want clean_kill", d.EndReason)
	}
	if d.Outcome != analysis.DuelOutcomeWon {
		t.Errorf("Outcome = %q, want won", d.Outcome)
	}
	if d.ShotCount != 0 {
		t.Errorf("ShotCount = %d, want 0", d.ShotCount)
	}
	if d.StartTick != 200 || d.EndTick != 200 {
		t.Errorf("expected synthetic single-tick window, got %d→%d", d.StartTick, d.EndTick)
	}
}

// TestDetectDuels_SpamIntoSmoke: a weapon_fire with no HitVictimSteamID
// and no plausible cone target (empty tick index) returns no duel.
func TestDetectDuels_SpamIntoSmoke(t *testing.T) {
	events := []demo.GameEvent{{
		Tick: 100, RoundNumber: 1, Type: "weapon_fire",
		AttackerSteamID: "100", Weapon: "ak47",
		ExtraData: &demo.WeaponFireExtra{Yaw: 0},
	}}
	rounds := []demo.RoundData{{
		Number: 1,
		Roster: []demo.RoundParticipant{
			{SteamID: "100", TeamSide: "T"},
			{SteamID: "200", TeamSide: "CT"},
		},
	}}
	got := analysis.DetectDuels(events, analysis.PerPlayerTickIndex{}, rounds, 64)
	if len(got) != 0 {
		t.Errorf("expected no duels for spam-into-smoke, got %d (%+v)", len(got), got)
	}
}

// TestDetectDuels_MutualLink: A and V fire at each other in overlapping
// windows. Detector emits two directed duels linked via MutualLocalID.
func TestDetectDuels_MutualLink(t *testing.T) {
	events := []demo.GameEvent{
		{
			Tick: 100, RoundNumber: 1, Type: "weapon_fire",
			AttackerSteamID: "100", Weapon: "ak47",
			ExtraData: &demo.WeaponFireExtra{Yaw: 0, HitVictimSteamID: "200"},
		},
		{
			Tick: 100, RoundNumber: 1, Type: "player_hurt",
			AttackerSteamID: "100", VictimSteamID: "200",
			ExtraData: &demo.PlayerHurtExtra{AttackerTeam: "T", VictimTeam: "CT"},
		},
		{
			Tick: 105, RoundNumber: 1, Type: "weapon_fire",
			AttackerSteamID: "200", Weapon: "m4a1",
			ExtraData: &demo.WeaponFireExtra{Yaw: 180, HitVictimSteamID: "100"},
		},
		{
			Tick: 105, RoundNumber: 1, Type: "player_hurt",
			AttackerSteamID: "200", VictimSteamID: "100",
			ExtraData: &demo.PlayerHurtExtra{AttackerTeam: "CT", VictimTeam: "T"},
		},
	}
	rounds := []demo.RoundData{{
		Number: 1,
		Roster: []demo.RoundParticipant{
			{SteamID: "100", TeamSide: "T"},
			{SteamID: "200", TeamSide: "CT"},
		},
	}}
	got := analysis.DetectDuels(events, analysis.PerPlayerTickIndex{}, rounds, 64)
	if len(got) != 2 {
		t.Fatalf("expected 2 directed duels (mutual), got %d (%+v)", len(got), got)
	}
	if got[0].MutualLocalID == -1 || got[1].MutualLocalID == -1 {
		t.Errorf("expected mutual link on both duels, got %+v", got)
	}
	if got[0].MutualLocalID != got[1].LocalID || got[1].MutualLocalID != got[0].LocalID {
		t.Errorf("mutual link not symmetric: %+v", got)
	}
}

// TestDetectDuels_TradeFlip: A→V kill resolved won; B kills A inside the
// trade window. Outcome flips to won_then_traded.
func TestDetectDuels_TradeFlip(t *testing.T) {
	events := []demo.GameEvent{
		{
			Tick: 100, RoundNumber: 1, Type: "kill",
			AttackerSteamID: "100", VictimSteamID: "200", Weapon: "awp",
			ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"},
		},
		{
			Tick: 150, RoundNumber: 1, Type: "kill",
			AttackerSteamID: "300", VictimSteamID: "100", Weapon: "m4a1",
			ExtraData: &demo.KillExtra{AttackerTeam: "CT", VictimTeam: "T"},
		},
	}
	rounds := []demo.RoundData{{
		Number: 1,
		Roster: []demo.RoundParticipant{
			{SteamID: "100", TeamSide: "T"},
			{SteamID: "200", TeamSide: "CT"},
			{SteamID: "300", TeamSide: "CT"},
		},
	}}
	got := analysis.DetectDuels(events, analysis.PerPlayerTickIndex{}, rounds, 64)
	// Expect 2 duels: 100→200 (won_then_traded) and 300→100 (won).
	if len(got) != 2 {
		t.Fatalf("expected 2 duels, got %d", len(got))
	}
	var firstDuel *analysis.Duel
	for i := range got {
		if got[i].AttackerSteam == "100" && got[i].VictimSteam == "200" {
			firstDuel = &got[i]
		}
	}
	if firstDuel == nil {
		t.Fatalf("first duel (100→200) missing from output: %+v", got)
	}
	if firstDuel.Outcome != analysis.DuelOutcomeWonTraded {
		t.Errorf("expected won_then_traded, got %q", firstDuel.Outcome)
	}
}

// TestDetectDuels_Teamkill_NoDuel: friendly-fire kills don't open a duel.
func TestDetectDuels_Teamkill_NoDuel(t *testing.T) {
	events := []demo.GameEvent{{
		Tick: 100, RoundNumber: 1, Type: "kill",
		AttackerSteamID: "100", VictimSteamID: "150", Weapon: "ak47",
		ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "T"},
	}}
	rounds := []demo.RoundData{{
		Number: 1,
		Roster: []demo.RoundParticipant{
			{SteamID: "100", TeamSide: "T"},
			{SteamID: "150", TeamSide: "T"},
		},
	}}
	got := analysis.DetectDuels(events, analysis.PerPlayerTickIndex{}, rounds, 64)
	if len(got) != 0 {
		t.Errorf("expected no duels for teamkill, got %d (%+v)", len(got), got)
	}
}

// TestDetectDuels_RoundBoundary closes any active duels at the round
// boundary and starts the next round's state machine fresh.
func TestDetectDuels_RoundBoundary(t *testing.T) {
	events := []demo.GameEvent{
		{
			Tick: 100, RoundNumber: 1, Type: "weapon_fire",
			AttackerSteamID: "100", Weapon: "ak47",
			ExtraData: &demo.WeaponFireExtra{Yaw: 0, HitVictimSteamID: "200"},
		},
		{
			Tick: 100, RoundNumber: 1, Type: "player_hurt",
			AttackerSteamID: "100", VictimSteamID: "200",
			ExtraData: &demo.PlayerHurtExtra{AttackerTeam: "T", VictimTeam: "CT"},
		},
		// Round 2 — different shot, should not extend the round-1 duel.
		{
			Tick: 5000, RoundNumber: 2, Type: "weapon_fire",
			AttackerSteamID: "100", Weapon: "ak47",
			ExtraData: &demo.WeaponFireExtra{Yaw: 0, HitVictimSteamID: "200"},
		},
		{
			Tick: 5000, RoundNumber: 2, Type: "player_hurt",
			AttackerSteamID: "100", VictimSteamID: "200",
			ExtraData: &demo.PlayerHurtExtra{AttackerTeam: "T", VictimTeam: "CT"},
		},
	}
	rounds := []demo.RoundData{
		{Number: 1, Roster: []demo.RoundParticipant{
			{SteamID: "100", TeamSide: "T"}, {SteamID: "200", TeamSide: "CT"},
		}},
		{Number: 2, Roster: []demo.RoundParticipant{
			{SteamID: "100", TeamSide: "T"}, {SteamID: "200", TeamSide: "CT"},
		}},
	}
	got := analysis.DetectDuels(events, analysis.PerPlayerTickIndex{}, rounds, 64)
	if len(got) != 2 {
		t.Fatalf("expected 2 duels (one per round), got %d (%+v)", len(got), got)
	}
	if got[0].RoundNumber != 1 || got[1].RoundNumber != 2 {
		t.Errorf("round assignment wrong: %+v", got)
	}
	if got[0].Outcome != analysis.DuelOutcomeInconclusive {
		t.Errorf("round-1 duel should be inconclusive (round boundary), got %q", got[0].Outcome)
	}
}

// TestAttributeMistakesToDuels_FireRule: a shot_while_moving mistake gets
// its DuelID populated when the same fire event opened a duel.
func TestAttributeMistakesToDuels_FireRule(t *testing.T) {
	events := []demo.GameEvent{
		{
			Tick: 100, RoundNumber: 1, Type: "weapon_fire",
			AttackerSteamID: "100", Weapon: "ak47",
			ExtraData: &demo.WeaponFireExtra{Yaw: 0, HitVictimSteamID: "200"},
		},
		{
			Tick: 100, RoundNumber: 1, Type: "player_hurt",
			AttackerSteamID: "100", VictimSteamID: "200",
			ExtraData: &demo.PlayerHurtExtra{AttackerTeam: "T", VictimTeam: "CT"},
		},
	}
	rounds := []demo.RoundData{{
		Number: 1,
		Roster: []demo.RoundParticipant{
			{SteamID: "100", TeamSide: "T"},
			{SteamID: "200", TeamSide: "CT"},
		},
	}}
	duels := analysis.DetectDuels(events, analysis.PerPlayerTickIndex{}, rounds, 64)
	if len(duels) != 1 {
		t.Fatalf("expected 1 duel, got %d", len(duels))
	}
	mistakes := []analysis.Mistake{{
		SteamID:     "100",
		RoundNumber: 1,
		Tick:        100,
		Kind:        string(analysis.MistakeKindShotWhileMoving),
	}}
	got := analysis.AttributeMistakesToDuels(mistakes, duels, events)
	if got[0].DuelID == nil {
		t.Fatalf("expected DuelID populated, got nil")
	}
	if *got[0].DuelID != int64(duels[0].LocalID) {
		t.Errorf("DuelID = %d, want %d", *got[0].DuelID, duels[0].LocalID)
	}
}

// TestAttributeMistakesToDuels_NoDuelKind: eco_misbuy / he_damage never
// attach to a duel even when the round has duels around.
func TestAttributeMistakesToDuels_NoDuelKind(t *testing.T) {
	events := []demo.GameEvent{{
		Tick: 100, RoundNumber: 1, Type: "kill",
		AttackerSteamID: "100", VictimSteamID: "200", Weapon: "ak47",
		ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"},
	}}
	rounds := []demo.RoundData{{
		Number: 1,
		Roster: []demo.RoundParticipant{
			{SteamID: "100", TeamSide: "T"}, {SteamID: "200", TeamSide: "CT"},
		},
	}}
	duels := analysis.DetectDuels(events, analysis.PerPlayerTickIndex{}, rounds, 64)
	mistakes := []analysis.Mistake{
		{SteamID: "100", RoundNumber: 1, Tick: 100, Kind: string(analysis.MistakeKindEcoMisbuy)},
		{SteamID: "100", RoundNumber: 1, Tick: 100, Kind: string(analysis.MistakeKindHeDamage)},
	}
	got := analysis.AttributeMistakesToDuels(mistakes, duels, events)
	for _, m := range got {
		if m.DuelID != nil {
			t.Errorf("%s should not attach to a duel, got %d", m.Kind, *m.DuelID)
		}
	}
}
