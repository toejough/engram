# Consolidate retrieve.ListMemories and memory.ListAll — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate the redundant `retrieve` package by consolidating `retrieve.ListMemories` and `memory.ListAll` into a single DI-based listing function in the `memory` package.

**Architecture:** `memory.ListAll` is the lower-level function but violates DI (calls `os.ReadDir`/`os.ReadFile` directly). The `retrieve` package wraps the same logic with DI but returns a different type (`*Stored` vs `StoredRecord`). We refactor `ListAll` to accept DI, add a `ListStored` convenience function (calls `ListAll` + converts via `ToStored`), then delete the entire `retrieve` package and update all callers.

**Tech Stack:** Go, TOML, gomega (tests), targ (build system)

---

## Analysis

### Current State

Two functions do the same thing — read `.toml` files from a directory:

| | `retrieve.ListMemories` | `memory.ListAll` |
|---|---|---|
| Location | `internal/retrieve/retrieve.go:33` | `internal/memory/readmodifywrite.go:74` |
| DI | Yes (`readDir`/`readFile` injected) | No (calls `os.ReadDir`/`os.ReadFile` directly) |
| Input | `(ctx, dataDir)` — appends `/memories/` | `(dir)` — caller provides full path |
| Return | `[]*memory.Stored` | `[]memory.StoredRecord` |
| Sorting | Yes (by UpdatedAt desc) | No |
| Callers | `surface` (via interface), `correct` (via func type), `cli` (3 call sites) | `maintain`, `cli/migrate_slugs`, `cli/refine` |

### Design Decision

**Keep `ListAll` in `memory` package, refactor for DI. Add `ListStored` wrapper. Delete `retrieve` package.**

Rationale:
- `ListAll` lives in the `memory` package where all memory types are defined — no cross-package type coupling.
- The `retrieve` package has exactly one exported method (`ListMemories`) and one type (`Retriever`). It's not worth a package.
- `maintain` callers need `StoredRecord` (access to `MemoryRecord` fields for diagnosis). `surface`/`correct` callers need `*Stored`. Two return-type wrappers over one core function is cleaner than two packages.
- DI via functional options matches the existing `Modifier` pattern in the same file.

### Callers to Update

**`ListMemories` callers (will switch to `memory.ListStored`):**
1. `internal/surface/surface.go:37` — `MemoryRetriever` interface definition
2. `internal/surface/surface.go:180` — `s.retriever.ListMemories(ctx, dataDir)`
3. `internal/surface/surface_test.go:825` — `fakeRetriever.ListMemories`
4. `internal/correct/correct.go:157` — `MemoryRetrieverFunc` type
5. `internal/correct/correct.go:110` — `c.memoryRetriever(ctx, dataDir)` call
6. `internal/cli/cli.go:329` — `buildRecallSurfacer`: `retriever.ListMemories(ctx, dataDir)`
7. `internal/cli/cli.go:344` — `surface.New(retriever, ...)` — passes retriever as `MemoryRetriever`
8. `internal/cli/cli.go:547-552` — correct pipeline: `retriever.ListMemories` passed as func
9. `internal/cli/cli.go:704,730` — surface pipeline: `retriever` passed to `surface.New`
10. `internal/cli/export_test.go:83-86` — `ExportNewRetriever()`
11. `internal/cli/adapters_test.go:353` — `cli.ExportNewRetriever()` + `surface.New(retriever)`

**`ListAll` callers (stay on `ListAll`, but signature changes):**
1. `internal/maintain/maintain.go:33` — `memory.ListAll(memDir)`
2. `internal/cli/migrate_slugs.go:15` — `memory.ListAll(dir)`
3. `internal/cli/migrate_slugs_test.go:65,113,161` — `memory.ListAll(memoriesDir)`
4. `internal/cli/refine.go:143` — `memory.ListAll(memoriesDir)`

---

## File Structure

| File | Action | Purpose |
|------|--------|---------|
| `internal/memory/readmodifywrite.go` | Modify | Refactor `ListAll` for DI, add `ListStored` |
| `internal/memory/readmodifywrite_test.go` | Modify | Update `ListAll` tests for DI, add `ListStored` tests |
| `internal/surface/surface.go` | Modify | Replace `MemoryRetriever` interface with `ListStoredFunc` type |
| `internal/surface/surface_test.go` | Modify | Update fake retriever to match new type |
| `internal/correct/correct.go` | Modify | Update `MemoryRetrieverFunc` to `memory.ListStoredFunc` |
| `internal/correct/correct_test.go` | Modify | Update test fakes |
| `internal/cli/cli.go` | Modify | Remove `retrieve` import, use `memory.ListStored`/`memory.NewLister` |
| `internal/cli/export_test.go` | Modify | Remove `ExportNewRetriever` |
| `internal/cli/migrate_slugs.go` | Modify | Pass DI-wired `ListAll` |
| `internal/cli/refine.go` | Modify | Pass DI-wired `ListAll` |
| `internal/maintain/maintain.go` | Modify | Pass DI-wired `ListAll` |
| `internal/retrieve/retrieve.go` | Delete | Entire package removed |
| `internal/retrieve/retrieve_test.go` | Delete | Entire package removed |

---

### Task 1: Refactor `memory.ListAll` for DI and add `ListStored`

**Files:**
- Modify: `internal/memory/readmodifywrite.go:67-105`
- Modify: `internal/memory/readmodifywrite_test.go`

This task changes `ListAll` to accept DI functions, adds a `Lister` type (matching the `Modifier` pattern), and adds `ListStored` that wraps `ListAll` + `ToStored` conversion + sorting.

- [ ] **Step 1: Write failing test for DI-based `ListAll`**

In `internal/memory/readmodifywrite_test.go`, add a test that verifies `ListAll` works with injected I/O:

```go
func TestLister_ListAll_UsesDI(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()

	content := `situation = "test situation"
action = "test action"
surfaced_count = 3
`
	writeErr := os.WriteFile(filepath.Join(dir, "mem1.toml"), []byte(content), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())
	if writeErr != nil {
		return
	}

	lister := memory.NewLister()
	records, err := lister.ListAll(dir)

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(records).To(HaveLen(1))
	g.Expect(records[0].Record.Situation).To(Equal("test situation"))
	g.Expect(records[0].Path).To(HaveSuffix("mem1.toml"))
}
```

Note: this test uses the default Lister (real filesystem). The DI is tested by the fact that `NewLister` wires `os.ReadDir`/`os.ReadFile` as defaults, and options can override them.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestLister_ListAll_UsesDI -v ./internal/memory/`
Expected: FAIL — `NewLister` and method `ListAll` on `Lister` don't exist yet.

- [ ] **Step 3: Write failing test for `ListStored`**

```go
func TestLister_ListStored_ReturnsSortedStored(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()

	older := `situation = "older memory"
updated_at = "2024-01-01T00:00:00Z"
`
	newer := `situation = "newer memory"
updated_at = "2024-06-15T00:00:00Z"
`

	writeErr := os.WriteFile(filepath.Join(dir, "old.toml"), []byte(older), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())
	if writeErr != nil {
		return
	}

	writeErr = os.WriteFile(filepath.Join(dir, "new.toml"), []byte(newer), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())
	if writeErr != nil {
		return
	}

	lister := memory.NewLister()
	stored, err := lister.ListStored(dir)

	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(stored).To(HaveLen(2))
	g.Expect(stored[0].Situation).To(Equal("newer memory"))
	g.Expect(stored[1].Situation).To(Equal("older memory"))
}
```

- [ ] **Step 4: Implement `Lister` type, `ListAll` method, and `ListStored`**

In `internal/memory/readmodifywrite.go`, replace the current `ListAll` function with:

```go
// Lister reads memory TOML files from a directory.
type Lister struct {
	readDir  func(string) ([]os.DirEntry, error)
	readFile func(string) ([]byte, error)
}

// NewLister creates a Lister with default I/O wired to the real filesystem.
func NewLister(opts ...ListerOption) *Lister {
	l := &Lister{
		readDir:  os.ReadDir,
		readFile: os.ReadFile,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// ListerOption configures a Lister.
type ListerOption func(*Lister)

// WithListerReadDir overrides the directory reading function.
func WithListerReadDir(fn func(string) ([]os.DirEntry, error)) ListerOption {
	return func(l *Lister) { l.readDir = fn }
}

// WithListerReadFile overrides the file reading function.
func WithListerReadFile(fn func(string) ([]byte, error)) ListerOption {
	return func(l *Lister) { l.readFile = fn }
}

// ListAll reads all .toml files from a directory, returning parsed records.
func (l *Lister) ListAll(dir string) ([]StoredRecord, error) {
	entries, err := l.readDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	records := make([]StoredRecord, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		data, readErr := l.readFile(path)
		if readErr != nil {
			continue
		}

		var record MemoryRecord

		_, decErr := toml.Decode(string(data), &record)
		if decErr != nil {
			continue
		}

		records = append(records, StoredRecord{Path: path, Record: record})
	}

	return records, nil
}

// ListStored reads all .toml files from a directory, converts them to Stored,
// and returns them sorted by UpdatedAt descending.
func (l *Lister) ListStored(dir string) ([]*Stored, error) {
	records, err := l.ListAll(dir)
	if err != nil {
		return nil, err
	}

	stored := make([]*Stored, 0, len(records))
	for idx := range records {
		stored = append(stored, records[idx].Record.ToStored(records[idx].Path))
	}

	sort.Slice(stored, func(i, j int) bool {
		return stored[i].UpdatedAt.After(stored[j].UpdatedAt)
	})

	return stored, nil
}
```

Add `"sort"` to the imports. Remove the old package-level `ListAll` function.

- [ ] **Step 5: Update existing `ListAll` tests**

The four existing tests (`TestListAll_EmptyDirectory`, `TestListAll_ReadsAllTOMLFiles`, `TestListAll_SkipsInvalidTOML`, `TestListAll_SkipsSubdirectories`) call `memory.ListAll(dir)`. Change each to:

```go
lister := memory.NewLister()
records, err := lister.ListAll(dir)
```

- [ ] **Step 6: Run all memory tests**

Run: `targ test -- -v ./internal/memory/`
Expected: PASS — all existing tests + new tests pass.

- [ ] **Step 7: Run `targ check-full`**

Run: `targ check-full`
Expected: PASS (or only pre-existing failures, which must be investigated and fixed).

- [ ] **Step 8: Commit**

```bash
git add internal/memory/readmodifywrite.go internal/memory/readmodifywrite_test.go
git commit -m "refactor(memory): add Lister type with DI for ListAll and ListStored

Replace package-level ListAll with Lister.ListAll method that accepts
injected readDir/readFile. Add ListStored for callers needing []*Stored
with UpdatedAt sorting. Prepares for retrieve package deletion (#454).

AI-Used: [claude]"
```

---

### Task 2: Update `maintain` callers to use `memory.NewLister()`

**Files:**
- Modify: `internal/maintain/maintain.go:31-33`
- Modify: `internal/cli/migrate_slugs.go:15`
- Modify: `internal/cli/migrate_slugs_test.go:65,113,161`
- Modify: `internal/cli/refine.go:143`

These callers use `ListAll` and return `[]StoredRecord`. They just need to create a `Lister` and call the method instead of the now-removed package function.

- [ ] **Step 1: Update `internal/maintain/maintain.go`**

At `maintain.go:31-33`, the current code is:
```go
memDir := memory.MemoriesDir(cfg.DataDir)
// blank line
records, err := memory.ListAll(memDir)
```

Replace with:
```go
memDir := memory.MemoriesDir(cfg.DataDir)

lister := memory.NewLister()
records, err := lister.ListAll(memDir)
```

- [ ] **Step 2: Update `internal/cli/migrate_slugs.go`**

At `migrate_slugs.go:15`, the current code is:
```go
records, listErr := memory.ListAll(dir)
```

Replace with:
```go
lister := memory.NewLister()
records, listErr := lister.ListAll(dir)
```

- [ ] **Step 3: Update `internal/cli/migrate_slugs_test.go`**

At lines 65, 113, and 161, the current code is:
```go
records, listErr := memory.ListAll(memoriesDir)
```

Replace each with:
```go
lister := memory.NewLister()
records, listErr := lister.ListAll(memoriesDir)
```

- [ ] **Step 4: Update `internal/cli/refine.go`**

At `refine.go:143`, the current code is:
```go
records, listErr := memory.ListAll(memoriesDir)
```

Replace with:
```go
lister := memory.NewLister()
records, listErr := lister.ListAll(memoriesDir)
```

- [ ] **Step 5: Run tests for modified packages**

Run: `targ test -- -v ./internal/maintain/ ./internal/cli/`
Expected: PASS — all existing tests pass with the new call syntax.

- [ ] **Step 6: Run `targ check-full`**

Run: `targ check-full`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/maintain/maintain.go internal/cli/migrate_slugs.go internal/cli/migrate_slugs_test.go internal/cli/refine.go
git commit -m "refactor(maintain): replace memory.ListAll calls with Lister method

Update maintain, migrate_slugs, and refine to use memory.NewLister().ListAll()
instead of the removed package-level function (#454).

AI-Used: [claude]"
```

---

### Task 3: Replace `retrieve` package with `memory.ListStored` in surface/correct/cli

**Files:**
- Modify: `internal/surface/surface.go:35-38`
- Modify: `internal/surface/surface_test.go:825`
- Modify: `internal/correct/correct.go:156-157`
- Modify: `internal/correct/correct_test.go` (all `MemoryRetrieverFunc` references)
- Modify: `internal/correct/correct.go:102-105` (remove unused `ctx` from `findCandidates`)
- Modify: `internal/cli/cli.go:27` (import), `:329`, `:344`, `:547-552`, `:704`, `:730`
- Modify: `internal/cli/adapters_test.go:353` (replace `ExportNewRetriever` with `memory.NewLister()`)
- Modify: `internal/cli/export_test.go:83-86` (delete `ExportNewRetriever`)
- Delete: `internal/retrieve/retrieve.go`
- Delete: `internal/retrieve/retrieve_test.go`

- [ ] **Step 1: Update `surface.MemoryRetriever` interface**

In `internal/surface/surface.go`, the current interface at line 35-38 is:

```go
// MemoryRetriever lists stored memories from disk (ARCH-9).
type MemoryRetriever interface {
	ListMemories(ctx context.Context, dataDir string) ([]*memory.Stored, error)
}
```

Replace with:

```go
// MemoryRetriever lists stored memories from disk (ARCH-9).
type MemoryRetriever interface {
	ListStored(dir string) ([]*memory.Stored, error)
}
```

Note: `ListStored` does not take `context.Context` (the original `ListMemories` accepted it but ignored it — `_ context.Context`). It also takes `dir` directly, not `dataDir` — the caller is now responsible for computing `memory.MemoriesDir(dataDir)` before calling.

- [ ] **Step 2: Update `surface.go` call site**

At `surface.go:180`, the current call is:

```go
memories, err := s.retriever.ListMemories(ctx, dataDir)
```

First, find the line where `dataDir` is computed. At approximately `surface.go:176-178`:

```go
dataDir := opts.DataDir
```

Update the call to:

```go
memories, err := s.retriever.ListStored(memory.MemoriesDir(dataDir))
```

If `memory` is not already imported, add `"engram/internal/memory"` to imports.

- [ ] **Step 3: Update `surface_test.go` fake**

At `surface_test.go:825`:

```go
func (f *fakeRetriever) ListMemories(_ context.Context, _ string) ([]*memory.Stored, error) {
```

Replace with:

```go
func (f *fakeRetriever) ListStored(_ string) ([]*memory.Stored, error) {
```

- [ ] **Step 4: Update `correct.MemoryRetrieverFunc`**

At `correct.go:156-157`:

```go
// MemoryRetrieverFunc retrieves all stored memories from a data directory.
type MemoryRetrieverFunc func(ctx context.Context, dataDir string) ([]*memory.Stored, error)
```

Replace with:

```go
// MemoryRetrieverFunc retrieves all stored memories from a directory.
type MemoryRetrieverFunc func(dir string) ([]*memory.Stored, error)
```

- [ ] **Step 5: Update `correct.go` call site**

At `correct.go:110`:

```go
allMemories, err := c.memoryRetriever(ctx, dataDir)
```

The `dataDir` here comes from the function parameter. The correct code needs to compute the memories dir. Find the `findCandidates` function signature — it takes `dataDir string`. Update the call:

```go
allMemories, err := c.memoryRetriever(memory.MemoriesDir(dataDir))
```

Add `"engram/internal/memory"` to the correct package imports if not already present.

- [ ] **Step 5b: Clean up vestigial `ctx` in `findCandidates`**

After removing `ctx` from `MemoryRetrieverFunc`, the `ctx` param in `findCandidates` is no longer used in the function body. At `correct.go:102-105`:

```go
func (c *Corrector) findCandidates(
	ctx context.Context,
	message, transcriptContext, dataDir string,
) ([]*memory.Stored, error) {
```

Remove the `ctx` parameter:

```go
func (c *Corrector) findCandidates(
	message, transcriptContext, dataDir string,
) ([]*memory.Stored, error) {
```

Update the call at `correct.go:74`:

```go
candidates, err := c.findCandidates(ctx, message, transcriptContext, dataDir)
```

Replace with:

```go
candidates, err := c.findCandidates(message, transcriptContext, dataDir)
```

Remove `"context"` import from `correct.go` if it becomes unused (check: `Run` method still takes `ctx` for its own use, so it likely stays).

- [ ] **Step 6: Update `correct_test.go` fakes**

All test fakes that define `MemoryRetrieverFunc` lambdas currently have signature `func(_ context.Context, _ string) ([]*memory.Stored, error)`. Update each to `func(_ string) ([]*memory.Stored, error)`.

Occurrences in `correct_test.go`:
- Line ~87
- Line ~141
- Line ~195
- Line ~329
- Line ~376-377
- Line ~431-432
- Line ~507
- Line ~543

Each looks like:
```go
func(_ context.Context, _ string) ([]*memory.Stored, error) {
```

Replace with:
```go
func(_ string) ([]*memory.Stored, error) {
```

Remove unused `"context"` import if it becomes unused.

- [ ] **Step 7: Update `cli.go` — remove `retrieve` import and update all call sites**

In `internal/cli/cli.go`:

**Import (line 27):** Remove `"engram/internal/retrieve"`.

**`buildRecallSurfacer` (lines 328-350):** Currently creates a `retrieve.New()` and calls `retriever.ListMemories`. Replace the whole function body:

Current:
```go
func buildRecallSurfacer(ctx context.Context, dataDir string) (recall.MemorySurfacer, error) {
	retriever := retrieve.New()

	_, memErr := retriever.ListMemories(ctx, dataDir)
	if memErr != nil {
		if errors.Is(memErr, os.ErrNotExist) {
			return nil, nil //nolint:nilnil // nil surfacer is valid when no memories exist
		}

		return nil, fmt.Errorf("listing memories: %w", memErr)
	}

	surfacerOpts := []surface.SurfacerOption{
		surface.WithSurfacingRecorder(recordSurfacing),
	}

	realSurfacer := surface.New(retriever, surfacerOpts...)

	return NewRecallSurfacer(
		&surfaceRunnerAdapter{surfacer: realSurfacer},
		dataDir,
	), nil
}
```

Replace with:
```go
func buildRecallSurfacer(ctx context.Context, dataDir string) (recall.MemorySurfacer, error) {
	lister := memory.NewLister()

	_, memErr := lister.ListStored(memory.MemoriesDir(dataDir))
	if memErr != nil {
		if errors.Is(memErr, os.ErrNotExist) {
			return nil, nil //nolint:nilnil // nil surfacer is valid when no memories exist
		}

		return nil, fmt.Errorf("listing memories: %w", memErr)
	}

	surfacerOpts := []surface.SurfacerOption{
		surface.WithSurfacingRecorder(recordSurfacing),
	}

	realSurfacer := surface.New(lister, surfacerOpts...)

	return NewRecallSurfacer(
		&surfaceRunnerAdapter{surfacer: realSurfacer},
		dataDir,
	), nil
}
```

Note: `ctx` parameter is kept in the function signature because the caller passes it, but `ListStored` no longer needs it. If `ctx` becomes unused in the function body, it may still be used elsewhere (check for `signalContext` or other uses). If truly unused, rename to `_ context.Context`.

**Correct pipeline (lines 547-552):** Currently:
```go
retriever := retrieve.New()

corrector := correct.New(
	correct.WithCaller(caller),
	correct.WithTranscriptReader(reader.Read),
	correct.WithMemoryRetriever(retriever.ListMemories),
```

Replace with:
```go
lister := memory.NewLister()

corrector := correct.New(
	correct.WithCaller(caller),
	correct.WithTranscriptReader(reader.Read),
	correct.WithMemoryRetriever(lister.ListStored),
```

**Surface pipeline (lines 704-730):** Currently:
```go
retriever := retrieve.New()
recorder := track.NewRecorder()
...
surfacer := surface.New(retriever, surfacerOpts...)
```

Replace with:
```go
lister := memory.NewLister()
recorder := track.NewRecorder()
...
surfacer := surface.New(lister, surfacerOpts...)
```

- [ ] **Step 8: Update `cli/adapters_test.go`**

At `adapters_test.go:353-354`, the current code is:
```go
retriever := cli.ExportNewRetriever()
surfacer := surface.New(retriever)
```

Replace with:
```go
lister := memory.NewLister()
surfacer := surface.New(lister)
```

Add `"engram/internal/memory"` to `adapters_test.go` imports if not already present.

- [ ] **Step 9: Delete `ExportNewRetriever` from `cli/export_test.go`**

Remove the `ExportNewRetriever` function (lines 83-86):
```go
// ExportNewRetriever creates a retrieve.Retriever for testing.
func ExportNewRetriever() *retrieve.Retriever {
	return retrieve.New()
}
```

Remove `"engram/internal/retrieve"` from the import block if it becomes unused.

- [ ] **Step 10: Delete the `retrieve` package**

```bash
rm internal/retrieve/retrieve.go internal/retrieve/retrieve_test.go
rmdir internal/retrieve
```

- [ ] **Step 11: Run full test suite**

Run: `targ test -- -v ./internal/...`
Expected: PASS — all packages compile and tests pass.

- [ ] **Step 12: Run `targ check-full`**

Run: `targ check-full`
Expected: PASS.

- [ ] **Step 13: Commit**

```bash
git add -A
git commit -m "refactor(retrieve): delete package, consolidate into memory.Lister (#454)

Replace retrieve.Retriever.ListMemories with memory.Lister.ListStored.
Update surface.MemoryRetriever interface, correct.MemoryRetrieverFunc type,
and all cli wiring to use the consolidated memory.Lister.

Closes #454.

AI-Used: [claude]"
```

---

## Migration Summary

| Before | After |
|--------|-------|
| `retrieve.New().ListMemories(ctx, dataDir)` | `memory.NewLister().ListStored(memory.MemoriesDir(dataDir))` |
| `memory.ListAll(dir)` | `memory.NewLister().ListAll(dir)` |
| `surface.MemoryRetriever` interface with `ListMemories(ctx, dataDir)` | `surface.MemoryRetriever` interface with `ListStored(dir)` |
| `correct.MemoryRetrieverFunc(ctx, dataDir)` | `correct.MemoryRetrieverFunc(dir)` |
| `internal/retrieve/` package (2 files) | Deleted |

## Risk Notes

- **`context.Context` removal:** `ListMemories` accepted `ctx` but never used it (`_ context.Context`). Removing it from the consolidated function is safe. If a future need arises, it can be added back.
- **`dataDir` vs `dir` semantics:** `ListMemories` took `dataDir` and internally called `MemoriesDir(dataDir)`. `ListStored` takes `dir` directly (the memories directory). Callers must now call `memory.MemoriesDir(dataDir)` themselves. This is more explicit and avoids hidden path manipulation.
- **Sorting:** `ListStored` sorts by `UpdatedAt` descending (matching `ListMemories` behavior). `ListAll` does not sort (matching original behavior). The maintain callers don't need sorting.
