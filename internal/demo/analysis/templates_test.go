package analysis

import "testing"

// allMistakeKinds enumerates every persisted MistakeKind so the test
// confirms templates.go stays in lock-step with analyzer.go. Add new kinds
// here when they land in analyzer.go.
var allMistakeKinds = []MistakeKind{
	MistakeKindShotWhileMoving,
	MistakeKindSlowReaction,
	MistakeKindMissedFirstShot,
	MistakeKindSprayDecay,
	MistakeKindNoCounterStrafe,
	MistakeKindIsolatedPeek,
	MistakeKindRepeatedDeathZone,
	MistakeKindEcoMisbuy,
	MistakeKindCaughtReloading,
	MistakeKindFlashAssist,
	MistakeKindHeDamage,
}

func TestTemplates_AllKindsHaveCopy(t *testing.T) {
	t.Parallel()

	for _, kind := range allMistakeKinds {
		k := string(kind)
		t.Run(k, func(t *testing.T) {
			tmpl, ok := templates[k]
			if !ok {
				t.Fatalf("templates[%q]: missing entry", k)
			}
			if tmpl.Title == "" {
				t.Errorf("templates[%q].Title is empty", k)
			}
			if tmpl.Suggestion == "" {
				t.Errorf("templates[%q].Suggestion is empty", k)
			}
			if tmpl.WhyItHurts == "" {
				t.Errorf("templates[%q].WhyItHurts is empty", k)
			}
		})
	}
}

func TestTemplateForKind_KnownKinds(t *testing.T) {
	t.Parallel()

	for _, kind := range allMistakeKinds {
		k := string(kind)
		t.Run(k, func(t *testing.T) {
			got := TemplateForKind(k)
			want, ok := templates[k]
			if !ok {
				t.Fatalf("templates[%q]: missing entry", k)
			}
			if got != want {
				t.Errorf("TemplateForKind(%q) = %+v, want %+v", k, got, want)
			}
		})
	}
}

func TestTemplateForKind_UnknownFallback(t *testing.T) {
	t.Parallel()

	const unknown = "__unknown__"
	got := TemplateForKind(unknown)

	if got.Title != unknown {
		t.Errorf("fallback Title = %q, want %q", got.Title, unknown)
	}
	if got.Suggestion != "" {
		t.Errorf("fallback Suggestion = %q, want empty", got.Suggestion)
	}
	if got.WhyItHurts != "" {
		t.Errorf("fallback WhyItHurts = %q, want empty", got.WhyItHurts)
	}
	if got.Category != CategoryRound {
		t.Errorf("fallback Category = %q, want %q", got.Category, CategoryRound)
	}
	if got.Severity != SeverityLow {
		t.Errorf("fallback Severity = %v, want %v", got.Severity, SeverityLow)
	}
}
