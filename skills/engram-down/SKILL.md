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

### Step 2: Shut down task agents first

For each tracked task agent (executors, planners, reviewers, researchers):

1. Post a `shutdown` message addressed to the agent
2. Wait 5s for any final `learned` or summary messages
3. Kill its pane by tracked pane ID:
   ```bash
   tmux kill-pane -t <pane-id>
   tmux select-layout main-vertical
   ```

**Why task agents first:** They may post `learned` messages during shutdown. The engram-agent must still be alive to receive them.

### Step 3: Shut down engram-agent last

1. Post `shutdown` addressed to `engram-agent`
2. Wait 10s (longer — engram-agent processes final `learned` messages)
3. Kill its pane:
   ```bash
   tmux kill-pane -t <engram-agent-pane-id>
   tmux select-layout main-vertical
   ```

### Step 4: Kill the chat tail pane

Use `-a` flag to search ALL panes across ALL windows — this works even if you're not in the coordinator window:

```bash
tmux list-panes -a -F '#{pane_id} #{pane_current_command}' \
  | grep tail \
  | awk '{print $1}' \
  | xargs -I{} tmux kill-pane -t {}
```

**Why `-a`:** Without `-a`, `tmux list-panes` only lists panes in the currently active window. If you've navigated away from the coordinator window, the tail pane won't be found. The `-a` flag lists all panes globally, making shutdown window-independent.

If you have the chat tail pane ID tracked from startup, prefer killing by ID directly:
```bash
tmux kill-pane -t <chat-tail-pane-id>
```

### Step 5: Drain background task IDs

Prevent zombie shell accumulation in Claude Code's background task queue:

- `CHAT_FSWATCH_TASK_ID` (the chat file watcher): `TaskOutput(task_id=CHAT_FSWATCH_TASK_ID, block=False)`
- Any other tracked background task IDs (READY check loops, health check tasks)

This must be done before the session ends or zombie shells persist into the next session.

### Step 6: Report session summary

Tell the user:
- Agents spawned (count and names)
- Tasks completed
- Memories surfaced / learned (if known from engram-agent messages)

### Step 7: Chat file

**Do NOT truncate or delete the chat file.** It persists across sessions for context continuity.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Kill engram-agent before task agents | Task agents shut down first so their `learned` messages get processed |
| Use `tmux list-panes` without `-a` | Works only in current window — use `-a` for global search |
| Skip draining background task IDs | Zombie shells accumulate across sessions |
| Truncate chat file | Chat file is persistent and append-only — never truncate |
| Kill by window index or name | Always kill by tracked pane ID |
