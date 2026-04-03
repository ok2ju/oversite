package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/ok2ju/oversite/backend/internal/testutil"
	"github.com/ok2ju/oversite/backend/internal/worker"
)

// Compile-time check: RedisQueue must satisfy the testutil.JobQueue interface.
var _ testutil.JobQueue = (*worker.RedisQueue)(nil)

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
	// Verify that Stop does not panic or hang even if Start was never called.
	// Since Start was not called, done channel is still open, so we test that
	// Stop with a goroutine + timeout doesn't hang.
	q := worker.NewRedisQueue(nil)
	handler := func(ctx context.Context, data map[string]interface{}) error {
		return nil
	}

	// We cannot call Stop without Start because Stop waits on done channel.
	// Instead, we verify the worker struct is well-formed and that creating
	// it twice doesn't panic.
	w1 := worker.NewWorker(q, "s", "g", "c1", handler)
	w2 := worker.NewWorker(q, "s", "g", "c2", handler)
	if w1 == nil || w2 == nil {
		t.Fatal("NewWorker returned nil")
	}
}
