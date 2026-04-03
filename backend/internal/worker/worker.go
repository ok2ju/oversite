package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultMaxRetry is the default number of delivery attempts before dead-lettering.
const DefaultMaxRetry = 3

// DefaultClaimInterval is the default interval for checking and reclaiming stale messages.
const DefaultClaimInterval = 10 * time.Second

// Message represents a stream message with an ID and key-value data.
type Message struct {
	ID     string
	Values map[string]interface{}
}

// JobHandler processes a single job message. Return an error to signal failure.
type JobHandler func(ctx context.Context, data map[string]interface{}) error

// Queue defines the interface for job queue operations needed by Worker.
type Queue interface {
	EnsureGroup(ctx context.Context, stream, group string) error
	Dequeue(ctx context.Context, stream, group, consumer string) (map[string]interface{}, string, error)
	Ack(ctx context.Context, stream, group, id string) error
	DeadLetter(ctx context.Context, stream, group, id string, data map[string]interface{}) error
	ClaimStale(ctx context.Context, stream, group, consumer string, threshold time.Duration) ([]Message, error)
	GetDeliveryCount(ctx context.Context, stream, group, id string) (int64, error)
}

// Worker consumes messages from a Redis Stream, processes them with a handler,
// and manages retries and dead-lettering.
type Worker struct {
	queue          Queue
	stream         string
	group          string
	consumer       string
	handler        JobHandler
	maxRetry       int
	staleThreshold time.Duration
	claimInterval  time.Duration
	stop           chan struct{}
	done           chan struct{}
	once           sync.Once
	started        atomic.Bool
}

// NewWorker creates a worker that consumes from the given stream/group.
func NewWorker(queue Queue, stream, group, consumer string, handler JobHandler) *Worker {
	return &Worker{
		queue:          queue,
		stream:         stream,
		group:          group,
		consumer:       consumer,
		handler:        handler,
		maxRetry:       DefaultMaxRetry,
		staleThreshold: DefaultStaleThreshold,
		claimInterval:  DefaultClaimInterval,
		stop:           make(chan struct{}),
		done:           make(chan struct{}),
	}
}

// WithMaxRetry sets the maximum number of delivery attempts before dead-lettering.
func (w *Worker) WithMaxRetry(n int) *Worker {
	w.maxRetry = n
	return w
}

// WithStaleThreshold sets the idle time threshold for claiming stale messages.
func (w *Worker) WithStaleThreshold(d time.Duration) *Worker {
	w.staleThreshold = d
	return w
}

// WithClaimInterval sets how often the worker checks for stale messages.
func (w *Worker) WithClaimInterval(d time.Duration) *Worker {
	w.claimInterval = d
	return w
}

// Start launches the consume loop in a background goroutine. It ensures the
// consumer group exists before consuming. Start must be called at most once.
func (w *Worker) Start(ctx context.Context) error {
	if w.started.Swap(true) {
		return fmt.Errorf("worker already started")
	}

	if err := w.queue.EnsureGroup(ctx, w.stream, w.group); err != nil {
		w.started.Store(false)
		return fmt.Errorf("ensuring consumer group: %w", err)
	}

	go w.run(ctx)
	slog.Info("worker started",
		"stream", w.stream,
		"group", w.group,
		"consumer", w.consumer,
	)
	return nil
}

// Stop signals the worker to shut down and waits for the in-flight job to complete.
func (w *Worker) Stop() {
	w.once.Do(func() {
		close(w.stop)
	})
	if w.started.Load() {
		<-w.done
	}
	slog.Info("worker stopped",
		"stream", w.stream,
		"group", w.group,
		"consumer", w.consumer,
	)
}

// run is the main consume loop. It alternates between dequeuing new messages
// and periodically reclaiming stale ones.
func (w *Worker) run(ctx context.Context) {
	defer close(w.done)

	claimTicker := time.NewTicker(w.claimInterval)
	defer claimTicker.Stop()

	for {
		select {
		case <-w.stop:
			return
		case <-claimTicker.C:
			w.reclaimStale(ctx)
		default:
		}

		// Check for stop signal before blocking on Dequeue.
		select {
		case <-w.stop:
			return
		default:
		}

		data, id, err := w.queue.Dequeue(ctx, w.stream, w.group, w.consumer)
		if err != nil {
			slog.Error("dequeue error",
				"stream", w.stream,
				"error", err,
			)
			// Brief pause before retrying to avoid tight error loops.
			select {
			case <-w.stop:
				return
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		if data == nil {
			// Timeout with no message; loop back.
			continue
		}

		w.processMessage(ctx, data, id)
	}
}

// processMessage invokes the handler and manages ack/retry/dead-letter.
func (w *Worker) processMessage(ctx context.Context, data map[string]interface{}, id string) {
	deliveryCount, err := w.queue.GetDeliveryCount(ctx, w.stream, w.group, id)
	if err != nil {
		slog.Error("get delivery count failed",
			"stream", w.stream,
			"id", id,
			"error", err,
		)
		deliveryCount = 1
	}

	slog.Debug("processing message",
		"stream", w.stream,
		"id", id,
		"delivery_count", deliveryCount,
	)

	if err := w.handler(ctx, data); err != nil {
		slog.Error("handler failed",
			"stream", w.stream,
			"id", id,
			"delivery_count", deliveryCount,
			"error", err,
		)

		if deliveryCount >= int64(w.maxRetry) {
			if dlErr := w.queue.DeadLetter(ctx, w.stream, w.group, id, data); dlErr != nil {
				slog.Error("dead-letter failed",
					"stream", w.stream,
					"id", id,
					"error", dlErr,
				)
			}
		}
		// If under max retries, don't ack — the message stays pending
		// and will be reclaimed after the stale threshold.
		return
	}

	if err := w.queue.Ack(ctx, w.stream, w.group, id); err != nil {
		slog.Error("ack failed",
			"stream", w.stream,
			"id", id,
			"error", err,
		)
	}
}

// reclaimStale finds and reprocesses messages that have been pending
// longer than the stale threshold.
func (w *Worker) reclaimStale(ctx context.Context) {
	msgs, err := w.queue.ClaimStale(ctx, w.stream, w.group, w.consumer, w.staleThreshold)
	if err != nil {
		slog.Error("claim stale failed",
			"stream", w.stream,
			"error", err,
		)
		return
	}

	for _, msg := range msgs {
		select {
		case <-w.stop:
			return
		default:
		}
		w.processMessage(ctx, msg.Values, msg.ID)
	}
}
