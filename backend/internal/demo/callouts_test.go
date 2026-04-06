package demo

import "testing"

func TestResolveCallout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mapName string
		x, y    float64
		want    string
	}{
		{"dust2 A Site", "de_dust2", 1000, 2000, "A Site"},
		{"dust2 B Site", "de_dust2", -1200, 1800, "B Site"},
		{"dust2 Mid", "de_dust2", 0, 800, "Mid"},
		{"dust2 T Spawn", "de_dust2", -300, -600, "T Spawn"},
		{"mirage A Site", "de_mirage", -500, -800, "A Site"},
		{"mirage B Site", "de_mirage", -1800, 200, "B Site"},
		{"inferno Banana", "de_inferno", -400, 800, "Banana"},
		{"unknown map fallback", "de_unknown", 100, 200, "(100, 200)"},
		{"known map no match", "de_dust2", 9999, 9999, "(9999, 9999)"},
		{"boundary min edge", "de_dust2", 800, 1800, "A Site"},
		{"boundary max edge", "de_dust2", 1500, 2800, "A Site"},
		{"anubis Mid", "de_anubis", 100, -1000, "Mid"},
		{"ancient B Site", "de_ancient", 600, 200, "B Site"},
		{"nuke Outside", "de_nuke", 800, -600, "Outside"},
		{"overpass A Site", "de_overpass", -400, -400, "A Site"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := resolveCallout(tt.mapName, tt.x, tt.y)
			if got != tt.want {
				t.Errorf("resolveCallout(%q, %v, %v) = %q, want %q", tt.mapName, tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestGrenadeDisplayName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		weapon string
		want   string
	}{
		{"Smoke Grenade", "Smoke"},
		{"Flashbang", "Flash"},
		{"HE Grenade", "HE"},
		{"Decoy Grenade", "Decoy"},
		{"Decoy", "Decoy"},
		{"Incendiary Grenade", "Molotov"},
		{"Molotov", "Molotov"},
		{"Unknown Weapon", "Unknown Weapon"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.weapon, func(t *testing.T) {
			t.Parallel()
			got := grenadeDisplayName(tt.weapon)
			if got != tt.want {
				t.Errorf("grenadeDisplayName(%q) = %q, want %q", tt.weapon, got, tt.want)
			}
		})
	}
}
