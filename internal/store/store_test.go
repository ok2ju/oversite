package store_test

import (
	"context"
	"testing"

	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

func TestCreateDemoAndList(t *testing.T) {
	q, _ := testutil.NewTestQueries(t)
	ctx := context.Background()

	demo, err := q.CreateDemo(ctx, store.CreateDemoParams{
		MapName:  "de_dust2",
		FilePath: "/demos/match1.dem",
		FileSize: 150_000_000,
		Status:   "imported",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}
	if demo.ID == 0 {
		t.Fatal("expected non-zero demo ID")
	}
	if demo.MapName != "de_dust2" {
		t.Errorf("MapName = %q, want %q", demo.MapName, "de_dust2")
	}
	if demo.Status != "imported" {
		t.Errorf("Status = %q, want %q", demo.Status, "imported")
	}

	_, err = q.CreateDemo(ctx, store.CreateDemoParams{
		MapName:  "de_inferno",
		FilePath: "/demos/match2.dem",
		FileSize: 120_000_000,
		Status:   "imported",
	})
	if err != nil {
		t.Fatalf("CreateDemo (second): %v", err)
	}

	demos, err := q.ListDemos(ctx, store.ListDemosParams{
		LimitVal: 10,
	})
	if err != nil {
		t.Fatalf("ListDemos: %v", err)
	}
	if len(demos) != 2 {
		t.Errorf("len(demos) = %d, want 2", len(demos))
	}

	count, err := q.CountDemos(ctx)
	if err != nil {
		t.Fatalf("CountDemos: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestUpdateDemoAfterParse(t *testing.T) {
	q, _ := testutil.NewTestQueries(t)
	ctx := context.Background()

	demo, err := q.CreateDemo(ctx, store.CreateDemoParams{
		MapName:  "de_dust2",
		FilePath: "/demos/match1.dem",
		FileSize: 150_000_000,
		Status:   "imported",
	})
	if err != nil {
		t.Fatalf("CreateDemo: %v", err)
	}

	parsed, err := q.UpdateDemoAfterParse(ctx, store.UpdateDemoAfterParseParams{
		ID:           demo.ID,
		MapName:      "de_mirage",
		TotalTicks:   128000,
		TickRate:     128.0,
		DurationSecs: 2400,
	})
	if err != nil {
		t.Fatalf("UpdateDemoAfterParse: %v", err)
	}

	if parsed.Status != "ready" {
		t.Errorf("Status = %q, want %q", parsed.Status, "ready")
	}
	if parsed.MapName != "de_mirage" {
		t.Errorf("MapName = %q, want %q", parsed.MapName, "de_mirage")
	}
	if parsed.TotalTicks != 128000 {
		t.Errorf("TotalTicks = %d, want 128000", parsed.TotalTicks)
	}
}

func TestStrategyBoardCRUD(t *testing.T) {
	q, _ := testutil.NewTestQueries(t)
	ctx := context.Background()

	board, err := q.CreateStrategyBoard(ctx, store.CreateStrategyBoardParams{
		Title:   "A-site execute",
		MapName: "de_inferno",
	})
	if err != nil {
		t.Fatalf("CreateStrategyBoard: %v", err)
	}
	if board.BoardState != "{}" {
		t.Errorf("BoardState = %q, want %q", board.BoardState, "{}")
	}

	updated, err := q.UpdateBoardState(ctx, store.UpdateBoardStateParams{
		ID:         board.ID,
		BoardState: `{"elements":[{"type":"arrow"}]}`,
	})
	if err != nil {
		t.Fatalf("UpdateBoardState: %v", err)
	}
	if updated.BoardState != `{"elements":[{"type":"arrow"}]}` {
		t.Errorf("BoardState mismatch after update")
	}

	boards, err := q.ListStrategyBoards(ctx)
	if err != nil {
		t.Fatalf("ListStrategyBoards: %v", err)
	}
	if len(boards) != 1 {
		t.Errorf("len(boards) = %d, want 1", len(boards))
	}
}
