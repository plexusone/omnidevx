# OmniDevX

Batteries-included distribution of **OmniDevX** — the PlexusOne
developer-experience telemetry domain. One import path composes the
canonical event model from
[`omnidevx-core`](https://github.com/plexusone/omnidevx-core) with every
available collector.

> **Not OmniDXI.** [`omnidxi`](https://github.com/plexusone/omnidxi) is
> Digital Experience Intelligence — product-analytics platforms (Amplitude,
> Mixpanel, Heap, Pendo). OmniDevX observes how developers and AI agents
> build software.

## Collectors

| Source | Package | Location |
|--------|---------|----------|
| Claude Code | `providers/claudecode` | omnidevx-core (thin, stdlib) |
| Codex CLI | `omni-openai/omnidevx` | omni-openai (thick, SQLite) |

## Usage

```go
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

Or compose explicitly:

```go
claude, _ := omnidevx.NewClaudeCodeCollector(omnidevx.ClaudeCodeOptions{})
codex, _ := omnidevx.NewCodexCollector(omnidevx.CodexConfig{})
engine := omnidevx.New(claude, codex)
```

## Privacy

Events carry metadata only — never prompt text, model responses, or file
contents. See omnidevx-core for the canonical contract.

## License

MIT
