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
	"github.com/sqlc-dev/pqtype"

	"github.com/ok2ju/oversite/backend/internal/handler"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// --- Event test mocks ---

type mockEventStore struct {
	getGameEventsByDemoIDFn       func(ctx context.Context, demoID uuid.UUID) ([]store.GameEvent, error)
	getGameEventsByDemoAndRoundFn func(ctx context.Context, arg store.GetGameEventsByDemoAndRoundParams) ([]store.GameEvent, error)
}

func (m *mockEventStore) GetGameEventsByDemoID(ctx context.Context, demoID uuid.UUID) ([]store.GameEvent, error) {
	return m.getGameEventsByDemoIDFn(ctx, demoID)
}

func (m *mockEventStore) GetGameEventsByDemoAndRound(ctx context.Context, arg store.GetGameEventsByDemoAndRoundParams) ([]store.GameEvent, error) {
	return m.getGameEventsByDemoAndRoundFn(ctx, arg)
}

// --- Helpers ---

var testRoundID = uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")

func sampleGameEvents() []store.GameEvent {
	return []store.GameEvent{
		{
			ID:        uuid.MustParse("990e8400-e29b-41d4-a716-446655440000"),
			DemoID:    testDemoID,
			RoundID:   uuid.NullUUID{UUID: testRoundID, Valid: true},
			Tick:      1024,
			EventType: "kill",
			AttackerSteamID: sql.NullString{
				String: "76561198000000001",
				Valid:  true,
			},
			VictimSteamID: sql.NullString{
				String: "76561198000000002",
				Valid:  true,
			},
			Weapon: sql.NullString{String: "AK-47", Valid: true},
			X:      sql.NullFloat64{Float64: -500.0, Valid: true},
			Y:      sql.NullFloat64{Float64: 1000.0, Valid: true},
			Z:      sql.NullFloat64{Float64: 100.0, Valid: true},
			ExtraData: pqtype.NullRawMessage{
				RawMessage: json.RawMessage(`{"headshot":true,"attacker_x":-600.0}`),
				Valid:      true,
			},
		},
		{
			ID:        uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
			DemoID:    testDemoID,
			RoundID:   uuid.NullUUID{UUID: testRoundID, Valid: true},
			Tick:      2048,
			EventType: "smoke_start",
			X:         sql.NullFloat64{Float64: 200.0, Valid: true},
			Y:         sql.NullFloat64{Float64: 300.0, Valid: true},
			Z:         sql.NullFloat64{Float64: 0.0, Valid: true},
		},
	}
}

func ownerDemoGetterForEvents() *mockDemoGetter {
	return &mockDemoGetter{
		getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
			return testDemo(), nil
		},
	}
}

func TestHandleGetEvents(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() (*handler.EventHandler, *http.Request)
		wantStatus int
		checkBody  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "returns events for valid demo",
			setup: func() (*handler.EventHandler, *http.Request) {
				es := &mockEventStore{
					getGameEventsByDemoIDFn: func(_ context.Context, id uuid.UUID) ([]store.GameEvent, error) {
						if id != testDemoID {
							t.Errorf("unexpected demo id: %v", id)
						}
						return sampleGameEvents(), nil
					},
				}
				h := handler.NewEventHandler(ownerDemoGetterForEvents(), es)
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/events", nil)
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
				if first["event_type"] != "kill" {
					t.Errorf("expected event_type=kill, got %v", first["event_type"])
				}
				if first["tick"] != float64(1024) {
					t.Errorf("expected tick=1024, got %v", first["tick"])
				}
				if first["weapon"] != "AK-47" {
					t.Errorf("expected weapon=AK-47, got %v", first["weapon"])
				}
			},
		},
		{
			name: "returns events filtered by round_id",
			setup: func() (*handler.EventHandler, *http.Request) {
				es := &mockEventStore{
					getGameEventsByDemoAndRoundFn: func(_ context.Context, arg store.GetGameEventsByDemoAndRoundParams) ([]store.GameEvent, error) {
						if arg.DemoID != testDemoID {
							t.Errorf("unexpected demo id: %v", arg.DemoID)
						}
						if !arg.RoundID.Valid || arg.RoundID.UUID != testRoundID {
							t.Errorf("unexpected round id: %v", arg.RoundID)
						}
						return sampleGameEvents(), nil
					},
				}
				h := handler.NewEventHandler(ownerDemoGetterForEvents(), es)
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/demos/%s/events?round_id=%s", testDemoID, testRoundID), nil)
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
			},
		},
		{
			name: "invalid round_id returns 400",
			setup: func() (*handler.EventHandler, *http.Request) {
				h := handler.NewEventHandler(ownerDemoGetterForEvents(), &mockEventStore{})
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/events?round_id=not-a-uuid", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "no auth returns 401",
			setup: func() (*handler.EventHandler, *http.Request) {
				h := handler.NewEventHandler(ownerDemoGetterForEvents(), &mockEventStore{})
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/events", nil)
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "demo not found returns 404",
			setup: func() (*handler.EventHandler, *http.Request) {
				dg := &mockDemoGetter{
					getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
						return store.Demo{}, store.ErrNotFound
					},
				}
				h := handler.NewEventHandler(dg, &mockEventStore{})
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/events", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "demo owned by different user returns 404",
			setup: func() (*handler.EventHandler, *http.Request) {
				h := handler.NewEventHandler(ownerDemoGetterForEvents(), &mockEventStore{})
				otherUser := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/events", nil)
				req = withUserID(req, otherUser.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "invalid demo UUID returns 400",
			setup: func() (*handler.EventHandler, *http.Request) {
				h := handler.NewEventHandler(ownerDemoGetterForEvents(), &mockEventStore{})
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/not-a-uuid/events", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", "not-a-uuid")
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "DB error returns 500",
			setup: func() (*handler.EventHandler, *http.Request) {
				es := &mockEventStore{
					getGameEventsByDemoIDFn: func(_ context.Context, _ uuid.UUID) ([]store.GameEvent, error) {
						return nil, errors.New("db down")
					},
				}
				h := handler.NewEventHandler(ownerDemoGetterForEvents(), es)
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/events", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "empty result returns 200 with empty array",
			setup: func() (*handler.EventHandler, *http.Request) {
				es := &mockEventStore{
					getGameEventsByDemoIDFn: func(_ context.Context, _ uuid.UUID) ([]store.GameEvent, error) {
						return nil, nil
					},
				}
				h := handler.NewEventHandler(ownerDemoGetterForEvents(), es)
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String()+"/events", nil)
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

			h.HandleGetEvents(rec, req)

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
