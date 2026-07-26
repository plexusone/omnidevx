# CLAUDE.md — omnidevx

OmniDevX is the batteries-included distribution of the OmniDevX developer-experience telemetry domain. It re-exports canonical types from omnidevx-core and composes the available collectors (Claude Code, Codex CLI) behind one import path.

## Conventions

- **Go module:** `github.com/plexusone/omnidevx`
- **Facade:** composes collectors from omnidevx-core and provider packages into a single engine
- **Not OmniDXI:** OmniDXI (`omnidxi`) is Digital Experience Intelligence for product analytics; OmniDevX is developer and coding-agent telemetry

## PRISM Control

This repo is registered in [prism-control](https://github.com/ProductBuildersHQ/prism-control). Use `prismctl work ready --repo github.com/plexusone/omnidevx` to find claimable work, and carry the `Refs: RMI-OMNIDEVX-<NNN>` trailer on every commit.
