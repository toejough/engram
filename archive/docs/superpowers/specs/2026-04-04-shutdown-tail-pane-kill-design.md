# Design: Fix Shutdown Tail Pane Kill (Issue #479)

**Date:** 2026-04-04
**Issue:** [#479 — bug: shutdown sequence fails to kill chat tail pane](https://github.com/toejough/engram/issues/479)
**File:** `skills/engram-tmux-lead/SKILL.md`

---

## Problem

The shutdown sequence in Section 3.4, Step 4 kills the chat tail pane using:

```bash
tmux list-panes -F '#{pane_id} #{pane_current_command}' | grep tail | awk '{print $1}' | xargs -I{} tmux kill-pane -t {}
```

This fails because `#{pane_current_command}` reports the parent shell (e.g., `fish`) — not the child `tail` process spawned inside it. The grep matches nothing, `xargs` receives no input, and `kill-pane` is never called. The chat tail pane survives shutdown.

---

## Root Cause

`pane_current_command` reflects what tmux observes as the foreground command for the pane. When `tmux split-window` runs `"tail -F $CHAT_FILE"`, the shell that tmux spawned for that pane is reported — not the `tail` child process. This is shell-dependent and unreliable.

---

## Fix

Section 1.3 already captures the pane ID at creation:

```bash
TAIL_PANE_ID=$(tmux split-window -h -d -P -F '#{pane_id}' "tail -F $CHAT_FILE")
```

Section 3.4 Step 4 should use this tracked ID directly, consistent with how all other panes are killed in the skill:

```bash
tmux kill-pane -t "$TAIL_PANE_ID" 2>/dev/null || true
```

The `2>/dev/null || true` guard handles cases where the pane is already dead (e.g., user killed it manually, or startup partially failed).

---

## Approaches Considered

| Approach | Description | Decision |
|---|---|---|
| **A — Use TAIL_PANE_ID** | Kill by tracked pane ID from Section 1.3 | ✅ Selected |
| B — Process-tree search | Walk `pane_pid` → child PIDs to find `tail` | Unnecessarily complex; still needs pane ID at end |
| C — Name pane at creation | Set `pane-title = "chat-tail"` in 1.3, grep by title in 3.4 | Indirect; we already have a direct ID |

---

## Scope

- **File changed:** `skills/engram-tmux-lead/SKILL.md` only
- **Lines changed:** Section 3.4 Step 4 (one code block replacement)
- **No other changes:** Section 1.3 already captures `TAIL_PANE_ID` correctly; no other sections reference the tail pane kill

---

## Edge Cases

| Case | Handled by |
|---|---|
| Pane already dead (user killed it) | `2>/dev/null || true` |
| `TAIL_PANE_ID` empty (startup failure) | `2>/dev/null || true` |
| Other panes with `tail` in name accidentally killed | Not possible — ID-based kill is exact |
