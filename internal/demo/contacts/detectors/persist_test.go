package detectors

import (
	"context"
	"database/sql"
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

// seedDemoWithContact creates a demo + round + contact_moment and
// returns the demo_id and contact_id. Used as the common fixture for
// Persist and Run-version-gate tests.
func seedDemoWithContact(t *testing.T, db *sql.DB) (demoID, contactID int64) {
	t.Helper()
	ctx := context.Background()
	q := store.New(db)

	d, err := q.CreateDemo(ctx, store.CreateDemoParams{
		MapName: "de_mirage", FilePath: "/tmp/test.dem", FileSize: 1,
		Status: "ready", MatchDate: "2026-01-01",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}

	round, err := q.CreateRound(ctx, store.CreateRoundParams{
		DemoID:        d.ID,
		RoundNumber:   1,
		StartTick:     0,
		FreezeEndTick: 100,
		EndTick:       1000,
		WinnerSide:    "T",
		WinReason:     "",
		CtScore:       0,
		TScore:        1,
		IsOvertime:    0,
		CtTeamName:    "",
		TTeamName:     "",
	})
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	cID, err := q.InsertContactMoment(ctx, store.InsertContactMomentParams{
		DemoID: d.ID, RoundID: round.ID, SubjectSteam: "S_P",
		TFirst: 9600, TLast: 9700, TPre: 9440, TPost: 9796,
		EnemiesJson: "[]", Outcome: "won_clean", SignalCount: 1,
		ExtrasJson: "{}", BuilderVersion: 1,
	})
	if err != nil {
		t.Fatalf("InsertContactMoment: %v", err)
	}
	return d.ID, cID
}

func TestPersist_RoundtripsIdempotently(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	demoID, contactID := seedDemoWithContact(t, db)
	rows := []BoundContactMistake{{
		ContactID: contactID,
		Mistake: ContactMistake{
			Kind: "slow_reaction", Category: "aim", Severity: 2, Phase: "pre",
			Tick: intPtr(9650), Extras: map[string]any{"reaction_ms": 300.0},
		},
	}}

	if err := Persist(ctx, db, demoID, rows); err != nil {
		t.Fatalf("first Persist: %v", err)
	}
	if err := Persist(ctx, db, demoID, rows); err != nil {
		t.Fatalf("second Persist: %v", err)
	}

	q := store.New(db)
	out, err := q.ListContactMistakesByContact(ctx, contactID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d rows, want 1", len(out))
	}
	if out[0].DetectorVersion != int64(DetectorVersion) {
		t.Errorf("detector_version: got %d, want %d", out[0].DetectorVersion, DetectorVersion)
	}
	if out[0].Kind != "slow_reaction" {
		t.Errorf("kind: got %q, want slow_reaction", out[0].Kind)
	}
}

func TestPersist_DeletesPriorRows(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	demoID, contactID := seedDemoWithContact(t, db)

	if err := Persist(ctx, db, demoID, []BoundContactMistake{{
		ContactID: contactID,
		Mistake: ContactMistake{
			Kind: "slow_reaction", Category: "aim", Severity: 2, Phase: "pre",
			Tick: intPtr(9650),
		},
	}}); err != nil {
		t.Fatalf("seed Persist: %v", err)
	}

	// Replace with an empty list — should wipe.
	if err := Persist(ctx, db, demoID, nil); err != nil {
		t.Fatalf("wipe Persist: %v", err)
	}

	q := store.New(db)
	out, err := q.ListContactMistakesByContact(ctx, contactID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected 0 rows after wipe, got %d", len(out))
	}
}

func TestRun_SkipsWhenDetectorVersionUpToDate(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	demoID, contactID := seedDemoWithContact(t, db)

	// Seed an existing row at the current DetectorVersion.
	if err := Persist(ctx, db, demoID, []BoundContactMistake{{
		ContactID: contactID,
		Mistake: ContactMistake{
			Kind: "slow_reaction", Category: "aim", Severity: 2, Phase: "pre",
			Tick: intPtr(9650),
		},
	}}); err != nil {
		t.Fatalf("seed Persist: %v", err)
	}

	result := &demo.ParseResult{Header: demo.MatchHeader{TickRate: 64}}
	contactList := []contacts.ContactMoment{{
		ID: contactID, DemoID: demoID, RoundNumber: 1, SubjectSteam: "S_P",
		TFirst: 9600, TLast: 9700, TPre: 9440, TPost: 9796,
	}}

	_, _, skipped, err := Run(ctx, db, demoID, result, contactList, RunOpts{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !skipped {
		t.Fatal("expected skipped=true; got false")
	}
}

func TestRun_ForceRebuildIgnoresVersion(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	demoID, contactID := seedDemoWithContact(t, db)

	if err := Persist(ctx, db, demoID, []BoundContactMistake{{
		ContactID: contactID,
		Mistake: ContactMistake{
			Kind: "slow_reaction", Category: "aim", Severity: 2, Phase: "pre",
			Tick: intPtr(9650),
		},
	}}); err != nil {
		t.Fatalf("seed Persist: %v", err)
	}

	result := &demo.ParseResult{Header: demo.MatchHeader{TickRate: 64}}
	contactList := []contacts.ContactMoment{{
		ID: contactID, DemoID: demoID, RoundNumber: 1, SubjectSteam: "S_P",
		TFirst: 9600, TLast: 9700, TPre: 9440, TPost: 9796,
	}}

	_, _, skipped, err := Run(ctx, db, demoID, result, contactList, RunOpts{Force: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if skipped {
		t.Error("expected skipped=false with Force=true; got true")
	}
}
