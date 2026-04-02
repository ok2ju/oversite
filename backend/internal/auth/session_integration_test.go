//go:build integration

package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/testutil"
)

func setupRedisSessionStore(t *testing.T) (*auth.RedisSessionStore, *redis.Client) {
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

	store := auth.NewRedisSessionStore(client)
	return store, client
}

func sampleSessionData() *auth.SessionData {
	return &auth.SessionData{
		UserID:       "user-abc",
		FaceitID:     "faceit-xyz",
		Nickname:     "testplayer",
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
	}
}

func TestRedisSessionStore_Create_StoresSession(t *testing.T) {
	store, _ := setupRedisSessionStore(t)
	ctx := context.Background()

	token, err := store.Create(ctx, sampleSessionData())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	got, err := store.Get(ctx, token)
	if err != nil {
		t.Fatalf("Get after Create: %v", err)
	}
	if got.FaceitID != "faceit-xyz" {
		t.Errorf("FaceitID: got %q, want %q", got.FaceitID, "faceit-xyz")
	}
}

func TestRedisSessionStore_Create_SetsCorrectTTL(t *testing.T) {
	store, client := setupRedisSessionStore(t)
	ctx := context.Background()

	token, err := store.Create(ctx, sampleSessionData())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ttl := client.TTL(ctx, "session:"+token).Val()
	// TTL should be close to 7 days (allow 10 second margin)
	expected := 7 * 24 * time.Hour
	if ttl < expected-10*time.Second || ttl > expected {
		t.Errorf("TTL: got %v, want ~%v", ttl, expected)
	}
}

func TestRedisSessionStore_Get_ValidSession(t *testing.T) {
	store, _ := setupRedisSessionStore(t)
	ctx := context.Background()

	data := sampleSessionData()
	token, err := store.Create(ctx, data)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.Get(ctx, token)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.UserID != "user-abc" {
		t.Errorf("UserID: got %q, want %q", got.UserID, "user-abc")
	}
	if got.Nickname != "testplayer" {
		t.Errorf("Nickname: got %q, want %q", got.Nickname, "testplayer")
	}
	if got.AccessToken != "access-123" {
		t.Errorf("AccessToken: got %q, want %q", got.AccessToken, "access-123")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if got.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be set")
	}
}

func TestRedisSessionStore_Get_MissingSession(t *testing.T) {
	store, _ := setupRedisSessionStore(t)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent-token")
	if err != auth.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestRedisSessionStore_Get_ExpiredSession(t *testing.T) {
	_, client := setupRedisSessionStore(t)
	ctx := context.Background()

	// Use a short-TTL store
	store := auth.NewRedisSessionStoreWithTTL(client, 100*time.Millisecond)
	token, err := store.Create(ctx, sampleSessionData())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	_, err = store.Get(ctx, token)
	if err != auth.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound for expired session, got %v", err)
	}
}

func TestRedisSessionStore_Delete_RemovesSession(t *testing.T) {
	store, _ := setupRedisSessionStore(t)
	ctx := context.Background()

	token, err := store.Create(ctx, sampleSessionData())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Delete(ctx, token); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = store.Get(ctx, token)
	if err != auth.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound after Delete, got %v", err)
	}
}

func TestRedisSessionStore_Delete_NonexistentSession(t *testing.T) {
	store, _ := setupRedisSessionStore(t)
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent-token")
	if err != nil {
		t.Errorf("Delete nonexistent should not error, got %v", err)
	}
}

func TestRedisSessionStore_Refresh_ExtendsTTL(t *testing.T) {
	_, client := setupRedisSessionStore(t)
	ctx := context.Background()

	store := auth.NewRedisSessionStoreWithTTL(client, 7*24*time.Hour)
	token, err := store.Create(ctx, sampleSessionData())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Shorten TTL to 10 seconds
	client.Expire(ctx, "session:"+token, 10*time.Second)

	// Refresh should restore full TTL
	if err := store.Refresh(ctx, token); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	ttl := client.TTL(ctx, "session:"+token).Val()
	if ttl < 7*24*time.Hour-10*time.Second {
		t.Errorf("TTL after Refresh: got %v, want ~7 days", ttl)
	}
}

func TestRedisSessionStore_Refresh_NonexistentSession(t *testing.T) {
	store, _ := setupRedisSessionStore(t)
	ctx := context.Background()

	err := store.Refresh(ctx, "nonexistent-token")
	if err != auth.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}
