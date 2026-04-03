//go:build integration

package worker_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ok2ju/oversite/backend/internal/testutil"
	"github.com/ok2ju/oversite/backend/internal/worker"
)

func setupRedisQueue(t *testing.T) (*worker.RedisQueue, *redis.Client) {
	t.Helper()
	ctx := context.Background()

	container, redisURL, err := testutil.RedisContainer(ctx)
	if err != nil {
		t.Fatalf("starting redis container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("parsing redis URL: %v", err)
	}
	client := redis.NewClient(opts)
	t.Cleanup(func() { _ = client.Close() })

	q := worker.NewRedisQueueWithTimeout(client, 1*time.Second)
	return q, client
}

func TestIntegration_EnqueueDequeue(t *testing.T) {
	q, _ := setupRedisQueue(t)
	ctx := context.Background()
	stream := "test:enqueue-dequeue"
	group := "grp"
	consumer := "c1"

	if err := q.EnsureGroup(ctx, stream, group); err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}

	data := map[string]interface{}{
		"job_type": "parse_demo",
		"demo_id":  "demo-123",
	}
	id, err := q.Enqueue(ctx, stream, data)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty message ID")
	}

	got, gotID, err := q.Dequeue(ctx, stream, group, consumer)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	if gotID != id {
		t.Errorf("Dequeue ID: got %q, want %q", gotID, id)
	}
	if got["job_type"] != "parse_demo" {
		t.Errorf("job_type: got %q, want %q", got["job_type"], "parse_demo")
	}
	if got["demo_id"] != "demo-123" {
		t.Errorf("demo_id: got %q, want %q", got["demo_id"], "demo-123")
	}
}

func TestIntegration_Ack(t *testing.T) {
	q, _ := setupRedisQueue(t)
	ctx := context.Background()
	stream := "test:ack"
	group := "grp"
	consumer := "c1"

	if err := q.EnsureGroup(ctx, stream, group); err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}

	_, err := q.Enqueue(ctx, stream, map[string]interface{}{"key": "val"})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	_, id, err := q.Dequeue(ctx, stream, group, consumer)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}

	// Before Ack, pending count should be 1.
	pending, err := q.GetPendingCount(ctx, stream, group)
	if err != nil {
		t.Fatalf("GetPendingCount: %v", err)
	}
	if pending != 1 {
		t.Errorf("pending before Ack: got %d, want 1", pending)
	}

	if err := q.Ack(ctx, stream, group, id); err != nil {
		t.Fatalf("Ack: %v", err)
	}

	// After Ack, pending count should be 0.
	pending, err = q.GetPendingCount(ctx, stream, group)
	if err != nil {
		t.Fatalf("GetPendingCount after Ack: %v", err)
	}
	if pending != 0 {
		t.Errorf("pending after Ack: got %d, want 0", pending)
	}
}

func TestIntegration_EnsureGroup_Idempotent(t *testing.T) {
	q, _ := setupRedisQueue(t)
	ctx := context.Background()
	stream := "test:idempotent"
	group := "grp"

	if err := q.EnsureGroup(ctx, stream, group); err != nil {
		t.Fatalf("EnsureGroup first call: %v", err)
	}

	// Second call should not error (BUSYGROUP is silently ignored).
	if err := q.EnsureGroup(ctx, stream, group); err != nil {
		t.Fatalf("EnsureGroup second call: %v", err)
	}
}

func TestIntegration_RetryFlow(t *testing.T) {
	q, _ := setupRedisQueue(t)
	ctx := context.Background()
	stream := "test:retry"
	group := "grp"
	consumer := "c1"

	if err := q.EnsureGroup(ctx, stream, group); err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}

	_, err := q.Enqueue(ctx, stream, map[string]interface{}{"task": "retry-me"})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Dequeue without Ack — message stays pending.
	data, id, err := q.Dequeue(ctx, stream, group, consumer)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	if data == nil {
		t.Fatal("expected message, got nil")
	}

	// Verify it's pending.
	pending, err := q.GetPendingCount(ctx, stream, group)
	if err != nil {
		t.Fatalf("GetPendingCount: %v", err)
	}
	if pending != 1 {
		t.Errorf("pending: got %d, want 1", pending)
	}

	// ClaimStale with a very short threshold to reclaim immediately.
	claimed, err := q.ClaimStale(ctx, stream, group, consumer, 0)
	if err != nil {
		t.Fatalf("ClaimStale: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("claimed count: got %d, want 1", len(claimed))
	}
	if claimed[0].ID != id {
		t.Errorf("claimed ID: got %q, want %q", claimed[0].ID, id)
	}

	// Now ack it.
	if err := q.Ack(ctx, stream, group, id); err != nil {
		t.Fatalf("Ack: %v", err)
	}

	pending, err = q.GetPendingCount(ctx, stream, group)
	if err != nil {
		t.Fatalf("GetPendingCount after Ack: %v", err)
	}
	if pending != 0 {
		t.Errorf("pending after Ack: got %d, want 0", pending)
	}
}

func TestIntegration_DeadLetter(t *testing.T) {
	q, client := setupRedisQueue(t)
	ctx := context.Background()
	stream := "test:deadletter"
	group := "grp"
	consumer := "c1"
	dlStream := stream + ":dead"

	if err := q.EnsureGroup(ctx, stream, group); err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}

	_, err := q.Enqueue(ctx, stream, map[string]interface{}{"task": "fail-me"})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	data, id, err := q.Dequeue(ctx, stream, group, consumer)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}

	// Dead-letter the message.
	if err := q.DeadLetter(ctx, stream, group, id, data); err != nil {
		t.Fatalf("DeadLetter: %v", err)
	}

	// Original should no longer be pending.
	pending, err := q.GetPendingCount(ctx, stream, group)
	if err != nil {
		t.Fatalf("GetPendingCount: %v", err)
	}
	if pending != 0 {
		t.Errorf("pending after dead-letter: got %d, want 0", pending)
	}

	// Dead-letter stream should have the message.
	msgs, err := client.XRange(ctx, dlStream, "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange on dead-letter: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("dead-letter messages: got %d, want 1", len(msgs))
	}
	if msgs[0].Values["task"] != "fail-me" {
		t.Errorf("dead-letter task: got %q, want %q", msgs[0].Values["task"], "fail-me")
	}
	if msgs[0].Values["_original_id"] != id {
		t.Errorf("dead-letter _original_id: got %q, want %q", msgs[0].Values["_original_id"], id)
	}
}

func TestIntegration_Worker_GracefulShutdown(t *testing.T) {
	q, _ := setupRedisQueue(t)
	ctx := context.Background()
	stream := "test:graceful"
	group := "grp"
	consumer := "c1"

	var (
		processed atomic.Int32
		mu        sync.Mutex
		started   = make(chan struct{})
	)

	handler := func(ctx context.Context, data map[string]interface{}) error {
		mu.Lock()
		close(started)
		mu.Unlock()
		// Simulate a slow job.
		time.Sleep(200 * time.Millisecond)
		processed.Add(1)
		return nil
	}

	w := worker.NewWorker(q, stream, group, consumer, handler)
	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Enqueue a message for the worker to process.
	if _, err := q.Enqueue(ctx, stream, map[string]interface{}{"task": "slow"}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Wait for the handler to begin executing.
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for handler to start")
	}

	// Stop while the job is in-flight — should wait for it to finish.
	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for graceful shutdown")
	}

	if got := processed.Load(); got != 1 {
		t.Errorf("processed: got %d, want 1", got)
	}
}

func TestIntegration_Worker_DeadLetterAfterMaxRetry(t *testing.T) {
	q, client := setupRedisQueue(t)
	ctx := context.Background()
	stream := "test:worker-dl"
	group := "grp"
	consumer := "c1"
	dlStream := stream + ":dead"

	var attempts atomic.Int32

	handler := func(ctx context.Context, data map[string]interface{}) error {
		attempts.Add(1)
		return errors.New("always fails")
	}

	w := worker.NewWorker(q, stream, group, consumer, handler).
		WithMaxRetry(3).
		WithStaleThreshold(50 * time.Millisecond).
		WithClaimInterval(100 * time.Millisecond)
	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer w.Stop()

	// Enqueue a message that will always fail.
	if _, err := q.Enqueue(ctx, stream, map[string]interface{}{"task": "doomed"}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Wait for the message to be dead-lettered after 3 attempts.
	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out: attempts=%d, expected message in dead-letter stream", attempts.Load())
		default:
		}

		msgs, err := client.XRange(ctx, dlStream, "-", "+").Result()
		if err != nil {
			// Stream may not exist yet.
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if len(msgs) >= 1 {
			if msgs[0].Values["task"] != "doomed" {
				t.Errorf("dead-letter task: got %q, want %q", msgs[0].Values["task"], "doomed")
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestIntegration_MultipleMessages(t *testing.T) {
	q, _ := setupRedisQueue(t)
	ctx := context.Background()
	stream := "test:multi"
	group := "grp"
	consumer := "c1"

	if err := q.EnsureGroup(ctx, stream, group); err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}

	// Enqueue 5 messages.
	for i := 0; i < 5; i++ {
		_, err := q.Enqueue(ctx, stream, map[string]interface{}{
			"index": fmt.Sprintf("%d", i),
		})
		if err != nil {
			t.Fatalf("Enqueue[%d]: %v", i, err)
		}
	}

	// Consume all 5 in order.
	for i := 0; i < 5; i++ {
		data, id, err := q.Dequeue(ctx, stream, group, consumer)
		if err != nil {
			t.Fatalf("Dequeue[%d]: %v", i, err)
		}
		if data == nil {
			t.Fatalf("Dequeue[%d]: expected message, got nil", i)
		}
		if id == "" {
			t.Fatalf("Dequeue[%d]: expected non-empty ID", i)
		}

		want := fmt.Sprintf("%d", i)
		if data["index"] != want {
			t.Errorf("Dequeue[%d] index: got %q, want %q", i, data["index"], want)
		}

		if err := q.Ack(ctx, stream, group, id); err != nil {
			t.Fatalf("Ack[%d]: %v", i, err)
		}
	}

	// No more messages.
	data, _, err := q.Dequeue(ctx, stream, group, consumer)
	if err != nil {
		t.Fatalf("Dequeue extra: %v", err)
	}
	if data != nil {
		t.Error("expected no message after consuming all 5")
	}
}

func TestIntegration_DequeueTimeout_ReturnsNil(t *testing.T) {
	q, _ := setupRedisQueue(t)
	ctx := context.Background()
	stream := "test:timeout"
	group := "grp"
	consumer := "c1"

	if err := q.EnsureGroup(ctx, stream, group); err != nil {
		t.Fatalf("EnsureGroup: %v", err)
	}

	// No messages enqueued — should return nil after block timeout.
	data, id, err := q.Dequeue(ctx, stream, group, consumer)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data, got %v", data)
	}
	if id != "" {
		t.Errorf("expected empty ID, got %q", id)
	}
}
