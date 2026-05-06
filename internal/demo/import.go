package demo

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ok2ju/oversite/internal/store"
)

// ImportService handles importing demo files into the database.
type ImportService struct {
	queries  *store.Queries
	db       *sql.DB
	demosDir string
}

// NewImportService creates a new ImportService. demosDir is the app-managed
// directory where imported demo files are copied to live.
func NewImportService(queries *store.Queries, db *sql.DB, demosDir string) *ImportService {
	return &ImportService{
		queries:  queries,
		db:       db,
		demosDir: demosDir,
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

// ImportFile validates a demo file and copies it into the app-managed demos
// directory before recording it in the database. Compressed .dem.zst files are
// decompressed during the copy. The returned Demo has Status "imported" with
// MapName and MatchDate empty, to be populated after parsing.
func (s *ImportService) ImportFile(ctx context.Context, filePath string) (*store.Demo, error) {
	if err := ValidateExtension(filePath); err != nil {
		return nil, err
	}

	if s.demosDir == "" {
		return nil, fmt.Errorf("demos directory not configured")
	}
	if err := os.MkdirAll(s.demosDir, 0o755); err != nil {
		return nil, fmt.Errorf("ensure demos dir: %w", err)
	}

	srcInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	if err := ValidateSize(srcInfo.Size()); err != nil {
		return nil, err
	}

	destBase := filepath.Base(filePath)
	if IsCompressedDemo(destBase) {
		destBase = strings.TrimSuffix(destBase, filepath.Ext(destBase)) // drop .zst suffix
	}
	destPath, err := uniquePath(s.demosDir, destBase)
	if err != nil {
		return nil, fmt.Errorf("resolve destination: %w", err)
	}

	if IsCompressedDemo(filePath) {
		if err := decompressZstdToFile(filePath, destPath); err != nil {
			return nil, fmt.Errorf("decompressing demo: %w", err)
		}
	} else {
		if err := copyFile(filePath, destPath); err != nil {
			return nil, fmt.Errorf("copying demo: %w", err)
		}
	}

	cleanupOnFail := func() {
		_ = os.Remove(destPath)
	}

	destInfo, err := os.Stat(destPath)
	if err != nil {
		cleanupOnFail()
		return nil, fmt.Errorf("stat copied file: %w", err)
	}
	if err := ValidateSize(destInfo.Size()); err != nil {
		cleanupOnFail()
		return nil, err
	}

	f, err := os.Open(destPath)
	if err != nil {
		cleanupOnFail()
		return nil, fmt.Errorf("open copied file: %w", err)
	}
	header := make([]byte, 8)
	n, err := f.Read(header)
	_ = f.Close()
	if err != nil {
		cleanupOnFail()
		return nil, fmt.Errorf("read header: %w", err)
	}
	if err := ValidateMagicBytes(header[:n]); err != nil {
		cleanupOnFail()
		return nil, err
	}

	demo, err := s.queries.CreateDemo(ctx, store.CreateDemoParams{
		FilePath: destPath,
		FileSize: destInfo.Size(),
		Status:   "imported",
		MapName:  "",
	})
	if err != nil {
		cleanupOnFail()
		return nil, fmt.Errorf("create demo: %w", err)
	}

	return &demo, nil
}

// uniquePath returns a path inside dir for the desired filename, appending
// " (1)", " (2)", … before the extension if a file already exists.
func uniquePath(dir, name string) (string, error) {
	candidate := filepath.Join(dir, name)
	if _, err := os.Stat(candidate); errIsNotExist(err) {
		return candidate, nil
	} else if err != nil {
		return "", err
	}

	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; i < 10000; i++ {
		alt := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, i, ext))
		if _, err := os.Stat(alt); errIsNotExist(err) {
			return alt, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("too many filename collisions for %q", name)
}

func errIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}
