// Package detectors runs mistake-detection rules over the persisted
// contact_moments rows produced in Phase 2 and writes findings to
// contact_mistakes. Each rule is a small, pure function over a single
// Contact + DetectorCtx — independently testable and ≤80 LOC.
//
// The Catalog (catalog.go) is the source of truth for which detectors
// run in v1. Cross-round / heuristic detectors live in the catalog as
// placeholders for v2 with the corresponding flag set.
package detectors

import (
	"github.com/ok2ju/oversite/internal/demo/contacts"
)

// Phase tags which window of the contact the mistake happened in.
// Persisted into contact_mistakes.phase. The phase also controls the
// lookback clamp in mistake.go: pre-phase detectors clamp their
// allowed read range against PreviousContactEnd (analysis §9.3),
// during-phase detectors against [TFirst, TLast], post-phase
// detectors against (TLast, TPost].
type Phase string

const (
	PhasePre    Phase = "pre"
	PhaseDuring Phase = "during"
	PhasePost   Phase = "post"
)

// ContactMistake is the persisted shape of one detector finding.
// Mirrors the root-package types.ContactMistake on the JSON wire so
// Phase 4's Wails bindings can re-encode without a transform. Defined
// here (rather than imported from package main, which is unreachable)
// so the detector package owns the only Go-side definition of the
// type. Phase 4's frontend binding maps store.ContactMistake →
// main.ContactMistake → this struct's shape.
type ContactMistake struct {
	Kind     string         `json:"kind"`
	Category string         `json:"category"`
	Severity int            `json:"severity"`
	Phase    string         `json:"phase"`
	Tick     *int32         `json:"tick,omitempty"`
	Extras   map[string]any `json:"extras"`
}

// Detector is the function shape every rule in this package
// implements. Pure: given the same Contact + DetectorCtx returns the
// same []ContactMistake every time. Allocations go through
// NewContactMistake (mistake.go) so the catalog metadata is uniformly
// applied. A nil/empty return means "no mistake found"; detectors
// don't signal errors.
type Detector func(c *contacts.Contact, ctx *DetectorCtx) []ContactMistake
