package omnidevx

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite" // fixture state database for the Codex reader
)

var contractBase = time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)

// claudeFixture writes one Claude Code session under a temporary home.
func claudeFixture(t *testing.T) SessionReader {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, "projects", "-Users-example-src-app")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	rec := func(minutes int, extra map[string]any) string {
		m := map[string]any{
			"type": "user", "timestamp": contractBase.Add(time.Duration(minutes) * time.Minute).Format(time.RFC3339),
			"cwd":     "/Users/example/src/app",
			"message": map[string]any{"role": "user", "content": "build the thing"}, "promptSource": "typed",
		}
		for k, v := range extra {
			m[k] = v
		}
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	lines := rec(0, nil) + "\n" + rec(30, map[string]any{"type": "assistant", "promptSource": "", "message": map[string]any{"role": "assistant", "content": "ok"}}) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "11111111-aaaa-bbbb-cccc-000000000001.jsonl"), []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := NewClaudeCodeSessionReader(ClaudeCodeOptions{Dir: home})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// codexFixture writes one Codex thread and rollout under a temporary home.
func codexFixture(t *testing.T) SessionReader {
	t.Helper()
	home := t.TempDir()
	rollout := filepath.Join(home, "rollout.jsonl")
	line, err := json.Marshal(map[string]any{
		"type": "response_item", "timestamp": contractBase.Add(time.Minute).Format(time.RFC3339),
		"payload": map[string]any{"type": "message", "role": "user",
			"content": []any{map[string]any{"type": "input_text", "text": "build the thing"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rollout, append(line, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(home, "state_5.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	}()
	if _, err := db.Exec(`CREATE TABLE threads (id TEXT PRIMARY KEY, rollout_path TEXT, created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL, cwd TEXT NOT NULL, first_user_message TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO threads VALUES ('0199aaaa-1111-7222-8333-444455556666', ?, ?, ?, '/Users/example/src/app', 'build the thing')`,
		rollout, contractBase.Unix(), contractBase.Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	r, err := NewCodexSessionReader(CodexConfig{Dir: home})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TestSessionReaderContract asserts the invariants every reader must keep so
// the catalog and CLI can treat harnesses uniformly.
func TestSessionReaderContract(t *testing.T) {
	readers := map[Harness]SessionReader{
		HarnessClaudeCode: claudeFixture(t),
		HarnessCodex:      codexFixture(t),
	}
	for harness, r := range readers {
		t.Run(string(harness), func(t *testing.T) {
			if r.Harness() != harness {
				t.Fatalf("Harness() = %s, want %s", r.Harness(), harness)
			}
			got, diags, err := r.List(context.Background(), SessionListOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if len(diags) != 0 || len(got) != 1 {
				t.Fatalf("sessions=%d diags=%+v, want exactly one clean session", len(got), diags)
			}
			s := got[0]
			if s.Harness != harness || s.ID == "" {
				t.Errorf("identity = %q/%q", s.Harness, s.ID)
			}
			if s.CWD == "" || s.Title == "" || s.TitleSource == "" {
				t.Errorf("cwd=%q title=%q source=%q: all must be set", s.CWD, s.Title, s.TitleSource)
			}
			if s.CreatedAt.IsZero() || s.LastActivityAt.Before(s.CreatedAt) {
				t.Errorf("created=%v lastActivity=%v", s.CreatedAt, s.LastActivityAt)
			}
			if !s.LastHumanActivityAt.IsZero() && s.LastHumanActivityAt.After(s.LastActivityAt) {
				t.Errorf("lastHuman %v is after lastActivity %v", s.LastHumanActivityAt, s.LastActivityAt)
			}
			switch s.State {
			case SessionResumable, SessionRunning, SessionUnknown:
			default:
				t.Errorf("state = %q", s.State)
			}
			if len(s.Resume.Argv) == 0 || s.Resume.Dir != s.CWD || !strings.Contains(strings.Join(s.Resume.Argv, " "), s.ID) {
				t.Errorf("Resume = %+v, want a command naming %s run from %s", s.Resume, s.ID, s.CWD)
			}

			// NoContent must remove prompt-derived text.
			got, _, err = r.List(context.Background(), SessionListOptions{NoContent: true})
			if err != nil {
				t.Fatal(err)
			}
			if len(got[0].RecentPrompts) != 0 || strings.Contains(got[0].Title, "build the thing") {
				t.Errorf("NoContent leaked prompt text: title=%q prompts=%+v", got[0].Title, got[0].RecentPrompts)
			}
		})
	}
}

func TestSessionCatalogOverBothHarnesses(t *testing.T) {
	cat := NewSessionCatalog(claudeFixture(t), codexFixture(t))
	all, _, err := cat.List(context.Background(), SessionListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d sessions, want 2", len(all))
	}
	// The Codex thread was updated an hour after the Claude session's last record.
	if all[0].Harness != HarnessCodex {
		t.Errorf("first = %s, want codex (newest activity)", all[0].Harness)
	}
	if _, err := ResolveSession(all, "codex:0199"); err != nil {
		t.Errorf("resolve by qualified prefix: %v", err)
	}
}
