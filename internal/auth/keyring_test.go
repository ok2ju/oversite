package auth

import (
	"testing"

	"github.com/ok2ju/oversite/internal/testutil"
)

func TestTokenStore_RefreshTokenRoundTrip(t *testing.T) {
	ts := NewTokenStore(testutil.NewMockKeyring())

	const token = "rt_abc123"
	if err := ts.SaveRefreshToken(token); err != nil {
		t.Fatalf("SaveRefreshToken: %v", err)
	}

	got, err := ts.GetRefreshToken()
	if err != nil {
		t.Fatalf("GetRefreshToken: %v", err)
	}
	if got != token {
		t.Errorf("GetRefreshToken = %q, want %q", got, token)
	}
}

func TestTokenStore_GetRefreshToken_NotFound(t *testing.T) {
	ts := NewTokenStore(testutil.NewMockKeyring())

	_, err := ts.GetRefreshToken()
	if err != testutil.ErrKeyNotFound {
		t.Fatalf("GetRefreshToken on empty store: err = %v, want ErrKeyNotFound", err)
	}
}

func TestTokenStore_DeleteRefreshToken(t *testing.T) {
	ts := NewTokenStore(testutil.NewMockKeyring())

	if err := ts.SaveRefreshToken("to-be-deleted"); err != nil {
		t.Fatalf("SaveRefreshToken: %v", err)
	}
	if err := ts.DeleteRefreshToken(); err != nil {
		t.Fatalf("DeleteRefreshToken: %v", err)
	}

	_, err := ts.GetRefreshToken()
	if err != testutil.ErrKeyNotFound {
		t.Fatalf("GetRefreshToken after delete: err = %v, want ErrKeyNotFound", err)
	}
}

func TestTokenStore_UserIDRoundTrip(t *testing.T) {
	ts := NewTokenStore(testutil.NewMockKeyring())

	const userID = "faceit-user-42"
	if err := ts.SaveUserID(userID); err != nil {
		t.Fatalf("SaveUserID: %v", err)
	}

	got, err := ts.GetUserID()
	if err != nil {
		t.Fatalf("GetUserID: %v", err)
	}
	if got != userID {
		t.Errorf("GetUserID = %q, want %q", got, userID)
	}
}

func TestTokenStore_Clear_RemovesBothKeys(t *testing.T) {
	ts := NewTokenStore(testutil.NewMockKeyring())

	// Pre-populate both keys.
	if err := ts.SaveRefreshToken("rt_xyz"); err != nil {
		t.Fatalf("SaveRefreshToken: %v", err)
	}
	if err := ts.SaveUserID("user-99"); err != nil {
		t.Fatalf("SaveUserID: %v", err)
	}

	if err := ts.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	// Both keys should be gone.
	if _, err := ts.GetRefreshToken(); err != testutil.ErrKeyNotFound {
		t.Errorf("GetRefreshToken after Clear: err = %v, want ErrKeyNotFound", err)
	}
	if _, err := ts.GetUserID(); err != testutil.ErrKeyNotFound {
		t.Errorf("GetUserID after Clear: err = %v, want ErrKeyNotFound", err)
	}
}

func TestTokenStore_Clear_EmptyStoreNoError(t *testing.T) {
	ts := NewTokenStore(testutil.NewMockKeyring())

	if err := ts.Clear(); err != nil {
		t.Fatalf("Clear on empty store: %v", err)
	}
}
