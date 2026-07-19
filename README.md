# OmniDevX

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Docs][docs-mkdoc-svg]][docs-mkdoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/omnidevx-core/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omnidevx-core/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omnidevx-core/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omnidevx-core/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/omnidevx-core/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/omnidevx-core/actions/workflows/go-sast-codeql.yaml
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omnidevx-core
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omnidevx-core
 [docs-mkdoc-svg]: https://img.shields.io/badge/Go-dev%20guide-blue.svg
 [docs-mkdoc-url]: https://plexusone.dev/omnidevx-core
 [viz-svg]: https://img.shields.io/badge/Go-visualizaton-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fomnidevx-core
 [loc-svg]: https://tokei.rs/b1/github/plexusone/omnidevx-core
 [repo-url]: https://github.com/plexusone/omnidevx-core
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omnidevx-core/blob/main/LICENSE

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
