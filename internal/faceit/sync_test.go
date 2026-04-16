package faceit

import (
	"context"
	"fmt"
	"testing"

	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

func newTestSyncService(t *testing.T, mock *testutil.MockFaceitClient) (*SyncService, *store.Queries, store.User) {
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
		mock := &testutil.MockFaceitClient{}
		svc, _, user := newTestSyncService(t, mock)

		inserted, err := svc.SyncMatches(context.Background(), user.ID, "faceit-123", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inserted != 0 {
			t.Errorf("inserted = %d, want 0", inserted)
		}
	})

	t.Run("new matches", func(t *testing.T) {
		mock := &testutil.MockFaceitClient{
			GetPlayerHistoryFn: func(_ context.Context, _ string, offset, limit int) (*testutil.FaceitMatchHistory, error) {
				if offset > 0 {
					return &testutil.FaceitMatchHistory{}, nil
				}
				return &testutil.FaceitMatchHistory{
					Items: []testutil.FaceitMatchSummary{
						{MatchID: "m1", Map: "de_dust2", StartedAt: 1712700000, Winner: "win", Score: map[string]int{"team": 16, "opponent": 10}},
						{MatchID: "m2", Map: "de_mirage", StartedAt: 1712786400, Winner: "loss", Score: map[string]int{"team": 14, "opponent": 16}},
					},
				}, nil
			},
			GetMatchDetailsFn: func(_ context.Context, matchID string) (*testutil.FaceitMatchDetails, error) {
				return &testutil.FaceitMatchDetails{
					MatchID: matchID,
					DemoURL: []string{"https://example.com/" + matchID + ".dem.gz"},
				}, nil
			},
		}

		svc, q, user := newTestSyncService(t, mock)
		inserted, err := svc.SyncMatches(context.Background(), user.ID, "faceit-123", nil)
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
	})

	t.Run("skip existing", func(t *testing.T) {
		mock := &testutil.MockFaceitClient{
			GetPlayerHistoryFn: func(_ context.Context, _ string, _, _ int) (*testutil.FaceitMatchHistory, error) {
				return &testutil.FaceitMatchHistory{
					Items: []testutil.FaceitMatchSummary{
						{MatchID: "existing-1", Map: "de_dust2", StartedAt: 1712700000, Winner: "win", Score: map[string]int{"team": 16, "opponent": 10}},
						{MatchID: "new-1", Map: "de_mirage", StartedAt: 1712786400, Winner: "loss", Score: map[string]int{"team": 14, "opponent": 16}},
					},
				}, nil
			},
		}

		svc, q, user := newTestSyncService(t, mock)
		ctx := context.Background()

		// Pre-insert one match.
		_, err := q.CreateFaceitMatch(ctx, store.CreateFaceitMatchParams{
			UserID: user.ID, FaceitMatchID: "existing-1", MapName: "de_dust2",
			ScoreTeam: 16, ScoreOpponent: 10, Result: "win",
			PlayedAt: "2024-04-10T10:00:00Z",
		})
		if err != nil {
			t.Fatalf("CreateFaceitMatch: %v", err)
		}

		inserted, err := svc.SyncMatches(ctx, user.ID, "faceit-123", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inserted != 1 {
			t.Errorf("inserted = %d, want 1 (should skip existing)", inserted)
		}
	})

	t.Run("progress callback", func(t *testing.T) {
		mock := &testutil.MockFaceitClient{
			GetPlayerHistoryFn: func(_ context.Context, _ string, _, _ int) (*testutil.FaceitMatchHistory, error) {
				return &testutil.FaceitMatchHistory{
					Items: []testutil.FaceitMatchSummary{
						{MatchID: "m1", Map: "de_dust2", StartedAt: 1712700000, Winner: "win", Score: map[string]int{"team": 16, "opponent": 10}},
					},
				}, nil
			},
		}

		svc, _, user := newTestSyncService(t, mock)
		var progressCalls []struct{ current, total int }
		_, err := svc.SyncMatches(context.Background(), user.ID, "faceit-123", func(current, total int) {
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
		mock := &testutil.MockFaceitClient{
			GetPlayerHistoryFn: func(_ context.Context, _ string, _, _ int) (*testutil.FaceitMatchHistory, error) {
				return nil, fmt.Errorf("API unavailable")
			},
		}

		svc, _, user := newTestSyncService(t, mock)
		_, err := svc.SyncMatches(context.Background(), user.ID, "faceit-123", nil)
		if err == nil {
			t.Fatal("expected error from API failure")
		}
	})
}
