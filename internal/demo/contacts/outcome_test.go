package contacts_test

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

func newContact(signals []contacts.Signal, enemies []string, extras contacts.ContactExtras) *contacts.Contact {
	return &contacts.Contact{
		Subject: "S_P",
		Enemies: enemies,
		Signals: signals,
		Extras:  extras,
		TLast:   signals[len(signals)-1].Tick,
	}
}

func enemyTeam() map[string]string {
	return map[string]string{
		"S_E1": "CT",
		"S_E2": "CT",
		"S_E3": "CT",
		"S_T2": "T",
	}
}

func TestClassify_WonClean(t *testing.T) {
	c := newContact([]contacts.Signal{
		{Tick: 500, EnemySteam: "S_E1", Kind: contacts.SignalVisibility, Subject: contacts.SubjectAggressor},
		{Tick: 510, EnemySteam: "S_E1", Kind: contacts.SignalKill, Subject: contacts.SubjectAggressor},
	}, []string{"S_E1"}, contacts.ContactExtras{})
	got := contacts.Classify(contacts.ClassifyInputs{
		Contact: c, Subject: "S_P", SubjectTeam: "T", EnemyTeam: enemyTeam(),
	})
	if got != contacts.OutcomeWonClean {
		t.Errorf("Classify = %s, want won_clean", got)
	}
}

func TestClassify_WonDamaged(t *testing.T) {
	c := newContact([]contacts.Signal{
		{Tick: 500, EnemySteam: "S_E1", Kind: contacts.SignalPlayerHurt, Subject: contacts.SubjectVictim, Extras: contacts.SignalExtras{HealthDamage: 30}},
		{Tick: 510, EnemySteam: "S_E1", Kind: contacts.SignalKill, Subject: contacts.SubjectAggressor},
	}, []string{"S_E1"}, contacts.ContactExtras{})
	got := contacts.Classify(contacts.ClassifyInputs{
		Contact: c, Subject: "S_P", SubjectTeam: "T", EnemyTeam: enemyTeam(),
	})
	if got != contacts.OutcomeWonDamaged {
		t.Errorf("Classify = %s, want won_damaged", got)
	}
}

func TestClassify_TradedWin(t *testing.T) {
	c := newContact([]contacts.Signal{
		{Tick: 500, EnemySteam: "S_E1", Kind: contacts.SignalKill, Subject: contacts.SubjectAggressor},
		{Tick: 510, EnemySteam: "S_E2", Kind: contacts.SignalKill, Subject: contacts.SubjectVictim},
	}, []string{"S_E1", "S_E2"}, contacts.ContactExtras{})
	got := contacts.Classify(contacts.ClassifyInputs{
		Contact: c, Subject: "S_P", SubjectTeam: "T", EnemyTeam: enemyTeam(),
		PostWindowKills: []demo.GameEvent{
			{Tick: 550, Type: "kill", AttackerSteamID: "S_T2", VictimSteamID: "S_E2", ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"}},
		},
	})
	if got != contacts.OutcomeTradedWin {
		t.Errorf("Classify = %s, want traded_win", got)
	}
}

func TestClassify_TradedDeath(t *testing.T) {
	c := newContact([]contacts.Signal{
		{Tick: 500, EnemySteam: "S_E1", Kind: contacts.SignalKill, Subject: contacts.SubjectVictim},
	}, []string{"S_E1"}, contacts.ContactExtras{})
	got := contacts.Classify(contacts.ClassifyInputs{
		Contact: c, Subject: "S_P", SubjectTeam: "T", EnemyTeam: enemyTeam(),
		PostWindowKills: []demo.GameEvent{
			{Tick: 550, Type: "kill", AttackerSteamID: "S_T2", VictimSteamID: "S_E1", ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"}},
		},
	})
	if got != contacts.OutcomeTradedDeath {
		t.Errorf("Classify = %s, want traded_death", got)
	}
}

func TestClassify_UntradedDeath(t *testing.T) {
	c := newContact([]contacts.Signal{
		{Tick: 500, EnemySteam: "S_E1", Kind: contacts.SignalKill, Subject: contacts.SubjectVictim},
	}, []string{"S_E1"}, contacts.ContactExtras{})
	got := contacts.Classify(contacts.ClassifyInputs{
		Contact: c, Subject: "S_P", SubjectTeam: "T", EnemyTeam: enemyTeam(),
	})
	if got != contacts.OutcomeUntradedDeath {
		t.Errorf("Classify = %s, want untraded_death", got)
	}
}

func TestClassify_Disengaged(t *testing.T) {
	c := newContact([]contacts.Signal{
		{Tick: 500, EnemySteam: "S_E1", Kind: contacts.SignalVisibility, Subject: contacts.SubjectAggressor},
	}, []string{"S_E1"}, contacts.ContactExtras{})
	got := contacts.Classify(contacts.ClassifyInputs{
		Contact: c, Subject: "S_P", SubjectTeam: "T", EnemyTeam: enemyTeam(),
	})
	if got != contacts.OutcomeDisengaged {
		t.Errorf("Classify = %s, want disengaged", got)
	}
}

func TestClassify_FlashOnly_ShortCircuits(t *testing.T) {
	c := newContact([]contacts.Signal{
		{Tick: 500, EnemySteam: "S_E1", Kind: contacts.SignalFlash, Subject: contacts.SubjectVictim, Extras: contacts.SignalExtras{FlashSeconds: 1.5}},
	}, []string{"S_E1"}, contacts.ContactExtras{FlashOnly: true})
	got := contacts.Classify(contacts.ClassifyInputs{
		Contact: c, Subject: "S_P", SubjectTeam: "T", EnemyTeam: enemyTeam(),
	})
	if got != contacts.OutcomeDisengaged {
		t.Errorf("Classify = %s, want disengaged (flash_only)", got)
	}
}

func TestClassify_PartialWin(t *testing.T) {
	c := newContact([]contacts.Signal{
		{Tick: 500, EnemySteam: "S_E1", Kind: contacts.SignalKill, Subject: contacts.SubjectAggressor},
		{Tick: 520, EnemySteam: "S_E2", Kind: contacts.SignalVisibility, Subject: contacts.SubjectAggressor},
	}, []string{"S_E1", "S_E2"}, contacts.ContactExtras{})
	got := contacts.Classify(contacts.ClassifyInputs{
		Contact: c, Subject: "S_P", SubjectTeam: "T", EnemyTeam: enemyTeam(),
	})
	if got != contacts.OutcomePartialWin {
		t.Errorf("Classify = %s, want partial_win", got)
	}
}

func TestClassify_MutualDamageNoKill(t *testing.T) {
	c := newContact([]contacts.Signal{
		{Tick: 500, EnemySteam: "S_E1", Kind: contacts.SignalPlayerHurt, Subject: contacts.SubjectVictim, Extras: contacts.SignalExtras{HealthDamage: 40}},
		{Tick: 510, EnemySteam: "S_E1", Kind: contacts.SignalPlayerHurt, Subject: contacts.SubjectAggressor, Extras: contacts.SignalExtras{HealthDamage: 20}},
	}, []string{"S_E1"}, contacts.ContactExtras{})
	got := contacts.Classify(contacts.ClassifyInputs{
		Contact: c, Subject: "S_P", SubjectTeam: "T", EnemyTeam: enemyTeam(),
	})
	if got != contacts.OutcomeMutualDamageNoKill {
		t.Errorf("Classify = %s, want mutual_damage_no_kill", got)
	}
}

// TestClassify_WallbangFlag_DoesNotAlterOutcome confirms the wallbang
// flag survives without changing the outcome label.
func TestClassify_WallbangFlag_DoesNotAlterOutcome(t *testing.T) {
	c := newContact([]contacts.Signal{
		{Tick: 500, EnemySteam: "S_E1", Kind: contacts.SignalPlayerHurt, Subject: contacts.SubjectVictim, Extras: contacts.SignalExtras{HealthDamage: 30, Penetrated: 1}},
	}, []string{"S_E1"}, contacts.ContactExtras{WallbangTaken: true})
	got := contacts.Classify(contacts.ClassifyInputs{
		Contact: c, Subject: "S_P", SubjectTeam: "T", EnemyTeam: enemyTeam(),
	})
	if got != contacts.OutcomeDisengaged {
		t.Errorf("Classify = %s, want disengaged (no kill, no return damage)", got)
	}
	if !c.Extras.WallbangTaken {
		t.Errorf("WallbangTaken flag was cleared by Classify")
	}
}
