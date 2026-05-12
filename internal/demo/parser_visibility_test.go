package demo_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

const fixtureDemoPath = "../../testdata/demos/1.dem"

// parseFixtureDemo opens the reference Faceit demo and runs the full parser
// over it. Skips the test if the file is not available (same convention as
// parser_regression_test.go).
func parseFixtureDemo(tb testing.TB, path string) (*demo.ParseResult, error) {
	tb.Helper()
	f, err := os.Open(path)
	if err != nil {
		tb.Skipf("testdata demo not available: %v", err)
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return demo.NewDemoParser().Parse(context.Background(), f)
}

// insertFixtureDemoRow inserts a stand-in demos row so the ingester's
// foreign-key constraints are satisfied. Returns the new demo ID.
func insertFixtureDemoRow(tb testing.TB, q *store.Queries, ctx context.Context) int64 {
	tb.Helper()
	row, err := q.CreateDemo(ctx, store.CreateDemoParams{
		MapName:   "de_unknown",
		FilePath:  "testdata/demos/1.dem",
		FileSize:  0,
		Status:    "ready",
		MatchDate: "2024-01-01T00:00:00Z",
	})
	if err != nil {
		tb.Fatalf("CreateDemo: %v", err)
	}
	return row.ID
}

// TestParseVisibility_Live exercises the full parse → ingest pipeline on the
// reference fixture demo. Asserts:
//   - visibility rows land in player_visibility (> 0, < 50k budget),
//   - ≥ 80 % of kills are preceded by a spotted_on row for (victim, attacker).
//
// Skipped under -short — the demo is ~400 MB and the parse takes seconds.
func TestParseVisibility_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live-demo parse in -short mode")
	}

	demoPath := filepath.Join(fixtureDemoPath)
	if _, err := os.Stat(demoPath); err != nil {
		t.Skipf("testdata demo not available: %v", err)
	}

	q, rawDB := testutil.NewTestQueries(t)
	ctx := context.Background()

	result, err := parseFixtureDemo(t, demoPath)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	demoID := insertFixtureDemoRow(t, q, ctx)

	roundMap, err := demo.IngestRounds(ctx, rawDB, demoID, result)
	if err != nil {
		t.Fatalf("ingest rounds: %v", err)
	}
	if _, err := demo.IngestGameEvents(ctx, rawDB, demoID, result.Events, roundMap); err != nil {
		t.Fatalf("ingest events: %v", err)
	}
	inserted, err := demo.IngestPlayerVisibility(ctx, rawDB, demoID, result.Visibility, roundMap)
	if err != nil {
		t.Fatalf("ingest visibility: %v", err)
	}

	if inserted == 0 {
		t.Fatalf("expected > 0 visibility rows, got 0")
	}
	if inserted > 50_000 {
		t.Fatalf("visibility row count %d exceeds 50k budget — fall back to RLE", inserted)
	}

	// Pair-and-kill correlation: ≥ 80% of kills should have a prior
	// spotted_on row for (victim, attacker) inside the same demo. Wallbangs
	// and pre-fires legitimately erode 100%, so 80% is the floor.
	matched, totalKills := killSpottedCorrelation(t, ctx, rawDB, demoID)
	if totalKills == 0 {
		t.Fatalf("no kills found — fixture demo likely wrong")
	}
	if got := float64(matched) / float64(totalKills); got < 0.80 {
		t.Fatalf("only %d/%d kills have prior spotted_on (%.1f%%) — expected >= 80%%",
			matched, totalKills, got*100)
	}
}

// killSpottedCorrelation returns (matched, total) — the number of kill events
// for which there's a prior spotted_on row for (victim_steam_id,
// attacker_steam_id) in the same demo, and the total kill count.
func killSpottedCorrelation(t *testing.T, ctx context.Context, db *sql.DB, demoID int64) (int, int) {
	t.Helper()

	// testutil opens the DB with SetMaxOpenConns(1), so we can't hold an open
	// rows cursor and run a second statement on the same DB — materialize the
	// kill list first, then query visibility one kill at a time.
	type killRow struct {
		tick     int64
		victim   string
		attacker string
	}
	rows, err := db.QueryContext(ctx, `
		SELECT ge.tick, ge.victim_steam_id, ge.attacker_steam_id
		FROM game_events ge
		WHERE ge.demo_id = ? AND ge.event_type = 'kill'
		  AND ge.victim_steam_id IS NOT NULL
		  AND ge.attacker_steam_id IS NOT NULL
	`, demoID)
	if err != nil {
		t.Fatalf("query kills: %v", err)
	}
	var kills []killRow
	for rows.Next() {
		var k killRow
		var victim, attacker sql.NullString
		if err := rows.Scan(&k.tick, &victim, &attacker); err != nil {
			_ = rows.Close()
			t.Fatalf("scan kill row: %v", err)
		}
		if !victim.Valid || !attacker.Valid {
			continue
		}
		k.victim = victim.String
		k.attacker = attacker.String
		kills = append(kills, k)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		t.Fatalf("iterate kill rows: %v", err)
	}
	_ = rows.Close()

	matched := 0
	for _, k := range kills {
		var hits int
		if err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM player_visibility
			WHERE demo_id = ?
			  AND state = 1
			  AND tick < ?
			  AND spotted_steam = ?
			  AND spotter_steam = ?
		`, demoID, k.tick, k.victim, k.attacker).Scan(&hits); err != nil {
			t.Fatalf("count visibility for kill: %v", err)
		}
		if hits > 0 {
			matched++
		}
	}
	return matched, len(kills)
}

// BenchmarkParseVisibility logs the per-demo visibility row count on the
// reference fixture. The output line is the number recorded into
// docs/knowledge/demo-parser.md per the Phase 1 plan.
func BenchmarkParseVisibility(b *testing.B) {
	if testing.Short() {
		b.Skip("live demo")
	}

	demoPath := filepath.Join(fixtureDemoPath)
	if _, err := os.Stat(demoPath); err != nil {
		b.Skipf("testdata demo not available: %v", err)
	}

	for i := 0; i < b.N; i++ {
		result, err := parseFixtureDemo(b, demoPath)
		if err != nil {
			b.Fatalf("parse: %v", err)
		}
		b.Logf("visibility rows: %d", len(result.Visibility))
		b.Logf("events rows: %d", len(result.Events))
	}
}
