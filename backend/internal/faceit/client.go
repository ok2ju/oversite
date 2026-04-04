package faceit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// DefaultBaseURL is the base URL for the Faceit Data API v4.
const DefaultBaseURL = "https://open.faceit.com/data/v4"

// Default cache TTLs.
const (
	DefaultPlayerCacheTTL  = 15 * time.Minute
	DefaultHistoryCacheTTL = 5 * time.Minute
	DefaultMatchCacheTTL   = 1 * time.Hour
)

// Sentinel errors.
var (
	ErrNotFound    = errors.New("faceit: resource not found")
	ErrRateLimited = errors.New("faceit: rate limited")
	ErrAPI         = errors.New("faceit: api error")
)

// FaceitAPI defines the interface for the Faceit Data API client.
type FaceitAPI interface {
	GetPlayer(ctx context.Context, playerID string) (*Player, error)
	GetPlayerHistory(ctx context.Context, playerID string, offset, limit int) (*MatchHistory, error)
	GetMatchDetails(ctx context.Context, matchID string) (*MatchDetails, error)
}

// ClientConfig holds configuration for the Faceit API client.
type ClientConfig struct {
	APIKey  string
	BaseURL string

	PlayerTTL  time.Duration
	HistoryTTL time.Duration
	MatchTTL   time.Duration
}

// Client implements FaceitAPI with HTTP calls, retry logic, and Redis caching.
type Client struct {
	httpClient *http.Client
	redis      *redis.Client
	apiKey     string
	baseURL    string

	playerTTL  time.Duration
	historyTTL time.Duration
	matchTTL   time.Duration

	baseDelay  time.Duration
	maxRetries int
}

// NewClient creates a new Faceit API client. Pass nil for redisClient to disable caching.
func NewClient(httpClient *http.Client, redisClient *redis.Client, cfg ClientConfig) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	playerTTL := cfg.PlayerTTL
	if playerTTL == 0 {
		playerTTL = DefaultPlayerCacheTTL
	}

	historyTTL := cfg.HistoryTTL
	if historyTTL == 0 {
		historyTTL = DefaultHistoryCacheTTL
	}

	matchTTL := cfg.MatchTTL
	if matchTTL == 0 {
		matchTTL = DefaultMatchCacheTTL
	}

	return &Client{
		httpClient: httpClient,
		redis:      redisClient,
		apiKey:     cfg.APIKey,
		baseURL:    baseURL,
		playerTTL:  playerTTL,
		historyTTL: historyTTL,
		matchTTL:   matchTTL,
		baseDelay:  1 * time.Second,
		maxRetries: 3,
	}
}

// GetPlayer fetches a player profile by ID.
func (c *Client) GetPlayer(ctx context.Context, playerID string) (*Player, error) {
	cacheKey := "faceit:player:" + playerID

	var cached Player
	if hit, _ := c.cacheGet(ctx, cacheKey, &cached); hit {
		return &cached, nil
	}

	var result Player
	if err := c.doRequest(ctx, "/players/"+playerID, &result); err != nil {
		return nil, err
	}

	c.cacheSet(ctx, cacheKey, &result, c.playerTTL)
	return &result, nil
}

// GetPlayerHistory fetches paginated match history for a player (CS2 only).
func (c *Client) GetPlayerHistory(ctx context.Context, playerID string, offset, limit int) (*MatchHistory, error) {
	cacheKey := fmt.Sprintf("faceit:history:%s:%d:%d", playerID, offset, limit)

	var cached MatchHistory
	if hit, _ := c.cacheGet(ctx, cacheKey, &cached); hit {
		return &cached, nil
	}

	path := fmt.Sprintf("/players/%s/history?game=cs2&offset=%d&limit=%d", playerID, offset, limit)
	var result MatchHistory
	if err := c.doRequest(ctx, path, &result); err != nil {
		return nil, err
	}

	c.cacheSet(ctx, cacheKey, &result, c.historyTTL)
	return &result, nil
}

// GetMatchDetails fetches full match details by match ID.
func (c *Client) GetMatchDetails(ctx context.Context, matchID string) (*MatchDetails, error) {
	cacheKey := "faceit:match:" + matchID

	var cached MatchDetails
	if hit, _ := c.cacheGet(ctx, cacheKey, &cached); hit {
		return &cached, nil
	}

	var result MatchDetails
	if err := c.doRequest(ctx, "/matches/"+matchID, &result); err != nil {
		return nil, err
	}

	c.cacheSet(ctx, cacheKey, &result, c.matchTTL)
	return &result, nil
}

// doRequest builds an HTTP request, executes it with retry, and unmarshals the response.
func (c *Client) doRequest(ctx context.Context, path string, result interface{}) error {
	var body []byte

	err := c.doWithRetry(ctx, func(ctx context.Context) (int, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
		if err != nil {
			return 0, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close() //nolint:errcheck // best-effort close on read path

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return resp.StatusCode, fmt.Errorf("reading response: %w", err)
		}

		return resp.StatusCode, nil
	})
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

// doWithRetry executes fn with exponential backoff on 429 responses.
// fn returns (statusCode, error). Non-retryable status codes are mapped to sentinel errors.
func (c *Client) doWithRetry(ctx context.Context, fn func(ctx context.Context) (int, error)) error {
	delay := c.baseDelay

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		statusCode, err := fn(ctx)
		if err != nil {
			return err
		}

		switch {
		case statusCode >= 200 && statusCode < 300:
			return nil
		case statusCode == http.StatusNotFound:
			return ErrNotFound
		case statusCode == http.StatusTooManyRequests:
			if attempt == c.maxRetries {
				return ErrRateLimited
			}
			// Wait before retrying
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			delay *= 2
			if delay > 8*time.Second {
				delay = 8 * time.Second
			}
		default:
			return fmt.Errorf("%w: status %d", ErrAPI, statusCode)
		}
	}

	return ErrRateLimited
}

// cacheGet attempts to read a value from Redis cache. Returns (true, nil) on hit.
// Returns (false, nil) on miss or if Redis is nil. Cache errors are logged but never propagated.
func (c *Client) cacheGet(ctx context.Context, key string, dest interface{}) (bool, error) {
	if c.redis == nil {
		return false, nil
	}

	val, err := c.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		slog.Warn("faceit cache get error", "key", key, "error", err)
		return false, nil
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		slog.Warn("faceit cache unmarshal error", "key", key, "error", err)
		return false, nil
	}

	return true, nil
}

// cacheSet writes a value to Redis cache. Errors are logged but never propagated.
func (c *Client) cacheSet(ctx context.Context, key string, val interface{}, ttl time.Duration) {
	if c.redis == nil {
		return
	}

	data, err := json.Marshal(val)
	if err != nil {
		slog.Warn("faceit cache marshal error", "key", key, "error", err)
		return
	}

	if err := c.redis.Set(ctx, key, data, ttl).Err(); err != nil {
		slog.Warn("faceit cache set error", "key", key, "error", err)
	}
}

