package auth

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStateStore implements StateStore using Redis for OAuth state and session storage.
type RedisStateStore struct {
	client *redis.Client
}

// NewRedisStateStore creates a new Redis-backed state store.
func NewRedisStateStore(client *redis.Client) *RedisStateStore {
	return &RedisStateStore{client: client}
}

func (s *RedisStateStore) Create(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	return s.client.Set(ctx, key, data, ttl).Err()
}

func (s *RedisStateStore) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

func (s *RedisStateStore) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

// Ping checks Redis connectivity.
func (s *RedisStateStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}
