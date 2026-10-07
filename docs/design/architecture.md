# Architecture

How OmniDevX is structured today: the modules, the data flow from harness
storage to reports, and the contracts each layer keeps. This page tracks
current behavior; for planned changes see the [specs](#related-specs).

## Module Layout

OmniDevX is split across modules so that heavy dependencies (SQLite, the
GitHub API client) stay out of the core, and provider-specific storage
handling lives with the provider that owns it.

| Module | Role | Contents |
|--------|------|----------|
| `github.com/plexusone/omnidevx-core` | Canonical IR and thin collectors | `Event` model, `Collector` contract, `store`, `report`, `identity`; stdlib-only Claude Code and Git collectors; OpenTelemetry receiver |
| `github.com/plexusone/omni-openai/omnidevx` | Codex CLI collector | Reads `~/.codex` SQLite state and rollout JSONL |
| `github.com/plexusone/omni-aws/omnidevx` | Kiro CLI collector | Reads the Kiro CLI database and optional session archives |
| `github.com/plexusone/omni-github/omnidevx` | GitHub collector | REST and GraphQL contribution data |
| `github.com/plexusone/omnidevx` (this repo) | Batteries-included distribution | Re-exports core types and provider constructors, `Engine`, `NewDefault`, the `omnidevx` CLI |

Boundary rules:

- Canonical types, contracts, and stdlib-only collectors belong in
  `omnidevx-core`.
- Storage-format handling for a vendor's tool belongs in that vendor's
  provider module, which is the source of truth for its schema drift.
- This repo composes; it does not parse harness storage itself.

## Data Flow

```text
 Harness storage / APIs                Collectors                    Canonical IR
 ──────────────────────                ──────────                    ────────────
 ~/.claude/projects/*.jsonl  ──→  claudecode (core)       ──┐
 ~/.codex/state_N.sqlite     ──→  omni-openai/omnidevx    ──┤
 ~/.codex/sessions/…jsonl                                   │
 Kiro CLI database           ──→  omni-aws/omnidevx       ──┼─→ []Event ─→ store ─→ report
 git repositories            ──→  git (core)              ──┤   (metadata   (JSONL)   (daily,
 GitHub API                  ──→  omni-github/omnidevx    ──┘    only)                 period)
```

`Engine` runs the collectors; the CLI's `collect` command writes their
events to the store. Reporting reads events back from the store.

## Collector Contract

Every source implements `Collector` from `omnidevx-core`:

```go
type Collector interface {
    Source() Source
    Collect(ctx context.Context, req CollectRequest) (*CollectionResult, error)
}
```

- A `CollectRequest` carries a half-open `Period` and an optional
  `SubjectRef`, which the collector stamps on every event.
- Collectors **normalize only**. They convert provider-native records into
  canonical events and never compute framework metrics.
- Damaged or unparseable records become `Diagnostic` entries in the result
  rather than being dropped silently. Only failures that prevent reading
  the source at all are returned as errors.

## Event Model

`Event` is the canonical observation:

| Field | Meaning |
|-------|---------|
| `ID` | Deterministic per source record, so re-collecting the same history deduplicates |
| `Type` | `ai.*` for coding-agent activity (sessions, prompts, messages, tools, patches, usage); `devx.*` for development work (commits, contributions, profiles) |
| `Timestamp` | When the activity happened |
| `Subject` | Whose activity it is (`PersonID`, plus local account and device for later identity resolution) |
| `Source` | Canonical provider and product names (e.g. `anthropic` / `claude-code`) |
| `Context` | Session ID, repository, workspace, git branch |
| `Attributes` | Metadata keyed by shared `Attr*` constants (model, token counts, tool, duration, commit stats, cost) |
| `Provenance` | Collection mode (`history`, `otel`, `hooks`, `api`, `survey`) and confidence in [0, 1] |

Providers use the shared attribute keys rather than ad-hoc names so
metrics can be computed across sources. Telemetry JSON uses snake_case
attribute keys; the event envelope uses camelCase field names.

## Privacy Contract

Canonical events carry **metadata only**: event types, timestamps,
durations, counts, models, token figures, and repository identifiers.
Prompt text, model responses, and file contents are never captured. Every
collector, including those in provider modules, must preserve this.

## Event Store

`omnidevx-core/store` persists events as daily JSONL files:

```text
~/.plexusone/omnidevx/data/events/YYYY/MM/DD/<product>.jsonl
```

- **Inspectable:** plain text, so a developer can read exactly what has been
  recorded about them.
- **Reprocessable:** reports are recomputed from stored events when metric
  formulas change.
- **Idempotent:** an event whose ID already exists in its day file is
  skipped, so overlapping collection runs are safe.
- **Private:** files are created with owner-only permissions.
- **Damage-tolerant:** unreadable lines surface as diagnostics on read.

## Reports

`omnidevx-core/report` builds daily summaries from events and rolls them up
into a per-developer period report (metrics, source coverage, data
quality). Token cost is estimated from an embedded, versioned model pricing
table when a source does not report cost directly.

## Engine

`Engine` composes collectors by constructor injection; there is no global
registry.

- `New(...)` and `Add(...)` assemble the collector set.
- `NewDefault()` builds the collectors that work with no configuration:
  Claude Code, Codex CLI, and Kiro CLI, reading their default local stores.
  Git (needs repository roots) and GitHub (needs credentials) are added
  explicitly.
- `Collect` runs every collector with the same request. One collector
  failing does not stop the others: successful results are returned
  together with a joined error naming each failing source.
- `Events` flattens results into one slice, preserving collector order.

This repo's tests check engine composition, failure isolation, and context
cancellation, and run the Claude Code, Codex CLI, and Kiro CLI constructors
against empty stores. Claude Code reports an error when its projects
directory is missing; Codex and Kiro return empty results.

## CLI

`cmd/omnidevx` is a thin adapter over the library:

- `omnidevx collect --person <id> --since <date> --until <date>` runs
  `NewDefault()` over the period and writes events to the store
  (`--store` overrides the root; `--dry-run` skips the write).
- `omnidevx version` prints the version.

## Adding a Collector

1. Implement `Collector` in the module that owns the source: `omnidevx-core`
   if it needs only the standard library, otherwise the vendor's provider
   module.
2. Emit canonical event types and shared attribute keys, with deterministic
   IDs and honest `Provenance`.
3. Re-export the constructor and options from `omnidevx.go`, and add it to
   `NewDefault()` only if it works without configuration.
4. Extend the constructor test in `engine_test.go` and document it in
   [Collectors](../guides/collectors.md).

## Related Specs

- [Session Recovery](../specs/initiatives/INIT-OMNIDEVX-001/PRD.md)
  (`INIT-OMNIDEVX-001`, planned): adds a session catalog that reads
  harness sessions under a separate content-access contract, alongside the
  metadata-only event stream described here.
