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
Task 6: implemented (commit fe2427ac), review pending
- R11 amendment needed: query cluster (T6) consumes the stubEmbedder local-override pattern EARLIER than T14/T15 (RunQuery derefs Embedder unconditionally); T14/T15 briefs must not assume first-use
- New flaky watch: TestForceExit_RealSignalDeliveryThroughForwardAsPulses panicked once under check-full load, cleared on rerun (T1-rework signal integration; SIGUSR2 pacing)
- Third standing residual: vault_fs.go listDirBySuffix 76.9% coverage (pre-existing from T5 fix round; dies at T7 — fold into T7)
- T6 Minor: TestTargets_QueryEmptyVault hand-inlines executeForTest's body (no deps-override hook exists); T15 needs the same shape — consider adding an executeForTestWithDeps helper AT T15 rather than duplicating a third time
- T15 note: query cluster's stub needs SUCCEEDING Embed (RunQuery embeds per phrase); T15's ModelID-only sites can use the fail-loud stubEmbedderForTargets per R11 — two different stub needs, don't conflate
Task 6: complete (commits fe2427ac + f52df6de walker extraction, task review APPROVED, Gate B APPROVED after fix round)
- listDirBySuffix coverage gap CLOSED (100%, two legacy-path branch tests; they die with their subject at T7 via the grep gate)
- New watch: check-coverage-for-fail flakes under full-parallel check-full load (test-run failure not threshold; same family as e2e timeout) — if it persists past this cycle, file an issue
BASE-T8: f52df6de99fca891702f90a38e0d1c29063827db
Task 8: implemented (commit 3763e684), task review APPROVED; Gate B pending
- T9 fold-in: cli.go:119 osManifestLock doc comment stale ("newOsIngestDeps" gone) — T9 deletes the whole func anyway
- Minor (style): TestIngestDepsReadTranscriptReadsViaFS lacks the house post-Expect err guard (value struct, not a nilaway issue)
Task 8: complete (commit 3763e684, task review APPROVED, Gate B APPROVED — one minor folded into T9: cli.go osManifestLock stale comment, moot at its T9 deletion)
BASE-T9: 3763e684aff6294a9415edd972251b49da0eed3a
- T9 Minor (pre-existing convention, both ingest+prune): WriteFile closures double-prefix command names over their callers' wraps (prune: writing manifest: prune: writing /path:) — harmless to errors.Is; polish candidate for final review
Task 9: complete (commit b2c256b5, task review APPROVED, Gate B APPROVED)
BASE-T10: b2c256b53657c1b90b83000c75b19d5bb8befec8
Task 10: complete (commits b54f436b parity tests + 80430e8f reorder-amend, task review ACK; Gate B N/A — tests-only, no refactor phase, skip named explicitly)
- All three parity tests GREEN on arrival — no writesafe-parity defect in the internal atomic-write composition
- T13 ledger CORRECTION (verified by implementer AND reviewer independently): atomicWriteFile has 5 remaining production callers, not the brief's 6 — cli.go:144 (osLearnFS.WriteSidecar → was T4's) already absorbed by T3's writeAtomicFromFS composition. Remaining: amend.go:351, resituate.go:169, activate.go:136, vocab_commands.go:1217 (→ T12) + embed.go:164 (→ T15). T13's gate expects 5 migrations; T4 has NO atomicWriteFile obligation left. Zero hits in cli.go/learn.go/qa.go.
- T11 brief amendments (orchestrator-verified against landed tree): plan's TestNewVaultFS_ReadFile_WrapsErrorWithPath is STALE (predates T5 wrap fix; landed wrap is "vault read: %w", no path) — brief binds the distinct-verb assertion instead; plan's fake names (fakeEdgeFS/fakeDirEntry/fakeLocker) collide with nothing in package cli (existing ones are cli_test-side)
BASE-T11: 80430e8ff48898e9cec586319e038f3ff9db6ce8
Task 11: dispatched
Task 11: complete (commit 16f1d2df, task review ACK; Gate B N/A — tests-only, skip named explicitly)
- Deviations verified TRUE by reviewer against linter configs: (1) testdeps_internal_test.go rename forced by testpackage default skip-regexp '(export|internal)_test\.go'; (2) //nolint:gosec dropped — gosec is path-excluded for _test.go, directives were nolintlint-flaggable
- Primitives literal reconciled field-by-field vs primitives.go AND main.go's production literal — behaviorally identical closures, StartSignalPulses nil per SIG-1, no wrapping on raw returns
- NEW STANDING HAZARD (added to dispatch notes): targ reorder-decls has NO path scoping — rewrites the 2 protected dev/eval please_step3_probe fixtures; executors must git-restore them by explicit path after any run
- ExportNewTestOsDeps consumer-less until T12 (expected); unused-linter did NOT fire
BASE-T12: 16f1d2dfdd68701dbc6507d63bd8de8a4b9da30c
Task 12: dispatched
Task 12: complete (commit 99f309fe, task review ACK, Gate B APPROVED — no fix round needed)
- All 3 implementer deviations validated on merits: legacy TestOsVaultFS tests kept (die at T7), TestOsVaultFS_ReadFile_MissingPathError added (osVaultFS live via embed.go:156 until T15), vocab-propose warning pinned (visibility fix — warning previously leaked to real stderr)
- R11-WARNING-ROUTING CLASS (dispatch note for T14/T15/T16/T17): flipping a family to logWarningTo(d.Stderr) makes formerly-process-stderr warnings visible to test assertions — expect empty-stderr assertions to flip; pin the exact warning text
- Gate B ledger items for FINAL CONSOLIDATION pass (both non-blocking, brief-baked not executor error): (1) vocab_commands.go WriteSidecar closure ≡ writeAtomicFromFS(d.FS, "write") — zero-risk one-liner swap; (2) wrap-with-path Write-closure class now at 3+ sites (amend.go:351/resituate.go:168 byte-identical, prune.go:106 label variant, ingest.go:513 MkdirAll variant) — writeAtomicFromFSWithPath extraction candidate, T6 walker precedent
- T7 BRIEF FIXED pre-dispatch (Gate B teeth item): vault_fs_test.go added to Files (3 legacy tests die with subject), step-1 gate expected-hit set widened, listDirBySuffix grep-gated deletion added — unfixed, T7's gate would false-STOP
- Brief-hygiene nit (task review): T12 brief step 6.5 named nonexistent osTestEdgeFS{}; landed realFSForTest() (correct established helper) — unlisted deviation, harmless
- atomicWriteFile callers post-T12: exactly 1 (embed.go:164 → T15), matching corrected T13 ledger
BASE-T4: 99f309fe57923c1047a37f0e52791325a94a0bc9
Task 4: dispatched
Task 4: complete (commit 51f2c04e, task review ACK, Gate B APPROVED — no fix round)
- cli.go now pure: 72 lines, imports errors/fmt/io/fs only; retained set intact; ADR-0013 lock chain verified by wiring-read (primLocker over injected syscall prims, lock-at-Run*-entry, K1 + concurrent-writers confirmed EXECUTING real flocks via targeted -run)
- Deviation VALID: TestOsLearnFS_Lock_ExclusiveAcrossSecondAcquisition compile-forced out (brief under-enumeration, not a STOP signal); replacement TestRealFlockLocker_SecondLockWaitsForUnlock is strictly stronger (bounded wait vs unbounded)
- NEW DISPATCH NOTE: briefs can under-enumerate TEST-side consumers of deleted symbols — executors grep _test.go files for every symbol before deleting it; a missed consumer is a compile-forced deviation to report, not a STOP
- Gate B minor (final-pass wording fix): comment + commit message cite "testFlocker" which never existed — T8 substituted newTestDeps(...).Lock (production primLocker, stronger); reword ingest_test.go R7 note to "production FileLocker composed over real OS primitives"
- Gate B judgment call (documented, accepted): primLocker bad-path open failure covered via fake injection only (repo test-categorization doctrine: wrap logic via mock, primitive is a declaration-free one-liner)
- T-final-1 false-positive check CLEARED: loser-symbol grep list does not include the deleted names; the 2 historical comment mentions are in _test.go, outside the purity rule
BASE-T14: 51f2c04ecbb1f44262b43b8294f25cb8d7438c00
Task 14: dispatched
Task 14: complete (commit 2e5388bd, task review ACK, Gate B APPROVED — no fix round; 8 deviations all accepted on verified merits)
- internal/embed purified: zero os/hugot imports outside tests; cmd/engram/hugot.go = empty struct + 2 checker-thin methods (check-thin-api PASS, "All 2 public API files"); E-1 closure honors the doctrine content rule INDEPENDENT of the checker gap (Gate B AST-traced targ's own checker source); E-2 single new field; E-3 two-tier exist-classification with real-OS dir-over-dir pin
- 4 production nolints ACCEPTED: wrapcheck ignore-globs cover only internal/* (cmd genuinely fires), single-line + contract-citing, S-1 precedent (main.go:53) consistent; forcetypeassert has no config exclusion
- Destroy-on-failure lifecycle owned internally, no session-leak path (Gate B traced all error branches)
- sharedEmbedder bridge quarantined: sole production consumer newOsEmbedDeps (dies at T15); deadcode contingency written into T15 header (fold-or-leave rule)
- -short smoke: skip verified, unskipped PASS 0.3s; targ's -short behavior unknown (black box, honestly flagged); real-binary learn probe produced genuine 384-dim sidecar
- NOTE for Joe: step-11 smoke required go install — the GLOBAL engram binary is now this worktree's build (ahead of main until merge)
- BundledModelFS: brief-mandated export, zero production callers — future surface-trim candidate
- T-final-1: no new loser symbols needed; depguard default-deny already covers hugot-import regression
BASE-T15: 2e5388bd07ccbee4b12ed79dd53c1e55f3890ecf
Task 15: dispatched
Task 15: complete (commit 742d80ed, task review ACK, Gate B APPROVED — no fix round)
- MIGRATION MILESTONE: all 16 internal/cli constructors now compose from Deps; zero newOs*Deps anywhere; remaining os/syscall production imports are EXACTLY the 3 sanctioned-until files: writesafe.go (T13), vault_fs.go (T7), update.go (T16/T17) — this is T-final-1's flip-readiness inventory
- Bridge FOLD fired + validated: lint-full unused-var was real; sharedEmbedder/bridgeEmbedder/wireSharedEmbedder/ExportNewBridgeEmbedder/embed_bridge_test.go all gone; one-embedder-per-process property PRESERVED BY CONSTRUCTION (Deps.Embed composed once inside NewDeps, main calls NewDeps once)
- executeForTestWithDeps landed; T6 hand-inline collapsed onto it; R11 two-stub doctrine verified (fail-loud stub never Embed-reached on ModelID-only sites)
- T7 brief amended AGAIN pre-dispatch (Gate B): FOUR legacy tests now (T15's coverage-forced TestOsVaultFS_RoundTrip_ListMDAndReadFile joins), gate expected-hit set updated, deps_compose.go:97 comment reword folded into T7
- FINAL-CONSOLIDATION ledger item grew: embed.go Write closure ≡ writeAtomicFromFS(d.FS, "write") — second instance of the T12 vocab WriteSidecar class (distinct from the writeAtomicFromFSWithPath candidate: amend/resituate/prune/ingest)
- Comment residue ledgered: primitives.go:94 "old sharedEmbedder singleton" — T-final-1 doc scrub
BASE-T7: 742d80ed9f9ed571835dce06045305f7de79bf09
Task 7: dispatched
Task 7: complete (commit b3a0f925, consolidated dual-mandate review ACK — task review + Gate B folded into one sonnet reviewer, consolidation named: small grep-gated deletion whose target state was pre-validated by T15 Gate B's milestone sweep)
- osVaultFS + listDirBySuffix + 4 legacy tests + shim gone in one commit; vault_fs.go pure (fmt + path/filepath only); vaultFS coverage held via T11 contract tests + real-FS blackbox suite (83.5% gating run)
- Both compile-forced deviations VALID: errors/io-fs/strings imports died with listDirBySuffix per the brief's own gate logic; shim line drift cosmetic (body verbatim)
- Impurity residual now EXACTLY: writesafe.go (T13) + update.go (T16/T17)
BASE-T13: b3a0f9257de1e04f81f31c9e21b3c9963913f0ad
Task 13: dispatched
Task 13: complete (commit 31cf7bb6 — amended from c570474b pre-review, message-only fix of the phantom testAtomicWrite clause, tree verified identical; consolidated dual-mandate review ACK)
- writesafe machinery gone (−265); ADR-0013 atomic-rename edge now SINGLE-IMPLEMENTATION (primFS.WriteFileAtomic); all 7 surviving atomic-write tests directly executed green; concurrent-manifest regression rides the PRODUCTION dance via realFSForTest (stronger than the brief's stale description)
- internal/cli impurity inventory for T-final-1: EXACTLY update.go:10 (dies at T16/T17)
BASE-T16: 31cf7bb6
Task 16: dispatched
Task 16: complete (commit e6a6efc5, consolidated dual-mandate review ACK)
- Sentinel + cutover byte-for-byte per brief; internal/update fully pure incl. tests; no user-visible stutter (both not-found paths re-wrap clean); T17 swap point localized to ONE errors.Is block (update.go:56-59)
- REVIEWER PROCESS INCIDENT (recovered + independently verified): T16 reviewer ran bare git stash/pop, briefly popped another session's parked #686 stash; reverted via reset --hard; orchestrator verified stash stack intact (both entries), tree clean, HEAD correct; reviewer /learn'd the lesson to the vault (git-stash-pop-not-a-safe-noop-pair)
BASE-T17: e6a6efc5
Task 17: dispatched
Task 17: complete (commit 1c8382ff, task review ACK + Gate B APPROVED — no fix round; all deviations accepted incl. the funlen tuple-compression, 21 inert nolints proven by lint-full silence, pwd -P re-verified empirically)
- INTERNAL/ FULLY PURE: the wide sweep (os/syscall/os-exec/os-signal/net/net-http/database-sql, non-test) is EMPTY — T-final-1 green light confirmed by three independent agents
- C-1 landed: RunCommand tuple-compressed closure (doctrine-cap preserved, mirrored in realPrimitives), NotFoundErr as bare sentinel VALUE (zero cmd logic); translation owned by internal primCommander; nil-NotFoundErr never-translates pinned by test
- FUNLEN DECISION FOR JOE (Gate B recommendation, NOT executed): main() at exactly 60/60 — next Primitives field trips it. Recommended: path-scoped funlen exclusion for cmd/engram/main.go with a why-comment (style gate, not purity; zero-carve-outs AC covers depguard/forbidigo only). Config change = Joe's call; surfaced in completion report. Nothing in #700's remaining scope trips it.
- FOLLOW-UP ISSUE CANDIDATE (Gate B finding): the production RunCommand closure in main.go is guarded only by its byte-identical realPrimitives mirror — no end-to-end path reaches Cmd.Run (update absent from cli_test.go; local-mode dry-run stops at resolveSource; same pre-existing class as OpenDebugFile/ENGRAM_DEBUG_LOG). File gh issue at completion: end-to-end update test reaching Cmd.Run (fake git on PATH or non-module cwd) or corrected doctrine claim.
- T17 implementer stall mid-task (no process, 25min) — SendMessage nudge recovered it; claims gates ran foreground
- Reviewer git incidents this cycle: T16 reviewer bare stash/pop (popped foreign #686 stash; recovered; /learn'd), T17 task reviewer detached HEAD (recovered) — orchestrator re-verified tree/branch/stashes after each
BASE-T-polish: 1c8382ff
Task T-polish: dispatched
Task T-polish: complete (commit 06d787fb, dual-mandate review ACK)
- All 5 items byte-identical/accurate verified against pre-commit tree; ZERO test-behavior edits; helper writeAtomicWithPathFromFS conforms to *FromFS convention, exactly 2 consumers (amend/resituate); prune/ingest/activate variants correctly untouched
- Non-blocking residue: writeAtomicFromFS doc comment phrasing slightly under-inclusive (still accurate); activate.go no-wrap closure noted as genuine variant
BASE-T-final-1: 06d787fb
Task T-final-1: dispatched
Task T-final-1: complete (commits 352569c4 comment-scrub + 13631846 enforcement flip, dual-mandate review ACK — reviewer re-ran the negative probe live)
- ENFORCEMENT ACTIVE: depguard internal-purity default-deny ('**/internal/**' + '!$test', R9 fallback — root-anchored glob empirically never fires) + forbidigo six-entry custom list + max-issues=0; zero carve-outs verified byte-by-byte; probe fires verbatim; testdata over-match question answered (structurally excluded by go/packages)
- Step 5.5 = call (a): cmd/engram entry-point-excluded (coverage denominator './, ./internal/...' structural in targ); recorded in commit body
- Issue #700 AC amended per R9 (body edit + evidence comment) — DONE by orchestrator
- FOR JOE (deliberate-call list, completion report): (1) funlen 60/60 on main() — Gate B recommends path-scoped exclusion with why-comment; (2) pre-existing issues.fix=true in lint config — auto-fix can silently modify files during lint runs, revisit deliberately
BASE-T-final-2: 13631846
Task T-final-2: dispatched
Task T-final-2: complete (commit a04f3541, verification review ACK — 5-line comment-only deletion, gates identical, Go-scoped grep zero)
STEP 4 (EXECUTE) COMPLETE: T1-rework, T2, T3, T5, T6, T8, T9, T10, T11, T12, T4, T14, T15, T7, T13, T16, T17, T-polish, T-final-1, T-final-2 — all review-ACK'd; enforcement active; FIXME resolved per the user's rule
STEP 5 (DOCUMENT) BEGINS: disposition table in T-final-2-brief.md governs
Step 5 (Document): commit 13940290, Gate C APPROVED — all claims tree-verified, ADR-0020 substance preserved, K13 verified next-free, mermaid mirrored; rulings: c1 runIngest case-drift = pre-existing (2026-06-19), fixed standalone post-gate; K11 signal.go drop correct (old citation now false; K13 carries it)
Step 6 (Complete): pre-merge state verified — origin/main unmoved at cd374577 (no rebase needed, merge-base confirmed), scope check clean (122 files all in-scope, note 150), targ test + check-full green (sole FAIL = 2 protected fixtures), Gate D CHANGES REQUIRED → 2 corrections applied verbatim (SIG-1 comment-gap honesty + three-agents wording); Gate D verified Draft 2 fully accurate incl. catching T17-report's own end-to-end overclaim. Merging ff via push HEAD:main.
Step 6 (Complete): DONE — merged ff cd374577..acbc7470 (72 commits), pushed; #700 CLOSED with Gate D-approved comment; follow-up #703 filed (RunCommand e2e guard gap); binary reinstalled from merged tip; primary checkout dirty so its local main left for Joe to pull.
Step 7 (Capture): DONE — closing sweep ran (fresh session chunks embedded, vocab OK, QA gate 8/20); lessons audit crystallized vault notes 329 (true downstream briefs against landed tree at every boundary), 330 (subagent git-hygiene fences in shared worktrees), 331 (dry-run smoke claims must name the code path reached). Cycle complete.
Decisions B/C SUPERSEDED by root-cause direction (Joe, 2026-07-20): the uninspectable-closure problem is the checker's, not engram's — scaffolding (sanctioned lists, mirror doctrine, identity carriers, e2e guard question) was compensating for check-thin-api not walking closures + rejecting literal returns. Filed toejough/targ#23 (LLM-resolvable spec: linear-thin-body grammar for FuncLits everywhere in checked files + composite-literal returns; engram's closures as pass/fail test material — S-1 WriteFileExcl deliberately listed as must-FAIL, to be decomposed engram-side once the checker lands). Joe resolves it in targ, then returns. ON HIS RETURN: bump go.mod to new targ; expect S-1 to fail the new checker → decompose (OpenFileExcl single-call eraser + error-merge logic internal); delete the four New*Prims identity carriers (literal returns now legal); re-evaluate the mirror-test doctrine + DRIFT text against machine-checked shape; then the B question reduces to optional behavioral e2e (fake-git) — Joe's call; then full closure bar re-check → re-close #700.
RESUME 2026-07-22 (targ#23 resolved same day; Joe: "resume the cleanup work in the 700 worktree"):
Task R1 (deps-gate probe): complete (commit 947aaaa5) — pin bumped to v0.0.0-20260722231854-0436d450c836; module pin alone activated the closure-walking checker (no binary reinstall — confirms targ#23's pickup claim); RED exactly as predicted: check-thin-api sole FAIL on the S-1 error-merge compound condition (main.go:84); fixture-reorder (c61d391a) and funlen (aa134a2c) fixes verified holding under new targ — reopen items 1 and 2 resolved.
Task R2 (S-1 decomposition): complete (commits 0e9a1b13 + f82354a8 + c8e42502 + 92bb8e43; three-lens review: behavior ACK first pass — old-closure semantics preserved exactly incl. ignored Write n; doctrine+test lenses CHANGES-REQUIRED → two same-tier fix rounds: AI-Used trailer invariant x2 (lesson crystallized, vault note 354), nolint justification restored, Close-on-write-error + short-write pins added, MatchError conformance) — FSPrims.WriteFileExcl → OpenFileExcl io.WriteCloser single-call eraser; error-merge internal in primFS, fake-tested; EdgeFS contract + fs.ErrExist chain unchanged.
Task R3 (carrier deletion): complete (commit 66b96132, fresh-review ACK 6/6, zero fix rounds) — four New*Prims identity carriers deleted, 11 call sites inlined with literals byte-identical, −44 lines.
Task R4 (doctrine re-base): complete (commit c34dc7b7, amended once — orchestrator ground-check caught c3 K13 + ADR-0020 carrier-era phrasing the agent's inventory missed) — DRIFT text now honest (e2e binary tests guard FS/lock only; C-1/OpenDebugFile mirror-only, OPEN on #700), ADR-0020 states exactly two multi-statement closures (C-1, SIG-1; S-1 now a one-line eraser), SIG-1 sanction comment normalized to C-1 idiom.
Gates at c34dc7b7: 8/8 PASS (check-thin-api: "All 2 public API files are thin wrappers"). origin/main at 8ee17408 = merge-base → branch is ff-clean, no rebase needed.
OPEN FOR JOE: (a) RunCommand/OpenDebugFile behavioral-e2e disposition (reopen item 3, folded back from #703); (b) merge + re-close call. Primary-checkout note: its two dirty sweeper-fixture files are reorder-fixer damage (doc comments destroyed), superseded by c61d391a — discard candidates after merge; Joe's tree, untouched.
Joe's dispositions (2026-07-22): (a) add BOTH e2e guards now; (b) merge ff + re-close once green.
Task R5 (e2e guards): complete after escalation — 680b60fd landed TestRunCommand_EndToEnd (fake git/go PATH shims, non-module cwd, marker proves production C-1 reached Cmd.Run) + TestOpenDebugFile_EndToEnd; cheap-tier fix round rationalized two failing gates → verifier reproduced the flake → escalated to mid per route ladder; root cause: the sink test spawned query subprocesses (~5-7s model load each, 4th concurrent load past the 30s package timeout under coverage) + empty-env negative control (stripped HOME/GOCOVERDIR); fixed by --help (~15ms) + envWithoutDebugLog() (c19df1f7/92ef2179 post-rewrite), t.Parallel kept (cost fixed, not hidden).
INCIDENT + REMEDIATION: the guard commit's original form (d5ea148b) swept internal/cli/.cache (90MB test-materialized model) into pushed main — survived three fresh reviews + verifier + orchestrator (scope checks tested subset, not set equality; lesson → vault note 360). Joe approved rewrite: history rebuilt without the six cache files (verified zero .cache paths in range), prevention commit 96acc76e pins subprocess cache to t.TempDir() via XDG_CACHE_HOME (code-verified: CacheDirFromHome, internal/cli/targets.go) + .gitignore .cache/; lease force-push f5ef2452→96acc76e.
Step 6'' (Complete): origin/main 8ee17408→96acc76e; #700 CLOSED with evidence comment; binary reinstalled from merged tip (smoke ok). Five consecutive clean 8/8 check-full sweeps across the landing sequence; trailer audit clean on the full range.
Step 7'' (Capture): vault notes 354 (briefs state commit invariants), 359 (wrong-cwd targ pin-sync), 360 (scope review = set equality) + closing crystallizations; route evidence updated across six work-kinds. Resume cycle complete.
