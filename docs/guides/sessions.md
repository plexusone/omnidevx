# Sessions

`omnidevx sessions` finds the Claude Code and Codex CLI sessions on this
machine, shows enough about each to recognize it, and resumes the one you
pick. It is for the moment after a reboot, a closed terminal, or a long
week, when you have many sessions and cannot remember which was which.

## List sessions

```bash
omnidevx sessions
```

```text
HARNESS  ID        CWD                          LAST HUMAN  LAST ACTIVE  STATE      TITLE
claude   3f9a1c20  github.com/acme/billing      4m          now          running    billing-api
codex    7b2e4d91  github.com/acme/web          2h          2h           unknown    fix flaky checkout test
claude   a04c77e5  …/acme/infra                 1d          1d           resumable  terraform module split

3 session(s). Details: omnidevx sessions show <id>   Resume: omnidevx sessions resume <id>
```

Sessions are listed newest first. The columns are:

| Column | Meaning |
|--------|---------|
| `HARNESS` | The tool: `claude` or `codex` |
| `ID` | The start of the session ID; in other commands, any unique prefix of at least four characters works |
| `CWD` | The directory the session started in, shortened to fit |
| `LAST HUMAN` | How long ago a person last typed a prompt |
| `LAST ACTIVE` | How long ago the session last did anything, including agent work |
| `STATE` | `running`, `resumable`, or `unknown` (see below) |
| `TITLE` | The title the tool recorded, else the first prompt, else the directory name |

`LAST HUMAN` and `LAST ACTIVE` are separate on purpose. A session that has
been running tool calls for an hour looks active, but if you last typed to it
yesterday it is not the one you were just working in.

### Filter the list

| Flag | Effect |
|------|--------|
| `--cwd <dir>` | Only sessions started in that directory or below. Use `.` for the current directory |
| `--harness <name>` | Only `claude-code` or `codex` |
| `--state <state>` | Only `running`, `resumable`, or `unknown` |
| `--since <duration>` | Only sessions active within that time, such as `90m`, `24h`, or `7d` |
| `-n, --limit <n>` | Show at most `n` sessions; `0` shows all |
| `--all` | Include archived sessions |
| `--json` | Print JSON instead of a table |
| `--no-content` | Omit prompt-derived text; titles fall back to the directory name |

For example, what you were doing in this repository this week:

```bash
omnidevx sessions --cwd . --since 7d
```

### What the states mean

| State | Meaning |
|-------|---------|
| `running` | A live process is attached to this session |
| `resumable` | No live process; the tool can pick the session back up |
| `unknown` | OmniDevX cannot tell |

Claude Code records a file for each live process, and OmniDevX checks that the
process exists and started when the file says it did, so a stale file left by
a crash is ignored.

Codex keeps no such record. OmniDevX reports a Codex session as `running` only
when a `codex` process names its thread ID on the command line, as
`codex resume <id>` does. A Codex session you started fresh and still have open
shows as `unknown`, not guessed.

On Windows, OmniDevX cannot verify that a Claude Code session is running, so
none is reported as `running`. Check for an open window before resuming.

## Show one session

```bash
omnidevx sessions show 3f9a
```

`show` accepts any unique prefix of the ID of at least four characters, and
prints the session's title, state, directory, branch, timestamps, recent
prompts, and the command that resumes it. Add `--json` for machine-readable output or `--no-content` to omit
prompt text.

If the prefix matches more than one session, `show` lists up to eight of the
candidates and how many matched. Add more characters, or qualify the
prefix with its harness, such as `codex:0199`.

## Resume a session

```bash
omnidevx sessions resume 3f9a
```

`resume` runs the tool's own resume command from the directory the session
started in, and replaces the `omnidevx` process, so you land directly in the
agent. That directory matters: Claude Code finds a session by the project
directory it started in.

Two guards protect you:

- A session that appears to be **running** is refused. Resuming one that is
  open elsewhere can fork or corrupt it. Use `--force` to override.
- If the session's **directory no longer exists**, `resume` stops and tells
  you, rather than starting the agent somewhere else.

To see the command without running it, for use in a script or another shell:

```bash
omnidevx sessions resume 3f9a --print
```

On Windows, the command runs as a child process and `omnidevx` exits with its
status, because Windows cannot replace a running process.

## Privacy

Session titles and prompts are read from local harness storage when you run
the command, shown in your terminal, and discarded. They are never written to
the telemetry event store and never leave your machine; the command makes no
network calls.

Use `--no-content` when you want the listing without any prompt-derived text,
for example in a screen share.

This is separate from `omnidevx collect`, which records metadata only and
never reads prompt text. See the
[OmniDevX Core privacy model](https://plexusone.github.io/omnidevx-core/guides/concepts/privacy/).

## Limits

- Only Claude Code and Codex CLI sessions are listed. Kiro CLI is not.
- The list shows what a session was about from its title and prompts. It does
  not yet show which repositories or files a session touched.
- `omnidevx sessions` reads local files only. A session from another machine
  does not appear.
