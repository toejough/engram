## Context

`Updater.Run` (internal/update/update.go:350) resolves the source and installs the new binary via `go install ./cmd/engram/` (update.go:370, local-clone mode at :1164, temp-clone remote mode at :1129) *before* planning and applying harness sync, then the old in-memory process finishes the run with its own — now stale — sync/check logic. There is no re-exec or self-restart mechanism anywhere in the repo. The CLI wrapper `runUpdate` (internal/cli/update.go:358) adds vault/vocab/chunk-index checks after `Updater.Run`.

Constraints: DI-everywhere (internal/ may not touch `os.*`/`exec` directly — new I/O goes through injected interfaces wired in `cli.NewDeps` from `cli.Primitives` supplied by cmd/engram/main.go, which is `targ check-thin-api`-enforced), pure Go, imptest/rapid/gomega test stack.

## Goals / Non-Goals

**Goals:**
- The sync/check phase of `engram update` always executes in the freshly installed binary when an install happened.
- No install/re-exec loop; single user-visible command; exit code and output faithfully reflect the phase that ran.
- Graceful in-process fallback when re-exec is impossible.

**Non-Goals:**
- Changing what sync/checks do (update-deploy-sync spec unchanged).
- Adding `git pull` to local mode (still builds the current checkout).
- Windows-specific binary-replacement handling (darwin/linux are the deployed targets).

## Decisions

**D1 — Re-exec as child process, not `syscall.Exec`.** After a successful install, the old process spawns the installed binary as a child with stdio inherited, waits, and exits with the child's exit code. Rationale: portable, testable through the existing `Cmd`-style runner interface, and keeps the DI seam trivial (`syscall.Exec` never returns, which is hostile to unit tests and to the thin-API pattern). Alternative rejected: process-image replacement via `syscall.Exec` — marginally cleaner process tree but untestable and unix-syscall-specific.

**D2 — Loop guard via environment sentinel `ENGRAM_UPDATE_REEXEC=1`, not a CLI flag.** The child runs `engram update <original args>` with the sentinel set in its environment; a set sentinel makes the child skip `resolveSource` (no install, no further re-exec) and run sync/checks only. Rationale: keeps the sentinel out of the public CLI surface (a hidden `--sync-only` flag would be discoverable, shell-history-visible, and invocable by users in ways we'd have to define semantics for). Env read/set goes through the injected primitives (env-getter already exists for `$HOME` resolution; extended, not bypassed).

**D3 — Re-exec whenever an install ran and succeeded; no version comparison.** `--dry-run` skips install and therefore never re-execs; a failed/skipped install also never re-execs. When install succeeds we re-exec unconditionally rather than comparing `BinaryVersion` to the running build. Rationale: deterministic, and the no-op case (binary unchanged) costs one process spawn of a local binary — not worth a version-equality code path plus its edge cases (dirty local builds share short-HEADs with different bytes).

**D4 — Re-exec target is `resolveBinaryPath` (update.go:2375), the installed path — never `os.Args[0]`.** The running image may be a stale path or a different location; the point is to run what `go install` just wrote. If that path is missing or fails to spawn, fall back per D5.

**D5 — Fallback: on re-exec failure, complete in-process and say so.** Spawn error → log a warning into the report ("re-exec failed, completed with pre-update logic: <err>") and continue with the old code path exactly as today. Rationale: an update that installed a new binary but failed to re-exec is still strictly better than an aborted update; next run is fresh anyway.

**D6 — Report attribution.** The child's report notes it ran re-execed (sentinel present + recorded `BinaryVersion` handed over via the environment or re-derived); the parent contributes only install output before handing off. Avoids double-printing the sync report.

**D7 — Split point is `Updater.Run`, not the CLI wrapper.** The re-exec happens inside `Run` immediately after a successful `resolveSource`; everything after (plan, applyOps) plus the CLI wrapper's vault checks then run only in the child (parent exits with child's code before reaching them). The sentinel check lives at the top of `resolveSource`.

## Risks / Trade-offs

- [Child inherits a mutated environment / recursion if sentinel is dropped] → sentinel set explicitly on the child's env slice; a test asserts the child invocation env contains it and that a sentinel-bearing run never calls the installer.
- [Exit-code or signal fidelity: child killed by signal] → runner maps signal-death to non-zero exit; parent propagates the child's exit code verbatim.
- [Double execution of side-effectful pre-install steps (harness detection runs in both parent and child)] → harness detection is read-only; acceptable. Anything write-bearing stays strictly after the re-exec boundary.
- [Local mode installs whatever is checked out, so "fresh" logic may still be old if the clone is behind] → out of scope (Non-Goal); behavior unchanged from today and re-exec is still correct (runs what was installed).
- [macOS replacing a running binary on disk] → safe on unix (inode unlink semantics); parent keeps running from the old image and the child opens the new file.

## Migration Plan

Pure code change; no data or vault migration. Rollback = revert commit. First run after shipping this change still executes the *previous* binary's in-process logic (the bootstrap run itself is the last stale one, by construction).

## Doc-surface enumeration (grep disposition list)

Grep: `go install|resolveSource|update flow|engram update` over docs/, README.md, CLAUDE.md, agent-instructions/, openspec/specs/.

| File | Disposition | Reason |
| --- | --- | --- |
| docs/architecture/adr.md | update | new ADR (install-then-re-exec) continuing ADR-0022 |
| docs/architecture/c1-system-context.md | update | update-flow sequence diagram + R5 prose describe install→in-process sync; also fix stale `go install @latest` remote-mode description (code is clone-based per #645) |
| docs/architecture/c2-containers.md | update | C2→S6 edge (line 30/54) states "go install, then syncs" — add re-exec step |
| docs/architecture/c3-components.md | update | K9 row + PU subgraph label state the same invariant |
| docs/GLOSSARY.md | update | `engram update` entry describes the flow |
| README.md | update | lines 40/101 describe update flow; one clause on re-exec |
| CLAUDE.md | keep | directory descriptions only, no flow invariant |
| agent-instructions/guidance/*.md | N/A | no update-flow content (grep hits are unrelated uses of "update") |
| docs/superpowers/plans/*.md | N/A | historical plan records, not living docs |
| openspec/specs/update-deploy-sync/spec.md | keep | sync semantics unchanged; new capability gets its own spec on archive |
| docs/ROADMAP.md, memory-invariants.md, memory-system-rigor.md | keep | tangential mentions, no install-order invariant |

## Open Questions

None blocking. (Deferred: whether remote mode should also `git pull` a stale local clone — separate change if wanted.)
