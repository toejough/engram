# Keyword Normalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Normalize keywords to a canonical form (lowercase, underscores) at every write site and comparison site so that `prefixed-ids` and `prefixed_IDs` are treated as the same keyword.

**Architecture:** Add `keyword.Normalize(kw string) string` (lowercase + hyphens→underscores) and apply it at all write paths (`unionKeywords`, `UpdateMerged`, `applyBroaden`, `applyKeywordsField`) and strengthen the comparison path in `keywordSet`. Also add always-on logging to `Consolidate()` so it's visible whether it ran.

**Tech Stack:** Go, gomega, `targ test` / `targ check-full`

---

## File Map

| File | Change |
|------|--------|
| `internal/keyword/normalize.go` | **Create** — `Normalize()` and `NormalizeAll()` |
| `internal/keyword/normalize_test.go` | **Create** — tests for the above |
| `internal/signal/consolidate.go` | **Modify** — strengthen `keywordSet()` + `countNewKeywords()` + add always-on logging to `Consolidate()` |
| `internal/signal/consolidate_test.go` | **Modify** — add test for mixed-format keyword clustering |
| `internal/learn/learn.go` | **Modify** — normalize in `unionKeywords()` so merged keywords are canonical |
| `internal/learn/learn_test.go` | **Modify** — add test for mixed-format keyword dedup in union |
| `internal/signal/apply.go` | **Modify** — normalize in `applyBroaden()` and `applyKeywordsField()` |
| `internal/signal/apply_test.go` | **Modify** — add tests for broaden and rewrite keyword normalization |

---

### Task 1: Add `keyword.Normalize` utility

**Files:**
- Create: `internal/keyword/normalize.go`
- Create: `internal/keyword/normalize_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/keyword/normalize_test.go
package keyword_test

import (
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/keyword"
)

func TestNormalize_LowercasesAndReplacesHyphens(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	cases := []struct {
		input    string
		expected string
	}{
		{"prefixed-ids", "prefixed_ids"},
		{"prefixed_IDs", "prefixed_ids"},
		{"collision-avoidance", "collision_avoidance"},
		{"collision_avoidance", "collision_avoidance"},
		{"already_normalized", "already_normalized"},
		{"UPPER_CASE", "upper_case"},
		{"mixed-Case_Thing", "mixed_case_thing"},
		{"", ""},
	}

	for _, tc := range cases {
		g.Expect(keyword.Normalize(tc.input)).To(gomega.Equal(tc.expected), "input: %q", tc.input)
	}
}

func TestNormalizeAll_NormalizesSlice(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	input := []string{"prefixed-ids", "prefixed_IDs", "collision-avoidance"}
	result := keyword.NormalizeAll(input)

	g.Expect(result).To(gomega.Equal([]string{"prefixed_ids", "prefixed_ids", "collision_avoidance"}))
}

func TestNormalizeAll_NilInputReturnsNil(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	g.Expect(keyword.NormalizeAll(nil)).To(gomega.BeNil())
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
targ test 2>&1 | grep -E "normalize|FAIL|undefined"
```
Expected: compile error — `keyword.Normalize` and `keyword.NormalizeAll` undefined.

- [ ] **Step 3: Implement `keyword.Normalize` and `keyword.NormalizeAll`**

```go
// internal/keyword/normalize.go
package keyword

import "strings"

// Normalize canonicalizes a keyword: lowercase, hyphens replaced with underscores.
// This ensures "prefixed-ids", "prefixed_IDs", and "prefixed_ids" all map to "prefixed_ids".
func Normalize(kw string) string {
	return strings.ReplaceAll(strings.ToLower(kw), "-", "_")
}

// NormalizeAll normalizes a slice of keywords, returning a new slice.
// Returns nil if input is nil.
func NormalizeAll(kws []string) []string {
	if kws == nil {
		return nil
	}

	result := make([]string, len(kws))

	for i, kw := range kws {
		result[i] = Normalize(kw)
	}

	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
targ test 2>&1 | grep -E "keyword|FAIL"
```
Expected: `ok  engram/internal/keyword`

- [ ] **Step 5: Commit**

```bash
git add internal/keyword/normalize.go internal/keyword/normalize_test.go
git commit -m "feat(keyword): add Normalize and NormalizeAll for canonical keyword form"
```

---

### Task 2: Strengthen consolidate.go comparison + add always-on logging

**Files:**
- Modify: `internal/signal/consolidate.go` (lines 514–557 for helpers, line 51–92 for Consolidate)
- Modify: `internal/signal/consolidate_test.go`

The `keywordSet()` and `countNewKeywords()` functions currently lowercase keywords but don't unify hyphens/underscores. Fix both. Also, `Consolidate()` only logs when it finds clusters — change it to always log scan count + cluster count.

- [ ] **Step 1: Write the failing test for mixed-format clustering**

Find the existing consolidate tests in `internal/signal/consolidate_test.go`. They use a `fakeLister` that returns `[]*memory.Stored`. Add a new test at the end:

```go
func TestConsolidate_DetectsMixedFormatKeywordDuplicates(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// These two memories are duplicates but have 0% overlap under exact matching
	// because of hyphens vs underscores.
	memA := &memory.Stored{
		Title:    "Memory A",
		FilePath: "/data/memories/mem-a.toml",
		Keywords: []string{"prefixed-ids", "collision-avoidance", "parallel-agents"},
	}
	memB := &memory.Stored{
		Title:    "Memory B",
		FilePath: "/data/memories/mem-b.toml",
		Keywords: []string{"prefixed_ids", "collision_avoidance", "parallel_agents"},
	}

	var mergedSurvivor string
	var deletedAbsorbed string

	consolidator := signal.NewConsolidator(
		signal.WithLister(&fakeLister{memories: []*memory.Stored{memA, memB}}),
		signal.WithMerger(&fakeMerger{onMerge: func(survivor, absorbed *memory.Stored, _, _ string) error {
			mergedSurvivor = survivor.FilePath
			deletedAbsorbed = absorbed.FilePath
			return nil
		}}),
		signal.WithFileWriter(&fakeFileWriter{}),
		signal.WithFileDeleter(func(_ string) error { return nil }),
		signal.WithBackupWriter(&fakeBackupWriter{}, "/tmp/backup"),
		signal.WithEntryRemover(func(_ string) error { return nil }),
		signal.WithEffectiveness(&fakeEffectiveness{}),
		signal.WithStderr(io.Discard),
		signal.WithTextSimilarityScorer(&fakeTextSimilarityScorer{}),
	)

	result, err := consolidator.Consolidate(context.Background())
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.MemoriesMerged).To(gomega.Equal(1))
	g.Expect(mergedSurvivor).NotTo(gomega.BeEmpty())
	g.Expect(deletedAbsorbed).NotTo(gomega.BeEmpty())
}
```

Stub types used above: `fakeLister`, `fakeMerger`, `fakeFileWriter`, `fakeBackupWriter`, `fakeEffectiveness`, `fakeTextSimilarityScorer` — all defined in `consolidate_test.go`. Do not invent new stubs.

- [ ] **Step 2: Run test to verify it fails**

```bash
targ test 2>&1 | grep -E "TestConsolidate_DetectsMixed|FAIL"
```
Expected: FAIL — `result.MemoriesMerged` is 0 because hyphen/underscore mismatch prevents detection.

- [ ] **Step 3: Fix `keywordSet()` to normalize separators**

In `internal/signal/consolidate.go`, change `keywordSet()`:

```go
// Before:
func keywordSet(keywords []string) map[string]struct{} {
	set := make(map[string]struct{}, len(keywords))
	for _, keyword := range keywords {
		set[strings.ToLower(keyword)] = struct{}{}
	}
	return set
}

// After:
func keywordSet(keywords []string) map[string]struct{} {
	set := make(map[string]struct{}, len(keywords))
	for _, kw := range keywords {
		set[keyword.Normalize(kw)] = struct{}{}
	}
	return set
}
```

Also fix `countNewKeywords()` to use `keyword.Normalize` instead of `strings.ToLower`:

```go
// Before:
func countNewKeywords(survivorKW, absorbedKW []string) int {
	existing := keywordSet(survivorKW)
	count := 0
	for _, keyword := range absorbedKW {
		if _, ok := existing[strings.ToLower(keyword)]; !ok {
			count++
		}
	}
	return count
}

// After:
func countNewKeywords(survivorKW, absorbedKW []string) int {
	existing := keywordSet(survivorKW)
	count := 0
	for _, kw := range absorbedKW {
		if _, ok := existing[keyword.Normalize(kw)]; !ok {
			count++
		}
	}
	return count
}
```

Add `"engram/internal/keyword"` to the file's import block. Do NOT remove `"strings"` — it is still used by `strings.Join` in `clusterConfidence()` elsewhere in the file.

- [ ] **Step 4: Add always-on logging to `Consolidate()`**

In `internal/signal/consolidate.go`, find the `Consolidate()` function. It currently logs only when clusters are found (around line 86). Add a log line that always fires after building clusters, before iterating:

```go
func (c *Consolidator) Consolidate(ctx context.Context) (ConsolidateResult, error) {
	// ... existing code: return early if no lister ...
	memories, err := c.lister.ListAll(ctx)
	// ... existing error handling ...

	clusters := buildClusters(memories)

	// Always log so operators can tell consolidation ran
	c.logStderrf("[engram] Consolidation: scanned %d memories, found %d candidate clusters\n",
		len(memories), len(clusters))

	// ... rest of existing loop ...
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
targ test 2>&1 | grep -E "signal|FAIL"
```
Expected: `ok  engram/internal/signal`

- [ ] **Step 6: Commit**

```bash
git add internal/signal/consolidate.go internal/signal/consolidate_test.go
git commit -m "fix(signal): normalize keyword separators in consolidator + add always-on logging"
```

---

### Task 3: Normalize in `learn.go` `unionKeywords()`

**Files:**
- Modify: `internal/learn/learn.go` (line 293–312)
- Modify: `internal/learn/learn_test.go`

`unionKeywords` currently uses `set[k] = struct{}{}` without normalization. Two keywords that are the same after normalization (e.g. `"prefixed-ids"` and `"prefixed_ids"`) would both be included. Fix by normalizing the key used in the set.

- [ ] **Step 1: Write the failing test**

Find `internal/learn/learn_test.go`. Add a test that verifies `unionKeywords` deduplicates mixed-format keywords. Since `unionKeywords` is unexported, look for an existing export in `internal/learn/export_test.go` — if none exists, create one:

```go
// internal/learn/export_test.go (create if doesn't exist, or add to existing)
package learn

// ExportUnionKeywords exposes unionKeywords for testing.
func ExportUnionKeywords(l *Learner, a, b []string) []string {
	return l.unionKeywords(a, b)
}
```

Then in `internal/learn/learn_test.go`:

```go
func TestUnionKeywords_DeduplicatesMixedFormat(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// "prefixed-ids" and "prefixed_ids" should unify to one normalized keyword.
	// learn.New signature: New(extractor, retriever, deduplicator, writer, dataDir)
	learner := learn.New(nil, nil, nil, nil, "")
	result := learn.ExportUnionKeywords(learner,
		[]string{"prefixed-ids", "collision-avoidance"},
		[]string{"prefixed_ids", "collision_avoidance", "new_keyword"},
	)

	g.Expect(result).To(gomega.ConsistOf("prefixed_ids", "collision_avoidance", "new_keyword"))
}
```

Check `internal/learn/export_test.go` to see what's already exported before adding the new export.

- [ ] **Step 2: Run test to verify it fails**

```bash
targ test 2>&1 | grep -E "TestUnionKeywords|learn|FAIL"
```
Expected: FAIL — result would have 5 elements (duplicates not removed).

- [ ] **Step 3: Fix `unionKeywords()` to normalize**

In `internal/learn/learn.go`, change `unionKeywords`:

```go
// Before:
func (l *Learner) unionKeywords(a, b []string) []string {
	set := make(map[string]struct{})
	for _, k := range a {
		set[k] = struct{}{}
	}
	for _, k := range b {
		set[k] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
	}
	return result
}

// After:
func (l *Learner) unionKeywords(a, b []string) []string {
	set := make(map[string]struct{})
	for _, k := range a {
		set[keyword.Normalize(k)] = struct{}{}
	}
	for _, k := range b {
		set[keyword.Normalize(k)] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
	}
	return result
}
```

The `keyword` package is already imported (`engram/internal/keyword` is already in learn.go's imports).

- [ ] **Step 4: Run tests to verify they pass**

```bash
targ test 2>&1 | grep -E "learn|FAIL"
```
Expected: `ok  engram/internal/learn`

- [ ] **Step 5: Commit**

```bash
git add internal/learn/learn.go internal/learn/learn_test.go internal/learn/export_test.go
git commit -m "fix(learn): normalize keywords in unionKeywords to deduplicate mixed formats"
```

---

### Task 4: Normalize in `mergewriter.go` `UpdateMerged()`

**Files:**
- Modify: `internal/learn/mergewriter.go` (line 48–95)
- Modify: `internal/learn/mergewriter_test.go` (check what exists)

`UpdateMerged()` receives `keywords []string` and writes them directly to TOML without normalization. Fix by normalizing the slice before writing.

- [ ] **Step 1: Write the failing test**

Find `internal/learn/mergewriter_test.go`. Add:

```go
func TestTOMLMergeWriter_NormalizesKeywords(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	tmpFile, err := os.CreateTemp(t.TempDir(), "memory-*.toml")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil { return }
	_ = tmpFile.Close()

	// Write initial content so ReadFile succeeds
	err = os.WriteFile(tmpFile.Name(), []byte(`title = "test"\n`), 0o600)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil { return }

	stored := &memory.Stored{FilePath: tmpFile.Name()}
	writer := &learn.ExportTOMLMergeWriter{}  // see export_test.go

	err = writer.UpdateMerged(stored, "principle", []string{"Mixed-Case", "hyphen-sep"}, nil, time.Now())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil { return }

	data, readErr := os.ReadFile(tmpFile.Name())
	g.Expect(readErr).NotTo(gomega.HaveOccurred())
	if readErr != nil { return }

	content := string(data)
	g.Expect(content).To(gomega.ContainSubstring(`"mixed_case"`))
	g.Expect(content).To(gomega.ContainSubstring(`"hyphen_sep"`))
	g.Expect(content).NotTo(gomega.ContainSubstring(`"Mixed-Case"`))
	g.Expect(content).NotTo(gomega.ContainSubstring(`"hyphen-sep"`))
}
```

Note: check `internal/learn/export_test.go` for how `TOMLMergeWriter` is exported. Add an export if needed:
```go
// in export_test.go
type ExportTOMLMergeWriter = TOMLMergeWriter
```

- [ ] **Step 2: Run test to verify it fails**

```bash
targ test 2>&1 | grep -E "TestTOMLMerge|learn|FAIL"
```
Expected: FAIL — keywords written as `"Mixed-Case"`, `"hyphen-sep"` without normalization.

- [ ] **Step 3: Fix `UpdateMerged()` to normalize keywords**

In `internal/learn/mergewriter.go`, replace the full `TOMLMergeWriter.UpdateMerged` function with this:

```go
func (w *TOMLMergeWriter) UpdateMerged(
	existing *memory.Stored,
	principle string,
	keywords, concepts []string,
	now time.Time,
) error {
	// Read the existing TOML file (just to verify it exists)
	_, err := os.ReadFile(existing.FilePath)
	if err != nil {
		return fmt.Errorf("reading existing memory: %w", err)
	}

	normalizedKeywords := keyword.NormalizeAll(keywords)

	// Build new TOML content
	var content strings.Builder

	fmt.Fprintf(&content, "principle = %q\n", principle)
	fmt.Fprintf(&content, "updated_at = %q\n", now.Format(time.RFC3339))

	content.WriteString("keywords = [")

	for i, k := range normalizedKeywords {
		if i > 0 {
			content.WriteString(", ")
		}

		fmt.Fprintf(&content, "%q", k)
	}

	content.WriteString("]\n")
	content.WriteString("concepts = [")

	for i, c := range concepts {
		if i > 0 {
			content.WriteString(", ")
		}

		fmt.Fprintf(&content, "%q", c)
	}

	content.WriteString("]\n")

	writeErr := os.WriteFile(existing.FilePath, []byte(content.String()), mergedFileMode)
	if writeErr != nil {
		return fmt.Errorf("writing merged memory: %w", writeErr)
	}

	return nil
}
```

Add `"engram/internal/keyword"` to the import block.

- [ ] **Step 4: Run tests to verify they pass**

```bash
targ test 2>&1 | grep -E "learn|FAIL"
```
Expected: `ok  engram/internal/learn`

- [ ] **Step 5: Commit**

```bash
git add internal/learn/mergewriter.go internal/learn/mergewriter_test.go internal/learn/export_test.go
git commit -m "fix(learn): normalize keywords in TOMLMergeWriter.UpdateMerged"
```

---

### Task 5: Normalize in `signal/apply.go` — `applyBroaden()` and `applyKeywordsField()`

**Files:**
- Modify: `internal/signal/apply.go` (line 73–91 for `applyBroaden`, line 223–243 for `applyKeywordsField`)
- Modify: `internal/signal/apply_test.go`

Two write paths in `apply.go` write keywords without normalizing:
- `applyBroaden()` — appends `action.Keywords` directly
- `applyKeywordsField()` — called by `applyRewrite` when the `keywords` field is included in a rewrite action; sets `stored.Keywords` directly from the decoded JSON slice

Both need normalization.

- [ ] **Step 1: Write the failing test**

In `internal/signal/apply_test.go`, find `TestApply_Broaden`. Add a new test:

```go
func TestApply_BroadenNormalizesKeywords(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	writer := &stubMemoryWriter{written: make(map[string]*memory.Stored)}

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return &memory.Stored{Keywords: []string{"existing_kw"}}, nil
		}),
		signal.WithWriteMemory(writer),
	)

	_, err := applier.Apply(context.Background(), signal.ApplyAction{
		Action:   "broaden_keywords",
		Memory:   "memories/gem.toml",
		Keywords: []string{"Mixed-Case", "hyphen-sep"},
	})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil { return }

	stored := writer.written["memories/gem.toml"]
	g.Expect(stored).NotTo(gomega.BeNil())
	if stored == nil { return }

	g.Expect(stored.Keywords).To(gomega.ConsistOf("existing_kw", "mixed_case", "hyphen_sep"))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
targ test 2>&1 | grep -E "TestApply_BroadenNorm|signal|FAIL"
```
Expected: FAIL — keywords are `"Mixed-Case"` and `"hyphen-sep"` instead of normalized.

- [ ] **Step 3: Fix `applyBroaden()` and `applyKeywordsField()` to normalize**

In `internal/signal/apply.go`, two changes:

**`applyBroaden()` (line ~88):**
```go
// Before:
stored.Keywords = append(stored.Keywords, action.Keywords...)

// After:
stored.Keywords = append(stored.Keywords, keyword.NormalizeAll(action.Keywords)...)
```

**`applyKeywordsField()` (line 223–243) — full replacement:**
```go
func applyKeywordsField(stored *memory.Stored, fields map[string]any) {
	kw, ok := fields["keywords"]
	if !ok {
		return
	}

	slice, isSlice := kw.([]any)
	if !isSlice {
		return
	}

	keywords := make([]string, 0, len(slice))

	for _, item := range slice {
		if strItem, isStr := item.(string); isStr {
			keywords = append(keywords, keyword.Normalize(strItem))
		}
	}

	stored.Keywords = keywords
}
```

Also add a test for rewrite normalization in `apply_test.go`:

```go
func TestApply_RewriteNormalizesKeywords(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	writer := &stubMemoryWriter{written: make(map[string]*memory.Stored)}

	applier := signal.NewApplier(
		signal.WithReadMemory(func(_ string) (*memory.Stored, error) {
			return &memory.Stored{Title: "Old"}, nil
		}),
		signal.WithWriteMemory(writer),
	)

	_, err := applier.Apply(context.Background(), signal.ApplyAction{
		Action: "rewrite",
		Memory: "memories/leech.toml",
		Fields: map[string]any{
			"keywords": []any{"Mixed-Case", "hyphen-sep"},
		},
	})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil { return }

	stored := writer.written["memories/leech.toml"]
	g.Expect(stored).NotTo(gomega.BeNil())
	if stored == nil { return }

	g.Expect(stored.Keywords).To(gomega.ConsistOf("mixed_case", "hyphen_sep"))
}
```

Add `"engram/internal/keyword"` to the import block.

- [ ] **Step 4: Run tests to verify they pass**

```bash
targ test 2>&1 | grep -E "signal|FAIL"
```
Expected: `ok  engram/internal/signal`

- [ ] **Step 5: Commit**

```bash
git add internal/signal/apply.go internal/signal/apply_test.go
git commit -m "fix(signal): normalize keywords in applyBroaden before appending"
```

---

### Task 6: Final check and close

- [ ] **Step 1: Run full quality check**

```bash
targ check-full 2>&1
```
Expected: only `check-uncommitted` fails (pre-commit hook). All linters, nil checks, coverage pass.

- [ ] **Step 2: Fix any issues found**

If `reorder-decls-check` fails: run `targ reorder-decls`
If lint fails: read the full output and fix before proceeding.
If coverage floor drops: add tests for uncovered paths.

- [ ] **Step 3: Close issue #349**

```bash
gh issue close 349 --comment "Implemented keyword normalization (lowercase + hyphens→underscores) at all write sites (unionKeywords, UpdateMerged, applyBroaden) and strengthened comparison-time normalization in keywordSet(). Added always-on Consolidate() logging. Existing memories with mixed formats will be detected as duplicates on next consolidation run."
```
