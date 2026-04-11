package auth_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// --- Spy/mock implementations for testing ---

type spyStateStore struct {
	data    map[string][]byte
	created []string
	deleted []string
}

func newSpyStateStore() *spyStateStore {
	return &spyStateStore{data: make(map[string][]byte)}
}

func (s *spyStateStore) Create(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	s.data[key] = data
	s.created = append(s.created, key)
	return nil
}

func (s *spyStateStore) Get(ctx context.Context, key string) ([]byte, error) {
	d, ok := s.data[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return d, nil
}

func (s *spyStateStore) Delete(ctx context.Context, key string) error {
	delete(s.data, key)
	s.deleted = append(s.deleted, key)
	return nil
}

type spyUserStore struct {
	users       map[string]store.User
	createdWith *store.CreateUserParams
	updatedWith *store.UpdateUserParams
}

func newSpyUserStore() *spyUserStore {
	return &spyUserStore{users: make(map[string]store.User)}
}

func (s *spyUserStore) GetUserByFaceitID(ctx context.Context, faceitID string) (store.User, error) {
	u, ok := s.users[faceitID]
	if !ok {
		return store.User{}, sql.ErrNoRows
	}
	return u, nil
}

func (s *spyUserStore) CreateUser(ctx context.Context, arg store.CreateUserParams) (store.User, error) {
	s.createdWith = &arg
	return store.User{
		ID:       uuid.New(),
		FaceitID: arg.FaceitID,
		Nickname: arg.Nickname,
	}, nil
}

func (s *spyUserStore) UpdateUser(ctx context.Context, arg store.UpdateUserParams) (store.User, error) {
	s.updatedWith = &arg
	u := s.users[arg.Nickname] // just return something
	u.ID = arg.ID
	return u, nil
}

type spyTokenExchanger struct {
	tokenResp *auth.TokenResponse
	tokenErr  error
	userInfo  *auth.FaceitUserInfo
	userErr   error
}

func (s *spyTokenExchanger) ExchangeCode(ctx context.Context, code, codeVerifier string) (*auth.TokenResponse, error) {
	if s.tokenErr != nil {
		return nil, s.tokenErr
	}
	return s.tokenResp, nil
}

func (s *spyTokenExchanger) GetUserInfo(ctx context.Context, accessToken string) (*auth.FaceitUserInfo, error) {
	if s.userErr != nil {
		return nil, s.userErr
	}
	return s.userInfo, nil
}

// --- PKCE tests ---

func TestGenerateCodeVerifier_Length(t *testing.T) {
	v, err := auth.GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v) < 43 || len(v) > 128 {
		t.Errorf("verifier length %d not in range [43, 128]", len(v))
	}
}

func TestGenerateCodeVerifier_Base64URLCharset(t *testing.T) {
	v, err := auth.GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	re := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	if !re.MatchString(v) {
		t.Errorf("verifier contains invalid characters: %q", v)
	}
}

func TestGenerateCodeVerifier_Uniqueness(t *testing.T) {
	v1, _ := auth.GenerateCodeVerifier()
	v2, _ := auth.GenerateCodeVerifier()
	if v1 == v2 {
		t.Error("two verifiers should differ")
	}
}

func TestGenerateCodeVerifierWithReader_Deterministic(t *testing.T) {
	input := bytes.Repeat([]byte{0xAB}, 32)
	reader := bytes.NewReader(input)
	v, err := auth.GenerateCodeVerifierWithReader(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := base64.RawURLEncoding.EncodeToString(input)
	if v != expected {
		t.Errorf("expected %q, got %q", expected, v)
	}
}

func TestComputeCodeChallenge_KnownVector(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	h := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(h[:])

	challenge := auth.ComputeCodeChallenge(verifier)
	if challenge != expected {
		t.Errorf("expected %q, got %q", expected, challenge)
	}
}

func TestComputeCodeChallenge_NoPadding(t *testing.T) {
	challenge := auth.ComputeCodeChallenge("test-verifier")
	if strings.Contains(challenge, "=") {
		t.Errorf("challenge should not contain padding: %q", challenge)
	}
}

func TestGeneratePKCEPair_ChallengeMatchesVerifier(t *testing.T) {
	pair, err := auth.GeneratePKCEPair()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := auth.ComputeCodeChallenge(pair.Verifier)
	if pair.Challenge != expected {
		t.Errorf("challenge %q does not match recomputed %q", pair.Challenge, expected)
	}
}

// --- State tests ---

func TestGenerateState_Length(t *testing.T) {
	s, err := auth.GenerateState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 64 {
		t.Errorf("expected state length 64, got %d", len(s))
	}
}

func TestGenerateState_HexCharset(t *testing.T) {
	s, err := auth.GenerateState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := hex.DecodeString(s); err != nil {
		t.Errorf("state is not valid hex: %q", s)
	}
}

func TestGenerateState_Uniqueness(t *testing.T) {
	s1, _ := auth.GenerateState()
	s2, _ := auth.GenerateState()
	if s1 == s2 {
		t.Error("two states should differ")
	}
}

// --- OAuthService tests ---

func testOAuthConfig() auth.FaceitOAuthConfig {
	return auth.FaceitOAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:3000/callback",
		AuthURL:      "https://accounts.faceit.com/accounts",
		TokenURL:     "https://api.faceit.com/auth/v1/oauth/token",
		UserInfoURL:  "https://api.faceit.com/auth/v1/resources/userinfo",
	}
}

func TestAuthorizationURL_ContainsRequiredParams(t *testing.T) {
	states := newSpyStateStore()
	users := newSpyUserStore()
	exchanger := &spyTokenExchanger{}
	svc := auth.NewOAuthService(testOAuthConfig(), states, users, exchanger)

	url, err := svc.AuthorizationURL(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, param := range []string{"response_type=code", "client_id=test-client-id", "redirect_uri=", "state=", "code_challenge=", "code_challenge_method=S256", "scope=openid+profile+email", "redirect_popup=true"} {
		if !strings.Contains(url, param) {
			t.Errorf("URL missing %q: %s", param, url)
		}
	}
}

func TestAuthorizationURL_StoresStateInRedis(t *testing.T) {
	states := newSpyStateStore()
	users := newSpyUserStore()
	exchanger := &spyTokenExchanger{}
	svc := auth.NewOAuthService(testOAuthConfig(), states, users, exchanger)

	_, err := svc.AuthorizationURL(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(states.created) != 1 {
		t.Fatalf("expected 1 state created, got %d", len(states.created))
	}
	if !strings.HasPrefix(states.created[0], "oauth_state:") {
		t.Errorf("state key should start with 'oauth_state:', got %q", states.created[0])
	}
}

func TestAuthorizationURL_BaseURLCorrect(t *testing.T) {
	states := newSpyStateStore()
	users := newSpyUserStore()
	exchanger := &spyTokenExchanger{}
	svc := auth.NewOAuthService(testOAuthConfig(), states, users, exchanger)

	url, err := svc.AuthorizationURL(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(url, "https://accounts.faceit.com/accounts?") {
		t.Errorf("URL should start with auth base URL, got %q", url)
	}
}

func TestHandleCallback_InvalidState(t *testing.T) {
	states := newSpyStateStore()
	users := newSpyUserStore()
	exchanger := &spyTokenExchanger{}
	svc := auth.NewOAuthService(testOAuthConfig(), states, users, exchanger)

	_, _, err := svc.HandleCallback(context.Background(), "some-code", "bad-state")
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
}

func TestHandleCallback_ExchangeCodeFailure(t *testing.T) {
	states := newSpyStateStore()
	states.data["oauth_state:valid-state"] = []byte(`{"code_verifier":"test-verifier"}`)
	users := newSpyUserStore()
	exchanger := &spyTokenExchanger{tokenErr: errors.New("exchange failed")}
	svc := auth.NewOAuthService(testOAuthConfig(), states, users, exchanger)

	_, _, err := svc.HandleCallback(context.Background(), "some-code", "valid-state")
	if err == nil {
		t.Fatal("expected error for exchange failure")
	}
}

func TestHandleCallback_GetUserInfoFailure(t *testing.T) {
	states := newSpyStateStore()
	states.data["oauth_state:valid-state"] = []byte(`{"code_verifier":"test-verifier"}`)
	users := newSpyUserStore()
	exchanger := &spyTokenExchanger{
		tokenResp: &auth.TokenResponse{AccessToken: "tok", RefreshToken: "ref", ExpiresIn: 3600},
		userErr:   errors.New("user info failed"),
	}
	svc := auth.NewOAuthService(testOAuthConfig(), states, users, exchanger)

	_, _, err := svc.HandleCallback(context.Background(), "some-code", "valid-state")
	if err == nil {
		t.Fatal("expected error for user info failure")
	}
}

func TestHandleCallback_NewUser_Created(t *testing.T) {
	states := newSpyStateStore()
	states.data["oauth_state:valid-state"] = []byte(`{"code_verifier":"test-verifier"}`)
	users := newSpyUserStore()
	exchanger := &spyTokenExchanger{
		tokenResp: &auth.TokenResponse{AccessToken: "tok", RefreshToken: "ref", ExpiresIn: 3600},
		userInfo:  &auth.FaceitUserInfo{PlayerID: "faceit-123", Nickname: "player1", Avatar: "https://example.com/avatar.png", Country: "US"},
	}
	svc := auth.NewOAuthService(testOAuthConfig(), states, users, exchanger)

	user, _, err := svc.HandleCallback(context.Background(), "some-code", "valid-state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if users.createdWith == nil {
		t.Fatal("expected CreateUser to be called")
	}
	if users.createdWith.FaceitID != "faceit-123" {
		t.Errorf("expected FaceitID 'faceit-123', got %q", users.createdWith.FaceitID)
	}
	if user.FaceitID != "faceit-123" {
		t.Errorf("expected returned user FaceitID 'faceit-123', got %q", user.FaceitID)
	}
}

func TestHandleCallback_ExistingUser_Updated(t *testing.T) {
	states := newSpyStateStore()
	states.data["oauth_state:valid-state"] = []byte(`{"code_verifier":"test-verifier"}`)
	existingID := uuid.New()
	users := newSpyUserStore()
	users.users["faceit-123"] = store.User{ID: existingID, FaceitID: "faceit-123", Nickname: "old-name"}
	exchanger := &spyTokenExchanger{
		tokenResp: &auth.TokenResponse{AccessToken: "tok", RefreshToken: "ref", ExpiresIn: 3600},
		userInfo:  &auth.FaceitUserInfo{PlayerID: "faceit-123", Nickname: "new-name", Avatar: "https://example.com/avatar.png", Country: "US"},
	}
	svc := auth.NewOAuthService(testOAuthConfig(), states, users, exchanger)

	_, _, err := svc.HandleCallback(context.Background(), "some-code", "valid-state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if users.updatedWith == nil {
		t.Fatal("expected UpdateUser to be called")
	}
	if users.updatedWith.ID != existingID {
		t.Errorf("expected update for user ID %s, got %s", existingID, users.updatedWith.ID)
	}
	if users.updatedWith.Nickname != "new-name" {
		t.Errorf("expected updated nickname 'new-name', got %q", users.updatedWith.Nickname)
	}
}

func TestHandleCallback_ExistingUser_PreservesEloAndLevel(t *testing.T) {
	states := newSpyStateStore()
	states.data["oauth_state:valid-state"] = []byte(`{"code_verifier":"test-verifier"}`)
	existingID := uuid.New()
	users := newSpyUserStore()
	users.users["faceit-123"] = store.User{
		ID:          existingID,
		FaceitID:    "faceit-123",
		Nickname:    "old-name",
		FaceitElo:   sql.NullInt32{Int32: 2100, Valid: true},
		FaceitLevel: sql.NullInt16{Int16: 10, Valid: true},
	}
	exchanger := &spyTokenExchanger{
		tokenResp: &auth.TokenResponse{AccessToken: "tok", RefreshToken: "ref", ExpiresIn: 3600},
		userInfo:  &auth.FaceitUserInfo{PlayerID: "faceit-123", Nickname: "new-name", Avatar: "https://example.com/avatar.png", Country: "US"},
	}
	svc := auth.NewOAuthService(testOAuthConfig(), states, users, exchanger)

	_, _, err := svc.HandleCallback(context.Background(), "some-code", "valid-state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if users.updatedWith == nil {
		t.Fatal("expected UpdateUser to be called")
	}
	if users.updatedWith.FaceitElo.Int32 != 2100 || !users.updatedWith.FaceitElo.Valid {
		t.Errorf("expected FaceitElo {2100, true}, got {%d, %v}", users.updatedWith.FaceitElo.Int32, users.updatedWith.FaceitElo.Valid)
	}
	if users.updatedWith.FaceitLevel.Int16 != 10 || !users.updatedWith.FaceitLevel.Valid {
		t.Errorf("expected FaceitLevel {10, true}, got {%d, %v}", users.updatedWith.FaceitLevel.Int16, users.updatedWith.FaceitLevel.Valid)
	}
}

func TestHandleCallback_DeletesStateAfterUse(t *testing.T) {
	states := newSpyStateStore()
	states.data["oauth_state:valid-state"] = []byte(`{"code_verifier":"test-verifier"}`)
	users := newSpyUserStore()
	exchanger := &spyTokenExchanger{
		tokenResp: &auth.TokenResponse{AccessToken: "tok", RefreshToken: "ref", ExpiresIn: 3600},
		userInfo:  &auth.FaceitUserInfo{PlayerID: "faceit-123", Nickname: "player1"},
	}
	svc := auth.NewOAuthService(testOAuthConfig(), states, users, exchanger)

	_, _, err := svc.HandleCallback(context.Background(), "some-code", "valid-state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(states.deleted) != 1 || states.deleted[0] != "oauth_state:valid-state" {
		t.Errorf("expected state 'oauth_state:valid-state' to be deleted, got deleted: %v", states.deleted)
	}
}

func TestHandleCallback_ReturnsUserAndTokens(t *testing.T) {
	states := newSpyStateStore()
	states.data["oauth_state:valid-state"] = []byte(`{"code_verifier":"test-verifier"}`)
	users := newSpyUserStore()
	exchanger := &spyTokenExchanger{
		tokenResp: &auth.TokenResponse{AccessToken: "access-tok", RefreshToken: "refresh-tok", ExpiresIn: 3600},
		userInfo:  &auth.FaceitUserInfo{PlayerID: "faceit-123", Nickname: "player1"},
	}
	svc := auth.NewOAuthService(testOAuthConfig(), states, users, exchanger)

	user, tokens, err := svc.HandleCallback(context.Background(), "some-code", "valid-state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user == nil {
		t.Fatal("expected user to be returned")
	}
	if tokens == nil {
		t.Fatal("expected tokens to be returned")
	}
	if tokens.AccessToken != "access-tok" {
		t.Errorf("expected access token 'access-tok', got %q", tokens.AccessToken)
	}
	if tokens.RefreshToken != "refresh-tok" {
		t.Errorf("expected refresh token 'refresh-tok', got %q", tokens.RefreshToken)
	}
}
