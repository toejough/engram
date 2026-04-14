# Spec 1: Memory Format & Facts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Evolve engram's memory format to support feedback + facts (SPO triples), migrate 269 existing memories, update the Go binary and engram-agent skill.

**Architecture:** Unified `MemoryRecord` struct with `ContentFields` nested under `[content]` in TOML. Two directories (`memory/feedback/`, `memory/facts/`). Migration script as a standalone Go program. Backward-compat reading of legacy format. Skill renamed from `memory-agent` to `engram-agent`.

**Tech Stack:** Go 1.25, TOML (BurntSushi), targ build system, Claude Code skills (Markdown)

---

### Task 1: Restructure MemoryRecord with ContentFields

**Files:**
- Modify: `internal/memory/record.go`
- Modify: `internal/memory/record_test.go`

- [ ] **Step 1: Write failing test for new struct shape**

In `internal/memory/record_test.go`, add a test that creates a MemoryRecord with the new `Content` field containing feedback fields, encodes to TOML, and verifies the output has `[content]` section with `behavior`, `impact`, `action` nested inside:

```go
func TestMemoryRecord_ContentFields_Feedback(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	record := memory.MemoryRecord{
		SchemaVersion: 1,
		Type:          "feedback",
		Situation:     "when running tests",
		Source:        "user correction",
		Content: memory.ContentFields{
			Behavior: "using go test directly",
			Impact:   "misses coverage",
			Action:   "use targ test",
		},
		CreatedAt: "2026-04-03T00:00:00Z",
		UpdatedAt: "2026-04-03T00:00:00Z",
	}

	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(record)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	encoded := buf.String()
	g.Expect(encoded).To(ContainSubstring(`type = "feedback"`))
	g.Expect(encoded).To(ContainSubstring("[content]"))
	g.Expect(encoded).To(ContainSubstring(`behavior = "using go test directly"`))
	g.Expect(encoded).NotTo(ContainSubstring(`subject =`))

	var decoded memory.MemoryRecord
	_, err = toml.Decode(encoded, &decoded)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }
	g.Expect(decoded).To(Equal(record))
}
```

- [ ] **Step 2: Write failing test for fact content fields**

```go
func TestMemoryRecord_ContentFields_Fact(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	record := memory.MemoryRecord{
		SchemaVersion:     1,
		Type:              "fact",
		Situation:         "when choosing build tools",
		Source:            "team discussion",
		InitialConfidence: 0.7,
		Content: memory.ContentFields{
			Subject:   "engram",
			Predicate: "uses",
			Object:    "targ for all build operations",
		},
		CreatedAt: "2026-04-03T00:00:00Z",
		UpdatedAt: "2026-04-03T00:00:00Z",
	}

	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(record)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	encoded := buf.String()
	g.Expect(encoded).To(ContainSubstring(`type = "fact"`))
	g.Expect(encoded).To(ContainSubstring(`subject = "engram"`))
	g.Expect(encoded).To(ContainSubstring(`predicate = "uses"`))
	g.Expect(encoded).NotTo(ContainSubstring(`behavior =`))

	var decoded memory.MemoryRecord
	_, err = toml.Decode(encoded, &decoded)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }
	g.Expect(decoded).To(Equal(record))
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `targ test`
Expected: compilation errors — `Type`, `Source`, `Content`, `ContentFields` not defined

- [ ] **Step 4: Implement new struct**

In `internal/memory/record.go`, replace the MemoryRecord struct:

```go
// ContentFields holds type-specific memory content.
// Feedback: Behavior, Impact, Action.
// Fact: Subject, Predicate, Object.
// Fields for the other type remain empty (omitempty).
type ContentFields struct {
	// Feedback fields
	Behavior string `toml:"behavior,omitempty"`
	Impact   string `toml:"impact,omitempty"`
	Action   string `toml:"action,omitempty"`

	// Fact fields
	Subject   string `toml:"subject,omitempty"`
	Predicate string `toml:"predicate,omitempty"`
	Object    string `toml:"object,omitempty"`
}

type MemoryRecord struct {
	SchemaVersion int    `toml:"schema_version"`
	Type          string `toml:"type"`
	Situation     string `toml:"situation"`
	Source        string `toml:"source,omitempty"`
	Core          bool   `toml:"core,omitempty"`

	ProjectScoped bool   `toml:"project_scoped"`
	ProjectSlug   string `toml:"project_slug,omitempty"`

	Content ContentFields `toml:"content"`

	CreatedAt string `toml:"created_at"`
	UpdatedAt string `toml:"updated_at"`

	SurfacedCount    int `toml:"surfaced_count"`
	FollowedCount    int `toml:"followed_count"`
	NotFollowedCount int `toml:"not_followed_count"`
	IrrelevantCount  int `toml:"irrelevant_count"`
	MissedCount      int `toml:"missed_count"`

	InitialConfidence float64 `toml:"initial_confidence,omitempty"`

	PendingEvaluations []PendingEvaluation `toml:"pending_evaluations,omitempty"`
}
```

Remove the old top-level `Situation`, `Behavior`, `Impact`, `Action` fields. Keep `PendingEvaluations` for backward-compat reading (omitempty means it won't be written).

- [ ] **Step 5: Update ToStored and SearchText**

Update `ToStored` in `record.go` to pass `Content` and `Type` to `Stored`. Update `SearchText` to use `Content` fields:

```go
func (r *MemoryRecord) SearchText() string {
	parts := []string{r.Situation}
	if r.Type == "feedback" {
		parts = append(parts, r.Content.Behavior, r.Content.Impact, r.Content.Action)
	} else {
		parts = append(parts, r.Content.Subject, r.Content.Predicate, r.Content.Object)
	}
	return strings.Join(parts, " ")
}
```

- [ ] **Step 6: Update Stored struct**

In `internal/memory/memory.go`, update `Stored` to include `Type`, `Content`, and `Core`:

```go
type Stored struct {
	Type             string
	Situation        string
	Content          ContentFields
	Core             bool
	ProjectScoped    bool
	ProjectSlug      string
	SurfacedCount    int
	FollowedCount    int
	NotFollowedCount int
	IrrelevantCount  int
	UpdatedAt        time.Time
	FilePath         string
	InitialConfidence float64
}
```

Update `Stored.SearchText()` to use `Content` fields (same pattern as `MemoryRecord.SearchText`).

- [ ] **Step 7: Update existing tests that reference old field locations**

All tests that create `MemoryRecord` with top-level `Behavior`, `Impact`, `Action` must move those into `Content`. All tests that create `Stored` with top-level `Behavior` etc. must move to `Content`. Find all with:

```bash
grep -rn "Behavior:" internal/ --include="*_test.go" | grep -v worktree | grep -v Content
```

Fix each one.

- [ ] **Step 8: Run targ check-full**

Run: `targ check-full`
Expected: all tests pass. Fix any remaining compilation errors from field moves.

- [ ] **Step 9: Commit**

```
feat(memory): restructure MemoryRecord with ContentFields and Type

Move behavior/impact/action under Content nested struct.
Add Type, Source, Core fields. Add fact fields
(subject/predicate/object) to ContentFields. Update Stored
struct and SearchText to work with new layout.

AI-Used: [claude]
```

---

### Task 2: Add directory path support for new layout

**Files:**
- Modify: `internal/memory/memory.go`
- Modify: `internal/memory/memory_test.go`

- [ ] **Step 1: Write failing test for new directory functions**

```go
func TestFeedbackDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(memory.FeedbackDir("/data")).To(Equal("/data/memory/feedback"))
}

func TestFactsDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(memory.FactsDir("/data")).To(Equal("/data/memory/facts"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`

- [ ] **Step 3: Implement directory functions**

In `internal/memory/memory.go`:

```go
// FeedbackDir returns the directory for feedback memory files.
func FeedbackDir(dataDir string) string {
	return filepath.Join(dataDir, "memory", "feedback")
}

// FactsDir returns the directory for fact memory files.
func FactsDir(dataDir string) string {
	return filepath.Join(dataDir, "memory", "facts")
}
```

Keep the existing `MemoriesDir` function — it's still needed for legacy/backward-compat reading.

- [ ] **Step 4: Run tests**

Run: `targ test`

- [ ] **Step 5: Commit**

```
feat(memory): add FeedbackDir and FactsDir path functions

New directory layout: data/memory/feedback/ and data/memory/facts/.
Legacy MemoriesDir kept for backward compatibility.

AI-Used: [claude]
```

---

### Task 3: Update recall and show to read from new paths

**Files:**
- Modify: `internal/cli/cli.go` (recall path references)
- Modify: `internal/cli/show.go` (show path references)
- Modify: any file that calls `memory.MemoriesDir` for listing memories

- [ ] **Step 1: Find all MemoriesDir call sites**

```bash
grep -rn "MemoriesDir\|memories" internal/ --include="*.go" | grep -v _test.go | grep -v worktree
```

- [ ] **Step 2: Update memory listing to check both new and legacy paths**

Create a helper function in `internal/memory/memory.go` that implements the precedence rule from the spec: if `memory/feedback/` exists and is non-empty, read from new paths only. Otherwise read from legacy `memories/`:

```go
// ListAllMemoryFiles returns paths to all memory TOML files,
// checking new layout first (memory/feedback/ + memory/facts/),
// falling back to legacy (memories/) if new layout doesn't exist.
func ListAllMemoryFiles(dataDir string, lister DirLister) ([]string, error) {
	feedbackDir := FeedbackDir(dataDir)
	files, err := lister.ListTOML(feedbackDir)
	if err == nil && len(files) > 0 {
		// New layout exists — read feedback + facts
		factsFiles, _ := lister.ListTOML(FactsDir(dataDir))
		return append(files, factsFiles...), nil
	}
	// Fall back to legacy
	return lister.ListTOML(MemoriesDir(dataDir))
}
```

Adapt the `DirLister` interface or create a new one if needed. The exact implementation depends on what exists — read the current listing code first.

- [ ] **Step 3: Update show command to search both directories**

In `internal/cli/show.go`, update the path resolution to check `memory/feedback/<slug>.toml`, then `memory/facts/<slug>.toml`, then legacy `memories/<slug>.toml`.

- [ ] **Step 4: Update show display format for facts**

When showing a fact, display:
```
Type: fact
Situation: <situation>
Subject: <subject>
Predicate: <predicate>
Object: <object>
Source: <source>
```

- [ ] **Step 5: Update recall surfacer to read from new paths**

Update the surfacer in `internal/cli/cli.go` (`buildRecallSurfacer`) to use `ListAllMemoryFiles` instead of `MemoriesDir` directly.

- [ ] **Step 6: Run targ check-full**

Run: `targ check-full`

- [ ] **Step 7: Commit**

```
feat(cli): update recall and show for new memory layout

Read from memory/feedback/ + memory/facts/ when available,
fall back to legacy memories/. Show command displays fact
format (subject/predicate/object).

AI-Used: [claude]
```

---

### Task 4: Write migration script

**Files:**
- Create: `cmd/migrate-v2/main.go`

- [ ] **Step 1: Write the migration program**

Create `cmd/migrate-v2/main.go` that:

1. Takes `--data-dir` flag (default: `~/.claude/engram/data`)
2. Reads all `.toml` files from `data/memories/`
3. For each file:
   - Parse as legacy format (top-level behavior/impact/action)
   - Transform to new format:
     - `schema_version = 1`
     - `type = "feedback"`
     - `source = ""`
     - `core = false`
     - Nest behavior/impact/action under `[content]`
     - Strip `pending_evaluations`
     - Preserve all other fields
   - Validate output parses correctly
   - Write atomically to `data/memory/feedback/<filename>` (temp + rename)
4. After all files succeed: rename `data/memories/` to `data/memories.v1-backup/`
5. Log each file to stdout
6. Exit non-zero on any failure

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// LegacyRecord represents the old memory format.
type LegacyRecord struct {
	SchemaVersion     int     `toml:"schema_version,omitempty"`
	Situation         string  `toml:"situation"`
	Behavior          string  `toml:"behavior"`
	Impact            string  `toml:"impact"`
	Action            string  `toml:"action"`
	ProjectScoped     bool    `toml:"project_scoped"`
	ProjectSlug       string  `toml:"project_slug,omitempty"`
	CreatedAt         string  `toml:"created_at"`
	UpdatedAt         string  `toml:"updated_at"`
	SurfacedCount     int     `toml:"surfaced_count"`
	FollowedCount     int     `toml:"followed_count"`
	NotFollowedCount  int     `toml:"not_followed_count"`
	IrrelevantCount   int     `toml:"irrelevant_count"`
	MissedCount       int     `toml:"missed_count"`
	InitialConfidence float64 `toml:"initial_confidence,omitempty"`
}

func main() {
	dataDir := flag.String("data-dir", filepath.Join(os.Getenv("HOME"), ".claude", "engram", "data"), "path to data directory")
	flag.Parse()

	srcDir := filepath.Join(*dataDir, "memories")
	dstDir := filepath.Join(*dataDir, "memory", "feedback")
	backupDir := filepath.Join(*dataDir, "memories.v1-backup")

	// Check source exists
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", srcDir, err)
		os.Exit(1)
	}

	// Create destination
	if err := os.MkdirAll(dstDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "error creating %s: %v\n", dstDir, err)
		os.Exit(1)
	}

	// Also create facts dir
	factsDir := filepath.Join(*dataDir, "memory", "facts")
	if err := os.MkdirAll(factsDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "error creating %s: %v\n", factsDir, err)
		os.Exit(1)
	}

	migrated := 0
	skipped := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		dstPath := filepath.Join(dstDir, entry.Name())

		// Idempotent: skip if already exists
		if _, statErr := os.Stat(dstPath); statErr == nil {
			fmt.Printf("SKIP (exists): %s\n", entry.Name())
			skipped++
			continue
		}

		srcPath := filepath.Join(srcDir, entry.Name())
		if migrateErr := migrateFile(srcPath, dstPath); migrateErr != nil {
			fmt.Fprintf(os.Stderr, "FAIL: %s: %v\n", entry.Name(), migrateErr)
			os.Exit(1)
		}

		fmt.Printf("OK: %s\n", entry.Name())
		migrated++
	}

	// Rename source to backup
	if err := os.Rename(srcDir, backupDir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not rename %s to %s: %v\n", srcDir, backupDir, err)
	}

	fmt.Printf("\nMigrated: %d, Skipped: %d\n", migrated, skipped)
}

func migrateFile(src, dst string) error {
	var legacy LegacyRecord
	if _, err := toml.DecodeFile(src, &legacy); err != nil {
		return fmt.Errorf("decoding: %w", err)
	}

	// Build new format
	type contentSection struct {
		Behavior string `toml:"behavior,omitempty"`
		Impact   string `toml:"impact,omitempty"`
		Action   string `toml:"action,omitempty"`
	}

	type newRecord struct {
		SchemaVersion     int            `toml:"schema_version"`
		Type              string         `toml:"type"`
		Situation         string         `toml:"situation"`
		Source            string         `toml:"source,omitempty"`
		Core              bool           `toml:"core,omitempty"`
		ProjectScoped     bool           `toml:"project_scoped"`
		ProjectSlug       string         `toml:"project_slug,omitempty"`
		Content           contentSection `toml:"content"`
		CreatedAt         string         `toml:"created_at"`
		UpdatedAt         string         `toml:"updated_at"`
		SurfacedCount     int            `toml:"surfaced_count"`
		FollowedCount     int            `toml:"followed_count"`
		NotFollowedCount  int            `toml:"not_followed_count"`
		IrrelevantCount   int            `toml:"irrelevant_count"`
		MissedCount       int            `toml:"missed_count"`
		InitialConfidence float64        `toml:"initial_confidence,omitempty"`
	}

	out := newRecord{
		SchemaVersion:     1,
		Type:              "feedback",
		Situation:         legacy.Situation,
		ProjectScoped:     legacy.ProjectScoped,
		ProjectSlug:       legacy.ProjectSlug,
		Content: contentSection{
			Behavior: legacy.Behavior,
			Impact:   legacy.Impact,
			Action:   legacy.Action,
		},
		CreatedAt:         legacy.CreatedAt,
		UpdatedAt:         legacy.UpdatedAt,
		SurfacedCount:     legacy.SurfacedCount,
		FollowedCount:     legacy.FollowedCount,
		NotFollowedCount:  legacy.NotFollowedCount,
		IrrelevantCount:   legacy.IrrelevantCount,
		MissedCount:       legacy.MissedCount,
		InitialConfidence: legacy.InitialConfidence,
	}

	// Atomic write: temp file + rename
	tmpPath := dst + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("creating temp: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(out); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("encoding: %w", err)
	}
	f.Close()

	// Validate by re-reading
	var verify newRecord
	if _, err := toml.DecodeFile(tmpPath, &verify); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("validation: %w", err)
	}

	return os.Rename(tmpPath, dst)
}
```

- [ ] **Step 2: Build and test on real data**

```bash
go build -o /tmp/migrate-v2 ./cmd/migrate-v2/
/tmp/migrate-v2 --data-dir ~/.claude/engram/data
```

Verify: all files migrated, backup created, new files parse correctly.

- [ ] **Step 3: Commit**

```
feat: add v2 migration script for memory format

Migrates 269 legacy memory files from data/memories/ to
data/memory/feedback/ with new format (type, source, core,
nested content). Creates v1 backup. Idempotent.

AI-Used: [claude]
```

---

### Task 5: Rename memory-agent skill to engram-agent and update for facts

**Files:**
- Delete: `skills/memory-agent/SKILL.md`
- Create: `skills/engram-agent/SKILL.md`

- [ ] **Step 1: Create engram-agent skill**

Create `skills/engram-agent/SKILL.md` based on the current `skills/memory-agent/SKILL.md`, with these changes:

1. Rename references from `memory-agent` to `engram-agent`
2. Update data paths from `~/.claude/engram/data/memories/` to `~/.claude/engram/data/memory/feedback/` and `~/.claude/engram/data/memory/facts/`
3. Add fact surfacing section:
   - On each intent, after checking feedback matches, check fact subjects/objects for overlap
   - Surface facts as INFO messages (not WAIT — no arguments for facts)
   - Format: `[FACT] <subject> <predicate> <object> (situation: <situation>)`
4. Add fact learning section:
   - Extract SPO triples from `intent` and `done` messages only
   - Confidence: 0.7 clear assertions, 0.4 inferred
   - Knowledge patterns table (simple fact, concept, decision, excerpt, process)
   - Negative examples (proposals, questions, hypotheticals, opinions)
   - Dedup: exact subject+predicate+object = skip
   - Conflict: same subject+predicate, different object = confidence-based resolution
5. Add tiered loading section:
   - Core (`core = true`): always loaded
   - Recent (updated in last 7 days): loaded on startup
   - On-demand: searched when core/recent match found
   - Auto-promotion: followed/surfaced > 0.7 AND surfaced >= 5
   - Auto-demotion: followed == 0 AND surfaced >= 10
   - Core cap: max 20 auto-promoted (user-pinned don't count)
6. Update processing order: feedback triggers first, then fact triggers
7. Update memory file format section to show new format with `[content]`

- [ ] **Step 2: Delete old skill**

```bash
rm -rf skills/memory-agent/
```

- [ ] **Step 3: Commit**

```
feat(skills): rename memory-agent to engram-agent, add fact support

New skill handles both feedback and facts. Fact surfacing via
INFO messages, fact learning from conversation with SPO triple
extraction, tiered loading (core/recent/on-demand).

AI-Used: [claude]
```

---

### Task 6: Update recall skill

**Files:**
- Modify: `skills/recall/SKILL.md`

- [ ] **Step 1: Update path and output format**

Update `skills/recall/SKILL.md`:
- The binary path stays the same (`~/.claude/engram/bin/engram recall`)
- Add note that output now includes facts: after `=== MEMORIES ===`, both feedback (SBIA format) and facts (`[FACT] subject predicate object`) may appear
- No other changes

- [ ] **Step 2: Commit**

```
docs(skills): update recall skill for fact output format

AI-Used: [claude]
```

---

### Task 7: Verify and fix session-start hook

**Files:**
- Modify: `hooks/session-start.sh` (if needed)

- [ ] **Step 1: Check if hook references old paths**

```bash
grep -n "memories" hooks/session-start.sh
```

If no references to the old `memories/` directory, no changes needed. The hook builds the binary and announces recall — path changes are handled by the Go code.

- [ ] **Step 2: Commit if changed**

```
chore(hooks): update session-start for new memory paths

AI-Used: [claude]
```

---

### Task 8: Final verification

- [ ] **Step 1: Run targ check-full**

Run: `targ check-full`
Expected: all checks pass.

- [ ] **Step 2: Run migration on real data**

```bash
go build -o /tmp/migrate-v2 ./cmd/migrate-v2/
/tmp/migrate-v2 --data-dir ~/.claude/engram/data
```

Verify all 269 files migrated. Check a few output files manually.

- [ ] **Step 3: Verify recall works with new paths**

```bash
~/.claude/engram/bin/engram recall --query "test"
~/.claude/engram/bin/engram show --name mandatory-full-build-before-merge
```

Both should return results from the new `memory/feedback/` directory.

- [ ] **Step 4: Dogfood — run engram-agent**

Start the engram-agent in a separate terminal. Send a test intent via chat.toml. Verify:
- Feedback surfacing works (existing behavior)
- Agent loads from new paths
- Agent reads the engram-agent skill (not the old memory-agent skill)

- [ ] **Step 5: Commit any final fixes**
