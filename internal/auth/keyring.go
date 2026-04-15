package auth

import (
	"errors"

	"github.com/ok2ju/oversite/internal/testutil"
)

const (
	// ServiceName is the OS keychain service identifier for Oversite.
	ServiceName     = "oversite-faceit-auth"
	keyRefreshToken = "refresh-token"
	keyUserID       = "user-id"
)

// TokenStore wraps a Keyring to manage Faceit OAuth tokens in the OS keychain.
type TokenStore struct {
	kr      testutil.Keyring
	service string
}

// NewTokenStore returns a TokenStore backed by the given Keyring implementation.
func NewTokenStore(kr testutil.Keyring) *TokenStore {
	return &TokenStore{kr: kr, service: ServiceName}
}

// SaveRefreshToken persists the OAuth refresh token.
func (ts *TokenStore) SaveRefreshToken(token string) error {
	return ts.kr.Set(ts.service, keyRefreshToken, token)
}

// GetRefreshToken retrieves the stored OAuth refresh token.
func (ts *TokenStore) GetRefreshToken() (string, error) {
	return ts.kr.Get(ts.service, keyRefreshToken)
}

// DeleteRefreshToken removes the stored OAuth refresh token.
func (ts *TokenStore) DeleteRefreshToken() error {
	return ts.kr.Delete(ts.service, keyRefreshToken)
}

// SaveUserID persists the Faceit user ID.
func (ts *TokenStore) SaveUserID(id string) error {
	return ts.kr.Set(ts.service, keyUserID, id)
}

// GetUserID retrieves the stored Faceit user ID.
func (ts *TokenStore) GetUserID() (string, error) {
	return ts.kr.Get(ts.service, keyUserID)
}

// Clear removes both the refresh token and user ID from the keychain.
// It ignores ErrKeyNotFound for either key so calling Clear on an empty
// store is not an error.
func (ts *TokenStore) Clear() error {
	if err := ts.kr.Delete(ts.service, keyRefreshToken); err != nil && !errors.Is(err, testutil.ErrKeyNotFound) {
		return err
	}
	if err := ts.kr.Delete(ts.service, keyUserID); err != nil && !errors.Is(err, testutil.ErrKeyNotFound) {
		return err
	}
	return nil
}
