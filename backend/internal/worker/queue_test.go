package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/ok2ju/oversite/backend/internal/testutil"
	"github.com/ok2ju/oversite/backend/internal/worker"
)

// Compile-time checks: RedisQueue must satisfy both interfaces.
var _ testutil.JobQueue = (*worker.RedisQueue)(nil)
var _ worker.Queue = (*worker.RedisQueue)(nil)

func TestNewRedisQueue_NotNil(t *testing.T) {
	// We can't connect to a real Redis in unit tests, but we verify
	// that NewRedisQueue returns a non-nil value with a nil client.
	q := worker.NewRedisQueue(nil)
	if q == nil {
		t.Fatal("NewRedisQueue returned nil")
	}
}

func TestNewWorker_Defaults(t *testing.T) {
	q := worker.NewRedisQueue(nil)
	handler := func(ctx context.Context, data map[string]interface{}) error {
		return nil
	}

	w := worker.NewWorker(q, "test-stream", "test-group", "test-consumer", handler)
	if w == nil {
		t.Fatal("NewWorker returned nil")
	}
}

func TestNewWorker_WithMaxRetry(t *testing.T) {
	q := worker.NewRedisQueue(nil)
	handler := func(ctx context.Context, data map[string]interface{}) error {
		return nil
	}

	w := worker.NewWorker(q, "s", "g", "c", handler).WithMaxRetry(5)
	if w == nil {
		t.Fatal("WithMaxRetry returned nil")
	}
}

func TestNewWorker_WithStaleThreshold(t *testing.T) {
	q := worker.NewRedisQueue(nil)
	handler := func(ctx context.Context, data map[string]interface{}) error {
		return nil
	}

	w := worker.NewWorker(q, "s", "g", "c", handler).WithStaleThreshold(1 * time.Minute)
	if w == nil {
		t.Fatal("WithStaleThreshold returned nil")
	}
}

func TestWorker_StopWithoutStart(t *testing.T) {
	q := worker.NewRedisQueue(nil)
	handler := func(ctx context.Context, data map[string]interface{}) error {
		return nil
	}

	w := worker.NewWorker(q, "s", "g", "c", handler)

	// Stop should return promptly even though Start was never called.
	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success: Stop returned without blocking.
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() blocked without Start() being called")
	}
}
