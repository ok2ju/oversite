package faceit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/demo"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// ImportStore is the subset of store.Queries needed by DemoImporter.
type ImportStore interface {
	GetFaceitMatchByID(ctx context.Context, id uuid.UUID) (store.FaceitMatch, error)
	CreateDemo(ctx context.Context, arg store.CreateDemoParams) (store.Demo, error)
	LinkFaceitMatchToDemo(ctx context.Context, arg store.LinkFaceitMatchToDemoParams) (store.FaceitMatch, error)
}

// ImportObjectStore is the subset of storage.MinIOClient needed by DemoImporter.
type ImportObjectStore interface {
	PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64) error
	DeleteObject(ctx context.Context, bucket, key string) error
}

// ImportJobEnqueuer enqueues background jobs.
type ImportJobEnqueuer interface {
	Enqueue(ctx context.Context, stream string, data map[string]interface{}) (string, error)
}

// HTTPDownloader executes HTTP requests (typically *http.Client).
type HTTPDownloader interface {
	Do(req *http.Request) (*http.Response, error)
}

// Sentinel errors for demo import.
var (
	ErrMatchNotFound     = errors.New("faceit match not found")
	ErrMatchForbidden    = errors.New("faceit match belongs to another user")
	ErrNoDemoURL         = errors.New("faceit match has no demo URL")
	ErrDemoAlreadyLinked = errors.New("faceit match already has a linked demo")
	ErrDownloadFailed    = errors.New("demo download failed")
	ErrInvalidDemo       = errors.New("downloaded file is not a valid demo")
)

// ImportResult holds the outcome of a successful demo import.
type ImportResult struct {
	DemoID   uuid.UUID
	FileSize int64
}

// DemoImporter downloads demos from Faceit URLs, uploads to S3,
// creates DB records, and enqueues parse jobs.
type DemoImporter struct {
	store      ImportStore
	s3         ImportObjectStore
	queue      ImportJobEnqueuer
	httpClient HTTPDownloader
	bucket     string
}

// NewDemoImporter creates a new DemoImporter.
func NewDemoImporter(store ImportStore, s3 ImportObjectStore, queue ImportJobEnqueuer, httpClient HTTPDownloader, bucket string) *DemoImporter {
	return &DemoImporter{
		store:      store,
		s3:         s3,
		queue:      queue,
		httpClient: httpClient,
		bucket:     bucket,
	}
}

// Import downloads a demo from demoURL, validates it, uploads to S3,
// creates a DB record linked to the Faceit match, and enqueues a parse job.
func (d *DemoImporter) Import(ctx context.Context, userID, matchID uuid.UUID, faceitMatchID, demoURL string, matchDate time.Time) (*ImportResult, error) {
	// 1. Download demo to a temp file
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, demoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", ErrDownloadFailed, resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "demo-import-*.dem")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	defer func() { _ = tmpFile.Close() }()

	size, err := io.Copy(tmpFile, io.LimitReader(resp.Body, demo.MaxUploadSize+1))
	if err != nil {
		return nil, fmt.Errorf("%w: reading response body: %v", ErrDownloadFailed, err)
	}

	// 2. Validate size (before writing more to disk)
	if err := demo.ValidateSize(size); err != nil {
		return nil, ErrInvalidDemo
	}

	// 3. Validate magic bytes
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seeking temp file: %w", err)
	}

	header := make([]byte, 8)
	if _, err := io.ReadFull(tmpFile, header); err != nil {
		return nil, fmt.Errorf("%w: cannot read header", ErrInvalidDemo)
	}
	if err := demo.ValidateMagicBytes(header); err != nil {
		return nil, ErrInvalidDemo
	}

	// 4. Upload to S3
	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seeking temp file: %w", err)
	}

	demoID := uuid.New()
	key := fmt.Sprintf("demos/%s/%s.dem", userID, demoID)

	if err := d.s3.PutObject(ctx, d.bucket, key, tmpFile, size); err != nil {
		return nil, fmt.Errorf("uploading to S3: %w", err)
	}

	// 5. Create DB record
	demoRecord, err := d.store.CreateDemo(ctx, store.CreateDemoParams{
		UserID:        userID,
		FaceitMatchID: sql.NullString{String: faceitMatchID, Valid: true},
		FilePath:      key,
		FileSize:      size,
		Status:        "uploaded",
		MatchDate:     sql.NullTime{Time: matchDate, Valid: true},
	})
	if err != nil {
		// Clean up S3 object on DB failure
		if delErr := d.s3.DeleteObject(ctx, d.bucket, key); delErr != nil {
			slog.Error("cleaning up S3 object after DB failure", "error", delErr, "key", key)
		}
		return nil, fmt.Errorf("creating demo record: %w", err)
	}

	// 6. Link faceit_match to demo
	if _, err := d.store.LinkFaceitMatchToDemo(ctx, store.LinkFaceitMatchToDemoParams{
		ID:     matchID,
		DemoID: uuid.NullUUID{UUID: demoRecord.ID, Valid: true},
	}); err != nil {
		// Clean up S3 object on link failure
		if delErr := d.s3.DeleteObject(ctx, d.bucket, key); delErr != nil {
			slog.Error("cleaning up S3 object after link failure", "error", delErr, "key", key)
		}
		return nil, fmt.Errorf("linking faceit match to demo: %w", err)
	}

	// 7. Enqueue parse job (non-fatal on failure)
	if _, err := d.queue.Enqueue(ctx, "demo_parse", map[string]interface{}{
		"demo_id":   demoRecord.ID.String(),
		"file_path": key,
		"user_id":   userID.String(),
	}); err != nil {
		slog.Warn("enqueueing parse job after import", "error", err, "demo_id", demoRecord.ID)
	}

	return &ImportResult{
		DemoID:   demoRecord.ID,
		FileSize: size,
	}, nil
}

// ImportEnqueuer enqueues demo import jobs for async processing.
type ImportEnqueuer struct {
	queue  ImportJobEnqueuer
	stream string
}

// NewImportEnqueuer creates an ImportEnqueuer that enqueues jobs to the given stream.
func NewImportEnqueuer(queue ImportJobEnqueuer, stream string) *ImportEnqueuer {
	return &ImportEnqueuer{queue: queue, stream: stream}
}

// EnqueueImport enqueues a demo import job for async processing by the worker.
func (e *ImportEnqueuer) EnqueueImport(ctx context.Context, userID, matchID uuid.UUID, faceitMatchID, demoURL string, matchDate time.Time) error {
	_, err := e.queue.Enqueue(ctx, e.stream, map[string]interface{}{
		"user_id":         userID.String(),
		"match_id":        matchID.String(),
		"faceit_match_id": faceitMatchID,
		"demo_url":        demoURL,
		"match_date":      matchDate.Format(time.RFC3339),
	})
	return err
}

// ImportByMatchID looks up a Faceit match by ID, validates ownership and state,
// then delegates to Import.
func (d *DemoImporter) ImportByMatchID(ctx context.Context, userID, matchID uuid.UUID) (*ImportResult, error) {
	match, err := d.store.GetFaceitMatchByID(ctx, matchID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMatchNotFound
		}
		return nil, fmt.Errorf("getting faceit match: %w", err)
	}

	if match.UserID != userID {
		return nil, ErrMatchForbidden
	}

	if !match.DemoUrl.Valid {
		return nil, ErrNoDemoURL
	}

	if match.DemoID.Valid {
		return nil, ErrDemoAlreadyLinked
	}

	return d.Import(ctx, userID, match.ID, match.FaceitMatchID, match.DemoUrl.String, match.PlayedAt)
}
