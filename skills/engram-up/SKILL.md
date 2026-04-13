---
name: engram-up
description: Use when the user says /engram, /engram-up, "start engram", or wants to begin a multi-agent orchestrated session with memory.
---

# Engram Up

Start the engram API server and initialize the lead agent.

## Startup Sequence

### Step 1: Choose agent name

Ask the user for a unique agent name (e.g., `lead-1`, `project-auth`). Write it to a marker file so hooks can read it. This name is used as `--from` in all engram CLI calls.

```bash
ENGRAM_DATA="${XDG_DATA_HOME:-$HOME/.local/share}/engram"
SLUG=$(echo "$PWD" | tr '/' '-')
mkdir -p "$ENGRAM_DATA/chat"
echo "<chosen-name>" > "$ENGRAM_DATA/chat/${SLUG}.agent-name"
```

Then set it locally for use in this session's CLI calls:

```bash
export ENGRAM_AGENT_NAME=<chosen-name>
```

### Step 2: Start the API server

**MCP mode:** The `engram-mcp` plugin auto-starts the API server when the first tool is called. No manual step needed. To test channel events in development, start Claude Code with:

```bash
claude --dangerously-load-development-channels 'plugin:engram@engram'
```

**CLI / manual mode:**

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
# Ensure files exist before tailing (server creates them lazily)
touch <chat-file> <log-file>

# Tail the chat file
CHAT_PANE=$(tmux split-window -h -P -F '#{pane_id}' "tail -F <chat-file>")
tmux set-option -p -t "$CHAT_PANE" @engram_name "chat-tail"

# Tail the debug log (readable with jq)
LOG_PANE=$(tmux split-window -h -P -F '#{pane_id}' "tail -F <log-file> | jq .")
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

## Troubleshooting

Debug logging is available at the server log file (specified with \`--log-file\` on \`engram server up\`). If engram is not working as expected, check the server log: \`tail -F <log-file> | jq .\`
