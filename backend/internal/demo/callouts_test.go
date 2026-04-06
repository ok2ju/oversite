package demo

import "testing"

func TestResolveCallout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mapName string
		x, y, z float64
		want    string
	}{
		{"dust2 A Site", "de_dust2", 1000, 2000, 0, "A Site"},
		{"dust2 B Site", "de_dust2", -1200, 1800, 0, "B Site"},
		{"dust2 Mid", "de_dust2", 0, 800, 0, "Mid"},
		{"dust2 T Spawn", "de_dust2", -300, -600, 0, "T Spawn"},
		{"mirage A Site", "de_mirage", -500, -800, 0, "A Site"},
		{"mirage B Site", "de_mirage", -1800, 200, 0, "B Site"},
		{"inferno Banana", "de_inferno", -400, 800, 0, "Banana"},
		{"unknown map fallback", "de_unknown", 100, 200, 0, "(100, 200)"},
		{"known map no match", "de_dust2", 9999, 9999, 0, "(9999, 9999)"},
		{"boundary min edge", "de_dust2", 800, 1800, 0, "A Site"},
		{"boundary max edge", "de_dust2", 1500, 2800, 0, "A Site"},
		{"anubis Mid", "de_anubis", 100, -1000, 0, "Mid"},
		{"ancient B Site", "de_ancient", 600, 200, 0, "B Site"},
		{"nuke Outside", "de_nuke", 800, -600, 0, "Outside"},
		{"overpass A Site", "de_overpass", -400, -400, 0, "A Site"},
		{"nuke A Site upper level", "de_nuke", -200, 200, -400, "A Site"},
		{"nuke B Site lower level", "de_nuke", -200, 200, -700, "B Site"},
		{"nuke Z outside both levels", "de_nuke", -500, 500, 100, "(-500, 500)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := resolveCallout(tt.mapName, tt.x, tt.y, tt.z)
			if got != tt.want {
				t.Errorf("resolveCallout(%q, %v, %v, %v) = %q, want %q", tt.mapName, tt.x, tt.y, tt.z, got, tt.want)
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
