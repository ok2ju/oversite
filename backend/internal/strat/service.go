package strat

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/store"
)

// Sentinel errors for the strat service.
var (
	ErrNotFound  = errors.New("strategy board not found")
	ErrForbidden = errors.New("not authorized to modify this board")
)

// Store is the subset of store.Queries needed by Service.
type Store interface {
	CreateStrategyBoard(ctx context.Context, arg store.CreateStrategyBoardParams) (store.StrategyBoard, error)
	GetStrategyBoardByID(ctx context.Context, id uuid.UUID) (store.StrategyBoard, error)
	ListStrategyBoardsByUserID(ctx context.Context, userID uuid.UUID) ([]store.StrategyBoard, error)
	UpdateStrategyBoard(ctx context.Context, arg store.UpdateStrategyBoardParams) (store.StrategyBoard, error)
	DeleteStrategyBoard(ctx context.Context, id uuid.UUID) error
}

// Service provides strategy board CRUD operations with authorization.
type Service struct {
	store Store
}

// NewService creates a new strat Service.
func NewService(s Store) *Service {
	return &Service{store: s}
}

// Create creates a new strategy board for the given user.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, title, mapName string) (store.StrategyBoard, error) {
	return s.store.CreateStrategyBoard(ctx, store.CreateStrategyBoardParams{
		UserID:  userID,
		Title:   title,
		MapName: mapName,
	})
}

// Get returns a strategy board by ID. Returns ErrNotFound if it doesn't exist.
func (s *Service) Get(ctx context.Context, boardID uuid.UUID) (store.StrategyBoard, error) {
	board, err := s.store.GetStrategyBoardByID(ctx, boardID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.StrategyBoard{}, ErrNotFound
		}
		return store.StrategyBoard{}, err
	}
	return board, nil
}

// List returns all strategy boards for the given user.
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]store.StrategyBoard, error) {
	boards, err := s.store.ListStrategyBoardsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if boards == nil {
		return []store.StrategyBoard{}, nil
	}
	return boards, nil
}

// Update modifies a strategy board. Returns ErrNotFound if missing,
// ErrForbidden if the caller is not the owner.
func (s *Service) Update(ctx context.Context, userID, boardID uuid.UUID, title, mapName, shareMode string) (store.StrategyBoard, error) {
	board, err := s.store.GetStrategyBoardByID(ctx, boardID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.StrategyBoard{}, ErrNotFound
		}
		return store.StrategyBoard{}, err
	}

	if board.UserID != userID {
		return store.StrategyBoard{}, ErrForbidden
	}

	return s.store.UpdateStrategyBoard(ctx, store.UpdateStrategyBoardParams{
		ID:        boardID,
		Title:     title,
		MapName:   mapName,
		ShareMode: shareMode,
	})
}

// Delete removes a strategy board. Returns ErrNotFound if missing,
// ErrForbidden if the caller is not the owner.
func (s *Service) Delete(ctx context.Context, userID, boardID uuid.UUID) error {
	board, err := s.store.GetStrategyBoardByID(ctx, boardID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if board.UserID != userID {
		return ErrForbidden
	}

	return s.store.DeleteStrategyBoard(ctx, boardID)
}
