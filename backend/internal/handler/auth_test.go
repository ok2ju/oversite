package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/handler"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// --- Test mocks ---

type mockStateStore struct {
	data map[string][]byte
}

func newMockStateStore() *mockStateStore {
	return &mockStateStore{data: make(map[string][]byte)}
}

func (m *mockStateStore) Create(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	m.data[key] = data
	return nil
}

func (m *mockStateStore) Get(ctx context.Context, key string) ([]byte, error) {
	d, ok := m.data[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return d, nil
}

func (m *mockStateStore) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

type mockUserStore struct {
	users map[string]store.User
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{users: make(map[string]store.User)}
}

func (m *mockUserStore) GetUserByFaceitID(ctx context.Context, faceitID string) (store.User, error) {
	u, ok := m.users[faceitID]
	if !ok {
		return store.User{}, sql.ErrNoRows
	}
	return u, nil
}

func (m *mockUserStore) CreateUser(ctx context.Context, arg store.CreateUserParams) (store.User, error) {
	return store.User{
		ID:       uuid.New(),
		FaceitID: arg.FaceitID,
		Nickname: arg.Nickname,
	}, nil
}

func (m *mockUserStore) UpdateUser(ctx context.Context, arg store.UpdateUserParams) (store.User, error) {
	return store.User{ID: arg.ID, Nickname: arg.Nickname}, nil
}

type mockTokenExchanger struct {
	tokenResp *auth.TokenResponse
	tokenErr  error
	userInfo  *auth.FaceitUserInfo
	userErr   error
}

func (m *mockTokenExchanger) ExchangeCode(ctx context.Context, code, codeVerifier string) (*auth.TokenResponse, error) {
	if m.tokenErr != nil {
		return nil, m.tokenErr
	}
	return m.tokenResp, nil
}

func (m *mockTokenExchanger) GetUserInfo(ctx context.Context, accessToken string) (*auth.FaceitUserInfo, error) {
	if m.userErr != nil {
		return nil, m.userErr
	}
	return m.userInfo, nil
}

// --- Helpers ---

func newTestAuthHandler(exchanger *mockTokenExchanger) (*handler.AuthHandler, *mockStateStore) {
	states := newMockStateStore()
	users := newMockUserStore()
	cfg := auth.FaceitOAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:3000/callback",
		AuthURL:      "https://cdn.faceit.com/widgets/sso/index.html",
		TokenURL:     "https://api.faceit.com/auth/v1/oauth/token",
		UserInfoURL:  "https://api.faceit.com/auth/v1/resources/userinfo",
	}
	oauth := auth.NewOAuthService(cfg, states, users, exchanger)
	return handler.NewAuthHandler(oauth, states, false), states
}

// --- Tests ---

func TestHandleLogin_Redirects(t *testing.T) {
	h, _ := newTestAuthHandler(&mockTokenExchanger{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/faceit", nil)
	rec := httptest.NewRecorder()

	h.HandleLogin(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rec.Code)
	}
}

func TestHandleLogin_LocationHasRequiredParams(t *testing.T) {
	h, _ := newTestAuthHandler(&mockTokenExchanger{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/faceit", nil)
	rec := httptest.NewRecorder()

	h.HandleLogin(rec, req)

	location := rec.Header().Get("Location")
	for _, param := range []string{"response_type=code", "client_id=test-client-id", "state=", "code_challenge=", "code_challenge_method=S256"} {
		if !strings.Contains(location, param) {
			t.Errorf("location missing %q: %s", param, location)
		}
	}
}

func TestHandleCallback_MissingCode(t *testing.T) {
	h, _ := newTestAuthHandler(&mockTokenExchanger{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/faceit/callback?state=abc", nil)
	rec := httptest.NewRecorder()

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleCallback_MissingState(t *testing.T) {
	h, _ := newTestAuthHandler(&mockTokenExchanger{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/faceit/callback?code=abc", nil)
	rec := httptest.NewRecorder()

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleCallback_InvalidState(t *testing.T) {
	h, _ := newTestAuthHandler(&mockTokenExchanger{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/faceit/callback?code=abc&state=bad", nil)
	rec := httptest.NewRecorder()

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleCallback_Success_SetsCookie(t *testing.T) {
	exchanger := &mockTokenExchanger{
		tokenResp: &auth.TokenResponse{AccessToken: "tok", RefreshToken: "ref", ExpiresIn: 3600},
		userInfo:  &auth.FaceitUserInfo{PlayerID: "faceit-123", Nickname: "player1"},
	}
	h, states := newTestAuthHandler(exchanger)

	// Pre-populate state
	states.data["oauth_state:valid-state"] = []byte(`{"code_verifier":"test-verifier"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/faceit/callback?code=test-code&state=valid-state", nil)
	rec := httptest.NewRecorder()

	h.HandleCallback(rec, req)

	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_token" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected session_token cookie to be set")
	}
	if !sessionCookie.HttpOnly {
		t.Error("expected cookie to be HttpOnly")
	}
	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite Lax, got %v", sessionCookie.SameSite)
	}
}

func TestHandleCallback_Success_RedirectsToDashboard(t *testing.T) {
	exchanger := &mockTokenExchanger{
		tokenResp: &auth.TokenResponse{AccessToken: "tok", RefreshToken: "ref", ExpiresIn: 3600},
		userInfo:  &auth.FaceitUserInfo{PlayerID: "faceit-123", Nickname: "player1"},
	}
	h, states := newTestAuthHandler(exchanger)
	states.data["oauth_state:valid-state"] = []byte(`{"code_verifier":"test-verifier"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/faceit/callback?code=test-code&state=valid-state", nil)
	rec := httptest.NewRecorder()

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/dashboard" {
		t.Errorf("expected redirect to /dashboard, got %q", loc)
	}
}

func TestHandleCallback_Success_CreatesSession(t *testing.T) {
	exchanger := &mockTokenExchanger{
		tokenResp: &auth.TokenResponse{AccessToken: "tok", RefreshToken: "ref", ExpiresIn: 3600},
		userInfo:  &auth.FaceitUserInfo{PlayerID: "faceit-123", Nickname: "player1"},
	}
	h, states := newTestAuthHandler(exchanger)
	states.data["oauth_state:valid-state"] = []byte(`{"code_verifier":"test-verifier"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/faceit/callback?code=test-code&state=valid-state", nil)
	rec := httptest.NewRecorder()

	h.HandleCallback(rec, req)

	// Session should be stored -- check that at least one session key exists
	cookies := rec.Result().Cookies()
	var token string
	for _, c := range cookies {
		if c.Name == "session_token" {
			token = c.Value
			break
		}
	}
	if token == "" {
		t.Fatal("no session token in cookie")
	}

	// Verify session was stored in state store
	sessionData, err := states.Get(context.Background(), "session:"+token)
	if err != nil {
		t.Fatalf("session not found in store: %v", err)
	}

	var session map[string]interface{}
	if err := json.Unmarshal(sessionData, &session); err != nil {
		t.Fatalf("failed to unmarshal session: %v", err)
	}
	if session["faceit_id"] != "faceit-123" {
		t.Errorf("expected faceit_id 'faceit-123' in session, got %v", session["faceit_id"])
	}
}

func TestHandleCallback_FullFlow(t *testing.T) {
	// Set up mock Faceit server
	faceitServer := httptest.NewServer(http.NewServeMux())
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "access-123",
			"refresh_token": "refresh-456",
			"expires_in":    3600,
		})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"guid":     "faceit-xyz",
			"nickname": "testplayer",
			"avatar":   "https://example.com/av.png",
			"country":  "DE",
		})
	})
	faceitServer.Close()
	faceitServer = httptest.NewServer(mux)
	defer faceitServer.Close()

	states := newMockStateStore()
	users := newMockUserStore()
	cfg := auth.FaceitOAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:3000/callback",
		AuthURL:      "https://cdn.faceit.com/widgets/sso/index.html",
		TokenURL:     faceitServer.URL + "/token",
		UserInfoURL:  faceitServer.URL + "/userinfo",
	}
	faceitClient := auth.NewFaceitClient(cfg)
	oauth := auth.NewOAuthService(cfg, states, users, faceitClient)
	h := handler.NewAuthHandler(oauth, states, false)

	// Step 1: Login redirect
	loginReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/faceit", nil)
	loginRec := httptest.NewRecorder()
	h.HandleLogin(loginRec, loginReq)

	if loginRec.Code != http.StatusFound {
		t.Fatalf("expected 302 from login, got %d", loginRec.Code)
	}

	location := loginRec.Header().Get("Location")
	if !strings.Contains(location, "response_type=code") {
		t.Fatalf("login redirect missing required params: %s", location)
	}

	// Extract the state from the redirect URL
	parts := strings.Split(location, "state=")
	if len(parts) < 2 {
		t.Fatalf("no state in redirect URL: %s", location)
	}
	state := strings.Split(parts[1], "&")[0]

	// Step 2: Callback
	callbackReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/faceit/callback?code=auth-code&state="+state, nil)
	callbackRec := httptest.NewRecorder()
	h.HandleCallback(callbackRec, callbackReq)

	if callbackRec.Code != http.StatusFound {
		t.Errorf("expected 302 from callback, got %d", callbackRec.Code)
	}
	if loc := callbackRec.Header().Get("Location"); loc != "/dashboard" {
		t.Errorf("expected redirect to /dashboard, got %q", loc)
	}

	// Verify cookie set
	cookies := callbackRec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "session_token" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected session_token cookie after callback")
	}
}
