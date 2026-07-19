# #700 Internal Purity — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Zero I/O-capable imports in `internal/` non-test code: relocate all eight production I/O adapters to `cmd/engram` (package main), thread env/clock/FS capabilities from the edge via `cli.Deps`, and land depguard default-deny + forbidigo enforcement with no in-boundary carve-outs (issue #700).

**Architecture:** `cmd/engram` becomes the composition root: it implements `cli.EdgeFS`/`cli.FileLocker`/commander/embedder/debug-sink adapters and wires them into `cli.Targets(deps cli.Deps)`; `internal/` keeps interfaces + pure logic only. Enforcement is config-only (depguard allow-list default-deny + forbidigo call-level rules), landing as the FINAL task so every prior task keeps `targ check-full` green.

**Tech Stack:** Go, targ (build/check), golangci-lint v2 (depguard, forbidigo), imptest + rapid + gomega, git worktree `worktree-700-internal-purity`.

## Global Constraints

- NEVER run `go test`/`go vet` directly — use `targ test` / `targ check-full`; binary install is `go install ./cmd/engram` (no targ build target).
- Every new/changed test: `t.Parallel()` in parent AND every subtest; no shared mutable state between subtests; exported test functions before private constants/types (reorder-decls); gomega assertions with `if err != nil { return }` after error expectations (nilaway); wrap errors with `%w` context; line length < 120; named constants over magic numbers; `make([]T, 0, cap)` when size known.
- `fs.FileMode`/`fs.FileInfo`/`fs.DirEntry` come from `io/fs`, never `os`, in internal code.
- ADR-0013 (vault flock + atomic rename) semantics are safety-critical: lock acquisition stays at `Run*` entry points; the concurrent-writers regression test must survive every task; never weaken atomic-rename.
- Implement the validated recipe from issue #700 — do not redesign mid-task (vault note 238). Any forced departure is a DESIGN FLAG escalated to the orchestrator, not an improvisation.
- `internal/update/update.go` diff stays minimal (concurrent Pi worktrees touch it).
- Each task ends with `targ check-full` green (or `targ test` where the task says so) and a commit with an `AI-Used: [claude]` trailer.

## Design flags resolved at plan time

**Wiring-core (foundation — binding for all tasks):**

- DESIGN FLAG: The embed interface is `embed.Embedder` (internal/embed/embedder.go:54: `Embed(ctx, text) ([]float32, error)`, `ModelID() string`, `Dims() int`). There is no `embed.Backend`. Per the spec's own fallback clause, `Deps.Embed` is typed `embed.Embedder`.
- DESIGN FLAG: `resolveVault` (internal/cli/learn.go:443) and `ResolveChunksDir` (internal/cli/ingest.go:63) are ALREADY pure — `home` and `getenv` are injected parameters. Only their call sites in targets.go change (`os.Getenv` → `deps.Getenv`, `homeOrEmpty()` → `homeOrEmpty(deps)`). No edits to learn.go/ingest.go from this cluster.
- DESIGN FLAG: debuglog's per-line `file.Sync()` (debuglog.go:67) is load-bearing for `tail -F` liveness. `io.Writer` has no Sync, so the cmd sink must wrap `*os.File` in a per-write-syncing writer (`syncWriter` below), or behavior regresses.
- DESIGN FLAG: current `debuglog.New("")` returns a NON-nil `&Logger{}` no-op and callers rely on the `l.file == nil` guard. The new API returns a nil `*Logger` for a nil writer; a zero-value `&Logger{}` becomes invalid (nil `now` func would panic). All construction must go through `New` — the only production caller is internal/cli/main.go:23, which this cluster deletes.
- DESIGN FLAG (sequencing): targets.go closures call `newOsXxxDeps()` constructors owned by other clusters. T2 keeps those calls (they compile unchanged) and swaps only `os.Getenv`/`os.UserHomeDir`/`stdout` for deps fields; the `newXxxDeps(deps)` swap is one line per closure, owned by each command cluster (exhaustive table in T2 step 3). Similarly `runLearnFromFeedbackArgs`/`runLearnFromFactArgs` (learn.go:497,520) and `runUpdate` (update.go:276) construct their own os deps internally — their deps-param conversion belongs to the learn/update clusters.
- DESIGN FLAG (sequencing): the final cmd/engram/main.go wires `Commander: &osCommander{}` (adapter destined for cmd/engram/os_update.go, update cluster) and `Embed: embed.NewLazyEmbedder(...)` (hugot construction moves to cmd/engram/hugot.go, embed cluster). Both wiring lines compile at T2 time ONLY if sequenced as noted in T2 step 5: `embed.NewLazyEmbedder`/`embed.BundledModelID` exist today (internal/embed/hugot.go:149,18) so `Embed` wires now; `osCommander` does not exist in package main yet, so T2 omits the `Commander:` field line (nothing consumes it until runUpdate converts) and the update cluster adds it.
- DESIGN FLAG: `cli.Targets` has 8 test call sites, not 4 — targets_test.go:153,289,307,439 AND vocab_commands_test.go:3580,3606,3630,3655 (all `package cli_test`, so one shared helper fixes all).
- DESIGN FLAG: T1's cmd/engram/os_signal.go needs the pure `ForceExitOnRepeatedSignal(<-chan struct{}, func(int))` signature, so the signal.go signature conversion happens in T1 (with an interim adapter inside `SetupSignalHandling`, which T2 then deletes). Also: real-signal integration testing uses SIGUSR2 self-delivery; pending-signal coalescing can swallow a rapid second delivery, so the test paces re-sends until exit fires (over-delivery still means exit — no false pass).
- DESIGN FLAG: `lint` risk in T1 — `registerForceExit`/`openDebugSink` are unused by main.go until T2. Their `_test.go` files in the same package use them, which satisfies golangci's `unused` (tests included by default). If `targ check-full` still flags them, fold T1 and T2 into one commit rather than suppressing.
- DESIGN FLAG: EdgeFS adapter methods wrap errors with `%w`, which PRESERVES `errors.Is(err, fs.ErrNotExist)` chains that internal callers rely on (e.g. embed.go readerFS:181). Safe to wrap everywhere per house rules.

**Learn-family:**

- DESIGN FLAG: EdgeFS has no exclusive-create primitive. `osLearnFS.WriteNew` (cli.go:115, `O_CREATE|O_EXCL`, the documented K1 backstop) and `osLearnFS.WriteFileIfMissing` (cli.go:90, `O_EXCL` + swallow-`ErrExist`, runs BEFORE lock acquisition — two concurrent first-learns race on vault bootstrap) cannot be composed from the spec's EdgeFS: `WriteFile` truncates, and `Stat`-then-`WriteFile` is TOCTOU. This draft adds ONE method to EdgeFS: `WriteFileExcl(path string, data []byte, perm fs.FileMode) error` (must satisfy `errors.Is(err, fs.ErrExist)` when path exists). Fallback if rejected: `Stat`+`WriteFileAtomic` — preserves under-lock behavior but drops the O_EXCL backstop and reopens the pre-lock bootstrap race. Not recommended (ADR-0013 safety-critical).
- DESIGN FLAG: `LearnDeps.Getenv` and `LearnQADeps.Getenv` are dead fields — assigned in the constructors (learn.go:353, qa.go:267) but never read (vault resolution happens in targets.go via `resolveVault`). Kept and wired from `d.Getenv` for surface stability; candidate for a follow-up removal.
- DESIGN FLAG (ordering): `osLearnFS.Lock` has four consumers OUTSIDE this cluster (activate.go:123, amend.go:345, resituate.go:163, vocab_commands.go:1211); `flockPath` two more via `osManifestLock` (ingest.go:491, prune.go:107); `logWarningToStderrf` (defined in learn.go:332) is consumed by activate/amend/resituate/vocab/qa constructors; `osFileReader` (cli.go:27) is consumed by ingest.go:488. Task L1 therefore keeps `osLearnFS` (Lock-only), `flockPath`, `osManifestLock`, and relocates `logWarningToStderrf` into cli.go; Task L2 (the cli.go purge) MUST be sequenced after the activate/amend/resituate/vocab/ingest/prune constructor migrations.
- DESIGN FLAG (coordination, os_fs.go owner): cmd/engram `osFS.ReadDir`/`Stat`/`WriteFileExcl` must return errors satisfying `errors.Is(err, fs.ErrNotExist)` / `fs.ErrExist` (wrap with `%w` or return the raw `*fs.PathError`) — the learn compositions replace `os.IsNotExist` with `errors.Is`. Also: cmd integration tests must cover flock-on-unwritable-path error (replaces `TestOsLearnFS_Lock_BadVaultReturnsError`, deleted in L2) and `WriteFileExcl` → `fs.ErrExist` on existing file.
- DESIGN FLAG (coordination, ingest cluster): `TestManifest_ConcurrentWritersDoNotLoseEntries` (ingest_test.go:319) consumes `cli.ExportFlockPath`. L2 re-implements `ExportFlockPath` as a test-only real flock inside export_test.go (test files are exempt from enforcement), so that test survives byte-for-byte unchanged. `TestOsManifestLock_MkdirError` (testhelpers_test.go) dies with `osManifestLock` in L2 — the mkdir-before-lock behavior must be re-covered by the ingest cluster's replacement composition.
- DESIGN FLAG (coordination, targets/foundation cluster): this draft assumes the foundation task lands first (`internal/cli/deps.go` with `Deps`/`EdgeFS`/`FileLocker`, `cmd/engram` `osFS`+`flockLocker`, `Targets(d Deps)` threading `d` into `learnUpdateTargets`, and `executeForTest` in targets_test.go re-wired to a real-FS test Deps). The shared test doubles `osEdgeFSForTest`/`flockLockerForTest`/`realFSDepsForTest` created in L1 (testhelpers_test.go) are intended for reuse by targets_test and the other family clusters. Shared compose helpers (`statDirFromFS`, `listMDFromFS`, `logWarningTo`, `vaultLockFromLocker`, `writeNoteAtomicFromFS`) are declared ONCE in `internal/cli/deps_compose.go` by this task — amend/resituate/vocab/activate clusters must consume, not re-declare.
- DESIGN FLAG: cli_test.go's end-to-end tests (`TestEngramLearn_Fact_EndToEnd` etc.) build and run the real binary — they gate the cmd/engram wiring automatically and need no changes.

**Query-family:**

- DESIGN FLAG: `osVaultFS` (vault_fs.go:14) is consumed by SEVEN non-cluster files: amend.go:342, learn.go:349, embed.go:156, qa.go:262, resituate.go:160, vocab_commands.go:1213/1215/1237/1238, plus test shim `ExportNewOsVaultFS` (export_test.go:573) used by vocab_trigger_test.go:251,441 and vocab_commands_test.go (10 sites). Deleting it inside this cluster's window breaks their compile. Split: Task Q1 adds the pure `vaultFS` and migrates this cluster's three consumers; Task Q3 (purge) deletes `osVaultFS` + its `os` import and MUST be sequenced after those clusters migrate to `newVaultFS(d.FS)`. Until Q3, vault_fs.go temporarily retains its `os` import (grep-gated in Q3).
- DESIGN FLAG: `listJSONLIndexes` (query_chunks.go:138) signature flip is cross-cluster-atomic. Consumers: query.go:1295 (mine), amend.go:365, prune.go:115, show_chunk.go:72. The flip and all four call sites must land in ONE commit, and the three foreign call sites need a `d Deps` in scope — so Task Q2 must be sequenced AFTER the amend/prune/show-chunk cluster tasks convert their constructors to `newXxxDeps(d Deps)`. (Those clusters keep calling the old os-backed `listJSONLIndexes` until Q2 flips it — that ordering works; the reverse does not.)
- DESIGN FLAG: the `os.IsNotExist` → `errors.Is(err, fs.ErrNotExist)` change at query_chunks.go:141 is load-bearing, not cosmetic: `os.IsNotExist` does NOT unwrap `%w`-wrapped errors, and EdgeFS implementations wrap. The cmd/engram `osFS` impl must wrap with `%w` (contract: `errors.Is(err, fs.ErrNotExist)` survives the adapter); Q1/Q2 add tests proving the internal side unwraps.
- DESIGN FLAG: `newQueryDeps` needs the LogWarning hook. Current `logWarningToStderrf` (learn.go:332) writes to `os.Stderr` — learn cluster's file. This plan consumes `logWarningTo(w io.Writer) func(format string, args ...any)`; exact definition included in Q2 step 3 with instructions to add it to deps.go ONLY if the learn cluster has not already landed it (coordinate — one definition, two consumers).
- DESIGN FLAG: preconditions from other clusters: (a) foundation task must have landed `internal/cli/deps.go` (`Deps`, `EdgeFS` — neither exists yet, verified by grep); (b) targets-cluster task must have landed `Targets(d Deps)` with `d` threaded into `ingestQueryTargets` (targets.go:144) — all five constructor renames edit call sites there; (c) `executeForTest` (targets_test.go:434-448) migration is the targets cluster's; `TestTargets_CountEmptyVault` (count_test.go:562) rides on it unchanged.
- DESIGN FLAG: `ExportNewOsChunkQueryDeps` is called from ingest_integration_test.go:100,204 (ingest cluster's test file) — Q2 updates those two lines in the same commit as the shim rename.
- DESIGN FLAG: query_nominations.go has ZERO os/time references (the grep hit at line 136 is the word "time" in a comment). Nothing to migrate in that file.

**Ingest-family:**

- DESIGN FLAG: the manifest lock is not a bare flock. `osManifestLock` (internal/cli/cli.go:227) does `MkdirAll(chunksDir, 0o700)` BEFORE flocking `chunksDir/.manifest.lock` — a shipped regression fix (prune on a fresh dir errored without it). `FileLocker.Lock(path)` alone cannot express this. The MkdirAll must stay in internal composition (`manifestLockFrom` below, via `deps.FS.MkdirAll`). "Lock acquisition stays at Run* entry points via injected FileLocker" holds only through this composition, never by handing `FileLocker.Lock` directly to Run*.
- DESIGN FLAG: `FileLocker.Lock` returns `(func() error, error)` but `IngestDeps.Lock`/`PruneDeps.Lock` and every Run* call site use `func()` releases. The composition adapts by discarding the unlock error (`func() { _ = unlock() }`) — identical to today's `flockPath` release, which already swallows unlock/close errors.
- DESIGN FLAG: hidden I/O not named in the central task list — `walkSourcesExcluding` (ingest.go:654) calls `filepath.WalkDir` directly (real disk walk) with an `os.DirEntry` callback. Migrated below via `deps.FS.WalkDir` + `fs.DirEntry` (`sweepListerFrom`).
- DESIGN FLAG: `defaultSessionDir` (ingest.go:247-258) reads BOTH `ENGRAM_TRANSCRIPT_DIR` (os.Getenv) and `os.UserHomeDir` — the migration needs `deps.Getenv` AND `deps.UserHomeDir`, not Getenv alone.
- DESIGN FLAG (cross-cluster, coordination required):
  1. `listJSONLIndexes` (query_chunks.go:138, `os.ReadDir`) is shared by prune + query-chunks + show-chunk + query + amend. My plan adds the pure `jsonlIndexListerFrom(readDir)` helper in prune.go and migrates only prune's use; the query cluster must migrate its four call sites to the same helper and delete the os-based `listJSONLIndexes` last. The cmd `osFS.ReadDir` MUST wrap errors with `%w` so `errors.Is(err, fs.ErrNotExist)` survives (cold-start = empty index, not error).
  2. `flockPath` (cli.go:169, syscall) is also used by `osLearnFS.Lock` — my tasks delete `osManifestLock` (its consumers are exactly ingest+prune) but must NOT delete `flockPath`; the learn cluster owns that removal.
  3. `osFileReader` (cli.go:27) has exactly one production consumer: ingest.go:488. Task I1 deletes it + its adapter tests (adapters_test.go:14-39) + `ExportNewOsFileReader`. If the core cluster also drafted this deletion, dedupe.
  4. The ADR-0013 concurrency regression test (ingest_test.go:319) today injects the REAL flock via `cli.ExportFlockPath`. After migration it uses a test-local syscall flock (test files are exempt from enforcement) — this preserves the Run*-holds-lock-across-read-modify-write proof, but production `flockLocker` mutual-exclusion coverage moves to cmd/engram; confirm the adapters cluster's plan includes a flockLocker exclusivity + fresh-dir integration test, else production-lock coverage regresses.
  5. `realFS.write` test helper (ingest_test.go:899) uses `cli.ExportAtomicWriteFile`; Task I1 switches it to a test-local atomic write so the writesafe cluster can delete `atomicWriteFile` without breaking ingest tests.
  6. Both tasks assume the foundation task has landed: `internal/cli/deps.go` (`Deps`, `EdgeFS`, `FileLocker`) and `deps Deps` threaded into `ingestQueryTargets`. Verify `internal/cli/deps.go` exists before starting; the two targets.go line edits below name that dependency explicitly.
  7. cli_test test-harness naming: Task I1 creates `ingest_family_deps_test.go` holding `osTestFS`/`testFlocker`/`testDeps` (package cli_test is one namespace across files) — if another cluster drafts an equivalent harness, consolidate into one file.

**Maintenance-family:**

- DESIGN FLAG: `atomicWriteFile` has callers OUTSIDE this cluster: internal/cli/learn.go:371 (LearnDeps.WriteNote), internal/cli/cli.go:144 (osLearnFS.WriteSidecar), internal/cli/embed.go:164 (osEmbedFS.Write), internal/cli/qa.go:283 (QA deps). Deleting internal/cli/writesafe.go must be gated until those clusters migrate — split into Task M4 with an explicit grep gate.
- DESIGN FLAG: internal/cli/ingest_test.go:899 (`realFS.write`, part of the ADR-0013-adjacent concurrent-manifest regression infra) calls `cli.ExportAtomicWriteFile`. It needs REAL temp+rename semantics (torn-read protection under the race). Task M2 provides a test-only os-backed EdgeFS (`ExportNewTestOsDeps`) whose `WriteFileAtomic` carries the real dance; M4 repoints ingest_test.go:899 at it.
- DESIGN FLAG: shared-helper collision risk. My constructors need four composition helpers other clusters also need: a `vaultgraph.VaultFS` adapter over EdgeFS (replaces osVaultFS), a `.luhmann.lock` adapter over FileLocker (replaces osLearnFS.Lock), a stderr warn-logger (replaces learn.go's `logWarningToStderrf`), and an injected `.jsonl` lister (replaces query_chunks.go's os-backed `listJSONLIndexes`, which reads via `os.ReadDir` at query_chunks.go:139). I draft them once in internal/cli/deps_compose.go; orchestrator must dedupe against the learn/query/embed cluster drafts and have those clusters consume these helpers.
- DESIGN FLAG: EdgeFS error contract — `edgeVaultFS.ListMD` and `jsonlIndexesLister` rely on `errors.Is(err, fs.ErrNotExist)` for the missing-dir→empty contract (current code uses `os.IsNotExist` on the raw error). The cmd osFS `ReadDir`/`Stat` implementations MUST wrap with `%w` (never `%v`) so sentinel matching survives. Ditto the test EdgeFS.
- DESIGN FLAG: targets.go call sites (lines 108, 113, 173, 278, 282, 286, 290) wire my constructors but the surrounding `Targets(deps Deps)` threading is the wiring cluster's charge. M3 lists the exact call-expression diffs; the wiring cluster owns adding the `deps Deps` parameter to `amendResituateTargets`/`ingestQueryTargets`/`vocabTargets`.
- DESIGN FLAG: sequencing — cmd/engram/os_fs.go's `osFS` type has no production caller until the cmd wiring task lands `Deps{FS: osFS{}}` in main.go. If M1 merges before wiring, `targ check-full`'s unused-symbol lint may flag `osFS`. Order M1 after (or in the same merge window as) the cmd-wiring task's os_fs.go creation; M1 below is written create-or-append.
- DESIGN FLAG: cross-cluster test-file touches — learn_test.go:132 uses `ExportNewOsAmendDeps` (my constructor, learn cluster's file); os_adapters_test.go:150 tests `logWarningToStderrf` (learn cluster should delete it when adopting `warnLoggerTo`); targets_test.go:413 covers resituate wiring through `Targets()` (wiring cluster updates). One-line diffs for the first are included in M3.
- DESIGN FLAG: vault_init.go and vocab.go are ALREADY pure (verified: vault_init.go imports fmt/io\/fs/path\/filepath only; vocab.go imports slices/strings/yaml/embed only; no os, no time.Now). No migration needed — M3 carries verify-only steps.

**Embed-family (numbered — Task T14/T15 text cites these as "DESIGN FLAG n"):**

1. **`embed.Backend` does not exist today.** The existing exported interface in `internal/embed/embedder.go:54` is `Embedder` (`Embed(ctx, text) ([]float32, error)`, `ModelID() string`, `Dims() int`). Per the spec's own escape hatch, the Deps field must be `Embed embed.Embedder`. This plan *additionally exports* a new `embed.Backend` (rename of today's unexported `hugotBackend`, `internal/embed/hugot.go:227`) as the constructor seam the cmd hugot adapter implements. Both names end up existing with distinct roles: `Deps.Embed` is the constructed `Embedder`; `Backend` is what cmd implements to build it.
2. **`sharedEmbedder` is a cross-cluster singleton.** `internal/cli/embed.go:110` constructs it; it is consumed by 7 files OUTSIDE this cluster (`qa.go:275`, `query_chunks.go:193`, `vocab_commands.go:1228`, `ingest.go:513`, `learn.go:360`, `resituate.go:176`, `amend.go:358`). Deleting it breaks them; leaving it breaks purity (its construction reads `os.UserHomeDir`/`os.Getenv` and, post-migration, needs the hugot backend that no longer exists in internal). Resolution: a race-safe transitional **bridge** (atomic pointer + forwarder value, full code in Task A) wired from `Targets(deps)`; each command cluster later replaces `Embedder: sharedEmbedder` with `d.Embed`; the enforcement sweep deletes the bridge.
3. **`newOsEmbedDeps()` has two external consumers**: `targets.go:226,230` (embed group closures) and `query.go:1288` (`newOsQueryDeps`). Task B swaps the targets.go call sites and makes a minimal `newOsQueryDeps(d Deps)` signature change + its call site `targets.go:155`. This overlaps the query cluster's territory — coordinate; the query task's full `newQueryDeps(d)` rewrite subsumes this edit if it lands first (then Task B skips step 6).
4. **Dead production machinery found**: `unpackModelToTemp`, `tempFS`, `productionTempFS` (`internal/embed/hugot.go:314-452`) are referenced by NO production code — `extractToCache` replaced them. Plan **deletes** them plus `unpack_test.go`/`tempfs_test.go` instead of relocating (git is the fallback). Note: `nonEmptyTestFS` (go:embed) is declared in `unpack_test.go:65-66` but used by `cache_test.go` — its declaration moves into `cache_test.go`.
5. **`parity_test.go`** (build tag `parity`) imports hugot directly and reads `assets/model` from disk. It is a test file and is excluded from normal builds; the enforcement task must exempt `_test.go` files (or this tagged file) from depguard, otherwise relocate it to cmd/engram in that task — not handled here.
6. **Behavioral contract shift on `CacheFS.Rename`** (verified, load-bearing): today `commitCache` sniffs `*os.LinkError`/ENOTEMPTY strings via `isExistErr` (`cache.go:196`). os classification must live beside `os.Rename`, so the sniffing moves into the cmd adapter and the internal contract becomes `errors.Is(err, fs.ErrExist)`. The fakes at `cache_test.go:54,107` currently return raw `&os.LinkError{...directory not empty...}` — they MUST be updated to the new contract or the race tests go red for the wrong reason (Task A step 5).
7. **`CacheDirFromHome` stays in `internal/cli/targets.go:56`** (pure, exported); cmd calls it with `os.UserHomeDir`/`os.Getenv` at the edge. If the wiring cluster relocates it, adjust the import in `cmd/engram/hugot.go`.
8. **Foundation dependency**: both tasks assume the foundation task has landed — `internal/cli/deps.go` (Deps + EdgeFS), `Targets(deps Deps) []any`, and cmd/engram building the Deps literal. Task A adds one field-init line (`Embed: newProductionEmbedder()`) to that literal and one statement to `Targets` — coordinate the merge. EdgeFS contract notes for foundation: `ReadFile`/`ReadDir` errors must satisfy `errors.Is(err, fs.ErrNotExist)` for missing paths (state classification and ListMD-missing-dir semantics depend on it).
9. **`internal/embed/embedder.go` and `internal/embed/state.go` are already pure** (imports: `context/errors/fmt` and `errors/io/fs`) — zero changes needed in this cluster's charge for them.
10. **Coverage config**: cmd/engram may be excluded from coverage as an entry point (user global rules). The relocated adapter tests still run under `targ test`; if `targ check-full` flags cmd coverage, that's a dev-tooling config question for the enforcement task, not a reason to keep adapters in internal.

**Update-family:**

- DESIGN FLAG: osUpdateFS/osUpdateEnv cannot literally "move to cmd/engram/os_update.go" and stay wired: the fixed cli.Deps carries no `update.Filesystem`/`update.Env` field (only `FS EdgeFS` + env func fields), and `cli.Targets(deps Deps)` is the sole channel into internal/cli. Moving them verbatim would strand them as production-dead code in cmd (hoarding). This draft absorbs them instead: only osCommander physically moves to cmd/engram/os_update.go (wired as `Deps.Commander`); osUpdateFS becomes a pure EdgeFS→update.Filesystem bridge in internal/cli (`updateFSFromEdge`, zero I/O — `fs.DirEntry`/`fs.FileInfo` structurally satisfy `update.DirEntry`/`update.FileInfo`, so `osDirEntry`/`osFileInfo` wrappers die too); osUpdateEnv becomes a pure Deps-func bridge (`updateEnvFromDeps`).
- DESIGN FLAG: spec cites exec.ErrNotFound at update.go:437/:545; actual worktree lines are 436 and 544, plus the doc comment at 541–542 names exec.ErrNotFound (updated in Task UF-1).
- DESIGN FLAG: two test files inject exec.ErrNotFound to simulate the commander — internal/update/runner_test.go:556 and internal/cli/invariants_u1_test.go:36. Both must switch to `update.ErrCommandNotFound` in the same commit as the sentinel cutover or the suite goes red (ErrGitNotFound/ErrGoNotFound classification tests).
- DESIGN FLAG (coordination with os_fs cluster): update's `isNotExist` and its planners tolerate missing dirs via `errors.Is(err, fs.ErrNotExist)`. The production EdgeFS impl (cmd/engram/os_fs.go) MUST preserve that chain on ReadFile/ReadDir/Stat (no chain-breaking wraps), and RemoveAll must keep os.RemoveAll's nil-on-absent semantics. The deleted TestOsUpdateFS_* round-trips (7 tests in internal/cli/update_test.go:275-441) hand their real-FS coverage to that cluster's cmd/engram/os_fs_test.go.
- DESIGN FLAG (coordination with wiring cluster): Task UF-2's targets.go call site assumes `learnUpdateTargets` receives `deps Deps` after the Targets(deps) migration; the `Commander: &osCommander{},` field lands in cmd/engram's Deps literal built by that cluster. Sequencing: UF-1 is independent and can land first; UF-2 requires deps.go (Deps+EdgeFS), Targets(deps) threading, and cmd/engram/os_fs.go to exist.
- DESIGN FLAG: export_test.go (package cli) does not currently import internal/update; UF-2's new export helper adds that import.
- DESIGN FLAG: this worktree's internal/update/update.go is clean; the uncommitted Pi-harness edit exists only in the main tree. The internal/update diff below is 4 hunks (sentinel var, two errors.Is swaps, one comment) + import line — minimal for concurrent Pi worktrees.

*Deduplicated: the learn/query/ingest/maintenance drafts each independently re-confirmed the `embed.Embedder`-not-`embed.Backend` finding (collapsed into the first wiring-core bullet and embed-family flag 1), and the maintenance draft restated the FileLocker unlock-error-discard decision (collapsed into the ingest-family FileLocker bullet).*

## Cross-task resolutions (BINDING — this table wins over any conflicting task-body text)

Parallel drafting produced overlapping symbol names and one ordering contradiction. These
resolutions are authoritative: where a task body's symbol name or file-mode (Create vs Modify)
conflicts with this section, THIS SECTION WINS. Every task dispatch must include this section
verbatim alongside the task text.

**R1 — `internal/cli/deps_compose.go` has ONE creator.** T3 (L1) creates it. T11 (M2) MODIFIES it
(its "Create" is downgraded): T11 consumes T3's helpers where they overlap and adds only the
genuinely new ones. Canonical helper names (losers in parentheses are NOT declared anywhere):
`vaultLockFromLocker` (not `vaultLuhmannLock`), `logWarningTo` (not `warnLoggerTo`). T11 still
adds `ExportNewTestOsDeps() Deps` (test-only) and any helper with no T3 equivalent.

**R2 — ONE `vaultgraph.VaultFS`-over-EdgeFS adapter.** T5's `vaultFS` via
`newVaultFS(fsys EdgeFS) *vaultFS` is canonical. T11 does NOT declare `edgeVaultFS`; T15 does NOT
declare `depsVaultFS`; both consume `newVaultFS(d.FS)`.

**R3 — ONE `.jsonl` index lister.** T6's `listJSONLIndexes(fsys EdgeFS) func(dir string) ([]string, error)`
is canonical. T9 consumes `listJSONLIndexes(d.FS)` instead of declaring `jsonlIndexListerFrom`;
T11 does NOT declare `jsonlIndexesLister`.

**R4 — EXECUTION ORDER (binding; document order is NOT execution order):**
T1 → T2 → T3 → T5 → T6 → T8 → T9 → T10 → T11 → T12 → T4 → T13 → T14 → T15 → T7 → T16 → T17 → T-final-1 → T-final-2.
Rationale: T4 (purge cli.go adapters) requires T8/T9/T12 done (its own heading says so — it sits
mid-document only because drafts were assembled by family); T7 (purge osVaultFS) runs after T15
(embed.go is osVaultFS's last consumer); deletions are grep-gated so a premature run fails loud.

**R5 — `osFileReader` is deleted ONCE, by T8.** T4's corresponding step becomes a grep
verification (`rg -n "osFileReader" internal/` → zero hits), not a second deletion.

**R6 — `embed.NewLazyEmbedder` arity handoff.** T2 wires `Embed: embed.NewLazyEmbedder(cacheDir)`
(today's 1-arg form). T14 changes the constructor to its 3-arg form AND updates that one line in
`cmd/engram/main.go` in the same commit — the executor of T14 must not skip the cmd-side line
(it is in T14's file list).

**R7 — `ExportFlockPath`.** T8 deletes it from `export_test.go`; T4 (running later per R4)
re-implements the flock probe test-locally exactly as its body specifies. No other task touches it.

## Tasks

### Task T1: Pure signal seam, cli.Deps carrier, cmd/engram OS adapters

**Files:**
- Create: `internal/cli/deps.go`
- Create: `cmd/engram/os_fs.go`, `cmd/engram/os_fs_test.go`
- Create: `cmd/engram/os_signal.go`, `cmd/engram/os_signal_test.go`
- Create: `cmd/engram/debuglog_sink.go`, `cmd/engram/debuglog_sink_test.go`
- Modify: `internal/cli/signal.go` (pure `ForceExitOnRepeatedSignal`; interim adapter in `SetupSignalHandling`)
- Modify: `internal/cli/signal_test.go` (struct{} pulses)

**Interfaces:**
- Consumes: `update.Commander` — `Run(ctx context.Context, dir, name string, args ...string) (stdout, stderr []byte, err error)` (internal/update/update.go:48); `embed.Embedder` — `Embed(ctx context.Context, text string) ([]float32, error)`, `ModelID() string`, `Dims() int` (internal/embed/embedder.go:54); `cli.ForceExitOnRepeatedSignal`, `cli.ExitCodeSigInt` (internal/cli/signal.go).
- Produces: `cli.Deps`, `cli.EdgeFS`, `cli.FileLocker` (new, exact code below); `func ForceExitOnRepeatedSignal(signals <-chan struct{}, exitFn func(int))` (changed signature); package-main `osFS`, `flockLocker`, `registerForceExit(exitFn func(int), sigs ...os.Signal)`, `forwardAsPulses`, `openDebugSink(path string) io.Writer`, `syncWriter`.

**Steps:**

1. [ ] RED — rewrite `internal/cli/signal_test.go` for the pure signature. Current file (lines 1–75) uses `chan os.Signal` + `syscall.SIGINT` and tests `SetupSignalHandling`. Replace the whole file with:

```go
package cli_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestForceExitOnRepeatedSignal(t *testing.T) {
	t.Parallel()

	t.Run("calls exit on second signal", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		const pulseBuffer = 2

		pulses := make(chan struct{}, pulseBuffer)
		exitCalled := make(chan int, 1)

		cli.ForceExitOnRepeatedSignal(pulses, func(code int) {
			exitCalled <- code
		})

		pulses <- struct{}{}

		pulses <- struct{}{}

		select {
		case code := <-exitCalled:
			g.Expect(code).To(Equal(cli.ExitCodeSigInt))
		case <-time.After(time.Second):
			t.Fatal("exit not called within 1s of second signal")
		}
	})

	t.Run("does not exit on first signal alone", func(t *testing.T) {
		t.Parallel()

		pulses := make(chan struct{}, 1)
		exitCalled := make(chan int, 1)

		cli.ForceExitOnRepeatedSignal(pulses, func(code int) {
			exitCalled <- code
		})

		pulses <- struct{}{} // first only

		const shortWait = 100 * time.Millisecond

		select {
		case <-exitCalled:
			t.Fatal("exit called after only one signal")
		case <-time.After(shortWait):
			// good — no exit after one signal
		}
	})
}
```

   Note `TestSetupSignalHandling_ReturnsTargets` (old lines 64–75) is deleted here — its target-count assertion already exists as targets_test.go:146 `TestTargets/"returns expected target count"`. Run `targ test` — expect a compile failure in internal/cli tests (`cannot use pulses (variable of type chan struct{}) as chan os.Signal`). That is the RED.

2. [ ] GREEN — convert `internal/cli/signal.go`. Current file imports `io`, `os`, `os/signal`, `sync/atomic`, `syscall`, debuglog, and `ForceExitOnRepeatedSignal` takes `<-chan os.Signal` (line 21). Replace the whole file with:

```go
package cli

import (
	"io"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/toejough/engram/internal/debuglog"
)

// Exported constants.
const (
	ExitCodeSigInt = 130
)

// ForceExitOnRepeatedSignal starts a goroutine that waits for two pulses
// on the given channel, then calls exitFn. The first pulse allows graceful
// shutdown; the second forces immediate exit. Pulses are pure struct{}
// units — cmd/engram adapts real os.Signal deliveries into them, so no
// os.Signal type crosses into internal/ (#700).
func ForceExitOnRepeatedSignal(signals <-chan struct{}, exitFn func(int)) {
	var signalCount atomic.Int32

	go func() {
		for range signals {
			count := signalCount.Add(1)
			if count >= secondSignal {
				// Second signal or later: force exit immediately
				exitFn(ExitCodeSigInt)

				return
			}
			// First signal: will be handled by targ's context cancellation
		}
	}()
}

// SetupSignalHandling registers signal handlers and starts the force-exit goroutine.
// Returns the configured targets for targ.Main.
//
// Deprecated: interim shim only — deleted by the #700 wiring task; cmd/engram
// registers signals itself and calls Targets directly.
func SetupSignalHandling(
	stdout, stderr io.Writer,
	exitFn func(int),
	logger *debuglog.Logger,
) []any {
	sigCh := make(chan os.Signal, signalChannelBuffer)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	pulses := make(chan struct{}, signalChannelBuffer)

	go func() {
		for range sigCh {
			pulses <- struct{}{}
		}
	}()

	ForceExitOnRepeatedSignal(pulses, exitFn)

	return Targets(stdout, stderr, exitFn, logger)
}

// unexported constants.
const (
	secondSignal        = 2  // Force exit on second signal
	signalChannelBuffer = 10 // Buffer size for signal channel
)
```

   Run `targ test` — expect all green (existing suite unchanged elsewhere; signal behavior identical through the interim adapter).

3. [ ] Create `internal/cli/deps.go` — the capability carrier, verbatim per the central spec (only `Embed`'s type adjusted to the real interface name):

```go
package cli

import (
	"io"
	"io/fs"
	"time"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/update"
)

// Deps carries every impure capability the CLI needs, wired by cmd/engram.
// internal/ code never calls os.*, exec, syscall, or time.Now directly —
// production I/O enters exclusively through this struct (#700, ADR-0001).
type Deps struct {
	// Stdout receives command output (production: os.Stdout).
	Stdout io.Writer
	// Stderr receives error output (production: os.Stderr).
	Stderr io.Writer
	// Exit terminates the process with a status code (production: os.Exit).
	Exit func(int)
	// Getenv reads an environment variable (production: os.Getenv).
	Getenv func(string) string
	// Now returns the current wall-clock time (production: time.Now).
	Now func() time.Time
	// Getwd returns the process working directory (production: os.Getwd).
	Getwd func() (string, error)
	// UserHomeDir returns the user's home directory (production: os.UserHomeDir).
	UserHomeDir func() (string, error)
	// FS is the filesystem edge (production: cmd/engram's osFS).
	FS EdgeFS
	// Lock acquires exclusive cross-process file locks (production: flockLocker).
	Lock FileLocker
	// Commander runs external commands for `engram update` (production: osCommander).
	Commander update.Commander
	// Embed is the production embedder backend (hugot-backed lazy embedder).
	Embed embed.Embedder
	// DebugLog is the debug-log sink; nil disables debug logging (no-op logger).
	DebugLog io.Writer
}

// EdgeFS is the filesystem capability surface for production wiring. All
// mode/info/entry types come from io/fs — never os — so internal/ stays
// free of I/O-capable imports (#700).
type EdgeFS interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	// WriteFileAtomic writes via temp file + same-directory rename so a
	// concurrent reader sees the old or the new content, never a torn
	// write (ADR-0013).
	WriteFileAtomic(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	MkdirTemp(dir, pattern string) (string, error)
	Stat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Remove(path string) error
	RemoveAll(path string) error
	Rename(oldPath, newPath string) error
	WalkDir(root string, fn fs.WalkDirFunc) error
}

// FileLocker acquires an exclusive advisory lock on the file at path,
// creating it if absent. unlock releases the lock and closes the handle.
// Production: flock(2) via cmd/engram's flockLocker (ADR-0013).
type FileLocker interface {
	Lock(path string) (unlock func() error, err error)
}
```

   Run `targ check-full` — expect clean (exported, not-yet-consumed types are legal and unflagged).

4. [ ] RED — create `cmd/engram/os_fs_test.go` (package main, real FS in `t.TempDir()`; fails to compile until step 5 — `undefined: osFS`):

```go
package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/gomega"
)

// Shared test constants for the package-main adapter tests.
const (
	testFilePerm fs.FileMode = 0o600
	testDirPerm  fs.FileMode = 0o750
)

func TestOsFS_ReadWriteRoundTrip(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	fsys := osFS{}

	g.Expect(fsys.WriteFile(path, []byte("hello"), testFilePerm)).To(gomega.Succeed())

	data, err := fsys.ReadFile(path)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(gomega.Equal("hello"))
}

func TestOsFS_ReadFileMissingSatisfiesErrNotExist(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fsys := osFS{}

	_, err := fsys.ReadFile(filepath.Join(t.TempDir(), "missing.txt"))
	g.Expect(err).To(gomega.MatchError(fs.ErrNotExist))
}

func TestOsFS_WriteFileAtomicReplacesContentAndCleansTemp(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	fsys := osFS{}

	g.Expect(fsys.WriteFile(path, []byte("v1"), testFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.WriteFileAtomic(path, []byte("v2"), testFilePerm)).To(gomega.Succeed())

	data, err := fsys.ReadFile(path)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(gomega.Equal("v2"))

	entries, readErr := fsys.ReadDir(dir)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(1), "temp files must be renamed or removed")
}

func TestOsFS_MkdirAllStatReadDir(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b")
	fsys := osFS{}

	g.Expect(fsys.MkdirAll(nested, testDirPerm)).To(gomega.Succeed())

	info, err := fsys.Stat(nested)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(info.IsDir()).To(gomega.BeTrue())

	entries, readErr := fsys.ReadDir(filepath.Join(dir, "a"))
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(1))
	g.Expect(entries[0].Name()).To(gomega.Equal("b"))
}

func TestOsFS_RenameRemoveRemoveAll(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	fsys := osFS{}

	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")
	g.Expect(fsys.WriteFile(oldPath, []byte("x"), testFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.Rename(oldPath, newPath)).To(gomega.Succeed())
	g.Expect(newPath).To(gomega.BeAnExistingFile())
	g.Expect(oldPath).NotTo(gomega.BeAnExistingFile())

	g.Expect(fsys.Remove(newPath)).To(gomega.Succeed())
	g.Expect(newPath).NotTo(gomega.BeAnExistingFile())

	sub := filepath.Join(dir, "sub")
	g.Expect(fsys.MkdirAll(sub, testDirPerm)).To(gomega.Succeed())
	g.Expect(fsys.WriteFile(filepath.Join(sub, "f"), []byte("x"), testFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.RemoveAll(sub)).To(gomega.Succeed())
	g.Expect(sub).NotTo(gomega.BeADirectory())
}

func TestOsFS_MkdirTempAndWalkDir(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	fsys := osFS{}

	tmpDir, err := fsys.MkdirTemp(dir, "pat-*")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(filepath.Base(tmpDir)).To(gomega.HavePrefix("pat-"))
	g.Expect(fsys.WriteFile(filepath.Join(tmpDir, "leaf.txt"), []byte("x"), testFilePerm)).To(gomega.Succeed())

	visited := make([]string, 0, 3)
	walkErr := fsys.WalkDir(dir, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		visited = append(visited, path)

		return nil
	})
	g.Expect(walkErr).NotTo(gomega.HaveOccurred())
	g.Expect(visited).To(gomega.ContainElement(filepath.Join(tmpDir, "leaf.txt")))
}

// TestFlockLocker_SecondLockWaitsForUnlock is the cmd-side ADR-0013 lock
// regression guard: a second locker on the same path must block until the
// first unlocks — never proceed concurrently, never fail.
func TestFlockLocker_SecondLockWaitsForUnlock(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	lockPath := filepath.Join(t.TempDir(), "test.lock")
	locker := flockLocker{}

	unlock, err := locker.Lock(lockPath)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	acquired := make(chan struct{})

	go func() {
		secondUnlock, secondErr := locker.Lock(lockPath)
		if secondErr == nil {
			_ = secondUnlock()
		}

		close(acquired)
	}()

	const holdWindow = 100 * time.Millisecond

	select {
	case <-acquired:
		t.Fatal("second locker acquired while first still held the lock")
	case <-time.After(holdWindow):
		// good — second locker is blocked while the lock is held
	}

	g.Expect(unlock()).To(gomega.Succeed())

	const releaseTimeout = 2 * time.Second

	select {
	case <-acquired:
		// good — released lock was acquired
	case <-time.After(releaseTimeout):
		t.Fatal("second locker did not acquire after unlock")
	}
}

// Silence the unused warning for os in files where only some tests need it.
var _ = os.Getpid
```

   (Drop the final `var _` line if `os` ends up referenced; as written `os` is unreferenced in this file — remove the `"os"` import and the `var _` line instead. Final import list: `io/fs`, `path/filepath`, `testing`, `time`, gomega.) Run `targ test` — expect compile failure `undefined: osFS` (RED).

5. [ ] GREEN — create `cmd/engram/os_fs.go`. `WriteFileAtomic` is a faithful port of internal/cli/writesafe.go:21–73 `doAtomicWrite` (CreateTemp in target dir, chmod, write, close, rename; temp removed on failure — ADR-0013 semantics preserved exactly). `flockLocker` is a port of internal/cli/cli.go:169–192 `flockPath` with the spec's `unlock func() error` shape:

```go
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"github.com/toejough/engram/internal/cli"
)

// Compile-time interface conformance for the production adapters.
var (
	_ cli.EdgeFS     = osFS{}
	_ cli.FileLocker = flockLocker{}
)

// osFS is the production cli.EdgeFS: thin wrappers over os.* with
// contextual error wrapping (%w preserves errors.Is chains such as
// fs.ErrNotExist). All production filesystem I/O lives here, not in
// internal/ (#700).
type osFS struct{}

// MkdirAll creates path with any missing parents; no-op when path exists.
func (osFS) MkdirAll(path string, perm fs.FileMode) error {
	err := os.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}

	return nil
}

// MkdirTemp creates a fresh unique directory in dir matching pattern.
func (osFS) MkdirTemp(dir, pattern string) (string, error) {
	made, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("mkdir temp in %s: %w", dir, err)
	}

	return made, nil
}

// ReadDir returns the directory entries of path.
func (osFS) ReadDir(path string) ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", path, err)
	}

	return entries, nil
}

// ReadFile reads the file at path.
func (osFS) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path from caller
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return data, nil
}

// Remove deletes the file or empty directory at path.
func (osFS) Remove(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}

	return nil
}

// RemoveAll deletes path and any children; no-op when path is absent.
func (osFS) RemoveAll(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("remove all %s: %w", path, err)
	}

	return nil
}

// Rename atomically renames oldPath to newPath (same-directory renames are
// atomic on POSIX — the ADR-0013 primitive).
func (osFS) Rename(oldPath, newPath string) error {
	err := os.Rename(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("rename %s -> %s: %w", oldPath, newPath, err)
	}

	return nil
}

// Stat returns the fs.FileInfo for path.
func (osFS) Stat(path string) (fs.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	return info, nil
}

// WalkDir walks the file tree rooted at root, calling fn for each entry.
func (osFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	err := filepath.WalkDir(root, fn)
	if err != nil {
		return fmt.Errorf("walk %s: %w", root, err)
	}

	return nil
}

// WriteFile writes data to path with perm.
func (osFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	err := os.WriteFile(path, data, perm) //nolint:gosec // path from caller
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// WriteFileAtomic writes data to path atomically: it creates a unique temp
// file in filepath.Dir(path), sets perms, writes, closes, then renames into
// place. A same-directory rename is atomic on POSIX — a concurrent reader
// sees either the old or the new file, never a partial one. On any error the
// temp file is removed and the original (if any) is left untouched
// (ADR-0013; port of internal/cli's doAtomicWrite).
func (osFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	tmp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return fmt.Errorf("atomic write %s: create temp: %w", path, err)
	}

	tmpName := tmp.Name()

	// Best-effort cleanup on any error path.
	success := false

	defer func() {
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	chmodErr := os.Chmod(tmpName, perm)
	if chmodErr != nil {
		_ = tmp.Close()

		return fmt.Errorf("atomic write %s: chmod temp: %w", path, chmodErr)
	}

	_, writeErr := tmp.Write(data)
	if writeErr != nil {
		_ = tmp.Close()

		return fmt.Errorf("atomic write %s: write temp: %w", path, writeErr)
	}

	closeErr := tmp.Close()
	if closeErr != nil {
		return fmt.Errorf("atomic write %s: close temp: %w", path, closeErr)
	}

	renameErr := os.Rename(tmpName, path)
	if renameErr != nil {
		return fmt.Errorf("atomic write %s: rename: %w", path, renameErr)
	}

	success = true

	return nil
}

// flockLocker is the production cli.FileLocker: opens path (O_CREATE|O_RDWR)
// and acquires an exclusive flock(2). unlock releases the lock and closes the
// handle. ADR-0013: advisory flock + atomic rename are the two safety
// primitives for concurrent vault writers (port of internal/cli's flockPath).
type flockLocker struct{}

// Lock acquires an exclusive flock on path, creating the file if absent.
func (flockLocker) Lock(path string) (func() error, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, lockFilePerm) //nolint:gosec // path from caller
	if err != nil {
		return nil, fmt.Errorf("open lock %s: %w", path, err)
	}

	fileDescriptor := int(f.Fd())

	flockErr := syscall.Flock(fileDescriptor, syscall.LOCK_EX)
	if flockErr != nil {
		_ = f.Close()

		return nil, fmt.Errorf("flock %s: %w", path, flockErr)
	}

	unlock := func() error {
		unlockErr := syscall.Flock(fileDescriptor, syscall.LOCK_UN)
		closeErr := f.Close()

		if unlockErr != nil {
			return fmt.Errorf("funlock %s: %w", path, unlockErr)
		}

		if closeErr != nil {
			return fmt.Errorf("close lock %s: %w", path, closeErr)
		}

		return nil
	}

	return unlock, nil
}

// unexported constants.
const (
	lockFilePerm = 0o600
)
```

   Run `targ test` — expect green (os_fs tests pass).

6. [ ] RED — create `cmd/engram/os_signal_test.go` (fails to compile — `undefined: forwardAsPulses`, `undefined: registerForceExit`):

```go
package main

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestForwardAsPulses_ForwardsEachSignal(t *testing.T) {
	t.Parallel()

	const signalCount = 2

	sigCh := make(chan os.Signal, signalCount)
	pulses := make(chan struct{}, signalCount)

	go forwardAsPulses(sigCh, pulses)

	sigCh <- syscall.SIGUSR1

	sigCh <- syscall.SIGUSR1

	const pulseTimeout = time.Second

	for range signalCount {
		select {
		case <-pulses:
		case <-time.After(pulseTimeout):
			t.Fatal("pulse not forwarded within timeout")
		}
	}
}

// TestRegisterForceExit_SecondSignalForcesExit delivers real OS signals to
// this process. SIGUSR2 is registered only by this test, and Notify overrides
// the default (terminate) disposition, so the test run is safe. Pending
// same-signal coalescing can swallow a rapid second delivery, so the test
// keeps signalling (paced) until exit fires — over-delivery still means
// "force exit", so this can never false-pass.
func TestRegisterForceExit_SecondSignalForcesExit(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	exitCodes := make(chan int, 1)

	registerForceExit(func(code int) { exitCodes <- code }, syscall.SIGUSR2)

	proc, err := os.FindProcess(os.Getpid())
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	const (
		pace    = 20 * time.Millisecond
		maxWait = 5 * time.Second
	)

	deadline := time.After(maxWait)

	for {
		g.Expect(proc.Signal(syscall.SIGUSR2)).To(gomega.Succeed())

		select {
		case code := <-exitCodes:
			g.Expect(code).To(gomega.Equal(cli.ExitCodeSigInt))

			return
		case <-deadline:
			t.Fatal("force-exit not fired within timeout")
		case <-time.After(pace):
		}
	}
}
```

7. [ ] GREEN — create `cmd/engram/os_signal.go`:

```go
package main

import (
	"os"
	"os/signal"

	"github.com/toejough/engram/internal/cli"
)

// registerForceExit subscribes to sigs and starts the force-exit watcher:
// the first signal is left to targ's context cancellation for graceful
// shutdown; the second forces exitFn (cli.ForceExitOnRepeatedSignal). Must
// run BEFORE targ.Main so the handler covers the whole run. os.Signal never
// crosses into internal/ — deliveries are adapted to pure struct{} pulses
// here (#700).
func registerForceExit(exitFn func(int), sigs ...os.Signal) {
	sigCh := make(chan os.Signal, signalChannelBuffer)
	signal.Notify(sigCh, sigs...)

	pulses := make(chan struct{}, signalChannelBuffer)

	go forwardAsPulses(sigCh, pulses)

	cli.ForceExitOnRepeatedSignal(pulses, exitFn)
}

// forwardAsPulses converts OS signal deliveries into unit pulses.
func forwardAsPulses(sigCh <-chan os.Signal, pulses chan<- struct{}) {
	for range sigCh {
		pulses <- struct{}{}
	}
}

// unexported constants.
const (
	signalChannelBuffer = 10 // buffer so a burst of signals is not dropped
)
```

   Run `targ test` — expect green.

8. [ ] RED — create `cmd/engram/debuglog_sink_test.go` (fails to compile — `undefined: openDebugSink`):

```go
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
)

func TestOpenDebugSink_EmptyPathReturnsNil(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	g.Expect(openDebugSink("")).To(gomega.BeNil())
}

func TestOpenDebugSink_AppendsAcrossOpens(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	path := filepath.Join(t.TempDir(), "debug.log")

	sink := openDebugSink(path)
	g.Expect(sink).NotTo(gomega.BeNil())

	if sink == nil {
		return
	}

	_, err := sink.Write([]byte("line one\n"))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// Re-open the same path: append mode must preserve the first line —
	// the tail -F contract debuglog documents.
	second := openDebugSink(path)
	g.Expect(second).NotTo(gomega.BeNil())

	if second == nil {
		return
	}

	_, err = second.Write([]byte("line two\n"))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	contents, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(contents)).To(gomega.Equal("line one\nline two\n"))
}

func TestOpenDebugSink_UnopenablePathReturnsNil(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Parent is a regular file, so opening a child path fails -> nil sink
	// (the CLI must run without debug logging rather than fail).
	dir := t.TempDir()
	blocked := filepath.Join(dir, "isfile")
	g.Expect(os.WriteFile(blocked, []byte("x"), testFilePerm)).To(gomega.Succeed())

	g.Expect(openDebugSink(filepath.Join(blocked, "debug.log"))).To(gomega.BeNil())
}
```

   (`testFilePerm` is shared from os_fs_test.go — same package.)

9. [ ] GREEN — create `cmd/engram/debuglog_sink.go`:

```go
package main

import (
	"fmt"
	"io"
	"os"
)

// openDebugSink opens path in append mode as the debug-log sink. An empty
// path or an open failure yields nil — debuglog.New treats a nil writer as
// "logging disabled", so the CLI still runs (matches the pre-#700 behavior
// where a failed open fell back to a no-op logger).
func openDebugSink(path string) io.Writer {
	if path == "" {
		return nil
	}

	// Path comes from operator-set env var (ENGRAM_DEBUG_LOG), not user input.
	//nolint:gosec // operator-controlled path
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, debugLogPerm)
	if err != nil {
		return nil
	}

	return &syncWriter{file: f}
}

// syncWriter wraps an *os.File so every write is flushed to disk. debuglog
// is documented tail -F friendly; the Logger now sees only an io.Writer, so
// the per-line sync lives here at the edge.
type syncWriter struct {
	file *os.File
}

// Write appends p and syncs to disk.
func (w *syncWriter) Write(p []byte) (int, error) {
	n, err := w.file.Write(p)
	if err != nil {
		return n, fmt.Errorf("debug log write: %w", err)
	}

	_ = w.file.Sync()

	return n, nil
}

// unexported constants.
const (
	debugLogPerm = 0o644
)
```

   Run `targ test` — expect green.

10. [ ] Run `targ check-full` — expect clean. If it flags `registerForceExit`/`openDebugSink` as unused (not yet called from main.go), do NOT suppress: fold this commit together with T2's instead.

11. [ ] Commit:

```
refactor(cli): #700 add Deps carrier, pure signal seam, cmd/engram OS adapters

Introduces cli.Deps/EdgeFS/FileLocker (the single impure-capability
carrier wired by cmd/engram), converts ForceExitOnRepeatedSignal to a
pure struct{} pulse channel, and lands the cmd-side production adapters
(osFS + flockLocker, signal adapter, syncing debug-log sink) with
integration tests on the real filesystem. ADR-0013 flock + atomic-rename
semantics ported verbatim and regression-tested.

AI-Used: [claude]
```

---

### Task T2: Targets(deps), cmd-side wiring, delete SetupSignalHandling/Main, purify debuglog

**Files:**
- Modify: `internal/debuglog/debuglog.go` (pure New/Log/Timed), `internal/debuglog/debuglog_test.go`
- Modify: `internal/cli/targets.go` (Targets(deps Deps) + all helper funcs; 22 `os.Getenv` sites → `deps.Getenv`; `os` import dropped)
- Modify: `internal/cli/signal.go` (delete `SetupSignalHandling` + `signalChannelBuffer` + io/os/signal/syscall/debuglog imports)
- Modify: `internal/cli/targets_test.go`, `internal/cli/vocab_commands_test.go` (8 `cli.Targets` call sites → `newTestDeps` helper)
- Modify: `cmd/engram/main.go` (full rewrite: newOsDeps + registerForceExit + targ.Main)
- Delete: `internal/cli/main.go` (old `Main`; resolves the FIXME(#700) at main.go:19–22)

**Interfaces:**
- Consumes: `cli.Deps` (T1); `debuglog.WithLogger(ctx, *Logger) context.Context`; `targ.Main(...any)`; `embed.NewLazyEmbedder(cacheDir string) *LazyEmbedder` (internal/embed/hugot.go:149); `embed.BundledModelID = "minilm-l6-v2@384"` (hugot.go:18); `cli.CacheDirFromHome(home, modelID string, getenv func(string) string) string` (targets.go:56).
- Produces: `func Targets(deps Deps) []any` (replaces `Targets(stdout, stderr io.Writer, exit func(int), logger *debuglog.Logger) []any`); `func New(w io.Writer, prefix string, now func() time.Time) *Logger` (replaces `New(path, comp string) (*Logger, error)`); package-main `newOsDeps() cli.Deps`.

**Steps:**

1. [ ] RED — rewrite `internal/debuglog/debuglog_test.go` against the pure API (deterministic clock, no filesystem). Compile failure against the old `New(path, comp) (*Logger, error)` is the RED. Full replacement:

```go
package debuglog_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	g "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/debuglog"
)

func TestLog_NilReceiverIsSafe(t *testing.T) {
	t.Parallel()

	var nilLogger *debuglog.Logger

	// Nil-receiver methods must not panic.
	nilLogger.Log("stage", "msg=%s", "value")
	closer := nilLogger.Timed("stage", "arg=%s", "v")
	closer()
}

func TestLog_NoopWhenDisabled(t *testing.T) {
	t.Parallel()

	gomega := g.NewWithT(t)

	// A nil writer means "logging disabled": New returns a nil *Logger whose
	// methods are all no-ops.
	logger := debuglog.New(nil, "", fixedNow)
	gomega.Expect(logger).To(g.BeNil())

	// Must not panic on a disabled (no-op) logger.
	logger.Log("stage", "msg=%s", "value")

	// Must also no-op when nothing is in ctx.
	debuglog.Log(context.Background(), "stage", "msg=%s", "value")
}

func TestLog_WritesTimestampedLine(t *testing.T) {
	t.Parallel()

	gomega := g.NewWithT(t)

	var out bytes.Buffer

	logger := debuglog.New(&out, "test", fixedNow)

	ctx := debuglog.WithLogger(context.Background(), logger)
	debuglog.Log(ctx, "some.stage", "key=%s val=%d", "hello", 42)

	gomega.Expect(out.String()).To(g.Equal(
		"2026-07-19T12:00:00Z [test] some.stage: key=hello val=42\n"))
}

func TestTimed_LogsStartAndEndWithDuration(t *testing.T) {
	t.Parallel()

	gomega := g.NewWithT(t)

	var out bytes.Buffer

	logger := debuglog.New(&out, "timed", steppingNow())

	ctx := debuglog.WithLogger(context.Background(), logger)
	closer := debuglog.Timed(ctx, "MyStage", "arg=%s", "val")
	closer()

	text := out.String()
	gomega.Expect(text).To(g.ContainSubstring("[timed] MyStage.start: arg=val"))
	gomega.Expect(text).To(g.ContainSubstring("[timed] MyStage.end: took=1s"))

	lines := strings.Split(strings.TrimSpace(text), "\n")
	gomega.Expect(lines).To(g.HaveLen(2))
}

func TestTimed_NoLoggerInContext(t *testing.T) {
	t.Parallel()

	// Package-level Timed with no logger in ctx returns a no-op closer.
	closer := debuglog.Timed(context.Background(), "stage", "arg=%s", "v")
	closer()
}

// fixedNow returns a constant instant so timestamp output is exact.
func fixedNow() time.Time {
	return time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
}

// steppingNow returns a clock that advances one second per call, making
// Timed's took= duration deterministic. Call sequence inside Timed:
// start-line timestamp, start capture, took argument, end-line timestamp —
// so took = 1s exactly.
func steppingNow() func() time.Time {
	current := fixedNow()

	return func() time.Time {
		now := current
		current = current.Add(time.Second)

		return now
	}
}
```

   Run `targ test` — expect compile failure in internal/debuglog tests (RED).

2. [ ] GREEN — rewrite `internal/debuglog/debuglog.go`. Current impure pieces: `os` import (line 13), `file *os.File` field (23), `os.OpenFile` (37), `time.Now()` (62, 79), `time.Since` (82), `filePerm` const (100–103). Full replacement (context.go is untouched; package-level `Log`/`Timed` kept verbatim):

```go
// Package debuglog provides a tail-friendly debug logger for engram
// pipelines. New wraps an injected io.Writer sink and returns a *Logger.
// Log calls write one line at a time under a mutex; the production sink
// (cmd/engram's debug-log sink) syncs to disk after every write so
// `tail -F` shows progress live. The package itself performs no I/O and
// reads no clock — the sink and the now func are injected at the edge (#700).
//
// Loggers are threaded through context (see WithLogger / LoggerFromContext).
// The package-level Log and Timed helpers read the logger from ctx, so call
// sites stay short while production wiring stays explicit.
package debuglog

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Logger writes structured debug lines to an injected sink. Methods are
// safe for concurrent use within one process and safe to call on a nil
// receiver (no-op), which means tests can pass a nil logger without panics.
type Logger struct {
	component string
	out       io.Writer
	now       func() time.Time
	mu        sync.Mutex
}

// New returns a *Logger tagged with prefix that writes to w, stamping each
// line via now. A nil w returns a nil *Logger — every method is a
// nil-receiver-safe no-op, preserving the "unset ENGRAM_DEBUG_LOG disables
// logging" behavior. now must be non-nil when w is non-nil.
func New(w io.Writer, prefix string, now func() time.Time) *Logger {
	if w == nil {
		return nil
	}

	return &Logger{component: prefix, out: w, now: now}
}

// Log writes a single line: <timestamp> [<component>] <stage>: <message>.
// Safe on a nil receiver (no-op) and safe for concurrent use.
//
//nolint:goprintffuncname // "Log" reads more naturally than "Logf" at call sites
func (l *Logger) Log(stage, format string, args ...any) {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := l.now().UTC().Format(time.RFC3339Nano)
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s: %s\n", timestamp, l.component, stage, msg)

	_, _ = io.WriteString(l.out, line)
}

// Timed wraps a stage with .start and .end log entries plus duration.
// Returns a defer-friendly closer:
//
//	defer logger.Timed("Cycle.Run", "projectDir=%s", projectDir)()
//
// Safe on a nil receiver.
func (l *Logger) Timed(stage, format string, args ...any) func() {
	if l == nil {
		return func() {}
	}

	l.Log(stage+".start", format, args...)

	start := l.now()

	return func() {
		l.Log(stage+".end", "took=%s", l.now().Sub(start))
	}
}

// Log reads a *Logger from ctx and writes a line. No-op when ctx carries
// no logger.
//
//nolint:goprintffuncname // mirrors Logger.Log naming
func Log(ctx context.Context, stage, format string, args ...any) {
	LoggerFromContext(ctx).Log(stage, format, args...)
}

// Timed reads a *Logger from ctx and starts a timed entry. No-op closer
// when ctx carries no logger.
func Timed(ctx context.Context, stage, format string, args ...any) func() {
	return LoggerFromContext(ctx).Timed(stage, format, args...)
}
```

   (Add `"context"` back to the import block for the package-level helpers — final imports: `context`, `fmt`, `io`, `sync`, `time`.) `targ test` still RED overall: internal/cli/main.go:23 now fails to compile (`debuglog.New(os.Getenv(...), "engram")` — wrong arity). That is expected; proceed.

3. [ ] Rewrite `internal/cli/targets.go`. Lines 1–79 (imports through `ProjectSlugFromPath`) change only in the import block — drop `"io"`? No: `newErrHandler` keeps `io.Writer`; drop only `"os"`. New import block replaces lines 3–14:

```go
import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/debuglog"
)
```

   Lines 16–79 (`CommonLearnArgs`, `LearnFactArgs`, `LearnFeedbackArgs`, `CacheDirFromHome`, `DataDirFromHome`, `ProjectSlugFromPath`) are unchanged. Every function from line 81 down except `newErrHandler` (252–264, unchanged) is replaced:

```go
// Targets returns all targ targets for the engram CLI, wired from the
// single production capability carrier (#700). The debug logger is
// constructed from deps.DebugLog (nil sink → no-op logger) and attached to
// each handler's ctx so downstream code can call debuglog.Log without an
// explicit logger argument.
func Targets(deps Deps) []any {
	errHandler := newErrHandler(deps.Stderr, deps.Exit)
	logger := debuglog.New(deps.DebugLog, "engram", deps.Now)

	withLog := func(ctx context.Context) context.Context {
		return debuglog.WithLogger(ctx, logger)
	}

	return append(
		coreTargets(deps, withLog, errHandler),
		maintenanceTargets(deps, withLog, errHandler)...,
	)
}

// amendResituateTargets returns the amend and resituate subcommands. Split out
// of maintenanceTargets to stay within the per-function length budget.
func amendResituateTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
	home string,
) []any {
	return []any{
		targ.Targ(func(ctx context.Context, a ResituateArgs) {
			a.Vault = resolveVault(a.Vault, home, deps.Getenv)
			errHandler(RunResituate(withLog(ctx), a, newOsResituateDeps(), deps.Stdout))
		}).Name("resituate").Description("Rewrite a note's situation in sync (frontmatter + body + sidecar) (D4/INV-S2)"),
		targ.Targ(func(ctx context.Context, a AmendArgs) {
			a.Vault = resolveVault(a.Vault, home, deps.Getenv)
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunAmend(withLog(ctx), a, newOsAmendDeps(), deps.Stdout))
		}).Name("amend").Description("Amend a note in place: supersedes, provenance-merge, field-replacement, activate"),
	}
}

// coreTargets returns the primary subcommands (learn, update, embed, query,
// show, check). Split from Targets to stay within the per-function length
// budget; the wiring mirrors maintenanceTargets exactly.
func coreTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
) []any {
	return append(
		ingestQueryTargets(deps, withLog, errHandler),
		learnUpdateTargets(deps, withLog, errHandler)...,
	)
}

// homeOrEmpty returns the user's home directory via the injected capability,
// or "" when it cannot be resolved (or is unwired, as in minimal test Deps).
// resolveVault tolerates an empty home (it falls back to env / XDG), so the
// error is intentionally discarded.
func homeOrEmpty(deps Deps) string {
	if deps.UserHomeDir == nil {
		return ""
	}

	home, _ := deps.UserHomeDir()

	return home
}

// ingestQueryTargets returns the read/write-vault subcommands (query, ingest,
// query-chunks, activate, show, check). Split from coreTargets to stay within
// the per-function length budget.
func ingestQueryTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
) []any {
	home := homeOrEmpty(deps)

	return []any{
		targ.Targ(func(ctx context.Context, a QueryArgs) {
			a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunQuery(withLog(ctx), a, newOsQueryDeps(), deps.Stdout))
		}).Name("query").Description("Semantic search over vault + chunk index (YAML output)"),
		targ.Targ(func(ctx context.Context, a IngestArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunIngest(withLog(ctx), a, newOsIngestDeps(), deps.Stdout))
		}).Name("ingest").Description("Chunk+embed transcripts/markdown into a chunk index (zero-LLM)"),
		targ.Targ(func(ctx context.Context, a PruneArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunPrune(withLog(ctx), a, newOsPruneDeps(), deps.Stdout))
		}).Name("prune").Description(
			"Detach chunk entries whose source file is gone: drop the stale manifest entry, " +
				"keep the embedded chunks (still searchable)"),
		targ.Targ(func(ctx context.Context, a ChunkQueryArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunChunkQuery(withLog(ctx), a, newOsChunkQueryDeps(), deps.Stdout))
		}).Name("query-chunks").Description("Semantic search over the chunk index (YAML output)"),
		targ.Targ(func(_ context.Context, a ActivateArgs) {
			a.Vault = resolveVault(a.Vault, home, deps.Getenv)
			errHandler(RunActivate(a, newOsActivateDeps()))
		}).Name("activate").Description("Mark note(s) as recently used (bumps LastUsed in sidecar)"),
		targ.Targ(func(_ context.Context, a CountArgs) {
			a.Vault = resolveVault(a.Vault, home, deps.Getenv)
			errHandler(RunCount(a, newOsCountDeps(), deps.Stdout))
		}).Name("count").Description(
			"Count notes by a frontmatter attribute or a note's wikilink in-degree (read-only)"),
		targ.Targ(func(ctx context.Context, a ShowArgs) {
			a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
			errHandler(RunShow(withLog(ctx), a, newOsShowDeps(), deps.Stdout))
		}).Name("show").Description("Print a note and its outbound wikilink targets (read-only)"),
		targ.Targ(func(ctx context.Context, a ShowChunkArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunShowChunk(withLog(ctx), a, newOsShowChunkDeps(), deps.Stdout))
		}).Name("show-chunk").Description("Print a chunk's text by its source#anchor id (read-only)"),
		targ.Targ(func(ctx context.Context, a CheckArgs) {
			a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
			errHandler(RunCheck(withLog(ctx), a, newOsCheckDeps(), deps.Stdout))
		}).Name("check").Description("Run vault-invariant checks (exit non-zero on FAIL)"),
	}
}

// learnUpdateTargets returns the learn and update subcommands (learn group,
// update, embed group). Split from coreTargets to stay within the
// per-function length budget.
func learnUpdateTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
) []any {
	home := homeOrEmpty(deps)

	return []any{
		targ.Group("learn",
			targ.Targ(func(ctx context.Context, a LearnFeedbackArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(runLearnFromFeedbackArgs(withLog(ctx), a, deps.Stdout))
			}).Name("feedback").Description("Write a feedback note to the vault"),
			targ.Targ(func(ctx context.Context, a LearnFactArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(runLearnFromFactArgs(withLog(ctx), a, deps.Stdout))
			}).Name("fact").Description("Write a fact note to the vault"),
			targ.Targ(func(ctx context.Context, a LearnQAArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunLearnQA(withLog(ctx), a, newOsLearnQADeps(), deps.Stdout))
			}).Name("qa").Description("Write a QA pair (Q+A notes) to the vault"),
		),
		targ.Targ(func(ctx context.Context, a UpdateArgs) {
			errHandler(runUpdate(withLog(ctx), a, deps.Stdout))
		}).Name("update").Description("Refresh engram binary and harness skills"),
		targ.Group("embed",
			targ.Targ(func(ctx context.Context, a EmbedApplyArgs) {
				a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
				errHandler(RunEmbedApply(withLog(ctx), a, newOsEmbedDeps(), deps.Stdout))
			}).Name("apply").Description("Embed notes (default: missing only)"),
			targ.Targ(func(ctx context.Context, a EmbedStatusArgs) {
				a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
				errHandler(RunEmbedStatus(withLog(ctx), a, newOsEmbedDeps(), deps.Stdout))
			}).Name("status").Description("Report embedding state counts"),
		),
	}
}

// maintenanceTargets returns the vault-maintenance subcommands (resituate,
// amend, vocab). Split out of Targets to keep each function within the length budget;
// the wiring mirrors the other targets exactly.
func maintenanceTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
) []any {
	home := homeOrEmpty(deps)

	return append(
		amendResituateTargets(deps, withLog, errHandler, home),
		vocabTargets(deps, withLog, errHandler, home)...,
	)
}

// vocabTargets returns the vocab group subcommands (bootstrap, stats,
// propose, refit).
func vocabTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
	home string,
) []any {
	return []any{
		targ.Group("vocab",
			targ.Targ(func(ctx context.Context, a VocabBootstrapArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunVocabBootstrap(withLog(ctx), a, newOsVocabDeps(), deps.Stdout))
			}).Name("bootstrap").Description("Seed vocab term notes + tag all existing notes (idempotent)"),
			targ.Targ(func(_ context.Context, a VocabStatsArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunVocabStats(a, newOsVocabStatsDeps(), deps.Stdout))
			}).Name("stats").Description("Print vocab health report (per-term counts, hubs, orphans, untagged rate)"),
			targ.Targ(func(ctx context.Context, a VocabProposeArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunVocabPropose(withLog(ctx), a, newOsVocabDeps(), deps.Stdout))
			}).Name("propose").Description("Add a new vocab term note + minor version bump (LLM gate runs agent-side)"),
			targ.Targ(func(ctx context.Context, a VocabRefitArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunVocabRefit(withLog(ctx), a, newOsVocabDeps(), deps.Stdout))
			}).Name("refit").Description("Apply a refit plan: renames, removals, additions, re-tag, major version bump"),
		),
	}
}
```

   Exhaustive closure inventory (current line numbers in targets.go; T2 change is `os.Getenv`→`deps.Getenv`, `stdout`→`deps.Stdout` shown above; final-form constructor swap is a later one-liner owned by each command cluster):

   | Closure | Lines | Constructor today (kept in T2) | Final form (owning cluster) |
   |---|---|---|---|
   | resituate | 106–109 | `newOsResituateDeps()` | `newResituateDeps(deps)` (resituate) |
   | amend | 110–114 | `newOsAmendDeps()` | `newAmendDeps(deps)` (amend) |
   | query | 152–156 | `newOsQueryDeps()` | `newQueryDeps(deps)` (query) |
   | ingest | 157–160 | `newOsIngestDeps()` | `newIngestDeps(deps)` (ingest) |
   | prune | 161–166 | `newOsPruneDeps()` | `newPruneDeps(deps)` (ingest/prune) |
   | query-chunks | 167–170 | `newOsChunkQueryDeps()` | `newChunkQueryDeps(deps)` (query) |
   | activate | 171–174 | `newOsActivateDeps()` | `newActivateDeps(deps)` (activate) |
   | count | 175–179 | `newOsCountDeps()` | `newCountDeps(deps)` (count) |
   | show | 180–183 | `newOsShowDeps()` | `newShowDeps(deps)` (show) |
   | show-chunk | 184–187 | `newOsShowChunkDeps()` | `newShowChunkDeps(deps)` (show) |
   | check | 188–191 | `newOsCheckDeps()` | `newCheckDeps(deps)` (check) |
   | learn feedback | 207–210 | inside `runLearnFromFeedbackArgs` (learn.go:520) | `runLearnFromFeedbackArgs(ctx, a, deps, stdout)` → `newLearnDeps(deps)` (learn) |
   | learn fact | 211–214 | inside `runLearnFromFactArgs` (learn.go:497) | `runLearnFromFactArgs(ctx, a, deps, stdout)` → `newLearnDeps(deps)` (learn) |
   | learn qa | 215–218 | `newOsLearnQADeps()` | `newLearnQADeps(deps)` (learn) |
   | update | 220–222 | inside `runUpdate` (update.go:276) | `runUpdate(ctx, a, deps, stdout)` using `deps.FS`/`deps.Commander`/`deps.Getenv` (update) |
   | embed apply | 224–227 | `newOsEmbedDeps()` | `newEmbedDeps(deps)` using `deps.Embed` (embed) |
   | embed status | 228–231 | `newOsEmbedDeps()` | `newEmbedDeps(deps)` (embed) |
   | vocab bootstrap | 276–279 | `newOsVocabDeps()` | `newVocabDeps(deps)` (vocab) |
   | vocab stats | 280–283 | `newOsVocabStatsDeps()` | `newVocabStatsDeps(deps)` (vocab) |
   | vocab propose | 284–287 | `newOsVocabDeps()` | `newVocabDeps(deps)` (vocab) |
   | vocab refit | 288–291 | `newOsVocabDeps()` | `newVocabDeps(deps)` (vocab) |

   `os.Getenv` sites replaced (all 22): lines 107, 111, 112, 153, 154, 158, 162, 168, 172, 176, 181, 185, 189, 208, 212, 216, 225, 229, 277, 281, 285, 289. `os.UserHomeDir` site replaced: line 136 (`homeOrEmpty`).

4. [ ] Update the 8 test call sites. Add to `internal/cli/targets_test.go` (package cli_test; add `"io"` and keep `"os"`/`"time"` imports):

```go
// newTestDeps builds a cli.Deps wired to real OS capabilities with captured
// stdout/stderr and a no-op exit — the test analog of cmd/engram's newOsDeps.
// Command clusters extend this as their constructors convert to Deps-based
// composition (#700).
func newTestDeps(stdout, stderr io.Writer) cli.Deps {
	return cli.Deps{
		Stdout:      stdout,
		Stderr:      stderr,
		Exit:        func(int) {},
		Getenv:      os.Getenv,
		Now:         time.Now,
		Getwd:       os.Getwd,
		UserHomeDir: os.UserHomeDir,
	}
}
```

   Replace each site:
   - targets_test.go:153, 289, 307 — current: `targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, func(int) {}, nil)` → `targets := cli.Targets(newTestDeps(&bytes.Buffer{}, &bytes.Buffer{}))`
   - targets_test.go:439 (inside `executeForTest`) — current: `targets := cli.Targets(&stdout, &stderr, func(int) {}, nil)` → `targets := cli.Targets(newTestDeps(&stdout, &stderr))`
   - vocab_commands_test.go:3580, 3606, 3630, 3655 — current: `targets := cli.Targets(&stdout, &stderr, func(int) {}, nil)` → `targets := cli.Targets(newTestDeps(&stdout, &stderr))`

5. [ ] Rewrite `cmd/engram/main.go` (replaces the whole 13-line file) and delete `internal/cli/main.go`; delete `SetupSignalHandling` (signal.go lines for the func, its `io`/`os`/`os/signal`/`syscall`/debuglog imports, and the `signalChannelBuffer` const — final signal.go keeps only `ExitCodeSigInt`, `ForceExitOnRepeatedSignal`, `secondSignal`, importing just `sync/atomic`):

```go
// Package main provides the engram CLI binary entry point (ARCH-6). All
// production I/O adapters live in this package; internal/ holds interfaces
// and pure logic (#700).
package main

import (
	"os"
	"syscall"
	"time"

	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

func main() {
	deps := newOsDeps()

	// Register BEFORE targ.Main: the first SIGINT/SIGTERM flows through
	// targ's context cancellation for graceful shutdown; the second forces
	// exit (cli.ForceExitOnRepeatedSignal).
	registerForceExit(deps.Exit, syscall.SIGINT, syscall.SIGTERM)

	targ.Main(cli.Targets(deps)...)
}

// newOsDeps wires every impure capability the CLI needs from the real OS.
// This is the single production composition root (#700): internal/ receives
// all I/O through this struct. The lazy embedder is constructed once here,
// preserving the one-unpack-per-process property of the old sharedEmbedder
// singleton.
func newOsDeps() cli.Deps {
	home, _ := os.UserHomeDir()

	return cli.Deps{
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Exit:        os.Exit,
		Getenv:      os.Getenv,
		Now:         time.Now,
		Getwd:       os.Getwd,
		UserHomeDir: os.UserHomeDir,
		FS:          osFS{},
		Lock:        flockLocker{},
		Commander:   &osCommander{},
		Embed:       embed.NewLazyEmbedder(cli.CacheDirFromHome(home, embed.BundledModelID, os.Getenv)),
		DebugLog:    openDebugSink(os.Getenv("ENGRAM_DEBUG_LOG")),
	}
}
```

   SEQUENCING: if the update cluster's `cmd/engram/os_update.go` (owner of `osCommander`) has not landed yet, omit the `Commander: &osCommander{},` line — the field stays nil and NOTHING consumes it until `runUpdate` converts; the update cluster adds the line together with the adapter. The `Embed:` line compiles today (`embed.NewLazyEmbedder`, hugot.go:149). The ENGRAM_DEBUG_LOG read (old internal/cli/main.go:23) and its FIXME(#700) comment die with the deleted file; the env read now happens here at the edge.

6. [ ] Run `targ test` — expect all green: debuglog tests (new API), cli tests (new Targets), cmd/engram adapter tests, plus cli_test.go's end-to-end binary build (`go build ./cmd/engram`) still passing.

7. [ ] Purity verification (expect zero matches from the first three; fourth expect only the `_ = targ` free none — file gone):

```
grep -n "os\." internal/cli/targets.go internal/cli/signal.go internal/debuglog/debuglog.go
grep -rn "SetupSignalHandling" --include="*.go" .
grep -n "time.Now\|time.Since" internal/debuglog/debuglog.go
ls internal/cli/main.go
```

   Expected: no `os.` in the three files; no `SetupSignalHandling` anywhere; no `time.Now`/`time.Since` in debuglog.go; `ls` errors (file deleted).

8. [ ] Run `targ check-full` — expect clean (T1's `registerForceExit`/`openDebugSink` now consumed by main.go).

9. [ ] Run the real binary (usable-system check): `go install ./cmd/engram && engram show-chunk --help` and `ENGRAM_DEBUG_LOG=/tmp/engram-700.log engram count --vault "$(mktemp -d)" --attribute type` then `cat /tmp/engram-700.log` — expect help text, a zero count table, and timestamped `[engram]` debug lines proving the sink+logger wiring is live.

10. [ ] Commit:

```
refactor(cli): #700 wire Targets(deps), move composition root to cmd/engram

Targets now takes the cli.Deps capability carrier; signal registration
and the targ.Main call move to cmd/engram/main.go (the single production
composition root). Deletes cli.Main and SetupSignalHandling, drops the
os import from targets.go, and purifies debuglog: New(w io.Writer,
prefix string, now func() time.Time) with a nil-writer no-op, per-line
disk sync now provided by the cmd-side sink.

AI-Used: [claude]
```

---

Key file paths: `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/internal/cli/targets.go` (81–294 rewritten), `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/internal/cli/signal.go`, `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/internal/cli/main.go` (deleted), `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/internal/debuglog/debuglog.go`, `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/cmd/engram/main.go`, plus new `internal/cli/deps.go` and `cmd/engram/{os_fs,os_signal,debuglog_sink}.go` with `_test.go` siblings.

### Task T3 (L1): learn-family — compose LearnDeps/LearnQADeps purely from cli.Deps

**Files**
- Create: `internal/cli/deps_compose.go` (shared EdgeFS/FileLocker composition helpers)
- Create: `internal/cli/deps_compose_test.go` (RED tests for the compositions)
- Modify: `internal/cli/deps.go` (foundation file: add `WriteFileExcl` to EdgeFS — flagged addition)
- Modify: `internal/cli/learn.go` (delete `newOsLearnDeps` + `logWarningToStderrf`; add `newLearnDeps(d Deps)`; re-sign `runLearnFrom*Args`; drop `os` import)
- Modify: `internal/cli/qa.go` (delete `newOsLearnQADeps`; add `newQaDeps(d Deps)`; drop `os` import)
- Modify: `internal/cli/cli.go` (re-parameterize `listRootNotes`; shrink `osLearnFS` to Lock-only; receive relocated `logWarningToStderrf`)
- Modify: `internal/cli/targets.go` (learn-group closures only)
- Modify: `internal/cli/export_test.go` (re-sign fact/feedback exports; add `ExportNewLearnDeps`/`ExportNewQaDeps`; drop `ExportNewOsLearnFS` uses of deleted methods)
- Modify: `internal/cli/testhelpers_test.go` (add `osEdgeFSForTest`, `flockLockerForTest`, `realFSDepsForTest`)
- Modify: `internal/cli/invariants_k1_property_test.go` (K1 drives production `newLearnDeps` over real FS + real flock)
- Modify: `internal/cli/learn_adapters_test.go` (delete `TestOsLearnFS_*` except Lock test; thread Deps into `ExportRunLearnFrom*Args` calls)

**Interfaces**
- Consumes (from foundation `internal/cli/deps.go`): `Deps{Stdout, Stderr io.Writer; Now func() time.Time; Getenv func(string) string; FS EdgeFS; Lock FileLocker; Embed embed.Embedder; ...}`, `EdgeFS`, `FileLocker{ Lock(path string) (unlock func() error, err error) }`
- Produces:
  - `func newLearnDeps(d Deps) LearnDeps`
  - `func newQaDeps(d Deps) LearnQADeps`
  - `func runLearnFromFactArgs(ctx context.Context, a LearnFactArgs, d Deps, stdout io.Writer) error`
  - `func runLearnFromFeedbackArgs(ctx context.Context, a LearnFeedbackArgs, d Deps, stdout io.Writer) error`
  - Shared helpers: `statDirFromFS(fsys EdgeFS) func(string) error`, `initVaultFromFS(fsys EdgeFS) func(string) error`, `listIDsFromFS`, `listBasenamesFromFS`, `listMDFromFS(fsys EdgeFS) func(string) ([]string, error)`, `vaultLockFromLocker(locker FileLocker) func(string) (func(), error)`, `writeNewFromFS`, `writeSidecarFromFS`, `writeNoteAtomicFromFS(fsys EdgeFS, perm fs.FileMode) func(string, []byte) error`, `logWarningTo(w io.Writer) func(string, ...any)`
  - EdgeFS addition (flagged): `WriteFileExcl(path string, data []byte, perm fs.FileMode) error`

**Steps**

- [ ] 1. **RED — real-FS test doubles + composition tests.** Add to `internal/cli/testhelpers_test.go` (package `cli_test`; test files are exempt from the depguard/forbidigo enforcement):

```go
// osEdgeFSForTest is a real-filesystem cli.EdgeFS for tests that drive
// production compositions against t.TempDir(). It mirrors cmd/engram's osFS
// adapter (which is integration-tested in cmd/engram os_fs_test.go).
type osEdgeFSForTest struct{}

func (osEdgeFSForTest) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) } //nolint:wrapcheck // test double

func (osEdgeFSForTest) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm) //nolint:wrapcheck // test double
}

func (osEdgeFSForTest) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}

	name := tmp.Name()

	if _, writeErr := tmp.Write(data); writeErr != nil {
		_ = tmp.Close()
		_ = os.Remove(name)

		return fmt.Errorf("write temp: %w", writeErr)
	}

	if closeErr := tmp.Close(); closeErr != nil {
		_ = os.Remove(name)

		return fmt.Errorf("close temp: %w", closeErr)
	}

	if chmodErr := os.Chmod(name, perm); chmodErr != nil {
		_ = os.Remove(name)

		return fmt.Errorf("chmod temp: %w", chmodErr)
	}

	if renameErr := os.Rename(name, path); renameErr != nil {
		_ = os.Remove(name)

		return fmt.Errorf("rename temp: %w", renameErr)
	}

	return nil
}

func (osEdgeFSForTest) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm) //nolint:gosec // test double
	if err != nil {
		return fmt.Errorf("open excl: %w", err)
	}

	defer func() { _ = f.Close() }()

	if _, writeErr := f.Write(data); writeErr != nil {
		return fmt.Errorf("write excl: %w", writeErr)
	}

	return nil
}

func (osEdgeFSForTest) MkdirAll(path string, perm fs.FileMode) error { return os.MkdirAll(path, perm) }   //nolint:wrapcheck // test double
func (osEdgeFSForTest) MkdirTemp(dir, pattern string) (string, error) { return os.MkdirTemp(dir, pattern) } //nolint:wrapcheck // test double
func (osEdgeFSForTest) Stat(path string) (fs.FileInfo, error)        { return os.Stat(path) }             //nolint:wrapcheck // test double
func (osEdgeFSForTest) ReadDir(path string) ([]fs.DirEntry, error)   { return os.ReadDir(path) }          //nolint:wrapcheck // test double
func (osEdgeFSForTest) Remove(path string) error                     { return os.Remove(path) }           //nolint:wrapcheck // test double
func (osEdgeFSForTest) RemoveAll(path string) error                  { return os.RemoveAll(path) }        //nolint:wrapcheck // test double
func (osEdgeFSForTest) Rename(oldPath, newPath string) error         { return os.Rename(oldPath, newPath) } //nolint:wrapcheck // test double

func (osEdgeFSForTest) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn) //nolint:wrapcheck // test double
}

// flockLockerForTest is a real syscall.Flock cli.FileLocker, mirroring
// cmd/engram's flockLocker so lock-protocol tests race on real OS locks.
type flockLockerForTest struct{}

func (flockLockerForTest) Lock(path string) (func() error, error) {
	const lockPerm = 0o600

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, lockPerm) //nolint:gosec // test double
	if err != nil {
		return nil, fmt.Errorf("open lock: %w", err)
	}

	fileDescriptor := int(f.Fd())

	if flockErr := syscall.Flock(fileDescriptor, syscall.LOCK_EX); flockErr != nil {
		_ = f.Close()

		return nil, fmt.Errorf("flock: %w", flockErr)
	}

	return func() error {
		_ = syscall.Flock(fileDescriptor, syscall.LOCK_UN)

		return f.Close() //nolint:wrapcheck // test double
	}, nil
}

// realFSDepsForTest is the test analogue of cmd/engram's production wiring:
// real filesystem, real flock, discarded output, nil Embed (auto-embed skips).
func realFSDepsForTest() cli.Deps {
	return cli.Deps{
		Stdout: io.Discard,
		Stderr: io.Discard,
		Now:    time.Now,
		Getenv: os.Getenv,
		FS:     osEdgeFSForTest{},
		Lock:   flockLockerForTest{},
	}
}
```

Then create `internal/cli/deps_compose_test.go` (package `cli_test`) — these fail to compile until step 3 (RED):

```go
package cli_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestNewLearnDeps_StatDir_MissingReturnsErrNotExist(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.ExportNewLearnDeps(realFSDepsForTest())

	err := deps.StatDir(filepath.Join(t.TempDir(), "absent"))
	g.Expect(errors.Is(err, fs.ErrNotExist)).To(BeTrue())
}

func TestNewLearnDeps_StatDir_FileIsNotADirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "file.txt")
	g.Expect(os.WriteFile(path, []byte("x"), 0o600)).To(Succeed())

	deps := cli.ExportNewLearnDeps(realFSDepsForTest())
	g.Expect(deps.StatDir(path)).To(MatchError(ContainSubstring("not a directory")))
}

func TestNewLearnDeps_WriteNew_PreservesErrExist(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "existing.md")
	g.Expect(os.WriteFile(path, []byte("already"), 0o600)).To(Succeed())

	deps := cli.ExportNewLearnDeps(realFSDepsForTest())
	err := deps.WriteNew(path, []byte("nope"))
	g.Expect(errors.Is(err, fs.ErrExist)).To(BeTrue(), "O_EXCL backstop must survive composition")

	got, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(got)).To(Equal("already"))
}

func TestNewLearnDeps_InitVault_IdempotentAndPreservesEdits(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "vault")
	deps := cli.ExportNewLearnDeps(realFSDepsForTest())

	g.Expect(deps.InitVault(vault)).To(Succeed())

	readme := filepath.Join(vault, "README.md")
	g.Expect(os.WriteFile(readme, []byte("user edit"), 0o644)).To(Succeed())

	g.Expect(deps.InitVault(vault)).To(Succeed())

	got, readErr := os.ReadFile(readme)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(got)).To(Equal("user edit"), "re-init must not clobber existing files")
}

func TestNewLearnDeps_ListIDs_MissingVaultIsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.ExportNewLearnDeps(realFSDepsForTest())
	got, err := deps.ListIDs(filepath.Join(t.TempDir(), "absent"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(BeEmpty())
}

func TestNewLearnDeps_ListBasenames_SkipsSubdirsAndNonLuhmann(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o700)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "1.2026-05-09.foo.md"), nil, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "README.md"), nil, 0o600)).To(Succeed())

	deps := cli.ExportNewLearnDeps(realFSDepsForTest())
	got, err := deps.ListBasenames(vault)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(ConsistOf("1.2026-05-09.foo"))
}

func TestNewLearnDeps_Lock_AcquiresVaultLuhmannLockFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	deps := cli.ExportNewLearnDeps(realFSDepsForTest())
	release, err := deps.Lock(vault)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	release()

	_, statErr := os.Stat(filepath.Join(vault, ".luhmann.lock"))
	g.Expect(statErr).NotTo(HaveOccurred(), "lock must live at vault/.luhmann.lock")
}

func TestNewLearnDeps_LogWarning_WritesToDepsStderr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stderr bytes.Buffer

	d := realFSDepsForTest()
	d.Stderr = &stderr

	deps := cli.ExportNewLearnDeps(d)
	deps.LogWarning("hello %s", "world")

	g.Expect(stderr.String()).To(Equal("warning: hello world\n"))
}

func TestNewQaDeps_WiresRemoveAndReadThroughFS(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	g.Expect(os.WriteFile(path, []byte("data"), 0o600)).To(Succeed())

	deps := cli.ExportNewQaDeps(realFSDepsForTest())

	got, readErr := deps.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(got)).To(Equal("data"))

	g.Expect(deps.RemoveFile(path)).To(Succeed())
	_, statErr := os.Stat(path)
	g.Expect(errors.Is(statErr, fs.ErrNotExist)).To(BeTrue())
}
```

Run: `targ test` → expected FAIL (compile error: `ExportNewLearnDeps`/`ExportNewQaDeps` undefined). This is the RED.

- [ ] 2. **EdgeFS addition (flagged).** In the foundation's `internal/cli/deps.go`, add to `EdgeFS`:

```go
	// WriteFileExcl creates path exclusively (O_CREATE|O_EXCL semantics): it
	// errors with an error satisfying errors.Is(err, fs.ErrExist) when path
	// already exists. The learn family's ID-collision backstop (ADR-0013 K1)
	// and idempotent vault bootstrap both require exclusive create.
	WriteFileExcl(path string, data []byte, perm fs.FileMode) error
```

and add the matching implementation to `cmd/engram/os_fs.go`'s `osFS` (coordinate with the os_fs owner; body is the current cli.go:115-135 `WriteNew` open/write/close with the caller-supplied perm) plus a cmd/engram integration test asserting `fs.ErrExist` on an existing file.

- [ ] 3. **GREEN — create `internal/cli/deps_compose.go`** (package `cli`; imports `errors`, `fmt`, `io`, `io/fs`, `path/filepath`, `strings` only — no `os`, no `syscall`, no `time`):

```go
package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
)

// unexported constants.
const (
	notePerm    fs.FileMode = 0o600
	sidecarPerm fs.FileMode = 0o600
)

// initVaultFromFS returns an InitVault func composed over the injected EdgeFS.
func initVaultFromFS(fsys EdgeFS) func(string) error {
	return func(path string) error { return initializeVault(edgeVaultInitFS{fsys: fsys}, path) }
}

// edgeVaultInitFS adapts EdgeFS to the VaultInitFS bootstrap surface.
// WriteFileIfMissing swallows fs.ErrExist so re-initialization is idempotent
// and never clobbers user-edited starter files.
type edgeVaultInitFS struct{ fsys EdgeFS }

func (e edgeVaultInitFS) MkdirAll(path string, perm fs.FileMode) error {
	err := e.fsys.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	return nil
}

func (e edgeVaultInitFS) WriteFileIfMissing(path string, data []byte, perm fs.FileMode) error {
	err := e.fsys.WriteFileExcl(path, data, perm)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil
		}

		return fmt.Errorf("write if missing: %w", err)
	}

	return nil
}

// listBasenamesFromFS returns a ListBasenames func: note basenames (filename
// minus .md) for luhmann-id notes at the flat vault root (D1).
func listBasenamesFromFS(fsys EdgeFS) func(string) ([]string, error) {
	return func(vault string) ([]string, error) {
		return listRootNotes(fsys.ReadDir, vault, func(name string) (string, bool) {
			if _, ok := extractLuhmannFromFilename(name); !ok {
				return "", false
			}

			return strings.TrimSuffix(name, ".md"), true
		})
	}
}

// listIDsFromFS returns a ListIDs func: Luhmann IDs from .md filenames at the
// flat vault root.
func listIDsFromFS(fsys EdgeFS) func(string) ([]string, error) {
	return func(vault string) ([]string, error) {
		return listRootNotes(fsys.ReadDir, vault, extractLuhmannFromFilename)
	}
}

// listMDFromFS returns a ListMD func with osVaultFS.ListMD semantics: the .md
// filenames directly inside dir; a missing dir yields (nil, nil).
func listMDFromFS(fsys EdgeFS) func(string) ([]string, error) {
	return func(dir string) ([]string, error) {
		entries, err := fsys.ReadDir(dir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, nil
			}

			return nil, fmt.Errorf("reading dir %s: %w", dir, err)
		}

		out := make([]string, 0, len(entries))

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			out = append(out, entry.Name())
		}

		return out, nil
	}
}

// logWarningTo returns the production LogWarning hook writing to w — the
// Deps-threaded replacement for the old os.Stderr-bound logWarningToStderrf.
func logWarningTo(w io.Writer) func(string, ...any) {
	return func(format string, args ...any) {
		_, _ = fmt.Fprintf(w, "warning: "+format+"\n", args...)
	}
}

// statDirFromFS returns a StatDir func: fs.ErrNotExist when the directory is
// missing, errNotADirectory when the path is a file, wrapped error otherwise.
func statDirFromFS(fsys EdgeFS) func(string) error {
	return func(path string) error {
		info, err := fsys.Stat(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fs.ErrNotExist
			}

			return fmt.Errorf("stat: %w", err)
		}

		if !info.IsDir() {
			return fmt.Errorf("%w: %s", errNotADirectory, path)
		}

		return nil
	}
}

// vaultLockFromLocker returns a vault-lock func over the injected FileLocker:
// an exclusive flock on vault/.luhmann.lock (ADR-0013). The locker's
// unlock-with-error is adapted to the deps structs' plain release func.
func vaultLockFromLocker(locker FileLocker) func(string) (func(), error) {
	return func(vault string) (func(), error) {
		unlock, err := locker.Lock(filepath.Join(vault, luhmannLockFile))
		if err != nil {
			return nil, fmt.Errorf("lock vault: %w", err)
		}

		return func() { _ = unlock() }, nil
	}
}

// writeNewFromFS returns a WriteNew func: exclusive create, preserving
// errors.Is(err, fs.ErrExist) — the K1 collision backstop under the vault lock.
func writeNewFromFS(fsys EdgeFS) func(string, []byte) error {
	return func(path string, data []byte) error {
		err := fsys.WriteFileExcl(path, data, notePerm)
		if err != nil {
			return fmt.Errorf("write new: %w", err)
		}

		return nil
	}
}

// writeNoteAtomicFromFS returns an atomic note-rewrite func at the given perm
// (temp+rename via EdgeFS.WriteFileAtomic — ADR-0013's atomic-rename edge).
func writeNoteAtomicFromFS(fsys EdgeFS, perm fs.FileMode) func(string, []byte) error {
	return func(path string, data []byte) error {
		err := fsys.WriteFileAtomic(path, data, perm)
		if err != nil {
			return fmt.Errorf("write note: %w", err)
		}

		return nil
	}
}

// writeSidecarFromFS returns a WriteSidecar func: atomic .vec.json write.
func writeSidecarFromFS(fsys EdgeFS) func(string, []byte) error {
	return func(path string, data []byte) error {
		err := fsys.WriteFileAtomic(path, data, sidecarPerm)
		if err != nil {
			return fmt.Errorf("write sidecar: %w", err)
		}

		return nil
	}
}
```

- [ ] 4. **cli.go: re-parameterize `listRootNotes`; shrink `osLearnFS` to Lock-only; receive `logWarningToStderrf`.** Replace cli.go:194-221 (`listRootNotes` — current signature `listRootNotes(vault string, extract func(name string) (string, bool))` reading `os.ReadDir(vault)` and checking `os.IsNotExist(err)`) with the injected-ReadDir form (this stays in cli.go, now pure):

```go
// listRootNotes reads the flat vault root via the injected readDir and
// collects one string per non-dir entry for which extract returns ok. A
// missing vault is treated as empty. Shared by listIDsFromFS and
// listBasenamesFromFS so the flat-root traversal lives in exactly one place.
func listRootNotes(
	readDir func(string) ([]fs.DirEntry, error),
	vault string,
	extract func(name string) (string, bool),
) ([]string, error) {
	out := []string{}

	entries, err := readDir(vault)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return out, nil
		}

		return nil, fmt.Errorf("read vault root: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		if s, ok := extract(e.Name()); ok {
			out = append(out, s)
		}
	}

	return out, nil
}
```

Delete `osLearnFS` methods `ListBasenames` (cli.go:39-47), `ListIDs` (49-52), `MkdirAll` (59-67), `StatDir` (69-86), `WriteFileIfMissing` (88-112), `WriteNew` (114-135), `WriteSidecar` (137-150). Keep — verbatim, temporarily, until Task L2 — `osLearnFS` with only its `Lock` method (cli.go:54-57), `flockPath` (165-192), `osManifestLock` (223-236), `osFileReader` (27-31), and `acquireOptionalLock`/`pathOf`/constants/vars. Append the relocated production stderr hook (moved from learn.go:330-334, still needed by activate/amend/resituate/vocab until their migrations):

```go
// logWarningToStderrf is the transitional os.Stderr-bound LogWarning hook.
// Deps-migrated constructors use logWarningTo(d.Stderr) instead; this stays
// only for the not-yet-migrated constructors and dies in the cli.go purge task.
func logWarningToStderrf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}
```

- [ ] 5. **learn.go: pure constructor + threaded wrappers.** Delete `logWarningToStderrf` (learn.go:330-334, relocated in step 4). Replace `newOsLearnDeps` (learn.go:347-378) with:

```go
// newLearnDeps composes LearnDeps purely from the injected edge capabilities.
// All I/O flows through d.FS / d.Lock / d.Embed / d.Stderr — no direct os use.
func newLearnDeps(d Deps) LearnDeps {
	return LearnDeps{
		Now:           d.Now,
		Getenv:        d.Getenv,
		StatDir:       statDirFromFS(d.FS),
		InitVault:     initVaultFromFS(d.FS),
		ListIDs:       listIDsFromFS(d.FS),
		ListBasenames: listBasenamesFromFS(d.FS),
		Lock:          vaultLockFromLocker(d.Lock),
		WriteNew:      writeNewFromFS(d.FS),
		Embedder:      d.Embed,
		WriteSidecar:  writeSidecarFromFS(d.FS),
		LogWarning:    logWarningTo(d.Stderr),
		// Vocab assignment wiring: no-op when the vault has no term notes.
		// Uses stored member centroids (vocab.centroids.json) when present,
		// falling back to description embeddings per term.
		LoadTermVectors: func(vault string) ([]TermWithVector, error) {
			return loadAssignmentTermVectors(vault, listMDFromFS(d.FS), d.FS.ReadFile)
		},
		ReadSidecar: d.FS.ReadFile,
		WriteNote:   writeNoteAtomicFromFS(d.FS, vocabNotePerm),
		// ListMD provides full .md filenames for the vocab trigger scan.
		// Must be full filenames (not stripped basenames) — ListBasenames
		// filters to Luhmann IDs, causing 100% false-fire on the untagged trigger.
		ListMD: listMDFromFS(d.FS),
	}
}
```

Re-sign the two wrappers (current learn.go:497-541, `deps := newOsLearnDeps()`):

```go
func runLearnFromFactArgs(ctx context.Context, a LearnFactArgs, d Deps, stdout io.Writer) error {
	return runLearn(ctx, LearnArgs{
		Type:         typeFact,
		Slug:         a.Slug,
		Vault:        a.Vault,
		Target:       a.Target,
		Position:     a.Position,
		Source:       a.Source,
		Project:      a.Project,
		Issue:        a.Issue,
		Tier:         a.Tier,
		Supersedes:   a.Supersedes,
		ChunkSources: a.ChunkSources,
		Tags:         a.Tags,
		Situation:    a.Situation,
		Subject:      a.Subject,
		Predicate:    a.Predicate,
		Object:       a.Object,
	}, newLearnDeps(d), stdout)
}

func runLearnFromFeedbackArgs(ctx context.Context, a LearnFeedbackArgs, d Deps, stdout io.Writer) error {
	return runLearn(ctx, LearnArgs{
		Type:         typeFeedback,
		Slug:         a.Slug,
		Vault:        a.Vault,
		Target:       a.Target,
		Position:     a.Position,
		Source:       a.Source,
		Project:      a.Project,
		Issue:        a.Issue,
		Tier:         a.Tier,
		Supersedes:   a.Supersedes,
		ChunkSources: a.ChunkSources,
		Tags:         a.Tags,
		Situation:    a.Situation,
		Behavior:     a.Behavior,
		Impact:       a.Impact,
		Action:       a.Action,
	}, newLearnDeps(d), stdout)
}
```

Remove `"os"` from learn.go's import block (learn.go:8 — after these edits nothing in the file references `os`; `time` stays for `time.Time` types only, no `time.Now` reference remains).

- [ ] 6. **qa.go: pure constructor.** Replace `newOsLearnQADeps` (qa.go:260-286) with:

```go
// newQaDeps composes LearnQADeps purely from the injected edge capabilities.
func newQaDeps(d Deps) LearnQADeps {
	return LearnQADeps{
		Now:          d.Now,
		Getenv:       d.Getenv,
		StatDir:      statDirFromFS(d.FS),
		InitVault:    initVaultFromFS(d.FS),
		ListMD:       listMDFromFS(d.FS),
		Lock:         vaultLockFromLocker(d.Lock),
		WriteNew:     writeNewFromFS(d.FS),
		RemoveFile:   d.FS.Remove,
		ReadFile:     d.FS.ReadFile,
		Embedder:     d.Embed,
		WriteSidecar: writeSidecarFromFS(d.FS),
		LogWarning:   logWarningTo(d.Stderr),
		LoadTermVectors: func(vault string) ([]TermWithVector, error) {
			return loadAssignmentTermVectors(vault, listMDFromFS(d.FS), d.FS.ReadFile)
		},
		ReadSidecar: d.FS.ReadFile,
		WriteNote:   writeNoteAtomicFromFS(d.FS, vocabNotePerm),
	}
}
```

Remove `"os"` and `"time"`?—no: keep `"time"` (`time.Time` in signatures); remove `"os"` from qa.go's imports (qa.go:9 — `os.Getenv`/`os.Remove` were its only uses).

- [ ] 7. **targets.go: thread Deps through the learn group.** In `learnUpdateTargets` (targets.go:198-234; foundation task threads `d Deps` into this function's signature — coordinate), replace the three learn closures (current lines 206-218):

```go
		targ.Group("learn",
			targ.Targ(func(ctx context.Context, a LearnFeedbackArgs) {
				a.Vault = resolveVault(a.Vault, home, os.Getenv)
				errHandler(runLearnFromFeedbackArgs(withLog(ctx), a, d, stdout))
			}).Name("feedback").Description("Write a feedback note to the vault"),
			targ.Targ(func(ctx context.Context, a LearnFactArgs) {
				a.Vault = resolveVault(a.Vault, home, os.Getenv)
				errHandler(runLearnFromFactArgs(withLog(ctx), a, d, stdout))
			}).Name("fact").Description("Write a fact note to the vault"),
			targ.Targ(func(ctx context.Context, a LearnQAArgs) {
				a.Vault = resolveVault(a.Vault, home, os.Getenv)
				errHandler(RunLearnQA(withLog(ctx), a, newQaDeps(d), stdout))
			}).Name("qa").Description("Write a QA pair (Q+A notes) to the vault"),
		),
```

(The `resolveVault(a.Vault, home, os.Getenv)` → `d.Getenv`/`d.UserHomeDir` swap is the targets cluster's charge — leave those expressions to that task.)

- [ ] 8. **export_test.go:** re-sign the two wrappers' exports (current lines 836-846) and add the constructor exports:

```go
// ExportNewLearnDeps exposes the pure Deps→LearnDeps composition for tests.
func ExportNewLearnDeps(d Deps) LearnDeps { return newLearnDeps(d) }

// ExportNewQaDeps exposes the pure Deps→LearnQADeps composition for tests.
func ExportNewQaDeps(d Deps) LearnQADeps { return newQaDeps(d) }

// ExportRunLearnFromFactArgs invokes the unexported runLearnFromFactArgs for testing.
func ExportRunLearnFromFactArgs(ctx context.Context, a LearnFactArgs, d Deps, stdout io.Writer) error {
	return runLearnFromFactArgs(ctx, a, d, stdout)
}

// ExportRunLearnFromFeedbackArgs invokes the unexported runLearnFromFeedbackArgs for testing.
func ExportRunLearnFromFeedbackArgs(
	ctx context.Context,
	a LearnFeedbackArgs,
	d Deps,
	stdout io.Writer,
) error {
	return runLearnFromFeedbackArgs(ctx, a, d, stdout)
}
```

Keep `ExportNewOsLearnFS` and `ExportFlockPath` untouched (Lock-only consumers survive until L2).

- [ ] 9. **learn_adapters_test.go:** delete `TestOsLearnFS_ListBasenames_*`, `TestOsLearnFS_ListIDs_*`, `TestOsLearnFS_MkdirAll_*`, `TestOsLearnFS_StatDir_*`, `TestOsLearnFS_WriteFileIfMissing_*`, `TestOsLearnFS_WriteNew_*` (lines 58-122, 133-288 — behavior now covered by deps_compose_test.go against the production composition + the cmd/engram osFS integration tests). Keep `TestOsLearnFS_Lock_BadVaultReturnsError` (124-131) until L2. Update every `cli.ExportRunLearnFromFactArgs(context.Background(), args, io.Discard)` / `...FeedbackArgs(...)` call (lines 37, 310, 371, 408, 456, 489) to `cli.ExportRunLearnFromFactArgs(context.Background(), args, realFSDepsForTest(), io.Discard)` (same for feedback). Note: with `Embed` nil these tests now skip auto-embed (they only assert `.md` presence/absence — assertions unchanged; the embed-on-write path stays covered by cli_test.go's real-binary end-to-end test, which asserts the `.vec.json` sidecar).

- [ ] 10. **K1 regression test survives (ADR-0013).** In `invariants_k1_property_test.go`, replace `k1RealLockDeps` (lines 122-140) — the test body (30-120) is unchanged:

```go
// unexported variables.
var errK1VaultMissing = errors.New("k1: vault should already exist")

// k1RealLockDeps wires LearnDeps through the PRODUCTION composition
// (newLearnDeps) over a real-filesystem EdgeFS and a real syscall.Flock
// FileLocker — the same flock + exclusive-create (WriteFileExcl) path the
// shipped binary composes in cmd/engram. Embed is nil (zero Deps field) so
// auto-embed skips; InitVault errors because the caller pre-creates the vault.
func k1RealLockDeps(vault string) cli.LearnDeps {
	deps := cli.ExportNewLearnDeps(realFSDepsForTest())

	deps.InitVault = func(string) error {
		return fmt.Errorf("%w: %s", errK1VaultMissing, vault)
	}

	return deps
}
```

(Add `"errors"` to the file's imports; `"time"` and the `os.Getenv` import uses move into `realFSDepsForTest`.) This upgrade means K1 now races the production composition layer itself — lock file `vault/.luhmann.lock`, span ListIDs→WriteNew, O_EXCL backstop — not a hand-wired double of it.

- [ ] 11. Run `targ test` → all green (RED tests from step 1 now pass; K1 passes at workers=2,5,10,20). Run `targ check-full` → clean. Run `go install ./cmd/engram && cd "$(mktemp -d)" && engram learn fact --slug smoke --vault "$(mktemp -d)/v" --position top --source smoke --situation "smoke" --subject s --predicate p --object o` → prints the note path; the note and `.vec.json` sidecar exist.

- [ ] 12. Commit:

```
refactor(cli): compose learn-family deps purely from injected Deps (#700)

newLearnDeps/newQaDeps replace newOsLearnDeps/newOsLearnQADeps; all learn/qa
I/O flows through EdgeFS/FileLocker/Embed/Stderr. Adds EdgeFS.WriteFileExcl
(exclusive create) so the ADR-0013 K1 O_EXCL backstop survives composition;
K1 concurrency property now drives the production composition over real flock.

AI-Used: [claude]
```

---

### Task T4 (L2): purge os/syscall adapters from internal/cli/cli.go (ORDER: after activate/amend/resituate/vocab/ingest/prune constructor migrations, before enforcement)

**Files**
- Modify: `internal/cli/cli.go` (delete `osFileReader`, `osLearnFS`, `flockPath`, `osManifestLock`, `logWarningToStderrf`; drop `os`/`syscall` imports)
- Modify: `internal/cli/export_test.go` (re-implement `ExportFlockPath` as test-local real flock; delete `ExportNewOsLearnFS`, `ExportOsManifestLock`, `ExportLogWarningToStderr`)
- Modify: `internal/cli/learn_adapters_test.go` (delete `TestOsLearnFS_Lock_BadVaultReturnsError`)
- Modify: `internal/cli/testhelpers_test.go` (delete `TestOsManifestLock_MkdirError`)
- Modify: `internal/cli/os_adapters_test.go` (delete `TestExportLogWarningToStderr_FormatsAndWrites` — superseded by `TestNewLearnDeps_LogWarning_WritesToDepsStderr`)

**Interfaces**
- Consumes: nothing new. Produces: none — pure deletion plus a test-only flock.

**Steps**

- [ ] 1. **Precondition gate (must pass before any edit):**
`grep -rn "osLearnFS\|flockPath\|osManifestLock\|logWarningToStderrf\|osFileReader" internal/ --include="*.go" | grep -v "_test.go"` → expected: hits ONLY inside `internal/cli/cli.go` (the definitions themselves). Any hit in activate.go/amend.go/resituate.go/vocab_commands.go/ingest.go/prune.go means a family migration hasn't landed — STOP and reorder.

- [ ] 2. **Keep the manifest concurrent-writers regression green (RED-equivalent for this refactor is the existing suite).** In `export_test.go`, replace the delegating `ExportFlockPath` (lines 342-347) with a self-contained real flock — test files are exempt from the depguard/forbidigo deny, so `syscall` is legal here and `TestManifest_ConcurrentWritersDoNotLoseEntries` (ingest_test.go:319) keeps racing real OS locks without any production flock symbol:

```go
// ExportFlockPath is a test-only real flock used by the concurrent-writers
// regression tests (ingest_test.go). The goroutines race on the SAME OS lock
// file to prove the locking protocol prevents lost updates; production locking
// lives in cmd/engram's flockLocker (integration-tested there).
func ExportFlockPath(lockPath string) (func(), error) {
	const lockPerm = 0o600

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, lockPerm) //nolint:gosec // test helper
	if err != nil {
		return nil, fmt.Errorf("open lock: %w", err)
	}

	fileDescriptor := int(f.Fd())

	flockErr := syscall.Flock(fileDescriptor, syscall.LOCK_EX)
	if flockErr != nil {
		_ = f.Close()

		return nil, fmt.Errorf("flock: %w", flockErr)
	}

	return func() {
		_ = syscall.Flock(fileDescriptor, syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}
```

(Add `"syscall"` to export_test.go's imports; `ExportManifestLockFile` stays — the constant it returns survives.) Delete `ExportNewOsLearnFS` (553-554), `ExportOsManifestLock` (687-690), and the `ExportLogWarningToStderr` alias (line 70).

- [ ] 3. **Delete from cli.go:** `osFileReader` (27-31), `osLearnFS` + its `Lock` method (33-57 remnant), `flockPath` (165-192), `osManifestLock` (223-236), `logWarningToStderrf` (relocated block from L1 step 4). Remove `"os"`, `"syscall"`, and `"path/filepath"` (if now unused — `filepath.Join` was only used by the deleted lock helpers; `vaultLockFromLocker` in deps_compose.go has its own import) from the import block. cli.go retains: package doc, `luhmannLockFile`/`manifestLockFile` constants (consumed by `vaultLockFromLocker` and the ingest cluster's manifest-lock composition + `ExportManifestLockFile`), `errNotADirectory` (consumed by `statDirFromFS`), `acquireOptionalLock` (152-163, pure), `listRootNotes` (pure since L1), `pathOf` (240-243).

- [ ] 4. Delete `TestOsLearnFS_Lock_BadVaultReturnsError` (learn_adapters_test.go:124-131), `TestOsManifestLock_MkdirError` (testhelpers_test.go), `TestExportLogWarningToStderr_FormatsAndWrites` (os_adapters_test.go:151-172). Confirm the replacement coverage exists before deleting: cmd/engram `flockLocker` integration test covers lock-open failure; the ingest cluster's manifest-lock composition test covers mkdir-before-lock; `TestNewLearnDeps_LogWarning_WritesToDepsStderr` covers the warning format.

- [ ] 5. Verify purity of the migrated files: `grep -n "\"os\"\|\"syscall\"\|os\.\|syscall\." internal/cli/cli.go internal/cli/learn.go internal/cli/qa.go internal/cli/deps_compose.go` → no hits. Run `targ test` → green, including `TestInvariant_K1_ConcurrentLearnNeverCollides` and `TestManifest_ConcurrentWritersDoNotLoseEntries`. Run `targ check-full` → clean. `go install ./cmd/engram` and re-run the step-11 smoke from L1.

- [ ] 6. Commit:

```
refactor(cli): delete os/syscall adapters from cli.go (#700)

osLearnFS, flockPath, osManifestLock, osFileReader, and the stderr warning
hook are gone from internal; production I/O lives in cmd/engram. The
concurrent-writers regression suites keep racing real OS flocks via a
test-local ExportFlockPath (test files are exempt from the purity deny).

AI-Used: [claude]
```

---

**ADR-0013 survival summary:** lock semantics are preserved exactly — same lock files (`vault/.luhmann.lock` via `vaultLockFromLocker`, `chunksDir/.manifest.lock` on the ingest side), same acquisition points (`writeLearnUnderLock` learn.go:657, `writeQANotesUnderLock` qa.go:427, both inside `Run*` entry flows), same span (ListIDs→WriteNew under one exclusive flock), same O_EXCL backstop (`EdgeFS.WriteFileExcl`), same atomic temp+rename edge (`EdgeFS.WriteFileAtomic`). The two regression tests survive: `TestInvariant_K1_ConcurrentLearnNeverCollides` (adapted in L1 step 10 to drive the production `newLearnDeps` composition over a real flock — strictly stronger than before) and `TestManifest_ConcurrentWritersDoNotLoseEntries` (unchanged; its `ExportFlockPath` dependency is re-implemented test-locally in L2 step 2).

## Complete os/time inventory for the cluster (line → replacement)

| File:Line | Current | Replacement |
|---|---|---|
| query.go:14 | `"time"` import | RETAINED — type-only use (`time.Time`/`time.Duration`) after fix |
| query.go:66,274-276,282-284,671,1302-1307,1403-1410,1426 | injected-clock/type usage | unchanged (already DI) |
| query.go:1288 | `newOsEmbedDeps()` (os-backed via embed.go) | direct composition: `ScanVault(newVaultFS(d.FS), …)` + `d.FS.ReadFile` + `d.Embed` |
| query.go:1294 | `logWarningToStderrf` (os.Stderr via learn.go:333) | `logWarningTo(d.Stderr)` |
| query.go:1295 | `ListChunkIndexes: listJSONLIndexes` (os.ReadDir) | `listJSONLIndexes(d.FS)` |
| query.go:1296 | `Now: time.Now` | `Now: d.Now` |
| query_chunks.go:8 | `"os"` import | deleted; add `"io/fs"` |
| query_chunks.go:139 | `os.ReadDir(dir)` | `fsys.ReadDir(dir)` (EdgeFS, via closure) |
| query_chunks.go:141 | `os.IsNotExist(err)` | `errors.Is(err, fs.ErrNotExist)` |
| query_chunks.go:188 | `fs := &osEmbedFS{}` | `d.FS` (newChunkQueryDeps) |
| query_chunks.go:193 | `Embedder: sharedEmbedder` | `Embedder: d.Embed` |
| query_chunks.go:12,206 | `"time"` / `now time.Time` param | RETAINED — pure |
| query_nominations.go | — | no os/time references |
| count.go:123 | `&osVaultFS{}` | `newVaultFS(d.FS)`; `newOsCountDeps()` → `newCountDeps(d Deps)` |
| show.go:69 | `&osVaultFS{}` | `newVaultFS(d.FS)`; `newOsShowDeps()` → `newShowDeps(d Deps)` |
| check.go:223 | `&osVaultFS{}` | `newVaultFS(d.FS)`; `newOsCheckDeps()` → `newCheckDeps(d Deps)` |
| vault_fs.go:6 | `"os"` import | deleted in Q3 (retained through Q1/Q2 for legacy `osVaultFS`) |
| vault_fs.go:23 | `os.ReadFile(filepath.Clean(path))` | `v.fs.ReadFile(filepath.Clean(path))` |
| vault_fs.go:35 | `os.ReadDir(dir)` | injected `readDir(dir)` (production = `EdgeFS.ReadDir`) |
| vault_fs.go:37 | `errors.Is(err, os.ErrNotExist)` | `errors.Is(err, fs.ErrNotExist)` (same sentinel; `io/fs`) |
| vault_fs.go:14-29 | `osVaultFS` type + methods | deleted in Q3 |
| targets.go:155,169,177,182,190 | `newOsQueryDeps()` / `newOsChunkQueryDeps()` / `newOsCountDeps()` / `newOsShowDeps()` / `newOsCheckDeps()` | `newQueryDeps(d)` / `newChunkQueryDeps(d)` / `newCountDeps(d)` / `newShowDeps(d)` / `newCheckDeps(d)` (targets cluster owns the file; same-commit edits) |

---

### Task T5 (Q1): Pure `vaultFS` adapter over EdgeFS; migrate count/show/check

**Files:**
- Create: `internal/cli/edgefs_os_test.go` (skip creation if an identical os-backed test EdgeFS already landed from another cluster — grep `osTestEdgeFS` first)
- Modify: `internal/cli/vault_fs.go`, `internal/cli/count.go`, `internal/cli/show.go`, `internal/cli/check.go`, `internal/cli/export_test.go`, `internal/cli/vault_fs_test.go`, `internal/cli/count_test.go`, `internal/cli/check_test.go`, `internal/cli/show_test.go`, `internal/cli/targets.go` (3 call-site lines)
- Delete: none (osVaultFS deletion deferred to Q3)

**Interfaces:**
- Consumes: `cli.Deps` / `cli.EdgeFS` (foundation task, internal/cli/deps.go); `vaultgraph.VaultFS` (`ListMD(dir string) ([]string, error)`; `ReadFile(path string) ([]byte, error)` — internal/vaultgraph/scanner.go:20); targets-cluster `d Deps` in `ingestQueryTargets`
- Produces: `func newVaultFS(fsys EdgeFS) *vaultFS` (satisfies vaultgraph.VaultFS); `func newCountDeps(d Deps) CountDeps`; `func newShowDeps(d Deps) ShowDeps`; `func newCheckDeps(d Deps) CheckDeps`; test shims `ExportNewVaultFS(fsys EdgeFS)`, `ExportNewCheckDeps(fsys EdgeFS) CheckDeps`, `ExportNewCountDeps(fsys EdgeFS) CountDeps`, `ExportNewShowDeps(fsys EdgeFS) ShowDeps`; shared test double `osTestEdgeFS` + `wrappedNotExistEdgeFS` (package cli_test)

**Steps:**

1. [ ] RED — create `internal/cli/edgefs_os_test.go` (package cli_test) with the real-FS EdgeFS double, and rewrite `internal/cli/vault_fs_test.go` to drive the not-yet-existing `cli.ExportNewVaultFS`. Run `targ test` — expected: compile failure (`undefined: cli.ExportNewVaultFS`, `undefined: cli.EdgeFS` absent if foundation not landed — foundation is a hard precondition).

   `edgefs_os_test.go` (complete):
   ```go
   package cli_test

   import (
   	"fmt"
   	"io/fs"
   	"os"
   	"path/filepath"
   )

   // osTestEdgeFS is a real-filesystem cli.EdgeFS for integration-style tests
   // over t.TempDir() fixtures. The production EdgeFS adapter lives in
   // cmd/engram (package main) and is not importable here; test files are
   // exempt from the internal/ purity rules, so this double calls os directly.
   // Errors are wrapped with %w, matching the production contract that
   // errors.Is(err, fs.ErrNotExist) must survive the adapter.
   // WriteFileAtomic is a plain write — ADR-0013 atomicity is exercised by
   // cmd/engram's own integration tests, never through this double.
   type osTestEdgeFS struct{}

   func (osTestEdgeFS) ReadFile(path string) ([]byte, error) {
   	data, err := os.ReadFile(path) //nolint:gosec // test fixture path
   	if err != nil {
   		return nil, fmt.Errorf("test edgefs: reading %s: %w", path, err)
   	}

   	return data, nil
   }

   func (osTestEdgeFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
   	err := os.WriteFile(path, data, perm)
   	if err != nil {
   		return fmt.Errorf("test edgefs: writing %s: %w", path, err)
   	}

   	return nil
   }

   func (osTestEdgeFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
   	err := os.WriteFile(path, data, perm)
   	if err != nil {
   		return fmt.Errorf("test edgefs: writing %s: %w", path, err)
   	}

   	return nil
   }

   func (osTestEdgeFS) MkdirAll(path string, perm fs.FileMode) error {
   	err := os.MkdirAll(path, perm)
   	if err != nil {
   		return fmt.Errorf("test edgefs: mkdir %s: %w", path, err)
   	}

   	return nil
   }

   func (osTestEdgeFS) MkdirTemp(dir, pattern string) (string, error) {
   	path, err := os.MkdirTemp(dir, pattern)
   	if err != nil {
   		return "", fmt.Errorf("test edgefs: mkdtemp %s: %w", dir, err)
   	}

   	return path, nil
   }

   func (osTestEdgeFS) Stat(path string) (fs.FileInfo, error) {
   	info, err := os.Stat(path)
   	if err != nil {
   		return nil, fmt.Errorf("test edgefs: stat %s: %w", path, err)
   	}

   	return info, nil
   }

   func (osTestEdgeFS) ReadDir(path string) ([]fs.DirEntry, error) {
   	entries, err := os.ReadDir(path)
   	if err != nil {
   		return nil, fmt.Errorf("test edgefs: readdir %s: %w", path, err)
   	}

   	return entries, nil
   }

   func (osTestEdgeFS) Remove(path string) error {
   	err := os.Remove(path)
   	if err != nil {
   		return fmt.Errorf("test edgefs: remove %s: %w", path, err)
   	}

   	return nil
   }

   func (osTestEdgeFS) RemoveAll(path string) error {
   	err := os.RemoveAll(path)
   	if err != nil {
   		return fmt.Errorf("test edgefs: removeall %s: %w", path, err)
   	}

   	return nil
   }

   func (osTestEdgeFS) Rename(oldPath, newPath string) error {
   	err := os.Rename(oldPath, newPath)
   	if err != nil {
   		return fmt.Errorf("test edgefs: rename %s: %w", oldPath, err)
   	}

   	return nil
   }

   func (osTestEdgeFS) WalkDir(root string, fn fs.WalkDirFunc) error {
   	err := filepath.WalkDir(root, fn)
   	if err != nil {
   		return fmt.Errorf("test edgefs: walk %s: %w", root, err)
   	}

   	return nil
   }
   ```

   `vault_fs_test.go` (complete replacement; current file drives `cli.ExportNewOsVaultFS()` at lines 22/37/56/65/77):
   ```go
   package cli_test

   import (
   	"fmt"
   	"io/fs"
   	"os"
   	"path/filepath"
   	"testing"

   	. "github.com/onsi/gomega"

   	"github.com/toejough/engram/internal/cli"
   )

   func TestVaultFS_ListMD_FiltersDirsAndNonMd(t *testing.T) {
   	t.Parallel()
   	g := NewWithT(t)

   	dir := t.TempDir()
   	g.Expect(os.MkdirAll(filepath.Join(dir, "subdir"), 0o750)).To(Succeed())
   	g.Expect(os.WriteFile(filepath.Join(dir, "note.md"), []byte("x"), 0o600)).To(Succeed())
   	g.Expect(os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("y"), 0o600)).To(Succeed())

   	vfs := cli.ExportNewVaultFS(osTestEdgeFS{})
   	names, err := vfs.ListMD(dir)
   	g.Expect(err).NotTo(HaveOccurred())

   	if err != nil {
   		return
   	}

   	g.Expect(names).To(ConsistOf("note.md"))
   }

   func TestVaultFS_ListMD_MissingDirReturnsEmpty(t *testing.T) {
   	t.Parallel()
   	g := NewWithT(t)

   	vfs := cli.ExportNewVaultFS(osTestEdgeFS{})
   	names, err := vfs.ListMD("/nonexistent/vault/dir")
   	g.Expect(err).NotTo(HaveOccurred())

   	if err != nil {
   		return
   	}

   	g.Expect(names).To(BeEmpty())
   }

   func TestVaultFS_ListMD_NonExistError(t *testing.T) {
   	t.Parallel()
   	g := NewWithT(t)

   	// dir is a regular file, not a directory → ReadDir returns ENOTDIR (not not-exist).
   	path := filepath.Join(t.TempDir(), "file")
   	g.Expect(os.WriteFile(path, []byte("x"), 0o600)).To(Succeed())

   	vfs := cli.ExportNewVaultFS(osTestEdgeFS{})
   	_, err := vfs.ListMD(path)
   	g.Expect(err).To(HaveOccurred())
   }

   func TestVaultFS_ListMD_WrappedNotExistReturnsEmpty(t *testing.T) {
   	t.Parallel()
   	g := NewWithT(t)

   	// EdgeFS implementations wrap errors with %w; missing-dir detection must
   	// survive wrapping (errors.Is unwraps; os.IsNotExist would not).
   	vfs := cli.ExportNewVaultFS(wrappedNotExistEdgeFS{})
   	names, err := vfs.ListMD("/anything")
   	g.Expect(err).NotTo(HaveOccurred())

   	if err != nil {
   		return
   	}

   	g.Expect(names).To(BeEmpty())
   }

   func TestVaultFS_ReadFile_MissingPathError(t *testing.T) {
   	t.Parallel()
   	g := NewWithT(t)

   	vfs := cli.ExportNewVaultFS(osTestEdgeFS{})
   	_, err := vfs.ReadFile("/nonexistent/path.md")
   	g.Expect(err).To(HaveOccurred())
   }

   func TestVaultFS_ReadFile_Success(t *testing.T) {
   	t.Parallel()
   	g := NewWithT(t)

   	path := filepath.Join(t.TempDir(), "f.md")
   	g.Expect(os.WriteFile(path, []byte("hello"), 0o600)).To(Succeed())

   	vfs := cli.ExportNewVaultFS(osTestEdgeFS{})
   	data, err := vfs.ReadFile(path)
   	g.Expect(err).NotTo(HaveOccurred())

   	if err != nil {
   		return
   	}

   	g.Expect(string(data)).To(Equal("hello"))
   }

   // wrappedNotExistEdgeFS overrides ReadDir to return a WRAPPED fs.ErrNotExist,
   // proving missing-dir detection unwraps through EdgeFS error wrapping.
   type wrappedNotExistEdgeFS struct{ osTestEdgeFS }

   func (wrappedNotExistEdgeFS) ReadDir(string) ([]fs.DirEntry, error) {
   	return nil, fmt.Errorf("listing: %w", fs.ErrNotExist)
   }
   ```

2. [ ] GREEN — rewrite `internal/cli/vault_fs.go`. Replace the whole file body EXCEPT keep the legacy `osVaultFS` (with its own inline `os.ReadDir` call routed through the shared helper) until Q3:
   ```go
   package cli

   import (
   	"errors"
   	"fmt"
   	"io/fs"
   	"os"
   	"path/filepath"
   	"strings"
   )

   // vaultFS adapts the injected EdgeFS to vaultgraph.VaultFS — pure
   // composition, all I/O flows through the EdgeFS wired by cmd/engram (#700).
   // Listing a non-existent directory returns an empty slice (not an error) —
   // the scanner uses this to skip missing subdirs (e.g. an absent MOCs/ on a
   // brand-new vault).
   type vaultFS struct {
   	fs EdgeFS
   }

   // newVaultFS returns a vaultgraph.VaultFS view over fsys.
   func newVaultFS(fsys EdgeFS) *vaultFS {
   	return &vaultFS{fs: fsys}
   }

   // ListMD returns the .md filenames in dir. Missing dir → empty, nil.
   func (v *vaultFS) ListMD(dir string) ([]string, error) {
   	return listDirBySuffix(v.fs.ReadDir, dir, ".md")
   }

   // ReadFile reads the file at path.
   func (v *vaultFS) ReadFile(path string) ([]byte, error) {
   	data, err := v.fs.ReadFile(filepath.Clean(path))
   	if err != nil {
   		return nil, fmt.Errorf("reading %s: %w", path, err)
   	}

   	return data, nil
   }

   // osVaultFS is the LEGACY direct-os adapter satisfying vaultgraph.VaultFS.
   // Deleted by the #700 purge task once amend/learn/qa/resituate/embed/vocab
   // migrate to newVaultFS(d.FS). Do not add new consumers.
   type osVaultFS struct{}

   // ListMD returns the .md filenames in dir. Missing dir → empty, nil.
   func (*osVaultFS) ListMD(dir string) ([]string, error) {
   	return listDirBySuffix(os.ReadDir, dir, ".md")
   }

   // ReadFile reads the file at path.
   func (*osVaultFS) ReadFile(path string) ([]byte, error) {
   	data, err := os.ReadFile(filepath.Clean(path))
   	if err != nil {
   		return nil, fmt.Errorf("reading %s: %w", path, err)
   	}

   	return data, nil
   }

   // listDirBySuffix returns the filenames directly inside dir whose name has
   // the given suffix, using the injected readDir (EdgeFS.ReadDir in
   // production). Missing dir → empty, nil — matched via errors.Is so wrapped
   // not-exist errors from EdgeFS adapters are recognized.
   func listDirBySuffix(
   	readDir func(string) ([]fs.DirEntry, error),
   	dir, suffix string,
   ) ([]string, error) {
   	entries, err := readDir(dir)
   	if err != nil {
   		if errors.Is(err, fs.ErrNotExist) {
   			return nil, nil
   		}

   		return nil, fmt.Errorf("reading dir %s: %w", dir, err)
   	}

   	out := make([]string, 0, len(entries))

   	for _, entry := range entries {
   		if entry.IsDir() {
   			continue
   		}

   		if !strings.HasSuffix(entry.Name(), suffix) {
   			continue
   		}

   		out = append(out, entry.Name())
   	}

   	return out, nil
   }
   ```
   (`os.ReadDir` is directly assignable to `func(string) ([]fs.DirEntry, error)` — `os.DirEntry` is a type alias of `fs.DirEntry`; `os.ErrNotExist` IS `fs.ErrNotExist`, so behavior is identical.)

3. [ ] Convert the three constructors (current code verified):
   - count.go:121-132, current:
     ```go
     // newOsCountDeps wires RunCount to the real filesystem vault reader/scanner.
     func newOsCountDeps() CountDeps {
     	fsys := &osVaultFS{}

     	return CountDeps{
     		ListMD:   fsys.ListMD,
     		ReadFile: fsys.ReadFile,
     		Scan: func(vault string) ([]vaultgraph.Note, error) {
     			return vaultgraph.ScanVault(fsys, vault)
     		},
     	}
     }
     ```
     replacement:
     ```go
     // newCountDeps wires RunCount from the injected CLI capabilities — pure
     // composition over EdgeFS (#700).
     func newCountDeps(d Deps) CountDeps {
     	fsys := newVaultFS(d.FS)

     	return CountDeps{
     		ListMD:   fsys.ListMD,
     		ReadFile: fsys.ReadFile,
     		Scan: func(vault string) ([]vaultgraph.Note, error) {
     			return vaultgraph.ScanVault(fsys, vault)
     		},
     	}
     }
     ```
   - show.go:67-77, current:
     ```go
     // newOsShowDeps wires RunShow to the real filesystem vault scanner and reader.
     func newOsShowDeps() ShowDeps {
     	fsys := &osVaultFS{}

     	return ShowDeps{
     		Scan: func(vault string) ([]vaultgraph.Note, error) {
     			return vaultgraph.ScanVault(fsys, vault)
     		},
     		Read: fsys.ReadFile,
     	}
     }
     ```
     replacement:
     ```go
     // newShowDeps wires RunShow from the injected CLI capabilities — pure
     // composition over EdgeFS (#700).
     func newShowDeps(d Deps) ShowDeps {
     	fsys := newVaultFS(d.FS)

     	return ShowDeps{
     		Scan: func(vault string) ([]vaultgraph.Note, error) {
     			return vaultgraph.ScanVault(fsys, vault)
     		},
     		Read: fsys.ReadFile,
     	}
     }
     ```
   - check.go:221-232, current:
     ```go
     // newOsCheckDeps wires RunCheck to the real filesystem vault scanner.
     func newOsCheckDeps() CheckDeps {
     	fsys := &osVaultFS{}

     	return CheckDeps{
     		Scan: func(vault string) ([]vaultgraph.Note, error) {
     			return vaultgraph.ScanVault(fsys, vault)
     		},
     		ReadNote:    fsys.ReadFile,
     		ReadSidecar: fsys.ReadFile,
     	}
     }
     ```
     replacement:
     ```go
     // newCheckDeps wires RunCheck from the injected CLI capabilities — pure
     // composition over EdgeFS (#700).
     func newCheckDeps(d Deps) CheckDeps {
     	fsys := newVaultFS(d.FS)

     	return CheckDeps{
     		Scan: func(vault string) ([]vaultgraph.Note, error) {
     			return vaultgraph.ScanVault(fsys, vault)
     		},
     		ReadNote:    fsys.ReadFile,
     		ReadSidecar: fsys.ReadFile,
     	}
     }
     ```

4. [ ] Update targets.go call sites (lines 177, 182, 190; requires targets-cluster `d Deps` in scope of `ingestQueryTargets` — adapt the variable name to what that task landed):
   - `errHandler(RunCount(a, newOsCountDeps(), stdout))` → `errHandler(RunCount(a, newCountDeps(d), stdout))`
   - `errHandler(RunShow(withLog(ctx), a, newOsShowDeps(), stdout))` → `errHandler(RunShow(withLog(ctx), a, newShowDeps(d), stdout))`
   - `errHandler(RunCheck(withLog(ctx), a, newOsCheckDeps(), stdout))` → `errHandler(RunCheck(withLog(ctx), a, newCheckDeps(d), stdout))`

5. [ ] Update export_test.go: delete the three var-block entries (lines 77, 78, 80):
   ```go
   ExportNewOsCheckDeps                   = newOsCheckDeps
   ExportNewOsCountDeps                   = newOsCountDeps
   ExportNewOsShowDeps                    = newOsShowDeps
   ```
   and add in the exported-functions section (alphabetical):
   ```go
   // ExportNewCheckDeps builds production CheckDeps over the given EdgeFS.
   func ExportNewCheckDeps(fsys EdgeFS) CheckDeps { return newCheckDeps(Deps{FS: fsys}) }

   // ExportNewCountDeps builds production CountDeps over the given EdgeFS.
   func ExportNewCountDeps(fsys EdgeFS) CountDeps { return newCountDeps(Deps{FS: fsys}) }

   // ExportNewShowDeps builds production ShowDeps over the given EdgeFS.
   func ExportNewShowDeps(fsys EdgeFS) ShowDeps { return newShowDeps(Deps{FS: fsys}) }

   // ExportNewVaultFS returns the pure EdgeFS→vaultgraph.VaultFS adapter.
   func ExportNewVaultFS(fsys EdgeFS) interface {
   	ListMD(dir string) ([]string, error)
   	ReadFile(path string) ([]byte, error)
   } {
   	return newVaultFS(fsys)
   }
   ```
   KEEP `ExportNewOsVaultFS` (vocab cluster tests still use it; deleted in Q3).

6. [ ] Update test call sites:
   - check_test.go lines 65, 86, 205, 235: `cli.ExportNewOsCheckDeps()` → `cli.ExportNewCheckDeps(osTestEdgeFS{})`
   - show_test.go line 89: `cli.ExportNewOsShowDeps()` → `cli.ExportNewShowDeps(osTestEdgeFS{})`
   - count_test.go line 464: `cli.ExportNewOsCountDeps()` → `cli.ExportNewCountDeps(osTestEdgeFS{})`; update the comment at 454-455 from "exercises newOsCountDeps" to "exercises newCountDeps over a real-FS EdgeFS" and at 560 from "newOsCountDeps + resolveVault" to "newCountDeps + resolveVault"

7. [ ] Run `targ test` — expected: all green (new vaultFS tests pass; count/show/check suites pass unchanged). Run `targ check-full` — expected: no new findings.
8. [ ] Commit: `refactor(cli): #700 query-family vault reads via EdgeFS-backed vaultFS`

---

### Task T6 (Q2): query + query-chunks compose from Deps (EdgeFS lister, injected clock)

Sequencing: AFTER Q1 and AFTER the amend/prune/show-chunk cluster tasks land `newAmendDeps(d)`, `newPruneDeps(d)`, `newShowChunkDeps(d)` (their call sites must have `d Deps` in scope for the atomic signature flip).

**Files:**
- Modify: `internal/cli/query_chunks.go`, `internal/cli/query.go`, `internal/cli/export_test.go`, `internal/cli/query_chunks_test.go`, `internal/cli/ingest_integration_test.go` (2 lines), `internal/cli/targets.go` (2 lines), `internal/cli/amend.go` (1 line), `internal/cli/prune.go` (1 line), `internal/cli/show_chunk.go` (1 line), `internal/cli/deps.go` (only if `logWarningTo` not yet landed by the learn cluster)
- Delete: none

**Interfaces:**
- Consumes: `cli.Deps{FS, Embed, Stderr, Now}`; `logWarningTo(w io.Writer) func(format string, args ...any)`
- Produces: `func listJSONLIndexes(fsys EdgeFS) func(dir string) ([]string, error)` (CANONICAL final shape — amend/prune/show-chunk clusters consume it as `listJSONLIndexes(d.FS)`); `func newChunkQueryDeps(d Deps) ChunkQueryDeps`; `func newQueryDeps(d Deps) QueryDeps`; test shim `ExportNewChunkQueryDeps(fsys EdgeFS, emb embed.Embedder) ChunkQueryDeps`

**Steps:**

1. [ ] RED — add to `internal/cli/query_chunks_test.go` (package cli_test; this file uses `gomega.NewWithT`, non-dot import — add `"os"`, `"path/filepath"` to its imports):
   ```go
   func TestChunkQueryDeps_ListIndexes_WrappedNotExistIsEmptyIndex(t *testing.T) {
   	t.Parallel()
   	g := gomega.NewWithT(t)

   	deps := cli.ExportNewChunkQueryDeps(wrappedNotExistEdgeFS{}, nil)
   	paths, err := deps.ListIndexes("/any/chunks/dir")
   	g.Expect(err).NotTo(gomega.HaveOccurred())

   	if err != nil {
   		return
   	}

   	g.Expect(paths).To(gomega.BeEmpty())
   }

   func TestChunkQueryDeps_ListIndexes_ListsOnlyJSONLFiles(t *testing.T) {
   	t.Parallel()
   	g := gomega.NewWithT(t)

   	dir := t.TempDir()
   	g.Expect(os.MkdirAll(filepath.Join(dir, "sub.jsonl"), 0o750)).To(gomega.Succeed())
   	g.Expect(os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte("{}"), 0o600)).To(gomega.Succeed())
   	g.Expect(os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0o600)).To(gomega.Succeed())

   	deps := cli.ExportNewChunkQueryDeps(osTestEdgeFS{}, nil)
   	paths, err := deps.ListIndexes(dir)
   	g.Expect(err).NotTo(gomega.HaveOccurred())

   	if err != nil {
   		return
   	}

   	g.Expect(paths).To(gomega.ConsistOf(filepath.Join(dir, "a.jsonl")))
   }
   ```
   Run `targ test` — expected: compile failure (`undefined: cli.ExportNewChunkQueryDeps` — the old shim is `ExportNewOsChunkQueryDeps`).

2. [ ] GREEN — rewrite `internal/cli/query_chunks.go` I/O seams. Imports: delete `"os"`, add `"io/fs"`. Replace lines 136-157 (current `listJSONLIndexes` shown in inventory) with:
   ```go
   // listJSONLIndexes returns a lister over fsys for the .jsonl files directly
   // under a dir. A missing dir is an empty index (cold start), not an error —
   // matched via errors.Is so EdgeFS implementations may wrap the not-exist
   // error (os.IsNotExist would not unwrap a %w chain).
   func listJSONLIndexes(fsys EdgeFS) func(dir string) ([]string, error) {
   	return func(dir string) ([]string, error) {
   		entries, err := fsys.ReadDir(dir)
   		if err != nil {
   			if errors.Is(err, fs.ErrNotExist) {
   				return nil, nil
   			}

   			return nil, fmt.Errorf("listing chunk indexes: %w", err)
   		}

   		var paths []string

   		for _, entry := range entries {
   			if !entry.IsDir() && filepath.Ext(entry.Name()) == jsonlExt {
   				paths = append(paths, filepath.Join(dir, entry.Name()))
   			}
   		}

   		return paths, nil
   	}
   }
   ```
   and replace lines 185-195 (current `newOsChunkQueryDeps` using `fs := &osEmbedFS{}` + `sharedEmbedder`) with:
   ```go
   // newChunkQueryDeps wires `engram query-chunks` from the injected CLI
   // capabilities — pure composition (#700).
   func newChunkQueryDeps(d Deps) ChunkQueryDeps {
   	return ChunkQueryDeps{
   		ListIndexes: listJSONLIndexes(d.FS),
   		ReadFile:    d.FS.ReadFile,
   		Embedder:    d.Embed,
   	}
   }
   ```

3. [ ] Replace query.go:1286-1298, current:
   ```go
   // newOsQueryDeps wires the production scan + read for the query command.
   func newOsQueryDeps() QueryDeps {
   	embedDeps := newOsEmbedDeps()

   	return QueryDeps{
   		Scan:             embedDeps.Scan,
   		Read:             embedDeps.Read,
   		Embedder:         embedDeps.Embedder,
   		LogWarning:       logWarningToStderrf,
   		ListChunkIndexes: listJSONLIndexes,
   		Now:              time.Now,
   	}
   }
   ```
   with:
   ```go
   // newQueryDeps wires the query command from the injected CLI capabilities —
   // pure composition, every I/O flows through d (#700).
   func newQueryDeps(d Deps) QueryDeps {
   	vfs := newVaultFS(d.FS)

   	return QueryDeps{
   		Scan: func(vault string) ([]vaultgraph.Note, error) {
   			return vaultgraph.ScanVault(vfs, vault)
   		},
   		Read:             d.FS.ReadFile,
   		Embedder:         d.Embed,
   		LogWarning:       logWarningTo(d.Stderr),
   		ListChunkIndexes: listJSONLIndexes(d.FS),
   		Now:              d.Now,
   	}
   }
   ```
   (`"time"` import stays — `time.Time`/`time.Duration` types remain throughout query.go; `time.Now` is now gone. Behavioral note: `Read` error wrap text changes from osEmbedFS's `"read: %w"` to the cmd adapter's wrap — non-behavioral, both consumers at query.go:1068 and :1434 only branch on error presence.) If the learn cluster has not yet landed `logWarningTo`, add to `internal/cli/deps.go`:
   ```go
   // logWarningTo returns a LogWarning hook writing "warning: ..." lines to w —
   // the pure replacement for the legacy logWarningToStderrf (#700).
   func logWarningTo(w io.Writer) func(format string, args ...any) {
   	return func(format string, args ...any) {
   		_, _ = fmt.Fprintf(w, "warning: "+format+"\n", args...)
   	}
   }
   ```

4. [ ] Atomic call-site flip for `listJSONLIndexes` (same commit; these constructors are d-scoped by their own cluster tasks at this point — apply the same one-line edit to whatever the constructor is named at rebase time):
   - amend.go:365 `ListIndexes: listJSONLIndexes,` → `ListIndexes: listJSONLIndexes(d.FS),`
   - prune.go:115 `ListIndexes: listJSONLIndexes,` → `ListIndexes: listJSONLIndexes(d.FS),`
   - show_chunk.go:72 `ListIndexes: listJSONLIndexes,` → `ListIndexes: listJSONLIndexes(d.FS),`

5. [ ] Update export_test.go lines 514-521, current:
   ```go
   // ExportNewOsChunkQueryDeps returns production ChunkQueryDeps with an
   // injected embedder, mirroring ExportNewOsIngestDeps.
   func ExportNewOsChunkQueryDeps(emb embed.Embedder) ChunkQueryDeps {
   	deps := newOsChunkQueryDeps()
   	deps.Embedder = emb

   	return deps
   }
   ```
   replacement:
   ```go
   // ExportNewChunkQueryDeps returns production ChunkQueryDeps over the given
   // EdgeFS with an injected embedder.
   func ExportNewChunkQueryDeps(fsys EdgeFS, emb embed.Embedder) ChunkQueryDeps {
   	deps := newChunkQueryDeps(Deps{FS: fsys})
   	deps.Embedder = emb

   	return deps
   }
   ```
   Update ingest_integration_test.go lines 100 and 204: `cli.ExportNewOsChunkQueryDeps(fakeIngestEmbedder{})` → `cli.ExportNewChunkQueryDeps(osTestEdgeFS{}, fakeIngestEmbedder{})`.

6. [ ] Update targets.go call sites: line 155 `newOsQueryDeps()` → `newQueryDeps(d)`; line 169 `newOsChunkQueryDeps()` → `newChunkQueryDeps(d)`.
7. [ ] Verify no os import remains in the migrated files: `grep -n '"os"' internal/cli/query_chunks.go internal/cli/query.go` — expected: no output. `grep -n 'time\.Now' internal/cli/query.go` — expected: no output.
8. [ ] Run `targ test` — expected: all green (step-1 tests now pass; ingest integration + query suites unchanged). Run `targ check-full` — expected: clean.
9. [ ] Commit: `refactor(cli): #700 query/query-chunks compose from Deps (EdgeFS lister, injected clock)`

---

### Task T7 (Q3): Purge legacy `osVaultFS` (grep-gated, runs LAST)

Sequencing: after the amend/learn/qa/resituate/embed/vocab clusters migrate to `newVaultFS(d.FS)` and vocab tests drop `ExportNewOsVaultFS`. Belongs immediately before the depguard/forbidigo enforcement task.

**Files:**
- Modify: `internal/cli/vault_fs.go` (delete `osVaultFS` + methods + `"os"` import), `internal/cli/export_test.go` (delete `ExportNewOsVaultFS`, lines currently 572-578)
- Delete: none

**Interfaces:**
- Consumes: nothing new. Produces: a pure vault_fs.go (zero I/O-capable imports).

**Steps:**
1. [ ] Gate: `grep -rn "osVaultFS" internal/cli --include='*.go'` — expected: hits ONLY in vault_fs.go (definition) and export_test.go (shim). Any other hit → STOP; that cluster has not migrated; do not proceed.
2. [ ] Delete from vault_fs.go: the `osVaultFS` type, its `ListMD`/`ReadFile` methods, and the `"os"` import (all other imports stay: errors, fmt, io/fs, path/filepath, strings).
3. [ ] Delete from export_test.go:
   ```go
   // ExportNewOsVaultFS returns the production osVaultFS adapter for testing.
   func ExportNewOsVaultFS() interface {
   	ListMD(dir string) ([]string, error)
   	ReadFile(path string) ([]byte, error)
   } {
   	return &osVaultFS{}
   }
   ```
4. [ ] Verify purity: `grep -n '"os"' internal/cli/vault_fs.go` — expected: no output.
5. [ ] Run `targ test` then `targ check-full` — expected: all green, no findings.
6. [ ] Commit: `refactor(cli): #700 delete legacy osVaultFS adapter`

---

Files read (worktree `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity`): `internal/cli/{query.go,query_chunks.go,query_nominations.go,count.go,show.go,check.go,vault_fs.go,targets.go,main.go,embed.go(80-269),learn.go(320-378),export_test.go,vault_fs_test.go,count_test.go(440-599),check_test.go(40-95),query_chunks_test.go(1-25),targets_test.go(420-455)}`, `internal/vaultgraph/scanner.go(14-30)`, `internal/embed/embedder.go(50-70)`, `cmd/engram/main.go`.

### Task T8 (I1): Migrate `engram ingest` wiring to cli.Deps composition

**Files:**
- Modify: `internal/cli/ingest.go` (replace `newOsIngestDeps`/`defaultSessionDir`/`walkSourcesExcluding` with pure compositions)
- Modify: `internal/cli/cli.go` (delete `osFileReader` only)
- Modify: `internal/cli/targets.go` (ingest call site — depends on foundation's `deps Deps` threading)
- Modify: `internal/cli/export_test.go` (swap `ExportNewOsIngestDeps` → `ExportNewIngestDeps`; delete `ExportFlockPath`, `ExportNewOsFileReader`)
- Create: `internal/cli/ingest_family_deps_test.go` (pure composition tests + cli_test OS harness)
- Modify: `internal/cli/ingest_auto_test.go`, `internal/cli/ingest_integration_test.go`, `internal/cli/ingest_test.go`, `internal/cli/adapters_test.go`

**Interfaces:**
- Consumes (foundation): `type Deps struct { …; Getenv func(string) string; Now func() time.Time; Getwd func() (string, error); UserHomeDir func() (string, error); FS EdgeFS; Lock FileLocker; Embed embed.Embedder; … }`; `EdgeFS` with `ReadFile/WriteFile/WriteFileAtomic/MkdirAll/MkdirTemp/Stat/ReadDir/Remove/RemoveAll/Rename/WalkDir`; `FileLocker.Lock(path string) (unlock func() error, err error)`; `transcript.NewJSONLReader(reader sessionctx.FileReader) *JSONLReader` where `sessionctx.FileReader` is `Read(path string) ([]byte, error)` (internal/context/context.go:7).
- Produces: `func newIngestDeps(d Deps) IngestDeps`; `func manifestLockFrom(d Deps) func(chunksDir string) (func(), error)`; `func sessionDirFrom(getenv func(string) string, userHomeDir func() (string, error)) func(string) string`; `func sweepListerFrom(walkDir func(root string, fn fs.WalkDirFunc) error) func(SweepRoot) ([]string, error)`; `type fsFileReader struct{ fs EdgeFS }` with `Read(path string) ([]byte, error)`; test export `func ExportNewIngestDeps(d Deps, emb embed.Embedder) IngestDeps`.

**Steps:**

- [ ] 1. RED — create `internal/cli/ingest_family_deps_test.go` (package `cli_test`). It references `cli.ExportNewIngestDeps` and `cli.Deps`, which do not exist yet, so the test package fails to compile. Full content:

```go
package cli_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestIngestDepsLockMkdirsChunksDirBeforeLocking(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var calls []string

	d := cli.Deps{
		FS: fakeEdgeFS{
			mkdirAll: func(path string, perm fs.FileMode) error {
				calls = append(calls, fmt.Sprintf("mkdirall %s %o", path, perm))

				return nil
			},
		},
		Lock: fakeLocker{lock: func(path string) (func() error, error) {
			calls = append(calls, "lock "+path)

			return func() error {
				calls = append(calls, "unlock")

				return nil
			}, nil
		}},
	}

	release, err := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{}).Lock("/chunks")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil || release == nil {
		return
	}

	release()

	g.Expect(calls).To(gomega.Equal([]string{
		"mkdirall /chunks 700",
		"lock " + filepath.Join("/chunks", cli.ExportManifestLockFile()),
		"unlock",
	}), "MkdirAll must precede flock (fresh-dir regression), release must unlock")
}

func TestIngestDepsLockMkdirErrorPropagates(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	d := cli.Deps{
		FS: fakeEdgeFS{
			mkdirAll: func(string, fs.FileMode) error { return errBoom },
		},
		Lock: fakeLocker{lock: func(string) (func() error, error) {
			t.Fatal("Lock must not be called when MkdirAll fails")

			return nil, nil
		}},
	}

	_, err := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{}).Lock("/chunks")
	g.Expect(err).To(gomega.MatchError(errBoom))
}

func TestIngestDepsSessionDirDefaultsToClaudeProjects(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	d := testDeps()
	d.Getenv = func(string) string { return "" }
	d.UserHomeDir = func() (string, error) { return "/home/u", nil }

	deps := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	g.Expect(deps.SessionDir("/anywhere")).
		To(gomega.Equal(filepath.Join("/home/u", ".claude", "projects")))

	d.UserHomeDir = func() (string, error) { return "", errBoom }
	deps = cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	g.Expect(deps.SessionDir("/anywhere")).To(gomega.BeEmpty(),
		"unresolvable home yields empty session dir, not a panic")
}

func TestIngestDepsSessionDirHonorsTranscriptDirEnv(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	d := testDeps()
	d.Getenv = func(key string) string {
		if key == "ENGRAAM_NEVER" { // guard: only the real key may match below
			return ""
		}

		if key == "ENGRAM_TRANSCRIPT_DIR" {
			return "/custom/sessions"
		}

		return ""
	}

	deps := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	g.Expect(deps.SessionDir("/anywhere")).To(gomega.Equal("/custom/sessions"))
}

func TestIngestDepsStatMapsFileInfoToSourceStat(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	mtime := time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC)

	const wantSize = int64(42)

	d := testDeps()
	d.FS = fakeEdgeFS{stat: func(string) (fs.FileInfo, error) {
		return fakeFileInfo{mtime: mtime, size: wantSize}, nil
	}}

	deps := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	stat, err := deps.Stat("/src.md")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stat).To(gomega.Equal(cli.SourceStat{
		MtimeUnixNano: mtime.UnixNano(),
		Size:          wantSize,
	}))

	d.FS = fakeEdgeFS{stat: func(string) (fs.FileInfo, error) { return nil, errBoom }}
	deps = cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	_, err = deps.Stat("/src.md")
	g.Expect(err).To(gomega.MatchError(errBoom))
}

func TestIngestDepsWriteFileMkdirsParentThenWritesAtomically(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var calls []string

	d := testDeps()
	d.FS = fakeEdgeFS{
		mkdirAll: func(path string, perm fs.FileMode) error {
			calls = append(calls, fmt.Sprintf("mkdirall %s %o", path, perm))

			return nil
		},
		writeFileAtomic: func(path string, _ []byte, perm fs.FileMode) error {
			calls = append(calls, fmt.Sprintf("writeatomic %s %o", path, perm))

			return nil
		},
	}

	deps := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	g.Expect(deps.WriteFile("/chunks/idx.jsonl", []byte("x"))).To(gomega.Succeed())
	g.Expect(calls).To(gomega.Equal([]string{
		"mkdirall /chunks 700",
		"writeatomic /chunks/idx.jsonl 600",
	}), "parent dir created before atomic 0600 write")
}

// --- cli_test OS harness: real-FS EdgeFS + real flock FileLocker ---
// Test files are exempt from the internal I/O purity enforcement; production
// adapters live in cmd/engram. These doubles back the ingest-family
// integration tests and the ADR-0013 concurrency regression.

type fakeEdgeFS struct {
	readFile        func(string) ([]byte, error)
	writeFile       func(string, []byte, fs.FileMode) error
	writeFileAtomic func(string, []byte, fs.FileMode) error
	mkdirAll        func(string, fs.FileMode) error
	mkdirTemp       func(string, string) (string, error)
	stat            func(string) (fs.FileInfo, error)
	readDir         func(string) ([]fs.DirEntry, error)
	remove          func(string) error
	removeAll       func(string) error
	rename          func(string, string) error
	walkDir         func(string, fs.WalkDirFunc) error
}

func (f fakeEdgeFS) ReadFile(path string) ([]byte, error) { return f.readFile(path) }

func (f fakeEdgeFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return f.writeFile(path, data, perm)
}

func (f fakeEdgeFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	return f.writeFileAtomic(path, data, perm)
}

func (f fakeEdgeFS) MkdirAll(path string, perm fs.FileMode) error { return f.mkdirAll(path, perm) }

func (f fakeEdgeFS) MkdirTemp(dir, pattern string) (string, error) {
	return f.mkdirTemp(dir, pattern)
}

func (f fakeEdgeFS) Stat(path string) (fs.FileInfo, error)     { return f.stat(path) }
func (f fakeEdgeFS) ReadDir(path string) ([]fs.DirEntry, error) { return f.readDir(path) }
func (f fakeEdgeFS) Remove(path string) error                   { return f.remove(path) }
func (f fakeEdgeFS) RemoveAll(path string) error                { return f.removeAll(path) }

func (f fakeEdgeFS) Rename(oldPath, newPath string) error { return f.rename(oldPath, newPath) }

func (f fakeEdgeFS) WalkDir(root string, fn fs.WalkDirFunc) error { return f.walkDir(root, fn) }

// fakeFileInfo satisfies fs.FileInfo for Stat-mapping tests.
type fakeFileInfo struct {
	mtime time.Time
	size  int64
	dir   bool
}

func (f fakeFileInfo) Name() string       { return "fake" }
func (f fakeFileInfo) Size() int64        { return f.size }
func (f fakeFileInfo) Mode() fs.FileMode  { return 0o600 }
func (f fakeFileInfo) ModTime() time.Time { return f.mtime }
func (f fakeFileInfo) IsDir() bool        { return f.dir }
func (f fakeFileInfo) Sys() any           { return nil }

// fakeLocker satisfies cli.FileLocker with an injected func.
type fakeLocker struct {
	lock func(string) (func() error, error)
}

func (l fakeLocker) Lock(path string) (func() error, error) { return l.lock(path) }

// osTestFS implements cli.EdgeFS over the real filesystem.
type osTestFS struct{}

func (osTestFS) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

func (osTestFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (osTestFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	return testAtomicWrite(path, data, perm)
}

func (osTestFS) MkdirAll(path string, perm fs.FileMode) error { return os.MkdirAll(path, perm) }

func (osTestFS) MkdirTemp(dir, pattern string) (string, error) {
	return os.MkdirTemp(dir, pattern)
}

func (osTestFS) Stat(path string) (fs.FileInfo, error)      { return os.Stat(path) }
func (osTestFS) ReadDir(path string) ([]fs.DirEntry, error) { return os.ReadDir(path) }
func (osTestFS) Remove(path string) error                   { return os.Remove(path) }
func (osTestFS) RemoveAll(path string) error                { return os.RemoveAll(path) }

func (osTestFS) Rename(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

func (osTestFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

// testFlocker implements cli.FileLocker with a real syscall flock, preserving
// the ADR-0013 concurrency regression's real-lock semantics inside cli_test.
type testFlocker struct{}

func (testFlocker) Lock(path string) (func() error, error) {
	const lockPerm = 0o600

	lockFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, lockPerm)
	if err != nil {
		return nil, fmt.Errorf("open lock: %w", err)
	}

	fileDescriptor := int(lockFile.Fd())

	flockErr := syscall.Flock(fileDescriptor, syscall.LOCK_EX)
	if flockErr != nil {
		_ = lockFile.Close()

		return nil, fmt.Errorf("flock: %w", flockErr)
	}

	return func() error {
		_ = syscall.Flock(fileDescriptor, syscall.LOCK_UN)

		if closeErr := lockFile.Close(); closeErr != nil {
			return fmt.Errorf("close lock: %w", closeErr)
		}

		return nil
	}, nil
}

// testAtomicWrite is temp+rename, mirroring production WriteFileAtomic.
func testAtomicWrite(path string, data []byte, perm fs.FileMode) error {
	tmp := path + ".tmp-test"

	err := os.WriteFile(tmp, data, perm)
	if err != nil {
		return fmt.Errorf("atomic write temp: %w", err)
	}

	err = os.Rename(tmp, path)
	if err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf("atomic write rename: %w", err)
	}

	return nil
}

// testDeps returns a real-OS cli.Deps for integration tests.
func testDeps() cli.Deps {
	return cli.Deps{
		FS:          osTestFS{},
		Lock:        testFlocker{},
		Getenv:      os.Getenv,
		Now:         time.Now,
		Getwd:       os.Getwd,
		UserHomeDir: os.UserHomeDir,
	}
}

var _ = errors.Is // keep errors imported if assertions above change shape
```

(Drop the trailing `var _ = errors.Is` line and the `errors` import if unused after final assembly — goimports will flag it.) Run: `targ test`. Expected: FAIL — `undefined: cli.ExportNewIngestDeps`, `undefined: cli.Deps` fields as applicable. This is the compile-RED for the whole task.

- [ ] 2. GREEN — rewrite `internal/cli/ingest.go` wiring. Replace the import block line `"os"` → (removed) and add `"io/fs"`. Exact current import block (lines 3-18) becomes:

```go
import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/transcript"
)
```

Update the two stale doc comments in `IngestDeps` (current lines 39-41 and 43-46): replace `// release func. Wired to flockPath(chunksDir/.manifest.lock) in newOsIngestDeps.` with `// release func. Wired to manifestLockFrom (MkdirAll + FileLocker flock) in newIngestDeps.` and replace `// callers guard with "if deps.Now != nil" before calling. Wire time.Now in` / `// newOsIngestDeps.` with `// callers guard with "if deps.Now != nil" before calling. Wired to deps.Now in` / `// newIngestDeps.`

Add named perms to the unexported constants block (current lines 129-138), after `manifestName`:

```go
	chunksDirPerm = 0o700
	indexFilePerm = 0o600
```

DELETE `defaultSessionDir` (current lines 244-258) entirely. DELETE `newOsIngestDeps` (current lines 483-523) entirely. DELETE `walkSourcesExcluding` (current lines 652-689) entirely. ADD these replacements (alphabetical placement per existing file order — `fsFileReader` near top-of-types is fine; the repo's reorder-decls lint will confirm):

```go
// fsFileReader adapts EdgeFS.ReadFile to the context.FileReader interface
// consumed by transcript.NewJSONLReader (production transcript stripping).
type fsFileReader struct {
	fs EdgeFS
}

// Read reads path via the injected edge filesystem.
func (r fsFileReader) Read(path string) ([]byte, error) {
	return r.fs.ReadFile(path)
}

// manifestLockFrom composes the ADR-0013 manifest lock from the CLI-edge
// Deps: ensure chunksDir exists first (regression: prune's lock on a fresh
// dir errored without MkdirAll), then take an exclusive advisory lock on
// chunksDir/.manifest.lock via the injected FileLocker. The unlock error is
// discarded to preserve the historical release-func() contract at every Run*
// entry point. Shared by newIngestDeps and newPruneDeps so the MkdirAll+lock
// pair lives in exactly one place.
func manifestLockFrom(d Deps) func(chunksDir string) (func(), error) {
	return func(chunksDir string) (func(), error) {
		err := d.FS.MkdirAll(chunksDir, chunksDirPerm)
		if err != nil {
			return nil, fmt.Errorf("creating chunks dir for lock: %w", err)
		}

		unlock, err := d.Lock.Lock(filepath.Join(chunksDir, manifestLockFile))
		if err != nil {
			return nil, fmt.Errorf("locking %s: %w", manifestLockFile, err)
		}

		return func() { _ = unlock() }, nil
	}
}

// newIngestDeps composes production IngestDeps from the CLI-edge Deps: every
// filesystem, clock, and environment capability flows through d — internal/
// stays free of direct os.* calls. WriteFile creates the chunks directory on
// demand so first ingest into a fresh dir succeeds.
func newIngestDeps(d Deps) IngestDeps {
	reader := transcript.NewJSONLReader(fsFileReader{fs: d.FS})

	return IngestDeps{
		Lock:     manifestLockFrom(d),
		ReadFile: d.FS.ReadFile,
		WriteFile: func(path string, data []byte) error {
			err := d.FS.MkdirAll(filepath.Dir(path), chunksDirPerm)
			if err != nil {
				return fmt.Errorf("ingest: creating chunks dir: %w", err)
			}

			err = d.FS.WriteFileAtomic(path, data, indexFilePerm)
			if err != nil {
				return fmt.Errorf("ingest: writing %s: %w", path, err)
			}

			return nil
		},
		Stat: func(path string) (SourceStat, error) {
			info, err := d.FS.Stat(path)
			if err != nil {
				return SourceStat{}, fmt.Errorf("ingest: stat %s: %w", path, err)
			}

			return SourceStat{MtimeUnixNano: info.ModTime().UnixNano(), Size: info.Size()}, nil
		},
		ListSources:    sweepListerFrom(d.FS.WalkDir),
		ReadTranscript: reader.ReadFrom,
		Embedder:       d.Embed,
		Now:            d.Now,
		IsDir: func(path string) bool {
			info, err := d.FS.Stat(path)

			return err == nil && info.IsDir()
		},
		Getwd:      d.Getwd,
		SessionDir: sessionDirFrom(d.Getenv, d.UserHomeDir),
	}
}

// sessionDirFrom resolves the root of ALL recorded session transcripts:
// ENGRAM_TRANSCRIPT_DIR when set (headless/eval cells get only their own
// sessions), else <home>/.claude/projects — every project, every conversation.
func sessionDirFrom(
	getenv func(string) string,
	userHomeDir func() (string, error),
) func(string) string {
	return func(_ string) string {
		if dir := getenv("ENGRAM_TRANSCRIPT_DIR"); dir != "" {
			return dir
		}

		home, err := userHomeDir()
		if err != nil {
			return ""
		}

		return filepath.Join(home, ".claude", "projects")
	}
}

// sweepListerFrom returns a SweepRoot lister walking via the injected walkDir
// (EdgeFS.WalkDir in production), pruning excluded directory names
// (build/dependency trees) and, when SkipHidden, every dot-directory.
func sweepListerFrom(
	walkDir func(root string, fn fs.WalkDirFunc) error,
) func(SweepRoot) ([]string, error) {
	return func(root SweepRoot) ([]string, error) {
		excluded := make(map[string]struct{}, len(root.ExcludeDirs))
		for _, name := range root.ExcludeDirs {
			excluded[name] = struct{}{}
		}

		var paths []string

		err := walkDir(root.Path, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if entry.IsDir() {
				if path == root.Path {
					return nil
				}

				hidden := root.SkipHidden && strings.HasPrefix(entry.Name(), ".")
				if hidden || shouldSkipDir(entry.Name(), excluded, root.ExcludePrefixes) {
					return filepath.SkipDir
				}

				return nil
			}

			paths = append(paths, path)

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("ingest: walking %s: %w", root.Path, err)
		}

		return paths, nil
	}
}
```

- [ ] 3. Delete `osFileReader` from `internal/cli/cli.go` — exact current lines 25-31:

```go
// I/O adapters for context package DI interfaces.

type osFileReader struct{}

func (r *osFileReader) Read(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec,wrapcheck // thin I/O adapter
}
```

(Its only production consumer was ingest.go:488, replaced in step 2. Do NOT touch `flockPath` or `osManifestLock` in this task — `osManifestLock` still backs `newOsPruneDeps` until Task I2; `flockPath` still backs `osLearnFS.Lock`.)

- [ ] 4. Update `internal/cli/targets.go` ingest call site. Current lines 157-160:

```go
		targ.Targ(func(ctx context.Context, a IngestArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, os.Getenv)
			errHandler(RunIngest(withLog(ctx), a, newOsIngestDeps(), stdout))
		}).Name("ingest").Description("Chunk+embed transcripts/markdown into a chunk index (zero-LLM)"),
```

Replacement (assumes the foundation task threads `deps Deps` into `ingestQueryTargets`; adopt its exact parameter name):

```go
		targ.Targ(func(ctx context.Context, a IngestArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunIngest(withLog(ctx), a, newIngestDeps(deps), stdout))
		}).Name("ingest").Description("Chunk+embed transcripts/markdown into a chunk index (zero-LLM)"),
```

- [ ] 5. Update `internal/cli/export_test.go`. Replace (current lines 543-551):

```go
// ExportNewOsIngestDeps returns production IngestDeps with an injected
// embedder so coverage tests can drive the wiring without unpacking the
// lazy bundled embedder.
func ExportNewOsIngestDeps(emb embed.Embedder) IngestDeps {
	deps := newOsIngestDeps()
	deps.Embedder = emb

	return deps
}
```

with:

```go
// ExportNewIngestDeps returns production IngestDeps composed from d with an
// injected embedder so coverage tests can drive the wiring without unpacking
// the lazy bundled embedder.
func ExportNewIngestDeps(d Deps, emb embed.Embedder) IngestDeps {
	deps := newIngestDeps(d)
	deps.Embedder = emb

	return deps
}
```

Delete `ExportFlockPath` (current lines 342-347) and `ExportNewOsFileReader` (current lines 536-541).

- [ ] 6. Adapt `internal/cli/ingest_test.go`. In `TestManifest_ConcurrentWritersDoNotLoseEntries`, both Lock closures (current lines 375-377 and 402-404):

```go
			Lock: func(dir string) (func(), error) {
				return cli.ExportFlockPath(dir + "/" + cli.ExportManifestLockFile())
			},
```

become (both occurrences):

```go
			Lock: func(dir string) (func(), error) {
				unlock, lockErr := testFlocker{}.Lock(dir + "/" + cli.ExportManifestLockFile())
				if lockErr != nil {
					return nil, lockErr
				}

				return func() { _ = unlock() }, nil
			},
```

And `realFS.write` (current lines 898-900):

```go
func (r *realFS) write(_, path string, data []byte) error {
	return cli.ExportAtomicWriteFile(path, data, 0o600)
}
```

becomes:

```go
func (r *realFS) write(_, path string, data []byte) error {
	return testAtomicWrite(path, data, 0o600)
}
```

- [ ] 7. Adapt `internal/cli/ingest_auto_test.go` (three call sites). `TestDefaultSessionDirHonorsTranscriptDirEnv` (current lines 48-57, uses `t.Setenv`) is DELETED — superseded by the pure `TestIngestDepsSessionDirHonorsTranscriptDirEnv` from step 1 (now parallel-safe, no env mutation). In `TestDefaultSessionDirIsAllProjects` (line 63) and `TestOsListSourcesSkipsExcludedDirs` (line 95), replace `deps := cli.ExportNewOsIngestDeps(fakeIngestEmbedder{})` with `deps := cli.ExportNewIngestDeps(testDeps(), fakeIngestEmbedder{})`. In `internal/cli/ingest_integration_test.go` line 188 replace `ingestDeps := cli.ExportNewOsIngestDeps(fakeIngestEmbedder{})` with `ingestDeps := cli.ExportNewIngestDeps(testDeps(), fakeIngestEmbedder{})`. Remove the now-unused `os` import from ingest_auto_test.go only if no other use remains (TestOsListSourcesSkipsExcludedDirs still uses os.MkdirAll/os.WriteFile — keep it).

- [ ] 8. Adapt `internal/cli/adapters_test.go`: delete `TestOsFileReader_Read` (lines 14-30) and `TestOsFileReader_ReadError` (lines 32-39). The transcript-read path is now covered by the pure `fsFileReader` composition (exercised end-to-end by `TestOsIngestThenChunkQuery` through `testDeps()`), and raw `os.ReadFile` adapter coverage belongs to cmd/engram's `osFS` integration tests (adapters cluster). Keep the `osLearnFS` tests untouched.

- [ ] 9. Verify: `targ test` — expected PASS (all new pure tests green; `TestOsIngestThenChunkQuery`, `TestManifest_ConcurrentWritersDoNotLoseEntries`, sweep/auto tests green). Then purity greps — both must print nothing:
  - `grep -nE '\bos\.|filepath\.WalkDir|time\.Now|syscall' internal/cli/ingest.go`
  - `grep -n 'osFileReader' internal/cli/*.go` (only historical mentions in comments must be gone too)
  Then `targ check-full` — expected: no new lint findings (fix any reorder-decls/lll it reports in the touched files before proceeding).

- [ ] 10. Commit:

```
git add internal/cli/ingest.go internal/cli/cli.go internal/cli/targets.go \
  internal/cli/export_test.go internal/cli/ingest_family_deps_test.go \
  internal/cli/ingest_auto_test.go internal/cli/ingest_integration_test.go \
  internal/cli/ingest_test.go internal/cli/adapters_test.go
git commit -m "refactor(cli): compose ingest deps from cli.Deps, no direct os in ingest.go (#700)

ENGRAM_TRANSCRIPT_DIR + home via deps.Getenv/UserHomeDir, walk via
EdgeFS.WalkDir, manifest lock via FileLocker behind MkdirAll composition
(ADR-0013 semantics preserved; concurrency regression adapted to a
test-local real flock).

AI-Used: [claude]"
```

---

### Task T9 (I2): Migrate `engram prune` wiring to cli.Deps composition and retire osManifestLock

**Files:**
- Modify: `internal/cli/prune.go` (replace `newOsPruneDeps` with `newPruneDeps`; add shared `jsonlIndexListerFrom`)
- Modify: `internal/cli/cli.go` (delete `osManifestLock` — last consumer gone)
- Modify: `internal/cli/targets.go` (prune call site)
- Modify: `internal/cli/export_test.go` (swap `ExportNewOsPruneDeps` → `ExportNewPruneDeps`; delete `ExportOsManifestLock`)
- Modify: `internal/cli/ingest_family_deps_test.go` (add prune composition tests)
- Modify: `internal/cli/prune_integration_test.go`, `internal/cli/testhelpers_test.go`

**Interfaces:**
- Consumes: `Deps`/`EdgeFS`/`FileLocker` (foundation), `manifestLockFrom` (Task I1), `chunksDirPerm`/`indexFilePerm` consts (Task I1).
- Produces: `func newPruneDeps(d Deps) PruneDeps`; `func jsonlIndexListerFrom(readDir func(string) ([]fs.DirEntry, error)) func(string) ([]string, error)` (shared helper — query/show/amend clusters migrate their `listJSONLIndexes` call sites to this; the os-based `listJSONLIndexes` in query_chunks.go is deleted by whichever cluster removes its last consumer).

**Steps:**

- [ ] 1. RED — append to `internal/cli/ingest_family_deps_test.go` (references `cli.ExportNewPruneDeps`, which does not exist yet → compile RED). Also add `fakeDirEntry` to the private helpers section:

```go
func TestPruneDepsExistsAndRemoveGoThroughFS(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var removed []string

	d := testDeps()
	d.FS = fakeEdgeFS{
		stat: func(path string) (fs.FileInfo, error) {
			if path == "/live.jsonl" {
				return fakeFileInfo{}, nil
			}

			return nil, fs.ErrNotExist
		},
		remove: func(path string) error {
			removed = append(removed, path)

			return nil
		},
	}

	deps := cli.ExportNewPruneDeps(d)

	g.Expect(deps.Exists("/live.jsonl")).To(gomega.BeTrue())
	g.Expect(deps.Exists("/dead.jsonl")).To(gomega.BeFalse())
	g.Expect(deps.Remove("/chunks/empty.jsonl")).To(gomega.Succeed())
	g.Expect(removed).To(gomega.Equal([]string{"/chunks/empty.jsonl"}))
}

func TestPruneDepsListIndexesColdStartAndFiltering(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	d := testDeps()
	d.FS = fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return nil, fmt.Errorf("reading dir: %w", fs.ErrNotExist)
	}}

	deps := cli.ExportNewPruneDeps(d)

	paths, err := deps.ListIndexes("/absent")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(paths).To(gomega.BeEmpty(), "missing chunks dir is a cold start, not an error")

	d.FS = fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			fakeDirEntry{name: "a.jsonl"},
			fakeDirEntry{name: "notes.md"},
			fakeDirEntry{name: "sub", dir: true},
			fakeDirEntry{name: "b.jsonl"},
		}, nil
	}}
	deps = cli.ExportNewPruneDeps(d)

	paths, err = deps.ListIndexes("/chunks")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(paths).To(gomega.Equal([]string{
		filepath.Join("/chunks", "a.jsonl"),
		filepath.Join("/chunks", "b.jsonl"),
	}), "only non-dir .jsonl entries, joined under dir")

	d.FS = fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) { return nil, errBoom }}
	deps = cli.ExportNewPruneDeps(d)

	_, err = deps.ListIndexes("/chunks")
	g.Expect(err).To(gomega.MatchError(errBoom))
}
```

and in the private-helpers section:

```go
// fakeDirEntry satisfies fs.DirEntry for index-lister tests.
type fakeDirEntry struct {
	name string
	dir  bool
}

func (e fakeDirEntry) Name() string { return e.name }
func (e fakeDirEntry) IsDir() bool  { return e.dir }

func (e fakeDirEntry) Type() fs.FileMode {
	if e.dir {
		return fs.ModeDir
	}

	return 0
}

func (e fakeDirEntry) Info() (fs.FileInfo, error) { return nil, fs.ErrInvalid }
```

Run `targ test` — expected FAIL: `undefined: cli.ExportNewPruneDeps`.

- [ ] 2. GREEN — rewrite `internal/cli/prune.go`. Import block (current lines 3-10) becomes:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
)
```

Update the stale `PruneDeps.Lock` doc comment (current lines 21-22): `// Wired to flockPath(chunksDir/.manifest.lock) in newOsPruneDeps.` → `// Wired to manifestLockFrom (MkdirAll + FileLocker flock) in newPruneDeps.` Replace `newOsPruneDeps` (current lines 102-118):

```go
// newOsPruneDeps wires the production filesystem for `engram prune`.
func newOsPruneDeps() PruneDeps {
	fs := &osEmbedFS{}

	return PruneDeps{
		Lock:      osManifestLock,
		ReadFile:  fs.Read,
		WriteFile: fs.Write,
		Exists: func(path string) bool {
			_, statErr := os.Stat(path)

			return statErr == nil
		},
		ListIndexes: listJSONLIndexes,
		Remove:      os.Remove,
	}
}
```

with:

```go
// jsonlIndexListerFrom returns a lister of the .jsonl files directly under a
// dir, reading via the injected readDir (EdgeFS.ReadDir in production). A
// missing dir is an empty index (cold start), not an error. Shared shape with
// listJSONLIndexes (query_chunks.go); query/show/amend compose from here as
// they migrate, after which the os-backed listJSONLIndexes is deleted.
func jsonlIndexListerFrom(
	readDir func(string) ([]fs.DirEntry, error),
) func(string) ([]string, error) {
	return func(dir string) ([]string, error) {
		entries, err := readDir(dir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, nil
			}

			return nil, fmt.Errorf("listing chunk indexes: %w", err)
		}

		var paths []string

		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == jsonlExt {
				paths = append(paths, filepath.Join(dir, entry.Name()))
			}
		}

		return paths, nil
	}
}

// newPruneDeps composes production PruneDeps from the CLI-edge Deps.
func newPruneDeps(d Deps) PruneDeps {
	return PruneDeps{
		Lock:     manifestLockFrom(d),
		ReadFile: d.FS.ReadFile,
		WriteFile: func(path string, data []byte) error {
			err := d.FS.WriteFileAtomic(path, data, indexFilePerm)
			if err != nil {
				return fmt.Errorf("prune: writing %s: %w", path, err)
			}

			return nil
		},
		Exists: func(path string) bool {
			_, statErr := d.FS.Stat(path)

			return statErr == nil
		},
		ListIndexes: jsonlIndexListerFrom(d.FS.ReadDir),
		Remove:      d.FS.Remove,
	}
}
```

- [ ] 3. Delete `osManifestLock` from `internal/cli/cli.go` — exact current lines 223-236 (function + doc comment). Its two consumers (`newOsIngestDeps`, `newOsPruneDeps`) are now both gone. Leave `flockPath` in place (still consumed by `osLearnFS.Lock`; the learn cluster removes it).

- [ ] 4. Update `internal/cli/targets.go` prune call site. Current lines 161-166:

```go
		targ.Targ(func(ctx context.Context, a PruneArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, os.Getenv)
			errHandler(RunPrune(withLog(ctx), a, newOsPruneDeps(), stdout))
		}).Name("prune").Description(
```

becomes:

```go
		targ.Targ(func(ctx context.Context, a PruneArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunPrune(withLog(ctx), a, newPruneDeps(deps), stdout))
		}).Name("prune").Description(
```

- [ ] 5. Update `internal/cli/export_test.go`: in the Exported-variables block delete the line `ExportNewOsPruneDeps = newOsPruneDeps` (current line 79); delete `ExportOsManifestLock` (current lines 687-690). Add near the other constructor exports:

```go
// ExportNewPruneDeps returns production PruneDeps composed from d.
func ExportNewPruneDeps(d Deps) PruneDeps { return newPruneDeps(d) }
```

- [ ] 6. Adapt tests. `internal/cli/prune_integration_test.go` line 48: `cli.ExportNewOsPruneDeps()` → `cli.ExportNewPruneDeps(testDeps())`. `internal/cli/testhelpers_test.go`: delete `TestOsManifestLock_MkdirError` (current lines 13-29) and its now-unused imports if any (`os`, `filepath`, `cli`, `gomega` — check remaining uses; `sliceIndex` stays) — its coverage is replaced by the pure `TestIngestDepsLockMkdirErrorPropagates` (Task I1) plus the shared `manifestLockFrom` path now exercised by both constructors.

- [ ] 7. Verify: `targ test` — expected PASS, including `TestOsPruneDetachesDeadSource` (real FS through `testDeps()`), `TestRunPrune_LocksManifestAroundReadModifyWrite` (untouched — pure injected deps), and `TestManifest_ConcurrentWritersDoNotLoseEntries`. Purity grep, must print nothing: `grep -nE '\bos\.|time\.Now|syscall' internal/cli/prune.go`. Then `targ check-full` — expected clean. Then run the real flow once (passing tests are not a usable system): `go install ./cmd/engram && cd $(mktemp -d) && ENGRAM_CHUNKS_DIR=$PWD/chunks engram prune` — expected stdout `prune: no manifest, nothing to prune` and a created `chunks/.manifest.lock` (proves the MkdirAll-before-lock composition against the real wired binary; requires the foundation cmd wiring to be in place).

- [ ] 8. Commit:

```
git add internal/cli/prune.go internal/cli/cli.go internal/cli/targets.go \
  internal/cli/export_test.go internal/cli/ingest_family_deps_test.go \
  internal/cli/prune_integration_test.go internal/cli/testhelpers_test.go
git commit -m "refactor(cli): compose prune deps from cli.Deps, retire osManifestLock (#700)

Stat/Remove/ReadDir via EdgeFS, manifest lock via shared manifestLockFrom
(MkdirAll-before-flock fresh-dir regression preserved). jsonlIndexListerFrom
added as the pure lister for the remaining listJSONLIndexes call sites to
migrate onto.

AI-Used: [claude]"
```

---

Verified source anchors: internal/cli/ingest.go (os at 248/252/496/504/516/520, time.Now at 514, filepath.WalkDir at 662), internal/cli/prune.go (os.Stat 111, os.Remove 117), internal/cli/cli.go (osFileReader 27, flockPath 169, osManifestLock 227, manifestLockFile 17), internal/cli/targets.go (157-166), internal/cli/query_chunks.go (listJSONLIndexes 138), internal/cli/embed.go (osEmbedFS 140, sharedEmbedder 110), internal/context/context.go (FileReader 7), internal/embed/embedder.go (Embedder 54), tests: ingest_test.go 319/376/403/899, ingest_auto_test.go 48-105, ingest_integration_test.go 188, prune_integration_test.go 48, testhelpers_test.go 13-29, adapters_test.go 14-39, export_test.go 79/342/536/543/687.

### Task T10 (M1): Move atomic write to cmd/engram as EdgeFS.WriteFileAtomic

**Files**
- Create (or append to, if the cmd-wiring task already created it): `cmd/engram/os_fs.go`
- Create: `cmd/engram/os_fs_atomic_test.go`
- (No internal deletions yet — writesafe.go still has out-of-cluster callers; deleted in M4.)

**Interfaces**
- Produces: `func (osFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error` — the EdgeFS method absorbing internal/cli/writesafe.go.
- Produces (unexported, test seam): `func doAtomicWrite(path string, data []byte, perm fs.FileMode, rename func(oldpath, newpath string) error) error`
- Consumes: `EdgeFS` method-set contract from internal/cli/deps.go (foundation task).

**Steps**

1. [ ] RED (relocation form: new adapter integration tests against code that doesn't exist yet — compile failure is the RED): create `cmd/engram/os_fs_atomic_test.go`, package `main` (white-box; package main is unimportable so no `_test` package). This is the verbatim relocation of internal/cli/writesafe_test.go with `cli.ExportAtomicWriteFile` → `(osFS{}).WriteFileAtomic` and `cli.ExportDoAtomicWrite` → `doAtomicWrite`:

```go
package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestWriteFileAtomic_FailureDoesNotTouchOriginal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// Create a subdirectory to hold the original file.
	subdir := filepath.Join(dir, "sub")
	mkErr := os.Mkdir(subdir, 0o700)
	g.Expect(mkErr).NotTo(HaveOccurred())

	if mkErr != nil {
		return
	}

	target := filepath.Join(subdir, "original.txt")
	original := []byte("original untouched content")

	writeErr := os.WriteFile(target, original, 0o600)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	// Make the directory read-only so CreateTemp fails.
	chmodErr := os.Chmod(subdir, 0o500)
	g.Expect(chmodErr).NotTo(HaveOccurred())

	if chmodErr != nil {
		return
	}

	// Restore permissions so TempDir cleanup can succeed.
	t.Cleanup(func() { _ = os.Chmod(subdir, 0o700) })

	// Make the directory readable again for the final assertions.
	defer func() { _ = os.Chmod(subdir, 0o700) }()

	err := (osFS{}).WriteFileAtomic(target, []byte("new content"), 0o600)
	g.Expect(err).To(HaveOccurred(), "write to read-only dir must fail")

	// Restore permissions to check the original.
	_ = os.Chmod(subdir, 0o700)

	got, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(got).To(Equal(original), "original file must be untouched after failure")

	// No leftover temp files.
	tmpFiles, globErr := filepath.Glob(filepath.Join(subdir, ".original.txt.tmp-*"))
	g.Expect(globErr).NotTo(HaveOccurred())
	g.Expect(tmpFiles).To(BeEmpty(), "no leftover .tmp-* files must remain after failure")
}

func TestWriteFileAtomic_NoLeftoverTempFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")

	err := (osFS{}).WriteFileAtomic(target, []byte("data"), 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tmpFiles, globErr := filepath.Glob(filepath.Join(dir, ".out.txt.tmp-*"))
	g.Expect(globErr).NotTo(HaveOccurred())
	g.Expect(tmpFiles).To(BeEmpty(), "no leftover .tmp-* files must remain after success")
}

func TestWriteFileAtomic_OverwritesExistingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")
	original := []byte("original content")
	updated := []byte("updated content")

	writeErr := os.WriteFile(target, original, 0o600)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	err := (osFS{}).WriteFileAtomic(target, updated, 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	got, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(got).To(Equal(updated), "file must contain the updated bytes")
}

func TestWriteFileAtomic_RenameFailure_CleansTempAndLeavesOriginalUntouched(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	original := []byte("original content")

	// Write the original file before the failing atomic write.
	writeErr := os.WriteFile(target, original, 0o600)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	// Track the temp path so we can verify the defer cleans it up.
	var tmpSeen string

	err := doAtomicWrite(target, []byte("new content"), 0o600,
		func(src, _ string) error {
			tmpSeen = src

			return errInjectedRename
		},
	)

	g.Expect(err).To(MatchError(ContainSubstring("rename")), "error must mention rename")

	// Original file must be untouched.
	got, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(got).To(Equal(original), "original must be untouched after rename failure")

	// Temp file must be removed by the defer cleanup.
	if tmpSeen != "" {
		_, statErr := os.Stat(tmpSeen)
		g.Expect(os.IsNotExist(statErr)).To(BeTrue(), "temp file must be cleaned up by defer")
	}
}

func TestWriteFileAtomic_WritesNewFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")
	content := []byte("hello atomic world")

	err := (osFS{}).WriteFileAtomic(target, content, 0o600)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	got, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(got).To(Equal(content), "file must contain exactly the written bytes")
}

// errInjectedRename is the sentinel injected by the rename-failure test.
var errInjectedRename = errors.New("injected rename failure")
```

2. [ ] Run `targ test` — expect FAIL (undefined: osFS / doAtomicWrite). That is the RED.

3. [ ] GREEN: add to `cmd/engram/os_fs.go` (if the cmd-wiring task hasn't created the file yet, create it with exactly this content; if it exists, append only the two functions — `osFS` and the imports will already be there). The moved code is verbatim from internal/cli/writesafe.go lines 9–73 with exactly two mechanical changes: `atomicWriteFile(path, ...)` becomes the method `(osFS) WriteFileAtomic`, and `perm os.FileMode` becomes `perm fs.FileMode` (alias types — `os.Chmod` accepts it unchanged):

```go
// Package main wires the engram CLI's impure edges (ADR: internal/ purity, #700).
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// osFS is the production EdgeFS implementation over the real filesystem.
type osFS struct{}

// WriteFileAtomic writes data to path atomically: it creates a unique temp
// file in filepath.Dir(path), sets perms, writes, closes, then renames into
// place. A same-directory rename is atomic on POSIX — a concurrent reader
// sees either the old or the new file, never a partial one. On any error the
// temp file is removed and the original (if any) is left untouched.
// Moved verbatim from internal/cli/writesafe.go (ADR-0013 semantics preserved).
func (osFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	return doAtomicWrite(path, data, perm, os.Rename)
}

// doAtomicWrite is the testable core of WriteFileAtomic. The rename parameter
// is injected so tests can trigger the rename-failure path and verify that the
// defer cleanup removes the temp file and the original is left untouched.
func doAtomicWrite(
	path string,
	data []byte,
	perm fs.FileMode,
	rename func(oldpath, newpath string) error,
) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	tmp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return fmt.Errorf("atomic write %s: create temp: %w", path, err)
	}

	tmpName := tmp.Name()

	// Best-effort cleanup on any error path.
	success := false

	defer func() {
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	chmodErr := os.Chmod(tmpName, perm)
	if chmodErr != nil {
		_ = tmp.Close()

		return fmt.Errorf("atomic write %s: chmod temp: %w", path, chmodErr)
	}

	_, writeErr := tmp.Write(data)
	if writeErr != nil {
		_ = tmp.Close()

		return fmt.Errorf("atomic write %s: write temp: %w", path, writeErr)
	}

	closeErr := tmp.Close()
	if closeErr != nil {
		return fmt.Errorf("atomic write %s: close temp: %w", path, closeErr)
	}

	renameErr := rename(tmpName, path)
	if renameErr != nil {
		return fmt.Errorf("atomic write %s: rename: %w", path, renameErr)
	}

	success = true

	return nil
}
```

4. [ ] Run `targ test` — expect PASS (5 new cmd tests green; internal suite untouched, still green).
5. [ ] Run `targ check-full` — expect clean, EXCEPT possibly an unused-`osFS` finding if the cmd-wiring task hasn't landed (see sequencing DESIGN FLAG; if flagged, hold this commit until wiring lands rather than suppressing).
6. [ ] Commit:

```
refactor(cli): move atomic write to cmd/engram as EdgeFS.WriteFileAtomic (#700)

Relocates the internal/cli/writesafe.go temp+chmod+write+close+rename dance
verbatim onto the production EdgeFS adapter; relocates its 5 regression tests
as package-main integration tests. internal/cli/writesafe.go stays until all
internal callers are migrated (removed by the follow-up purge task).

AI-Used: [claude]
```

---

### Task T11 (M2): Pure edge-composition helpers + os-backed test Deps

**Files**
- Create: `internal/cli/deps_compose.go`
- Create: `internal/cli/deps_compose_internal_test.go`
- Create: `internal/cli/testdeps_test.go`

**Interfaces**
- Consumes: `type Deps struct{...}`, `type EdgeFS interface{...}`, `type FileLocker interface{...}` from internal/cli/deps.go (foundation task — M2 is blocked on it); `luhmannLockFile` const (internal/cli/cli.go:16, pure, stays); `jsonlExt` const (query_chunks.go, pure, stays).
- Produces:
  - `type edgeVaultFS struct{ fs EdgeFS }` with `ListMD(dir string) ([]string, error)` and `ReadFile(path string) ([]byte, error)` — satisfies `vaultgraph.VaultFS`.
  - `func vaultLuhmannLock(locker FileLocker) func(vault string) (func(), error)`
  - `func warnLoggerTo(w io.Writer) func(format string, args ...any)`
  - `func jsonlIndexesLister(edgeFS EdgeFS) func(dir string) ([]string, error)`
  - Test-only: `func ExportNewTestOsDeps() Deps` (package cli, _test.go file).

**Steps**

1. [ ] RED: create `internal/cli/deps_compose_internal_test.go` (package `cli`, internal-test precedent: resituate_internal_test.go). Compile failure on the missing helpers is the RED:

```go
package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestEdgeVaultFS_ListMD_FiltersToMDFilesSkippingDirs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vaultFS := edgeVaultFS{fs: fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			fakeDirEntry{name: "1.2026-01-01.note.md"},
			fakeDirEntry{name: "sidecar.vec.json"},
			fakeDirEntry{name: "subdir", dir: true},
		}, nil
	}}}

	names, err := vaultFS.ListMD("/vault")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(names).To(Equal([]string{"1.2026-01-01.note.md"}))
}

func TestEdgeVaultFS_ListMD_MissingDirIsEmptyNotError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vaultFS := edgeVaultFS{fs: fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return nil, fmt.Errorf("read dir: %w", fs.ErrNotExist)
	}}}

	names, err := vaultFS.ListMD("/missing")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(names).To(BeEmpty())
}

func TestEdgeVaultFS_ReadFile_WrapsErrorWithPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vaultFS := edgeVaultFS{fs: fakeEdgeFS{readFile: func(string) ([]byte, error) {
		return nil, errInjectedCompose
	}}}

	_, err := vaultFS.ReadFile("/vault/x.md")
	g.Expect(err).To(MatchError(errInjectedCompose))
	g.Expect(err).To(MatchError(ContainSubstring("/vault/x.md")))
}

func TestJSONLIndexesLister_FiltersAndTreatsMissingDirAsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := jsonlIndexesLister(fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			fakeDirEntry{name: "s.jsonl"},
			fakeDirEntry{name: "manifest.json"},
			fakeDirEntry{name: "nested", dir: true},
		}, nil
	}})

	paths, err := lister("/chunks")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(paths).To(Equal([]string{"/chunks/s.jsonl"}))

	missing := jsonlIndexesLister(fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return nil, fmt.Errorf("read dir: %w", fs.ErrNotExist)
	}})

	paths, err = missing("/gone")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(paths).To(BeEmpty())
}

func TestVaultLuhmannLock_LocksVaultLuhmannLockFileAndReleases(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var lockedPath string

	unlocked := false
	locker := fakeLocker{lock: func(path string) (func() error, error) {
		lockedPath = path

		return func() error { unlocked = true; return nil }, nil
	}}

	lock := vaultLuhmannLock(locker)

	release, err := lock("/vault")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(lockedPath).To(Equal("/vault/.luhmann.lock"))

	release()
	g.Expect(unlocked).To(BeTrue())
}

func TestWarnLoggerTo_FormatsWithWarningPrefixAndNewline(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	warnLoggerTo(&buf)("amend: %s failed after %d tries", "embed", 2)
	g.Expect(buf.String()).To(Equal("warning: amend: embed failed after 2 tries\n"))
}

// errInjectedCompose is the sentinel injected by the compose-helper tests.
var errInjectedCompose = errors.New("injected compose failure")

// fakeDirEntry is a minimal fs.DirEntry for ListMD/lister tests.
type fakeDirEntry struct {
	name string
	dir  bool
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return f.dir }
func (f fakeDirEntry) Type() fs.FileMode          { return 0 }
func (f fakeDirEntry) Info() (fs.FileInfo, error) { return nil, fs.ErrInvalid }

// fakeEdgeFS overrides only the EdgeFS methods a test exercises; calling an
// un-overridden method panics via the embedded nil interface (test bug, loud).
type fakeEdgeFS struct {
	EdgeFS

	readDir  func(string) ([]fs.DirEntry, error)
	readFile func(string) ([]byte, error)
}

func (f fakeEdgeFS) ReadDir(path string) ([]fs.DirEntry, error) { return f.readDir(path) }

func (f fakeEdgeFS) ReadFile(path string) ([]byte, error) { return f.readFile(path) }

// fakeLocker records the locked path.
type fakeLocker struct {
	lock func(string) (func() error, error)
}

func (f fakeLocker) Lock(path string) (func() error, error) { return f.lock(path) }

// silence unused-import when time is only used by other tests in this package.
var _ = time.Now
```

   (Drop the trailing `var _ = time.Now` and the `time` import if the linter flags them — they exist only if a later step in this file needs a clock; expected final state has neither.)

2. [ ] Run `targ test` — expect FAIL (undefined: edgeVaultFS, jsonlIndexesLister, vaultLuhmannLock, warnLoggerTo). RED confirmed.

3. [ ] GREEN: create `internal/cli/deps_compose.go` — pure composition, allowed imports only (errors/fmt/io/io\/fs/path\/filepath/strings):

```go
package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
)

// edgeVaultFS adapts the injected EdgeFS to vaultgraph.VaultFS (ListMD +
// ReadFile) and to every per-command dep needing the same shape. Listing a
// non-existent directory returns empty (not an error) — the scanner uses this
// to skip missing dirs, matching the retired osVaultFS contract. Relies on
// EdgeFS.ReadDir wrapping its error with %w so fs.ErrNotExist survives.
type edgeVaultFS struct {
	fs EdgeFS
}

// ListMD returns the .md filenames in dir. Missing dir → empty, nil.
func (v edgeVaultFS) ListMD(dir string) ([]string, error) {
	entries, err := v.fs.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading dir %s: %w", dir, err)
	}

	out := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		out = append(out, entry.Name())
	}

	return out, nil
}

// ReadFile reads the file at path, wrapping errors with the path for parity
// with the retired osVaultFS.ReadFile.
func (v edgeVaultFS) ReadFile(path string) ([]byte, error) {
	data, err := v.fs.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return data, nil
}

// jsonlIndexesLister returns a chunk-index lister backed by the injected
// EdgeFS: the .jsonl files directly under dir. A missing dir is an empty
// index (cold start), not an error. Injected twin of the retired os-backed
// listJSONLIndexes (query_chunks.go) — same contract, including never
// matching manifest.json (not a .jsonl file).
func jsonlIndexesLister(edgeFS EdgeFS) func(dir string) ([]string, error) {
	return func(dir string) ([]string, error) {
		entries, err := edgeFS.ReadDir(dir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, nil
			}

			return nil, fmt.Errorf("listing chunk indexes: %w", err)
		}

		paths := make([]string, 0, len(entries))

		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == jsonlExt {
				paths = append(paths, filepath.Join(dir, entry.Name()))
			}
		}

		return paths, nil
	}
}

// vaultLuhmannLock adapts the injected FileLocker to the per-command
// Lock(vault) shape: an exclusive lock on vault/.luhmann.lock whose release
// discards the unlock error — exact parity with the retired flockPath release
// (which swallowed LOCK_UN and Close errors). Lock acquisition stays at Run*
// entry points only (ADR-0013); helpers must never re-acquire.
func vaultLuhmannLock(locker FileLocker) func(vault string) (func(), error) {
	return func(vault string) (func(), error) {
		unlock, err := locker.Lock(filepath.Join(vault, luhmannLockFile))
		if err != nil {
			return nil, fmt.Errorf("acquiring %s: %w", luhmannLockFile, err)
		}

		return func() { _ = unlock() }, nil
	}
}

// warnLoggerTo returns a LogWarning func writing "warning: ..." lines to w —
// the injected-Stderr replacement for the retired logWarningToStderrf.
func warnLoggerTo(w io.Writer) func(format string, args ...any) {
	return func(format string, args ...any) {
		_, _ = fmt.Fprintf(w, "warning: "+format+"\n", args...)
	}
}
```

4. [ ] Run `targ test` — expect PASS (M2 tests green).

5. [ ] Create `internal/cli/testdeps_test.go` (package `cli` — test file, exempt from purity enforcement; mirrors the production cmd adapters so wiring tests exercise real-FS composition). The `WriteFileAtomic` body is intentionally a copy of the production dance (cmd/engram is unimportable):

```go
package cli

// Test-only os-backed edge implementations so wiring-integration tests can
// drive the production composition (newAmendDeps etc.) against a real
// filesystem. Production adapters live in cmd/engram (unimportable package
// main), so this file mirrors them; _test.go files are exempt from the #700
// internal-purity enforcement.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// ExportNewTestOsDeps returns a Deps wired to the real filesystem and clock
// for wiring tests. Embed is nil — tests needing an embedder override the
// per-command deps field after construction (existing pattern: fakeEmbedder).
func ExportNewTestOsDeps() Deps {
	return Deps{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Getenv: os.Getenv,
		Now:    time.Now,
		FS:     testOsFS{},
		Lock:   testFlockLocker{},
	}
}

// testFlockLocker implements FileLocker via an exclusive flock, mirroring the
// production cmd/engram flockLocker (ADR-0013).
type testFlockLocker struct{}

// Lock opens path (O_CREATE|O_RDWR), acquires LOCK_EX, and returns an unlock
// that releases and closes.
func (testFlockLocker) Lock(path string) (func() error, error) {
	const perm = 0o600

	lockFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, perm) //nolint:gosec // test helper
	if err != nil {
		return nil, fmt.Errorf("open lock: %w", err)
	}

	fileDescriptor := int(lockFile.Fd())

	flockErr := syscall.Flock(fileDescriptor, syscall.LOCK_EX)
	if flockErr != nil {
		_ = lockFile.Close()

		return nil, fmt.Errorf("flock: %w", flockErr)
	}

	return func() error {
		_ = syscall.Flock(fileDescriptor, syscall.LOCK_UN)

		closeErr := lockFile.Close()
		if closeErr != nil {
			return fmt.Errorf("close lock: %w", closeErr)
		}

		return nil
	}, nil
}

// testOsFS implements EdgeFS over the real filesystem for wiring tests.
type testOsFS struct{}

func (testOsFS) MkdirAll(path string, perm fs.FileMode) error {
	err := os.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}

	return nil
}

func (testOsFS) MkdirTemp(dir, pattern string) (string, error) {
	made, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("mkdtemp %s: %w", dir, err)
	}

	return made, nil
}

func (testOsFS) ReadDir(path string) ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", path, err)
	}

	return entries, nil
}

func (testOsFS) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path) //nolint:gosec // test helper, path from test
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return data, nil
}

func (testOsFS) Remove(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}

	return nil
}

func (testOsFS) RemoveAll(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("remove all %s: %w", path, err)
	}

	return nil
}

func (testOsFS) Rename(oldPath, newPath string) error {
	err := os.Rename(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("rename %s: %w", oldPath, err)
	}

	return nil
}

func (testOsFS) Stat(path string) (fs.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	return info, nil
}

func (testOsFS) WalkDir(root string, walkFn fs.WalkDirFunc) error {
	err := filepath.WalkDir(root, walkFn)
	if err != nil {
		return fmt.Errorf("walk %s: %w", root, err)
	}

	return nil
}

func (testOsFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	err := os.WriteFile(path, data, perm) //nolint:gosec // test helper
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// WriteFileAtomic mirrors the production cmd/engram temp+rename dance — the
// real semantics matter here: ingest's concurrent-manifest regression test
// (ADR-0013) races goroutines through this writer and depends on readers
// never seeing a torn write.
func (testOsFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	tmp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return fmt.Errorf("atomic write %s: create temp: %w", path, err)
	}

	tmpName := tmp.Name()

	success := false

	defer func() {
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	chmodErr := os.Chmod(tmpName, perm)
	if chmodErr != nil {
		_ = tmp.Close()

		return fmt.Errorf("atomic write %s: chmod temp: %w", path, chmodErr)
	}

	_, writeErr := tmp.Write(data)
	if writeErr != nil {
		_ = tmp.Close()

		return fmt.Errorf("atomic write %s: write temp: %w", path, writeErr)
	}

	closeErr := tmp.Close()
	if closeErr != nil {
		return fmt.Errorf("atomic write %s: close temp: %w", path, closeErr)
	}

	renameErr := os.Rename(tmpName, path)
	if renameErr != nil {
		return fmt.Errorf("atomic write %s: rename: %w", path, renameErr)
	}

	success = true

	return nil
}
```

6. [ ] Run `targ test` then `targ check-full` — expect PASS/clean. (`ExportNewTestOsDeps` is unreferenced until M3; if the unused linter flags it, land M2+M3 as one PR — do not suppress.)
7. [ ] Commit:

```
refactor(cli): add pure edge-composition helpers + os-backed test Deps (#700)

edgeVaultFS/vaultLuhmannLock/warnLoggerTo/jsonlIndexesLister compose
per-command deps from the injected Deps without touching os/syscall in
production code; testdeps_test.go mirrors the cmd adapters so wiring
tests keep exercising the real filesystem.

AI-Used: [claude]
```

---

### Task T12 (M3): Maintenance-family constructors compose from Deps

**Files**
- Modify: `internal/cli/amend.go`, `internal/cli/resituate.go`, `internal/cli/activate.go`, `internal/cli/vocab_commands.go`
- Modify: `internal/cli/export_test.go`, `internal/cli/activate_test.go`, `internal/cli/amend_test.go`, `internal/cli/resituate_test.go`, `internal/cli/vocab_commands_test.go`, `internal/cli/vocab_trigger_test.go`, `internal/cli/learn_test.go` (one line — cross-cluster, see flag)
- Modify (call expressions only; signature threading owned by wiring cluster): `internal/cli/targets.go`
- Verify-only (no edits): `internal/cli/vocab.go`, `internal/cli/vault_init.go`

**Interfaces**
- Consumes: `Deps` (deps.go), M2 helpers, `d.FS.WriteFileAtomic`, `d.Embed embed.Embedder`.
- Produces: `func newAmendDeps(d Deps) AmendDeps`, `func newResituateDeps(d Deps) ResituateDeps`, `func newActivateDeps(d Deps) ActivateDeps`, `func newVocabDeps(d Deps) VocabDeps`, `func newVocabStatsDeps(d Deps) VocabStatsDeps` — replacing `newOsAmendDeps()`, `newOsResituateDeps()`, `newOsActivateDeps()`, `newOsVocabDeps()`, `newOsVocabStatsDeps()`. Deletes `osWriteSidecar`.

**Steps**

1. [ ] RED (refactor form — the existing wiring-integration suite IS the safety net; first make the test call sites demand the new signatures): apply these test edits, run `targ test`, expect compile FAIL (undefined new names):
   - activate_test.go:121: `deps := cli.ExportNewOsActivateDeps()` → `deps := cli.ExportNewActivateDeps(cli.ExportNewTestOsDeps())`
   - amend_test.go:818: `deps := cli.ExportNewOsAmendDeps()` → `deps := cli.ExportNewAmendDeps(cli.ExportNewTestOsDeps())`
   - learn_test.go:132: `deps := cli.ExportNewOsAmendDeps()` → `deps := cli.ExportNewAmendDeps(cli.ExportNewTestOsDeps())`
   - vocab_commands_test.go:270, 737, 1020, 3790 and vocab_trigger_test.go:411: `cli.ExportNewOsVocabDeps()` → `cli.ExportNewVocabDeps(cli.ExportNewTestOsDeps())`
   - resituate_test.go:251, 295, 409, 519: `cli.ExportNewOsResituateDeps(successEmbedder{})` → `cli.ExportNewResituateDeps(cli.ExportNewTestOsDeps(), successEmbedder{})`
   - vocab_commands_test.go:737 comment header `// ── Coverage: newOsVocabDeps closures ──…` and the TestNewOsVocabDeps_ClosuresCalled name/doc: rename to `TestNewVocabDeps_ClosuresCalled` / "closures inside newVocabDeps".

2. [ ] export_test.go shims. In the var block (keep alphabetical; these lines currently read as shown), replace:

```go
	ExportNewOsActivateDeps                = newOsActivateDeps
	ExportNewOsAmendDeps                   = newOsAmendDeps
```
with
```go
	ExportNewActivateDeps                  = newActivateDeps
	ExportNewAmendDeps                     = newAmendDeps
```
and
```go
	ExportNewOsVocabDeps                   = newOsVocabDeps
```
with
```go
	ExportNewVocabDeps                     = newVocabDeps
```
(re-sort the block; `ExportNewActivateDeps`/`ExportNewAmendDeps` sort before `ExportNewErrHandler`). Replace the resituate func shim:

```go
// ExportNewOsResituateDeps returns production ResituateDeps with an injected
// embedder so coverage tests can drive Scan/Read/Write without unpacking the
// lazy bundled embedder.
func ExportNewOsResituateDeps(emb embed.Embedder) ResituateDeps {
	deps := newOsResituateDeps()
	deps.Embedder = emb

	return deps
}
```
with
```go
// ExportNewResituateDeps returns production-composed ResituateDeps with an
// injected embedder so coverage tests can drive Scan/Read/Write without
// unpacking the lazy bundled embedder.
func ExportNewResituateDeps(d Deps, emb embed.Embedder) ResituateDeps {
	deps := newResituateDeps(d)
	deps.Embedder = emb

	return deps
}
```

3. [ ] GREEN — amend.go. Replace `newOsAmendDeps` (current code at amend.go:337-378, shown in Files context above) with:

```go
// newAmendDeps composes RunAmend's dependencies from the injected edge Deps
// (pure composition — no direct I/O; #700). ChunksDir flows through
// AmendArgs, not here.
func newAmendDeps(d Deps) AmendDeps {
	const perm = 0o600

	vaultFS := edgeVaultFS{fs: d.FS}

	return AmendDeps{
		Lock: vaultLuhmannLock(d.Lock),
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(vaultFS, vault)
		},
		Read: vaultFS.ReadFile,
		Write: func(path string, data []byte) error {
			err := d.FS.WriteFileAtomic(path, data, perm)
			if err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}

			return nil
		},
		Embedder:     d.Embed,
		Now:          d.Now,
		LoadChunkIDs: buildChunkIDSet,
		// jsonlIndexesLister lists *.jsonl chunk indexes, treats an absent dir
		// as empty (not an error), and never matches manifest.json — exactly
		// the contract the retired os-backed listJSONLIndexes provided.
		ListIndexes: jsonlIndexesLister(d.FS),
		LogWarning:  warnLoggerTo(d.Stderr),
		// Vocab assignment wiring: no-op when the vault has no term notes.
		// Uses stored member centroids (vocab.centroids.json) when present,
		// falling back to description embeddings per term.
		LoadTermVectors: func(vault string) ([]TermWithVector, error) {
			return loadAssignmentTermVectors(vault, vaultFS.ListMD, vaultFS.ReadFile)
		},
		// ListMD provides full .md filenames for the vocab trigger scan.
		// Must use ListMD (not stripped basenames) — basename filtering causes
		// false-fire on the untagged trigger.
		ListMD: vaultFS.ListMD,
	}
}
```
Doc-comment touch-ups in the same file: AmendDeps struct comment line 43 `The production wiring in newOsAmendDeps supplies os.ReadDir/os.ReadFile via closures.` → `The production wiring in newAmendDeps supplies the injected EdgeFS via closures.`; Lock field comment line 47 `Wired to vaultFS.Lock in newOsAmendDeps.` → `Wired via vaultLuhmannLock in newAmendDeps.`

4. [ ] resituate.go. Replace `newOsResituateDeps` (resituate.go:155-184) with:

```go
// newResituateDeps composes RunResituate's dependencies from the injected
// edge Deps (pure composition — no direct I/O; #700).
func newResituateDeps(d Deps) ResituateDeps {
	const perm = 0o600

	vaultFS := edgeVaultFS{fs: d.FS}

	return ResituateDeps{
		Lock: vaultLuhmannLock(d.Lock),
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(vaultFS, vault)
		},
		Read: vaultFS.ReadFile,
		Write: func(path string, data []byte) error {
			err := d.FS.WriteFileAtomic(path, data, perm)
			if err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}

			return nil
		},
		Embedder: d.Embed,
		LoadTermVectors: func(vault string) ([]TermWithVector, error) {
			return loadAssignmentTermVectors(vault, vaultFS.ListMD, vaultFS.ReadFile)
		},
		ListMD:     vaultFS.ListMD,
		LogWarning: warnLoggerTo(d.Stderr),
		Now:        d.Now,
	}
}
```
ResituateDeps.Lock comment line 28-29 `Wired to vaultFS.Lock in newOsResituateDeps.` → `Wired via vaultLuhmannLock in newResituateDeps.`

5. [ ] activate.go. Delete the `os` import; replace `newOsActivateDeps` + `osWriteSidecar` (activate.go:120-137) with:

```go
// newActivateDeps composes RunActivate's dependencies from the injected edge
// Deps (pure composition — no direct I/O; #700). Sidecar writes go through
// WriteFileAtomic (temp+rename) so concurrent readers always see either the
// old or new file.
func newActivateDeps(d Deps) ActivateDeps {
	const sidecarPerm = 0o600

	return ActivateDeps{
		Lock: vaultLuhmannLock(d.Lock),
		Now:  d.Now,
		Read: d.FS.ReadFile,
		Write: func(path string, data []byte) error {
			return d.FS.WriteFileAtomic(path, data, sidecarPerm)
		},
		LogWarning: warnLoggerTo(d.Stderr),
	}
}
```
Comment touch-ups: ActivateDeps.Lock comment line 23 `Wired to vaultFS.Lock in newOsActivateDeps.` → `Wired via vaultLuhmannLock in newActivateDeps.`; bumpLastUsed comment lines 86-87 `Sidecar writes go through atomicWriteFile (temp+rename) AND RunActivate holds the vault flock` → `Sidecar writes go through the injected atomic write (WriteFileAtomic, temp+rename) AND RunActivate holds the vault flock`.

6. [ ] vocab_commands.go. Delete the `os` import; replace `newOsVocabDeps` + `newOsVocabStatsDeps` (vocab_commands.go:1208-1240) with (behavior parity: WriteFile/DeleteFile error text preserved; WriteSidecar keeps osEmbedFS.Write's `"write: %w"` wrap):

```go
// newVocabDeps composes VocabDeps from the injected edge Deps (pure
// composition — no direct I/O; #700).
func newVocabDeps(d Deps) VocabDeps {
	const sidecarPerm = 0o600

	vaultFS := edgeVaultFS{fs: d.FS}

	return VocabDeps{
		Lock:     vaultLuhmannLock(d.Lock),
		ListMD:   vaultFS.ListMD,
		ReadFile: vaultFS.ReadFile,
		WriteFile: func(path string, data []byte) error {
			return d.FS.WriteFileAtomic(path, data, vocabNotePerm)
		},
		DeleteFile: func(path string) error {
			deleteErr := d.FS.Remove(filepath.Clean(path))
			if deleteErr != nil {
				return fmt.Errorf("deleting %s: %w", path, deleteErr)
			}

			return nil
		},
		WriteSidecar: func(path string, data []byte) error {
			err := d.FS.WriteFileAtomic(path, data, sidecarPerm)
			if err != nil {
				return fmt.Errorf("write: %w", err)
			}

			return nil
		},
		Embedder:   d.Embed,
		LogWarning: warnLoggerTo(d.Stderr),
		Now:        d.Now,
	}
}

// newVocabStatsDeps composes the read-only vocab stats deps from the injected
// edge Deps.
func newVocabStatsDeps(d Deps) VocabStatsDeps {
	vaultFS := edgeVaultFS{fs: d.FS}

	return VocabStatsDeps{
		ListMD:   vaultFS.ListMD,
		ReadFile: vaultFS.ReadFile,
	}
}
```

7. [ ] targets.go call-expression updates (coordinate with wiring cluster's `deps Deps` threading through `amendResituateTargets`/`ingestQueryTargets`/`vocabTargets`; only the constructor expressions belong to this task):
   - line 108: `newOsResituateDeps()` → `newResituateDeps(deps)`
   - line 113: `newOsAmendDeps()` → `newAmendDeps(deps)`
   - line 173: `newOsActivateDeps()` → `newActivateDeps(deps)`
   - lines 278/286/290: `newOsVocabDeps()` → `newVocabDeps(deps)`
   - line 282: `newOsVocabStatsDeps()` → `newVocabStatsDeps(deps)`

8. [ ] Run `targ test` — expect PASS: the relocated wiring-integration tests (activate/amend/resituate/vocab against real t.TempDir vaults) prove the composed deps behave identically; resituate tests still inject `successEmbedder{}`; vocab tests still override `deps.Embedder = &fakeEmbedder{}`.
9. [ ] Purity verification for this cluster (enforcement task lands later; this is the leave-nothing-behind check the central spec demands):
   - `grep -n "\"os\"\|os\.\|syscall\|time\.Now\|time\.Since\|time\.Tick" internal/cli/amend.go internal/cli/resituate.go internal/cli/activate.go internal/cli/vocab.go internal/cli/vocab_commands.go internal/cli/vault_init.go` — expected: NO import of `os`/`syscall`, no `time.Now/Since/Tick` calls; only comment mentions (scrub remaining comment references: amend.go:43 handled in step 3; vocab_commands.go:1126 `os.ReadDir sorts by name` → reword to `the OS-backed lister sorts by name`; resituate.go:128 `wiring provides time.Now` → `wiring provides the injected clock`).
   - Verify-only: vocab.go and vault_init.go unchanged (imports already pure; `fs.FileMode` from io/fs stays per spec).
10. [ ] Run `targ check-full` — expect clean (lint + coverage; the composed constructors are covered by the wiring tests, matching the coverage intent behind the old named `osWriteSidecar`/`logWarningToStderrf` pattern).
11. [ ] Commit:

```
refactor(cli): compose maintenance-family deps from injected edge Deps (#700)

newAmendDeps/newResituateDeps/newActivateDeps/newVocabDeps/newVocabStatsDeps
replace their newOsXxx forms: flock via FileLocker (.luhmann.lock at Run*
entry only, ADR-0013), atomic note/sidecar writes via EdgeFS.WriteFileAtomic,
clock via Deps.Now, warnings via Deps.Stderr, embedder via Deps.Embed.
activate.go and vocab_commands.go drop their os imports; vocab.go and
vault_init.go verified already pure.

AI-Used: [claude]
```

---

### Task T13 (M4): Purge internal atomic write (gated)

**GATE (do not start until true):** `grep -rn "atomicWriteFile" internal/cli --include="*.go" | grep -v _test | grep -v writesafe.go` returns EMPTY — i.e. learn.go:371, cli.go:144, embed.go:164, qa.go:283 have been migrated by their clusters.

**Files**
- Delete: `internal/cli/writesafe.go`, `internal/cli/writesafe_test.go`
- Modify: `internal/cli/export_test.go` (remove two shims), `internal/cli/ingest_test.go` (one line)

**Interfaces**
- Removes: `atomicWriteFile`, `doAtomicWrite`, `ExportAtomicWriteFile`, `ExportDoAtomicWrite` from internal/cli.

**Steps**

1. [ ] Repoint the ADR-0013 concurrent-manifest regression infra (must survive per spec — relocate, never delete). ingest_test.go:898-900, current:

```go
func (r *realFS) write(_, path string, data []byte) error {
	return cli.ExportAtomicWriteFile(path, data, 0o600)
}
```
replacement (same real temp+rename semantics via the M2 test EdgeFS):

```go
func (r *realFS) write(_, path string, data []byte) error {
	return cli.ExportNewTestOsDeps().FS.WriteFileAtomic(path, data, 0o600)
}
```

2. [ ] Delete `internal/cli/writesafe.go` and `internal/cli/writesafe_test.go` (all five behaviors live on as cmd/engram/os_fs_atomic_test.go from M1).
3. [ ] In export_test.go, delete the two function shims (lines 204-207 and 331-340 in current numbering):

```go
// ExportAtomicWriteFile exposes atomicWriteFile for writesafe tests.
func ExportAtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	return atomicWriteFile(path, data, perm)
}
```
and
```go
// ExportDoAtomicWrite exposes doAtomicWrite for writesafe tests that need to
// inject a failing rename to cover the rename-error and defer-cleanup paths.
func ExportDoAtomicWrite(
	path string,
	data []byte,
	perm os.FileMode,
	rename func(oldpath, newpath string) error,
) error {
	return doAtomicWrite(path, data, perm, rename)
}
```
(If these were export_test.go's last uses of the `os` import, drop that import too — check compile.)

4. [ ] Verify gate held: `grep -rn "atomicWriteFile\|doAtomicWrite" internal/` — expected EMPTY.
5. [ ] Run `targ test` — expect PASS, including the ingest concurrent-writers regression test (its lock is still real flock via ExportFlockPath — other cluster — and its writer is now the test EdgeFS atomic write).
6. [ ] Run `targ check-full` — expect clean.
7. [ ] Commit:

```
refactor(cli): delete internal atomic-write after all callers migrated (#700)

writesafe.go's dance now lives solely on cmd/engram's EdgeFS adapter (and its
test mirror); the ADR-0013 concurrent-manifest regression test now writes
through the test EdgeFS's WriteFileAtomic with identical temp+rename
semantics.

AI-Used: [claude]
```

---

Key file paths: /Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/internal/cli/{writesafe.go, writesafe_test.go, amend.go, resituate.go, activate.go, vocab.go, vocab_commands.go, vault_init.go, cli.go, targets.go, export_test.go, ingest_test.go, learn_test.go}, /Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/cmd/engram/main.go.

### Task T14 (A): Move hugot backend + model-cache FS out of internal/embed; inject Backend/CacheFS seams

**Files**

- Modify: `internal/embed/hugot.go` (full rewrite below)
- Modify: `internal/embed/cache.go` (full rewrite below)
- Modify: `internal/embed/export_test.go` (full rewrite below)
- Modify: `internal/embed/hugot_test.go`, `internal/embed/cache_test.go`, `internal/embed/buildembedder_test.go`, `internal/embed/overlength_test.go`, `internal/embed/embedder_fake_test.go`
- Delete: `internal/embed/production_cache_test.go`, `internal/embed/production_hugot_test.go`, `internal/embed/unpack_test.go`, `internal/embed/tempfs_test.go`
- Create: `cmd/engram/hugot.go`, `cmd/engram/hugot_test.go`, `cmd/engram/os_cachefs_test.go`, `cmd/engram/bundled_smoke_test.go`, `cmd/engram/testdata/model-stub.txt`
- Modify: `internal/cli/embed.go` (sharedEmbedder → bridge; delete `modelCacheDir`), `internal/cli/targets.go` (wire bridge), `internal/cli/export_test.go` (bridge export), cmd Deps literal (one line, foundation's file)
- Create: `internal/cli/embed_bridge_test.go`

**Interfaces**

- Produces: `embed.Backend` — `OpenPipeline(ctx context.Context, modelDir string) (PipelineHandle, error)`; `embed.PipelineHandle` — `RunPipeline(ctx context.Context, inputs []string) (FeatureOutput, error)`, `Destroy() error`; `embed.FeatureOutput{ Embeddings [][]float32 }`; `embed.CacheFS` (exported rename of `cacheFS`, same 7 methods, Rename contract = `errors.Is(err, fs.ErrExist)`); `embed.BundledModelFS() stdembed.FS`; `embed.BundledModelDir = "assets/model"`; new constructor signatures `NewBundledHugotEmbedder(ctx, backend Backend, cfs CacheFS, cacheDir string)`, `NewHugotEmbedderFromDir(ctx, backend Backend, modelDir, modelID string)`, `NewHugotEmbedderFromFS(ctx, backend Backend, cfs CacheFS, modelFS stdembed.FS, modelDir, modelID, cacheDir string)`, `NewLazyEmbedder(backend Backend, cfs CacheFS, cacheDir string)`; cmd-side `newHugotBackend() hugotBackend`, `osCacheFS`, `newProductionEmbedder() *embed.LazyEmbedder`; cli-side `wireSharedEmbedder(embed.Embedder)`.
- Consumes: `cli.CacheDirFromHome(home, modelID string, getenv func(string) string) string` (targets.go:56, unchanged); foundation's `cli.Deps.Embed embed.Embedder` field.

**Steps**

- [ ] 1. **RED — new cmd adapter integration tests first.** Create `cmd/engram/testdata/model-stub.txt` containing `stub model payload` (one line). Create `cmd/engram/os_cachefs_test.go`:

```go
package main

import (
	stdembed "embed"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

//go:embed testdata
var testModelFS stdembed.FS

// errBackendUnused aborts embedder construction after extraction so
// extraction-only behavior is assertable without a Hugot runtime.
var errBackendUnused = errors.New("backend intentionally failing")

// TestOsCacheFS_Methods exercises each osCacheFS method directly so the
// success paths and error wraps are covered on real disk.
func TestOsCacheFS_Methods(t *testing.T) {
	t.Parallel()

	t.Run("MkdirAll creates nested dirs", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := osCacheFS{}
		path := filepath.Join(t.TempDir(), "a", "b", "c")
		g.Expect(cfs.MkdirAll(path)).To(Succeed())
		_, err := os.Stat(path)
		g.Expect(err).NotTo(HaveOccurred())
	})

	t.Run("MkdirAll on non-writable parent fails", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		g.Expect(osCacheFS{}.MkdirAll("/no/such/root/path")).NotTo(Succeed())
	})

	t.Run("MkdirTemp creates a temp dir", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		tmp, err := osCacheFS{}.MkdirTemp(t.TempDir(), ".tmp-test-*")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(tmp).To(ContainSubstring(".tmp-test-"))
	})

	t.Run("MkdirTemp under non-existent parent fails", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		_, err := osCacheFS{}.MkdirTemp("/no/such/path", ".tmp-*")
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("WriteFile writes and RemoveAll cleans up", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := osCacheFS{}
		dir := t.TempDir()
		path := filepath.Join(dir, "test.bin")

		g.Expect(cfs.WriteFile(path, []byte("payload"))).To(Succeed())
		g.Expect(cfs.RemoveAll(dir)).To(Succeed())

		_, err := os.Stat(dir)
		g.Expect(os.IsNotExist(err)).To(BeTrue())
	})

	t.Run("WriteFile to non-existent dir fails", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		g.Expect(osCacheFS{}.WriteFile("/no/such/dir/x.bin", []byte("x"))).NotTo(Succeed())
	})

	t.Run("WriteSentinel writes .complete file", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		dir := t.TempDir()
		g.Expect(osCacheFS{}.WriteSentinel(dir)).To(Succeed())

		_, err := os.Stat(filepath.Join(dir, sentinelFileName))
		g.Expect(err).NotTo(HaveOccurred())
	})

	t.Run("WriteSentinel to non-existent dir fails", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		g.Expect(osCacheFS{}.WriteSentinel("/no/such/dir")).NotTo(Succeed())
	})

	t.Run("StatSentinel false when missing, true when present", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		cfs := osCacheFS{}
		dir := t.TempDir()

		present, err := cfs.StatSentinel(dir)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(present).To(BeFalse())

		g.Expect(os.WriteFile(filepath.Join(dir, sentinelFileName), []byte{}, 0o600)).To(Succeed())

		present, err = cfs.StatSentinel(dir)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(present).To(BeTrue())
	})

	t.Run("Rename moves dir atomically", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		parent := t.TempDir()
		src := filepath.Join(parent, "src")
		dst := filepath.Join(parent, "dst")

		g.Expect(os.Mkdir(src, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(src, "f.txt"), []byte("hi"), 0o600)).To(Succeed())

		g.Expect(osCacheFS{}.Rename(src, dst)).To(Succeed())

		_, srcErr := os.Stat(src)
		g.Expect(os.IsNotExist(srcErr)).To(BeTrue())

		_, dstErr := os.Stat(dst)
		g.Expect(dstErr).NotTo(HaveOccurred())
	})

	t.Run("Rename over non-existent path fails wrapped", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		g.Expect(osCacheFS{}.Rename("/no/such/src", "/no/such/dst")).NotTo(Succeed())
	})

	t.Run("Rename onto populated dir satisfies fs.ErrExist contract", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		parent := t.TempDir()
		src := filepath.Join(parent, "src")
		dst := filepath.Join(parent, "dst")
		g.Expect(os.Mkdir(src, 0o755)).To(Succeed())
		g.Expect(os.Mkdir(dst, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(dst, "f.txt"), []byte("hi"), 0o600)).To(Succeed())

		err := osCacheFS{}.Rename(src, dst)
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, fs.ErrExist)).To(BeTrue(),
			"CacheFS.Rename contract: destination-exists must satisfy errors.Is(err, fs.ErrExist)")
	})
}

// TestRenameIsExist covers the classification helper for the LinkError /
// string-match branches (relocated from internal/embed isExistErr tests).
func TestRenameIsExist(t *testing.T) {
	t.Parallel()

	t.Run("nil is not an exist error", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		g.Expect(renameIsExist(nil)).To(BeFalse())
	})

	t.Run("os.ErrExist is recognized", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		g.Expect(renameIsExist(os.ErrExist)).To(BeTrue())
	})

	t.Run("LinkError with ErrExist is recognized", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		linkErr := &os.LinkError{Op: "rename", Old: "a", New: "b", Err: os.ErrExist}
		g.Expect(renameIsExist(linkErr)).To(BeTrue())
	})

	t.Run("LinkError with 'directory not empty' string is recognized", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		linkErr := &os.LinkError{Op: "rename", Old: "a", New: "b",
			Err: errStringError("directory not empty"),
		}
		g.Expect(renameIsExist(linkErr)).To(BeTrue())
	})

	t.Run("unrelated error is not an exist error", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		linkErr := &os.LinkError{Op: "rename", Old: "a", New: "b", Err: os.ErrPermission}
		g.Expect(renameIsExist(linkErr)).To(BeFalse())
	})
}

// TestOsCacheFS_ExtractToCacheRealOS drives the internal extraction logic
// through the production adapter on real disk: first call extracts and
// stamps the sentinel; second call reuses without re-extracting. The
// injected backend fails so no Hugot runtime is needed (extraction happens
// before the backend opens).
func TestOsCacheFS_ExtractToCacheRealOS(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cacheDir := filepath.Join(t.TempDir(), "models", "stub@1")

	_, err := embed.NewHugotEmbedderFromFS(
		t.Context(), failingBackend{}, osCacheFS{}, testModelFS, "testdata", "stub@1", cacheDir)
	g.Expect(err).To(MatchError(errBackendUnused))

	_, sentinelErr := os.Stat(filepath.Join(cacheDir, sentinelFileName))
	g.Expect(sentinelErr).NotTo(HaveOccurred(), ".complete sentinel must be written after first extraction")

	entries1, readErr1 := os.ReadDir(cacheDir)
	g.Expect(readErr1).NotTo(HaveOccurred())

	fileCount1 := len(entries1)
	g.Expect(fileCount1).To(BeNumerically(">", 1), "cache dir must contain model files + sentinel")

	_, err = embed.NewHugotEmbedderFromFS(
		t.Context(), failingBackend{}, osCacheFS{}, testModelFS, "testdata", "stub@1", cacheDir)
	g.Expect(err).To(MatchError(errBackendUnused))

	entries2, readErr2 := os.ReadDir(cacheDir)
	g.Expect(readErr2).NotTo(HaveOccurred())
	g.Expect(entries2).To(HaveLen(fileCount1),
		"second call must not add/modify files — cache reused as-is")
}

// errStringError is a minimal error whose message matches the
// "directory not empty" string.
type errStringError string

func (e errStringError) Error() string { return string(e) }

// failingBackend implements embed.Backend and always refuses to open.
type failingBackend struct{}

func (failingBackend) OpenPipeline(context.Context, string) (embed.PipelineHandle, error) {
	return nil, errBackendUnused
}
```

Create `cmd/engram/hugot_test.go` (relocated from `internal/embed/production_hugot_test.go`, now driving the unexported cmd types directly):

```go
package main

import (
	"context"
	"errors"
	"testing"

	"github.com/knights-analytics/hugot/pipelines"
	. "github.com/onsi/gomega"
)

// TestOpenPipeline_SessionFailPropagates exercises the first error branch
// of hugotBackend.OpenPipeline: openSession returns an error.
func TestOpenPipeline_SessionFailPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessionErr := errors.New("session blocked")

	backend := hugotBackend{
		openSession: func(context.Context) (hugotSession, error) {
			return nil, sessionErr
		},
		openPipeline: func(hugotSession, string) (hugotRawPipeline, error) {
			return nil, errors.New("openPipeline must not be called when openSession fails")
		},
	}

	_, err := backend.OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).To(MatchError(ContainSubstring("hugot session")))
	g.Expect(err).To(MatchError(ContainSubstring("session blocked")))
}

// TestOpenPipeline_PipelineFailDestroysSession exercises the second error
// branch: openPipeline fails and the session's Destroy is called.
func TestOpenPipeline_PipelineFailDestroysSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	pipeErr := errors.New("pipeline blocked")
	session := &fakeSession{}

	backend := hugotBackend{
		openSession: func(context.Context) (hugotSession, error) {
			return session, nil
		},
		openPipeline: func(hugotSession, string) (hugotRawPipeline, error) {
			return nil, pipeErr
		},
	}

	_, err := backend.OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).To(MatchError(ContainSubstring("hugot pipeline")))
	g.Expect(err).To(MatchError(ContainSubstring("pipeline blocked")))
	g.Expect(session.destroyCalls).
		To(Equal(1), "session.Destroy must be called on pipeline failure")
}

// TestHugotPipeline_DestroyErrorPropagates exercises the error branch of
// hugotPipeline.Destroy.
func TestHugotPipeline_DestroyErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	destroyErr := errors.New("destroy blocked")
	handle := &hugotPipeline{session: &fakeSession{destroyErr: destroyErr}}

	err := handle.Destroy()
	g.Expect(err).To(MatchError(ContainSubstring("hugot session destroy")))
	g.Expect(err).To(MatchError(ContainSubstring("destroy blocked")))
}

// TestHugotPipeline_RunPipelineErrorPropagates exercises the error branch
// of hugotPipeline.RunPipeline.
func TestHugotPipeline_RunPipelineErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runErr := errors.New("run blocked")
	handle := &hugotPipeline{
		session:  &fakeSession{},
		pipeline: &fakeRawPipeline{runErr: runErr},
	}

	_, err := handle.RunPipeline(t.Context(), []string{"hello"})
	g.Expect(err).To(MatchError(ContainSubstring("hugot run")))
	g.Expect(err).To(MatchError(ContainSubstring("run blocked")))
}

// TestHugotPipeline_RunPipelineHappyPath maps the raw output shape into
// embed.FeatureOutput.
func TestHugotPipeline_RunPipelineHappyPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	handle := &hugotPipeline{
		session:  &fakeSession{},
		pipeline: &fakeRawPipeline{},
	}

	out, err := handle.RunPipeline(t.Context(), []string{"hello"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(out.Embeddings).To(Equal([][]float32{{1, 2}}))
}

type fakeRawPipeline struct {
	runErr error
}

func (f *fakeRawPipeline) RunPipeline(
	_ context.Context,
	_ []string,
) (*pipelines.FeatureExtractionOutput, error) {
	if f.runErr != nil {
		return nil, f.runErr
	}

	return &pipelines.FeatureExtractionOutput{Embeddings: [][]float32{{1, 2}}}, nil
}

type fakeSession struct {
	destroyCalls int
	destroyErr   error
}

func (s *fakeSession) Destroy() error {
	s.destroyCalls++

	return s.destroyErr
}
```

Create `cmd/engram/bundled_smoke_test.go` (relocated `TestBundledHugotEmbedder_Smoke` from `internal/embed/hugot_test.go:16` and `TestNewHugotEmbedderFromFS_InvalidModelFails` from `production_hugot_test.go:18`):

```go
package main

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// TestBundledEmbedder_Smoke exercises the full production wiring
// end-to-end: real hugot backend, real cache FS, bundled model assets.
// Skipped under -short because it unpacks the ~90MB ONNX.
func TestBundledEmbedder_Smoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bundled-embedder smoke test under -short")
	}

	t.Parallel()

	g := NewWithT(t)

	embedder, err := embed.NewBundledHugotEmbedder(
		t.Context(), newHugotBackend(), osCacheFS{}, filepath.Join(t.TempDir(), "model-cache"))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	defer func() {
		_ = embedder.Close()
	}()

	const expectedDims = 384

	g.Expect(embedder.ModelID()).To(Equal(embed.BundledModelID))
	g.Expect(embedder.Dims()).To(Equal(expectedDims))

	vec, embErr := embedder.Embed(t.Context(), "hello world")
	g.Expect(embErr).NotTo(HaveOccurred())
	g.Expect(vec).To(HaveLen(expectedDims))

	const longLen = 4000

	longText := make([]byte, longLen)
	for i := range longText {
		longText[i] = 'a' + byte(i%26)
	}

	vec2, embErr2 := embedder.Embed(t.Context(), string(longText))
	g.Expect(embErr2).NotTo(HaveOccurred())
	g.Expect(vec2).To(HaveLen(expectedDims))
}

// TestHugotRejectsInvalidModelDir exercises the embedder-construction error
// branch of NewHugotEmbedderFromFS: extraction succeeds (files exist) but
// Hugot rejects the directory because it has no valid model.onnx.
func TestHugotRejectsInvalidModelDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cacheDir := filepath.Join(t.TempDir(), "model-cache")
	_, err := embed.NewHugotEmbedderFromFS(
		t.Context(), newHugotBackend(), osCacheFS{}, testModelFS, "testdata", "fake@1", cacheDir)
	g.Expect(err).To(HaveOccurred())
}
```

Run: `targ test` — expected RED (compile failures: `osCacheFS`, `hugotBackend`, `sentinelFileName`, `renameIsExist`, new `embed.*` signatures don't exist yet).

- [ ] 2. **GREEN — create `cmd/engram/hugot.go`** (the full moved backend file):

```go
// Production hugot + model-cache adapters. This is the only place in the
// repo (outside _test files) that imports hugot or touches os for the
// embedder path — internal/embed depends only on the embed.Backend and
// embed.CacheFS seams (#700).
package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/pipelines"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// unexported constants.
const (
	// sentinelFileName marks a fully extracted model cache dir.
	sentinelFileName = ".complete"

	cacheDirPerm  = 0o755
	cacheFilePerm = 0o600
)

// hugotBackend wires the real hugot.NewGoSession + NewPipeline behind
// embed.Backend. The openSession and openPipeline fields are injectable so
// each error branch of OpenPipeline is testable without a Hugot runtime.
type hugotBackend struct {
	openSession  func(ctx context.Context) (hugotSession, error)
	openPipeline func(session hugotSession, modelDir string) (hugotRawPipeline, error)
}

// OpenPipeline opens a session, then a feature-extraction pipeline on it,
// destroying the session if pipeline creation fails.
func (b hugotBackend) OpenPipeline(
	ctx context.Context, modelDir string,
) (embed.PipelineHandle, error) {
	session, err := b.openSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("hugot session: %w", err)
	}

	pipeline, pipeErr := b.openPipeline(session, modelDir)
	if pipeErr != nil {
		_ = session.Destroy()

		return nil, fmt.Errorf("hugot pipeline: %w", pipeErr)
	}

	return &hugotPipeline{session: session, pipeline: pipeline}, nil
}

// hugotPipeline pairs a Hugot pipeline with the session that owns it so
// Destroy releases both together.
type hugotPipeline struct {
	session  hugotSession
	pipeline hugotRawPipeline
}

// Destroy releases the owning session (which owns the pipeline).
func (p *hugotPipeline) Destroy() error {
	err := p.session.Destroy()
	if err != nil {
		return fmt.Errorf("hugot session destroy: %w", err)
	}

	return nil
}

// RunPipeline runs the model and maps the raw output into the
// runtime-neutral embed.FeatureOutput shape.
func (p *hugotPipeline) RunPipeline(
	ctx context.Context, inputs []string,
) (embed.FeatureOutput, error) {
	out, err := p.pipeline.RunPipeline(ctx, inputs)
	if err != nil {
		return embed.FeatureOutput{}, fmt.Errorf("hugot run: %w", err)
	}

	return embed.FeatureOutput{Embeddings: out.Embeddings}, nil
}

// hugotRawPipeline is the minimal Hugot pipeline surface, extracted so
// tests can inject a failing implementation without a real Hugot session.
type hugotRawPipeline interface {
	RunPipeline(ctx context.Context, inputs []string) (*pipelines.FeatureExtractionOutput, error)
}

// hugotSession is the minimal Hugot session surface needed: cleanup on
// pipeline-creation failure and on normal close.
type hugotSession interface {
	Destroy() error
}

// osCacheFS is the os-backed embed.CacheFS used to extract the bundled
// model into its persistent XDG cache dir. Rename translates the platform's
// destination-exists quirks into the fs.ErrExist contract internal/embed
// classifies against.
type osCacheFS struct{}

// MkdirAll ensures the parent directory of the cache dir exists.
func (osCacheFS) MkdirAll(path string) error {
	err := os.MkdirAll(path, cacheDirPerm)
	if err != nil {
		return fmt.Errorf("mkdir all: %w", err)
	}

	return nil
}

// MkdirTemp creates a temporary sibling dir for atomic extraction.
func (osCacheFS) MkdirTemp(parent, pattern string) (string, error) {
	tmp, err := os.MkdirTemp(parent, pattern)
	if err != nil {
		return "", fmt.Errorf("mkdir temp: %w", err)
	}

	return tmp, nil
}

// RemoveAll deletes path. os.RemoveAll's error contract is already
// caller-friendly (nil on missing paths); wrapping adds noise.
func (osCacheFS) RemoveAll(path string) error {
	return os.RemoveAll(path) //nolint:wrapcheck // thin adapter; nil on missing paths
}

// Rename renames src to dst atomically. When the destination already
// exists (including macOS ENOTEMPTY for dir-over-dir renames), the returned
// error satisfies errors.Is(err, fs.ErrExist) per the embed.CacheFS contract.
func (osCacheFS) Rename(src, dst string) error {
	err := os.Rename(src, dst)
	if err != nil {
		if renameIsExist(err) {
			return fmt.Errorf("%w: %w", fs.ErrExist, err)
		}

		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

// StatSentinel reports whether cacheDir already has a .complete sentinel.
func (osCacheFS) StatSentinel(cacheDir string) (bool, error) {
	_, err := os.Stat(filepath.Join(cacheDir, sentinelFileName))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("stat sentinel: %w", err)
	}

	return true, nil
}

// WriteFile writes data to path (copies model files into the temp dir).
func (osCacheFS) WriteFile(path string, data []byte) error {
	err := os.WriteFile(path, data, cacheFilePerm)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// WriteSentinel writes the .complete sentinel into tmpDir.
func (osCacheFS) WriteSentinel(tmpDir string) error {
	err := os.WriteFile(filepath.Join(tmpDir, sentinelFileName), []byte{}, cacheFilePerm)
	if err != nil {
		return fmt.Errorf("write sentinel: %w", err)
	}

	return nil
}

// newHugotBackend wires the real hugot.NewGoSession + NewPipeline.
func newHugotBackend() hugotBackend {
	return hugotBackend{
		openSession: func(ctx context.Context) (hugotSession, error) {
			return hugot.NewGoSession(ctx)
		},
		// openPipeline type-asserts session to *hugot.Session because
		// openSession always returns *hugot.Session in production; test code
		// injects its own openPipeline that never performs this assertion.
		openPipeline: func(session hugotSession, modelDir string) (hugotRawPipeline, error) {
			config := hugot.FeatureExtractionConfig{
				ModelPath:    modelDir,
				Name:         "engram-embed",
				OnnxFilename: "model.onnx",
			}

			//nolint:forcetypeassert // production invariant
			return hugot.NewPipeline(
				session.(*hugot.Session),
				config,
			)
		},
	}
}

// newProductionEmbedder wires the bundled lazy embedder: real hugot
// backend, real cache FS, XDG-keyed model cache dir. Env + home reads
// happen here, at the impure edge.
func newProductionEmbedder() *embed.LazyEmbedder {
	home, _ := os.UserHomeDir()
	cacheDir := cli.CacheDirFromHome(home, embed.BundledModelID, os.Getenv)

	return embed.NewLazyEmbedder(newHugotBackend(), osCacheFS{}, cacheDir)
}

// renameIsExist reports whether err (from os.Rename) is a "destination
// exists" error. On macOS, renaming over an existing dir returns ENOTEMPTY
// ("directory not empty") rather than EEXIST, so the string is also checked.
func renameIsExist(err error) bool {
	if errors.Is(err, os.ErrExist) {
		return true
	}

	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		if errors.Is(linkErr.Err, os.ErrExist) {
			return true
		}

		errStr := linkErr.Err.Error()
		if errStr == "file exists" || errStr == "directory not empty" {
			return true
		}
	}

	return false
}
```

- [ ] 3. **Rewrite `internal/embed/hugot.go`** — hugot and os imports leave internal; `Backend`/`PipelineHandle`/`FeatureOutput` exported; constructors take injected seams; `buildEmbedder` folds into `NewHugotEmbedderFromDir`; dead `tempFS`/`productionTempFS`/`unpackModelToTemp` deleted. Full replacement:

```go
package embed

import (
	"context"
	stdembed "embed"
	"errors"
	"fmt"
	"sync"
)

// Exported constants.
const (
	BundledModelID = "minilm-l6-v2@384"
	// BundledModelDir is the directory inside BundledModelFS holding the
	// bundled model files.
	BundledModelDir = "assets/model"
)

// Exported variables.
var (
	ErrBundledModelUnavailable = errors.New(
		"bundled model missing or empty — rebuild the binary with the model in place, " +
			"or set ENGRAM_MODEL_PATH to a directory containing model.onnx",
	)
	ErrHugotEmbedEmpty = errors.New("hugot embed: empty result")
	ErrHugotProbeEmpty = errors.New("hugot probe returned no embedding")
)

// Backend opens an embedding pipeline for an on-disk model directory. The
// production implementation wraps Hugot and lives in cmd/engram so the
// hugot import stays out of internal/ (#700); tests inject fakes to
// exercise every constructor branch.
type Backend interface {
	OpenPipeline(ctx context.Context, modelDir string) (PipelineHandle, error)
}

// FeatureOutput is the embedding shape returned by
// PipelineHandle.RunPipeline, mirrored here so implementations don't leak
// their runtime's own output types.
type FeatureOutput struct {
	Embeddings [][]float32
}

// HugotEmbedder wraps an embedding pipeline. Safe for concurrent use — the
// production pipeline runs the model under its own lock.
type HugotEmbedder struct {
	pipeline interface {
		RunPipeline(ctx context.Context, inputs []string) (out FeatureOutput, err error)
	}
	modelID string
	dims    int

	// Capture the close logic at construction time so the destroy chain
	// stays encapsulated even if the backend's session type changes.
	close func() error
}

// PipelineHandle is the runtime surface of an opened pipeline plus its
// owning session; Destroy releases both together.
type PipelineHandle interface {
	RunPipeline(ctx context.Context, inputs []string) (FeatureOutput, error)
	Destroy() error
}

// BundledModelFS returns the go:embed-ed bundled model assets, rooted at
// BundledModelDir. Exposed so cmd/engram (and its integration tests) can
// hand the bundled assets to the injectable constructors.
func BundledModelFS() stdembed.FS { return bundledModel }

// NewBundledHugotEmbedder is the production constructor: bundled assets FS,
// fixed model directory, fixed model ID, and caller-supplied backend, cache
// FS, and cache dir. The cache dir is the XDG-keyed path where the model is
// extracted once and reused across all subsequent invocations.
func NewBundledHugotEmbedder(
	ctx context.Context, backend Backend, cfs CacheFS, cacheDir string,
) (*HugotEmbedder, error) {
	return NewHugotEmbedderFromFS(
		ctx, backend, cfs, bundledModel, BundledModelDir, BundledModelID, cacheDir)
}

// NewHugotEmbedderFromDir constructs an embedder reading the model from a
// directory on disk via the injected backend, probing once to learn the
// embedding dimensionality. Every error branch (pipeline open, probe run,
// empty probe) is unit-testable with a fake Backend.
func NewHugotEmbedderFromDir(
	ctx context.Context, backend Backend, modelDir, modelID string,
) (*HugotEmbedder, error) {
	handle, openErr := backend.OpenPipeline(ctx, modelDir)
	if openErr != nil {
		return nil, openErr
	}

	probe, probeErr := handle.RunPipeline(ctx, []string{"probe"})
	if probeErr != nil {
		_ = handle.Destroy()

		return nil, probeErr
	}

	if len(probe.Embeddings) == 0 || len(probe.Embeddings[0]) == 0 {
		_ = handle.Destroy()

		return nil, ErrHugotProbeEmpty
	}

	runner := &pipelineRunner{run: handle.RunPipeline}

	return &HugotEmbedder{
		pipeline: runner,
		modelID:  modelID,
		dims:     len(probe.Embeddings[0]),
		close:    handle.Destroy,
	}, nil
}

// NewHugotEmbedderFromFS constructs an embedder from any stdembed.FS rooted
// at modelDir. cacheDir is the stable directory where the model is
// extracted once via cfs and reused across invocations (XDG-keyed). Tests
// pass an empty FS to verify UAT 10's clear-error path.
func NewHugotEmbedderFromFS(
	ctx context.Context, backend Backend, cfs CacheFS,
	modelFS stdembed.FS, modelDir, modelID, cacheDir string,
) (*HugotEmbedder, error) {
	dir, extractErr := extractToCache(cfs, modelFS, modelDir, cacheDir)
	if extractErr != nil {
		return nil, extractErr
	}

	return NewHugotEmbedderFromDir(ctx, backend, dir, modelID)
}

// Close releases the underlying session. Safe to call multiple times. The
// model cache dir is NOT removed — it is a shared, persistent cache reused
// across all engram invocations.
func (h *HugotEmbedder) Close() error {
	if h.close != nil {
		err := h.close()
		h.close = nil

		return err
	}

	return nil
}

// Dims reports the embedding dimensionality.
func (h *HugotEmbedder) Dims() int { return h.dims }

// Embed runs the pipeline on text (truncated to fit the model's context
// window) and returns the resulting vector.
//
// The char guard assumes prose density; code-dense text can still exceed the
// model's 512-token positional limit within the char limit (observed: 1500
// chars of transcript tokenizing to 538 tokens, panicking graph compilation).
// On failure the input is halved and retried until it succeeds or bottoms out,
// so a single dense chunk degrades to a shorter prefix instead of failing the
// whole ingest.
func (h *HugotEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if len(text) > hugotInputCharLimit {
		text = text[:hugotInputCharLimit]
	}

	for {
		out, err := h.pipeline.RunPipeline(ctx, []string{text})
		if err != nil {
			if len(text) >= hugotRetryFloorChars {
				text = text[:len(text)/2]

				continue
			}

			return nil, err
		}

		if len(out.Embeddings) == 0 {
			return nil, ErrHugotEmbedEmpty
		}

		return out.Embeddings[0], nil
	}
}

// ModelID reports the configured model identifier.
func (h *HugotEmbedder) ModelID() string { return h.modelID }

// LazyEmbedder defers construction of an embedder until first use so
// commands that don't need it (help, update, transcript) don't pay the
// model-unpack cost or die if model loading fails. The construction is
// factory-injected so tests can drive both the success and failure
// init paths without a real backend.
type LazyEmbedder struct {
	once    sync.Once
	factory func() (*HugotEmbedder, error)
	emb     *HugotEmbedder
	initErr error
}

// NewLazyEmbedder returns a wrapper around NewBundledHugotEmbedder that
// extracts the bundled model to cacheDir at most once (on first Embed /
// ModelID / Dims call) using the injected backend and cache FS. cacheDir
// should be the XDG-keyed stable cache path for the model, e.g.
// $XDG_CACHE_HOME/engram/models/<model_id>/.
func NewLazyEmbedder(backend Backend, cfs CacheFS, cacheDir string) *LazyEmbedder {
	return &LazyEmbedder{
		// Background context: the lazy init runs at most once per process;
		// a request-scoped context could cancel construction partway through
		// model extraction and leave a partial temp dir.
		factory: func() (*HugotEmbedder, error) {
			return NewBundledHugotEmbedder(context.Background(), backend, cfs, cacheDir)
		},
	}
}

// Dims lazily constructs the embedder, then delegates. Returns 0 when
// construction failed; callers should detect via an Embed error.
func (l *LazyEmbedder) Dims() int {
	l.init()

	if l.initErr != nil {
		return 0
	}

	return l.emb.Dims()
}

// Embed lazily constructs the embedder, then delegates.
func (l *LazyEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	l.init()

	if l.initErr != nil {
		return nil, fmt.Errorf("embedder unavailable: %w", l.initErr)
	}

	return l.emb.Embed(ctx, text)
}

// ModelID lazily constructs the embedder, then delegates. Returns the
// bundled model id when construction has not been attempted yet so
// status-style callers can avoid paying the unpack cost.
func (l *LazyEmbedder) ModelID() string {
	if l.emb == nil && l.initErr == nil {
		return BundledModelID
	}

	if l.initErr != nil {
		return BundledModelID
	}

	return l.emb.ModelID()
}

// init runs at most once per LazyEmbedder via sync.Once. The factory
// is provided at construction time so tests can drive both success and
// failure init paths without a real backend.
func (l *LazyEmbedder) init() {
	l.once.Do(func() {
		l.emb, l.initErr = l.factory()
	})
}

// unexported constants.
const (
	hugotInputCharLimit = 1500
	// hugotRetryFloorChars stops the over-length halving retry: below this
	// the failure is not a token-budget problem and must surface.
	hugotRetryFloorChars = 100
)

//go:embed assets/model/*
var bundledModel stdembed.FS

// pipelineRunner adapts a PipelineHandle's run function to the small
// interface HugotEmbedder depends on; isolating the dependency makes
// backend version bumps a surgical edit instead of a sweep.
type pipelineRunner struct {
	run func(ctx context.Context, inputs []string) (FeatureOutput, error)
}

func (p *pipelineRunner) RunPipeline(ctx context.Context, inputs []string) (FeatureOutput, error) {
	return p.run(ctx, inputs)
}
```

- [ ] 4. **Rewrite `internal/embed/cache.go`** — `CacheFS` exported, os import gone, exist-classification becomes the `fs.ErrExist` contract. Full replacement:

```go
package embed

import (
	stdembed "embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// CacheFS is the I/O surface extractToCache depends on. The production
// implementation (os-backed) lives in cmd/engram; tests inject fakes to
// exercise every branch without touching the real disk.
type CacheFS interface {
	// StatSentinel reports whether the cache dir already has a .complete sentinel.
	StatSentinel(cacheDir string) (bool, error)
	// MkdirAll ensures the parent directory of the cache dir exists.
	MkdirAll(path string) error
	// MkdirTemp creates a temporary directory sibling of cacheDir for atomic extraction.
	MkdirTemp(parent, pattern string) (string, error)
	// WriteFile writes data to path (used to copy model files into the temp dir).
	WriteFile(path string, data []byte) error
	// WriteSentinel writes the .complete sentinel into tmpDir.
	WriteSentinel(tmpDir string) error
	// Rename renames src to dst atomically. When dst already exists
	// (concurrent-race scenario), the returned error MUST satisfy
	// errors.Is(err, fs.ErrExist) — implementations translate platform
	// quirks (e.g. macOS ENOTEMPTY on dir-over-dir renames) before returning.
	Rename(src, dst string) error
	// RemoveAll deletes path recursively (used to clean up temp on rename race).
	RemoveAll(path string) error
}

// commitCache atomically renames tmp into cacheDir. If the rename fails with
// a destination-exists error (concurrent-process race), it re-checks the
// sentinel: if the winner completed the cache, discards tmp and returns
// cacheDir. Otherwise returns the rename error.
func commitCache(cfs CacheFS, tmp, cacheDir string) (string, error) {
	renameErr := cfs.Rename(tmp, cacheDir)
	if renameErr == nil {
		return cacheDir, nil
	}

	// If the rename failed because the destination exists, check whether
	// another process just won the race and completed the cache. If so,
	// discard our temp.
	if errors.Is(renameErr, fs.ErrExist) {
		complete, statErr := cfs.StatSentinel(cacheDir)
		if statErr == nil && complete {
			_ = cfs.RemoveAll(tmp)

			return cacheDir, nil
		}
	}

	// True rename failure (or sentinel absent after race check).
	_ = cfs.RemoveAll(tmp)

	return "", fmt.Errorf("cache rename: %w", renameErr)
}

// copyModelFiles copies every non-directory entry from modelFS/modelDir into tmpDir.
func copyModelFiles(cfs CacheFS, modelFS stdembed.FS, modelDir, tmpDir string) error {
	entries, _ := modelFS.ReadDir(modelDir) // already validated by caller

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, readErr := modelFS.ReadFile(filepath.Join(modelDir, entry.Name()))
		if readErr != nil {
			return fmt.Errorf("read embedded %s: %w", entry.Name(), readErr)
		}

		writeErr := cfs.WriteFile(filepath.Join(tmpDir, entry.Name()), data)
		if writeErr != nil {
			return fmt.Errorf("unpack %s: %w", entry.Name(), writeErr)
		}
	}

	return nil
}

// extractToCache ensures that <cacheDir> contains the fully extracted model
// and the .complete sentinel. On first call it extracts into a sibling temp
// dir and atomically renames it into place. On subsequent calls (sentinel
// present) it returns immediately without any I/O. A concurrent-process race
// (rename fails because another process just won) is handled by discarding
// the temp dir and using the pre-existing complete cache.
func extractToCache(
	cfs CacheFS,
	modelFS stdembed.FS,
	modelDir string,
	cacheDir string,
) (string, error) {
	// Fast path: already extracted.
	ok, statErr := cfs.StatSentinel(cacheDir)
	if statErr != nil {
		return "", statErr
	}

	if ok {
		return cacheDir, nil
	}

	return populateCache(cfs, modelFS, modelDir, cacheDir)
}

// populateCache handles the slow path of extractToCache: verifying the model
// FS, creating a sibling temp dir, copying model files, and atomically renaming
// the temp dir into place.
func populateCache(
	cfs CacheFS,
	modelFS stdembed.FS,
	modelDir string,
	cacheDir string,
) (string, error) {
	// Verify the model FS has files before creating any directories.
	entries, dirErr := modelFS.ReadDir(modelDir)
	if dirErr != nil || len(entries) == 0 {
		return "", fmt.Errorf("%w: dir %s (underlying: %w)",
			ErrBundledModelUnavailable, modelDir, dirErr,
		)
	}

	// Ensure the parent directory exists.
	parent := filepath.Dir(cacheDir)

	mkdirErr := cfs.MkdirAll(parent)
	if mkdirErr != nil {
		return "", fmt.Errorf("cache parent dir: %w", mkdirErr)
	}

	// Extract into a sibling temp dir so the rename is atomic.
	tmp, tmpErr := cfs.MkdirTemp(parent, ".tmp-engram-model-*")
	if tmpErr != nil {
		return "", fmt.Errorf("cache temp dir: %w", tmpErr)
	}

	copyErr := copyModelFiles(cfs, modelFS, modelDir, tmp)
	if copyErr != nil {
		_ = cfs.RemoveAll(tmp)

		return "", copyErr
	}

	sentinelErr := cfs.WriteSentinel(tmp)
	if sentinelErr != nil {
		_ = cfs.RemoveAll(tmp)

		return "", fmt.Errorf("cache sentinel: %w", sentinelErr)
	}

	return commitCache(cfs, tmp, cacheDir)
}
```

- [ ] 5. **Adapt internal/embed tests.** Delete files: `internal/embed/production_cache_test.go`, `internal/embed/production_hugot_test.go`, `internal/embed/unpack_test.go`, `internal/embed/tempfs_test.go`. Rewrite `internal/embed/export_test.go` (full replacement):

```go
package embed

import (
	"context"
	stdembed "embed"
)

// Exported variables.
var (
	ExportNotExist = notExist
)

// ExportExtractToCache exposes the unexported extractToCache helper so
// tests can exercise the sentinel / race / error branches with a fake
// CacheFS without touching the real disk.
func ExportExtractToCache(
	cfs CacheFS,
	modelFS stdembed.FS,
	modelDir string,
	cacheDir string,
) (string, error) {
	return extractToCache(cfs, modelFS, modelDir, cacheDir)
}

// NewHugotEmbedderWithPipelineForTest constructs a HugotEmbedder around
// a caller-supplied pipeline implementation. Tests use this to exercise
// the Embed/Close error branches without depending on a real backend.
func NewHugotEmbedderWithPipelineForTest(
	modelID string, dims int,
	runFn func(text string) ([][]float32, error),
	closeFn func() error,
) *HugotEmbedder {
	runner := &pipelineRunner{
		run: func(_ context.Context, inputs []string) (FeatureOutput, error) {
			out, err := runFn(inputs[0])
			if err != nil {
				return FeatureOutput{}, err
			}

			return FeatureOutput{Embeddings: out}, nil
		},
	}

	return &HugotEmbedder{
		pipeline: runner,
		modelID:  modelID,
		dims:     dims,
		close:    closeFn,
	}
}

// NewLazyEmbedderWithFactoryForTest constructs a LazyEmbedder with a
// caller-supplied factory so tests can drive both init success and
// failure paths without a real backend.
func NewLazyEmbedderWithFactoryForTest(factory func() (*HugotEmbedder, error)) *LazyEmbedder {
	return &LazyEmbedder{factory: factory}
}

// SetCacheDirForTest is a no-op test helper for the Close-does-not-delete
// test. HugotEmbedder no longer holds a tmpDir field — Close only closes the
// backend session and never removes any directory. The function is kept for
// test readability; the test creates its own dir and verifies it survives.
func SetCacheDirForTest(_ *HugotEmbedder, _ string) {}
```

Then mechanical edits, all in `internal/embed/*_test.go`:
  - Everywhere: `embed.ExportHugotBackend` → `embed.Backend`; `embed.ExportHugotPipelineHandle` → `embed.PipelineHandle`; `embed.ExportFeatureOutput` → `embed.FeatureOutput`; `embed.ExportCacheFS` → `embed.CacheFS`; `embed.BuildEmbedderForTest(` → `embed.NewHugotEmbedderFromDir(` (identical argument lists — verified).
  - `cache_test.go`: (a) delete `TestExtractToCache_RealOS` (lines 118-149); (b) replace both race-fake errors at lines 54 and 107 — current: `renameErr: &os.LinkError{Op: "rename", Err: errors.New("directory not empty")}` — new: `renameErr: fmt.Errorf("%w: directory not empty", fs.ErrExist)` (add `"fmt"`, `"io/fs"` imports); (c) type-assertion line 212: `_ embed.ExportCacheFS = (*fakeCacheFS)(nil)` → `_ embed.CacheFS = (*fakeCacheFS)(nil)`; (d) move the `nonEmptyTestFS` declaration here from the deleted `unpack_test.go` (add `stdembed "embed"` import):

```go
//go:embed testdata/gen-reference.py
var nonEmptyTestFS stdembed.FS
```

  - `hugot_test.go`: delete `TestBundledHugotEmbedder_Smoke` (relocated to cmd, step 1); adapt T10 (fakes never reached — extraction fails first on the empty FS):

```go
func TestT10_MissingBundledModel_ClearError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cacheDir := filepath.Join(t.TempDir(), "model-cache")
	_, err := embed.NewHugotEmbedderFromFS(
		context.Background(), &fakeBackend{}, &fakeCacheFS{}, emptyFS, "assets/model", "x@1", cacheDir)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("ENGRAM_MODEL_PATH"))
}
```

  (`fakeBackend` lives in `buildembedder_test.go`, `fakeCacheFS` in `cache_test.go` — same `embed_test` package; zero-value `fakeCacheFS.StatSentinel` returns `(false, nil)` — verified.)
  - `embedder_fake_test.go`: `embed.NewLazyEmbedder(t.TempDir())` → `embed.NewLazyEmbedder(&fakeBackend{}, &fakeCacheFS{}, t.TempDir())`.

- [ ] 6. **Bridge `internal/cli/embed.go` + wire `Targets`.** In `internal/cli/embed.go` delete `modelCacheDir()` (lines 232-239) and replace the singleton block (lines 107-111):

Current:
```go
// unexported variables.
var (
	//nolint:gochecknoglobals // shared lazy singleton across CLI commands
	sharedEmbedder = embed.NewLazyEmbedder(modelCacheDir())
)
```

Replacement (add `"sync/atomic"` import):
```go
// unexported variables.
var (
	// errEmbedderUnwired reports an Embed call before Targets wired Deps.Embed.
	errEmbedderUnwired = errors.New("embedder not wired: cli.Targets(deps) has not run")

	// sharedEmbedderPtr holds the Deps-wired production embedder, stored by
	// wireSharedEmbedder (called from Targets). Atomic because tests build
	// Targets concurrently.
	// TRANSITIONAL (#700): deleted once every per-command deps constructor
	// takes Deps and reads d.Embed directly.
	//nolint:gochecknoglobals // transitional bridge, see comment
	sharedEmbedderPtr atomic.Pointer[embed.Embedder]

	// sharedEmbedder is the value legacy per-command constructors wire into
	// their deps structs; it forwards to the Targets-wired embedder.
	// TRANSITIONAL (#700) — same removal condition as sharedEmbedderPtr.
	//nolint:gochecknoglobals // transitional bridge, see comment
	sharedEmbedder embed.Embedder = bridgeEmbedder{ptr: &sharedEmbedderPtr}
)

// bridgeEmbedder forwards Embedder calls to the embedder wired by Targets.
// Pre-wiring fallbacks mirror LazyEmbedder's pre-init behavior: ModelID
// reports the bundled constant, Dims reports 0, Embed errors.
type bridgeEmbedder struct {
	ptr *atomic.Pointer[embed.Embedder]
}

// Dims delegates to the wired embedder; 0 before wiring.
func (b bridgeEmbedder) Dims() int {
	emb := b.load()
	if emb == nil {
		return 0
	}

	return emb.Dims()
}

// Embed delegates to the wired embedder; errors before wiring.
func (b bridgeEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	emb := b.load()
	if emb == nil {
		return nil, errEmbedderUnwired
	}

	return emb.Embed(ctx, text)
}

// ModelID delegates to the wired embedder; bundled constant before wiring
// (keeps status-style callers unpack-free, matching LazyEmbedder).
func (b bridgeEmbedder) ModelID() string {
	emb := b.load()
	if emb == nil {
		return embed.BundledModelID
	}

	return emb.ModelID()
}

// load returns the wired embedder or nil before wiring.
func (b bridgeEmbedder) load() embed.Embedder {
	ptr := b.ptr.Load()
	if ptr == nil || *ptr == nil {
		return nil
	}

	return *ptr
}

// wireSharedEmbedder points the transitional sharedEmbedder bridge at the
// Deps-wired embedder. Called by Targets.
func wireSharedEmbedder(embedder embed.Embedder) {
	sharedEmbedderPtr.Store(&embedder)
}
```

In `internal/cli/targets.go`, insert as the first statement of `Targets` (post-foundation shape `func Targets(deps Deps) []any` — coordinate with foundation task): `wireSharedEmbedder(deps.Embed)`. In cmd/engram's Deps literal (foundation's file) add: `Embed: newProductionEmbedder(),`.

- [ ] 7. **Bridge behavior tests (parallel-safe, no global state).** Add to `internal/cli/export_test.go`:

```go
// ExportNewBridgeEmbedder returns a fresh transitional shared-embedder
// bridge plus its wire function, backed by an isolated pointer so bridge
// behavior tests never touch the package-global embedder.
func ExportNewBridgeEmbedder() (embed.Embedder, func(embed.Embedder)) {
	ptr := &atomic.Pointer[embed.Embedder]{}
	bridge := bridgeEmbedder{ptr: ptr}
	wire := func(e embed.Embedder) { ptr.Store(&e) }

	return bridge, wire
}
```

Create `internal/cli/embed_bridge_test.go`:

```go
package cli_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestBridgeEmbedder_UnwiredFallbacks asserts the pre-wiring behavior
// mirrors LazyEmbedder's pre-init semantics.
func TestBridgeEmbedder_UnwiredFallbacks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bridge, _ := cli.ExportNewBridgeEmbedder()

	g.Expect(bridge.ModelID()).To(Equal(embed.BundledModelID))
	g.Expect(bridge.Dims()).To(Equal(0))

	_, err := bridge.Embed(context.Background(), "x")
	g.Expect(err).To(MatchError(ContainSubstring("embedder not wired")))
}

// TestBridgeEmbedder_DelegatesAfterWire asserts all three methods forward
// to the wired embedder.
func TestBridgeEmbedder_DelegatesAfterWire(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bridge, wire := cli.ExportNewBridgeEmbedder()
	wire(bridgeStubEmbedder{})

	g.Expect(bridge.ModelID()).To(Equal("stub@4"))
	g.Expect(bridge.Dims()).To(Equal(4))

	vec, err := bridge.Embed(context.Background(), "x")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(vec).To(Equal([]float32{0, 0, 0, 0}))
}

type bridgeStubEmbedder struct{}

func (bridgeStubEmbedder) Dims() int { return 4 }

func (bridgeStubEmbedder) Embed(context.Context, string) ([]float32, error) {
	return []float32{0, 0, 0, 0}, nil
}

func (bridgeStubEmbedder) ModelID() string { return "stub@4" }
```

- [ ] 8. **Verify.** Run `targ test` — expected: all green (relocated adapter tests pass in cmd/engram; internal embed suite green with fakes; bridge tests green; `TestTargets_EmbedStatus` stays green via bridge → LazyEmbedder pre-init ModelID). Run `targ check-full` — expected: clean; confirm `grep -rn '"os"\|knights-analytics' internal/embed/*.go | grep -v _test` returns nothing. Run `go install ./cmd/engram && engram embed status --vault "$(mktemp -d)"` from a non-repo cwd — expected: the six status lines with all-zero counts (real-binary check per house rules).
- [ ] 9. **Commit:**

```
refactor(embed): move hugot backend + model-cache FS to cmd/engram (#700)

internal/embed now depends only on injected Backend/CacheFS seams; the
hugot and os imports live in cmd/engram/hugot.go. CacheFS.Rename adopts
an errors.Is(fs.ErrExist) contract (platform-quirk sniffing moved into
the os adapter). Dead tempFS/unpackModelToTemp machinery deleted. cli
gains a transitional sharedEmbedder bridge wired from Deps.Embed.

AI-Used: [claude]
```

---

### Task T15 (B): internal/cli/embed.go — compose EmbedDeps from cli.Deps, delete osEmbedFS

**Depends on:** Task A + foundation task (Deps.FS `EdgeFS`, Deps.Embed) landed.

**Files**

- Modify: `internal/cli/embed.go` (delete `osEmbedFS`, `newOsEmbedDeps`; add `newEmbedDeps`)
- Modify: `internal/cli/vault_fs.go` (add `depsVaultFS` — shared-territory, see DESIGN FLAG 3's sibling note: other clusters should reuse this type when migrating `osVaultFS` call sites)
- Modify: `internal/cli/targets.go` (lines 226, 230, 155), `internal/cli/query.go` (line 1287-1288)
- Modify: `internal/cli/export_test.go` (replace `ExportNewOsEmbedDeps`), `internal/cli/os_adapters_test.go`

**Interfaces**

- Produces: `newEmbedDeps(d Deps) EmbedDeps` (pure composition); `depsVaultFS` (vaultgraph.VaultFS over EdgeFS); `ExportNewEmbedDeps(d Deps) EmbedDeps`.
- Consumes: `Deps.FS EdgeFS` (`ReadFile`, `WriteFileAtomic(path, data, perm fs.FileMode)`, `ReadDir(path) ([]fs.DirEntry, error)`), `Deps.Embed embed.Embedder`, `vaultgraph.ScanVault(fs VaultFS, vaultPath string) ([]Note, error)` with `VaultFS{ ListMD(dir string) ([]string, error); ReadFile(path string) ([]byte, error) }` (verified at `internal/vaultgraph/scanner.go:20-32`).

**Steps**

- [ ] 1. **RED — adapt the integration tests to the composed deps first.** In `internal/cli/os_adapters_test.go`: replace the three `cli.ExportNewOsEmbedDeps(<embedder>)` calls (lines 89, 125, 190) with `cli.ExportNewEmbedDeps(cli.Deps{FS: osTestEdgeFS{}, Embed: <same embedder>})`; rename `TestOsEmbedFS_ReadWriteScanRoundTrip` → `TestEmbedDeps_ReadWriteScanRoundTrip` and update its comment to say it exercises the composed Scan/Read/Write against a real tempdir vault; append the test-only EdgeFS implementation (test files are exempt from the purity rule; dedupe with foundation's equivalent if one exists — DESIGN FLAG 9):

```go
// osTestEdgeFS implements cli.EdgeFS over the real filesystem for
// integration tests. Test files are exempt from the internal purity rule.
type osTestEdgeFS struct{}

func (osTestEdgeFS) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) MkdirTemp(dir, pattern string) (string, error) {
	return os.MkdirTemp(dir, pattern) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) Remove(path string) error {
	return os.Remove(path) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) RemoveAll(path string) error {
	return os.RemoveAll(path) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	tmp := path + ".tmp-test"

	err := os.WriteFile(tmp, data, perm)
	if err != nil {
		return fmt.Errorf("writing temp %s: %w", tmp, err)
	}

	err = os.Rename(tmp, path)
	if err != nil {
		return fmt.Errorf("renaming %s: %w", tmp, err)
	}

	return nil
}
```

(add `"fmt"` and `"io/fs"` to the file's imports). Run `targ test` — expected RED: `ExportNewEmbedDeps` undefined.

- [ ] 2. **GREEN — compose.** In `internal/cli/embed.go` delete `osEmbedFS` and its three methods (lines 136-170) and `newOsEmbedDeps` (lines 241-252); the `"os"` import goes with them. Add:

```go
// newEmbedDeps composes the embed-command dependencies from the CLI-wide
// impure capability set. Pure composition — all I/O flows through d.FS and
// d.Embed, wired by cmd/engram. Sidecar writes go through WriteFileAtomic
// (temp+rename) so concurrent readers always see either the old or new
// file, never a torn write (ADR-0013 semantics preserved).
func newEmbedDeps(d Deps) EmbedDeps {
	const sidecarPerm = 0o600

	return EmbedDeps{
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(&depsVaultFS{fs: d.FS}, vault)
		},
		Read: d.FS.ReadFile,
		Write: func(path string, data []byte) error {
			err := d.FS.WriteFileAtomic(path, data, sidecarPerm)
			if err != nil {
				return fmt.Errorf("write: %w", err)
			}

			return nil
		},
		Embedder: d.Embed,
	}
}
```

In `internal/cli/vault_fs.go` add (below `osVaultFS`; imports gain `"io/fs"`):

```go
// depsVaultFS satisfies vaultgraph.VaultFS on top of the injected EdgeFS.
// Missing dirs list as empty (not an error), matching osVaultFS semantics —
// relies on the EdgeFS contract that ReadDir errors on absent paths satisfy
// errors.Is(err, fs.ErrNotExist).
type depsVaultFS struct {
	fs EdgeFS
}

// ListMD returns the .md filenames in dir. Missing dir → empty, nil.
func (v *depsVaultFS) ListMD(dir string) ([]string, error) {
	entries, err := v.fs.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading dir %s: %w", dir, err)
	}

	out := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		out = append(out, entry.Name())
	}

	return out, nil
}

// ReadFile reads the file at path via the injected EdgeFS.
func (v *depsVaultFS) ReadFile(path string) ([]byte, error) {
	data, err := v.fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return data, nil
}
```

In `internal/cli/export_test.go`, replace `ExportNewOsEmbedDeps` (lines 526-534):

```go
// ExportNewEmbedDeps exposes newEmbedDeps so integration tests can drive
// the composed Scan/Read/Write against a test EdgeFS without waking the
// bundled embedder (set Deps.Embed to a stub).
func ExportNewEmbedDeps(d Deps) EmbedDeps { return newEmbedDeps(d) }
```

- [ ] 3. **Rewire call sites.** `internal/cli/targets.go` embed group (current lines 226 and 230): `newOsEmbedDeps()` → `newEmbedDeps(deps)` (identifier per foundation's threading of Deps into `coreTargets`). `internal/cli/query.go:1287-1288` (coordinate with query cluster, DESIGN FLAG 3 — skip if its migration already landed):

Current:
```go
// newOsQueryDeps wires the production scan + read for the query command.
func newOsQueryDeps() QueryDeps {
	embedDeps := newOsEmbedDeps()
```

New:
```go
// newOsQueryDeps wires the production scan + read for the query command.
// TRANSITIONAL (#700): takes Deps for the embed composition; the query
// cluster's migration replaces the remaining os-backed fields.
func newOsQueryDeps(d Deps) QueryDeps {
	embedDeps := newEmbedDeps(d)
```

and its call site `internal/cli/targets.go:155`: `newOsQueryDeps()` → `newOsQueryDeps(deps)`.

- [ ] 4. **Verify.** `targ test` — expected green (embed_test.go's in-memory deps untouched; adapted os_adapters tests pass through `newEmbedDeps` + `osTestEdgeFS`; `TestTargets_EmbedApplyDryRun` / `TestTargets_EmbedStatus` green through the new wiring). `targ check-full` — clean; confirm `grep -n '"os"' internal/cli/embed.go` returns nothing. Real-binary check: `go install ./cmd/engram`, then in a temp dir: create `note.md` with a body, run `engram embed apply --vault . --dry-run` (expect `would-embed note.md (missing)`), then `engram embed apply --vault .` (expect `embedded  note.md (missing)` and a `note.vec.json` sidecar with `"embedding_model_id": "minilm-l6-v2@384"`), then `engram embed status --vault .` (expect `with-embeddings: 1`).
- [ ] 5. **Commit:**

```
refactor(cli): compose embed command deps from cli.Deps (#700)

newEmbedDeps(d Deps) replaces newOsEmbedDeps: Scan/Read/Write flow
through the injected EdgeFS (WriteFileAtomic preserves the ADR-0013
temp+rename sidecar semantics) and the embedder comes from Deps.Embed.
osEmbedFS deleted; internal/cli/embed.go no longer imports os.

AI-Used: [claude]
```

---

**Post-cluster residue for the enforcement task** (not handled here): delete the `sharedEmbedder`/`bridgeEmbedder` transitional block in `internal/cli/embed.go` once `grep -rn "sharedEmbedder" internal/cli --include='*.go' | grep -v _test` shows only its own definition; decide `parity_test.go` exemption (DESIGN FLAG 5); delete `osVaultFS` once all its consumers migrate to `depsVaultFS`.

### Task T16 (UF-1): `update.ErrCommandNotFound` sentinel + commander translation (drops os/exec from internal/update)

**Files**
- Modify: `internal/update/update.go` (add sentinel; swap two `errors.Is` checks; drop `os/exec` import; fix one comment)
- Modify: `internal/update/runner_test.go` (inject sentinel instead of exec.ErrNotFound; drop `os/exec` import)
- Modify: `internal/cli/update.go` (osCommander translates exec.ErrNotFound → sentinel)
- Modify: `internal/cli/update_test.go` (new RED test)
- Modify: `internal/cli/invariants_u1_test.go` (inject sentinel; drop `os/exec` import; comment fixes)

**Interfaces**
- Produces: `var ErrCommandNotFound = errors.New("command not found")` in package `update` — the Commander contract: implementations translate their platform not-found error to this sentinel before returning.
- Consumes: `update.Commander` (unchanged), `exec.ErrNotFound` (now only in the adapter, later only in cmd).

**Steps**

1. [ ] Add the sentinel (pure addition, keeps everything compiling for the RED test). In `internal/update/update.go`, replace the exported-variables block (lines 34–44):
   ```go
   // Exported variables.
   var (
   	ErrGitNotFound = errors.New("git binary not found on PATH")
   	ErrGoNotFound  = errors.New("go binary not found on PATH")
   	// ErrModelLFSStub means the cloned model.onnx is a Git-LFS pointer file,
   	// not the real model — building from it would embed a 133-byte stub and
   	// every embedding call would fail (issue #645).
   	ErrModelLFSStub     = errors.New("model.onnx is a git-lfs pointer stub")
   	ErrNoHarness        = errors.New("no supported harness found")
   	ErrSkillsSrcMissing = errors.New("skills source dir missing")
   )
   ```
   with:
   ```go
   // Exported variables.
   var (
   	// ErrCommandNotFound is the Commander contract for "binary not on PATH":
   	// implementations translate their platform's not-found error (e.g.
   	// exec.ErrNotFound) to this sentinel before returning, keeping this
   	// package free of os/exec (#700).
   	ErrCommandNotFound = errors.New("command not found")
   	ErrGitNotFound     = errors.New("git binary not found on PATH")
   	ErrGoNotFound      = errors.New("go binary not found on PATH")
   	// ErrModelLFSStub means the cloned model.onnx is a Git-LFS pointer file,
   	// not the real model — building from it would embed a 133-byte stub and
   	// every embedding call would fail (issue #645).
   	ErrModelLFSStub     = errors.New("model.onnx is a git-lfs pointer stub")
   	ErrNoHarness        = errors.New("no supported harness found")
   	ErrSkillsSrcMissing = errors.New("skills source dir missing")
   )
   ```
   Run `targ test` → expect PASS (no behavior change yet).

2. [ ] RED: in `internal/cli/update_test.go`, insert after `TestOsCommander_RunsCommand` (line 256, alphabetical order preserved):
   ```go
   func TestOsCommander_TranslatesNotFound(t *testing.T) {
   	t.Parallel()

   	g := NewWithT(t)

   	cmd := cli.ExportNewOsCommander()

   	_, _, err := cmd.Run(context.Background(), "", "engram-no-such-binary-7f3a")
   	g.Expect(err).To(MatchError(update.ErrCommandNotFound))
   }
   ```
   Run `targ test` → expect FAIL on exactly this test (current wrap `fmt.Errorf("%s %v: %w", name, args, err)` carries exec.ErrNotFound, not the sentinel).

3. [ ] GREEN: in `internal/cli/update.go`, inside `(*osCommander).Run` (lines 54–57), replace:
   ```go
   	err := cmd.Run()
   	if err != nil {
   		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("%s %v: %w", name, args, err)
   	}
   ```
   with:
   ```go
   	err := cmd.Run()
   	if err != nil {
   		if errors.Is(err, exec.ErrNotFound) {
   			return stdout.Bytes(), stderr.Bytes(),
   				fmt.Errorf("%s %v: %w: %w", name, args, update.ErrCommandNotFound, err)
   		}

   		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("%s %v: %w", name, args, err)
   	}
   ```
   (`errors` is already imported at line 6; go 1.26 supports the double `%w`.) Run `targ test` → expect PASS (internal/update still checks exec.ErrNotFound, which remains in the chain — both checks are satisfied during this step).

4. [ ] Cut internal/update over to the sentinel — all four hunks in one pass, tests updated in the same step:
   - `internal/update/update.go` line 436, replace:
     ```go
     		if errors.Is(cloneErr, exec.ErrNotFound) {
     ```
     with:
     ```go
     		if errors.Is(cloneErr, ErrCommandNotFound) {
     ```
   - `internal/update/update.go` lines 540–544, replace:
     ```go
     // classifyGoInstallErr maps a `go install` failure to ErrGoNotFound when the go
     // binary is absent from PATH (exec.ErrNotFound), otherwise wrapping the raw
     // error with the install mode for context.
     func classifyGoInstallErr(mode string, runErr error) error {
     	if errors.Is(runErr, exec.ErrNotFound) {
     ```
     with:
     ```go
     // classifyGoInstallErr maps a `go install` failure to ErrGoNotFound when the go
     // binary is absent from PATH (ErrCommandNotFound from the Commander), otherwise
     // wrapping the raw error with the install mode for context.
     func classifyGoInstallErr(mode string, runErr error) error {
     	if errors.Is(runErr, ErrCommandNotFound) {
     ```
   - `internal/update/update.go` imports (lines 6–15): delete the line `"os/exec"`.
   - `internal/update/runner_test.go` line 556, replace `cmd := &fakeCmd{err: exec.ErrNotFound}` with `cmd := &fakeCmd{err: update.ErrCommandNotFound}`; delete `"os/exec"` from its imports (line 8 — only use in the file).
   - `internal/cli/invariants_u1_test.go`: replace line 36
     ```go
     		Cmd: &u1FailCmd{err: fmt.Errorf(`go [install ./cmd/engram/]: %w`, exec.ErrNotFound)},
     ```
     with:
     ```go
     		Cmd: &u1FailCmd{err: fmt.Errorf(`go [install ./cmd/engram/]: %w`, update.ErrCommandNotFound)},
     ```
     delete `"os/exec"` from imports (line 8); replace comment lines 21–22 (`// missing 'go' binary (surfaced as exec.ErrNotFound from the go install` / `// command) must make update FAIL...`) with `// missing 'go' binary (surfaced as update.ErrCommandNotFound from the injected` / `// Commander) must make update FAIL with the update.ErrGoNotFound sentinel, and`; replace comment lines 32–33 with `// Commander fails the way the production adapter does when 'go' is absent:` / `// the update.ErrCommandNotFound chain is preserved through its %w wrap.`
   Run `targ test` → expect PASS. Run `targ check-full` → expect clean (verifies no unused-import leftovers, line lengths).

5. [ ] Verify zero os/exec in internal non-test files of this family: `grep -rn '"os/exec"' internal/update/ internal/cli/update.go | grep -v _test.go` → expect only `internal/cli/update.go` (removed by Task UF-2).

6. [ ] Commit (via the commit skill):
   ```
   refactor(update): ErrCommandNotFound sentinel replaces exec.ErrNotFound (#700)

   internal/update no longer imports os/exec: Commander implementations now
   translate their platform not-found error to the sentinel at the adapter
   boundary; the two errors.Is call sites classify against it.

   AI-Used: [claude]
   ```

---

### Task T17 (UF-2): move osCommander to cmd/engram; compose update deps purely from cli.Deps

Sequencing precondition: `internal/cli/deps.go` (Deps + EdgeFS), `cli.Targets(deps Deps)` threading down to `learnUpdateTargets`, and `cmd/engram/os_fs.go` (production EdgeFS) have landed.

**Files**
- Create: `cmd/engram/os_update.go`
- Create: `cmd/engram/os_update_test.go`
- Modify: `cmd/engram/main.go` (add `Commander: &osCommander{},` to the Deps literal built by the wiring cluster)
- Modify: `internal/cli/update.go` (delete osCommander/osUpdateFS/osUpdateEnv/osDirEntry/osFileInfo; add updateDeps/newUpdateDeps/updateFSFromEdge/updateEnvFromDeps; new runUpdate signature; drop `os`, `os/exec` imports)
- Modify: `internal/cli/targets.go` (update target call site)
- Modify: `internal/cli/export_test.go` (drop 3 adapter exports; add updateDeps exports + internal/update import)
- Modify: `internal/cli/update_test.go` (delete 10 adapter tests; rewrite 2 runUpdate smoke tests over test doubles)
- Create: `internal/cli/update_deps_test.go` (pure-composition unit tests)

**Interfaces**
- Consumes: `cli.Deps` fields `FS EdgeFS`, `Getenv func(string) string`, `Getwd func() (string, error)`, `UserHomeDir func() (string, error)`, `Commander update.Commander`, `Stdout io.Writer`; EdgeFS methods `ReadFile/WriteFile/MkdirAll/Stat/ReadDir/RemoveAll`; `update.ErrCommandNotFound` (UF-1).
- Produces: `func newUpdateDeps(d Deps) updateDeps` (pure); `func runUpdate(ctx context.Context, args UpdateArgs, deps updateDeps, stdout io.Writer) error`; cmd/engram `osCommander` implementing `update.Commander` with not-found translation.

**Steps**

1. [ ] Create `cmd/engram/os_update.go` (verbatim relocation of the UF-1 osCommander, package main):
   ```go
   package main

   import (
   	"bytes"
   	"context"
   	"errors"
   	"fmt"
   	"os/exec"

   	"github.com/toejough/engram/internal/update"
   )

   // unexported variables.
   var _ update.Commander = (*osCommander)(nil)

   // osCommander is the production update.Commander: runs external commands
   // via os/exec, capturing stdout/stderr. A missing binary is translated to
   // update.ErrCommandNotFound per the Commander contract (#700) — internal/
   // never imports os/exec.
   type osCommander struct{}

   func (*osCommander) Run(
   	ctx context.Context, dir, name string, args ...string,
   ) ([]byte, []byte, error) {
   	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // name/args from internal callers
   	cmd.Dir = dir
   	stdout := &bytes.Buffer{}
   	stderr := &bytes.Buffer{}
   	cmd.Stdout = stdout
   	cmd.Stderr = stderr

   	err := cmd.Run()
   	if err != nil {
   		if errors.Is(err, exec.ErrNotFound) {
   			return stdout.Bytes(), stderr.Bytes(),
   				fmt.Errorf("%s %v: %w: %w", name, args, update.ErrCommandNotFound, err)
   		}

   		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("%s %v: %w", name, args, err)
   	}

   	return stdout.Bytes(), stderr.Bytes(), nil
   }
   ```

2. [ ] Create `cmd/engram/os_update_test.go` (relocated + strengthened integration tests, package main):
   ```go
   package main

   import (
   	"context"
   	"path/filepath"
   	"strings"
   	"testing"

   	. "github.com/onsi/gomega"

   	"github.com/toejough/engram/internal/update"
   )

   func TestOsCommander_ReportsFailure(t *testing.T) {
   	t.Parallel()

   	g := NewWithT(t)

   	cmd := &osCommander{}

   	_, _, err := cmd.Run(context.Background(), "", "false")
   	g.Expect(err).To(HaveOccurred())
   }

   func TestOsCommander_RunsCommand(t *testing.T) {
   	t.Parallel()

   	g := NewWithT(t)

   	cmd := &osCommander{}

   	stdout, _, err := cmd.Run(context.Background(), "", "echo", "hello world")
   	g.Expect(err).NotTo(HaveOccurred())
   	g.Expect(strings.TrimSpace(string(stdout))).To(Equal("hello world"))
   }

   func TestOsCommander_RunsInDir(t *testing.T) {
   	t.Parallel()

   	g := NewWithT(t)

   	cmd := &osCommander{}
   	dir := t.TempDir()

   	// macOS TempDir sits under a symlink (/tmp → /private/tmp); compare
   	// against the resolved path so `pwd` output matches.
   	resolved, evalErr := filepath.EvalSymlinks(dir)
   	g.Expect(evalErr).NotTo(HaveOccurred())

   	if evalErr != nil {
   		return
   	}

   	stdout, _, err := cmd.Run(context.Background(), dir, "pwd")
   	g.Expect(err).NotTo(HaveOccurred())
   	g.Expect(strings.TrimSpace(string(stdout))).To(Equal(resolved))
   }

   func TestOsCommander_TranslatesNotFound(t *testing.T) {
   	t.Parallel()

   	g := NewWithT(t)

   	cmd := &osCommander{}

   	_, _, err := cmd.Run(context.Background(), "", "engram-no-such-binary-7f3a")
   	g.Expect(err).To(MatchError(update.ErrCommandNotFound))
   }
   ```
   Run `targ test` → expect PASS (this is refactor-RED: existing suite green + new adapter integration tests green; the internal copies still exist at this point).

3. [ ] Rewrite `internal/cli/update.go` — delete the adapters, add pure composition, retarget runUpdate.
   - Imports (lines 3–17): delete `"os"` and `"os/exec"`; keep `bytes`, `context`, `errors`, `fmt`, `io`, `io/fs`, `path/filepath`, `slices`, `strings`, and the `internal/update` import.
   - Replace the unexported-variables block (lines 34–40):
     ```go
     // unexported variables.
     var (
     	_                      update.Filesystem = (*osUpdateFS)(nil)
     	errSomeHarnessesFailed                   = errors.New(
     		"update: one or more detected harnesses failed",
     	)
     )
     ```
     with:
     ```go
     // unexported variables.
     var (
     	_ update.Env        = (*updateEnvFromDeps)(nil)
     	_ update.Filesystem = (*updateFSFromEdge)(nil)

     	errSomeHarnessesFailed = errors.New(
     		"update: one or more detected harnesses failed",
     	)
     )
     ```
   - Delete entirely: `osCommander` type + `Run` method (lines 42–60), `osDirEntry` + methods (62–66), `osFileInfo` + method (68–70), `osUpdateEnv` + methods (72–84), the `// --- production adapters ---` comment (86), `osUpdateFS` + all six methods (88–151).
   - In their place add the pure composition (zero I/O — depguard-safe):
     ```go
     // updateDeps carries the injected surfaces Updater.Run needs. Composed
     // from the CLI-wide Deps by newUpdateDeps — pure plumbing, no I/O (#700).
     type updateDeps struct {
     	FS  update.Filesystem
     	Cmd update.Commander
     	Env update.Env
     }

     // newUpdateDeps composes update's dependency surface from cli.Deps.
     func newUpdateDeps(d Deps) updateDeps {
     	return updateDeps{
     		FS:  &updateFSFromEdge{fs: d.FS},
     		Cmd: d.Commander,
     		Env: &updateEnvFromDeps{
     			getenv:      d.Getenv,
     			getwd:       d.Getwd,
     			userHomeDir: d.UserHomeDir,
     		},
     	}
     }

     // updateEnvFromDeps adapts cli.Deps' env funcs to update.Env.
     type updateEnvFromDeps struct {
     	getenv      func(string) string
     	getwd       func() (string, error)
     	userHomeDir func() (string, error)
     }

     func (e *updateEnvFromDeps) Getenv(key string) string { return e.getenv(key) }

     func (e *updateEnvFromDeps) Getwd() (string, error) { return e.getwd() }

     func (e *updateEnvFromDeps) UserHomeDir() (string, error) { return e.userHomeDir() }

     // updateFSFromEdge adapts the CLI-wide EdgeFS to update.Filesystem. Pure
     // interface plumbing: fs.DirEntry / fs.FileInfo structurally satisfy
     // update.DirEntry / update.FileInfo. Errors pass through unwrapped so
     // errors.Is(err, fs.ErrNotExist) checks in the update package keep working.
     type updateFSFromEdge struct {
     	fs EdgeFS
     }

     func (a *updateFSFromEdge) MkdirAll(path string, perm fs.FileMode) error {
     	return a.fs.MkdirAll(path, perm) //nolint:wrapcheck // pass-through; update core adds context
     }

     func (a *updateFSFromEdge) ReadDir(path string) ([]update.DirEntry, error) {
     	entries, err := a.fs.ReadDir(path)
     	if err != nil {
     		//nolint:wrapcheck // caller distinguishes fs.ErrNotExist via errors.Is
     		return nil, err
     	}

     	out := make([]update.DirEntry, 0, len(entries))
     	for _, entry := range entries {
     		out = append(out, entry)
     	}

     	return out, nil
     }

     func (a *updateFSFromEdge) ReadFile(path string) ([]byte, error) {
     	data, err := a.fs.ReadFile(path)
     	if err != nil {
     		//nolint:wrapcheck // caller distinguishes fs.ErrNotExist via errors.Is
     		return nil, err
     	}

     	return data, nil
     }

     func (a *updateFSFromEdge) RemoveAll(path string) error {
     	return a.fs.RemoveAll(path) //nolint:wrapcheck // pass-through; update core adds context
     }

     func (a *updateFSFromEdge) Stat(path string) (update.FileInfo, error) {
     	info, err := a.fs.Stat(path)
     	if err != nil {
     		//nolint:wrapcheck // caller distinguishes fs.ErrNotExist via errors.Is
     		return nil, err
     	}

     	return info, nil
     }

     func (a *updateFSFromEdge) WriteFile(path string, data []byte, perm fs.FileMode) error {
     	return a.fs.WriteFile(path, data, perm) //nolint:wrapcheck // pass-through; update core adds context
     }
     ```
   - Replace `runUpdate` (lines 275–295):
     ```go
     // runUpdate wires production adapters and invokes Updater.Run.
     func runUpdate(ctx context.Context, args UpdateArgs, stdout io.Writer) error {
     	updater := &update.Updater{
     		FS:  &osUpdateFS{},
     		Cmd: &osCommander{},
     		Env: &osUpdateEnv{},
     	}

     	report, runErr := updater.Run(ctx, update.Options{
     		DryRun:       args.DryRun,
     		WithGuidance: args.WithGuidance,
     	})
     	if runErr == nil {
     		vaultPath := resolveVault("", report.Home, updater.Env.Getenv)
     		report.VaultHasOldVocabFiles = oldVocabFilesPresent(vaultPath, updater.FS)
     		chunksDir := ResolveChunksDir("", report.Home, updater.Env.Getenv)
     		report.ChunkIndexHasEmptyFiles = chunkIndexHasEmptyFiles(chunksDir, updater.FS)
     	}

     	return finishUpdate(stdout, report, runErr)
     }
     ```
     with:
     ```go
     // runUpdate invokes Updater.Run over the injected dependency surface.
     func runUpdate(ctx context.Context, args UpdateArgs, deps updateDeps, stdout io.Writer) error {
     	updater := &update.Updater{
     		FS:  deps.FS,
     		Cmd: deps.Cmd,
     		Env: deps.Env,
     	}

     	report, runErr := updater.Run(ctx, update.Options{
     		DryRun:       args.DryRun,
     		WithGuidance: args.WithGuidance,
     	})
     	if runErr == nil {
     		vaultPath := resolveVault("", report.Home, deps.Env.Getenv)
     		report.VaultHasOldVocabFiles = oldVocabFilesPresent(vaultPath, deps.FS)
     		chunksDir := ResolveChunksDir("", report.Home, deps.Env.Getenv)
     		report.ChunkIndexHasEmptyFiles = chunkIndexHasEmptyFiles(chunksDir, deps.FS)
     	}

     	return finishUpdate(stdout, report, runErr)
     }
     ```
     (ENGRAM_VAULT_PATH / ENGRAM_CHUNKS_DIR reads thereby flow through `deps.Env.Getenv` ← `cli.Deps.Getenv` — no separate env work needed for this family.)

4. [ ] Retarget the call site in `internal/cli/targets.go` (current lines 220–222; `learnUpdateTargets` has `deps Deps` in scope post-wiring-cluster). Replace:
   ```go
   		targ.Targ(func(ctx context.Context, a UpdateArgs) {
   			errHandler(runUpdate(withLog(ctx), a, stdout))
   		}).Name("update").Description("Refresh engram binary and harness skills"),
   ```
   with:
   ```go
   		targ.Targ(func(ctx context.Context, a UpdateArgs) {
   			errHandler(runUpdate(withLog(ctx), a, newUpdateDeps(deps), deps.Stdout))
   		}).Name("update").Description("Refresh engram binary and harness skills"),
   ```
   (If the wiring cluster still passes `stdout` as a separate arg to `learnUpdateTargets`, use `stdout` in place of `deps.Stdout` — same value, wired from `Deps.Stdout` upstream.)

5. [ ] Wire the commander in `cmd/engram/main.go`: inside the Deps literal built by the wiring cluster, add the field:
   ```go
   		Commander: &osCommander{},
   ```
   (Exact surrounding literal owned by the wiring cluster; this field assignment is this task's contract.)

6. [ ] `internal/cli/export_test.go`:
   - Delete lines 523–524 (`ExportNewOsCommander`), 566–567 (`ExportNewOsUpdateEnv`), 569–570 (`ExportNewOsUpdateFS`) including their doc comments.
   - `ExportRunUpdate = runUpdate` (line 108) stays — the alias picks up the new signature.
   - Add to the import block `"github.com/toejough/engram/internal/update"`, and add:
     ```go
     // ExportNewUpdateDeps exposes the production pure composition for tests.
     var ExportNewUpdateDeps = newUpdateDeps

     // ExportNewUpdateDepsFrom builds the unexported updateDeps from explicit
     // surfaces so black-box tests can drive runUpdate with test doubles.
     func ExportNewUpdateDepsFrom(fs update.Filesystem, cmd update.Commander, env update.Env) updateDeps {
     	return updateDeps{FS: fs, Cmd: cmd, Env: env}
     }
     ```
     (Place the var alphabetically in the existing var block; the func with the other Export funcs.)

7. [ ] `internal/cli/update_test.go`:
   - Delete `TestOsCommander_ReportsFailure`, `TestOsCommander_RunsCommand`, `TestOsCommander_TranslatesNotFound` (relocated in step 2), `TestOsUpdateEnv_ReturnsValues`, and all seven `TestOsUpdateFS_*` tests plus the `// osUpdateFS round-trip tests:` comment (lines 235–441 in the current file, minus `TestPluralFile`).
   - Rewrite the two runUpdate smoke tests and add the test doubles (file-local; `os` and `io/fs` already importable in _test.go — `fs` needs adding to imports):
     ```go
     // liveUpdateEnv adapts the real process environment to update.Env for the
     // dry-run smoke tests (production Env is composed from cli.Deps).
     type liveUpdateEnv struct{}

     func (liveUpdateEnv) Getenv(key string) string { return os.Getenv(key) }

     func (liveUpdateEnv) Getwd() (string, error) {
     	return os.Getwd() //nolint:wrapcheck // test adapter
     }

     func (liveUpdateEnv) UserHomeDir() (string, error) {
     	return os.UserHomeDir() //nolint:wrapcheck // test adapter
     }

     // liveUpdateFS is an os-backed update.Filesystem for the dry-run smoke
     // tests (dry-run never writes; write methods exist to satisfy the interface).
     type liveUpdateFS struct{}

     func (liveUpdateFS) MkdirAll(path string, perm fs.FileMode) error {
     	return os.MkdirAll(path, perm) //nolint:wrapcheck // test adapter
     }

     func (liveUpdateFS) ReadDir(path string) ([]update.DirEntry, error) {
     	entries, err := os.ReadDir(path)
     	if err != nil {
     		return nil, err //nolint:wrapcheck // errors.Is(fs.ErrNotExist) must survive
     	}

     	out := make([]update.DirEntry, 0, len(entries))
     	for _, entry := range entries {
     		out = append(out, entry)
     	}

     	return out, nil
     }

     func (liveUpdateFS) ReadFile(path string) ([]byte, error) {
     	return os.ReadFile(path) //nolint:wrapcheck,gosec // test adapter; test-chosen paths
     }

     func (liveUpdateFS) RemoveAll(path string) error {
     	return os.RemoveAll(path) //nolint:wrapcheck // test adapter
     }

     func (liveUpdateFS) Stat(path string) (update.FileInfo, error) {
     	info, err := os.Stat(path)
     	if err != nil {
     		return nil, err //nolint:wrapcheck // errors.Is(fs.ErrNotExist) must survive
     	}

     	return info, nil
     }

     func (liveUpdateFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
     	return os.WriteFile(path, data, perm) //nolint:wrapcheck // test adapter
     }

     // stubCommander satisfies update.Commander; dry-run local mode never runs it.
     type stubCommander struct{}

     func (stubCommander) Run(context.Context, string, string, ...string) ([]byte, []byte, error) {
     	return nil, nil, nil
     }
     ```
     and replace the two tests' bodies:
     ```go
     func TestRunUpdate_DryRunFromCwd(t *testing.T) {
     	t.Parallel()

     	g := NewWithT(t)

     	stdout := &bytes.Buffer{}
     	deps := cli.ExportNewUpdateDepsFrom(liveUpdateFS{}, stubCommander{}, liveUpdateEnv{})

     	// Dry-run against the live filesystem: cwd is inside the engram
     	// worktree, so source resolution picks local mode without `go install`.
     	err := cli.ExportRunUpdate(context.Background(), cli.UpdateArgs{DryRun: true}, deps, stdout)
     	out := stdout.String()

     	if err != nil {
     		g.Expect(err.Error()).To(ContainSubstring("update"))

     		return
     	}

     	g.Expect(out).To(ContainSubstring("[dry-run] engram update"))
     	g.Expect(out).To(ContainSubstring("source: local clone at "))
     }

     func TestRunUpdate_WithGuidanceFlagMapsToOptions(t *testing.T) {
     	t.Parallel()

     	g := NewWithT(t)

     	stdout := &bytes.Buffer{}
     	deps := cli.ExportNewUpdateDepsFrom(liveUpdateFS{}, stubCommander{}, liveUpdateEnv{})

     	// Dry-run with --with-guidance; only verifies the flag maps to Options.
     	err := cli.ExportRunUpdate(
     		context.Background(), cli.UpdateArgs{DryRun: true, WithGuidance: true}, deps, stdout)
     	if err != nil {
     		g.Expect(err.Error()).To(ContainSubstring("update"))
     	}
     }
     ```
     Add `"io/fs"` to the file's imports (as `fs`), keep `os`, `context`, etc.

8. [ ] Create `internal/cli/update_deps_test.go` — pure-composition unit tests over a fake EdgeFS (hand fakes match this family's precedent: u1FS/fakeCmd):
    ```go
    package cli_test

    import (
    	"context"
    	"errors"
    	"io/fs"
    	"testing"
    	"testing/fstest"

    	. "github.com/onsi/gomega"

    	"github.com/toejough/engram/internal/cli"
    	"github.com/toejough/engram/internal/update"
    )

    func TestNewUpdateDeps_CommanderPassesThrough(t *testing.T) {
    	t.Parallel()

    	g := NewWithT(t)

    	cmd := stubCommander{}
    	deps := cli.ExportNewUpdateDeps(cli.Deps{Commander: cmd, FS: fakeEdgeFS{}})

    	stdout, stderr, err := deps.Cmd.Run(context.Background(), "", "x")
    	g.Expect(err).NotTo(HaveOccurred())
    	g.Expect(stdout).To(BeNil())
    	g.Expect(stderr).To(BeNil())
    }

    func TestNewUpdateDeps_EnvDelegatesToDepsFuncs(t *testing.T) {
    	t.Parallel()

    	g := NewWithT(t)

    	deps := cli.ExportNewUpdateDeps(cli.Deps{
    		FS:          fakeEdgeFS{},
    		Getenv:      func(key string) string { return "env:" + key },
    		Getwd:       func() (string, error) { return "/cwd", nil },
    		UserHomeDir: func() (string, error) { return "/home/x", nil },
    	})

    	g.Expect(deps.Env.Getenv("K")).To(Equal("env:K"))

    	cwd, cwdErr := deps.Env.Getwd()
    	g.Expect(cwdErr).NotTo(HaveOccurred())
    	g.Expect(cwd).To(Equal("/cwd"))

    	home, homeErr := deps.Env.UserHomeDir()
    	g.Expect(homeErr).NotTo(HaveOccurred())
    	g.Expect(home).To(Equal("/home/x"))
    }

    func TestNewUpdateDeps_FSAdapterPreservesNotExist(t *testing.T) {
    	t.Parallel()

    	g := NewWithT(t)

    	deps := cli.ExportNewUpdateDeps(cli.Deps{FS: fakeEdgeFS{}})

    	_, readErr := deps.FS.ReadFile("/missing")
    	g.Expect(errors.Is(readErr, fs.ErrNotExist)).To(BeTrue())

    	_, dirErr := deps.FS.ReadDir("/missing")
    	g.Expect(errors.Is(dirErr, fs.ErrNotExist)).To(BeTrue())

    	_, statErr := deps.FS.Stat("/missing")
    	g.Expect(errors.Is(statErr, fs.ErrNotExist)).To(BeTrue())
    }

    func TestNewUpdateDeps_FSAdapterReadsThroughEdgeFS(t *testing.T) {
    	t.Parallel()

    	g := NewWithT(t)

    	deps := cli.ExportNewUpdateDeps(cli.Deps{FS: fakeEdgeFS{
    		"skills/learn/SKILL.md": &fstest.MapFile{Data: []byte("learn")},
    	}})

    	data, readErr := deps.FS.ReadFile("skills/learn/SKILL.md")
    	g.Expect(readErr).NotTo(HaveOccurred())
    	g.Expect(string(data)).To(Equal("learn"))

    	entries, dirErr := deps.FS.ReadDir("skills")
    	g.Expect(dirErr).NotTo(HaveOccurred())

    	if dirErr != nil {
    		return
    	}

    	g.Expect(entries).To(HaveLen(1))
    	g.Expect(entries[0].Name()).To(Equal("learn"))
    	g.Expect(entries[0].IsDir()).To(BeTrue())

    	info, statErr := deps.FS.Stat("skills/learn/SKILL.md")
    	g.Expect(statErr).NotTo(HaveOccurred())

    	if statErr != nil || info == nil {
    		return
    	}

    	g.Expect(info.IsDir()).To(BeFalse())
    }

    // fakeEdgeFS is a read-only in-memory cli.EdgeFS over fstest.MapFS.
    // Write-side methods return errUnsupported: the update dry-run/read paths
    // under test never invoke them.
    type fakeEdgeFS fstest.MapFS

    func (m fakeEdgeFS) MkdirAll(string, fs.FileMode) error { return errUnsupported }

    func (m fakeEdgeFS) MkdirTemp(string, string) (string, error) { return "", errUnsupported }

    func (m fakeEdgeFS) ReadDir(path string) ([]fs.DirEntry, error) {
    	return fs.ReadDir(fstest.MapFS(m), path) //nolint:wrapcheck // fake passes chains through
    }

    func (m fakeEdgeFS) ReadFile(path string) ([]byte, error) {
    	return fs.ReadFile(fstest.MapFS(m), path) //nolint:wrapcheck // fake passes chains through
    }

    func (m fakeEdgeFS) Remove(string) error { return errUnsupported }

    func (m fakeEdgeFS) RemoveAll(string) error { return errUnsupported }

    func (m fakeEdgeFS) Rename(string, string) error { return errUnsupported }

    func (m fakeEdgeFS) Stat(path string) (fs.FileInfo, error) {
    	return fs.Stat(fstest.MapFS(m), path) //nolint:wrapcheck // fake passes chains through
    }

    func (m fakeEdgeFS) WalkDir(root string, fn fs.WalkDirFunc) error {
    	return fs.WalkDir(fstest.MapFS(m), root, fn) //nolint:wrapcheck // fake passes chains through
    }

    func (m fakeEdgeFS) WriteFile(string, []byte, fs.FileMode) error { return errUnsupported }

    func (m fakeEdgeFS) WriteFileAtomic(string, []byte, fs.FileMode) error { return errUnsupported }

    // unexported variables.
    var errUnsupported = errors.New("fakeEdgeFS: write path not supported")
    ```
    Note for the executor: sync `fakeEdgeFS`'s method set with the LANDED `cli.EdgeFS` (the deps cluster may also ship a shared fake — reuse it and delete this one if so). `fstest.MapFS` paths are slash-relative (no leading `/`), hence the relative paths above; its error chains wrap `fs.ErrNotExist`, which is exactly the property under test.
    Run `targ test` → expect PASS. Run `targ check-full` → expect clean.

9. [ ] Purity verification for this family:
    - `grep -rn '"os"\|"os/exec"' internal/cli/update.go internal/update/update.go` → no hits.
    - `go install ./cmd/engram && engram update --dry-run` from the worktree root → expect `[dry-run] engram update` + `source: local clone at ...` output (real-binary check per house rule; exercises the full Deps wiring including osCommander).

10. [ ] Commit (via the commit skill):
    ```
    refactor(cli): move update commander to cmd/engram, compose update deps from cli.Deps (#700)

    osCommander (the only os/exec user) relocates to cmd/engram/os_update.go
    with the ErrCommandNotFound translation; osUpdateFS/osUpdateEnv are
    absorbed into pure adapters over cli.Deps (EdgeFS bridge + env-func
    bridge). internal/cli/update.go is now I/O-import-free.

    AI-Used: [claude]
    ```

### Task T-final-1: Enforcement flip — depguard + forbidigo land with zero carve-outs

**Files:**
- Modify: `dev/golangci-lint.toml`

**Interfaces:**
- Consumes: the fully-migrated tree (all prior tasks complete; no os/exec/signal/syscall/hugot imports and no time.Now/Since/Tick references remain in internal/ non-test code).
- Produces: the enforced purity boundary; `targ check-full` fails on any regression.

- [ ] **Step 1: RED — add the depguard rule and confirm it currently passes only because migration is done.** Add to `dev/golangci-lint.toml` alongside the existing `[linters.settings.depguard.rules.all]`:

```toml
# #700: internal/ purity — default-deny. Anything not prefix-matched below is denied
# in internal non-test code. NO file carve-outs (all I/O adapters live in cmd/engram).
# Glob form '**/internal/**' is the prototype-confirmed form (2026-07-19).
[linters.settings.depguard.rules.internal-purity]
files = ['**/internal/**', '!$test']
allow = [
	'strings', 'fmt', 'errors', 'sort', 'slices', 'maps', 'strconv', 'unicode',
	'bufio', 'bytes', 'io', 'path', 'regexp',
	'encoding/json', 'encoding/hex', 'crypto/sha256', 'hash/fnv', 'math', 'time',
	'context', 'sync', 'embed',
	'github.com/toejough/engram',
	'go.yaml.in/yaml/v3',
	'github.com/toejough/targ',
]
```

Notes: prefix matching means `io` admits `io/fs`, `path` admits `path/filepath`, `sync` admits `sync/atomic`, `math` admits `math/rand/v2` (kmeans' seeded PCG — misuse is forbidigo's job below), `time` admits types/parsing (clock calls are forbidigo's job). No `deny` entries: a `math/rand` deny would prefix-catch the legal seeded v2 import.

- [ ] **Step 2: enable forbidigo with the custom list.** Remove `'forbidigo'` from the `disable` list and add:

```toml
[linters.settings.forbidigo]
# Custom list REPLACES the fmt.Print defaults — printing stays legal (repo prints on purpose).
analyze-types = true

[[linters.settings.forbidigo.forbid]]
pattern = '^time\.Now$'
msg = 'inject a clock via cli.Deps.Now (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '^time\.Since$'
msg = 'inject a clock via cli.Deps.Now (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '^time\.Tick$'
msg = 'inject a clock via cli.Deps.Now (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '^targ\.Main$'
msg = 'targ dispatch is edge work — call from cmd/engram only (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '.*'
pkg = '^math/rand$'
msg = 'math/rand v1 is banned — use a seeded math/rand/v2 *rand.Rand (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '^rand\.(N|Int32|Int64|Int32N|Int64N|IntN|Int|Uint32|Uint64|Uint32N|Uint64N|UintN|Uint|Float32|Float64|NormFloat64|ExpFloat64|Perm|Shuffle)$'
pkg = '^math/rand/v2$'
msg = 'auto-seeded global PRNG is banned — use a seeded *rand.Rand (#700)'
```

And scope it with exclusion rules (forbidigo applies only to internal non-test code):

```toml
[[linters.exclusions.rules]]
linters = ['forbidigo']
path-except = '^internal/'

[[linters.exclusions.rules]]
linters = ['forbidigo']
path = '_test\.go$'
```

- [ ] **Step 3: set `max-issues-per-linter = 0`** (deliberate: never truncate findings; the default 10 hid 14 of 24 findings during the plan-time prototype). Find the existing `max-issues-per-linter` line and set it to `0`.

- [ ] **Step 4: verify — `targ check-full`.** Expected: GREEN. If depguard/forbidigo report findings, the migration missed a site — fix the SITE (relocate/thread it per the relevant task's pattern); NEVER add a carve-out, nolint, or allow-list entry to make it pass (that violates the issue's zero-grandfathering acceptance criteria; escalate to the orchestrator if a finding looks structural).

- [ ] **Step 5: negative self-test of the gate (temporary, not committed).** Add `_ = os.Getenv("PROBE")` (+ `"os"` import) to any internal/ non-test file; run `targ check-full`; expect a depguard finding naming `internal-purity`. Revert the probe. This proves the rule fires (a green gate that can't fail is no gate).

- [ ] **Step 5.5: coverage stance for cmd/engram (issue AC).** `cmd/engram` now holds adapter code with integration tests, not just a thin entry point. Inspect how the coverage gate treats it: read the check-full output's coverage section (and `rg -n "cmd/" dev/targs.go dev/*.toml` for exclusion patterns). Record the finding + decision as a one-paragraph note in the commit body of Step 6: either (a) cmd/engram stays coverage-excluded because its files are integration-tested I/O wrappers (unit-coverage-exempt per the repo's test-categorization doctrine), or (b) it enters coverage with the integration tests counting. Do not silently leave the stance undecided — the issue AC requires a deliberate call, surfaced to the orchestrator if the tooling makes (b) awkward.

- [ ] **Step 6: Commit.**

```bash
git add dev/golangci-lint.toml
git commit -m "check(#700): enforce internal/ purity — depguard default-deny + forbidigo clock/PRNG

Zero carve-outs: all I/O adapters now live in cmd/engram, so the rule
needs no file exceptions. Custom forbidigo list replaces print defaults
(printing stays legal). max-issues-per-linter=0 so findings never truncate.

AI-Used: [claude]"
```

### Task T-final-2: FIXME removal + issue closure prep

**Files:**
- Modify: `internal/cli/main.go` (or its successor location for the comment — wherever the FIXME(#700) block lives after the wiring-core task)

**Interfaces:**
- Consumes: T-final-1 complete (`targ check-full` green with enforcement active).
- Produces: the resolved FIXME per the user's rule ("remove the FIXME only when the issue is resolved").

- [ ] **Step 1: verify the enforcement is green**: `targ check-full` → GREEN (fresh run, not a cached claim).
- [ ] **Step 2: delete the 4-line `FIXME(#700)` comment block** (it begins `// FIXME(#700): this os.Getenv call is IO.`). Note: the wiring-core task already deleted the `os.Getenv` call itself; if the comment was relocated or already removed by that task's diff, verify with `rg -n "FIXME\(#700\)" .` that ZERO hits remain — that grep result is this step's deliverable either way.
- [ ] **Step 3: Commit.**

```bash
git add -A
git commit -m "chore(#700): remove resolved FIXME — purity boundary enforced

AI-Used: [claude]"
```

## Documentation surface (step-5 dispositions, Gate C verifies)

| File | Disposition | Reason |
|---|---|---|
| `CLAUDE.md` | update | directory-structure + Key Files: `cmd/engram` becomes composition root (adapters + wiring + integration tests); DI bullet gains "lint-enforced (depguard/forbidigo, #700)" |
| `README.md` | update | line 127 "cmd/engram/ CLI entry point (thin wiring layer)" → composition root with adapter implementations + integration tests |
| `docs/architecture/c3-components.md` | update | K11 debuglog row cites `cli/signal.go` (moved to cmd); add the edge/adapter composition to the component table + mermaid |
| `docs/architecture/adr.md` | update | status addenda on ADR-0001 (composition root now cmd/engram, lint-enforced) and ADR-0013 (flock/atomic impls relocated to cmd, semantics unchanged, regression test carried); NEW ADR-0020 recording the enforced purity boundary (config-only enforcement, zero carve-outs) |
| `docs/GLOSSARY.md` | keep (verify) | cited files remain; `targets.go` still wires subcommands (now from `cli.Deps`) |
| `docs/architecture/c1-system-context.md` | keep (verify citations) | flows unchanged; update-flow + query citations still valid |
| `docs/architecture/c2-containers.md` | keep (verify) | C1/C2 skill-binary seam unchanged |
| `dev/eval/LEDGER.md` | keep | historical vintage-stamped measurement records — never retro-edited |
| `docs/superpowers/plans/2026-07-18-646-recency-value-proof.md` | keep | historical plan artifact |
| `skills/`, `commands/`, `guidance/` | n/a | grep-verified 2026-07-19: no Go-path references |

## Merge protocol (repo rules)

Review-before-merge with argumentation; rebase on main + re-test before merging; `git merge --ff-only` only; rebase loop if another branch (two live Pi worktrees!) lands first; never push unreviewed work.
