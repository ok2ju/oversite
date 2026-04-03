package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// DefaultBlockTimeout is the default duration XREADGROUP will block waiting for new messages.
const DefaultBlockTimeout = 2 * time.Second

// DefaultStaleThreshold is the default idle time after which a pending message is considered stale.
const DefaultStaleThreshold = 30 * time.Second

// RedisQueue implements a job queue backed by Redis Streams.
type RedisQueue struct {
	client       *redis.Client
	blockTimeout time.Duration
}

// NewRedisQueue creates a new RedisQueue with default settings.
func NewRedisQueue(client *redis.Client) *RedisQueue {
	return &RedisQueue{
		client:       client,
		blockTimeout: DefaultBlockTimeout,
	}
}

// NewRedisQueueWithTimeout creates a new RedisQueue with a custom block timeout.
func NewRedisQueueWithTimeout(client *redis.Client, blockTimeout time.Duration) *RedisQueue {
	return &RedisQueue{
		client:       client,
		blockTimeout: blockTimeout,
	}
}

// Enqueue adds a message to the given stream via XADD. Returns the message ID.
func (q *RedisQueue) Enqueue(ctx context.Context, stream string, data map[string]interface{}) (string, error) {
	id, err := q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: data,
	}).Result()
	if err != nil {
		return "", fmt.Errorf("xadd to %q: %w", stream, err)
	}
	return id, nil
}

// EnsureGroup creates a consumer group for the stream. It is idempotent:
// if the group already exists (BUSYGROUP), the error is silently ignored.
// It starts from ID "0" so that existing messages are included.
func (q *RedisQueue) EnsureGroup(ctx context.Context, stream, group string) error {
	err := q.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && !isBusyGroupError(err) {
		return fmt.Errorf("xgroup create %q/%q: %w", stream, group, err)
	}
	return nil
}

// Dequeue reads one message from the stream using XREADGROUP. It blocks for
// the configured timeout. Returns (nil, "", nil) if no message is available.
func (q *RedisQueue) Dequeue(ctx context.Context, stream, group, consumer string) (map[string]interface{}, string, error) {
	result, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    1,
		Block:    q.blockTimeout,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("xreadgroup %q/%q: %w", stream, group, err)
	}

	if len(result) == 0 || len(result[0].Messages) == 0 {
		return nil, "", nil
	}

	msg := result[0].Messages[0]
	return msg.Values, msg.ID, nil
}

// Ack acknowledges a message in the consumer group via XACK.
func (q *RedisQueue) Ack(ctx context.Context, stream, group, id string) error {
	if err := q.client.XAck(ctx, stream, group, id).Err(); err != nil {
		return fmt.Errorf("xack %q/%q/%s: %w", stream, group, id, err)
	}
	return nil
}

// GetPendingCount returns the total number of pending (unacknowledged) messages
// in the consumer group.
func (q *RedisQueue) GetPendingCount(ctx context.Context, stream, group string) (int64, error) {
	pending, err := q.client.XPending(ctx, stream, group).Result()
	if err != nil {
		return 0, fmt.Errorf("xpending %q/%q: %w", stream, group, err)
	}
	return pending.Count, nil
}

// ClaimStale uses XAUTOCLAIM to reclaim messages that have been idle longer
// than the given threshold. Returns the claimed messages (data + IDs).
func (q *RedisQueue) ClaimStale(ctx context.Context, stream, group, consumer string, threshold time.Duration) ([]redis.XMessage, error) {
	msgs, _, err := q.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   stream,
		Group:    group,
		Consumer: consumer,
		MinIdle:  threshold,
		Start:    "0-0",
		Count:    10,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("xautoclaim %q/%q: %w", stream, group, err)
	}
	return msgs, nil
}

// DeadLetter moves a failed message to a dead-letter stream ({stream}:dead)
// by adding it there and acknowledging the original.
func (q *RedisQueue) DeadLetter(ctx context.Context, stream, group, id string, data map[string]interface{}) error {
	dlStream := stream + ":dead"

	// Copy data to dead-letter stream with the original message ID.
	dlData := make(map[string]interface{}, len(data)+1)
	for k, v := range data {
		dlData[k] = v
	}
	dlData["_original_id"] = id
	dlData["_original_stream"] = stream

	if _, err := q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: dlStream,
		Values: dlData,
	}).Result(); err != nil {
		return fmt.Errorf("xadd to dead-letter %q: %w", dlStream, err)
	}

	if err := q.Ack(ctx, stream, group, id); err != nil {
		return fmt.Errorf("ack after dead-letter: %w", err)
	}

	slog.Warn("message dead-lettered",
		"stream", stream,
		"id", id,
		"dead_letter_stream", dlStream,
	)
	return nil
}

// isBusyGroupError returns true if the error is the Redis BUSYGROUP error
// indicating a consumer group already exists.
func isBusyGroupError(err error) bool {
	if err == nil {
		return false
	}
	return redis.HasErrorPrefix(err, "BUSYGROUP")
}
