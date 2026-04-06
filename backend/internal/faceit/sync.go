package faceit

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/store"
)

// SyncStore is the subset of store.Queries needed by SyncService.
type SyncStore interface {
	GetExistingFaceitMatchIDs(ctx context.Context, userID uuid.UUID) ([]string, error)
	UpsertFaceitMatch(ctx context.Context, arg store.UpsertFaceitMatchParams) (store.FaceitMatch, error)
}

// SyncService syncs a user's Faceit match history into the local database.
type SyncService struct {
	api   FaceitAPI
	store SyncStore
}

// NewSyncService creates a new SyncService.
func NewSyncService(api FaceitAPI, store SyncStore) *SyncService {
	return &SyncService{api: api, store: store}
}

// maxPages is the maximum number of history pages to fetch.
const maxPages = 5

// pageSize is the number of matches per page.
const pageSize = 20

// maxConcurrency is the max number of concurrent GetMatchDetails calls.
const maxConcurrency = 5

// Sync fetches the user's Faceit match history and upserts new matches.
// Returns the number of newly inserted matches.
func (s *SyncService) Sync(ctx context.Context, userID uuid.UUID, faceitID string) (int, error) {
	// 1. Fetch match history pages
	var allMatches []MatchSummary
	for page := 0; page < maxPages; page++ {
		history, err := s.api.GetPlayerHistory(ctx, faceitID, page*pageSize, pageSize)
		if err != nil {
			return 0, fmt.Errorf("fetching match history page %d: %w", page, err)
		}
		allMatches = append(allMatches, history.Items...)
		if len(history.Items) < pageSize {
			break // last page
		}
	}

	if len(allMatches) == 0 {
		return 0, nil
	}

	// 2. Get existing match IDs from DB
	existingIDs, err := s.store.GetExistingFaceitMatchIDs(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("getting existing match IDs: %w", err)
	}
	existingSet := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existingSet[id] = struct{}{}
	}

	// 3. Filter to new matches only
	var newMatches []MatchSummary
	for _, m := range allMatches {
		if _, exists := existingSet[m.MatchID]; !exists {
			newMatches = append(newMatches, m)
		}
	}

	if len(newMatches) == 0 {
		return 0, nil
	}

	// 4. Reverse to chronological order (API returns newest-first)
	for i, j := 0, len(newMatches)-1; i < j; i, j = i+1, j-1 {
		newMatches[i], newMatches[j] = newMatches[j], newMatches[i]
	}

	// 5. Fetch match details concurrently with bounded concurrency
	type detailResult struct {
		index   int
		details *MatchDetails
		err     error
	}

	detailsCh := make(chan detailResult, len(newMatches))
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i, m := range newMatches {
		wg.Add(1)
		go func(idx int, matchID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			details, err := s.api.GetMatchDetails(ctx, matchID)
			detailsCh <- detailResult{index: idx, details: details, err: err}
		}(i, m.MatchID)
	}

	wg.Wait()
	close(detailsCh)

	details := make([]*MatchDetails, len(newMatches))
	for r := range detailsCh {
		if r.err != nil {
			return 0, fmt.Errorf("fetching match details for %s: %w", newMatches[r.index].MatchID, r.err)
		}
		details[r.index] = r.details
	}

	// 6. Get current ELO for the player (used as elo_after for the latest match)
	player, err := s.api.GetPlayer(ctx, faceitID)
	if err != nil {
		return 0, fmt.Errorf("fetching player profile: %w", err)
	}
	var currentElo int
	if game, ok := player.Games["cs2"]; ok {
		currentElo = game.FaceitElo
	}

	// 7. Build and upsert each match
	inserted := 0
	for i, m := range newMatches {
		faction, tp, found := findPlayerInTeams(m.Teams, faceitID)
		if !found {
			slog.Warn("player not found in match teams, skipping",
				"match_id", m.MatchID,
				"faceit_id", faceitID,
			)
			continue
		}

		result := determineResult(m.Results, faction)
		scoreTeam, scoreOpponent := extractScores(m.Results, faction)

		// ELO chain: elo_before = player's ELO in this match
		eloBefore := int32(tp.FaceitElo)

		// elo_after = player's ELO in next match, or current ELO for the last match
		var eloAfter int32
		if i < len(newMatches)-1 {
			_, nextTP, nextFound := findPlayerInTeams(newMatches[i+1].Teams, faceitID)
			if nextFound {
				eloAfter = int32(nextTP.FaceitElo)
			} else {
				eloAfter = int32(currentElo)
			}
		} else {
			eloAfter = int32(currentElo)
		}

		mapName := "unknown"
		if details[i] != nil {
			mapName = details[i].MapName()
		}

		var demoURL sql.NullString
		if details[i] != nil && len(details[i].DemoURL) > 0 && details[i].DemoURL[0] != "" {
			demoURL = sql.NullString{String: details[i].DemoURL[0], Valid: true}
		}

		_, err := s.store.UpsertFaceitMatch(ctx, store.UpsertFaceitMatchParams{
			UserID:        userID,
			FaceitMatchID: m.MatchID,
			MapName:       mapName,
			ScoreTeam:     int16(scoreTeam),
			ScoreOpponent: int16(scoreOpponent),
			Result:        result,
			EloBefore:     sql.NullInt32{Int32: eloBefore, Valid: true},
			EloAfter:      sql.NullInt32{Int32: eloAfter, Valid: true},
			DemoUrl:       demoURL,
			PlayedAt:      time.Unix(m.StartedAt, 0),
		})
		// ON CONFLICT DO NOTHING returns sql.ErrNoRows — treat as success
		if err != nil && err != sql.ErrNoRows {
			return inserted, fmt.Errorf("upserting match %s: %w", m.MatchID, err)
		}
		inserted++
	}

	return inserted, nil
}

// findPlayerInTeams locates a player in the match teams by faceit ID.
// Returns the faction name ("faction1" or "faction2"), the player, and whether found.
func findPlayerInTeams(teams map[string]Team, faceitID string) (string, TeamPlayer, bool) {
	for faction, team := range teams {
		for _, p := range team.Players {
			if p.PlayerID == faceitID {
				return faction, p, true
			}
		}
	}
	return "", TeamPlayer{}, false
}

// determineResult returns "W" or "L" based on whether the player's faction won.
func determineResult(results MatchResults, playerFaction string) string {
	if results.Winner == playerFaction {
		return "W"
	}
	return "L"
}

// extractScores returns the player's team score and opponent score.
func extractScores(results MatchResults, playerFaction string) (team, opponent int16) {
	teamScore := int16(results.Score[playerFaction])
	var opponentScore int16
	for faction, score := range results.Score {
		if faction != playerFaction {
			opponentScore = int16(score)
			break
		}
	}
	return teamScore, opponentScore
}
