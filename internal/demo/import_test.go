package demo_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/testutil"
)

// cs2Header returns a minimal valid CS2 demo file header (magic bytes + padding).
func cs2Header() []byte {
	header := make([]byte, 64)
	copy(header, "PBDEMS2\x00")
	return header
}

// writeTempDem creates a temp .dem file with the given contents and returns its path.
func writeTempDem(t *testing.T, dir string, contents []byte) string {
	t.Helper()
	path := filepath.Join(dir, "test.dem")
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// newImportService is a test helper that wires up an ImportService with an
// in-memory database and a temp demos directory.
func newImportService(t *testing.T) (*demo.ImportService, string) {
	t.Helper()
	q, db := testutil.NewTestQueries(t)
	demosDir := t.TempDir()
	return demo.NewImportService(q, db, demosDir), demosDir
}

func TestImportFile_Success(t *testing.T) {
	svc, demosDir := newImportService(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	demPath := writeTempDem(t, tmpDir, cs2Header())

	got, err := svc.ImportFile(ctx, demPath)
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}

	if got.ID == 0 {
		t.Error("expected non-zero demo ID")
	}
	wantPath := filepath.Join(demosDir, "test.dem")
	if got.FilePath != wantPath {
		t.Errorf("FilePath = %q, want %q", got.FilePath, wantPath)
	}
	if _, err := os.Stat(got.FilePath); err != nil {
		t.Errorf("expected demo file at %q: %v", got.FilePath, err)
	}
	// Source file should remain (we copy, not move).
	if _, err := os.Stat(demPath); err != nil {
		t.Errorf("expected source demo to remain at %q: %v", demPath, err)
	}
	if got.FileSize != int64(len(cs2Header())) {
		t.Errorf("FileSize = %d, want %d", got.FileSize, len(cs2Header()))
	}
	if got.Status != "imported" {
		t.Errorf("Status = %q, want %q", got.Status, "imported")
	}
	if got.MapName != "" {
		t.Errorf("MapName = %q, want empty", got.MapName)
	}
}

func TestImportFile_DeduplicatesFilenames(t *testing.T) {
	svc, demosDir := newImportService(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	demPath := writeTempDem(t, tmpDir, cs2Header())

	first, err := svc.ImportFile(ctx, demPath)
	if err != nil {
		t.Fatalf("first ImportFile: %v", err)
	}
	second, err := svc.ImportFile(ctx, demPath)
	if err != nil {
		t.Fatalf("second ImportFile: %v", err)
	}

	if first.FilePath == second.FilePath {
		t.Errorf("expected unique paths, both = %q", first.FilePath)
	}
	if filepath.Dir(second.FilePath) != demosDir {
		t.Errorf("second file dir = %q, want %q", filepath.Dir(second.FilePath), demosDir)
	}
}

func TestImportFile_InvalidExtension(t *testing.T) {
	svc, demosDir := newImportService(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(txtPath, cs2Header(), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := svc.ImportFile(ctx, txtPath)
	if !errors.Is(err, demo.ErrInvalidExtension) {
		t.Errorf("ImportFile error = %v, want %v", err, demo.ErrInvalidExtension)
	}
	assertDemosDirEmpty(t, demosDir)
}

func TestImportFile_InvalidMagicBytes(t *testing.T) {
	svc, demosDir := newImportService(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	randomBytes := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A}
	demPath := writeTempDem(t, tmpDir, randomBytes)

	_, err := svc.ImportFile(ctx, demPath)
	if !errors.Is(err, demo.ErrInvalidMagicBytes) {
		t.Errorf("ImportFile error = %v, want %v", err, demo.ErrInvalidMagicBytes)
	}
	assertDemosDirEmpty(t, demosDir)
}

func TestImportFile_FileNotFound(t *testing.T) {
	svc, _ := newImportService(t)
	ctx := context.Background()

	_, err := svc.ImportFile(ctx, "/nonexistent/path/match.dem")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestValidateFile_Valid(t *testing.T) {
	svc, _ := newImportService(t)

	tmpDir := t.TempDir()
	demPath := writeTempDem(t, tmpDir, cs2Header())

	if err := svc.ValidateFile(demPath); err != nil {
		t.Errorf("ValidateFile: %v", err)
	}
}

func TestValidateFile_InvalidExtension(t *testing.T) {
	svc, _ := newImportService(t)

	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(txtPath, cs2Header(), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := svc.ValidateFile(txtPath)
	if !errors.Is(err, demo.ErrInvalidExtension) {
		t.Errorf("ValidateFile error = %v, want %v", err, demo.ErrInvalidExtension)
	}
}

func TestValidateFile_InvalidMagicBytes(t *testing.T) {
	svc, _ := newImportService(t)

	tmpDir := t.TempDir()
	demPath := writeTempDem(t, tmpDir, []byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8})

	err := svc.ValidateFile(demPath)
	if !errors.Is(err, demo.ErrInvalidMagicBytes) {
		t.Errorf("ValidateFile error = %v, want %v", err, demo.ErrInvalidMagicBytes)
	}
}

func TestValidateFile_FileNotFound(t *testing.T) {
	svc, _ := newImportService(t)

	err := svc.ValidateFile("/nonexistent/path/match.dem")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestImportFile_CSGOMagicBytes(t *testing.T) {
	svc, _ := newImportService(t)
	ctx := context.Background()

	header := make([]byte, 64)
	copy(header, "HL2DEMO\x00")

	tmpDir := t.TempDir()
	demPath := writeTempDem(t, tmpDir, header)

	got, err := svc.ImportFile(ctx, demPath)
	if err != nil {
		t.Fatalf("ImportFile with CSGO magic: %v", err)
	}
	if got.Status != "imported" {
		t.Errorf("Status = %q, want %q", got.Status, "imported")
	}
}

// assertDemosDirEmpty fails the test if any files were left in the demos dir
// after a failed import.
func assertDemosDirEmpty(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("expected demos dir empty, got %v", names)
	}
}
