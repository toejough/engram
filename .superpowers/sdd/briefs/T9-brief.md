### Task T9 (I2): Migrate `engram prune` wiring to cli.Deps composition and retire osManifestLock

**Files:**
- Modify: `internal/cli/prune.go` (replace `newOsPruneDeps` with `newPruneDeps`; flip its lister line from the transitional `osListJSONLIndexes` to T6's canonical `listJSONLIndexes(d.FS)` — per R3, T9 declares NO lister helper)
- Modify: `internal/cli/cli.go` (delete `osManifestLock` — last consumer gone)
- Modify: `internal/cli/targets.go` (prune call site)
- Modify: `internal/cli/export_test.go` (swap `ExportNewOsPruneDeps` → `ExportNewPruneDeps`; delete `ExportOsManifestLock`)
- Modify: `internal/cli/ingest_family_deps_test.go` (add prune composition tests)
- Modify: `internal/cli/prune_integration_test.go`, `internal/cli/testhelpers_test.go`

**Interfaces:**
- Consumes: `Deps`/`EdgeFS`/`FileLocker` (foundation), `manifestLockFrom` (Task I1), `chunksDirPerm`/`indexFilePerm` consts (Task I1), `listJSONLIndexes(fsys EdgeFS)` (T6's canonical curried lister — landed earlier per R4).
- Produces: `func newPruneDeps(d Deps) PruneDeps`. Per R3, T9 declares NO lister helper (`jsonlIndexListerFrom` is not created anywhere): it flips prune.go's line off the transitional `osListJSONLIndexes` onto `listJSONLIndexes(d.FS)`; the transitional lister itself is deleted later by T12 (amend, its last consumer), grep-gated.

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

- [ ] 2. GREEN — rewrite `internal/cli/prune.go`. Import block (current lines 3-10; `filepath` stays for the `filepath.Join` calls in `RunPrune`) becomes:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
)
```

Update the stale `PruneDeps.Lock` doc comment (current line 22 — the full line, prefix included, so a literal find/replace works): `// release func. Wired to flockPath(chunksDir/.manifest.lock) in newOsPruneDeps.` → `// release func. Wired to manifestLockFrom (MkdirAll + FileLocker flock) in newPruneDeps.` Replace `newOsPruneDeps` (current lines 102-118; note the `ListIndexes` line reads `osListJSONLIndexes` at this point — T6's mechanical rename landed earlier per R4):

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
		ListIndexes: osListJSONLIndexes,
		Remove:      os.Remove,
	}
}
```

with (per R3: NO lister helper is declared here — the lister is T6's canonical `listJSONLIndexes(d.FS)`; prune.go was one of `osListJSONLIndexes`'s two remaining consumers, and after this flip only amend.go remains, so T12 deletes it there, grep-gated):

```go
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
		ListIndexes: listJSONLIndexes(d.FS),
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

- [ ] 6. Adapt tests. `internal/cli/prune_integration_test.go` line 48: `cli.ExportNewOsPruneDeps()` → `cli.ExportNewPruneDeps(testDeps())`. `internal/cli/testhelpers_test.go`: delete `TestOsManifestLock_MkdirError` (current lines 13-28) and its now-unused imports if any (`os`, `filepath`, `cli`, `gomega` — check remaining uses; `sliceIndex` stays) — its coverage is replaced by the pure `TestIngestDepsLockMkdirErrorPropagates` (Task I1) plus the shared `manifestLockFrom` path now exercised by both constructors.

- [ ] 7. Verify: `targ test` — expected PASS, including `TestOsPruneDetachesDeadSource` (real FS through `testDeps()`), `TestRunPrune_LocksManifestAroundReadModifyWrite` (untouched — pure injected deps), and `TestManifest_ConcurrentWritersDoNotLoseEntries`. Purity grep, must print nothing: `grep -nE '\bos\.|time\.Now|syscall' internal/cli/prune.go`. Then `targ check-full` — expected clean. Then run the real flow once (passing tests are not a usable system): `go install ./cmd/engram && cd $(mktemp -d) && ENGRAM_CHUNKS_DIR=$PWD/chunks engram prune` — expected stdout `prune: no manifest, nothing to prune` and a created `chunks/.manifest.lock` (proves the MkdirAll-before-lock composition against the real wired binary; requires the foundation cmd wiring to be in place).

- [ ] 8. Commit:

```
git add internal/cli/prune.go internal/cli/cli.go internal/cli/targets.go \
  internal/cli/export_test.go internal/cli/ingest_family_deps_test.go \
  internal/cli/prune_integration_test.go internal/cli/testhelpers_test.go
git commit -m "refactor(cli): compose prune deps, retire osManifestLock (#700)

Stat/Remove/ReadDir via EdgeFS, manifest lock via shared manifestLockFrom
(MkdirAll-before-flock fresh-dir regression preserved). Prune's index lister
now flows through the canonical listJSONLIndexes(d.FS) (T6); the transitional
osListJSONLIndexes keeps amend.go as its sole consumer until T12 deletes it.

AI-Used: [claude]"
```

---

Verified source anchors: internal/cli/ingest.go (os at 248/252/496/504/516/520, time.Now at 514, filepath.WalkDir at 662), internal/cli/prune.go (os.Stat 111, os.Remove 116), internal/cli/cli.go (osFileReader 27, flockPath 169, osManifestLock 227, manifestLockFile 17), internal/cli/targets.go (157-166), internal/cli/query_chunks.go (listJSONLIndexes 138), internal/cli/embed.go (osEmbedFS 140, sharedEmbedder 110), internal/context/context.go (FileReader 7), internal/embed/embedder.go (Embedder 54), tests: ingest_test.go 319/376/403/899, ingest_auto_test.go 48-105, ingest_integration_test.go 188, prune_integration_test.go 48, testhelpers_test.go 13-28, adapters_test.go 14-39, export_test.go 79/342/536/543/687.

