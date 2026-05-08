package sysinfo

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func totalRAM() (uint64, error) {
	var info unix.Sysinfo_t
	if err := unix.Sysinfo(&info); err != nil {
		return 0, fmt.Errorf("sysinfo: %w", err)
	}
	return uint64(info.Totalram) * uint64(info.Unit), nil
}
