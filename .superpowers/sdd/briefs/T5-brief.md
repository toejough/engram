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
   // over t.TempDir() fixtures. The production EdgeFS impl is internal/cli's
   // unexported primFS (composed by cli.NewDeps — T1-rework); test files are
   // exempt from the internal/ purity rules, so this double calls os directly.
   // Errors are wrapped with %w, matching the production contract that
   // errors.Is(err, fs.ErrNotExist) / errors.Is(err, fs.ErrExist) must
   // survive the adapter.
   // WriteFileAtomic is a plain write — ADR-0013 atomicity is exercised by
   // the internal edgefs/primitives integration suites (T1-rework), never
   // through this double.
   // WriteFileExcl is a real exclusive create (O_CREATE|O_EXCL) matching
   // T3's EdgeFS addition.
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

   func (osTestEdgeFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
   	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm) //nolint:gosec // test fixture path
   	if err != nil {
   		return fmt.Errorf("test edgefs: opening excl %s: %w", path, err)
   	}

   	defer func() { _ = file.Close() }()

   	_, err = file.Write(data)
   	if err != nil {
   		return fmt.Errorf("test edgefs: writing excl %s: %w", path, err)
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
   // composition, all I/O flows through the injected EdgeFS (#700).
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

   The executed targets-level tests riding these flips (`TestTargets_CountEmptyVault` count_test.go:562, the show test targets_test.go:160-177) dereference `d.FS` through `newVaultFS(d.FS)` — `newTestDeps` already carries `FS`/`Lock` since T3 (R11); no `newTestDeps` edit here.

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

7. [ ] Run `targ test` — expected: all green (new vaultFS tests pass; count/show/check suites pass unchanged). Run `targ check-full` — expected: no new findings. Run `targ check-thin-api` — expected: PASS (`All N public API files are thin wrappers.`); this task adds no cmd/engram declarations, so any finding predates it — escalate per Global Constraints, never suppress.
8. [ ] Commit: `refactor(cli): vault reads via EdgeFS-backed vaultFS (#700)`

---

