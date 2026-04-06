package config

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration values for the application.
type Config struct {
	Port                 string
	WSPort               string
	DatabaseURL          string
	RedisURL             string
	MinioEndpoint        string
	MinioAccessKey       string
	MinioSecretKey       string
	MinioUseSSL          bool
	MinioBucket          string
	FaceitClientID       string
	FaceitClientSecret   string
	FaceitRedirectURI    string
	FaceitAPIKey         string
	FaceitAPIBaseURL     string
	Environment          string
	LogLevel             string
	WorkerMaxRetry       int
	WorkerBlockTimeout   time.Duration
	WorkerStaleThreshold time.Duration
	WorkerClaimInterval  time.Duration
	IngestBatchSize      int
}

// Load reads configuration from environment variables.
// Required variables: DATABASE_URL, REDIS_URL, MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY,
// FACEIT_CLIENT_ID, FACEIT_CLIENT_SECRET, FACEIT_REDIRECT_URI, FACEIT_API_KEY.
// Returns an error if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{
		Port:                 getEnvOrDefault("PORT", "8080"),
		WSPort:               getEnvOrDefault("WS_PORT", "8081"),
		MinioBucket:          getEnvOrDefault("MINIO_BUCKET", "oversite-demos"),
		MinioUseSSL:          getEnvOrDefault("MINIO_USE_SSL", "false") == "true",
		Environment:          getEnvOrDefault("GO_ENV", "development"),
		LogLevel:             getEnvOrDefault("LOG_LEVEL", "info"),
		WorkerMaxRetry:       getEnvOrDefaultInt("WORKER_MAX_RETRY", 3),
		WorkerBlockTimeout:   getEnvOrDefaultDuration("WORKER_BLOCK_TIMEOUT", 2*time.Second),
		WorkerStaleThreshold: getEnvOrDefaultDuration("WORKER_STALE_THRESHOLD", 30*time.Second),
		WorkerClaimInterval:  getEnvOrDefaultDuration("WORKER_CLAIM_INTERVAL", 10*time.Second),
		FaceitAPIBaseURL:     getEnvOrDefault("FACEIT_API_BASE_URL", "https://open.faceit.com/data/v4"),
		IngestBatchSize:      getEnvOrDefaultInt("INGEST_BATCH_SIZE", 10000),
	}

	// Required vars
	required := map[string]*string{
		"DATABASE_URL":         &cfg.DatabaseURL,
		"REDIS_URL":            &cfg.RedisURL,
		"MINIO_ENDPOINT":       &cfg.MinioEndpoint,
		"MINIO_ACCESS_KEY":     &cfg.MinioAccessKey,
		"MINIO_SECRET_KEY":     &cfg.MinioSecretKey,
		"FACEIT_CLIENT_ID":     &cfg.FaceitClientID,
		"FACEIT_CLIENT_SECRET": &cfg.FaceitClientSecret,
		"FACEIT_REDIRECT_URI":  &cfg.FaceitRedirectURI,
		"FACEIT_API_KEY":       &cfg.FaceitAPIKey,
	}

	var missing []string
	for key, ptr := range required {
		val := os.Getenv(key)
		if val == "" {
			missing = append(missing, key)
		}
		*ptr = val
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

// WSConfig holds configuration for the WebSocket server.
// Requires DATABASE_URL and REDIS_URL (for Yjs state persistence).
type WSConfig struct {
	WSPort              string
	DatabaseURL         string
	RedisURL            string
	Environment         string
	LogLevel            string
	YjsAutoSaveInterval time.Duration
}

// LoadWS reads configuration for the WebSocket server from environment variables.
// Required variables: DATABASE_URL, REDIS_URL.
func LoadWS() (*WSConfig, error) {
	cfg := &WSConfig{
		WSPort:              getEnvOrDefault("WS_PORT", "8081"),
		Environment:         getEnvOrDefault("GO_ENV", "development"),
		LogLevel:            getEnvOrDefault("LOG_LEVEL", "info"),
		YjsAutoSaveInterval: getEnvOrDefaultDuration("YJS_AUTO_SAVE_INTERVAL", 30*time.Second),
	}

	required := map[string]*string{
		"DATABASE_URL": &cfg.DatabaseURL,
		"REDIS_URL":    &cfg.RedisURL,
	}

	var missing []string
	for key, ptr := range required {
		val := os.Getenv(key)
		if val == "" {
			missing = append(missing, key)
		}
		*ptr = val
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

// WorkerConfig holds configuration for the background worker process.
type WorkerConfig struct {
	DatabaseURL          string
	RedisURL             string
	FaceitAPIKey         string
	FaceitAPIBaseURL     string
	Environment          string
	LogLevel             string
	WorkerMaxRetry       int
	WorkerBlockTimeout   time.Duration
	WorkerStaleThreshold time.Duration
	WorkerClaimInterval  time.Duration
}

// LoadWorker reads configuration for the worker process from environment variables.
// Required variables: DATABASE_URL, REDIS_URL, FACEIT_API_KEY.
func LoadWorker() (*WorkerConfig, error) {
	cfg := &WorkerConfig{
		FaceitAPIBaseURL:     getEnvOrDefault("FACEIT_API_BASE_URL", "https://open.faceit.com/data/v4"),
		Environment:          getEnvOrDefault("GO_ENV", "development"),
		LogLevel:             getEnvOrDefault("LOG_LEVEL", "info"),
		WorkerMaxRetry:       getEnvOrDefaultInt("WORKER_MAX_RETRY", 3),
		WorkerBlockTimeout:   getEnvOrDefaultDuration("WORKER_BLOCK_TIMEOUT", 2*time.Second),
		WorkerStaleThreshold: getEnvOrDefaultDuration("WORKER_STALE_THRESHOLD", 30*time.Second),
		WorkerClaimInterval:  getEnvOrDefaultDuration("WORKER_CLAIM_INTERVAL", 10*time.Second),
	}

	required := map[string]*string{
		"DATABASE_URL":  &cfg.DatabaseURL,
		"REDIS_URL":     &cfg.RedisURL,
		"FACEIT_API_KEY": &cfg.FaceitAPIKey,
	}

	var missing []string
	for key, ptr := range required {
		val := os.Getenv(key)
		if val == "" {
			missing = append(missing, key)
		}
		*ptr = val
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvOrDefaultInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return defaultVal
}

func getEnvOrDefaultDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
