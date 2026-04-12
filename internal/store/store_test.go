package store_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

func createTestUser(t *testing.T, q *store.Queries) store.User {
	t.Helper()
	user, err := q.CreateUser(context.Background(), store.CreateUserParams{
		FaceitID:    "faceit-abc-123",
		Nickname:    "testplayer",
		AvatarUrl:   "https://example.com/avatar.png",
		FaceitElo:   2100,
		FaceitLevel: 10,
		Country:     "US",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	return user
}

func TestCreateAndGetUser(t *testing.T) {
	q, _ := testutil.NewTestQueries(t)
	ctx := context.Background()

	created := createTestUser(t, q)

	if created.ID == 0 {
		t.Fatal("expected non-zero user ID")
	}
	if created.Nickname != "testplayer" {
		t.Errorf("Nickname = %q, want %q", created.Nickname, "testplayer")
	}

	got, err := q.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}

	if got.FaceitID != created.FaceitID {
		t.Errorf("FaceitID = %q, want %q", got.FaceitID, created.FaceitID)
	}
	if got.FaceitElo != 2100 {
		t.Errorf("FaceitElo = %d, want 2100", got.FaceitElo)
	}
	if got.CreatedAt == "" {
		t.Error("CreatedAt should be populated by default")
	}

	gotByFaceit, err := q.GetUserByFaceitID(ctx, "faceit-abc-123")
	if err != nil {
		t.Fatalf("GetUserByFaceitID: %v", err)
	}
	if gotByFaceit.ID != created.ID {
		t.Errorf("GetUserByFaceitID ID = %d, want %d", gotByFaceit.ID, created.ID)
	}
}

func TestUpdateUser(t *testing.T) {
	q, _ := testutil.NewTestQueries(t)
	ctx := context.Background()

	created := createTestUser(t, q)

	updated, err := q.UpdateUser(ctx, store.UpdateUserParams{
		ID:          created.ID,
		Nickname:    "newname",
		AvatarUrl:   created.AvatarUrl,
		FaceitElo:   2200,
		FaceitLevel: 10,
		Country:     "DE",
	})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	if updated.Nickname != "newname" {
		t.Errorf("Nickname = %q, want %q", updated.Nickname, "newname")
	}
	if updated.FaceitElo != 2200 {
		t.Errorf("FaceitElo = %d, want 2200", updated.FaceitElo)
	}
	if updated.UpdatedAt == "" {
		t.Error("UpdatedAt should be populated after update")
	}
	if updated.Country != "DE" {
		t.Errorf("Country = %q, want %q", updated.Country, "DE")
	}
}

func TestCreateDemoAndList(t *testing.T) {
	q, _ := testutil.NewTestQueries(t)
	ctx := context.Background()

	user := createTestUser(t, q)

	demo, err := q.CreateDemo(ctx, store.CreateDemoParams{
		UserID:   user.ID,
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

	// Insert a second demo.
	_, err = q.CreateDemo(ctx, store.CreateDemoParams{
		UserID:   user.ID,
		MapName:  "de_inferno",
		FilePath: "/demos/match2.dem",
		FileSize: 120_000_000,
		Status:   "imported",
	})
	if err != nil {
		t.Fatalf("CreateDemo (second): %v", err)
	}

	demos, err := q.ListDemosByUserID(ctx, store.ListDemosByUserIDParams{
		UserID:   user.ID,
		LimitVal: 10,
	})
	if err != nil {
		t.Fatalf("ListDemosByUserID: %v", err)
	}
	if len(demos) != 2 {
		t.Errorf("len(demos) = %d, want 2", len(demos))
	}

	count, err := q.CountDemosByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountDemosByUserID: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestUpdateDemoAfterParse(t *testing.T) {
	q, _ := testutil.NewTestQueries(t)
	ctx := context.Background()

	user := createTestUser(t, q)
	demo, err := q.CreateDemo(ctx, store.CreateDemoParams{
		UserID:   user.ID,
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

func TestDeleteUser(t *testing.T) {
	q, _ := testutil.NewTestQueries(t)
	ctx := context.Background()

	user := createTestUser(t, q)

	if err := q.DeleteUser(ctx, user.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	_, err := q.GetUserByID(ctx, user.ID)
	if err != sql.ErrNoRows {
		t.Errorf("GetUserByID after delete: err = %v, want sql.ErrNoRows", err)
	}
}
