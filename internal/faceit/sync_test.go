package faceit

import (
	"context"
	"fmt"
	"testing"

	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

// resultOf returns the Result stored in the DB for a given match (by faceit_match_id).
func resultOf(matches []store.FaceitMatch, faceitMatchID string) string {
	for _, m := range matches {
		if m.FaceitMatchID == faceitMatchID {
			return m.Result
		}
	}
	return ""
}

// matchByID finds a match by faceit_match_id in a slice.
func matchByID(matches []store.FaceitMatch, faceitMatchID string) *store.FaceitMatch {
	for i := range matches {
		if matches[i].FaceitMatchID == faceitMatchID {
			return &matches[i]
		}
	}
	return nil
}

func newTestSyncService(t *testing.T, mock *MockFaceitClient) (*SyncService, *store.Queries, store.User) {
	t.Helper()
	q, _ := testutil.NewTestQueries(t)
	ctx := context.Background()
	user, err := q.CreateUser(ctx, store.CreateUserParams{
		FaceitID: "faceit-123", Nickname: "tester",
		AvatarUrl: "", FaceitElo: 2000, FaceitLevel: 10, Country: "US",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	svc := NewSyncService(mock, q)
	return svc, q, user
}

func TestSyncMatches(t *testing.T) {
	t.Run("empty history", func(t *testing.T) {
		mock := &MockFaceitClient{}
		svc, _, user := newTestSyncService(t, mock)

		inserted, err := svc.SyncMatches(context.Background(), user.ID, "faceit-123", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inserted != 0 {
			t.Errorf("inserted = %d, want 0", inserted)
		}
	})

	t.Run("new matches with stats", func(t *testing.T) {
		const playerID = "player-abc"
		mock := &MockFaceitClient{
			GetPlayerHistoryFn: func(_ context.Context, _ string, offset, limit int) (*FaceitMatchHistory, error) {
				if offset > 0 {
					return &FaceitMatchHistory{}, nil
				}
				return &FaceitMatchHistory{
					Items: []FaceitMatchSummary{
						{
							MatchID: "m1", StartedAt: 1712700000,
							Winner: "faction1",
							Score:  map[string]int{"faction1": 16, "faction2": 10},
							Teams:  map[string][]string{"faction1": {playerID, "other-1"}, "faction2": {"opp-1", "opp-2"}},
						},
						{
							MatchID: "m2", StartedAt: 1712786400,
							Winner: "faction2",
							Score:  map[string]int{"faction1": 14, "faction2": 16},
							Teams:  map[string][]string{"faction1": {playerID, "other-1"}, "faction2": {"opp-1", "opp-2"}},
						},
					},
				}, nil
			},
			GetMatchDetailsFn: func(_ context.Context, matchID string) (*FaceitMatchDetails, error) {
				return &FaceitMatchDetails{
					MatchID: matchID,
					Map:     "de_dust2",
					DemoURL: []string{"https://example.com/" + matchID + ".dem.gz"},
				}, nil
			},
			GetMatchStatsFn: func(_ context.Context, matchID string, _ string) (*FaceitPlayerMatchStats, error) {
				switch matchID {
				case "m1":
					return &FaceitPlayerMatchStats{Kills: 25, Deaths: 15, Assists: 5, Headshots: 10}, nil
				case "m2":
					return &FaceitPlayerMatchStats{Kills: 18, Deaths: 20, Assists: 3, Headshots: 7}, nil
				}
				return nil, fmt.Errorf("unknown match")
			},
		}

		svc, q, user := newTestSyncService(t, mock)
		inserted, err := svc.SyncMatches(context.Background(), user.ID, playerID, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inserted != 2 {
			t.Errorf("inserted = %d, want 2", inserted)
		}

		// Verify matches in DB.
		matches, err := q.GetFaceitMatchesByUserID(context.Background(), store.GetFaceitMatchesByUserIDParams{
			UserID: user.ID, LimitVal: 10,
		})
		if err != nil {
			t.Fatalf("GetFaceitMatchesByUserID: %v", err)
		}
		if len(matches) != 2 {
			t.Fatalf("len(matches) = %d, want 2", len(matches))
		}

		// Verify demo URL was fetched.
		if matches[0].DemoUrl == "" {
			t.Error("expected demo URL to be set")
		}
		// Verify map name comes from match details.
		if matches[0].MapName != "de_dust2" {
			t.Errorf("map = %q, want %q", matches[0].MapName, "de_dust2")
		}
		// Verify result is stored as "W"/"L" based on faction comparison.
		if r := resultOf(matches, "m1"); r != "W" {
			t.Errorf("m1 result = %q, want %q", r, "W")
		}
		if r := resultOf(matches, "m2"); r != "L" {
			t.Errorf("m2 result = %q, want %q", r, "L")
		}

		// Verify K/D/A was stored from match stats.
		m1 := matchByID(matches, "m1")
		if m1.Kills != 25 {
			t.Errorf("m1 kills = %d, want 25", m1.Kills)
		}
		if m1.Deaths != 15 {
			t.Errorf("m1 deaths = %d, want 15", m1.Deaths)
		}
		if m1.Assists != 5 {
			t.Errorf("m1 assists = %d, want 5", m1.Assists)
		}

		m2 := matchByID(matches, "m2")
		if m2.Kills != 18 {
			t.Errorf("m2 kills = %d, want 18", m2.Kills)
		}
		if m2.Deaths != 20 {
			t.Errorf("m2 deaths = %d, want 20", m2.Deaths)
		}
	})

	t.Run("skip existing", func(t *testing.T) {
		const playerID = "player-abc"
		mock := &MockFaceitClient{
			GetPlayerHistoryFn: func(_ context.Context, _ string, _, _ int) (*FaceitMatchHistory, error) {
				return &FaceitMatchHistory{
					Items: []FaceitMatchSummary{
						{
							MatchID: "existing-1", StartedAt: 1712700000,
							Winner: "faction1",
							Score:  map[string]int{"faction1": 16, "faction2": 10},
							Teams:  map[string][]string{"faction1": {playerID}, "faction2": {"opp-1"}},
						},
						{
							MatchID: "new-1", StartedAt: 1712786400,
							Winner: "faction2",
							Score:  map[string]int{"faction1": 14, "faction2": 16},
							Teams:  map[string][]string{"faction1": {playerID}, "faction2": {"opp-1"}},
						},
					},
				}, nil
			},
		}

		svc, q, user := newTestSyncService(t, mock)
		ctx := context.Background()

		// Pre-insert one match.
		_, err := q.CreateFaceitMatch(ctx, store.CreateFaceitMatchParams{
			UserID: user.ID, FaceitMatchID: "existing-1", MapName: "de_dust2",
			ScoreTeam: 16, ScoreOpponent: 10, Result: "W",
			PlayedAt: "2024-04-10T10:00:00Z",
		})
		if err != nil {
			t.Fatalf("CreateFaceitMatch: %v", err)
		}

		inserted, err := svc.SyncMatches(ctx, user.ID, playerID, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inserted != 1 {
			t.Errorf("inserted = %d, want 1 (should skip existing)", inserted)
		}
	})

	t.Run("progress callback", func(t *testing.T) {
		const playerID = "player-abc"
		mock := &MockFaceitClient{
			GetPlayerHistoryFn: func(_ context.Context, _ string, _, _ int) (*FaceitMatchHistory, error) {
				return &FaceitMatchHistory{
					Items: []FaceitMatchSummary{
						{
							MatchID: "m1", StartedAt: 1712700000,
							Winner: "faction1",
							Score:  map[string]int{"faction1": 16, "faction2": 10},
							Teams:  map[string][]string{"faction1": {playerID}, "faction2": {"opp-1"}},
						},
					},
				}, nil
			},
		}

		svc, _, user := newTestSyncService(t, mock)
		var progressCalls []struct{ current, total int }
		_, err := svc.SyncMatches(context.Background(), user.ID, playerID, func(current, total int) {
			progressCalls = append(progressCalls, struct{ current, total int }{current, total})
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(progressCalls) == 0 {
			t.Error("expected progress callback to be called")
		}
	})

	t.Run("API error", func(t *testing.T) {
		mock := &MockFaceitClient{
			GetPlayerHistoryFn: func(_ context.Context, _ string, _, _ int) (*FaceitMatchHistory, error) {
				return nil, fmt.Errorf("API unavailable")
			},
		}

		svc, _, user := newTestSyncService(t, mock)
		_, err := svc.SyncMatches(context.Background(), user.ID, "faceit-123", nil)
		if err == nil {
			t.Fatal("expected error from API failure")
		}
	})

	t.Run("stats fetch failure non-fatal", func(t *testing.T) {
		const playerID = "player-abc"
		mock := &MockFaceitClient{
			GetPlayerHistoryFn: func(_ context.Context, _ string, _, _ int) (*FaceitMatchHistory, error) {
				return &FaceitMatchHistory{
					Items: []FaceitMatchSummary{
						{
							MatchID: "m1", StartedAt: 1712700000,
							Winner: "faction1",
							Score:  map[string]int{"faction1": 16, "faction2": 10},
							Teams:  map[string][]string{"faction1": {playerID}, "faction2": {"opp-1"}},
						},
					},
				}, nil
			},
			GetMatchStatsFn: func(_ context.Context, _ string, _ string) (*FaceitPlayerMatchStats, error) {
				return nil, fmt.Errorf("stats unavailable")
			},
		}

		svc, q, user := newTestSyncService(t, mock)
		inserted, err := svc.SyncMatches(context.Background(), user.ID, playerID, nil)
		if err != nil {
			t.Fatalf("sync should succeed even when stats fail: %v", err)
		}
		if inserted != 1 {
			t.Errorf("inserted = %d, want 1", inserted)
		}

		// Match should be inserted with zero stats (non-fatal).
		matches, err := q.GetFaceitMatchesByUserID(context.Background(), store.GetFaceitMatchesByUserIDParams{
			UserID: user.ID, LimitVal: 10,
		})
		if err != nil {
			t.Fatalf("GetFaceitMatchesByUserID: %v", err)
		}
		if len(matches) != 1 {
			t.Fatalf("len(matches) = %d, want 1", len(matches))
		}
		if matches[0].Kills != 0 {
			t.Errorf("kills = %d, want 0 (stats fetch failed)", matches[0].Kills)
		}
	})
}
