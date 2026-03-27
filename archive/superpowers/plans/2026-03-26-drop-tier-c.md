# Drop Tier C Memories — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop creating tier C memories and delete all existing ones. Playground experiment showed tier C contributes zero value to surfacing.

**Architecture:** Add a tier filter in `learn.go` (alongside the existing generalizability filter) that drops tier C candidates before writing. Delete existing tier C memory files. Update tests.

**Tech Stack:** Go, gomega test framework, targ build system

---

## Task 1: Filter tier C candidates in the learn pipeline

**Files:**
- Modify: `internal/learn/learn.go`
- Modify: `internal/learn/learn_test.go`

- [ ] **Step 1: Write failing test**

In `learn_test.go`, add a test that verifies tier C candidates are filtered out and not written:

```go
// TestTierCCandidatesDropped verifies tier C candidates are silently
// dropped and not written to disk (#395).
func TestTierCCandidatesDropped(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []memory.CandidateLearning{
		{Title: "tier A rule", Tier: "A", Generalizability: 3, Keywords: []string{"test"}},
		{Title: "tier B correction", Tier: "B", Generalizability: 3, Keywords: []string{"test"}},
		{Title: "tier C context", Tier: "C", Generalizability: 3, Keywords: []string{"test"}},
	}

	writer := &fakeWriter{}
	extractor := &fakeExtractor{candidates: candidates}
	retriever := &fakeRetriever{}
	dedup := &fakeDeduplicator{passThrough: true}

	learner := learn.New(extractor, writer, retriever, dedup, "/tmp/test-data")

	result, err := learner.Run(context.Background(), "test transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Only 2 memories written (tier A and B), tier C dropped
	g.Expect(writer.received).To(HaveLen(2))
	g.Expect(result.Created).To(HaveLen(2))

	for _, mem := range writer.received {
		g.Expect(mem.Confidence).NotTo(Equal("C"))
	}
}
```

Adapt this to the existing test helper patterns in learn_test.go (fakeWriter, fakeExtractor, etc. already exist — reuse them).

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — tier C candidate is currently written.

- [ ] **Step 3: Add tier C filter in learn.go**

Add a new filter function after `filterByGeneralizability` and before deduplication (line 109 area):

```go
// filterTierC drops tier C candidates — contextual facts that don't
// generalize and add noise to surfacing (#395).
func (l *Learner) filterTierC(
	candidates []memory.CandidateLearning,
) ([]memory.CandidateLearning, int) {
	filtered := make([]memory.CandidateLearning, 0, len(candidates))
	droppedCount := 0

	for _, candidate := range candidates {
		if candidate.Tier == "C" {
			droppedCount++

			if l.stderr != nil {
				_, _ = fmt.Fprintf(l.stderr,
					"[engram] dropped (tier C): %q\n",
					candidate.Title)
			}

			continue
		}

		filtered = append(filtered, candidate)
	}

	return filtered, droppedCount
}
```

In `Run()`, call it after `filterByGeneralizability`:

```go
candidates, droppedCount := l.filterByGeneralizability(candidates)
candidates, tierCDropped := l.filterTierC(candidates)
droppedCount += tierCDropped
```

- [ ] **Step 4: Run tests**

Run: `targ test`
Expected: All pass.

- [ ] **Step 5: Update existing test T-60**

`TestT60_WrittenMemories_UseTierFromExtraction` in learn_test.go currently tests that tier C is written with `Confidence "C"`. Update it: tier C candidate should be dropped, only A and B written. Adjust the test expectations.

- [ ] **Step 6: Run tests**

Run: `targ test`
Expected: All pass.

- [ ] **Step 7: Commit**

```
feat: filter tier C candidates in learn pipeline (#395)

Tier C memories (contextual facts) add noise without improving
surfacing quality. Drop them at extraction time, same as null tier.

AI-Used: [claude]
```

---

## Task 2: Update classify test for tier C

**Files:**
- Modify: `internal/classify/classify_test.go`

- [ ] **Step 1: Update T-4 tier C test**

`TestT4_TierGatedAntiPattern` has a subtest "tier C has empty anti-pattern" that verifies tier C classification works. This test is still valid — the classifier still *classifies* tier C, the learn pipeline just drops it. No change needed to the classify test.

However, verify that no classify tests assume tier C memories are written to disk. If any do, update them.

- [ ] **Step 2: Run tests**

Run: `targ test`
Expected: All pass.

- [ ] **Step 3: Commit (only if changes were needed)**

```
test: update classify tests for tier C filtering (#395)

AI-Used: [claude]
```

---

## Task 3: Delete existing tier C memory files

**Files:**
- Script operation on `~/.claude/engram/data/memories/`

This task operates on the user's live data directory, not the repo. It should be implemented as an `engram maintain` subcommand or a standalone script.

- [ ] **Step 1: Write a test for tier C deletion logic**

In the appropriate test file, add a test that:
- Creates a temp directory with 3 TOML files (tier A, B, C)
- Calls the deletion function
- Verifies only tier C is removed

```go
func TestDeleteTierCMemories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	tmpDir := t.TempDir()

	// Write 3 memory files
	for _, tc := range []struct {
		name string
		tier string
	}{
		{"tier-a.toml", "A"},
		{"tier-b.toml", "B"},
		{"tier-c.toml", "C"},
	} {
		content := fmt.Sprintf("title = %q\nconfidence = %q\n", tc.name, tc.tier)
		err := os.WriteFile(filepath.Join(tmpDir, tc.name), []byte(content), 0o644)
		g.Expect(err).NotTo(HaveOccurred())
	}

	deleted, err := deleteTierCMemories(tmpDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(deleted).To(Equal(1))

	// tier-a.toml and tier-b.toml still exist
	_, errA := os.Stat(filepath.Join(tmpDir, "tier-a.toml"))
	g.Expect(errA).NotTo(HaveOccurred())

	_, errB := os.Stat(filepath.Join(tmpDir, "tier-b.toml"))
	g.Expect(errB).NotTo(HaveOccurred())

	// tier-c.toml deleted
	_, errC := os.Stat(filepath.Join(tmpDir, "tier-c.toml"))
	g.Expect(os.IsNotExist(errC)).To(BeTrue())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — function doesn't exist yet.

- [ ] **Step 3: Implement deleteTierCMemories**

Add to `internal/maintain/` (or `internal/cli/`) a function that:
1. Lists all `.toml` files in the memories directory
2. Reads each, checks `confidence` field
3. Deletes files where `confidence = "C"`
4. Returns count of deleted files

```go
func deleteTierCMemories(memoriesDir string) (int, error) {
	entries, err := os.ReadDir(memoriesDir)
	if err != nil {
		return 0, fmt.Errorf("read memories dir: %w", err)
	}

	deleted := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		path := filepath.Join(memoriesDir, entry.Name())

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var record struct {
			Confidence string `toml:"confidence"`
		}

		if err := toml.Unmarshal(data, &record); err != nil {
			continue
		}

		if record.Confidence == "C" {
			if err := os.Remove(path); err != nil {
				return deleted, fmt.Errorf("delete %s: %w", entry.Name(), err)
			}

			deleted++
		}
	}

	return deleted, nil
}
```

- [ ] **Step 4: Wire into CLI**

Add a `--purge-tier-c` flag to `engram maintain` or create a dedicated subcommand. When invoked, calls `deleteTierCMemories` on the data directory and reports the count.

- [ ] **Step 5: Run tests**

Run: `targ test`
Expected: All pass.

- [ ] **Step 6: Run the purge on live data**

Run: `engram maintain --purge-tier-c` (or equivalent)
Expected: ~1,177 tier C files deleted. Confirm with `ls ~/.claude/engram/data/memories/*.toml | wc -l` before and after.

**Important:** This is a destructive operation on live data. The user should confirm before execution.

- [ ] **Step 7: Commit**

```
feat: add tier C memory purge to maintain command (#395)

Deletes all memory TOML files with confidence="C". These contextual
facts add noise to surfacing without improving relevance. Closes #395.

AI-Used: [claude]
```

---

## Task 4: Run full checks

- [ ] **Step 1: Run targ check-full**

Run: `targ check-full`
Expected: No new failures. Fix any reorder-decls with `targ reorder-decls`.

- [ ] **Step 2: Verify production behavior**

After purging tier C memories, verify:
- `engram review` still works (no broken references)
- `engram surface --mode prompt --message "test"` still works
- Memory count dropped from ~2,800 to ~1,600

- [ ] **Step 3: Commit any remaining fixes**

```
fix: address check-full issues from tier C removal (#395)

AI-Used: [claude]
```
