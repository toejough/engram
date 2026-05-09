# `engram quick` — fleeting note write CLI

## Purpose

Offload the mechanical filename-derivation and file-write step of fleeting note capture from the `capturing-fleeting-notes` skill into a CLI subcommand. The skill currently uses the `Write` tool — one model turn per note (~15–20s). A CLI call via `Bash` is a subprocess (~50ms) and parallelizes more cheaply.

The skill remains in charge of *what* to capture (slug, body content). Engram handles the file location and write.

## Non-goals

- No body templating (no auto source-line, no auto-header). The skill still authors the full body.
- No batch mode in v1. The agent issues N parallel `Bash` calls for N notes.
- No file-locking. Each filename is unique by the agent's slug choice; collisions are an agent-side concern (same as today with the `Write` tool).
- No Luhmann ID assignment. Fleeting notes do not have IDs; promotion still owns ID assignment.
- No changes to vault layout, recall behavior, promotion behavior, or other skills.

## Inputs

- `--slug <slug>` (required) — kebab-case tag, validated against `[a-z0-9-]+`
- `--content <text>` OR stdin (required, one-or-other) — full body markdown as the agent wrote it (header + observation + `**Source:**` line)
- `--vault <path>` (optional) — vault root directory; falls back to `ENGRAM_VAULT_DIR` env; errors if neither is set

## Behavior

1. Resolve vault path: `--vault` flag, else `ENGRAM_VAULT_DIR` env, else error.
2. Verify `<vault>/Fleeting/` exists; error if not.
3. Validate slug: matches `^[a-z0-9-]+$`; error otherwise.
4. Read content from `--content` flag or stdin (mutually exclusive — error if both or neither).
5. Compute filename: `<vault>/Fleeting/<YYYY-MM-DD>.<slug>.md` using current local date.
6. If file exists, error (no overwrite).
7. Write content to the file.
8. Print the written path on stdout.

## Errors

| Condition | Behavior |
|---|---|
| `--slug` missing or empty | Exit non-zero with message naming the flag |
| Slug fails `[a-z0-9-]+` regex | Exit non-zero, name the offending slug |
| Both `--content` and stdin provided | Exit non-zero |
| Neither `--content` nor stdin provided | Exit non-zero |
| `--vault` and `ENGRAM_VAULT_DIR` both unset | Exit non-zero |
| `<vault>` doesn't exist | Exit non-zero, name the path |
| `<vault>/Fleeting/` doesn't exist | Exit non-zero, name the path |
| Target file already exists | Exit non-zero, name the path |
| Filesystem write fails | Exit non-zero, propagate error |

All errors go to stderr; exit code is non-zero (any standard non-zero is fine for v1).

## Code structure

Following project conventions (CLAUDE.md, internal/cli pattern):

- `internal/cli/quick.go` — `runQuick(ctx, args, deps, stdout) error` plus a `QuickDeps` struct for DI (clock + filesystem). Pure business logic, no `os.*` calls.
- `internal/cli/quick_test.go` — unit tests via DI mocks (imptest if used elsewhere; otherwise hand-rolled fakes consistent with current test style)
- `internal/cli/targets.go`:
  - Add `QuickArgs` struct alongside `LearnFactArgs` etc., with `targ:"flag,..."` tags
  - Register `targ.Targ(func(ctx context.Context, a QuickArgs) { ... })` in `Targets()`
- `internal/cli/cli.go` — wire DI: clock from `time.Now`, filesystem reads/writes from a thin wrapper struct (mirroring existing `osFileReader`)

`QuickArgs`:
```go
type QuickArgs struct {
    Slug    string `targ:"flag,name=slug,desc=kebab-case tag"`
    Content string `targ:"flag,name=content,desc=full body markdown (or use stdin)"`
    Vault   string `targ:"flag,name=vault,env=ENGRAM_VAULT_DIR,desc=vault root directory"`
}
```

`QuickDeps`:
```go
type QuickDeps struct {
    Now       func() time.Time
    Stdin     io.Reader
    StatVault func(string) error            // checks vault + Fleeting/ exist
    WriteFile func(path string, data []byte) error  // errors if path exists
}
```

The "errors if path exists" semantic is the v1 way to enforce no-overwrite — implement via `os.OpenFile` with `O_CREATE|O_EXCL|O_WRONLY` in the wrapper, surface the `errors.Is(err, fs.ErrExist)` to a clear message in the runner.

## Testing

Per CLAUDE.md test categorization:
- **Unit tests** (`quick_test.go`) cover business logic via DI: slug validation, vault resolution priority, file-existence error path, content-source mutex, filename derivation, success path. No real filesystem.
- **Integration tests** are minimal — one happy-path test that exercises the real `os.OpenFile` wrapper and confirms a file appears at the expected path. Per project convention, this lives separately or is gated.

Coverage target: same as rest of `internal/cli/` (`targ check-full` enforces).

## Skill change (after CLI ships)

`capturing-fleeting-notes` step 5 changes:
- "Hard rule: all `Write` calls go in a single parallel tool-use block" → "Hard rule: all `engram quick` Bash calls go in a single parallel tool-use block"
- Update example commands and the common-mistakes table accordingly
- Note: subagent-dispatch threshold (>20 notes) probably moves up since per-call cost is much lower; revisit empirically after the first uses

This is a separate change from the CLI itself and follows the writing-skills TDD discipline (the empirical RED is "agent uses Write tool when engram quick exists").

## Out of scope (potential v2 ideas, not for now)

- `--batch` mode accepting JSON array on stdin for one-shell-call N-note writes
- Body templating (auto source-line, auto-header)
- File-locking for concurrent same-slug writes
- Promotion integration
