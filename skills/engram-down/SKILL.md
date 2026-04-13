---
name: engram-down
description: Use when shutting down an engram multi-agent session, when the user says "done", "shut down", "stand down", "close engram", or "stop engram".
---

# Engram Down

Shutdown sequence for engram sessions.

## Shutdown Sequence

### Step 1: Signal the engram-agent

Post a shutdown message so engram-agent can finalize any in-flight memory work.

**MCP mode:**
```
engram_post(from=<agent-name>, to="engram-agent", text="shutdown")
```

**CLI mode:**
```bash
engram post \
  --from "${ENGRAM_AGENT_NAME}" \
  --to engram-agent \
  --text "shutdown"
```

### Step 2: Stop the API server

```bash
curl -s -X POST http://localhost:7932/shutdown
```

Or use the CLI if it exposes a shutdown command:

```bash
engram server down
```

The server drains the engram-agent queue before stopping.

### Step 3: Kill monitoring panes (tmux only)

If `$TMUX` is set, kill the chat-tail and log-tail panes:

```bash
if [ -n "$TMUX" ]; then
  tmux list-panes -a -F '#{pane_id} #{@engram_name}' \
    | grep -E 'chat-tail|log-tail' \
    | awk '{print $1}' \
    | xargs -I{} tmux kill-pane -t {}
fi
```

Uses `@engram_name` (tmux user option, immune to OSC 2 terminal overwrites).

### Step 4: Report session summary

Tell the user: subagents spawned, tasks completed, decisions made, facts learned, open questions.

### Step 5: Preserve chat file

**Do NOT truncate or delete the chat file.** It is persistent across sessions and is the source of truth for all past communication.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Skipping the shutdown post | engram-agent may have queued work; signal first |
| Truncating the chat file | Never — persistent record |
| Using `pane_title` for tail-pane lookup | Use `@engram_name` — terminal OSC 2 overwrites `pane_title` |
| Killing panes outside tmux | Check `$TMUX` first |

## What Is Retired

The following steps are **retired** and must not be run:
- `engram dispatch drain`
- `engram dispatch stop`
- Scanning for `LEARNED:` messages (marker protocol is retired)

## Troubleshooting

Debug logging is available at the server log file (specified with \`--log-file\` on \`engram server up\`). If engram is not working as expected, check the server log: \`tail -f <log-file> | jq .\`
