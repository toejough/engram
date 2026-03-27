# Eliminate Registry Abstraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove `internal/registry/` and `internal/register/` packages — all memory metadata lives in TOML files via `memory.MemoryRecord` with a shared `ReadModifyWrite` helper.

**Architecture:** Expand `MemoryRecord` with registry-only fields (links, absorbed, enforcement, provenance). Add `ReadModifyWrite` and `ListAll` helpers. Rewire each consumer one at a time (learn, surface, evaluate, maintain, signal, graph). Delete crossref external instruction tracking. Delete registry and register packages last.

**Tech Stack:** Go, BurntSushi/toml, gomega

**Spec:** `docs/superpowers/specs/2026-03-20-eliminate-registry-abstraction.md`

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/memory/record.go` | Modify | Add nested types + registry-only fields to `MemoryRecord` |
| `internal/memory/readmodifywrite.go` | Create | Atomic `ReadModifyWrite` + `ListAll` helpers |
| `internal/memory/readmodifywrite_test.go` | Create | Tests for helpers |
| `internal/cli/cli.go` | Modify | Remove all registry adapters and wiring |
| `internal/learn/learn.go` | Modify | Remove `RegistryRegistrar`/`RegistryAbsorber`, use injected `ReadModifyWrite` |
| `internal/surface/surface.go` | Modify | Remove `RegistryRecorder`, use injected recorder func |
| `internal/evaluate/evaluator.go` | Modify | Remove `RegistryRecorder`, use injected recorder func |
| `internal/maintain/apply.go` | Modify | Remove `RegistryUpdater`, use injected file remover |
| `internal/cli/signal.go` | Modify | Remove registry adapters, use injected funcs |
| `internal/graph/recompute.go` | Modify | Remove `RegistryLinker`, use new interfaces backed by `MemoryRecord` |
| `internal/crossref/extract.go` | Modify | Remove registry entry creation |
| `internal/register/` | Delete | Entire package |
| `internal/registry/` | Delete | Entire package |

---

## Task 1: Expand MemoryRecord with nested types and registry-only fields

**Files:**
- Modify: `internal/memory/record.go`
- Modify: `internal/memory/record_test.go`

- [ ] **Step 1: Write the failing test**

Add a round-trip test that includes all new fields (links, absorbed with nested evaluations, transitions, provenance, enforcement):

```go
func TestMemoryRecord_RoundTrip_RegistryFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	original := memory.MemoryRecord{
		Title:            "test",
		Content:          "content",
		SourceType:       "memory",
		SourcePath:       "/path/to/source",
		ContentHash:      "abc123",
		EnforcementLevel: "advisory",
		Transitions: []memory.TransitionRecord{{
			From: "advisory", To: "reminder", At: "2026-01-01T00:00:00Z", Reason: "test",
		}},
		Links: []memory.LinkRecord{{
			Target: "other.toml", Weight: 0.8, Basis: "concept_overlap", CoSurfacingCount: 3,
		}},
		Absorbed: []memory.AbsorbedRecord{{
			From: "old.toml", SurfacedCount: 5, ContentHash: "def456", MergedAt: "2026-01-02T00:00:00Z",
			Evaluations: memory.EvaluationCounters{Followed: 2, Contradicted: 1, Ignored: 0},
		}},
	}

	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(original)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	var decoded memory.MemoryRecord
	_, err = toml.Decode(buf.String(), &decoded)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	g.Expect(decoded).To(Equal(original))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `memory.TransitionRecord`, `memory.LinkRecord`, `memory.AbsorbedRecord`, `memory.EvaluationCounters` undefined

- [ ] **Step 3: Add the types and fields to record.go**

Add to `internal/memory/record.go`:

```go
// Nested types for registry-originated fields.

type LinkRecord struct {
	Target           string  `toml:"target"`
	Weight           float64 `toml:"weight"`
	Basis            string  `toml:"basis"`
	CoSurfacingCount int     `toml:"co_surfacing_count,omitempty"`
}

type AbsorbedRecord struct {
	From          string             `toml:"from"`
	SurfacedCount int                `toml:"surfaced_count"`
	Evaluations   EvaluationCounters `toml:"evaluations"`
	ContentHash   string             `toml:"content_hash"`
	MergedAt      string             `toml:"merged_at"`
}

type EvaluationCounters struct {
	Followed     int `toml:"followed"`
	Contradicted int `toml:"contradicted"`
	Ignored      int `toml:"ignored"`
}

type TransitionRecord struct {
	From   string `toml:"from"`
	To     string `toml:"to"`
	At     string `toml:"at"`
	Reason string `toml:"reason"`
}
```

Add fields to `MemoryRecord`:

```go
// Provenance.
SourceType  string `toml:"source_type,omitempty"`
SourcePath  string `toml:"source_path,omitempty"`
ContentHash string `toml:"content_hash,omitempty"`

// Enforcement escalation.
EnforcementLevel string             `toml:"enforcement_level,omitempty"`
Transitions      []TransitionRecord `toml:"transitions,omitempty"`

// Relationships.
Links    []LinkRecord     `toml:"links,omitempty"`
Absorbed []AbsorbedRecord `toml:"absorbed,omitempty"`
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full checks**

Run: `targ check-full`

- [ ] **Step 6: Commit**

```
git commit -m "feat(memory): add registry-originated fields to MemoryRecord (#354)

Links, absorbed records, enforcement level, transitions, and
provenance fields now part of the canonical struct. Prerequisite
for eliminating the registry abstraction.

AI-Used: [claude]"
```

---

## Task 2: Add ReadModifyWrite and ListAll helpers

**Files:**
- Create: `internal/memory/readmodifywrite.go`
- Create: `internal/memory/readmodifywrite_test.go`

- [ ] **Step 1: Write the failing test for ReadModifyWrite**

```go
func TestReadModifyWrite_IncrementsField(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")

	// Write initial TOML
	initial := MemoryRecord{Title: "test", SurfacedCount: 3}
	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(initial)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }
	err = os.WriteFile(path, buf.Bytes(), 0o644)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	// ReadModifyWrite to increment
	err = ReadModifyWrite(path, func(r *MemoryRecord) {
		r.SurfacedCount++
	})
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	// Read back and verify
	data, err := os.ReadFile(path)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }
	var result MemoryRecord
	_, err = toml.Decode(string(data), &result)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	g.Expect(result.SurfacedCount).To(Equal(4))
	g.Expect(result.Title).To(Equal("test"))
}
```

Note: This test uses real file I/O because `ReadModifyWrite` is an edge function (like `tomlwriter.Write`). It lives in the `memory` package as a thin I/O wrapper. Consumers inject it via `func(string, func(*MemoryRecord)) error` for testability.

- [ ] **Step 2: Write the failing test for ListAll**

```go
func TestListAll_ReadsAllTOMLFiles(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()

	// Write two memory files
	for _, name := range []string{"a.toml", "b.toml"} {
		rec := MemoryRecord{Title: name}
		var buf bytes.Buffer
		_ = toml.NewEncoder(&buf).Encode(rec)
		_ = os.WriteFile(filepath.Join(dir, name), buf.Bytes(), 0o644)
	}
	// Write a non-TOML file (should be skipped)
	_ = os.WriteFile(filepath.Join(dir, "readme.md"), []byte("skip"), 0o644)

	records, err := ListAll(dir)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	g.Expect(records).To(HaveLen(2))
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL — `ReadModifyWrite` and `ListAll` undefined

- [ ] **Step 4: Implement ReadModifyWrite and ListAll**

```go
// internal/memory/readmodifywrite.go
package memory

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ReadModifyWrite atomically reads a memory TOML, applies a mutation, and writes back.
func ReadModifyWrite(path string, mutate func(*MemoryRecord)) error {
	data, err := os.ReadFile(path) //nolint:gosec // path from trusted internal source
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var record MemoryRecord
	_, err = toml.Decode(string(data), &record)
	if err != nil {
		return fmt.Errorf("decoding %s: %w", path, err)
	}

	mutate(&record)

	var buf bytes.Buffer
	err = toml.NewEncoder(&buf).Encode(record)
	if err != nil {
		return fmt.Errorf("encoding %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	tmpPath := filepath.Join(dir, ".tmp-rmw")

	const filePerm = 0o644
	err = os.WriteFile(tmpPath, buf.Bytes(), filePerm)
	if err != nil {
		return fmt.Errorf("writing temp: %w", err)
	}

	return os.Rename(tmpPath, path)
}

// StoredRecord pairs a file path with its parsed MemoryRecord.
type StoredRecord struct {
	Path   string
	Record MemoryRecord
}

// ListAll reads all .toml files from a directory, returning parsed records.
func ListAll(dir string) ([]StoredRecord, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var records []StoredRecord
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, readErr := os.ReadFile(path) //nolint:gosec // trusted dir
		if readErr != nil {
			continue // skip unreadable files
		}

		var record MemoryRecord
		_, decErr := toml.Decode(string(data), &record)
		if decErr != nil {
			continue // skip unparseable files
		}

		records = append(records, StoredRecord{Path: path, Record: record})
	}

	return records, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 6: Run full checks**

Run: `targ check-full`

- [ ] **Step 7: Commit**

```
git commit -m "feat(memory): add ReadModifyWrite and ListAll helpers (#354)

Atomic read-modify-write for memory TOMLs and directory listing.
These replace the Registry interface methods for all consumers.

AI-Used: [claude]"
```

---

## Task 3: Rewire Surface consumer

Surface is the simplest consumer — just `RecordSurfacing(id)` which increments `surfaced_count`. Also has `LinkReader`/`LinkUpdater` for spreading activation.

**Files:**
- Modify: `internal/surface/surface.go`
- Modify: `internal/surface/surface_test.go` (as needed)
- Modify: `internal/cli/cli.go` (remove `surfaceRegistryAdapter`, update wiring)

- [ ] **Step 1: Read the current surface.go interfaces and their usage**

Read `internal/surface/surface.go` to find:
- `RegistryRecorder` interface (lines 122-125) and where it's called
- `LinkReader` interface (lines 84-87) and where it's called
- `LinkUpdater` interface (lines 89-93) and where it's called
- `WithRegistry`, `WithLinkReader`, `WithLinkUpdater` option funcs

- [ ] **Step 2: Replace RegistryRecorder with a simpler injected function**

In `surface.go`:
- Delete the `RegistryRecorder` interface
- Replace the `registryRecorder` field on `Surfacer` with `recordSurfacing func(path string) error`
- Update `WithRegistry` option to accept `func(path string) error` instead
- Update all call sites where `s.registryRecorder.RecordSurfacing(id)` is called

- [ ] **Step 3: Replace LinkReader/LinkUpdater with MemoryRecord-based interfaces**

The `LinkGraphLink` type in surface.go maps to `memory.LinkRecord`. Replace:
- `LinkReader.GetEntryLinks(id)` → read TOML, return `record.Links`
- `LinkUpdater.SetEntryLinks(id, links)` → `ReadModifyWrite(path, func(r) { r.Links = links })`

Define new simpler interfaces (or use `func` types) that operate on `memory.LinkRecord` instead of `LinkGraphLink`.

- [ ] **Step 4: Update cli.go wiring**

In `cli.go`:
- Delete `surfaceRegistryAdapter` struct
- Replace `surface.WithRegistry(&surfaceRegistryAdapter{reg: registry})` with:
  ```go
  surface.WithSurfacingRecorder(func(path string) error {
      return memory.ReadModifyWrite(path, func(r *memory.MemoryRecord) {
          r.SurfacedCount++
          r.LastSurfacedAt = time.Now().UTC().Format(time.RFC3339)
      })
  })
  ```
- Update link reader/updater wiring similarly

- [ ] **Step 5: Update tests**

Update surface tests that mock `RegistryRecorder` to mock the new `func` type instead.

- [ ] **Step 6: Run tests and full checks**

Run: `targ test` then `targ check-full`

- [ ] **Step 7: Commit**

```
git commit -m "refactor(surface): replace RegistryRecorder with direct ReadModifyWrite (#354)

Surface pipeline no longer depends on internal/registry. Surfacing
events and link updates go through memory.ReadModifyWrite.

AI-Used: [claude]"
```

---

## Task 4: Rewire Evaluate consumer

**Files:**
- Modify: `internal/evaluate/evaluator.go`
- Modify: `internal/evaluate/evaluator_test.go`
- Modify: `internal/cli/cli.go` (remove `evaluateRegistryAdapter`)

- [ ] **Step 1: Read evaluator.go to find RegistryRecorder usage**

Find `RegistryRecorder` interface (lines 313-316), `WithRegistry` option, and call sites.

- [ ] **Step 2: Replace RegistryRecorder with injected function**

- Delete `RegistryRecorder` interface
- Replace field with `recordEvaluation func(path, outcome string) error`
- Update `WithRegistry` to accept `func(path, outcome string) error`

- [ ] **Step 3: Update cli.go wiring**

Delete `evaluateRegistryAdapter`. Replace with:
```go
evaluate.WithEvaluationRecorder(func(path, outcome string) error {
    return memory.ReadModifyWrite(path, func(r *memory.MemoryRecord) {
        switch outcome {
        case "followed":
            r.FollowedCount++
        case "contradicted":
            r.ContradictedCount++
        case "ignored":
            r.IgnoredCount++
        }
    })
})
```

- [ ] **Step 4: Update tests, run checks, commit**

Run: `targ test` then `targ check-full`

```
git commit -m "refactor(evaluate): replace RegistryRecorder with direct ReadModifyWrite (#354)

AI-Used: [claude]"
```

---

## Task 5: Rewire Maintain consumer

**Files:**
- Modify: `internal/maintain/apply.go`
- Modify: `internal/maintain/apply_test.go`
- Modify: `internal/cli/cli.go` (remove `registryEntryRemover`)

- [ ] **Step 1: Read apply.go to find RegistryUpdater usage**

Find `RegistryUpdater` interface (lines 292-295), `WithRegistry` option, and call sites.

- [ ] **Step 2: Replace RegistryUpdater with injected file remover**

- Delete `RegistryUpdater` interface (method is `RemoveEntry(id string) error`, not `Remove`)
- Replace field with `removeFile func(path string) error`
- Update `WithRegistry` to `WithFileRemover(func(path string) error)`
- Update all call sites that call `e.registry.RemoveEntry(...)` to call the new func

- [ ] **Step 3: Update cli.go wiring**

Delete `registryEntryRemover`. Replace with:
```go
maintain.WithFileRemover(os.Remove)
```

- [ ] **Step 4: Update tests, run checks, commit**

Run: `targ test` then `targ check-full`

```
git commit -m "refactor(maintain): replace RegistryUpdater with file remover (#354)

AI-Used: [claude]"
```

---

## Task 6: Rewire Learn consumer

**Files:**
- Modify: `internal/learn/learn.go`
- Modify: `internal/learn/learn_test.go`
- Modify: `internal/cli/cli.go` (remove `learnRegistryAdapter`)

- [ ] **Step 1: Read learn.go to find RegistryRegistrar and RegistryAbsorber usage**

Find both interfaces, their setters, and call sites.

- [ ] **Step 2: Replace RegistryRegistrar**

The learn pipeline already writes memory TOMLs via `tomlwriter.Write`. The registrar just sets provenance (source_type, content_hash). Replace with an injected function that does `ReadModifyWrite` to set provenance fields after the TOML is written.

- [ ] **Step 3: Replace RegistryAbsorber**

The absorber appends to the target memory's `absorbed` array. Replace with an injected function that does `ReadModifyWrite` to append an `AbsorbedRecord`.

- [ ] **Step 4: Update cli.go wiring**

Delete `learnRegistryAdapter`. Replace with injected funcs backed by `memory.ReadModifyWrite`.

- [ ] **Step 5: Update tests, run checks, commit**

Run: `targ test` then `targ check-full`

```
git commit -m "refactor(learn): replace registry interfaces with ReadModifyWrite (#354)

AI-Used: [claude]"
```

---

## Task 7: Rewire Signal consumer

**Files:**
- Modify: `internal/cli/signal.go` (remove `consolidatorRegistryAdapter`, `registryUpdaterAdapter`, `openRegistry`)
- Modify: `internal/cli/cli.go` (update signal wiring)
- Modify: `internal/signal/` tests as needed

- [ ] **Step 1: Read signal.go to find all registry adapter usage**

Find `consolidatorRegistryAdapter` (lines 38-56), `registryUpdaterAdapter` (lines 208-240), and `openRegistry` usage.

- [ ] **Step 2: Replace consolidatorRegistryAdapter**

The consolidator uses `RemoveEntry(path)` to delete absorbed memories. Replace with injected `func(path string) error` backed by `os.Remove`.

- [ ] **Step 3: Replace registryUpdaterAdapter**

**Important:** The `registryUpdaterAdapter` satisfies BOTH `signal.RegistryUpdater` AND `maintain.EnforcementApplier` (defined in `internal/maintain/escalation.go`). The replacement must cover both interfaces or provide separate implementations.

Three methods on `registryUpdaterAdapter`:
- `Remove(id)` → injected `func(path string) error` backed by `os.Remove`
- `SetEnforcementLevel(id, level, reason)` → injected func backed by `memory.ReadModifyWrite` — this is used by both `signal.WithRegistry` and `signal.WithEnforcementApplier`
- `UpdateContentHash(id, hash)` → injected func backed by `memory.ReadModifyWrite`

The `WithEnforcementApplier` wiring (signal.go line 513) must also be updated to use a `ReadModifyWrite`-backed func, not the old adapter.

- [ ] **Step 4: Remove openRegistry calls from signal.go**

Delete the `openRegistry` function if it's only used for signal wiring. (It may also be used by other commands in cli.go — check before deleting.)

- [ ] **Step 5: Update tests, run checks, commit**

Run: `targ test` then `targ check-full`

```
git commit -m "refactor(signal): replace registry adapters with injected funcs (#354)

AI-Used: [claude]"
```

---

## Task 8: Rewire Graph consumer

**Files:**
- Modify: `internal/graph/recompute.go`
- Modify: `internal/graph/graph.go`
- Modify: `internal/graph/recompute_test.go`

- [ ] **Step 1: Read recompute.go to find RegistryLinker usage**

Find `RegistryLinker` interface (lines 87-91) and `RecomputeMergeLinks` function.

- [ ] **Step 2: Replace RegistryLinker with MemoryRecord-based interfaces**

```go
type MemoryLister interface {
    ListAll() ([]memory.StoredRecord, error)
}

type LinkWriter interface {
    WriteLinks(path string, links []memory.LinkRecord) error
}
```

Update `RecomputeMergeLinks` to accept these instead of `RegistryLinker`. The conversion from `registry.InstructionEntry` + `registry.Link` to `memory.StoredRecord` + `memory.LinkRecord` happens here.

- [ ] **Step 3: Update graph.go Builder API signatures**

The `Builder` public methods use `registry.InstructionEntry` and `registry.Link` as parameter and return types:
- `BuildConceptOverlap(entry registry.InstructionEntry, existing []registry.InstructionEntry) []registry.Link`
- `BuildContentSimilarity(entry registry.InstructionEntry, existing []registry.InstructionEntry) []registry.Link`

These signatures must change to:
- `BuildConceptOverlap(entry memory.MemoryRecord, existing []memory.MemoryRecord) []memory.LinkRecord`
- `BuildContentSimilarity(entry memory.MemoryRecord, existing []memory.MemoryRecord) []memory.LinkRecord`

Also in `recompute.go`, the internal construction of `registry.InstructionEntry` (around line 51-56) must be replaced with `memory.MemoryRecord` field access. The `recompute.go` function `RecomputeMergeLinks` internally converts `StoredRecord` → `InstructionEntry` for the builder — this conversion is eliminated.

- [ ] **Step 4: Update cli.go/signal.go wiring for graph**

The `graphLinkRecomputer` in signal.go creates the graph builder and passes the registry. Replace with wiring that passes `ListAll` and `ReadModifyWrite`-backed link writer.

- [ ] **Step 5: Update tests, run checks, commit**

Run: `targ test` then `targ check-full`

```
git commit -m "refactor(graph): replace RegistryLinker with MemoryRecord interfaces (#354)

AI-Used: [claude]"
```

---

## Task 9: Remove CrossRef external instruction tracking

**Files:**
- Modify: `internal/crossref/extract.go`
- Modify: `internal/crossref/extract_test.go`
- Modify: `internal/cli/cli.go` (remove crossref registry wiring)

- [ ] **Step 1: Read extract.go to understand what it does**

It creates `registry.InstructionEntry` objects from CLAUDE.md, rules, and skills. These are registered for cross-reference checking during surfacing.

- [ ] **Step 2: Remove registry.InstructionEntry creation**

Delete or modify the extractors so they no longer create `registry.InstructionEntry` objects. If the extracted data is used for anything other than registry registration (e.g., cross-reference suppression in surfacing), preserve that functionality using a simpler type.

- [ ] **Step 3: Update cli.go wiring**

Remove any code that passes crossref results to the registry.

- [ ] **Step 4: Update tests, run checks, commit**

Run: `targ test` then `targ check-full`

```
git commit -m "refactor(crossref): drop external instruction tracking (#354)

CLAUDE.md, rules, and skills are loaded every session and don't
need effectiveness tracking.

AI-Used: [claude]"
```

---

## Task 10: Delete internal/register/ package

**Files:**
- Delete: `internal/register/register.go`
- Delete: `internal/register/register_test.go`
- Modify: `internal/cli/cli.go` (remove register wiring)

- [ ] **Step 1: Verify no remaining imports**

Run: `grep -rn '"engram/internal/register"' --include='*.go' | grep -v '_test.go'`

If any imports remain, rewire them first.

- [ ] **Step 2: Remove register wiring from cli.go**

Delete any code that creates a `register.Registrar` and calls it.

- [ ] **Step 3: Delete the package**

```bash
rm -rf internal/register/
```

- [ ] **Step 4: Run tests and full checks**

Run: `targ test` then `targ check-full`

- [ ] **Step 5: Commit**

```
git commit -m "chore: delete internal/register/ package (#354)

Registrar pipeline replaced by direct ReadModifyWrite calls.

AI-Used: [claude]"
```

---

## Task 11: Delete internal/registry/ package

**Files:**
- Delete: `internal/registry/` (all files)
- Modify: any remaining files that import `engram/internal/registry`

- [ ] **Step 1: Verify no remaining imports**

Run: `grep -rn '"engram/internal/registry"' --include='*.go' | grep -v '_test.go'`

If any imports remain, they must be rewired first. This should be zero after Tasks 3-10.

- [ ] **Step 2: Delete the package**

```bash
rm -rf internal/registry/
```

- [ ] **Step 3: Run tests and full checks**

Run: `targ test` then `targ check-full`

- [ ] **Step 4: Commit**

```
git commit -m "chore: delete internal/registry/ package (#354)

Registry abstraction eliminated. All memory metadata managed
directly via memory.MemoryRecord and ReadModifyWrite.

AI-Used: [claude]"
```

---

## Task 12: Clean up openRegistry and remaining references

**Files:**
- Modify: `internal/cli/cli.go` (delete `openRegistry` func if now unused)
- Modify: any files with stale UC-23 comments that reference the deleted registry

- [ ] **Step 1: Search for remaining registry references**

```bash
grep -rn 'openRegistry\|regpkg\|registry\.' --include='*.go' | grep -v '_test.go' | grep -v 'internal/registry/'
```

- [ ] **Step 2: Remove stale references**

Delete `openRegistry` function, remove `regpkg` import alias, update any stale comments.

- [ ] **Step 3: Run full checks and commit**

Run: `targ check-full`

```
git commit -m "chore: clean up remaining registry references (#354)

AI-Used: [claude]"
```
