package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/handler"
	"github.com/ok2ju/oversite/backend/internal/store"
)

type mockQueue struct {
	lastStream string
	lastData   map[string]interface{}
	err        error
}

func (m *mockQueue) Enqueue(_ context.Context, stream string, data map[string]interface{}) (string, error) {
	m.lastStream = stream
	m.lastData = data
	if m.err != nil {
		return "", m.err
	}
	return "msg-1", nil
}

type mockFaceitStore struct {
	// Profile/streak fields
	user       store.User
	userErr    error
	matchCount int64
	countErr   error
	streak     []string
	streakErr  error
	eloHistory []store.GetEloHistoryRow
	eloErr     error
	// Match list fields
	countFilteredFn func(ctx context.Context, arg store.CountFaceitMatchesFilteredParams) (int64, error)
	listFilteredFn  func(ctx context.Context, arg store.GetFaceitMatchesFilteredParams) ([]store.FaceitMatch, error)
}

func (m *mockFaceitStore) GetUserByID(_ context.Context, _ uuid.UUID) (store.User, error) {
	return m.user, m.userErr
}

func (m *mockFaceitStore) GetEloHistory(_ context.Context, _ store.GetEloHistoryParams) ([]store.GetEloHistoryRow, error) {
	return m.eloHistory, m.eloErr
}

func (m *mockFaceitStore) CountFaceitMatchesByUserID(_ context.Context, _ uuid.UUID) (int64, error) {
	return m.matchCount, m.countErr
}

func (m *mockFaceitStore) GetCurrentStreak(_ context.Context, _ uuid.UUID) ([]string, error) {
	return m.streak, m.streakErr
}

func (m *mockFaceitStore) CountFaceitMatchesFiltered(ctx context.Context, arg store.CountFaceitMatchesFilteredParams) (int64, error) {
	return m.countFilteredFn(ctx, arg)
}

func (m *mockFaceitStore) GetFaceitMatchesFiltered(ctx context.Context, arg store.GetFaceitMatchesFilteredParams) ([]store.FaceitMatch, error) {
	return m.listFilteredFn(ctx, arg)
}

var faceitTestUserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
var faceitTestDemoID = uuid.MustParse("11111111-2222-3333-4444-555555555555")

func testUser() store.User {
	return store.User{
		ID:          faceitTestUserID,
		FaceitID:    "faceit-abc",
		Nickname:    "TestPlayer",
		AvatarUrl:   sql.NullString{String: "https://example.com/avatar.png", Valid: true},
		FaceitElo:   sql.NullInt32{Int32: 1850, Valid: true},
		FaceitLevel: sql.NullInt16{Int16: 8, Valid: true},
		Country:     sql.NullString{String: "US", Valid: true},
	}
}

func sampleMatches() []store.FaceitMatch {
	return []store.FaceitMatch{
		{
			ID:            uuid.New(),
			UserID:        faceitTestUserID,
			FaceitMatchID: "1-abc",
			MapName:       "de_dust2",
			ScoreTeam:     16,
			ScoreOpponent: 10,
			Result:        "W",
			EloBefore:     sql.NullInt32{Int32: 2000, Valid: true},
			EloAfter:      sql.NullInt32{Int32: 2025, Valid: true},
			Kills:         sql.NullInt16{Int16: 22, Valid: true},
			Deaths:        sql.NullInt16{Int16: 15, Valid: true},
			Assists:       sql.NullInt16{Int16: 5, Valid: true},
			DemoUrl:       sql.NullString{String: "https://demo.url/1", Valid: true},
			DemoID:        uuid.NullUUID{UUID: faceitTestDemoID, Valid: true},
			PlayedAt:      time.Date(2026, 3, 10, 18, 0, 0, 0, time.UTC),
		},
		{
			ID:            uuid.New(),
			UserID:        faceitTestUserID,
			FaceitMatchID: "1-def",
			MapName:       "de_mirage",
			ScoreTeam:     12,
			ScoreOpponent: 16,
			Result:        "L",
			EloBefore:     sql.NullInt32{},
			EloAfter:      sql.NullInt32{},
			Kills:         sql.NullInt16{},
			Deaths:        sql.NullInt16{},
			Assists:       sql.NullInt16{},
			DemoUrl:       sql.NullString{},
			DemoID:        uuid.NullUUID{},
			PlayedAt:      time.Date(2026, 3, 9, 14, 0, 0, 0, time.UTC),
		},
	}
}

func TestFaceitHandleGetMatches(t *testing.T) {
	matches := sampleMatches()

	tests := []struct {
		name       string
		userID     string
		query      string
		countFn    func(ctx context.Context, arg store.CountFaceitMatchesFilteredParams) (int64, error)
		listFn     func(ctx context.Context, arg store.GetFaceitMatchesFilteredParams) ([]store.FaceitMatch, error)
		wantStatus int
		check      func(t *testing.T, body map[string]interface{})
	}{
		{
			name:   "valid request returns 200 with data and meta",
			userID: faceitTestUserID.String(),
			countFn: func(_ context.Context, _ store.CountFaceitMatchesFilteredParams) (int64, error) {
				return 2, nil
			},
			listFn: func(_ context.Context, _ store.GetFaceitMatchesFilteredParams) ([]store.FaceitMatch, error) {
				return matches, nil
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].([]interface{})
				if len(data) != 2 {
					t.Errorf("data length = %d, want 2", len(data))
				}
				meta := body["meta"].(map[string]interface{})
				if meta["total"].(float64) != 2 {
					t.Errorf("total = %v, want 2", meta["total"])
				}
				if meta["page"].(float64) != 1 {
					t.Errorf("page = %v, want 1", meta["page"])
				}
				if meta["per_page"].(float64) != 20 {
					t.Errorf("per_page = %v, want 20", meta["per_page"])
				}
				// Check first match fields
				first := data[0].(map[string]interface{})
				if first["map_name"] != "de_dust2" {
					t.Errorf("map_name = %v, want de_dust2", first["map_name"])
				}
				if first["elo_change"].(float64) != 25 {
					t.Errorf("elo_change = %v, want 25", first["elo_change"])
				}
				if first["has_demo"] != true {
					t.Errorf("has_demo = %v, want true", first["has_demo"])
				}
				// Check second match null elo
				second := data[1].(map[string]interface{})
				if second["elo_change"] != nil {
					t.Errorf("elo_change = %v, want nil", second["elo_change"])
				}
				if second["has_demo"] != false {
					t.Errorf("has_demo = %v, want false", second["has_demo"])
				}
			},
		},
		{
			name:       "no auth returns 401",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "default pagination uses page=1 per_page=20",
			userID: faceitTestUserID.String(),
			countFn: func(_ context.Context, _ store.CountFaceitMatchesFilteredParams) (int64, error) {
				return 0, nil
			},
			listFn: func(_ context.Context, arg store.GetFaceitMatchesFilteredParams) ([]store.FaceitMatch, error) {
				if arg.Limit != 20 {
					t.Errorf("limit = %d, want 20", arg.Limit)
				}
				if arg.Offset != 0 {
					t.Errorf("offset = %d, want 0", arg.Offset)
				}
				return nil, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "custom pagination page=2 per_page=10 produces offset=10",
			userID: faceitTestUserID.String(),
			query:  "page=2&per_page=10",
			countFn: func(_ context.Context, _ store.CountFaceitMatchesFilteredParams) (int64, error) {
				return 15, nil
			},
			listFn: func(_ context.Context, arg store.GetFaceitMatchesFilteredParams) ([]store.FaceitMatch, error) {
				if arg.Limit != 10 {
					t.Errorf("limit = %d, want 10", arg.Limit)
				}
				if arg.Offset != 10 {
					t.Errorf("offset = %d, want 10", arg.Offset)
				}
				return nil, nil
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body map[string]interface{}) {
				meta := body["meta"].(map[string]interface{})
				if meta["page"].(float64) != 2 {
					t.Errorf("page = %v, want 2", meta["page"])
				}
				if meta["per_page"].(float64) != 10 {
					t.Errorf("per_page = %v, want 10", meta["per_page"])
				}
			},
		},
		{
			name:   "map_name filter populates NullString",
			userID: faceitTestUserID.String(),
			query:  "map_name=de_dust2",
			countFn: func(_ context.Context, arg store.CountFaceitMatchesFilteredParams) (int64, error) {
				if !arg.MapName.Valid || arg.MapName.String != "de_dust2" {
					t.Errorf("count map_name = %v, want de_dust2", arg.MapName)
				}
				return 1, nil
			},
			listFn: func(_ context.Context, arg store.GetFaceitMatchesFilteredParams) ([]store.FaceitMatch, error) {
				if !arg.MapName.Valid || arg.MapName.String != "de_dust2" {
					t.Errorf("list map_name = %v, want de_dust2", arg.MapName)
				}
				return matches[:1], nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "result filter populates NullString",
			userID: faceitTestUserID.String(),
			query:  "result=W",
			countFn: func(_ context.Context, arg store.CountFaceitMatchesFilteredParams) (int64, error) {
				if !arg.Result.Valid || arg.Result.String != "W" {
					t.Errorf("count result = %v, want W", arg.Result)
				}
				return 1, nil
			},
			listFn: func(_ context.Context, arg store.GetFaceitMatchesFilteredParams) ([]store.FaceitMatch, error) {
				if !arg.Result.Valid || arg.Result.String != "W" {
					t.Errorf("list result = %v, want W", arg.Result)
				}
				return matches[:1], nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "both filters combined",
			userID: faceitTestUserID.String(),
			query:  "map_name=de_dust2&result=W",
			countFn: func(_ context.Context, arg store.CountFaceitMatchesFilteredParams) (int64, error) {
				if !arg.MapName.Valid || arg.MapName.String != "de_dust2" {
					t.Errorf("count map_name = %v, want de_dust2", arg.MapName)
				}
				if !arg.Result.Valid || arg.Result.String != "W" {
					t.Errorf("count result = %v, want W", arg.Result)
				}
				return 1, nil
			},
			listFn: func(_ context.Context, arg store.GetFaceitMatchesFilteredParams) ([]store.FaceitMatch, error) {
				return matches[:1], nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "empty result returns empty data array",
			userID: faceitTestUserID.String(),
			countFn: func(_ context.Context, _ store.CountFaceitMatchesFilteredParams) (int64, error) {
				return 0, nil
			},
			listFn: func(_ context.Context, _ store.GetFaceitMatchesFilteredParams) ([]store.FaceitMatch, error) {
				return nil, nil
			},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].([]interface{})
				if len(data) != 0 {
					t.Errorf("data length = %d, want 0", len(data))
				}
				meta := body["meta"].(map[string]interface{})
				if meta["total"].(float64) != 0 {
					t.Errorf("total = %v, want 0", meta["total"])
				}
			},
		},
		{
			name:   "count DB error returns 500",
			userID: faceitTestUserID.String(),
			countFn: func(_ context.Context, _ store.CountFaceitMatchesFilteredParams) (int64, error) {
				return 0, errors.New("db down")
			},
			listFn: func(_ context.Context, _ store.GetFaceitMatchesFilteredParams) ([]store.FaceitMatch, error) {
				return nil, nil
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "list DB error returns 500",
			userID: faceitTestUserID.String(),
			countFn: func(_ context.Context, _ store.CountFaceitMatchesFilteredParams) (int64, error) {
				return 2, nil
			},
			listFn: func(_ context.Context, _ store.GetFaceitMatchesFilteredParams) ([]store.FaceitMatch, error) {
				return nil, errors.New("db down")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockFaceitStore{}
			if tt.countFn != nil {
				ms.countFilteredFn = tt.countFn
			}
			if tt.listFn != nil {
				ms.listFilteredFn = tt.listFn
			}

			h := handler.NewFaceitHandler(&mockQueue{}, ms)

			url := "/api/v1/faceit/matches"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			ctx := req.Context()
			if tt.userID != "" {
				ctx = context.WithValue(ctx, auth.UserIDKey, tt.userID)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			h.HandleGetMatches(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.check != nil {
				var body map[string]interface{}
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("decoding response: %v", err)
				}
				tt.check(t, body)
			}
		})
	}
}

func TestFaceitHandleSync(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		faceitID   string
		queueErr   error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid request returns 202",
			userID:     "user-123",
			faceitID:   "faceit-456",
			wantStatus: http.StatusAccepted,
			wantBody:   "sync_queued",
		},
		{
			name:       "missing auth returns 401",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing faceit_id returns 500",
			userID:     "user-123",
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "queue error returns 500",
			userID:     "user-123",
			faceitID:   "faceit-456",
			queueErr:   errors.New("redis down"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &mockQueue{err: tt.queueErr}
			h := handler.NewFaceitHandler(q, &mockFaceitStore{})

			req := httptest.NewRequest(http.MethodPost, "/api/v1/faceit/sync", nil)
			ctx := req.Context()
			if tt.userID != "" {
				ctx = context.WithValue(ctx, auth.UserIDKey, tt.userID)
			}
			if tt.faceitID != "" {
				ctx = context.WithValue(ctx, auth.FaceitIDKey, tt.faceitID)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			h.HandleSync(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantBody != "" {
				var body map[string]string
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("decoding response: %v", err)
				}
				if body["status"] != tt.wantBody {
					t.Errorf("body status = %q, want %q", body["status"], tt.wantBody)
				}
			}

			// Verify queue was called with correct data
			if tt.wantStatus == http.StatusAccepted {
				if q.lastStream != "faceit_sync" {
					t.Errorf("stream = %q, want faceit_sync", q.lastStream)
				}
				if q.lastData["user_id"] != tt.userID {
					t.Errorf("user_id = %v, want %s", q.lastData["user_id"], tt.userID)
				}
				if q.lastData["faceit_id"] != tt.faceitID {
					t.Errorf("faceit_id = %v, want %s", q.lastData["faceit_id"], tt.faceitID)
				}
			}
		})
	}
}

func TestFaceitHandleGetProfile(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		store      *mockFaceitStore
		wantStatus int
		wantCheck  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:   "valid request returns profile",
			userID: faceitTestUserID.String(),
			store: &mockFaceitStore{
				user:       testUser(),
				matchCount: 142,
				streak:     []string{"W", "W", "W", "L"},
			},
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].(map[string]interface{})
				if data["nickname"] != "TestPlayer" {
					t.Errorf("nickname = %v, want TestPlayer", data["nickname"])
				}
				if data["elo"].(float64) != 1850 {
					t.Errorf("elo = %v, want 1850", data["elo"])
				}
				if data["level"].(float64) != 8 {
					t.Errorf("level = %v, want 8", data["level"])
				}
				if data["country"] != "US" {
					t.Errorf("country = %v, want US", data["country"])
				}
				if data["matches_played"].(float64) != 142 {
					t.Errorf("matches_played = %v, want 142", data["matches_played"])
				}
				streak := data["current_streak"].(map[string]interface{})
				if streak["type"] != "win" {
					t.Errorf("streak type = %v, want win", streak["type"])
				}
				if streak["count"].(float64) != 3 {
					t.Errorf("streak count = %v, want 3", streak["count"])
				}
			},
		},
		{
			name:       "missing auth returns 401",
			store:      &mockFaceitStore{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "user not found returns 404",
			userID: faceitTestUserID.String(),
			store: &mockFaceitStore{
				userErr: sql.ErrNoRows,
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "db error on user lookup returns 500",
			userID: faceitTestUserID.String(),
			store: &mockFaceitStore{
				userErr: errors.New("connection refused"),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "count error returns 500",
			userID: faceitTestUserID.String(),
			store: &mockFaceitStore{
				user:     testUser(),
				countErr: errors.New("db timeout"),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "streak error returns 500",
			userID: faceitTestUserID.String(),
			store: &mockFaceitStore{
				user:       testUser(),
				matchCount: 10,
				streakErr:  errors.New("db timeout"),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "no matches returns zero streak",
			userID: faceitTestUserID.String(),
			store: &mockFaceitStore{
				user:       testUser(),
				matchCount: 0,
				streak:     nil,
			},
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].(map[string]interface{})
				streak := data["current_streak"].(map[string]interface{})
				if streak["type"] != "none" {
					t.Errorf("streak type = %v, want none", streak["type"])
				}
				if streak["count"].(float64) != 0 {
					t.Errorf("streak count = %v, want 0", streak["count"])
				}
			},
		},
		{
			name:   "loss streak computed correctly",
			userID: faceitTestUserID.String(),
			store: &mockFaceitStore{
				user:       testUser(),
				matchCount: 10,
				streak:     []string{"L", "L", "W"},
			},
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].(map[string]interface{})
				streak := data["current_streak"].(map[string]interface{})
				if streak["type"] != "loss" {
					t.Errorf("streak type = %v, want loss", streak["type"])
				}
				if streak["count"].(float64) != 2 {
					t.Errorf("streak count = %v, want 2", streak["count"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handler.NewFaceitHandler(&mockQueue{}, tt.store)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/faceit/profile", nil)
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), auth.UserIDKey, tt.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.HandleGetProfile(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantCheck != nil {
				var body map[string]interface{}
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("decoding response: %v", err)
				}
				tt.wantCheck(t, body)
			}
		})
	}
}

func TestFaceitHandleGetEloHistory(t *testing.T) {
	now := time.Now()
	sampleHistory := []store.GetEloHistoryRow{
		{ID: uuid.New(), FaceitMatchID: "m1", MapName: "de_dust2", EloAfter: sql.NullInt32{Int32: 1800, Valid: true}, PlayedAt: now.Add(-48 * time.Hour)},
		{ID: uuid.New(), FaceitMatchID: "m2", MapName: "de_mirage", EloAfter: sql.NullInt32{Int32: 1820, Valid: true}, PlayedAt: now.Add(-24 * time.Hour)},
	}

	tests := []struct {
		name       string
		userID     string
		query      string
		store      *mockFaceitStore
		wantStatus int
		wantLen    int
	}{
		{
			name:       "default 30 days returns data",
			userID:     faceitTestUserID.String(),
			query:      "",
			store:      &mockFaceitStore{eloHistory: sampleHistory},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:       "90 days returns data",
			userID:     faceitTestUserID.String(),
			query:      "?days=90",
			store:      &mockFaceitStore{eloHistory: sampleHistory},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:       "180 days returns data",
			userID:     faceitTestUserID.String(),
			query:      "?days=180",
			store:      &mockFaceitStore{eloHistory: sampleHistory},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:       "all time (days=0) returns data",
			userID:     faceitTestUserID.String(),
			query:      "?days=0",
			store:      &mockFaceitStore{eloHistory: sampleHistory},
			wantStatus: http.StatusOK,
			wantLen:    2,
		},
		{
			name:       "invalid days returns 400",
			userID:     faceitTestUserID.String(),
			query:      "?days=45",
			store:      &mockFaceitStore{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "non-numeric days returns 400",
			userID:     faceitTestUserID.String(),
			query:      "?days=abc",
			store:      &mockFaceitStore{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing auth returns 401",
			store:      &mockFaceitStore{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "empty history returns empty array",
			userID:     faceitTestUserID.String(),
			store:      &mockFaceitStore{eloHistory: nil},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:   "store error returns 500",
			userID: faceitTestUserID.String(),
			store:  &mockFaceitStore{eloErr: errors.New("db down")},

			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handler.NewFaceitHandler(&mockQueue{}, tt.store)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/faceit/elo-history"+tt.query, nil)
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), auth.UserIDKey, tt.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.HandleGetEloHistory(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var body map[string]interface{}
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("decoding response: %v", err)
				}
				data := body["data"].([]interface{})
				if len(data) != tt.wantLen {
					t.Errorf("data length = %d, want %d", len(data), tt.wantLen)
				}
			}
		})
	}
}
