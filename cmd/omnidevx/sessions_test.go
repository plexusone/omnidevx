package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	omnidevxcore "github.com/plexusone/omnidevx-core"
	core "github.com/plexusone/omnidevx-core/sessions"
)

var now = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

type fakeReader struct {
	h  core.Harness
	ss []core.Session
}

func (f fakeReader) Harness() core.Harness { return f.h }

func (f fakeReader) List(_ context.Context, opts core.ListOptions) ([]core.Session, []omnidevxcore.Diagnostic, error) {
	var out []core.Session
	for _, s := range f.ss {
		if !opts.IncludeArchived && s.Archived {
			continue
		}
		if !opts.Since.IsZero() && s.LastActivityAt.Before(opts.Since) {
			continue
		}
		out = append(out, s)
	}
	return out, nil, nil
}

func fixture(t *testing.T) deps {
	t.Helper()
	dir := t.TempDir()
	mk := func(h core.Harness, id, cwd, title string, ago time.Duration, state core.State) core.Session {
		return core.Session{
			Harness: h, ID: id, CWD: cwd, Title: title, State: state,
			CreatedAt: now.Add(-ago - time.Hour), LastActivityAt: now.Add(-ago), LastHumanActivityAt: now.Add(-ago - time.Minute),
			Resume: core.ResumeSpec{Argv: []string{string(h), "resume", id}, Dir: dir},
		}
	}
	claude := fakeReader{core.HarnessClaudeCode, []core.Session{
		mk(core.HarnessClaudeCode, "aaaa1111-0000", "/Users/example/go/src/github.com/org/app", "App work", 5*time.Minute, core.StateRunning),
		mk(core.HarnessClaudeCode, "aaaa2222-0000", "/Users/example/go/src/github.com/org/lib", "Lib work", 30*time.Hour, core.StateResumable),
	}}
	codex := fakeReader{core.HarnessCodex, []core.Session{
		mk(core.HarnessCodex, "bbbb3333-0000", "/Users/example/go/src/github.com/org/app/sub", "Codex work", 2*time.Hour, core.StateUnknown),
		{Harness: core.HarnessCodex, ID: "bbbb4444-0000", CWD: dir, Title: "Old", Archived: true,
			LastActivityAt: now.Add(-100 * time.Hour), Resume: core.ResumeSpec{Argv: []string{"codex", "resume", "bbbb4444-0000"}, Dir: dir}},
	}}
	return deps{
		catalog: func() (*core.Catalog, error) { return core.NewCatalog(claude, codex), nil },
		now:     func() time.Time { return now },
		home:    "/Users/example",
		exec:    func(core.ResumeSpec) error { return errors.New("exec must not run in this test") },
	}
}

func run(t *testing.T, d deps, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := newRootCmd(d)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errb.String(), err
}

func TestSessionsListDefault(t *testing.T) {
	out, _, err := run(t, fixture(t), "sessions")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"HARNESS", "LAST HUMAN", "App work", "Lib work", "Codex work",
		"github.com/org/app", "running", "resumable", "unknown", "3 session(s)"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Old") {
		t.Errorf("archived session listed without --all:\n%s", out)
	}
	// Newest activity first.
	if strings.Index(out, "App work") > strings.Index(out, "Codex work") || strings.Index(out, "Codex work") > strings.Index(out, "Lib work") {
		t.Errorf("not newest first:\n%s", out)
	}
	if !strings.Contains(out, "aaaa1111") || strings.Contains(out, "aaaa1111-0000") {
		t.Errorf("IDs should be shortened to 8 characters:\n%s", out)
	}
}

func TestSessionsListFilters(t *testing.T) {
	d := fixture(t)
	tests := []struct {
		name string
		args []string
		want []string
		not  []string
	}{
		{"list subcommand", []string{"sessions", "list"}, []string{"App work", "Lib work"}, nil},
		{"since", []string{"sessions", "--since", "24h"}, []string{"App work", "Codex work"}, []string{"Lib work"}},
		{"harness", []string{"sessions", "--harness", "codex"}, []string{"Codex work"}, []string{"App work", "Lib work"}},
		{"state", []string{"sessions", "--state", "running"}, []string{"App work"}, []string{"Lib work", "Codex work"}},
		{"cwd", []string{"sessions", "--cwd", "/Users/example/go/src/github.com/org/app"}, []string{"App work", "Codex work"}, []string{"Lib work"}},
		{"limit", []string{"sessions", "-n", "1"}, []string{"App work"}, []string{"Lib work", "Codex work"}},
		{"all includes archived", []string{"sessions", "--all"}, []string{"Old"}, nil},
		{"no content", []string{"sessions", "--no-content"}, []string{"App work"}, nil}, // fake reader ignores NoContent
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := run(t, d, tt.args...)
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range tt.want {
				if !strings.Contains(out, w) {
					t.Errorf("missing %q:\n%s", w, out)
				}
			}
			for _, n := range tt.not {
				if strings.Contains(out, n) {
					t.Errorf("unexpected %q:\n%s", n, out)
				}
			}
		})
	}
}

func TestSessionsListFlagErrors(t *testing.T) {
	d := fixture(t)
	for args, want := range map[string]string{
		"--since soon":     "--since",
		"--harness vim":    "unknown harness",
		"--state sleeping": "unknown state",
	} {
		_, _, err := run(t, d, append([]string{"sessions"}, strings.Fields(args)...)...)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v, want containing %q", args, err, want)
		}
	}
}

func TestSessionsListJSON(t *testing.T) {
	out, _, err := run(t, fixture(t), "sessions", "--json", "--state", "running")
	if err != nil {
		t.Fatal(err)
	}
	var got []core.Session
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(got) != 1 || got[0].ID != "aaaa1111-0000" || got[0].State != core.StateRunning {
		t.Fatalf("got %+v", got)
	}

	out, _, err = run(t, fixture(t), "sessions", "--json", "--harness", "codex", "--state", "running")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("empty result = %q, want []", out)
	}
}

func TestSessionsListEmpty(t *testing.T) {
	d := fixture(t)
	out, _, err := run(t, d, "sessions", "--state", "running", "--harness", "codex")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No sessions found.") {
		t.Errorf("out = %q", out)
	}
}

func TestSessionsShow(t *testing.T) {
	d := fixture(t)
	out, _, err := run(t, d, "sessions", "show", "aaaa2222")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"claude session aaaa2222-0000", "Lib work", "/Users/example/go/src/github.com/org/lib", "Last human", "Resume", "claude-code resume aaaa2222-0000"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}

	out, _, err = run(t, d, "sessions", "show", "aaaa2222", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var s core.Session
	if err := json.Unmarshal([]byte(out), &s); err != nil || s.ID != "aaaa2222-0000" {
		t.Fatalf("json = %v, %+v", err, s)
	}

	if _, _, err := run(t, d, "sessions", "show", "aaaa"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("ambiguous prefix: err = %v", err)
	}
	if _, _, err := run(t, d, "sessions", "show", "ffff"); err == nil || !strings.Contains(err.Error(), "no session matches") {
		t.Errorf("unknown id: err = %v", err)
	}
}

func TestSessionsShowArchivedByID(t *testing.T) {
	out, _, err := run(t, fixture(t), "sessions", "show", "bbbb4444")
	if err != nil {
		t.Fatalf("an archived session must still be inspectable by ID: %v", err)
	}
	if !strings.Contains(out, "Archived") {
		t.Errorf("missing archived marker:\n%s", out)
	}
}

func TestSessionsResume(t *testing.T) {
	d := fixture(t)

	t.Run("print", func(t *testing.T) {
		out, _, err := run(t, d, "sessions", "resume", "aaaa2222", "--print")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(out, "cd ") || !strings.Contains(out, "&& claude-code resume aaaa2222-0000") {
			t.Errorf("out = %q", out)
		}
	})

	t.Run("exec receives the spec", func(t *testing.T) {
		var got core.ResumeSpec
		d := d
		d.exec = func(spec core.ResumeSpec) error { got = spec; return nil }
		_, stderr, err := run(t, d, "sessions", "resume", "bbbb3333")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(got.Argv, " ") != "codex resume bbbb3333-0000" {
			t.Errorf("exec spec = %+v", got)
		}
		if !strings.Contains(stderr, "Resuming codex session bbbb3333") {
			t.Errorf("stderr = %q", stderr)
		}
	})

	t.Run("running is refused without force", func(t *testing.T) {
		_, _, err := run(t, d, "sessions", "resume", "aaaa1111")
		if err == nil || !strings.Contains(err.Error(), "already running") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("force resumes a running session", func(t *testing.T) {
		out, _, err := run(t, d, "sessions", "resume", "aaaa1111", "--force", "--print")
		if err != nil || !strings.Contains(out, "aaaa1111-0000") {
			t.Fatalf("out=%q err=%v", out, err)
		}
	})

	t.Run("missing directory", func(t *testing.T) {
		d := d
		cat := fakeReader{core.HarnessClaudeCode, []core.Session{{
			Harness: core.HarnessClaudeCode, ID: "cccc5555-0000", CWD: "/gone", LastActivityAt: now,
			Resume: core.ResumeSpec{Argv: []string{"claude", "--resume", "cccc5555-0000"}, Dir: "/definitely/not/here"},
		}}}
		d.catalog = func() (*core.Catalog, error) { return core.NewCatalog(cat), nil }
		_, _, err := run(t, d, "sessions", "resume", "cccc5555")
		if err == nil || !strings.Contains(err.Error(), "no longer exists") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestSessionsPartialFailureStillLists(t *testing.T) {
	d := fixture(t)
	d.catalog = func() (*core.Catalog, error) {
		ok := fakeReader{core.HarnessClaudeCode, []core.Session{{
			Harness: core.HarnessClaudeCode, ID: "dddd6666-0000", Title: "Survivor", LastActivityAt: now,
		}}}
		return core.NewCatalog(ok, failingReader{}), nil
	}
	out, stderr, err := run(t, d, "sessions")
	if err != nil {
		t.Fatalf("a failing harness must not hide the others: %v", err)
	}
	if !strings.Contains(out, "Survivor") || !strings.Contains(stderr, "codex") {
		t.Errorf("out=%q stderr=%q", out, stderr)
	}

	d.catalog = func() (*core.Catalog, error) { return core.NewCatalog(failingReader{}), nil }
	if _, _, err := run(t, d, "sessions"); err == nil {
		t.Error("expected error when every harness fails")
	}
}

type failingReader struct{}

func (failingReader) Harness() core.Harness { return core.HarnessCodex }

func (failingReader) List(context.Context, core.ListOptions) ([]core.Session, []omnidevxcore.Diagnostic, error) {
	return nil, nil, errors.New("store unreadable")
}

func TestAgeAndShortPath(t *testing.T) {
	for _, c := range []struct {
		d    time.Duration
		want string
	}{{10 * time.Second, "now"}, {5 * time.Minute, "5m"}, {3 * time.Hour, "3h"}, {47 * time.Hour, "47h"}, {72 * time.Hour, "3d"}} {
		if got := age(now.Add(-c.d), now); got != c.want {
			t.Errorf("age(%v) = %s, want %s", c.d, got, c.want)
		}
	}
	if got := age(time.Time{}, now); got != "-" {
		t.Errorf("age(zero) = %s", got)
	}

	home := "/Users/example"
	for _, c := range []struct {
		in    string
		width int
		want  string
	}{
		{home + "/go/src/github.com/org/app", 34, "github.com/org/app"},
		{home + "/go/src/github.com/ProductBuildersHQ/betterstack-go", 34, "…/ProductBuildersHQ/betterstack-go"},
		{home + "/notes", 34, "~/notes"},
		{"/opt/elsewhere", 34, "/opt/elsewhere"},
		{home + "/go/src/github.com/org/app", 0, "github.com/org/app"},
	} {
		if got := shortPath(c.in, home, c.width); got != c.want {
			t.Errorf("shortPath(%q, %d) = %q, want %q", c.in, c.width, got, c.want)
		}
	}
}

func TestVersionAndCollectValidation(t *testing.T) {
	out, _, err := run(t, fixture(t), "version")
	if err != nil || strings.TrimSpace(out) != "omnidevx v0.1.0" {
		t.Fatalf("version out=%q err=%v", out, err)
	}
	_, _, err = run(t, fixture(t), "collect")
	if err == nil || !strings.Contains(err.Error(), "--person, --since, and --until are required") {
		t.Fatalf("collect without flags: err = %v", err)
	}
}
