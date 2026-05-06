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

func TestImportFile_Success(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)
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
	if got.FilePath != demPath {
		t.Errorf("FilePath = %q, want %q", got.FilePath, demPath)
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

	fetched, err := q.GetDemoByID(ctx, got.ID)
	if err != nil {
		t.Fatalf("GetDemoByID: %v", err)
	}
	if fetched.FilePath != demPath {
		t.Errorf("fetched FilePath = %q, want %q", fetched.FilePath, demPath)
	}
}

func TestImportFile_InvalidExtension(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)
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
}

func TestImportFile_InvalidMagicBytes(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)
	ctx := context.Background()

	tmpDir := t.TempDir()
	randomBytes := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A}
	demPath := writeTempDem(t, tmpDir, randomBytes)

	_, err := svc.ImportFile(ctx, demPath)
	if !errors.Is(err, demo.ErrInvalidMagicBytes) {
		t.Errorf("ImportFile error = %v, want %v", err, demo.ErrInvalidMagicBytes)
	}
}

func TestImportFile_FileNotFound(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)
	ctx := context.Background()

	_, err := svc.ImportFile(ctx, "/nonexistent/path/match.dem")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestValidateFile_Valid(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)

	tmpDir := t.TempDir()
	demPath := writeTempDem(t, tmpDir, cs2Header())

	if err := svc.ValidateFile(demPath); err != nil {
		t.Errorf("ValidateFile: %v", err)
	}
}

func TestValidateFile_InvalidExtension(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)

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
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)

	tmpDir := t.TempDir()
	demPath := writeTempDem(t, tmpDir, []byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8})

	err := svc.ValidateFile(demPath)
	if !errors.Is(err, demo.ErrInvalidMagicBytes) {
		t.Errorf("ValidateFile error = %v, want %v", err, demo.ErrInvalidMagicBytes)
	}
}

func TestValidateFile_FileNotFound(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)

	err := svc.ValidateFile("/nonexistent/path/match.dem")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestImportFile_CSGOMagicBytes(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)
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
