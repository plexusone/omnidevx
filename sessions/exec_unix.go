//go:build !windows

package sessions

import (
	"os"
	"syscall"
)

// execProcess replaces the current process with the command.
func execProcess(path string, argv []string) error {
	return syscall.Exec(path, argv, os.Environ()) //nolint:gosec // argv comes from the session's own reader
}
