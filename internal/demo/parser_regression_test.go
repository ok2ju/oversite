package demo

import (
	"context"
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
	defer func() { _ = f.Close() }()

	result, err := NewDemoParser().Parse(context.Background(), f)
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
	// any round with a winner (no draws in Faceit MR12). The parser sources
	// the score from ScoreUpdated (authoritative, fires before RoundEnd in
	// v5) and no longer reads WinnerState.Score() at all — this guards
	// against a future library version aligning with its docs and silently
	// double-counting.
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

	// Sanity-check weapon_fire output: every live round in a real match has
	// many shots, no shot may use a grenade or knife (the parser filters by
	// EquipmentClass), and every shot must carry a finite yaw in extra_data.
	var fireCount int
	firesByRound := make(map[int]int)
	for _, ev := range result.Events {
		if ev.Type != "weapon_fire" {
			continue
		}
		fireCount++
		firesByRound[ev.RoundNumber]++

		lw := strings.ToLower(ev.Weapon)
		if strings.Contains(lw, "grenade") || strings.Contains(lw, "molotov") ||
			strings.Contains(lw, "flashbang") || strings.Contains(lw, "decoy") ||
			strings.Contains(lw, "knife") || strings.Contains(lw, "bayonet") {
			t.Errorf("weapon_fire emitted for non-firearm %q at tick %d", ev.Weapon, ev.Tick)
		}

		wf, ok := ev.ExtraData.(*WeaponFireExtra)
		if !ok || wf == nil {
			t.Errorf("weapon_fire at tick %d missing typed WeaponFireExtra", ev.Tick)
			continue
		}
		if wf.Yaw != wf.Yaw { // NaN check
			t.Errorf("weapon_fire at tick %d has NaN yaw", ev.Tick)
		}
	}
	if fireCount == 0 {
		t.Error("expected at least some weapon_fire events in a 24-round match, got 0")
	}
	for round := 1; round <= len(result.Rounds); round++ {
		if firesByRound[round] == 0 {
			t.Errorf("round %d has no weapon_fire events", round)
		}
	}
}
