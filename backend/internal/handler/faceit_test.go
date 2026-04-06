package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/handler"
)

type mockQueue struct {
	lastStream string
	lastData   map[string]interface{}
	err        error
}

func (m *mockQueue) Enqueue(_ context.Context, stream string, data map[string]interface{}) (string, error) {
	m.lastStream = stream
	m.lastData = data
	if m.err != nil {
		return "", m.err
	}
	return "msg-1", nil
}

func TestFaceitHandleSync(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		faceitID   string
		queueErr   error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid request returns 202",
			userID:     "user-123",
			faceitID:   "faceit-456",
			wantStatus: http.StatusAccepted,
			wantBody:   "sync_queued",
		},
		{
			name:       "missing auth returns 401",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing faceit_id returns 500",
			userID:     "user-123",
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "queue error returns 500",
			userID:     "user-123",
			faceitID:   "faceit-456",
			queueErr:   errors.New("redis down"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &mockQueue{err: tt.queueErr}
			h := handler.NewFaceitHandler(q)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/faceit/sync", nil)
			ctx := req.Context()
			if tt.userID != "" {
				ctx = context.WithValue(ctx, auth.UserIDKey, tt.userID)
			}
			if tt.faceitID != "" {
				ctx = context.WithValue(ctx, auth.FaceitIDKey, tt.faceitID)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			h.HandleSync(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantBody != "" {
				var body map[string]string
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("decoding response: %v", err)
				}
				if body["status"] != tt.wantBody {
					t.Errorf("body status = %q, want %q", body["status"], tt.wantBody)
				}
			}

			// Verify queue was called with correct data
			if tt.wantStatus == http.StatusAccepted {
				if q.lastStream != "faceit_sync" {
					t.Errorf("stream = %q, want faceit_sync", q.lastStream)
				}
				if q.lastData["user_id"] != tt.userID {
					t.Errorf("user_id = %v, want %s", q.lastData["user_id"], tt.userID)
				}
				if q.lastData["faceit_id"] != tt.faceitID {
					t.Errorf("faceit_id = %v, want %s", q.lastData["faceit_id"], tt.faceitID)
				}
			}
		})
	}
}
