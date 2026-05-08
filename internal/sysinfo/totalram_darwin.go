package sysinfo

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func totalRAM() (uint64, error) {
	v, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return 0, fmt.Errorf("sysctl hw.memsize: %w", err)
	}
	return v, nil
}
