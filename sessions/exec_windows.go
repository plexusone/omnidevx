//go:build windows

package sessions

import (
	"errors"
	"os"
	"os/exec"
)

// execProcess runs the command with the terminal attached and exits with its
// status, because Windows cannot replace a running process.
func execProcess(path string, argv []string) error {
	cmd := exec.Command(path, argv[1:]...) //nolint:gosec // argv comes from the session's own reader
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	if err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
