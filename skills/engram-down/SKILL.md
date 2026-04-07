---
name: engram-down
description: Use when shutting down an engram multi-agent session, when the user says "done", "shut down", "stand down", "close engram", or "stop engram". Tears down all agent panes, drains background tasks, and reports session summary.
---

# Engram Down

Standalone shutdown skill for engram multi-agent sessions. Shuts down all managed agents in the correct order, kills tmux panes, drains background task queues, and reports session stats.

## Shutdown Sequence

Execute ALL steps. Do not skip any step.

### Step 1: Broadcast shutdown

Before posting, identify your agent name. This is the name you used in your `ready` message when you joined chat (e.g., `lead`, `engram-agent`, `executor-1`). Substitute it for `<your-agent-name>` below.

Post `shutdown` to chat addressed to `"all"` so every agent knows the session is ending:

```toml
[[message]]
from = "<your-agent-name>"
to = "all"
thread = "lifecycle"
type = "shutdown"
ts = "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
text = "Session complete. Shutting down."
```

### Step 2: Wait, then kill all agent panes

Wait 10 seconds for all agents to complete in-flight work and post their final messages:

```bash
sleep 10
```

Then kill all registered agents via the binary:

```bash
# Kill all running agents via binary
engram agent list | jq -r '.name' | while read -r agent_name; do
  engram agent kill --name "$agent_name"
done
```

**Why 10s wait:** The broadcast `shutdown` reaches all agents simultaneously. The 10s wait is empirically sufficient for all agents to complete in-flight work — task agents (simple planners/executors) finish fast, engram-agent (which processes final `learned` messages) finishes within 10s. This is an empirical observation, not a protocol guarantee.

### Step 3: Kill the chat tail pane

Use `-a` flag to search ALL panes across ALL windows — this works even if you're not in the coordinator window.

**Primary: kill by pane title** (`pane_title` is set to `chat-tail` by engram-tmux-lead §1.3 via `tmux select-pane -T`):

```bash
tmux list-panes -a -F '#{pane_id} #{pane_title}' \
  | grep chat-tail \
  | awk '{print $1}' \
  | xargs -I{} tmux kill-pane -t {}
```

**Fallback: kill by pane command** (handles sessions started before the title was set):

```bash
tmux list-panes -a -F '#{pane_id} #{pane_current_command}' \
  | grep tail \
  | awk '{print $1}' \
  | xargs -I{} tmux kill-pane -t {}
```

**Why title over command:** `pane_current_command` reflects the foreground process. When tmux spawns the pane via a shell (fish/zsh/bash) that then runs `tail -F`, `pane_current_command` shows `fish`, not `tail`. `pane_title` is set explicitly at spawn time and never changes — it reliably identifies the pane regardless of what foreground process is running.

**Why `-a`:** Without `-a`, `tmux list-panes` only lists panes in the currently active window. If you've navigated away from the coordinator window, the tail pane won't be found. The `-a` flag lists all panes globally, making shutdown window-independent.

### Step 4: Drain background task IDs

Prevent zombie shell accumulation in Claude Code's background task queue.

Drain **only** the background task IDs you have tracked in this session. **Skip any ID you never set.** Common IDs:

- `CHAT_MONITOR_TASK_ID` (chat file watcher — set by lead and most agents): if set, call `TaskOutput(task_id=CHAT_MONITOR_TASK_ID, block=False)`
- Hold detection task IDs (lead-only): drain each with `TaskOutput(task_id=<id>, block=False)`

Non-lead agents typically only have `CHAT_MONITOR_TASK_ID`. If you have none, skip this step.

### Step 5: Report session summary

Tell the user:
- Agents spawned (count and names)
- Tasks completed
- Memories surfaced / learned (if known from engram-agent messages)

### Step 6: Chat file

**Do NOT truncate or delete the chat file.** It persists across sessions for context continuity.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Kill engram-agent before task agents | Broadcast shutdown to all first, wait 10s — all agents get the message simultaneously and wrap up in order |
| Use `tmux list-panes` without `-a` for chat tail | Works only in current window — use `-a` so the tail pane is found regardless of which window is active |
| Grep `pane_current_command` for `tail` only | Shell (fish/zsh) is the foreground process, not `tail` — grep `pane_title` for `chat-tail` first, then fall back to `pane_current_command` |
| Skip draining background task IDs | Zombie shells accumulate across sessions — drain all IDs you have tracked |
| Drain task IDs you never set | Only drain IDs you actually set in this session — skip those that belong to other roles (e.g., lead-only IDs) |
| Truncate chat file | Chat file is persistent and append-only — never truncate |
| Post `from = "lead"` when not lead | Substitute your actual agent name from your `ready` message |
