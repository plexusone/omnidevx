package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	core "github.com/plexusone/omnidevx-core/sessions"
	"github.com/plexusone/omnidevx/sessions"
)

// deps are the CLI's external dependencies, replaceable in tests.
type deps struct {
	catalog func() (*core.Catalog, error)
	now     func() time.Time
	home    string
	exec    func(core.ResumeSpec) error
}

func defaultDeps() deps {
	home, _ := os.UserHomeDir() //nolint:errcheck // an empty home only disables ~ shortening
	return deps{
		catalog: sessions.NewDefaultCatalog,
		now:     time.Now,
		home:    home,
		exec:    sessions.Exec,
	}
}

// listFlags are the options of "sessions list" (also the default command).
type listFlags struct {
	since     string
	cwd       string
	harness   string
	state     string
	limit     int
	all       bool
	noContent bool
	json      bool
}

func (f *listFlags) bind(cmd *cobra.Command) {
	fl := cmd.Flags()
	fl.StringVar(&f.since, "since", "", "only sessions active within this long, e.g. 90m, 24h, 7d")
	fl.StringVar(&f.cwd, "cwd", "", "only sessions started in this directory or below (use . for the current directory)")
	fl.StringVar(&f.harness, "harness", "", "only this harness: claude-code or codex")
	fl.StringVar(&f.state, "state", "", "only this state: running, resumable, or unknown")
	fl.IntVarP(&f.limit, "limit", "n", 0, "show at most this many sessions (0 for all)")
	fl.BoolVar(&f.all, "all", false, "include archived sessions")
	fl.BoolVar(&f.noContent, "no-content", false, "omit prompt-derived text (titles fall back to the directory name)")
	fl.BoolVar(&f.json, "json", false, "print JSON")
}

func newSessionsCmd(d deps) *cobra.Command {
	var lf listFlags
	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "List, inspect, and resume coding-agent sessions",
		Long: `List the Claude Code and Codex CLI sessions on this machine, newest first,
with enough context to recognize each one, and resume the one you want.

Session titles and prompts are read from local harness storage on demand and
shown here; they are not written to the telemetry event store.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return runList(cmd, d, lf) },
	}
	lf.bind(cmd)

	var lf2 listFlags
	list := &cobra.Command{
		Use:   "list",
		Short: "List sessions (the default)",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return runList(cmd, d, lf2) },
	}
	lf2.bind(list)

	var showJSON, showNoContent bool
	show := &cobra.Command{
		Use:   "show <id>",
		Short: "Show one session in detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(cmd, d, args[0], showJSON, showNoContent)
		},
	}
	show.Flags().BoolVar(&showJSON, "json", false, "print JSON")
	show.Flags().BoolVar(&showNoContent, "no-content", false, "omit prompt-derived text")

	var printOnly, force bool
	resume := &cobra.Command{
		Use:   "resume <id>",
		Short: "Resume a session in its original directory",
		Long: `Resume a session with its harness's own resume command, run from the
directory the session started in. By default this replaces the omnidevx
process, so you land directly in the agent.

A session that is already running is refused unless --force is given.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResume(cmd, d, args[0], printOnly, force)
		},
	}
	resume.Flags().BoolVar(&printOnly, "print", false, "print the shell command instead of running it")
	resume.Flags().BoolVar(&force, "force", false, "resume even if the session appears to be running")

	cmd.AddCommand(list, show, resume)
	return cmd
}

// loadSessions lists sessions through the catalog. Partial failures (one
// harness unreadable, damaged files) are reported on stderr and do not hide
// the sessions that were read; they are an error only if nothing was read.
func loadSessions(cmd *cobra.Command, d deps, opts core.ListOptions) ([]core.Session, error) {
	cat, err := d.catalog()
	if err != nil {
		return nil, fmt.Errorf("open session stores: %w", err)
	}
	all, diags, err := cat.List(cmd.Context(), opts)
	if err != nil {
		if len(all) == 0 {
			return nil, err
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", err)
	}
	if len(diags) > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %d session file(s) could not be read\n", len(diags))
	}
	return all, nil
}

func runList(cmd *cobra.Command, d deps, f listFlags) error {
	opts := core.ListOptions{IncludeArchived: f.all, NoContent: f.noContent}
	if f.since != "" {
		since, err := sessions.ParseSince(f.since, d.now())
		if err != nil {
			return fmt.Errorf("--since: %w", err)
		}
		opts.Since = since
	}
	filter := sessions.Filter{
		Harness: core.Harness(f.harness),
		State:   core.State(f.state),
	}
	if f.harness != "" && f.harness != string(core.HarnessClaudeCode) && f.harness != string(core.HarnessCodex) {
		return fmt.Errorf("--harness: unknown harness %q (want claude-code or codex)", f.harness)
	}
	if f.state != "" && f.state != string(core.StateRunning) && f.state != string(core.StateResumable) && f.state != string(core.StateUnknown) {
		return fmt.Errorf("--state: unknown state %q (want running, resumable, or unknown)", f.state)
	}
	if f.cwd != "" {
		abs, err := filepath.Abs(f.cwd)
		if err != nil {
			return fmt.Errorf("--cwd: %w", err)
		}
		filter.CWD = abs
	}

	all, err := loadSessions(cmd, d, opts)
	if err != nil {
		return err
	}
	out := filter.Apply(all)
	if f.limit > 0 && len(out) > f.limit {
		out = out[:f.limit]
	}

	w := cmd.OutOrStdout()
	if f.json {
		if out == nil {
			out = []core.Session{}
		}
		return writeJSON(w, out)
	}
	return writeTable(w, out, d)
}

func runShow(cmd *cobra.Command, d deps, query string, asJSON, noContent bool) error {
	all, err := loadSessions(cmd, d, core.ListOptions{IncludeArchived: true, NoContent: noContent})
	if err != nil {
		return err
	}
	s, err := core.Resolve(all, query)
	if err != nil {
		return err
	}
	if asJSON {
		return writeJSON(cmd.OutOrStdout(), s)
	}
	return writeDetail(cmd.OutOrStdout(), *s, d)
}

func runResume(cmd *cobra.Command, d deps, query string, printOnly, force bool) error {
	all, err := loadSessions(cmd, d, core.ListOptions{IncludeArchived: true})
	if err != nil {
		return err
	}
	s, err := core.Resolve(all, query)
	if err != nil {
		return err
	}
	spec, err := sessions.Prepare(*s, force)
	if err != nil {
		return err
	}
	if printOnly {
		fmt.Fprintln(cmd.OutOrStdout(), spec.Command())
		return nil
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Resuming %s session %s in %s\n", s.Harness, shortID(s.ID), shortPath(spec.Dir, d.home, 60))
	return d.exec(spec)
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeTable(w io.Writer, ss []core.Session, d deps) error {
	if len(ss) == 0 {
		_, err := fmt.Fprintln(w, "No sessions found.")
		return err
	}
	now := d.now()
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "HARNESS\tID\tCWD\tLAST HUMAN\tLAST ACTIVE\tSTATE\tTITLE")
	for _, s := range ss {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			harnessLabel(s.Harness), shortID(s.ID), shortPath(s.CWD, d.home, 34),
			age(s.LastHumanActivityAt, now), age(s.LastActivityAt, now), s.State, core.Truncate(s.Title, 60))
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "\n%d session(s). Details: omnidevx sessions show <id>   Resume: omnidevx sessions resume <id>\n", len(ss))
	return err
}

func writeDetail(w io.Writer, s core.Session, d deps) error {
	now := d.now()
	var b strings.Builder
	row := func(label, value string) {
		if value != "" {
			fmt.Fprintf(&b, "%-14s%s\n", label, value)
		}
	}
	fmt.Fprintf(&b, "%s session %s\n\n", harnessLabel(s.Harness), s.ID)
	row("Title", s.Title)
	row("State", string(s.State))
	if s.Runtime != nil && s.Runtime.PID > 0 {
		row("PID", fmt.Sprint(s.Runtime.PID))
	}
	row("Working dir", s.CWD)
	row("Branch", s.GitBranch)
	row("Remote", s.GitOrigin)
	row("Created", stamp(s.CreatedAt, now))
	row("Last active", stamp(s.LastActivityAt, now))
	row("Last human", stamp(s.LastHumanActivityAt, now))
	if s.Archived {
		row("Archived", "yes")
	}
	if len(s.RecentPrompts) > 0 {
		b.WriteString("\nRecent human prompts\n")
		for _, p := range s.RecentPrompts {
			fmt.Fprintf(&b, "  %s  %s\n", p.At.Local().Format("01-02 15:04"), core.Truncate(p.Text, 110))
		}
	}
	fmt.Fprintf(&b, "\nResume\n  %s\n", s.Resume.Command())
	_, err := io.WriteString(w, b.String())
	return err
}

func harnessLabel(h core.Harness) string {
	if h == core.HarnessClaudeCode {
		return "claude"
	}
	return string(h)
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// age renders how long before now t was: "-" for unknown, then m, h, d.
func age(t, now time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func stamp(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s (%s ago)", t.Local().Format("2006-01-02 15:04"), age(t, now))
}

// shortPath makes a working directory readable: the home directory becomes
// ~, a leading ~/go/src/ is dropped (the host/org/repo that follows is what
// identifies a project), and when width > 0 leading components are elided
// until it fits.
func shortPath(p, home string, width int) string {
	if home != "" && (p == home || strings.HasPrefix(p, home+"/")) {
		p = "~" + strings.TrimPrefix(p, home)
	}
	p = strings.TrimPrefix(p, "~/go/src/")
	if width <= 0 || len([]rune(p)) <= width {
		return p
	}
	parts := strings.Split(p, "/")
	for len(parts) > 1 {
		parts = parts[1:]
		if cand := "…/" + strings.Join(parts, "/"); len([]rune(cand)) <= width {
			return cand
		}
	}
	return "…/" + core.Truncate(parts[0], width-2)
}
