package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// MaxDemoIDs is the maximum number of demo IDs allowed in a single heatmap request.
const MaxDemoIDs = 20

// HeatmapDemoChecker fetches demos for ownership verification.
type HeatmapDemoChecker interface {
	GetDemosByIDs(ctx context.Context, demoIds []uuid.UUID) ([]store.GetDemosByIDsRow, error)
}

// HeatmapStore provides heatmap aggregation data.
type HeatmapStore interface {
	GetHeatmapAggregation(ctx context.Context, arg store.GetHeatmapAggregationParams) ([]store.GetHeatmapAggregationRow, error)
}

// HeatmapHandler handles heatmap HTTP endpoints.
type HeatmapHandler struct {
	demos    HeatmapDemoChecker
	heatmaps HeatmapStore
}

// NewHeatmapHandler creates a new HeatmapHandler.
func NewHeatmapHandler(demos HeatmapDemoChecker, heatmaps HeatmapStore) *HeatmapHandler {
	return &HeatmapHandler{demos: demos, heatmaps: heatmaps}
}

var weaponCategories = map[string][]string{
	"rifle":   {"AK-47", "M4A4", "M4A1", "FAMAS", "Galil AR", "AUG", "SG 553"},
	"pistol":  {"Glock-18", "USP-S", "P2000", "P250", "Five-SeveN", "Tec-9", "CZ75 Auto", "Desert Eagle", "R8 Revolver", "Dual Berettas"},
	"smg":     {"MP9", "MAC-10", "MP7", "UMP-45", "P90", "PP-Bizon", "MP5-SD"},
	"sniper":  {"AWP", "SSG 08", "SCAR-20", "G3SG1"},
	"shotgun": {"Nova", "XM1014", "Sawed-Off", "MAG-7"},
}

type heatmapRequest struct {
	DemoIDs []string       `json:"demo_ids"`
	Filters heatmapFilters `json:"filters"`
}

type heatmapFilters struct {
	Side           string `json:"side"`
	WeaponCategory string `json:"weapon_category"`
	PlayerSteamID  string `json:"player_steam_id"`
}

type heatmapPointResponse struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Intensity float64 `json:"intensity"`
}

// HandleAggregate returns aggregated kill heatmap data for one or more demos.
func (h *HeatmapHandler) HandleAggregate(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user id"})
		return
	}

	var req heatmapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.DemoIDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "demo_ids is required and must not be empty"})
		return
	}

	if len(req.DemoIDs) > MaxDemoIDs {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "demo_ids must not exceed 20"})
		return
	}

	// Parse, validate, and deduplicate demo UUIDs.
	seen := make(map[uuid.UUID]struct{}, len(req.DemoIDs))
	demoIDs := make([]uuid.UUID, 0, len(req.DemoIDs))
	for _, idStr := range req.DemoIDs {
		id, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid demo id: " + idStr})
			return
		}
		if _, dup := seen[id]; !dup {
			seen[id] = struct{}{}
			demoIDs = append(demoIDs, id)
		}
	}

	// Validate filters.
	if req.Filters.Side != "" && req.Filters.Side != "CT" && req.Filters.Side != "T" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "side must be CT or T"})
		return
	}

	var weapons []string
	if req.Filters.WeaponCategory != "" {
		var ok bool
		weapons, ok = weaponCategories[req.Filters.WeaponCategory]
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid weapon_category"})
			return
		}
	}

	// Ownership check: all demos must exist and belong to the authenticated user.
	demos, err := h.demos.GetDemosByIDs(r.Context(), demoIDs)
	if err != nil {
		slog.Error("getting demos for heatmap", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get demos"})
		return
	}

	if len(demos) != len(demoIDs) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "one or more demos not found"})
		return
	}

	var mapName string
	for _, d := range demos {
		if d.UserID != userID {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "one or more demos not found"})
			return
		}
		if !d.MapName.Valid || d.MapName.String == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "one or more demos has no map data"})
			return
		}
		if mapName == "" {
			mapName = d.MapName.String
		} else if d.MapName.String != mapName {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "all demos must be from the same map"})
			return
		}
	}

	// Build query params.
	params := store.GetHeatmapAggregationParams{
		DemoIds: demoIDs,
		Weapons: weapons,
	}
	if params.Weapons == nil {
		params.Weapons = []string{}
	}

	if req.Filters.PlayerSteamID != "" {
		params.PlayerSteamID = sql.NullString{String: req.Filters.PlayerSteamID, Valid: true}
	}

	if req.Filters.Side != "" {
		params.Side = sql.NullString{String: req.Filters.Side, Valid: true}
	}

	rows, err := h.heatmaps.GetHeatmapAggregation(r.Context(), params)
	if err != nil {
		slog.Error("querying heatmap aggregation", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query heatmap data"})
		return
	}

	// Normalize intensity.
	points := make([]heatmapPointResponse, 0, len(rows))
	if len(rows) > 0 {
		var maxCount int64
		for _, row := range rows {
			if row.KillCount > maxCount {
				maxCount = row.KillCount
			}
		}

		for _, row := range rows {
			points = append(points, heatmapPointResponse{
				X:         row.X.Float64,
				Y:         row.Y.Float64,
				Intensity: float64(row.KillCount) / float64(maxCount),
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": points,
	})
}
