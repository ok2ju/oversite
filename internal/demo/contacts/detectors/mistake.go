package detectors

import (
	"fmt"

	"github.com/ok2ju/oversite/internal/demo/contacts"
)

// NewContactMistake constructs a ContactMistake with the catalog-derived
// Category, Severity, and Phase filled in. Detectors call this rather
// than building the struct literal so the "where do severity / category
// come from?" question is answered once.
//
// tick can be nil (pre-window mistakes with no point-in-time anchor).
// extras is passed through unmodified; nil maps are replaced with an
// empty map so downstream marshalers consistently emit "{}".
func NewContactMistake(kind string, tick *int32, extras map[string]any) ContactMistake {
	e := lookupCatalogEntry(kind)
	if extras == nil {
		extras = map[string]any{}
	}
	return ContactMistake{
		Kind:     kind,
		Category: e.Category,
		Severity: e.Severity,
		Phase:    string(e.Phase),
		Tick:     tick,
		Extras:   extras,
	}
}

// ClampPreLookback returns the lower bound a pre-window detector should
// honor when reading tick samples in (-inf, c.TFirst). Encodes the
// analysis §9.3 contract: pre-window mistakes for contact N+1 are
// filtered to ticks strictly after contact N's TLast.
//
// Returns c.TPre when no previous contact, otherwise
// max(c.TPre, PreviousContactEnd+1).
func ClampPreLookback(c *contacts.Contact, ctx *DetectorCtx) int32 {
	lower := c.TPre
	if ctx.PreviousContactEnd > 0 && ctx.PreviousContactEnd+1 > lower {
		lower = ctx.PreviousContactEnd + 1
	}
	return lower
}

// lookupCatalogEntry returns the Registered Entry for kind. Panics on
// unknown kind — this is a programmer mistake (a detector emitting a
// kind it didn't register), not a runtime condition. catalog_test.go
// covers every kind a v1 detector emits.
func lookupCatalogEntry(kind string) Entry {
	for _, e := range Registered {
		if e.Kind == kind {
			return e
		}
	}
	panic(fmt.Sprintf("detectors: unknown mistake kind %q (not in Registered)", kind))
}
