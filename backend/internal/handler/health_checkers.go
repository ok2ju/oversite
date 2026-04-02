package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

// DBChecker wraps *sql.DB to satisfy HealthChecker.
type DBChecker struct {
	DB *sql.DB
}

func (c *DBChecker) Ping(ctx context.Context) error {
	return c.DB.PingContext(ctx)
}

// MinIOChecker checks MinIO connectivity via its health endpoint.
type MinIOChecker struct {
	Endpoint string
	UseSSL   bool
}

func (c *MinIOChecker) Ping(ctx context.Context) error {
	scheme := "http"
	if c.UseSSL {
		scheme = "https"
	}
	url := scheme + "://" + c.Endpoint + "/minio/health/live"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("minio health check returned status %d", resp.StatusCode)
	}
	return nil
}
