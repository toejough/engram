# Stop Hook Memory Surfacing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Surface memories based on the agent's recent output when the Stop hook fires, so engram can react to agent claims.

**Architecture:** CLI-layer translation: `--mode stop` reads transcript delta, strips to assistant-only text, then delegates to existing prompt-mode surfacing. No changes to the surface package.

**Tech Stack:** Go, bash, JSONL parsing

**Spec:** `docs/superpowers/specs/2026-03-25-stop-hook-surfacing-design.md`

---

### Task 1: Add stop mode to runSurface CLI handler

**Files:**
- Modify: `internal/cli/cli.go:1497-1583` (runSurface function)
- Test: `internal/cli/cli_test.go` or new test file

- [ ] **Step 1: Write the failing test**

Test that `--mode stop` with a transcript file containing assistant output surfaces memories. Needs:
- A temp dir with a memory TOML that has keywords matching agent text
- A JSONL transcript file with an assistant message containing those keywords
- Verify the surface output contains the memory

```go
func TestRunSurface_StopMode(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	memDir := filepath.Join(dir, "memories")
	_ = os.MkdirAll(memDir, 0o755)

	// Create a memory about pre-existing issues
	memContent := `title = "No exceptions for pre-existing issues"
principle = "If it is broken fix it"
keywords = ["pre-existing", "broken", "fix"]
surfaced_count = 0
`
	_ = os.WriteFile(filepath.Join(memDir, "no-preexisting.toml"), []byte(memContent), 0o600)

	// Create a JSONL transcript with an assistant message
	transcript := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"The 3 failing tests are pre-existing in internal/hooks."}]}}` + "\n"
	transcriptPath := filepath.Join(dir, "transcript.jsonl")
	_ = os.WriteFile(transcriptPath, []byte(transcript), 0o600)

	var stdout bytes.Buffer
	err := runSurface([]string{
		"--mode", "stop",
		"--data-dir", dir,
		"--transcript-path", transcriptPath,
		"--session-id", "test-session",
		"--format", "json",
	}, &stdout)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// Should have surfaced the pre-existing memory
	g.Expect(stdout.String()).To(gomega.ContainSubstring("pre-existing"))
}

func TestRunSurface_StopModeNoTranscript(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	var stdout bytes.Buffer
	err := runSurface([]string{
		"--mode", "stop",
		"--data-dir", dir,
		"--format", "json",
	}, &stdout)

	// Should error — transcript-path required for stop mode
	g.Expect(err).To(gomega.HaveOccurred())
}
```

- [ ] **Step 2: Run test to verify it fails**

Expected: FAIL — unknown flag `--transcript-path` or unsupported mode `stop`

- [ ] **Step 3: Add flags and stop mode handling**

Add two new flags to `runSurface`:
```go
transcriptPath := fs.String("transcript-path", "", "transcript JSONL path (stop mode)")
sessionID := fs.String("session-id", "", "session ID (stop mode)")
```

Add stop mode handling before constructing `opts`, after flag parsing:
```go
if *mode == "stop" {
	if *transcriptPath == "" {
		return fmt.Errorf("surface: --transcript-path required for stop mode")
	}

	assistantText, offsetErr := extractAssistantDelta(*dataDir, *transcriptPath, *sessionID)
	if offsetErr != nil {
		return fmt.Errorf("surface: %w", offsetErr)
	}

	if assistantText == "" {
		// No new assistant output — return empty result
		if *format == surface.FormatJSON {
			return json.NewEncoder(stdout).Encode(surface.Result{})
		}
		return nil
	}

	// Delegate to prompt mode with assistant text as the message
	*mode = surface.ModePrompt
	*message = assistantText
}
```

Add `extractAssistantDelta` helper (in cli.go or a new file):
```go
func extractAssistantDelta(dataDir, transcriptPath, sessionID string) (string, error) {
	offsetPath := filepath.Join(dataDir, "stop-surface-offset.json")
	store := &osOffsetStore{}

	stored, readErr := store.Read(offsetPath)
	if readErr != nil {
		stored = learn.Offset{}
	}

	offset := stored.Offset
	if sessionID != stored.SessionID {
		offset = 0
	}

	reader := &osFileReader{}
	delta := sessionctx.NewDeltaReader(reader)

	lines, newOffset, deltaErr := delta.Read(transcriptPath, offset)
	if deltaErr != nil {
		return "", fmt.Errorf("reading transcript delta: %w", deltaErr)
	}

	// Always update offset, even if no assistant text found
	_ = store.Write(offsetPath, learn.Offset{Offset: newOffset, SessionID: sessionID})

	if len(lines) == 0 {
		return "", nil
	}

	stripped := sessionctx.Strip(lines)

	// Filter to assistant-only lines
	var assistantLines []string
	for _, line := range stripped {
		if strings.HasPrefix(line, "ASSISTANT: ") {
			assistantLines = append(assistantLines, strings.TrimPrefix(line, "ASSISTANT: "))
		}
	}

	return strings.Join(assistantLines, "\n"), nil
}
```

Note: `sessionctx` is the import alias for `engram/internal/context`. Check existing imports in cli.go for the correct alias. `osFileReader` and `osOffsetStore` already exist in cli.go.

- [ ] **Step 4: Run test to verify it passes**

Expected: PASS

- [ ] **Step 5: Run full check**

Run: `targ check-full`

- [ ] **Step 6: Commit**

Message: `feat: add stop mode to surface command for transcript-based memory surfacing (#382)`

---

### Task 2: Add SurfaceArgs fields for targ target definition

**Files:**
- Modify: `internal/cli/targets.go:101-111` (SurfaceArgs struct)

- [ ] **Step 1: Add TranscriptPath and SessionID to SurfaceArgs**

```go
type SurfaceArgs struct {
	Mode           string `targ:"flag,name=mode,desc=surface mode: prompt or tool or stop"`
	DataDir        string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
	Message        string `targ:"flag,name=message,desc=user message (prompt mode)"`
	ToolName       string `targ:"flag,name=tool-name,desc=tool name (tool mode)"`
	ToolInput      string `targ:"flag,name=tool-input,desc=tool input JSON (tool mode)"`
	ToolOutput     string `targ:"flag,name=tool-output,desc=tool output or error text (tool mode)"`
	ToolErrored    bool   `targ:"flag,name=tool-errored,desc=true if tool call failed (tool mode)"`
	Format         string `targ:"flag,name=format,desc=output format: json"`
	TranscriptPath string `targ:"flag,name=transcript-path,desc=transcript JSONL path (stop mode)"`
	SessionID      string `targ:"flag,name=session-id,desc=session ID (stop mode)"`
}
```

- [ ] **Step 2: Run full check**

Run: `targ check-full`

- [ ] **Step 3: Commit**

Message: `feat: add transcript-path and session-id to SurfaceArgs (#382)`

---

### Task 3: Create stop-surface.sh hook and update hooks.json

**Files:**
- Create: `hooks/stop-surface.sh`
- Modify: `hooks/hooks.json`

- [ ] **Step 1: Create stop-surface.sh**

```bash
#!/usr/bin/env bash
set -euo pipefail

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
ENGRAM_HOME="${HOME}/.claude/engram"
ENGRAM_BIN="${ENGRAM_HOME}/bin/engram"

# Build if missing or stale
NEEDS_BUILD=false
if [[ ! -x "$ENGRAM_BIN" ]]; then
    NEEDS_BUILD=true
elif [[ -d "$PLUGIN_ROOT" ]]; then
    if find "$PLUGIN_ROOT" -name '*.go' -newer "$ENGRAM_BIN" -print -quit 2>/dev/null | grep -q .; then
        NEEDS_BUILD=true
    fi
fi

if [[ "$NEEDS_BUILD" == "true" ]]; then
    mkdir -p "${ENGRAM_HOME}/bin"
    cd "$PLUGIN_ROOT"
    go build -o "$ENGRAM_BIN" ./cmd/engram/ 2>/dev/null || { echo "[engram] build failed" >&2; exit 0; }
fi

# Read hook JSON from stdin
HOOK_JSON="$(cat)"
TRANSCRIPT_PATH="$(echo "$HOOK_JSON" | jq -r '.transcript_path // empty')"
SESSION_ID="$(echo "$HOOK_JSON" | jq -r '.session_id // empty')"

if [[ -z "$TRANSCRIPT_PATH" ]]; then
    exit 0
fi

# Surface memories based on agent's recent output
SURFACE_OUTPUT=$("$ENGRAM_BIN" surface --mode stop \
    --transcript-path "$TRANSCRIPT_PATH" \
    --session-id "$SESSION_ID" \
    --format json 2>/dev/null) || SURFACE_OUTPUT=""

if [[ -n "$SURFACE_OUTPUT" ]]; then
    SUMMARY=$(echo "$SURFACE_OUTPUT" | jq -r '.summary // empty')
    CONTEXT=$(echo "$SURFACE_OUTPUT" | jq -r '.context // empty')
    if [[ -n "$CONTEXT" ]]; then
        jq -n \
            --arg summary "$SUMMARY" \
            --arg ctx "$CONTEXT" \
            '{
                systemMessage: $summary,
                hookSpecificOutput: {
                    hookEventName: "Stop",
                    additionalContext: $ctx
                }
            }'
        exit 0
    fi
fi
```

- [ ] **Step 2: Make executable**

```bash
chmod +x hooks/stop-surface.sh
```

- [ ] **Step 3: Update hooks.json**

Change the Stop entry from a single async hook to two hooks — sync surface first, then async flush:

```json
"Stop": [
  {
    "hooks": [
      {
        "type": "command",
        "command": "${CLAUDE_PLUGIN_ROOT}/hooks/stop-surface.sh",
        "timeout": 15
      },
      {
        "type": "command",
        "command": "${CLAUDE_PLUGIN_ROOT}/hooks/stop.sh",
        "timeout": 120,
        "async": true
      }
    ]
  }
]
```

- [ ] **Step 4: Commit**

```bash
git add hooks/stop-surface.sh hooks/hooks.json
```
Message: `feat: add sync stop-surface hook for memory surfacing (#382)`

---

### Task 4: Final verification

- [ ] **Step 1: Run full test suite**

Run: `targ check-full`

- [ ] **Step 2: Manual smoke test**

```bash
echo '{"transcript_path":"/dev/null","session_id":"test"}' | hooks/stop-surface.sh
```

Expected: Empty output (no transcript content).

- [ ] **Step 3: Push and close issue**

```bash
git push
gh issue close 382 --comment "Implemented: stop-surface hook + engram surface --mode stop"
```
