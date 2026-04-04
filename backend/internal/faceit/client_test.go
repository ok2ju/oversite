package faceit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func testClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c := NewClient(http.DefaultClient, nil, ClientConfig{
		APIKey:  "test-api-key",
		BaseURL: srv.URL,
	})
	c.baseDelay = 10 * time.Millisecond
	c.maxRetries = 3
	return c
}

func TestGetPlayer_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/players/player-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(Player{
			PlayerID: "player-123",
			Nickname: "testplayer",
			Avatar:   "https://avatar.example.com/img.jpg",
			Country:  "US",
			Games: map[string]Game{
				"cs2": {GameID: "cs2", Region: "EU", SkillLevel: 8, FaceitElo: 1850},
			},
		})
	})

	c := testClient(t, handler)
	p, err := c.GetPlayer(context.Background(), "player-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.PlayerID != "player-123" {
		t.Errorf("PlayerID: got %q, want %q", p.PlayerID, "player-123")
	}
	if p.Nickname != "testplayer" {
		t.Errorf("Nickname: got %q, want %q", p.Nickname, "testplayer")
	}
	if p.Country != "US" {
		t.Errorf("Country: got %q, want %q", p.Country, "US")
	}
	g, ok := p.Games["cs2"]
	if !ok {
		t.Fatal("expected cs2 game entry")
	}
	if g.FaceitElo != 1850 {
		t.Errorf("FaceitElo: got %d, want %d", g.FaceitElo, 1850)
	}
	if g.SkillLevel != 8 {
		t.Errorf("SkillLevel: got %d, want %d", g.SkillLevel, 8)
	}
}

func TestGetPlayer_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	c := testClient(t, handler)
	_, err := c.GetPlayer(context.Background(), "nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetPlayer_MalformedJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{invalid json`))
	})

	c := testClient(t, handler)
	_, err := c.GetPlayer(context.Background(), "player-123")
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestGetPlayerHistory_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(MatchHistory{
			Items: []MatchSummary{
				{
					MatchID:   "match-1",
					GameID:    "cs2",
					GameMode:  "5v5",
					StartedAt: 1700000000,
					Results:   MatchResults{Winner: "faction1", Score: map[string]int{"faction1": 13, "faction2": 7}},
				},
				{
					MatchID:  "match-2",
					GameID:   "cs2",
					GameMode: "5v5",
				},
			},
			Start: 0,
			End:   2,
		})
	})

	c := testClient(t, handler)
	h, err := c.GetPlayerHistory(context.Background(), "player-123", 0, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(h.Items) != 2 {
		t.Fatalf("Items length: got %d, want 2", len(h.Items))
	}
	if h.Items[0].MatchID != "match-1" {
		t.Errorf("Items[0].MatchID: got %q, want %q", h.Items[0].MatchID, "match-1")
	}
	if h.Items[0].Results.Winner != "faction1" {
		t.Errorf("Items[0].Results.Winner: got %q, want %q", h.Items[0].Results.Winner, "faction1")
	}
}

func TestGetPlayerHistory_Pagination(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("game") != "cs2" {
			t.Errorf("game param: got %q, want %q", q.Get("game"), "cs2")
		}
		if q.Get("offset") != "10" {
			t.Errorf("offset param: got %q, want %q", q.Get("offset"), "10")
		}
		if q.Get("limit") != "5" {
			t.Errorf("limit param: got %q, want %q", q.Get("limit"), "5")
		}
		_ = json.NewEncoder(w).Encode(MatchHistory{Start: 10, End: 15})
	})

	c := testClient(t, handler)
	_, err := c.GetPlayerHistory(context.Background(), "player-123", 10, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetMatchDetails_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/matches/match-abc" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(MatchDetails{
			MatchID: "match-abc",
			GameID:  "cs2",
			Region:  "EU",
			Status:  "finished",
			DemoURL: []string{"https://demos.example.com/match-abc.dem.gz"},
			Results: MatchResults{Winner: "faction2", Score: map[string]int{"faction1": 10, "faction2": 13}},
		})
	})

	c := testClient(t, handler)
	m, err := c.GetMatchDetails(context.Background(), "match-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.MatchID != "match-abc" {
		t.Errorf("MatchID: got %q, want %q", m.MatchID, "match-abc")
	}
	if m.Status != "finished" {
		t.Errorf("Status: got %q, want %q", m.Status, "finished")
	}
	if len(m.DemoURL) != 1 {
		t.Fatalf("DemoURL length: got %d, want 1", len(m.DemoURL))
	}
	if m.DemoURL[0] != "https://demos.example.com/match-abc.dem.gz" {
		t.Errorf("DemoURL[0]: got %q", m.DemoURL[0])
	}
	if m.Results.Winner != "faction2" {
		t.Errorf("Results.Winner: got %q, want %q", m.Results.Winner, "faction2")
	}
}

func TestGetMatchDetails_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	c := testClient(t, handler)
	_, err := c.GetMatchDetails(context.Background(), "nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRateLimit_429Backoff(t *testing.T) {
	var calls atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode(Player{PlayerID: "player-123", Nickname: "retried"})
	})

	c := testClient(t, handler)
	p, err := c.GetPlayer(context.Background(), "player-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Nickname != "retried" {
		t.Errorf("Nickname: got %q, want %q", p.Nickname, "retried")
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 calls, got %d", calls.Load())
	}
}

func TestRateLimit_429ExhaustedRetries(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})

	c := testClient(t, handler)
	_, err := c.GetPlayer(context.Background(), "player-123")
	if err != ErrRateLimited {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

func TestAuthorizationHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("Authorization header: got %q, want %q", auth, "Bearer test-api-key")
		}
		_ = json.NewEncoder(w).Encode(Player{PlayerID: "player-123"})
	})

	c := testClient(t, handler)
	_, err := c.GetPlayer(context.Background(), "player-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequestContext_Cancelled(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(Player{PlayerID: "player-123"})
	})

	c := testClient(t, handler)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.GetPlayer(ctx, "player-123")
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}
