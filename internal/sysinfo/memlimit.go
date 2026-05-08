package sysinfo

// HeapLimits is the pair of memory budgets the parser/runtime should respect
// for the current host.
//
//   - GOMEMLIMIT is the soft target the Go GC steers heap usage toward; once
//     live heap approaches it, GC runs aggressively to relieve pressure.
//   - KillSwitch is the hard ceiling the parser watchdog uses to abort a
//     runaway parse before the process pages the OS into a freeze. It must
//     sit above GOMEMLIMIT so a healthy parse doesn't trip it.
type HeapLimits struct {
	GOMEMLIMIT uint64
	KillSwitch uint64
}

const (
	minHeapLimit uint64 = 1 << 30 // 1 GiB
	maxHeapLimit uint64 = 4 << 30 // 4 GiB
)

// RecommendedHeapLimits returns memory budgets sized to a fraction of total
// host RAM, clamped to [1 GiB, 4 GiB].
//
// We deliberately undershoot total RAM. A clean parse holds ~500 MB live; the
// soft limit at 12.5% gives Go room to absorb spikes without blowing past the
// kill-switch on healthy demos, and the kill-switch at 18.75% leaves ~80% of
// RAM for the WebView2 process, the OS, file-system cache, and anything else
// the user has open. On a 16 GB Windows host that's 2 GiB soft / 3 GiB hard
// versus the previous static 4 GiB ceiling, which the OS was already paging
// against by the time the watchdog fired.
//
// totalRAM == 0 (detection failed) returns the conservative minimum so we
// never over-promise on an unknown host.
func RecommendedHeapLimits(totalRAM uint64) HeapLimits {
	if totalRAM == 0 {
		return HeapLimits{
			GOMEMLIMIT: minHeapLimit,
			KillSwitch: minHeapLimit + (minHeapLimit / 2),
		}
	}
	soft := clampHeap(totalRAM / 8)        // 12.5%
	hard := clampHeap((totalRAM * 3) / 16) // 18.75%
	if hard < soft {
		hard = soft
	}
	return HeapLimits{GOMEMLIMIT: soft, KillSwitch: hard}
}

func clampHeap(v uint64) uint64 {
	if v < minHeapLimit {
		return minHeapLimit
	}
	if v > maxHeapLimit {
		return maxHeapLimit
	}
	return v
}
