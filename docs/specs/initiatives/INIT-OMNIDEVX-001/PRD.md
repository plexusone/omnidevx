# PRD — Session Recovery — Discover, Understand, and Resume Coding-Agent Sessions

**Initiative:** `INIT-OMNIDEVX-001`

## Problem

Developers who work with coding agents routinely run many concurrent harness
sessions — Claude Code and Codex CLI in separate terminals, often across
several repositories. After a reboot, a closed terminal, or simply an
accumulation of 15–20 sessions, it is hard to answer:

- Which sessions exist, and which ones was I actually working in?
- What was each session doing, and where did it leave off?
- When did I last interact with it, versus when did the agent last act?
- How do I get back into the one I want, in the right directory and the
  terminal layout I recognize?

Both harnesses persist sessions to disk and can resume them by ID
(`claude --resume <id>`, `codex resume <id>`), and both offer a picker. But
the pickers are per-harness, show little context, and know nothing about
the repositories, commits, or tracked work a session touched. The
information needed to recognize a session exists locally; nothing assembles
it.

## Vision

One command that lists every resumable coding-agent session on the machine
with enough deterministic context to recognize it, and a second command
that resumes the chosen one exactly where it was:

```text
DISCOVER → UNDERSTAND → SELECT → RESUME
```

```text
$ omnidevx sessions
HARNESS  SESSION   CWD          LAST HUMAN  LAST ACTIVITY  STATE      TITLE
claude   a83f12…   omnidevx     18m         3m             running    Session discovery and resume
codex    0199ab…   example-api  2h          1h             resumable  Add GitHub App authentication
claude   e721c0…   example-web  1d          1d             resumable  Refactor roadmap schema

$ omnidevx sessions resume e721
```

**Design criterion for every field:** *does this help me decide whether this
is the session I want to restart?*

## Users

- **Primary — individual developer running many agent sessions.** Needs to
  recover working context after a reboot or context switch, and to resume
  the right sessions quickly.
- **Secondary — coding agents.** The CLI with `--json` output is directly
  usable by an agent (e.g. "find the session that was working on X").
- **Secondary — GUI clients.** A terminal-hosting desktop app can render the
  catalog as a dashboard and launch resumes, consuming the JSON contract
  rather than harness storage formats.

## Goals

1. **Discover** Claude Code and Codex CLI sessions from their local storage,
   with no harness changes and no network access.
2. **Understand** each session through deterministic facts: harness,
   session ID, working directory, branch, created / last activity / last
   human activity times, title, and running vs. resumable state.
3. **Enrich** sessions with deterministic evidence: repositories read or
   modified, commits created or referenced, and work-tracking references
   (configurable ID patterns such as `INIT-…` / `RMI-…`).
4. **Resume** a selected session via its native harness command in its
   original working directory, optionally recreating its remembered tmux
   session name.
5. **Expose a stable JSON contract** (`--json`) so agents and GUI clients
   never parse harness-internal formats.
6. **Explain** sessions with an optional, LLM-generated summary grounded in
   the deterministic evidence (later phase).

## Non-Goals (V1)

- **No transcript content in the telemetry event stream.** The existing
  metadata-only `Event` contract and store are unchanged.
- **No GUI.** The CLI and JSON contract are the deliverable; GUI clients
  build on them.
- **No work-session orchestration.** Grouping harness sessions into larger
  units of work, assignment, or coordination belongs to higher layers.
- **No daemon or long-running service.** The catalog is computed on demand,
  with an optional local cache.
- **No remote or multi-machine sessions.** Local storage only.
- **No harness control beyond resume.** OmniDevX does not drive, prompt, or
  terminate harness processes.
- **No LLM calls in `list`, `show`, or `resume`.** Model calls happen only in
  an explicit `summarize` command.
- **Kiro CLI sessions** are evaluated in a later phase, not committed for V1.

## Key Product Decisions

1. **Harness session ID is the identity.** The pair (harness, session ID) is
   durable across process exit and reboot. PIDs, terminals, and tmux names
   are runtime attachments, never identity.
2. **Facts are deterministic; LLMs only explain.** Working directory,
   timestamps, repositories, commits, and work references are extracted
   directly from harness records and git. A summary may interpret those
   facts but never establishes them.
3. **Content access is opt-in and separate from telemetry.** Session
   discovery reads titles and prompts that the telemetry collectors
   deliberately ignore. This happens in a separate package with its own
   documented contract, is local-only, and never writes content into the
   telemetry store.
4. **Last human activity is a first-class timestamp.** The gap between
   last activity and last human activity distinguishes interactive sessions
   from agents that kept working unattended.
5. **tmux is optional but sticky.** Sessions started without tmux are fully
   supported. Once a session is observed in or launched into a named tmux
   session, that name is remembered and preferred when resuming.
6. **Providers describe; the CLI executes.** Each harness returns a
   resume specification (argv, working directory, environment); the CLI or
   a GUI client decides how to run it (replace process, new tmux session).
7. **Prefer native metadata over heuristics.** Use harness-provided titles,
   previews, and live-session records where they exist before inferring.

## Success Criteria

- **Acceptance test (V1):** after a reboot, `omnidevx sessions` lists the
  previous day's Claude Code and Codex sessions; the developer recognizes
  the ones they want from the listing alone; `omnidevx sessions resume <id>`
  reopens each in its original directory with the native harness.
- `omnidevx sessions` completes in under 2 seconds on a machine with
  hundreds of session files (warm cache).
- Last human activity matches the developer's last typed prompt for every
  session in a hand-checked sample of 10, across both harnesses.
- Running vs. resumable state is correct for all live Claude Code sessions.
- A session's modified repositories match `git log` evidence for commits it
  created, for a hand-checked sample.
- `--json` output validates against the generated JSON Schema.
