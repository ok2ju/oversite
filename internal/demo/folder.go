package demo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ok2ju/oversite/internal/store"
)

// FolderImportError records a file that failed to import.
type FolderImportError struct {
	FilePath string
	Err      error
}

func (e *FolderImportError) Error() string {
	return fmt.Sprintf("%s: %s", e.FilePath, e.Err)
}

// FolderImportResult holds the outcome of a folder import operation.
type FolderImportResult struct {
	Imported []*store.Demo
	Errors   []FolderImportError
}

// ImportFolder recursively scans dirPath for .dem files and imports each one.
// Valid demos are returned in Imported; files that fail validation are collected
// in Errors rather than aborting the entire operation.
func (s *ImportService) ImportFolder(ctx context.Context, dirPath string, userID int64) (*FolderImportResult, error) {
	var demPaths []string

	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".dem") {
			demPaths = append(demPaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning directory: %w", err)
	}

	result := &FolderImportResult{}

	for _, path := range demPaths {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		demo, err := s.ImportFile(ctx, path, userID)
		if err != nil {
			result.Errors = append(result.Errors, FolderImportError{
				FilePath: path,
				Err:      err,
			})
			continue
		}
		result.Imported = append(result.Imported, demo)
	}

	return result, nil
}
