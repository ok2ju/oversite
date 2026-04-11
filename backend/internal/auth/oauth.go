package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/ok2ju/oversite/backend/internal/store"
)

// FaceitOAuthConfig holds the OAuth 2.0 configuration for Faceit.
type FaceitOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
}

// PKCEPair holds a PKCE code verifier and its corresponding challenge.
type PKCEPair struct {
	Verifier  string
	Challenge string
}

// OAuthState is stored in Redis during the OAuth flow.
type OAuthState struct {
	CodeVerifier string `json:"code_verifier"`
}

// TokenResponse represents the token response from Faceit.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// FaceitUserInfo represents the user info returned by the Faceit API.
type FaceitUserInfo struct {
	PlayerID string `json:"guid"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Country  string `json:"country"`
}

// StateStore is the interface for storing and retrieving OAuth state and sessions.
type StateStore interface {
	Create(ctx context.Context, key string, data []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

// UserStore is the interface for user persistence.
// *store.Queries satisfies this implicitly.
type UserStore interface {
	GetUserByFaceitID(ctx context.Context, faceitID string) (store.User, error)
	CreateUser(ctx context.Context, arg store.CreateUserParams) (store.User, error)
	UpdateUser(ctx context.Context, arg store.UpdateUserParams) (store.User, error)
}

// TokenExchanger handles the HTTP calls to Faceit for code exchange and user info.
type TokenExchanger interface {
	ExchangeCode(ctx context.Context, code, codeVerifier string) (*TokenResponse, error)
	GetUserInfo(ctx context.Context, accessToken string) (*FaceitUserInfo, error)
}

// GenerateCodeVerifier generates a cryptographically random PKCE code verifier.
func GenerateCodeVerifier() (string, error) {
	return GenerateCodeVerifierWithReader(rand.Reader)
}

// GenerateCodeVerifierWithReader generates a PKCE code verifier from the given reader.
func GenerateCodeVerifierWithReader(r io.Reader) (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", fmt.Errorf("generating code verifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ComputeCodeChallenge computes the S256 PKCE code challenge for a verifier.
func ComputeCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// GeneratePKCEPair generates a PKCE verifier + challenge pair.
func GeneratePKCEPair() (*PKCEPair, error) {
	verifier, err := GenerateCodeVerifier()
	if err != nil {
		return nil, err
	}
	return &PKCEPair{
		Verifier:  verifier,
		Challenge: ComputeCodeChallenge(verifier),
	}, nil
}

// GenerateState generates a cryptographically random hex state string.
func GenerateState() (string, error) {
	return GenerateStateWithReader(rand.Reader)
}

// GenerateStateWithReader generates a hex state string from the given reader.
func GenerateStateWithReader(r io.Reader) (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// OAuthService orchestrates the Faceit OAuth 2.0 + PKCE flow.
type OAuthService struct {
	config    FaceitOAuthConfig
	states    StateStore
	users     UserStore
	exchanger TokenExchanger
}

// NewOAuthService creates a new OAuthService.
func NewOAuthService(cfg FaceitOAuthConfig, states StateStore, users UserStore, exchanger TokenExchanger) *OAuthService {
	return &OAuthService{
		config:    cfg,
		states:    states,
		users:     users,
		exchanger: exchanger,
	}
}

// AuthorizationURL generates the Faceit authorize URL with PKCE and stores state in Redis.
func (s *OAuthService) AuthorizationURL(ctx context.Context) (string, error) {
	pkce, err := GeneratePKCEPair()
	if err != nil {
		return "", fmt.Errorf("generating PKCE pair: %w", err)
	}

	state, err := GenerateState()
	if err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}

	oauthState := OAuthState{CodeVerifier: pkce.Verifier}
	data, err := json.Marshal(oauthState)
	if err != nil {
		return "", fmt.Errorf("marshaling state: %w", err)
	}

	key := "oauth_state:" + state
	if err := s.states.Create(ctx, key, data, 5*time.Minute); err != nil {
		return "", fmt.Errorf("storing state: %w", err)
	}

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {s.config.ClientID},
		"redirect_uri":          {s.config.RedirectURI},
		"state":                 {state},
		"code_challenge":        {pkce.Challenge},
		"code_challenge_method": {"S256"},
		"scope":                 {"openid profile email"},
		"redirect_popup":        {"true"},
	}

	return s.config.AuthURL + "?" + params.Encode(), nil
}

// HandleCallback processes the OAuth callback: validates state, exchanges code, upserts user.
func (s *OAuthService) HandleCallback(ctx context.Context, code, state string) (*store.User, *TokenResponse, error) {
	key := "oauth_state:" + state
	data, err := s.states.Get(ctx, key)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid state: %w", err)
	}

	var oauthState OAuthState
	if err := json.Unmarshal(data, &oauthState); err != nil {
		return nil, nil, fmt.Errorf("unmarshaling state: %w", err)
	}

	// Delete state after retrieval (single-use)
	if err := s.states.Delete(ctx, key); err != nil {
		return nil, nil, fmt.Errorf("deleting state: %w", err)
	}

	tokens, err := s.exchanger.ExchangeCode(ctx, code, oauthState.CodeVerifier)
	if err != nil {
		return nil, nil, fmt.Errorf("exchanging code: %w", err)
	}

	userInfo, err := s.exchanger.GetUserInfo(ctx, tokens.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("getting user info: %w", err)
	}

	user, err := s.upsertUser(ctx, userInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("upserting user: %w", err)
	}

	return &user, tokens, nil
}

func (s *OAuthService) upsertUser(ctx context.Context, info *FaceitUserInfo) (store.User, error) {
	existing, err := s.users.GetUserByFaceitID(ctx, info.PlayerID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return store.User{}, fmt.Errorf("looking up user: %w", err)
	}

	if errors.Is(err, sql.ErrNoRows) {
		return s.users.CreateUser(ctx, store.CreateUserParams{
			FaceitID:  info.PlayerID,
			Nickname:  info.Nickname,
			AvatarUrl: sql.NullString{String: info.Avatar, Valid: info.Avatar != ""},
			Country:   sql.NullString{String: info.Country, Valid: info.Country != ""},
		})
	}

	return s.users.UpdateUser(ctx, store.UpdateUserParams{
		ID:          existing.ID,
		Nickname:    info.Nickname,
		AvatarUrl:   sql.NullString{String: info.Avatar, Valid: info.Avatar != ""},
		FaceitElo:   existing.FaceitElo,
		FaceitLevel: existing.FaceitLevel,
		Country:     sql.NullString{String: info.Country, Valid: info.Country != ""},
	})
}
