package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/demo"
	"github.com/ok2ju/oversite/backend/internal/handler"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// --- Demo test mocks ---

type mockDemoStore struct {
	createDemoFn       func(ctx context.Context, arg store.CreateDemoParams) (store.Demo, error)
	listDemosByUserFn  func(ctx context.Context, arg store.ListDemosByUserIDParams) ([]store.Demo, error)
	countDemosByUserFn func(ctx context.Context, userID uuid.UUID) (int64, error)
	getDemoByIDFn      func(ctx context.Context, id uuid.UUID) (store.Demo, error)
	deleteDemoFn       func(ctx context.Context, id uuid.UUID) error
}

func (m *mockDemoStore) CreateDemo(ctx context.Context, arg store.CreateDemoParams) (store.Demo, error) {
	return m.createDemoFn(ctx, arg)
}

func (m *mockDemoStore) ListDemosByUserID(ctx context.Context, arg store.ListDemosByUserIDParams) ([]store.Demo, error) {
	return m.listDemosByUserFn(ctx, arg)
}

func (m *mockDemoStore) CountDemosByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	return m.countDemosByUserFn(ctx, userID)
}

func (m *mockDemoStore) GetDemoByID(ctx context.Context, id uuid.UUID) (store.Demo, error) {
	return m.getDemoByIDFn(ctx, id)
}

func (m *mockDemoStore) DeleteDemo(ctx context.Context, id uuid.UUID) error {
	return m.deleteDemoFn(ctx, id)
}

type mockObjectStore struct {
	putErr    error
	deleteErr error
	deleted   []string
}

func (m *mockObjectStore) PutObject(_ context.Context, _, _ string, _ io.Reader, _ int64) error {
	return m.putErr
}

func (m *mockObjectStore) DeleteObject(_ context.Context, _, key string) error {
	m.deleted = append(m.deleted, key)
	return m.deleteErr
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

var (
	testUserID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	testDemoID = uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
)

func testDemo() store.Demo {
	return store.Demo{
		ID:           testDemoID,
		UserID:       testUserID,
		MapName:      sql.NullString{String: "de_dust2", Valid: true},
		FilePath:     "demos/550e8400-e29b-41d4-a716-446655440000/660e8400-e29b-41d4-a716-446655440000.dem",
		FileSize:     1024000,
		Status:       "ready",
		TotalTicks:   sql.NullInt32{Int32: 128000, Valid: true},
		TickRate:     sql.NullFloat64{Float64: 64, Valid: true},
		DurationSecs: sql.NullInt32{Int32: 2000, Valid: true},
	}
}

// withChiURLParam sets chi URL params on the request context.
func withChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
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

func TestHandleListDemos(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() (*handler.DemoHandler, *http.Request)
		wantStatus int
		checkBody  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "valid request returns demos and meta",
			setup: func() (*handler.DemoHandler, *http.Request) {
				ms := &mockDemoStore{
					countDemosByUserFn: func(_ context.Context, _ uuid.UUID) (int64, error) {
						return 1, nil
					},
					listDemosByUserFn: func(_ context.Context, _ store.ListDemosByUserIDParams) ([]store.Demo, error) {
						return []store.Demo{testDemo()}, nil
					},
				}
				h := handler.NewDemoHandler(ms, &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos", nil)
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				data, ok := body["data"].([]interface{})
				if !ok {
					t.Fatal("expected 'data' array")
				}
				if len(data) != 1 {
					t.Fatalf("expected 1 demo, got %d", len(data))
				}
				meta, ok := body["meta"].(map[string]interface{})
				if !ok {
					t.Fatal("expected 'meta' object")
				}
				if meta["total"] != float64(1) {
					t.Errorf("expected total=1, got %v", meta["total"])
				}
			},
		},
		{
			name: "no auth returns 401",
			setup: func() (*handler.DemoHandler, *http.Request) {
				h := handler.NewDemoHandler(&mockDemoStore{}, &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos", nil)
				return h, req
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "default pagination page=1 per_page=20",
			setup: func() (*handler.DemoHandler, *http.Request) {
				var gotLimit, gotOffset int32
				ms := &mockDemoStore{
					countDemosByUserFn: func(_ context.Context, _ uuid.UUID) (int64, error) {
						return 0, nil
					},
					listDemosByUserFn: func(_ context.Context, arg store.ListDemosByUserIDParams) ([]store.Demo, error) {
						gotLimit = arg.Limit
						gotOffset = arg.Offset
						return nil, nil
					},
				}
				h := handler.NewDemoHandler(ms, &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos", nil)
				req = withUserID(req, testUserID.String())
				t.Cleanup(func() {
					if gotLimit != 20 {
						t.Errorf("expected limit=20, got %d", gotLimit)
					}
					if gotOffset != 0 {
						t.Errorf("expected offset=0, got %d", gotOffset)
					}
				})
				return h, req
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "custom pagination page=2 per_page=10",
			setup: func() (*handler.DemoHandler, *http.Request) {
				var gotLimit, gotOffset int32
				ms := &mockDemoStore{
					countDemosByUserFn: func(_ context.Context, _ uuid.UUID) (int64, error) {
						return 25, nil
					},
					listDemosByUserFn: func(_ context.Context, arg store.ListDemosByUserIDParams) ([]store.Demo, error) {
						gotLimit = arg.Limit
						gotOffset = arg.Offset
						return nil, nil
					},
				}
				h := handler.NewDemoHandler(ms, &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos?page=2&per_page=10", nil)
				req = withUserID(req, testUserID.String())
				t.Cleanup(func() {
					if gotLimit != 10 {
						t.Errorf("expected limit=10, got %d", gotLimit)
					}
					if gotOffset != 10 {
						t.Errorf("expected offset=10, got %d", gotOffset)
					}
				})
				return h, req
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "empty result returns 200 with empty array",
			setup: func() (*handler.DemoHandler, *http.Request) {
				ms := &mockDemoStore{
					countDemosByUserFn: func(_ context.Context, _ uuid.UUID) (int64, error) {
						return 0, nil
					},
					listDemosByUserFn: func(_ context.Context, _ store.ListDemosByUserIDParams) ([]store.Demo, error) {
						return nil, nil
					},
				}
				h := handler.NewDemoHandler(ms, &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos", nil)
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				data, ok := body["data"].([]interface{})
				if !ok {
					t.Fatal("expected 'data' array")
				}
				if len(data) != 0 {
					t.Errorf("expected empty array, got %d items", len(data))
				}
			},
		},
		{
			name: "DB error returns 500",
			setup: func() (*handler.DemoHandler, *http.Request) {
				ms := &mockDemoStore{
					countDemosByUserFn: func(_ context.Context, _ uuid.UUID) (int64, error) {
						return 0, errors.New("db down")
					},
				}
				h := handler.NewDemoHandler(ms, &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos", nil)
				req = withUserID(req, testUserID.String())
				return h, req
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, req := tt.setup()
			rec := httptest.NewRecorder()

			h.HandleList(rec, req)

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

func TestHandleGetDemo(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() (*handler.DemoHandler, *http.Request)
		wantStatus int
		checkBody  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "valid request returns demo",
			setup: func() (*handler.DemoHandler, *http.Request) {
				ms := &mockDemoStore{
					getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
						return testDemo(), nil
					},
				}
				h := handler.NewDemoHandler(ms, &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String(), nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				data, ok := body["data"].(map[string]interface{})
				if !ok {
					t.Fatal("expected 'data' object")
				}
				if data["id"] != testDemoID.String() {
					t.Errorf("expected id=%s, got %v", testDemoID, data["id"])
				}
				if data["map_name"] != "de_dust2" {
					t.Errorf("expected map_name=de_dust2, got %v", data["map_name"])
				}
			},
		},
		{
			name: "invalid UUID returns 400",
			setup: func() (*handler.DemoHandler, *http.Request) {
				h := handler.NewDemoHandler(&mockDemoStore{}, &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/not-a-uuid", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", "not-a-uuid")
				return h, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found returns 404",
			setup: func() (*handler.DemoHandler, *http.Request) {
				ms := &mockDemoStore{
					getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
						return store.Demo{}, sql.ErrNoRows
					},
				}
				h := handler.NewDemoHandler(ms, &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String(), nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "different user's demo returns 404",
			setup: func() (*handler.DemoHandler, *http.Request) {
				ms := &mockDemoStore{
					getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
						return testDemo(), nil
					},
				}
				h := handler.NewDemoHandler(ms, &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				otherUser := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String(), nil)
				req = withUserID(req, otherUser.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "no auth returns 401",
			setup: func() (*handler.DemoHandler, *http.Request) {
				h := handler.NewDemoHandler(&mockDemoStore{}, &mockObjectStore{}, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/demos/"+testDemoID.String(), nil)
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, req
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, req := tt.setup()
			rec := httptest.NewRecorder()

			h.HandleGet(rec, req)

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

func TestHandleDeleteDemo(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() (*handler.DemoHandler, *mockObjectStore, *http.Request)
		wantStatus int
		checkS3    func(t *testing.T, s3 *mockObjectStore)
	}{
		{
			name: "valid delete returns 204",
			setup: func() (*handler.DemoHandler, *mockObjectStore, *http.Request) {
				ms := &mockDemoStore{
					getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
						return testDemo(), nil
					},
					deleteDemoFn: func(_ context.Context, _ uuid.UUID) error {
						return nil
					},
				}
				s3 := &mockObjectStore{}
				h := handler.NewDemoHandler(ms, s3, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodDelete, "/api/v1/demos/"+testDemoID.String(), nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, s3, req
			},
			wantStatus: http.StatusNoContent,
			checkS3: func(t *testing.T, s3 *mockObjectStore) {
				if len(s3.deleted) != 1 {
					t.Fatalf("expected 1 S3 delete, got %d", len(s3.deleted))
				}
			},
		},
		{
			name: "invalid UUID returns 400",
			setup: func() (*handler.DemoHandler, *mockObjectStore, *http.Request) {
				s3 := &mockObjectStore{}
				h := handler.NewDemoHandler(&mockDemoStore{}, s3, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodDelete, "/api/v1/demos/not-a-uuid", nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", "not-a-uuid")
				return h, s3, req
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found returns 404",
			setup: func() (*handler.DemoHandler, *mockObjectStore, *http.Request) {
				ms := &mockDemoStore{
					getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
						return store.Demo{}, sql.ErrNoRows
					},
				}
				s3 := &mockObjectStore{}
				h := handler.NewDemoHandler(ms, s3, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodDelete, "/api/v1/demos/"+testDemoID.String(), nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, s3, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "wrong user returns 404",
			setup: func() (*handler.DemoHandler, *mockObjectStore, *http.Request) {
				ms := &mockDemoStore{
					getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
						return testDemo(), nil
					},
				}
				s3 := &mockObjectStore{}
				h := handler.NewDemoHandler(ms, s3, &mockJobEnqueuer{}, "demos")
				otherUser := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				req := httptest.NewRequest(http.MethodDelete, "/api/v1/demos/"+testDemoID.String(), nil)
				req = withUserID(req, otherUser.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, s3, req
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "no auth returns 401",
			setup: func() (*handler.DemoHandler, *mockObjectStore, *http.Request) {
				s3 := &mockObjectStore{}
				h := handler.NewDemoHandler(&mockDemoStore{}, s3, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodDelete, "/api/v1/demos/"+testDemoID.String(), nil)
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, s3, req
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "DB delete error returns 500",
			setup: func() (*handler.DemoHandler, *mockObjectStore, *http.Request) {
				ms := &mockDemoStore{
					getDemoByIDFn: func(_ context.Context, _ uuid.UUID) (store.Demo, error) {
						return testDemo(), nil
					},
					deleteDemoFn: func(_ context.Context, _ uuid.UUID) error {
						return errors.New("db down")
					},
				}
				s3 := &mockObjectStore{}
				h := handler.NewDemoHandler(ms, s3, &mockJobEnqueuer{}, "demos")
				req := httptest.NewRequest(http.MethodDelete, "/api/v1/demos/"+testDemoID.String(), nil)
				req = withUserID(req, testUserID.String())
				req = withChiURLParam(req, "id", testDemoID.String())
				return h, s3, req
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, s3, req := tt.setup()
			rec := httptest.NewRecorder()

			h.HandleDelete(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d; body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			if tt.checkS3 != nil {
				tt.checkS3(t, s3)
			}
		})
	}
}
