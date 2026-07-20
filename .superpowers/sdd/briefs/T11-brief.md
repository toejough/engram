# DISPATCH HEADER (orchestrator)

- Worktree: `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity` (branch `worktree-700-internal-purity`). Work ONLY here — never cd to the main checkout.
- BASE-T11: 80430e8f (T10 complete). Constraints mirror: `.superpowers/sdd/constraints-and-resolutions.md` — READ IT FIRST; its supersession map governs over plan-prose flags.
- ACCUMULATED DISPATCH NOTES (binding):
  - threaded variable is `deps` not `d`; adapt brief snippets mechanically
  - EdgeFS-layer error wraps: distinct-word/no-path ("list md: %w", "vault read: %w") — never repeat EdgeFS's verb+path.
  - **STALE SNIPPET — orchestrator-verified fix (binding):** the plan's `TestNewVaultFS_ReadFile_WrapsErrorWithPath` predates T5's wrap-convention fix. The LANDED `vaultFS.ReadFile` (vault_fs.go:47-50) wraps `"vault read: %w"` with NO path — the snippet's `ContainSubstring("/vault/x.md")` assertion would FAIL. Write it as `TestNewVaultFS_ReadFile_WrapsWithDistinctVerb`: keep `MatchError(errInjectedCompose)`, replace the path assertion with `MatchError(ContainSubstring("vault read"))`. Do NOT edit production code to make the stale snippet pass.
  - test builders: newTestDeps(stdout,stderr) [flows through NewDeps] + realFSForTest(); realFSDepsForTest/osTestEdgeFS DELETED
  - writeAtomicFromFS(fsys, opName) — perm param removed, atomicFilePerm inside
  - gates run FOREGROUND (no background-run-and-yield); stage EXPLICIT paths only (never `git add -A`/`-u` — other agents may have in-flight files)
  - check-full residual set (NOT yours to fix): e2e-under-load coverage flake (re-run check-coverage-for-fail standalone to confirm) + 2 dev/eval please_step3_probe reorder fixtures; lint-full must be 0
- Name-collision status (orchestrator-verified): `fakeEdgeFS`/`fakeDirEntry`/`fakeLocker` exist ONLY in `ingest_family_deps_test.go`, which is package `cli_test` — no collision with your package-`cli` files (only export_test.go and resituate_internal_test.go are package `cli`, and neither claims these names). Declare the plan's fakes as written. General rule stands (T10 lesson): before declaring any OTHER helper name, `rg` it across `internal/cli/` and check the claiming file's package line.
- The `Primitives` struct in `internal/cli/primitives.go` is authoritative for the step-5 literal's field set — reconcile the snippet against it. If a snippet field doesn't exist on the struct (or the struct has a required field the snippet lacks), STOP and escalate; do not invent or drop fields silently.
- Step-6 contingency: if the unused linter flags `ExportNewTestOsDeps` (unreferenced until T12), STOP and report — landing M2+M3 together is an orchestrator call, not yours. Never suppress.
- House rules: `t.Parallel()` on every test; gomega + nilaway guards; named constants; descriptive names; <120 char lines; run `targ reorder-decls` on created files if the check flags section order.
- REPORT: write `.superpowers/sdd/briefs/T11-report.md` BEFORE your final message — status, commit SHA(s), gate outcomes verbatim (test / check-full / check-thin-api + standalone re-runs), every deviation with rationale, concerns/watch items. Final message: STATUS line, SHAs, one-paragraph summary, concerns.

---

### Task T11 (M2): os-backed test Deps + contract tests over the landed canonical helpers

**Files**
- Create: `internal/cli/deps_compose_internal_test.go` (fake-driven contract tests over the LANDED canonical helpers)
- Create: `internal/cli/testdeps_test.go` (`ExportNewTestOsDeps` built through `cli.NewDeps` over a real-OS `Primitives` literal — NO hand-rolled adapter mirrors; composition doctrine)
- Verify-only (NO edit — R1): `internal/cli/deps_compose.go`. T3 created it and it already carries everything this task's draft once declared: the four parallel-drafted helpers (`edgeVaultFS`, `vaultLuhmannLock`, `warnLoggerTo`, `jsonlIndexesLister`) are LOSERS per R1/R2/R3 — their canonical equivalents are T5's `newVaultFS(fsys EdgeFS)` (vault_fs.go), T3's `vaultLockFromLocker`/`logWarningTo` (deps_compose.go), and T6's `listJSONLIndexes(fsys EdgeFS)` (query_chunks.go). T11 appends NOTHING to deps_compose.go; NEVER apply a full-file `package cli` replacement (it would clobber T3's landed `vaultLockFromLocker`/`logWarningTo`/`statDirFromFS`/`initVaultFromFS`/`list*FromFS`/`write*FromFS` helpers).

**Interfaces**
- Consumes: `type Deps struct{...}`, `type EdgeFS interface{...}`, `type FileLocker interface{...}` from internal/cli/deps.go (foundation task — M2 is blocked on it); `luhmannLockFile` const (internal/cli/cli.go:16, pure, stays); the landed canonical helpers `newVaultFS` (T5), `listJSONLIndexes` (T6), `vaultLockFromLocker` + `logWarningTo` (T3) — all in place before T11 per R4; `NewDeps`, `Primitives` (T1-rework/T2 — the base struct already carries the `WriteFileExcl` exclusive-create primitive, survivor S-1), and `WriteSyncer` for the step-5 literal.
- Produces:
  - Test-only: `func ExportNewTestOsDeps() Deps` (package cli, _test.go file; body is one `NewDeps` call over an inline real-OS `Primitives` literal) — this task's ONLY new symbol.
  - Additional fake-driven test coverage of the canonical helpers' contracts (wrapped-ErrNotExist handling, lock path, warning format).

**Steps**

1. [ ] Create `internal/cli/deps_compose_internal_test.go` (package `cli`, internal-test precedent: resituate_internal_test.go). These tests drive the LANDED canonical helpers (T3/T5/T6 — no new production code in this task, so there is no compile-RED; they add fake-driven contract coverage the real-FS suites can't express as directly):

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

func TestNewVaultFS_ListMD_FiltersToMDFilesSkippingDirs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vfs := newVaultFS(fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			fakeDirEntry{name: "1.2026-01-01.note.md"},
			fakeDirEntry{name: "sidecar.vec.json"},
			fakeDirEntry{name: "subdir", dir: true},
		}, nil
	}})

	names, err := vfs.ListMD("/vault")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(names).To(Equal([]string{"1.2026-01-01.note.md"}))
}

func TestNewVaultFS_ListMD_MissingDirIsEmptyNotError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vfs := newVaultFS(fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return nil, fmt.Errorf("read dir: %w", fs.ErrNotExist)
	}})

	names, err := vfs.ListMD("/missing")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(names).To(BeEmpty())
}

func TestNewVaultFS_ReadFile_WrapsErrorWithPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vfs := newVaultFS(fakeEdgeFS{readFile: func(string) ([]byte, error) {
		return nil, errInjectedCompose
	}})

	_, err := vfs.ReadFile("/vault/x.md")
	g.Expect(err).To(MatchError(errInjectedCompose))
	g.Expect(err).To(MatchError(ContainSubstring("/vault/x.md")))
}

func TestListJSONLIndexes_FiltersAndTreatsMissingDirAsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := listJSONLIndexes(fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			fakeDirEntry{name: "s.jsonl"},
			fakeDirEntry{name: "manifest.json"},
			fakeDirEntry{name: "nested", dir: true},
		}, nil
	}})

	paths, err := lister("/chunks")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(paths).To(Equal([]string{"/chunks/s.jsonl"}))

	missing := listJSONLIndexes(fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return nil, fmt.Errorf("read dir: %w", fs.ErrNotExist)
	}})

	paths, err = missing("/gone")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(paths).To(BeEmpty())
}

func TestVaultLockFromLocker_LocksVaultLuhmannLockFileAndReleases(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var lockedPath string

	unlocked := false
	locker := fakeLocker{lock: func(path string) (func() error, error) {
		lockedPath = path

		return func() error { unlocked = true; return nil }, nil
	}}

	lock := vaultLockFromLocker(locker)

	release, err := lock("/vault")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(lockedPath).To(Equal("/vault/.luhmann.lock"))

	release()
	g.Expect(unlocked).To(BeTrue())
}

func TestLogWarningTo_FormatsWithWarningPrefixAndNewline(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	logWarningTo(&buf)("amend: %s failed after %d tries", "embed", 2)
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

2. [ ] Run `targ test` — expect PASS immediately (the canonical helpers already landed: `vaultLockFromLocker`/`logWarningTo` in T3's deps_compose.go, `newVaultFS` in T5's vault_fs.go, `listJSONLIndexes` in T6's query_chunks.go). There is NO compile-RED in this task because it adds no production code. If any of the four is undefined, an earlier task did not land — STOP and fix the ordering, do NOT declare the missing helper here.

3. [ ] Verify deps_compose.go needs nothing from this task (R1 guard — verify, never edit): `rg -n "func vaultLockFromLocker|func logWarningTo" internal/cli/deps_compose.go` → both present; `rg -n "func newVaultFS" internal/cli/vault_fs.go` → present; `rg -n "func listJSONLIndexes" internal/cli/query_chunks.go` → present. Also confirm no loser symbol was ever declared: `rg -n "edgeVaultFS|jsonlIndexesLister|vaultLuhmannLock|warnLoggerTo" internal/` → zero hits outside this plan's prose (i.e., none in the tree).

4. [ ] (Absorbed into steps 2-3 — no separate GREEN; this task's only production-adjacent artifact is the test-only Deps in step 5.)

5. [ ] Create `internal/cli/testdeps_test.go` (package `cli` — test file, exempt from purity enforcement). Under the composition doctrine this file contains NO adapter implementations: the production `primFS`/`primLocker`/debug-sink impls live in internal (T1-rework) and are reached only through `NewDeps`. The ONLY declaration is `ExportNewTestOsDeps`; its real-OS `Primitives` literal mirrors `cmd/engram/main.go`'s (doctrine flag DRIFT — cli_test.go's end-to-end binary tests guard the production literal; a package-`cli` test file cannot consume cli_test's `realPrimitives()`, hence this second one-screen mirror) minus `StartSignalPulses`, which stays nil so `startForceExit` skips (SIG-1: tests never subscribe process signals). Primitive closures return RAW os errors BY CONTRACT — the `%w` wrap happens exactly once, inside `primFS`/`primLocker`; do NOT add wrapping here (it would double-wrap), and if a linter flags the raw returns in this _test file, escalate rather than wrap. `Deps.Embed` comes out as the NewDeps-wired lazy embedder — production parity with today's `sharedEmbedder`-backed constructors (nothing loads until an embed call); wiring tests needing determinism override the per-command deps field after construction (existing patterns: `fakeEmbedder`, `successEmbedder`). The ADR-0013 concurrent-manifest regression rides `primFS.WriteFileAtomic` — the REAL production dance, not a copy:

```go
package cli

// Test-only composition of production Deps over real OS primitives, so
// wiring-integration tests drive the exact primFS/primLocker/debug-sink
// implementations the binary ships. Composition doctrine (#700): NewDeps is
// the single composition root — no hand-rolled adapter mirrors anywhere.

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// ExportNewTestOsDeps returns production-composed Deps for wiring tests:
// one NewDeps call over an inline real-OS Primitives literal (mirrors
// cmd/engram/main.go's — doctrine flag DRIFT — minus StartSignalPulses,
// nil so startForceExit skips per SIG-1). Closures return raw os errors by
// primitive contract; primFS/primLocker add the single %w wrap.
func ExportNewTestOsDeps() Deps {
	return NewDeps(Primitives{
		ReadFile:    os.ReadFile,
		WriteFile:   os.WriteFile,
		MkdirAll:    os.MkdirAll,
		MkdirTemp:   os.MkdirTemp,
		Stat:        os.Stat,
		ReadDir:     os.ReadDir,
		Remove:      os.Remove,
		RemoveAll:   os.RemoveAll,
		Rename:      os.Rename,
		WalkDir:     filepath.WalkDir,
		Chmod:       os.Chmod,
		Getenv:      os.Getenv,
		Now:         time.Now,
		Getwd:       os.Getwd,
		UserHomeDir: os.UserHomeDir,
		WriteFileExcl: func(path string, data []byte, perm fs.FileMode) error {
			//nolint:gosec // test helper, path from test
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
			if err != nil {
				return err
			}

			_, err = file.Write(data)
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}

			return err
		},
		OpenLockFile: func(path string, perm fs.FileMode) (uintptr, error) {
			fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR, uint32(perm))

			return uintptr(fd), err
		},
		FlockExclusive: func(fd uintptr) error {
			return syscall.Flock(int(fd), syscall.LOCK_EX)
		},
		FlockUnlock: func(fd uintptr) error {
			return syscall.Flock(int(fd), syscall.LOCK_UN)
		},
		CloseFD: func(fd uintptr) error {
			return syscall.Close(int(fd))
		},
		OpenDebugFile: func(path string, perm fs.FileMode) (WriteSyncer, error) {
			return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm) //nolint:gosec // test helper, operator-controlled path
		},
	}, os.Stdout, os.Stderr, func(int) {})
}
```

6. [ ] Run `targ test` then `targ check-full` — expect PASS/clean. Run `targ check-thin-api` — expect PASS (this task adds internal test files only; per Global Constraints any finding escalates to the orchestrator, never suppressed). (`ExportNewTestOsDeps` is unreferenced until M3; if the unused linter flags it, land M2+M3 as one PR — do not suppress.)
7. [ ] Commit:

```
test(cli): NewDeps-composed test Deps + compose-contract tests (#700)

ExportNewTestOsDeps builds Deps through cli.NewDeps over real OS
primitives — the same primFS/primLocker composition the binary ships —
so wiring tests exercise production composition, not a hand-rolled os
adapter mirror (composition doctrine). Fake-driven contract tests cover
the landed canonical helpers (newVaultFS, listJSONLIndexes,
vaultLockFromLocker, logWarningTo). The M2 draft's own helpers were
losers per R1/R2/R3 and were never declared.

AI-Used: [claude]
```

---

