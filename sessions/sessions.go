// Package sessions composes the harness session readers into the catalog
// behind "omnidevx sessions", and prepares and runs a resume.
//
// Readers return what to run (a ResumeSpec); this package decides whether
// it is safe to run it and, if asked, replaces the current process with it.
package sessions

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	codex "github.com/plexusone/omni-openai/omnidevx"
	"github.com/plexusone/omnidevx-core/providers/claudecode"
	core "github.com/plexusone/omnidevx-core/sessions"
)

// NewDefaultCatalog returns a catalog over the local Claude Code and Codex
// CLI stores under the user's home directory.
func NewDefaultCatalog() (*core.Catalog, error) {
	claude, err := claudecode.NewSessionReader(claudecode.Options{})
	if err != nil {
		return nil, err
	}
	cdx, err := codex.NewSessionReader(codex.Config{})
	if err != nil {
		return nil, err
	}
	return core.NewCatalog(claude, cdx), nil
}

// Filter narrows a session list. Zero fields match everything.
type Filter struct {
	Harness core.Harness
	State   core.State
	// CWD keeps sessions whose working directory is this path or below it.
	CWD string
}

// Apply returns the sessions that match the filter, preserving order.
func (f Filter) Apply(in []core.Session) []core.Session {
	cwd := filepath.Clean(f.CWD)
	var out []core.Session
	for _, s := range in {
		if f.Harness != "" && s.Harness != f.Harness {
			continue
		}
		if f.State != "" && s.State != f.State {
			continue
		}
		if f.CWD != "" && s.CWD != cwd && !strings.HasPrefix(s.CWD, cwd+string(filepath.Separator)) {
			continue
		}
		out = append(out, s)
	}
	return out
}

// ParseSince converts a look-back such as "90m", "24h", or "7d" into the
// absolute time that far before now.
func ParseSince(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if days, ok := strings.CutSuffix(s, "d"); ok {
		n, err := strconv.Atoi(days)
		if err != nil || n < 0 {
			return time.Time{}, fmt.Errorf("invalid duration %q: want a number of days like 7d", s)
		}
		return now.AddDate(0, 0, -n), nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return time.Time{}, fmt.Errorf("invalid duration %q: use forms like 90m, 24h, or 7d", s)
	}
	return now.Add(-d), nil
}

// Prepare checks that a session can be resumed and returns what to run.
// A running session is refused unless force is set, because resuming one
// that is live in another terminal would attach a second agent to the
// same conversation. The working directory must still exist: Claude Code
// finds a session by the directory it started in.
func Prepare(s core.Session, force bool) (core.ResumeSpec, error) {
	if s.State == core.StateRunning && !force {
		where := ""
		if s.Runtime != nil && s.Runtime.PID > 0 {
			where = fmt.Sprintf(" (pid %d)", s.Runtime.PID)
		}
		return core.ResumeSpec{}, fmt.Errorf("session %s is already running%s; use --force to resume it anyway", s.ID, where)
	}
	spec := s.Resume
	if len(spec.Argv) == 0 {
		return core.ResumeSpec{}, fmt.Errorf("session %s has no resume command", s.ID)
	}
	if spec.Dir != "" {
		info, err := os.Stat(spec.Dir)
		if err != nil || !info.IsDir() {
			return core.ResumeSpec{}, fmt.Errorf("working directory %s no longer exists; restore it or resume by hand", spec.Dir)
		}
	}
	return spec, nil
}

// Exec replaces the current process with the resume command, run from the
// spec's directory. It returns only if the command could not be started.
func Exec(spec core.ResumeSpec) error {
	if len(spec.Argv) == 0 {
		return fmt.Errorf("empty resume command")
	}
	path, err := exec.LookPath(spec.Argv[0])
	if err != nil {
		return fmt.Errorf("find %s: %w", spec.Argv[0], err)
	}
	if spec.Dir != "" {
		if err := os.Chdir(spec.Dir); err != nil {
			return fmt.Errorf("change to %s: %w", spec.Dir, err)
		}
	}
	return syscall.Exec(path, spec.Argv, os.Environ()) //nolint:gosec // argv comes from the session's own reader
}
