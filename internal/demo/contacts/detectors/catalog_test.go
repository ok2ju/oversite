package detectors

import "testing"

func TestCatalogInvariants(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range Registered {
		if seen[e.Kind] {
			t.Errorf("duplicate kind: %q", e.Kind)
		}
		seen[e.Kind] = true

		if e.Func != nil {
			if e.Phase == "" {
				t.Errorf("v1 detector %q missing Phase", e.Kind)
			}
			if e.Severity < 1 || e.Severity > 3 {
				t.Errorf("v1 detector %q invalid Severity %d", e.Kind, e.Severity)
			}
			if e.Category == "" {
				t.Errorf("v1 detector %q missing Category", e.Kind)
			}
		} else {
			if !e.CrossRound && !e.V2Priority {
				t.Errorf("v2 placeholder %q must set CrossRound or V2Priority", e.Kind)
			}
		}
	}

	v1Kinds := []string{}
	for _, e := range V1() {
		v1Kinds = append(v1Kinds, e.Kind)
	}
	want := []string{
		"slow_reaction", "missed_first_shot", "isolated_peek",
		"bad_crosshair_height", "peek_while_reloading",
		"shot_while_moving", "aim_while_flashed", "lost_hp_advantage",
		"no_reposition_after_kill", "no_reload_with_cover",
	}
	if !sameStringSets(v1Kinds, want) {
		t.Errorf("V1 kinds: got %v, want %v", v1Kinds, want)
	}
	if len(v1Kinds) != 10 {
		t.Errorf("V1 should have 10 detectors, got %d", len(v1Kinds))
	}
}

func TestNewContactMistake_UsesCatalogMetadata(t *testing.T) {
	m := NewContactMistake("slow_reaction", intPtr(9650), map[string]any{"reaction_ms": 300.0})
	if m.Category != "aim" {
		t.Errorf("Category: got %q, want %q", m.Category, "aim")
	}
	if m.Severity != 2 {
		t.Errorf("Severity: got %d, want %d", m.Severity, 2)
	}
	if m.Phase != string(PhasePre) {
		t.Errorf("Phase: got %q, want %q", m.Phase, PhasePre)
	}
	if m.Tick == nil || *m.Tick != 9650 {
		t.Errorf("Tick: got %v, want 9650", m.Tick)
	}
}

func TestNewContactMistake_UnknownKindPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on unknown kind")
		}
	}()
	NewContactMistake("not_a_kind", nil, nil)
}
