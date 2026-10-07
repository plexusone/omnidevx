# PLAN — Session Recovery — Discover, Understand, and Resume Coding-Agent Sessions

**Initiative:** `INIT-OMNIDEVX-001`

## Sequencing Rationale

The plan is ordered so the core recovery loop — list, recognize, resume —
works end to end as early as possible, and every later phase improves
recognition rather than enabling it.

1. **Contract and readers first.** Harness storage formats are undocumented
   and change between versions, so the readers are built and verified
   against real files before anything depends on them. Native metadata
   (Claude `ai-title` records, Codex `threads` titles, Claude per-process
   records) makes the first catalog useful without heuristics or models.
2. **CLI second, ending in the V1 acceptance test.** Phase 2 ships a
   release that already solves the post-reboot problem.
3. **Evidence third.** Repositories, commits, and work references sharpen
   recognition when titles are ambiguous (several sessions in one repo).
4. **Runtime binding fourth.** tmux continuity matters for the long-running
   workflow but is optional by design, so it builds on a working resume.
5. **Semantics last.** Summaries and search are layered on top of the
   evidence they must be grounded in.

## Phases

| Phase | Outcome | Repos |
|---|---|---|
| 1 — Session Contract & Readers | Provider-neutral `Session` model; Claude Code and Codex readers with correct timestamps, titles, and live state | omnidevx-core, omni-openai |
| 2 — Catalog CLI | `sessions list/show/resume`, Cobra CLI, V1 release | omnidevx |
| 3 — Deterministic Evidence | Repos read/modified/mentioned, commits, work references | omnidevx-core, omni-openai, omnidevx |
| 4 — Runtime Binding & tmux | Running/resumable state, sticky tmux names, `resume --tmux` | omnidevx |
| 5 — Semantic Layer & Extensions | `summarize`, `search`, Kiro CLI evaluation | omnidevx |

## Milestones

- **M1 — Readers verified:** both readers list this machine's real sessions;
  hand-checked titles and last-human timestamps match for 10 sessions.
- **M2 — V1 release:** `omnidevx sessions` and `omnidevx sessions resume`
  pass the post-reboot acceptance test. Releases tagged in order:
  `omnidevx-core`, `omni-openai`, `omnidevx`.
- **M3 — Evidence release:** `show` lists modified repositories and created
  commits that match `git log` for sampled sessions.
- **M4 — tmux continuity:** after a reboot, `resume --tmux` restores
  previously named tmux sessions without collisions.
- **M5 — Initiative complete:** summaries and search shipped; Kiro decision
  recorded.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Harness storage formats change without notice | Probe schemas instead of assuming them; parse failures become diagnostics; fixture tests per observed format version; fail soft per session, never per catalog |
| Prompt content leaks into telemetry or off-machine | Separate package and documented content-access contract; no store writes from readers; `--no-content`; only `summarize` sends data, with a bounded package |
| `list` is slow with hundreds of sessions | Head/tail reads plus a (path, size, mtime) cache; Codex lists from a single SQLite query |
| Misclassified human activity (tool results recorded as user messages) | Explicit classification rule with fixtures for current and legacy record shapes; hand-checked acceptance sample |
| Stale live-process records report sessions as running | Verify PID liveness and process start time before reporting `running` |
| Resuming a session that is already running elsewhere | Refuse by default and show where it runs; `--force` to override |
| Codex has no live-process record | Spike in Phase 4; report `unknown` rather than guess |
| Multi-repo release coordination | Fixed release order (core → openai → omnidevx); no local `replace` directives at push time |

## Conventions

- Commits carry `Refs: RMI-<REPOSLUG>-<NNN>` trailers for the RMI they
  implement, in the repository that owns that RMI.
- Go structs are the source of truth for the session contract; the JSON
  Schema is generated, linted with `--property-case camelCase`, and
  embedded.
- Unit tests use sanitized fixtures only; real session data is never
  committed.
- Each release updates `CHANGELOG.json`, regenerates `CHANGELOG.md`, adds a
  release note under `docs/releases/`, and is recorded against this
  initiative.
