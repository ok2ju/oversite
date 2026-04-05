package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/demo"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// DemoStore is the subset of store.Queries needed by DemoHandler.
type DemoStore interface {
	CreateDemo(ctx context.Context, arg store.CreateDemoParams) (store.Demo, error)
	ListDemosByUserID(ctx context.Context, arg store.ListDemosByUserIDParams) ([]store.Demo, error)
	CountDemosByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	GetDemoByID(ctx context.Context, id uuid.UUID) (store.Demo, error)
	DeleteDemo(ctx context.Context, id uuid.UUID) error
}

// ObjectStore is the subset of storage.MinIOClient needed by DemoHandler.
type ObjectStore interface {
	PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64) error
	DeleteObject(ctx context.Context, bucket, key string) error
}

// JobEnqueuer is the subset of worker.RedisQueue needed by DemoHandler.
type JobEnqueuer interface {
	Enqueue(ctx context.Context, stream string, data map[string]interface{}) (string, error)
}

// DemoHandler handles demo-related HTTP endpoints.
type DemoHandler struct {
	store  DemoStore
	s3     ObjectStore
	queue  JobEnqueuer
	bucket string
}

// NewDemoHandler creates a new DemoHandler.
func NewDemoHandler(store DemoStore, s3 ObjectStore, queue JobEnqueuer, bucket string) *DemoHandler {
	return &DemoHandler{
		store:  store,
		s3:     s3,
		queue:  queue,
		bucket: bucket,
	}
}

// HandleUpload processes a demo file upload via multipart/form-data.
// It validates the file, streams it to object storage, creates a DB record,
// and enqueues a parse job. Returns 202 Accepted on success.
func (h *DemoHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
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

	// Cap request body to prevent abuse. Add 1 MiB headroom for multipart
	// boundary markers and headers so a file exactly at MaxUploadSize is accepted.
	r.Body = http.MaxBytesReader(w, r.Body, demo.MaxUploadSize+1<<20)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "file too large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file field"})
		return
	}
	defer func() { _ = file.Close() }()

	if err := demo.ValidateExtension(header.Filename); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := demo.ValidateSize(header.Size); err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": err.Error()})
		return
	}

	// Peek first 8 bytes for magic bytes validation.
	peeked := make([]byte, 8)
	n, err := io.ReadFull(file, peeked)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot read file"})
		return
	}
	peeked = peeked[:n]

	if err := demo.ValidateMagicBytes(peeked); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Reconstruct the full stream (peeked bytes + rest of file).
	reader := io.MultiReader(bytes.NewReader(peeked), file)

	demoID := uuid.New()
	key := fmt.Sprintf("demos/%s/%s.dem", userID, demoID)

	// Upload to object storage.
	if err := h.s3.PutObject(r.Context(), h.bucket, key, reader, header.Size); err != nil {
		slog.Error("uploading demo to S3", "error", err, "key", key, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store file"})
		return
	}

	// Create DB record.
	d, err := h.store.CreateDemo(r.Context(), store.CreateDemoParams{
		UserID:   userID,
		FilePath: key,
		FileSize: header.Size,
		Status:   "uploaded",
	})
	if err != nil {
		slog.Error("creating demo record", "error", err, "key", key, "request_id", chimw.GetReqID(r.Context()))
		// Clean up the orphaned S3 object.
		if delErr := h.s3.DeleteObject(r.Context(), h.bucket, key); delErr != nil {
			slog.Error("cleaning up S3 object after DB failure", "error", delErr, "key", key, "request_id", chimw.GetReqID(r.Context()))
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create demo record"})
		return
	}

	// Enqueue parse job — non-fatal on failure.
	if _, err := h.queue.Enqueue(r.Context(), "demo_parse", map[string]interface{}{
		"demo_id":   d.ID.String(),
		"file_path": key,
		"user_id":   userID.String(),
	}); err != nil {
		slog.Warn("enqueueing parse job", "error", err, "demo_id", d.ID, "request_id", chimw.GetReqID(r.Context()))
	}

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"data": map[string]interface{}{
			"id":         d.ID.String(),
			"status":     d.Status,
			"file_size":  d.FileSize,
			"created_at": d.CreatedAt,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// demoResponse is the JSON-safe representation of a store.Demo.
type demoResponse struct {
	ID           string   `json:"id"`
	MapName      *string  `json:"map_name"`
	FileSize     int64    `json:"file_size"`
	Status       string   `json:"status"`
	TotalTicks   *int32   `json:"total_ticks"`
	TickRate     *float64 `json:"tick_rate"`
	DurationSecs *int32   `json:"duration_secs"`
	MatchDate    *string  `json:"match_date"`
	CreatedAt    string   `json:"created_at"`
}

func demoToJSON(d store.Demo) demoResponse {
	resp := demoResponse{
		ID:        d.ID.String(),
		FileSize:  d.FileSize,
		Status:    d.Status,
		CreatedAt: d.CreatedAt.Format(time.RFC3339),
	}
	if d.MapName.Valid {
		resp.MapName = &d.MapName.String
	}
	if d.TotalTicks.Valid {
		resp.TotalTicks = &d.TotalTicks.Int32
	}
	if d.TickRate.Valid {
		resp.TickRate = &d.TickRate.Float64
	}
	if d.DurationSecs.Valid {
		resp.DurationSecs = &d.DurationSecs.Int32
	}
	if d.MatchDate.Valid {
		s := d.MatchDate.Time.Format(time.RFC3339)
		resp.MatchDate = &s
	}
	return resp
}

// HandleList returns a paginated list of demos for the authenticated user.
func (h *DemoHandler) HandleList(w http.ResponseWriter, r *http.Request) {
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

	page, perPage := parsePagination(r)

	total, err := h.store.CountDemosByUserID(r.Context(), userID)
	if err != nil {
		slog.Error("counting demos", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to count demos"})
		return
	}

	demos, err := h.store.ListDemosByUserID(r.Context(), store.ListDemosByUserIDParams{
		UserID: userID,
		Limit:  int32(perPage),
		Offset: int32((page - 1) * perPage),
	})
	if err != nil {
		slog.Error("listing demos", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list demos"})
		return
	}

	items := make([]demoResponse, 0, len(demos))
	for _, d := range demos {
		items = append(items, demoToJSON(d))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": items,
		"meta": map[string]interface{}{
			"total":    total,
			"page":     page,
			"per_page": perPage,
		},
	})
}

// HandleGet returns a single demo by ID for the authenticated user.
func (h *DemoHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
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

	d, err := h.store.GetDemoByID(r.Context(), demoID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "demo not found"})
			return
		}
		slog.Error("getting demo", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get demo"})
		return
	}

	if d.UserID != userID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "demo not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": demoToJSON(d),
	})
}

// HandleDelete removes a demo by ID for the authenticated user.
func (h *DemoHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
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

	d, err := h.store.GetDemoByID(r.Context(), demoID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "demo not found"})
			return
		}
		slog.Error("getting demo for delete", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get demo"})
		return
	}

	if d.UserID != userID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "demo not found"})
		return
	}

	if err := h.store.DeleteDemo(r.Context(), demoID); err != nil {
		slog.Error("deleting demo", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete demo"})
		return
	}

	// Best-effort S3 cleanup.
	if err := h.s3.DeleteObject(r.Context(), h.bucket, d.FilePath); err != nil {
		slog.Error("deleting S3 object", "error", err, "key", d.FilePath, "request_id", chimw.GetReqID(r.Context()))
	}

	w.WriteHeader(http.StatusNoContent)
}

func parsePagination(r *http.Request) (page, perPage int) {
	page = 1
	perPage = 20

	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && pp > 0 {
			perPage = pp
		}
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}
