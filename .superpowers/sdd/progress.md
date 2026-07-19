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
