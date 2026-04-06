package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// RosterStore is the subset of store.Queries needed by RosterHandler.
type RosterStore interface {
	GetRoundByDemoAndNumber(ctx context.Context, arg store.GetRoundByDemoAndNumberParams) (store.Round, error)
	GetPlayerRoundsByRoundID(ctx context.Context, roundID uuid.UUID) ([]store.PlayerRound, error)
}

// RosterHandler handles player roster HTTP endpoints.
type RosterHandler struct {
	demos DemoGetter
	store RosterStore
}

// NewRosterHandler creates a new RosterHandler.
func NewRosterHandler(demos DemoGetter, store RosterStore) *RosterHandler {
	return &RosterHandler{demos: demos, store: store}
}

// playerRosterEntry is the JSON representation of a single roster entry.
type playerRosterEntry struct {
	SteamID    string `json:"steam_id"`
	PlayerName string `json:"player_name"`
	TeamSide   string `json:"team_side"`
}

// HandleGetPlayers returns the player roster for a given demo round.
// Route: GET /api/v1/demos/{id}/rounds/{roundNumber}/players
func (h *RosterHandler) HandleGetPlayers(w http.ResponseWriter, r *http.Request) {
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

	roundNum64, err := strconv.ParseInt(chi.URLParam(r, "roundNumber"), 10, 16)
	if err != nil || roundNum64 < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid round number"})
		return
	}

	// Ownership check.
	d, err := h.demos.GetDemoByID(r.Context(), demoID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "demo not found"})
			return
		}
		slog.Error("getting demo for roster", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get demo"})
		return
	}

	if d.UserID != userID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "demo not found"})
		return
	}

	// Fetch round.
	round, err := h.store.GetRoundByDemoAndNumber(r.Context(), store.GetRoundByDemoAndNumberParams{
		DemoID:      demoID,
		RoundNumber: int16(roundNum64),
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "round not found"})
			return
		}
		slog.Error("getting round for roster", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get round"})
		return
	}

	// Fetch player rounds.
	playerRounds, err := h.store.GetPlayerRoundsByRoundID(r.Context(), round.ID)
	if err != nil {
		slog.Error("getting player rounds for roster", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get players"})
		return
	}

	items := make([]playerRosterEntry, 0, len(playerRounds))
	for _, pr := range playerRounds {
		items = append(items, playerRosterEntry{
			SteamID:    pr.SteamID,
			PlayerName: pr.PlayerName,
			TeamSide:   pr.TeamSide,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": items,
	})
}
