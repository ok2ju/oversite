package config_test

import (
	"testing"

	"github.com/ok2ju/oversite/backend/internal/config"
)

// setRequiredEnv sets all required environment variables for config.Load().
func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/oversite")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	t.Setenv("MINIO_SECRET_KEY", "minioadmin")
	t.Setenv("FACEIT_CLIENT_ID", "test-client-id")
	t.Setenv("FACEIT_CLIENT_SECRET", "test-client-secret")
	t.Setenv("FACEIT_REDIRECT_URI", "http://localhost:3000/api/v1/auth/faceit/callback")
	t.Setenv("FACEIT_API_KEY", "test-api-key")
}

func TestLoadWithAllRequiredVars(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DatabaseURL != "postgres://localhost:5432/oversite" {
		t.Errorf("expected DatabaseURL 'postgres://localhost:5432/oversite', got %q", cfg.DatabaseURL)
	}
	if cfg.RedisURL != "redis://localhost:6379" {
		t.Errorf("expected RedisURL 'redis://localhost:6379', got %q", cfg.RedisURL)
	}
	if cfg.MinioEndpoint != "localhost:9000" {
		t.Errorf("expected MinioEndpoint 'localhost:9000', got %q", cfg.MinioEndpoint)
	}
	if cfg.MinioAccessKey != "minioadmin" {
		t.Errorf("expected MinioAccessKey 'minioadmin', got %q", cfg.MinioAccessKey)
	}
	if cfg.MinioSecretKey != "minioadmin" {
		t.Errorf("expected MinioSecretKey 'minioadmin', got %q", cfg.MinioSecretKey)
	}
}

func TestLoadDefaults(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("expected default Port '8080', got %q", cfg.Port)
	}
	if cfg.WSPort != "8081" {
		t.Errorf("expected default WSPort '8081', got %q", cfg.WSPort)
	}
	if cfg.Environment != "development" {
		t.Errorf("expected default Environment 'development', got %q", cfg.Environment)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default LogLevel 'info', got %q", cfg.LogLevel)
	}
	if cfg.MinioBucket != "oversite-demos" {
		t.Errorf("expected default MinioBucket 'oversite-demos', got %q", cfg.MinioBucket)
	}
	if cfg.MinioUseSSL != false {
		t.Errorf("expected default MinioUseSSL false, got %v", cfg.MinioUseSSL)
	}
}

func TestLoadOverrideDefaults(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("PORT", "9090")
	t.Setenv("WS_PORT", "9091")
	t.Setenv("GO_ENV", "production")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("MINIO_BUCKET", "custom-bucket")
	t.Setenv("MINIO_USE_SSL", "true")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("expected Port '9090', got %q", cfg.Port)
	}
	if cfg.WSPort != "9091" {
		t.Errorf("expected WSPort '9091', got %q", cfg.WSPort)
	}
	if cfg.Environment != "production" {
		t.Errorf("expected Environment 'production', got %q", cfg.Environment)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got %q", cfg.LogLevel)
	}
	if cfg.MinioBucket != "custom-bucket" {
		t.Errorf("expected MinioBucket 'custom-bucket', got %q", cfg.MinioBucket)
	}
	if cfg.MinioUseSSL != true {
		t.Errorf("expected MinioUseSSL true, got %v", cfg.MinioUseSSL)
	}
}

func TestLoadFaceitConfig(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.FaceitClientID != "test-client-id" {
		t.Errorf("expected FaceitClientID 'test-client-id', got %q", cfg.FaceitClientID)
	}
	if cfg.FaceitClientSecret != "test-client-secret" {
		t.Errorf("expected FaceitClientSecret 'test-client-secret', got %q", cfg.FaceitClientSecret)
	}
	if cfg.FaceitRedirectURI != "http://localhost:3000/api/v1/auth/faceit/callback" {
		t.Errorf("expected FaceitRedirectURI 'http://localhost:3000/api/v1/auth/faceit/callback', got %q", cfg.FaceitRedirectURI)
	}
}

func TestLoadMissingFaceitClientID(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("FACEIT_CLIENT_ID", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing FACEIT_CLIENT_ID, got nil")
	}
}

func TestLoadMissingFaceitClientSecret(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("FACEIT_CLIENT_SECRET", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing FACEIT_CLIENT_SECRET, got nil")
	}
}

func TestLoadMissingFaceitRedirectURI(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("FACEIT_REDIRECT_URI", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing FACEIT_REDIRECT_URI, got nil")
	}
}

func TestLoadMissingDatabaseURL(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DATABASE_URL", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL, got nil")
	}
}

func TestLoadMissingRedisURL(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("REDIS_URL", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing REDIS_URL, got nil")
	}
}

func TestLoadMissingMinioEndpoint(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("MINIO_ENDPOINT", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing MINIO_ENDPOINT, got nil")
	}
}

func TestLoadMissingMinioAccessKey(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("MINIO_ACCESS_KEY", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing MINIO_ACCESS_KEY, got nil")
	}
}

func TestLoadMissingMinioSecretKey(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("MINIO_SECRET_KEY", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing MINIO_SECRET_KEY, got nil")
	}
}

func TestLoadFaceitAPIConfig(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.FaceitAPIKey != "test-api-key" {
		t.Errorf("expected FaceitAPIKey 'test-api-key', got %q", cfg.FaceitAPIKey)
	}
	if cfg.FaceitAPIBaseURL != "https://open.faceit.com/data/v4" {
		t.Errorf("expected FaceitAPIBaseURL default, got %q", cfg.FaceitAPIBaseURL)
	}
}

func TestLoadFaceitAPIBaseURLOverride(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("FACEIT_API_BASE_URL", "https://custom.faceit.com/v4")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.FaceitAPIBaseURL != "https://custom.faceit.com/v4" {
		t.Errorf("expected custom base URL, got %q", cfg.FaceitAPIBaseURL)
	}
}

func TestLoadMissingFaceitAPIKey(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("FACEIT_API_KEY", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing FACEIT_API_KEY, got nil")
	}
}

func TestLoadMissingAllRequiredVars(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("MINIO_ENDPOINT", "")
	t.Setenv("MINIO_ACCESS_KEY", "")
	t.Setenv("MINIO_SECRET_KEY", "")
	t.Setenv("FACEIT_CLIENT_ID", "")
	t.Setenv("FACEIT_CLIENT_SECRET", "")
	t.Setenv("FACEIT_REDIRECT_URI", "")
	t.Setenv("FACEIT_API_KEY", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing all required vars, got nil")
	}
}
