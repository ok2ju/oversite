package demo

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/ok2ju/oversite/internal/store"
)

// ImportService handles importing demo files into the database.
type ImportService struct {
	queries *store.Queries
	db      *sql.DB
}

// NewImportService creates a new ImportService.
func NewImportService(queries *store.Queries, db *sql.DB) *ImportService {
	return &ImportService{
		queries: queries,
		db:      db,
	}
}

// ValidateFile runs all validation checks on a demo file without persisting
// anything to the database. It checks extension, file size, and magic bytes.
func (s *ImportService) ValidateFile(filePath string) error {
	if err := ValidateExtension(filePath); err != nil {
		return err
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	if err := ValidateSize(info.Size()); err != nil {
		return err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	header := make([]byte, 8)
	n, err := f.Read(header)
	if err != nil {
		return fmt.Errorf("read header: %w", err)
	}
	if err := ValidateMagicBytes(header[:n]); err != nil {
		return err
	}

	return nil
}

// ImportFile validates a demo file and creates a database record for it.
// Compressed .dem.zst files are decompressed in-place before validation.
// The returned Demo has Status "imported" with MapName and MatchDate empty,
// to be populated after parsing.
func (s *ImportService) ImportFile(ctx context.Context, filePath string, userID int64) (*store.Demo, error) {
	if err := ValidateExtension(filePath); err != nil {
		return nil, err
	}

	// Decompress .dem.zst files before validation.
	if strings.HasSuffix(strings.ToLower(filePath), ".dem.zst") {
		demPath, err := DecompressZstd(filePath)
		if err != nil {
			return nil, fmt.Errorf("decompressing demo: %w", err)
		}
		filePath = demPath
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	if err := ValidateSize(info.Size()); err != nil {
		return nil, err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	header := make([]byte, 8)
	n, err := f.Read(header)
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if err := ValidateMagicBytes(header[:n]); err != nil {
		return nil, err
	}

	demo, err := s.queries.CreateDemo(ctx, store.CreateDemoParams{
		UserID:   userID,
		FilePath: filePath,
		FileSize: info.Size(),
		Status:   "imported",
		MapName:  "",
	})
	if err != nil {
		return nil, fmt.Errorf("create demo: %w", err)
	}

	return &demo, nil
}
