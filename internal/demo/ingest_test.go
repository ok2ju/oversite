package demo_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

// createTestDemo creates a demo record in the test database.
func createTestDemo(t *testing.T, q *store.Queries) store.Demo {
	t.Helper()
	d, err := q.CreateDemo(context.Background(), store.CreateDemoParams{
		FilePath: "/test.dem",
		Status:   "imported",
		FileSize: 1000,
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}
	return d
}

// makeSyntheticTicks generates numPlayers * numTicks synthetic tick snapshots.
func makeSyntheticTicks(numPlayers, numTicks int) []demo.TickSnapshot {
	ticks := make([]demo.TickSnapshot, 0, numPlayers*numTicks)
	for tick := 1; tick <= numTicks; tick++ {
		for p := 0; p < numPlayers; p++ {
			ticks = append(ticks, demo.TickSnapshot{
				Tick:    tick * 4,
				SteamID: fmt.Sprintf("7656119800000%04d", p),
				X:       float64(tick) * 1.5,
				Y:       float64(p) * 2.0,
				Z:       100.0,
				Yaw:     90.0,
				Health:  100,
				Armor:   100,
				IsAlive: true,
				Weapon:  "AK-47",
			})
		}
	}
	return ticks
}

func TestTickIngester_Ingest(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	d := createTestDemo(t, q)

	ticks := makeSyntheticTicks(10, 10) // 100 ticks total
	ingester := demo.NewTickIngester(db, 50)

	count, err := ingester.Ingest(ctx, d.ID, ticks)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if count != 100 {
		t.Errorf("Ingest count = %d, want 100", count)
	}

	// Verify data in DB via GetTickDataByRange.
	rows, err := q.GetTickDataByRange(ctx, store.GetTickDataByRangeParams{
		DemoID:    d.ID,
		StartTick: 0,
		EndTick:   1000,
	})
	if err != nil {
		t.Fatalf("GetTickDataByRange: %v", err)
	}
	if len(rows) != 100 {
		t.Fatalf("DB row count = %d, want 100", len(rows))
	}

	// Spot-check first row field values.
	first := rows[0]
	if first.DemoID != d.ID {
		t.Errorf("DemoID = %d, want %d", first.DemoID, d.ID)
	}
	if first.Tick != 4 {
		t.Errorf("Tick = %d, want 4", first.Tick)
	}
	if first.Health != 100 {
		t.Errorf("Health = %d, want 100", first.Health)
	}
	if first.Armor != 100 {
		t.Errorf("Armor = %d, want 100", first.Armor)
	}
	if first.IsAlive != 1 {
		t.Errorf("IsAlive = %d, want 1", first.IsAlive)
	}
	if first.Weapon != "AK-47" {
		t.Errorf("Weapon = %q, want %q", first.Weapon, "AK-47")
	}
}

func TestTickIngester_Idempotent(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	d := createTestDemo(t, q)

	ticks := makeSyntheticTicks(5, 10) // 50 ticks
	ingester := demo.NewTickIngester(db, 100)

	// First ingest.
	count1, err := ingester.Ingest(ctx, d.ID, ticks)
	if err != nil {
		t.Fatalf("first Ingest: %v", err)
	}
	if count1 != 50 {
		t.Errorf("first Ingest count = %d, want 50", count1)
	}

	// Second ingest — should replace, not duplicate.
	count2, err := ingester.Ingest(ctx, d.ID, ticks)
	if err != nil {
		t.Fatalf("second Ingest: %v", err)
	}
	if count2 != 50 {
		t.Errorf("second Ingest count = %d, want 50", count2)
	}

	rows, err := q.GetTickDataByRange(ctx, store.GetTickDataByRangeParams{
		DemoID:    d.ID,
		StartTick: 0,
		EndTick:   1000,
	})
	if err != nil {
		t.Fatalf("GetTickDataByRange: %v", err)
	}
	if len(rows) != 50 {
		t.Errorf("DB row count after two ingests = %d, want 50", len(rows))
	}
}

func TestTickIngester_EmptyTicks(t *testing.T) {
	_, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	ingester := demo.NewTickIngester(db, 100)

	count, err := ingester.Ingest(ctx, 999, nil)
	if err != nil {
		t.Fatalf("Ingest empty: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestTickIngester_BoolConversion(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	d := createTestDemo(t, q)

	ticks := []demo.TickSnapshot{
		{Tick: 4, SteamID: "alive_player", X: 1, Y: 2, Z: 3, Yaw: 90, Health: 100, Armor: 50, IsAlive: true, Weapon: "AK-47"},
		{Tick: 4, SteamID: "dead_player", X: 4, Y: 5, Z: 6, Yaw: 180, Health: 0, Armor: 0, IsAlive: false, Weapon: ""},
	}

	ingester := demo.NewTickIngester(db, 100)
	count, err := ingester.Ingest(ctx, d.ID, ticks)
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	rows, err := q.GetTickDataByRange(ctx, store.GetTickDataByRangeParams{
		DemoID:    d.ID,
		StartTick: 0,
		EndTick:   100,
	})
	if err != nil {
		t.Fatalf("GetTickDataByRange: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("row count = %d, want 2", len(rows))
	}

	// Rows are ordered by tick, steam_id. "alive_player" < "dead_player" lexicographically.
	alive := rows[0]
	dead := rows[1]

	if alive.SteamID != "alive_player" {
		t.Fatalf("expected alive_player first, got %q", alive.SteamID)
	}
	if alive.IsAlive != 1 {
		t.Errorf("alive player IsAlive = %d, want 1", alive.IsAlive)
	}
	if dead.IsAlive != 0 {
		t.Errorf("dead player IsAlive = %d, want 0", dead.IsAlive)
	}
}

// TestTickIngester_IngestStream covers the streaming path used by the
// parse → channel → ingest pipeline in app.parseDemo. The buffer is
// intentionally smaller than the input so the producer goroutine has to
// block on the consumer at least once, exercising the backpressure path.
func TestTickIngester_IngestStream(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	d := createTestDemo(t, q)

	ticks := makeSyntheticTicks(5, 30) // 150 ticks total
	ch := make(chan demo.TickSnapshot, 16)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(ch)
		for _, t := range ticks {
			ch <- t
		}
	}()

	ingester := demo.NewTickIngester(db, 50)
	count, err := ingester.IngestStream(ctx, d.ID, ch)
	if err != nil {
		t.Fatalf("IngestStream: %v", err)
	}
	wg.Wait()

	if count != int64(len(ticks)) {
		t.Errorf("IngestStream count = %d, want %d", count, len(ticks))
	}

	rows, err := q.GetTickDataByRange(ctx, store.GetTickDataByRangeParams{
		DemoID:    d.ID,
		StartTick: 0,
		EndTick:   1000,
	})
	if err != nil {
		t.Fatalf("GetTickDataByRange: %v", err)
	}
	if len(rows) != len(ticks) {
		t.Errorf("DB row count = %d, want %d", len(rows), len(ticks))
	}
}

// TestTickIngester_IngestStream_PartialBatch ensures the partial-batch flush
// fires when the row count is not a multiple of the batch size — the easiest
// place for a streaming bug to drop the last <batchSize rows.
func TestTickIngester_IngestStream_PartialBatch(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	d := createTestDemo(t, q)

	// 7 ticks with batch size 5: one full batch, one partial of 2.
	ticks := makeSyntheticTicks(1, 7)
	ch := make(chan demo.TickSnapshot, len(ticks))
	for _, t := range ticks {
		ch <- t
	}
	close(ch)

	ingester := demo.NewTickIngester(db, 5)
	count, err := ingester.IngestStream(ctx, d.ID, ch)
	if err != nil {
		t.Fatalf("IngestStream: %v", err)
	}
	if count != int64(len(ticks)) {
		t.Errorf("IngestStream count = %d, want %d", count, len(ticks))
	}

	rows, err := q.GetTickDataByRange(ctx, store.GetTickDataByRangeParams{
		DemoID:    d.ID,
		StartTick: 0,
		EndTick:   1000,
	})
	if err != nil {
		t.Fatalf("GetTickDataByRange: %v", err)
	}
	if len(rows) != len(ticks) {
		t.Errorf("DB row count = %d, want %d (partial-batch flush dropped rows?)", len(rows), len(ticks))
	}
}

// TestTickIngester_IngestStream_EmptyChannel covers the case where the parser
// closes the channel before sending anything (e.g. an empty / extremely short
// demo). The transaction must commit cleanly with zero rows.
func TestTickIngester_IngestStream_EmptyChannel(t *testing.T) {
	_, db := testutil.NewTestQueries(t)
	ctx := context.Background()

	ch := make(chan demo.TickSnapshot)
	close(ch)

	ingester := demo.NewTickIngester(db, 100)
	count, err := ingester.IngestStream(ctx, 999, ch)
	if err != nil {
		t.Fatalf("IngestStream empty: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

// TestTickIngester_IngestStream_CtxCancel verifies that a ctx-cancel mid-stream
// causes IngestStream to return ctx.Err() promptly, with the deferred rollback
// undoing any partial inserts. This is the path the parser→ingester errgroup
// uses to abort when the producer fails.
func TestTickIngester_IngestStream_CtxCancel(t *testing.T) {
	q, db := testutil.NewTestQueries(t)

	d := createTestDemo(t, q)

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan demo.TickSnapshot, 4)

	// Cancel before any reads happen — the ingester's select picks up Done()
	// instead of pulling from ch and returns ctx.Err().
	cancel()

	ingester := demo.NewTickIngester(db, 50)
	_, err := ingester.IngestStream(ctx, d.ID, ch)
	if err == nil {
		t.Fatalf("expected ctx-cancel error, got nil")
	}
	if !errorIsCanceled(err) {
		t.Errorf("expected context.Canceled-derived error, got %v", err)
	}

	// Verify rollback wiped any inserts. The DeleteTickDataByDemoID call ran
	// inside the tx, so a clean rollback leaves any pre-existing rows alone —
	// since we never inserted any to begin with, the row count is 0.
	rows, err := q.GetTickDataByRange(context.Background(), store.GetTickDataByRangeParams{
		DemoID:    d.ID,
		StartTick: 0,
		EndTick:   1000,
	})
	if err != nil {
		t.Fatalf("GetTickDataByRange: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("DB row count after rollback = %d, want 0", len(rows))
	}
}

// errorIsCanceled returns true if err's chain includes context.Canceled,
// allowing the test to be tolerant of database/sql wrapping the ctx error.
func errorIsCanceled(err error) bool {
	for e := err; e != nil; {
		if e == context.Canceled || e.Error() == context.Canceled.Error() {
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := e.(unwrapper)
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}
