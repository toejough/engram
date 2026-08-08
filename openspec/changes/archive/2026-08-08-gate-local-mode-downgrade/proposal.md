## Why

`engram update`'s local-mode install leg (`resolveSource` in `internal/update/update.go`) runs `go install ./cmd/engram/` against whatever git-tracked module root it discovers by walking up from `cwd`, with zero comparison against what's currently installed at `~/go/bin/engram`. Running `engram update` from a stale clone or an older worktree silently overwrites a newer installed binary with older code — no warning, no version shown. This bit #706's e2e verification directly: a scratch worktree at HEAD rebuilt the pre-sync-engine binary mid-test, invalidating two test runs before the regression was noticed (vault note 628). Remote mode already resolves and reports a git revision after cloning (`SourceInfo.Version`); local mode's `SourceInfo.Version` is permanently empty — the field is commented `// resolved version string (remote only)`. See issue #715.

## What Changes

- Local mode now resolves and reports the module root's git revision (short SHA via `git rev-parse --short HEAD` in `root`), the same way remote mode already does — visible in the update report's `describeSource` line unconditionally.
- Before running `go install` in local mode, capture the currently-installed binary's embedded VCS revision (via `go version -m <BinaryPath>` — Go auto-embeds `vcs.revision` for binaries built from a git-tracked module, no extra build flags required) if a binary already exists at the install path.
- When both revisions are known and the module root's git history proves the installed revision is *not* an ancestor of the resolved commit in a way that shows regression (`git merge-base --is-ancestor <installedRev> <resolvedRev>` fails, meaning the resolved commit does NOT descend from what's currently installed) — refuse the install and report the provable downgrade, requiring a new `--allow-downgrade` flag to proceed anyway.
- When ancestry can't be determined (binary predates VCS-embedding, commit unknown to this clone, rebased/divergent history, or no prior binary exists) — proceed as today, but the resolved revision is still shown for visibility. This is a deliberate fail-open: an unprovable case must never block, only a provable regression does.
- `--dry-run` and the re-exec sentinel child (which never runs `go install` per ADR-0023/D2) are unaffected — the downgrade check only runs on an install that would actually occur.

## Capabilities

### New Capabilities

- `update-local-install-safety`: local-mode install-leg revision resolution, visibility, and provable-downgrade gating for `engram update`.

### Modified Capabilities

(none — this is new behavior in a previously-unspecified code path; no existing requirement described local-mode version resolution)

## Impact

- `internal/update/update.go`: `resolveSource`'s local-mode branch (~line 1219-1242) gains revision resolution (git rev-parse), pre-install binary-version capture (go version -m), and ancestry-check gating logic. `Options` gains `AllowDowngrade bool`. `SourceInfo.Version`'s doc comment (`// resolved version string (remote only)`) becomes stale and needs updating since local mode now sets it too. A new sentinel error (e.g. `ErrLocalDowngrade`) for the CLI layer to render distinctly from other install failures.
- `internal/cli/update.go`: `UpdateArgs` gains `AllowDowngrade bool` flag (`--allow-downgrade`); `describeSource` needs no change (already switches on `Source.Mode` and prints `Source.Version` — local mode just starts having a non-empty one to print, might want a small format tweak to match "local clone at ~/path (rev abc1234)").
- Tests: `internal/update/update_test.go` (or a new focused test file) for the ancestry-check gate, the fail-open cases, and revision visibility; `internal/cli/update_test.go` for the new flag wiring and error rendering.
- `Cmd` (the injected command-runner interface) needs no new methods — `git rev-parse`, `go version -m`, and `git merge-base --is-ancestor` are all invocable through the existing `u.Cmd.Run` used elsewhere in this file.
