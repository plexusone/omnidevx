# Usage Guide

`omnidevx` is the batteries-included package for applications that want a
single import path for developer and AI-assistant telemetry collectors.

## Choosing the Bundle

Use `NewDefault` when the application should read local AI-assistant stores
from the current machine:

```go
engine, err := omnidevx.NewDefault()
```

The default bundle includes:

| Collector | Source | Notes |
|-----------|--------|-------|
| Claude Code | `omnidevx-core/providers/claudecode` | Reads Claude Code local session data |
| Codex CLI | `omni-openai/omnidevx` | Reads local Codex SQLite and rollout data |
| Kiro CLI | `omni-aws/omnidevx` | Reads local Kiro conversation data and may estimate usage |

Use explicit composition when the caller needs provider-specific
configuration or wants to add Git and GitHub:

```go
claude, _ := omnidevx.NewClaudeCodeCollector(omnidevx.ClaudeCodeOptions{})
codex, _ := omnidevx.NewCodexCollector(omnidevx.CodexConfig{})
kiro, _ := omnidevx.NewKiroCollector(omnidevx.KiroConfig{})
git, _ := omnidevx.NewGitCollector(omnidevx.GitOptions{Roots: []string{"~/go/src"}})
gh, _ := omnidevx.NewGitHubCollector(omnidevx.GitHubConfig{
    Token:    os.Getenv("GITHUB_TOKEN"),
    Username: "octocat",
})

engine := omnidevx.New(claude, codex, kiro).Add(git, gh)
```

## Handling Partial Results

Collectors are independent. `Engine.Collect` returns successful results even
when one provider fails, and joins provider errors so callers can decide
whether partial data is acceptable:

```go
results, err := engine.Collect(ctx, req)
events := omnidevx.Events(results)
if err != nil {
    // Log or surface the provider error without discarding successful events.
}
```

## Provider Boundaries

Thin collectors with minimal dependencies live in
`github.com/plexusone/omnidevx-core`. Thick collectors live in provider
modules that already own heavier SDK, API, or storage dependencies:

| Provider Module | Collector |
|-----------------|-----------|
| `github.com/plexusone/omni-openai` | Codex CLI |
| `github.com/plexusone/omni-aws` | Kiro CLI |
| `github.com/plexusone/omni-github` | GitHub |

This package imports those provider modules and re-exports their public
collector constructors so consumers can depend on `omnidevx` directly.
