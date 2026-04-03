//go:build integration

package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/ok2ju/oversite/backend/internal/testutil"
)

const (
	testAccessKey = "minioadmin"
	testSecretKey = "minioadmin"
	testBucket    = "integration-test-bucket"
)

// setupMinIO starts a MinIO container and returns a connected MinIOClient.
// The returned cleanup function terminates the container.
func setupMinIO(t *testing.T) *MinIOClient {
	t.Helper()

	ctx := context.Background()

	container, endpoint, err := testutil.MinIOContainer(ctx)
	if err != nil {
		t.Fatalf("starting minio container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("terminating minio container: %v", err)
		}
	})

	client, err := NewMinIOClient(endpoint, testAccessKey, testSecretKey, false)
	if err != nil {
		t.Fatalf("creating minio client: %v", err)
	}

	// Create the test bucket.
	if err := client.EnsureBucket(ctx, testBucket); err != nil {
		t.Fatalf("creating test bucket: %v", err)
	}

	return client
}

func TestIntegration_EnsureBucket_Idempotent(t *testing.T) {
	client := setupMinIO(t)
	ctx := context.Background()

	bucket := "idempotent-test-bucket"

	// First call creates the bucket.
	if err := client.EnsureBucket(ctx, bucket); err != nil {
		t.Fatalf("first EnsureBucket: %v", err)
	}

	// Second call should be a no-op, not an error.
	if err := client.EnsureBucket(ctx, bucket); err != nil {
		t.Fatalf("second EnsureBucket (idempotent): %v", err)
	}
}

func TestIntegration_PutGetObject(t *testing.T) {
	client := setupMinIO(t)
	ctx := context.Background()

	key := "test-objects/hello.txt"
	content := []byte("Hello, MinIO!")

	// Put the object.
	if err := client.PutObject(ctx, testBucket, key, bytes.NewReader(content), int64(len(content))); err != nil {
		t.Fatalf("PutObject: %v", err)
	}

	// Get the object.
	rc, err := client.GetObject(ctx, testBucket, key)
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading object body: %v", err)
	}

	if !bytes.Equal(got, content) {
		t.Fatalf("content mismatch: got %q, want %q", got, content)
	}
}

func TestIntegration_DeleteObject_ThenNotExists(t *testing.T) {
	client := setupMinIO(t)
	ctx := context.Background()

	key := "test-objects/to-delete.txt"
	content := []byte("delete me")

	// Upload first.
	if err := client.PutObject(ctx, testBucket, key, bytes.NewReader(content), int64(len(content))); err != nil {
		t.Fatalf("PutObject: %v", err)
	}

	// Verify it exists.
	exists, err := client.ObjectExists(ctx, testBucket, key)
	if err != nil {
		t.Fatalf("ObjectExists before delete: %v", err)
	}
	if !exists {
		t.Fatal("expected object to exist before deletion")
	}

	// Delete it.
	if err := client.DeleteObject(ctx, testBucket, key); err != nil {
		t.Fatalf("DeleteObject: %v", err)
	}

	// Verify it no longer exists.
	exists, err = client.ObjectExists(ctx, testBucket, key)
	if err != nil {
		t.Fatalf("ObjectExists after delete: %v", err)
	}
	if exists {
		t.Fatal("expected object to not exist after deletion")
	}
}

func TestIntegration_PresignedGetURL(t *testing.T) {
	client := setupMinIO(t)
	ctx := context.Background()

	key := "test-objects/presigned.txt"
	content := []byte("presigned content")

	// Upload.
	if err := client.PutObject(ctx, testBucket, key, bytes.NewReader(content), int64(len(content))); err != nil {
		t.Fatalf("PutObject: %v", err)
	}

	// Generate presigned URL.
	presignedURL, err := client.PresignedGetURL(ctx, testBucket, key, 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignedGetURL: %v", err)
	}

	if presignedURL == "" {
		t.Fatal("expected non-empty presigned URL")
	}

	// HTTP GET the presigned URL to verify it works.
	resp, err := http.Get(presignedURL) //nolint:gosec // test URL
	if err != nil {
		t.Fatalf("HTTP GET presigned URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading presigned response body: %v", err)
	}

	if !bytes.Equal(got, content) {
		t.Fatalf("presigned content mismatch: got %q, want %q", got, content)
	}
}

func TestIntegration_ObjectExists_NonExistent(t *testing.T) {
	client := setupMinIO(t)
	ctx := context.Background()

	exists, err := client.ObjectExists(ctx, testBucket, "does-not-exist/nope.txt")
	if err != nil {
		t.Fatalf("ObjectExists for non-existent key: %v", err)
	}
	if exists {
		t.Fatal("expected ObjectExists to return false for non-existent key")
	}
}

func TestIntegration_GetObject_NonExistent(t *testing.T) {
	client := setupMinIO(t)
	ctx := context.Background()

	_, err := client.GetObject(ctx, testBucket, "does-not-exist/nope.txt")
	if err == nil {
		t.Fatal("expected error for GetObject on non-existent key")
	}
}
