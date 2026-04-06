package demo

import (
	"testing"
)

func TestExtractGrenadeLineups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mapName string
		events  []GameEvent
		want    int // expected number of lineups
		check   func(t *testing.T, lineups []GrenadeLineup)
	}{
		{
			name:    "empty events",
			mapName: "de_dust2",
			events:  nil,
			want:    0,
		},
		{
			name:    "non-grenade events ignored",
			mapName: "de_dust2",
			events: []GameEvent{
				{Tick: 100, Type: "kill", AttackerSteamID: "123"},
				{Tick: 200, Type: "bomb_plant", AttackerSteamID: "123"},
			},
			want: 0,
		},
		{
			name:    "single throw-detonate pair",
			mapName: "de_dust2",
			events: []GameEvent{
				{
					Tick: 100, RoundNumber: 1, Type: "grenade_throw",
					AttackerSteamID: "111", Weapon: "Smoke Grenade",
					X: -300, Y: -600, Z: 10,
					ExtraData: map[string]interface{}{
						"entity_id":   42,
						"throw_yaw":   90.5,
						"throw_pitch": -15.2,
					},
				},
				{
					Tick: 200, RoundNumber: 1, Type: "smoke_start",
					AttackerSteamID: "111", Weapon: "Smoke Grenade",
					X: 1000, Y: 2000, Z: 20,
					ExtraData: map[string]interface{}{"entity_id": 42},
				},
			},
			want: 1,
			check: func(t *testing.T, lineups []GrenadeLineup) {
				l := lineups[0]
				if l.SteamID != "111" {
					t.Errorf("SteamID = %q, want 111", l.SteamID)
				}
				if l.GrenadeType != "Smoke" {
					t.Errorf("GrenadeType = %q, want Smoke", l.GrenadeType)
				}
				if l.ThrowX != -300 || l.ThrowY != -600 || l.ThrowZ != 10 {
					t.Errorf("throw position = (%v,%v,%v), want (-300,-600,10)", l.ThrowX, l.ThrowY, l.ThrowZ)
				}
				if l.LandX != 1000 || l.LandY != 2000 || l.LandZ != 20 {
					t.Errorf("land position = (%v,%v,%v), want (1000,2000,20)", l.LandX, l.LandY, l.LandZ)
				}
				if l.ThrowYaw != 90.5 {
					t.Errorf("ThrowYaw = %v, want 90.5", l.ThrowYaw)
				}
				if l.ThrowPitch != -15.2 {
					t.Errorf("ThrowPitch = %v, want -15.2", l.ThrowPitch)
				}
				if l.Tick != 100 {
					t.Errorf("Tick = %d, want 100", l.Tick)
				}
				if l.RoundNumber != 1 {
					t.Errorf("RoundNumber = %d, want 1", l.RoundNumber)
				}
				if l.MapName != "de_dust2" {
					t.Errorf("MapName = %q, want de_dust2", l.MapName)
				}
			},
		},
		{
			name:    "multiple players different entity IDs",
			mapName: "de_dust2",
			events: []GameEvent{
				{
					Tick: 100, RoundNumber: 1, Type: "grenade_throw",
					AttackerSteamID: "111", Weapon: "Flashbang",
					X: 100, Y: 200,
					ExtraData: map[string]interface{}{"entity_id": 10},
				},
				{
					Tick: 110, RoundNumber: 1, Type: "grenade_throw",
					AttackerSteamID: "222", Weapon: "HE Grenade",
					X: 300, Y: 400,
					ExtraData: map[string]interface{}{"entity_id": 11},
				},
				{
					Tick: 200, RoundNumber: 1, Type: "grenade_detonate",
					AttackerSteamID: "222", Weapon: "HE Grenade",
					X: 500, Y: 600,
					ExtraData: map[string]interface{}{"entity_id": 11},
				},
				{
					Tick: 210, RoundNumber: 1, Type: "grenade_detonate",
					AttackerSteamID: "111", Weapon: "Flashbang",
					X: 700, Y: 800,
					ExtraData: map[string]interface{}{"entity_id": 10},
				},
			},
			want: 2,
			check: func(t *testing.T, lineups []GrenadeLineup) {
				// First detonation matched is player 222 (HE at tick 200).
				if lineups[0].SteamID != "222" {
					t.Errorf("first lineup SteamID = %q, want 222", lineups[0].SteamID)
				}
				if lineups[0].GrenadeType != "HE" {
					t.Errorf("first lineup GrenadeType = %q, want HE", lineups[0].GrenadeType)
				}
				if lineups[1].SteamID != "111" {
					t.Errorf("second lineup SteamID = %q, want 111", lineups[1].SteamID)
				}
				if lineups[1].GrenadeType != "Flash" {
					t.Errorf("second lineup GrenadeType = %q, want Flash", lineups[1].GrenadeType)
				}
			},
		},
		{
			name:    "same player multiple grenades FIFO",
			mapName: "de_dust2",
			events: []GameEvent{
				{
					Tick: 100, RoundNumber: 1, Type: "grenade_throw",
					AttackerSteamID: "111", Weapon: "Smoke Grenade",
					X: 10, Y: 20,
					ExtraData: map[string]interface{}{"entity_id": 5},
				},
				{
					Tick: 110, RoundNumber: 1, Type: "grenade_throw",
					AttackerSteamID: "111", Weapon: "Smoke Grenade",
					X: 30, Y: 40,
					ExtraData: map[string]interface{}{"entity_id": 5},
				},
				{
					Tick: 200, RoundNumber: 1, Type: "smoke_start",
					AttackerSteamID: "111", Weapon: "Smoke Grenade",
					X: 50, Y: 60,
					ExtraData: map[string]interface{}{"entity_id": 5},
				},
				{
					Tick: 210, RoundNumber: 1, Type: "smoke_start",
					AttackerSteamID: "111", Weapon: "Smoke Grenade",
					X: 70, Y: 80,
					ExtraData: map[string]interface{}{"entity_id": 5},
				},
			},
			want: 2,
			check: func(t *testing.T, lineups []GrenadeLineup) {
				// First throw (x=10) should match first detonation.
				if lineups[0].ThrowX != 10 {
					t.Errorf("first lineup ThrowX = %v, want 10", lineups[0].ThrowX)
				}
				if lineups[0].LandX != 50 {
					t.Errorf("first lineup LandX = %v, want 50", lineups[0].LandX)
				}
				// Second throw (x=30) matches second detonation.
				if lineups[1].ThrowX != 30 {
					t.Errorf("second lineup ThrowX = %v, want 30", lineups[1].ThrowX)
				}
				if lineups[1].LandX != 70 {
					t.Errorf("second lineup LandX = %v, want 70", lineups[1].LandX)
				}
			},
		},
		{
			name:    "unmatched throw silently dropped",
			mapName: "de_dust2",
			events: []GameEvent{
				{
					Tick: 100, Type: "grenade_throw",
					AttackerSteamID: "111", Weapon: "Flashbang",
					ExtraData: map[string]interface{}{"entity_id": 7},
				},
			},
			want: 0,
		},
		{
			name:    "unmatched detonation silently dropped",
			mapName: "de_dust2",
			events: []GameEvent{
				{
					Tick: 200, Type: "grenade_detonate",
					AttackerSteamID: "111", Weapon: "HE Grenade",
					ExtraData: map[string]interface{}{"entity_id": 8},
				},
			},
			want: 0,
		},
		{
			name:    "missing entity_id skipped",
			mapName: "de_dust2",
			events: []GameEvent{
				{
					Tick: 100, Type: "grenade_throw",
					AttackerSteamID: "111", Weapon: "Smoke Grenade",
					ExtraData: map[string]interface{}{},
				},
				{
					Tick: 200, Type: "smoke_start",
					AttackerSteamID: "111", Weapon: "Smoke Grenade",
					ExtraData: map[string]interface{}{"entity_id": 0},
				},
			},
			want: 0,
		},
		{
			name:    "nil ExtraData skipped",
			mapName: "de_dust2",
			events: []GameEvent{
				{Tick: 100, Type: "grenade_throw", AttackerSteamID: "111"},
				{Tick: 200, Type: "grenade_detonate", AttackerSteamID: "111"},
			},
			want: 0,
		},
		{
			name:    "entity_id as float64 from JSON",
			mapName: "de_dust2",
			events: []GameEvent{
				{
					Tick: 100, Type: "grenade_throw",
					AttackerSteamID: "111", Weapon: "HE Grenade",
					ExtraData: map[string]interface{}{"entity_id": float64(42)},
				},
				{
					Tick: 200, Type: "grenade_detonate",
					AttackerSteamID: "111", Weapon: "HE Grenade",
					ExtraData: map[string]interface{}{"entity_id": float64(42)},
				},
			},
			want: 1,
		},
		{
			name:    "decoy_start matches throw",
			mapName: "de_dust2",
			events: []GameEvent{
				{
					Tick: 100, Type: "grenade_throw",
					AttackerSteamID: "111", Weapon: "Decoy Grenade",
					ExtraData: map[string]interface{}{"entity_id": 3},
				},
				{
					Tick: 200, Type: "decoy_start",
					AttackerSteamID: "111", Weapon: "Decoy Grenade",
					ExtraData: map[string]interface{}{"entity_id": 3},
				},
			},
			want: 1,
			check: func(t *testing.T, lineups []GrenadeLineup) {
				if lineups[0].GrenadeType != "Decoy" {
					t.Errorf("GrenadeType = %q, want Decoy", lineups[0].GrenadeType)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ExtractGrenadeLineups(tt.mapName, tt.events)
			if len(got) != tt.want {
				t.Fatalf("got %d lineups, want %d", len(got), tt.want)
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestGenerateTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		mapName                string
		grenadeDisplay         string
		throwX, throwY, throwZ float64
		landX, landY, landZ    float64
		want                   string
	}{
		{
			name:           "known callouts",
			mapName:        "de_dust2",
			grenadeDisplay: "Smoke",
			throwX:         -300, throwY: -600, throwZ: 0,
			landX: 1000, landY: 2000, landZ: 0,
			want: "Smoke T Spawn → A Site",
		},
		{
			name:           "unknown throw position",
			mapName:        "de_dust2",
			grenadeDisplay: "Flash",
			throwX:         9999, throwY: 9999, throwZ: 0,
			landX: 1000, landY: 2000, landZ: 0,
			want: "Flash (9999, 9999) → A Site",
		},
		{
			name:           "unknown map",
			mapName:        "de_unknown",
			grenadeDisplay: "HE",
			throwX:         100, throwY: 200, throwZ: 0,
			landX: 300, landY: 400, landZ: 0,
			want: "HE (100, 200) → (300, 400)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := generateTitle(tt.mapName, tt.grenadeDisplay, tt.throwX, tt.throwY, tt.throwZ, tt.landX, tt.landY, tt.landZ)
			if got != tt.want {
				t.Errorf("generateTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractEntityID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		extra map[string]interface{}
		want  int
	}{
		{"nil map", nil, 0},
		{"missing key", map[string]interface{}{}, 0},
		{"int value", map[string]interface{}{"entity_id": 42}, 42},
		{"float64 value", map[string]interface{}{"entity_id": float64(42)}, 42},
		{"int64 value", map[string]interface{}{"entity_id": int64(42)}, 42},
		{"zero value", map[string]interface{}{"entity_id": 0}, 0},
		{"string value", map[string]interface{}{"entity_id": "42"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractEntityID(tt.extra)
			if got != tt.want {
				t.Errorf("extractEntityID() = %d, want %d", got, tt.want)
			}
		})
	}
}
