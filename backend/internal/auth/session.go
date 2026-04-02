package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrSessionNotFound is returned when a session does not exist or has expired.
var ErrSessionNotFound = errors.New("session not found")

// DefaultSessionTTL is the default time-to-live for sessions (7 days).
const DefaultSessionTTL = 7 * 24 * time.Hour

const sessionKeyPrefix = "session:"

// SessionData holds typed session information stored in Redis.
type SessionData struct {
	UserID       string    `json:"user_id"`
	FaceitID     string    `json:"faceit_id"`
	Nickname     string    `json:"nickname"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// SessionStore manages user sessions with typed data and sliding expiration.
type SessionStore interface {
	Create(ctx context.Context, data *SessionData) (token string, err error)
	Get(ctx context.Context, token string) (*SessionData, error)
	Delete(ctx context.Context, token string) error
	Refresh(ctx context.Context, token string) error
}

// GenerateSessionToken generates a cryptographically random 64-char hex token.
func GenerateSessionToken() (string, error) {
	return GenerateSessionTokenWithReader(rand.Reader)
}

// GenerateSessionTokenWithReader generates a 64-char hex token from the given reader.
func GenerateSessionTokenWithReader(r io.Reader) (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", fmt.Errorf("generating session token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// RedisSessionStore implements SessionStore using Redis.
type RedisSessionStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisSessionStore creates a new Redis-backed session store with the default 7-day TTL.
func NewRedisSessionStore(client *redis.Client) *RedisSessionStore {
	return &RedisSessionStore{client: client, ttl: DefaultSessionTTL}
}

// NewRedisSessionStoreWithTTL creates a new Redis-backed session store with a custom TTL.
func NewRedisSessionStoreWithTTL(client *redis.Client, ttl time.Duration) *RedisSessionStore {
	return &RedisSessionStore{client: client, ttl: ttl}
}

func (s *RedisSessionStore) Create(ctx context.Context, data *SessionData) (string, error) {
	token, err := GenerateSessionToken()
	if err != nil {
		return "", err
	}

	now := time.Now()
	data.CreatedAt = now
	data.ExpiresAt = now.Add(s.ttl)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshaling session data: %w", err)
	}

	key := sessionKeyPrefix + token
	if err := s.client.Set(ctx, key, jsonData, s.ttl).Err(); err != nil {
		return "", fmt.Errorf("storing session: %w", err)
	}

	return token, nil
}

func (s *RedisSessionStore) Get(ctx context.Context, token string) (*SessionData, error) {
	key := sessionKeyPrefix + token
	val, err := s.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting session: %w", err)
	}

	var data SessionData
	if err := json.Unmarshal(val, &data); err != nil {
		return nil, fmt.Errorf("unmarshaling session data: %w", err)
	}
	return &data, nil
}

func (s *RedisSessionStore) Delete(ctx context.Context, token string) error {
	key := sessionKeyPrefix + token
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

func (s *RedisSessionStore) Refresh(ctx context.Context, token string) error {
	key := sessionKeyPrefix + token
	result, err := s.client.Expire(ctx, key, s.ttl).Result()
	if err != nil {
		return fmt.Errorf("refreshing session: %w", err)
	}
	if !result {
		return ErrSessionNotFound
	}
	return nil
}
