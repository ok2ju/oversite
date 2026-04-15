package auth

import (
	"errors"

	"github.com/ok2ju/oversite/internal/testutil"
	"github.com/zalando/go-keyring"
)

// RealKeyring implements testutil.Keyring using the OS keychain
// via github.com/zalando/go-keyring.
type RealKeyring struct{}

// Set stores a secret in the OS keychain.
func (r *RealKeyring) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

// Get retrieves a secret from the OS keychain.
// Returns testutil.ErrKeyNotFound if the key does not exist.
func (r *RealKeyring) Get(service, user string) (string, error) {
	s, err := keyring.Get(service, user)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", testutil.ErrKeyNotFound
	}
	return s, err
}

// Delete removes a secret from the OS keychain.
// Returns testutil.ErrKeyNotFound if the key does not exist.
func (r *RealKeyring) Delete(service, user string) error {
	err := keyring.Delete(service, user)
	if errors.Is(err, keyring.ErrNotFound) {
		return testutil.ErrKeyNotFound
	}
	return err
}
