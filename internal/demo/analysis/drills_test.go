package analysis

import (
	"testing"
)

func TestPickNextDrill(t *testing.T) {
	row := func(key HabitKey, status Status) HabitRow {
		return HabitRow{Key: key, Status: status}
	}

	cases := []struct {
		name     string
		rows     []HabitRow
		wantKey  HabitKey
		wantWhy  string // exact match against Norm.Description; "" => check non-empty for catalog drills, exact for maintenance
		wantNote string
	}{
		{
			name:    "bad beats warn even when warn ranks higher",
			rows:    []HabitRow{row(HabitCounterStrafe, StatusWarn), row(HabitTradeTiming, StatusBad)},
			wantKey: HabitTradeTiming,
		},
		{
			name:    "ties broken by impact rank — counter_strafe wins over reaction",
			rows:    []HabitRow{row(HabitReaction, StatusBad), row(HabitCounterStrafe, StatusBad)},
			wantKey: HabitCounterStrafe,
		},
		{
			name:    "ties broken by impact rank — first_shot beats shooting_in_motion",
			rows:    []HabitRow{row(HabitShootingInMotion, StatusBad), row(HabitFirstShotAcc, StatusBad)},
			wantKey: HabitFirstShotAcc,
		},
		{
			name:    "warn picked when no bad",
			rows:    []HabitRow{row(HabitReaction, StatusWarn), row(HabitFirstShotAcc, StatusGood)},
			wantKey: HabitReaction,
		},
		{
			name:     "all good returns maintenance drill",
			rows:     []HabitRow{row(HabitCounterStrafe, StatusGood), row(HabitReaction, StatusGood)},
			wantKey:  "",
			wantNote: "maintenance",
		},
		{
			name:     "empty rows returns maintenance drill",
			rows:     nil,
			wantKey:  "",
			wantNote: "maintenance",
		},
		{
			name:    "habits not in drill catalog are ignored",
			rows:    []HabitRow{row(HabitUntradedDeaths, StatusBad), row(HabitReaction, StatusWarn)},
			wantKey: HabitReaction,
		},
		{
			name:     "all-bad habits but none in drill catalog returns maintenance",
			rows:     []HabitRow{row(HabitUntradedDeaths, StatusBad), row(HabitIsolatedPeekDeaths, StatusBad)},
			wantKey:  "",
			wantNote: "maintenance",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PickNextDrill(tc.rows)
			if got.Key != tc.wantKey {
				t.Fatalf("Key = %q, want %q", got.Key, tc.wantKey)
			}
			if tc.wantNote == "maintenance" {
				if got.Title != MaintenanceDrill.Title {
					t.Errorf("Title = %q, want maintenance %q", got.Title, MaintenanceDrill.Title)
				}
				if got.Why != MaintenanceDrill.Why {
					t.Errorf("Why = %q, want maintenance %q", got.Why, MaintenanceDrill.Why)
				}
				return
			}
			if got.Title == "" {
				t.Errorf("Title is empty for catalog drill %q", got.Key)
			}
			if got.Duration == "" {
				t.Errorf("Duration is empty for catalog drill %q", got.Key)
			}
			if got.Why == "" {
				t.Errorf("Why is empty for catalog drill %q — expected Norm.Description", got.Key)
			}
			if n, ok := LookupNorm(got.Key); ok && got.Why != n.Description {
				t.Errorf("Why = %q, want Norm.Description %q", got.Why, n.Description)
			}
		})
	}
}

func TestDrillCatalogCoversReferenceHabits(t *testing.T) {
	// §6.2 of the plan lists 7 drills; lock the catalog to that set so a
	// future drift surfaces here, not in production data.
	want := []HabitKey{
		HabitCounterStrafe,
		HabitFirstShotAcc,
		HabitShootingInMotion,
		HabitReaction,
		HabitCrouchBeforeShot,
		HabitFlickBalance,
		HabitTradeTiming,
	}
	if len(drillCatalog) != len(want) {
		t.Fatalf("drillCatalog has %d entries, want %d", len(drillCatalog), len(want))
	}
	for i, d := range drillCatalog {
		if d.Key != want[i] {
			t.Errorf("drillCatalog[%d].Key = %q, want %q", i, d.Key, want[i])
		}
		if d.Title == "" {
			t.Errorf("drillCatalog[%d].Title is empty", i)
		}
		if d.Duration == "" {
			t.Errorf("drillCatalog[%d].Duration is empty", i)
		}
		if len(d.Chips) == 0 {
			t.Errorf("drillCatalog[%d].Chips is empty", i)
		}
	}
}

func TestMaintenanceDrillHasFallbackCopy(t *testing.T) {
	if MaintenanceDrill.Title == "" || MaintenanceDrill.Why == "" || MaintenanceDrill.Duration == "" {
		t.Errorf("MaintenanceDrill missing copy: %+v", MaintenanceDrill)
	}
	if MaintenanceDrill.Key != "" {
		t.Errorf("MaintenanceDrill.Key = %q, want empty", MaintenanceDrill.Key)
	}
}
