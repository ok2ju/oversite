//go:build integration

package websocket

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"

	"github.com/ok2ju/oversite/backend/internal/store"
	"github.com/ok2ju/oversite/backend/internal/testutil"
)

var (
	pgContainer testcontainers.Container
	testDB      *sql.DB
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var pgConnURL string
	var err error
	pgContainer, pgConnURL, err = testutil.PostgresContainer(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "starting postgres container: %v\n", err)
		os.Exit(1)
	}

	if err := testutil.RunMigrations(pgConnURL); err != nil {
		fmt.Fprintf(os.Stderr, "running migrations: %v\n", err)
		pgContainer.Terminate(ctx)
		os.Exit(1)
	}

	testDB, err = sql.Open("postgres", pgConnURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "opening db: %v\n", err)
		pgContainer.Terminate(ctx)
		os.Exit(1)
	}

	code := m.Run()

	testDB.Close()
	pgContainer.Terminate(context.Background())
	os.Exit(code)
}

func createTestUserAndBoard(t *testing.T, q *store.Queries) (store.User, store.StrategyBoard) {
	t.Helper()
	ctx := context.Background()

	user, err := q.CreateUser(ctx, store.CreateUserParams{
		FaceitID: fmt.Sprintf("yjs-test-%d", time.Now().UnixNano()),
		Nickname: "YjsTestPlayer",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	board, err := q.CreateStrategyBoard(ctx, store.CreateStrategyBoardParams{
		UserID:  user.ID,
		Title:   "Integration Test Board",
		MapName: "de_dust2",
	})
	if err != nil {
		t.Fatalf("CreateStrategyBoard: %v", err)
	}

	return user, board
}

func TestPgStateStore_SaveAndLoad(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)
	user, board := createTestUserAndBoard(t, q)
	defer func() {
		q.DeleteStrategyBoard(ctx, board.ID)
		q.DeleteUser(ctx, user.ID)
	}()

	ss := NewPgStateStore(q)

	// Save some Yjs state.
	updates := [][]byte{
		{yjsMsgSync, 0x01, 0x02},
		{yjsMsgSync, 0x03, 0x04, 0x05},
	}
	encoded := EncodeUpdates(updates)

	if err := ss.SaveYjsState(ctx, board.ID.String(), encoded); err != nil {
		t.Fatalf("SaveYjsState: %v", err)
	}

	// Load it back.
	loaded, err := ss.LoadYjsState(ctx, board.ID.String())
	if err != nil {
		t.Fatalf("LoadYjsState: %v", err)
	}
	if !bytes.Equal(loaded, encoded) {
		t.Error("loaded state does not match saved state")
	}

	// Verify it decodes correctly.
	decoded, err := DecodeUpdates(loaded)
	if err != nil {
		t.Fatalf("DecodeUpdates: %v", err)
	}
	if len(decoded) != 2 {
		t.Fatalf("expected 2 updates, got %d", len(decoded))
	}
	for i := range decoded {
		if !bytes.Equal(decoded[i], updates[i]) {
			t.Errorf("update[%d] mismatch", i)
		}
	}
}

func TestPgStateStore_LoadNewBoard(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)
	user, board := createTestUserAndBoard(t, q)
	defer func() {
		q.DeleteStrategyBoard(ctx, board.ID)
		q.DeleteUser(ctx, user.ID)
	}()

	ss := NewPgStateStore(q)

	// Fresh board has NULL yjs_state — should return nil, no error.
	loaded, err := ss.LoadYjsState(ctx, board.ID.String())
	if err != nil {
		t.Fatalf("LoadYjsState: %v", err)
	}
	if loaded != nil {
		t.Errorf("expected nil for fresh board, got %d bytes", len(loaded))
	}
}

func TestPgStateStore_LoadNonExistent(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)
	ss := NewPgStateStore(q)

	// Non-existent UUID — should return nil, no error (sql.ErrNoRows handled).
	loaded, err := ss.LoadYjsState(ctx, uuid.New().String())
	if err != nil {
		t.Fatalf("LoadYjsState: %v", err)
	}
	if loaded != nil {
		t.Errorf("expected nil for non-existent board, got %d bytes", len(loaded))
	}
}

func TestPgStateStore_InvalidBoardID(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)
	ss := NewPgStateStore(q)

	_, err := ss.LoadYjsState(ctx, "not-a-uuid")
	if err == nil {
		t.Fatal("expected error for invalid board ID")
	}

	err = ss.SaveYjsState(ctx, "not-a-uuid", []byte{0x01})
	if err == nil {
		t.Fatal("expected error for invalid board ID")
	}
}

func TestRelay_FullCycle_Integration(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)
	user, board := createTestUserAndBoard(t, q)
	defer func() {
		q.DeleteStrategyBoard(ctx, board.ID)
		q.DeleteUser(ctx, user.ID)
	}()

	ss := NewPgStateStore(q)
	relay := NewYjsRelay(ss, 30*time.Second) // Won't auto-save during test.

	boardID := board.ID.String()

	// 1. First client joins fresh board.
	msgs, err := relay.OnFirstClientJoin(ctx, boardID)
	if err != nil {
		t.Fatalf("OnFirstClientJoin (first): %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(msgs))
	}

	// 2. Accumulate sync messages.
	syncMsg1 := []byte{yjsMsgSync, 0xAA, 0xBB}
	syncMsg2 := []byte{yjsMsgSync, 0xCC, 0xDD, 0xEE}
	relay.HandleMessage(boardID, syncMsg1)
	relay.HandleMessage(boardID, syncMsg2)

	// 3. Last client leaves — persists to real DB.
	relay.OnLastClientLeave(ctx, boardID)

	// 4. Verify DB directly — yjs_state should be non-null.
	dbBoard, err := q.GetStrategyBoardByID(ctx, board.ID)
	if err != nil {
		t.Fatalf("GetStrategyBoardByID: %v", err)
	}
	if dbBoard.YjsState == nil {
		t.Fatal("expected non-null yjs_state in DB after save")
	}

	// 5. First client joins again — should restore messages.
	msgs, err = relay.OnFirstClientJoin(ctx, boardID)
	if err != nil {
		t.Fatalf("OnFirstClientJoin (second): %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 restored messages, got %d", len(msgs))
	}
	if !bytes.Equal(msgs[0], syncMsg1) {
		t.Errorf("msg[0] = %v, want %v", msgs[0], syncMsg1)
	}
	if !bytes.Equal(msgs[1], syncMsg2) {
		t.Errorf("msg[1] = %v, want %v", msgs[1], syncMsg2)
	}

	// Clean up relay.
	relay.OnLastClientLeave(ctx, boardID)
}

func TestRelay_AutoSave_Integration(t *testing.T) {
	ctx := context.Background()
	q := store.New(testDB)
	user, board := createTestUserAndBoard(t, q)
	defer func() {
		q.DeleteStrategyBoard(ctx, board.ID)
		q.DeleteUser(ctx, user.ID)
	}()

	ss := NewPgStateStore(q)
	relay := NewYjsRelay(ss, 100*time.Millisecond) // Fast auto-save for testing.

	boardID := board.ID.String()

	// First client joins + sends a message.
	msgs, err := relay.OnFirstClientJoin(ctx, boardID)
	if err != nil {
		t.Fatalf("OnFirstClientJoin: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(msgs))
	}

	relay.HandleMessage(boardID, []byte{yjsMsgSync, 0xFF})

	// Poll DB until yjs_state is non-null (auto-save should fire within ~100ms).
	deadline := time.After(5 * time.Second)
	for {
		dbBoard, err := q.GetStrategyBoardByID(ctx, board.ID)
		if err != nil {
			t.Fatalf("GetStrategyBoardByID: %v", err)
		}
		if dbBoard.YjsState != nil {
			break
		}
		select {
		case <-deadline:
			t.Fatal("auto-save did not write yjs_state within deadline")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Clean up.
	relay.OnLastClientLeave(ctx, boardID)
}
