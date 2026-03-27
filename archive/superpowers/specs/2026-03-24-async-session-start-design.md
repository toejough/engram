# Async SessionStart Hook (#370)

## Problem

The SessionStart hook runs `engram maintain` synchronously, blocking the user's first interaction for ~7.5s. Users must wait before they can start working.

## Design

Make SessionStart fully non-blocking by splitting it into sync (fast static messages) and async (maintain + build), with results surfaced at the next tool-use hook.

### Sync SessionStart Hook (`session-start-sync.sh`)

Emits static context only — completes in <100ms:

```json
{
  "systemMessage": "[engram] Say /recall to load context from previous sessions, or /recall <query> to search session history.",
  "additionalContext": "[engram] Mid-turn user messages (delivered via system-reminder) bypass engram hooks. If you receive a mid-turn correction or instruction, capture it by running: ~/.claude/engram/bin/engram correct --message '<the user message>'"
}
```

No build check, no maintain call, no I/O beyond stdout. No triage placeholder — triage results arrive when they're ready via the pending file mechanism.

### Async SessionStart Hook (`session-start.sh`, refactored)

Runs the slow work in the background:

1. Delete any stale `$ENGRAM_HOME/pending-maintenance.json` from a previous session
2. Build check + symlink setup (same as today)
3. `engram maintain` (with `|| true` — failures produce no file, no error surfaced)
4. Parse proposals into triage format (same jq logic as today)
5. Write result to a temp file and atomically rename to `$ENGRAM_HOME/pending-maintenance.json` (prevents Pre/PostToolUse from reading a partial file mid-write)
6. Replace the current stdout `jq` output with the file write — no stdout (async hooks have it ignored)

If maintain produces no proposals, no file is written.

All script paths use `$ENGRAM_HOME` (i.e., `$HOME/.claude/engram`), never literal `~`. (Tilde in user-facing instruction text is fine — it's guidance for Claude, not a shell-evaluated path.)

### Pending File Format

`$ENGRAM_HOME/pending-maintenance.json` contains a pre-built context blob:

```json
{
  "systemMessage": "[engram] Memory triage: 3 refine keywords, 1 leech pending. Say \"triage\" to review, or ignore to proceed.",
  "additionalContext": "[engram] Memory triage details...\n## Leech (1 memories)\n..."
}
```

### Pre/PostToolUse Pending Check

Both `pre-tool-use.sh` and `post-tool-use.sh` get a new block **after the engram-command filter but before any early-exit paths** (Bash-only exit in both hooks, Write/Edit advisory exit in post-tool-use). This ensures the pending file is consumed regardless of tool type (Read, Write, Agent, etc.).

Consumption logic (non-exiting — stores into shell variables):

1. Check if `$ENGRAM_HOME/pending-maintenance.json` exists
2. If yes: `mv` to a temp file (atomic on POSIX — prevents duplicate reads if two hooks race), read `systemMessage` and `additionalContext` from temp into `PENDING_SYS` and `PENDING_CTX` shell variables, delete temp
3. Do NOT emit or exit — later output points merge the pending content

This means the pending check is a variable-setting preamble, not an exit path. All existing output points (Write/Edit advisory, memory surfacing, or no-op) check for `PENDING_SYS`/`PENDING_CTX` and prepend them to their output. If a hook reaches the end with no other output but has pending content, it emits the pending content alone in the hook-appropriate envelope.

**PreToolUse envelope:**
```json
{
  "systemMessage": "<from file>",
  "continue": true,
  "suppressOutput": false,
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow",
    "permissionDecisionReason": "",
    "additionalContext": "<from file>"
  }
}
```

**PostToolUse envelope:**
```json
{
  "systemMessage": "<from file>",
  "continue": true,
  "suppressOutput": false,
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "<from file>"
  }
}
```

**Merge behavior:** Since the pending check stores into variables without exiting, all downstream output points naturally merge. Each output point (advisory, surfacing, or standalone pending) prepends `PENDING_SYS` to its `systemMessage` and `PENDING_CTX` to its `additionalContext`. The hook always emits a single JSON blob. If multiple output paths would fire (e.g., pending + advisory + surfacing), the first to `exit 0` wins — this matches current behavior where advisory exits before surfacing.

### hooks.json Changes

Two SessionStart entries:

```json
"SessionStart": [
  {
    "hooks": [{
      "type": "command",
      "command": "${CLAUDE_PLUGIN_ROOT}/hooks/session-start-sync.sh",
      "timeout": 5
    }]
  },
  {
    "hooks": [{
      "type": "command",
      "command": "${CLAUDE_PLUGIN_ROOT}/hooks/session-start.sh",
      "timeout": 120,
      "async": true
    }]
  }
]
```

### What Stays, What Moves

| Component | Before | After |
|-----------|--------|-------|
| Build check | session-start.sh (sync) | session-start.sh (async) |
| Symlink setup | session-start.sh (sync) | session-start.sh (async) |
| `engram maintain` | session-start.sh (sync) | session-start.sh (async) |
| Triage jq parsing | session-start.sh → stdout | session-start.sh → file |
| `/recall` reminder | session-start.sh (sync) | session-start-sync.sh (sync) |
| Mid-turn note | session-start.sh (sync) | session-start-sync.sh (sync) |
| Pre/PostToolUse build check | unchanged | unchanged |
| Pre/PostToolUse surfacing | unchanged | + pending file check |

### Testing

- Structural tests: `hooks.json` has two SessionStart entries (sync + async)
- Structural tests: pending file path uses `$ENGRAM_HOME` consistently across writer and readers
- Structural tests: Pre/PostToolUse pending check is before the Bash-only exit
- Manual verification: session starts without blocking, triage appears on first tool call

### Risks

- **Double build race:** If async SessionStart and first PreToolUse both trigger `go build -o $ENGRAM_BIN` simultaneously, concurrent writes could corrupt the binary. Mitigation: the async hook builds to a temp file and atomically renames (`go build -o "$ENGRAM_BIN.tmp" && mv "$ENGRAM_BIN.tmp" "$ENGRAM_BIN"`). Pre/PostToolUse hooks already self-build — their build check sees the binary as executable and skips.
- **Pending file never consumed:** If a session has zero tool calls after start, the file persists. The async hook deletes stale files on next session start before running maintain.
