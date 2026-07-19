### Task T12 (M3): Maintenance-family constructors compose from Deps

**Files**
- Modify: `internal/cli/amend.go`, `internal/cli/resituate.go`, `internal/cli/activate.go`, `internal/cli/vocab_commands.go`
- Modify: `internal/cli/query_chunks.go` (delete the transitional `osListJSONLIndexes` + its `"os"` import — amend.go, converted here, was its last consumer; step 8, grep-gated per R3)
- Modify: `internal/cli/export_test.go`, `internal/cli/activate_test.go`, `internal/cli/amend_test.go`, `internal/cli/resituate_test.go`, `internal/cli/vocab_commands_test.go`, `internal/cli/vocab_trigger_test.go`, `internal/cli/learn_test.go` (one line — cross-cluster, see flag)
- Modify (call expressions only; signature threading owned by wiring cluster): `internal/cli/targets.go`
- Verify-only (no edits): `internal/cli/vocab.go`, `internal/cli/vault_init.go`

**Interfaces**
- Consumes: `Deps` (deps.go), the canonical composition helpers — T5's `newVaultFS(d.FS)`, T3's `vaultLockFromLocker(d.Lock)` + `logWarningTo(d.Stderr)` (deps_compose.go), T6's `listJSONLIndexes(fsys EdgeFS)` curried lister — plus `d.FS.WriteFileAtomic`, `d.Embed embed.Embedder`. (The M2 draft's `edgeVaultFS`/`vaultLuhmannLock`/`warnLoggerTo` are losers per R1/R2 and exist nowhere.)
- Produces: `func newAmendDeps(d Deps) AmendDeps`, `func newResituateDeps(d Deps) ResituateDeps`, `func newActivateDeps(d Deps) ActivateDeps`, `func newVocabDeps(d Deps) VocabDeps`, `func newVocabStatsDeps(d Deps) VocabStatsDeps` — replacing `newOsAmendDeps()`, `newOsResituateDeps()`, `newOsActivateDeps()`, `newOsVocabDeps()`, `newOsVocabStatsDeps()`. Deletes `osWriteSidecar` and (per R3, grep-gated) the transitional `osListJSONLIndexes`.

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

	vfs := newVaultFS(d.FS)

	return AmendDeps{
		Lock: vaultLockFromLocker(d.Lock),
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(vfs, vault)
		},
		Read: vfs.ReadFile,
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
		// listJSONLIndexes(d.FS) lists *.jsonl chunk indexes, treats an absent
		// dir as empty (not an error), and never matches manifest.json —
		// exactly the contract the transitional os-backed osListJSONLIndexes
		// provided (deleted in step 8 now that this, its last consumer, flips).
		ListIndexes: listJSONLIndexes(d.FS),
		LogWarning:  logWarningTo(d.Stderr),
		// Vocab assignment wiring: no-op when the vault has no term notes.
		// Uses stored member centroids (vocab.centroids.json) when present,
		// falling back to description embeddings per term.
		LoadTermVectors: func(vault string) ([]TermWithVector, error) {
			return loadAssignmentTermVectors(vault, vfs.ListMD, vfs.ReadFile)
		},
		// ListMD provides full .md filenames for the vocab trigger scan.
		// Must use ListMD (not stripped basenames) — basename filtering causes
		// false-fire on the untagged trigger.
		ListMD: vfs.ListMD,
	}
}
```
Doc-comment touch-ups in the same file: AmendDeps struct comment line 43 `The production wiring in newOsAmendDeps supplies os.ReadDir/os.ReadFile via closures.` → `The production wiring in newAmendDeps supplies the injected EdgeFS via closures.`; Lock field comment line 47 `Wired to vaultFS.Lock in newOsAmendDeps.` → `Wired via vaultLockFromLocker in newAmendDeps.`

4. [ ] resituate.go. Replace `newOsResituateDeps` (resituate.go:155-184) with:

```go
// newResituateDeps composes RunResituate's dependencies from the injected
// edge Deps (pure composition — no direct I/O; #700).
func newResituateDeps(d Deps) ResituateDeps {
	const perm = 0o600

	vfs := newVaultFS(d.FS)

	return ResituateDeps{
		Lock: vaultLockFromLocker(d.Lock),
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(vfs, vault)
		},
		Read: vfs.ReadFile,
		Write: func(path string, data []byte) error {
			err := d.FS.WriteFileAtomic(path, data, perm)
			if err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}

			return nil
		},
		Embedder: d.Embed,
		LoadTermVectors: func(vault string) ([]TermWithVector, error) {
			return loadAssignmentTermVectors(vault, vfs.ListMD, vfs.ReadFile)
		},
		ListMD:     vfs.ListMD,
		LogWarning: logWarningTo(d.Stderr),
		Now:        d.Now,
	}
}
```
ResituateDeps.Lock comment line 28-29 `Wired to vaultFS.Lock in newOsResituateDeps.` → `Wired via vaultLockFromLocker in newResituateDeps.`

5. [ ] activate.go. Delete the `os` import; replace `newOsActivateDeps` + `osWriteSidecar` (activate.go:120-137) with:

```go
// newActivateDeps composes RunActivate's dependencies from the injected edge
// Deps (pure composition — no direct I/O; #700). Sidecar writes go through
// WriteFileAtomic (temp+rename) so concurrent readers always see either the
// old or new file.
func newActivateDeps(d Deps) ActivateDeps {
	const sidecarPerm = 0o600

	return ActivateDeps{
		Lock: vaultLockFromLocker(d.Lock),
		Now:  d.Now,
		Read: d.FS.ReadFile,
		Write: func(path string, data []byte) error {
			return d.FS.WriteFileAtomic(path, data, sidecarPerm)
		},
		LogWarning: logWarningTo(d.Stderr),
	}
}
```
Comment touch-ups: ActivateDeps.Lock comment line 23 `Wired to vaultFS.Lock in newOsActivateDeps.` → `Wired via vaultLockFromLocker in newActivateDeps.`; bumpLastUsed comment lines 86-87 `Sidecar writes go through atomicWriteFile (temp+rename) AND RunActivate holds the vault flock` → `Sidecar writes go through the injected atomic write (WriteFileAtomic, temp+rename) AND RunActivate holds the vault flock`.

6. [ ] vocab_commands.go. Delete the `os` import; replace `newOsVocabDeps` + `newOsVocabStatsDeps` (vocab_commands.go:1208-1240) with (behavior parity: WriteFile/DeleteFile error text preserved; WriteSidecar keeps osEmbedFS.Write's `"write: %w"` wrap):

```go
// newVocabDeps composes VocabDeps from the injected edge Deps (pure
// composition — no direct I/O; #700).
func newVocabDeps(d Deps) VocabDeps {
	const sidecarPerm = 0o600

	vfs := newVaultFS(d.FS)

	return VocabDeps{
		Lock:     vaultLockFromLocker(d.Lock),
		ListMD:   vfs.ListMD,
		ReadFile: vfs.ReadFile,
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
		LogWarning: logWarningTo(d.Stderr),
		Now:        d.Now,
	}
}

// newVocabStatsDeps composes the read-only vocab stats deps from the injected
// edge Deps.
func newVocabStatsDeps(d Deps) VocabStatsDeps {
	vfs := newVaultFS(d.FS)

	return VocabStatsDeps{
		ListMD:   vfs.ListMD,
		ReadFile: vfs.ReadFile,
	}
}
```

- [ ] 6.5. **Migrate the `ExportNewOsVaultFS` call sites (R12 — this task owns the vocab test files).** Replace every `osFS := cli.ExportNewOsVaultFS()` with `osFS := cli.ExportNewVaultFS(osTestEdgeFS{})` (T5's export over the cli_test real-FS EdgeFS double — same `ListMD`/`ReadFile` shape and semantics; `osTestEdgeFS` lives in T5's edgefs_os_test.go, same package cli_test). Sites (verified in the pristine tree; locate by the call expression, not line): vocab_trigger_test.go:251, 441; vocab_commands_test.go:96, 131, 198, 231, 543, 559, 613, 651, 3856, 3874. After the edit: `rg -n "ExportNewOsVaultFS" internal/cli --include='*_test.go'` → hits ONLY in export_test.go (the shim definition, which T7 deletes). Without this step T7's shim deletion is a compile break its gate grep cannot see (R12).

7. [ ] targets.go call-expression updates (coordinate with wiring cluster's `deps Deps` threading through `amendResituateTargets`/`ingestQueryTargets`/`vocabTargets`; only the constructor expressions belong to this task):
   - line 108: `newOsResituateDeps()` → `newResituateDeps(deps)`
   - line 113: `newOsAmendDeps()` → `newAmendDeps(deps)`
   - line 173: `newOsActivateDeps()` → `newActivateDeps(deps)`
   - lines 278/286/290: `newOsVocabDeps()` → `newVocabDeps(deps)`
   - line 282: `newOsVocabStatsDeps()` → `newVocabStatsDeps(deps)`

8. [ ] Delete the transitional lister (per R3 — amend.go, converted in step 3, was its LAST consumer). Gate first: `grep -rn "osListJSONLIndexes" internal/ --include='*.go'` — expected: the definition in query_chunks.go ONLY (step 3 already removed amend.go's reference). Any hit in another file → STOP: that file's task has not landed; do not delete. Then delete `osListJSONLIndexes` (func + doc comment) from query_chunks.go and its now-unused `"os"` import. Verify: re-run the grep — zero hits; `grep -n '"os"\|os\.' internal/cli/query_chunks.go` — no output (query_chunks.go fully pure as of this task).
9. [ ] Run `targ test` — expect PASS: the relocated wiring-integration tests (activate/amend/resituate/vocab against real t.TempDir vaults) prove the composed deps behave identically; resituate tests still inject `successEmbedder{}`; vocab tests still override `deps.Embedder = &fakeEmbedder{}`. The executed targets-level tests riding this task's flips (vocab bootstrap/propose/refit/stats via targ.Execute, activate/resituate/amend via executeForTest) dereference `d.FS` and — on the propose success path — `d.Lock`; both fields are already in `newTestDeps` since T3 (R11), and `Embed` stays nil (vocab's embed path skips on nil Embedder at vocab_commands.go:833).
10. [ ] Purity verification for this cluster (enforcement task lands later; this is the leave-nothing-behind check the central spec demands):
   - `grep -n "\"os\"\|os\.\|syscall\|time\.Now\|time\.Since\|time\.Tick" internal/cli/amend.go internal/cli/resituate.go internal/cli/activate.go internal/cli/vocab.go internal/cli/vocab_commands.go internal/cli/vault_init.go` — expected: NO import of `os`/`syscall`, no `time.Now/Since/Tick` calls; only comment mentions (scrub remaining comment references: amend.go:43 handled in step 3; vocab_commands.go:1126 `os.ReadDir sorts by name` → reword to `the OS-backed lister sorts by name`; resituate.go:128 `wiring provides time.Now` → `wiring provides the injected clock`).
   - Verify-only: vocab.go and vault_init.go unchanged (imports already pure; `fs.FileMode` from io/fs stays per spec).
11. [ ] Run `targ check-full` — expect clean (lint + coverage; the composed constructors are covered by the wiring tests, matching the coverage intent behind the old named `osWriteSidecar`/`logWarningToStderrf` pattern). Run `targ check-thin-api` — expect PASS (`All N public API files are thin wrappers.`); this task adds no cmd/engram declarations, so any finding predates it — escalate per Global Constraints, never suppress.
12. [ ] Commit:

```
refactor(cli): compose maintenance deps from Deps (#700)

newAmendDeps/newResituateDeps/newActivateDeps/newVocabDeps/newVocabStatsDeps
replace their newOsXxx forms: flock via FileLocker (.luhmann.lock at Run*
entry only, ADR-0013), atomic note/sidecar writes via EdgeFS.WriteFileAtomic,
clock via Deps.Now, warnings via Deps.Stderr, embedder via Deps.Embed.
activate.go and vocab_commands.go drop their os imports; vocab.go and
vault_init.go verified already pure. The transitional osListJSONLIndexes
(T6) dies here with query_chunks.go's os import — amend was its last
consumer (grep-gated).

AI-Used: [claude]
```

---

