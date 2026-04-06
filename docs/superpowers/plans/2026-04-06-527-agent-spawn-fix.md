# Agent Spawn Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix two production bugs in `engram agent spawn`: state directory not auto-created before lock acquisition, and pane dying immediately because the prompt is run as a shell command instead of being sent to an interactive claude session.

**Architecture:** Two targeted changes to `internal/cli/cli_agent.go`. Bug 1: reorder `os.MkdirAll` in `readModifyWriteStateFile` to precede `osStateFileLock`. Bug 2: replace the single-step `tmux new-window ... sh -c <prompt>` call in `osTmuxSpawnWith` with a three-step protocol: create pane (no command) → start claude via send-keys → wait for `❯` → send prompt via send-keys.

**Tech Stack:** Go 1.22+, `targ` build system (use `targ test` / `targ check-full`), gomega matchers in tests.

---

## File Map

| File | Change |
|------|--------|
| `internal/cli/cli_agent.go` | Move `MkdirAll` before `osStateFileLock`; rewrite `osTmuxSpawnWith`; add constants |
| `internal/cli/cli_test.go` | Add 3 new tests; update 1 existing test |

No changes to `export_test.go` — `ExportReadModifyWriteStateFile` and `ExportOsTmuxSpawnWith` are already exported.

---

## Task 1: Fix Bug 1 — state directory not created before lock (TDD)

**Files:**
- Modify: `internal/cli/cli_test.go` (add failing test)
- Modify: `internal/cli/cli_agent.go` (move MkdirAll)

### Step 1.1 — Write failing test

Add this test to `internal/cli/cli_test.go` (after `TestReadModifyWriteStateFile_CreatesFileWhenAbsent`, around line 521):

```go
func TestReadModifyWriteStateFile_MissingDir_CreatesDirAndFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	base := t.TempDir()
	// state/ subdirectory does NOT exist yet
	stateFile := filepath.Join(base, "state", "test.toml")

	err := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
		return agentpkg.AddAgent(sf, agentpkg.AgentRecord{Name: "test-agent", State: "STARTING"})
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stateFile).To(BeAnExistingFile())

	data, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("test-agent"))
}
```

### Step 1.2 — Run test to confirm it fails

```bash
cd /Users/joe/repos/personal/engram && targ test 2>&1 | grep -A5 "MissingDir"
```

Expected: `FAIL` — `acquiring state file lock: creating state lock: open .../state/test.toml.lock: no such file or directory`

### Step 1.3 — Move MkdirAll before osStateFileLock

Replace the entire `readModifyWriteStateFile` function in `internal/cli/cli_agent.go` (lines 291–333):

```go
// readModifyWriteStateFile performs a locked read-modify-write on the state file.
// Creates the file and its parent directory if they do not exist.
func readModifyWriteStateFile(stateFilePath string, modify func(agentpkg.StateFile) agentpkg.StateFile) error {
	dir := filepath.Dir(stateFilePath)

	if mkdirErr := os.MkdirAll(dir, chatDirMode); mkdirErr != nil {
		return fmt.Errorf("creating state directory: %w", mkdirErr)
	}

	lockPath := stateFilePath + ".lock"

	unlock, lockErr := osStateFileLock(lockPath)
	if lockErr != nil {
		return fmt.Errorf("acquiring state file lock: %w", lockErr)
	}

	defer func() { _ = unlock() }()

	data, readErr := os.ReadFile(stateFilePath) //nolint:gosec
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("reading state file: %w", readErr)
	}

	currentState, parseErr := agentpkg.ParseStateFile(data)
	if parseErr != nil {
		return fmt.Errorf("parsing state file: %w", parseErr)
	}

	currentState = modify(currentState)

	newData, marshalErr := agentpkg.MarshalStateFile(currentState)
	if marshalErr != nil {
		return fmt.Errorf("marshaling state file: %w", marshalErr)
	}

	writeErr := os.WriteFile(stateFilePath, newData, chatFileMode)
	if writeErr != nil {
		return fmt.Errorf("writing state file: %w", writeErr)
	}

	return nil
}
```

Key changes: `dir` and `MkdirAll` moved to the top; the duplicate `dir := filepath.Dir(stateFilePath)` and `MkdirAll` block that was just before `os.WriteFile` is removed.

### Step 1.4 — Run tests to confirm pass

```bash
cd /Users/joe/repos/personal/engram && targ test 2>&1 | grep -E "PASS|FAIL|MissingDir"
```

Expected: `PASS` for the new test; all existing `TestReadModifyWriteStateFile_*` tests still pass.

### Step 1.5 — Commit

```bash
cd /Users/joe/repos/personal/engram && git add internal/cli/cli_agent.go internal/cli/cli_test.go && git commit -m "$(cat <<'EOF'
fix(cli): create state dir before acquiring lock in readModifyWriteStateFile

osStateFileLock calls os.OpenFile(O_CREATE|O_EXCL) on the .lock file.
If the parent directory doesn't exist, this fails immediately.
MkdirAll was called after the lock — too late. Moving it to the top
of the function fixes the 'no such file or directory' error on first run.

Fixes #527 (bug 1 of 2).

AI-Used: [claude]
EOF
)"
```

---

## Task 2: Fix Bug 2 — pane dies immediately (TDD, part A: add constants and update `osTmuxSpawnWith`)

**Files:**
- Modify: `internal/cli/cli_test.go` (add 3 failing tests, update 1 existing test)
- Modify: `internal/cli/cli_agent.go` (new constants + rewrite `osTmuxSpawnWith`)

### Step 2.1 — Write three failing tests

Add the following tests to `internal/cli/cli_test.go` after `TestOsTmuxSpawnWith_UnexpectedOutput_ReturnsError` (around line 334).

**Test A** — verifies `send-keys` is called with the prompt (not `sh -c`):

```go
func TestOsTmuxSpawnWith_SendsPromptViaKeysNotShellCmd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	fakeTmux := filepath.Join(tmpDir, "tmux")
	callLog := filepath.Join(tmpDir, "calls.txt")

	script := "#!/bin/sh\n" +
		"echo \"$@\" >> " + callLog + "\n" +
		"case \"$1\" in\n" +
		"  new-window) echo '%my-pane $mysession' ;;\n" +
		"  capture-pane) printf '❯\\n' ;;\n" +
		"  send-keys) ;;\n" +
		"esac\n"
	g.Expect(os.WriteFile(fakeTmux, []byte(script), 0o700)).To(Succeed())

	paneID, sessionID, err := cli.ExportOsTmuxSpawnWith(t.Context(), fakeTmux, "myagent", "my-prompt-text")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(paneID).To(Equal("%my-pane"))
	g.Expect(sessionID).To(Equal("$mysession"))

	calls, readErr := os.ReadFile(callLog)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	// send-keys must be called with the prompt text
	g.Expect(string(calls)).To(ContainSubstring("send-keys"))
	g.Expect(string(calls)).To(ContainSubstring("my-prompt-text"))
	// new-window must NOT use sh -c
	g.Expect(string(calls)).NotTo(ContainSubstring("sh -c"))
}
```

**Test B** — send-keys for prompt fails → error returned:

```go
func TestOsTmuxSpawnWith_SendKeysFails_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	fakeTmux := filepath.Join(tmpDir, "tmux")

	script := "#!/bin/sh\n" +
		"case \"$1\" in\n" +
		"  new-window) echo '%my-pane $mysession' ;;\n" +
		"  capture-pane) printf '❯\\n' ;;\n" +
		"  send-keys) exit 1 ;;\n" +
		"  *) exit 1 ;;\n" +
		"esac\n"
	g.Expect(os.WriteFile(fakeTmux, []byte(script), 0o700)).To(Succeed())

	_, _, err := cli.ExportOsTmuxSpawnWith(t.Context(), fakeTmux, "myagent", "my-prompt-text")

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("tmux send-keys"))
	}
}
```

### Step 2.2 — Update existing `TestOsTmuxSpawnWith_Success_ReturnsPaneAndSession`

Replace the fake tmux script in the test so it handles all three subcommands. Find the line:

```go
g.Expect(os.WriteFile(fakeTmux, []byte("#!/bin/sh\necho '%my-pane $mysession'\n"), 0o700)).To(Succeed())
```

Replace with:

```go
script := "#!/bin/sh\n" +
    "case \"$1\" in\n" +
    "  new-window) echo '%my-pane $mysession' ;;\n" +
    "  capture-pane) printf '❯\\n' ;;\n" +
    "  send-keys) ;;\n" +
    "  *) exit 1 ;;\n" +
    "esac\n"
g.Expect(os.WriteFile(fakeTmux, []byte(script), 0o700)).To(Succeed())
```

Also update the prompt argument from `"sh -c 'echo hello'"` to `"my-prompt-text"` on the `ExportOsTmuxSpawnWith` call line in that test (the argument is descriptive only — any string works).

### Step 2.3 — Run tests to confirm they fail

```bash
cd /Users/joe/repos/personal/engram && targ test 2>&1 | grep -E "FAIL|SendsPrompt|SendKeys|Success_Returns"
```

Expected: `TestOsTmuxSpawnWith_SendsPromptViaKeysNotShellCmd` and `TestOsTmuxSpawnWith_SendKeysFails_ReturnsError` fail (new tests). `TestOsTmuxSpawnWith_Success_ReturnsPaneAndSession` also fails (updated fake hangs or errors with current `sh -c` implementation).

### Step 2.4 — Add constants and rewrite `osTmuxSpawnWith`

**In `internal/cli/cli_agent.go`, add to the `const` block** (lines 22–26):

```go
const (
	claudeReadyMaxRetries   = 30
	claudeReadyPollInterval = time.Second
	claudeSettings          = `{"statusLine":{"type":"command","command":"true"}}`
	spawnAckMaxWait         = 30 * time.Second
	stateFileLockDelay      = 25 * time.Millisecond
	stateFileLockRetries    = 200
)
```

**Replace `osTmuxSpawnWith`** (lines 164–177 in `internal/cli/cli_agent.go`):

```go
// osTmuxSpawnWith creates a tmux window using the given binary path, starts claude
// interactively in it, waits for the claude prompt (❯), then sends the prompt text
// via send-keys. Returns pane-id and session-id.
// Extracted so tests can supply a fake binary path without modifying global state.
func osTmuxSpawnWith(ctx context.Context, tmuxBin, name, prompt string) (paneID, sessionID string, err error) {
	// Step 1: Create pane with default shell (no command — pane stays alive).
	out, cmdErr := exec.CommandContext(ctx, tmuxBin, //nolint:gosec
		"new-window",
		"-d",
		"-n", name,
		"-P", "-F", "#{pane_id} #{session_id}",
	).Output()
	if cmdErr != nil {
		return "", "", fmt.Errorf("tmux new-window: %w", cmdErr)
	}

	paneID, sessionID, parseErr := parseTmuxOutput(out)
	if parseErr != nil {
		return "", "", parseErr
	}

	// Step 2: Start claude in the pane.
	claudeCmd := "claude --dangerously-skip-permissions --model sonnet --settings '" + claudeSettings + "'"

	if startErr := exec.CommandContext(ctx, tmuxBin, //nolint:gosec
		"send-keys", "-t", paneID, claudeCmd, "Enter",
	).Run(); startErr != nil {
		return "", "", fmt.Errorf("tmux send-keys: %w", startErr)
	}

	// Step 3: Wait for claude's input prompt (❯), up to claudeReadyMaxRetries seconds.
	for range claudeReadyMaxRetries {
		if ctx.Err() != nil {
			break
		}

		paneContent, captureErr := exec.CommandContext(ctx, tmuxBin, //nolint:gosec
			"capture-pane", "-t", paneID, "-p",
		).Output()
		if captureErr == nil && strings.Contains(string(paneContent), "❯") {
			break
		}

		time.Sleep(claudeReadyPollInterval)
	}

	// Step 4: Send the prompt text to claude (best-effort even if readiness timed out).
	if sendErr := exec.CommandContext(ctx, tmuxBin, //nolint:gosec
		"send-keys", "-t", paneID, prompt, "Enter",
	).Run(); sendErr != nil {
		return "", "", fmt.Errorf("tmux send-keys prompt: %w", sendErr)
	}

	return paneID, sessionID, nil
}
```

### Step 2.5 — Run tests to confirm they pass

```bash
cd /Users/joe/repos/personal/engram && targ test 2>&1 | grep -E "PASS|FAIL|SendsPrompt|SendKeys|Success_Returns"
```

Expected: all four targeted tests PASS.

### Step 2.6 — Run full check

```bash
cd /Users/joe/repos/personal/engram && targ check-full 2>&1 | tail -20
```

Expected: `PASS:8` (or your current passing count), no new failures.

### Step 2.7 — Commit

```bash
cd /Users/joe/repos/personal/engram && git add internal/cli/cli_agent.go internal/cli/cli_test.go && git commit -m "$(cat <<'EOF'
fix(cli): start claude in pane and send prompt via send-keys in osTmuxSpawnWith

Previously osTmuxSpawnWith ran 'sh -c <prompt>' where prompt was a skill
invocation (e.g. /use-engram-chat-as ...). sh exited immediately — pane died.

New protocol matches the old SPAWN-PANE skill macro:
1. new-window (no command — default shell stays alive)
2. send-keys: start claude --dangerously-skip-permissions --model sonnet
3. poll capture-pane until ❯ (claude ready), max 30s
4. send-keys: send prompt text to claude

Fixes #527 (bug 2 of 2).

AI-Used: [claude]
EOF
)"
```

---

## Task 3: Verify end-to-end with real binary

### Step 3.1 — Build

```bash
cd /Users/joe/repos/personal/engram && targ build
```

Expected: binary built successfully.

### Step 3.2 — Verify state dir creation (Bug 1)

If `~/.local/share/engram/state/` exists from a previous run, remove it first to simulate a fresh environment:

```bash
rm -rf ~/.local/share/engram/state/
```

Then spawn an agent:

```bash
engram agent spawn --name test-verify --prompt "echo hello and exit"
```

Expected: no `no such file or directory` error; state dir created; output is `<pane-id>|<session-id>`.

### Step 3.3 — Verify pane survives (Bug 2)

Capture the pane-id from the previous spawn and verify it's alive:

```bash
RESULT=$(engram agent spawn --name test-pane --prompt "/use-engram-chat-as test agent")
PANE_ID=$(echo "$RESULT" | cut -d'|' -f1)
echo "Pane: $PANE_ID"
sleep 3
tmux capture-pane -t "$PANE_ID" -p 2>&1 | head -5
```

Expected: `tmux capture-pane` succeeds (non-empty output or at least no "can't find pane" error), confirming the pane is alive with claude running.

### Step 3.4 — Clean up test panes

```bash
engram agent kill --name test-verify 2>/dev/null || true
engram agent kill --name test-pane 2>/dev/null || true
```