# ROADMAP — Session Recovery — Discover, Understand, and Resume Coding-Agent Sessions

**Initiative:** `INIT-OMNIDEVX-001`
**Repository:** `github.com/plexusone/omnidevx`

RMIs span three repositories: `RMI-OMNIDEVXCORE-*` in
`github.com/plexusone/omnidevx-core`, `RMI-OMNIOPENAI-*` in
`github.com/plexusone/omni-openai`, and `RMI-OMNIDEVX-*` in this repository.

## Phase 1 — Session Contract & Readers

**Theme:** Establish the provider-neutral session model and read real Claude Code and Codex CLI sessions with correct timestamps, titles, and live state.

- [ ] `RMI-OMNIDEVXCORE-004` Session types, Reader interface, ResumeSpec, and content-access contract
- [ ] `RMI-OMNIDEVXCORE-005` Generated JSON Schema for the session contract (camelCase, embedded)
  - Depends on: `RMI-OMNIDEVXCORE-004`
- [ ] `RMI-OMNIDEVXCORE-006` Claude Code session reader with human-prompt classification, harness titles, and cached head/tail reads
  - Depends on: `RMI-OMNIDEVXCORE-004`
- [ ] `RMI-OMNIDEVXCORE-007` Claude Code live-session detection from per-process session records
  - Depends on: `RMI-OMNIDEVXCORE-006`
- [ ] `RMI-OMNIOPENAI-001` Codex CLI session reader from the threads index and rollout tail
  - Depends on: `RMI-OMNIDEVXCORE-004`

## Phase 2 — Catalog CLI: List, Show, Resume

**Theme:** Deliver the V1 acceptance test — after a reboot, recognize yesterday's sessions and resume one.

- [ ] `RMI-OMNIDEVX-001` Migrate the CLI to Cobra, preserving collect and version behavior
- [ ] `RMI-OMNIDEVX-002` Session catalog: reader composition, merged ordering, prefix ID resolution, and shared reader contract test
  - Depends on: `RMI-OMNIDEVXCORE-006`
  - Depends on: `RMI-OMNIOPENAI-001`
- [ ] `RMI-OMNIDEVX-003` omnidevx sessions list with filters, --json, and --no-content
  - Depends on: `RMI-OMNIDEVX-001`
  - Depends on: `RMI-OMNIDEVX-002`
- [ ] `RMI-OMNIDEVX-004` omnidevx sessions show for a single session
  - Depends on: `RMI-OMNIDEVX-003`
- [ ] `RMI-OMNIDEVX-005` omnidevx sessions resume with exec, --print, and running/missing-directory guards
  - Depends on: `RMI-OMNIDEVX-001`
  - Depends on: `RMI-OMNIDEVX-002`
- [ ] `RMI-OMNIDEVX-006` V1 acceptance on real sessions after reboot, sessions usage guide, and release
  - Depends on: `RMI-OMNIDEVX-004`
  - Depends on: `RMI-OMNIDEVX-005`

## Phase 3 — Deterministic Evidence

**Theme:** Ground each session in facts — repositories read and modified, commits created and referenced, and work references — without any model calls.

- [ ] `RMI-OMNIDEVXCORE-008` Repository index with workspace roots, longest-prefix path resolution, and remote-derived IDs
  - Depends on: `RMI-OMNIDEVXCORE-004`
- [ ] `RMI-OMNIDEVXCORE-009` Work-reference extraction rules and resolver interface
  - Depends on: `RMI-OMNIDEVXCORE-004`
- [ ] `RMI-OMNIDEVXCORE-010` Claude Code evidence: file and repository activity, created and referenced commits, shared git resolver
  - Depends on: `RMI-OMNIDEVXCORE-006`
  - Depends on: `RMI-OMNIDEVXCORE-008`
- [ ] `RMI-OMNIOPENAI-002` Codex CLI evidence: file and repository activity and created commits
  - Depends on: `RMI-OMNIOPENAI-001`
  - Depends on: `RMI-OMNIDEVXCORE-008`
- [ ] `RMI-OMNIDEVX-007` Surface evidence in sessions show and list, including --repo filtering
  - Depends on: `RMI-OMNIDEVX-004`
  - Depends on: `RMI-OMNIDEVXCORE-009`
  - Depends on: `RMI-OMNIDEVXCORE-010`
  - Depends on: `RMI-OMNIOPENAI-002`

## Phase 4 — Runtime Binding & tmux Continuity

**Theme:** Know which sessions are live and where, and resume into the tmux session name the developer recognizes.

- [ ] `RMI-OMNIDEVX-008` Spike: Codex CLI live-process detection
- [ ] `RMI-OMNIDEVX-009` tmux pane discovery and process-ancestry binding to harness sessions
  - Depends on: `RMI-OMNIDEVXCORE-007`
  - Depends on: `RMI-OMNIDEVX-002`
- [ ] `RMI-OMNIDEVX-010` Binding store for preferred and last-observed tmux names
  - Depends on: `RMI-OMNIDEVX-009`
- [ ] `RMI-OMNIDEVX-011` resume --tmux: recreate the preferred name with collision handling and client switching
  - Depends on: `RMI-OMNIDEVX-005`
  - Depends on: `RMI-OMNIDEVX-010`
- [ ] `RMI-OMNIDEVX-012` Running, resumable, and unknown state with tmux targets in list and show
  - Depends on: `RMI-OMNIDEVX-009`

## Phase 5 — Semantic Layer & Extensions

**Theme:** Explain sessions on top of the evidence, make them searchable, and evaluate further harnesses.

- [ ] `RMI-OMNIDEVX-013` Summarizer interface, bounded evidence package, and summary cache
  - Depends on: `RMI-OMNIDEVX-007`
- [ ] `RMI-OMNIDEVX-014` omnidevx sessions summarize with a configurable model provider
  - Depends on: `RMI-OMNIDEVX-013`
- [ ] `RMI-OMNIDEVX-015` omnidevx sessions search over titles, prompts, and evidence
  - Depends on: `RMI-OMNIDEVX-007`
- [ ] `RMI-OMNIDEVX-016` Spike: Kiro CLI session discovery and resume feasibility
- [ ] `RMI-OMNIDEVX-017` Sessions documentation, JSON contract reference, and release
  - Depends on: `RMI-OMNIDEVX-011`
  - Depends on: `RMI-OMNIDEVX-012`
  - Depends on: `RMI-OMNIDEVX-014`
  - Depends on: `RMI-OMNIDEVX-015`
