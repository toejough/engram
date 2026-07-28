## Why

`engram update` installs the new binary first (`resolveSource` → `go install ./cmd/engram/`, internal/update/update.go:370) and then the still-running old process performs all planning, harness sync, and vault checks with the previous release's logic. Any release that changes *how* update works (plan shape, deploy rules, new checks — e.g. the recent deploy-as-sync removal propagation) only takes effect on the run *after* the one that installed it, so users are always one update behind on update behavior itself.

## What Changes

- After a successful binary install, `engram update` re-execs the freshly installed binary to perform the sync/check phase, so those phases always run with the just-installed logic.
- The re-exec invocation carries a sentinel (flag or env var) meaning "install already done — sync only," preventing an install/re-exec loop.
- `--dry-run` continues to skip the install and therefore never re-execs.
- If re-exec fails (e.g. installed binary missing or not executable), update falls back to completing in-process with the old logic and reports the fallback.
- No change to what the sync/check phases do — only which binary executes them.

## Capabilities

### New Capabilities
- `update-self-reexec`: the self-update lifecycle of `engram update` — install-then-re-exec ordering, loop-guard sentinel, dry-run exemption, and in-process fallback on re-exec failure.

### Modified Capabilities

<!-- none — update-deploy-sync requirements (what sync does) are unchanged; this change governs which binary runs them -->

## Impact

- `internal/update/update.go` — `Updater.Run` split/ordered around the re-exec boundary; re-exec through an injected exec primitive (DI: no direct `syscall.Exec`/`os` calls in internal/).
- `internal/cli/` + `cmd/engram/main.go` — new raw exec primitive wired through `cli.Primitives`/`cli.NewDeps`; sentinel flag/env plumbed into the update target.
- `internal/update` tests — re-exec ordering, loop-guard, fallback, and dry-run contract tests.
- User-visible: single `engram update` run now applies the new release's sync behavior immediately; report notes when a re-exec happened.
