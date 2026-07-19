### Task T3 (L1): learn-family — compose LearnDeps/LearnQADeps purely from cli.Deps

**Files**
- Create: `internal/cli/deps_compose.go` (shared EdgeFS/FileLocker composition helpers)
- Create: `internal/cli/deps_compose_test.go` (RED tests for the compositions)
- Modify: `internal/cli/deps.go` (foundation file: add `WriteFileExcl` to EdgeFS — flagged addition)
- Modify: `internal/cli/primitives.go` (add `Primitives.OpenExcl` — the flag-X-1 exclusive-create primitive)
- Modify: `internal/cli/edgefs.go` (add `primFS.WriteFileExcl` — open/write/close lifecycle internal)
- Modify: `internal/cli/edgefs_test.go` (fake-primitive contract tests for `WriteFileExcl`)
- Modify: `internal/cli/primitives_integration_test.go` (real-FS `WriteFileExcl` → `fs.ErrExist` round-trip; extend `realPrimitives()` with the `OpenExcl` closure)
- Modify: `cmd/engram/main.go` (ONE new field line in the `cli.Primitives` literal — the `OpenExcl` single-call closure — plus its `io` import; nothing else)
- Modify: `internal/cli/learn.go` (delete `newOsLearnDeps` + `logWarningToStderrf`; add `newLearnDeps(d Deps)`; re-sign `runLearnFrom*Args`; drop `os` import)
- Modify: `internal/cli/qa.go` (delete `newOsLearnQADeps`; add `newQaDeps(d Deps)`; drop `os` import)
- Modify: `internal/cli/cli.go` (re-parameterize `listRootNotes`; shrink `osLearnFS` to Lock-only; receive relocated `logWarningToStderrf`)
- Modify: `internal/cli/targets.go` (learn-group closures only)
- Modify: `internal/cli/targets_test.go` (extend `newTestDeps` with `FS`/`Lock` — R11; this task's closure flip makes the executed learn tests dereference both)
- Modify: `internal/cli/export_test.go` (re-sign fact/feedback exports; add `ExportNewLearnDeps`/`ExportNewQaDeps`; drop `ExportNewOsLearnFS` uses of deleted methods)
- Modify: `internal/cli/testhelpers_test.go` (add `realFSDepsForTest` — composed over T1-rework's cli_test helpers; the draft's `osEdgeFSForTest`/`flockLockerForTest` os doubles are NOT declared, per the composition doctrine — see step 1 and the NOTE(R11-naming) in step 7)
- Modify: `internal/cli/invariants_k1_property_test.go` (K1 drives production `newLearnDeps` over the composed primFS/primLocker with real primitives)
- Modify: `internal/cli/learn_adapters_test.go` (delete `TestOsLearnFS_*` except Lock test; thread Deps into `ExportRunLearnFrom*Args` calls)

**Interfaces**
- Consumes (from foundation `internal/cli/deps.go`): `Deps{Stdout, Stderr io.Writer; Now func() time.Time; Getenv func(string) string; FS EdgeFS; Lock FileLocker; Embed embed.Embedder; ...}`, `EdgeFS`, `FileLocker{ Lock(path string) (unlock func() error, err error) }`
- Consumes (from T1-rework): `cli.Primitives` + `cli.NewDeps` (this task adds the `OpenExcl` field per flag X-1), `primFS`/`primLocker` (unexported, reached only through `NewDeps`), and the cli_test helpers `realPrimitives()`, `realDepsForTest()`, `realFSForTest()`, `fsFromPrims`, `lockerFromPrims`, const `atomicPerm`/`realFSFilePerm`
- Produces:
  - `func newLearnDeps(d Deps) LearnDeps`
  - `func newQaDeps(d Deps) LearnQADeps`
  - `func runLearnFromFactArgs(ctx context.Context, a LearnFactArgs, d Deps, stdout io.Writer) error`
  - `func runLearnFromFeedbackArgs(ctx context.Context, a LearnFeedbackArgs, d Deps, stdout io.Writer) error`
  - Shared helpers: `statDirFromFS(fsys EdgeFS) func(string) error`, `initVaultFromFS(fsys EdgeFS) func(string) error`, `listIDsFromFS`, `listBasenamesFromFS`, `listMDFromFS(fsys EdgeFS) func(string) ([]string, error)`, `vaultLockFromLocker(locker FileLocker) func(string) (func(), error)`, `writeNewFromFS`, `writeSidecarFromFS`, `writeNoteAtomicFromFS(fsys EdgeFS, perm fs.FileMode) func(string, []byte) error`, `logWarningTo(w io.Writer) func(string, ...any)`
  - EdgeFS addition (flagged): `WriteFileExcl(path string, data []byte, perm fs.FileMode) error`
  - Primitives addition (flag X-1 resolution): `OpenExcl func(path string, perm fs.FileMode) (io.WriteCloser, error)` + `primFS.WriteFileExcl` (internal lifecycle)

**Steps**

- [ ] 1. **RED — composed test Deps + contract tests.** NO os-backed doubles are declared in this task — the composition doctrine forbids hand-rolled adapter mirrors. The FS/Lock capabilities for every test below are the PRODUCTION `primFS`/`primLocker` compositions, reached through T1-rework's cli_test helpers (`realDepsForTest()` = `cli.NewDeps(realPrimitives(), io.Discard, io.Discard, func(int) {})`). Add to `internal/cli/testhelpers_test.go` (package `cli_test`; test files are exempt from the depguard/forbidigo enforcement):

```go
// realFSDepsForTest is the learn-family test Deps: production Deps composed
// by cli.NewDeps over real OS primitives (T1-rework's realDepsForTest), with
// Embed forced nil so auto-embed skips — unit tests must not load the
// bundled model (the embed-on-write path stays covered by cli_test.go's
// real-binary end-to-end test). No signal registration occurs:
// realPrimitives() omits StartSignalPulses, so startForceExit nil-skips
// (doctrine flag SIG-1).
func realFSDepsForTest() cli.Deps {
	deps := realDepsForTest()
	deps.Embed = nil

	return deps
}
```

Add to `internal/cli/edgefs_test.go` (package `cli_test` — T1-rework's fake-primitive EdgeFS suite; reuses its `fsFromPrims` helper and `atomicPerm` const, and its existing `errors`/`io`/`io/fs` imports) the `WriteFileExcl` contract tests. These fail to compile until step 2 lands the `OpenExcl` field and the `primFS.WriteFileExcl` method (RED):

```go
func TestEdgeFS_WriteFileExclPreservesErrExistAndAddsPath(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fsys := fsFromPrims(cli.Primitives{
		OpenExcl: func(path string, _ fs.FileMode) (io.WriteCloser, error) {
			return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrExist}
		},
	})

	err := fsys.WriteFileExcl("existing.md", []byte("x"), atomicPerm)
	g.Expect(err).To(gomega.MatchError(fs.ErrExist),
		"K1 backstop: errors.Is(err, fs.ErrExist) must survive the internal wrap")
	g.Expect(err.Error()).To(gomega.ContainSubstring("existing.md"), "wrap must add path context")
}

func TestEdgeFS_WriteFileExclLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("happy path passes perm, writes data, closes once", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		file := &exclFileFake{}
		fsys := fsFromPrims(cli.Primitives{
			OpenExcl: func(_ string, perm fs.FileMode) (io.WriteCloser, error) {
				g.Expect(perm).To(gomega.Equal(atomicPerm), "caller perm must reach the primitive")

				return file, nil
			},
		})

		g.Expect(fsys.WriteFileExcl("new.md", []byte("body"), atomicPerm)).To(gomega.Succeed())
		g.Expect(string(file.wrote)).To(gomega.Equal("body"))
		g.Expect(file.closes).To(gomega.Equal(1))
	})

	t.Run("write failure reported and file still closed", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		boom := errors.New("write boom")
		file := &exclFileFake{writeErr: boom}
		fsys := fsFromPrims(cli.Primitives{
			OpenExcl: func(string, fs.FileMode) (io.WriteCloser, error) { return file, nil },
		})

		err := fsys.WriteFileExcl("new.md", []byte("body"), atomicPerm)
		g.Expect(err).To(gomega.MatchError(boom))
		g.Expect(file.closes).To(gomega.Equal(1), "write failure must not leak the handle")
	})

	t.Run("close failure is reported", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		boom := errors.New("close boom")
		file := &exclFileFake{closeErr: boom}
		fsys := fsFromPrims(cli.Primitives{
			OpenExcl: func(string, fs.FileMode) (io.WriteCloser, error) { return file, nil },
		})

		err := fsys.WriteFileExcl("new.md", []byte("body"), atomicPerm)
		g.Expect(err).To(gomega.MatchError(boom))
		g.Expect(err.Error()).To(gomega.ContainSubstring("close"))
	})
}

// exclFileFake is a recording io.WriteCloser for WriteFileExcl lifecycle tests.
type exclFileFake struct {
	writeErr error
	closeErr error
	closes   int
	wrote    []byte
}

func (f *exclFileFake) Write(data []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}

	f.wrote = append(f.wrote, data...)

	return len(data), nil
}

func (f *exclFileFake) Close() error {
	f.closes++

	return f.closeErr
}
```

Add to `internal/cli/primitives_integration_test.go` (package `cli_test`; existing `io/fs`/`path/filepath` imports suffice) the real-primitive round-trip — RED until step 2 extends `realPrimitives()` (compile fails on the unknown `OpenExcl` field first; after 2b, a missed 2d panics on the nil func — either way loud):

```go
func TestRealEdgeFS_WriteFileExclRefusesExistingFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	path := filepath.Join(t.TempDir(), "note.md")
	fsys := realFSForTest()

	g.Expect(fsys.WriteFileExcl(path, []byte("first"), realFSFilePerm)).To(gomega.Succeed())

	err := fsys.WriteFileExcl(path, []byte("second"), realFSFilePerm)
	g.Expect(err).To(gomega.MatchError(fs.ErrExist), "O_EXCL contract: existing path must satisfy fs.ErrExist")

	data, readErr := fsys.ReadFile(path)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(gomega.Equal("first"), "the losing writer must not clobber the existing note")
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

Run: `targ test` → expected FAIL (compile errors: unknown field `OpenExcl` in `cli.Primitives`; `WriteFileExcl` not in `cli.EdgeFS`; `ExportNewLearnDeps`/`ExportNewQaDeps` undefined). This is the RED.

- [ ] 2. **GREEN (flag X-1) — `EdgeFS.WriteFileExcl`, composed internally over a new exclusive-create primitive.** X-1 resolution (recorded): the primitive is `OpenExcl func(path string, perm fs.FileMode) (io.WriteCloser, error)` — cmd-side value is a single `os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)` call (one closure, one external call; `*os.File` satisfies `io.WriteCloser`, and its raw error already satisfies `errors.Is(err, fs.ErrExist)` via `*fs.PathError`). The flag's `uintptr`-fd sketch would need a second `WriteFD` primitive — rejected as the less clean shape. The write/close lifecycle (close-on-write-failure, close-error reporting, single `%w` wrap) lives in `primFS`. NO cmd/engram adapter file exists or is created — the supersession map re-points every "cmd/engram os_fs.go `osFS`" obligation to internal/cli/edgefs.go. Five sub-edits, one commit:

   **2a.** In the foundation's `internal/cli/deps.go`, add to `EdgeFS`:

```go
	// WriteFileExcl creates path exclusively (O_CREATE|O_EXCL semantics): it
	// errors with an error satisfying errors.Is(err, fs.ErrExist) when path
	// already exists. The learn family's ID-collision backstop (ADR-0013 K1)
	// and idempotent vault bootstrap both require exclusive create.
	WriteFileExcl(path string, data []byte, perm fs.FileMode) error
```

   **2b.** In `internal/cli/primitives.go`, add to the `Primitives` struct's filesystem block (`io` is already imported for `NewDeps`):

```go
	// Exclusive create (single-call closure; write/close lifecycle internal).
	OpenExcl func(path string, perm fs.FileMode) (io.WriteCloser, error) // os.OpenFile O_CREATE|O_EXCL|O_WRONLY
```

   **2c.** In `internal/cli/edgefs.go`, add to `primFS`:

```go
// WriteFileExcl creates path exclusively via the OpenExcl primitive
// (O_CREATE|O_EXCL — the ADR-0013 K1 collision backstop). The raw primitive
// error is wrapped exactly once here, preserving the fs.ErrExist chain; the
// write/close lifecycle is internal (doctrine flag X-1).
func (p primFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	file, err := p.prims.OpenExcl(path, perm)
	if err != nil {
		return fmt.Errorf("write excl %s: %w", path, err)
	}

	if _, writeErr := file.Write(data); writeErr != nil {
		_ = file.Close()

		return fmt.Errorf("write excl %s: %w", path, writeErr)
	}

	if closeErr := file.Close(); closeErr != nil {
		return fmt.Errorf("write excl %s: close: %w", path, closeErr)
	}

	return nil
}
```

   **2d.** In `internal/cli/primitives_integration_test.go`, extend `realPrimitives()` with the mirror closure (keeps the DRIFT-flag mirror exact):

```go
		OpenExcl: func(path string, perm fs.FileMode) (io.WriteCloser, error) {
			return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm) //nolint:gosec // test helper, path from test
		},
```

   **2e.** In `cmd/engram/main.go`, add ONE field line to the `cli.Primitives` literal (beside `CreateTemp`) and `"io"` to the import block. This is a single-call closure inside an expression — `check-thin-api` walks declarations only, and the doctrine caps the closure body at this single call:

```go
			OpenExcl: func(path string, perm fs.FileMode) (io.WriteCloser, error) {
				// Vault paths are operator-supplied CLI args, not untrusted input.
				//nolint:gosec // operator-controlled path
				return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
			},
```

Run `targ test` → the step-1 `WriteFileExcl` contract + integration tests go green; the deps_compose tests stay RED until steps 3-8 land.

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

In the same commit, extend `newTestDeps` in `internal/cli/targets_test.go` (R11 — this task owns the FS/Lock extension because its closure flip is the FIRST to make an executed targets-level test dereference them: the learn feedback/fact/qa tests at targets_test.go:206-263 run through `executeForTest`, and the qa test asserts real note files on disk; nil `FS`/`Lock` is a nil-interface panic inside `newLearnDeps`/`newQaDeps`). Exact diff — the two added field VALUES are the production compositions via T1-rework's cli_test helpers (same package cli_test):

```go
func newTestDeps(stdout, stderr io.Writer) cli.Deps {
	return cli.Deps{
		Stdout:      stdout,
		Stderr:      stderr,
		Exit:        func(int) {},
		Getenv:      os.Getenv,
		Now:         time.Now,
		Getwd:       os.Getwd,
		UserHomeDir: os.UserHomeDir,
		FS:          realFSForTest(),
		Lock:        lockerFromPrims(realPrimitives()),
	}
}
```

NOTE(R11-naming): R11's text names the draft doubles `osEdgeFSForTest{}`/`flockLockerForTest{}` as the field values; under the composition doctrine those hand-rolled os doubles are NOT declared anywhere — the values are the composed `primFS`/`primLocker` reached through T1-rework's helpers. R11's OWNERSHIP rule is unchanged: T3 is the sole extender, fields are exactly `FS`+`Lock`. The helper itself remains a `cli.Deps` literal (it never calls `cli.NewDeps` directly); the T1-rework helpers it consumes do call `NewDeps`, but retain only `.FS`/`.Lock` — no signal registration occurs (`realPrimitives()` omits `StartSignalPulses`, SIG-1) and the transiently-constructed lazy embedder is discarded unloaded, so T2's "no embedder/signal wiring" intent holds behaviorally.

No later task extends `newTestDeps` further (R11); every later executed targets-level path (count/show T5, show-chunk T6, ingest T8, prune T9, vocab/amend/resituate/activate T12) rides these same two fields. (`Embed` stays absent — see R11's recorded CONFLICT for the T15 embed closures.)

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

Keep `ExportNewOsLearnFS` and `ExportFlockPath` untouched here (`ExportNewOsLearnFS`'s Lock-only consumer survives until T4/L2; `ExportFlockPath` is deleted later by T8, which also repoints its consumers — R7).

- [ ] 9. **learn_adapters_test.go:** delete `TestOsLearnFS_ListBasenames_*`, `TestOsLearnFS_ListIDs_*`, `TestOsLearnFS_MkdirAll_*`, `TestOsLearnFS_StatDir_*`, `TestOsLearnFS_WriteFileIfMissing_*`, `TestOsLearnFS_WriteNew_*` (lines 58-122, 133-288 — behavior now covered by deps_compose_test.go against the production composition + the internal primitives integration suite, `internal/cli/primitives_integration_test.go`; the former cmd adapter suites were deleted by T1-rework). Keep `TestOsLearnFS_Lock_BadVaultReturnsError` (124-131) until L2. Update every `cli.ExportRunLearnFromFactArgs(context.Background(), args, io.Discard)` / `...FeedbackArgs(...)` call (lines 37, 310, 371, 408, 456, 489) to `cli.ExportRunLearnFromFactArgs(context.Background(), args, realFSDepsForTest(), io.Discard)` (same for feedback). Note: with `Embed` forced nil in `realFSDepsForTest` these tests skip auto-embed (they only assert `.md` presence/absence — assertions unchanged; the embed-on-write path stays covered by cli_test.go's real-binary end-to-end test, which asserts the `.vec.json` sidecar).

- [ ] 10. **K1 regression test survives (ADR-0013).** In `invariants_k1_property_test.go`, replace `k1RealLockDeps` (lines 122-140) — the test body (30-120) is unchanged:

```go
// unexported variables.
var errK1VaultMissing = errors.New("k1: vault should already exist")

// k1RealLockDeps wires LearnDeps through the PRODUCTION composition
// (newLearnDeps) over the internally-composed primFS EdgeFS and primLocker
// FileLocker with real OS primitives — the exact flock + exclusive-create
// (WriteFileExcl over the OpenExcl primitive) path the shipped binary builds
// via cli.NewDeps. Embed is nil (realFSDepsForTest forces it) so auto-embed
// skips; InitVault errors because the caller pre-creates the vault.
func k1RealLockDeps(vault string) cli.LearnDeps {
	deps := cli.ExportNewLearnDeps(realFSDepsForTest())

	deps.InitVault = func(string) error {
		return fmt.Errorf("%w: %s", errK1VaultMissing, vault)
	}

	return deps
}
```

(Add `"errors"` to the file's imports; the old hand-wired deps' `"time"`/`os.Getenv` uses disappear — drop those imports if nothing else in the file needs them.) This upgrade means K1 now races the production composition layer itself — lock file `vault/.luhmann.lock`, span ListIDs→WriteNew, O_EXCL backstop through `primFS.WriteFileExcl` — not a hand-wired double of it.

- [ ] 11. Run `targ test` → all green (RED tests from step 1 now pass; K1 passes at workers=2,5,10,20). Run `targ check-full` → clean. Run `targ check-thin-api` → PASS: this task's only cmd/engram delta is the single-call `OpenExcl` closure inside the `Primitives` literal (an expression, not a declaration). If the checker flags ANYTHING, escalate the exact finding to the orchestrator (global constraint / doctrine item 5) — do not suppress, do not restructure ad hoc. Then run `go install ./cmd/engram && cd "$(mktemp -d)" && engram learn fact --slug smoke --vault "$(mktemp -d)/v" --position top --source smoke --situation "smoke" --subject s --predicate p --object o` → prints the note path; the note and `.vec.json` sidecar exist.

- [ ] 12. Commit:

```
refactor(cli): compose learn-family deps from Deps (#700)

newLearnDeps/newQaDeps replace newOsLearnDeps/newOsLearnQADeps; all learn/qa
I/O flows through EdgeFS/FileLocker/Embed/Stderr. Adds EdgeFS.WriteFileExcl,
composed internally in primFS over the new single-call OpenExcl primitive
(doctrine flag X-1), so the ADR-0013 K1 O_EXCL backstop survives composition;
K1 concurrency property now drives the production composition over real flock.

AI-Used: [claude]
```

---

