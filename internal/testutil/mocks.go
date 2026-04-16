package testutil

import (
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
