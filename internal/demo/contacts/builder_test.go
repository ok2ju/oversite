package contacts_test

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

// TestBuilderScenarios runs the five named golden scenarios from
// .claude/plans/timeline-contact-moments/phase-2/06-tests.md §5.
func TestBuilderScenarios(t *testing.T) {
	scenarios := []string{
		"1v1_won_clean",
		"1v2_traded_death",
		"isolated_untraded_death",
		"wallbang_taken",
		"disengaged_smoke",
	}
	for _, name := range scenarios {
		t.Run(name, func(t *testing.T) {
			runScenario(t, name)
		})
	}
}

// TestBuild_WorkedExample mirrors the analysis §3.2 worked example.
// 8 signals across two enemies cluster into one Contact with
// TFirst=9600, TLast=9661, Enemies=["S_E1","S_E2"].
func TestBuild_WorkedExample(t *testing.T) {
	signals := []contacts.Signal{
		{Tick: 9600, EnemySteam: "S_E1", Kind: contacts.SignalVisibility, Subject: contacts.SubjectAggressor},
		{Tick: 9612, EnemySteam: "S_E1", Kind: contacts.SignalWeaponFireHit, Subject: contacts.SubjectAggressor},
		{Tick: 9620, EnemySteam: "S_E1", Kind: contacts.SignalPlayerHurt, Subject: contacts.SubjectAggressor},
		{Tick: 9621, EnemySteam: "S_E1", Kind: contacts.SignalKill, Subject: contacts.SubjectAggressor},
		{Tick: 9648, EnemySteam: "S_E2", Kind: contacts.SignalVisibility, Subject: contacts.SubjectAggressor},
		{Tick: 9656, EnemySteam: "S_E2", Kind: contacts.SignalWeaponFireHit, Subject: contacts.SubjectVictim},
		{Tick: 9660, EnemySteam: "S_E2", Kind: contacts.SignalPlayerHurt, Subject: contacts.SubjectVictim},
		{Tick: 9661, EnemySteam: "S_E2", Kind: contacts.SignalKill, Subject: contacts.SubjectVictim},
	}
	round := demo.RoundData{
		Number: 14, StartTick: 9000, FreezeEndTick: 9100, EndTick: 12000,
	}
	built := contacts.Build(contacts.BuildInputs{
		Subject:      "S_P",
		Round:        round,
		SubjectAlive: contacts.AliveRange{SpawnTick: 9100, DeathTick: 9661},
		Signals:      signals,
	})
	if len(built) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(built))
	}
	c := built[0]
	if c.TFirst != 9600 {
		t.Errorf("TFirst = %d, want 9600", c.TFirst)
	}
	if c.TLast != 9661 {
		t.Errorf("TLast = %d, want 9661", c.TLast)
	}
	if len(c.Enemies) != 2 || c.Enemies[0] != "S_E1" || c.Enemies[1] != "S_E2" {
		t.Errorf("Enemies = %v, want [S_E1, S_E2]", c.Enemies)
	}
}

// TestBuild_EmptySignals confirms that an empty signal slice yields
// no contacts (and doesn't panic).
func TestBuild_EmptySignals(t *testing.T) {
	got := contacts.Build(contacts.BuildInputs{
		Subject: "S_P",
		Round:   demo.RoundData{Number: 1, FreezeEndTick: 100, EndTick: 1000},
	})
	if got != nil {
		t.Errorf("expected nil, got %d contacts", len(got))
	}
}

// TestBuild_GapSplits proves a signal gap > MergeWindowTicks opens a
// fresh contact.
func TestBuild_GapSplits(t *testing.T) {
	signals := []contacts.Signal{
		{Tick: 500, EnemySteam: "S_E1", Kind: contacts.SignalVisibility, Subject: contacts.SubjectAggressor},
		{Tick: 500 + contacts.MergeWindowTicks + 1, EnemySteam: "S_E1", Kind: contacts.SignalVisibility, Subject: contacts.SubjectAggressor},
	}
	got := contacts.Build(contacts.BuildInputs{
		Subject:      "S_P",
		Round:        demo.RoundData{Number: 1, FreezeEndTick: 100, EndTick: 5000},
		SubjectAlive: contacts.AliveRange{SpawnTick: 100, DeathTick: 0},
		Signals:      signals,
	})
	if len(got) != 2 {
		t.Fatalf("gap > merge window should split: got %d", len(got))
	}
}

// TestBuild_FlashOnly sets the FlashOnly extras flag when every signal
// is a flash.
func TestBuild_FlashOnly(t *testing.T) {
	signals := []contacts.Signal{
		{Tick: 500, EnemySteam: "S_E1", Kind: contacts.SignalFlash, Subject: contacts.SubjectVictim, Extras: contacts.SignalExtras{FlashSeconds: 1.5}},
	}
	got := contacts.Build(contacts.BuildInputs{
		Subject:      "S_P",
		Round:        demo.RoundData{Number: 1, FreezeEndTick: 100, EndTick: 5000},
		SubjectAlive: contacts.AliveRange{SpawnTick: 100, DeathTick: 0},
		Signals:      signals,
	})
	if len(got) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(got))
	}
	if !got[0].Extras.FlashOnly {
		t.Errorf("FlashOnly = false, want true")
	}
}
