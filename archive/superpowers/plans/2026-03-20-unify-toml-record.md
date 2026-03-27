# Unify tomlRecord Struct Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create one canonical TOML struct for memory files so feedback counters survive surfacing cycles (#353).

**Architecture:** Define `MemoryRecord` in `internal/memory/record.go` with all content + tracking fields. Replace all 6+ divergent struct definitions with this single type. The critical fix is that `RecordSurfacing` stops stripping tracking fields. `MemoryRecord` is for TOML I/O fidelity; `memory.Stored` remains the in-memory domain type — they don't need to be 1:1. Adding missing content fields to `Stored` is out of scope for #353.

**Tech Stack:** Go, BurntSushi/toml, gomega

**Scope notes:**
- `feedback.go` currently calls `os.ReadFile` directly (DI violation). Fixing that is deferred — this plan focuses on the data-loss bug.
- `internal/evaluate/evaluator.go` has its own `memoryTOML` struct (read-only, 4 fields). Out of scope — it can't cause data loss since it only reads, never writes.
- `storedMemoryWriter.Write` in signal.go doesn't carry content fields that `Stored` lacks (ObservationType, Rationale, etc.). This content-field loss is a pre-existing issue, not caused by #353. Filed as follow-up in #354's scope.

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/memory/record.go` | Create | Canonical `MemoryRecord` struct |
| `internal/memory/record_test.go` | Create | Round-trip encoding test |
| `internal/track/recorder.go` | Modify | Use `memory.MemoryRecord`, stop stripping fields |
| `internal/track/recorder_test.go` | Modify | Test fields are preserved (not stripped), replace `fullTOMLRecord` |
| `internal/tomlwriter/tomlwriter.go` | Modify | Use `memory.MemoryRecord` |
| `internal/tomlwriter/tomlwriter_test.go` | Modify | Verify tracking fields present in raw TOML |
| `internal/retrieve/retrieve.go` | Modify | Use `memory.MemoryRecord` |
| `internal/retrieve/retrieve_test.go` | Modify | Verify all fields parsed |
| `internal/cli/show.go` | Modify | Use `memory.MemoryRecord` |
| `internal/cli/show_test.go` | Modify | Update assertions |
| `internal/cli/signal.go` | Modify | Use `memory.MemoryRecord` for read/write (function-scoped types) |
| `internal/cli/feedback.go` | Modify | Use `memory.MemoryRecord` instead of `map[string]any` |
| `internal/cli/feedback_test.go` | Modify | Update for struct-based approach |

---

## Task 1: Define canonical MemoryRecord

**Files:**
- Create: `internal/memory/record.go`
- Create: `internal/memory/record_test.go`

- [ ] **Step 1: Write the round-trip test**

```go
// internal/memory/record_test.go
package memory_test

import (
	"bytes"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

func TestMemoryRecord_RoundTrip(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	original := memory.MemoryRecord{
		Title:             "test title",
		Content:           "test content",
		ObservationType:   "workflow_instruction",
		Concepts:          []string{"a", "b"},
		Keywords:          []string{"k1", "k2"},
		Principle:         "test principle",
		AntiPattern:       "test anti-pattern",
		Rationale:         "test rationale",
		Confidence:        "A",
		CreatedAt:         "2026-01-01T00:00:00Z",
		UpdatedAt:         "2026-01-02T00:00:00Z",
		SurfacedCount:     5,
		FollowedCount:     3,
		ContradictedCount: 1,
		IgnoredCount:      2,
		IrrelevantCount:   4,
		LastSurfacedAt:    "2026-01-03T00:00:00Z",
	}

	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(original)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var decoded memory.MemoryRecord
	_, err = toml.Decode(buf.String(), &decoded)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(decoded).To(Equal(original))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `memory.MemoryRecord` undefined

- [ ] **Step 3: Implement MemoryRecord**

```go
// internal/memory/record.go
package memory

// MemoryRecord is the canonical struct for reading and writing memory TOML files.
// ALL code that touches memory TOML must use this struct to prevent field loss.
// See #353 for the bug caused by divergent struct definitions.
type MemoryRecord struct {
	// Content fields.
	Title           string   `toml:"title"`
	Content         string   `toml:"content"`
	ObservationType string   `toml:"observation_type"`
	Concepts        []string `toml:"concepts"`
	Keywords        []string `toml:"keywords"`
	Principle       string   `toml:"principle"`
	AntiPattern     string   `toml:"anti_pattern"`
	Rationale       string   `toml:"rationale"`
	Confidence      string   `toml:"confidence"`
	CreatedAt       string   `toml:"created_at"`
	UpdatedAt       string   `toml:"updated_at"`

	// Tracking fields — feedback counters and surfacing metadata.
	SurfacedCount     int    `toml:"surfaced_count"`
	FollowedCount     int    `toml:"followed_count"`
	ContradictedCount int    `toml:"contradicted_count"`
	IgnoredCount      int    `toml:"ignored_count"`
	IrrelevantCount   int    `toml:"irrelevant_count"`
	LastSurfacedAt    string `toml:"last_surfaced_at"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: Only `check-uncommitted` fails

- [ ] **Step 6: Commit**

```
git add internal/memory/record.go internal/memory/record_test.go
git commit -m "feat(memory): add canonical MemoryRecord struct (#353)

All memory TOML read/write paths will use this single struct
to prevent field loss from divergent struct definitions.

AI-Used: [claude]"
```

---

## Task 2: Fix RecordSurfacing — stop stripping tracking fields (critical fix)

This is the core bug fix. `RecordSurfacing` currently decodes into a `tomlRecord` struct that lacks tracking fields, then re-encodes — stripping feedback counters.

**Files:**
- Modify: `internal/track/recorder.go`
- Modify: `internal/track/recorder_test.go`

- [ ] **Step 1: Write the failing test**

First, update the `writeCapture.decodeTOML` helper and `fullTOMLRecord` to use `memory.MemoryRecord` so tests can verify tracking fields. Then add the new test:

In `recorder_test.go`, replace the `fullTOMLRecord` struct (lines 398-412) and `decodeTOML` method (lines 438-450):

```go
// Delete fullTOMLRecord entirely — replaced by memory.MemoryRecord.

func (w *writeCapture) decodeTOML(g Gomega) memory.MemoryRecord {
	g.Expect(w.tmpPath).NotTo(BeEmpty(), "no temp file was written")

	data, err := os.ReadFile(w.tmpPath)
	g.Expect(err).NotTo(HaveOccurred())

	var record memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &record)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	return record
}
```

Add import for `"engram/internal/memory"` (already imported).

Then add the new test:

```go
// TestT353_RecordSurfacingPreservesTrackingFields verifies that feedback
// counters survive a RecordSurfacing cycle (#353).
func TestT353_RecordSurfacingPreservesTrackingFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	capture := &writeCapture{}

	tomlWithTracking := baseTOML +
		"surfaced_count = 5\n" +
		"followed_count = 3\n" +
		"contradicted_count = 1\n" +
		"ignored_count = 2\n" +
		"irrelevant_count = 4\n" +
		"last_surfaced_at = \"2026-01-03T00:00:00Z\"\n"

	recorder := track.NewRecorder(
		track.WithReadFile(func(_ string) ([]byte, error) {
			return []byte(tomlWithTracking), nil
		}),
		track.WithCreateTemp(capture.createTemp(t)),
		track.WithRename(func(_, _ string) error { return nil }),
		track.WithRemove(func(_ string) error { return nil }),
	)

	memories := []*memory.Stored{
		{FilePath: "/fake/memory.toml"},
	}

	err := recorder.RecordSurfacing(context.Background(), memories, "tool")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	record := capture.decodeTOML(g)

	g.Expect(record.FollowedCount).To(Equal(3))
	g.Expect(record.ContradictedCount).To(Equal(1))
	g.Expect(record.IgnoredCount).To(Equal(2))
	g.Expect(record.IrrelevantCount).To(Equal(4))
	g.Expect(record.SurfacedCount).To(Equal(5))
	g.Expect(record.LastSurfacedAt).To(Equal("2026-01-03T00:00:00Z"))

	// Also verify raw TOML contains tracking keys.
	data, readErr := os.ReadFile(capture.tmpPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	raw := string(data)
	g.Expect(raw).To(ContainSubstring("followed_count"))
	g.Expect(raw).To(ContainSubstring("contradicted_count"))
	g.Expect(raw).To(ContainSubstring("ignored_count"))
	g.Expect(raw).To(ContainSubstring("irrelevant_count"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `followed_count` is 0 (stripped by current tomlRecord)

- [ ] **Step 3: Update recorder.go AND old tests together**

In `recorder.go`:
- Delete the local `tomlRecord` struct (lines 155-167) and its comment (lines 151-154)
- Add import `"engram/internal/memory"`
- In `updateMemoryFile` (line 76): change `var record tomlRecord` to `var record memory.MemoryRecord`
- In `writeAtomic` (line 86): change parameter from `*tomlRecord` to `*memory.MemoryRecord`

In `recorder_test.go`, update the tests that assert tracking fields are **stripped** — they must now assert tracking fields are **preserved**:

**`TestREQ22AC2_RecordSurfacingPreservesNonTrackingFields`** (line 42):
- Line 113: flip `NotTo(ContainSubstring("surfaced_count"))` to `To(ContainSubstring("surfaced_count"))`
- Lines 114-115: KEEP as-is — `last_surfaced` and `surfacing_contexts` are legacy fields NOT in `MemoryRecord` and should still be stripped
- Add a struct-level assertion:
  ```go
  g.Expect(record.SurfacedCount).To(Equal(7))
  ```

**`TestT77_RecordSurfacingStripsTrackingFields`** (line 164):
- Rename to `TestT77_RecordSurfacingPreservesTrackingFields`
- Change the raw string assertions (lines 205-207) from `NotTo(ContainSubstring(...))` to `To(ContainSubstring(...))`:
  ```go
  g.Expect(raw).To(ContainSubstring("surfaced_count"))
  ```
- Note: the old TOML also had `last_surfaced` and `surfacing_contexts` (legacy fields). These are NOT in `MemoryRecord` and SHOULD still be stripped. So keep:
  ```go
  g.Expect(raw).NotTo(ContainSubstring("surfacing_contexts"))
  ```
  But change:
  ```go
  g.Expect(raw).To(ContainSubstring("surfaced_count"))
  ```

- [ ] **Step 4: Run test to verify all tests pass**

Run: `targ test`
Expected: PASS — all tracking fields preserved, legacy fields still stripped

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: Only `check-uncommitted` fails

- [ ] **Step 6: Commit**

```
git add internal/track/recorder.go internal/track/recorder_test.go
git commit -m "fix(track): RecordSurfacing preserves tracking fields (#353)

RecordSurfacing was stripping feedback counters on every surfacing
event because its local tomlRecord struct lacked tracking fields.
Now uses the canonical memory.MemoryRecord, preserving all data
through read-modify-write cycles.

Legacy fields (surfacing_contexts) are still stripped because they
are not in MemoryRecord.

AI-Used: [claude]"
```

---

## Task 3: Update tomlwriter to use MemoryRecord

**Files:**
- Modify: `internal/tomlwriter/tomlwriter.go`
- Modify: `internal/tomlwriter/tomlwriter_test.go`

- [ ] **Step 1: Write the failing test**

Add a test that verifies new memories include tracking field keys in the raw TOML output (not just that they decode to zero — zero values decode identically whether the key exists or not):

```go
func TestWrite_IncludesTrackingFieldKeys(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	mem := &memory.Enriched{
		Title:           "test memory",
		Content:         "test content",
		FilenameSummary: "tracking-field-test",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	writer := tomlwriter.New()

	path, err := writer.Write(mem, dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	raw := string(data)
	g.Expect(raw).To(ContainSubstring("surfaced_count"), "tracking field key must be present")
	g.Expect(raw).To(ContainSubstring("followed_count"), "tracking field key must be present")
	g.Expect(raw).To(ContainSubstring("contradicted_count"), "tracking field key must be present")
	g.Expect(raw).To(ContainSubstring("ignored_count"), "tracking field key must be present")
	g.Expect(raw).To(ContainSubstring("irrelevant_count"), "tracking field key must be present")
	g.Expect(raw).To(ContainSubstring("last_surfaced_at"), "tracking field key must be present")
}
```

Note: `tomlwriter.Write` takes `*memory.Enriched` (not `*memory.Stored`). `Enriched` has content fields only — no tracking fields. When switching to `MemoryRecord`, tracking fields will be zero-valued in the struct and thus present in the TOML output.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — tracking field keys absent in raw TOML

- [ ] **Step 3: Replace local tomlRecord with memory.MemoryRecord**

In `tomlwriter.go`:
- Delete the local `tomlRecord` struct (lines 187-199)
- Add import `"engram/internal/memory"`
- In `Write()`: change the record construction to use `memory.MemoryRecord`
- Update `writeAtomic` parameter type

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Update existing tests**

Replace any inline struct or `tomlRecord` references in `tomlwriter_test.go` with `memory.MemoryRecord`.

- [ ] **Step 6: Run full checks**

Run: `targ check-full`
Expected: Only `check-uncommitted` fails

- [ ] **Step 7: Commit**

```
git add internal/tomlwriter/tomlwriter.go internal/tomlwriter/tomlwriter_test.go
git commit -m "refactor(tomlwriter): use canonical MemoryRecord (#353)

New memories are written with all tracking field keys present
(zero-valued), ready for feedback and surfacing counters.

AI-Used: [claude]"
```

---

## Task 4: Update retrieve to use MemoryRecord

**Files:**
- Modify: `internal/retrieve/retrieve.go`
- Modify: `internal/retrieve/retrieve_test.go`

**Scope note:** `memory.Stored` is the in-memory domain type and doesn't have all fields that `MemoryRecord` does (e.g., `ObservationType`, `Rationale`, `Confidence`, `CreatedAt`, `LastSurfacedAt`). That's OK — `MemoryRecord` is for TOML I/O fidelity. `parseMemoryFile` decodes into `MemoryRecord` and copies what `Stored` needs. Adding fields to `Stored` is out of scope.

- [ ] **Step 1: Write the failing test**

Add a test that the retrieve path doesn't lose counter fields. The current `tomlRecord` in retrieve.go HAS counter fields, so this test may already pass. If so, this is a pure refactor (no red phase needed) — just verify all existing tests pass after the struct swap.

```go
func TestParseMemoryFile_ReadsCounters(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// TOML with all counter fields
	// ... parse via existing DI patterns ...

	g.Expect(stored.FollowedCount).To(Equal(3))
	g.Expect(stored.ContradictedCount).To(Equal(1))
	g.Expect(stored.IgnoredCount).To(Equal(2))
	g.Expect(stored.IrrelevantCount).To(Equal(4))
	g.Expect(stored.SurfacedCount).To(Equal(5))
}
```

- [ ] **Step 2: Replace local tomlRecord with memory.MemoryRecord**

In `retrieve.go`:
- Delete the local `tomlRecord` (lines 102-115)
- Add import `"engram/internal/memory"`
- In `parseMemoryFile`: change `var record tomlRecord` to `var record memory.MemoryRecord`
- Update the `memory.Stored` construction to copy all available fields

- [ ] **Step 3: Run tests**

Run: `targ test`
Expected: PASS

- [ ] **Step 4: Run full checks**

Run: `targ check-full`
Expected: Only `check-uncommitted` fails

- [ ] **Step 5: Commit**

```
git add internal/retrieve/retrieve.go internal/retrieve/retrieve_test.go
git commit -m "refactor(retrieve): use canonical MemoryRecord (#353)

Retriever now decodes TOML via the canonical struct, ensuring
no fields are silently dropped during parsing.

AI-Used: [claude]"
```

---

## Task 5: Update cli/show.go to use MemoryRecord

**Files:**
- Modify: `internal/cli/show.go`
- Modify: `internal/cli/show_test.go`

- [ ] **Step 1: Read show.go and show_test.go**

Read both files to understand the current `showTOMLRecord` usage and output format.

- [ ] **Step 2: Replace showTOMLRecord with memory.MemoryRecord**

In `show.go`:
- Delete `showTOMLRecord` (lines 23-35)
- Add import `"engram/internal/memory"`
- Use `memory.MemoryRecord` for TOML decoding
- Update output formatting to include any newly-available fields

- [ ] **Step 3: Run tests**

Run: `targ test`
Expected: PASS (output format may need minor test updates)

- [ ] **Step 4: Run full checks**

Run: `targ check-full`
Expected: Only `check-uncommitted` fails

- [ ] **Step 5: Commit**

```
git add internal/cli/show.go internal/cli/show_test.go
git commit -m "refactor(cli): show uses canonical MemoryRecord (#353)

AI-Used: [claude]"
```

---

## Task 6: Update cli/signal.go to use MemoryRecord

**Important context:** `writeRecord` and `readRecord` are **function-scoped type declarations** inside `storedMemoryWriter.Write` and `readStoredMemory` respectively. They are not package-level types. The replacement involves deleting the inline type AND changing the field mapping code.

**Files:**
- Modify: `internal/cli/signal.go`
- Modify: `internal/cli/signal_test.go` (if it exists)

- [ ] **Step 1: Write the failing test**

Add a test that verifies `storedMemoryWriter.Write` preserves all counter fields from `memory.Stored` (not just `SurfacedCount`):

```go
func TestStoredMemoryWriter_PreservesAllCounters(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Create a Stored with all counter fields populated
	stored := &memory.Stored{
		Title:             "test",
		Content:           "content",
		SurfacedCount:     5,
		FollowedCount:     3,
		ContradictedCount: 1,
		IgnoredCount:      2,
		IrrelevantCount:   4,
	}

	// ... write via storedMemoryWriter using DI patterns ...
	// ... read back raw TOML ...

	g.Expect(raw).To(ContainSubstring("followed_count"))
	g.Expect(raw).To(ContainSubstring("contradicted_count"))
	g.Expect(raw).To(ContainSubstring("ignored_count"))
	g.Expect(raw).To(ContainSubstring("irrelevant_count"))
}
```

Follow existing test patterns for how `storedMemoryWriter` is tested. Also test `readStoredMemory` reads all counter fields.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `followed_count` etc. absent in written TOML

- [ ] **Step 3: Replace inline structs with memory.MemoryRecord**

In `storedMemoryWriter.Write` (line 275):
- Delete the `writeRecord` type declaration (lines 276-285)
- Create a `memory.MemoryRecord` and populate from `stored`:
  ```go
  record := memory.MemoryRecord{
      Title:             stored.Title,
      Content:           stored.Content,
      Concepts:          stored.Concepts,
      Keywords:          stored.Keywords,
      AntiPattern:       stored.AntiPattern,
      Principle:         stored.Principle,
      SurfacedCount:     stored.SurfacedCount,
      FollowedCount:     stored.FollowedCount,
      ContradictedCount: stored.ContradictedCount,
      IgnoredCount:      stored.IgnoredCount,
      IrrelevantCount:   stored.IrrelevantCount,
      UpdatedAt:         time.Now().UTC().Format(time.RFC3339),
  }
  ```
  **Known limitation:** `Stored` doesn't carry ObservationType/Rationale/Confidence/CreatedAt/LastSurfacedAt. When `storedMemoryWriter.Write` overwrites an existing file (e.g., signal rewrite actions), those content fields will be zeroed out. This is a pre-existing data-loss path unrelated to #353's feedback counter bug. It's scoped under #354 (instruction registry cleanup) since the signal applier was part of the UC-23 registry design. The correct long-term fix is read-modify-write, but that's out of scope here.

In `readStoredMemory` (line 434):
- Delete the `readRecord` type declaration (lines 435-444)
- Decode into `memory.MemoryRecord` instead
- Update the `memory.Stored` construction to copy all counter fields:
  ```go
  return &memory.Stored{
      Title:             record.Title,
      Content:           record.Content,
      Concepts:          record.Concepts,
      Keywords:          record.Keywords,
      AntiPattern:       record.AntiPattern,
      Principle:         record.Principle,
      SurfacedCount:     record.SurfacedCount,
      FollowedCount:     record.FollowedCount,
      ContradictedCount: record.ContradictedCount,
      IgnoredCount:      record.IgnoredCount,
      IrrelevantCount:   record.IrrelevantCount,
      UpdatedAt:         updatedAt,
      FilePath:          path,
  }, nil
  ```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: Only `check-uncommitted` fails

- [ ] **Step 6: Commit**

```
git add internal/cli/signal.go internal/cli/signal_test.go
git commit -m "refactor(cli): signal uses canonical MemoryRecord (#353)

Signal read/write operations now preserve all counter fields
from memory.Stored instead of silently dropping them.

AI-Used: [claude]"
```

---

## Task 7: Update cli/feedback.go to use MemoryRecord

**Files:**
- Modify: `internal/cli/feedback.go`
- Modify: `internal/cli/feedback_test.go`

**Scope note:** `feedback.go` calls `os.ReadFile` directly (DI violation). Fixing that DI issue is deferred — this task only replaces `map[string]any` with the canonical struct for type safety and consistency.

- [ ] **Step 1: Write the failing test**

Add a test that verifies feedback preserves all content fields through a struct-based round-trip (not just map-based):

```go
func TestFeedback_PreservesAllContentFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Write a TOML with all content + tracking fields to a temp file
	// ... run feedback with --relevant --used ...
	// ... decode result into memory.MemoryRecord ...
	// ... assert content fields unchanged, followed_count incremented ...

	g.Expect(decoded.ObservationType).To(Equal("workflow_instruction"))
	g.Expect(decoded.Rationale).To(Equal("test rationale"))
	g.Expect(decoded.Confidence).To(Equal("A"))
	g.Expect(decoded.FollowedCount).To(Equal(1))
}
```

Follow existing patterns in `feedback_test.go`.

- [ ] **Step 2: Run test to verify behavior**

Run: `targ test`
Expected: May pass (map[string]any preserves all fields). If so, this is a pure refactor.

- [ ] **Step 3: Replace map[string]any with memory.MemoryRecord**

In `feedback.go`:

```go
func readFeedbackTOML(memPath, slug string) (*memory.MemoryRecord, error) {
	data, err := os.ReadFile(memPath) //nolint:gosec // user-provided path at CLI boundary
	if err != nil {
		return nil, fmt.Errorf("feedback: reading %s: %w", slug, err)
	}

	var record memory.MemoryRecord

	_, decErr := toml.Decode(string(data), &record)
	if decErr != nil {
		return nil, fmt.Errorf("feedback: decoding %s: %w", slug, decErr)
	}

	return &record, nil
}

func applyFeedbackCounters(record *memory.MemoryRecord, relevant, used, notused bool) string {
	if !relevant {
		record.IrrelevantCount++
		return "irrelevant"
	}

	if used {
		record.FollowedCount++
		return "relevant, used"
	}

	if notused {
		record.IgnoredCount++
		return "relevant, not used"
	}

	return "relevant"
}

func writeFeedbackTOML(memPath string, record *memory.MemoryRecord, slug string) error {
	var buf bytes.Buffer

	encErr := toml.NewEncoder(&buf).Encode(record)
	if encErr != nil {
		return fmt.Errorf("feedback: encoding %s: %w", slug, encErr)
	}

	dir := filepath.Dir(memPath)
	tmpPath := filepath.Join(dir, ".tmp-feedback")

	const filePerm = 0o644

	writeErr := os.WriteFile(tmpPath, buf.Bytes(), filePerm)
	if writeErr != nil {
		return fmt.Errorf("feedback: writing temp: %w", writeErr)
	}

	renameErr := os.Rename(tmpPath, memPath)
	if renameErr != nil {
		return fmt.Errorf("feedback: renaming temp: %w", renameErr)
	}

	return nil
}
```

Update `runFeedback` to use the new signatures.

- [ ] **Step 4: Update feedback_test.go**

Replace map-based assertions with struct-based assertions. The `applyFeedbackCounters` tests change from:
```go
record := map[string]any{"followed_count": int64(0)}
applyFeedbackCounters(record, true, true, false)
// assert record["followed_count"] == int64(1)
```
to:
```go
record := &memory.MemoryRecord{FollowedCount: 0}
applyFeedbackCounters(record, true, true, false)
g.Expect(record.FollowedCount).To(Equal(1))
```

- [ ] **Step 5: Run tests**

Run: `targ test`
Expected: PASS

- [ ] **Step 6: Run full checks**

Run: `targ check-full`
Expected: Only `check-uncommitted` fails

- [ ] **Step 7: Commit**

```
git add internal/cli/feedback.go internal/cli/feedback_test.go
git commit -m "refactor(cli): feedback uses canonical MemoryRecord (#353)

Type-safe counter updates instead of map[string]any.
Consistent with all other TOML read/write paths.

AI-Used: [claude]"
```
