### Task T4 (L2): purge os/syscall adapters from internal/cli/cli.go (ORDER: after activate/amend/resituate/vocab/ingest/prune constructor migrations, before enforcement)

**Depends on:** T8, T9, T12 complete (ingest/prune/maintenance constructors already compose from Deps) — see binding execution order R4.

**Files**
- Modify: `internal/cli/cli.go` (delete `osLearnFS`, `flockPath`, `logWarningToStderrf`; drop `os`/`syscall` imports; `osFileReader` and `osManifestLock` are already gone — deleted by T8 per R5 and T9 per R10, grep-verified here)
- Modify: `internal/cli/export_test.go` (delete `ExportNewOsLearnFS`, `ExportLogWarningToStderr`; `ExportFlockPath` and `ExportOsManifestLock` are already gone — deleted by T8 per R7 and T9 per R10, grep-verified here)
- Modify: `internal/cli/learn_adapters_test.go` (delete `TestOsLearnFS_Lock_BadVaultReturnsError`)
- Modify: `internal/cli/os_adapters_test.go` (delete `TestExportLogWarningToStderr_FormatsAndWrites` — superseded by `TestNewLearnDeps_LogWarning_WritesToDepsStderr`)
- Verify-only (no edit): `internal/cli/testhelpers_test.go` (`TestOsManifestLock_MkdirError` already deleted by T9 per R10), `internal/cli/ingest_test.go` (Lock closures already ride T8's `testFlocker{}` per R7)

**Interfaces**
- Consumes: nothing new. Produces: none — pure deletion.

NOTE (anchors): line numbers cited below are from the pristine tree and shift after T3's deletions (and the other families' migrations that land before T4 per R4); the symbol-based gates govern, not absolute lines.

**Steps**

- [ ] 1. **Precondition gate (must pass before any edit):**
`grep -rn "osLearnFS\|flockPath\|logWarningToStderrf" internal/ --include="*.go" | grep -v "_test.go"` → expected: hits ONLY inside `internal/cli/cli.go` (the definitions themselves). Any hit in activate.go/amend.go/resituate.go/vocab_commands.go/ingest.go/prune.go means a family migration hasn't landed — STOP and reorder. Additionally verify the earlier owners' deletions landed: `rg -n "osFileReader" internal/` → zero hits (T8, R5) and `rg -n "osManifestLock" internal/` → zero hits (T9, R10). Any hit → that task did not complete — STOP.

- [ ] 2. **Grep-verify the flock handoff (dead-work guard — R7).** T8 already deleted `ExportFlockPath` and repointed the `TestManifest_ConcurrentWritersDoNotLoseEntries` Lock closures to its test-local real-flock `testFlocker{}`, so the concurrent-writers regression keeps racing real OS locks without any production flock symbol and NOTHING is re-implemented here. Verify: `rg -n "ExportFlockPath" internal/` → zero hits (any hit → T8 incomplete, STOP). (`ExportManifestLockFile` stays — the constant it returns survives; `ExportOsManifestLock` was deleted by T9 per R10, covered by step 1's grep.) Then delete `ExportNewOsLearnFS` (553-554) and the `ExportLogWarningToStderr` alias (line 70) from export_test.go.

- [ ] 3. **Delete from cli.go:** `osLearnFS` + its `Lock` method (33-57 remnant), `flockPath` (165-192), `logWarningToStderrf` (relocated block from L1 step 4). (`osFileReader` and `osManifestLock` are NOT deleted here — T8/T9 already did, per R5/R10; step 1's greps proved it.) Remove `"os"`, `"syscall"`, and `"path/filepath"` (if now unused — `filepath.Join` was only used by the deleted lock helpers; `vaultLockFromLocker` in deps_compose.go has its own import) from the import block. cli.go retains: package doc, `luhmannLockFile`/`manifestLockFile` constants (consumed by `vaultLockFromLocker` and the ingest cluster's manifest-lock composition + `ExportManifestLockFile`), `errNotADirectory` (consumed by `statDirFromFS`), `acquireOptionalLock` (152-163, pure), `listRootNotes` (pure since L1), `pathOf` (240-243).

- [ ] 4. Delete `TestOsLearnFS_Lock_BadVaultReturnsError` (learn_adapters_test.go:124-131) and `TestExportLogWarningToStderr_FormatsAndWrites` (os_adapters_test.go:150-172). (`TestOsManifestLock_MkdirError` was already deleted by T9 per R10 — verify with `rg -n "TestOsManifestLock_MkdirError" internal/` → zero hits.) Confirm the replacement coverage exists before deleting: the internal locker suite covers lock-open failure (`TestPrimLocker_OpenFailureWrapsWithPath`, locker_test.go — T1-rework; per the supersession map the cmd-integration-test obligation landed as internal `_test` files); the ingest cluster's manifest-lock composition test covers mkdir-before-lock; `TestNewLearnDeps_LogWarning_WritesToDepsStderr` covers the warning format.

- [ ] 5. Verify purity of the migrated files: `grep -n "\"os\"\|\"syscall\"\|os\.\|syscall\." internal/cli/cli.go internal/cli/learn.go internal/cli/qa.go internal/cli/deps_compose.go` → no hits. Run `targ test` → green, including `TestInvariant_K1_ConcurrentLearnNeverCollides` and `TestManifest_ConcurrentWritersDoNotLoseEntries`. Run `targ check-full` → clean. Run `targ check-thin-api` → PASS (`All N public API files are thin wrappers.`); this task adds no cmd/engram declarations, so any finding predates it — escalate per Global Constraints, never suppress. `go install ./cmd/engram` and re-run the step-11 smoke from L1.

- [ ] 6. Commit:

```
refactor(cli): delete os/syscall adapters from cli.go (#700)

osLearnFS, flockPath, and the stderr warning hook are gone from internal
(osFileReader/osManifestLock fell earlier to T8/T9 — grep-verified);
raw I/O enters only as cmd/engram primitives composed by cli.NewDeps.
The concurrent-writers regression suites keep racing real OS flocks
via T8's test-local testFlocker (test files are exempt from the
purity deny).

AI-Used: [claude]
```

---

**ADR-0013 survival summary:** lock semantics are preserved exactly — same lock files (`vault/.luhmann.lock` via `vaultLockFromLocker`, `chunksDir/.manifest.lock` on the ingest side), same acquisition points (`writeLearnUnderLock` learn.go:657, `writeQANotesUnderLock` qa.go:427, both inside `Run*` entry flows), same span (ListIDs→WriteNew under one exclusive flock), same O_EXCL backstop (`EdgeFS.WriteFileExcl`), same atomic temp+rename edge (`EdgeFS.WriteFileAtomic`). The two regression tests survive: `TestInvariant_K1_ConcurrentLearnNeverCollides` (adapted in L1 step 10 to drive the production `newLearnDeps` composition over a real flock — strictly stronger than before) and `TestManifest_ConcurrentWritersDoNotLoseEntries` (its Lock closures repointed by T8 to the test-local real-flock `testFlocker{}` per R7 — still real OS flocks; L2/T4 only grep-verifies `ExportFlockPath` is gone).

## Complete os/time inventory for the cluster (line → replacement)

| File:Line | Current | Replacement |
|---|---|---|
| query.go:14 | `"time"` import | RETAINED — type-only use (`time.Time`/`time.Duration`) after fix |
| query.go:66,274-276,282-284,671,1302-1307,1403-1410,1426 | injected-clock/type usage | unchanged (already DI) |
| query.go:1288 | `newOsEmbedDeps()` (os-backed via embed.go) | direct composition: `ScanVault(newVaultFS(d.FS), …)` + `d.FS.ReadFile` + `d.Embed` |
| query.go:1294 | `logWarningToStderrf` (os.Stderr via learn.go:333) | `logWarningTo(d.Stderr)` |
| query.go:1295 | `ListChunkIndexes: listJSONLIndexes` (os.ReadDir) | `listJSONLIndexes(d.FS)` |
| query.go:1296 | `Now: time.Now` | `Now: d.Now` |
| query_chunks.go:8 | `"os"` import | retained through T6 (transitional osListJSONLIndexes), deleted in T12 (R3); add `"io/fs"` |
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

