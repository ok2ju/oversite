package demo_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/testutil"
)

func TestImportFolder_ImportsOnlyDemFiles(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)
	ctx := context.Background()
	user := createTestUser(t, q)

	dir := t.TempDir()

	// Create mix of files.
	writeTempDem(t, dir, cs2Header()) // test.dem
	writeFile(t, filepath.Join(dir, "notes.txt"), []byte("not a demo"))
	writeFile(t, filepath.Join(dir, "replay.DEM"), cs2Header()) // uppercase extension
	writeFile(t, filepath.Join(dir, "image.png"), []byte{0x89, 0x50, 0x4E, 0x47})

	result, err := svc.ImportFolder(ctx, dir, user.ID)
	if err != nil {
		t.Fatalf("ImportFolder: %v", err)
	}

	if len(result.Imported) != 2 {
		t.Errorf("imported %d demos, want 2", len(result.Imported))
	}
	if len(result.Errors) != 0 {
		t.Errorf("got %d errors, want 0: %v", len(result.Errors), result.Errors)
	}
}

func TestImportFolder_RecursiveSubdirectories(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)
	ctx := context.Background()
	user := createTestUser(t, q)

	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub", "nested")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	writeFile(t, filepath.Join(dir, "top.dem"), cs2Header())
	writeFile(t, filepath.Join(subDir, "deep.dem"), cs2Header())

	result, err := svc.ImportFolder(ctx, dir, user.ID)
	if err != nil {
		t.Fatalf("ImportFolder: %v", err)
	}

	if len(result.Imported) != 2 {
		t.Errorf("imported %d demos, want 2", len(result.Imported))
	}
}

func TestImportFolder_EmptyDirectory(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)
	ctx := context.Background()
	user := createTestUser(t, q)

	dir := t.TempDir()

	result, err := svc.ImportFolder(ctx, dir, user.ID)
	if err != nil {
		t.Fatalf("ImportFolder: %v", err)
	}

	if len(result.Imported) != 0 {
		t.Errorf("imported %d demos, want 0", len(result.Imported))
	}
	if len(result.Errors) != 0 {
		t.Errorf("got %d errors, want 0", len(result.Errors))
	}
}

func TestImportFolder_InvalidFilesCapturedAsErrors(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)
	ctx := context.Background()
	user := createTestUser(t, q)

	dir := t.TempDir()

	// Valid demo.
	writeFile(t, filepath.Join(dir, "good.dem"), cs2Header())
	// Invalid demo (bad magic bytes).
	writeFile(t, filepath.Join(dir, "bad.dem"), []byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8})

	result, err := svc.ImportFolder(ctx, dir, user.ID)
	if err != nil {
		t.Fatalf("ImportFolder: %v", err)
	}

	if len(result.Imported) != 1 {
		t.Errorf("imported %d demos, want 1", len(result.Imported))
	}
	if len(result.Errors) != 1 {
		t.Errorf("got %d errors, want 1", len(result.Errors))
	}
}

func TestImportFolder_CancelledContext(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)
	user := createTestUser(t, q)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.dem"), cs2Header())
	writeFile(t, filepath.Join(dir, "b.dem"), cs2Header())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	result, err := svc.ImportFolder(ctx, dir, user.ID)
	if err == nil {
		t.Fatal("expected context error, got nil")
	}
	// Some or none may have been imported before cancellation.
	_ = result
}

func TestImportFolder_NonexistentDirectory(t *testing.T) {
	q, db := testutil.NewTestQueries(t)
	svc := demo.NewImportService(q, db)
	ctx := context.Background()
	user := createTestUser(t, q)

	_, err := svc.ImportFolder(ctx, "/nonexistent/directory/path", user.ID)
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}

// writeFile is a helper that creates a file with the given contents.
func writeFile(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}
