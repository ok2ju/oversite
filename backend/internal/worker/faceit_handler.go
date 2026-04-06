package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// FaceitSyncStream is the Redis Stream name for Faceit sync jobs.
const FaceitSyncStream = "faceit_sync"

// Syncer abstracts the Faceit sync operation for testability.
type Syncer interface {
	Sync(ctx context.Context, userID uuid.UUID, faceitID string) (int, error)
}

// NewFaceitSyncHandler creates a JobHandler that processes Faceit sync jobs.
func NewFaceitSyncHandler(syncer Syncer) JobHandler {
	return func(ctx context.Context, data map[string]interface{}) error {
		userIDStr, ok := data["user_id"].(string)
		if !ok || userIDStr == "" {
			return fmt.Errorf("missing or invalid user_id in job data")
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return fmt.Errorf("invalid user_id UUID %q: %w", userIDStr, err)
		}

		faceitID, ok := data["faceit_id"].(string)
		if !ok || faceitID == "" {
			return fmt.Errorf("missing or invalid faceit_id in job data")
		}

		count, err := syncer.Sync(ctx, userID, faceitID)
		if err != nil {
			return fmt.Errorf("faceit sync failed: %w", err)
		}

		slog.Info("faceit sync completed",
			"user_id", userID,
			"faceit_id", faceitID,
			"matches_synced", count,
		)
		return nil
	}
}
