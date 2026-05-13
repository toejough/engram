# `engram update` — Design

## Goal

A single command that refreshes both the `engram` binary and the
harness-installed skills/commands on the current machine. Replaces the
removed install-time hooks. Users run `engram update` after pulling new
changes (or whenever they want the latest).

## Scope

- Update the `engram` binary via `go install`.
- Copy skill directories into harness-specific user-level skill paths.
- Copy OpenCode command files into the OpenCode user-level commands dir.
- Detect harnesses implicitly by checking config-directory existence.
- Detect source (local repo vs. remote module) implicitly by walking up
  from `cwd` looking for the engram `go.mod`.

Out of scope: prebuilt binaries, version pinning, partial updates,
uninstall, network proxies / private GOPROXY auth, Windows
path semantics (the binary's user base is macOS / Linux today).

## CLI shape

```
engram update [--dry-run]
```

`--dry-run`: print everything that would change without executing
`go install` or writing files. Exit 0.

No other flags in v1.

## Source resolution

Walk up from `cwd` looking for a `go.mod` whose first `module` directive
is `github.com/toejough/engram`.

| Found?     | Mode    | Behavior                                              |
|------------|---------|-------------------------------------------------------|
| Yes        | local   | `go install ./cmd/engram/` from the repo root; copy skill/command files directly from `<repo>/skills/` and `<repo>/opencode/commands/`. Picks up uncommitted local changes. |
| No         | remote  | `go install github.com/toejough/engram/cmd/engram@latest`. Resolve module path via `go list -m -json github.com/toejough/engram@latest`. Copy from `<modcache>/github.com/toejough/engram@<v>/skills/` and `.../opencode/commands/`. |

The mode is reported in output ("local clone at /Users/joe/src/engram"
or "remote module @ vX.Y.Z").

## Harness detection

For each known harness, check whether its top-level config directory
exists. If yes, install. If no, skip silently. Both can be present;
both get the same skills.

| Harness     | Probe path             | Skills target                                  | Commands target                            |
|-------------|------------------------|------------------------------------------------|--------------------------------------------|
| Claude Code | `~/.claude/`           | `~/.claude/skills/<name>/`                     | (none — this plugin ships no Claude commands) |
| OpenCode    | `~/.config/opencode/`  | `~/.config/opencode/skills/<name>/`            | `~/.config/opencode/commands/<file>`        |

If neither probe exists, exit with a clear message ("no supported harness
found at `~/.claude/` or `~/.config/opencode/`") and exit code 0 — not
finding a harness is not an error.

## Copy semantics

- **Skills.** For every directory under the source `skills/`, copy
  recursively into the harness skills dir. Each skill dir keeps its
  internal structure (including `tests/`, `references/`, etc.).
- **OpenCode commands.** Copy every `.md` under source `opencode/commands/`
  into `~/.config/opencode/commands/`.
- **Overwrite semantics.** Plain overwrite. No backup. No diff prompt.
  This is an update command; the user invoked it to refresh.
- **No deletion.** If a skill was removed upstream, `engram update` does
  not delete the stale local copy. Users who want a clean state can `rm
  -rf` the target dirs first. (Tracked as a v2 consideration.)

## Output

```
engram update
  source: local clone at /Users/joe/src/engram
  binary: go install ./cmd/engram/ ... ok (engram v0.1.0 → ~/go/bin/engram)
  Claude Code (~/.claude/):
    skills/learn/ → ~/.claude/skills/learn/  (3 files)
    skills/recall/ → ~/.claude/skills/recall/  (1 file)
  OpenCode (~/.config/opencode/):
    skills/learn/ → ~/.config/opencode/skills/learn/  (3 files)
    skills/recall/ → ~/.config/opencode/skills/recall/  (1 file)
    opencode/commands/learn.md → ~/.config/opencode/commands/learn.md
    opencode/commands/recall.md → ~/.config/opencode/commands/recall.md
done.
```

`--dry-run` adds the prefix `[dry-run] ` and skips execution.

## Implementation layout (per engram conventions)

- `internal/update/` — pure logic, DI'd:
  - `Updater` struct with injected interfaces:
    - `Filesystem` (Stat, MkdirAll, ReadFile, WriteFile, ReadDir, RemoveAll)
    - `Commander` (Run cmd with args, returns stdout/stderr/exit)
    - `Env` (Getenv, UserHomeDir)
  - Pure functions for path resolution, harness detection, plan
    construction.
  - `Run(ctx, opts) (Report, error)` is the entry point.
- `internal/cli/update.go` — wires subcommand, parses flags, instantiates
  real implementations of the interfaces, calls `Updater.Run`, formats
  the report.
- Subcommand registered in `internal/cli/targets.go`.

## Testing

- Unit tests in `internal/update/update_test.go` (blackbox package
  `update_test`) with imptest-generated mocks driving each scenario:
  local mode happy path, remote mode happy path, no harness present,
  both harnesses present, dry-run, `go install` failure, `go list` JSON
  parse failure, skill source dir missing.
- Integration test in `internal/update/integration_test.go` exercising
  the real `Commander` against a temp `GOPATH` and a fake module cache —
  optional; gated on env var if it requires network.

## Failure modes

| Failure                              | Behavior                                              |
|--------------------------------------|-------------------------------------------------------|
| `go` not on PATH                     | Error: "go binary not found on PATH". Exit 1.         |
| `go install` exits non-zero          | Print stderr; exit 1. Skill copy does not proceed.    |
| `go list -m -json ...` fails (remote)| Print stderr; suggest network / proxy. Exit 1.        |
| Module cache path doesn't exist      | Error: "module cache miss after go install". Exit 1.  |
| Source `skills/` missing             | Error naming the missing path; exit 1.                |
| Target dir not writable              | Error naming the path; exit 1.                        |
| One harness writable, other not      | Report partial success per-harness; exit 1.           |
