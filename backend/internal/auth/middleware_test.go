package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ok2ju/oversite/backend/internal/auth"
)

// --- Test mock ---

type mockSessionStore struct {
	sessions   map[string]*auth.SessionData
	refreshed  []string
	refreshErr error
	getErr     error
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{sessions: make(map[string]*auth.SessionData)}
}

func (m *mockSessionStore) Create(_ context.Context, data *auth.SessionData) (string, error) {
	return "mock-token", nil
}

func (m *mockSessionStore) Get(_ context.Context, token string) (*auth.SessionData, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	d, ok := m.sessions[token]
	if !ok {
		return nil, auth.ErrSessionNotFound
	}
	return d, nil
}

func (m *mockSessionStore) Delete(_ context.Context, token string) error {
	delete(m.sessions, token)
	return nil
}

func (m *mockSessionStore) Refresh(_ context.Context, token string) error {
	if m.refreshErr != nil {
		return m.refreshErr
	}
	m.refreshed = append(m.refreshed, token)
	return nil
}

// --- Helper to decode JSON error response ---

func decodeErrorResponse(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding error response: %v", err)
	}
	return body["error"]
}

// dummyHandler is a next handler that records whether it was called
// and captures the userID from context.
func dummyHandler() (http.HandlerFunc, *bool, *string) {
	called := false
	var capturedUserID string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if uid, ok := auth.UserIDFromContext(r.Context()); ok {
			capturedUserID = uid
		}
		w.WriteHeader(http.StatusOK)
	})
	return h, &called, &capturedUserID
}

// --- Tests ---

func TestRequireAuth_NoCookie_Returns401(t *testing.T) {
	store := newMockSessionStore()
	mw := auth.RequireAuth(store)

	next, called, _ := dummyHandler()
	handler := mw(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
	if *called {
		t.Error("next handler should not have been called")
	}

	errMsg := decodeErrorResponse(t, rec)
	if errMsg != "unauthorized" {
		t.Errorf("expected error 'unauthorized', got %q", errMsg)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

func TestRequireAuth_InvalidToken_Returns401(t *testing.T) {
	store := newMockSessionStore()
	// No sessions stored — any token is invalid
	mw := auth.RequireAuth(store)

	next, called, _ := dummyHandler()
	handler := mw(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "bad-token"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
	if *called {
		t.Error("next handler should not have been called")
	}

	errMsg := decodeErrorResponse(t, rec)
	if errMsg != "unauthorized" {
		t.Errorf("expected error 'unauthorized', got %q", errMsg)
	}
}

func TestRequireAuth_StoreError_Returns401(t *testing.T) {
	store := newMockSessionStore()
	store.getErr = errors.New("redis connection failed")
	mw := auth.RequireAuth(store)

	next, called, _ := dummyHandler()
	handler := mw(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "some-token"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
	if *called {
		t.Error("next handler should not have been called")
	}
}

func TestRequireAuth_ValidSession_CallsNext(t *testing.T) {
	store := newMockSessionStore()
	store.sessions["valid-token"] = &auth.SessionData{
		UserID:   "user-123",
		FaceitID: "faceit-456",
		Nickname: "player1",
	}
	mw := auth.RequireAuth(store)

	next, called, capturedUserID := dummyHandler()
	handler := mw(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-token"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !*called {
		t.Error("next handler should have been called")
	}
	if *capturedUserID != "user-123" {
		t.Errorf("expected userID 'user-123' in context, got %q", *capturedUserID)
	}
}

func TestRequireAuth_ValidSession_RefreshCalled(t *testing.T) {
	store := newMockSessionStore()
	store.sessions["valid-token"] = &auth.SessionData{
		UserID:   "user-123",
		FaceitID: "faceit-456",
		Nickname: "player1",
	}
	mw := auth.RequireAuth(store)

	next, _, _ := dummyHandler()
	handler := mw(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-token"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if len(store.refreshed) != 1 {
		t.Fatalf("expected Refresh to be called once, called %d times", len(store.refreshed))
	}
	if store.refreshed[0] != "valid-token" {
		t.Errorf("expected Refresh called with 'valid-token', got %q", store.refreshed[0])
	}
}

func TestRequireAuth_RefreshError_StillCallsNext(t *testing.T) {
	store := newMockSessionStore()
	store.sessions["valid-token"] = &auth.SessionData{
		UserID:   "user-123",
		FaceitID: "faceit-456",
		Nickname: "player1",
	}
	store.refreshErr = errors.New("refresh failed")
	mw := auth.RequireAuth(store)

	next, called, _ := dummyHandler()
	handler := mw(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-token"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Refresh failure should not block the request
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !*called {
		t.Error("next handler should still be called even if refresh fails")
	}
}

func TestUserIDFromContext_NotSet(t *testing.T) {
	ctx := context.Background()
	uid, ok := auth.UserIDFromContext(ctx)
	if ok {
		t.Error("expected ok to be false for empty context")
	}
	if uid != "" {
		t.Errorf("expected empty userID, got %q", uid)
	}
}

func TestUserIDFromContext_Set(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.UserIDKey, "user-abc")
	uid, ok := auth.UserIDFromContext(ctx)
	if !ok {
		t.Error("expected ok to be true")
	}
	if uid != "user-abc" {
		t.Errorf("expected 'user-abc', got %q", uid)
	}
}
