### Task T10 (M1): Atomic-write parity on the internal composition (relocation absorbed by T1-rework; consumers migrate in T12/T4/T15)

**Doctrine note (supersedes this task's original relocate-to-cmd brief):** under the revised composition doctrine there is no `cmd/engram/os_fs.go` to relocate onto — the ADR-0013 dance already lives INTERNAL as `primFS.WriteFileAtomic` (`internal/cli/edgefs.go`, landed by T1-rework; flag P-4 sequence CreateTemp→Chmod→WriteFile→Rename, + Remove on any failure), unit-tested with fake primitives (`TestEdgeFS_WriteFileAtomicHappyPathDance`, `TestEdgeFS_WriteFileAtomicFailuresRemoveTemp` in edgefs_test.go) and integration-tested with real ones (`TestRealEdgeFS_WriteFileAtomicReplacesContentAndCleansTemp` in primitives_integration_test.go). The supersession map's "T10 reduces to migrating its internal consumers onto `deps.FS.WriteFileAtomic`" is realized BY the owning tasks, not here: every remaining caller sits inside an os-backed constructor/adapter that its own task replaces wholesale — learn.go/qa.go (T3, already landed at this slot in R4's order), amend/resituate/activate/vocab (T12), cli.go's `osLearnFS.WriteSidecar` (T4), embed.go's `osEmbedFS.Write` (T15) — exactly T13's gate ledger, which is UNCHANGED. A standalone call-site flip in this task is meaningless (the enclosing adapters die whole). What the M1 slot still owes the cluster before T13 may delete writesafe.go + writesafe_test.go: re-prove, against the composed implementation over REAL primitives, the writesafe regression behaviors T1-rework did not carry over, and verify the caller ledger matches T13's gate.

Behavior-parity ledger (writesafe_test.go behavior → coverage on the composed `primFS`):

| writesafe_test.go behavior | Re-proven by | Landed in |
|---|---|---|
| `TestAtomicWriteFile_OverwritesExistingFile` | `TestRealEdgeFS_WriteFileAtomicReplacesContentAndCleansTemp` | T1-rework |
| `TestAtomicWriteFile_NoLeftoverTempFiles` | same test's closing `HaveLen(1)` temp-count assertion | T1-rework |
| `TestAtomicWriteFile_WritesNewFile` | `TestRealEdgeFS_WriteFileAtomicWritesNewFile` | THIS TASK |
| `TestAtomicWriteFile_FailureDoesNotTouchOriginal` | `TestRealEdgeFS_WriteFileAtomicCreateTempFailureLeavesOriginalUntouched` | THIS TASK |
| `TestAtomicWriteFile_RenameFailure_CleansTempAndLeavesOriginalUntouched` | `TestRealEdgeFS_WriteFileAtomicRenameFailureCleansTempAndOriginal` (real FS, injected `Rename` primitive — no export shim needed) plus the fake-prims rename case of `TestEdgeFS_WriteFileAtomicFailuresRemoveTemp` | THIS TASK |

**Files**
- Modify (append tests, one import, one const block, one sentinel var): `internal/cli/primitives_integration_test.go` (created by T1-rework)
- (No production edits. No deletions — writesafe.go keeps its callers until T12/T4/T15 migrate them; T13 deletes it.)

**Interfaces**
- Consumes: `cli.EdgeFS.WriteFileAtomic` through T1-rework's cli_test helpers — `realFSForTest()`, `realPrimitives()`, `fsFromPrims(...)` (edgefs_test.go; package `cli_test` is one namespace across files) — and const `realFSFilePerm`.
- Produces: the three THIS-TASK parity tests in the ledger; test-only consts `writableDirPerm`/`readOnlyDirPerm` and sentinel `errInjectedRename` (collision-checked free in package `cli_test`). No production symbols. (The original brief's cmd-side `osFS.WriteFileAtomic`/`doAtomicWrite` are NOT produced — nothing may reference them.)

**Steps**

1. [ ] Preflight — verify the T1-rework/T2 landed state this task builds on (any miss → STOP: an upstream task is incomplete; escalate rather than building the missing piece here):
   - `rg -n "func \(f primFS\) WriteFileAtomic" internal/cli/edgefs.go` → exactly one hit.
   - `rg -n "TestEdgeFS_WriteFileAtomicHappyPathDance|TestEdgeFS_WriteFileAtomicFailuresRemoveTemp" internal/cli/edgefs_test.go` → both present.
   - `rg -n "TestRealEdgeFS_WriteFileAtomicReplacesContentAndCleansTemp|func realPrimitives|func realFSForTest" internal/cli/primitives_integration_test.go` → all present.
   - `ls cmd/engram/` → `main.go` only (no os_fs.go / os_signal.go / debuglog_sink.go — the pre-rework layout is gone; a trivial wiring smoke test, if T2 kept one, is also acceptable).

2. [ ] Parity tests — relocation onto ALREADY-LANDED code, so this step is verify-form, not RED/GREEN: all three tests are expected GREEN on arrival. Any FAILURE is a genuine parity defect in the landed dance — STOP, fix `internal/cli/edgefs.go` under its unit suite, re-run; never bend the test to the defect. Append to `internal/cli/primitives_integration_test.go`, adding `"errors"` to its import block:

```go
func TestRealEdgeFS_WriteFileAtomicWritesNewFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")
	content := []byte("hello atomic world")
	fsys := realFSForTest()

	g.Expect(fsys.WriteFileAtomic(target, content, realFSFilePerm)).To(gomega.Succeed())

	got, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(got).To(gomega.Equal(content), "file must contain exactly the written bytes")
}

func TestRealEdgeFS_WriteFileAtomicCreateTempFailureLeavesOriginalUntouched(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	fsys := realFSForTest()

	// A read-only directory makes the CreateTemp primitive fail before the
	// dance writes a single byte (relocated writesafe_test.go behavior).
	subdir := filepath.Join(dir, "sub")
	g.Expect(os.Mkdir(subdir, writableDirPerm)).To(gomega.Succeed())

	target := filepath.Join(subdir, "original.txt")
	original := []byte("original untouched content")
	g.Expect(os.WriteFile(target, original, realFSFilePerm)).To(gomega.Succeed())

	g.Expect(os.Chmod(subdir, readOnlyDirPerm)).To(gomega.Succeed())

	// Restore permissions so TempDir cleanup can succeed.
	t.Cleanup(func() { _ = os.Chmod(subdir, writableDirPerm) })

	err := fsys.WriteFileAtomic(target, []byte("new content"), realFSFilePerm)
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("create temp")),
		"write into a read-only dir must fail at temp creation")

	// Make the directory readable again for the assertions.
	g.Expect(os.Chmod(subdir, writableDirPerm)).To(gomega.Succeed())

	got, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(got).To(gomega.Equal(original), "original file must be untouched after failure")

	tmpFiles, globErr := filepath.Glob(filepath.Join(subdir, ".original.txt.tmp-*"))
	g.Expect(globErr).NotTo(gomega.HaveOccurred())
	g.Expect(tmpFiles).To(gomega.BeEmpty(), "no leftover .tmp-* files must remain after failure")
}

func TestRealEdgeFS_WriteFileAtomicRenameFailureCleansTempAndOriginal(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	original := []byte("original content")

	g.Expect(os.WriteFile(target, original, realFSFilePerm)).To(gomega.Succeed())

	// Real primitives except Rename — parameterizing the dance over
	// Primitives is what lets this test inject the rename failure with no
	// export shim (replaces writesafe_test.go's injected-rename test).
	var tmpSeen string

	prims := realPrimitives()
	prims.Rename = func(oldPath, _ string) error {
		tmpSeen = oldPath

		return errInjectedRename
	}

	err := fsFromPrims(prims).WriteFileAtomic(target, []byte("new content"), realFSFilePerm)
	g.Expect(err).To(gomega.MatchError(errInjectedRename))
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("rename")),
		"error must name the failing dance step")

	got, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(got).To(gomega.Equal(original), "original must be untouched after rename failure")

	g.Expect(tmpSeen).NotTo(gomega.BeEmpty(), "the dance must reach the rename step")
	g.Expect(tmpSeen).NotTo(gomega.BeAnExistingFile(), "temp file must be removed by the failure cleanup")

	tmpFiles, globErr := filepath.Glob(filepath.Join(dir, ".file.txt.tmp-*"))
	g.Expect(globErr).NotTo(gomega.HaveOccurred())
	g.Expect(tmpFiles).To(gomega.BeEmpty(), "no leftover .tmp-* files must remain after failure")
}

// errInjectedRename is the sentinel injected by the rename-failure parity test.
var errInjectedRename = errors.New("injected rename failure")

// Directory permission modes for the read-only-dir parity test.
const (
	writableDirPerm fs.FileMode = 0o700
	readOnlyDirPerm fs.FileMode = 0o500
)
```

3. [ ] Run `targ test` — expect PASS (the three parity tests green against the landed dance; internal/cli/writesafe_test.go's originals also still green — they are deleted by T13, not here).

4. [ ] Caller-ledger check (verify-only, ZERO edits): `grep -rn "atomicWriteFile(" internal/cli --include="*.go" | grep -v _test | grep -v writesafe.go` → exactly the six callers T13's gate assigns to later tasks (current numbering): amend.go:351 + resituate.go:169 + activate.go:136 + vocab_commands.go:1217 (→ T12), cli.go:144 (→ T4), embed.go:164 (→ T15). A learn.go or qa.go hit → T3 incomplete, STOP. Any caller OUTSIDE this ledger → T13's gate can never clear; escalate to the orchestrator with the exact hit (do not migrate it ad hoc — the owning task's brief must absorb it).

5. [ ] Run `targ check-full` — expect clean. Run `targ check-thin-api` — expect PASS (this task touches no cmd code, so any finding predates it → escalate per Global Constraints, never suppress).

6. [ ] Commit:

```
test(cli): prove writesafe parity on internal atomic write (#700)

The ADR-0013 dance lives on internal/cli's primFS.WriteFileAtomic
(landed by the T1 rework). This closes the writesafe_test.go parity
gap with real-primitive regression tests — create-temp failure leaves
the original untouched, injected rename failure cleans the temp, and
new-file write — so the purge task (T13) can delete writesafe.go with
zero behavior-coverage loss. Call-site migration stays with the owning
tasks (T12/T4/T15) per T13's gate.

AI-Used: [claude]
```

---

