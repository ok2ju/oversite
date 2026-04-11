package strat

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/store"
)

// --- mock Store ---

type mockStore struct {
	boards map[uuid.UUID]store.StrategyBoard
	err    error
}

func newMockStore() *mockStore {
	return &mockStore{boards: make(map[uuid.UUID]store.StrategyBoard)}
}

func (m *mockStore) CreateStrategyBoard(_ context.Context, arg store.CreateStrategyBoardParams) (store.StrategyBoard, error) {
	if m.err != nil {
		return store.StrategyBoard{}, m.err
	}
	b := store.StrategyBoard{
		ID:        uuid.New(),
		UserID:    arg.UserID,
		Title:     arg.Title,
		MapName:   arg.MapName,
		ShareMode: "private",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.boards[b.ID] = b
	return b, nil
}

func (m *mockStore) GetStrategyBoardByID(_ context.Context, id uuid.UUID) (store.StrategyBoard, error) {
	if m.err != nil {
		return store.StrategyBoard{}, m.err
	}
	b, ok := m.boards[id]
	if !ok {
		return store.StrategyBoard{}, sql.ErrNoRows
	}
	return b, nil
}

func (m *mockStore) ListStrategyBoardsByUserID(_ context.Context, userID uuid.UUID) ([]store.StrategyBoard, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []store.StrategyBoard
	for _, b := range m.boards {
		if b.UserID == userID {
			result = append(result, b)
		}
	}
	return result, nil
}

func (m *mockStore) UpdateStrategyBoard(_ context.Context, arg store.UpdateStrategyBoardParams) (store.StrategyBoard, error) {
	if m.err != nil {
		return store.StrategyBoard{}, m.err
	}
	b, ok := m.boards[arg.ID]
	if !ok {
		return store.StrategyBoard{}, sql.ErrNoRows
	}
	b.Title = arg.Title
	b.MapName = arg.MapName
	b.ShareMode = arg.ShareMode
	b.UpdatedAt = time.Now()
	m.boards[arg.ID] = b
	return b, nil
}

func (m *mockStore) DeleteStrategyBoard(_ context.Context, id uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.boards[id]; !ok {
		return sql.ErrNoRows
	}
	delete(m.boards, id)
	return nil
}

// seed adds a board to the mock store and returns it.
func (m *mockStore) seed(userID uuid.UUID, title, mapName string) store.StrategyBoard {
	b := store.StrategyBoard{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     title,
		MapName:   mapName,
		ShareMode: "private",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.boards[b.ID] = b
	return b
}

// --- Tests ---

func TestService_Create(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr bool
	}{
		{"success", nil, false},
		{"store error", sql.ErrConnDone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := newMockStore()
			ms.err = tt.err
			svc := NewService(ms)

			board, err := svc.Create(context.Background(), uuid.New(), "Test Board", "de_dust2")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if board.Title != "Test Board" {
				t.Errorf("title = %q, want %q", board.Title, "Test Board")
			}
			if board.MapName != "de_dust2" {
				t.Errorf("map_name = %q, want %q", board.MapName, "de_dust2")
			}
		})
	}
}

func TestService_Get(t *testing.T) {
	ms := newMockStore()
	userID := uuid.New()
	board := ms.seed(userID, "My Board", "de_mirage")
	svc := NewService(ms)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{"found", board.ID, nil},
		{"not found", uuid.New(), ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.Get(context.Background(), tt.id)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err != tt.wantErr {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != board.ID {
				t.Errorf("id = %v, want %v", got.ID, board.ID)
			}
		})
	}
}

func TestService_List(t *testing.T) {
	ms := newMockStore()
	userID := uuid.New()
	otherUserID := uuid.New()
	ms.seed(userID, "Board 1", "de_dust2")
	ms.seed(userID, "Board 2", "de_mirage")
	ms.seed(otherUserID, "Other Board", "de_inferno")
	svc := NewService(ms)

	tests := []struct {
		name  string
		uid   uuid.UUID
		count int
	}{
		{"returns user boards", userID, 2},
		{"empty for unknown user", uuid.New(), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boards, err := svc.List(context.Background(), tt.uid)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(boards) != tt.count {
				t.Errorf("got %d boards, want %d", len(boards), tt.count)
			}
		})
	}
}

func TestService_Update(t *testing.T) {
	ms := newMockStore()
	ownerID := uuid.New()
	otherID := uuid.New()
	board := ms.seed(ownerID, "Original", "de_dust2")
	svc := NewService(ms)

	tests := []struct {
		name    string
		userID  uuid.UUID
		boardID uuid.UUID
		wantErr error
	}{
		{"success", ownerID, board.ID, nil},
		{"not found", ownerID, uuid.New(), ErrNotFound},
		{"forbidden", otherID, board.ID, ErrForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Update(context.Background(), tt.userID, tt.boardID, "Updated", "de_mirage", "team")
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err != tt.wantErr {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestService_Delete(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()

	tests := []struct {
		name    string
		userID  uuid.UUID
		setup   func(*mockStore) uuid.UUID
		wantErr error
	}{
		{
			"success",
			ownerID,
			func(ms *mockStore) uuid.UUID {
				return ms.seed(ownerID, "Delete Me", "de_dust2").ID
			},
			nil,
		},
		{
			"not found",
			ownerID,
			func(_ *mockStore) uuid.UUID { return uuid.New() },
			ErrNotFound,
		},
		{
			"forbidden",
			otherID,
			func(ms *mockStore) uuid.UUID {
				return ms.seed(ownerID, "Not Yours", "de_dust2").ID
			},
			ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := newMockStore()
			boardID := tt.setup(ms)
			svc := NewService(ms)

			err := svc.Delete(context.Background(), tt.userID, boardID)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err != tt.wantErr {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
