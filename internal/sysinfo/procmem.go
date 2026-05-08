package sysinfo

// ProcessMemoryInfo holds the current process's memory counters as reported by
// the OS. On platforms other than Windows the struct is zero-valued and the
// caller is expected to fall back to runtime.MemStats — Windows is the only
// platform where the OS-reported working set diverges enough from the Go heap
// (because the runtime is slow to scavenge unused pages back to the OS) that
// the difference matters for the parser kill-switch.
type ProcessMemoryInfo struct {
	WorkingSetSize uint64 // resident, OS-reported (private + shareable in working set)
	PrivateUsage   uint64 // commit charge / private bytes
}

// ProcessMemory returns the current process's memory counters from the OS.
// On Windows this calls psapi!GetProcessMemoryInfo. On other platforms it
// returns a zero-value struct and nil error — callers should treat unknown
// counters as "fall back to runtime.MemStats" rather than as an error.
func ProcessMemory() (ProcessMemoryInfo, error) {
	return processMemory()
}
