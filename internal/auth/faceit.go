package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/ok2ju/oversite/internal/faceit"
)

// UserInfoURL is the Faceit OpenID Connect userinfo endpoint.
const UserInfoURL = "https://api.faceit.com/auth/v1/resources/userinfo"

// HTTPFaceitClient implements faceit.FaceitClient using real HTTP calls to
// the Faceit Data API.
type HTTPFaceitClient struct {
	httpClient  *http.Client
	userInfoURL string
	apiKey      string // Server API key for open.faceit.com/data/v4 endpoints
}

// NewHTTPFaceitClient creates a FaceitClient that talks to the real Faceit API.
// apiKey is a server-side API key for the Faceit Data API v4 (open.faceit.com).
// When empty, Data API calls fall back to the user's OAuth token (which may 403).
// If transport is nil, http.DefaultTransport is used.
func NewHTTPFaceitClient(apiKey string, transport http.RoundTripper) *HTTPFaceitClient {
	client := &http.Client{}
	if transport != nil {
		client.Transport = transport
	}
	return &HTTPFaceitClient{
		httpClient:  client,
		userInfoURL: UserInfoURL,
		apiKey:      apiKey,
	}
}

// faceitUserInfo is the raw JSON shape returned by the Faceit OIDC userinfo
// endpoint. The elo and skill_level fields may be present depending on the
// OAuth scopes granted — they serve as a fallback when the Data API v4 call
// fails (e.g. when using an OAuth token instead of a server API key).
type faceitUserInfo struct {
	PlayerID string `json:"guid"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"picture"`
	Country  string `json:"locale"`
	Elo      int    `json:"elo"`
	Level    int    `json:"skill_level"`
}

// faceitPlayerResponse is the raw JSON shape from the Data API v4
// GET /players/{id} endpoint. ELO and skill level are nested under games.cs2.
type faceitPlayerResponse struct {
	PlayerID string                          `json:"player_id"`
	Nickname string                          `json:"nickname"`
	Avatar   string                          `json:"avatar"`
	Country  string                          `json:"country"`
	Games    map[string]faceitGameStatsEntry `json:"games"`
}

type faceitGameStatsEntry struct {
	FaceitElo  int `json:"faceit_elo"`
	SkillLevel int `json:"skill_level"`
}

// GetPlayer fetches a player profile. When playerID is "me", it first calls the
// OIDC userinfo endpoint to obtain the player's guid, then calls the Data API v4
// to get the full profile (including elo and skill level). For other IDs, it
// calls the Data API v4 directly.
func (c *HTTPFaceitClient) GetPlayer(ctx context.Context, playerID string) (*faceit.FaceitPlayer, error) {
	// For "me", resolve the guid via the OIDC userinfo endpoint first.
	if playerID == "me" {
		info, err := c.fetchUserInfo(ctx)
		if err != nil {
			return nil, err
		}
		playerID = info.PlayerID

		// Fetch full profile from Data API v4 (for elo/level).
		player, err := c.fetchPlayerV4(ctx, playerID)
		if err != nil {
			// Fall back to userinfo data including elo/level if the v4
			// endpoint rejects the OAuth token (it may require a server
			// API key). The OIDC endpoint often includes these fields.
			slog.Warn("faceit Data API v4 player fetch failed, using OIDC fallback", "err", err)
			return &faceit.FaceitPlayer{
				PlayerID:   info.PlayerID,
				Nickname:   info.Nickname,
				Avatar:     info.Avatar,
				Country:    info.Country,
				FaceitElo:  info.Elo,
				SkillLevel: info.Level,
			}, nil
		}
		return player, nil
	}

	return c.fetchPlayerV4(ctx, playerID)
}

// fetchUserInfo calls the Faceit OIDC userinfo endpoint and returns basic identity fields.
func (c *HTTPFaceitClient) fetchUserInfo(ctx context.Context) (*faceitUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating userinfo request: %w", err)
	}
	token := accessTokenFromContext(ctx)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching userinfo: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading userinfo response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var info faceitUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("decoding userinfo response: %w", err)
	}
	return &info, nil
}

// dataAPIToken returns the bearer token to use for Data API v4 calls.
// Prefers the server API key; falls back to the user's OAuth access token.
func (c *HTTPFaceitClient) dataAPIToken(ctx context.Context) string {
	if c.apiKey != "" {
		return c.apiKey
	}
	return accessTokenFromContext(ctx)
}

// fetchPlayerV4 calls the Data API v4 /players/{id} endpoint and returns the
// full player profile including elo and skill level from the cs2 game entry.
func (c *HTTPFaceitClient) fetchPlayerV4(ctx context.Context, playerID string) (*faceit.FaceitPlayer, error) {
	url := fmt.Sprintf("https://open.faceit.com/data/v4/players/%s", playerID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating player request: %w", err)
	}
	if token := c.dataAPIToken(ctx); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching player: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading player response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("player request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var raw faceitPlayerResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decoding player response: %w", err)
	}

	player := &faceit.FaceitPlayer{
		PlayerID: raw.PlayerID,
		Nickname: raw.Nickname,
		Avatar:   raw.Avatar,
		Country:  raw.Country,
	}
	if cs2, ok := raw.Games["cs2"]; ok {
		player.FaceitElo = cs2.FaceitElo
		player.SkillLevel = cs2.SkillLevel
	}
	return player, nil
}

// GetPlayerLifetimeStats fetches aggregated lifetime stats for a player in CS2.
func (c *HTTPFaceitClient) GetPlayerLifetimeStats(ctx context.Context, playerID string) (*faceit.FaceitLifetimeStats, error) {
	url := fmt.Sprintf("https://open.faceit.com/data/v4/players/%s/stats/cs2", playerID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating player stats request: %w", err)
	}
	if token := c.dataAPIToken(ctx); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching player stats: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading player stats response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("player stats request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var raw faceitPlayerStatsResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decoding player stats response: %w", err)
	}

	result := &faceit.FaceitLifetimeStats{}
	if raw.Lifetime != nil {
		if v, ok := raw.Lifetime["Matches"]; ok {
			if s, ok := v.(string); ok {
				fmt.Sscanf(s, "%d", &result.Matches)
			}
		}
	}
	return result, nil
}

// --- JSON response types for Faceit Data API v4 ---

// faceitPlayerStatsResponse maps the GET /players/{id}/stats/{game} response.
// Lifetime values are mixed types (strings and arrays), so we use interface{}.
type faceitPlayerStatsResponse struct {
	Lifetime map[string]interface{} `json:"lifetime"`
}

// faceitHistoryResponse maps the GET /players/{id}/history response.
type faceitHistoryResponse struct {
	Items []faceitHistoryItem `json:"items"`
}

type faceitHistoryItem struct {
	MatchID    string                       `json:"match_id"`
	StartedAt  int64                        `json:"started_at"`
	FinishedAt int64                        `json:"finished_at"`
	Results    *faceitMatchResults          `json:"results"`
	Teams      map[string]*faceitTeamRoster `json:"teams"`
}

type faceitMatchResults struct {
	Winner string         `json:"winner"`
	Score  map[string]int `json:"score"`
}

type faceitTeamRoster struct {
	Roster []faceitRosterEntry `json:"players"`
}

type faceitRosterEntry struct {
	PlayerID string `json:"player_id"`
}

// faceitMatchDetailResponse maps the GET /matches/{id} response.
type faceitMatchDetailResponse struct {
	MatchID    string            `json:"match_id"`
	DemoURL    []string          `json:"demo_url"`
	StartedAt  int64             `json:"started_at"`
	FinishedAt int64             `json:"finished_at"`
	Voting     *faceitVotingData `json:"voting"`
}

type faceitVotingData struct {
	Map *faceitVotingMap `json:"map"`
}

type faceitVotingMap struct {
	Pick []string `json:"pick"`
}

// GetPlayerHistory fetches match history for a player.
func (c *HTTPFaceitClient) GetPlayerHistory(ctx context.Context, playerID string, offset, limit int) (*faceit.FaceitMatchHistory, error) {
	url := fmt.Sprintf("https://open.faceit.com/data/v4/players/%s/history?game=cs2&offset=%d&limit=%d", playerID, offset, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating history request: %w", err)
	}

	if token := c.dataAPIToken(ctx); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching history: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading history response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("history request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var raw faceitHistoryResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decoding history response: %w", err)
	}

	// Convert raw API response to domain types.
	items := make([]faceit.FaceitMatchSummary, len(raw.Items))
	for i, item := range raw.Items {
		summary := faceit.FaceitMatchSummary{
			MatchID:    item.MatchID,
			StartedAt:  item.StartedAt,
			FinishedAt: item.FinishedAt,
		}
		if item.Results != nil {
			summary.Winner = item.Results.Winner
			summary.Score = item.Results.Score
		}
		if item.Teams != nil {
			summary.Teams = make(map[string][]string, len(item.Teams))
			for faction, team := range item.Teams {
				if team == nil {
					continue
				}
				pids := make([]string, len(team.Roster))
				for j, p := range team.Roster {
					pids[j] = p.PlayerID
				}
				summary.Teams[faction] = pids
			}
		}
		items[i] = summary
	}

	return &faceit.FaceitMatchHistory{Items: items}, nil
}

// GetMatchDetails fetches full details for a specific match.
func (c *HTTPFaceitClient) GetMatchDetails(ctx context.Context, matchID string) (*faceit.FaceitMatchDetails, error) {
	url := fmt.Sprintf("https://open.faceit.com/data/v4/matches/%s", matchID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating match request: %w", err)
	}

	if token := c.dataAPIToken(ctx); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching match: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading match response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("match request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var raw faceitMatchDetailResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decoding match response: %w", err)
	}

	result := &faceit.FaceitMatchDetails{
		MatchID:    raw.MatchID,
		DemoURL:    raw.DemoURL,
		StartedAt:  raw.StartedAt,
		FinishedAt: raw.FinishedAt,
	}

	// Extract map name from voting data.
	if raw.Voting != nil && raw.Voting.Map != nil && len(raw.Voting.Map.Pick) > 0 {
		result.Map = raw.Voting.Map.Pick[0]
	}

	return result, nil
}

// --- Match stats types for /matches/{id}/stats endpoint ---

type faceitMatchStatsResponse struct {
	Rounds []faceitStatsRound `json:"rounds"`
}

type faceitStatsRound struct {
	Teams []faceitStatsTeam `json:"teams"`
}

type faceitStatsTeam struct {
	Players []faceitStatsPlayer `json:"players"`
}

type faceitStatsPlayer struct {
	PlayerID    string            `json:"player_id"`
	PlayerStats map[string]string `json:"player_stats"`
}

// GetMatchStats fetches per-player stats for a match from the Data API v4.
func (c *HTTPFaceitClient) GetMatchStats(ctx context.Context, matchID string, playerID string) (*faceit.FaceitPlayerMatchStats, error) {
	url := fmt.Sprintf("https://open.faceit.com/data/v4/matches/%s/stats", matchID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating match stats request: %w", err)
	}
	if token := c.dataAPIToken(ctx); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching match stats: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading match stats response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("match stats request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var raw faceitMatchStatsResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decoding match stats response: %w", err)
	}

	// Find the player's stats across all rounds (use the last/summary round).
	for _, round := range raw.Rounds {
		for _, team := range round.Teams {
			for _, p := range team.Players {
				if p.PlayerID == playerID {
					return &faceit.FaceitPlayerMatchStats{
						Kills:     parseStatInt(p.PlayerStats, "Kills"),
						Deaths:    parseStatInt(p.PlayerStats, "Deaths"),
						Assists:   parseStatInt(p.PlayerStats, "Assists"),
						Headshots: parseStatInt(p.PlayerStats, "Headshots"),
					}, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("player %s not found in match %s stats", playerID, matchID)
}

// parseStatInt extracts an integer from a string-keyed stats map.
func parseStatInt(stats map[string]string, key string) int {
	v, ok := stats[key]
	if !ok {
		return 0
	}
	var n int
	fmt.Sscanf(v, "%d", &n)
	return n
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
