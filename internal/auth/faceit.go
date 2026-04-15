package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ok2ju/oversite/internal/testutil"
)

// UserInfoURL is the Faceit OpenID Connect userinfo endpoint.
const UserInfoURL = "https://api.faceit.com/auth/v1/resources/userinfo"

// HTTPFaceitClient implements testutil.FaceitClient using real HTTP calls
// to the Faceit Data API.
type HTTPFaceitClient struct {
	httpClient  *http.Client
	userInfoURL string
}

// NewHTTPFaceitClient creates a FaceitClient that talks to the real Faceit API.
func NewHTTPFaceitClient() *HTTPFaceitClient {
	return &HTTPFaceitClient{
		httpClient:  &http.Client{},
		userInfoURL: UserInfoURL,
	}
}

// faceitUserInfo is the raw JSON shape returned by the Faceit userinfo endpoint.
type faceitUserInfo struct {
	PlayerID string `json:"guid"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"picture"`
	Country  string `json:"locale"`
	Elo      int    `json:"elo"`
	Level    int    `json:"skill_level"`
}

// GetPlayer fetches a player profile. When playerID is "me", it uses the
// userinfo endpoint with the given access token context (set via WithAccessToken).
func (c *HTTPFaceitClient) GetPlayer(ctx context.Context, playerID string) (*testutil.FaceitPlayer, error) {
	var url string
	if playerID == "me" {
		url = c.userInfoURL
	} else {
		url = fmt.Sprintf("https://open.faceit.com/data/v4/players/%s", playerID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating player request: %w", err)
	}

	token := accessTokenFromContext(ctx)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching player: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading player response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("player request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var info faceitUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("decoding player response: %w", err)
	}

	return &testutil.FaceitPlayer{
		PlayerID:   info.PlayerID,
		Nickname:   info.Nickname,
		Avatar:     info.Avatar,
		Country:    info.Country,
		SkillLevel: info.Level,
		FaceitElo:  info.Elo,
	}, nil
}

// GetPlayerHistory fetches match history for a player.
func (c *HTTPFaceitClient) GetPlayerHistory(ctx context.Context, playerID string, offset, limit int) (*testutil.FaceitMatchHistory, error) {
	url := fmt.Sprintf("https://open.faceit.com/data/v4/players/%s/history?game=cs2&offset=%d&limit=%d", playerID, offset, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating history request: %w", err)
	}

	token := accessTokenFromContext(ctx)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching history: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading history response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("history request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result testutil.FaceitMatchHistory
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding history response: %w", err)
	}

	return &result, nil
}

// GetMatchDetails fetches full details for a specific match.
func (c *HTTPFaceitClient) GetMatchDetails(ctx context.Context, matchID string) (*testutil.FaceitMatchDetails, error) {
	url := fmt.Sprintf("https://open.faceit.com/data/v4/matches/%s", matchID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating match request: %w", err)
	}

	token := accessTokenFromContext(ctx)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching match: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading match response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("match request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result testutil.FaceitMatchDetails
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding match response: %w", err)
	}

	return &result, nil
}

// contextKey is an unexported type for context keys in this package.
type contextKey int

const accessTokenKey contextKey = iota

// WithAccessToken returns a context carrying the given access token.
func WithAccessToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, accessTokenKey, token)
}

// accessTokenFromContext extracts the access token from a context.
func accessTokenFromContext(ctx context.Context) string {
	v, _ := ctx.Value(accessTokenKey).(string)
	return v
}
