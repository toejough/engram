---
name: engram-down
description: Use when shutting down an engram multi-agent session, when the user says "done", "shut down", "stand down", "close engram", or "stop engram". Drains in-flight dispatch work and reports session summary.
---

# Engram Down

Shutdown skill for engram multi-agent sessions. Drains in-flight dispatch work, stops agents gracefully, and reports session stats.

## Shutdown Sequence

### Step 1: Broadcast shutdown

Post `shutdown` to `"all"` (use your agent name from your `ready` message):

```toml
[[message]]
from = "lead"
to = "all"
thread = "lifecycle"
type = "shutdown"
text = "Session complete. Shutting down."
```

### Step 2: Drain in-flight work

```
engram dispatch drain --secs 30
```

Blocks until all in-flight tasks complete or timeout elapses. Do not skip — stopping before drain risks losing work.

### Step 3: Stop dispatch

```
engram dispatch stop
```

### Step 4: Scan for LEARNED messages

Before posting the session summary, read from your session-start cursor forward and collect all `LEARNED` messages. Skipping this risks posting a summary before final facts arrive.

### Step 5: Report session summary

Tell the user: agents spawned, tasks completed vs in-flight, decisions made, facts learned, open questions.

### Step 6: Preserve chat file

**Do NOT truncate or delete the chat file.** Persistent across sessions.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Skip `dispatch drain` | In-flight tasks are lost |
| Post summary before scanning LEARNED | Facts may arrive after summary — scan first |
| Truncate chat file | Never — persistent record |
| Wrong `from` name | Use your name from the `ready` message, not `lead` |
