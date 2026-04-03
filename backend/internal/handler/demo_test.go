package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/demo"
	"github.com/ok2ju/oversite/backend/internal/handler"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// --- Demo test mocks ---

type mockDemoStore struct {
	createDemoFn func(ctx context.Context, arg store.CreateDemoParams) (store.Demo, error)
}

func (m *mockDemoStore) CreateDemo(ctx context.Context, arg store.CreateDemoParams) (store.Demo, error) {
	return m.createDemoFn(ctx, arg)
}

type mockObjectStore struct {
	putErr error
}

func (m *mockObjectStore) PutObject(_ context.Context, _, _ string, _ io.Reader, _ int64) error {
	return m.putErr
}

type mockJobEnqueuer struct {
	enqueueErr error
}

func (m *mockJobEnqueuer) Enqueue(_ context.Context, _ string, _ map[string]interface{}) (string, error) {
	if m.enqueueErr != nil {
		return "", m.enqueueErr
	}
	return "msg-1", nil
}

// --- Helpers ---

// createUploadRequest builds a multipart/form-data request with the given filename and content.
func createUploadRequest(t *testing.T, filename string, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("creating form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("writing content: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("closing writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/demos", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

// withUserID adds a userID to the request context, simulating RequireAuth middleware.
func withUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), auth.UserIDKey, userID)
	return r.WithContext(ctx)
}

// validDemoContent returns file content starting with CS2 magic bytes.
func validDemoContent() []byte {
	content := make([]byte, 64)
	copy(content, demo.MagicCS2)
	return content
}

// defaultMockStore returns a mock that returns a successful demo creation.
func defaultMockStore() *mockDemoStore {
	return &mockDemoStore{
		createDemoFn: func(_ context.Context, arg store.CreateDemoParams) (store.Demo, error) {
			return store.Demo{
				UserID:   arg.UserID,
				FilePath: arg.FilePath,
				FileSize: arg.FileSize,
				Status:   arg.Status,
			}, nil
		},
	}
}

func TestHandleUpload(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() (*handler.DemoHandler, *http.Request)
		wantStatus int
		checkBody  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "valid CS2 demo upload",
			setup: func() (*handler.DemoHandler, *http.Request) {
				h := handler.NewDemoHandler(defaultMockStore(), &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				req := createUploadRequest(t, "match.dem", validDemoContent())
				req = withUserID(req, "550e8400-e29b-41d4-a716-446655440000")
				return h, req
			},
			wantStatus: http.StatusAccepted,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				data, ok := body["data"].(map[string]interface{})
				if !ok {
					t.Fatal("expected 'data' key in response")
				}
				if _, ok := data["id"]; !ok {
					t.Error("expected 'id' in data")
				}
				if data["status"] != "uploaded" {
					t.Errorf("expected status 'uploaded', got %v", data["status"])
				}
				if _, ok := data["file_size"]; !ok {
					t.Error("expected 'file_size' in data")
				}
				if _, ok := data["created_at"]; !ok {
					t.Error("expected 'created_at' in data")
				}
			},
		},
		{
			name: "invalid extension",
			setup: func() (*handler.DemoHandler, *http.Request) {
				h := handler.NewDemoHandler(defaultMockStore(), &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				req := createUploadRequest(t, "match.txt", validDemoContent())
				req = withUserID(req, "550e8400-e29b-41d4-a716-446655440000")
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "bad magic bytes",
			setup: func() (*handler.DemoHandler, *http.Request) {
				h := handler.NewDemoHandler(defaultMockStore(), &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				req := createUploadRequest(t, "match.dem", []byte("not-a-demo-file-at-all"))
				req = withUserID(req, "550e8400-e29b-41d4-a716-446655440000")
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "no auth context",
			setup: func() (*handler.DemoHandler, *http.Request) {
				h := handler.NewDemoHandler(defaultMockStore(), &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				req := createUploadRequest(t, "match.dem", validDemoContent())
				// No withUserID call — simulates missing auth
				return h, req
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "no file field",
			setup: func() (*handler.DemoHandler, *http.Request) {
				h := handler.NewDemoHandler(defaultMockStore(), &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				// Empty multipart form with no file
				var buf bytes.Buffer
				w := multipart.NewWriter(&buf)
				_ = w.WriteField("other", "value")
				_ = w.Close()
				req := httptest.NewRequest(http.MethodPost, "/api/v1/demos", &buf)
				req.Header.Set("Content-Type", w.FormDataContentType())
				req = withUserID(req, "550e8400-e29b-41d4-a716-446655440000")
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "S3 failure",
			setup: func() (*handler.DemoHandler, *http.Request) {
				h := handler.NewDemoHandler(
					defaultMockStore(),
					&mockObjectStore{putErr: errors.New("s3 down")},
					&mockJobEnqueuer{},
					"demos",
				)
				req := createUploadRequest(t, "match.dem", validDemoContent())
				req = withUserID(req, "550e8400-e29b-41d4-a716-446655440000")
				return h, req
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "DB failure",
			setup: func() (*handler.DemoHandler, *http.Request) {
				h := handler.NewDemoHandler(
					&mockDemoStore{
						createDemoFn: func(_ context.Context, _ store.CreateDemoParams) (store.Demo, error) {
							return store.Demo{}, errors.New("db down")
						},
					},
					&mockObjectStore{},
					&mockJobEnqueuer{},
					"demos",
				)
				req := createUploadRequest(t, "match.dem", validDemoContent())
				req = withUserID(req, "550e8400-e29b-41d4-a716-446655440000")
				return h, req
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "queue failure is non-fatal",
			setup: func() (*handler.DemoHandler, *http.Request) {
				h := handler.NewDemoHandler(
					defaultMockStore(),
					&mockObjectStore{},
					&mockJobEnqueuer{enqueueErr: errors.New("redis down")},
					"demos",
				)
				req := createUploadRequest(t, "match.dem", validDemoContent())
				req = withUserID(req, "550e8400-e29b-41d4-a716-446655440000")
				return h, req
			},
			wantStatus: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, req := tt.setup()
			rec := httptest.NewRecorder()

			h.HandleUpload(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d; body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			if tt.checkBody != nil {
				var body map[string]interface{}
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("decoding response body: %v", err)
				}
				tt.checkBody(t, body)
			}
		})
	}
}
