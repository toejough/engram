# DISPATCH HEADER (orchestrator)

- Worktree: `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity` (branch `worktree-700-internal-purity`). Work ONLY here — never cd to the main checkout.
- BASE-T-polish: <SET AT DISPATCH — after T17 ACK>. Constraints mirror: `.superpowers/sdd/constraints-and-resolutions.md`.
- This task consolidates the review-ledgered polish items accumulated across T4/T12/T15 Gate B verdicts. It runs BEFORE T-final-1 so the enforcement flip validates the final tree. It is behavior-neutral by contract: NO error-text changes, NO signature changes visible to callers, NO test-assertion changes except where a moved helper makes an assertion's subject move with it.

### Task T-polish: final consolidation — wrap-closure dedup + comment scrubs

**Items (each independently verifiable):**

1. **vocab_commands.go `WriteSidecar` closure → `writeAtomicFromFS(d.FS, "write")`** (T12 Gate B): the hand-rolled closure is byte-identical in behavior to the existing helper (deps_compose.go:152's writeAtomicFromFS with atomicFilePerm=0o600 and "write: %w" wrap). Verify byte-identical error text BEFORE swapping: read both, confirm `fmt.Errorf("write: %w", err)` over WriteFileAtomic at 0o600 in both. Swap, delete the local closure + its `sidecarPerm` const IF now unused (rg first — T4 lesson).
2. **embed.go `Write` closure → `writeAtomicFromFS(d.FS, "write")`** (T15 Gate B): same class, same verification, same swap. Delete the local const if orphaned.
3. **`writeAtomicFromFSWithPath` extraction — amend.go + resituate.go ONLY** (T12 Gate B): their Write closures are byte-identical (`WriteFileAtomic(path, data, 0o600)` then `fmt.Errorf("write %s: %w", path, err)`). Extract ONE helper in deps_compose.go (name: `writeAtomicWithPathFromFS(fsys EdgeFS) func(string, []byte) error`, or match the existing helper naming convention — read deps_compose.go's neighbors and conform); point both constructors at it. Do NOT touch prune.go (different label text "prune: writing %s: %w") or ingest.go (MkdirAll variant) — their shapes are variants, not duplicates; leave them.
4. **Comment scrub — primitives.go:94** (T15 ledger): the doc comment mentions the deleted "sharedEmbedder singleton"; reword to describe the current single-composition reality (Deps.Embed composed once inside NewDeps). Locate by text.
5. **Comment scrub — ingest_test.go R7 note** (T4 Gate B): the comment cites "testFlocker", a symbol that never existed (T8 substituted the production locker). Reword to "the production FileLocker composed over real OS primitives". Locate by text (rg "testFlocker" — expect exactly this one hit plus possibly adapters_test.go:15's historical mention; reword BOTH if both name the phantom, report what you found).

**Constraints:**
- Behavior parity is the spec: after items 1–3, `targ test` must pass with ZERO test edits (error texts unchanged). If any test fails, you changed behavior — revert and report, do not adapt the test.
- T4 lesson: rg every symbol you delete/rename across internal/ + cmd/ INCLUDING _test.go first.
- Gates FOREGROUND: `targ test`, `targ check-full`, `targ check-thin-api`. lint-full 0; residual set: e2e-under-load coverage flake (re-run check-coverage-for-fail standalone) + the 2 protected dev/eval reorder fixtures.
- reorder-decls HAZARD: UNSCOPED — restore the 2 protected fixtures by explicit path if run.
- Stage EXPLICIT paths only. One commit:

```
refactor(cli): consolidate atomic-write wrap closures, scrub stale comments (#700)

vocab WriteSidecar and embed Write ride the existing writeAtomicFromFS
helper (byte-identical wraps); amend/resituate share a new
path-labeled variant. Behavior-neutral: zero error-text changes, zero
test edits. Scrubs two comments naming symbols that no longer (or
never) existed (sharedEmbedder singleton, testFlocker).

AI-Used: [claude]
```

- REPORT: `.superpowers/sdd/briefs/T-polish-report.md` BEFORE your final message — status, commit SHA, verbatim gate outcomes, per-item verification evidence (the byte-identical checks), deviations, concerns. Final message: STATUS line, SHA, summary, concerns.
