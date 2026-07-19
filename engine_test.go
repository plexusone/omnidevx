package omnidevx

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	core "github.com/plexusone/omnidevx-core"
)

type fakeCollector struct {
	source core.Source
	events []core.Event
	err    error
}

func (f *fakeCollector) Source() core.Source { return f.source }

func (f *fakeCollector) Collect(_ context.Context, req core.CollectRequest) (*core.CollectionResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &core.CollectionResult{
		Source:      f.source,
		Subject:     req.Subject,
		Period:      req.Period,
		Events:      f.events,
		CollectedAt: time.Now().UTC(),
	}, nil
}

var _ core.Collector = (*fakeCollector)(nil)

func TestEngineCollectAll(t *testing.T) {
	a := &fakeCollector{
		source: core.Source{Provider: "anthropic", Product: "claude-code"},
		events: []core.Event{{ID: "a:1", Type: core.EventPromptSubmitted}},
	}
	b := &fakeCollector{
		source: core.Source{Provider: "openai", Product: "codex-cli"},
		events: []core.Event{{ID: "b:1", Type: core.EventToolCompleted}, {ID: "b:2", Type: core.EventTaskCompleted}},
	}

	results, err := New(a, b).Collect(context.Background(), core.CollectRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results: got %d, want 2", len(results))
	}
	events := Events(results)
	if len(events) != 3 {
		t.Errorf("flattened events: got %d, want 3", len(events))
	}
}

func TestEngineIsolatesFailures(t *testing.T) {
	sentinel := errors.New("store locked")
	failing := &fakeCollector{
		source: core.Source{Provider: "openai", Product: "codex-cli"},
		err:    sentinel,
	}
	working := &fakeCollector{
		source: core.Source{Provider: "anthropic", Product: "claude-code"},
		events: []core.Event{{ID: "a:1", Type: core.EventPromptSubmitted}},
	}

	results, err := New(failing, working).Collect(context.Background(), core.CollectRequest{})
	if err == nil {
		t.Fatal("expected joined error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("error should wrap the sentinel: %v", err)
	}
	if !strings.Contains(err.Error(), "openai/codex-cli") {
		t.Errorf("error should name the failing source: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("surviving results: got %d, want 1", len(results))
	}
	if results[0].Source.Product != "claude-code" {
		t.Errorf("surviving result: got %s", results[0].Source.Product)
	}
}

func TestEngineAddAndCollectors(t *testing.T) {
	a := &fakeCollector{source: core.Source{Provider: "x", Product: "a"}}
	b := &fakeCollector{source: core.Source{Provider: "x", Product: "b"}}
	e := New(a).Add(b)
	if got := len(e.Collectors()); got != 2 {
		t.Errorf("collectors: got %d, want 2", got)
	}
}

func TestEngineCanceledContext(t *testing.T) {
	a := &fakeCollector{source: core.Source{Provider: "x", Product: "a"}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New(a).Collect(ctx, core.CollectRequest{}); !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestProviderConstructors proves the re-exported constructors compose real
// providers into the engine against empty stores.
func TestProviderConstructors(t *testing.T) {
	claude, err := NewClaudeCodeCollector(ClaudeCodeOptions{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	cdx, err := NewCodexCollector(CodexConfig{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	engine := New(claude, cdx)

	// Claude's collector errors on a missing projects/ dir; Codex tolerates
	// an empty home. The engine must surface the former and keep the latter.
	results, err := engine.Collect(context.Background(), CollectRequest{})
	if err == nil {
		t.Fatal("expected error from claude collector on empty dir")
	}
	if !strings.Contains(err.Error(), "anthropic/claude-code") {
		t.Errorf("error should name claude collector: %v", err)
	}
	if len(results) != 1 || results[0].Source.Product != "codex-cli" {
		t.Fatalf("expected surviving codex result, got %+v", results)
	}
}
