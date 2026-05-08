package sysinfo

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// memoryStatusEx mirrors the Win32 MEMORYSTATUSEX struct. We declare it here
// because golang.org/x/sys/windows doesn't expose this API; it's available via
// kernel32!GlobalMemoryStatusEx, which we resolve through a LazySystemDLL.
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

var (
	modKernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalMemoryStatusEx = modKernel32.NewProc("GlobalMemoryStatusEx")
)

func totalRAM() (uint64, error) {
	var status memoryStatusEx
	status.Length = uint32(unsafe.Sizeof(status))
	r1, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status)))
	if r1 == 0 {
		return 0, fmt.Errorf("GlobalMemoryStatusEx: %w", err)
	}
	return status.TotalPhys, nil
}
