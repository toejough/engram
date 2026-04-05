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

Then kill all Claude panes in this tmux **session** (not all sessions), excluding your own pane:

```bash
OWN_PANE=$(tmux display-message -p '#{pane_id}')
tmux list-panes -s -F '#{pane_id} #{pane_current_command}' \
  | grep -i claude \
  | grep -v "^$OWN_PANE " \
  | awk '{print $1}' \
  | xargs -I{} tmux kill-pane -t {}
tmux select-layout main-vertical
```

**Why 10s wait:** The broadcast `shutdown` reaches all agents simultaneously. The 10s wait is empirically sufficient for all agents to complete in-flight work — task agents (simple planners/executors) finish fast, engram-agent (which processes final `learned` messages) finishes within 10s. This is an empirical observation, not a protocol guarantee.

**Why `-i` (case-insensitive grep):** `pane_current_command` shows the foreground binary name. Claude Code's CLI binary is named `claude` (lowercase), so `grep -i claude` matches it regardless of case variants. Note: if Claude Code ever runs inside a wrapper process (e.g., `node` or `sh`), this filter will not match it — that is a known limitation. In standard engram usage, agents run `claude` directly.

**Why `-s` (not `-a`):** `-s` lists panes in the current tmux session only, so you don't accidentally kill Claude panes from unrelated projects open in other sessions. The chat tail pane (Step 3) uses `-a` because it is identified by command (`tail`), not by agent identity.

**Why exclude own pane:** The caller's pane must stay alive to post the session summary (Step 5). The user reads the summary from this pane.

### Step 3: Kill the chat tail pane

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

### Step 4: Drain background task IDs

Prevent zombie shell accumulation in Claude Code's background task queue.

Drain **only** the background task IDs you have tracked in this session. **Skip any ID you never set.** Common IDs:

- `CHAT_FSWATCH_TASK_ID` (chat file watcher — set by lead and most agents): if set, call `TaskOutput(task_id=CHAT_FSWATCH_TASK_ID, block=False)`
- `HEALTH_CHECK_TASK_ID` (lead-only): if set, call `TaskOutput(task_id=HEALTH_CHECK_TASK_ID, block=False)`
- Hold detection task IDs (lead-only): drain each with `TaskOutput(task_id=<id>, block=False)`

Non-lead agents typically only have `CHAT_FSWATCH_TASK_ID`. If you have none, skip this step.

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
| Kill engram-agent before task agents | Task agents shut down first so their `learned` messages get processed |
| Use `tmux list-panes` without `-a` | Works only in current window — use `-a` for global search |
| Skip draining background task IDs | Zombie shells accumulate across sessions |
| Truncate chat file | Chat file is persistent and append-only — never truncate |
| Kill by window index or name | Always kill by tracked pane ID |
