package config_test

import (
	"testing"

	"github.com/ok2ju/oversite/backend/internal/config"
)

func TestLoadWithAllRequiredVars(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/oversite")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	t.Setenv("MINIO_SECRET_KEY", "minioadmin")

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
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/oversite")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	t.Setenv("MINIO_SECRET_KEY", "minioadmin")

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
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/oversite")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	t.Setenv("MINIO_SECRET_KEY", "minioadmin")
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

func TestLoadMissingDatabaseURL(t *testing.T) {
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	t.Setenv("MINIO_SECRET_KEY", "minioadmin")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL, got nil")
	}
}

func TestLoadMissingRedisURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/oversite")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	t.Setenv("MINIO_SECRET_KEY", "minioadmin")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing REDIS_URL, got nil")
	}
}

func TestLoadMissingMinioEndpoint(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/oversite")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	t.Setenv("MINIO_SECRET_KEY", "minioadmin")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing MINIO_ENDPOINT, got nil")
	}
}

func TestLoadMissingMinioAccessKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/oversite")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_SECRET_KEY", "minioadmin")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing MINIO_ACCESS_KEY, got nil")
	}
}

func TestLoadMissingMinioSecretKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/oversite")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ACCESS_KEY", "minioadmin")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing MINIO_SECRET_KEY, got nil")
	}
}

func TestLoadMissingAllRequiredVars(t *testing.T) {
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing all required vars, got nil")
	}
}
