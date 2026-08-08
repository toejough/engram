## Context

`resolveSource` (`internal/update/update.go:1219`) picks local vs. remote install mode by walking up from `cwd` for a `go.mod`. Remote mode (`resolveRemoteByClone`, line 1170) clones the repo fresh every time and resolves a git SHA post-clone — a stale binary is structurally impossible there since the clone is always at the repo's current default branch tip. Local mode has no equivalent safety net: it trusts whatever `cwd`'s ancestor module root happens to contain, which could be a long-stale clone, an old worktree, or a detached-HEAD checkout mid-bisect.

`resolveBinaryPath` (line 422 in `Run`) computes the install target path *before* `resolveSource` runs (line 433), so there's an existing window to inspect the pre-existing binary before it's overwritten.

Go's toolchain auto-embeds VCS metadata into binaries built via `go install`/`go build` from a version-controlled module — no build flags required, this has been default since Go 1.18 (`-buildvcs=auto`). `go version -m <path>` reads that metadata back out, printing lines like `build vcs.revision=<full sha>` and `build vcs.modified=<bool>`.

## Goals / Non-Goals

**Goals:**
- Local mode reports a git revision in the update report, matching remote mode's existing behavior.
- A *provable* downgrade (new commit is not a descendant of what's installed) is refused by default, with an explicit `--allow-downgrade` escape hatch.
- Every case where provability fails (no prior binary, pre-VCS-embedding binary, unknown commit, divergent history) fails open — proceeds exactly as today, degrading gracefully to visibility-only.

**Non-Goals:**
- No attempt to reconstruct history across truly divergent branches, rebase awareness, or "which came later in wall-clock time" heuristics — ancestry is the only signal used, and only when it's cleanly determinable.
- No change to remote-mode behavior — it already resolves and shows a revision, and re-cloning always means "install what's actually at the repo tip," so there's no analogous downgrade risk to gate there.
- No general "binary version management" feature (rollback, pinning, etc.) — scoped strictly to preventing a silent local-mode regression.

## Decisions

- **Ancestry check via `git merge-base --is-ancestor <installedRev> <resolvedRev>`, run in `root`** (the local module clone, not wherever the installed binary was originally built from — it's the only git history available to us). Exit 0 means the resolved commit descends from (or equals) the installed one — safe, proceed. Exit 1 means it does *not* — either a genuine downgrade, or divergent/unrelated history; both are treated as "provable downgrade" for gating purposes since we can't distinguish them and both mean "the thing about to be installed is not verifiably forward progress from what's running." Any other exit status (e.g. `installedRev` unknown to this clone — `fatal: Not a valid commit name`) means "can't determine" — fail open.
- **Revision capture order**: read the *pre-existing* binary's embedded revision via `go version -m <BinaryPath>` BEFORE calling `go install` (which overwrites it). If no file exists at that path yet (first-ever install), skip the check entirely — nothing to compare against, and gating would block the common case of installing on a fresh machine.
- **`--allow-downgrade` is a hard bypass, not a softer warning.** When passed, the install proceeds exactly as it does today (no check performed, or check performed but result ignored) — same as the current unconditional behavior, opt-in named explicitly so the flag's presence in a command line or script is self-documenting evidence someone chose to override.
- **New capability `update-local-install-safety` rather than folding into `update-self-reexec`.** The re-exec spec's Purpose is scoped to *post-install* lifecycle (loop-guard, handoff report, phase re-execution) — this change is about a *pre-install* safety gate, a different concern with a different failure mode. Keeping them separate avoids overloading one capability's Purpose statement and keeps validation ownership clean (this change's tests target `resolveSource`, not the re-exec sentinel).
- **`SourceInfo.Version`'s existing field is reused, not duplicated.** Its doc comment (`// resolved version string (remote only)`) just becomes inaccurate and needs a one-line update — no new field needed since local mode populating the same field remote mode already uses is exactly the parity this change wants.

Alternatives considered:
- *Warn instead of gate, always allow*: rejected per your explicit choice — visibility alone already exists as the weaker option and was the fallback you didn't pick.
- *Compare `vcs.time` (build/commit timestamps) instead of ancestry*: rejected — wall-clock time doesn't establish "this is worse code," and clock skew/rebases make it a noisier, less honest signal than ancestry, which directly answers "does the new build contain everything the old one did."
- *Block on ANY non-ancestor result, including truly divergent branches (e.g. legitimate feature-branch worktree)*: this is what the chosen design actually does — flagged here because it's a real trade-off, not an oversight. A developer running `engram update` from a feature-branch worktree to test their own changes will hit the gate if their branch hasn't merged `main`'s tip. That's arguably correct (you ARE installing something that doesn't contain main's latest), and `--allow-downgrade` is the intended escape hatch for exactly that case.

## Risks / Trade-offs

- [Risk] `go version -m` behavior/output format could vary across Go toolchain versions. → Mitigation: parse defensively — a missing `vcs.revision` line means "can't determine," fails open, never crashes.
- [Risk] A legitimate feature-branch worktree workflow now requires `--allow-downgrade` every time until merged. → Mitigation: this is the intended, disclosed trade-off (see Decisions); the flag exists precisely for this. Document it in the flag's `--desc` text and in README's update section.
- [Risk] Shelling out to `git merge-base`/`go version -m` adds two more subprocess calls to every local-mode update run. → Mitigation: both are fast, local, no-network operations; negligible relative to `go install` itself.

## Migration Plan

Single-PR addition, no rollback complexity — purely additive behavior gated behind a new check that fails open on any ambiguity:
1. Add revision resolution (git rev-parse) to local mode, update `SourceInfo.Version` doc comment.
2. Add pre-install binary-version capture + ancestry check + `ErrLocalDowngrade` sentinel + `AllowDowngrade` option threading.
3. Wire `--allow-downgrade` CLI flag.
4. Tests: gate fires on provable downgrade; fails open on no-prior-binary, unknown-commit, and pre-VCS-embedding cases; `--allow-downgrade` bypasses; revision shows in report unconditionally.
5. Update `openspec/specs/update-local-install-safety/spec.md` (new capability) — no existing spec is modified.

## Open Questions

None.
