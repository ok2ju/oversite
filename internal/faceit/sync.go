package faceit

import (
	"context"
	"fmt"
	"time"

	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

// SyncService fetches Faceit match history and upserts it into the local database.
type SyncService struct {
	faceit  testutil.FaceitClient
	queries *store.Queries
}

// NewSyncService creates a new SyncService.
func NewSyncService(faceit testutil.FaceitClient, queries *store.Queries) *SyncService {
	return &SyncService{faceit: faceit, queries: queries}
}

// SyncMatches fetches match history from Faceit and upserts new matches into SQLite.
// Returns the number of newly inserted matches. The onProgress callback reports
// (current, total) as pages are fetched.
func (s *SyncService) SyncMatches(ctx context.Context, userID int64, faceitID string, onProgress func(current, total int)) (int, error) {
	// Build a set of existing match IDs for O(1) dedup.
	existingIDs, err := s.queries.GetExistingFaceitMatchIDs(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("getting existing match IDs: %w", err)
	}
	existing := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existing[id] = struct{}{}
	}

	const pageSize = 20
	const maxMatches = 200
	inserted := 0

	for offset := 0; offset < maxMatches; offset += pageSize {
		history, err := s.faceit.GetPlayerHistory(ctx, faceitID, offset, pageSize)
		if err != nil {
			return inserted, fmt.Errorf("fetching history (offset %d): %w", offset, err)
		}

		if len(history.Items) == 0 {
			break
		}

		for _, item := range history.Items {
			if _, ok := existing[item.MatchID]; ok {
				continue
			}

			// Optionally fetch match details for demo URL.
			var demoURL string
			details, err := s.faceit.GetMatchDetails(ctx, item.MatchID)
			if err == nil && details != nil && len(details.DemoURL) > 0 {
				demoURL = details.DemoURL[0]
			}

			playedAt := time.Unix(item.StartedAt, 0).UTC().Format(time.RFC3339)

			_, err = s.queries.UpsertFaceitMatch(ctx, store.UpsertFaceitMatchParams{
				UserID:        userID,
				FaceitMatchID: item.MatchID,
				MapName:       item.Map,
				ScoreTeam:     int64(item.Score["team"]),
				ScoreOpponent: int64(item.Score["opponent"]),
				Result:        item.Winner,
				DemoUrl:       demoURL,
				PlayedAt:      playedAt,
			})
			if err != nil {
				return inserted, fmt.Errorf("upserting match %s: %w", item.MatchID, err)
			}
			inserted++
			existing[item.MatchID] = struct{}{}
		}

		if onProgress != nil {
			total := offset + pageSize
			if len(history.Items) < pageSize {
				total = offset + len(history.Items)
			}
			onProgress(offset+len(history.Items), total)
		}

		// Rate limit: 100ms between API calls.
		time.Sleep(100 * time.Millisecond)

		if len(history.Items) < pageSize {
			break
		}
	}

	return inserted, nil
}
