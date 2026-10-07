# TRD — Session Recovery — Discover, Understand, and Resume Coding-Agent Sessions

**Initiative:** `INIT-OMNIDEVX-001`

## Architecture Overview

The session catalog is a **separate read path** alongside the existing
telemetry collectors. Both read the same harness files; they produce
different outputs under different contracts.

```text
                       ┌─→ Collector ─→ metadata-only Event ─→ store / reports   (unchanged)
Claude Code files ─────┤
                       └─→ SessionReader ─┐
                                          ├─→ Catalog ─→ list / show / resume / summarize
Codex CLI files ───────→ SessionReader ───┘      │
                                                 ├── RepoIndex      (path → repository)
                                                 ├── git            (commit resolution)
                                                 ├── WorkRef rules  (INIT-/RMI- patterns)
                                                 └── Binding store  (tmux preferences)
```

Placement follows the existing provider boundaries:

| Module | Responsibility |
|---|---|
| `omnidevx-core/sessions` | Provider-neutral types, `Reader` interface, `ResumeSpec`, repository index, work-reference extraction, git commit resolution, JSON Schema |
| `omnidevx-core/providers/claudecode` | Claude Code `SessionReader` |
| `omni-openai/omnidevx` | Codex CLI `SessionReader` |
| `omnidevx/sessions` | Catalog composition, ID resolution, runtime binding (tmux), binding store, resume execution, summarizer |
| `omnidevx/cmd/omnidevx` | Thin Cobra CLI over `omnidevx/sessions` |

## Content-Access Contract

The telemetry collectors guarantee that prompt text, model responses, and
file contents are never captured. Session discovery necessarily reads some
content (titles, prompts, tool inputs) to be useful. The contract:

1. Content is read **only** by `SessionReader` implementations, on demand,
   from local files.
2. Nothing produced by a `SessionReader` is written to the telemetry event
   store.
3. Prompt-derived text (`Title`, `RecentPrompts`, summaries) is held in
   memory and rendered; it is persisted only in the optional catalog cache
   under `~/.plexusone/omnidevx/sessions/`, which is user-local, documented,
   and removable with `omnidevx sessions cache clear`.
4. Output is never sent off-machine except by the explicit `summarize`
   command, which documents its model provider and sends only the bounded
   evidence package described below.
5. `--no-content` suppresses all prompt-derived fields in `list`/`show`
   output (titles fall back to working-directory name).

## Core Types (`omnidevx-core/sessions`)

Go structs are the source of truth; JSON Schema is generated with
`invopop/jsonschema` and linted with `schemakit lint --property-case camelCase`
(this is a document/API surface, not a telemetry event).

```go
type Harness string // "claude-code" | "codex"

type Key struct {
    Harness Harness `json:"harness"`
    ID      string  `json:"id"`
}

type Session struct {
    Key

    CWD       string `json:"cwd"`
    GitBranch string `json:"gitBranch,omitempty"`
    GitOrigin string `json:"gitOrigin,omitempty"`

    CreatedAt           time.Time `json:"createdAt"`
    LastActivityAt      time.Time `json:"lastActivityAt"`
    LastHumanActivityAt time.Time `json:"lastHumanActivityAt,omitzero"`

    Title         string   `json:"title,omitempty"`
    TitleSource   string   `json:"titleSource,omitempty"` // harness | first-prompt | cwd
    RecentPrompts []Prompt `json:"recentPrompts,omitempty"`

    MessageCount  int `json:"messageCount"`
    ToolCallCount int `json:"toolCallCount"`

    Archived bool    `json:"archived,omitempty"`
    State    State   `json:"state"` // running | resumable | unknown
    Runtime  *Runtime `json:"runtime,omitempty"`

    Evidence *Evidence `json:"evidence,omitempty"` // populated by Get / --evidence
    Resume   ResumeSpec `json:"resume"`
}

type Prompt struct {
    At   time.Time `json:"at"`
    Text string    `json:"text"` // truncated
}

type Runtime struct {
    PID  int   `json:"pid,omitempty"`
    Tmux *Tmux `json:"tmux,omitempty"`
}

type Tmux struct {
    SessionName string `json:"sessionName"`
    WindowIndex int    `json:"windowIndex"`
    PaneID      string `json:"paneId"`
}

type ResumeSpec struct {
    Argv []string `json:"argv"` // e.g. ["claude", "--resume", "<id>"]
    Dir  string   `json:"dir"`
    Env  []string `json:"env,omitempty"`
}
```

### Reader interface

```go
type ListOptions struct {
    Since          time.Time // filter on LastActivityAt
    IncludeArchived bool
    NoContent      bool
}

type Reader interface {
    Harness() Harness
    List(ctx context.Context, opts ListOptions) ([]Session, []omnidevx.Diagnostic, error)
    Get(ctx context.Context, id string, opts GetOptions) (*Session, error) // full parse + evidence
}
```

`List` must be cheap: it may use file metadata, harness indexes, and tail
reads, and returns `Evidence == nil`. `Get` performs a full parse and
populates `Evidence`. Parse failures become diagnostics, matching the
existing collector convention; only I/O failures on the storage root are
errors.

Resume is deliberately **not** a `Reader` method: readers return a
`ResumeSpec`, and execution lives in `omnidevx/sessions`.

### Evidence

```go
type Evidence struct {
    Repos    []RepoActivity `json:"repos,omitempty"`
    Files    []FileActivity `json:"files,omitempty"`
    Commits  []CommitRef    `json:"commits,omitempty"`
    WorkRefs []WorkRef      `json:"workRefs,omitempty"`
}

type RepoActivity struct {
    Root      string    `json:"root"`
    ID        string    `json:"id,omitempty"` // e.g. github.com/org/repo
    CWD       bool      `json:"cwd"`
    Read      bool      `json:"read"`
    Modified  bool      `json:"modified"`
    Mentioned bool      `json:"mentioned"`
    FileCount int       `json:"fileCount"`
    FirstSeen time.Time `json:"firstSeen"`
    LastSeen  time.Time `json:"lastSeen"`
}

type FileActivity struct {
    Path     string    `json:"path"`
    Read     bool      `json:"read"`
    Modified bool      `json:"modified"`
    LastSeen time.Time `json:"lastSeen"`
}

type CommitRef struct {
    SHA       string    `json:"sha"`
    Repo      string    `json:"repo,omitempty"`
    Relation  string    `json:"relation"` // created | referenced
    Subject   string    `json:"subject,omitempty"`
    At        time.Time `json:"at"`
}

type WorkRef struct {
    ID      string   `json:"id"`      // e.g. RMI-EXAMPLE-012
    Kind    string   `json:"kind"`    // rule name, e.g. initiative | rmi
    Sources []string `json:"sources"` // prompt | branch | commit | path
    Title   string   `json:"title,omitempty"` // from optional resolver
}
```

### Repository index

- Configurable workspace roots (`workspace-roots` in the OmniDevX config;
  default: none, plus every session CWD).
- Discovery walks each root to a bounded depth for `.git` entries (dirs or
  worktree files), and caches the root list.
- Path resolution is a longest-prefix match against known roots; unknown
  paths fall back to walking up for `.git` and are added to the index.
- Repository IDs derive from the `origin` remote URL when present
  (`github.com/org/repo`), else the root path.

### Work-reference extraction

- Rules are named regexes in config; the default rule set matches
  `\bINIT-[A-Z0-9]+-\d{3}\b` and `\bRMI-[A-Z0-9]+-\d{3}\b`.
- Sources scanned: human prompts, git branch, created-commit messages, and
  file paths. Assistant text is not scanned in V1 (too noisy).
- An optional `WorkRefResolver` interface supplies titles; no resolver is
  built in.

### Git resolution

- **Created commits:** a shell tool call whose command runs `git commit`
  and whose result contains a `[branch <sha>]` line yields the SHA; the
  repository is the command's working directory resolved through the index.
- **Referenced commits:** 7–40 hex tokens in human prompts that resolve via
  `git cat-file -e` in a repository the session touched.
- Subjects and timestamps come from `git show -s --format=…`, executed with
  a timeout and only during `Get`.

## Claude Code Reader (`omnidevx-core/providers/claudecode`)

Storage (observed; undocumented and version-dependent):

- `~/.claude/projects/<encoded-cwd>/<session-id>.jsonl` — one transcript per
  session.
- `~/.claude/sessions/<pid>.json` — one record per live process with `pid`,
  `sessionId`, `cwd`, `startedAt`, `updatedAt`, `status`, `name`.

Field derivation:

| Field | Source |
|---|---|
| ID, CWD, GitBranch | `sessionId`, `cwd`, `gitBranch` on records (last value wins for branch) |
| CreatedAt | first record timestamp |
| LastActivityAt | last record timestamp (tail read in `List`) |
| LastHumanActivityAt | last `type=user` record that is a human prompt (below) |
| Title | latest `ai-title` record; else first human prompt; else CWD base name |
| RecentPrompts | `last-prompt` record and last N human prompts |
| State / Runtime.PID | `~/.claude/sessions/*.json` whose `pid` is alive and whose process start matches `procStart` |

**Human prompt rule:** a `type=user` record with `promptSource` of `typed`
or `queued`. For records without `promptSource` (older versions): string or
`text` content, not `isMeta`, not `isSidechain`, no `tool_result` blocks,
and not a harness-injected wrapper (e.g. command or reminder tags).

**Performance:** `List` reads the head (for `CreatedAt`, first prompt) and
a bounded tail (for last activity, title, last prompt) of each file, and
caches results keyed by (path, size, mtime).

Resume spec: `argv = ["claude", "--resume", id]`, `dir = CWD`. The CWD is
required because Claude Code locates sessions by project directory.

## Codex CLI Reader (`omni-openai/omnidevx`)

Storage (observed):

- `~/.codex/state_N.sqlite`, table `threads` — `id`, `rollout_path`, `cwd`,
  `title`, `name`, `first_user_message`, `preview`, `created_at_ms`,
  `updated_at_ms`, `archived`, `git_branch`, `git_origin_url`, `git_sha`.
  The highest `N` present is used; the schema is probed, not assumed.
- `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` — full event log.

`List` reads from `threads` (one query) and derives last human activity
from a tail read of the rollout file. `Get` parses the rollout for evidence.
Title precedence: `name` → `title` → `first_user_message` → CWD base name.

Resume spec: `argv = ["codex", "resume", id]`, `dir = CWD`.

Live-process detection for Codex has no known equivalent of Claude's PID
records; it is investigated as a spike. Until then Codex sessions report
`state: unknown` unless bound by OmniDevX itself.

## Catalog (`omnidevx/sessions`)

- Composes the default readers (Claude Code, Codex CLI), runs `List`
  concurrently, merges and sorts by `LastActivityAt` descending.
- **ID resolution:** accepts full IDs or unique prefixes (minimum 4
  characters) across all harnesses; an optional `harness:` qualifier
  disambiguates. Ambiguous prefixes return an error listing the candidates.
- **Binding store:** `~/.plexusone/omnidevx/sessions/bindings.json`, keyed
  by `Key`, holding `PreferredTmuxName`, `LastTmuxName`, and timestamps.
  Written when OmniDevX resumes into tmux, or when a running session is
  observed in a named tmux pane.

### Runtime binding (tmux)

1. `tmux list-panes -a -F '#{session_name} #{window_index} #{pane_id} #{pane_pid}'`
   (skipped silently when tmux is not installed or no server runs).
2. For each live harness PID known from reader runtime data, walk the
   process ancestry (`ps -o ppid=`) to find a pane PID.
3. A match yields `Runtime.Tmux` and updates the binding store.

### Resume execution

- Refuses if `State == running` unless `--force`, printing where it runs
  (tmux target, if known).
- Fails clearly if the CWD no longer exists.
- Default: `chdir` + `syscall.Exec` of `ResumeSpec.Argv` (replaces the
  OmniDevX process — the user lands directly in the harness).
- `--tmux[=name]`: name precedence is flag → `PreferredTmuxName` → CWD base
  name. If the name exists and is bound to a different live session, append
  `-2`, `-3`, …; if it exists and is idle, create a new window in it. Uses
  `tmux new-session -d -s <name> -c <dir> -- <argv…>` and then attaches
  (or `switch-client` when already inside tmux).
- `--print`: emit the shell command instead of executing (for GUI clients
  and agents).

### Summarizer (later phase)

- `Summarizer` interface over an evidence package: CWD, repos, files
  touched, commits, work refs (with resolved titles), first prompt, last N
  human prompts, last N tool activities, and any previous summary.
- Output: `Title`, `Objective`, `Completed`, `CurrentWork`, `LikelyNext`.
- Cached per (session key, last activity time); regenerated only when the
  session has changed.
- Model provider is configurable; no provider is required for any other
  command.

## CLI

The CLI moves from stdlib `flag` to Cobra; `collect` and `version` keep
their current flags and behavior.

```text
omnidevx sessions [list]       [--since 24h] [--cwd .] [--harness claude-code|codex]
                               [--repo github.com/org/repo] [--state running|resumable]
                               [--all] [--no-content] [--json]
omnidevx sessions show <id>    [--json] [--no-content]
omnidevx sessions resume <id>  [--tmux[=name]] [--print] [--force]
omnidevx sessions summarize <id> [--json]          (later phase)
omnidevx sessions search <query> [--json]          (later phase)
omnidevx sessions cache clear
```

Default `list` columns: harness, short ID, CWD base name, last human
(relative), last activity (relative), state, tmux name, title.

## Repository Layout

```text
omnidevx-core/
  sessions/            types, Reader, RepoIndex, WorkRef rules, git resolver
  sessions/schema/     generated session.schema.json + //go:embed
  providers/claudecode/sessions.go
omni-openai/omnidevx/sessions.go
omnidevx/
  sessions/            catalog, resolve, bindings, tmux, resume, summarize
  cmd/omnidevx/        Cobra commands (thin)
```

## Conformance & Testing

- Reader tests run against sanitized fixture files in `testdata/`
  (synthetic paths such as `/Users/example/src/example`), covering:
  current and legacy Claude record formats, tool-result vs. human prompt
  classification, sidechain records, missing `ai-title`, and a Codex
  `threads` schema with and without optional columns.
- A shared contract test (in `omnidevx`) asserts every reader satisfies
  `sessions.Reader` invariants: non-empty keys, `LastActivityAt ≥
  CreatedAt`, `LastHumanActivityAt ≤ LastActivityAt`, non-empty
  `ResumeSpec.Argv` and `Dir`.
- tmux and process inspection sit behind small interfaces with fakes; no
  unit test requires tmux or live processes.
- Resume execution is tested via `--print` output and an injectable exec
  function.
- Schema: generated, linted with `schemakit lint --property-case camelCase`,
  embedded, and validated against `--json` output in tests.
- Real-data verification is manual and local; real session data is never
  committed.

## Dependencies

- `github.com/spf13/cobra` (latest release, verified at implementation time).
- `github.com/invopop/jsonschema` for schema generation (via `//go:build
  ignore` generator plus a `tools.go` guard).
- `modernc.org/sqlite` — already used by the Codex collector.
- Release order: `omnidevx-core` → `omni-openai` → `omnidevx`.
