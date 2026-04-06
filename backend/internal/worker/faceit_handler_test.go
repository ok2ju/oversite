package worker_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/worker"
)

type mockSyncer struct {
	calledUserID   uuid.UUID
	calledFaceitID string
	result         int
	err            error
}

func (m *mockSyncer) Sync(_ context.Context, userID uuid.UUID, faceitID string) (int, error) {
	m.calledUserID = userID
	m.calledFaceitID = faceitID
	return m.result, m.err
}

func TestFaceitSyncHandler(t *testing.T) {
	validUserID := uuid.New()

	tests := []struct {
		name       string
		data       map[string]interface{}
		syncer     *mockSyncer
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid payload calls Sync",
			data: map[string]interface{}{
				"user_id":   validUserID.String(),
				"faceit_id": "faceit-123",
			},
			syncer: &mockSyncer{result: 5},
		},
		{
			name:       "missing user_id returns error",
			data:       map[string]interface{}{"faceit_id": "faceit-123"},
			syncer:     &mockSyncer{},
			wantErr:    true,
			wantErrMsg: "missing or invalid user_id",
		},
		{
			name: "invalid UUID returns error",
			data: map[string]interface{}{
				"user_id":   "not-a-uuid",
				"faceit_id": "faceit-123",
			},
			syncer:     &mockSyncer{},
			wantErr:    true,
			wantErrMsg: "invalid user_id UUID",
		},
		{
			name: "missing faceit_id returns error",
			data: map[string]interface{}{
				"user_id": validUserID.String(),
			},
			syncer:     &mockSyncer{},
			wantErr:    true,
			wantErrMsg: "missing or invalid faceit_id",
		},
		{
			name: "Sync error propagates",
			data: map[string]interface{}{
				"user_id":   validUserID.String(),
				"faceit_id": "faceit-123",
			},
			syncer:     &mockSyncer{err: errors.New("api timeout")},
			wantErr:    true,
			wantErrMsg: "faceit sync failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := worker.NewFaceitSyncHandler(tt.syncer)
			err := handler(context.Background(), tt.data)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrMsg != "" && !contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErrMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.syncer.calledUserID != validUserID {
				t.Errorf("called with userID = %s, want %s", tt.syncer.calledUserID, validUserID)
			}
			if tt.syncer.calledFaceitID != "faceit-123" {
				t.Errorf("called with faceitID = %s, want faceit-123", tt.syncer.calledFaceitID)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
