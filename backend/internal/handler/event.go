package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// EventStore is the subset of store.Queries needed by EventHandler.
type EventStore interface {
	GetGameEventsByDemoID(ctx context.Context, demoID uuid.UUID) ([]store.GameEvent, error)
	GetGameEventsByDemoAndRound(ctx context.Context, arg store.GetGameEventsByDemoAndRoundParams) ([]store.GameEvent, error)
}

// EventHandler handles game event HTTP endpoints.
type EventHandler struct {
	demos  DemoGetter
	events EventStore
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(demos DemoGetter, events EventStore) *EventHandler {
	return &EventHandler{demos: demos, events: events}
}

// gameEventResponse is the JSON representation of a single game event.
type gameEventResponse struct {
	ID              string          `json:"id"`
	DemoID          string          `json:"demo_id"`
	RoundID         *string         `json:"round_id"`
	Tick            int32           `json:"tick"`
	EventType       string          `json:"event_type"`
	AttackerSteamID *string         `json:"attacker_steam_id"`
	VictimSteamID   *string         `json:"victim_steam_id"`
	Weapon          *string         `json:"weapon"`
	X               *float64        `json:"x"`
	Y               *float64        `json:"y"`
	Z               *float64        `json:"z"`
	ExtraData       json.RawMessage `json:"extra_data"`
}

func gameEventToResponse(e store.GameEvent) gameEventResponse {
	resp := gameEventResponse{
		ID:        e.ID.String(),
		DemoID:    e.DemoID.String(),
		Tick:      e.Tick,
		EventType: e.EventType,
	}
	if e.RoundID.Valid {
		s := e.RoundID.UUID.String()
		resp.RoundID = &s
	}
	if e.AttackerSteamID.Valid {
		resp.AttackerSteamID = &e.AttackerSteamID.String
	}
	if e.VictimSteamID.Valid {
		resp.VictimSteamID = &e.VictimSteamID.String
	}
	if e.Weapon.Valid {
		resp.Weapon = &e.Weapon.String
	}
	if e.X.Valid {
		resp.X = &e.X.Float64
	}
	if e.Y.Valid {
		resp.Y = &e.Y.Float64
	}
	if e.Z.Valid {
		resp.Z = &e.Z.Float64
	}
	if e.ExtraData.Valid {
		resp.ExtraData = e.ExtraData.RawMessage
	}
	return resp
}

// HandleGetEvents returns game events for a demo.
// Optional query param: round_id (UUID) to filter by round.
func (h *EventHandler) HandleGetEvents(w http.ResponseWriter, r *http.Request) {
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

	// Ownership check.
	d, err := h.demos.GetDemoByID(r.Context(), demoID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "demo not found"})
			return
		}
		slog.Error("getting demo for events", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get demo"})
		return
	}

	if d.UserID != userID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "demo not found"})
		return
	}

	// Query events, optionally filtered by round.
	var rows []store.GameEvent
	roundIDStr := r.URL.Query().Get("round_id")
	if roundIDStr != "" {
		roundID, parseErr := uuid.Parse(roundIDStr)
		if parseErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid round_id"})
			return
		}
		rows, err = h.events.GetGameEventsByDemoAndRound(r.Context(), store.GetGameEventsByDemoAndRoundParams{
			DemoID:  demoID,
			RoundID: uuid.NullUUID{UUID: roundID, Valid: true},
		})
	} else {
		rows, err = h.events.GetGameEventsByDemoID(r.Context(), demoID)
	}

	if err != nil {
		slog.Error("querying game events", "error", err, "demo_id", demoID, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query events"})
		return
	}

	items := make([]gameEventResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, gameEventToResponse(row))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": items,
	})
}
