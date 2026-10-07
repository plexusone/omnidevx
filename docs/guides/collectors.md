# Collectors

`omnidevx` re-exports a constructor for every collector available across the
ecosystem. `NewDefault` composes the local AI-assistant collectors that can
use default on-disk stores; the rest need explicit setup and are added with
`Engine.Add`.

| Source | Package | Location | Setup |
|--------|---------|----------|-------|
| Claude Code | `providers/claudecode` | omnidevx-core (thin, stdlib) | none — reads local session store |
| Codex CLI | `omni-openai/omnidevx` | omni-openai (thick, SQLite) | none — reads local `~/.codex` store |
| Kiro CLI | `omni-aws/omnidevx` | omni-aws (thick, SQLite) | none — reads local Kiro CLI database and optional `~/.kiro_sessions` archives |
| Git | `providers/git` | omnidevx-core (thin, stdlib via [`gogit`](https://github.com/grokify/gogit)) | `Roots []string` — repository roots to scan |
| GitHub | `omni-github/omnidevx` | omni-github (thick, REST + GraphQL) | `Token`, `Username` |

## Claude Code, Codex CLI, and Kiro CLI

Included in `NewDefault()`:

```go
engine, err := omnidevx.NewDefault()
```

Or individually:

```go
claude, err := omnidevx.NewClaudeCodeCollector(omnidevx.ClaudeCodeOptions{})
codex, err := omnidevx.NewCodexCollector(omnidevx.CodexConfig{})
kiro, err := omnidevx.NewKiroCollector(omnidevx.KiroConfig{})
```

Kiro local records do not always expose exact token accounting. The Kiro
collector emits historical-estimate usage events with reduced confidence
when it must reconstruct tokens from local conversation history.

The provider-specific packages remain the source of truth for local schema
handling and path drift. `omnidevx` verifies that each provider satisfies
the shared collector contract and gives applications one import path for
the combined bundle.

## Git

Emits `devx.change.committed` events, one per commit, with AI co-author
attribution ported from the canonical `KnownAITools` registry. Needs
explicit repository roots — there is no default scan scope:

```go
git, err := omnidevx.NewGitCollector(omnidevx.GitOptions{
    Roots: []string{"~/go/src"},
})
```

## GitHub

Emits `devx.profile.snapshot`, `devx.contribution.snapshot` (per
repository), and `devx.contribution.recorded` (daily calendar) events via
GitHub's REST and GraphQL APIs:

```go
gh, err := omnidevx.NewGitHubCollector(omnidevx.GitHubConfig{
    Token:    os.Getenv("GITHUB_TOKEN"),
    Username: "octocat",
})
```

The GitHub collector requires a bounded `Period` (both `Start` and `End`
set) — the contribution query needs an explicit range.

## Composing All Five

```go
engine := omnidevx.New(claude, codex, kiro).Add(git, gh)

results, err := engine.Collect(ctx, omnidevx.CollectRequest{
    Period:  omnidevx.Period{Start: weekStart, End: weekEnd},
    Subject: omnidevx.SubjectRef{PersonID: "person:jane"},
})
```

One collector failing does not stop the others: `Collect` always returns
whatever results succeeded, alongside a joined error naming each failing
source.
