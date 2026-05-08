package sysinfo

import "testing"

func TestRecommendedHeapLimits(t *testing.T) {
	const gib = uint64(1 << 30)

	tests := []struct {
		name        string
		totalRAM    uint64
		wantSoft    uint64
		wantHard    uint64
		description string
	}{
		{
			name:        "detection failed falls back to floor",
			totalRAM:    0,
			wantSoft:    1 * gib,
			wantHard:    gib + gib/2, // 1.5 GiB
			description: "unknown host gets the most conservative budget",
		},
		{
			name:        "8 GB host clamps to floor",
			totalRAM:    8 * gib,
			wantSoft:    1 * gib,     // 12.5% = 1 GiB
			wantHard:    gib + gib/2, // 18.75% = 1.5 GiB
			description: "small box: floor wins for soft, ratio for hard",
		},
		{
			name:        "16 GB host (the failing Windows case)",
			totalRAM:    16 * gib,
			wantSoft:    2 * gib, // 12.5% = 2 GiB
			wantHard:    3 * gib, // 18.75% = 3 GiB
			description: "leaves ~13 GB for WebView2 + OS",
		},
		{
			name:        "32 GB host hits the ceiling",
			totalRAM:    32 * gib,
			wantSoft:    4 * gib, // 12.5% = 4 GiB (ceiling)
			wantHard:    4 * gib, // 18.75% = 6 GiB → clamped to 4 GiB
			description: "no demo should eat more than 4 GiB regardless of host",
		},
		{
			name:        "64 GB workstation also caps at ceiling",
			totalRAM:    64 * gib,
			wantSoft:    4 * gib,
			wantHard:    4 * gib,
			description: "ceiling protects against runaway on plenty-of-RAM hosts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RecommendedHeapLimits(tt.totalRAM)

			if got.GOMEMLIMIT != tt.wantSoft {
				t.Errorf("GOMEMLIMIT = %d, want %d (%s)", got.GOMEMLIMIT, tt.wantSoft, tt.description)
			}
			if got.KillSwitch != tt.wantHard {
				t.Errorf("KillSwitch = %d, want %d (%s)", got.KillSwitch, tt.wantHard, tt.description)
			}

			if got.GOMEMLIMIT < minHeapLimit || got.GOMEMLIMIT > maxHeapLimit {
				t.Errorf("GOMEMLIMIT out of [1 GiB, 4 GiB] range: %d", got.GOMEMLIMIT)
			}
			if got.KillSwitch < got.GOMEMLIMIT {
				t.Errorf("KillSwitch (%d) must be >= GOMEMLIMIT (%d)", got.KillSwitch, got.GOMEMLIMIT)
			}
			if got.KillSwitch > maxHeapLimit {
				t.Errorf("KillSwitch (%d) exceeds ceiling (%d)", got.KillSwitch, maxHeapLimit)
			}
		})
	}
}
