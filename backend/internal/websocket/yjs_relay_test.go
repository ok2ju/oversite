package websocket

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// --- mock StateStore ---

type mockStateStore struct {
	mu       sync.Mutex
	state    map[string][]byte
	loadErr  error
	saveErr  error
	saveCnt  int
	lastSave []byte
}

func newMockStateStore() *mockStateStore {
	return &mockStateStore{state: make(map[string][]byte)}
}

func (m *mockStateStore) LoadYjsState(_ context.Context, boardID string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return m.state[boardID], nil
}

func (m *mockStateStore) SaveYjsState(_ context.Context, boardID string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveErr != nil {
		return m.saveErr
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	m.state[boardID] = cp
	m.lastSave = cp
	m.saveCnt++
	return nil
}

func (m *mockStateStore) getSaveCnt() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveCnt
}

// --- Encode/Decode tests ---

func TestEncodeDecodeUpdates_RoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		updates [][]byte
	}{
		{"zero updates", nil},
		{"one update", [][]byte{{0x00, 0x01, 0x02}}},
		{"multiple updates", [][]byte{
			{0x00, 0x01},
			{0x01, 0xFF, 0xFE, 0xFD},
			{0x00},
		}},
		{"large update", [][]byte{bytes.Repeat([]byte{0xAA}, 1024)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeUpdates(tt.updates)
			decoded, err := DecodeUpdates(encoded)
			if err != nil {
				t.Fatalf("DecodeUpdates error: %v", err)
			}
			if len(tt.updates) == 0 {
				if decoded != nil {
					t.Fatalf("expected nil, got %d updates", len(decoded))
				}
				return
			}
			if len(decoded) != len(tt.updates) {
				t.Fatalf("decoded %d updates, want %d", len(decoded), len(tt.updates))
			}
			for i := range decoded {
				if !bytes.Equal(decoded[i], tt.updates[i]) {
					t.Errorf("update[%d] mismatch", i)
				}
			}
		})
	}
}

func TestDecodeUpdates_NilAndEmpty(t *testing.T) {
	for _, input := range [][]byte{nil, {}} {
		updates, err := DecodeUpdates(input)
		if err != nil {
			t.Errorf("DecodeUpdates(%v) error: %v", input, err)
		}
		if updates != nil {
			t.Errorf("DecodeUpdates(%v) = %v, want nil", input, updates)
		}
	}
}

func TestDecodeUpdates_Truncated(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"too short for count", []byte{0x00, 0x01}},
		{"count=1 but no length", []byte{0x00, 0x00, 0x00, 0x01}},
		{"count=1 length=5 but only 2 bytes", []byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x05, 0xAA, 0xBB}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeUpdates(tt.data)
			if err == nil {
				t.Fatal("expected error for truncated data")
			}
		})
	}
}

// --- HandleMessage tests ---

func newTestRelay(s StateStore) *YjsRelay {
	return NewYjsRelay(s, 30*time.Second)
}

func TestHandleMessage_SyncRelayed(t *testing.T) {
	relay := newTestRelay(newMockStateStore())
	relay.mu.Lock()
	relay.rooms["board-1"] = &RelayRoom{boardID: "board-1", done: make(chan struct{})}
	relay.mu.Unlock()

	if !relay.HandleMessage("board-1", []byte{yjsMsgSync, 0x01, 0x02}) {
		t.Error("sync message should be relayed")
	}
}

func TestHandleMessage_AwarenessRelayed(t *testing.T) {
	relay := newTestRelay(newMockStateStore())
	relay.mu.Lock()
	relay.rooms["board-1"] = &RelayRoom{boardID: "board-1", done: make(chan struct{})}
	relay.mu.Unlock()

	if !relay.HandleMessage("board-1", []byte{yjsMsgAwareness, 0xAA}) {
		t.Error("awareness message should be relayed")
	}
}

func TestHandleMessage_UnknownDropped(t *testing.T) {
	relay := newTestRelay(newMockStateStore())

	for _, msgType := range []byte{2, 3, 0xFF} {
		if relay.HandleMessage("board-1", []byte{msgType, 0x01}) {
			t.Errorf("message type %d should be dropped", msgType)
		}
	}
}

func TestHandleMessage_EmptyDropped(t *testing.T) {
	relay := newTestRelay(newMockStateStore())

	if relay.HandleMessage("board-1", nil) {
		t.Error("nil message should be dropped")
	}
	if relay.HandleMessage("board-1", []byte{}) {
		t.Error("empty message should be dropped")
	}
}

func TestHandleMessage_SyncAccumulated(t *testing.T) {
	relay := newTestRelay(newMockStateStore())
	room := &RelayRoom{boardID: "board-1", done: make(chan struct{})}
	relay.mu.Lock()
	relay.rooms["board-1"] = room
	relay.mu.Unlock()

	syncMsg := []byte{yjsMsgSync, 0x01, 0x02}
	awarenessMsg := []byte{yjsMsgAwareness, 0xAA}

	relay.HandleMessage("board-1", syncMsg)
	relay.HandleMessage("board-1", awarenessMsg)
	relay.HandleMessage("board-1", syncMsg)

	room.mu.Lock()
	defer room.mu.Unlock()

	if len(room.updates) != 2 {
		t.Fatalf("expected 2 accumulated updates, got %d", len(room.updates))
	}
	for _, u := range room.updates {
		if u[0] != yjsMsgSync {
			t.Errorf("accumulated update type = %d, want %d", u[0], yjsMsgSync)
		}
	}
}

func TestHandleMessage_CopiesData(t *testing.T) {
	relay := newTestRelay(newMockStateStore())
	room := &RelayRoom{boardID: "board-1", done: make(chan struct{})}
	relay.mu.Lock()
	relay.rooms["board-1"] = room
	relay.mu.Unlock()

	original := []byte{yjsMsgSync, 0x01, 0x02}
	relay.HandleMessage("board-1", original)

	// Mutate original — stored copy should be unaffected.
	original[1] = 0xFF

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.updates[0][1] == 0xFF {
		t.Error("mutating original affected stored update — data was not copied")
	}
}

// --- OnFirstClientJoin tests ---

func TestOnFirstClientJoin_LoadsState(t *testing.T) {
	ms := newMockStateStore()
	original := [][]byte{
		{yjsMsgSync, 0x01, 0x02},
		{yjsMsgSync, 0x03, 0x04},
	}
	ms.state["board-1"] = EncodeUpdates(original)

	relay := newTestRelay(ms)
	msgs, err := relay.OnFirstClientJoin(context.Background(), "board-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	for i := range msgs {
		if !bytes.Equal(msgs[i], original[i]) {
			t.Errorf("message[%d] mismatch", i)
		}
	}

	// Room should exist and have auto-save running.
	relay.mu.Lock()
	room, ok := relay.rooms["board-1"]
	relay.mu.Unlock()
	if !ok {
		t.Fatal("room not created")
	}

	// Clean up.
	close(room.done)
	room.saveTicker.Stop()
}

func TestOnFirstClientJoin_NilState(t *testing.T) {
	ms := newMockStateStore()

	relay := newTestRelay(ms)
	msgs, err := relay.OnFirstClientJoin(context.Background(), "board-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}

	// Clean up.
	relay.mu.Lock()
	room := relay.rooms["board-1"]
	relay.mu.Unlock()
	close(room.done)
	room.saveTicker.Stop()
}

func TestOnFirstClientJoin_LoadError(t *testing.T) {
	ms := newMockStateStore()
	ms.loadErr = fmt.Errorf("db connection failed")

	relay := newTestRelay(ms)
	_, err := relay.OnFirstClientJoin(context.Background(), "board-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- OnClientJoin tests ---

func TestOnClientJoin_ReturnsAccumulatedUpdates(t *testing.T) {
	relay := newTestRelay(newMockStateStore())
	room := &RelayRoom{
		boardID: "board-1",
		updates: [][]byte{
			{yjsMsgSync, 0x01},
			{yjsMsgSync, 0x02},
		},
		done: make(chan struct{}),
	}
	relay.mu.Lock()
	relay.rooms["board-1"] = room
	relay.mu.Unlock()

	copies := relay.OnClientJoin("board-1")
	if len(copies) != 2 {
		t.Fatalf("expected 2 copies, got %d", len(copies))
	}

	// Verify they are actual copies, not the same slice.
	copies[0][1] = 0xFF
	room.mu.Lock()
	defer room.mu.Unlock()
	if room.updates[0][1] == 0xFF {
		t.Error("returned slice is not a copy")
	}
}

// --- OnLastClientLeave tests ---

func TestOnLastClientLeave_PersistsState(t *testing.T) {
	ms := newMockStateStore()
	relay := newTestRelay(ms)

	updates := [][]byte{
		{yjsMsgSync, 0x01},
		{yjsMsgSync, 0x02},
	}
	room := &RelayRoom{
		boardID:    "board-1",
		updates:    updates,
		saveTicker: time.NewTicker(time.Hour), // won't fire
		done:       make(chan struct{}),
	}
	relay.mu.Lock()
	relay.rooms["board-1"] = room
	relay.mu.Unlock()

	relay.OnLastClientLeave(context.Background(), "board-1")

	if ms.getSaveCnt() != 1 {
		t.Fatalf("expected 1 save call, got %d", ms.getSaveCnt())
	}

	// Verify saved data round-trips.
	decoded, err := DecodeUpdates(ms.state["board-1"])
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(decoded) != 2 {
		t.Fatalf("expected 2 decoded updates, got %d", len(decoded))
	}
}

func TestOnLastClientLeave_NoUpdates_NoSave(t *testing.T) {
	ms := newMockStateStore()
	relay := newTestRelay(ms)

	room := &RelayRoom{
		boardID:    "board-1",
		saveTicker: time.NewTicker(time.Hour),
		done:       make(chan struct{}),
	}
	relay.mu.Lock()
	relay.rooms["board-1"] = room
	relay.mu.Unlock()

	relay.OnLastClientLeave(context.Background(), "board-1")

	if ms.getSaveCnt() != 0 {
		t.Errorf("expected 0 save calls, got %d", ms.getSaveCnt())
	}
}

func TestOnLastClientLeave_CleansUpRoom(t *testing.T) {
	ms := newMockStateStore()
	relay := newTestRelay(ms)

	room := &RelayRoom{
		boardID:    "board-1",
		saveTicker: time.NewTicker(time.Hour),
		done:       make(chan struct{}),
	}
	relay.mu.Lock()
	relay.rooms["board-1"] = room
	relay.mu.Unlock()

	relay.OnLastClientLeave(context.Background(), "board-1")

	relay.mu.Lock()
	_, exists := relay.rooms["board-1"]
	relay.mu.Unlock()
	if exists {
		t.Error("room should be removed after last client leaves")
	}
}

// --- Auto-save tests ---

func TestAutoSave_TriggersOnInterval(t *testing.T) {
	ms := newMockStateStore()
	relay := NewYjsRelay(ms, 50*time.Millisecond)

	room := &RelayRoom{
		boardID:    "board-1",
		done:       make(chan struct{}),
		stopped:    make(chan struct{}),
		saveTicker: time.NewTicker(50 * time.Millisecond),
	}
	room.updates = [][]byte{{yjsMsgSync, 0x01}}
	relay.mu.Lock()
	relay.rooms["board-1"] = room
	relay.mu.Unlock()

	relay.startAutoSave(room)

	deadline := time.After(2 * time.Second)
	for ms.getSaveCnt() == 0 {
		select {
		case <-deadline:
			t.Fatal("auto-save did not trigger within deadline")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	close(room.done)
	room.saveTicker.Stop()
}

func TestAutoSave_PreservesUpdatesAfterSave(t *testing.T) {
	ms := newMockStateStore()
	relay := NewYjsRelay(ms, 50*time.Millisecond)

	room := &RelayRoom{
		boardID:    "board-1",
		done:       make(chan struct{}),
		stopped:    make(chan struct{}),
		saveTicker: time.NewTicker(50 * time.Millisecond),
	}
	room.updates = [][]byte{{yjsMsgSync, 0x01}}
	relay.mu.Lock()
	relay.rooms["board-1"] = room
	relay.mu.Unlock()

	relay.startAutoSave(room)

	deadline := time.After(2 * time.Second)
	for ms.getSaveCnt() == 0 {
		select {
		case <-deadline:
			t.Fatal("auto-save did not trigger within deadline")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	room.mu.Lock()
	updLen := len(room.updates)
	room.mu.Unlock()

	if updLen == 0 {
		t.Error("expected updates to be preserved after auto-save, got 0")
	}

	close(room.done)
	room.saveTicker.Stop()
}

// --- saveRoom error path tests ---

func TestSaveRoom_ErrorKeepsUpdatesForRetry(t *testing.T) {
	ms := newMockStateStore()
	ms.saveErr = fmt.Errorf("db write failed")

	relay := NewYjsRelay(ms, time.Hour)
	room := &RelayRoom{
		boardID: "board-1",
		updates: [][]byte{
			{yjsMsgSync, 0x01},
			{yjsMsgSync, 0x02},
		},
		done: make(chan struct{}),
	}
	relay.mu.Lock()
	relay.rooms["board-1"] = room
	relay.mu.Unlock()

	// Auto-save (final=false) should keep updates on error for retry.
	relay.saveRoom(context.Background(), room, false)

	room.mu.Lock()
	updLen := len(room.updates)
	room.mu.Unlock()

	if updLen != 2 {
		t.Errorf("expected 2 updates preserved after failed auto-save, got %d", updLen)
	}

	if ms.getSaveCnt() != 0 {
		t.Errorf("expected 0 successful saves (all should fail), got %d", ms.getSaveCnt())
	}
}

func TestSaveRoom_FinalSaveClearsUpdates(t *testing.T) {
	ms := newMockStateStore()

	relay := NewYjsRelay(ms, time.Hour)
	room := &RelayRoom{
		boardID: "board-1",
		updates: [][]byte{
			{yjsMsgSync, 0x01},
		},
		done: make(chan struct{}),
	}

	// Final save (final=true) should clear updates even on success.
	relay.saveRoom(context.Background(), room, true)

	room.mu.Lock()
	updLen := len(room.updates)
	room.mu.Unlock()

	if updLen != 0 {
		t.Errorf("expected 0 updates after final save, got %d", updLen)
	}

	if ms.getSaveCnt() != 1 {
		t.Errorf("expected 1 save, got %d", ms.getSaveCnt())
	}
}

func TestSaveRoom_FinalSaveErrorClearsUpdates(t *testing.T) {
	ms := newMockStateStore()
	ms.saveErr = fmt.Errorf("db write failed")

	relay := NewYjsRelay(ms, time.Hour)
	room := &RelayRoom{
		boardID: "board-1",
		updates: [][]byte{
			{yjsMsgSync, 0x01},
		},
		done: make(chan struct{}),
	}

	// Final save clears updates even on error — room is being torn down.
	relay.saveRoom(context.Background(), room, true)

	room.mu.Lock()
	updLen := len(room.updates)
	room.mu.Unlock()

	if updLen != 0 {
		t.Errorf("expected 0 updates after final save (even on error), got %d", updLen)
	}
}

// --- Binary passthrough test ---

func TestBinaryPassthrough(t *testing.T) {
	relay := newTestRelay(newMockStateStore())
	room := &RelayRoom{boardID: "board-1", done: make(chan struct{})}
	relay.mu.Lock()
	relay.rooms["board-1"] = room
	relay.mu.Unlock()

	// A realistic Yjs sync message.
	msg := make([]byte, 256)
	msg[0] = yjsMsgSync
	for i := 1; i < len(msg); i++ {
		msg[i] = byte(i % 256)
	}

	relay.HandleMessage("board-1", msg)

	room.mu.Lock()
	stored := room.updates[0]
	room.mu.Unlock()

	if !bytes.Equal(stored, msg) {
		t.Error("stored message is not byte-identical to input")
	}
}

// --- Full persistence cycle tests ---

func TestRelay_FullPersistenceCycle(t *testing.T) {
	ms := newMockStateStore()
	relay := newTestRelay(ms) // 30s interval, won't tick during test

	ctx := context.Background()
	boardID := "board-full-cycle"

	// 1. First client joins a new board — no persisted state.
	msgs, err := relay.OnFirstClientJoin(ctx, boardID)
	if err != nil {
		t.Fatalf("OnFirstClientJoin (first): %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages on fresh board, got %d", len(msgs))
	}

	// 2. Accumulate sync messages.
	syncMsg1 := []byte{yjsMsgSync, 0x01, 0x02}
	syncMsg2 := []byte{yjsMsgSync, 0x03, 0x04, 0x05}

	relay.HandleMessage(boardID, syncMsg1)
	relay.HandleMessage(boardID, syncMsg2)

	// 3. Last client leaves — triggers final save.
	relay.OnLastClientLeave(ctx, boardID)

	// 4. Verify the mock store received saved data.
	if ms.getSaveCnt() != 1 {
		t.Fatalf("expected 1 save, got %d", ms.getSaveCnt())
	}
	ms.mu.Lock()
	savedData := ms.state[boardID]
	ms.mu.Unlock()
	if len(savedData) == 0 {
		t.Fatal("expected non-empty saved state")
	}

	// 5. First client joins again — should load persisted state.
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

	// 6. Latecomer joins — also gets the same messages.
	copies := relay.OnClientJoin(boardID)
	if len(copies) != 2 {
		t.Fatalf("latecomer expected 2 messages, got %d", len(copies))
	}
	if !bytes.Equal(copies[0], syncMsg1) {
		t.Errorf("latecomer msg[0] = %v, want %v", copies[0], syncMsg1)
	}
	if !bytes.Equal(copies[1], syncMsg2) {
		t.Errorf("latecomer msg[1] = %v, want %v", copies[1], syncMsg2)
	}

	// Clean up.
	relay.OnLastClientLeave(ctx, boardID)
}

func TestRelay_EmptyBoard_FullCycle(t *testing.T) {
	ms := newMockStateStore()
	relay := newTestRelay(ms)

	ctx := context.Background()
	boardID := "board-empty-cycle"

	// 1. First client joins — new board, no state.
	msgs, err := relay.OnFirstClientJoin(ctx, boardID)
	if err != nil {
		t.Fatalf("OnFirstClientJoin (first): %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(msgs))
	}

	// 2. No HandleMessage calls — client leaves immediately.
	relay.OnLastClientLeave(ctx, boardID)

	// 3. No save should happen (empty updates).
	if ms.getSaveCnt() != 0 {
		t.Errorf("expected 0 saves for empty board, got %d", ms.getSaveCnt())
	}

	// 4. First client joins again — still empty.
	msgs, err = relay.OnFirstClientJoin(ctx, boardID)
	if err != nil {
		t.Fatalf("OnFirstClientJoin (second): %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages after empty cycle, got %d", len(msgs))
	}

	// Clean up.
	relay.OnLastClientLeave(ctx, boardID)
}
