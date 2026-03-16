# Unified Tool Surfacing Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unify all tool-hook memory surfacing through `surface --mode tool` with new `--tool-output` and `--tool-errored` flags, replacing the separate `remind` command entirely.

**Architecture:** Extend the existing `surface` tool mode to accept tool output/error context, enriching BM25 queries. Delete the `remind` package and all code only it used. Update all three tool hooks (PreToolUse, PostToolUse, PostToolUseFailure) to call `surface --mode tool` with the appropriate flags.

**Tech Stack:** Go, bash, BM25 text scoring

---

## File Structure

**Create:**
- (none — all changes are modifications or deletions)

**Modify:**
- `internal/surface/surface.go` — add `ToolOutput`/`ToolErrored` to Options, enrich BM25 query, adjust floors on failure
- `internal/cli/cli.go` — add `--tool-output`/`--tool-errored` flags to `runSurface`, delete `runRemind` and remind-only adapters
- `internal/cli/targets.go` — add fields to `SurfaceArgs`, delete `RemindArgs`/`RemindFlags`
- `internal/cli/export_test.go` — delete remind-only exports
- `internal/cli/cli_test.go` — delete remind tests
- `internal/cli/adapters_test.go` — delete 9 remind-only tests (see Task 3)
- `internal/cli/targets_test.go` — remove "remind" from subcommand list
- `hooks/post-tool-use.sh` — replace `remind` call with `surface --mode tool --tool-output`
- `hooks/post-tool-use-failure.sh` — add `surface --mode tool --tool-errored` call, append memories to static advisory
- `hooks/pre-tool-use.sh` — no changes needed (already calls surface)

**Delete:**
- `internal/remind/remind.go`
- `internal/remind/remind_test.go`

---

## Chunk 1: Delete `remind` package and all remind-only code

### Task 1: Delete `internal/remind/` package

**Files:**
- Delete: `internal/remind/remind.go`
- Delete: `internal/remind/remind_test.go`

- [ ] **Step 1: Delete the remind package files**

```bash
rm internal/remind/remind.go internal/remind/remind_test.go
rmdir internal/remind
```

- [ ] **Step 2: Verify no other imports remain**

Run: `grep -r '"engram/internal/remind"' internal/ --include='*.go'`
Expected: Only `internal/cli/cli.go` (which we'll fix next)

### Task 2: Remove remind wiring from CLI

**Files:**
- Modify: `internal/cli/cli.go:35` — remove `remind` import
- Modify: `internal/cli/cli.go:141-142` — remove `case "remind"`
- Modify: `internal/cli/cli.go:484` — remove `errRemindMissingFlags`
- Modify: `internal/cli/cli.go:491-494` — remove "remind" from usage string
- Modify: `internal/cli/cli.go:631-635` — remove `noopTranscriptReader` type
- Modify: `internal/cli/cli.go:691-713` — remove `osMemoryLoader` type
- Modify: `internal/cli/cli.go:757-775` — remove `osRemindConfigReader` type
- Modify: `internal/cli/cli.go:1066-1115` — remove `parseRemindersToml` function
- Modify: `internal/cli/cli.go:1528-1569` — remove `runRemind` function

- [ ] **Step 1: Remove the `remind` import**

In `internal/cli/cli.go`, remove the import line for `"engram/internal/remind"`.

- [ ] **Step 2: Remove `case "remind"` from the command dispatch switch**

Delete lines 141-142:
```go
	case "remind":
		return runRemind(subArgs, stdout)
```

- [ ] **Step 3: Remove `errRemindMissingFlags`**

Delete:
```go
	errRemindMissingFlags   = errors.New("remind: --data-dir required")
```

- [ ] **Step 4: Remove "remind" from usage error string**

Change:
```go
	errUsage = errors.New(
		"usage: engram <audit|correct|surface|learn|evaluate" +
			"|review|maintain|remind|instruct" +
			"|context-update> [flags]",
	)
```
To:
```go
	errUsage = errors.New(
		"usage: engram <audit|correct|surface|learn|evaluate" +
			"|review|maintain|instruct" +
			"|context-update> [flags]",
	)
```

- [ ] **Step 5: Remove `noopTranscriptReader` type (lines 631-635)**

Delete the type and its method.

- [ ] **Step 6: Remove `osMemoryLoader` type (lines 691-713)**

Delete the type and its `LoadPrinciple` method.

- [ ] **Step 7: Remove `osRemindConfigReader` type (lines 757-775)**

Delete the type and its `ReadConfig` method.

- [ ] **Step 8: Remove `parseRemindersToml` function (lines 1066-1115)**

Delete the entire function.

- [ ] **Step 9: Remove `runRemind` function (lines 1528-1569)**

Delete the entire function.

- [ ] **Step 10: Run `targ check-full` to verify compilation and find any remaining references**

Run: `targ check-full`
Expected: Compilation errors in test files only (export_test.go, adapters_test.go, cli_test.go, targets_test.go) — those are cleaned up in the next task.

### Task 3: Clean up remind-only test code

**Files:**
- Modify: `internal/cli/export_test.go:18` — remove `ExportParseRemindersToml`
- Modify: `internal/cli/export_test.go:52-57` — remove `ExportNewNoopTranscriptReader`
- Modify: `internal/cli/export_test.go:67-70` — remove `ExportNewOsMemoryLoader`
- Modify: `internal/cli/export_test.go:77-80` — remove `ExportNewOsRemindConfigReader`
- Modify: `internal/cli/adapters_test.go` — remove all remind-only tests:
  - `TestNoopTranscriptReader_ReadRecent` (line 162)
  - `TestOsMemoryLoader_LoadPrinciple_Found` (line 210)
  - `TestOsMemoryLoader_LoadPrinciple_NotFound` (line 240)
  - `TestOsRemindConfigReader_ReadConfig_Missing` (line 288)
  - `TestOsRemindConfigReader_ReadConfig_WithFile` (line 305)
  - `TestParseRemindersToml_EmptyInput` (line 370)
  - `TestParseRemindersToml_InstructionsWithoutSection` (line 385)
  - `TestParseRemindersToml_NoEqualsSign` (line 403)
  - `TestParseRemindersToml_ValidInput` (line 421)
- Modify: `internal/cli/cli_test.go` — remove `TestRunRemind_EmptyDataDir`, `TestRunRemind_FlagParseError`, `TestRunRemind_MissingFlags`
- Modify: `internal/cli/targets_test.go:189` — remove "remind" from subcommand list
- Modify: `internal/cli/targets.go:97-101` — remove `RemindArgs` struct
- Modify: `internal/cli/targets.go:184-185` — remove remind targ registration
- Modify: `internal/cli/targets.go:274-277` — remove `RemindFlags` function

- [ ] **Step 1: Remove remind exports from `export_test.go`**

Remove `ExportParseRemindersToml` from the var block, and remove the three factory functions: `ExportNewNoopTranscriptReader`, `ExportNewOsMemoryLoader`, `ExportNewOsRemindConfigReader`.

- [ ] **Step 2: Remove all remind-only tests from `adapters_test.go`**

Delete these 9 test functions:
- `TestNoopTranscriptReader_ReadRecent`
- `TestOsMemoryLoader_LoadPrinciple_Found`
- `TestOsMemoryLoader_LoadPrinciple_NotFound`
- `TestOsRemindConfigReader_ReadConfig_Missing`
- `TestOsRemindConfigReader_ReadConfig_WithFile`
- `TestParseRemindersToml_EmptyInput`
- `TestParseRemindersToml_InstructionsWithoutSection`
- `TestParseRemindersToml_NoEqualsSign`
- `TestParseRemindersToml_ValidInput`

- [ ] **Step 3: Remove remind tests from `cli_test.go`**

Delete `TestRunRemind_EmptyDataDir`, `TestRunRemind_FlagParseError`, and `TestRunRemind_MissingFlags`.

- [ ] **Step 4: Remove remind from `targets.go`**

Delete `RemindArgs` struct, the targ registration line, and `RemindFlags` function.

- [ ] **Step 5: Remove "remind" from `targets_test.go` subcommand list**

Remove the "remind" entry from the expected subcommand list.

- [ ] **Step 6: Run `targ check-full`**

Run: `targ check-full`
Expected: All tests pass, no compilation errors.

- [ ] **Step 7: Commit**

```bash
git add -A internal/remind/ internal/cli/
git commit -m "refactor: delete remind command and all remind-only code

The remind command's functionality (file-path-based proactive
reminders on PostToolUse) will be replaced by extending surface
--mode tool with --tool-output support, providing richer BM25
matching across all tool hooks.

Deletes: internal/remind/ package, runRemind, osRemindConfigReader,
osMemoryLoader, noopTranscriptReader, parseRemindersToml, RemindArgs,
RemindFlags, and all associated tests.

AI-Used: [claude]"
```

---

## Chunk 2: Extend `surface --mode tool` with `--tool-output` and `--tool-errored`

### Task 4: Add `ToolOutput` and `ToolErrored` to Options and CLI flags

**Files:**
- Modify: `internal/surface/surface.go:109-118` — add fields to `Options`
- Modify: `internal/cli/cli.go` in `runSurface` (~line 1571) — add `--tool-output` and `--tool-errored` flags
- Modify: `internal/cli/targets.go` in `SurfaceArgs` (line 110) — add fields
- Modify: `internal/cli/targets.go` in `SurfaceFlags` (line 293) — add flag builders

- [ ] **Step 1: Write failing test — `ToolOutput` enriches BM25 query**

In an appropriate test file (likely `internal/surface/surface_test.go`), add a test that creates a memory with anti-pattern text matching error output (e.g., "dirty working tree"), sets `ToolInput` to something generic (e.g., `{"command": "git commit"}`), and sets `ToolOutput` to `"error: dirty working tree"`. Assert that the memory is surfaced (it would NOT match on `ToolInput` alone).

```go
func TestToolOutput_EnrichesBM25Query(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Memory whose anti-pattern mentions "dirty working tree"
	target := &memory.Stored{
		FilePath:    "/data/memories/stash-before-ops.toml",
		Title:       "stash before operations",
		Principle:   "Always stash or commit before running commands that require clean tree",
		AntiPattern: "dirty working tree uncommitted changes",
		Keywords:    []string{"git", "stash", "working tree"},
	}

	// Filler memories for IDF contrast (need enough to make BM25 selective)
	memories := []*memory.Stored{target}
	for i := range 20 {
		memories = append(memories, &memory.Stored{
			FilePath:    fmt.Sprintf("/data/memories/filler-%d.toml", i),
			AntiPattern: fmt.Sprintf("unrelated pattern %d about something else", i),
		})
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	var buf bytes.Buffer
	err := s.Run(context.Background(), &buf, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: `{"command": "git commit -m 'fix'"}`,
		ToolOutput: "fatal: cannot commit in dirty working tree",
		Format:    surface.FormatJSON,
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf.String()).To(ContainSubstring("stash-before-ops"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `ToolOutput` field doesn't exist on `Options`

- [ ] **Step 3: Add `ToolOutput` and `ToolErrored` to `Options` struct**

In `internal/surface/surface.go`, add to the `Options` struct:

```go
type Options struct {
	Mode             string
	DataDir          string
	Message          string // for prompt mode
	ToolName         string // for tool mode
	ToolInput        string // for tool mode
	ToolOutput       string // for tool mode: tool result or error text
	ToolErrored      bool   // for tool mode: true if tool call failed
	Format           string // output format: "" (plain) or "json"
	Budget           int    // token budget override (precompact mode)
	TranscriptWindow string // recent transcript text for transcript suppression (REQ-P4f-3)
}
```

- [ ] **Step 4: Enrich BM25 query in `matchToolMemories`**

Change `matchToolMemories` signature to accept `toolOutput string`:

```go
func matchToolMemories(
	_, toolInput, toolOutput string,
	memories []*memory.Stored,
	effectiveness map[string]EffectivenessStat,
) []toolMatch {
```

Update the BM25 scoring query to include tool output:

```go
	query := toolInput
	if toolOutput != "" {
		query = toolInput + " " + toolOutput
	}
	scored := scorer.Score(query, docs)
```

Update the call site in `runTool` (~line 708):

```go
	candidates := matchToolMemories(opts.ToolName, opts.ToolInput, opts.ToolOutput, memories, effectiveness)
```

- [ ] **Step 5: Run test to verify it passes**

Run: `targ test`
Expected: PASS — the enriched query now matches the memory

- [ ] **Step 6: Add CLI flags for `--tool-output` and `--tool-errored`**

In `internal/cli/cli.go` `runSurface` function, add:

```go
	toolOutput := fs.String("tool-output", "", "tool output or error text (tool mode)")
	toolErrored := fs.Bool("tool-errored", false, "true if tool call failed (tool mode)")
```

And pass them through to Options:

```go
	return surfacer.Run(ctx, stdout, surface.Options{
		Mode:        *mode,
		DataDir:     *dataDir,
		Message:     *message,
		ToolName:    *toolName,
		ToolInput:   *toolInput,
		ToolOutput:  *toolOutput,
		ToolErrored: *toolErrored,
		Format:      *format,
		Budget:      *budget,
	})
```

- [ ] **Step 7: Add to `SurfaceArgs` and `SurfaceFlags` in targets.go**

```go
type SurfaceArgs struct {
	Mode        string `targ:"flag,name=mode,desc=surface mode: session-start or prompt or tool"`
	DataDir     string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Message     string `targ:"flag,name=message,desc=user message (prompt mode)"`
	ToolName    string `targ:"flag,name=tool-name,desc=tool name (tool mode)"`
	ToolInput   string `targ:"flag,name=tool-input,desc=tool input JSON (tool mode)"`
	ToolOutput  string `targ:"flag,name=tool-output,desc=tool output or error text (tool mode)"`
	ToolErrored bool   `targ:"flag,name=tool-errored,desc=true if tool call failed (tool mode)"`
	Format      string `targ:"flag,name=format,desc=output format: json"`
}
```

Update `SurfaceFlags`:
```go
func SurfaceFlags(a SurfaceArgs) []string {
	flags := BuildFlags(
		"--mode", a.Mode,
		"--data-dir", a.DataDir,
		"--message", a.Message,
		"--tool-name", a.ToolName,
		"--tool-input", a.ToolInput,
		"--tool-output", a.ToolOutput,
		"--format", a.Format,
	)
	return AddBoolFlag(flags, "--tool-errored", a.ToolErrored)
}
```

- [ ] **Step 8: Run `targ check-full`**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/surface/surface.go internal/cli/cli.go internal/cli/targets.go internal/surface/*_test.go
git commit -m "feat(surface): add --tool-output and --tool-errored flags to tool mode

Enriches BM25 query with tool output text, enabling memory surfacing
based on error messages and command results — not just tool input.
ToolErrored flag marks failure context for downstream behavior tuning.

Closes #308

AI-Used: [claude]"
```

### Task 5: Lower relevance floor when `ToolErrored` is true

**Files:**
- Modify: `internal/surface/surface.go` in `runTool` and `matchToolMemories`

- [ ] **Step 1: Write failing test — errored lowers BM25 floor**

Add a test where a memory scores between `minRelevanceScore` (0.05) and `unprovenBM25FloorTool` (0.30) — it should be surfaced when `ToolErrored=true` even though it's unproven, because the floor drops.

```go
func TestToolErrored_LowersUnprovenFloor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Memory with weak keyword overlap — scores above 0.05 but below 0.30
	target := &memory.Stored{
		FilePath:    "/data/memories/check-binary-path.toml",
		Title:       "verify binary path",
		Principle:   "Check that binary exists at expected path before invoking",
		AntiPattern: "binary not found command path",
		Keywords:    []string{"binary", "path"},
	}

	memories := []*memory.Stored{target}
	for i := range 20 {
		memories = append(memories, &memory.Stored{
			FilePath:    fmt.Sprintf("/data/memories/filler-%d.toml", i),
			AntiPattern: fmt.Sprintf("completely different topic %d about unrelated things", i),
		})
	}

	retriever := &fakeRetriever{memories: memories}
	s := surface.New(retriever)

	// Without errored — unproven floor (0.30) filters it out
	var buf1 bytes.Buffer
	err := s.Run(context.Background(), &buf1, surface.Options{
		Mode:      surface.ModeTool,
		DataDir:   "/tmp/data",
		ToolName:  "Bash",
		ToolInput: `{"command": "engram build"}`,
		ToolOutput: "command not found",
		Format:    surface.FormatJSON,
	})
	g.Expect(err).NotTo(HaveOccurred())
	// May or may not match — the key assertion is the errored case below

	// With errored — floor drops to minRelevanceScore (0.05)
	var buf2 bytes.Buffer
	err = s.Run(context.Background(), &buf2, surface.Options{
		Mode:        surface.ModeTool,
		DataDir:     "/tmp/data",
		ToolName:    "Bash",
		ToolInput:   `{"command": "engram build"}`,
		ToolOutput:  "command not found",
		ToolErrored: true,
		Format:      surface.FormatJSON,
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(buf2.String()).To(ContainSubstring("check-binary-path"))
}
```

Note: This test may need tuning based on actual BM25 scores. The key behavior to test is that `ToolErrored=true` uses `minRelevanceScore` as the floor for unproven memories instead of `unprovenBM25FloorTool`.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — `ToolErrored` not yet used in matching logic

- [ ] **Step 3: Pass `ToolErrored` into `matchToolMemories` and lower floor**

Update signature:
```go
func matchToolMemories(
	_, toolInput, toolOutput string,
	errored bool,
	memories []*memory.Stored,
	effectiveness map[string]EffectivenessStat,
) []toolMatch {
```

Update floor logic:
```go
	for _, result := range scored {
		floor := minRelevanceScore
		if isUnproven(result.ID, effectiveness) && !errored {
			floor = unprovenBM25FloorTool
		}
		// When errored, all memories use the base minRelevanceScore floor —
		// any potentially relevant memory is worth surfacing during failures.
```

Update call site in `runTool`:
```go
	candidates := matchToolMemories(opts.ToolName, opts.ToolInput, opts.ToolOutput, opts.ToolErrored, memories, effectiveness)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Run `targ check-full`**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/surface/surface.go internal/surface/*_test.go
git commit -m "feat(surface): lower unproven floor when tool errored

When a tool call fails, any relevant memory is high-value. Drop the
unproven BM25 floor from 0.30 to the base 0.05 so marginal matches
still surface during failures.

AI-Used: [claude]"
```

---

## Chunk 3: Update hooks to use unified surfacing

### Task 6: Update `post-tool-use.sh` to call `surface` instead of `remind`

**Files:**
- Modify: `hooks/post-tool-use.sh`

- [ ] **Step 1: Rewrite `post-tool-use.sh`**

The hook receives `tool_name`, `tool_input`, and `tool_response` in stdin JSON (note: Claude Code uses `tool_response`, not `tool_output`). Replace the `remind` call with `surface --mode tool`:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Read tool details from stdin JSON
STDIN_JSON="$(cat)"
TOOL_NAME="$(echo "$STDIN_JSON" | jq -r '.tool_name // empty')"
TOOL_INPUT="$(echo "$STDIN_JSON" | jq -c '.tool_input // {}')"
TOOL_RESPONSE="$(echo "$STDIN_JSON" | jq -r '.tool_response // empty')"
FILE_PATH="$(echo "$STDIN_JSON" | jq -r '.tool_input.file_path // empty')"

# Only fire for Write and Edit tools (T-213)
if [[ "$TOOL_NAME" != "Write" && "$TOOL_NAME" != "Edit" ]]; then
    exit 0
fi

ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"
DATA_DIR="${ENGRAM_DATA_DIR:-${ENGRAM_HOME}/data}"

# Skill/command file advisory
if [[ "$FILE_PATH" == */skills/* || "$FILE_PATH" == */.claude/commands/* ]]; then
    jq -n '{
        continue: true,
        suppressOutput: false,
        hookSpecificOutput: {
            hookEventName: "PostToolUse",
            additionalContext: "You just edited a skill/command file — did you pressure-test the changes? Verify it still triggers correctly and handles edge cases."
        }
    }'
    exit 0
fi

# Surface memories relevant to this tool call and its output
if [[ -x "$ENGRAM_BIN" ]]; then
    SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode tool \
        --tool-name "$TOOL_NAME" --tool-input "$TOOL_INPUT" \
        --tool-output "$TOOL_RESPONSE" \
        --data-dir "$DATA_DIR" --format json) || true
    if [[ -n "$SURFACE_OUTPUT" ]]; then
        echo "$SURFACE_OUTPUT" | jq '{
            continue: true,
            suppressOutput: false,
            hookSpecificOutput: {
                hookEventName: "PostToolUse",
                additionalContext: .context
            }
        }'
        exit 0
    fi
fi
```

- [ ] **Step 2: Test the hook manually**

Run: `echo '{"tool_name":"Write","tool_input":{"file_path":"test.go"},"tool_response":"success"}' | hooks/post-tool-use.sh`
Expected: Either empty (no matching memories) or JSON with additionalContext

- [ ] **Step 3: Commit**

```bash
git add hooks/post-tool-use.sh
git commit -m "refactor(hooks): replace remind with surface in post-tool-use

PostToolUse now calls surface --mode tool with --tool-output,
enabling BM25 matching against both input and output. This replaces
the removed remind command.

AI-Used: [claude]"
```

### Task 7: Update `post-tool-use-failure.sh` to surface memories alongside static advisory

**Files:**
- Modify: `hooks/post-tool-use-failure.sh`

- [ ] **Step 1: Rewrite `post-tool-use-failure.sh`**

Combine static recovery advisory with memory surfacing:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Read failure details from stdin JSON
STDIN_JSON="$(cat)"
TOOL_NAME="$(echo "$STDIN_JSON" | jq -r '.tool_name // "unknown"')"
TOOL_INPUT="$(echo "$STDIN_JSON" | jq -c '.tool_input // {}')"
ERROR="$(echo "$STDIN_JSON" | jq -r '.error // "unknown error"')"
IS_INTERRUPT="$(echo "$STDIN_JSON" | jq -r '.is_interrupt // false')"

# Skip advisory when user intentionally cancelled
if [[ "$IS_INTERRUPT" == "true" ]]; then
    exit 0
fi

ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"
DATA_DIR="${ENGRAM_DATA_DIR:-${ENGRAM_HOME}/data}"

# Build targeted advice based on tool type
case "$TOOL_NAME" in
    Read)
        ADVICE="Tool failed. Check that the file path exists and is correct, then retry or try an alternative. Continue working toward the intended outcome."
        ;;
    Bash)
        ADVICE="Tool failed. Diagnose the error from the output, fix the command or try an alternative approach. Continue working toward the intended outcome."
        ;;
    Edit)
        ADVICE="Tool failed. The old_string likely didn't match — re-read the file to get the exact current content, then retry. Continue working toward the intended outcome."
        ;;
    Write)
        ADVICE="Tool failed. Check that the directory exists and you have the correct path, then retry. Continue working toward the intended outcome."
        ;;
    Grep|Glob)
        ADVICE="Tool failed. Check the pattern syntax and path, then retry or try a different search approach. Continue working toward the intended outcome."
        ;;
    *)
        ADVICE="Tool failed. Diagnose the error, fix or try an alternative. Continue working toward the intended outcome."
        ;;
esac

# Surface relevant memories about this failure
MEMORY_CONTEXT=""
if [[ -x "$ENGRAM_BIN" ]]; then
    MEMORY_CONTEXT=$("$ENGRAM_BIN" surface --mode tool \
        --tool-name "$TOOL_NAME" --tool-input "$TOOL_INPUT" \
        --tool-output "$ERROR" --tool-errored \
        --data-dir "$DATA_DIR" --format json 2>/dev/null \
        | jq -r '.context // empty') || true
fi

# Combine static advisory with memory context
if [[ -n "$MEMORY_CONTEXT" ]]; then
    COMBINED="${ADVICE}
${MEMORY_CONTEXT}"
else
    COMBINED="$ADVICE"
fi

jq -n --arg ctx "$COMBINED" '{
    continue: true,
    suppressOutput: false,
    hookSpecificOutput: {
        hookEventName: "PostToolUseFailure",
        additionalContext: $ctx
    }
}'
```

- [ ] **Step 2: Test the hook manually**

Run: `echo '{"tool_name":"Bash","tool_input":{"command":"targ build"},"error":"command not found","is_interrupt":false}' | hooks/post-tool-use-failure.sh`
Expected: JSON with static advice + any matching memories

Run: `echo '{"tool_name":"Read","error":"no such file","is_interrupt":true}' | hooks/post-tool-use-failure.sh`
Expected: No output (interrupt skipped)

- [ ] **Step 3: Commit**

```bash
git add hooks/post-tool-use-failure.sh
git commit -m "feat(hooks): surface memories alongside failure advisory

PostToolUseFailure now calls surface --mode tool with --tool-errored
and the error text as --tool-output. Relevant memories are appended
to the static recovery advisory, giving the LLM both general
guidance and specific learned context about similar past failures.

AI-Used: [claude]"
```

### Task 8: Final verification

- [ ] **Step 1: Run `targ check-full`**

Run: `targ check-full`
Expected: All tests pass, no lint errors

- [ ] **Step 2: Rebuild the engram binary**

Run: `targ build && cp cmd/engram/engram ~/.claude/engram/bin/engram`
Expected: Build succeeds, binary installed

- [ ] **Step 3: Verify all three hooks work end-to-end**

```bash
# PreToolUse (unchanged, should still work)
echo '{"tool_name":"Bash","tool_input":{"command":"git commit"}}' | hooks/pre-tool-use.sh

# PostToolUse (now uses surface instead of remind)
echo '{"tool_name":"Write","tool_input":{"file_path":"test.go"},"tool_response":"ok"}' | hooks/post-tool-use.sh

# PostToolUseFailure (new, with memory surfacing)
echo '{"tool_name":"Bash","tool_input":{"command":"targ build"},"error":"exit 1","is_interrupt":false}' | hooks/post-tool-use-failure.sh
```

Expected: All produce valid JSON or exit cleanly

- [ ] **Step 4: Verify `remind` subcommand is gone**

Run: `~/.claude/engram/bin/engram remind --help`
Expected: "unknown command" error

- [ ] **Step 5: Commit any fixups, if needed**
