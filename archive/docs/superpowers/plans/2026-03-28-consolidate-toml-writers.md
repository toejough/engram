# Consolidate Atomic TOML Writers — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate 4 independent atomic TOML write implementations into one shared pattern via `tomlwriter.Writer`.

**Architecture:** Add a public `AtomicWrite(targetPath string, record any) error` method to the existing `tomlwriter.Writer`. Migrate `track.Recorder`, `maintain.TOMLRewriter`, and `memory.ReadModifyWrite` to delegate to it. This also fixes DI violations in `ReadModifyWrite` and missing cleanup in `ReadModifyWrite`/`TOMLRewriter`.

**Tech Stack:** Go, BurntSushi/toml, os

---

### Task 1: Add AtomicWrite method to tomlwriter.Writer

**Files:**
- Modify: `internal/tomlwriter/tomlwriter.go` — add `AtomicWrite` public method
- Modify: `internal/tomlwriter/tomlwriter_test.go` — add tests for `AtomicWrite`

- [ ] **Step 1: Write failing test for AtomicWrite**

```go
func TestAtomicWrite_WritesAndRenames(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	targetPath := filepath.Join(dir, "test.toml")

	writer := tomlwriter.New()

	record := map[string]string{"key": "value"}
	err := writer.AtomicWrite(targetPath, record)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(targetPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring(`key = "value"`))
}

func TestAtomicWrite_CleansUpOnEncodeError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	targetPath := filepath.Join(dir, "test.toml")

	// A channel cannot be TOML-encoded — this will fail during encode.
	writer := tomlwriter.New()
	err := writer.AtomicWrite(targetPath, map[string]any{"bad": make(chan int)})
	g.Expect(err).To(HaveOccurred())

	// Verify no temp files remain.
	entries, dirErr := os.ReadDir(dir)
	g.Expect(dirErr).NotTo(HaveOccurred())

	if dirErr != nil {
		return
	}

	g.Expect(entries).To(BeEmpty())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test -- -run 'TestAtomicWrite_' ./internal/tomlwriter/...`
Expected: FAIL — method does not exist

- [ ] **Step 3: Implement AtomicWrite**

Extract the core atomic write logic from the existing private `writeAtomic` method into a new public `AtomicWrite` method. The existing `writeAtomic` (used by `Write`) can then call `AtomicWrite` internally.

```go
// AtomicWrite writes record as TOML to targetPath atomically via temp file + rename.
// The record must be TOML-serializable. On any failure, the temp file is cleaned up.
func (w *Writer) AtomicWrite(targetPath string, record any) error {
	dir := filepath.Dir(filepath.Clean(targetPath))
	cleanPath := filepath.Clean(targetPath)

	tempFile, err := w.createTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	tempPath := tempFile.Name()
	remove := func() { _ = w.remove(tempPath) }

	if encErr := toml.NewEncoder(tempFile).Encode(record); encErr != nil {
		_ = tempFile.Close()
		remove()

		return fmt.Errorf("encoding TOML: %w", encErr)
	}

	if closeErr := tempFile.Close(); closeErr != nil {
		remove()

		return fmt.Errorf("closing temp file: %w", closeErr)
	}

	if renameErr := w.rename(tempPath, cleanPath); renameErr != nil {
		remove()

		return fmt.Errorf("renaming temp file: %w", renameErr)
	}

	return nil
}
```

Then update the existing `writeAtomic` to delegate:
```go
func (w *Writer) writeAtomic(memoriesDir, finalPath string, record memory.MemoryRecord) error {
	return w.AtomicWrite(finalPath, record)
}
```

- [ ] **Step 4: Run all tomlwriter tests to verify they pass**

Run: `targ test -- ./internal/tomlwriter/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tomlwriter/
git commit -m "refactor(tomlwriter): add public AtomicWrite method

Extracts the atomic temp-file-then-rename pattern for reuse by other packages.

Refs #410

AI-Used: [claude]"
```

### Task 2: Migrate track.Recorder to use tomlwriter.AtomicWrite

**Files:**
- Modify: `internal/track/recorder.go` — replace private `writeAtomic` with tomlwriter delegation
- Modify: `internal/track/recorder_test.go` (if it exists) — update test setup

- [ ] **Step 1: Run existing track tests to confirm green baseline**

Run: `targ test -- ./internal/track/...`
Expected: PASS

- [ ] **Step 2: Refactor Recorder**

Remove from `Recorder`:
- `createTemp`, `rename`, `remove` fields (replaced by `tomlwriter.Writer`)
- The private `writeAtomic` method

Add a `writer *tomlwriter.Writer` field. Update constructor:
```go
func New(opts ...Option) *Recorder {
	r := &Recorder{
		readFile: os.ReadFile,
		writer:   tomlwriter.New(),
		now:      time.Now,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}
```

Add `WithWriter` option for test injection:
```go
func WithWriter(w *tomlwriter.Writer) Option {
	return func(r *Recorder) { r.writer = w }
}
```

In `updateMemoryFile`, replace `r.writeAtomic(record, mem.FilePath)` with `r.writer.AtomicWrite(mem.FilePath, record)`.

Remove now-unused `WithCreateTemp`, `WithRename`, `WithRemove` options if they exist.

- [ ] **Step 3: Run track tests to verify they pass**

Run: `targ test -- ./internal/track/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/track/
git commit -m "refactor(track): delegate atomic writes to tomlwriter

Removes duplicated atomic write logic.

Refs #410

AI-Used: [claude]"
```

### Task 3: Migrate maintain.TOMLRewriter to use tomlwriter.AtomicWrite

**Files:**
- Modify: `internal/maintain/rewriter.go` — delegate to tomlwriter

- [ ] **Step 1: Run existing maintain tests to confirm green baseline**

Run: `targ test -- ./internal/maintain/...`
Expected: PASS

- [ ] **Step 2: Refactor TOMLRewriter**

Remove from `TOMLRewriter`:
- `writeFile` and `rename` fields (replaced by `tomlwriter.Writer`)

Add a `writer *tomlwriter.Writer` field. Update constructor and options similarly to Task 2.

In `Rewrite`, replace the encode-to-buffer / writeFile / rename block with `rewriter.writer.AtomicWrite(path, existing)`. This also fixes the missing temp file cleanup.

- [ ] **Step 3: Run maintain tests to verify they pass**

Run: `targ test -- ./internal/maintain/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/maintain/
git commit -m "refactor(maintain): delegate atomic writes to tomlwriter

Also fixes missing temp file cleanup on write/rename failure.

Refs #410

AI-Used: [claude]"
```

### Task 4: Refactor memory.ReadModifyWrite to use DI and tomlwriter

**Files:**
- Modify: `internal/memory/readmodifywrite.go` — convert to struct with DI, delegate atomic write

- [ ] **Step 1: Run existing memory tests to confirm green baseline**

Run: `targ test -- ./internal/memory/...`
Expected: PASS

- [ ] **Step 2: Refactor ReadModifyWrite**

Convert the package-level `ReadModifyWrite` function to a method on a new `Modifier` struct (or accept a `tomlwriter.Writer` parameter). The function currently takes `(path string, mutate func(*MemoryRecord)) error` and calls `os.ReadFile`, `os.WriteFile`, `os.Rename` directly.

Option A (minimal change): Add `readFile` and `writer` parameters:
```go
func ReadModifyWrite(path string, mutate func(*MemoryRecord), readFile func(string) ([]byte, error), writer *tomlwriter.Writer) error
```

Option B (struct): Create a `Modifier` struct with DI fields. This is more consistent with the rest of the codebase.

Use Option B for consistency. Update all callers to create a `Modifier` (check cli.go for call sites).

This also fixes: hardcoded `.tmp-rmw` temp name (now random), missing cleanup on failure, and DI violation.

- [ ] **Step 3: Run full test suite and lint**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/memory/ internal/cli/
git commit -m "refactor(memory): use DI and tomlwriter for ReadModifyWrite

Fixes DI violation (direct os.* calls), hardcoded temp file name,
and missing cleanup on failure.

Closes #410

AI-Used: [claude]"
```
