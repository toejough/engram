---
name: engram-up
description: Use when the user says /engram, /engram-up, "start engram", or wants to begin a multi-agent orchestrated session with memory.
---

# Engram Up

Start the engram API server and initialize the lead agent.

## Startup Sequence

### Step 1: Choose agent name

Ask the user for a unique agent name (e.g., `lead-1`, `project-auth`). This will be set as `ENGRAM_AGENT_NAME` and used as `--from` in all engram CLI calls.

```bash
export ENGRAM_AGENT_NAME=<chosen-name>
```

### Step 2: Start the API server

```bash
engram server up \
  --chat-file ~/.engram/sessions/<session-name>.toml \
  --log-file ~/.engram/sessions/<session-name>.log
```

The server prints its address (`localhost:7932` by default) on startup. Keep this running — it manages the engram-agent and watches the chat file.

**Flags:**
- `--chat-file` — path to the TOML chat file (persistent across sessions)
- `--log-file` — path to the structured JSON debug log
- `--addr` — override default address (default: `localhost:7932`)

### Step 3: Open monitoring panes (tmux only)

If `$TMUX` is set:

```bash
# Tail the chat file
CHAT_PANE=$(tmux split-window -h -P -F '#{pane_id}' "tail -f <chat-file>")
tmux set-option -p -t "$CHAT_PANE" @engram_name "chat-tail"

# Tail the debug log (readable with jq)
LOG_PANE=$(tmux split-window -h -P -F '#{pane_id}' "tail -f <log-file> | jq .")
tmux set-option -p -t "$LOG_PANE" @engram_name "log-tail"
```

Skip silently if not in tmux.

### Step 4: Load skills

Load the coordination and lead skills:

```
/use-engram-chat-as
/engram-lead
```

### Step 5: Verify server

```bash
engram status
```

Expected output: server health JSON with connected agents and status.

## Expected Output

After startup, the server logs (viewable via the log-tail pane):
- Server start event with address, chat-file, log-file
- Engram-agent first invocation (session ID captured)

The lead is now ready to route work. Announce yourself via:

```bash
engram post --from "${ENGRAM_AGENT_NAME}" --to engram-agent --text "Lead ready."
```
