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
	api      FaceitAPI
	store    SyncStore
	importer *DemoImporter
}

// NewSyncService creates a new SyncService.
func NewSyncService(api FaceitAPI, store SyncStore) *SyncService {
	return &SyncService{api: api, store: store}
}

// WithAutoImport enables automatic demo downloading during sync.
func (s *SyncService) WithAutoImport(importer *DemoImporter) *SyncService {
	s.importer = importer
	return s
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
	failed := make(map[int]bool)
	for r := range detailsCh {
		if r.err != nil {
			slog.Warn("fetching match details failed, skipping match",
				"match_id", newMatches[r.index].MatchID,
				"error", r.err,
			)
			failed[r.index] = true
			continue
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

	// 7. Filter to processable matches (have details and player in teams)
	type processableMatch struct {
		summary MatchSummary
		details *MatchDetails
		faction string
		player  TeamPlayer
	}
	var processable []processableMatch
	for i, m := range newMatches {
		if failed[i] {
			continue
		}
		faction, tp, found := findPlayerInTeams(m.Teams, faceitID)
		if !found {
			slog.Warn("player not found in match teams, skipping",
				"match_id", m.MatchID,
				"faceit_id", faceitID,
			)
			continue
		}
		processable = append(processable, processableMatch{
			summary: m,
			details: details[i],
			faction: faction,
			player:  tp,
		})
	}

	// 8. Build and upsert each match with correct ELO chain
	inserted := 0
	for i, pm := range processable {
		result := determineResult(pm.summary.Results, pm.faction)
		scoreTeam, scoreOpponent := extractScores(pm.summary.Results, pm.faction)

		// ELO chain: elo_before = player's ELO in this match
		eloBefore := int32(pm.player.FaceitElo)

		// elo_after = elo_before of the next processable match, or current ELO for the last
		var eloAfter int32
		if i < len(processable)-1 {
			eloAfter = int32(processable[i+1].player.FaceitElo)
		} else {
			eloAfter = int32(currentElo)
		}

		mapName := "unknown"
		if pm.details != nil {
			mapName = pm.details.MapName()
		}

		var demoURL sql.NullString
		if pm.details != nil && len(pm.details.DemoURL) > 0 && pm.details.DemoURL[0] != "" {
			demoURL = sql.NullString{String: pm.details.DemoURL[0], Valid: true}
		}

		match, err := s.store.UpsertFaceitMatch(ctx, store.UpsertFaceitMatchParams{
			UserID:        userID,
			FaceitMatchID: pm.summary.MatchID,
			MapName:       mapName,
			ScoreTeam:     int16(scoreTeam),
			ScoreOpponent: int16(scoreOpponent),
			Result:        result,
			EloBefore:     sql.NullInt32{Int32: eloBefore, Valid: true},
			EloAfter:      sql.NullInt32{Int32: eloAfter, Valid: true},
			DemoUrl:       demoURL,
			PlayedAt:      time.Unix(pm.summary.StartedAt, 0),
		})
		if err == sql.ErrNoRows {
			// ON CONFLICT DO NOTHING — already existed, skip import
			inserted++
			continue
		}
		if err != nil {
			return inserted, fmt.Errorf("upserting match %s: %w", pm.summary.MatchID, err)
		}
		inserted++

		// Auto-import demo if enabled and demo URL present
		if s.importer != nil && demoURL.Valid {
			if _, importErr := s.importer.Import(ctx, userID, match.ID, match.FaceitMatchID, demoURL.String, time.Unix(pm.summary.StartedAt, 0)); importErr != nil {
				slog.Warn("auto-import failed, continuing sync",
					"match_id", pm.summary.MatchID,
					"error", importErr,
				)
			}
		}
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
