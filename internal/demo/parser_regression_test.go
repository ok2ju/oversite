package demo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseDemo_NoKnifeRounds is a slow end-to-end regression against the real
// Faceit demo in testdata/demos/1.dem. The demo contains a knife round plus a
// post-knife warmup round followed by 24 live rounds; the parser must drop the
// first two and renumber the remainder contiguously from 1.
//
// Skipped under -short (the Stop hook runs tests with -short, so this only
// fires when invoked manually).
func TestParseDemo_NoKnifeRounds(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: parses a ~400MB demo end-to-end")
	}

	demoPath := filepath.Join("..", "..", "testdata", "demos", "1.dem")
	f, err := os.Open(demoPath)
	if err != nil {
		t.Skipf("testdata demo not available: %v", err)
	}
	defer f.Close()

	result, err := NewDemoParser().Parse(f)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if got := len(result.Rounds); got != 24 {
		t.Errorf("len(Rounds) = %d, want 24", got)
	}
	if len(result.Rounds) == 0 {
		return
	}
	for i, rd := range result.Rounds {
		if rd.Number != i+1 {
			t.Errorf("Rounds[%d].Number = %d, want %d (contiguous numbering)", i, rd.Number, i+1)
		}
	}

	// Score invariant: after round N, (ct_score + t_score) must equal N for
	// any round with a winner (no draws in Faceit MR12). The previous parser
	// added +1 to WinnerState.Score() under the assumption scores were
	// pre-update at RoundEnd, which double-counted when the library updated
	// them in-place — that's what this guards against.
	for _, rd := range result.Rounds {
		if rd.WinnerSide == "" {
			continue
		}
		if rd.CTScore+rd.TScore != rd.Number {
			t.Errorf("round %d: CTScore+TScore = %d+%d = %d, want %d",
				rd.Number, rd.CTScore, rd.TScore, rd.CTScore+rd.TScore, rd.Number)
		}
	}

	killsByRound := make(map[int][]string)
	for _, ev := range result.Events {
		if ev.Type == "kill" {
			killsByRound[ev.RoundNumber] = append(killsByRound[ev.RoundNumber], ev.Weapon)
		}
	}
	for round, weapons := range killsByRound {
		if len(weapons) < 2 {
			continue
		}
		allKnife := true
		for _, w := range weapons {
			lw := strings.ToLower(w)
			if !strings.Contains(lw, "knife") && !strings.Contains(lw, "bayonet") {
				allKnife = false
				break
			}
		}
		if allKnife {
			t.Errorf("round %d survived filter but all %d kills are knife kills", round, len(weapons))
		}
	}
}
