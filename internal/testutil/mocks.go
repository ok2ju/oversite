package testutil

import (
	"context"
	"fmt"
	"sync"
)

// --- Keyring ---

// Keyring abstracts OS keychain operations (Set, Get, Delete).
// The real implementation wraps zalando/go-keyring; this interface allows
// tests to avoid touching the real OS keychain.
type Keyring interface {
	Set(service, user, password string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

// ErrKeyNotFound is returned when no secret exists for the given service+user.
var ErrKeyNotFound = fmt.Errorf("secret not found in keyring")

// MockKeyring is an in-memory Keyring implementation for tests.
type MockKeyring struct {
	mu   sync.RWMutex
	data map[string]string // key = "service\x00user"
}

// NewMockKeyring returns a ready-to-use in-memory keyring.
func NewMockKeyring() *MockKeyring {
	return &MockKeyring{data: make(map[string]string)}
}

func keyringKey(service, user string) string { return service + "\x00" + user }

func (m *MockKeyring) Set(service, user, password string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[keyringKey(service, user)] = password
	return nil
}

func (m *MockKeyring) Get(service, user string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[keyringKey(service, user)]
	if !ok {
		return "", ErrKeyNotFound
	}
	return v, nil
}

func (m *MockKeyring) Delete(service, user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := keyringKey(service, user)
	if _, ok := m.data[k]; !ok {
		return ErrKeyNotFound
	}
	delete(m.data, k)
	return nil
}

// --- Faceit API ---

// FaceitPlayer represents a Faceit player profile.
type FaceitPlayer struct {
	PlayerID   string
	Nickname   string
	Avatar     string
	Country    string
	SkillLevel int
	FaceitElo  int
}

// FaceitMatchHistory is a paginated match history response.
type FaceitMatchHistory struct {
	Items []FaceitMatchSummary
}

// FaceitMatchSummary is a single match entry.
type FaceitMatchSummary struct {
	MatchID    string
	Map        string
	StartedAt  int64
	FinishedAt int64
	Winner     string
	Score      map[string]int
}

// FaceitMatchDetails holds full match information.
type FaceitMatchDetails struct {
	MatchID    string
	Map        string
	DemoURL    []string
	StartedAt  int64
	FinishedAt int64
}

// FaceitClient abstracts the Faceit Data API.
// The real implementation will live in internal/faceit; this interface is
// defined here so test mocks can be written before that package exists.
type FaceitClient interface {
	GetPlayer(ctx context.Context, playerID string) (*FaceitPlayer, error)
	GetPlayerHistory(ctx context.Context, playerID string, offset, limit int) (*FaceitMatchHistory, error)
	GetMatchDetails(ctx context.Context, matchID string) (*FaceitMatchDetails, error)
}

// MockFaceitClient is a configurable stub for the Faceit API.
// Set the exported fields to control what each method returns.
type MockFaceitClient struct {
	GetPlayerFn        func(ctx context.Context, playerID string) (*FaceitPlayer, error)
	GetPlayerHistoryFn func(ctx context.Context, playerID string, offset, limit int) (*FaceitMatchHistory, error)
	GetMatchDetailsFn  func(ctx context.Context, matchID string) (*FaceitMatchDetails, error)
}

func (m *MockFaceitClient) GetPlayer(ctx context.Context, playerID string) (*FaceitPlayer, error) {
	if m.GetPlayerFn != nil {
		return m.GetPlayerFn(ctx, playerID)
	}
	return nil, nil
}

func (m *MockFaceitClient) GetPlayerHistory(ctx context.Context, playerID string, offset, limit int) (*FaceitMatchHistory, error) {
	if m.GetPlayerHistoryFn != nil {
		return m.GetPlayerHistoryFn(ctx, playerID, offset, limit)
	}
	return &FaceitMatchHistory{}, nil
}

func (m *MockFaceitClient) GetMatchDetails(ctx context.Context, matchID string) (*FaceitMatchDetails, error) {
	if m.GetMatchDetailsFn != nil {
		return m.GetMatchDetailsFn(ctx, matchID)
	}
	return nil, nil
}
