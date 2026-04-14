# Engram v3: Memory Skills Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace engram's multi-agent coordination system with four focused memory skills (/recall, /prepare, /learn, /remember) backed by a lean Go binary with Haiku-based relevance matching.

**Architecture:** Skills orchestrate calls to the `engram` binary, which handles transcript parsing, memory CRUD, and Haiku-based search. The binary reads session transcripts (read-only) and reads/writes memory TOML files. BM25, tracking counters, project scoping, and the entire multi-agent chat protocol are removed.

**Tech Stack:** Go 1.25+, BurntSushi/toml, targ CLI framework, Anthropic Haiku API, Claude Code skills (markdown).

**Spec:** `docs/superpowers/specs/2026-04-14-engram-v3-memory-skills-design.md`

---

### Task 1: Delete multi-agent infrastructure

**Files:**
- Delete: `skills/engram-agent/SKILL.md`
- Delete: `skills/engram-lead/SKILL.md`
- Delete: `skills/engram-tmux-lead/SKILL.md`
- Delete: `skills/engram-up/SKILL.md`
- Delete: `skills/engram-down/SKILL.md`
- Delete: `skills/use-engram-chat-as/SKILL.md`
- Delete: `hooks/agent-stop.sh`
- Delete: `hooks/subagent-stop.sh`
- Delete: `hooks/user-prompt.sh`
- Delete: `internal/bm25/` (all files)
- Delete: `internal/tokenize/` (all files)
- Delete: `internal/surface/` (all files)
- Delete: `internal/policy/` (all files)
- Delete: `internal/cli/recallsurfacer.go`
- Delete: `internal/cli/recallsurfacer_test.go`
- Modify: `internal/cli/cli.go` (remove surface/bm25/policy imports, remove buildRecallSurfacer, remove recordSurfacing)
- Modify: `internal/recall/orchestrate.go` (remove MemorySurfacer dependency)
- Modify: `hooks/hooks.json` (remove agent-stop, subagent-stop, user-prompt hooks if referenced)

- [ ] **Step 1: Delete skill directories**

```bash
rm -rf skills/engram-agent skills/engram-lead skills/engram-tmux-lead skills/engram-up skills/engram-down skills/use-engram-chat-as
```

- [ ] **Step 2: Delete hook scripts**

```bash
rm hooks/agent-stop.sh hooks/subagent-stop.sh hooks/user-prompt.sh
```

- [ ] **Step 3: Delete internal packages**

```bash
rm -rf internal/bm25 internal/tokenize internal/surface internal/policy
```

- [ ] **Step 4: Delete recallsurfacer files**

```bash
rm internal/cli/recallsurfacer.go internal/cli/recallsurfacer_test.go
```

- [ ] **Step 5: Remove surface dependency from cli.go**

In `internal/cli/cli.go`, remove the `"engram/internal/surface"` import and all code that depends on it:
- Remove `buildRecallSurfacer` function entirely
- Remove `recordSurfacing` function entirely
- Remove `surfaceRunnerAdapter` type and its methods (if present)
- Remove `defaultModifier` global (used only by recordSurfacing)
- Remove the `"engram/internal/tomlwriter"` import (only used by defaultModifier — will be re-added later)
- In `runRecall`, remove the `memorySurfacer` lines (buildRecallSurfacer call and passing surfacer to NewOrchestrator)

- [ ] **Step 6: Remove MemorySurfacer from recall orchestrator**

In `internal/recall/orchestrate.go`:
- Remove `MemorySurfacer` interface
- Remove `surfacer` field from `Orchestrator` struct
- Remove `surfacer` param from `NewOrchestrator`
- Remove `surfaceMemories` method
- In `recallModeA` and `recallModeB`, remove `memories := o.surfaceMemories(...)` calls — set `Memories: ""` or just omit

In `internal/cli/cli.go`, update `runRecall` to call `NewOrchestrator(finder, reader, summarizer)` (3 args instead of 4).

- [ ] **Step 7: Verify build and tests**

Run: `targ check-full`
Expected: All remaining tests pass. Deleted packages' tests are gone. Build succeeds.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "refactor: delete multi-agent infrastructure

Remove 6 multi-agent skills (engram-agent, engram-lead, engram-tmux-lead,
engram-up, engram-down, use-engram-chat-as), 3 hook scripts, and 4 internal
packages (bm25, tokenize, surface, policy). Remove MemorySurfacer from
recall orchestrator.

AI-Used: [claude]"
```

---

### Task 2: Simplify memory types

**Files:**
- Modify: `internal/memory/record.go`
- Modify: `internal/memory/memory.go`
- Modify: `internal/memory/record_test.go`
- Modify: `internal/memory/memory_test.go`
- Modify: `internal/memory/maintenance_test.go` (if it references removed fields)
- Modify: `internal/cli/show.go` (remove effectiveness/relevance/scope rendering)
- Modify: `internal/cli/show_test.go`
- Modify: `internal/cli/export_test.go` (if it exports removed functions)

- [ ] **Step 1: Write failing test for simplified MemoryRecord**

In `internal/memory/record_test.go`, add a test that verifies the new schema:

```go
func TestMemoryRecord_V2_HasNoLegacyFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := memory.MemoryRecord{
		SchemaVersion: 2,
		Type:          "feedback",
		Source:        "human",
		Situation:     "When running tests",
		Content: memory.ContentFields{
			Behavior: "running go test",
			Impact:   "misses coverage",
			Action:   "use targ test",
		},
		CreatedAt: "2026-04-14T10:00:00Z",
		UpdatedAt: "2026-04-14T10:00:00Z",
	}

	// Encode to TOML and verify no legacy fields appear
	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(record)
	g.Expect(err).NotTo(HaveOccurred())

	encoded := buf.String()
	g.Expect(encoded).NotTo(ContainSubstring("core"))
	g.Expect(encoded).NotTo(ContainSubstring("project_scoped"))
	g.Expect(encoded).NotTo(ContainSubstring("project_slug"))
	g.Expect(encoded).NotTo(ContainSubstring("surfaced_count"))
	g.Expect(encoded).NotTo(ContainSubstring("followed_count"))
	g.Expect(encoded).NotTo(ContainSubstring("not_followed_count"))
	g.Expect(encoded).NotTo(ContainSubstring("irrelevant_count"))
	g.Expect(encoded).NotTo(ContainSubstring("missed_count"))
	g.Expect(encoded).NotTo(ContainSubstring("initial_confidence"))
	g.Expect(encoded).NotTo(ContainSubstring("pending_evaluations"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — MemoryRecord still has legacy fields that encode to TOML.

- [ ] **Step 3: Simplify MemoryRecord in record.go**

Replace the `MemoryRecord` struct in `internal/memory/record.go`:

```go
type MemoryRecord struct {
	SchemaVersion int    `toml:"schema_version,omitempty"`
	Type          string `toml:"type"`
	Source        string `toml:"source"`
	Situation     string `toml:"situation"`

	Content ContentFields `toml:"content"`

	CreatedAt string `toml:"created_at"`
	UpdatedAt string `toml:"updated_at"`
}
```

Remove `PendingEvaluation` type entirely. Remove `TotalEvaluations` method.

- [ ] **Step 4: Simplify Stored in memory.go**

Replace the `Stored` struct:

```go
type Stored struct {
	Type      string
	Situation string
	Source    string
	Content   ContentFields
	UpdatedAt time.Time
	FilePath  string
}
```

Remove `SearchText` method (BM25 is gone). Remove `TotalEvaluations` method. Remove `appendContentFields` and `appendNonEmpty` helpers.

Update `ToStored` in record.go to match:

```go
func (r *MemoryRecord) ToStored(filePath string) *Stored {
	updatedAt, parseErr := time.Parse(time.RFC3339, r.UpdatedAt)
	if parseErr != nil && r.UpdatedAt != "" {
		fmt.Fprintf(os.Stderr, "engram: memory: parsing updated_at %q for %s: %v\n",
			r.UpdatedAt, filePath, parseErr)
	}

	return &Stored{
		Type:      r.Type,
		Situation: r.Situation,
		Source:    r.Source,
		Content:   r.Content,
		UpdatedAt: updatedAt,
		FilePath:  filePath,
	}
}
```

- [ ] **Step 5: Simplify show.go rendering**

In `internal/cli/show.go`:
- Remove `effectivenessPercent` function
- Remove `renderMemoryMeta` function (it renders project scope, effectiveness, relevance — all removed fields)
- Update `renderMemory` to just call `renderMemoryContent` (drop `renderMemoryMeta` call)
- Add created/updated timestamps to `renderMemoryContent` instead:

```go
func renderMemoryContent(writer io.Writer, mem *memory.MemoryRecord) {
	if mem.Type != "" {
		_, _ = fmt.Fprintf(writer, "Type: %s\n", mem.Type)
	}

	if mem.Type == "fact" {
		renderFactContent(writer, mem)
	} else {
		renderFeedbackContent(writer, mem)
	}

	if mem.Source != "" {
		_, _ = fmt.Fprintf(writer, "Source: %s\n", mem.Source)
	}

	if mem.CreatedAt != "" {
		_, _ = fmt.Fprintf(writer, "Created: %s\n", mem.CreatedAt)
	}

	if mem.UpdatedAt != "" {
		_, _ = fmt.Fprintf(writer, "Updated: %s\n", mem.UpdatedAt)
	}
}
```

- [ ] **Step 6: Fix all existing tests**

Update any tests that reference removed fields (`Core`, `ProjectScoped`, `ProjectSlug`, `SurfacedCount`, `FollowedCount`, `NotFollowedCount`, `IrrelevantCount`, `MissedCount`, `InitialConfidence`, `PendingEvaluations`). Remove tests for removed functions (`SearchText`, `TotalEvaluations`, `effectivenessPercent`). Update show tests that check for effectiveness/relevance output.

Remove any export_test.go exports for deleted functions.

- [ ] **Step 7: Run tests**

Run: `targ check-full`
Expected: All tests pass.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "refactor: simplify memory types to v2 schema

Remove core, project_scoped, project_slug, all tracking counters,
initial_confidence, pending_evaluations from MemoryRecord and Stored.
Remove SearchText (BM25 gone), TotalEvaluations, and effectiveness
rendering from show command.

AI-Used: [claude]"
```

---

### Task 3: Tool call extraction in transcript parsing

**Files:**
- Modify: `internal/context/stripconfig.go`
- Create: `internal/context/toolsummary.go`
- Modify: `internal/context/stripconfig_test.go`

- [ ] **Step 1: Write failing test for tool call summary format**

In `internal/context/stripconfig_test.go`:

```go
func TestStripWithConfig_ToolSummaryMode_FormatsToolCalls(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	toolUseLine := buildToolUseLine("Read", map[string]any{
		"file_path": "/src/main.go",
		"offset":    0,
		"limit":     100,
	})
	toolResultLine := buildToolResultLine("package main\n\nimport \"fmt\"\n", false)

	cfg := sessionctx.StripConfig{ToolSummaryMode: true}
	result := sessionctx.StripWithConfig([]string{toolUseLine, toolResultLine}, cfg)

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0]).To(HavePrefix("[tool] Read("))
	g.Expect(result[0]).To(ContainSubstring("file_path="))
	g.Expect(result[0]).To(ContainSubstring("exit 0"))
	g.Expect(result[0]).To(ContainSubstring("package main"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `ToolSummaryMode` field doesn't exist on StripConfig.

- [ ] **Step 3: Add ToolSummaryMode to StripConfig**

In `internal/context/stripconfig.go`, add the field:

```go
type StripConfig struct {
	KeepToolCalls    bool
	ToolSummaryMode  bool
	ToolArgsTruncate int
	ToolResultTruncate int
}
```

- [ ] **Step 4: Create toolsummary.go with extraction logic**

Create `internal/context/toolsummary.go`:

```go
package context

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	toolSummaryArgsCap    = 120
	toolSummaryOutputCap  = 120
)

// toolSummaryPair holds a pending tool_use waiting for its tool_result.
type toolSummaryPair struct {
	name string
	args string
}

// formatToolSummaryArgs formats tool input as key=value pairs, truncated.
func formatToolSummaryArgs(input json.RawMessage) string {
	var fields map[string]json.RawMessage

	err := json.Unmarshal(input, &fields)
	if err != nil {
		raw := string(input)
		if len(raw) > toolSummaryArgsCap {
			return raw[:toolSummaryArgsCap]
		}

		return raw
	}

	parts := make([]string, 0, len(fields))

	for key, val := range fields {
		parts = append(parts, fmt.Sprintf("%s=%s", key, string(val)))
	}

	result := strings.Join(parts, ", ")
	if len(result) > toolSummaryArgsCap {
		return result[:toolSummaryArgsCap]
	}

	return result
}

// formatToolSummaryLine produces the final [tool] line from a use+result pair.
func formatToolSummaryLine(name, args string, exitCode int, firstLine string) string {
	if len(firstLine) > toolSummaryOutputCap {
		firstLine = firstLine[:toolSummaryOutputCap]
	}

	return fmt.Sprintf("[tool] %s(%s) → exit %d | %s", name, args, exitCode, firstLine)
}

// firstNonEmptyLine returns the first non-blank line from content.
func firstNonEmptyLine(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}
```

- [ ] **Step 5: Integrate ToolSummaryMode into StripWithConfig**

In `internal/context/stripconfig.go`, update `StripWithConfig`:

```go
func StripWithConfig(lines []string, cfg StripConfig) []string {
	if !cfg.KeepToolCalls && !cfg.ToolSummaryMode {
		return Strip(lines)
	}

	if cfg.ToolSummaryMode {
		return stripWithToolSummary(lines)
	}

	// existing KeepToolCalls logic...
}
```

Add `stripWithToolSummary` function that:
1. Parses each JSONL line
2. For text blocks: emits `USER:` / `ASSISTANT:` lines (same as Strip)
3. For `tool_use` blocks: stores the pending tool call (name + formatted args)
4. For `tool_result` blocks: pairs with pending tool_use, determines exit code (0 for ok, 1 for is_error), extracts first non-empty line of output, emits `[tool]` summary line

```go
func stripWithToolSummary(lines []string) []string {
	result := make([]string, 0, len(lines))
	var pending *toolSummaryPair

	for _, line := range lines {
		if !isKeptType(line) {
			continue
		}

		cleaned := replaceBase64(line)

		var entry jsonlLine

		err := json.Unmarshal([]byte(cleaned), &entry)
		if err != nil {
			continue
		}

		role := normalizeRole(entry)
		if role == "" {
			continue
		}

		raw := entry.Message.Content
		if len(raw) == 0 {
			continue
		}

		// Try plain string first
		var str string
		if json.Unmarshal(raw, &str) == nil {
			if !isSystemReminder(str) {
				rolePrefix := "USER: "
				if role == roleAssistant {
					rolePrefix = "ASSISTANT: "
				}

				result = append(result, truncateContent(rolePrefix+str))
			}

			continue
		}

		// Array of blocks
		var rawBlocks []json.RawMessage

		if json.Unmarshal(raw, &rawBlocks) != nil {
			continue
		}

		rolePrefix := "USER: "
		if role == roleAssistant {
			rolePrefix = "ASSISTANT: "
		}

		for _, block := range rawBlocks {
			var blockType rawBlockType

			if json.Unmarshal(block, &blockType) != nil {
				continue
			}

			switch blockType.Type {
			case "text":
				extracted := extractTextBlock(block, rolePrefix)
				result = append(result, extracted...)
			case "tool_use":
				var use toolUseBlock
				if json.Unmarshal(block, &use) == nil {
					pending = &toolSummaryPair{
						name: use.Name,
						args: formatToolSummaryArgs(use.Input),
					}
				}
			case "tool_result":
				var res toolResultBlock
				if json.Unmarshal(block, &res) == nil && pending != nil {
					exitCode := 0
					if res.IsError {
						exitCode = 1
					}

					firstLine := firstNonEmptyLine(res.Content)
					summary := formatToolSummaryLine(pending.name, pending.args, exitCode, firstLine)
					result = append(result, summary)
					pending = nil
				}
			}
		}
	}

	return result
}
```

- [ ] **Step 6: Write additional tests**

Test cases for `internal/context/stripconfig_test.go`:
- Tool use with error result → exit 1
- Args longer than 120 chars → truncated
- Output longer than 120 chars → truncated
- Multiple tool calls in sequence
- Mixed text and tool calls
- Tool use without matching result (orphaned)

- [ ] **Step 7: Run tests**

Run: `targ check-full`
Expected: All tests pass.

- [ ] **Step 8: Commit**

```bash
git commit -m "feat: add tool call summary mode to transcript parsing

New ToolSummaryMode in StripConfig produces compact one-line summaries:
[tool] ToolName(args) → exit N | first line of output
Args truncated at 120 chars, output at 120 chars.

AI-Used: [claude]"
```

---

### Task 4: `engram list` command

**Files:**
- Create: `internal/cli/list.go`
- Create: `internal/cli/list_test.go`
- Modify: `internal/cli/cli.go` (add "list" case to Run)
- Modify: `internal/cli/targets.go` (add ListArgs and list target)

- [ ] **Step 1: Write failing test**

Create `internal/cli/list_test.go`:

```go
package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestList_OutputsTypeNameSituation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	tomlContent := `schema_version = 2
type = "feedback"
source = "human"
situation = "When running tests in Go projects"

[content]
behavior = "running go test"
impact = "misses coverage"
action = "use targ test"

created_at = "2026-04-14T10:00:00Z"
updated_at = "2026-04-14T10:00:00Z"
`
	g.Expect(os.WriteFile(
		filepath.Join(feedbackDir, "use-targ-for-tests.toml"),
		[]byte(tomlContent), 0o640,
	)).To(Succeed())

	var stdout bytes.Buffer
	err := cli.Run(
		[]string{"engram", "list", "--data-dir", dataDir},
		&stdout, &bytes.Buffer{}, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("feedback"))
	g.Expect(output).To(ContainSubstring("use-targ-for-tests"))
	g.Expect(output).To(ContainSubstring("When running tests in Go projects"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — "list" is not a recognized command.

- [ ] **Step 3: Add ListArgs to targets.go**

In `internal/cli/targets.go`, add:

```go
type ListArgs struct {
	DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}
```

Add to `BuildTargets`:

```go
targ.Targ(func(a ListArgs) { run("list", ListFlags(a)) }).
	Name("list").Description("List all memories with type, name, and situation"),
```

Add `ListFlags`:

```go
func ListFlags(a ListArgs) []string {
	return BuildFlags("--data-dir", a.DataDir)
}
```

- [ ] **Step 4: Create list.go**

Create `internal/cli/list.go`:

```go
package cli

import (
	"fmt"
	"io"

	"engram/internal/memory"
)

func runList(args []string, stdout io.Writer) error {
	fs := newFlagSet("list")

	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("list: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("list: %w", defaultErr)
	}

	lister := memory.NewLister()

	memories, err := lister.ListAllMemories(*dataDir)
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}

	for _, mem := range memories {
		name := memory.NameFromPath(mem.FilePath)
		_, writeErr := fmt.Fprintf(stdout, "%s | %s | %s\n", mem.Type, name, mem.Situation)
		if writeErr != nil {
			return fmt.Errorf("list: %w", writeErr)
		}
	}

	return nil
}
```

- [ ] **Step 5: Add "list" case to cli.go Run()**

In `internal/cli/cli.go`, add to the switch:

```go
case "list":
	return runList(subArgs, stdout)
```

Update `errUsage` to include "list":

```go
errUsage = errors.New("usage: engram <recall|show|list> [flags]")
```

- [ ] **Step 6: Run tests**

Run: `targ check-full`
Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
git commit -m "feat: add engram list command

Lists all memories as compact index entries: type | name | situation.
Used by Haiku for two-phase memory search.

AI-Used: [claude]"
```

---

### Task 5: `engram recall` — integrate tool summaries and per-session memory windowing

**Files:**
- Modify: `internal/recall/recall.go` (TranscriptReader now uses ToolSummaryMode)
- Modify: `internal/recall/orchestrate.go` (add per-session memory windowing to mode A)
- Modify: `internal/recall/recall_test.go`
- Modify: `internal/recall/orchestrate_test.go`
- Modify: `internal/cli/cli.go` (pass memory lister to orchestrator)

- [ ] **Step 1: Write failing test for tool summary in transcript reading**

In `internal/recall/recall_test.go`, add a test that verifies the TranscriptReader produces tool call summaries:

```go
func TestTranscriptReader_IncludesToolSummaries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Build a JSONL transcript with a tool call
	toolUseLine := `{"type":"assistant","message":{"role":"assistant","content":[` +
		`{"type":"tool_use","id":"1","name":"Bash","input":{"command":"targ test"}}` +
		`]}}`
	toolResultLine := `{"type":"user","message":{"role":"user","content":[` +
		`{"type":"tool_result","tool_use_id":"1","content":"PASS\nok engram 0.5s","is_error":false}` +
		`]}}`

	content := toolUseLine + "\n" + toolResultLine + "\n"

	reader := recall.NewTranscriptReader(&fakeFileReader{content: content})
	result, _, err := reader.Read("test.jsonl", 50*1024)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(ContainSubstring("[tool] Bash("))
	g.Expect(result).To(ContainSubstring("exit 0"))
	g.Expect(result).To(ContainSubstring("PASS"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — TranscriptReader still uses `context.Strip` which drops tool calls.

- [ ] **Step 3: Update TranscriptReader to use ToolSummaryMode**

In `internal/recall/recall.go`, update the `Read` method to use `StripWithConfig` with `ToolSummaryMode: true` instead of `Strip`:

```go
func (r *TranscriptReader) Read(path string, budgetBytes int) (string, int, error) {
	data, err := r.reader.Read(path)
	if err != nil {
		return "", 0, fmt.Errorf("reading transcript: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	cfg := sessionctx.StripConfig{ToolSummaryMode: true}
	stripped := sessionctx.StripWithConfig(lines, cfg)

	// accumulate from tail backwards up to budget
	// (existing logic)
}
```

Add the import for `sessionctx "engram/internal/context"` if not already present.

- [ ] **Step 4: Write failing test for per-session memory windowing**

In `internal/recall/orchestrate_test.go`, add a test that verifies mode A includes memories created during the session window:

```go
func TestOrchestrator_ModeA_IncludesTimeWindowedMemories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Session transcript from 2026-04-14 09:00-10:00
	finder := &fakeFinder{paths: []string{"/sessions/s1.jsonl"}}
	reader := &fakeReader{
		content: "USER: working on tests\nASSISTANT: ok",
		mtime:   time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC),
	}

	// Memory created at 09:30 (within window)
	memoryLister := &fakeMemoryLister{
		memories: []*memory.Stored{{
			Type:      "feedback",
			Situation: "When testing",
			Source:    "human",
			UpdatedAt: time.Date(2026, 4, 14, 9, 30, 0, 0, time.UTC),
			FilePath:  "/memories/test-mem.toml",
		}},
	}

	orch := recall.NewOrchestrator(finder, reader, nil, memoryLister)
	result, err := orch.Recall(context.Background(), "/project", "")

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Summary).To(ContainSubstring("working on tests"))
	g.Expect(result.Memories).To(ContainSubstring("When testing"))
}
```

- [ ] **Step 5: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — Orchestrator doesn't accept a MemoryLister, and mode A doesn't include time-windowed memories.

- [ ] **Step 6: Add MemoryLister interface and per-session windowing**

In `internal/recall/orchestrate.go`:

Add a new interface:

```go
// MemoryLister lists stored memories for time-windowed surfacing.
type MemoryLister interface {
	ListAllMemories(dataDir string) ([]*memory.Stored, error)
}
```

Update `Orchestrator` struct and constructor to accept an optional MemoryLister and dataDir:

```go
type Orchestrator struct {
	finder       Finder
	reader       Reader
	summarizer   SummarizerI
	memoryLister MemoryLister
	dataDir      string
}

func NewOrchestrator(
	finder Finder,
	reader Reader,
	summarizer SummarizerI,
	memoryLister MemoryLister,
	dataDir string,
) *Orchestrator {
	return &Orchestrator{
		finder:       finder,
		reader:       reader,
		summarizer:   summarizer,
		memoryLister: memoryLister,
		dataDir:      dataDir,
	}
}
```

Update `Reader` interface to also return the file's mtime (needed to determine session time window):

```go
type Reader interface {
	Read(path string, budgetBytes int) (string, int, time.Time, error)
}
```

The `time.Time` return is the session file's modification time, used as the session end time. The session start time can be derived from the earliest timestamp in the transcript, or approximated as `mtime - duration` based on byte count.

In `recallModeA`, after reading transcripts, query the memory lister for memories whose `UpdatedAt` falls within each session's time window, and append them to the result.

- [ ] **Step 7: Update cli.go to pass MemoryLister**

In `internal/cli/cli.go`, update `runRecall` to create a `memory.NewLister()` and pass it plus `dataDir` to the orchestrator.

- [ ] **Step 8: Fix all existing tests**

Update `orchestrate_test.go` fakes and callers to match new `NewOrchestrator` signature and `Reader` interface.

- [ ] **Step 9: Run tests**

Run: `targ check-full`
Expected: All tests pass.

- [ ] **Step 10: Commit**

```bash
git commit -m "feat: add tool summaries and per-session memory windowing to recall

TranscriptReader now uses ToolSummaryMode for compact tool call output.
Mode A (no query) includes memories created/updated within each session's
time window.

AI-Used: [claude]"
```

---

### Task 6: `engram recall` — Haiku-based memory matching with `--memories-only`

**Files:**
- Modify: `internal/recall/orchestrate.go` (add mode C: memories-only with Haiku matching)
- Modify: `internal/recall/orchestrate_test.go`
- Modify: `internal/cli/cli.go` (add --memories-only and --limit flags)
- Modify: `internal/cli/targets.go` (add flags to RecallArgs)
- Modify: `internal/cli/targets_test.go`

- [ ] **Step 1: Write failing test for --memories-only mode**

In `internal/recall/orchestrate_test.go`:

```go
func TestOrchestrator_MemoriesOnly_ReturnsHaikuFilteredMemories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryLister := &fakeMemoryLister{
		memories: []*memory.Stored{
			{Type: "feedback", Situation: "When running tests", Source: "human",
				FilePath: "/mem/test.toml",
				UpdatedAt: time.Now()},
			{Type: "fact", Situation: "When writing docs", Source: "agent",
				FilePath: "/mem/docs.toml",
				UpdatedAt: time.Now()},
		},
	}

	summarizer := &fakeSummarizer{
		response: "feedback | test | When running tests",
	}

	orch := recall.NewOrchestrator(nil, nil, summarizer, memoryLister, "/data")
	result, err := orch.RecallMemoriesOnly(
		context.Background(), "how to run tests", 10,
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Memories).To(ContainSubstring("When running tests"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `RecallMemoriesOnly` method doesn't exist.

- [ ] **Step 3: Implement Haiku-based memory matching**

In `internal/recall/orchestrate.go`, add two-phase memory search:

Phase 1 — build index from all memory situations (via `engram list` internally):
```go
func (o *Orchestrator) buildMemoryIndex() (string, error) {
	memories, err := o.memoryLister.ListAllMemories(o.dataDir)
	if err != nil {
		return "", fmt.Errorf("listing memories: %w", err)
	}

	var builder strings.Builder
	for _, mem := range memories {
		name := memory.NameFromPath(mem.FilePath)
		fmt.Fprintf(&builder, "%s | %s | %s\n", mem.Type, name, mem.Situation)
	}

	return builder.String(), nil
}
```

Phase 2 — ask Haiku which are relevant:
```go
func (o *Orchestrator) RecallMemoriesOnly(
	ctx context.Context,
	query string,
	limit int,
) (*Result, error) {
	index, err := o.buildMemoryIndex()
	if err != nil {
		return nil, err
	}

	if index == "" {
		return &Result{}, nil
	}

	// Ask Haiku to select relevant memory names from the index
	systemPrompt := "You are a memory retrieval assistant. Given a query and a list of memories " +
		"(format: type | name | situation), return ONLY the names of memories relevant to the query, " +
		"one per line. Return nothing if none are relevant. Maximum " + fmt.Sprintf("%d", limit) + " names."
	userPrompt := fmt.Sprintf("Query: %s\n\nMemories:\n%s", query, index)

	response, haikuErr := o.summarizer.ExtractRelevant(ctx, userPrompt, systemPrompt)
	// ... parse response, load full content of matched memories, format output
}
```

Note: This repurposes `SummarizerI.ExtractRelevant` — the systemPrompt/userPrompt semantics need to be flexible enough. If the current interface is too rigid, add a new interface method or a separate Haiku caller.

Load matched memories fully via `memory.Lister.ListAll` + filter by name, then render content. Sort by human-sourced first, then agent, then by most recent.

- [ ] **Step 4: Add --memories-only and --limit flags**

In `internal/cli/targets.go`, update `RecallArgs`:

```go
type RecallArgs struct {
	DataDir      string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	ProjectSlug  string `targ:"flag,name=project-slug,desc=project directory slug"`
	Query        string `targ:"flag,name=query,desc=search query (omit for summary mode)"`
	MemoriesOnly bool   `targ:"flag,name=memories-only,desc=search only memory files"`
	Limit        int    `targ:"flag,name=limit,desc=max memories to return (default 10)"`
}
```

Update `RecallFlags` to include the new flags.

In `runRecall` in `cli.go`, check for `memoriesOnly` flag and route to `orch.RecallMemoriesOnly(...)` when set.

- [ ] **Step 5: Also integrate Haiku memory matching into query mode (mode B)**

In mode B (query + transcripts), after extracting transcript content, also run the two-phase memory search and merge results. This replaces the old BM25-based surfacing.

- [ ] **Step 6: Run tests**

Run: `targ check-full`
Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
git commit -m "feat: add Haiku-based memory matching and --memories-only to recall

Two-phase search: scan memory situations via index, ask Haiku for
relevance, load full content of matches. Replaces BM25 surfacing.
New --memories-only flag for memory-only search, --limit flag for
capping results.

AI-Used: [claude]"
```

---

### Task 7: `engram learn feedback` and `engram learn fact` commands

**Files:**
- Create: `internal/cli/learn.go`
- Create: `internal/cli/learn_test.go`
- Modify: `internal/cli/cli.go` (add "learn" case)
- Modify: `internal/cli/targets.go` (add LearnFeedbackArgs, LearnFactArgs)
- Modify: `internal/tomlwriter/tomlwriter.go` (update Write to use new layout: feedback/ or facts/)

- [ ] **Step 1: Write failing test for learn feedback**

Create `internal/cli/learn_test.go`:

```go
package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestLearnFeedback_WritesMemoryFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout bytes.Buffer
	err := cli.Run(
		[]string{"engram", "learn", "feedback",
			"--situation", "When running tests",
			"--behavior", "running go test directly",
			"--impact", "misses coverage thresholds",
			"--action", "use targ test instead",
			"--source", "human",
			"--data-dir", dataDir,
			"--no-dup-check",
		},
		&stdout, &bytes.Buffer{}, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify file was created in feedback dir
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	entries, dirErr := os.ReadDir(feedbackDir)
	g.Expect(dirErr).NotTo(HaveOccurred())
	if dirErr != nil {
		return
	}
	g.Expect(entries).To(HaveLen(1))

	// Verify stdout contains the name
	output := stdout.String()
	g.Expect(output).To(ContainSubstring("when-running-tests"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — "learn" not recognized.

- [ ] **Step 3: Update tomlwriter.Write to support new layout**

In `internal/tomlwriter/tomlwriter.go`, update `Write` to accept the memory type and write to the correct directory (`memory/feedback/` or `memory/facts/`):

```go
func (w *Writer) Write(record *memory.MemoryRecord, slug, dataDir string) (string, error) {
	var targetDir string
	switch record.Type {
	case "fact":
		targetDir = memory.FactsDir(dataDir)
	default:
		targetDir = memory.FeedbackDir(dataDir)
	}

	mkdirErr := w.mkdirAll(targetDir, memoriesDirPerm)
	if mkdirErr != nil {
		return "", fmt.Errorf("tomlwriter: create dir: %w", mkdirErr)
	}

	// ... rest is same (slugify, availablePath, timestamps, AtomicWrite)
	// but use targetDir instead of memoriesDir
}
```

- [ ] **Step 4: Create learn.go with feedback subcommand**

Create `internal/cli/learn.go`:

```go
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"engram/internal/memory"
	"engram/internal/tomlwriter"
)

var (
	errLearnMissingSubcommand = errors.New("learn: subcommand required (feedback|fact)")
	errLearnInvalidSource     = errors.New("learn: --source must be 'human' or 'agent'")
)

func runLearn(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errLearnMissingSubcommand
	}

	subcmd := args[0]
	subArgs := args[1:]

	switch subcmd {
	case "feedback":
		return runLearnFeedback(subArgs, stdout)
	case "fact":
		return runLearnFact(subArgs, stdout)
	default:
		return fmt.Errorf("learn: unknown subcommand: %s", subcmd)
	}
}

func runLearnFeedback(args []string, stdout io.Writer) error {
	fs := newFlagSet("learn feedback")

	dataDir := fs.String("data-dir", "", "path to data directory")
	situation := fs.String("situation", "", "when this applies")
	behavior := fs.String("behavior", "", "what was done")
	impact := fs.String("impact", "", "what happened as a result")
	action := fs.String("action", "", "what to do instead")
	source := fs.String("source", "", "human or agent")
	noDupCheck := fs.Bool("no-dup-check", false, "skip duplicate detection")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("learn feedback: %w", parseErr)
	}

	if err := applyDataDirDefault(dataDir); err != nil {
		return fmt.Errorf("learn feedback: %w", err)
	}

	if err := validateSource(*source); err != nil {
		return err
	}

	record := &memory.MemoryRecord{
		SchemaVersion: 2,
		Type:          "feedback",
		Source:        *source,
		Situation:     *situation,
		Content: memory.ContentFields{
			Behavior: *behavior,
			Impact:   *impact,
			Action:   *action,
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if !*noDupCheck {
		conflict, err := checkForConflicts(record, *dataDir, stdout)
		if err != nil {
			return fmt.Errorf("learn feedback: %w", err)
		}
		if conflict {
			return nil // conflicts were printed to stdout
		}
	}

	writer := tomlwriter.New()
	slug := tomlwriter.Slugify(*situation)
	path, writeErr := writer.Write(record, slug, *dataDir)
	if writeErr != nil {
		return fmt.Errorf("learn feedback: %w", writeErr)
	}

	name := memory.NameFromPath(path)
	_, _ = fmt.Fprintf(stdout, "CREATED: %s\n", name)

	return nil
}

func validateSource(source string) error {
	if source != "human" && source != "agent" {
		return errLearnInvalidSource
	}

	return nil
}
```

- [ ] **Step 5: Implement conflict checking with Haiku**

In `internal/cli/learn.go`, add `checkForConflicts`:

```go
func checkForConflicts(
	record *memory.MemoryRecord,
	dataDir string,
	stdout io.Writer,
) (bool, error) {
	ctx, cancel := signalContext()
	defer cancel()

	token := resolveToken(ctx)
	if token == "" {
		// No API token — skip dedup check, write directly
		return false, nil
	}

	lister := memory.NewLister()
	memories, err := lister.ListAllMemories(dataDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("listing memories: %w", err)
	}

	if len(memories) == 0 {
		return false, nil
	}

	// Build index
	var indexBuilder strings.Builder
	for _, mem := range memories {
		name := memory.NameFromPath(mem.FilePath)
		fmt.Fprintf(&indexBuilder, "%s | %s | %s\n", mem.Type, name, mem.Situation)
	}

	// Ask Haiku to check for duplicates/contradictions
	systemPrompt := `You are checking if a new memory duplicates or contradicts existing memories.
Given a new memory and a list of existing memories (type | name | situation),
identify any that are DUPLICATES (same lesson) or CONTRADICTIONS (conflicting advice for the same situation).
Return one line per match: "DUPLICATE: <name>" or "CONTRADICTION: <name>".
Return "NONE" if no matches.`

	var newMemDesc string
	if record.Type == "feedback" {
		newMemDesc = fmt.Sprintf("Type: feedback\nSituation: %s\nBehavior: %s\nImpact: %s\nAction: %s",
			record.Situation, record.Content.Behavior, record.Content.Impact, record.Content.Action)
	} else {
		newMemDesc = fmt.Sprintf("Type: fact\nSituation: %s\nSubject: %s\nPredicate: %s\nObject: %s",
			record.Situation, record.Content.Subject, record.Content.Predicate, record.Content.Object)
	}

	userPrompt := fmt.Sprintf("New memory:\n%s\n\nExisting memories:\n%s", newMemDesc, indexBuilder.String())

	caller := makeAnthropicCaller(token)
	response, callErr := caller(ctx, anthropic.HaikuModel, systemPrompt, userPrompt)
	if callErr != nil {
		// Haiku unavailable — skip check, write directly
		return false, nil
	}

	if strings.TrimSpace(response) == "NONE" {
		return false, nil
	}

	// Parse matches and load full content for output
	// ... parse DUPLICATE:/CONTRADICTION: lines, load matched memories, print details
	_, _ = fmt.Fprint(stdout, response+"\n")

	// Load and print full content of matched memories
	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		var matchType, matchName string
		if strings.HasPrefix(line, "DUPLICATE: ") {
			matchType = "DUPLICATE"
			matchName = strings.TrimPrefix(line, "DUPLICATE: ")
		} else if strings.HasPrefix(line, "CONTRADICTION: ") {
			matchType = "CONTRADICTION"
			matchName = strings.TrimPrefix(line, "CONTRADICTION: ")
		} else {
			continue
		}

		memPath := memory.ResolveMemoryPath(dataDir, matchName, fileExists)
		mem, loadErr := loadMemoryTOML(memPath)
		if loadErr != nil {
			continue
		}

		_, _ = fmt.Fprintf(stdout, "\n%s: %s\n", matchType, matchName)
		renderMemoryContent(stdout, mem)
	}

	return true, nil
}
```

- [ ] **Step 6: Implement learn fact subcommand**

In `internal/cli/learn.go`, add `runLearnFact` — same pattern as `runLearnFeedback` but with subject/predicate/object flags:

```go
func runLearnFact(args []string, stdout io.Writer) error {
	fs := newFlagSet("learn fact")

	dataDir := fs.String("data-dir", "", "path to data directory")
	situation := fs.String("situation", "", "context where this fact is relevant")
	subject := fs.String("subject", "", "subject of the fact")
	predicate := fs.String("predicate", "", "relationship")
	object := fs.String("object", "", "object of the fact")
	source := fs.String("source", "", "human or agent")
	noDupCheck := fs.Bool("no-dup-check", false, "skip duplicate detection")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("learn fact: %w", parseErr)
	}

	if err := applyDataDirDefault(dataDir); err != nil {
		return fmt.Errorf("learn fact: %w", err)
	}

	if err := validateSource(*source); err != nil {
		return err
	}

	record := &memory.MemoryRecord{
		SchemaVersion: 2,
		Type:          "fact",
		Source:        *source,
		Situation:     *situation,
		Content: memory.ContentFields{
			Subject:   *subject,
			Predicate: *predicate,
			Object:    *object,
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if !*noDupCheck {
		conflict, err := checkForConflicts(record, *dataDir, stdout)
		if err != nil {
			return fmt.Errorf("learn fact: %w", err)
		}
		if conflict {
			return nil
		}
	}

	writer := tomlwriter.New()
	slug := tomlwriter.Slugify(*situation)
	path, writeErr := writer.Write(record, slug, *dataDir)
	if writeErr != nil {
		return fmt.Errorf("learn fact: %w", writeErr)
	}

	name := memory.NameFromPath(path)
	_, _ = fmt.Fprintf(stdout, "CREATED: %s\n", name)

	return nil
}
```

- [ ] **Step 7: Add "learn" case to cli.go**

In `internal/cli/cli.go`:

```go
case "learn":
	return runLearn(subArgs, stdout)
```

Update `errUsage`.

Add learn targets to `BuildTargets` in `targets.go`.

- [ ] **Step 8: Export slugify from tomlwriter**

Rename `slugify` to `Slugify` in `internal/tomlwriter/tomlwriter.go` so learn.go can generate slugs from situation text.

- [ ] **Step 9: Write test for learn fact**

Add test in `learn_test.go` for fact creation, similar to feedback test.

- [ ] **Step 10: Write test for duplicate detection**

Add test that pre-creates a memory, then tries to learn a duplicate and verifies the `DUPLICATE:` output.

- [ ] **Step 11: Run tests**

Run: `targ check-full`
Expected: All tests pass.

- [ ] **Step 12: Commit**

```bash
git commit -m "feat: add engram learn feedback and learn fact commands

Creates memory files with Haiku-based duplicate/contradiction detection.
Returns DUPLICATE or CONTRADICTION with full content when conflicts found.
--no-dup-check flag to bypass detection.

AI-Used: [claude]"
```

---

### Task 8: `engram update` command

**Files:**
- Create: `internal/cli/update.go`
- Create: `internal/cli/update_test.go`
- Modify: `internal/cli/cli.go` (add "update" case)
- Modify: `internal/cli/targets.go` (add UpdateArgs)

- [ ] **Step 1: Write failing test**

Create `internal/cli/update_test.go`:

```go
package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/memory"
)

func TestUpdate_ModifiesSituationField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	original := memory.MemoryRecord{
		SchemaVersion: 2,
		Type:          "feedback",
		Source:        "human",
		Situation:     "narrow situation",
		Content: memory.ContentFields{
			Behavior: "bad thing",
			Impact:   "bad result",
			Action:   "good thing",
		},
		CreatedAt: "2026-04-14T10:00:00Z",
		UpdatedAt: "2026-04-14T10:00:00Z",
	}

	memPath := filepath.Join(feedbackDir, "test-mem.toml")
	f, createErr := os.Create(memPath)
	g.Expect(createErr).NotTo(HaveOccurred())
	if createErr != nil {
		return
	}
	g.Expect(toml.NewEncoder(f).Encode(original)).To(Succeed())
	f.Close()

	var stdout bytes.Buffer
	err := cli.Run(
		[]string{"engram", "update",
			"--name", "test-mem",
			"--data-dir", dataDir,
			"--situation", "broader situation including more cases",
		},
		&stdout, &bytes.Buffer{}, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	// Re-read and verify
	var updated memory.MemoryRecord
	_, decErr := toml.DecodeFile(memPath, &updated)
	g.Expect(decErr).NotTo(HaveOccurred())
	g.Expect(updated.Situation).To(Equal("broader situation including more cases"))
	g.Expect(updated.Content.Behavior).To(Equal("bad thing")) // unchanged
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — "update" not recognized.

- [ ] **Step 3: Create update.go**

Create `internal/cli/update.go`:

```go
package cli

import (
	"fmt"
	"io"
	"time"

	"engram/internal/memory"
	"engram/internal/tomlwriter"
)

func runUpdate(args []string, stdout io.Writer) error {
	fs := newFlagSet("update")

	dataDir := fs.String("data-dir", "", "path to data directory")
	name := fs.String("name", "", "memory name (slug)")
	situation := fs.String("situation", "", "new situation text")
	behavior := fs.String("behavior", "", "new behavior text")
	impact := fs.String("impact", "", "new impact text")
	action := fs.String("action", "", "new action text")
	subject := fs.String("subject", "", "new subject text")
	predicate := fs.String("predicate", "", "new predicate text")
	object := fs.String("object", "", "new object text")
	source := fs.String("source", "", "new source (human|agent)")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("update: %w", parseErr)
	}

	if err := applyDataDirDefault(dataDir); err != nil {
		return fmt.Errorf("update: %w", err)
	}

	if *name == "" {
		return fmt.Errorf("update: --name required")
	}

	memPath := memory.ResolveMemoryPath(*dataDir, *name, fileExists)

	record, loadErr := loadMemoryTOML(memPath)
	if loadErr != nil {
		return fmt.Errorf("update: %w", loadErr)
	}

	// Apply only provided fields
	if *situation != "" {
		record.Situation = *situation
	}
	if *behavior != "" {
		record.Content.Behavior = *behavior
	}
	if *impact != "" {
		record.Content.Impact = *impact
	}
	if *action != "" {
		record.Content.Action = *action
	}
	if *subject != "" {
		record.Content.Subject = *subject
	}
	if *predicate != "" {
		record.Content.Predicate = *predicate
	}
	if *object != "" {
		record.Content.Object = *object
	}
	if *source != "" {
		if err := validateSource(*source); err != nil {
			return err
		}
		record.Source = *source
	}

	record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	writer := tomlwriter.New()
	writeErr := writer.AtomicWrite(memPath, record)
	if writeErr != nil {
		return fmt.Errorf("update: %w", writeErr)
	}

	_, _ = fmt.Fprintf(stdout, "UPDATED: %s\n", *name)

	return nil
}
```

- [ ] **Step 4: Add "update" case to cli.go**

```go
case "update":
	return runUpdate(subArgs, stdout)
```

Add `UpdateArgs` to `targets.go` and register the target in `BuildTargets`.

- [ ] **Step 5: Run tests**

Run: `targ check-full`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git commit -m "feat: add engram update command

Updates individual fields of an existing memory by name. Only provided
fields are changed, timestamps updated automatically.

AI-Used: [claude]"
```

---

### Task 9: Create `/recall` skill

**Files:**
- Create: `skills/recall/SKILL.md` (replace existing)

- [ ] **Step 1: Write skill file**

Create `skills/recall/SKILL.md`:

```markdown
---
name: recall
description: >
  Use when the user says "/recall", "what was I working on", "load previous
  context", "search session history", or wants to resume work from a previous
  session. Reads recent session transcripts and surfaces relevant memories.
---

You are loading context from previous sessions.

## Flow

### Self-query (agent-internal — do not show to user)

Run this command and internalize the results. These guide YOUR behavior, not the user's:

\`\`\`bash
engram recall --memories-only --query "when to call /prepare or /learn in the current situation"
\`\`\`

Follow any guidance from the results (e.g., "call /prepare before starting X").

### No-args mode (user said `/recall` with no query)

Run:

\`\`\`bash
engram recall
\`\`\`

Summarize the output for the user:
- What was being discussed and decided
- What work was done (filter mundane tool calls — share what's relevant)
- What memories were active during that period

### Query mode (user said `/recall <query>`)

Run:

\`\`\`bash
engram recall --query "<the user's query>"
\`\`\`

Present the filtered results to the user.
```

- [ ] **Step 2: Commit**

```bash
git commit -m "feat: add /recall skill

Replaces old recall skill with v3 version including self-query for
operational guidance and clear agent/user output separation.

AI-Used: [claude]"
```

---

### Task 10: Create `/prepare` skill

**Files:**
- Create: `skills/prepare/SKILL.md`

- [ ] **Step 1: Write skill file**

Create `skills/prepare/SKILL.md`:

```markdown
---
name: prepare
description: >
  Use before starting new work, switching tasks, beginning a feature, changing
  direction, or tackling an issue. Should be called before implementation,
  debugging, or any significant new effort. Identifies the current situation
  and recalls relevant context and memories.
---

You are mentally preparing for upcoming work by loading relevant context.

## Flow

### Step 1: Self-query (agent-internal — do not show to user)

Run this command and internalize the results:

\`\`\`bash
engram recall --memories-only --query "how to prepare for <brief situation summary>, and when to call /prepare or /learn"
\`\`\`

Follow any operational guidance from the results. These instruct YOUR behavior.

### Step 2: Analyze the situation

Based on the current conversation, identify:
- What the user is asking for
- What kind of work is about to happen
- What domain/technology/codebase area is involved

### Step 3: Make targeted recall queries

Run 2-3 targeted queries based on your analysis:

\`\`\`bash
engram recall --query "<specific topic 1>"
engram recall --query "<specific topic 2>"
\`\`\`

Choose queries that would surface relevant prior work, decisions, patterns, and pitfalls.

### Step 4: Present briefing to user

Summarize the relevant context and memories from the domain queries for the user's awareness.

### Step 5: Internalize for your own use

Treat the recalled memories as instructions and important context that should guide your behavior during the upcoming work. Feedback memories tell you what to do and not do. Fact memories provide knowledge. Act on them.
```

- [ ] **Step 2: Commit**

```bash
git commit -m "feat: add /prepare skill

Proactive context loading before new work. Self-queries for operational
guidance, makes targeted recall queries, presents briefing to user.

AI-Used: [claude]"
```

---

### Task 11: Create `/remember` skill

**Files:**
- Create: `skills/remember/SKILL.md`

- [ ] **Step 1: Write skill file**

Create `skills/remember/SKILL.md`:

```markdown
---
name: remember
description: >
  Use when the user says "remember this", "remember that", "don't forget",
  "save this for later", or /remember. Captures explicit knowledge as
  feedback or fact memories with user approval.
---

The user wants to explicitly save something to memory.

## Flow

### Step 1: Self-query (agent-internal — do not show to user)

\`\`\`bash
engram recall --memories-only --query "when to call /prepare or /learn in the current situation"
\`\`\`

Internalize any guidance.

### Step 2: Analyze and classify

Determine what the user wants to remember. Classify as:
- **Feedback** (behavioral): situation → behavior → impact → action
- **Fact** (knowledge): situation → subject → predicate → object
- Could be **multiple** memories (e.g., "DI means Dependency Injection, not Do It" = two facts)

### Step 3: Draft and present to user

For each memory, draft all required fields and present to the user for approval:

**Feedback example:**
- Situation: "When running tests in Go projects"
- Behavior: "Running go test directly"
- Impact: "Misses coverage thresholds and lint checks"
- Action: "Use targ test instead"

**Fact example:**
- Situation: "When reading abbreviations in code"
- Subject: "DI"
- Predicate: "means"
- Object: "Dependency Injection"

Ask the user to approve or edit the fields.

### Step 4: Save approved memories

For each approved memory, run:

\`\`\`bash
# Feedback:
engram learn feedback --situation "..." --behavior "..." --impact "..." --action "..." --source human

# Fact:
engram learn fact --situation "..." --subject "..." --predicate "..." --object "..." --source human
\`\`\`

### Step 5: Handle results

- **`CREATED: <name>`** — Confirm to user.
- **`DUPLICATE: <name>`** — The system already knows this. Trigger diagnostic (see below).
- **`CONTRADICTION: <name>`** — Present the conflict. Ask user: update existing, replace it, or keep both (use `--no-dup-check`)?

### Step 6: Duplicate diagnostic

When a duplicate is found, the system already knew this but failed to use it. Analyze:

1. **Was there a `/recall` or `/prepare` call this session that should have surfaced this?**
   - **Yes, but queries missed it:** Suggest additional queries that would have found it. Draft these as behavioral feedback memories with situations matching the self-query format (e.g., "how to prepare for <topic>") so future self-queries will find them. Present to user for approval.
   - **Yes, but memory wording too narrow:** Suggest a rewrite of the existing memory's situation field. Use `engram update --name <name> --situation "broader situation"` after user approval.

2. **No relevant `/recall` or `/prepare` call:**
   - Suggest a behavioral memory: "When <situation>, call /prepare before proceeding." Draft with a situation field matching the self-query format. Present to user for approval.
```

- [ ] **Step 2: Commit**

```bash
git commit -m "feat: add /remember skill

Explicit memory capture with SBIA/fact classification, user approval,
and self-correcting duplicate diagnostics.

AI-Used: [claude]"
```

---

### Task 12: Create `/learn` skill

**Files:**
- Create: `skills/learn/SKILL.md`

- [ ] **Step 1: Write skill file**

Create `skills/learn/SKILL.md`:

```markdown
---
name: learn
description: >
  Use after completing a task, finishing work, changing direction, or when the
  user says "review what we learned" or /learn. Should be called after
  implementation, after resolving a bug, after completing a plan step.
  Reviews the recent session for learnable moments.
---

You are reviewing the recent session for things worth remembering.

## Flow

### Step 1: Self-query (agent-internal — do not show to user)

\`\`\`bash
engram recall --memories-only --query "how to review sessions for learnable moments, and when to call /prepare or /learn"
\`\`\`

Internalize any guidance.

### Step 2: Load session context

\`\`\`bash
engram recall
\`\`\`

Review the output for learnable moments:
- **User corrections** — the user told you to do something differently
- **Failed approaches** — something was tried and didn't work
- **Discovered facts** — new knowledge about the codebase, tools, or domain
- **Patterns** — recurring behaviors that should be codified

### Step 3: Draft findings

For each learnable moment, draft a memory:
- Corrections/failures → feedback (SBIA)
- Knowledge/patterns → fact (situation + subject/predicate/object)

Present all findings to the user for approval. Each should have all fields filled.

### Step 4: Save approved memories

For each approved memory:

\`\`\`bash
# Feedback:
engram learn feedback --situation "..." --behavior "..." --impact "..." --action "..." --source agent

# Fact:
engram learn fact --situation "..." --subject "..." --predicate "..." --object "..." --source agent
\`\`\`

Note: source is `agent` because these are agent-identified learnings, not explicit user instructions.

### Step 5: Handle conflicts

Same as `/remember` — handle DUPLICATE and CONTRADICTION responses identically, including the duplicate diagnostic for self-correction.

### Step 6: Internalize operational guidance

Follow any operational guidance from the Step 1 self-query.
```

- [ ] **Step 2: Commit**

```bash
git commit -m "feat: add /learn skill

Reflective session review for learnable moments. Identifies corrections,
failures, facts, and patterns. Same self-correcting duplicate diagnostics
as /remember.

AI-Used: [claude]"
```

---

### Task 13: Update plugin manifest and hooks

**Files:**
- Modify: `.claude-plugin/plugin.json`
- Modify: `hooks/hooks.json`
- Modify: `hooks/session-start.sh` (if it references deleted commands)

- [ ] **Step 1: Read current plugin.json**

Read `.claude-plugin/plugin.json` to understand current structure.

- [ ] **Step 2: Update plugin.json**

Ensure it references only the 4 remaining skills: recall, prepare, learn, remember. Remove references to deleted skills.

- [ ] **Step 3: Update hooks.json**

Ensure only SessionStart hook remains. Remove any references to agent-stop, subagent-stop, user-prompt hooks.

- [ ] **Step 4: Update session-start.sh**

Remove any references to chat commands or agent commands. Keep the binary rebuild logic and the skill announcement. Update the announcement to mention `/recall`, `/prepare`, `/learn`, `/remember`.

- [ ] **Step 5: Run tests**

Run: `targ check-full`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git commit -m "chore: update plugin manifest and hooks for v3

Remove references to deleted multi-agent skills and hooks. Update
session-start announcement for new skill set.

AI-Used: [claude]"
```

---

### Task 14: Memory migration

**Files:**
- Create: `internal/cli/migrate.go`
- Create: `internal/cli/migrate_test.go`
- Modify: `internal/cli/cli.go` (add "migrate" case)

- [ ] **Step 1: Write failing test**

Create `internal/cli/migrate_test.go`:

```go
package cli_test

func TestMigrate_StripsLegacyFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	// Write a v1 memory with all legacy fields
	v1Content := `schema_version = 1
type = "feedback"
source = "user correction, 2026-04-02"
core = true
situation = "When running tests"
project_scoped = true
project_slug = "engram"
surfaced_count = 5
followed_count = 3
not_followed_count = 1
irrelevant_count = 1
missed_count = 0
initial_confidence = 0.9

[content]
behavior = "running go test"
impact = "misses coverage"
action = "use targ test"

created_at = "2026-04-02T10:00:00Z"
updated_at = "2026-04-02T10:00:00Z"
`
	memPath := filepath.Join(feedbackDir, "test-mem.toml")
	g.Expect(os.WriteFile(memPath, []byte(v1Content), 0o640)).To(Succeed())

	var stdout bytes.Buffer
	err := cli.Run(
		[]string{"engram", "migrate", "--data-dir", dataDir},
		&stdout, &bytes.Buffer{}, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	// Re-read and verify
	data, readErr := os.ReadFile(memPath)
	g.Expect(readErr).NotTo(HaveOccurred())
	if readErr != nil {
		return
	}
	content := string(data)
	g.Expect(content).To(ContainSubstring("schema_version = 2"))
	g.Expect(content).To(ContainSubstring(`source = "human"`))
	g.Expect(content).NotTo(ContainSubstring("core"))
	g.Expect(content).NotTo(ContainSubstring("project_scoped"))
	g.Expect(content).NotTo(ContainSubstring("surfaced_count"))
	g.Expect(content).NotTo(ContainSubstring("initial_confidence"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — "migrate" not recognized.

- [ ] **Step 3: Create migrate.go**

Create `internal/cli/migrate.go`:

```go
package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"engram/internal/memory"
	"engram/internal/tomlwriter"
)

func runMigrate(args []string, stdout io.Writer) error {
	fs := newFlagSet("migrate")

	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("migrate: %w", parseErr)
	}

	if err := applyDataDirDefault(dataDir); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	lister := memory.NewLister()
	writer := tomlwriter.New()

	var migrated, skipped int

	for _, dir := range []string{
		memory.FeedbackDir(*dataDir),
		memory.FactsDir(*dataDir),
		memory.MemoriesDir(*dataDir),
	} {
		records, err := lister.ListAll(dir)
		if err != nil {
			continue // directory may not exist
		}

		for _, sr := range records {
			record := sr.Record

			if record.SchemaVersion >= 2 {
				skipped++
				continue
			}

			// Bump schema version
			record.SchemaVersion = 2

			// Normalize source
			record.Source = normalizeSource(record.Source)

			// Ensure situation exists
			if record.Situation == "" {
				record.Situation = inferSituation(&record)
			}

			// Strip legacy fields by writing only v2 fields
			v2 := memory.MemoryRecord{
				SchemaVersion: 2,
				Type:          record.Type,
				Source:        record.Source,
				Situation:     record.Situation,
				Content:       record.Content,
				CreatedAt:     record.CreatedAt,
				UpdatedAt:     record.UpdatedAt,
			}

			writeErr := writer.AtomicWrite(sr.Path, v2)
			if writeErr != nil {
				_, _ = fmt.Fprintf(stdout, "ERROR: %s: %v\n", sr.Path, writeErr)
				continue
			}

			migrated++
		}
	}

	_, _ = fmt.Fprintf(stdout, "Migrated %d memories, skipped %d (already v2)\n", migrated, skipped)

	return nil
}

func normalizeSource(source string) string {
	lower := strings.ToLower(source)
	if strings.Contains(lower, "user") || strings.Contains(lower, "human") {
		return "human"
	}

	return "agent"
}

func inferSituation(record *memory.MemoryRecord) string {
	if record.Type == "fact" {
		if record.Content.Subject != "" {
			return fmt.Sprintf("When working with %s", record.Content.Subject)
		}

		return "General knowledge"
	}

	// Feedback: derive from behavior
	if record.Content.Behavior != "" {
		return fmt.Sprintf("When %s", strings.ToLower(record.Content.Behavior))
	}

	return "General development"
}
```

- [ ] **Step 4: Add "migrate" case to cli.go**

```go
case "migrate":
	return runMigrate(subArgs, stdout)
```

- [ ] **Step 5: Write test for source normalization**

Test that "user correction, 2026-04-02" → "human", "agent observation" → "agent", empty → "agent".

- [ ] **Step 6: Write test for situation inference**

Test that missing situation gets inferred from content fields.

- [ ] **Step 7: Run tests**

Run: `targ check-full`
Expected: All tests pass.

- [ ] **Step 8: Commit**

```bash
git commit -m "feat: add engram migrate command

One-time migration for v1→v2 memory format. Strips legacy fields (core,
project_scoped, tracking counters, confidence), normalizes source to
human/agent, infers missing situation fields from content.

AI-Used: [claude]"
```

---

### Task 15: Update README and archive stale docs

**Files:**
- Modify: `README.md`
- Move: stale docs to `archive/`

- [ ] **Step 1: Read current README**

Read `README.md` to understand current content.

- [ ] **Step 2: Rewrite README**

Update to reflect v3:
- Description: self-correcting memory for LLM agents
- Four skills: /recall, /prepare, /learn, /remember
- Core loop: prepare → work → learn
- Binary commands: recall, list, learn, update, show, migrate
- Memory format: v2 TOML (type, situation, source, content, timestamps)
- Installation instructions
- Data directory layout (no chat files, no policy.toml)
- Remove all references to multi-agent coordination, effectiveness quadrants, adaptation, /adapt, /memory-triage

- [ ] **Step 3: Archive stale docs**

Move docs that reference deleted systems to `archive/`:
- Any docs referencing chat protocol, agent spawning, tmux orchestration, BM25, surface pipeline, policy
- Keep docs that are still relevant (memory format basics, installation)

- [ ] **Step 4: Commit**

```bash
git commit -m "docs: update README and archive stale docs for v3

Rewrite README for new four-skill architecture. Archive docs referencing
deleted multi-agent infrastructure.

AI-Used: [claude]"
```

---

### Task 16: Final verification

- [ ] **Step 1: Full test suite**

Run: `targ check-full`
Expected: All tests pass, no lint errors, coverage thresholds met.

- [ ] **Step 2: Build binary**

Run: `targ build`
Expected: Binary compiles successfully.

- [ ] **Step 3: Smoke test commands**

```bash
./engram list --data-dir ~/.local/share/engram
./engram recall --query "test" --data-dir ~/.local/share/engram
./engram show --name <pick-a-memory>
./engram learn feedback --situation "test" --behavior "test" --impact "test" --action "test" --source agent --no-dup-check --data-dir /tmp/engram-test
./engram update --name <created-name> --situation "updated test" --data-dir /tmp/engram-test
```

- [ ] **Step 4: Run migration on real data**

```bash
./engram migrate --data-dir ~/.local/share/engram
```

Inspect a few migrated files to verify v2 format.

- [ ] **Step 5: Verify skills load**

Start a new Claude Code session in the engram project. Verify `/recall`, `/prepare`, `/learn`, `/remember` appear in skill list. Run `/recall` to verify end-to-end.
