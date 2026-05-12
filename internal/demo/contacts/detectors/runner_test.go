package detectors

import (
	"context"
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
	"github.com/ok2ju/oversite/internal/testutil"
)

// TestRunner_EmptyContacts ensures Run returns cleanly when no contacts
// exist for the demo (fresh import, empty roster, etc.).
func TestRunner_EmptyContacts(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	result := &demo.ParseResult{Header: demo.MatchHeader{TickRate: 64}}
	perContact, aggregate, skipped, err := Run(ctx, db, 1, result, nil, RunOpts{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(perContact) != 0 || len(aggregate) != 0 || skipped {
		t.Errorf("expected empty/false; got perContact=%v aggregate=%v skipped=%v",
			perContact, aggregate, skipped)
	}
}

// TestRunner_PreviousContactEndClampsPreWindow is the regression test
// for analysis §9.3. Two adjacent contacts in the same round: the
// second's pre-window must clamp at contact1.TLast+1.
func TestRunner_PreviousContactEndClampsPreWindow(t *testing.T) {
	subject := "S_P"
	c2 := &contacts.Contact{
		Subject: subject, RoundNumber: 1,
		TFirst: 9200, TLast: 9300, TPre: 9040, TPost: 9396,
	}
	dctx := &DetectorCtx{PreviousContactEnd: 9100}
	lower := ClampPreLookback(c2, dctx)
	if lower != 9101 {
		t.Errorf("ClampPreLookback: got %d, want 9101 (c1.TLast+1)", lower)
	}
}

// TestRunner_AggregateWriteback exercises the WriteAggregate
// distinction. A single contact tripping slow_reaction (WriteAggregate)
// and bad_crosshair_height (NOT WriteAggregate) — only slow_reaction
// should appear in the aggregate slice.
func TestRunner_AggregateWriteback(t *testing.T) {
	subject := "1"
	enemy := "2"

	ticks := []demo.AnalysisTick{
		// Subject at (0,0,0) with pitch 30° (way off head height at
		// horiz=1000, dz=0).
		mkTick(9600, 1, 0, 0, 0, 0, 30, 0, 0, true, 30),
		mkTick(9600, 2, 1000, 0, 0, 0, 0, 0, 0, true, 30),
	}

	c := &contacts.Contact{
		Subject:     subject,
		RoundNumber: 1,
		TFirst:      9600,
		TLast:       9700,
		TPre:        9440,
		TPost:       9796,
		Enemies:     []string{enemy},
		Signals: []contacts.Signal{
			{Tick: 9600, EnemySteam: enemy, Kind: contacts.SignalVisibility, Subject: contacts.SubjectAggressor},
			{Tick: 9660, EnemySteam: enemy, Kind: contacts.SignalWeaponFireHit, Subject: contacts.SubjectAggressor},
		},
	}
	dctx := &DetectorCtx{
		Subject: subject, SubjectTeam: "T", TickRate: 64, Ticks: mkTickIndex(ticks),
		PreviousContactEnd: -1,
	}

	out := runAllDetectors(c, dctx)
	perKinds := kindsOf(out.PerContact)
	aggKinds := analysisKindsOf(out.Aggregate)

	if !containsAll(perKinds, []string{"slow_reaction", "bad_crosshair_height"}) {
		t.Errorf("per-contact missing required kinds: got %v", perKinds)
	}
	if !contains(aggKinds, "slow_reaction") {
		t.Errorf("aggregate missing slow_reaction: got %v", aggKinds)
	}
	if contains(aggKinds, "bad_crosshair_height") {
		t.Errorf("aggregate should NOT carry bad_crosshair_height: got %v", aggKinds)
	}
}
