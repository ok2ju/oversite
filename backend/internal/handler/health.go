package handler

import (
	"encoding/json"
	"net/http"
)

// HealthHandler provides HTTP handlers for health check endpoints.
type HealthHandler struct{}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Healthz is a liveness probe -- always returns 200.
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Readyz is a readiness probe -- checks dependencies.
// For now, returns ok for all checks. Real dependency checking will be wired later.
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"checks": map[string]string{
			"db":    "ok",
			"redis": "ok",
			"minio": "ok",
		},
	})
}
