package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// RoundStore is the subset of store.Queries needed by RoundHandler.
type RoundStore interface {
	GetRoundsByDemoID(ctx context.Context, demoID uuid.UUID) ([]store.Round, error)
}

// RoundHandler handles round HTTP endpoints.
type RoundHandler struct {
	demos  DemoGetter
	rounds RoundStore
}

// NewRoundHandler creates a new RoundHandler.
func NewRoundHandler(demos DemoGetter, rounds RoundStore) *RoundHandler {
	return &RoundHandler{demos: demos, rounds: rounds}
}

// roundResponse is the JSON representation of a single round.
type roundResponse struct {
	ID          string `json:"id"`
	RoundNumber int16  `json:"round_number"`
	StartTick   int32  `json:"start_tick"`
	EndTick     int32  `json:"end_tick"`
	WinnerSide  string `json:"winner_side"`
	CtScore     int16  `json:"ct_score"`
	TScore      int16  `json:"t_score"`
}

func roundToResponse(r store.Round) roundResponse {
	return roundResponse{
		ID:          r.ID.String(),
		RoundNumber: r.RoundNumber,
		StartTick:   r.StartTick,
		EndTick:     r.EndTick,
		WinnerSide:  r.WinnerSide,
		CtScore:     r.CtScore,
		TScore:      r.TScore,
	}
}

// HandleGetRounds returns rounds for a demo.
func (h *RoundHandler) HandleGetRounds(w http.ResponseWriter, r *http.Request) {
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
		slog.Error("getting demo for rounds", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get demo"})
		return
	}

	if d.UserID != userID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "demo not found"})
		return
	}

	rows, err := h.rounds.GetRoundsByDemoID(r.Context(), demoID)
	if err != nil {
		slog.Error("querying rounds", "error", err, "demo_id", demoID, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query rounds"})
		return
	}

	items := make([]roundResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, roundToResponse(row))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": items,
	})
}
