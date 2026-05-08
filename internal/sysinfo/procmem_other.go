//go:build !windows

package sysinfo

// processMemory on non-Windows platforms returns a zero-value struct and nil
// error. The watchdog falls back to runtime.MemStats on these hosts — the Go
// runtime is more aggressive about returning pages to the OS on macOS/Linux,
// so HeapSys tracks the actual working set closely enough.
func processMemory() (ProcessMemoryInfo, error) {
	return ProcessMemoryInfo{}, nil
}
