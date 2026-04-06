# Design: Fix engram agent spawn failures (issue #527)

**Date:** 2026-04-06
**Issue:** toejough/engram#527

---

## Problem

`engram agent spawn --name X --prompt Y` fails in two independent ways in a fresh environment.

### Bug 1: State directory not auto-created

`readModifyWriteStateFile` calls `osStateFileLock` (which creates a `.lock` file via `os.OpenFile(O_CREATE|O_EXCL)`) **before** calling `os.MkdirAll`. If `~/.local/share/engram/state/` does not exist, the `OpenFile` call fails immediately with:

```
open ~/.local/share/engram/state/<slug>.toml.lock: no such file or directory
```

`os.MkdirAll` is called later in the function, after the lock — too late.

### Bug 2: Pane dies immediately after spawn

`osTmuxSpawnWith` runs:
```
tmux new-window -d -n <name> -P -F "#{pane_id} #{session_id}" -- sh -c <prompt>
```

The `prompt` argument is a skill invocation (e.g., `/use-engram-chat-as reactive memory agent named engram-agent`), not a shell command. `sh -c /use-engram-chat-as ...` fails with "command not found", exits immediately, and the tmux window closes. The binary returns a valid pane-id, but the pane is already dead by the time the caller uses it.

---

## Root Cause Analysis

**Bug 1:** The order of operations in `readModifyWriteStateFile` is wrong. `MkdirAll` must precede any file operation on a path in that directory.

**Bug 2:** The `--prompt` parameter was designed (per the `dd43db2` skill update) to hold the first message text sent to an interactive `claude` session, not a shell command to execute. The current implementation misinterprets it as a shell command.

The correct pattern — verified from the old `SPAWN-PANE` skill macro — is:
1. Create a tmux window (default shell starts)
2. Start `claude --dangerously-skip-permissions --model sonnet` via `tmux send-keys`
3. Wait for claude's input prompt character (`❯`)
4. Send the prompt text via `tmux send-keys`

---

## Design

### Fix 1: Move MkdirAll before lock acquisition

**File:** `internal/cli/cli_agent.go` — `readModifyWriteStateFile`

Move the directory creation to the top of the function, before `osStateFileLock`:

```go
func readModifyWriteStateFile(stateFilePath string, modify func(agentpkg.StateFile) agentpkg.StateFile) error {
    dir := filepath.Dir(stateFilePath)
    if mkdirErr := os.MkdirAll(dir, chatDirMode); mkdirErr != nil {
        return fmt.Errorf("creating state directory: %w", mkdirErr)
    }

    lockPath := stateFilePath + ".lock"
    unlock, lockErr := osStateFileLock(lockPath)
    // ... rest unchanged, remove the duplicate MkdirAll below
```

Remove the now-duplicate `MkdirAll` from later in the function.

### Fix 2: Three-step pane lifecycle in `osTmuxSpawnWith`

**File:** `internal/cli/cli_agent.go` — `osTmuxSpawnWith`

**New constants (with existing constants block):**
```go
claudeReadyMaxRetries   = 30
claudeReadyPollInterval = time.Second
claudeSettings          = `{"statusLine":{"type":"command","command":"true"}}`
```

**New implementation:**

```
Step 1: tmux new-window -d -n <name> -P -F "#{pane_id} #{session_id}"
        (no command — tmux starts the default shell; pane stays alive)
        → parse pane-id and session-id

Step 2: tmux send-keys -t <pane-id>
        "claude --dangerously-skip-permissions --model sonnet --settings '<claudeSettings>'" Enter
        (sends command to the shell, which starts claude)

Step 3: Poll tmux capture-pane -t <pane-id> -p
        until output contains "❯" (claude input prompt) or claudeReadyMaxRetries exceeded
        (sleep claudeReadyPollInterval between retries; respect ctx.Done())

Step 4: tmux send-keys -t <pane-id> "<prompt>" Enter
        (sends the skill invocation / task description to claude)

Return pane-id, session-id
```

**Error handling:**
- `new-window` fails → return error (unchanged)
- `send-keys` for claude command fails → return `fmt.Errorf("tmux send-keys: %w", err)`
- capture-pane never shows `❯` (timeout) → proceed anyway (best-effort; log is omitted to avoid I/O in this function)
- `send-keys` for prompt fails → return `fmt.Errorf("tmux send-keys prompt: %w", err)`

---

## Test Changes

### New tests

| Test | Verifies |
|------|---------|
| `TestReadModifyWriteStateFile_MissingDir_CreatesDirAndFile` | State dir created when absent before lock attempt |
| `TestOsTmuxSpawnWith_SendsPromptViaKeysNotShellCmd` | `new-window` does not use `sh -c`; `send-keys` receives the prompt text |
| `TestOsTmuxSpawnWith_SendKeysFails_ReturnsError` | Error returned when `send-keys` for prompt exits non-zero |

### Updated tests

| Test | Change |
|------|--------|
| `TestOsTmuxSpawnWith_Success_ReturnsPaneAndSession` | Fake tmux script handles three subcommands: `new-window` → pane-id/session output; `capture-pane` → `❯`; `send-keys` → no-op |

### Unchanged tests

`TestOsTmuxSpawnWith_CommandFails_ReturnsError`, `TestOsTmuxSpawnWith_UnexpectedOutput_ReturnsError`, `TestOsTmuxSpawn_CancelledContext_ReturnsError` — all test `new-window` error paths, unaffected by send-keys changes.

---

## Acceptance Criteria

1. `engram agent spawn --name X --prompt Y` succeeds in a fresh environment (no pre-existing `~/.local/share/engram/state/`)
2. After spawn returns a pane-id, the pane remains alive — `tmux capture-pane -t <pane-id>` succeeds
3. `claude` is running in the spawned pane
4. `targ check-full` passes at PASS:8

---

## Out of Scope

- Configurable claude flags (model, settings) — hardcoded for now, matches old skill behavior
- Timeout behavior when claude never shows `❯` — best-effort proceed, no user-visible error
