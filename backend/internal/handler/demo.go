package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/demo"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// DemoStore is the subset of store.Queries needed by DemoHandler.
type DemoStore interface {
	CreateDemo(ctx context.Context, arg store.CreateDemoParams) (store.Demo, error)
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
