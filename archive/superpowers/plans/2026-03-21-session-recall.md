# Session Recall Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace per-turn session-context rolling summary with opt-in `/recall` skill that reads session transcripts on demand, with both summary and search modes.

**Architecture:** Delete the context-update pipeline (orchestrator, delta reader, session file, summarizer). Inline flush to just `learn`. Add new `recall` CLI subcommand that reads Claude Code session transcripts, strips noise, and uses Haiku to summarize (mode A) or extract relevant content (mode B). Wire it through a new `/recall` skill.

**Tech Stack:** Go, Haiku API, Claude Code plugin skills, bash hooks

**Spec:** `docs/superpowers/specs/2026-03-21-session-recall-design.md`

---

### Task 1: Delete context-update pipeline and simplify flush

Remove all session-context machinery. Inline FlushRunner since it's a one-step wrapper after removing context-update.

**Files:**
- Delete: `internal/context/orchestrate.go` (contains `MaxSummaryBytes`, `Orchestrator` — both removed)
- Delete: `internal/context/file.go` (contains `SessionFile`, `SessionContext` — both removed)
- Delete: `internal/context/summarize.go` (contains `Summarizer` — removed)
- Keep: `internal/context/strip.go` — `Strip()` is used by recall's `TranscriptReader`
- Keep: `internal/context/delta.go` — `DeltaReader` is used by recall's `TranscriptReader`
- Modify: `internal/context/context.go` — remove `HaikuClient`, `DirCreator`, `FileWriter`, `Renamer`, `Timestamper` interfaces. Keep `FileReader` interface (used by `DeltaReader` and recall).
- Modify: `internal/cli/flush.go` — inline FlushRunner; delete struct, constructor, and `Run` method
- Modify: `internal/cli/cli.go:99-125` — remove `case "context-update"` from switch; remove `runContextUpdate` function (~lines 992-1054); remove `contextSummarizationPrompt`, `errContextUpdateMissingFlags` constants; remove `haikuClientAdapter` struct (~lines 522-538); update `errUsage` string to remove `context-update` (add `recall` in Task 6)
- Modify: `internal/cli/targets.go` — delete `ContextUpdateArgs` struct and `ContextUpdateFlags` func; remove `ContextPath` field from `FlushArgs`; remove context-update target from `BuildTargets`; remove `ContextPath` from `FlushFlags`
- Modify: `internal/context/context_test.go` — delete all tests except Strip tests (TestT138-T143) and DeltaReader tests (TestT134-T137, TestDeltaReader_FileReadError)
- Modify: `internal/cli/flush_test.go` — delete `TestFlush_WithTranscript_RunsPipeline`, `TestT337_FlushIntegration_PipelineOrdering`, `TestT370_FlushRunsInOrder`, `TestT371_FlushStopsOnError`; remove `--context-path` from remaining tests
- Modify: `internal/cli/cli_test.go` — delete context-update tests (grep for "context-update" and "haikuClientAdapter")

- [ ] **Step 1: Delete context package files (orchestrate, file, summarize only)**

```bash
rm internal/context/orchestrate.go internal/context/file.go internal/context/summarize.go
```

Note: keep `delta.go` and `strip.go` — recall needs `DeltaReader` and `Strip()`.

- [ ] **Step 2: Strip context.go to FileReader only**

Remove from `internal/context/context.go`: `HaikuClient`, `DirCreator`, `FileWriter`, `Renamer`, `Timestamper` interfaces. Keep only `FileReader` interface (used by `DeltaReader` and recall's `TranscriptReader`).

- [ ] **Step 3: Delete context_test.go tests for deleted components**

Remove all tests except: Strip tests (TestT138-T143), DeltaReader tests (TestT134-T137, TestDeltaReader_FileReadError), and their supporting fakes. Remove unused fakes: `fakeHaikuClient`, `fakeFileWriter`, `fakeDirCreator`, `fakeRenamer`, `fakeTimestamper`, `renameCall`, `newFakeFileWriter`. Keep `fakeFileReader`.

- [ ] **Step 4: Inline FlushRunner in flush.go**

Replace `flush.go` contents. `runFlush` should call `RunLearn` directly instead of through `FlushRunner`. Remove `FlushRunner` struct, `NewFlushRunner`, and `Run` method. Remove `--context-path` flag.

```go
func runFlush(args []string, _ io.Writer, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("flush", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	transcriptPath := fs.String("transcript-path", "", "path to session transcript")
	sessionID := fs.String("session-id", "", "session identifier")
	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("flush: %w", parseErr)
	}

	if *dataDir == "" {
		return errFlushMissingDataDir
	}

	// Clean up surfacing log — evaluate no longer consumes it (#348).
	_ = os.Remove(filepath.Join(*dataDir, "surfacing-log.jsonl"))

	if *transcriptPath == "" || *sessionID == "" {
		return nil
	}

	learnArgs := []string{
		"--transcript-path", *transcriptPath,
		"--session-id", *sessionID,
		"--data-dir", *dataDir,
	}

	token := os.Getenv("ENGRAM_API_TOKEN")

	return RunLearn(learnArgs, token, stderr, stdin, nil)
}
```

- [ ] **Step 5: Remove context-update from cli.go**

Remove `case "context-update"` from the switch (line 118-119). Delete `runContextUpdate` function (~lines 992-1054). Delete `contextSummarizationPrompt` constant (line 391-394). Delete `errContextUpdateMissingFlags` variable (line 407-409). Delete `haikuClientAdapter` struct and its `Summarize` method (~lines 522-538).

- [ ] **Step 6: Clean up targets.go**

Delete `ContextUpdateArgs` struct (lines 24-30). Delete `ContextUpdateFlags` func (lines 177-185). Remove `ContextPath` field from `FlushArgs` (line 55). Remove `ContextPath` from `FlushFlags` func (line 213). Remove context-update target from `BuildTargets` (lines 164-165).

- [ ] **Step 7: Delete/update flush tests**

In `flush_test.go`: delete `TestFlush_WithTranscript_RunsPipeline` (references `--context-path`), `TestT337_FlushIntegration_PipelineOrdering` (tests context-update integration), `TestT370_FlushRunsInOrder` (tests FlushRunner ordering), `TestT371_FlushStopsOnError` (tests FlushRunner error handling). Keep: `TestFlush_BadTranscriptPath_SkipsGracefully`, `TestFlush_DeletesSurfacingLog`, `TestFlush_FlagParseError`, `TestFlush_MissingDataDir`, `TestFlush_NoTranscript_SkipsGracefully`.

- [ ] **Step 8: Delete context-update tests from cli_test.go**

Search for and delete any tests referencing `context-update` or `haikuClientAdapter`.

- [ ] **Step 9: Run `targ check-full` and fix**

```bash
targ check-full
```

Fix any compilation errors, lint issues, or test failures.

- [ ] **Step 10: Commit**

```
chore: delete context-update pipeline, inline flush to learn-only

The per-turn Haiku summarization was low-value: 1024-byte hard truncation,
emoji-heavy output, and content redundant with git log + memories.
Replaced by opt-in /recall skill (next commit).
```

---

### Task 2: Simplify session-start hook

Remove memory surfacing and session context injection from session-start. Keep maintain and add `/recall` notification.

**Files:**
- Modify: `hooks/session-start.sh`

- [ ] **Step 1: Read current session-start.sh**

Already read. Key sections to change:
- Line 44: `SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode session-start ...)` — delete
- Lines 49-56: Session context file reading — delete
- Lines 61-166: maintain parsing — keep
- Lines 168-191: Output assembly — simplify

- [ ] **Step 2: Rewrite session-start.sh**

Remove `surface` call, session context reading, and `SURFACE_OUTPUT`/`SESSION_CONTEXT` assembly. Keep maintain. Add `/recall` notification. Simplified output:

```bash
# Build output
ADDITIONAL_CTX="$MIDTURN_NOTE"
if [[ -n "$TRIAGE_CTX" ]]; then
    ADDITIONAL_CTX="${ADDITIONAL_CTX}
${TRIAGE_CTX}"
fi

SYSTEM_MSG="[engram] Say /recall to load context from previous sessions, or /recall <query> to search session history."
if [[ -n "$DIRECTIVE" ]]; then
    SYSTEM_MSG="${DIRECTIVE}
${SYSTEM_MSG}"
fi

jq -n \
    --arg sys "$SYSTEM_MSG" \
    --arg add "$ADDITIONAL_CTX" \
    '{systemMessage: $sys, additionalContext: $add}'
```

- [ ] **Step 3: Run `targ check-full` and fix**

- [ ] **Step 4: Commit**

```
refactor(hooks): remove session-start memory surfacing and context injection

Session-start now only runs maintain (triage) and shows /recall notification.
Memory surfacing continues via PreToolUse/PostToolUse hooks as conversation
develops. Explicit /recall skill replaces automatic context loading.
```

---

### Task 3: Simplify stop hook

Remove `--context-path` from flush invocation.

**Files:**
- Modify: `hooks/stop.sh`

- [ ] **Step 1: Remove context-path from stop.sh**

Remove lines 42-44 (PROJECT_SLUG, CONTEXT_DIR, mkdir) and line 50 (`--context-path` flag). The flush command no longer accepts this flag.

- [ ] **Step 2: Commit**

```
refactor(hooks): remove --context-path from stop hook flush call
```

---

### Task 4: Build `recall` CLI subcommand — session finder + reader

New internal package for session transcript discovery and reading. DI for filesystem access.

**Files:**
- Create: `internal/recall/recall.go` — `SessionFinder` (globs for transcripts, sorts by mtime), `TranscriptReader` (reads + strips using existing `Strip()`)
- Create: `internal/recall/recall_test.go`

- [ ] **Step 1: Write failing test for SessionFinder**

Test that `SessionFinder.Find` returns transcript paths sorted by mtime descending.

```go
func TestSessionFinder_SortsByMtimeDescending(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fs := &fakeFS{
		entries: []fileEntry{
			{path: "/projects/slug/old.jsonl", mtime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
			{path: "/projects/slug/new.jsonl", mtime: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
			{path: "/projects/slug/mid.jsonl", mtime: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)},
		},
	}

	finder := recall.NewSessionFinder(fs)
	paths, err := finder.Find("/projects/slug")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	g.Expect(paths).To(Equal([]string{
		"/projects/slug/new.jsonl",
		"/projects/slug/mid.jsonl",
		"/projects/slug/old.jsonl",
	}))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
targ test
```

Expected: FAIL — `recall` package doesn't exist.

- [ ] **Step 3: Implement SessionFinder**

```go
package recall

import "sort"

type DirLister interface {
	ListJSONL(dir string) ([]FileEntry, error)
}

type FileEntry struct {
	Path  string
	Mtime time.Time
}

type SessionFinder struct {
	lister DirLister
}

func NewSessionFinder(lister DirLister) *SessionFinder {
	return &SessionFinder{lister: lister}
}

func (f *SessionFinder) Find(projectDir string) ([]string, error) {
	entries, err := f.lister.ListJSONL(projectDir)
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Mtime.After(entries[j].Mtime)
	})

	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.Path)
	}
	return paths, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

- [ ] **Step 5: Write failing test for TranscriptReader**

Test that `TranscriptReader.Read` reads a file, applies `Strip()`, and respects byte budget.

```go
func TestTranscriptReader_RespectsStripAndBudget(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	content := strings.Join([]string{
		`{"role":"user","content":"hello"}`,
		`{"role":"toolResult","content":"big data"}`,
		`{"role":"assistant","content":"hi there"}`,
	}, "\n") + "\n"

	fr := &fakeFileReader{contents: map[string][]byte{"/t.jsonl": []byte(content)}}
	reader := recall.NewTranscriptReader(fr)

	stripped, bytesRead, err := reader.Read("/t.jsonl", 100000)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	// toolResult should be stripped
	g.Expect(stripped).To(HaveLen(2))
	g.Expect(bytesRead).To(BeNumerically(">", 0))
}
```

- [ ] **Step 6: Implement TranscriptReader**

Uses the existing `context.FileReader` interface and `context.Strip()` function.

- [ ] **Step 7: Run tests, verify pass**

- [ ] **Step 8: Commit**

```
feat(recall): add SessionFinder and TranscriptReader

SessionFinder globs for .jsonl transcript files and sorts by mtime descending.
TranscriptReader reads and strips transcripts using existing Strip() logic
with a configurable byte budget.
```

---

### Task 5: Build `recall` CLI subcommand — Haiku summarizer + search

Two modes: summarize (mode A) and extract-relevant (mode B).

**Files:**
- Create: `internal/recall/summarize.go` — `Summarizer` struct with `Summarize` and `ExtractRelevant` methods
- Create: `internal/recall/summarize_test.go`

- [ ] **Step 1: Write failing test for Summarize (mode A)**

```go
func TestSummarizer_SummarizesStrippedContent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	client := &fakeHaikuClient{result: "Working on recall feature, added session finder"}
	s := recall.NewSummarizer(client)

	summary, err := s.Summarize(context.Background(), "stripped transcript content here")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	g.Expect(summary).To(Equal("Working on recall feature, added session finder"))
	g.Expect(client.systemPrompt).To(ContainSubstring("Summarize these session transcripts"))
	g.Expect(client.systemPrompt).To(ContainSubstring("No emoji"))
}
```

- [ ] **Step 2: Implement Summarizer.Summarize**

Uses a `HaikuCaller` interface (same as `makeAnthropicCaller` signature). System prompt from spec section 2.

- [ ] **Step 3: Run test, verify pass**

- [ ] **Step 4: Write failing test for ExtractRelevant (mode B)**

```go
func TestSummarizer_ExtractsRelevantContent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	client := &fakeHaikuClient{result: "Decided to use TF-IDF for keyword matching"}
	s := recall.NewSummarizer(client)

	extracted, err := s.ExtractRelevant(context.Background(), "full transcript", "keyword matching")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	g.Expect(extracted).To(ContainSubstring("TF-IDF"))
	g.Expect(client.userPrompt).To(ContainSubstring("keyword matching"))
}
```

- [ ] **Step 5: Implement Summarizer.ExtractRelevant**

- [ ] **Step 6: Run tests, verify pass**

- [ ] **Step 7: Commit**

```
feat(recall): add Haiku summarizer with summarize and extract-relevant modes

Mode A (no query): summarizes stripped transcript with structured prompt
prioritizing current status, open questions, decisions, and failed attempts.
Mode B (with query): extracts only content relevant to a specific query.
```

---

### Task 6: Build `recall` orchestrator + CLI wiring

Compose finder + reader + summarizer into the full recall pipeline. Wire into CLI.

**Files:**
- Create: `internal/recall/orchestrate.go` — `Orchestrator` composing the full pipeline
- Create: `internal/recall/orchestrate_test.go`
- Modify: `internal/cli/cli.go` — add `case "recall"` and `runRecall` function
- Modify: `internal/cli/targets.go` — add `RecallArgs` struct and `RecallFlags` func, add to `BuildTargets`

- [ ] **Step 1: Write failing test for Orchestrator mode A**

Test that orchestrator finds sessions, reads+strips, summarizes, surfaces memories, and returns output. Use DI interfaces for all collaborators.

```go
func TestOrchestrator_ModeA_Summarizes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	orch := recall.NewOrchestrator(
		&fakeFinder{paths: []string{"/p/a.jsonl", "/p/b.jsonl"}},
		&fakeReader{content: "stripped content", bytesRead: 500},
		&fakeSummarizer{result: "Working on recall feature"},
		&fakeSurfacer{memories: "memory: use targ for builds"},
	)

	result, err := orch.Recall(context.Background(), "/p", "")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	g.Expect(result.Summary).To(Equal("Working on recall feature"))
	g.Expect(result.Memories).To(ContainSubstring("use targ"))
}
```

- [ ] **Step 2: Implement Orchestrator with DI interfaces**

All collaborators injected as interfaces per CLAUDE.md:

```go
type Finder interface {
	Find(projectDir string) ([]string, error)
}

type Reader interface {
	Read(path string, budgetBytes int) (string, int, error)
}

type HaikuSummarizer interface {
	Summarize(ctx context.Context, content string) (string, error)
	ExtractRelevant(ctx context.Context, content, query string) (string, error)
}

type MemorySurfacer interface {
	Surface(query string) (string, error)
}

type RecallResult struct {
	Summary  string `json:"summary"`
	Memories string `json:"memories"`
}

type Orchestrator struct {
	finder     Finder
	reader     Reader
	summarizer HaikuSummarizer
	surfacer   MemorySurfacer
}
```

Mode A: find sessions → read+strip (50KB budget) → summarize → surface memories using summary as query → return `RecallResult`.

- [ ] **Step 3: Run test, verify pass**

- [ ] **Step 4: Write failing test for Orchestrator mode B (with query)**

Test iterative extraction: for each session, extract relevant content, accumulate until 1500 bytes. Surface memories using the original query.

- [ ] **Step 5: Implement mode B in Orchestrator**

Iterative loop: for each session, read+strip, extract relevant, accumulate until 1500 bytes. Surface memories using query (not extracted content).

- [ ] **Step 6: Run tests, verify pass**

- [ ] **Step 7: Wire into CLI**

Add to `cli.go`:
```go
case "recall":
    return runRecall(subArgs, stdout)
```

Add `runRecall` function that parses `--data-dir`, `--project-slug`, `--query` flags, constructs real I/O adapters (including a `MemorySurfacer` adapter that shells out to `engram surface --mode prompt --message <query> --data-dir <dir> --format json`), creates orchestrator, calls `Recall`, and outputs JSON.

Also update `errUsage` string: remove `context-update`, add `recall`.

- [ ] **Step 8: Add RecallArgs to targets.go**

```go
type RecallArgs struct {
	DataDir     string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ProjectSlug string `targ:"flag,name=project-slug,desc=project directory slug"`
	Query       string `targ:"flag,name=query,desc=search query (omit for summary mode)"`
}
```

Add to `BuildTargets` and implement `RecallFlags`.

- [ ] **Step 9: Run `targ check-full` and fix**

- [ ] **Step 10: Commit**

```
feat(recall): wire orchestrator and CLI subcommand

engram recall --data-dir ... --project-slug ... [--query "..."]
Mode A (no query): summarizes recent session transcripts.
Mode B (with query): searches session history for relevant content.
Outputs JSON for skill consumption.
```

---

### Task 7: Create `/recall` skill

The skill file that users invoke as `/recall` or `/recall <query>`.

**Files:**
- Create: `skills/recall/SKILL.md`

- [ ] **Step 1: Create skill directory and file**

```bash
mkdir -p skills/recall
```

- [ ] **Step 2: Write SKILL.md**

```markdown
---
name: recall
description: |
  Use when the user says "/recall", "what was I working on", "load previous
  context", "search session history", or wants to resume work from a previous
  session. Reads recent session transcripts, summarizes or searches them,
  and surfaces relevant memories.
---

# Session Recall

Load context from previous sessions in this project.

## Usage

- `/recall` — summarize recent session history ("where was I?")
- `/recall <query>` — search session history for specific content

## How It Works

Invokes the engram recall subcommand to read Claude Code session transcripts
from `~/.claude/projects/`, strip noise, and use Haiku to produce a focused
summary or extract relevant content.

## Execution

Run the following command, replacing the arguments:

\```bash
PROJECT_SLUG="$(echo "$PWD" | tr '/' '-')"
~/.claude/engram/bin/engram recall \
  --data-dir ~/.claude/engram/data \
  --project-slug "$PROJECT_SLUG" \
  [--query "<user's query if provided>"]
\```

Parse the JSON output and present:
1. The summary/extracted content to the user
2. Any surfaced memories as additional context

If the command fails or returns empty, inform the user that no previous
session data was found for this project.
```

- [ ] **Step 3: Commit**

```
feat(skills): add /recall skill for session history retrieval

Two modes: /recall for recent session summary, /recall <query> for
searching session history. Replaces automatic session-context injection
with explicit, user-controlled context loading.
```

---

### Task 8: Integration test + cleanup

End-to-end test, README update, and delete stale session-context files.

**Files:**
- Modify: `internal/cli/cli_test.go` — add recall integration test
- Modify: `README.md` — update session lifecycle docs

- [ ] **Step 1: Write integration test for recall CLI**

Test that `engram recall --data-dir ... --project-slug ...` with a mock Haiku server returns valid JSON output. Pattern follows `TestT337_FlushIntegration_PipelineOrdering`.

```go
func TestRecall_Integration_ModeA(t *testing.T) {
	g := NewWithT(t)

	// Create a temp dir shaped like ~/.claude/projects/{slug}/
	projectDir := t.TempDir()

	// Write a fake transcript file
	transcriptPath := filepath.Join(projectDir, "test-session.jsonl")
	g.Expect(os.WriteFile(
		transcriptPath,
		[]byte(`{"role":"user","content":"help with recall"}`+"\n"+
			`{"role":"assistant","content":"sure, working on it"}`+"\n"),
		0o644,
	)).To(Succeed())

	// Mock Haiku server
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(
				`{"content":[{"type":"text","text":"Working on recall feature"}]}`,
			))
		},
	))
	defer server.Close()

	original := cli.AnthropicAPIURL
	cli.AnthropicAPIURL = server.URL
	defer func() { cli.AnthropicAPIURL = original }()

	t.Setenv("ENGRAM_API_TOKEN", "fake-token")

	dataDir := t.TempDir()
	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "recall",
			"--data-dir", dataDir,
			"--project-slug", filepath.Base(projectDir),
		},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	// Verify JSON output contains summary
	g.Expect(stdout.String()).To(ContainSubstring("summary"))
}
```

- [ ] **Step 2: Run test, verify it fails (recall subcommand not yet wired)**

Note: This test will pass because Task 6 already added the wiring. Verify it passes.

- [ ] **Step 3: Verify test passes**

- [ ] **Step 3: Update README**

Update session lifecycle section to reflect new architecture: session-start is lightweight, `/recall` is opt-in, per-turn flush only runs `learn`.

- [ ] **Step 4: Run `targ check-full` — everything green**

- [ ] **Step 5: Commit**

```
test(recall): add integration test and update README

Verifies end-to-end recall pipeline through CLI. Updates session lifecycle
documentation to reflect opt-in /recall replacing automatic context injection.
```

- [ ] **Step 6: Delete stale session-context.md files**

```bash
find ~/.claude/engram/data/projects -name "session-context.md" -delete
```

This is a local cleanup, not committed.
