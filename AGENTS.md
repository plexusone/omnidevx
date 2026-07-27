# AGENTS.md — omnidevx

OmniDevX is the batteries-included distribution of the OmniDevX
developer-experience telemetry domain. It re-exports canonical types from
`omnidevx-core` and composes available collectors behind one import path.

## Conventions

- **Go module:** `github.com/plexusone/omnidevx`
- **Facade:** composes collectors from `omnidevx-core` and provider packages into a single engine
- **Default local bundle:** Claude Code, Codex CLI, and Kiro CLI
- **Explicit collectors:** Git and GitHub need explicit roots or credentials
- **Not OmniDXI:** OmniDXI (`omnidxi`) is Digital Experience Intelligence for product analytics; OmniDevX is developer and coding-agent telemetry

## Provider Boundaries

- Keep canonical event types, contracts, and thin stdlib collectors in
  `github.com/plexusone/omnidevx-core`.
- Keep OpenAI/Codex CLI storage handling in
  `github.com/plexusone/omni-openai/omnidevx`.
- Keep AWS/Kiro CLI storage handling in
  `github.com/plexusone/omni-aws/omnidevx`.
- Keep GitHub API handling in `github.com/plexusone/omni-github/omnidevx`.
- This repo should re-export provider constructors, compose defaults, and
  test that each provider satisfies the shared collector contract.

## Release Maintenance

- Update `CHANGELOG.json` for every release and regenerate `CHANGELOG.md`
  with `schangelog generate CHANGELOG.json -o CHANGELOG.md`.
- Use `schangelog parse-commits --since <previous-tag>` to review commits
  before writing release entries.
- Add a matching release note under `docs/releases/` and wire it into
  `mkdocs.yml`.
- Run `go test ./...` and `mkdocs build --strict` before committing release
  documentation.

## PRISM Control

This repo is registered in [prism-control](https://github.com/ProductBuildersHQ/prism-control). Use `prismctl work ready --repo github.com/plexusone/omnidevx` to find claimable work, and carry the `Refs: RMI-OMNIDEVX-<NNN>` trailer on every commit.
