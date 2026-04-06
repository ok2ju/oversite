package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/handler"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// --- Roster test mocks ---

type mockRosterDemoGetter struct {
	getDemoByIDFn func(ctx context.Context, id uuid.UUID) (store.Demo, error)
}

func (m *mockRosterDemoGetter) GetDemoByID(ctx context.Context, id uuid.UUID) (store.Demo, error) {
	return m.getDemoByIDFn(ctx, id)
}

type mockRosterStore struct {
	getRoundByDemoAndNumberFn  func(ctx context.Context, arg store.GetRoundByDemoAndNumberParams) (store.Round, error)
	getPlayerRoundsByRoundIDFn func(ctx context.Context, roundID uuid.UUID) ([]store.PlayerRound, error)
}

func (m *mockRosterStore) GetRoundByDemoAndNumber(ctx context.Context, arg store.GetRoundByDemoAndNumberParams) (store.Round, error) {
	return m.getRoundByDemoAndNumberFn(ctx, arg)
}

func (m *mockRosterStore) GetPlayerRoundsByRoundID(ctx context.Context, roundID uuid.UUID) ([]store.PlayerRound, error) {
	return m.getPlayerRoundsByRoundIDFn(ctx, roundID)
}

// --- Helpers ---

var testRoundID = uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")

func ownerRosterDemoGetter() *mockRosterDemoGetter {
	return &mockRosterDemoGetter{
		getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
			return testDemo(), nil
		},
	}
}

func sampleRound() store.Round {
	return store.Round{
		ID:          testRoundID,
		DemoID:      testDemoID,
		RoundNumber: 1,
		StartTick:   0,
		EndTick:     3200,
		WinnerSide:  "CT",
		WinReason:   "target_saved",
	}
}

func samplePlayerRounds() []store.PlayerRound {
	return []store.PlayerRound{
		{
			ID:       uuid.New(),
			RoundID:  testRoundID,
			SteamID:  "76561198000000001",
			PlayerName: "player1",
			TeamSide: "CT",
		},
		{
			ID:       uuid.New(),
			RoundID:  testRoundID,
			SteamID:  "76561198000000002",
			PlayerName: "player2",
			TeamSide: "T",
		},
	}
}

func successRosterStore() *mockRosterStore {
	return &mockRosterStore{
		getRoundByDemoAndNumberFn: func(_ context.Context, _ store.GetRoundByDemoAndNumberParams) (store.Round, error) {
			return sampleRound(), nil
		},
		getPlayerRoundsByRoundIDFn: func(_ context.Context, _ uuid.UUID) ([]store.PlayerRound, error) {
			return samplePlayerRounds(), nil
		},
	}
}

func withChiURLParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestHandleGetPlayers(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() (*handler.RosterHandler, *http.Request)
		wantStatus int
		checkBody  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "success returns player roster",
			setup: func() (*handler.RosterHandler, *http.Request) {
				h := handler.NewRosterHandler(ownerRosterDemoGetter(), successRosterStore())
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds/1/players", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParams(req, map[string]string{"id": testDemoID.String(), "roundNumber": "1"})
				return h, req
			},
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				data, ok := body["data"].([]interface{})
				if !ok {
					t.Fatal("expected 'data' array")
				}
				if len(data) != 2 {
					t.Fatalf("expected 2 players, got %d", len(data))
				}
				first := data[0].(map[string]interface{})
				if first["steam_id"] != "76561198000000001" {
					t.Errorf("expected steam_id=76561198000000001, got %v", first["steam_id"])
				}
				if first["player_name"] != "player1" {
					t.Errorf("expected player_name=player1, got %v", first["player_name"])
				}
				if first["team_side"] != "CT" {
					t.Errorf("expected team_side=CT, got %v", first["team_side"])
				}
			},
		},
		{
			name: "no auth returns 401",
			setup: func() (*handler.RosterHandler, *http.Request) {
				h := handler.NewRosterHandler(ownerRosterDemoGetter(), successRosterStore())
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds/1/players", nil)
				req = withChiURLParams(req, map[string]string{"id": testDemoID.String(), "roundNumber": "1"})
				return h, req
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid demo UUID returns 400",
			setup: func() (*handler.RosterHandler, *http.Request) {
				h := handler.NewRosterHandler(ownerRosterDemoGetter(), successRosterStore())
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/not-a-uuid/rounds/1/players", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParams(req, map[string]string{"id": "not-a-uuid", "roundNumber": "1"})
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid round number returns 400",
			setup: func() (*handler.RosterHandler, *http.Request) {
				h := handler.NewRosterHandler(ownerRosterDemoGetter(), successRosterStore())
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds/abc/players", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParams(req, map[string]string{"id": testDemoID.String(), "roundNumber": "abc"})
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "round number zero returns 400",
			setup: func() (*handler.RosterHandler, *http.Request) {
				h := handler.NewRosterHandler(ownerRosterDemoGetter(), successRosterStore())
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds/0/players", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParams(req, map[string]string{"id": testDemoID.String(), "roundNumber": "0"})
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "demo not found returns 404",
			setup: func() (*handler.RosterHandler, *http.Request) {
				dg := &mockRosterDemoGetter{
					getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
						return store.Demo{}, store.ErrNotFound
					},
				}
				h := handler.NewRosterHandler(dg, successRosterStore())
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds/1/players", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParams(req, map[string]string{"id": testDemoID.String(), "roundNumber": "1"})
				return h, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "wrong owner returns 404",
			setup: func() (*handler.RosterHandler, *http.Request) {
				h := handler.NewRosterHandler(ownerRosterDemoGetter(), successRosterStore())
				otherUser := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds/1/players", nil)
				req = withUserID(req, otherUser.String())
				req = withChiURLParams(req, map[string]string{"id": testDemoID.String(), "roundNumber": "1"})
				return h, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "round not found returns 404",
			setup: func() (*handler.RosterHandler, *http.Request) {
				rs := &mockRosterStore{
					getRoundByDemoAndNumberFn: func(_ context.Context, _ store.GetRoundByDemoAndNumberParams) (store.Round, error) {
						return store.Round{}, store.ErrNotFound
					},
					getPlayerRoundsByRoundIDFn: func(_ context.Context, _ uuid.UUID) ([]store.PlayerRound, error) {
						return nil, nil
					},
				}
				h := handler.NewRosterHandler(ownerRosterDemoGetter(), rs)
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds/99/players", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParams(req, map[string]string{"id": testDemoID.String(), "roundNumber": "99"})
				return h, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "demo DB error returns 500",
			setup: func() (*handler.RosterHandler, *http.Request) {
				dg := &mockRosterDemoGetter{
					getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
						return store.Demo{}, errors.New("db down")
					},
				}
				h := handler.NewRosterHandler(dg, successRosterStore())
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds/1/players", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParams(req, map[string]string{"id": testDemoID.String(), "roundNumber": "1"})
				return h, req
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "round DB error returns 500",
			setup: func() (*handler.RosterHandler, *http.Request) {
				rs := &mockRosterStore{
					getRoundByDemoAndNumberFn: func(_ context.Context, _ store.GetRoundByDemoAndNumberParams) (store.Round, error) {
						return store.Round{}, errors.New("db down")
					},
					getPlayerRoundsByRoundIDFn: func(_ context.Context, _ uuid.UUID) ([]store.PlayerRound, error) {
						return nil, nil
					},
				}
				h := handler.NewRosterHandler(ownerRosterDemoGetter(), rs)
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds/1/players", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParams(req, map[string]string{"id": testDemoID.String(), "roundNumber": "1"})
				return h, req
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "empty roster returns 200 with empty array",
			setup: func() (*handler.RosterHandler, *http.Request) {
				rs := &mockRosterStore{
					getRoundByDemoAndNumberFn: func(_ context.Context, _ store.GetRoundByDemoAndNumberParams) (store.Round, error) {
						return sampleRound(), nil
					},
					getPlayerRoundsByRoundIDFn: func(_ context.Context, _ uuid.UUID) ([]store.PlayerRound, error) {
						return nil, nil
					},
				}
				h := handler.NewRosterHandler(ownerRosterDemoGetter(), rs)
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds/1/players", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParams(req, map[string]string{"id": testDemoID.String(), "roundNumber": "1"})
				return h, req
			},
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				data, ok := body["data"].([]interface{})
				if !ok {
					t.Fatal("expected 'data' array")
				}
				if len(data) != 0 {
					t.Errorf("expected empty array, got %d items", len(data))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, req := tt.setup()
			rec := httptest.NewRecorder()

			h.HandleGetPlayers(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d; body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			if tt.checkBody != nil {
				var body map[string]interface{}
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("decoding response body: %v", err)
				}
				tt.checkBody(t, body)
			}
		})
	}
}
