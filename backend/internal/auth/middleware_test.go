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

func TestRequireAuth(t *testing.T) {
	validSession := &auth.SessionData{
		UserID:   "user-123",
		FaceitID: "faceit-456",
		Nickname: "player1",
	}

	tests := []struct {
		name           string
		cookie         *http.Cookie
		sessions       map[string]*auth.SessionData
		getErr         error
		refreshErr     error
		wantStatus     int
		wantNextCalled bool
		wantUserID     string
		wantError      string
		wantRefreshed  bool
	}{
		{
			name:           "no cookie returns 401",
			cookie:         nil,
			wantStatus:     http.StatusUnauthorized,
			wantNextCalled: false,
			wantError:      "unauthorized",
		},
		{
			name:           "invalid token returns 401",
			cookie:         &http.Cookie{Name: "session_token", Value: "bad-token"},
			sessions:       map[string]*auth.SessionData{},
			wantStatus:     http.StatusUnauthorized,
			wantNextCalled: false,
			wantError:      "unauthorized",
		},
		{
			name:           "store error returns 401",
			cookie:         &http.Cookie{Name: "session_token", Value: "some-token"},
			getErr:         errors.New("redis connection failed"),
			wantStatus:     http.StatusUnauthorized,
			wantNextCalled: false,
		},
		{
			name:           "valid session calls next with userID in context",
			cookie:         &http.Cookie{Name: "session_token", Value: "valid-token"},
			sessions:       map[string]*auth.SessionData{"valid-token": validSession},
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
			wantUserID:     "user-123",
			wantRefreshed:  true,
		},
		{
			name:           "valid session refreshes TTL",
			cookie:         &http.Cookie{Name: "session_token", Value: "valid-token"},
			sessions:       map[string]*auth.SessionData{"valid-token": validSession},
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
			wantRefreshed:  true,
		},
		{
			name:           "refresh error still calls next",
			cookie:         &http.Cookie{Name: "session_token", Value: "valid-token"},
			sessions:       map[string]*auth.SessionData{"valid-token": validSession},
			refreshErr:     errors.New("refresh failed"),
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
			wantUserID:     "user-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockSessionStore()
			if tt.sessions != nil {
				store.sessions = tt.sessions
			}
			store.getErr = tt.getErr
			store.refreshErr = tt.refreshErr

			next, called, capturedUserID := dummyHandler()
			handler := auth.RequireAuth(store)(next)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if *called != tt.wantNextCalled {
				t.Errorf("next called = %v, want %v", *called, tt.wantNextCalled)
			}
			if tt.wantUserID != "" && *capturedUserID != tt.wantUserID {
				t.Errorf("userID = %q, want %q", *capturedUserID, tt.wantUserID)
			}
			if tt.wantError != "" {
				errMsg := decodeErrorResponse(t, rec)
				if errMsg != tt.wantError {
					t.Errorf("error = %q, want %q", errMsg, tt.wantError)
				}
				ct := rec.Header().Get("Content-Type")
				if ct != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", ct)
				}
			}
			if tt.wantRefreshed && len(store.refreshed) == 0 {
				t.Error("expected Refresh to be called, but it was not")
			}
		})
	}
}

func TestUserIDFromContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantUID string
		wantOK  bool
	}{
		{
			name:    "not set returns empty and false",
			ctx:     context.Background(),
			wantUID: "",
			wantOK:  false,
		},
		{
			name:    "set returns value and true",
			ctx:     context.WithValue(context.Background(), auth.UserIDKey, "user-abc"),
			wantUID: "user-abc",
			wantOK:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uid, ok := auth.UserIDFromContext(tt.ctx)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if uid != tt.wantUID {
				t.Errorf("uid = %q, want %q", uid, tt.wantUID)
			}
		})
	}
}
