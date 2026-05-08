package sysinfo

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// processMemoryCountersEx mirrors the Win32 PROCESS_MEMORY_COUNTERS_EX struct.
// We declare it here because golang.org/x/sys/windows doesn't expose
// GetProcessMemoryInfo; psapi.dll is available via NewLazySystemDLL the same
// way GlobalMemoryStatusEx is wired in totalram_windows.go.
//
// SIZE_T is pointer-sized (uintptr) on Windows. cb is set to sizeof(struct)
// before the call so the OS can validate the buffer size.
type processMemoryCountersEx struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
}

var (
	modPsapi                 = windows.NewLazySystemDLL("psapi.dll")
	procGetProcessMemoryInfo = modPsapi.NewProc("GetProcessMemoryInfo")
)

func processMemory() (ProcessMemoryInfo, error) {
	var counters processMemoryCountersEx
	counters.CB = uint32(unsafe.Sizeof(counters))
	r1, _, err := procGetProcessMemoryInfo.Call(
		uintptr(windows.CurrentProcess()),
		uintptr(unsafe.Pointer(&counters)),
		uintptr(counters.CB),
	)
	if r1 == 0 {
		return ProcessMemoryInfo{}, fmt.Errorf("GetProcessMemoryInfo: %w", err)
	}
	return ProcessMemoryInfo{
		WorkingSetSize: uint64(counters.WorkingSetSize),
		PrivateUsage:   uint64(counters.PrivateUsage),
	}, nil
}
