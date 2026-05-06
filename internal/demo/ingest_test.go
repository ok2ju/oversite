package demo_test

import (
	"context"
	"fmt"
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

func TestChunkTickParams(t *testing.T) {
	// Build 10 dummy params.
	params := make([]store.InsertTickDataParams, 10)
	for i := range params {
		params[i] = store.InsertTickDataParams{Tick: int64(i)}
	}

	tests := []struct {
		name     string
		input    []store.InsertTickDataParams
		n        int
		wantLens []int
	}{
		{
			name:     "even split",
			input:    params[:6],
			n:        3,
			wantLens: []int{3, 3},
		},
		{
			name:     "uneven split",
			input:    params,
			n:        3,
			wantLens: []int{3, 3, 3, 1},
		},
		{
			name:     "single batch",
			input:    params[:5],
			n:        10,
			wantLens: []int{5},
		},
		{
			name:     "empty input",
			input:    nil,
			n:        3,
			wantLens: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := demo.ChunkTickParamsForTest(tt.input, tt.n)

			if tt.wantLens == nil {
				if got != nil {
					t.Errorf("expected nil, got %d chunks", len(got))
				}
				return
			}

			if len(got) != len(tt.wantLens) {
				t.Fatalf("chunk count = %d, want %d", len(got), len(tt.wantLens))
			}
			for i, wantLen := range tt.wantLens {
				if len(got[i]) != wantLen {
					t.Errorf("chunk[%d] len = %d, want %d", i, len(got[i]), wantLen)
				}
			}
		})
	}
}
