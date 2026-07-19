### Task T15 (B): internal/cli/embed.go — compose EmbedDeps from cli.Deps, delete osEmbedFS

**Depends on:** Task T14 (A) + T1-rework/T2 landed (Deps.FS `EdgeFS`; `Deps.Embed` composed INSIDE `cli.NewDeps` per R6/D-1 — this task never touches cmd/engram or the embed wiring, only internal composition; verified unaffected by the T14 doctrine rework).

**Files**

- Modify: `internal/cli/embed.go` (delete `osEmbedFS`, `newOsEmbedDeps`; add `newEmbedDeps`)
- Modify: `internal/cli/targets.go` (lines 226, 230, 155), `internal/cli/query.go` (line 1287-1288)
- Modify: `internal/cli/export_test.go` (replace `ExportNewOsEmbedDeps`), `internal/cli/os_adapters_test.go`
- NOT touched (R2): `internal/cli/vault_fs.go` — this task declares NO VaultFS adapter; the draft's `depsVaultFS` is a loser. It consumes T5's landed `newVaultFS(d.FS)`.

**Interfaces**

- Produces: `newEmbedDeps(d Deps) EmbedDeps` (pure composition); `ExportNewEmbedDeps(d Deps) EmbedDeps`. (No `depsVaultFS` — R2; the vaultgraph.VaultFS view comes from T5's `newVaultFS`.)
- Consumes: `Deps.FS EdgeFS` (`ReadFile`, `WriteFileAtomic(path, data, perm fs.FileMode)`, `ReadDir(path) ([]fs.DirEntry, error)`), `Deps.Embed embed.Embedder`, `vaultgraph.ScanVault(fs VaultFS, vaultPath string) ([]Note, error)` with `VaultFS{ ListMD(dir string) ([]string, error); ReadFile(path string) ([]byte, error) }` (verified at `internal/vaultgraph/scanner.go:20-32`).

**Steps**

- [ ] 1. **RED — adapt the integration tests to the composed deps first.** In `internal/cli/os_adapters_test.go`: replace the three `cli.ExportNewOsEmbedDeps(<embedder>)` calls (lines 89, 125, 190) with `cli.ExportNewEmbedDeps(cli.Deps{FS: osTestEdgeFS{}, Embed: <same embedder>})`; rename `TestOsEmbedFS_ReadWriteScanRoundTrip` → `TestEmbedDeps_ReadWriteScanRoundTrip` and update its comment to say it exercises the composed Scan/Read/Write against a real tempdir vault. Do NOT append the EdgeFS double below if `osTestEdgeFS` already exists in package cli_test — per R4, T5's edgefs_os_test.go landed it before this task runs, so the expected action is to CONSUME that one and skip this block (a second declaration in the same package is a compile error; the block below survives only for the contingency that T5's was somehow renamed/removed — DESIGN FLAG 9):

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

func (osTestEdgeFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm) //nolint:gosec // thin test adapter
	if err != nil {
		return fmt.Errorf("opening excl %s: %w", path, err)
	}

	defer func() { _ = file.Close() }()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("writing excl %s: %w", path, err)
	}

	return nil
}
```

(add `"fmt"` and `"io/fs"` to the file's imports). Run `targ test` — expected RED: `ExportNewEmbedDeps` undefined.

- [ ] 2. **GREEN — compose.** In `internal/cli/embed.go` delete `osEmbedFS` and its three methods (lines 136-170) and `newOsEmbedDeps` (lines 241-252); the `"os"` import goes with them. Add:

```go
// newEmbedDeps composes the embed-command dependencies from the CLI-wide
// impure capability set. Pure composition — all I/O flows through d.FS and
// d.Embed, wired via cli.NewDeps at the edge. Sidecar writes go through WriteFileAtomic
// (temp+rename) so concurrent readers always see either the old or new
// file, never a torn write (ADR-0013 semantics preserved).
func newEmbedDeps(d Deps) EmbedDeps {
	const sidecarPerm = 0o600

	return EmbedDeps{
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(newVaultFS(d.FS), vault)
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

No VaultFS adapter is declared here (R2): `newVaultFS` already landed in T5's vault_fs.go and provides exactly the vaultgraph.VaultFS-over-EdgeFS view (missing dir → empty, wrapped-ErrNotExist unwrapped via errors.Is) — vault_fs.go is not touched by this task.

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

- [ ] 3.5. **Wire the targets-level embed tests per R11.** `newTestDeps.Embed` is nil by design (R11 — `newTestDeps` builds the `cli.Deps` literal directly and never calls `NewDeps`, so T14's internal Embed composition does not reach it; VERIFIED unaffected by the T14 doctrine rework: the stub below satisfies `embed.Embedder`, which the rework leaves unchanged, and `embed.BundledModelID` survives). The two executed embed tests dereference `ModelID()`, so give them a local fail-loud stub. In `internal/cli/targets_test.go`, add (exported-test-func-before-private-decls per reorder-decls — place the type after the last Test func or in the file's existing helper region):

```go
// stubEmbedderForTargets satisfies embed.Embedder for targets-level tests that
// only need ModelID/Dims. Embed fails loud: no targets-level test may silently
// real-embed (R11). Named to avoid cli_test's existing stubEmbedder (embed_test.go).
type stubEmbedderForTargets struct{}

func (stubEmbedderForTargets) Embed(context.Context, string) ([]float32, error) {
	return nil, errors.New("stubEmbedderForTargets: Embed not expected in targets-level tests")
}

func (stubEmbedderForTargets) ModelID() string { return embed.BundledModelID }

func (stubEmbedderForTargets) Dims() int { return 384 }
```

In `TestTargets_EmbedApplyDryRun` (targets_test.go:340) and `TestTargets_EmbedStatus` (:355), where the test builds its deps, override: `d := newTestDeps(stdout, stderr); d.Embed = stubEmbedderForTargets{}` (adapt to the tests' actual deps-construction shape — they currently ride `cli.Targets(newTestDeps(...))`; introduce the local variable form for these two tests only). Add the `context`/`errors`/`embed` imports if absent.

- [ ] 4. **Verify.** `targ test` — expected green (embed_test.go's in-memory deps untouched; adapted os_adapters tests pass through `newEmbedDeps` + `osTestEdgeFS`; `TestTargets_EmbedApplyDryRun` / `TestTargets_EmbedStatus` green through the new wiring). `targ check-full` — clean; confirm `grep -n '"os"' internal/cli/embed.go` returns nothing. `targ check-thin-api` — expected PASS (this task touches no cmd/engram file, so a failure here means an earlier task regressed the thin edge — ESCALATE the exact finding, do not fix ad hoc). Real-binary check: `go install ./cmd/engram`, then in a temp dir: create `note.md` with a body, run `engram embed apply --vault . --dry-run` (expect `would-embed note.md (missing)`), then `engram embed apply --vault .` (expect `embedded  note.md (missing)` and a `note.vec.json` sidecar with `"embedding_model_id": "minilm-l6-v2@384"`), then `engram embed status --vault .` (expect `with-embeddings: 1`).
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

**Post-cluster residue for the enforcement task** (not handled here): delete the `sharedEmbedder`/`bridgeEmbedder` transitional block in `internal/cli/embed.go` once `grep -rn "sharedEmbedder" internal/cli --include='*.go' | grep -v _test` shows only its own definition; decide `parity_test.go` exemption (DESIGN FLAG 5); `osVaultFS` deletion is T7's, gated on all consumers having migrated to `newVaultFS(d.FS)` (R2).

