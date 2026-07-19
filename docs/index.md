# OmniDevX

Batteries-included distribution of **OmniDevX** — the PlexusOne
developer-experience telemetry domain. One import path composes the
canonical event model from
[`omnidevx-core`](https://github.com/plexusone/omnidevx-core) with every
available collector.

!!! note "Not OmniDXI"
    [`omnidxi`](https://github.com/plexusone/omnidxi) is Digital Experience
    Intelligence — product-analytics platforms (Amplitude, Mixpanel, Heap,
    Pendo). OmniDevX observes how developers and AI agents build software.

## Overview

Composition is by constructor injection through an `Engine` — there is no
global registry. Every canonical type, event/mode/severity constant, and
provider constructor from `omnidevx-core` and the thick provider modules is
re-exported under this one import path.

## Quick Start

=== "Default (Claude Code + Codex)"

    ```go
    import "github.com/plexusone/omnidevx"

    engine, err := omnidevx.NewDefault() // local Claude Code + Codex stores
    if err != nil {
        // handle
    }
    results, err := engine.Collect(ctx, omnidevx.CollectRequest{
        Period:  omnidevx.Period{Start: weekStart, End: weekEnd},
        Subject: omnidevx.SubjectRef{PersonID: "person:jane"},
    })
    // One collector failing does not discard the others' results: inspect both.
    events := omnidevx.Events(results)
    ```

=== "Explicit composition"

    ```go
    claude, _ := omnidevx.NewClaudeCodeCollector(omnidevx.ClaudeCodeOptions{})
    codex, _ := omnidevx.NewCodexCollector(omnidevx.CodexConfig{})
    git, _ := omnidevx.NewGitCollector(omnidevx.GitOptions{Roots: []string{"~/go/src"}})
    gh, _ := omnidevx.NewGitHubCollector(omnidevx.GitHubConfig{
        Token:    os.Getenv("GITHUB_TOKEN"),
        Username: "octocat",
    })
    engine := omnidevx.New(claude, codex, git, gh)
    ```

See [Collectors](collectors.md) for the full list and what each one needs.

## Privacy

Events carry metadata only — never prompt text, model responses, or file
contents. See [omnidevx-core](https://github.com/plexusone/omnidevx-core)
for the canonical contract.

## Installation

```bash
go get github.com/plexusone/omnidevx
```

## Requirements

- Go 1.26 or later

## Related Projects

- [omnidevx-core](https://github.com/plexusone/omnidevx-core) - Canonical event model and contracts
- [omni-github](https://github.com/plexusone/omni-github) - GitHub DevX collector
- [omni-openai](https://github.com/plexusone/omni-openai) - Codex CLI collector
- [omnidxi](https://github.com/plexusone/omnidxi) - Digital Experience Intelligence (not OmniDevX)
