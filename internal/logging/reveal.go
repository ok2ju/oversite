package logging

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Reveal opens the given path in the OS file manager (Finder on macOS,
// Explorer on Windows, the configured handler on Linux).
func Reveal(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("reveal %q: %w", path, err)
	}
	// `explorer` on Windows returns exit code 1 even on success — ignore Wait.
	go func() { _ = cmd.Wait() }()
	return nil
}
