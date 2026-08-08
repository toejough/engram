## 1. Red: revision visibility

- [x] 1.1 Write a failing test asserting local-mode `resolveSource` returns a non-empty `SourceInfo.Version` (short git SHA) when run against a git-tracked module root, and that the CLI report's source description includes it.

## 2. Green: revision visibility

- [x] 2.1 In `internal/update/update.go`'s local-mode branch of `resolveSource`, add `git rev-parse --short HEAD` in `root` (mirroring `resolveRemoteByClone`'s existing call) and set `SourceInfo.Version`.
- [x] 2.2 Update `SourceInfo.Version`'s doc comment (currently `// resolved version string (remote only)`) to reflect that local mode now sets it too.
- [x] 2.3 If needed, adjust `describeSource` in `internal/cli/update.go` to include the revision in the local-mode branch (e.g. `"local clone at ~/path (rev abc1234)"`).
- [x] 2.4 Run `targ test` and confirm task 1's test passes (GREEN).

## 3. Red: provable-downgrade gate

- [x] 3.1 Write failing tests for: (a) a provable downgrade is refused and reports the condition, exiting nonzero; (b) `--allow-downgrade` bypasses a provable downgrade; (c) no prior binary at the install path proceeds with no check; (d) an installed revision unknown to the module root's git history proceeds (fails open); (e) an installed binary with no parseable VCS revision (simulating a pre-VCS-embedding binary) proceeds (fails open).

## 4. Green: provable-downgrade gate

- [x] 4.1 Add `AllowDowngrade bool` to `Options` in `internal/update/update.go`, threaded from a new `--allow-downgrade` flag on `UpdateArgs` in `internal/cli/update.go`.
- [x] 4.2 Before local mode's `go install` call, if a file exists at the resolved `BinaryPath`, run `go version -m <BinaryPath>` and parse out `vcs.revision` (defensively — missing/unparseable means "unknown," not an error).
- [x] 4.3 When an installed revision is known, run `git merge-base --is-ancestor <installedRev> <resolvedRootHEAD>` in `root`. Exit 0 → proceed. Exit 1 → provable downgrade: if `!opts.AllowDowngrade`, return a new sentinel error (e.g. `ErrLocalDowngrade`) instead of running `go install`, with enough context (both revisions) for the CLI to render a clear message. Any other exit/error → fails open, proceed.
- [x] 4.4 Wire `ErrLocalDowngrade` rendering in `internal/cli/update.go` (nonzero exit, clear message naming both revisions and the `--allow-downgrade` escape hatch).
- [x] 4.5 Run `targ test` and confirm all of task 3's tests pass (GREEN).

## 5. Spec, docs, and final verification

- [x] 5.1 Add the `update-local-install-safety` capability spec (this change's `specs/update-local-install-safety/spec.md`) to `openspec/specs/` at archive time.
- [x] 5.2 Add a one-line mention of `--allow-downgrade` to README's `engram update` flag documentation.
- [x] 5.3 Run `targ check-full` and confirm all checks pass (aside from `check-uncommitted`, expected pre-commit).
