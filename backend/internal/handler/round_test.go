package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/handler"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// --- Round test mocks ---

type mockRoundStore struct {
	getRoundsByDemoIDFn func(ctx context.Context, demoID uuid.UUID) ([]store.Round, error)
}

func (m *mockRoundStore) GetRoundsByDemoID(ctx context.Context, demoID uuid.UUID) ([]store.Round, error) {
	return m.getRoundsByDemoIDFn(ctx, demoID)
}

// --- Helpers ---

func sampleRounds() []store.Round {
	return []store.Round{
		{
			ID:          uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440001"),
			DemoID:      testDemoID,
			RoundNumber: 1,
			StartTick:   0,
			EndTick:     3200,
			WinnerSide:  "CT",
			WinReason:   "TargetBombed",
			CtScore:     1,
			TScore:      0,
			IsOvertime:  false,
		},
		{
			ID:          uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440002"),
			DemoID:      testDemoID,
			RoundNumber: 2,
			StartTick:   3200,
			EndTick:     6400,
			WinnerSide:  "T",
			WinReason:   "BombDefused",
			CtScore:     1,
			TScore:      1,
			IsOvertime:  false,
		},
	}
}

func ownerDemoGetterForRounds() *mockDemoGetter {
	return &mockDemoGetter{
		getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
			return testDemo(), nil
		},
	}
}

func TestHandleGetRounds(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() (*handler.RoundHandler, *http.Request)
		wantStatus int
		checkBody  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "returns rounds for valid demo",
			setup: func() (*handler.RoundHandler, *http.Request) {
				rs := &mockRoundStore{
					getRoundsByDemoIDFn: func(_ context.Context, id uuid.UUID) ([]store.Round, error) {
						if id != testDemoID {
							t.Errorf("unexpected demo id: %v", id)
						}
						return sampleRounds(), nil
					},
				}
				h := handler.NewRoundHandler(ownerDemoGetterForRounds(), rs)
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				data, ok := body["data"].([]interface{})
				if !ok {
					t.Fatal("expected 'data' array")
				}
				if len(data) != 2 {
					t.Fatalf("expected 2 items, got %d", len(data))
				}
				first := data[0].(map[string]interface{})
				if first["round_number"] != float64(1) {
					t.Errorf("expected round_number=1, got %v", first["round_number"])
				}
				if first["start_tick"] != float64(0) {
					t.Errorf("expected start_tick=0, got %v", first["start_tick"])
				}
				if first["end_tick"] != float64(3200) {
					t.Errorf("expected end_tick=3200, got %v", first["end_tick"])
				}
				if first["winner_side"] != "CT" {
					t.Errorf("expected winner_side=CT, got %v", first["winner_side"])
				}
				if first["win_reason"] != "TargetBombed" {
					t.Errorf("expected win_reason=TargetBombed, got %v", first["win_reason"])
				}
				if first["is_overtime"] != false {
					t.Errorf("expected is_overtime=false, got %v", first["is_overtime"])
				}
				second := data[1].(map[string]interface{})
				if second["round_number"] != float64(2) {
					t.Errorf("expected round_number=2, got %v", second["round_number"])
				}
			},
		},
		{
			name: "demo not found returns 404",
			setup: func() (*handler.RoundHandler, *http.Request) {
				dg := &mockDemoGetter{
					getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
						return store.Demo{}, store.ErrNotFound
					},
				}
				h := handler.NewRoundHandler(dg, &mockRoundStore{})
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "demo owned by different user returns 404",
			setup: func() (*handler.RoundHandler, *http.Request) {
				h := handler.NewRoundHandler(ownerDemoGetterForRounds(), &mockRoundStore{})
				otherUser := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds", nil)
				req = withUserID(req, otherUser.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "empty rounds returns 200 with empty array",
			setup: func() (*handler.RoundHandler, *http.Request) {
				rs := &mockRoundStore{
					getRoundsByDemoIDFn: func(_ context.Context, _ uuid.UUID) ([]store.Round, error) {
						return nil, nil
					},
				}
				h := handler.NewRoundHandler(ownerDemoGetterForRounds(), rs)
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
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
		{
			name: "invalid demo UUID returns 400",
			setup: func() (*handler.RoundHandler, *http.Request) {
				h := handler.NewRoundHandler(ownerDemoGetterForRounds(), &mockRoundStore{})
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/not-a-uuid/rounds", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", "not-a-uuid")
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "no auth returns 401",
			setup: func() (*handler.RoundHandler, *http.Request) {
				h := handler.NewRoundHandler(ownerDemoGetterForRounds(), &mockRoundStore{})
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds", nil)
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "DB error returns 500",
			setup: func() (*handler.RoundHandler, *http.Request) {
				rs := &mockRoundStore{
					getRoundsByDemoIDFn: func(_ context.Context, _ uuid.UUID) ([]store.Round, error) {
						return nil, errors.New("db down")
					},
				}
				h := handler.NewRoundHandler(ownerDemoGetterForRounds(), rs)
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/rounds", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, req := tt.setup()
			rec := httptest.NewRecorder()

			h.HandleGetRounds(rec, req)

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
