package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestdataPath returns the absolute path to the testdata directory.
func TestdataPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata")
}

// LoadFixture reads a JSON fixture file from testdata/ and unmarshals it into v.
func LoadFixture(t *testing.T, name string, v interface{}) {
	t.Helper()
	path := filepath.Join(TestdataPath(), name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("loading fixture %s: %v", name, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("unmarshaling fixture %s: %v", name, err)
	}
}

// LoadFixtureBytes reads a raw fixture file from testdata/.
func LoadFixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join(TestdataPath(), name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("loading fixture %s: %v", name, err)
	}
	return data
}
