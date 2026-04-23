package faceit

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ok2ju/oversite/internal/store"
)

// SyncService fetches Faceit match history and upserts it into the local database.
type SyncService struct {
	faceit  FaceitClient
	queries *store.Queries
}

// NewSyncService creates a new SyncService.
func NewSyncService(faceit FaceitClient, queries *store.Queries) *SyncService {
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

	// Matches stored before stats tracking was added have adr=0 and need
	// a one-time backfill of per-player kills/deaths/assists/adr.
	missingADRIDs, err := s.queries.GetFaceitMatchIDsMissingADR(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("getting matches missing ADR: %w", err)
	}
	needsBackfill := make(map[string]struct{}, len(missingADRIDs))
	for _, id := range missingADRIDs {
		needsBackfill[id] = struct{}{}
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
			// Determine win/loss by finding the user's faction.
			userFaction := findUserFaction(item.Teams, faceitID)
			result := "L"
			if userFaction != "" && item.Winner == userFaction {
				result = "W"
			}

			// Extract scores relative to the user's faction.
			var scoreTeam, scoreOpponent int64
			if userFaction != "" && item.Score != nil {
				scoreTeam = int64(item.Score[userFaction])
				for faction, score := range item.Score {
					if faction != userFaction {
						scoreOpponent = int64(score)
						break
					}
				}
			}

			if _, ok := existing[item.MatchID]; ok {
				// Repair existing matches that may have bad data from
				// a previous sync (e.g. wrong scores or results).
				_ = s.queries.UpdateMatchScoreResult(ctx, store.UpdateMatchScoreResultParams{
					UserID:        userID,
					FaceitMatchID: item.MatchID,
					ScoreTeam:     scoreTeam,
					ScoreOpponent: scoreOpponent,
					Result:        result,
				})

				// Backfill per-player stats (kills/deaths/assists/adr) for
				// matches that predate stats tracking.
				if _, needs := needsBackfill[item.MatchID]; needs {
					stats, statsErr := s.faceit.GetMatchStats(ctx, item.MatchID, faceitID)
					if statsErr != nil {
						slog.Warn("faceit match stats backfill failed", "match_id", item.MatchID, "err", statsErr)
					} else if stats != nil {
						if err := s.queries.UpdateMatchStats(ctx, store.UpdateMatchStatsParams{
							UserID:        userID,
							FaceitMatchID: item.MatchID,
							Kills:         int64(stats.Kills),
							Deaths:        int64(stats.Deaths),
							Assists:       int64(stats.Assists),
							Adr:           stats.ADR,
						}); err != nil {
							slog.Warn("faceit match stats update failed", "match_id", item.MatchID, "err", err)
						}
					}
				}
				continue
			}

			// Fetch match details for demo URL and map name.
			var demoURL string
			var mapName string
			details, detailsErr := s.faceit.GetMatchDetails(ctx, item.MatchID)
			if detailsErr == nil && details != nil {
				if len(details.DemoURL) > 0 {
					demoURL = details.DemoURL[0]
				}
				mapName = details.Map
			}

			playedAt := time.Unix(item.StartedAt, 0).UTC().Format(time.RFC3339)

			// Fetch per-player match stats (kills/deaths/assists/adr).
			var kills, deaths, assists int64
			var adr float64
			stats, statsErr := s.faceit.GetMatchStats(ctx, item.MatchID, faceitID)
			if statsErr == nil && stats != nil {
				kills = int64(stats.Kills)
				deaths = int64(stats.Deaths)
				assists = int64(stats.Assists)
				adr = stats.ADR
			} else if statsErr != nil {
				slog.Warn("faceit match stats unavailable", "match_id", item.MatchID, "err", statsErr)
			}

			_, err = s.queries.UpsertFaceitMatch(ctx, store.UpsertFaceitMatchParams{
				UserID:        userID,
				FaceitMatchID: item.MatchID,
				MapName:       mapName,
				ScoreTeam:     scoreTeam,
				ScoreOpponent: scoreOpponent,
				Result:        result,
				Kills:         kills,
				Deaths:        deaths,
				Assists:       assists,
				Adr:           adr,
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

// findUserFaction returns the faction ID (e.g., "faction1") that the given
// player belongs to, or "" if not found.
func findUserFaction(teams map[string][]string, faceitID string) string {
	for faction, playerIDs := range teams {
		for _, pid := range playerIDs {
			if pid == faceitID {
				return faction
			}
		}
	}
	return ""
}
