package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/handler"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// --- Tick test mocks ---

type mockDemoGetter struct {
	getDemoByIDFn func(ctx context.Context, id uuid.UUID) (store.Demo, error)
}

func (m *mockDemoGetter) GetDemoByID(ctx context.Context, id uuid.UUID) (store.Demo, error) {
	return m.getDemoByIDFn(ctx, id)
}

type mockTickStore struct {
	getTickDataByRangeFn           func(ctx context.Context, arg store.GetTickDataByRangeParams) ([]store.TickDatum, error)
	getTickDataByRangeAndPlayersFn func(ctx context.Context, arg store.GetTickDataByRangeAndPlayersParams) ([]store.TickDatum, error)
}

func (m *mockTickStore) GetTickDataByRange(ctx context.Context, arg store.GetTickDataByRangeParams) ([]store.TickDatum, error) {
	return m.getTickDataByRangeFn(ctx, arg)
}

func (m *mockTickStore) GetTickDataByRangeAndPlayers(ctx context.Context, arg store.GetTickDataByRangeAndPlayersParams) ([]store.TickDatum, error) {
	return m.getTickDataByRangeAndPlayersFn(ctx, arg)
}

// --- Helpers ---

func sampleTickData() []store.TickDatum {
	return []store.TickDatum{
		{
			DemoID:  testDemoID,
			Tick:    100,
			SteamID: "76561198000000001",
			X:       1.5,
			Y:       2.5,
			Z:       3.5,
			Yaw:     90.0,
			Health:  100,
			Armor:   100,
			IsAlive: true,
		},
		{
			DemoID:  testDemoID,
			Tick:    100,
			SteamID: "76561198000000002",
			X:       10.0,
			Y:       20.0,
			Z:       30.0,
			Yaw:     180.0,
			Health:  75,
			Armor:   50,
			IsAlive: true,
		},
	}
}

func ownerDemoGetter() *mockDemoGetter {
	return &mockDemoGetter{
		getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
			return testDemo(), nil
		},
	}
}

func TestHandleGetTicks(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() (*handler.TickHandler, *http.Request)
		wantStatus int
		checkBody  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "valid range returns tick data",
			setup: func() (*handler.TickHandler, *http.Request) {
				ts := &mockTickStore{
					getTickDataByRangeFn: func(_ context.Context, arg store.GetTickDataByRangeParams) ([]store.TickDatum, error) {
						if arg.DemoID != testDemoID || arg.Tick != 0 || arg.Tick_2 != 640 {
							t.Errorf("unexpected args: %+v", arg)
						}
						return sampleTickData(), nil
					},
				}
				h := handler.NewTickHandler(ownerDemoGetter(), ts)
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/ticks?start_tick=0&end_tick=640", nil)
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
				if first["steam_id"] != "76561198000000001" {
					t.Errorf("expected steam_id=76561198000000001, got %v", first["steam_id"])
				}
				if first["health"] != float64(100) {
					t.Errorf("expected health=100, got %v", first["health"])
				}
			},
		},
		{
			name: "steam_ids filter passes to store",
			setup: func() (*handler.TickHandler, *http.Request) {
				ts := &mockTickStore{
					getTickDataByRangeAndPlayersFn: func(_ context.Context, arg store.GetTickDataByRangeAndPlayersParams) ([]store.TickDatum, error) {
						if len(arg.Column4) != 2 {
							t.Errorf("expected 2 steam_ids, got %d", len(arg.Column4))
						}
						return sampleTickData()[:1], nil
					},
				}
				h := handler.NewTickHandler(ownerDemoGetter(), ts)
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/ticks?start_tick=0&end_tick=640&steam_ids=76561198000000001,76561198000000002", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].([]interface{})
				if len(data) != 1 {
					t.Fatalf("expected 1 item, got %d", len(data))
				}
			},
		},
		{
			name: "missing start_tick returns 400",
			setup: func() (*handler.TickHandler, *http.Request) {
				h := handler.NewTickHandler(ownerDemoGetter(), &mockTickStore{})
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/ticks?end_tick=640", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing end_tick returns 400",
			setup: func() (*handler.TickHandler, *http.Request) {
				h := handler.NewTickHandler(ownerDemoGetter(), &mockTickStore{})
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/ticks?start_tick=0", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "range exceeds 6400 returns 400",
			setup: func() (*handler.TickHandler, *http.Request) {
				h := handler.NewTickHandler(ownerDemoGetter(), &mockTickStore{})
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/demos/%s/ticks?start_tick=0&end_tick=6401", testDemoID), nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "start > end returns 400",
			setup: func() (*handler.TickHandler, *http.Request) {
				h := handler.NewTickHandler(ownerDemoGetter(), &mockTickStore{})
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/demos/%s/ticks?start_tick=500&end_tick=100", testDemoID), nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "negative tick returns 400",
			setup: func() (*handler.TickHandler, *http.Request) {
				h := handler.NewTickHandler(ownerDemoGetter(), &mockTickStore{})
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/demos/%s/ticks?start_tick=-1&end_tick=100", testDemoID), nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "non-numeric start_tick returns 400",
			setup: func() (*handler.TickHandler, *http.Request) {
				h := handler.NewTickHandler(ownerDemoGetter(), &mockTickStore{})
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/demos/%s/ticks?start_tick=abc&end_tick=100", testDemoID), nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "non-numeric end_tick returns 400",
			setup: func() (*handler.TickHandler, *http.Request) {
				h := handler.NewTickHandler(ownerDemoGetter(), &mockTickStore{})
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/demos/%s/ticks?start_tick=0&end_tick=xyz", testDemoID), nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "demo not found returns 404",
			setup: func() (*handler.TickHandler, *http.Request) {
				dg := &mockDemoGetter{
					getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
						return store.Demo{}, sql.ErrNoRows
					},
				}
				h := handler.NewTickHandler(dg, &mockTickStore{})
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/demos/%s/ticks?start_tick=0&end_tick=640", testDemoID), nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "wrong user returns 404",
			setup: func() (*handler.TickHandler, *http.Request) {
				h := handler.NewTickHandler(ownerDemoGetter(), &mockTickStore{})
				otherUser := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/demos/%s/ticks?start_tick=0&end_tick=640", testDemoID), nil)
				req = withUserID(req, otherUser.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "no auth returns 401",
			setup: func() (*handler.TickHandler, *http.Request) {
				h := handler.NewTickHandler(ownerDemoGetter(), &mockTickStore{})
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/demos/%s/ticks?start_tick=0&end_tick=640", testDemoID), nil)
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid UUID returns 400",
			setup: func() (*handler.TickHandler, *http.Request) {
				h := handler.NewTickHandler(ownerDemoGetter(), &mockTickStore{})
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/not-a-uuid/ticks?start_tick=0&end_tick=640", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", "not-a-uuid")
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "DB error returns 500",
			setup: func() (*handler.TickHandler, *http.Request) {
				ts := &mockTickStore{
					getTickDataByRangeFn: func(_ context.Context, _ store.GetTickDataByRangeParams) ([]store.TickDatum, error) {
						return nil, errors.New("db down")
					},
				}
				h := handler.NewTickHandler(ownerDemoGetter(), ts)
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/demos/%s/ticks?start_tick=0&end_tick=640", testDemoID), nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "empty result returns 200 with empty array",
			setup: func() (*handler.TickHandler, *http.Request) {
				ts := &mockTickStore{
					getTickDataByRangeFn: func(_ context.Context, _ store.GetTickDataByRangeParams) ([]store.TickDatum, error) {
						return nil, nil
					},
				}
				h := handler.NewTickHandler(ownerDemoGetter(), ts)
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/demos/%s/ticks?start_tick=0&end_tick=640", testDemoID), nil)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, req := tt.setup()
			rec := httptest.NewRecorder()

			h.HandleGetTicks(rec, req)

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
