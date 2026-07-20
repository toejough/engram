Task 1: pending
BASE-T1: 2ee344183e4a1a4622ba5db671e7bdf45ca2cfc8
Task 1-rework: pending
BASE-T1-rework: 3fd3a1866885febf0b76b19705cfe1b2deda66ec
Minor findings for final-review triage:
- T5 step 4 uses `d` where T2 lands `deps` (T5's own hedge covers it; executor must adapt)
- T8 step 3 anchor off-by-one: osFileReader block is cli.go:24-30, not 25-31 (content byte-exact)
- Consolidation candidate: cli_test accumulates 3 real-FS EdgeFS doubles (osEdgeFSForTest, osTestEdgeFS, osTestFS) + 2 deps builders (newTestDeps, testDeps) — compiles (names unclaimed) but triplicated harness machinery
- T4 anchor drift: acquireOptionalLock at cli.go:157 not 152-163 (symbol gates govern)
- T17 "13 adapter tests" is correct AT EXECUTION TIME (T16 adds the 13th; tree has 12 today) — do not "fix" the count
- T12 cites: vocab header at :726 not :737; resituate comment 27-28; nil-Embedder guard :834 (locate-by-text governs)
- R11 cite: tallyStates at embed.go:273 not :275
- T15 step 3's newOsQueryDeps edit is the CERTAIN skip path under R4 (T6 precedes); its skip-clause covers it
- Executor reminder: design flags at plan lines ~244/311/315 are superseded — always read through the supersession map (in every dispatch's constraints file)
Task 1-rework: implemented (commit b1ea7ca3), review pending
Task 1-rework: complete (commit b1ea7ca3, task review APPROVED, Gate B APPROVED)
BASE-T2: b1ea7ca3db478107b58c2be81bbff9d034e93dcb
Task 2: implemented (commit d946a145), review pending
Task 2: complete (commit d946a145, task review APPROVED, Gate B APPROVED)
- Minor (Gate B, polish): add inline flag comments (// P-2/P-3, // SIG-1) to the non-S-1 sanctioned closures in cmd/engram/main.go for reader-visible sanction parity
BASE-T3: d946a145701ca636410e1595273d7751a363e3d4
Task 3: implemented (commit d98b0ca0), review pending
Task 3: complete (commits d98b0ca0 + 240a2a95 fix, task review APPROVED, Gate B APPROVED after fix round)
- T5 reviewer watch (Gate B residual): newVaultFS.ListMD must NOT port "reading dir %s: %w" verbatim over EdgeFS — use the distinct-word/no-path shape ("list md: %w"); EdgeFS.ReadDir already carries the path
- Non-blocking note: newTestDeps now flows through NewDeps, so targets-tests get a real DebugLog sink iff ENGRAM_DEBUG_LOG is set in the env (production-faithful, benign)
BASE-T5: 240a2a955608e8b9cebcd07663118a8a2c7da36a
Task 5: implemented (commit 205e9acf), review pending; T3-fix fallout (3 findings from 240a2a95) queued
- T5 Minor (task review): vaultFS.ReadFile wraps 'reading %s' over EdgeFS's own wrap (double-wrap; pre-existing pattern; ListMD fix was scoped narrower) — consider distinct-word sweep when T7 deletes osVaultFS
T3-fallout: cleared (commit 8dda0d72 — lll, unparam→atomicFilePerm consolidation, reorder-decls)
Task 5: complete (commits 205e9acf + 5f0a1670 review fixes, task review APPROVED, Gate B APPROVED after fix round)
BASE-T6: 5f0a1670c3ffec24981bf6b40074933ebe93cc7f
ACCUMULATED DISPATCH NOTES (include in every implementer dispatch):
- threaded variable is `deps` not `d`; adapt brief snippets mechanically
- EdgeFS-layer error wraps: distinct-word/no-path ("list md: %w", "vault read: %w") — never repeat EdgeFS's verb+path
- test builders: newTestDeps(stdout,stderr) [flows through NewDeps] + realFSForTest(); realFSDepsForTest/osTestEdgeFS DELETED
- writeAtomicFromFS(fsys, opName) — perm param removed, atomicFilePerm inside
- gates run FOREGROUND (no background-run-and-yield); stage EXPLICIT paths only (never add -A)
- check-full residual set: e2e-under-load coverage timeout + 2 dev/eval reorder fixtures; lint-full 0
- reviewers: task reviewer (spec+quality, sonnet) + Gate B design-fit (sonnet) per task; fix rounds re-ACK with the same reviewer
