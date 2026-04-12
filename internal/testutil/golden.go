package testutil

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

// TestdataPath returns the absolute path to the project-root testdata/
// directory, regardless of which package calls it.
func TestdataPath() string {
	_, filename, _, _ := runtime.Caller(0)
	// internal/testutil/golden.go → ../../testdata
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata")
}

// UpdateGolden writes got to the named golden file when -update is passed.
// The file is created under testdata/<name>.
func UpdateGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	if !*update {
		return
	}
	path := filepath.Join(TestdataPath(), name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for golden %s: %v", name, err)
	}
	if err := os.WriteFile(path, got, 0o644); err != nil {
		t.Fatalf("write golden %s: %v", name, err)
	}
}

// CompareGolden reads the named golden file and compares it to got.
// If -update is set the golden file is written first, so the comparison
// always passes on an update run.
func CompareGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	UpdateGolden(t, name, got)

	path := filepath.Join(TestdataPath(), name)
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update to create)", name, err)
	}
	if string(got) != string(want) {
		t.Errorf("golden %s mismatch:\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
	}
}

// LoadFixture reads a JSON fixture file from testdata/ and unmarshals it into v.
func LoadFixture(t *testing.T, name string, v interface{}) {
	t.Helper()
	data := LoadFixtureBytes(t, name)
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", name, err)
	}
}

// LoadFixtureBytes reads a raw file from testdata/.
func LoadFixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join(TestdataPath(), name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
	}
	return data
}
