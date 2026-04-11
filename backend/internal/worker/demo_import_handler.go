package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// DemoImportStream is the Redis Stream name for demo import jobs.
const DemoImportStream = "demo_import"

// ImportFunc is the function signature for importing a demo by match ID.
// It should return nil for expected skip conditions (already imported, no URL).
type ImportFunc func(ctx context.Context, userID, matchID uuid.UUID) error

// NewDemoImportHandler creates a JobHandler that processes demo import jobs.
func NewDemoImportHandler(importFn ImportFunc) JobHandler {
	return func(ctx context.Context, data map[string]interface{}) error {
		userIDStr, ok := data["user_id"].(string)
		if !ok || userIDStr == "" {
			return fmt.Errorf("missing or invalid user_id in job data")
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return fmt.Errorf("invalid user_id UUID %q: %w", userIDStr, err)
		}

		matchIDStr, ok := data["match_id"].(string)
		if !ok || matchIDStr == "" {
			return fmt.Errorf("missing or invalid match_id in job data")
		}

		matchID, err := uuid.Parse(matchIDStr)
		if err != nil {
			return fmt.Errorf("invalid match_id UUID %q: %w", matchIDStr, err)
		}

		if err := importFn(ctx, userID, matchID); err != nil {
			return fmt.Errorf("demo import failed: %w", err)
		}

		slog.Info("demo import completed",
			"user_id", userID,
			"match_id", matchID,
		)
		return nil
	}
}
