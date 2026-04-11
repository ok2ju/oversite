package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/store"
	"github.com/ok2ju/oversite/backend/internal/worker"
)

// FaceitStore is the subset of store.Queries needed by FaceitHandler.
type FaceitStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (store.User, error)
	GetEloHistory(ctx context.Context, arg store.GetEloHistoryParams) ([]store.GetEloHistoryRow, error)
	CountFaceitMatchesByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	GetCurrentStreak(ctx context.Context, userID uuid.UUID) ([]string, error)
}

// FaceitHandler handles Faceit-related HTTP endpoints.
type FaceitHandler struct {
	queue JobEnqueuer
	store FaceitStore
}

// NewFaceitHandler creates a new FaceitHandler.
func NewFaceitHandler(queue JobEnqueuer, store FaceitStore) *FaceitHandler {
	return &FaceitHandler{queue: queue, store: store}
}

// HandleSync enqueues a Faceit match history sync job for the authenticated user.
// Returns 202 Accepted on success.
func (h *FaceitHandler) HandleSync(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	faceitID, ok := auth.FaceitIDFromContext(r.Context())
	if !ok {
		slog.Error("faceit_id not found in context", "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "faceit_id not available"})
		return
	}

	if _, err := h.queue.Enqueue(r.Context(), worker.FaceitSyncStream, map[string]interface{}{
		"user_id":   userIDStr,
		"faceit_id": faceitID,
	}); err != nil {
		slog.Error("enqueueing faceit sync job", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to enqueue sync job"})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "sync_queued"})
}

// streakResult computes the current streak from a list of recent match results.
func streakResult(results []string) (string, int) {
	if len(results) == 0 {
		return "none", 0
	}
	streakType := results[0]
	count := 0
	for _, r := range results {
		if r != streakType {
			break
		}
		count++
	}
	return streakType, count
}

type profileResponse struct {
	Data profileData `json:"data"`
}

type profileData struct {
	Nickname      string        `json:"nickname"`
	AvatarURL     *string       `json:"avatar_url"`
	Elo           *int32        `json:"elo"`
	Level         *int16        `json:"level"`
	Country       *string       `json:"country"`
	MatchesPlayed int64         `json:"matches_played"`
	CurrentStreak currentStreak `json:"current_streak"`
}

type currentStreak struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// HandleGetProfile returns the authenticated user's Faceit profile data.
func (h *FaceitHandler) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
		return
	}

	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		slog.Error("getting user", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	matchCount, err := h.store.CountFaceitMatchesByUserID(r.Context(), userID)
	if err != nil {
		slog.Error("counting matches", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	recentResults, err := h.store.GetCurrentStreak(r.Context(), userID)
	if err != nil {
		slog.Error("getting streak", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	streakType, streakCount := streakResult(recentResults)

	pd := profileData{
		Nickname:      user.Nickname,
		MatchesPlayed: matchCount,
		CurrentStreak: currentStreak{Type: streakType, Count: streakCount},
	}
	if user.AvatarUrl.Valid {
		pd.AvatarURL = &user.AvatarUrl.String
	}
	if user.FaceitElo.Valid {
		pd.Elo = &user.FaceitElo.Int32
	}
	if user.FaceitLevel.Valid {
		pd.Level = &user.FaceitLevel.Int16
	}
	if user.Country.Valid {
		pd.Country = &user.Country.String
	}

	writeJSON(w, http.StatusOK, profileResponse{Data: pd})
}

type eloHistoryResponse struct {
	Data []eloHistoryPoint `json:"data"`
}

type eloHistoryPoint struct {
	Elo      *int32 `json:"elo"`
	MapName  string `json:"map_name"`
	PlayedAt string `json:"played_at"`
}

// HandleGetEloHistory returns the authenticated user's ELO history.
func (h *FaceitHandler) HandleGetEloHistory(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		d, err := strconv.Atoi(daysStr)
		if err != nil || (d != 30 && d != 90 && d != 180 && d != 0) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "days must be 0, 30, 90, or 180"})
			return
		}
		days = d
	}

	var since time.Time
	if days > 0 {
		since = time.Now().AddDate(0, 0, -days)
	}

	rows, err := h.store.GetEloHistory(r.Context(), store.GetEloHistoryParams{
		UserID:   userID,
		PlayedAt: since,
	})
	if err != nil {
		slog.Error("getting elo history", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	points := make([]eloHistoryPoint, 0, len(rows))
	for _, row := range rows {
		p := eloHistoryPoint{
			MapName:  row.MapName,
			PlayedAt: row.PlayedAt.Format(time.RFC3339),
		}
		if row.EloAfter.Valid {
			p.Elo = &row.EloAfter.Int32
		}
		points = append(points, p)
	}

	writeJSON(w, http.StatusOK, eloHistoryResponse{Data: points})
}
