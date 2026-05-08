// Package sysinfo exposes platform-specific system information used to size
// runtime tunables (e.g. GOMEMLIMIT) at startup. Detection failures are
// non-fatal: callers fall back to a conservative default.
package sysinfo

// TotalRAM returns the host's total physical memory in bytes. Returns 0 and a
// non-nil error if detection fails on the current platform; callers should
// substitute a conservative default rather than treating this as fatal.
func TotalRAM() (uint64, error) {
	return totalRAM()
}
