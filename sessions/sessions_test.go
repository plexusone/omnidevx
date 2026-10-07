package sessions

import (
	"strings"
	"testing"
	"time"

	core "github.com/plexusone/omnidevx-core/sessions"
)

func TestFilterApply(t *testing.T) {
	in := []core.Session{
		{ID: "1", Harness: core.HarnessClaudeCode, State: core.StateRunning, CWD: "/Users/example/src/app"},
		{ID: "2", Harness: core.HarnessCodex, State: core.StateUnknown, CWD: "/Users/example/src/app/sub"},
		{ID: "3", Harness: core.HarnessClaudeCode, State: core.StateResumable, CWD: "/Users/example/src/app-other"},
		{ID: "4", Harness: core.HarnessCodex, State: core.StateResumable, CWD: "/Users/example/src/lib"},
	}
	ids := func(ss []core.Session) string {
		var out []string
		for _, s := range ss {
			out = append(out, s.ID)
		}
		return strings.Join(out, ",")
	}
	tests := []struct {
		name string
		f    Filter
		want string
	}{
		{"empty keeps all", Filter{}, "1,2,3,4"},
		{"harness", Filter{Harness: core.HarnessCodex}, "2,4"},
		{"state", Filter{State: core.StateResumable}, "3,4"},
		{"cwd includes the directory and below, not siblings with a shared prefix",
			Filter{CWD: "/Users/example/src/app"}, "1,2"},
		{"cwd trailing slash is cleaned", Filter{CWD: "/Users/example/src/app/"}, "1,2"},
		{"combined", Filter{Harness: core.HarnessCodex, CWD: "/Users/example/src"}, "2,4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ids(tt.f.Apply(in)); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestParseSince(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		in   string
		want time.Time
		err  bool
	}{
		{"90m", now.Add(-90 * time.Minute), false},
		{"24h", now.Add(-24 * time.Hour), false},
		{"7d", now.AddDate(0, 0, -7), false},
		{" 2d ", now.AddDate(0, 0, -2), false},
		{"soon", time.Time{}, true},
		{"-1h", time.Time{}, true},
		{"d", time.Time{}, true},
		{"", time.Time{}, true},
	}
	for _, tt := range tests {
		got, err := ParseSince(tt.in, now)
		if tt.err {
			if err == nil {
				t.Errorf("ParseSince(%q) = %v, want error", tt.in, got)
			}
			continue
		}
		if err != nil || !got.Equal(tt.want) {
			t.Errorf("ParseSince(%q) = %v, %v; want %v", tt.in, got, err, tt.want)
		}
	}
}

func TestPrepare(t *testing.T) {
	dir := t.TempDir()
	ok := core.Session{ID: "abcd1234", State: core.StateResumable,
		Resume: core.ResumeSpec{Argv: []string{"claude", "--resume", "abcd1234"}, Dir: dir}}

	t.Run("resumable session", func(t *testing.T) {
		spec, err := Prepare(ok, false)
		if err != nil {
			t.Fatal(err)
		}
		if spec.Dir != dir || spec.Argv[0] != "claude" {
			t.Fatalf("spec = %+v", spec)
		}
	})

	t.Run("unknown state is allowed", func(t *testing.T) {
		s := ok
		s.State = core.StateUnknown
		if _, err := Prepare(s, false); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("running is refused with its pid", func(t *testing.T) {
		s := ok
		s.State, s.Runtime = core.StateRunning, &core.Runtime{PID: 4242}
		_, err := Prepare(s, false)
		if err == nil || !strings.Contains(err.Error(), "4242") || !strings.Contains(err.Error(), "--force") {
			t.Fatalf("err = %v, want refusal naming pid and --force", err)
		}
	})

	t.Run("force overrides running", func(t *testing.T) {
		s := ok
		s.State = core.StateRunning
		if _, err := Prepare(s, true); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("missing directory", func(t *testing.T) {
		s := ok
		s.Resume.Dir = dir + "/gone"
		_, err := Prepare(s, false)
		if err == nil || !strings.Contains(err.Error(), "no longer exists") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("no command", func(t *testing.T) {
		s := ok
		s.Resume.Argv = nil
		if _, err := Prepare(s, false); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestExecRejectsBadSpecs(t *testing.T) {
	if err := Exec(core.ResumeSpec{}); err == nil {
		t.Fatal("expected error for empty spec")
	}
	err := Exec(core.ResumeSpec{Argv: []string{"definitely-not-a-real-command-xyz"}})
	if err == nil || !strings.Contains(err.Error(), "find definitely-not-a-real-command-xyz") {
		t.Fatalf("err = %v", err)
	}
}
