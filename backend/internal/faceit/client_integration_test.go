//go:build integration

package faceit_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ok2ju/oversite/backend/internal/faceit"
	"github.com/ok2ju/oversite/backend/internal/testutil"
)

func setupRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	ctx := context.Background()

	container, redisURL, err := testutil.RedisContainer(ctx)
	if err != nil {
		t.Fatalf("starting redis container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("parsing redis URL: %v", err)
	}
	client := redis.NewClient(opts)
	t.Cleanup(func() { _ = client.Close() })

	return client
}

func testClientWithRedis(t *testing.T, handler http.Handler, redisClient *redis.Client, cfg faceit.ClientConfig) *faceit.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg.BaseURL = srv.URL
	if cfg.APIKey == "" {
		cfg.APIKey = "test-api-key"
	}
	cfg.BaseDelay = 10 * time.Millisecond
	cfg.MaxRetries = 3

	return faceit.NewClient(http.DefaultClient, redisClient, cfg)
}

func TestCache_PlayerProfile(t *testing.T) {
	tests := []struct {
		name      string
		ttl       time.Duration
		sleepTTL  time.Duration
		wantCalls int32
	}{
		{
			name:      "hit on second call",
			ttl:       5 * time.Second,
			wantCalls: 1,
		},
		{
			name:      "miss on cold cache",
			ttl:       5 * time.Second,
			wantCalls: 1,
		},
		{
			name:      "refetch after TTL expiry",
			ttl:       100 * time.Millisecond,
			sleepTTL:  200 * time.Millisecond,
			wantCalls: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			redisClient := setupRedisClient(t)

			var calls atomic.Int32
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				_ = json.NewEncoder(w).Encode(faceit.Player{PlayerID: "player-1", Nickname: "cached"})
			})

			c := testClientWithRedis(t, handler, redisClient, faceit.ClientConfig{PlayerTTL: tc.ttl})
			ctx := context.Background()

			p1, err := c.GetPlayer(ctx, "player-1")
			if err != nil {
				t.Fatalf("first call: %v", err)
			}
			if p1.Nickname != "cached" {
				t.Errorf("Nickname: got %q, want %q", p1.Nickname, "cached")
			}

			if tc.sleepTTL > 0 {
				time.Sleep(tc.sleepTTL)
			}

			_, err = c.GetPlayer(ctx, "player-1")
			if err != nil {
				t.Fatalf("second call: %v", err)
			}

			if calls.Load() != tc.wantCalls {
				t.Errorf("API calls: got %d, want %d", calls.Load(), tc.wantCalls)
			}
		})
	}
}

func TestCache_MatchHistory(t *testing.T) {
	tests := []struct {
		name      string
		ttl       time.Duration
		sleepTTL  time.Duration
		wantCalls int32
	}{
		{
			name:      "hit on second call",
			ttl:       5 * time.Second,
			wantCalls: 1,
		},
		{
			name:      "refetch after TTL expiry",
			ttl:       100 * time.Millisecond,
			sleepTTL:  200 * time.Millisecond,
			wantCalls: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			redisClient := setupRedisClient(t)

			var calls atomic.Int32
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				_ = json.NewEncoder(w).Encode(faceit.MatchHistory{
					Items: []faceit.MatchSummary{{MatchID: "m-1"}},
					Start: 0,
					End:   1,
				})
			})

			c := testClientWithRedis(t, handler, redisClient, faceit.ClientConfig{HistoryTTL: tc.ttl})
			ctx := context.Background()

			_, err := c.GetPlayerHistory(ctx, "player-1", 0, 20)
			if err != nil {
				t.Fatalf("first call: %v", err)
			}

			if tc.sleepTTL > 0 {
				time.Sleep(tc.sleepTTL)
			}

			_, err = c.GetPlayerHistory(ctx, "player-1", 0, 20)
			if err != nil {
				t.Fatalf("second call: %v", err)
			}

			if calls.Load() != tc.wantCalls {
				t.Errorf("API calls: got %d, want %d", calls.Load(), tc.wantCalls)
			}
		})
	}
}

func TestCache_MatchDetails(t *testing.T) {
	redisClient := setupRedisClient(t)

	var calls atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(w).Encode(faceit.MatchDetails{MatchID: "match-abc", Status: "finished"})
	})

	c := testClientWithRedis(t, handler, redisClient, faceit.ClientConfig{MatchTTL: 5 * time.Second})
	ctx := context.Background()

	_, err := c.GetMatchDetails(ctx, "match-abc")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	_, err = c.GetMatchDetails(ctx, "match-abc")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if calls.Load() != 1 {
		t.Errorf("expected 1 API call, got %d", calls.Load())
	}
}

func TestCache_KeyFormat(t *testing.T) {
	redisClient := setupRedisClient(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/players/p-1":
			_ = json.NewEncoder(w).Encode(faceit.Player{PlayerID: "p-1"})
		case r.URL.Path == "/players/p-1/history":
			_ = json.NewEncoder(w).Encode(faceit.MatchHistory{Items: []faceit.MatchSummary{{MatchID: "m-1"}}})
		case r.URL.Path == "/matches/m-1":
			_ = json.NewEncoder(w).Encode(faceit.MatchDetails{MatchID: "m-1"})
		}
	})

	c := testClientWithRedis(t, handler, redisClient, faceit.ClientConfig{
		PlayerTTL:  5 * time.Second,
		HistoryTTL: 5 * time.Second,
		MatchTTL:   5 * time.Second,
	})
	ctx := context.Background()

	c.GetPlayer(ctx, "p-1")
	c.GetPlayerHistory(ctx, "p-1", 0, 20)
	c.GetMatchDetails(ctx, "m-1")

	keys := []struct {
		key  string
		desc string
	}{
		{"faceit:player:p-1", "player cache key"},
		{"faceit:history:p-1:0:20", "history cache key"},
		{"faceit:match:m-1", "match cache key"},
	}

	for _, k := range keys {
		exists := redisClient.Exists(ctx, k.key).Val()
		if exists != 1 {
			t.Errorf("%s %q not found in Redis", k.desc, k.key)
		}
	}
}
