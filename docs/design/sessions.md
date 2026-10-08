# Sessions

How `omnidevx sessions` is built today: which module does what, how a command
runs, and the rules that keep prompt content out of the telemetry pipeline.
For how to use it, see the [Sessions guide](../guides/sessions.md). For the
planned work, see the [Session Recovery initiative](../specs/initiatives/INIT-OMNIDEVX-001/PRD.md).

## Where the code lives

The session feature follows the same boundary rule as the collectors: this
repository composes, and vendor storage formats live with their owners.

| Concern | Module | Package |
|---------|--------|---------|
| `Session`, `Reader`, `Catalog`, ID resolution, resume spec | `omnidevx-core` | `sessions` |
| Claude Code session reader | `omnidevx-core` | `providers/claudecode` |
| Codex CLI session reader | `omni-openai` | `omnidevx` |
| Default catalog, filters, resume guards, process hand-off | this repo | `sessions` |
| Commands and output formatting | this repo | `cmd/omnidevx` |
| Re-exports for library users | this repo | root package (`sessionreaders.go`) |

This repository does not parse any harness file. A new harness gets a reader
in the module that owns its format, and is added to `NewDefaultCatalog`.

## Anatomy of a command

```text
cmd/omnidevx            sessions                     omnidevx-core / omni-openai
────────────            ────────                     ───────────────────────────
flags ──→ ListOptions ──→ NewDefaultCatalog ──→ Catalog.List ──→ []Session (newest first)
                         Filter.Apply  ←─────────────────────────┘
table / JSON ←── Resolve(prefix) ──→ Prepare (guards) ──→ Exec
```

- **Dependencies are injected.** The commands take a `deps` value holding the
  catalog constructor, the clock, the home directory, and the exec function.
  Tests substitute fakes, so no test touches real harness storage or starts a
  process.
- **Readers list cheaply.** A reader returns metadata from indexes and
  bounded reads of a transcript's start and end, never a full parse, so
  listing stays fast with thousands of sessions.
- **The catalog isolates failures.** One failing reader reports an error and
  the other harness's sessions are still listed.
- **Filtering happens after listing.** `--harness`, `--state` and `--cwd` are
  applied by `Filter`; `--since` and `--all` are passed to the readers so they
  can skip work. The `--cwd` match compares cleaned paths, so a directory and
  its subdirectories match but a sibling sharing a name prefix does not.

## Resume

A reader returns a `ResumeSpec`: the command and the directory to run it in.
It never runs anything. This package decides whether running it is safe:

1. **Resolve** the argument to one session by unique prefix of at least four
   characters, optionally qualified by harness (`codex:0199`). An ambiguous
   prefix is an error that lists up to eight candidates.
2. **Prepare** refuses a `running` session unless `--force` is given, because
   resuming one that is live in another terminal attaches a second agent to
   the same conversation. It also refuses a session whose directory is gone,
   since Claude Code finds a session by the directory it started in.
3. **Exec** changes to the directory and replaces the process, so the agent
   gets the terminal directly. `--print` skips this step and prints the
   command.

On Windows a process cannot be replaced, so `Exec` runs the command as a child
with the terminal attached and exits with its status. That platform split
lives in `process.Exec` from [`oscompat`](https://github.com/grokify/oscompat),
which this package calls.

## Privacy boundary

The telemetry pipeline is metadata only. The session feature is the one place
that reads prompt text, so it is kept apart from that pipeline:

- Session readers never produce telemetry events, and nothing a session
  command reads is written to the event store.
- The commands make no network calls and write no files.
- `--no-content` is passed to the readers, which leave prompt-derived fields
  out of what they return, so the text is not held by the command at all.
- `omnidevx collect` and `omnidevx sessions` share no code path that handles
  content.

Do not route session data into `collect`, and do not add prompt text to the
canonical event. See the
[OmniDevX Core privacy model](https://plexusone.github.io/omnidevx-core/guides/concepts/privacy/).

## Running state

State comes from the reader, because only the harness knows how it records a
live process:

- **Claude Code** writes a record per live process. The reader accepts it only
  if the process exists and started when the record says it did. The check
  is the same on macOS, Linux, and Windows.
- **Codex CLI** writes no such record. The reader reports `running` only when
  a `codex` process names the thread ID on its command line, and otherwise
  reports `unknown` rather than guessing. It lists processes with `ps`, so on
  Windows, which has no `ps`, every Codex session is `unknown`.

## Version

The CLI reports the version set at build time with
`-ldflags "-X main.version=vX.Y.Z"`, else the module version embedded by
`go install`, else `dev`.
