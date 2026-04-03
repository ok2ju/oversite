package storage

import (
	"testing"

	"github.com/ok2ju/oversite/backend/internal/testutil"
)

// Compile-time check: MinIOClient must satisfy testutil.S3Client.
var _ testutil.S3Client = (*MinIOClient)(nil)

func TestNewMinIOClient_ValidParams(t *testing.T) {
	// NewMinIOClient with valid params should not error.
	// Note: this does not actually connect — the MinIO SDK is lazy.
	client, err := NewMinIOClient("localhost:9000", "access", "secret", false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.client == nil {
		t.Fatal("expected non-nil underlying minio.Client")
	}
}

func TestNewMinIOClient_MissingEndpoint(t *testing.T) {
	_, err := NewMinIOClient("", "access", "secret", false)
	if err == nil {
		t.Fatal("expected error for empty endpoint")
	}
}

func TestNewMinIOClient_MissingAccessKey(t *testing.T) {
	_, err := NewMinIOClient("localhost:9000", "", "secret", false)
	if err == nil {
		t.Fatal("expected error for empty access key")
	}
}

func TestNewMinIOClient_MissingSecretKey(t *testing.T) {
	_, err := NewMinIOClient("localhost:9000", "access", "", false)
	if err == nil {
		t.Fatal("expected error for empty secret key")
	}
}

func TestStubS3Client_SatisfiesInterface(t *testing.T) {
	// Verify the stub satisfies the interface (compile-time check).
	var _ testutil.S3Client = &testutil.StubS3Client{}
}
