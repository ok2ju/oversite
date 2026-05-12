package detectors

import "testing"

func TestDefaultWeaponCatalog(t *testing.T) {
	c := DefaultWeaponCatalog()
	cases := []struct {
		name        string
		wantMaxClip int
		wantAuto    bool
	}{
		{"ak47", 30, true},
		{"m4a1", 30, true},
		{"awp", 5, false},
		{"deagle", 7, false},
		{"p90", 50, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info, ok := c.Lookup(tc.name)
			if !ok {
				t.Fatalf("missing %s", tc.name)
			}
			if info.MaxClip != tc.wantMaxClip {
				t.Errorf("MaxClip: got %d, want %d", info.MaxClip, tc.wantMaxClip)
			}
			if info.IsAuto != tc.wantAuto {
				t.Errorf("IsAuto: got %v, want %v", info.IsAuto, tc.wantAuto)
			}
		})
	}

	if _, ok := c.Lookup("knife"); ok {
		t.Error("knife should not be in catalog")
	}
	if _, ok := c.Lookup(""); ok {
		t.Error("empty weapon should not match")
	}

	if len(c) < 25 {
		t.Errorf("catalog should have >=25 entries; got %d", len(c))
	}
}
