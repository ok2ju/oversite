package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// MaxTickRange is the maximum number of ticks that can be requested in a single call.
const MaxTickRange = 6400

// DemoGetter is a narrow interface for ownership checks.
type DemoGetter interface {
	GetDemoByID(ctx context.Context, id uuid.UUID) (store.Demo, error)
}

// TickStore is the subset of store.Queries needed by TickHandler.
type TickStore interface {
	GetTickDataByRange(ctx context.Context, arg store.GetTickDataByRangeParams) ([]store.TickDatum, error)
	GetTickDataByRangeAndPlayers(ctx context.Context, arg store.GetTickDataByRangeAndPlayersParams) ([]store.TickDatum, error)
}

// TickHandler handles tick data HTTP endpoints.
type TickHandler struct {
	demos DemoGetter
	ticks TickStore
}

// NewTickHandler creates a new TickHandler.
func NewTickHandler(demos DemoGetter, ticks TickStore) *TickHandler {
	return &TickHandler{demos: demos, ticks: ticks}
}

// tickDataResponse is the JSON representation of a single tick data row.
type tickDataResponse struct {
	Tick    int32   `json:"tick"`
	SteamID string  `json:"steam_id"`
	X       float32 `json:"x"`
	Y       float32 `json:"y"`
	Z       float32 `json:"z"`
	Yaw     float32 `json:"yaw"`
	Health  int16   `json:"health"`
	Armor   int16   `json:"armor"`
	IsAlive bool    `json:"is_alive"`
	Weapon  *string `json:"weapon"`
}

func tickDatumToResponse(td store.TickDatum) tickDataResponse {
	resp := tickDataResponse{
		Tick:    td.Tick,
		SteamID: td.SteamID,
		X:       td.X,
		Y:       td.Y,
		Z:       td.Z,
		Yaw:     td.Yaw,
		Health:  td.Health,
		Armor:   td.Armor,
		IsAlive: td.IsAlive,
	}
	if td.Weapon.Valid {
		resp.Weapon = &td.Weapon.String
	}
	return resp
}

// HandleGetTicks returns tick data for a demo within a tick range.
// Query params: start_tick, end_tick (required), steam_ids (optional, comma-separated).
func (h *TickHandler) HandleGetTicks(w http.ResponseWriter, r *http.Request) {
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

	demoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid demo id"})
		return
	}

	// Parse and validate tick range.
	startStr := r.URL.Query().Get("start_tick")
	endStr := r.URL.Query().Get("end_tick")
	if startStr == "" || endStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "start_tick and end_tick are required"})
		return
	}

	startTick, err := strconv.ParseInt(startStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid start_tick"})
		return
	}

	endTick, err := strconv.ParseInt(endStr, 10, 32)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid end_tick"})
		return
	}

	if startTick < 0 || endTick < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tick values must be non-negative"})
		return
	}

	if startTick > endTick {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "start_tick must not exceed end_tick"})
		return
	}

	if endTick-startTick >= MaxTickRange {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tick range exceeds maximum of 6400"})
		return
	}

	// Ownership check.
	d, err := h.demos.GetDemoByID(r.Context(), demoID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "demo not found"})
			return
		}
		slog.Error("getting demo for ticks", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get demo"})
		return
	}

	if d.UserID != userID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "demo not found"})
		return
	}

	// Query tick data.
	var rows []store.TickDatum

	steamIDsStr := r.URL.Query().Get("steam_ids")
	if steamIDsStr != "" {
		parts := strings.Split(steamIDsStr, ",")
		steamIDs := make([]string, 0, len(parts))
		for _, s := range parts {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				steamIDs = append(steamIDs, trimmed)
			}
		}
		rows, err = h.ticks.GetTickDataByRangeAndPlayers(r.Context(), store.GetTickDataByRangeAndPlayersParams{
			DemoID:  demoID,
			Tick:    int32(startTick),
			Tick_2:  int32(endTick),
			Column4: steamIDs,
		})
	} else {
		rows, err = h.ticks.GetTickDataByRange(r.Context(), store.GetTickDataByRangeParams{
			DemoID: demoID,
			Tick:   int32(startTick),
			Tick_2: int32(endTick),
		})
	}

	if err != nil {
		slog.Error("querying tick data", "error", err, "demo_id", demoID, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query tick data"})
		return
	}

	items := make([]tickDataResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, tickDatumToResponse(row))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": items,
	})
}
