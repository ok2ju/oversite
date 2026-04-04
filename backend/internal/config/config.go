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
	Environment          string
	LogLevel             string
	WorkerMaxRetry       int
	WorkerBlockTimeout   time.Duration
	WorkerStaleThreshold time.Duration
	WorkerClaimInterval  time.Duration
}

// Load reads configuration from environment variables.
// Required variables: DATABASE_URL, REDIS_URL, MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY.
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
// Only requires REDIS_URL (the WS server is a stateless Yjs relay with no DB access).
type WSConfig struct {
	WSPort      string
	RedisURL    string
	Environment string
	LogLevel    string
}

// LoadWS reads configuration for the WebSocket server from environment variables.
// Required variables: REDIS_URL.
func LoadWS() (*WSConfig, error) {
	cfg := &WSConfig{
		WSPort:      getEnvOrDefault("WS_PORT", "8081"),
		Environment: getEnvOrDefault("GO_ENV", "development"),
		LogLevel:    getEnvOrDefault("LOG_LEVEL", "info"),
	}

	required := map[string]*string{
		"REDIS_URL": &cfg.RedisURL,
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
