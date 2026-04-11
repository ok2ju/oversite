package faceit_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/faceit"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// --- Mocks ---

type mockFaceitAPI struct {
	player       *faceit.Player
	playerErr    error
	history      map[string]*faceit.MatchHistory // key: "offset:limit"
	historyErr   error
	details      map[string]*faceit.MatchDetails
	detailsErr   error
	mu           sync.Mutex
	detailsCalls []string
}

func (m *mockFaceitAPI) GetPlayer(_ context.Context, _ string) (*faceit.Player, error) {
	if m.playerErr != nil {
		return nil, m.playerErr
	}
	return m.player, nil
}

func (m *mockFaceitAPI) GetPlayerHistory(_ context.Context, _ string, offset, limit int) (*faceit.MatchHistory, error) {
	if m.historyErr != nil {
		return nil, m.historyErr
	}
	key := historyKey(offset, limit)
	if h, ok := m.history[key]; ok {
		return h, nil
	}
	return &faceit.MatchHistory{}, nil
}

func (m *mockFaceitAPI) GetMatchDetails(_ context.Context, matchID string) (*faceit.MatchDetails, error) {
	m.mu.Lock()
	m.detailsCalls = append(m.detailsCalls, matchID)
	m.mu.Unlock()
	if m.detailsErr != nil {
		return nil, m.detailsErr
	}
	if d, ok := m.details[matchID]; ok {
		return d, nil
	}
	return &faceit.MatchDetails{MatchID: matchID}, nil
}

func historyKey(offset, limit int) string {
	return fmt.Sprintf("%d:%d", offset, limit)
}

// mockFaceitAPIPartialFail wraps mockFaceitAPI but fails GetMatchDetails for specific match IDs.
type mockFaceitAPIPartialFail struct {
	*mockFaceitAPI
	failMatchIDs map[string]bool
	details      map[string]*faceit.MatchDetails
}

func (m *mockFaceitAPIPartialFail) GetMatchDetails(_ context.Context, matchID string) (*faceit.MatchDetails, error) {
	if m.failMatchIDs[matchID] {
		return nil, errors.New("api timeout")
	}
	if d, ok := m.details[matchID]; ok {
		return d, nil
	}
	return &faceit.MatchDetails{MatchID: matchID}, nil
}

type mockSyncStore struct {
	existingIDs []string
	existingErr error
	upserted    []store.UpsertFaceitMatchParams
	upsertErr   error
}

func (m *mockSyncStore) GetExistingFaceitMatchIDs(_ context.Context, _ uuid.UUID) ([]string, error) {
	if m.existingErr != nil {
		return nil, m.existingErr
	}
	return m.existingIDs, nil
}

func (m *mockSyncStore) UpsertFaceitMatch(_ context.Context, arg store.UpsertFaceitMatchParams) (store.FaceitMatch, error) {
	if m.upsertErr != nil {
		return store.FaceitMatch{}, m.upsertErr
	}
	m.upserted = append(m.upserted, arg)
	return store.FaceitMatch{
		ID:            uuid.New(),
		UserID:        arg.UserID,
		FaceitMatchID: arg.FaceitMatchID,
		MapName:       arg.MapName,
		ScoreTeam:     arg.ScoreTeam,
		ScoreOpponent: arg.ScoreOpponent,
		Result:        arg.Result,
		EloBefore:     arg.EloBefore,
		EloAfter:      arg.EloAfter,
		PlayedAt:      arg.PlayedAt,
		CreatedAt:     time.Now(),
	}, nil
}

// --- Helpers ---

func makeMatch(id string, startedAt int64, playerElo int, winner string) faceit.MatchSummary {
	return faceit.MatchSummary{
		MatchID:    id,
		GameID:     "cs2",
		StartedAt:  startedAt,
		FinishedAt: startedAt + 3600,
		Teams: map[string]faceit.Team{
			"faction1": {
				TeamID:   "team-a",
				Nickname: "Team A",
				Players: []faceit.TeamPlayer{
					{PlayerID: "player-1", Nickname: "player1", FaceitElo: playerElo, SkillLevel: 10},
					{PlayerID: "player-2", Nickname: "player2", FaceitElo: 2000, SkillLevel: 10},
				},
			},
			"faction2": {
				TeamID:   "team-b",
				Nickname: "Team B",
				Players: []faceit.TeamPlayer{
					{PlayerID: "player-3", Nickname: "player3", FaceitElo: 1900, SkillLevel: 9},
					{PlayerID: "player-4", Nickname: "player4", FaceitElo: 1800, SkillLevel: 9},
				},
			},
		},
		Results: faceit.MatchResults{
			Winner: winner,
			Score:  map[string]int{"faction1": 16, "faction2": 10},
		},
	}
}

func makeDetails(id, mapName string, demoURL string) *faceit.MatchDetails {
	d := &faceit.MatchDetails{
		MatchID: id,
		Voting: faceit.Voting{
			Map: faceit.VotingCategory{Pick: []string{mapName}},
		},
	}
	if demoURL != "" {
		d.DemoURL = []string{demoURL}
	}
	return d
}

var (
	testUserID   = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testFaceitID = "player-1"
)

// --- Tests ---

func TestSync_HappyPath(t *testing.T) {
	// 3 new matches, newest first (API order)
	api := &mockFaceitAPI{
		player: &faceit.Player{
			PlayerID: testFaceitID,
			Games:    map[string]faceit.Game{"cs2": {FaceitElo: 2050}},
		},
		history: map[string]*faceit.MatchHistory{
			"0:20": {Items: []faceit.MatchSummary{
				makeMatch("match-3", 1003, 2030, "faction1"), // newest
				makeMatch("match-2", 1002, 2010, "faction2"), // middle
				makeMatch("match-1", 1001, 2000, "faction1"), // oldest
			}},
		},
		details: map[string]*faceit.MatchDetails{
			"match-1": makeDetails("match-1", "de_dust2", "https://demo/1.dem"),
			"match-2": makeDetails("match-2", "de_mirage", "https://demo/2.dem"),
			"match-3": makeDetails("match-3", "de_inferno", "https://demo/3.dem"),
		},
	}
	mockStore := &mockSyncStore{}

	svc := faceit.NewSyncService(api, mockStore)
	count, err := svc.Sync(context.Background(), testUserID, testFaceitID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("inserted = %d, want 3", count)
	}
	if len(mockStore.upserted) != 3 {
		t.Fatalf("upserted count = %d, want 3", len(mockStore.upserted))
	}

	// After reversal, match-1 should be first (chronological)
	first := mockStore.upserted[0]
	if first.FaceitMatchID != "match-1" {
		t.Errorf("first upserted match = %s, want match-1", first.FaceitMatchID)
	}
	if first.MapName != "de_dust2" {
		t.Errorf("map = %s, want de_dust2", first.MapName)
	}
	if first.Result != "W" {
		t.Errorf("result = %s, want W", first.Result)
	}
	if first.DemoUrl.String != "https://demo/1.dem" {
		t.Errorf("demo_url = %s, want https://demo/1.dem", first.DemoUrl.String)
	}
	if first.ScoreTeam != 16 {
		t.Errorf("score_team = %d, want 16", first.ScoreTeam)
	}
	if first.ScoreOpponent != 10 {
		t.Errorf("score_opponent = %d, want 10", first.ScoreOpponent)
	}

	// Verify middle match is a loss
	second := mockStore.upserted[1]
	if second.FaceitMatchID != "match-2" {
		t.Errorf("second upserted match = %s, want match-2", second.FaceitMatchID)
	}
	if second.Result != "L" {
		t.Errorf("result = %s, want L", second.Result)
	}
}

func TestSync_IncrementalSync(t *testing.T) {
	// 3 matches from API, 2 already exist
	api := &mockFaceitAPI{
		player: &faceit.Player{
			PlayerID: testFaceitID,
			Games:    map[string]faceit.Game{"cs2": {FaceitElo: 2050}},
		},
		history: map[string]*faceit.MatchHistory{
			"0:20": {Items: []faceit.MatchSummary{
				makeMatch("match-3", 1003, 2030, "faction1"),
				makeMatch("match-2", 1002, 2010, "faction1"),
				makeMatch("match-1", 1001, 2000, "faction1"),
			}},
		},
		details: map[string]*faceit.MatchDetails{
			"match-3": makeDetails("match-3", "de_inferno", ""),
		},
	}
	mockStore := &mockSyncStore{
		existingIDs: []string{"match-1", "match-2"},
	}

	svc := faceit.NewSyncService(api, mockStore)
	count, err := svc.Sync(context.Background(), testUserID, testFaceitID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("inserted = %d, want 1", count)
	}
	if len(mockStore.upserted) != 1 {
		t.Fatalf("upserted count = %d, want 1", len(mockStore.upserted))
	}
	if mockStore.upserted[0].FaceitMatchID != "match-3" {
		t.Errorf("upserted match = %s, want match-3", mockStore.upserted[0].FaceitMatchID)
	}
}

func TestSync_AllDuplicates(t *testing.T) {
	api := &mockFaceitAPI{
		history: map[string]*faceit.MatchHistory{
			"0:20": {Items: []faceit.MatchSummary{
				makeMatch("match-1", 1001, 2000, "faction1"),
				makeMatch("match-2", 1002, 2010, "faction1"),
			}},
		},
	}
	mockStore := &mockSyncStore{
		existingIDs: []string{"match-1", "match-2"},
	}

	svc := faceit.NewSyncService(api, mockStore)
	count, err := svc.Sync(context.Background(), testUserID, testFaceitID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("inserted = %d, want 0", count)
	}
	// Should NOT call GetMatchDetails
	if len(api.detailsCalls) != 0 {
		t.Errorf("GetMatchDetails called %d times, want 0", len(api.detailsCalls))
	}
}

func TestSync_EmptyHistory(t *testing.T) {
	api := &mockFaceitAPI{
		history: map[string]*faceit.MatchHistory{
			"0:20": {Items: []faceit.MatchSummary{}},
		},
	}
	mockStore := &mockSyncStore{}

	svc := faceit.NewSyncService(api, mockStore)
	count, err := svc.Sync(context.Background(), testUserID, testFaceitID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("inserted = %d, want 0", count)
	}
}

func TestSync_PlayerNotInTeams(t *testing.T) {
	// Match where our player is not in any team
	noPlayerMatch := faceit.MatchSummary{
		MatchID:   "match-orphan",
		StartedAt: 1001,
		Teams: map[string]faceit.Team{
			"faction1": {Players: []faceit.TeamPlayer{{PlayerID: "other-1"}}},
			"faction2": {Players: []faceit.TeamPlayer{{PlayerID: "other-2"}}},
		},
		Results: faceit.MatchResults{Winner: "faction1", Score: map[string]int{"faction1": 16, "faction2": 5}},
	}

	api := &mockFaceitAPI{
		player: &faceit.Player{
			PlayerID: testFaceitID,
			Games:    map[string]faceit.Game{"cs2": {FaceitElo: 2000}},
		},
		history: map[string]*faceit.MatchHistory{
			"0:20": {Items: []faceit.MatchSummary{noPlayerMatch}},
		},
		details: map[string]*faceit.MatchDetails{
			"match-orphan": makeDetails("match-orphan", "de_dust2", ""),
		},
	}
	mockStore := &mockSyncStore{}

	svc := faceit.NewSyncService(api, mockStore)
	count, err := svc.Sync(context.Background(), testUserID, testFaceitID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("inserted = %d, want 0 (player not in teams)", count)
	}
}

func TestSync_MapNameMissing(t *testing.T) {
	api := &mockFaceitAPI{
		player: &faceit.Player{
			PlayerID: testFaceitID,
			Games:    map[string]faceit.Game{"cs2": {FaceitElo: 2050}},
		},
		history: map[string]*faceit.MatchHistory{
			"0:20": {Items: []faceit.MatchSummary{
				makeMatch("match-1", 1001, 2000, "faction1"),
			}},
		},
		details: map[string]*faceit.MatchDetails{
			"match-1": {MatchID: "match-1", Voting: faceit.Voting{}}, // no map picks
		},
	}
	mockStore := &mockSyncStore{}

	svc := faceit.NewSyncService(api, mockStore)
	_, err := svc.Sync(context.Background(), testUserID, testFaceitID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mockStore.upserted) != 1 {
		t.Fatalf("upserted = %d, want 1", len(mockStore.upserted))
	}
	if mockStore.upserted[0].MapName != "unknown" {
		t.Errorf("map = %q, want 'unknown'", mockStore.upserted[0].MapName)
	}
}

func TestSync_SingleNewMatch_EloFromGetPlayer(t *testing.T) {
	api := &mockFaceitAPI{
		player: &faceit.Player{
			PlayerID: testFaceitID,
			Games:    map[string]faceit.Game{"cs2": {FaceitElo: 2100}},
		},
		history: map[string]*faceit.MatchHistory{
			"0:20": {Items: []faceit.MatchSummary{
				makeMatch("match-1", 1001, 2050, "faction1"),
			}},
		},
		details: map[string]*faceit.MatchDetails{
			"match-1": makeDetails("match-1", "de_nuke", ""),
		},
	}
	mockStore := &mockSyncStore{}

	svc := faceit.NewSyncService(api, mockStore)
	count, err := svc.Sync(context.Background(), testUserID, testFaceitID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("inserted = %d, want 1", count)
	}
	m := mockStore.upserted[0]
	if m.EloBefore.Int32 != 2050 {
		t.Errorf("elo_before = %d, want 2050", m.EloBefore.Int32)
	}
	if m.EloAfter.Int32 != 2100 {
		t.Errorf("elo_after = %d, want 2100 (current ELO from GetPlayer)", m.EloAfter.Int32)
	}
}

func TestSync_APIErrorFromGetMatchDetails_PartialSuccess(t *testing.T) {
	// 2 matches: match-2 details fail, match-1 should still be inserted
	api := &mockFaceitAPI{
		player: &faceit.Player{
			PlayerID: testFaceitID,
			Games:    map[string]faceit.Game{"cs2": {FaceitElo: 2050}},
		},
		history: map[string]*faceit.MatchHistory{
			"0:20": {Items: []faceit.MatchSummary{
				makeMatch("match-2", 1002, 2020, "faction1"),
				makeMatch("match-1", 1001, 2000, "faction1"),
			}},
		},
		details: map[string]*faceit.MatchDetails{
			"match-1": makeDetails("match-1", "de_dust2", ""),
		},
	}
	// Override GetMatchDetails to fail only for match-2
	origGetDetails := api.details
	api.details = nil // clear so we use custom logic below

	customAPI := &mockFaceitAPIPartialFail{
		mockFaceitAPI: api,
		failMatchIDs:  map[string]bool{"match-2": true},
		details:       origGetDetails,
	}

	mockStore := &mockSyncStore{}

	svc := faceit.NewSyncService(customAPI, mockStore)
	count, err := svc.Sync(context.Background(), testUserID, testFaceitID)
	if err != nil {
		t.Fatalf("expected no error for partial failure, got: %v", err)
	}
	if count != 1 {
		t.Errorf("inserted = %d, want 1 (match-2 skipped)", count)
	}
	if len(mockStore.upserted) != 1 {
		t.Fatalf("upserted count = %d, want 1", len(mockStore.upserted))
	}
	if mockStore.upserted[0].FaceitMatchID != "match-1" {
		t.Errorf("upserted match = %s, want match-1", mockStore.upserted[0].FaceitMatchID)
	}
}

func TestSync_AllDetailsFail(t *testing.T) {
	// All match details fail — should return 0 inserted, no error
	api := &mockFaceitAPI{
		player: &faceit.Player{
			PlayerID: testFaceitID,
			Games:    map[string]faceit.Game{"cs2": {FaceitElo: 2000}},
		},
		history: map[string]*faceit.MatchHistory{
			"0:20": {Items: []faceit.MatchSummary{
				makeMatch("match-1", 1001, 2000, "faction1"),
			}},
		},
		detailsErr: errors.New("api timeout"),
	}
	mockStore := &mockSyncStore{}

	svc := faceit.NewSyncService(api, mockStore)
	count, err := svc.Sync(context.Background(), testUserID, testFaceitID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if count != 0 {
		t.Errorf("inserted = %d, want 0", count)
	}
}

func TestSync_EloChain(t *testing.T) {
	// 3 matches chronologically: elo_before[i] = player's ELO in match i,
	// elo_after[i] = player's ELO in match i+1, last uses current ELO
	api := &mockFaceitAPI{
		player: &faceit.Player{
			PlayerID: testFaceitID,
			Games:    map[string]faceit.Game{"cs2": {FaceitElo: 2060}},
		},
		history: map[string]*faceit.MatchHistory{
			"0:20": {Items: []faceit.MatchSummary{
				makeMatch("match-3", 1003, 2040, "faction1"), // newest, elo=2040
				makeMatch("match-2", 1002, 2020, "faction1"), // middle, elo=2020
				makeMatch("match-1", 1001, 2000, "faction1"), // oldest, elo=2000
			}},
		},
		details: map[string]*faceit.MatchDetails{
			"match-1": makeDetails("match-1", "de_dust2", ""),
			"match-2": makeDetails("match-2", "de_mirage", ""),
			"match-3": makeDetails("match-3", "de_inferno", ""),
		},
	}
	mockStore := &mockSyncStore{}

	svc := faceit.NewSyncService(api, mockStore)
	_, err := svc.Sync(context.Background(), testUserID, testFaceitID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockStore.upserted) != 3 {
		t.Fatalf("upserted = %d, want 3", len(mockStore.upserted))
	}

	// After reversal: match-1 (idx 0), match-2 (idx 1), match-3 (idx 2)
	// match-1: elo_before=2000, elo_after=2020 (match-2's elo)
	// match-2: elo_before=2020, elo_after=2040 (match-3's elo)
	// match-3: elo_before=2040, elo_after=2060 (current from GetPlayer)
	expectations := []struct {
		matchID   string
		eloBefore int32
		eloAfter  int32
	}{
		{"match-1", 2000, 2020},
		{"match-2", 2020, 2040},
		{"match-3", 2040, 2060},
	}

	for i, exp := range expectations {
		m := mockStore.upserted[i]
		if m.FaceitMatchID != exp.matchID {
			t.Errorf("[%d] match = %s, want %s", i, m.FaceitMatchID, exp.matchID)
		}
		if m.EloBefore.Int32 != exp.eloBefore {
			t.Errorf("[%d] elo_before = %d, want %d", i, m.EloBefore.Int32, exp.eloBefore)
		}
		if m.EloAfter.Int32 != exp.eloAfter {
			t.Errorf("[%d] elo_after = %d, want %d", i, m.EloAfter.Int32, exp.eloAfter)
		}
	}
}

func TestSync_EloChain_IncrementalSync(t *testing.T) {
	// 4 matches from API, 2 already exist (match-1, match-3).
	// New matches are match-2 (elo=2020) and match-4 (elo=2060).
	// These are NOT chronologically adjacent in the full history,
	// but ELO chain should still work correctly within the new set.
	api := &mockFaceitAPI{
		player: &faceit.Player{
			PlayerID: testFaceitID,
			Games:    map[string]faceit.Game{"cs2": {FaceitElo: 2080}},
		},
		history: map[string]*faceit.MatchHistory{
			"0:20": {Items: []faceit.MatchSummary{
				makeMatch("match-4", 1004, 2060, "faction1"), // newest
				makeMatch("match-3", 1003, 2040, "faction1"), // exists
				makeMatch("match-2", 1002, 2020, "faction1"), // new
				makeMatch("match-1", 1001, 2000, "faction1"), // exists
			}},
		},
		details: map[string]*faceit.MatchDetails{
			"match-2": makeDetails("match-2", "de_mirage", ""),
			"match-4": makeDetails("match-4", "de_inferno", ""),
		},
	}
	mockStore := &mockSyncStore{
		existingIDs: []string{"match-1", "match-3"},
	}

	svc := faceit.NewSyncService(api, mockStore)
	count, err := svc.Sync(context.Background(), testUserID, testFaceitID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("inserted = %d, want 2", count)
	}
	if len(mockStore.upserted) != 2 {
		t.Fatalf("upserted count = %d, want 2", len(mockStore.upserted))
	}

	// After reversal + filtering: match-2 (idx 0), match-4 (idx 1)
	// match-2: elo_before=2020, elo_after=2060 (match-4's elo, the next processable)
	// match-4: elo_before=2060, elo_after=2080 (current from GetPlayer)
	expectations := []struct {
		matchID   string
		eloBefore int32
		eloAfter  int32
	}{
		{"match-2", 2020, 2060},
		{"match-4", 2060, 2080},
	}

	for i, exp := range expectations {
		m := mockStore.upserted[i]
		if m.FaceitMatchID != exp.matchID {
			t.Errorf("[%d] match = %s, want %s", i, m.FaceitMatchID, exp.matchID)
		}
		if m.EloBefore.Int32 != exp.eloBefore {
			t.Errorf("[%d] elo_before = %d, want %d", i, m.EloBefore.Int32, exp.eloBefore)
		}
		if m.EloAfter.Int32 != exp.eloAfter {
			t.Errorf("[%d] elo_after = %d, want %d", i, m.EloAfter.Int32, exp.eloAfter)
		}
	}
}

func TestSync_ImportFailureDoesNotBlockSync(t *testing.T) {
	// 2 matches with demo URLs; importer that always fails.
	// All matches should still be upserted.
	api := &mockFaceitAPI{
		player: &faceit.Player{
			PlayerID: testFaceitID,
			Games:    map[string]faceit.Game{"cs2": {FaceitElo: 2050}},
		},
		history: map[string]*faceit.MatchHistory{
			"0:20": {Items: []faceit.MatchSummary{
				makeMatch("match-2", 1002, 2020, "faction1"),
				makeMatch("match-1", 1001, 2000, "faction1"),
			}},
		},
		details: map[string]*faceit.MatchDetails{
			"match-1": makeDetails("match-1", "de_dust2", "https://demo/1.dem"),
			"match-2": makeDetails("match-2", "de_mirage", "https://demo/2.dem"),
		},
	}
	syncStore := &mockSyncStore{}

	// Create a DemoImporter with a downloader that always fails
	failDL := &failingHTTPDownloader{}
	failS3 := &stubImportS3{}
	failQ := &stubImportQueue{}
	failStore := &stubImportStore{}
	importer := faceit.NewDemoImporter(failStore, failS3, failQ, failDL, "test-bucket")

	svc := faceit.NewSyncService(api, syncStore).WithAutoImport(importer)
	count, err := svc.Sync(context.Background(), testUserID, testFaceitID)
	if err != nil {
		t.Fatalf("sync should succeed even when import fails: %v", err)
	}
	if count != 2 {
		t.Errorf("inserted = %d, want 2", count)
	}
	if len(syncStore.upserted) != 2 {
		t.Errorf("upserted count = %d, want 2", len(syncStore.upserted))
	}
}

// Stubs for the auto-import failure test

type failingHTTPDownloader struct{}

func (f *failingHTTPDownloader) Do(_ *http.Request) (*http.Response, error) {
	return nil, errors.New("network error")
}

type stubImportStore struct{}

func (s *stubImportStore) GetFaceitMatchByID(_ context.Context, _ uuid.UUID) (store.FaceitMatch, error) {
	return store.FaceitMatch{}, nil
}
func (s *stubImportStore) CreateDemo(_ context.Context, _ store.CreateDemoParams) (store.Demo, error) {
	return store.Demo{}, nil
}
func (s *stubImportStore) LinkFaceitMatchToDemo(_ context.Context, _ store.LinkFaceitMatchToDemoParams) (store.FaceitMatch, error) {
	return store.FaceitMatch{}, nil
}

type stubImportS3 struct{}

func (s *stubImportS3) PutObject(_ context.Context, _, _ string, _ io.Reader, _ int64) error {
	return nil
}
func (s *stubImportS3) DeleteObject(_ context.Context, _, _ string) error { return nil }

type stubImportQueue struct{}

func (s *stubImportQueue) Enqueue(_ context.Context, _ string, _ map[string]interface{}) (string, error) {
	return "", nil
}
