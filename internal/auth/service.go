package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

// AuthService orchestrates the login flow: OAuth -> token exchange ->
// Faceit profile fetch -> user upsert -> keychain storage.
type AuthService struct {
	oauth       OAuthConfig
	tokens      *TokenStore
	faceit      testutil.FaceitClient
	queries     *store.Queries
	openBrowser BrowserOpener

	mu          sync.RWMutex
	accessToken string
	currentUser *store.User
}

// NewAuthService creates an AuthService with all dependencies injected.
func NewAuthService(
	oauth OAuthConfig,
	tokens *TokenStore,
	faceit testutil.FaceitClient,
	queries *store.Queries,
	openBrowser BrowserOpener,
) *AuthService {
	return &AuthService{
		oauth:       oauth,
		tokens:      tokens,
		faceit:      faceit,
		queries:     queries,
		openBrowser: openBrowser,
	}
}

// Login runs the full loopback OAuth flow, fetches the Faceit profile,
// upserts the user in SQLite, and stores the refresh token in the keychain.
func (s *AuthService) Login(ctx context.Context) (*store.User, error) {
	tokenResp, err := StartLoopbackFlow(ctx, s.oauth, s.openBrowser)
	if err != nil {
		return nil, fmt.Errorf("oauth flow: %w", err)
	}

	// Fetch profile using the new access token.
	profileCtx := WithAccessToken(ctx, tokenResp.AccessToken)
	player, err := s.faceit.GetPlayer(profileCtx, "me")
	if err != nil {
		return nil, fmt.Errorf("fetching profile: %w", err)
	}

	// Upsert user in SQLite.
	user, err := s.upsertUser(ctx, player)
	if err != nil {
		return nil, fmt.Errorf("upserting user: %w", err)
	}

	// Store refresh token + user ID in OS keychain.
	if err := s.tokens.SaveRefreshToken(tokenResp.RefreshToken); err != nil {
		return nil, fmt.Errorf("saving refresh token: %w", err)
	}
	if err := s.tokens.SaveUserID(strconv.FormatInt(user.ID, 10)); err != nil {
		return nil, fmt.Errorf("saving user ID: %w", err)
	}

	// Cache in memory.
	s.mu.Lock()
	s.accessToken = tokenResp.AccessToken
	s.currentUser = &user
	s.mu.Unlock()

	return &user, nil
}

// GetCurrentUser returns the currently authenticated user.
// Check order: in-memory cache -> keychain+SQLite lookup -> nil (not logged in).
// A nil return with nil error means "not logged in" (not an error condition).
func (s *AuthService) GetCurrentUser(ctx context.Context) (*store.User, error) {
	// 1. Check in-memory cache.
	s.mu.RLock()
	if s.currentUser != nil {
		u := *s.currentUser
		s.mu.RUnlock()
		return &u, nil
	}
	s.mu.RUnlock()

	// 2. Check keychain for stored user ID.
	userIDStr, err := s.tokens.GetUserID()
	if errors.Is(err, testutil.ErrKeyNotFound) {
		return nil, nil // Not logged in.
	}
	if err != nil {
		return nil, fmt.Errorf("reading user ID from keychain: %w", err)
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing user ID %q: %w", userIDStr, err)
	}

	// 3. Lookup user in SQLite.
	user, err := s.queries.GetUserByID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		// User ID in keychain but not in DB -- stale session.
		_ = s.tokens.Clear()
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("looking up user: %w", err)
	}

	// Cache for future calls.
	s.mu.Lock()
	s.currentUser = &user
	s.mu.Unlock()

	return &user, nil
}

// GetAccessToken returns the in-memory access token (may be empty if not logged in).
func (s *AuthService) GetAccessToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accessToken
}

// Logout clears the keychain and in-memory auth state.
func (s *AuthService) Logout() error {
	if err := s.tokens.Clear(); err != nil {
		return fmt.Errorf("clearing keychain: %w", err)
	}

	s.mu.Lock()
	s.accessToken = ""
	s.currentUser = nil
	s.mu.Unlock()

	return nil
}

// upsertUser creates or updates a user based on the Faceit player profile.
func (s *AuthService) upsertUser(ctx context.Context, player *testutil.FaceitPlayer) (store.User, error) {
	existing, err := s.queries.GetUserByFaceitID(ctx, player.PlayerID)
	if err == nil {
		// User exists -- update profile fields.
		return s.queries.UpdateUser(ctx, store.UpdateUserParams{
			ID:          existing.ID,
			Nickname:    player.Nickname,
			AvatarUrl:   player.Avatar,
			FaceitElo:   int64(player.FaceitElo),
			FaceitLevel: int64(player.SkillLevel),
			Country:     player.Country,
		})
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return store.User{}, fmt.Errorf("checking existing user: %w", err)
	}

	// New user -- create.
	return s.queries.CreateUser(ctx, store.CreateUserParams{
		FaceitID:    player.PlayerID,
		Nickname:    player.Nickname,
		AvatarUrl:   player.Avatar,
		FaceitElo:   int64(player.FaceitElo),
		FaceitLevel: int64(player.SkillLevel),
		Country:     player.Country,
	})
}
