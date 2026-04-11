package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/handler"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// --- Heatmap test mocks ---

type mockHeatmapDemoChecker struct {
	getDemosByIDsFn func(ctx context.Context, demoIds []uuid.UUID) ([]store.GetDemosByIDsRow, error)
}

func (m *mockHeatmapDemoChecker) GetDemosByIDs(ctx context.Context, demoIds []uuid.UUID) ([]store.GetDemosByIDsRow, error) {
	return m.getDemosByIDsFn(ctx, demoIds)
}

type mockHeatmapStore struct {
	getHeatmapAggregationFn func(ctx context.Context, arg store.GetHeatmapAggregationParams) ([]store.GetHeatmapAggregationRow, error)
}

func (m *mockHeatmapStore) GetHeatmapAggregation(ctx context.Context, arg store.GetHeatmapAggregationParams) ([]store.GetHeatmapAggregationRow, error) {
	return m.getHeatmapAggregationFn(ctx, arg)
}

// --- Helpers ---

var testDemoID2 = uuid.MustParse("770e8400-e29b-41d4-a716-446655440001")

func ownerDemoChecker() *mockHeatmapDemoChecker {
	return &mockHeatmapDemoChecker{
		getDemosByIDsFn: func(_ context.Context, ids []uuid.UUID) ([]store.GetDemosByIDsRow, error) {
			rows := make([]store.GetDemosByIDsRow, 0, len(ids))
			for _, id := range ids {
				rows = append(rows, store.GetDemosByIDsRow{
					ID:      id,
					UserID:  testUserID,
					MapName: sql.NullString{String: "de_dust2", Valid: true},
				})
			}
			return rows, nil
		},
	}
}

func heatmapRequest(demoIDs []string, filters map[string]string) *bytes.Buffer {
	body := map[string]interface{}{
		"demo_ids": demoIDs,
	}
	if filters != nil {
		body["filters"] = filters
	}
	buf, _ := json.Marshal(body)
	return bytes.NewBuffer(buf)
}

func TestHandleAggregate(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() (*handler.HeatmapHandler, *http.Request)
		wantStatus int
		checkBody  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "valid single-demo request returns normalized data",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				hs := &mockHeatmapStore{
					getHeatmapAggregationFn: func(_ context.Context, _ store.GetHeatmapAggregationParams) ([]store.GetHeatmapAggregationRow, error) {
						return []store.GetHeatmapAggregationRow{
							{X: sql.NullFloat64{Float64: -512.5, Valid: true}, Y: sql.NullFloat64{Float64: 1024.3, Valid: true}, KillCount: 4},
							{X: sql.NullFloat64{Float64: 100.0, Valid: true}, Y: sql.NullFloat64{Float64: 200.0, Valid: true}, KillCount: 2},
						}, nil
					},
				}
				h := handler.NewHeatmapHandler(ownerDemoChecker(), hs)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String()}, nil))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
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
				if first["x"] != -512.5 {
					t.Errorf("expected x=-512.5, got %v", first["x"])
				}
				if first["y"] != 1024.3 {
					t.Errorf("expected y=1024.3, got %v", first["y"])
				}
				if first["intensity"] != 1.0 {
					t.Errorf("expected intensity=1.0, got %v", first["intensity"])
				}
			},
		},
		{
			name: "multi-demo aggregation returns combined results",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				hs := &mockHeatmapStore{
					getHeatmapAggregationFn: func(_ context.Context, arg store.GetHeatmapAggregationParams) ([]store.GetHeatmapAggregationRow, error) {
						if len(arg.DemoIds) != 2 {
							t.Errorf("expected 2 demo IDs, got %d", len(arg.DemoIds))
						}
						return []store.GetHeatmapAggregationRow{
							{X: sql.NullFloat64{Float64: 10.0, Valid: true}, Y: sql.NullFloat64{Float64: 20.0, Valid: true}, KillCount: 3},
						}, nil
					},
				}
				h := handler.NewHeatmapHandler(ownerDemoChecker(), hs)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String(), testDemoID2.String()}, nil))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
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
			name: "side filter passed correctly to store",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				hs := &mockHeatmapStore{
					getHeatmapAggregationFn: func(_ context.Context, arg store.GetHeatmapAggregationParams) ([]store.GetHeatmapAggregationRow, error) {
						if !arg.Side.Valid || arg.Side.String != "CT" {
							t.Errorf("expected side=CT, got %v", arg.Side)
						}
						return nil, nil
					},
				}
				h := handler.NewHeatmapHandler(ownerDemoChecker(), hs)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String()}, map[string]string{"side": "CT"}))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "weapon category expanded to weapon names array",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				hs := &mockHeatmapStore{
					getHeatmapAggregationFn: func(_ context.Context, arg store.GetHeatmapAggregationParams) ([]store.GetHeatmapAggregationRow, error) {
						if len(arg.Weapons) == 0 {
							t.Error("expected weapon names, got empty")
						}
						// Check that at least AK-47 is in the rifle category.
						found := false
						for _, w := range arg.Weapons {
							if w == "AK-47" {
								found = true
								break
							}
						}
						if !found {
							t.Error("expected AK-47 in weapons list")
						}
						return nil, nil
					},
				}
				h := handler.NewHeatmapHandler(ownerDemoChecker(), hs)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String()}, map[string]string{"weapon_category": "rifle"}))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "player filter passed correctly to store",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				hs := &mockHeatmapStore{
					getHeatmapAggregationFn: func(_ context.Context, arg store.GetHeatmapAggregationParams) ([]store.GetHeatmapAggregationRow, error) {
						if !arg.PlayerSteamID.Valid || arg.PlayerSteamID.String != "76561198000000001" {
							t.Errorf("expected player_steam_id=76561198000000001, got %v", arg.PlayerSteamID)
						}
						return nil, nil
					},
				}
				h := handler.NewHeatmapHandler(ownerDemoChecker(), hs)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String()}, map[string]string{"player_steam_id": "76561198000000001"}))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "combined filters work together",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				hs := &mockHeatmapStore{
					getHeatmapAggregationFn: func(_ context.Context, arg store.GetHeatmapAggregationParams) ([]store.GetHeatmapAggregationRow, error) {
						if !arg.Side.Valid || arg.Side.String != "T" {
							t.Errorf("expected side=T, got %v", arg.Side)
						}
						if !arg.PlayerSteamID.Valid || arg.PlayerSteamID.String != "76561198000000001" {
							t.Errorf("expected player_steam_id, got %v", arg.PlayerSteamID)
						}
						if len(arg.Weapons) == 0 {
							t.Error("expected sniper weapons")
						}
						return nil, nil
					},
				}
				h := handler.NewHeatmapHandler(ownerDemoChecker(), hs)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String()}, map[string]string{
					"side":            "T",
					"weapon_category": "sniper",
					"player_steam_id": "76561198000000001",
				}))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "empty result returns 200 with empty array",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				hs := &mockHeatmapStore{
					getHeatmapAggregationFn: func(_ context.Context, _ store.GetHeatmapAggregationParams) ([]store.GetHeatmapAggregationRow, error) {
						return nil, nil
					},
				}
				h := handler.NewHeatmapHandler(ownerDemoChecker(), hs)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String()}, nil))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
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
			name: "missing demo_ids returns 400",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				h := handler.NewHeatmapHandler(ownerDemoChecker(), &mockHeatmapStore{})
				body, _ := json.Marshal(map[string]interface{}{"filters": map[string]string{}})
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "empty demo_ids array returns 400",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				h := handler.NewHeatmapHandler(ownerDemoChecker(), &mockHeatmapStore{})
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{}, nil))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid demo UUID in array returns 400",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				h := handler.NewHeatmapHandler(ownerDemoChecker(), &mockHeatmapStore{})
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{"not-a-uuid"}, nil))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid side value returns 400",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				h := handler.NewHeatmapHandler(ownerDemoChecker(), &mockHeatmapStore{})
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String()}, map[string]string{"side": "BOTH"}))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid weapon_category returns 400",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				h := handler.NewHeatmapHandler(ownerDemoChecker(), &mockHeatmapStore{})
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String()}, map[string]string{"weapon_category": "melee"}))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "no auth returns 401",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				h := handler.NewHeatmapHandler(ownerDemoChecker(), &mockHeatmapStore{})
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String()}, nil))
				req.Header.Set("Content-Type", "application/json")
				return h, req
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "demo not found returns 404",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				dc := &mockHeatmapDemoChecker{
					getDemosByIDsFn: func(_ context.Context, _ []uuid.UUID) ([]store.GetDemosByIDsRow, error) {
						return nil, nil // returns 0 rows for 1 requested
					},
				}
				h := handler.NewHeatmapHandler(dc, &mockHeatmapStore{})
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String()}, nil))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "demo owned by different user returns 404",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				otherUser := uuid.MustParse("880e8400-e29b-41d4-a716-446655440099")
				dc := &mockHeatmapDemoChecker{
					getDemosByIDsFn: func(_ context.Context, ids []uuid.UUID) ([]store.GetDemosByIDsRow, error) {
						rows := make([]store.GetDemosByIDsRow, 0, len(ids))
						for _, id := range ids {
							rows = append(rows, store.GetDemosByIDsRow{ID: id, UserID: otherUser, MapName: sql.NullString{String: "de_dust2", Valid: true}})
						}
						return rows, nil
					},
				}
				h := handler.NewHeatmapHandler(dc, &mockHeatmapStore{})
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String()}, nil))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "mixed ownership returns 404",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				otherUser := uuid.MustParse("880e8400-e29b-41d4-a716-446655440099")
				dc := &mockHeatmapDemoChecker{
					getDemosByIDsFn: func(_ context.Context, ids []uuid.UUID) ([]store.GetDemosByIDsRow, error) {
						return []store.GetDemosByIDsRow{
							{ID: ids[0], UserID: testUserID, MapName: sql.NullString{String: "de_dust2", Valid: true}},
							{ID: ids[1], UserID: otherUser, MapName: sql.NullString{String: "de_dust2", Valid: true}},
						}, nil
					},
				}
				h := handler.NewHeatmapHandler(dc, &mockHeatmapStore{})
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String(), testDemoID2.String()}, nil))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "duplicate demo IDs are deduplicated",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				hs := &mockHeatmapStore{
					getHeatmapAggregationFn: func(_ context.Context, arg store.GetHeatmapAggregationParams) ([]store.GetHeatmapAggregationRow, error) {
						if len(arg.DemoIds) != 1 {
							t.Errorf("expected 1 deduplicated demo ID, got %d", len(arg.DemoIds))
						}
						return nil, nil
					},
				}
				dc := &mockHeatmapDemoChecker{
					getDemosByIDsFn: func(_ context.Context, ids []uuid.UUID) ([]store.GetDemosByIDsRow, error) {
						if len(ids) != 1 {
							t.Errorf("expected 1 deduplicated ID in query, got %d", len(ids))
						}
						return []store.GetDemosByIDsRow{
							{ID: ids[0], UserID: testUserID, MapName: sql.NullString{String: "de_dust2", Valid: true}},
						}, nil
					},
				}
				h := handler.NewHeatmapHandler(dc, hs)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String(), testDemoID.String()}, nil))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "demos from different maps returns 400",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				dc := &mockHeatmapDemoChecker{
					getDemosByIDsFn: func(_ context.Context, ids []uuid.UUID) ([]store.GetDemosByIDsRow, error) {
						return []store.GetDemosByIDsRow{
							{ID: ids[0], UserID: testUserID, MapName: sql.NullString{String: "de_dust2", Valid: true}},
							{ID: ids[1], UserID: testUserID, MapName: sql.NullString{String: "de_mirage", Valid: true}},
						}, nil
					},
				}
				h := handler.NewHeatmapHandler(dc, &mockHeatmapStore{})
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String(), testDemoID2.String()}, nil))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "demo with null map_name returns 400",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				dc := &mockHeatmapDemoChecker{
					getDemosByIDsFn: func(_ context.Context, ids []uuid.UUID) ([]store.GetDemosByIDsRow, error) {
						return []store.GetDemosByIDsRow{
							{ID: ids[0], UserID: testUserID, MapName: sql.NullString{}},
						}, nil
					},
				}
				h := handler.NewHeatmapHandler(dc, &mockHeatmapStore{})
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String()}, nil))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "DB error returns 500",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				hs := &mockHeatmapStore{
					getHeatmapAggregationFn: func(_ context.Context, _ store.GetHeatmapAggregationParams) ([]store.GetHeatmapAggregationRow, error) {
						return nil, errors.New("db down")
					},
				}
				h := handler.NewHeatmapHandler(ownerDemoChecker(), hs)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String()}, nil))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "intensity normalization: max gets 1.0, others proportional",
			setup: func() (*handler.HeatmapHandler, *http.Request) {
				hs := &mockHeatmapStore{
					getHeatmapAggregationFn: func(_ context.Context, _ store.GetHeatmapAggregationParams) ([]store.GetHeatmapAggregationRow, error) {
						return []store.GetHeatmapAggregationRow{
							{X: sql.NullFloat64{Float64: 0, Valid: true}, Y: sql.NullFloat64{Float64: 0, Valid: true}, KillCount: 10},
							{X: sql.NullFloat64{Float64: 1, Valid: true}, Y: sql.NullFloat64{Float64: 1, Valid: true}, KillCount: 5},
							{X: sql.NullFloat64{Float64: 2, Valid: true}, Y: sql.NullFloat64{Float64: 2, Valid: true}, KillCount: 1},
						}, nil
					},
				}
				h := handler.NewHeatmapHandler(ownerDemoChecker(), hs)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/heatmaps/aggregate", heatmapRequest([]string{testDemoID.String()}, nil))
				req.Header.Set("Content-Type", "application/json")
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].([]interface{})
				if len(data) != 3 {
					t.Fatalf("expected 3 items, got %d", len(data))
				}
				first := data[0].(map[string]interface{})
				if first["intensity"] != 1.0 {
					t.Errorf("expected max intensity=1.0, got %v", first["intensity"])
				}
				second := data[1].(map[string]interface{})
				if second["intensity"] != 0.5 {
					t.Errorf("expected intensity=0.5, got %v", second["intensity"])
				}
				third := data[2].(map[string]interface{})
				if third["intensity"] != 0.1 {
					t.Errorf("expected intensity=0.1, got %v", third["intensity"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, req := tt.setup()
			rec := httptest.NewRecorder()

			h.HandleAggregate(rec, req)

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
