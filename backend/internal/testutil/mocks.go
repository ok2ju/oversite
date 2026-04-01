package testutil

import (
	"context"
	"io"
	"time"
)

// S3Client defines the interface for object storage operations.
type S3Client interface {
	PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64) error
	GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	DeleteObject(ctx context.Context, bucket, key string) error
	ObjectExists(ctx context.Context, bucket, key string) (bool, error)
}

// SessionStore defines the interface for session management (Redis-backed).
type SessionStore interface {
	Create(ctx context.Context, token string, data []byte, ttl time.Duration) error
	Get(ctx context.Context, token string) ([]byte, error)
	Delete(ctx context.Context, token string) error
}

// JobQueue defines the interface for the background job queue (Redis Streams).
type JobQueue interface {
	Enqueue(ctx context.Context, stream string, data map[string]interface{}) (string, error)
	Dequeue(ctx context.Context, stream, group, consumer string) (map[string]interface{}, string, error)
	Ack(ctx context.Context, stream, group, id string) error
}

// FaceitAPI defines the interface for the Faceit Data API client.
type FaceitAPI interface {
	GetPlayer(ctx context.Context, playerID string) (interface{}, error)
	GetPlayerHistory(ctx context.Context, playerID string, offset, limit int) (interface{}, error)
	GetMatchDetails(ctx context.Context, matchID string) (interface{}, error)
}

// --- Stub implementations for testing ---

// StubS3Client is a no-op S3 client for unit tests.
type StubS3Client struct{}

func (s *StubS3Client) PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64) error {
	return nil
}

func (s *StubS3Client) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	return io.NopCloser(io.LimitReader(nil, 0)), nil
}

func (s *StubS3Client) DeleteObject(ctx context.Context, bucket, key string) error {
	return nil
}

func (s *StubS3Client) ObjectExists(ctx context.Context, bucket, key string) (bool, error) {
	return false, nil
}

// StubSessionStore is a no-op session store for unit tests.
type StubSessionStore struct{}

func (s *StubSessionStore) Create(ctx context.Context, token string, data []byte, ttl time.Duration) error {
	return nil
}

func (s *StubSessionStore) Get(ctx context.Context, token string) ([]byte, error) {
	return nil, nil
}

func (s *StubSessionStore) Delete(ctx context.Context, token string) error {
	return nil
}

// StubJobQueue is a no-op job queue for unit tests.
type StubJobQueue struct{}

func (s *StubJobQueue) Enqueue(ctx context.Context, stream string, data map[string]interface{}) (string, error) {
	return "stub-id", nil
}

func (s *StubJobQueue) Dequeue(ctx context.Context, stream, group, consumer string) (map[string]interface{}, string, error) {
	return nil, "", nil
}

func (s *StubJobQueue) Ack(ctx context.Context, stream, group, id string) error {
	return nil
}

// StubFaceitAPI is a no-op Faceit API client for unit tests.
type StubFaceitAPI struct{}

func (s *StubFaceitAPI) GetPlayer(ctx context.Context, playerID string) (interface{}, error) {
	return nil, nil
}

func (s *StubFaceitAPI) GetPlayerHistory(ctx context.Context, playerID string, offset, limit int) (interface{}, error) {
	return nil, nil
}

func (s *StubFaceitAPI) GetMatchDetails(ctx context.Context, matchID string) (interface{}, error) {
	return nil, nil
}
