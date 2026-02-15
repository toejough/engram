# ISSUE-232: Remove Re-Query Feedback Detection

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove the re-query similarity detection that bulk-inserts bogus "wrong" feedback on hook-driven queries.

**Architecture:** Delete the re-query detection block, its supporting functions/types, related tests, and the `last_query.json` cache file. Reset `flagged_for_review` on all embeddings since flags were set by bogus feedback.

**Tech Stack:** Go, SQLite

---

### Task 1: Remove re-query detection logic from Query()

**Files:**
- Modify: `internal/memory/memory.go:502-540`

**Step 1: Delete the re-query detection block and SaveLastQueryResults call**

Remove lines 502-540 (the entire block from `// ISSUE-214: Implicit re-ask detection` through the `SaveLastQueryResults` call):

```go
// DELETE THIS ENTIRE BLOCK (memory.go:502-540):

	// ISSUE-214: Implicit re-ask detection
	previousResults, previousQuery, loadErr := LoadLastQueryResults(opts.MemoryRoot)
	if loadErr == nil && previousQuery != "" && len(previousResults) > 0 {
		// ... similarity detection and RecordFeedback loop ...
	}

	// Save current query results for next re-ask detection
	if saveErr := SaveLastQueryResults(results, opts.Text, opts.MemoryRoot); saveErr != nil {
		// Log error but don't fail the query
		fmt.Fprintf(os.Stderr, "Warning: failed to save last query results: %v\n", saveErr)
	}
```

**Step 2: Verify build**

Run: `go build ./cmd/projctl/`
Expected: Clean build (no compile errors)

---

### Task 2: Remove LastQueryCache, SaveLastQueryResults, LoadLastQueryResults

**Files:**
- Modify: `internal/memory/memory.go:1193-1242`

**Step 1: Delete the LastQueryCache struct and both functions**

Remove the entire section from line 1193 (`// ISSUE-214: Last query caching`) through line 1242 (end of `LoadLastQueryResults`).

```go
// DELETE THIS ENTIRE SECTION (memory.go:1193-1242):

// ISSUE-214: Last query caching for implicit re-ask detection
// ============================================================================

// LastQueryCache represents cached query results for re-ask detection.
type LastQueryCache struct { ... }

func SaveLastQueryResults(...) error { ... }

func LoadLastQueryResults(...) ([]QueryResult, string, error) { ... }
```

**Step 2: Verify build**

Run: `go build ./cmd/projctl/`
Expected: Clean build

---

### Task 3: Remove related tests

**Files:**
- Modify: `internal/memory/feedback_test.go`

**Step 1: Delete TestSaveLoadLastQueryResults**

Remove `TestSaveLoadLastQueryResults` (feedback_test.go:311-343).

**Step 2: Delete TestImplicitReAskDetection**

Remove `TestImplicitReAskDetection` (feedback_test.go:345-386).

**Step 3: Remove workaround comment in TestRecordFeedback_Helpful**

In `TestRecordFeedback_Helpful` (around line 148), remove the comment `// Query again with completely different text to avoid re-ask detection` — the workaround is no longer needed. The test logic itself is fine and should stay.

**Step 4: Run tests**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run "TestSaveLoad|TestImplicit|TestRecordFeedback_Helpful" -v`
Expected: Only `TestRecordFeedback_Helpful` runs and passes. The other two should not exist.

---

### Task 4: Reset flagged_for_review and clean up last_query.json

**Files:**
- Modify: `internal/memory/memory.go` (add a `CleanupReQueryArtifacts` function)

**Step 1: Write failing test**

Create test in `internal/memory/feedback_test.go`:

```go
func TestCleanupReQueryArtifacts(t *testing.T) {
	g := NewWithT(t)
	memoryRoot := t.TempDir()

	// Set up DB with flagged entries
	db, err := memory.InitTestDB(filepath.Join(memoryRoot, "embeddings.db"))
	g.Expect(err).To(BeNil())

	_, err = db.Exec(`INSERT INTO embeddings (content, source, flagged_for_review) VALUES ('test1', 'memory', 1)`)
	g.Expect(err).To(BeNil())
	_, err = db.Exec(`INSERT INTO embeddings (content, source, flagged_for_review) VALUES ('test2', 'memory', 0)`)
	g.Expect(err).To(BeNil())

	// Create stale last_query.json
	lastQueryPath := filepath.Join(memoryRoot, "last_query.json")
	err = os.WriteFile(lastQueryPath, []byte(`{"query_text":"test"}`), 0644)
	g.Expect(err).To(BeNil())

	db.Close()

	// Run cleanup
	count, err := memory.CleanupReQueryArtifacts(memoryRoot)
	g.Expect(err).To(BeNil())
	g.Expect(count).To(Equal(1)) // Only 1 entry was flagged

	// Verify flag reset
	db2, err := memory.InitTestDB(filepath.Join(memoryRoot, "embeddings.db"))
	g.Expect(err).To(BeNil())
	defer db2.Close()

	var flaggedCount int
	err = db2.QueryRow("SELECT COUNT(*) FROM embeddings WHERE flagged_for_review = 1").Scan(&flaggedCount)
	g.Expect(err).To(BeNil())
	g.Expect(flaggedCount).To(Equal(0))

	// Verify last_query.json deleted
	_, err = os.Stat(lastQueryPath)
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}
```

**Step 2: Run test to verify it fails**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestCleanupReQueryArtifacts -v`
Expected: FAIL — `CleanupReQueryArtifacts` not defined

**Step 3: Implement CleanupReQueryArtifacts**

Add to `internal/memory/memory.go` (near the end, before the activation scoring section):

```go
// CleanupReQueryArtifacts removes stale re-query detection artifacts (ISSUE-232).
// Resets flagged_for_review on all embeddings and deletes last_query.json.
// Returns the number of entries that had their flag reset.
func CleanupReQueryArtifacts(memoryRoot string) (int, error) {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Reset flagged_for_review
	result, err := db.Exec("UPDATE embeddings SET flagged_for_review = 0 WHERE flagged_for_review = 1")
	if err != nil {
		return 0, fmt.Errorf("failed to reset flagged_for_review: %w", err)
	}
	count, _ := result.RowsAffected()

	// Delete last_query.json
	_ = os.Remove(filepath.Join(memoryRoot, "last_query.json"))

	return int(count), nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestCleanupReQueryArtifacts -v`
Expected: PASS

---

### Task 5: Wire cleanup into optimize and run full test suite

**Files:**
- Modify: `internal/memory/optimize_interactive.go` (add cleanup call after DB open)

**Step 1: Add cleanup call in OptimizeInteractive**

In `optimize_interactive.go`, after the DB is opened (line 115), add:

```go
	// ISSUE-232: Clean up stale re-query detection artifacts
	if cleaned, err := CleanupReQueryArtifacts(filepath.Dir(opts.DBPath)); err != nil {
		fmt.Fprintf(opts.Output, "Warning: failed to clean re-query artifacts: %v\n", err)
	} else if cleaned > 0 {
		fmt.Fprintf(opts.Output, "Cleaned %d stale flagged_for_review entries from re-query detection.\n", cleaned)
	}
```

**Step 2: Run full test suite**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -v`
Expected: All tests pass

**Step 3: Build and install**

Run: `go build ./cmd/projctl/`
Expected: Clean build

---

### Task 6: Commit

**Step 1: Stage and commit all changes**

```bash
git add internal/memory/memory.go internal/memory/feedback_test.go internal/memory/optimize_interactive.go
git commit -m "fix(memory): remove re-query detection that poisoned feedback table (ISSUE-232)

Removes implicit re-ask detection that bulk-inserted 'wrong' feedback
on every hook-driven query. Adds one-time cleanup of flagged_for_review
and last_query.json artifacts during optimize.

AI-Used: [claude]"
```

### Task 7: Update issue status

**Files:**
- Modify: `docs/issues.md` (ISSUE-232 entry)

**Step 1: Mark ISSUE-232 as closed with resolution notes**

Update the issue status to Closed and add a Resolution section summarizing what was done.

**Step 2: Commit**

```bash
git add docs/issues.md
git commit -m "docs: close ISSUE-232 with resolution notes

AI-Used: [claude]"
```
