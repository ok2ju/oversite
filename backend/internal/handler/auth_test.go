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

func (m *mockStateStore) Create(_ context.Context, key string, data []byte, _ time.Duration) error {
	m.data[key] = data
	return nil
}

func (m *mockStateStore) Get(_ context.Context, key string) ([]byte, error) {
	d, ok := m.data[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return d, nil
}

func (m *mockStateStore) Delete(_ context.Context, key string) error {
	delete(m.data, key)
	return nil
}

type mockSessionStore struct {
	sessions  map[string]*auth.SessionData
	lastToken string
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{sessions: make(map[string]*auth.SessionData)}
}

func (m *mockSessionStore) Create(_ context.Context, data *auth.SessionData) (string, error) {
	token := "mock-session-token"
	m.sessions[token] = data
	m.lastToken = token
	return token, nil
}

func (m *mockSessionStore) Get(_ context.Context, token string) (*auth.SessionData, error) {
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
	if _, ok := m.sessions[token]; !ok {
		return auth.ErrSessionNotFound
	}
	return nil
}

type mockUserStore struct {
	users map[string]store.User
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{users: make(map[string]store.User)}
}

func (m *mockUserStore) GetUserByFaceitID(_ context.Context, faceitID string) (store.User, error) {
	u, ok := m.users[faceitID]
	if !ok {
		return store.User{}, sql.ErrNoRows
	}
	return u, nil
}

func (m *mockUserStore) CreateUser(_ context.Context, arg store.CreateUserParams) (store.User, error) {
	return store.User{
		ID:       uuid.New(),
		FaceitID: arg.FaceitID,
		Nickname: arg.Nickname,
	}, nil
}

func (m *mockUserStore) UpdateUser(_ context.Context, arg store.UpdateUserParams) (store.User, error) {
	return store.User{ID: arg.ID, Nickname: arg.Nickname}, nil
}

type mockTokenExchanger struct {
	tokenResp *auth.TokenResponse
	tokenErr  error
	userInfo  *auth.FaceitUserInfo
	userErr   error
}

func (m *mockTokenExchanger) ExchangeCode(_ context.Context, _, _ string) (*auth.TokenResponse, error) {
	if m.tokenErr != nil {
		return nil, m.tokenErr
	}
	return m.tokenResp, nil
}

func (m *mockTokenExchanger) GetUserInfo(_ context.Context, _ string) (*auth.FaceitUserInfo, error) {
	if m.userErr != nil {
		return nil, m.userErr
	}
	return m.userInfo, nil
}

// --- Helpers ---

func newTestAuthHandler(exchanger *mockTokenExchanger) (*handler.AuthHandler, *mockStateStore, *mockSessionStore) {
	states := newMockStateStore()
	sessions := newMockSessionStore()
	users := newMockUserStore()
	cfg := auth.FaceitOAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:3000/callback",
		AuthURL:      "https://accounts.faceit.com/accounts",
		TokenURL:     "https://api.faceit.com/auth/v1/oauth/token",
		UserInfoURL:  "https://api.faceit.com/auth/v1/resources/userinfo",
	}
	oauth := auth.NewOAuthService(cfg, states, users, exchanger)
	return handler.NewAuthHandler(oauth, sessions, false), states, sessions
}

// --- Login tests ---

func TestHandleLogin_Redirects(t *testing.T) {
	h, _, _ := newTestAuthHandler(&mockTokenExchanger{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/faceit", nil)
	rec := httptest.NewRecorder()

	h.HandleLogin(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rec.Code)
	}
}

func TestHandleLogin_LocationHasRequiredParams(t *testing.T) {
	h, _, _ := newTestAuthHandler(&mockTokenExchanger{})

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

// --- Callback tests ---

func TestHandleCallback_MissingCode(t *testing.T) {
	h, _, _ := newTestAuthHandler(&mockTokenExchanger{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/faceit/callback?state=abc", nil)
	rec := httptest.NewRecorder()

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleCallback_MissingState(t *testing.T) {
	h, _, _ := newTestAuthHandler(&mockTokenExchanger{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/faceit/callback?code=abc", nil)
	rec := httptest.NewRecorder()

	h.HandleCallback(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleCallback_InvalidState(t *testing.T) {
	h, _, _ := newTestAuthHandler(&mockTokenExchanger{})

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
	h, states, _ := newTestAuthHandler(exchanger)

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
	h, states, _ := newTestAuthHandler(exchanger)
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
	h, states, sessions := newTestAuthHandler(exchanger)
	states.data["oauth_state:valid-state"] = []byte(`{"code_verifier":"test-verifier"}`)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/faceit/callback?code=test-code&state=valid-state", nil)
	rec := httptest.NewRecorder()

	h.HandleCallback(rec, req)

	// Verify cookie is set
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

	// Verify session was stored in session store
	session, err := sessions.Get(context.Background(), token)
	if err != nil {
		t.Fatalf("session not found in store: %v", err)
	}
	if session.FaceitID != "faceit-123" {
		t.Errorf("expected faceit_id 'faceit-123', got %q", session.FaceitID)
	}
	if session.Nickname != "player1" {
		t.Errorf("expected nickname 'player1', got %q", session.Nickname)
	}
}

func TestHandleCallback_FullFlow(t *testing.T) {
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
	sessions := newMockSessionStore()
	users := newMockUserStore()
	cfg := auth.FaceitOAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:3000/callback",
		AuthURL:      "https://accounts.faceit.com/accounts",
		TokenURL:     faceitServer.URL + "/token",
		UserInfoURL:  faceitServer.URL + "/userinfo",
	}
	faceitClient := auth.NewFaceitClient(cfg)
	oauth := auth.NewOAuthService(cfg, states, users, faceitClient)
	h := handler.NewAuthHandler(oauth, sessions, false)

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

	// Verify session was created in typed session store
	if len(sessions.sessions) == 0 {
		t.Error("expected session to be stored in session store")
	}
}

// --- Logout tests ---

func TestHandleLogout_Success(t *testing.T) {
	h, _, sessions := newTestAuthHandler(&mockTokenExchanger{})

	// Pre-populate a session
	sessions.sessions["existing-token"] = &auth.SessionData{
		UserID:   "user-1",
		FaceitID: "faceit-1",
		Nickname: "player1",
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "existing-token"})
	rec := httptest.NewRecorder()

	h.HandleLogout(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Session should be deleted
	if _, ok := sessions.sessions["existing-token"]; ok {
		t.Error("expected session to be deleted")
	}

	// Cookie should be cleared
	cookies := rec.Result().Cookies()
	var clearCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_token" {
			clearCookie = c
			break
		}
	}
	if clearCookie == nil {
		t.Fatal("expected clearing cookie to be set")
	}
	if clearCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge -1, got %d", clearCookie.MaxAge)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", body["status"])
	}
}

func TestHandleLogout_NoCookie(t *testing.T) {
	h, _, _ := newTestAuthHandler(&mockTokenExchanger{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()

	h.HandleLogout(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

// --- Me tests ---

func TestHandleMe_ValidSession(t *testing.T) {
	h, _, sessions := newTestAuthHandler(&mockTokenExchanger{})

	sessions.sessions["valid-token"] = &auth.SessionData{
		UserID:       "user-123",
		FaceitID:     "faceit-456",
		Nickname:     "player1",
		AccessToken:  "secret-access",
		RefreshToken: "secret-refresh",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-token"})
	rec := httptest.NewRecorder()

	h.HandleMe(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if body["user_id"] != "user-123" {
		t.Errorf("expected user_id 'user-123', got %q", body["user_id"])
	}
	if body["faceit_id"] != "faceit-456" {
		t.Errorf("expected faceit_id 'faceit-456', got %q", body["faceit_id"])
	}
	if body["nickname"] != "player1" {
		t.Errorf("expected nickname 'player1', got %q", body["nickname"])
	}

	// Tokens must NOT be leaked
	if _, ok := body["access_token"]; ok {
		t.Error("access_token should not be in /me response")
	}
	if _, ok := body["refresh_token"]; ok {
		t.Error("refresh_token should not be in /me response")
	}
}

func TestHandleMe_NoCookie(t *testing.T) {
	h, _, _ := newTestAuthHandler(&mockTokenExchanger{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	h.HandleMe(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestHandleMe_InvalidSession(t *testing.T) {
	h, _, _ := newTestAuthHandler(&mockTokenExchanger{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "bad-token"})
	rec := httptest.NewRecorder()

	h.HandleMe(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}
