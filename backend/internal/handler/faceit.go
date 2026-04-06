package handler

import (
	"log/slog"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/worker"
)

// FaceitHandler handles Faceit-related HTTP endpoints.
type FaceitHandler struct {
	queue JobEnqueuer
}

// NewFaceitHandler creates a new FaceitHandler.
func NewFaceitHandler(queue JobEnqueuer) *FaceitHandler {
	return &FaceitHandler{queue: queue}
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
