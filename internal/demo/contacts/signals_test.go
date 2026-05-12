package contacts_test

import (
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

func baseInputs(events []demo.GameEvent, vis []demo.VisibilityChange) contacts.CollectInputs {
	return contacts.CollectInputs{
		Subject:     "S_P",
		SubjectTeam: "T",
		Round: demo.RoundData{
			Number:        1,
			StartTick:     0,
			FreezeEndTick: 100,
			EndTick:       5000,
		},
		Events:     events,
		Visibility: vis,
		EnemyTeam: map[string]string{
			"S_P":  "T",
			"S_T2": "T",
			"S_E1": "CT",
			"S_E2": "CT",
		},
		SubjectAlive: contacts.AliveRange{SpawnTick: 100, DeathTick: 0},
	}
}

func TestCollectSignals_VisibilityOnly_SubjectVisible(t *testing.T) {
	in := baseInputs(nil, []demo.VisibilityChange{
		{RoundNumber: 1, Tick: 200, SpottedSteam: "S_P", SpotterSteam: "S_E1", State: 1},
	})
	got := contacts.CollectSignals(in)
	if len(got) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(got))
	}
	if got[0].Kind != contacts.SignalVisibility || got[0].Subject != contacts.SubjectVictim {
		t.Errorf("kind/role mismatch: %+v", got[0])
	}
}

func TestCollectSignals_VisibilitySpottedOff_Ignored(t *testing.T) {
	in := baseInputs(nil, []demo.VisibilityChange{
		{RoundNumber: 1, Tick: 200, SpottedSteam: "S_E1", SpotterSteam: "S_P", State: 0},
	})
	if got := contacts.CollectSignals(in); len(got) != 0 {
		t.Errorf("expected 0 signals, got %d", len(got))
	}
}

func TestCollectSignals_WeaponFire_HitSubject(t *testing.T) {
	in := baseInputs([]demo.GameEvent{
		{
			Tick: 200, RoundNumber: 1, Type: "weapon_fire", AttackerSteamID: "S_E1",
			Weapon: "m4a1", ExtraData: &demo.WeaponFireExtra{HitVictimSteamID: "S_P"},
		},
	}, nil)
	got := contacts.CollectSignals(in)
	if len(got) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(got))
	}
	if got[0].Kind != contacts.SignalWeaponFireHit || got[0].Subject != contacts.SubjectVictim {
		t.Errorf("kind/role mismatch: %+v", got[0])
	}
}

func TestCollectSignals_WeaponFire_NoHit_Ignored(t *testing.T) {
	in := baseInputs([]demo.GameEvent{
		{
			Tick: 200, RoundNumber: 1, Type: "weapon_fire", AttackerSteamID: "S_P",
			Weapon: "ak47", ExtraData: &demo.WeaponFireExtra{},
		},
	}, nil)
	if got := contacts.CollectSignals(in); len(got) != 0 {
		t.Errorf("expected 0 signals, got %d", len(got))
	}
}

func TestCollectSignals_PlayerHurt_Friendly_Filtered(t *testing.T) {
	in := baseInputs([]demo.GameEvent{
		{
			Tick: 200, RoundNumber: 1, Type: "player_hurt",
			AttackerSteamID: "S_T2", VictimSteamID: "S_P", Weapon: "ak47",
			ExtraData: &demo.PlayerHurtExtra{HealthDamage: 30, AttackerTeam: "T", VictimTeam: "T"},
		},
	}, nil)
	if got := contacts.CollectSignals(in); len(got) != 0 {
		t.Errorf("friendly fire should be filtered, got %d signals", len(got))
	}
}

func TestCollectSignals_PlayerHurt_Bomb_Filtered(t *testing.T) {
	in := baseInputs([]demo.GameEvent{
		{
			Tick: 200, RoundNumber: 1, Type: "player_hurt",
			AttackerSteamID: "S_E1", VictimSteamID: "S_P", Weapon: "c4",
			ExtraData: &demo.PlayerHurtExtra{HealthDamage: 30, AttackerTeam: "CT", VictimTeam: "T"},
		},
	}, nil)
	if got := contacts.CollectSignals(in); len(got) != 0 {
		t.Errorf("bomb damage should be filtered, got %d signals", len(got))
	}
}

func TestCollectSignals_PlayerHurt_HE_BecomesUtilityDamage(t *testing.T) {
	in := baseInputs([]demo.GameEvent{
		{
			Tick: 200, RoundNumber: 1, Type: "player_hurt",
			AttackerSteamID: "S_E1", VictimSteamID: "S_P", Weapon: "hegrenade",
			ExtraData: &demo.PlayerHurtExtra{HealthDamage: 50, AttackerTeam: "CT", VictimTeam: "T"},
		},
	}, nil)
	got := contacts.CollectSignals(in)
	if len(got) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(got))
	}
	if got[0].Kind != contacts.SignalUtilityDamage {
		t.Errorf("expected utility_damage kind, got %s", got[0].Kind)
	}
}

func TestCollectSignals_Kill_Suicide_Filtered(t *testing.T) {
	in := baseInputs([]demo.GameEvent{
		{
			Tick: 200, RoundNumber: 1, Type: "kill",
			AttackerSteamID: "S_P", VictimSteamID: "S_P", Weapon: "world",
			ExtraData: &demo.KillExtra{},
		},
	}, nil)
	if got := contacts.CollectSignals(in); len(got) != 0 {
		t.Errorf("suicide should be filtered, got %d signals", len(got))
	}
}

func TestCollectSignals_Kill_Friendly_Filtered(t *testing.T) {
	in := baseInputs([]demo.GameEvent{
		{
			Tick: 200, RoundNumber: 1, Type: "kill",
			AttackerSteamID: "S_T2", VictimSteamID: "S_P", Weapon: "ak47",
			ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "T"},
		},
	}, nil)
	if got := contacts.CollectSignals(in); len(got) != 0 {
		t.Errorf("friendly kill should be filtered, got %d signals", len(got))
	}
}

func TestCollectSignals_Flash_BelowThreshold_Filtered(t *testing.T) {
	in := baseInputs([]demo.GameEvent{
		{
			Tick: 200, RoundNumber: 1, Type: "player_flashed",
			AttackerSteamID: "S_E1", VictimSteamID: "S_P",
			ExtraData: &demo.PlayerFlashedExtra{DurationSecs: 0.5},
		},
	}, nil)
	if got := contacts.CollectSignals(in); len(got) != 0 {
		t.Errorf("sub-threshold flash should be filtered, got %d signals", len(got))
	}
}

func TestCollectSignals_Flash_AboveThreshold(t *testing.T) {
	in := baseInputs([]demo.GameEvent{
		{
			Tick: 200, RoundNumber: 1, Type: "player_flashed",
			AttackerSteamID: "S_E1", VictimSteamID: "S_P",
			ExtraData: &demo.PlayerFlashedExtra{DurationSecs: 0.9, AttackerTeam: "CT", VictimTeam: "T"},
		},
	}, nil)
	got := contacts.CollectSignals(in)
	if len(got) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(got))
	}
	if got[0].Kind != contacts.SignalFlash || got[0].Extras.FlashSeconds != 0.9 {
		t.Errorf("flash signal mismatch: %+v", got[0])
	}
}

// TestCollectSignals_SortOrderStable verifies that multiple signals at
// the same tick sort by (Tick, Kind, EnemySteam).
func TestCollectSignals_SortOrderStable(t *testing.T) {
	in := baseInputs([]demo.GameEvent{
		{
			Tick: 200, RoundNumber: 1, Type: "kill",
			AttackerSteamID: "S_P", VictimSteamID: "S_E2", Weapon: "ak47",
			ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"},
		},
		{
			Tick: 200, RoundNumber: 1, Type: "kill",
			AttackerSteamID: "S_P", VictimSteamID: "S_E1", Weapon: "ak47",
			ExtraData: &demo.KillExtra{AttackerTeam: "T", VictimTeam: "CT"},
		},
	}, []demo.VisibilityChange{
		{RoundNumber: 1, Tick: 200, SpottedSteam: "S_E2", SpotterSteam: "S_P", State: 1},
	})
	got := contacts.CollectSignals(in)
	if len(got) != 3 {
		t.Fatalf("expected 3 signals, got %d", len(got))
	}
	// Kind sort: kill < visibility lexicographically; SignalKill="kill",
	// SignalVisibility="visibility". "kill" < "visibility" so kills come first.
	// Within kills, enemy steam ascending: S_E1 < S_E2.
	if got[0].Kind != contacts.SignalKill || got[0].EnemySteam != "S_E1" {
		t.Errorf("got[0]: %+v", got[0])
	}
	if got[1].Kind != contacts.SignalKill || got[1].EnemySteam != "S_E2" {
		t.Errorf("got[1]: %+v", got[1])
	}
	if got[2].Kind != contacts.SignalVisibility {
		t.Errorf("got[2]: %+v", got[2])
	}
}
