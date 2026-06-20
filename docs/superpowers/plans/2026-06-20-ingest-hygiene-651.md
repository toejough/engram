# Ingest Hygiene (issue 651) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep throwaway eval/test workspaces out of the main chunk index — prevent non-persistent workspaces from entering on `ingest --auto`, add a GC to prune chunks whose source file is gone, and preserve a deliberate opt-in path for testing.

**Architecture:** Three complementary changes, all in `internal/cli`. (1) **Prevention** — `SweepSpec` gains a `NonPersistentPrefixes` list; `ResolveSweepRoots` attaches it to the session-logs root only; the sweep walk prunes any project-dir whose *name* starts with one of those slugified-cwd prefixes (e.g. `-private-tmp-`). (2) **Cleanup** — a new `engram prune` subcommand walks the manifest and deletes the index file + manifest entry for every source whose file no longer exists. (3) **Opt-in** — prevention is scoped to `--auto`'s session-logs root, so explicit `--sweep`/`--transcript`/`--markdown` and isolated `ENGRAM_CHUNKS_DIR` already bypass it; covered by a test and docs.

**Tech Stack:** Go, `targ` build system, imptest + gomega test stack (DI-everywhere, no `os.*` in `internal/`).

## Global Constraints

- DI everywhere: no `os.*`/`http.*` calls in `internal/`; all I/O through injected deps wired in a `newOs*Deps` constructor.
- Tests: `targ test`. Lint + coverage: `targ check-full`. Build: `targ build`. NEVER run `go test`/`go vet`/`go build` directly.
- Every test + subtest gets `t.Parallel()` with no shared mutable state.
- Name constants, not magic strings/numbers; wrap errors with `fmt.Errorf("...: %w", err)`; line length under 120.
- Blackbox tests: `package cli_test`. Reuse existing fakes (`memFS`, `sweepFS`, `specFS`, `countingEmbedder`) and `export_test.go` re-exports.
- The dead sources are session transcripts under `~/.claude/projects/<slug>/`, where `<slug>` is the original cwd with `/`→`-` (e.g. cwd `/private/tmp/cummatrix-x` → dir `-private-tmp-cummatrix-x`). Prevention keys off the **project-dir name prefix**, NOT the source path (which is never literally under `/private/tmp`).

---

### Task 1: Prevention — prune non-persistent project dirs from `--auto` session sweep

**Files:**
- Modify: `internal/cli/sweepspec.go` (add `SweepSpec.NonPersistentPrefixes`, `SweepRoot.ExcludePrefixes`, default values, wire in `ResolveSweepRoots`)
- Modify: `internal/cli/ingest.go` (`walkSourcesExcluding` — prune dirs by prefix; extract a pure decision helper)
- Test: `internal/cli/sweepspec_test.go`, `internal/cli/ingest_sweep_test.go`
- Modify: `internal/cli/export_test.go` (re-export the pure helper for unit test)

**Interfaces:**
- Produces: `SweepSpec.NonPersistentPrefixes []string` (json `non_persistent_prefixes`); `SweepRoot.ExcludePrefixes []string`; pure `shouldPruneDir(name string, excludeNames map[string]struct{}, excludePrefixes []string) bool`.
- Consumes: existing `ResolveSweepRoots(spec, env)`, `walkSourcesExcluding(root)`.

- [ ] **Step 1: Write the failing test for the default + the pure decision helper**

In `sweepspec_test.go`:

```go
func TestDefaultSweepSpecSkipsNonPersistentWorkspaces(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	spec := cli.DefaultSweepSpec()

	g.Expect(spec.NonPersistentPrefixes).To(gomega.ContainElements("-private-tmp-", "-tmp-", "-var-folders-"))
}

func TestResolveSweepRootsAttachesPrefixesToSessionRootOnly(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fs := specFS{dirs: map[string]bool{
		"/home/dev/proj/.git":          true,
		"/sessions/-home-dev-proj":     true,
	}}

	roots := cli.ResolveSweepRoots(cli.DefaultSweepSpec(), cli.SweepEnv{
		Cwd: "/home/dev/proj", SessionDir: "/sessions/-home-dev-proj", IsDir: fs.isDir,
	})

	prefixesByPath := map[string][]string{}
	for _, root := range roots {
		prefixesByPath[root.Path] = root.ExcludePrefixes
	}

	g.Expect(prefixesByPath["/sessions/-home-dev-proj"]).To(gomega.ContainElement("-private-tmp-"),
		"session-logs root prunes non-persistent project dirs")
	g.Expect(prefixesByPath["/home/dev/proj"]).To(gomega.BeEmpty(),
		"repo-markdown root carries no non-persistent prefixes")
}
```

In `ingest_sweep_test.go` (pure helper, exercised via re-export):

```go
func TestShouldPruneDir(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	excludeNames := map[string]struct{}{"node_modules": {}}
	prefixes := []string{"-private-tmp-", "-tmp-"}

	g.Expect(cli.ExportShouldPruneDir("node_modules", excludeNames, prefixes)).To(gomega.BeTrue(), "name match")
	g.Expect(cli.ExportShouldPruneDir("-private-tmp-cummatrix-x", excludeNames, prefixes)).To(gomega.BeTrue(), "prefix match")
	g.Expect(cli.ExportShouldPruneDir("-Users-joe-repos-engram", excludeNames, prefixes)).To(gomega.BeFalse(), "persistent dir kept")
}
```

Add to `export_test.go`:

```go
ExportShouldPruneDir = shouldPruneDir
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `targ test`
Expected: FAIL — `spec.NonPersistentPrefixes` undefined, `cli.ExportShouldPruneDir` undefined, `root.ExcludePrefixes` undefined.

- [ ] **Step 3: Add the spec field, the SweepRoot field, and the default**

In `sweepspec.go`, add to `SweepRoot`:

```go
	// ExcludePrefixes prunes any directory whose NAME starts with one of these
	// slugified-cwd prefixes — used to keep non-persistent workspaces (e.g.
	// session logs under `-private-tmp-…`) out of the main index. Empty for
	// manual --sweep roots, so deliberate test ingestion still works.
	ExcludePrefixes []string
```

Add to `SweepSpec` (after `ClaudeExcludeDirs`):

```go
	// NonPersistentPrefixes name project-dir prefixes that `--auto` skips:
	// session logs whose slugified cwd lives under a throwaway root
	// (`/private/tmp`, `/tmp`, macOS `$TMPDIR` at `/var/folders`). Eval/test
	// runs never bloat the main index; explicit --sweep/--transcript bypass it.
	NonPersistentPrefixes []string `json:"non_persistent_prefixes"` //nolint:tagliatelle // developer-facing config uses snake_case
```

In `DefaultSweepSpec()`, add the field to the returned literal:

```go
		NonPersistentPrefixes: []string{"-private-tmp-", "-tmp-", "-var-folders-"},
```

In `ResolveSweepRoots`, set the prefixes on the session-logs root only:

```go
	if spec.SessionLogs && env.SessionDir != "" && env.IsDir(env.SessionDir) {
		roots = append(roots, SweepRoot{
			Path:            env.SessionDir,
			ExcludeDirs:     spec.ExcludeDirs,
			ExcludePrefixes: spec.NonPersistentPrefixes,
			SkipHidden:      skipHidden,
		})
	}
```

- [ ] **Step 4: Add the pure helper and use it in the walk**

In `ingest.go`, add:

```go
// shouldPruneDir reports whether a swept subdirectory should be skipped: its
// name is an excluded build/dependency name, or it starts with a
// non-persistent-workspace prefix (a slugified throwaway cwd).
func shouldPruneDir(name string, excludeNames map[string]struct{}, excludePrefixes []string) bool {
	if _, named := excludeNames[name]; named {
		return true
	}

	for _, prefix := range excludePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}
```

In `walkSourcesExcluding`, replace the dir-pruning block:

```go
		if entry.IsDir() {
			if path == root.Path {
				return nil
			}

			hidden := root.SkipHidden && strings.HasPrefix(entry.Name(), ".")
			if hidden || shouldPruneDir(entry.Name(), excluded, root.ExcludePrefixes) {
				return filepath.SkipDir
			}

			return nil
		}
```

(`excluded` is the existing `map[string]struct{}` built at the top of the function.)

- [ ] **Step 5: Run the tests to verify they pass**

Run: `targ test`
Expected: PASS (new tests green; existing sweep tests unaffected — manual `--sweep` roots have empty `ExcludePrefixes`).

- [ ] **Step 6: Run full checks**

Run: `targ check-full`
Expected: no lint/coverage failures.

- [ ] **Step 7: Commit**

```bash
git add internal/cli/sweepspec.go internal/cli/ingest.go internal/cli/sweepspec_test.go internal/cli/ingest_sweep_test.go internal/cli/export_test.go
git commit -m "feat(ingest): skip non-persistent workspaces in --auto sweep"
```

---

### Task 2: Cleanup — `engram prune` GC for dead sources

**Files:**
- Create: `internal/cli/prune.go` (`PruneArgs`, `PruneDeps`, `RunPrune`, `newOsPruneDeps`)
- Test: `internal/cli/prune_test.go`
- Modify: `internal/cli/export_test.go` (re-export `newOsPruneDeps` if an integration test needs it — optional)

**Interfaces:**
- Produces: `RunPrune(ctx context.Context, args PruneArgs, deps PruneDeps, stdout io.Writer) error`; `PruneArgs{ChunksDir string}`; `PruneDeps{ReadFile, WriteFile func(...), Exists func(path string) bool, Remove func(path string) error}`.
- Consumes: existing `readManifest`/`writeManifestFile` helpers, `sourceSlug`, `manifestName`, `jsonlExt` from `ingest.go`; manifest type `ingestManifest`.

- [ ] **Step 1: Write the failing test**

In `prune_test.go`:

```go
package cli_test

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestPruneRemovesDeadSources(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	live := "/sessions/live.jsonl"
	dead := "/sessions/-private-tmp-eval/dead.jsonl"
	manifest := map[string]map[string]any{
		live: {"mtime_unix_nano": 1, "size": 10, "file_hash": "sha256:a"},
		dead: {"mtime_unix_nano": 2, "size": 20, "file_hash": "sha256:b"},
	}
	manBytes, _ := json.Marshal(manifest)

	fs := newPruneFS()
	fs.files["/chunks/manifest.json"] = manBytes
	fs.files["/chunks/"+cli.ExportIndexFileName(live)] = []byte("[]")
	fs.files["/chunks/"+cli.ExportIndexFileName(dead)] = []byte("[]")
	fs.exists[live] = true // dead source file is absent

	err := cli.RunPrune(context.Background(),
		cli.PruneArgs{ChunksDir: "/chunks"}, fs.pruneDeps(), io.Discard)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}

	_, deadIndexPresent := fs.files["/chunks/"+cli.ExportIndexFileName(dead)]
	g.Expect(deadIndexPresent).To(gomega.BeFalse(), "dead source index removed")

	_, liveIndexPresent := fs.files["/chunks/"+cli.ExportIndexFileName(live)]
	g.Expect(liveIndexPresent).To(gomega.BeTrue(), "live source index kept")

	var rewritten map[string]any
	g.Expect(json.Unmarshal(fs.files["/chunks/manifest.json"], &rewritten)).To(gomega.Succeed())
	g.Expect(rewritten).To(gomega.HaveKey(live))
	g.Expect(rewritten).NotTo(gomega.HaveKey(dead), "dead source dropped from manifest")
}
```

Add the small fake at the bottom of `prune_test.go`:

```go
type pruneFS struct {
	files  map[string][]byte
	exists map[string]bool
}

func newPruneFS() *pruneFS {
	return &pruneFS{files: map[string][]byte{}, exists: map[string]bool{}}
}

func (p *pruneFS) pruneDeps() cli.PruneDeps {
	return cli.PruneDeps{
		ReadFile:  func(path string) ([]byte, error) { return p.read(path) },
		WriteFile: func(path string, data []byte) error { p.files[path] = data; return nil },
		Exists:    func(path string) bool { return p.exists[path] },
		Remove:    func(path string) error { delete(p.files, path); return nil },
	}
}

func (p *pruneFS) read(path string) ([]byte, error) {
	data, ok := p.files[path]
	if !ok {
		return nil, io.ErrUnexpectedEOF
	}

	return data, nil
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `targ test`
Expected: FAIL — `cli.RunPrune`, `cli.PruneArgs`, `cli.PruneDeps` undefined.

- [ ] **Step 3: Implement `prune.go`**

```go
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// PruneArgs holds parsed flags for `engram prune`.
type PruneArgs struct {
	ChunksDir string `targ:"flag,name=chunks-dir,desc=chunk index dir (default $XDG_DATA_HOME/engram/chunks)"`
}

// PruneDeps holds injected dependencies for RunPrune.
type PruneDeps struct {
	ReadFile  func(path string) ([]byte, error)
	WriteFile func(path string, data []byte) error
	Exists    func(path string) bool
	Remove    func(path string) error
}

// RunPrune garbage-collects the chunk index: every manifest source whose file
// no longer exists has its per-source index file deleted and its manifest entry
// dropped. Append-only history is preserved for live sources. Zero-LLM.
func RunPrune(_ context.Context, args PruneArgs, deps PruneDeps, stdout io.Writer) error {
	manifest := ingestManifest{}

	data, err := deps.ReadFile(filepath.Join(args.ChunksDir, manifestName))
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "prune: no manifest, nothing to prune")

		return nil //nolint:nilerr // absent manifest = empty index, not an error
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("prune: reading manifest: %w", err)
	}

	pruned := 0

	for source := range manifest {
		if deps.Exists(source) {
			continue
		}

		indexPath := filepath.Join(args.ChunksDir, sourceSlug(source)+jsonlExt)
		if err := deps.Remove(indexPath); err != nil {
			return fmt.Errorf("prune: removing index %s: %w", indexPath, err)
		}

		delete(manifest, source)
		pruned++
	}

	if pruned == 0 {
		_, _ = fmt.Fprintln(stdout, "prune: no dead sources")

		return nil
	}

	out, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("prune: encoding manifest: %w", err)
	}

	if err := deps.WriteFile(filepath.Join(args.ChunksDir, manifestName), out); err != nil {
		return fmt.Errorf("prune: writing manifest: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "prune: removed %d dead source(s)\n", pruned)

	return nil
}

// newOsPruneDeps wires the production filesystem for `engram prune`.
func newOsPruneDeps() PruneDeps {
	fs := &osEmbedFS{}

	return PruneDeps{
		ReadFile:  fs.Read,
		WriteFile: fs.Write,
		Exists: func(path string) bool {
			_, err := os.Stat(path)

			return err == nil
		},
		Remove: os.Remove,
	}
}
```

Add `"os"` to the import block (and confirm `osEmbedFS.Write` exists — it is used by `newOsIngestDeps`).

- [ ] **Step 4: Run the test to verify it passes**

Run: `targ test`
Expected: PASS.

- [ ] **Step 5: Wire the subcommand in `targets.go`**

In `ingestQueryTargets` (`internal/cli/targets.go`), add after the `ingest` target:

```go
		targ.Targ(func(ctx context.Context, a PruneArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, homeOrEmpty(), os.Getenv)
			errHandler(RunPrune(withLog(ctx), a, newOsPruneDeps(), stdout))
		}).Name("prune").Description("Remove chunk index entries whose source file no longer exists (GC)"),
```

- [ ] **Step 6: Run full checks + build**

Run: `targ check-full && targ build`
Expected: green; `engram prune` appears in `--help`.

- [ ] **Step 7: Verify with the real binary**

Run: `engram prune` (against the real chunks dir — the manifest still references the dead `-private-tmp-cummatrix-*` / `-private-tmp-gate-*` sources).
Expected: prints `prune: removed N dead source(s)` with N > 0; a follow-up `engram ingest --auto` no longer prints the dead-source `skip …` lines.

- [ ] **Step 8: Commit**

```bash
git add internal/cli/prune.go internal/cli/prune_test.go internal/cli/targets.go
git commit -m "feat(prune): add engram prune to GC dead chunk sources"
```

---

### Task 3: Opt-in regression test (deliberate test ingestion still works)

**Files:**
- Test: `internal/cli/ingest_sweep_test.go`

**Interfaces:**
- Consumes: `cli.RunIngest`, `sweepDeps`, `sweepFS` (Task 1 unchanged).

- [ ] **Step 1: Write the test proving manual `--sweep` of a non-persistent path still ingests**

```go
func TestExplicitSweepIngestsNonPersistentWorkspace(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	src := "/private/tmp/eval-ws/s.jsonl"
	fs := newSweepFS()
	fs.put(src, "USER: run the eval harness here\nASSISTANT: ingested into the isolated index", 100)

	emb := &countingEmbedder{}
	deps := sweepDeps(fs, emb, src) // ListSources returns src verbatim for a manual --sweep root

	err := cli.RunIngest(context.Background(),
		cli.IngestArgs{Sweep: []string{"/private/tmp/eval-ws"}, ChunksDir: "/chunks"}, deps, io.Discard)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}

	_, present := fs.files["/chunks/"+cli.ExportIndexFileName(src)]
	g.Expect(present).To(gomega.BeTrue(), "explicit --sweep bypasses non-persistent prevention")
}
```

- [ ] **Step 2: Run to verify it passes immediately**

Run: `targ test`
Expected: PASS — prevention lives only on the `--auto` session-logs root; manual `--sweep` roots carry no `ExcludePrefixes`, so this is a green regression guard (no code change needed). If it fails, prevention leaked into manual roots — fix Task 1's `assembleSweepRoots` to keep manual roots prefix-free.

- [ ] **Step 3: Commit**

```bash
git add internal/cli/ingest_sweep_test.go
git commit -m "test(ingest): guard deliberate --sweep of non-persistent workspace"
```

---

## Self-Review

**Spec coverage:**
- Req 1 (Prevention) → Task 1 (`NonPersistentPrefixes` skip on `--auto` session sweep). ✓
- Req 2 (Cleanup/GC) → Task 2 (`engram prune` removes index + manifest entry for missing sources). ✓
- Req 3 (Still usable for testing) → two opt-in levers, both delivered:
  - **Explicit-override lever** → Task 1 scoping + Task 3 regression test: manual `--sweep`/`--transcript`/`--markdown` carry empty `ExcludePrefixes`, so deliberate ingestion of a throwaway workspace works. ✓
  - **Isolated-index lever** → `ENGRAM_CHUNKS_DIR`/`ENGRAM_VAULT_PATH` is a **pre-existing, already-tested** mechanism (`ResolveChunksDir` in `ingest.go:59`, covered by `TestResolveChunksDirPrecedence`) — no new code needed; it is surfaced in the Document step (see below). ✓

**Documentation deliverables (workflow Document step — Gate A docs review flagged these):**
- `docs/architecture/c3-components.md` — add `engram prune` to the L3 process subgraph + component catalog; note the live/dead-source GC decision in the manifest-staleness section.
- `docs/architecture/c1-system-context.md` — note `engram prune` as operator-run cleanup (outside the recall/learn/please flows).
- `docs/architecture/c2-containers.md` (and/or `CLAUDE.md`) — surface that the `--auto` sweep skips non-persistent workspaces by default via `SweepSpec.NonPersistentPrefixes`, overridable in `.engram/sweep.json`, and document the two opt-in levers above for deliberate eval/test ingestion.

**Open design questions (from the issue), resolved:**
- "What counts as non-persistent?" → slugified-cwd prefix match (`-private-tmp-`, `-tmp-`, `-var-folders-`), overridable via `.engram/sweep.json` `non_persistent_prefixes`.
- "Prevention via skip-by-default + opt-in vs allowlist?" → skip-by-default prefix list + opt-in via explicit flags/isolation (not an allowlist — simpler, matches existing exclude model).
- "Cleanup: standalone vs `ingest --gc`?" → standalone `engram prune` (discoverable, single responsibility), prune by missing-source-file (the issue's stated scope).
- "Testing isolation sufficient?" → both levers kept: explicit `--sweep`/`--transcript` override AND `ENGRAM_CHUNKS_DIR` isolation.

**Placeholder scan:** none — every code/test step shows complete content.

**Type consistency:** `shouldPruneDir` / `ExportShouldPruneDir`, `ExcludePrefixes`, `NonPersistentPrefixes`, `RunPrune`/`PruneArgs`/`PruneDeps`, `ExportIndexFileName` all used consistently across tasks and match existing exports (`ExportIndexFileName` already exists; `sourceSlug`, `manifestName`, `jsonlExt`, `readManifest` are package-internal and reused by `prune.go` in-package).
