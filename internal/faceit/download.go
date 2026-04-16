package faceit

import (
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/store"
)

// DownloadService downloads Faceit match demos and imports them.
type DownloadService struct {
	httpClient    *http.Client
	importService *demo.ImportService
	queries       *store.Queries
	downloadDir   string
}

// NewDownloadService creates a new DownloadService.
func NewDownloadService(
	httpClient *http.Client,
	importService *demo.ImportService,
	queries *store.Queries,
	downloadDir string,
) *DownloadService {
	return &DownloadService{
		httpClient:    httpClient,
		importService: importService,
		queries:       queries,
		downloadDir:   downloadDir,
	}
}

// DownloadAndImport downloads the demo for a Faceit match, decompresses if
// gzipped, imports it, and links it to the Faceit match record.
func (d *DownloadService) DownloadAndImport(
	ctx context.Context,
	faceitMatchID int64,
	userID int64,
	onProgress func(bytesDownloaded, totalBytes int64),
) (*store.Demo, error) {
	match, err := d.queries.GetFaceitMatchByID(ctx, faceitMatchID)
	if err != nil {
		return nil, fmt.Errorf("getting faceit match: %w", err)
	}
	if match.DemoUrl == "" {
		return nil, fmt.Errorf("match %d has no demo URL", faceitMatchID)
	}

	// Download the demo file.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, match.DemoUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("creating download request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading demo: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed (status %d)", resp.StatusCode)
	}

	// Ensure download directory exists.
	if err := os.MkdirAll(d.downloadDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating download dir: %w", err)
	}

	// Stream to a temp file.
	tmpFile, err := os.CreateTemp(d.downloadDir, "faceit-demo-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath) // Clean up temp file on error.
	}()

	var reader io.Reader = resp.Body
	if onProgress != nil {
		reader = &progressReader{
			reader:     resp.Body,
			total:      resp.ContentLength,
			onProgress: onProgress,
		}
	}

	if _, err := io.Copy(tmpFile, reader); err != nil {
		_ = tmpFile.Close()
		return nil, fmt.Errorf("writing demo file: %w", err)
	}
	_ = tmpFile.Close()

	// Decompress .gz if needed.
	demPath := tmpPath
	if strings.HasSuffix(match.DemoUrl, ".gz") {
		demPath, err = decompressGzip(tmpPath, d.downloadDir)
		if err != nil {
			return nil, fmt.Errorf("decompressing demo: %w", err)
		}
		_ = os.Remove(tmpPath) // Remove compressed file.
	}

	// Rename to final .dem file.
	finalPath := filepath.Join(d.downloadDir, fmt.Sprintf("faceit_%s.dem", match.FaceitMatchID))
	if err := os.Rename(demPath, finalPath); err != nil {
		return nil, fmt.Errorf("renaming demo file: %w", err)
	}

	// Import into the database.
	imported, err := d.importService.ImportFile(ctx, finalPath, userID)
	if err != nil {
		return nil, fmt.Errorf("importing demo: %w", err)
	}

	// Link the Faceit match to the imported demo.
	_, err = d.queries.LinkFaceitMatchToDemo(ctx, store.LinkFaceitMatchToDemoParams{
		DemoID: sql.NullInt64{Int64: imported.ID, Valid: true},
		ID:     faceitMatchID,
	})
	if err != nil {
		return nil, fmt.Errorf("linking match to demo: %w", err)
	}

	return imported, nil
}

// progressReader wraps an io.Reader and calls onProgress after each Read.
type progressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	onProgress func(bytesDownloaded, totalBytes int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.downloaded += int64(n)
	pr.onProgress(pr.downloaded, pr.total)
	return n, err
}

// decompressGzip decompresses a gzip file to a .dem file in the target directory.
func decompressGzip(gzPath, targetDir string) (string, error) {
	gzFile, err := os.Open(gzPath)
	if err != nil {
		return "", err
	}
	defer gzFile.Close() //nolint:errcheck

	gz, err := gzip.NewReader(gzFile)
	if err != nil {
		return "", err
	}
	defer gz.Close() //nolint:errcheck

	outFile, err := os.CreateTemp(targetDir, "faceit-demo-*.dem")
	if err != nil {
		return "", err
	}
	outPath := outFile.Name()

	if _, err := io.Copy(outFile, gz); err != nil {
		_ = outFile.Close()
		_ = os.Remove(outPath)
		return "", err
	}
	_ = outFile.Close()

	return outPath, nil
}
