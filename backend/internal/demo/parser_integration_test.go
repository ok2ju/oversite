//go:build integration

package demo

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

const (
	fixtureDir = "../../testdata/demos"
	goldenDir  = "../../testdata/golden"
	fixture    = "small_match.dem"
)

func openFixture(t *testing.T) *os.File {
	t.Helper()
	path := filepath.Join(fixtureDir, fixture)
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		t.Skipf("fixture not found: %s", path)
	}
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	return f
}

func parseFixture(t *testing.T) *ParseResult {
	t.Helper()
	f := openFixture(t)
	defer f.Close()

	dp := NewDemoParser()
	result, err := dp.Parse(f)
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}
	return result
}

func goldenPath(name string) string {
	return filepath.Join(goldenDir, name+".golden.json")
}

func writeGolden(t *testing.T, name string, v interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshalling golden data: %v", err)
	}
	if err := os.MkdirAll(goldenDir, 0o755); err != nil {
		t.Fatalf("creating golden dir: %v", err)
	}
	if err := os.WriteFile(goldenPath(name), append(data, '\n'), 0o644); err != nil {
		t.Fatalf("writing golden file: %v", err)
	}
}

func loadGolden(t *testing.T, name string, v interface{}) {
	t.Helper()
	data, err := os.ReadFile(goldenPath(name))
	if errors.Is(err, os.ErrNotExist) {
		t.Skipf("golden file not found: %s (run with -update to generate)", goldenPath(name))
	}
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("unmarshalling golden data: %v", err)
	}
}

func TestParseDemo_GoldenHeader(t *testing.T) {
	result := parseFixture(t)

	type headerAndRounds struct {
		Header MatchHeader
		Rounds []RoundData
	}
	got := headerAndRounds{Header: result.Header, Rounds: result.Rounds}

	name := "small_match_rounds"
	if *update {
		writeGolden(t, name, got)
		t.Logf("updated golden file: %s", goldenPath(name))
		return
	}

	var want headerAndRounds
	loadGolden(t, name, &want)

	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("header/rounds mismatch\ngot:  %s\nwant: %s", gotJSON, wantJSON)
	}
}

func TestParseDemo_GoldenTicks(t *testing.T) {
	result := parseFixture(t)

	// Only compare first 100 ticks to keep golden file small.
	ticks := result.Ticks
	if len(ticks) > 100 {
		ticks = ticks[:100]
	}

	name := "small_match_ticks"
	if *update {
		writeGolden(t, name, ticks)
		t.Logf("updated golden file: %s", goldenPath(name))
		return
	}

	var want []TickSnapshot
	loadGolden(t, name, &want)

	gotJSON, _ := json.Marshal(ticks)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("ticks mismatch\ngot:  %s\nwant: %s", gotJSON, wantJSON)
	}
}

func TestParseDemo_GoldenEvents(t *testing.T) {
	result := parseFixture(t)

	name := "small_match_events"
	if *update {
		writeGolden(t, name, result.Events)
		t.Logf("updated golden file: %s", goldenPath(name))
		return
	}

	var want []GameEvent
	loadGolden(t, name, &want)

	gotJSON, _ := json.Marshal(result.Events)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("events mismatch\ngot:  %s\nwant: %s", gotJSON, wantJSON)
	}
}

func TestParseDemo_GoldenLineups(t *testing.T) {
	result := parseFixture(t)

	name := "small_match_lineups"
	if *update {
		writeGolden(t, name, result.Lineups)
		t.Logf("updated golden file: %s", goldenPath(name))
		return
	}

	var want []GrenadeLineup
	loadGolden(t, name, &want)

	gotJSON, _ := json.Marshal(result.Lineups)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("lineups mismatch\ngot:  %s\nwant: %s", gotJSON, wantJSON)
	}
}

func TestParseDemo_BasicAssertions(t *testing.T) {
	result := parseFixture(t)

	if result.Header.MapName == "" {
		t.Error("expected non-empty map name")
	}
	if result.Header.TickRate <= 0 {
		t.Errorf("expected positive tick rate, got %f", result.Header.TickRate)
	}
	if result.Header.TotalTicks <= 0 {
		t.Errorf("expected positive total ticks, got %d", result.Header.TotalTicks)
	}
	if len(result.Rounds) == 0 {
		t.Error("expected at least one round")
	}
	if len(result.Ticks) == 0 {
		t.Error("expected tick snapshots")
	}
	if len(result.Events) == 0 {
		t.Error("expected game events")
	}

	// Every round should have consistent data.
	for _, rd := range result.Rounds {
		if rd.Number <= 0 {
			t.Errorf("round number should be positive, got %d", rd.Number)
		}
		if rd.WinnerSide != "CT" && rd.WinnerSide != "T" && rd.WinnerSide != "" {
			t.Errorf("unexpected winner side %q for round %d", rd.WinnerSide, rd.Number)
		}
		if rd.EndTick < rd.StartTick {
			t.Errorf("round %d: end tick %d < start tick %d", rd.Number, rd.EndTick, rd.StartTick)
		}
	}
}
