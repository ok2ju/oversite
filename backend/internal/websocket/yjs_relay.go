package websocket

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/store"
)

// Yjs message type bytes (first byte of each binary message).
const (
	yjsMsgSync      byte = 0 // Yjs sync protocol
	yjsMsgAwareness byte = 1 // Yjs awareness protocol
)

// StateStore abstracts persistence of Yjs document state.
type StateStore interface {
	LoadYjsState(ctx context.Context, boardID string) ([]byte, error)
	SaveYjsState(ctx context.Context, boardID string, state []byte) error
}

// PgStateStore implements StateStore using sqlc-generated Queries.
type PgStateStore struct {
	q *store.Queries
}

// NewPgStateStore creates a PgStateStore wrapping the given Queries.
func NewPgStateStore(q *store.Queries) *PgStateStore {
	return &PgStateStore{q: q}
}

func (s *PgStateStore) LoadYjsState(ctx context.Context, boardID string) ([]byte, error) {
	id, err := uuid.Parse(boardID)
	if err != nil {
		return nil, fmt.Errorf("invalid board ID %q: %w", boardID, err)
	}

	board, err := s.q.GetStrategyBoardByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return board.YjsState, nil
}

func (s *PgStateStore) SaveYjsState(ctx context.Context, boardID string, state []byte) error {
	id, err := uuid.Parse(boardID)
	if err != nil {
		return fmt.Errorf("invalid board ID %q: %w", boardID, err)
	}

	return s.q.UpdateStrategyBoardYjsState(ctx, store.UpdateStrategyBoardYjsStateParams{
		ID:       id,
		YjsState: state,
	})
}

// EncodeUpdates packs a slice of raw Yjs messages into a binary format:
//
//	[uint32 BE count][uint32 BE len1][msg1 bytes][uint32 BE len2][msg2 bytes]...
func EncodeUpdates(updates [][]byte) []byte {
	if len(updates) == 0 {
		return nil
	}

	size := 4 // count header
	for _, u := range updates {
		size += 4 + len(u)
	}

	buf := make([]byte, size)
	binary.BigEndian.PutUint32(buf[0:4], uint32(len(updates)))
	offset := 4
	for _, u := range updates {
		binary.BigEndian.PutUint32(buf[offset:offset+4], uint32(len(u)))
		offset += 4
		copy(buf[offset:], u)
		offset += len(u)
	}
	return buf
}

// DecodeUpdates unpacks binary data into individual Yjs messages.
// Returns nil, nil for nil or empty input.
func DecodeUpdates(data []byte) ([][]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	if len(data) < 4 {
		return nil, fmt.Errorf("truncated data: need at least 4 bytes for count, got %d", len(data))
	}

	count := binary.BigEndian.Uint32(data[0:4])
	offset := 4
	updates := make([][]byte, 0, count)

	for i := uint32(0); i < count; i++ {
		if offset+4 > len(data) {
			return nil, fmt.Errorf("truncated data at message %d: need length header at offset %d", i, offset)
		}
		msgLen := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4
		if offset+msgLen > len(data) {
			return nil, fmt.Errorf("truncated data at message %d: need %d bytes at offset %d, have %d", i, msgLen, offset, len(data)-offset)
		}
		msg := make([]byte, msgLen)
		copy(msg, data[offset:offset+msgLen])
		offset += msgLen
		updates = append(updates, msg)
	}
	return updates, nil
}

// RelayRoom holds per-room state for accumulated Yjs sync messages.
type RelayRoom struct {
	boardID    string
	updates    [][]byte
	mu         sync.Mutex
	saveTicker *time.Ticker
	done       chan struct{}
	stopped    chan struct{} // closed when auto-save goroutine exits
}

// YjsRelay manages Yjs message routing and state persistence across rooms.
// It remains a "dumb relay" — it reads only the first byte (message type)
// for routing but never parses Yjs CRDT content.
type YjsRelay struct {
	store        StateStore
	rooms        map[string]*RelayRoom
	mu           sync.Mutex
	saveInterval time.Duration
}

// NewYjsRelay creates a YjsRelay with the given state store and auto-save interval.
func NewYjsRelay(store StateStore, saveInterval time.Duration) *YjsRelay {
	return &YjsRelay{
		store:        store,
		rooms:        make(map[string]*RelayRoom),
		saveInterval: saveInterval,
	}
}

// OnFirstClientJoin loads persisted state from the DB, starts the auto-save
// ticker, and returns the decoded messages to send to the joining client.
func (r *YjsRelay) OnFirstClientJoin(ctx context.Context, boardID string) ([][]byte, error) {
	data, err := r.store.LoadYjsState(ctx, boardID)
	if err != nil {
		return nil, fmt.Errorf("loading yjs state for board %s: %w", boardID, err)
	}

	var messages [][]byte
	if len(data) > 0 {
		messages, err = DecodeUpdates(data)
		if err != nil {
			return nil, fmt.Errorf("decoding yjs state for board %s: %w", boardID, err)
		}
	}

	room := &RelayRoom{
		boardID:    boardID,
		done:       make(chan struct{}),
		stopped:    make(chan struct{}),
		saveTicker: time.NewTicker(r.saveInterval),
	}
	if messages != nil {
		room.updates = make([][]byte, len(messages))
		copy(room.updates, messages)
	}

	r.mu.Lock()
	if old, exists := r.rooms[boardID]; exists {
		close(old.done)
		if old.saveTicker != nil {
			old.saveTicker.Stop()
		}
	}
	r.rooms[boardID] = room
	r.mu.Unlock()

	r.startAutoSave(room)

	return messages, nil
}

// OnClientJoin returns copies of accumulated sync messages for client catch-up.
func (r *YjsRelay) OnClientJoin(boardID string) [][]byte {
	r.mu.Lock()
	room, ok := r.rooms[boardID]
	r.mu.Unlock()
	if !ok {
		return nil
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if len(room.updates) == 0 {
		return nil
	}

	copies := make([][]byte, len(room.updates))
	for i, u := range room.updates {
		cp := make([]byte, len(u))
		copy(cp, u)
		copies[i] = cp
	}
	return copies
}

// HandleMessage inspects the first byte of a Yjs binary message for routing.
// Sync messages (type 0) are relayed and accumulated. Awareness messages
// (type 1) are relayed but not accumulated. Unknown types are dropped.
func (r *YjsRelay) HandleMessage(boardID string, data []byte) (shouldRelay bool) {
	if len(data) == 0 {
		return false
	}

	msgType := data[0]
	switch msgType {
	case yjsMsgSync:
		// Copy data since WS library may reuse buffers.
		cp := make([]byte, len(data))
		copy(cp, data)

		r.mu.Lock()
		room, ok := r.rooms[boardID]
		r.mu.Unlock()

		if ok {
			room.mu.Lock()
			room.updates = append(room.updates, cp)
			room.mu.Unlock()
		}
		return true

	case yjsMsgAwareness:
		return true

	default:
		return false
	}
}

// OnLastClientLeave stops the auto-save ticker, waits for it to exit,
// persists state to the DB, and removes the room.
func (r *YjsRelay) OnLastClientLeave(ctx context.Context, boardID string) {
	r.mu.Lock()
	room, ok := r.rooms[boardID]
	if !ok {
		r.mu.Unlock()
		return
	}
	delete(r.rooms, boardID)
	r.mu.Unlock()

	// Stop auto-save goroutine and wait for it to fully exit
	// before the final save, preventing concurrent saveRoom calls.
	close(room.done)
	if room.saveTicker != nil {
		room.saveTicker.Stop()
	}
	if room.stopped != nil {
		<-room.stopped
	}

	r.saveRoom(ctx, room, true)
}

// saveRoom encodes the full accumulated updates and persists them to the DB.
// When final is true (last client leaving), the in-memory updates are released.
// When final is false (periodic auto-save), updates are kept in memory so the
// DB always receives the complete state on each write.
// It is a no-op when the updates slice is empty.
func (r *YjsRelay) saveRoom(ctx context.Context, room *RelayRoom, final bool) {
	room.mu.Lock()
	if len(room.updates) == 0 {
		room.mu.Unlock()
		return
	}
	encoded := EncodeUpdates(room.updates)
	if final {
		room.updates = nil
	}
	room.mu.Unlock()

	if err := r.store.SaveYjsState(ctx, room.boardID, encoded); err != nil {
		// No request_id available: saveRoom runs in a background goroutine
		// outside any HTTP request context.
		slog.Error("failed to save yjs state", "board_id", room.boardID, "error", err)
	}
}

// startAutoSave launches a goroutine that periodically persists room state.
// The room's saveTicker and stopped channel must be initialized before calling.
func (r *YjsRelay) startAutoSave(room *RelayRoom) {
	go func() {
		defer close(room.stopped)
		for {
			select {
			case <-room.done:
				return
			case <-room.saveTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				r.saveRoom(ctx, room, false)
				cancel()
			}
		}
	}()
}
