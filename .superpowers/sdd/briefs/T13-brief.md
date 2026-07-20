# DISPATCH HEADER (orchestrator)

- Worktree: `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity` (branch `worktree-700-internal-purity`). Work ONLY here — never cd to the main checkout.
- BASE-T13: b3a0f925 (T7 complete; docs-only ledger commits atop it are fine). Constraints mirror: `.superpowers/sdd/constraints-and-resolutions.md` — READ IT FIRST; supersession map governs.
- **LEDGER CORRECTION (twice-verified at T10, confirmed at T12/T15):** the migration count was 5, not the GATE paragraph's 6 — cli.go:144 never reached T4; T3's writeAtomicFromFS absorbed it. History: learn.go+qa.go (T3), amend/resituate/activate/vocab (T12), embed.go (T15). The gate GREP is unchanged and was already verified EMPTY at T15 close — re-run it anyway; non-empty = STOP.
- ACCUMULATED DISPATCH NOTES (binding):
  - **T4 lesson:** before deleting ANY symbol, `rg` it across `internal/` and `cmd/` INCLUDING `_test.go` files — the brief's Files list may under-enumerate test consumers of `atomicWriteFile`/`doAtomicWrite`/`ExportAtomicWriteFile`/`ExportDoAtomicWrite`; a missed consumer is a compile-forced deviation to handle and report, not a STOP.
  - Verify-only files are VERIFY-ONLY (ingest_test.go, edgefs_test.go, primitives_integration_test.go — the surviving atomic-write coverage, incl. T10's three parity tests, must remain untouched and passing).
  - **reorder-decls HAZARD:** `targ reorder-decls` is UNSCOPED — rewrites the 2 protected dev/eval please_step3_probe fixtures; if run, `git restore` those two paths explicitly afterward.
  - gates run FOREGROUND (no background-run-and-yield); stage EXPLICIT paths only (never `git add -A`/`-u`)
  - check-full residual set (NOT yours to fix): e2e-under-load coverage flake (re-run check-coverage-for-fail standalone to confirm) + the 2 dev/eval reorder fixtures; lint-full must be 0
- House rules: gomega + nilaway guards on touched tests; <120 char lines.
- REPORT: write `.superpowers/sdd/briefs/T13-report.md` BEFORE your final message — status, commit SHA(s), verbatim gate outcomes, every deviation with rationale, concerns. Final message: STATUS line, SHAs, one-paragraph summary, concerns.

---

### Task T13 (M4): Purge internal atomic write (gated)

**GATE (do not start until true; per R4 this task runs after T15):** `grep -rn "atomicWriteFile" internal/cli --include="*.go" | grep -v _test | grep -v writesafe.go` returns EMPTY — i.e. every internal caller has been migrated by its own task: learn.go:371 + qa.go:283 (T3), amend.go:351 + resituate.go:169 + activate.go:136 + vocab_commands.go:1217 (T12), cli.go:144 (T4), and embed.go:164 (`osEmbedFS.Write` — deleted by T15, the LAST caller standing, which is why R4 orders T13 after T15).

**Files**
- Delete: `internal/cli/writesafe.go`, `internal/cli/writesafe_test.go`
- Modify: `internal/cli/export_test.go` (remove two shims)
- Verify-only (no edit): `internal/cli/ingest_test.go` (`realFS.write` already repointed by T8 step 6 — step 1 verifies); `internal/cli/edgefs_test.go` + `internal/cli/primitives_integration_test.go` (the surviving atomic-write coverage — step 2 verifies presence, no edits)

**Interfaces**
- Removes: `atomicWriteFile`, `doAtomicWrite`, `ExportAtomicWriteFile`, `ExportDoAtomicWrite` from internal/cli.

**Steps**

1. [ ] Verify the ADR-0013 concurrent-manifest regression infra is already repointed (must survive per spec — T8 step 6 moved `realFS.write` off `cli.ExportAtomicWriteFile` onto its test-local `testAtomicWrite`, which carries the same real temp+rename semantics; no edit here). Check ingest_test.go's `realFS.write` reads:

```go
func (r *realFS) write(_, path string, data []byte) error {
	return testAtomicWrite(path, data, 0o600)
}
```

and `rg -n "ExportAtomicWriteFile" internal/cli/ingest_test.go` → zero hits. Any `ExportAtomicWriteFile` reference remaining outside writesafe_test.go → T8 incomplete, STOP.

2. [ ] Delete `internal/cli/writesafe.go` and `internal/cli/writesafe_test.go`. All five writesafe behaviors live on INTERNALLY against the composed `primFS.WriteFileAtomic` (internal/cli/edgefs.go, landed by T1-rework — the revised doctrine's relocation target; nothing relocates to cmd/engram, which holds only the declaration-free main.go): the fake-prims dance suite (`TestEdgeFS_WriteFileAtomicHappyPathDance`, `TestEdgeFS_WriteFileAtomicFailuresRemoveTemp`, `TestEdgeFS_WriteFileAtomicUniqueNameRetry` — edgefs_test.go) plus the real-primitive suite (`TestRealEdgeFS_WriteFileAtomic*` — primitives_integration_test.go), completed by T10's parity tests per T10's behavior-parity ledger. Verify before deleting: `rg -n "TestRealEdgeFS_WriteFileAtomicWritesNewFile|TestRealEdgeFS_WriteFileAtomicExclCreateFailureLeavesOriginalUntouched|TestRealEdgeFS_WriteFileAtomicRenameFailureCleansTempAndOriginal" internal/cli/primitives_integration_test.go` → all three present (any miss → T10 incomplete, STOP).

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
5. [ ] Run `targ test` — expect PASS, including the ingest concurrent-writers regression test (its lock is still real flock via T8's test-local `testFlocker` — R7 — and its writer is T8's `testAtomicWrite`) and the surviving internal atomic-write suites named in step 2.
6. [ ] Run `targ check-full` — expect clean. Run `targ check-thin-api` — expect PASS (this task touches no cmd code; per Global Constraints any finding escalates to the orchestrator, never suppressed).
7. [ ] Commit:

```
refactor(cli): delete internal atomic-write (#700)

writesafe.go's dance now lives solely on internal/cli/edgefs.go's
primFS.WriteFileAtomic, composed from Primitives and covered by the
fake-prims dance suite plus the real-primitive parity suite (T10's
ledger carries all five writesafe behaviors). The ADR-0013
concurrent-manifest regression test writes through T8's test-local
testAtomicWrite with identical temp+rename semantics.

AI-Used: [claude]
```

---

Key file paths: /Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/internal/cli/{writesafe.go, writesafe_test.go, edgefs.go, edgefs_test.go, primitives_integration_test.go, amend.go, resituate.go, activate.go, vocab.go, vocab_commands.go, vault_init.go, cli.go, targets.go, export_test.go, ingest_test.go, learn_test.go}, /Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/cmd/engram/main.go.

