# Batch Session Learning Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `projctl memory learn-sessions` command that retroactively extracts learnings from old Claude Code session transcripts, with evaluation tracking to support iterative improvement of extraction quality.

**Architecture:** New CLI command discovers transcripts under `~/.claude/projects/`, filters by recency and evaluation state (tracked in a new `processed_sessions` SQLite table), then processes each sequentially via the existing `ExtractSession` pipeline. The existing `extract-session` hook command also records to the same table for deduplication.

**Tech Stack:** Go, SQLite (embeddings.db), targ CLI framework, existing memory internals

---

### Task 1: Add `processed_sessions` table to schema

**Files:**
- Modify: `internal/memory/embeddings.go:204-211` (after `embeddings_archive` table creation)

**Step 1: Write the failing test**

File: `internal/memory/embeddings_test.go` (or whichever file tests schema init)

```go
func TestProcessedSessionsTableExists(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := InitDBForTest(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Table should exist and accept inserts
	_, err = db.Exec(`INSERT INTO processed_sessions (session_id, project, processed_at, items_found, status)
		VALUES ('test-session', 'projctl', '2026-02-15T00:00:00Z', 3, 'success')`)
	require.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM processed_sessions").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
```

**Step 2: Run test to verify it fails**

Run: `targ test -- -run TestProcessedSessionsTableExists -v ./internal/memory/`
Expected: FAIL — `no such table: processed_sessions`

**Step 3: Write minimal implementation**

Add to `internal/memory/embeddings.go` after the `embeddings_archive` table creation (around line 211):

```go
// Batch session learning: track which sessions have been processed
_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS processed_sessions (
	session_id   TEXT PRIMARY KEY,
	project      TEXT NOT NULL,
	processed_at TEXT NOT NULL,
	items_found  INTEGER NOT NULL DEFAULT 0,
	status       TEXT NOT NULL DEFAULT 'success'
)`)
```

**Step 4: Run test to verify it passes**

Run: `targ test -- -run TestProcessedSessionsTableExists -v ./internal/memory/`
Expected: PASS

**Step 5: Commit**

```
feat(memory): add processed_sessions table to schema
```

---

### Task 2: Add session recording helpers to internal/memory

**Files:**
- Create: `internal/memory/processed_sessions.go`
- Create: `internal/memory/processed_sessions_test.go`

**Step 1: Write the failing tests**

File: `internal/memory/processed_sessions_test.go`

```go
package memory_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toejough/projctl/internal/memory"
)

func TestRecordProcessedSession(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := memory.InitDBForTest(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	err = memory.RecordProcessedSession(db, "abc123.jsonl", "projctl", 4, "success")
	require.NoError(t, err)

	// Verify it was recorded
	processed, err := memory.IsSessionProcessed(db, "abc123.jsonl")
	require.NoError(t, err)
	assert.True(t, processed)
}

func TestIsSessionProcessed_NotProcessed(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := memory.InitDBForTest(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	processed, err := memory.IsSessionProcessed(db, "unknown.jsonl")
	require.NoError(t, err)
	assert.False(t, processed)
}

func TestResetLastNSessions(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := memory.InitDBForTest(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Record 3 sessions with different timestamps
	require.NoError(t, memory.RecordProcessedSession(db, "old.jsonl", "projctl", 2, "success"))
	require.NoError(t, memory.RecordProcessedSession(db, "mid.jsonl", "projctl", 3, "success"))
	require.NoError(t, memory.RecordProcessedSession(db, "new.jsonl", "projctl", 1, "success"))

	// Reset last 2
	count, err := memory.ResetLastNSessions(db, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// The oldest should still be processed
	processed, err := memory.IsSessionProcessed(db, "old.jsonl")
	require.NoError(t, err)
	assert.True(t, processed)

	// The two newest should be cleared
	processed, err = memory.IsSessionProcessed(db, "new.jsonl")
	require.NoError(t, err)
	assert.False(t, processed)
}
```

**Step 2: Run tests to verify they fail**

Run: `targ test -- -run 'TestRecordProcessedSession|TestIsSessionProcessed|TestResetLastNSessions' -v ./internal/memory/`
Expected: FAIL — functions don't exist

**Step 3: Write minimal implementation**

File: `internal/memory/processed_sessions.go`

```go
package memory

import (
	"database/sql"
	"time"
)

// RecordProcessedSession records that a session transcript has been processed.
func RecordProcessedSession(db *sql.DB, sessionID, project string, itemsFound int, status string) error {
	_, err := db.Exec(
		`INSERT OR REPLACE INTO processed_sessions (session_id, project, processed_at, items_found, status)
		 VALUES (?, ?, ?, ?, ?)`,
		sessionID, project, time.Now().UTC().Format(time.RFC3339), itemsFound, status,
	)
	return err
}

// IsSessionProcessed checks whether a session has already been processed.
func IsSessionProcessed(db *sql.DB, sessionID string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM processed_sessions WHERE session_id = ?", sessionID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ResetLastNSessions clears evaluation flags for the N most recently processed sessions.
// Returns the number of sessions reset.
func ResetLastNSessions(db *sql.DB, n int) (int64, error) {
	result, err := db.Exec(
		`DELETE FROM processed_sessions WHERE session_id IN (
			SELECT session_id FROM processed_sessions ORDER BY processed_at DESC LIMIT ?
		)`, n,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
```

**Step 4: Run tests to verify they pass**

Run: `targ test -- -run 'TestRecordProcessedSession|TestIsSessionProcessed|TestResetLastNSessions' -v ./internal/memory/`
Expected: PASS

**Step 5: Commit**

```
feat(memory): add processed session tracking helpers
```

---

### Task 3: Add session discovery function

**Files:**
- Create: `internal/memory/discover_sessions.go`
- Create: `internal/memory/discover_sessions_test.go`

**Step 1: Write the failing test**

File: `internal/memory/discover_sessions_test.go`

```go
package memory_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toejough/projctl/internal/memory"
)

func TestDiscoverSessions_FindsJSONL(t *testing.T) {
	projectsDir := t.TempDir()

	// Create a project dir with session transcripts
	projDir := filepath.Join(projectsDir, "-Users-joe-repos-personal-projctl")
	require.NoError(t, os.MkdirAll(projDir, 0o755))

	// Create a session file with enough content to pass min-size
	content := make([]byte, 9000) // > 8KB
	require.NoError(t, os.WriteFile(filepath.Join(projDir, "abc123.jsonl"), content, 0o644))

	sessions, err := memory.DiscoverSessions(memory.DiscoverOpts{
		ProjectsDir: projectsDir,
		Days:        7,
		MinSize:     8192,
	})
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "abc123.jsonl", sessions[0].SessionID)
	assert.Equal(t, "projctl", sessions[0].Project)
}

func TestDiscoverSessions_SkipsSubagents(t *testing.T) {
	projectsDir := t.TempDir()
	projDir := filepath.Join(projectsDir, "-Users-joe-repos-personal-projctl")
	subagentDir := filepath.Join(projDir, "someid", "subagents")
	require.NoError(t, os.MkdirAll(subagentDir, 0o755))

	content := make([]byte, 9000)
	require.NoError(t, os.WriteFile(filepath.Join(subagentDir, "agent-abc.jsonl"), content, 0o644))

	sessions, err := memory.DiscoverSessions(memory.DiscoverOpts{
		ProjectsDir: projectsDir,
		Days:        7,
		MinSize:     8192,
	})
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
}

func TestDiscoverSessions_SkipsSmallFiles(t *testing.T) {
	projectsDir := t.TempDir()
	projDir := filepath.Join(projectsDir, "-Users-joe-repos-personal-projctl")
	require.NoError(t, os.MkdirAll(projDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(projDir, "tiny.jsonl"), []byte("small"), 0o644))

	sessions, err := memory.DiscoverSessions(memory.DiscoverOpts{
		ProjectsDir: projectsDir,
		Days:        7,
		MinSize:     8192,
	})
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
}

func TestDiscoverSessions_FiltersByDays(t *testing.T) {
	projectsDir := t.TempDir()
	projDir := filepath.Join(projectsDir, "-Users-joe-repos-personal-projctl")
	require.NoError(t, os.MkdirAll(projDir, 0o755))

	content := make([]byte, 9000)
	path := filepath.Join(projDir, "old.jsonl")
	require.NoError(t, os.WriteFile(path, content, 0o644))

	// Set modification time to 30 days ago
	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(path, oldTime, oldTime))

	sessions, err := memory.DiscoverSessions(memory.DiscoverOpts{
		ProjectsDir: projectsDir,
		Days:        7,
		MinSize:     8192,
	})
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
}

func TestDiscoverSessions_SortsMostRecentFirst(t *testing.T) {
	projectsDir := t.TempDir()
	projDir := filepath.Join(projectsDir, "-Users-joe-repos-personal-projctl")
	require.NoError(t, os.MkdirAll(projDir, 0o755))

	content := make([]byte, 9000)

	// Create two sessions with different timestamps
	path1 := filepath.Join(projDir, "older.jsonl")
	path2 := filepath.Join(projDir, "newer.jsonl")
	require.NoError(t, os.WriteFile(path1, content, 0o644))
	require.NoError(t, os.WriteFile(path2, content, 0o644))

	// Make path1 older
	olderTime := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(path1, olderTime, olderTime))

	sessions, err := memory.DiscoverSessions(memory.DiscoverOpts{
		ProjectsDir: projectsDir,
		Days:        7,
		MinSize:     8192,
	})
	require.NoError(t, err)
	require.Len(t, sessions, 2)
	assert.Equal(t, "newer.jsonl", sessions[0].SessionID)
	assert.Equal(t, "older.jsonl", sessions[1].SessionID)
}
```

**Step 2: Run tests to verify they fail**

Run: `targ test -- -run 'TestDiscoverSessions' -v ./internal/memory/`
Expected: FAIL — types and functions don't exist

**Step 3: Write minimal implementation**

File: `internal/memory/discover_sessions.go`

```go
package memory

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DiscoveredSession represents a session transcript found on disk.
type DiscoveredSession struct {
	SessionID string // filename (e.g. "abc123.jsonl")
	Project   string // derived project name
	Path      string // full filesystem path
	ModTime   time.Time
	Size      int64
}

// DiscoverOpts configures session discovery.
type DiscoverOpts struct {
	ProjectsDir string // e.g. ~/.claude/projects
	Days        int    // only sessions modified within last N days (0 = no filter)
	Last        int    // only the N most recent sessions (0 = no limit, overrides Days)
	MinSize     int64  // skip files smaller than this (bytes)
}

// DiscoverSessions scans ProjectsDir for .jsonl transcript files,
// filtering by recency and size. Results are sorted most-recent-first.
func DiscoverSessions(opts DiscoverOpts) ([]DiscoveredSession, error) {
	var sessions []DiscoveredSession
	cutoff := time.Time{}
	if opts.Days > 0 && opts.Last == 0 {
		cutoff = time.Now().Add(-time.Duration(opts.Days) * 24 * time.Hour)
	}

	err := filepath.Walk(opts.ProjectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible dirs
		}
		if info.IsDir() {
			// Skip subagent directories
			if info.Name() == "subagents" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".jsonl") {
			return nil
		}
		if opts.MinSize > 0 && info.Size() < opts.MinSize {
			return nil
		}
		if !cutoff.IsZero() && info.ModTime().Before(cutoff) {
			return nil
		}

		// Derive project from the directory name immediately under ProjectsDir
		rel, _ := filepath.Rel(opts.ProjectsDir, path)
		parts := strings.SplitN(rel, string(filepath.Separator), 2)
		project := DeriveProjectName(reverseClaudeProjectDir(parts[0]))

		sessions = append(sessions, DiscoveredSession{
			SessionID: info.Name(),
			Project:   project,
			Path:      path,
			ModTime:   info.ModTime(),
			Size:      info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort most recent first
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ModTime.After(sessions[j].ModTime)
	})

	// Apply --last limit
	if opts.Last > 0 && len(sessions) > opts.Last {
		sessions = sessions[:opts.Last]
	}

	return sessions, nil
}

// reverseClaudeProjectDir converts Claude's project dir encoding back to a path.
// e.g. "-Users-joe-repos-personal-projctl" -> "/Users/joe/repos/personal/projctl"
func reverseClaudeProjectDir(dirName string) string {
	if dirName == "" || dirName == "-" {
		return "/"
	}
	// Replace leading dash and internal dashes with path separators
	return "/" + strings.ReplaceAll(strings.TrimPrefix(dirName, "-"), "-", "/")
}
```

Note: `DeriveProjectName` already exists in `hook_input.go` — it returns `filepath.Base(cwd)` which gives the last path component. Combined with `reverseClaudeProjectDir`, this turns `-Users-joe-repos-personal-projctl` into `projctl`.

**Step 4: Run tests to verify they pass**

Run: `targ test -- -run 'TestDiscoverSessions' -v ./internal/memory/`
Expected: PASS

**Step 5: Commit**

```
feat(memory): add session transcript discovery
```

---

### Task 4: Wire up the `learn-sessions` CLI command

**Files:**
- Create: `cmd/projctl/memory_learn_sessions.go`
- Modify: `cmd/projctl/main.go:91` (add command registration after `extract-session`)

**Step 1: Write the CLI command**

File: `cmd/projctl/memory_learn_sessions.go`

```go
package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/toejough/projctl/internal/memory"
)

type memoryLearnSessionsArgs struct {
	Days      int    `targ:"flag,desc=Process sessions modified within last N days (default 7)"`
	Last      int    `targ:"flag,desc=Process the N most recent unevaluated sessions (overrides --days)"`
	MinSize   string `targ:"flag,name=min-size,desc=Skip transcripts smaller than threshold (default 8KB)"`
	DryRun    bool   `targ:"flag,name=dry-run,desc=Show what would be processed without extracting"`
	ResetLast int    `targ:"flag,name=reset-last,desc=Clear evaluation flags for last N processed sessions"`
	MemoryRoot string `targ:"flag,name=memory-root,desc=Memory root directory (default: ~/.claude/memory)"`
}

func memoryLearnSessions(args memoryLearnSessionsArgs) error {
	// Set up memory root
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = filepath.Join(home, ".claude", "memory")
	}

	// Open DB
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := memory.InitEmbeddingsDBForTest(dbPath) // TODO: export a non-test init
	if err != nil {
		return fmt.Errorf("failed to open embeddings database: %w", err)
	}
	defer db.Close()

	// Handle --reset-last
	if args.ResetLast > 0 {
		return handleResetLast(db, args.ResetLast)
	}

	// Discover sessions
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	projectsDir := filepath.Join(home, ".claude", "projects")

	days := args.Days
	if days == 0 && args.Last == 0 {
		days = 7
	}

	minSize := int64(8192)
	if args.MinSize != "" {
		parsed, err := parseSize(args.MinSize)
		if err != nil {
			return fmt.Errorf("invalid --min-size: %w", err)
		}
		minSize = parsed
	}

	sessions, err := memory.DiscoverSessions(memory.DiscoverOpts{
		ProjectsDir: projectsDir,
		Days:        days,
		Last:        args.Last,
		MinSize:     minSize,
	})
	if err != nil {
		return fmt.Errorf("failed to discover sessions: %w", err)
	}

	// Filter already-processed
	var unprocessed []memory.DiscoveredSession
	for _, s := range sessions {
		processed, err := memory.IsSessionProcessed(db, s.SessionID)
		if err != nil {
			return fmt.Errorf("failed to check session %s: %w", s.SessionID, err)
		}
		if !processed {
			unprocessed = append(unprocessed, s)
		}
	}

	if len(unprocessed) == 0 {
		fmt.Println("No unevaluated sessions found.")
		return nil
	}

	// Count projects and total size
	projectSet := make(map[string]bool)
	var totalSize int64
	for _, s := range unprocessed {
		projectSet[s.Project] = true
		totalSize += s.Size
	}

	fmt.Printf("Found %d unevaluated sessions across %d projects (~%s)\n",
		len(unprocessed), len(projectSet), formatSize(totalSize))

	if args.DryRun {
		for i, s := range unprocessed {
			fmt.Printf("  [%d] %s (%s) %s %s\n",
				i+1, s.SessionID, s.Project, formatSize(s.Size), s.ModTime.Format("2006-01-02"))
		}
		return nil
	}

	// Process sessions
	return processSessionsBatch(db, unprocessed, memoryRoot)
}

func handleResetLast(db *sql.DB, n int) error {
	count, err := memory.ResetLastNSessions(db, n)
	if err != nil {
		return fmt.Errorf("failed to reset sessions: %w", err)
	}
	fmt.Printf("Reset %d session evaluation flags.\n", count)
	return nil
}

func processSessionsBatch(db *sql.DB, sessions []memory.DiscoveredSession, memoryRoot string) error {
	matcher := memory.NewMemoryStoreSemanticMatcher(memoryRoot)
	extractor := memory.NewLLMExtractor()
	if extractor == nil {
		return fmt.Errorf("LLM extractor unavailable (keychain auth failed)")
	}

	var totalLearnings int
	for i, s := range sessions {
		fmt.Printf("[%d/%d] Extracting %s (%s)...\n", i+1, len(sessions), s.SessionID, s.Project)

		result, status := extractWithTimeout(s, memoryRoot, matcher, extractor)

		itemsFound := 0
		if result != nil {
			itemsFound = result.ItemsExtracted
			if result.Status == "partial" {
				status = "partial"
			}
		}

		if err := memory.RecordProcessedSession(db, s.SessionID, s.Project, itemsFound, status); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: failed to record session: %v\n", err)
		}

		totalLearnings += itemsFound
		fmt.Printf("  -> %d learnings extracted\n", itemsFound)
	}

	fmt.Printf("\nProcessed %d sessions, extracted %d learnings.\n", len(sessions), totalLearnings)
	return nil
}

func extractWithTimeout(s memory.DiscoveredSession, memoryRoot string, matcher memory.SemanticMatcher, extractor memory.Extractor) (*memory.ExtractSessionResult, string) {
	type extractResult struct {
		result *memory.ExtractSessionResult
		err    error
	}
	done := make(chan extractResult, 1)
	go func() {
		r, err := memory.ExtractSession(memory.ExtractSessionOpts{
			TranscriptPath: s.Path,
			MemoryRoot:     memoryRoot,
			Project:        s.Project,
			Matcher:        matcher,
			Extractor:      extractor,
		})
		done <- extractResult{r, err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: extraction failed: %v\n", r.err)
			return nil, "partial"
		}
		return r.result, "success"
	case <-time.After(60 * time.Second):
		fmt.Fprintf(os.Stderr, "  Warning: timed out after 60s, skipping\n")
		return nil, "timeout"
	}
}

// parseSize parses a human-readable size like "8KB" into bytes.
func parseSize(s string) (int64, error) {
	// Simple parser for common sizes
	var num int64
	var unit string
	_, err := fmt.Sscanf(s, "%d%s", &num, &unit)
	if err != nil {
		// Try without unit (raw bytes)
		_, err = fmt.Sscanf(s, "%d", &num)
		if err != nil {
			return 0, fmt.Errorf("cannot parse %q", s)
		}
		return num, nil
	}
	switch unit {
	case "KB", "kb", "K", "k":
		return num * 1024, nil
	case "MB", "mb", "M", "m":
		return num * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("unknown unit %q", unit)
	}
}

// formatSize formats bytes into a human-readable string.
func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.0fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
```

Note: `InitEmbeddingsDBForTest` is used above as a placeholder. The actual approach depends on whether there's a public init function available. Check if a non-test wrapper exists; if not, add one (e.g. `InitEmbeddingsDB(path string) (*sql.DB, error)`).

**Step 2: Register in main.go**

Add after the `extract-session` line (line 91 in `cmd/projctl/main.go`):

```go
targ.Targ(memoryLearnSessions).Name("learn-sessions").Description("Learn from unevaluated session transcripts"),
```

**Step 3: Build and verify help**

Run: `targ install-binary && projctl memory learn-sessions --help`
Expected: Shows flags: `--days`, `--last`, `--min-size`, `--dry-run`, `--reset-last`, `--memory-root`

**Step 4: Commit**

```
feat(memory): add learn-sessions CLI command
```

---

### Task 5: Record processed sessions from extract-session hook

**Files:**
- Modify: `cmd/projctl/memory_extract_session.go`

**Step 1: Write the change**

After the successful extraction in `memory_extract_session.go` (after the `select` block, before building `SessionSummary`), add recording to `processed_sessions`:

```go
// Record this session as processed (for learn-sessions dedup)
sessionID := filepath.Base(transcriptPath)
dbPath := filepath.Join(memoryRoot, "embeddings.db")
if recDB, err := memory.InitEmbeddingsDBForTest(dbPath); err == nil {
	_ = memory.RecordProcessedSession(recDB, sessionID, project, result.ItemsExtracted, result.Status)
	_ = recDB.Close()
}
```

This is best-effort — failure to record doesn't fail the extraction.

**Step 2: Build and verify**

Run: `targ install-binary`
Expected: Compiles without errors

**Step 3: Commit**

```
feat(memory): record extract-session results for learn-sessions dedup
```

---

### Task 6: Smoke test with one real session

**Step 1: Dry run to see what's available**

Run: `projctl memory learn-sessions --last 1 --dry-run`
Expected: Shows 1 unevaluated session with its ID, project, size, and date

**Step 2: Extract from one session**

Run: `projctl memory learn-sessions --last 1`
Expected:
```
Found 1 unevaluated sessions across 1 projects (~3.1MB)
[1/1] Extracting abc123.jsonl (projctl)...
  -> N learnings extracted
Processed 1 sessions, extracted N learnings.
```

**Step 3: Verify idempotency**

Run: `projctl memory learn-sessions --last 1`
Expected: `No unevaluated sessions found.`

**Step 4: Test reset**

Run: `projctl memory learn-sessions --reset-last 1`
Expected: `Reset 1 session evaluation flags.`

Run: `projctl memory learn-sessions --last 1`
Expected: Finds and re-processes the same session

**Step 5: Check learning quality**

Run: `projctl memory optimize --review`
Expected: New learnings appear and are actionable (not canned boilerplate)

**Step 6: Commit** (if any fixes were needed)

```
fix(memory): [describe any fixes from smoke testing]
```

---

### Task 7: Export a non-test DB init function

If Task 4 required using `InitEmbeddingsDBForTest`, rename or add a public init:

**Files:**
- Modify: `internal/memory/embeddings.go`

**Step 1: Add public function**

```go
// InitEmbeddingsDB initializes and returns the embeddings database at the given path.
// The caller is responsible for closing the returned *sql.DB.
func InitEmbeddingsDB(dbPath string) (*sql.DB, error) {
	return initEmbeddingsDB(dbPath)
}
```

**Step 2: Update callers**

Replace `InitEmbeddingsDBForTest` calls in `cmd/projctl/memory_learn_sessions.go` and `cmd/projctl/memory_extract_session.go` with `InitEmbeddingsDB`.

**Step 3: Build**

Run: `targ install-binary`
Expected: Compiles

**Step 4: Commit**

```
refactor(memory): export InitEmbeddingsDB for non-test callers
```
