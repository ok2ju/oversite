package demo

import (
	"fmt"
	"testing"
)

func TestExtractEntityID(t *testing.T) {
	tests := []struct {
		name  string
		extra map[string]interface{}
		want  int
	}{
		{
			name:  "int value returns int",
			extra: map[string]interface{}{"entity_id": 42},
			want:  42,
		},
		{
			name:  "float64 value converts to int",
			extra: map[string]interface{}{"entity_id": float64(99)},
			want:  99,
		},
		{
			name:  "int64 value converts to int",
			extra: map[string]interface{}{"entity_id": int64(7)},
			want:  7,
		},
		{
			name:  "missing key returns 0",
			extra: map[string]interface{}{"other_key": 1},
			want:  0,
		},
		{
			name:  "nil map returns 0",
			extra: nil,
			want:  0,
		},
		{
			name:  "string value returns 0",
			extra: map[string]interface{}{"entity_id": "42"},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractEntityID(tt.extra)
			if got != tt.want {
				t.Errorf("extractEntityID(%v) = %d, want %d", tt.extra, got, tt.want)
			}
		})
	}
}

func TestGenerateTitle(t *testing.T) {
	tests := []struct {
		name                   string
		mapName                string
		grenadeDisplay         string
		throwX, throwY, throwZ float64
		landX, landY, landZ    float64
		want                   string
	}{
		{
			name:           "known map callout uses region names",
			mapName:        "de_dust2",
			grenadeDisplay: "Smoke",
			// T Spawn region: MinX: -700, MaxX: 200, MinY: -1100, MaxY: -200
			throwX: 0, throwY: -600, throwZ: 0,
			// A Site region: MinX: 800, MaxX: 1500, MinY: 1800, MaxY: 2800
			landX: 1000, landY: 2000, landZ: 0,
			want: "Smoke T Spawn → A Site",
		},
		{
			name:           "unknown map falls back to coordinate format",
			mapName:        "de_unknown",
			grenadeDisplay: "HE",
			throwX:         100, throwY: 200, throwZ: 0,
			landX: 500, landY: 600, landZ: 0,
			want: fmt.Sprintf("HE (%.0f, %.0f) → (%.0f, %.0f)", 100.0, 200.0, 500.0, 600.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateTitle(tt.mapName, tt.grenadeDisplay, tt.throwX, tt.throwY, tt.throwZ, tt.landX, tt.landY, tt.landZ)
			if got != tt.want {
				t.Errorf("generateTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractGrenadeLineups_BasicSmoke(t *testing.T) {
	events := []GameEvent{
		{
			Tick: 100, RoundNumber: 1, Type: "grenade_throw",
			AttackerSteamID: "123", Weapon: "Smoke Grenade",
			X: 100, Y: 200, Z: 0,
			ExtraData: map[string]interface{}{
				"entity_id":   42,
				"throw_yaw":   90.0,
				"throw_pitch": -10.0,
			},
		},
		{
			Tick: 200, RoundNumber: 1, Type: "smoke_start",
			AttackerSteamID: "123", Weapon: "Smoke Grenade",
			X: 500, Y: 600, Z: 0,
			ExtraData: map[string]interface{}{"entity_id": 42},
		},
	}

	lineups := ExtractGrenadeLineups("de_dust2", events)
	if len(lineups) != 1 {
		t.Fatalf("ExtractGrenadeLineups returned %d lineups, want 1", len(lineups))
	}

	got := lineups[0]
	if got.ThrowX != 100 || got.ThrowY != 200 || got.ThrowZ != 0 {
		t.Errorf("throw position = (%.0f, %.0f, %.0f), want (100, 200, 0)", got.ThrowX, got.ThrowY, got.ThrowZ)
	}
	if got.LandX != 500 || got.LandY != 600 || got.LandZ != 0 {
		t.Errorf("land position = (%.0f, %.0f, %.0f), want (500, 600, 0)", got.LandX, got.LandY, got.LandZ)
	}
	if got.ThrowYaw != 90.0 {
		t.Errorf("ThrowYaw = %.1f, want 90.0", got.ThrowYaw)
	}
	if got.ThrowPitch != -10.0 {
		t.Errorf("ThrowPitch = %.1f, want -10.0", got.ThrowPitch)
	}
	if got.GrenadeType != "Smoke" {
		t.Errorf("GrenadeType = %q, want %q", got.GrenadeType, "Smoke")
	}
	if got.SteamID != "123" {
		t.Errorf("SteamID = %q, want %q", got.SteamID, "123")
	}
	if got.Tick != 100 {
		t.Errorf("Tick = %d, want 100", got.Tick)
	}
}

func TestExtractGrenadeLineups_HEGrenade(t *testing.T) {
	events := []GameEvent{
		{
			Tick: 50, RoundNumber: 2, Type: "grenade_throw",
			AttackerSteamID: "456", Weapon: "HE Grenade",
			X: 10, Y: 20, Z: 5,
			ExtraData: map[string]interface{}{"entity_id": 7},
		},
		{
			Tick: 80, RoundNumber: 2, Type: "grenade_detonate",
			AttackerSteamID: "456", Weapon: "HE Grenade",
			X: 300, Y: 400, Z: 10,
			ExtraData: map[string]interface{}{"entity_id": 7},
		},
	}

	lineups := ExtractGrenadeLineups("de_mirage", events)
	if len(lineups) != 1 {
		t.Fatalf("ExtractGrenadeLineups returned %d lineups, want 1", len(lineups))
	}
	if lineups[0].GrenadeType != "HE" {
		t.Errorf("GrenadeType = %q, want %q", lineups[0].GrenadeType, "HE")
	}
}

func TestExtractGrenadeLineups_FireStart_IncendiaryFix(t *testing.T) {
	events := []GameEvent{
		{
			Tick: 100, Type: "grenade_throw",
			AttackerSteamID: "123", Weapon: "Incendiary Grenade",
			X: 50, Y: 60, Z: 0,
			ExtraData: map[string]interface{}{"entity_id": 55},
		},
		{
			Tick: 250, Type: "fire_start",
			AttackerSteamID: "123", Weapon: "Incendiary Grenade",
			X: 300, Y: 400, Z: 0,
			ExtraData: map[string]interface{}{"entity_id": 55},
		},
	}

	lineups := ExtractGrenadeLineups("de_inferno", events)
	if len(lineups) != 1 {
		t.Fatalf("ExtractGrenadeLineups returned %d lineups, want 1 (fire_start should match)", len(lineups))
	}
	if lineups[0].GrenadeType != "Molotov" {
		t.Errorf("GrenadeType = %q, want %q", lineups[0].GrenadeType, "Molotov")
	}
}

func TestExtractGrenadeLineups_FIFOOrdering(t *testing.T) {
	// Player throws two smokes with the same recycled entity ID; FIFO must pair them correctly.
	events := []GameEvent{
		{
			Tick: 100, RoundNumber: 1, Type: "grenade_throw",
			AttackerSteamID: "111", Weapon: "Smoke Grenade",
			X: 10, Y: 20, Z: 0,
			ExtraData: map[string]interface{}{"entity_id": 9},
		},
		{
			Tick: 200, RoundNumber: 1, Type: "grenade_throw",
			AttackerSteamID: "111", Weapon: "Smoke Grenade",
			X: 30, Y: 40, Z: 0,
			ExtraData: map[string]interface{}{"entity_id": 9},
		},
		{
			Tick: 300, RoundNumber: 1, Type: "smoke_start",
			AttackerSteamID: "111", Weapon: "Smoke Grenade",
			X: 500, Y: 600, Z: 0,
			ExtraData: map[string]interface{}{"entity_id": 9},
		},
		{
			Tick: 400, RoundNumber: 1, Type: "smoke_start",
			AttackerSteamID: "111", Weapon: "Smoke Grenade",
			X: 700, Y: 800, Z: 0,
			ExtraData: map[string]interface{}{"entity_id": 9},
		},
	}

	lineups := ExtractGrenadeLineups("de_dust2", events)
	if len(lineups) != 2 {
		t.Fatalf("ExtractGrenadeLineups returned %d lineups, want 2", len(lineups))
	}

	// First detonation should match first throw (tick 100, throw pos 10,20).
	first := lineups[0]
	if first.Tick != 100 {
		t.Errorf("first lineup Tick = %d, want 100 (FIFO order)", first.Tick)
	}
	if first.ThrowX != 10 || first.ThrowY != 20 {
		t.Errorf("first lineup throw pos = (%.0f, %.0f), want (10, 20)", first.ThrowX, first.ThrowY)
	}
	if first.LandX != 500 || first.LandY != 600 {
		t.Errorf("first lineup land pos = (%.0f, %.0f), want (500, 600)", first.LandX, first.LandY)
	}

	// Second detonation should match second throw (tick 200, throw pos 30,40).
	second := lineups[1]
	if second.Tick != 200 {
		t.Errorf("second lineup Tick = %d, want 200 (FIFO order)", second.Tick)
	}
	if second.ThrowX != 30 || second.ThrowY != 40 {
		t.Errorf("second lineup throw pos = (%.0f, %.0f), want (30, 40)", second.ThrowX, second.ThrowY)
	}
	if second.LandX != 700 || second.LandY != 800 {
		t.Errorf("second lineup land pos = (%.0f, %.0f), want (700, 800)", second.LandX, second.LandY)
	}
}

func TestExtractGrenadeLineups_OrphanedThrow(t *testing.T) {
	events := []GameEvent{
		{
			Tick: 100, Type: "grenade_throw",
			AttackerSteamID: "123", Weapon: "Smoke Grenade",
			X: 10, Y: 20, Z: 0,
			ExtraData: map[string]interface{}{"entity_id": 77},
		},
		// No matching detonation event.
	}

	lineups := ExtractGrenadeLineups("de_dust2", events)
	if len(lineups) != 0 {
		t.Errorf("ExtractGrenadeLineups returned %d lineups, want 0 for orphaned throw", len(lineups))
	}
}

func TestExtractGrenadeLineups_OrphanedDetonation(t *testing.T) {
	events := []GameEvent{
		// Detonation with no prior throw.
		{
			Tick: 200, Type: "smoke_start",
			AttackerSteamID: "123", Weapon: "Smoke Grenade",
			X: 500, Y: 600, Z: 0,
			ExtraData: map[string]interface{}{"entity_id": 88},
		},
	}

	lineups := ExtractGrenadeLineups("de_dust2", events)
	if len(lineups) != 0 {
		t.Errorf("ExtractGrenadeLineups returned %d lineups, want 0 for orphaned detonation", len(lineups))
	}
}

func TestExtractGrenadeLineups_MixedGrenadeTypes(t *testing.T) {
	events := []GameEvent{
		{
			Tick: 100, Type: "grenade_throw",
			AttackerSteamID: "111", Weapon: "Smoke Grenade",
			X: 10, Y: 20, Z: 0,
			ExtraData: map[string]interface{}{"entity_id": 1},
		},
		{
			Tick: 110, Type: "grenade_throw",
			AttackerSteamID: "111", Weapon: "HE Grenade",
			X: 15, Y: 25, Z: 0,
			ExtraData: map[string]interface{}{"entity_id": 2},
		},
		{
			Tick: 200, Type: "smoke_start",
			AttackerSteamID: "111", Weapon: "Smoke Grenade",
			X: 500, Y: 600, Z: 0,
			ExtraData: map[string]interface{}{"entity_id": 1},
		},
		{
			Tick: 210, Type: "grenade_detonate",
			AttackerSteamID: "111", Weapon: "HE Grenade",
			X: 300, Y: 350, Z: 0,
			ExtraData: map[string]interface{}{"entity_id": 2},
		},
	}

	lineups := ExtractGrenadeLineups("de_nuke", events)
	if len(lineups) != 2 {
		t.Fatalf("ExtractGrenadeLineups returned %d lineups, want 2", len(lineups))
	}

	typeSet := map[string]bool{}
	for _, l := range lineups {
		typeSet[l.GrenadeType] = true
	}
	if !typeSet["Smoke"] {
		t.Error("expected a Smoke lineup in mixed result")
	}
	if !typeSet["HE"] {
		t.Error("expected an HE lineup in mixed result")
	}
}

func TestExtractGrenadeLineups_EmptyEvents(t *testing.T) {
	lineups := ExtractGrenadeLineups("de_dust2", []GameEvent{})
	if len(lineups) != 0 {
		t.Errorf("ExtractGrenadeLineups returned %d lineups, want 0 for empty events", len(lineups))
	}
}

func TestExtractGrenadeLineups_MissingEntityID(t *testing.T) {
	// Throw with no entity_id in ExtraData should be skipped entirely.
	events := []GameEvent{
		{
			Tick: 100, Type: "grenade_throw",
			AttackerSteamID: "123", Weapon: "Smoke Grenade",
			X: 10, Y: 20, Z: 0,
			ExtraData: map[string]interface{}{
				"throw_yaw":   45.0,
				"throw_pitch": -5.0,
				// no entity_id
			},
		},
		{
			Tick: 200, Type: "smoke_start",
			AttackerSteamID: "123", Weapon: "Smoke Grenade",
			X: 500, Y: 600, Z: 0,
			ExtraData: map[string]interface{}{"entity_id": 0},
		},
	}

	lineups := ExtractGrenadeLineups("de_dust2", events)
	if len(lineups) != 0 {
		t.Errorf("ExtractGrenadeLineups returned %d lineups, want 0 when entity_id is missing", len(lineups))
	}
}
