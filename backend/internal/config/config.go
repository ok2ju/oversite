package config

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// Config holds all configuration values for the application.
type Config struct {
	Port               string
	WSPort             string
	DatabaseURL        string
	RedisURL           string
	MinioEndpoint      string
	MinioAccessKey     string
	MinioSecretKey     string
	MinioUseSSL        bool
	MinioBucket        string
	FaceitClientID     string
	FaceitClientSecret string
	FaceitRedirectURI  string
	Environment        string
	LogLevel           string
}

// Load reads configuration from environment variables.
// Required variables: DATABASE_URL, REDIS_URL, MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY.
// Returns an error if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnvOrDefault("PORT", "8080"),
		WSPort:      getEnvOrDefault("WS_PORT", "8081"),
		MinioBucket: getEnvOrDefault("MINIO_BUCKET", "oversite-demos"),
		MinioUseSSL: getEnvOrDefault("MINIO_USE_SSL", "false") == "true",
		Environment: getEnvOrDefault("GO_ENV", "development"),
		LogLevel:    getEnvOrDefault("LOG_LEVEL", "info"),
	}

	// Required vars
	required := map[string]*string{
		"DATABASE_URL":        &cfg.DatabaseURL,
		"REDIS_URL":           &cfg.RedisURL,
		"MINIO_ENDPOINT":      &cfg.MinioEndpoint,
		"MINIO_ACCESS_KEY":    &cfg.MinioAccessKey,
		"MINIO_SECRET_KEY":    &cfg.MinioSecretKey,
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

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
