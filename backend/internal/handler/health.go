package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// HealthChecker defines a dependency that can be pinged for health.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// HealthHandler provides HTTP handlers for health check endpoints.
type HealthHandler struct {
	db    HealthChecker
	redis HealthChecker
	minio HealthChecker
}

// NewHealthHandler creates a new HealthHandler with real dependency checkers.
// Any checker may be nil, in which case it reports "not_configured".
func NewHealthHandler(db, redis, minio HealthChecker) *HealthHandler {
	return &HealthHandler{db: db, redis: redis, minio: minio}
}

// Healthz is a liveness probe -- always returns 200.
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Readyz is a readiness probe -- checks dependencies.
// Returns 200 if all checks pass, 503 if any fail.
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{}
	allOK := true

	for name, checker := range map[string]HealthChecker{
		"db":    h.db,
		"redis": h.redis,
		"minio": h.minio,
	} {
		if checker == nil {
			checks[name] = "not_configured"
			allOK = false
			continue
		}
		if err := checker.Ping(ctx); err != nil {
			checks[name] = "fail"
			allOK = false
		} else {
			checks[name] = "ok"
		}
	}

	status := "ok"
	httpStatus := http.StatusOK
	if !allOK {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": status,
		"checks": checks,
	})
}
