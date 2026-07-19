### Task T8 (I1): Migrate `engram ingest` wiring to cli.Deps composition

**Files:**
- Modify: `internal/cli/ingest.go` (replace `newOsIngestDeps`/`defaultSessionDir`/`walkSourcesExcluding` with pure compositions)
- Modify: `internal/cli/cli.go` (delete `osFileReader` only)
- Modify: `internal/cli/targets.go` (ingest call site — depends on foundation's `deps Deps` threading)
- Modify: `internal/cli/export_test.go` (swap `ExportNewOsIngestDeps` → `ExportNewIngestDeps`; delete `ExportFlockPath`, `ExportNewOsFileReader`)
- Create: `internal/cli/ingest_family_deps_test.go` (pure composition tests + cli_test OS harness)
- Modify: `internal/cli/ingest_auto_test.go`, `internal/cli/ingest_integration_test.go`, `internal/cli/ingest_test.go`, `internal/cli/adapters_test.go`

**Interfaces:**
- Consumes (foundation): `type Deps struct { …; Getenv func(string) string; Now func() time.Time; Getwd func() (string, error); UserHomeDir func() (string, error); FS EdgeFS; Lock FileLocker; Embed embed.Embedder; … }`; `EdgeFS` with `ReadFile/WriteFile/WriteFileAtomic/WriteFileExcl/MkdirAll/MkdirTemp/Stat/ReadDir/Remove/RemoveAll/Rename/WalkDir` (`WriteFileExcl` is T3's flagged addition); `FileLocker.Lock(path string) (unlock func() error, err error)`; `transcript.NewJSONLReader(reader sessionctx.FileReader) *JSONLReader` where `sessionctx.FileReader` is `Read(path string) ([]byte, error)` (internal/context/context.go:7).
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
// Test files are exempt from the internal I/O purity enforcement; the
// production adapters are composed in internal/cli from cmd-supplied
// primitives (T1-rework). These doubles back the ingest-family
// integration tests and the ADR-0013 concurrency regression.

type fakeEdgeFS struct {
	readFile        func(string) ([]byte, error)
	writeFile       func(string, []byte, fs.FileMode) error
	writeFileAtomic func(string, []byte, fs.FileMode) error
	writeFileExcl   func(string, []byte, fs.FileMode) error
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

func (f fakeEdgeFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	return f.writeFileExcl(path, data, perm)
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

func (osTestFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	return testWriteFileExcl(path, data, perm)
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

// testWriteFileExcl is exclusive create (O_CREATE|O_EXCL), mirroring the
// production WriteFileExcl; a wrapped fs.ErrExist survives errors.Is.
func testWriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("excl write open: %w", err)
	}

	defer func() { _ = file.Close() }()

	_, writeErr := file.Write(data)
	if writeErr != nil {
		return fmt.Errorf("excl write: %w", writeErr)
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

- [ ] 8. Adapt `internal/cli/adapters_test.go`: delete `TestOsFileReader_Read` (lines 14-30) and `TestOsFileReader_ReadError` (lines 32-39). The transcript-read path is now covered by the pure `fsFileReader` composition (exercised end-to-end by `TestOsIngestThenChunkQuery` through `testDeps()`), and raw `os.ReadFile` adapter coverage lives in the internal real-primitives EdgeFS suite (`TestRealEdgeFS_ReadWriteRoundTrip` / `TestRealEdgeFS_ReadFileMissingSatisfiesErrNotExist`, primitives_integration_test.go — T1-rework). Keep the `osLearnFS` tests untouched.

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
git commit -m "refactor(cli): compose ingest deps from cli.Deps (#700)

ENGRAM_TRANSCRIPT_DIR + home via deps.Getenv/UserHomeDir, walk via
EdgeFS.WalkDir, manifest lock via FileLocker behind MkdirAll composition
(ADR-0013 semantics preserved; concurrency regression adapted to a
test-local real flock).

AI-Used: [claude]"
```

---

