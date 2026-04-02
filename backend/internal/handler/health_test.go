package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ok2ju/oversite/backend/internal/handler"
)

type stubChecker struct {
	err error
}

func (s *stubChecker) Ping(_ context.Context) error {
	return s.err
}

func TestHealthz(t *testing.T) {
	h := handler.NewHealthHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.Healthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", body["status"])
	}
}

func TestReadyz_AllHealthy(t *testing.T) {
	h := handler.NewHealthHandler(
		&stubChecker{},
		&stubChecker{},
		&stubChecker{},
	)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.Readyz(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", body.Status)
	}

	for _, name := range []string{"db", "redis", "minio"} {
		if body.Checks[name] != "ok" {
			t.Errorf("expected check %q to be 'ok', got %q", name, body.Checks[name])
		}
	}
}

func TestReadyz_DependencyFailing(t *testing.T) {
	h := handler.NewHealthHandler(
		&stubChecker{},
		&stubChecker{err: errors.New("connection refused")},
		&stubChecker{},
	)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.Readyz(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body.Status != "degraded" {
		t.Errorf("expected status 'degraded', got %q", body.Status)
	}

	if body.Checks["redis"] != "fail" {
		t.Errorf("expected redis check 'fail', got %q", body.Checks["redis"])
	}
}

func TestReadyz_NilChecker(t *testing.T) {
	h := handler.NewHealthHandler(
		&stubChecker{},
		nil,
		&stubChecker{},
	)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.Readyz(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body.Checks["redis"] != "not_configured" {
		t.Errorf("expected redis check 'not_configured', got %q", body.Checks["redis"])
	}
}
