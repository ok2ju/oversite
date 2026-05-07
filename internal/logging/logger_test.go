package logging

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// reset clears the package-level state so tests can Init repeatedly.
// Tests do not share process state with production code — each test gets
// a fresh temp directory and resets afterwards.
func reset(t *testing.T) {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()
	if errorsFile != nil {
		_ = errorsFile.Close()
		errorsFile = nil
	}
	initialized = false
}

func TestInit_WritesInfoAndAbove(t *testing.T) {
	t.Cleanup(func() { reset(t) })

	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	slog.Debug("should-not-appear")
	slog.Info("info-message", "k", "v")
	slog.Warn("warn-message", "k", "v")
	slog.Error("error-message", "err", "boom")

	if err := Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, errorsFileName))
	if err != nil {
		t.Fatalf("reading errors.txt: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "should-not-appear") {
		t.Errorf("DEBUG message leaked into errors.txt:\n%s", content)
	}
	if !strings.Contains(content, "info-message") {
		t.Errorf("INFO message missing from errors.txt:\n%s", content)
	}
	if !strings.Contains(content, "warn-message") {
		t.Errorf("WARN message missing from errors.txt:\n%s", content)
	}
	if !strings.Contains(content, "error-message") {
		t.Errorf("ERROR message missing from errors.txt:\n%s", content)
	}
}

func TestInit_RotatesAtSize(t *testing.T) {
	t.Cleanup(func() { reset(t) })

	dir := t.TempDir()
	// Tiny max size (1 MB) so we don't spend a minute writing 5MB.
	if err := initWith(dir, 1, 3); err != nil {
		t.Fatalf("initWith: %v", err)
	}

	// Write ~1.2 MB of WARN lines.
	big := strings.Repeat("x", 512)
	for i := 0; i < 2500; i++ {
		slog.Warn(big)
	}

	if err := Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	var rotated int
	var foundCurrent bool
	for _, e := range entries {
		name := e.Name()
		switch {
		case name == errorsFileName:
			foundCurrent = true
		case strings.HasPrefix(name, "errors-") && strings.HasSuffix(name, ".txt"):
			rotated++
		}
	}

	if !foundCurrent {
		t.Errorf("expected current errors.txt, got entries: %v", entries)
	}
	if rotated == 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected at least one rotated backup, got: %v", names)
	}
}

func TestInit_StdLibLogBridge(t *testing.T) {
	t.Cleanup(func() { reset(t) })

	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	log.Printf("boom: %d", 42)

	if err := Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, errorsFileName))
	if err != nil {
		t.Fatalf("reading errors.txt: %v", err)
	}
	if !strings.Contains(string(data), "boom: 42") {
		t.Errorf("stdlib log.Printf not captured, got:\n%s", string(data))
	}
}
