# Remaining Simplifications (#412-#423) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 12 code quality, efficiency, and reuse issues identified during codebase review.

**Architecture:** Grouped by dependency chains. Execute groups sequentially; within a group, tasks are sequential.

**Tech Stack:** Go

---

## Group A: JSONL + creationlog (#412, #414)

### Task A1: Extract generic JSONL parser (#412)

**Files:**
- Create: `internal/jsonlutil/jsonlutil.go`
- Create: `internal/jsonlutil/jsonlutil_test.go`
- Modify: `internal/surfacinglog/logger.go` — use shared parser
- Modify: `internal/creationlog/creationlog.go` — use shared parser, fix path concatenation to filepath.Join

- [ ] **Step 1: Write failing tests for generic JSONL parser**

```go
package jsonlutil_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/jsonlutil"
)

func TestParseLines_ValidJSON(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	type item struct {
		Name string `json:"name"`
	}

	data := []byte("{\"name\":\"a\"}\n{\"name\":\"b\"}\n")
	result := jsonlutil.ParseLines[item](data)
	g.Expect(result).To(HaveLen(2))
	g.Expect(result[0].Name).To(Equal("a"))
	g.Expect(result[1].Name).To(Equal("b"))
}

func TestParseLines_SkipsEmptyAndMalformed(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	type item struct {
		Name string `json:"name"`
	}

	data := []byte("{\"name\":\"a\"}\n\nbad json\n{\"name\":\"b\"}\n")
	result := jsonlutil.ParseLines[item](data)
	g.Expect(result).To(HaveLen(2))
}

func TestParseLines_EmptyInput(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	type item struct{}

	result := jsonlutil.ParseLines[item](nil)
	g.Expect(result).To(BeEmpty())
}
```

- [ ] **Step 2: Run tests — expect FAIL**
- [ ] **Step 3: Implement**

```go
// Package jsonlutil provides generic JSONL parsing.
package jsonlutil

import (
	"encoding/json"
	"strings"
)

// ParseLines parses JSONL data into a slice of T, skipping empty/malformed lines.
func ParseLines[T any](data []byte) []T {
	if len(data) == 0 {
		return nil
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	result := make([]T, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		var entry T
		if err := json.Unmarshal([]byte(trimmed), &entry); err != nil {
			continue
		}

		result = append(result, entry)
	}

	return result
}
```

- [ ] **Step 4: Run tests — expect PASS**
- [ ] **Step 5: Migrate surfacinglog.ReadAndClear and creationlog.ReadAndClear to use jsonlutil.ParseLines**

Replace the inline split/trim/unmarshal loops with `jsonlutil.ParseLines[SurfacingEvent](data)` and `jsonlutil.ParseLines[LogEntry](data)`.

Also fix path construction: replace `l.dataDir + "/" + logFilename` with `filepath.Join(l.dataDir, logFilename)` in both packages.

- [ ] **Step 6: Run tests — expect PASS**
- [ ] **Step 7: Commit**

### Task A2: Fix creationlog.Append to use O_APPEND (#414)

**Files:**
- Modify: `internal/creationlog/creationlog.go` — replace read-then-rewrite with O_APPEND

- [ ] **Step 1: Read existing Append method and tests**
- [ ] **Step 2: Replace Append logic**

Replace the read-entire-file → rebuild → write-atomic pattern with:
```go
func (w *LogWriter) Append(entry LogEntry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = w.now()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling log entry: %w", err)
	}

	path := filepath.Join(w.dataDir, logFilename)

	f, err := w.openFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing log entry: %w", err)
	}

	return nil
}
```

Note: The DI pattern needs an `openFile` function field on `LogWriter`. Add it with a default of `os.OpenFile`.

- [ ] **Step 3: Run tests — expect PASS**
- [ ] **Step 4: Commit**

---

## Group B: SearchText + memoization (#413, #415, #416, #423)

### Task B1: Add SearchText method to memory.Stored (#413)

**Files:**
- Modify: `internal/memory/memory.go` (or wherever Stored is defined)
- Create test in appropriate test file
- Modify: `internal/contradict/contradict.go` — use SearchText
- Modify: `internal/surface/surface.go` — use SearchText

- [ ] **Step 1: Write failing test for SearchText**

```go
func TestStored_SearchText(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		Title:     "Test Title",
		Content:   "some content",
		Principle: "a principle",
		Keywords:  []string{"kw1", "kw2"},
		Concepts:  []string{"concept1"},
	}

	text := mem.SearchText()
	g.Expect(text).To(ContainSubstring("Test Title"))
	g.Expect(text).To(ContainSubstring("some content"))
	g.Expect(text).To(ContainSubstring("a principle"))
	g.Expect(text).To(ContainSubstring("kw1"))
	g.Expect(text).To(ContainSubstring("concept1"))
}
```

- [ ] **Step 2: Implement SearchText on Stored**

```go
// SearchText returns a concatenation of all searchable fields for retrieval scoring.
func (s *Stored) SearchText() string {
	var parts []string
	if s.Title != "" {
		parts = append(parts, s.Title)
	}
	if s.Content != "" {
		parts = append(parts, s.Content)
	}
	if s.Principle != "" {
		parts = append(parts, s.Principle)
	}
	for _, kw := range s.Keywords {
		parts = append(parts, kw)
	}
	for _, c := range s.Concepts {
		parts = append(parts, c)
	}
	return strings.Join(parts, " ")
}
```

- [ ] **Step 3: Migrate contradict.memText to use mem.SearchText()**

Replace `memText(a)` calls with `a.SearchText()`. Remove the `memText` function.

- [ ] **Step 4: Migrate surface.concatenatePromptFields to use mem.SearchText()**

Replace `concatenatePromptFields(mem)` calls with `mem.SearchText()`. Remove the function.

- [ ] **Step 5: Run tests — expect PASS**
- [ ] **Step 6: Commit**

### Task B2: Memoize memText in contradict (#415)

**After B1, memText is replaced by SearchText. The memoization is now about caching SearchText results.**

**Files:**
- Modify: `internal/contradict/contradict.go`

- [ ] **Step 1: Pre-compute SearchText for all candidates before the pair loop**

```go
texts := make([]string, len(candidates))
for i, c := range candidates {
	texts[i] = strings.ToLower(c.SearchText())
}
```

Use `texts[i]` and `texts[j]` in the pair iterations instead of calling SearchText per pair.

- [ ] **Step 2: Run tests — expect PASS**
- [ ] **Step 3: Commit**

### Task B3: Memoize concatenatePromptFields in surface (#416)

**After B1, concatenatePromptFields is replaced by SearchText. Store the result in promptMatch.**

**Files:**
- Modify: `internal/surface/surface.go` — store SearchText in promptMatch
- Modify: `internal/surface/budget.go` — accept pre-computed text

- [ ] **Step 1: Add `searchText` field to `promptMatch` struct**
- [ ] **Step 2: Populate it during BM25 document building**
- [ ] **Step 3: Use it in EstimateMemoryTokens instead of re-computing**
- [ ] **Step 4: Run tests — expect PASS**
- [ ] **Step 5: Commit**

### Task B4: Call time.Now() once before suppression loops (#423)

**Files:**
- Modify: `internal/surface/suppress_p4f.go`

- [ ] **Step 1: In suppressByCrossRef, capture `now := time.Now()` before the loop, use in Timestamp field**
- [ ] **Step 2: Same for suppressByTranscript**
- [ ] **Step 3: Run tests — expect PASS**
- [ ] **Step 4: Commit**

---

## Group C: Parallel recall (#417)

### Task C1: Parallelize LLM calls in recallModeB

**Files:**
- Modify: `internal/recall/orchestrate.go`

- [ ] **Step 1: Replace sequential loop with errgroup**

```go
import "golang.org/x/sync/errgroup"

// In recallModeB:
type extractResult struct {
	index int
	text  string
}

results := make([]extractResult, 0, len(sessions))
var mu sync.Mutex

eg, egCtx := errgroup.WithContext(ctx)
eg.SetLimit(3) // bound concurrent LLM calls

for i, path := range sessions {
	eg.Go(func() error {
		content, _, readErr := o.reader.Read(path, DefaultStripBudget)
		if readErr != nil { return nil } // skip, don't fail

		extracted, extErr := o.summarizer.ExtractRelevant(egCtx, content, query)
		if extErr != nil { return nil }

		mu.Lock()
		results = append(results, extractResult{index: i, text: extracted})
		mu.Unlock()
		return nil
	})
}
_ = eg.Wait()

// Sort by index to preserve order, concatenate up to cap
sort.Slice(results, func(a, b int) bool { return results[a].index < results[b].index })
```

- [ ] **Step 2: Check if errgroup is already a dependency, add if not**
- [ ] **Step 3: Run tests — expect PASS**
- [ ] **Step 4: Commit**

---

## Group D: instruct cleanups (#419, #420, #422)

### Task D1: Remove dead projectDir param from ScanAll (#419)

**Files:**
- Modify: `internal/instruct/scanner.go` — remove second parameter
- Modify: `internal/instruct/audit.go` — update caller
- Modify: `internal/instruct/scanner_test.go` — update test callers

- [ ] **Step 1: Remove `_ string` from ScanAll signature**
- [ ] **Step 2: Update all callers (audit.go line 31, test file lines 48/102/153/182)**
- [ ] **Step 3: Run tests — expect PASS**
- [ ] **Step 4: Commit**

### Task D2: Add named constant for "contradicted" (#420)

**Files:**
- Modify: `internal/instruct/audit.go` — add constant, replace raw string

- [ ] **Step 1: Add constant at package level**

```go
const outcomeContradicted = "contradicted"
```

- [ ] **Step 2: Replace `"contradicted"` on line ~152 with `outcomeContradicted`**
- [ ] **Step 3: Run tests — expect PASS**
- [ ] **Step 4: Commit**

### Task D3: Remove redundant copy in diagnoseBottom (#422)

**Files:**
- Modify: `internal/instruct/audit.go`

- [ ] **Step 1: Replace scored copy with slices.Clone or direct sort**

The copy exists to avoid mutating the caller's slice (needed for sort). Replace:
```go
scored := make([]InstructionItem, 0, len(items))
scored = append(scored, items...)
```
With:
```go
scored := slices.Clone(items)
```

Fix or remove the misleading "Filter to items with effectiveness data" comment.

- [ ] **Step 2: Run tests — expect PASS**
- [ ] **Step 3: Commit**

---

## Group E: extractBullets regex (#421)

### Task E1: Remove redundant regex match

**Files:**
- Modify: `internal/crossref/extract.go`

- [ ] **Step 1: Replace the MatchString + ReplaceAllString pattern**

Change:
```go
if !bulletPrefix.MatchString(line) {
    continue
}
text := bulletPrefix.ReplaceAllString(line, "")
```

To:
```go
text := bulletPrefix.ReplaceAllString(line, "")
if text == line {
    continue
}
```

- [ ] **Step 2: Run tests — expect PASS**
- [ ] **Step 3: Commit**

---

## Group F: Dead params in track.Recorder (#418)

**Note: #410 may have already modified recorder.go. Check the current state first.**

### Task F1: Remove dead parameters from updateMemoryFile

**Files:**
- Modify: `internal/track/recorder.go`

- [ ] **Step 1: Check if updateMemoryFile still has dead params after #410 changes**
- [ ] **Step 2: Remove `_ string` and `_ time.Time` from signature**
- [ ] **Step 3: Update caller RecordSurfacing to not pass mode and now**
- [ ] **Step 4: Run tests — expect PASS**
- [ ] **Step 5: Commit**

---

## Final Verification

- [ ] Run `targ check-full` after all groups complete
- [ ] Push all commits
