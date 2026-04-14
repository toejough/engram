# Engram Server & MCP Design Spec

**Date:** 2026-04-12
**Status:** Draft
**Problem:** Engram's multi-agent memory system is unreliable because deterministic routing and lifecycle management are delegated to Claude agents, which forget, misunderstand, or ignore responsibilities. The lead agent doesn't reliably post to chat, doesn't reliably dispatch workers, doesn't notice messages meant for it, and spawns spurious subprocesses.

**Solution:** Move all deterministic work into a pure Go API server. The server watches the TOML chat file, manages the engram-agent (memory specialist) via `claude -p --resume`, exposes an HTTP API, and owns all validation and routing logic. Clients (CLI, MCP, hooks) are thin API consumers. The TOML chat file stays the source of truth.

---

## 1. System Overview

Four components:

1. **API server** (`engram server up`) -- pure Go long-running process. Binds to localhost only (no auth needed -- local tool). Watches TOML chat file via fsnotify. Manages per-agent goroutines. Invokes engram-agent via `claude -p --resume`. Owns all validation, routing, and skill refresh logic. Completely agnostic about which client is calling -- CLI, MCP, hooks, or anything else hitting the HTTP API.

2. **CLI client** (`engram post`, `engram intent`, `engram learn`, `engram subscribe`) -- thin HTTP client that calls the API server. Used by agents in stage 1 and by hooks in both stages. Supports both synchronous (intent, learn) and async (subscribe) interaction patterns. No validation, no chat file access, no intelligence.

3. **MCP server** (`engram mcp up`) -- thin API client providing MCP tools for interactive agents. Same API calls as CLI, different transport. Auto-starts the API server if not running (checks via `GET /status`; uses a startup lock to prevent double-start on slow startup). No validation, no chat file access, no intelligence. Replaces CLI as the lead's client in stage 2.

4. **Hooks** -- installed in the engram plugin's `hooks.json`. Call the API server via the CLI client. User-prompt hook posts user words. Stop hook posts agent output. SubagentStop hook posts subagent output.

**Key design principle:** The API server is the only component with intelligence. CLI and MCP are interchangeable thin clients over the same HTTP API. Splitting the CLI out of the server into a proper API client is the first implementation step.

**What stays the same:**

- TOML chat file remains the source of truth for all inter-agent communication
- Memory files, recall, and the memory format are unchanged

**What goes away:**

- The lead doesn't run `dispatch start` or manage agent lifecycle. Dispatch, holds, and worker management are retired (deferred to future multi-agent orchestration).
- No `chat watch` subprocesses spawned by Claude instances
- The lead doesn't need to track cursors or poll

**Future enhancements (explicitly deferred):**

- Multi-cluster (multiple TOML chat files)
- Worker agents managed by the server via `claude -p`
- Multi-agent dispatch routing and holds

---

## 2. API Server

**Command:** `engram server up [--chat-file <path>] [--log-file <path>] [--addr localhost:7932]`

The API server is completely client-agnostic. It does not know or care whether a request comes from the CLI, the MCP server, a hook, or a direct HTTP call. All clients look the same: HTTP requests. The server binds to localhost only.

### Responsibilities

1. **Chat management** -- Read/write the TOML chat file using existing `internal/chat` libraries (FilePoster, FileWatcher). The server is the primary writer; external tools still work since FilePoster uses file-level locking. Note: external tools writing directly to the chat file will not trigger goroutine notifications (only the server's own fsnotify watch does).

2. **Per-agent goroutines** -- Each agent goroutine maintains its own cursor position in the chat file, reads new messages when notified, and decides what to do. Goroutines own all validation of messages to/from their agent. The **engram-agent goroutine is special**: it watches ALL messages in the chat file (not just those addressed to it), since it is the memory specialist and needs full visibility to make surfacing/learning judgments.

3. **Chat file watching and fan-out** -- A single fsnotify watcher monitors the TOML file. On change, it broadcasts to all agent goroutines via buffered channels (buffer=1, "something changed" signal). Goroutines read from their cursor, advance it, and act on relevant messages. Goroutines busy with a `claude -p` call will see the buffered notification when they finish their current work.

4. **Engram-agent goroutine** -- Dedicated goroutine managing the engram-agent lifecycle (see section 5).

5. **Skill refresh tracking** -- The server tracks interaction count independently per agent. Every 13 invocations of the engram-agent, the server prepends skill reload instructions directly into the prompt. Every 13 messages delivered to the lead, the server posts a skill refresh message to the chat file addressed to the lead (e.g., "Reload your engram skills: `/use-engram-chat-as` and `/engram-lead`."). Skill refresh is entirely server-side -- clients don't need to know about it. The counter is named as a constant (e.g., `const skillRefreshInterval = 13`).

### HTTP API Endpoints

- `POST /message` -- Post a message to chat. Validated by the target agent's goroutine before acceptance. Returns cursor position on success, error with guidance on validation failure.
- `GET /wait-for-response?from=<agent>&to=<agent>&after-cursor=N` -- Long-poll for a response message. Independently watches the chat file from the provided `after-cursor` position, scanning for a message matching the `from`/`to` filter. Does NOT rely on any goroutine's cursor state (avoids TOCTOU race). Returns the matching message when found.
- `GET /subscribe?agent=<name>&after-cursor=N` -- Long-poll subscription for messages addressed to the named agent, starting from the given cursor position. Returns new messages when they arrive. Cursor-based (resumable) -- clients pass the cursor from the previous response to avoid missing messages between long-poll cycles.
- `GET /status` -- Server health, connected agents, invocation counts. Returns JSON.
- `POST /shutdown` -- Graceful shutdown.
- `POST /reset-agent` -- Force session reset on engram-agent.

### Debug Logging

Structured slog output (JSON handler) to stderr and optionally to `--log-file`.

Events logged:

| Event                          | Fields                                                     |
| ------------------------------ | ---------------------------------------------------------- |
| Server start/stop              | addr, chat-file, log-file                                  |
| Message posted to chat         | from, to, text length, cursor after                        |
| Message validation failed      | from, to, error, attempt number                            |
| Engram-agent invoked           | session-id, turn number, is-skill-refresh                  |
| Engram-agent responded         | session-id, duration, action type, memories surfaced count |
| Engram-agent malformed output  | session-id, raw output, re-prompt attempt number           |
| Engram-agent error             | session-id, error, consecutive-failure-count               |
| Engram-agent session reset     | old session-id, reason                                     |
| Engram-agent escalation        | full failure log                                           |
| Learn validation failure       | from, error, attempt number                                |
| Learn fallback to engram-agent | raw content                                                |
| Skill refresh triggered        | agent, turn number                                         |

Viewable via: `tail -f <log-file> | jq .`

---

## 3. API Clients

CLI and MCP are both thin API clients. They translate between their transport (command-line args or MCP protocol) and the HTTP API. They contain no validation, no routing logic, and no chat file access.

### CLI Client

The CLI is rewritten as a thin HTTP client to the API server. In stage 0, new commands are added alongside the old ones. In stage 1.5, the old commands (`engram chat post`, `engram dispatch start`, `engram agent spawn`, etc.) and their backing internal code are deleted.

**New commands:**

- `engram post --from <name> --to <name> --text <text> [--addr <api-addr>]` -- Calls `POST /message`. Prints cursor on success, error on failure. Fire-and-forget.
- `engram intent --from <name> --to <name> --situation <text> --planned-action <text> [--addr <api-addr>]` -- Calls `POST /message` (gets cursor N), then blocks on `GET /wait-for-response?after-cursor=N`. Prints surfaced memories. Synchronous. Positional correlation: since the engram-agent is serialized, only one response can appear between cursor N and the next matching message.
- `engram learn --from <name> --type feedback|fact --situation <text> [content fields] [--addr <api-addr>]` -- Calls `POST /message`. Prints success or validation error with format guidance.
- `engram subscribe --agent <name> [--after-cursor <N>] [--addr <api-addr>]` -- Long-polls `GET /subscribe`. Prints messages as they arrive. Tracks cursor for resumable delivery.
- `engram status [--addr <api-addr>]` -- Calls `GET /status`. Prints server health.

All CLI commands pass through API responses unmodified, including validation errors. The agent (or hook) calling the CLI sees exactly what the server returned.

### MCP Server

**Command:** `engram mcp up [--api-addr localhost:7932]`

**Startup:** Checks if the API server is running via `GET /status`. If not, starts it as a subprocess using a startup lock file to prevent double-start. Registered as a Claude Code MCP server in the plugin config.

**MCP Tools:**

1. **`engram_post`** -- Calls `POST /message`. Returns cursor on success, API error on failure.
2. **`engram_intent`** -- Calls `POST /message` + `GET /wait-for-response`. Returns surfaced memories synchronously. Same two-step as CLI `engram intent`.
3. **`engram_learn`** -- Calls `POST /message`. Returns success or validation error with format guidance.
4. **`engram_status`** -- Calls `GET /status`.

**Async push:** MCP server long-polls `GET /subscribe?agent=<name>&after-cursor=N` (cursor-based, resumable) and pushes messages to the lead agent as context on its next turn.

Skill refresh is handled entirely by the server (posts refresh messages to chat). Clients don't need to know about it.

---

## 4. Hooks

Installed in `hooks/hooks.json` as part of the engram plugin. Hooks call the API server via the CLI client.

### Hook 1: User-prompt hook (UserPromptSubmit)

- Fires when the user submits a prompt
- Stage 1: calls `engram intent --from <agent-name>:user --to engram-agent --situation <full prompt>` (blocks, returns surfaced memories as additionalContext)
- Stage 2: calls `engram post --from <agent-name>:user --to engram-agent --text <full prompt>` (fire-and-forget, memories arrive via MCP push)
- Agent name configured during `/use-engram` setup -- user chooses a unique name

### Hook 2: Stop hook (Stop)

- Fires when the agent completes a turn
- Stage 1: calls `engram intent --from <agent-name> --to engram-agent --situation <full agent output>` (blocks, returns surfaced memories as additionalContext)
- Stage 2: calls `engram post --from <agent-name> --to engram-agent --text <full agent output>` (fire-and-forget)

### Hook 3: SubagentStop hook

- Posts subagent output to chat, addressed to engram-agent (the lead already sees subagent output natively -- addressing to the lead would be confusing double-messaging)
- `engram post --from <agent-name>:subagent:<subagent-id> --to engram-agent --text <full subagent output>`

### What hooks do NOT do

- No skill reinforcement (API server tracks this and includes reminders in responses)
- No agent lifecycle management
- No validation (API server handles that)

### Known trade-offs

**Stage 1 latency:** Every user prompt and agent turn blocks on a `claude -p --resume` round-trip (typically 3-15 seconds). The user sees this as a pause before the lead agent responds. This is the cost of synchronous memory surfacing. Stage 2 eliminates this by switching to fire-and-forget hooks with async MCP push.

**Stage 2 timing:** In stage 2, hook-triggered memories arrive after the agent has started (or finished) its current turn. Memories are supplementary context for the next turn. For time-critical surfacing, the agent should use `engram_intent` explicitly before significant actions (Flow 3, synchronous).

---

## 5. Engram-Agent Management

The server's most important job -- managing the engram-agent that does the actual memory work.

### Invocation

The server builds its own `exec.Cmd` following the same pattern as the existing `buildClaudeCmd` in `internal/cli/cli_agent.go`. Flags: `-p --dangerously-skip-permissions --verbose --output-format=stream-json` (+ `--resume <session-id>` on subsequent calls). Note: the existing `buildClaudeCmd` is unexported and tightly coupled to CLI state file machinery, so the server implements its own equivalent (same 3-line pattern) rather than importing it.

### Stream Parsing

The server needs a **new stream parser** for the engram-agent's output. The existing `claude.Runner.ProcessStream` is designed for speech-marker-based protocol (`INTENT:`, `ACK:`, `WAIT:` markers), not structured JSON. The new parser reads the `stream-json` JSONL envelope, extracts assistant text events, and parses the inner structured JSON (`{"action": "surface", ...}`). This is a new function in the server package, not a reuse of `ProcessStream`.

### Lifecycle

1. **First invocation:** `claude -p --output-format=stream-json` with prompt loading `/use-engram-chat-as` and `/engram-agent`. Captures session ID from stream-json response.
2. **Subsequent invocations:** `claude -p --resume <session-id> --output-format=stream-json` with the new message from the chat.
3. **Every 13th invocation:** Prepend skill reload instructions to the prompt.
4. **Serialization:** One `claude -p` call at a time. Requests wait in the goroutine. Prevents interleaved conversation context. If requests queue faster than the engram-agent can process, the queue grows. The server logs queue depth but does not drop messages.

### Structured Output Contract

The engram-agent responds with structured JSON:

```json
{"action": "surface", "to": "lead-1", "text": "Relevant memory: ..."}
{"action": "log-only", "text": "Nothing relevant found."}
{"action": "learn", "saved": true, "path": "facts/...", "to": "lead-1"}
```

Server reads `action` and routes mechanically:

- `surface` -- post to chat addressed to `to` field (the requesting agent)
- `log-only` -- post to chat with `to=log` (sentinel; no goroutine claims this, file-only record). No notifications.
- `learn` -- post outcome to chat addressed to `to` field (the agent that sent the learning), log the decision

### Unified Error/Recovery Protocol

Any engram-agent failure (execution error, timeout, malformed output) follows the same ladder:

1. **Retry on same session (up to 3 times)** -- for malformed output, re-prompt asking for proper structure and pointing at the skill file. For execution errors, retry the same invocation.
2. **Session reset** -- kill session, start fresh with full skill load. Feed the last 3 chat messages that were sent to the engram-agent as initial context. Ask it to evaluate the most recent message and respond.
3. **Retry on fresh session (up to 3 times)** -- same as step 1 on the new session.
4. **Escalate** -- post error to chat addressed to the lead. Log full failure sequence to debug log as critical error. Stop invoking engram-agent until user intervenes (server stays up).

Total: 7 attempts before giving up (3 on original session + 3 on fresh session + the initial attempt that triggered the recovery).

### Session Reset Triggers

- Exhausted retries on current session (step 2 above)
- Explicit server restart
- Admin API call (`POST /reset-agent`)

---

## 6. Validation

All validation lives in agent goroutines in the API server. Clients (CLI or MCP) pass through errors unmodified to the agent.

### Learn Message Validation (Lead Goroutine)

When the lead goroutine receives a learn message via `POST /message`:

1. Validate structured content fields against memory format. Feedback requires: situation, behavior, impact, action. Fact requires: situation, subject, predicate, object.
2. If valid -- accept, post to chat, return success.
3. If malformed -- reject with HTTP error. Include format guidance pointing to the skill file. Client (CLI or MCP) passes error to agent. Agent retries.
4. Up to 3 rejected attempts.
5. After 3 failures -- accept the raw content, post to chat addressed to engram-agent as best-effort. Log as error. Engram-agent does its best to interpret and file.

### Engram-Agent Output Validation

See unified error/recovery protocol in section 5.

### Connected Agent Names

Every connected agent must have a unique name. The `GET /subscribe` endpoint serves as the implicit connection point -- subscribing with a name registers the agent. If two agents try to subscribe with the same name, the second is rejected. Hooks (which don't subscribe) are not subject to name uniqueness.

---

## 7. Interaction Flows

**Core mechanism:** All communication flows through the TOML chat file. The API server watches the file via a single fsnotify watcher that fans out to agent goroutines via buffered channels. Each agent goroutine maintains its own cursor position, reads new messages, and decides what to do. The engram-agent goroutine watches ALL messages; other goroutines watch only messages addressed to them. Clients (CLI, MCP, hooks) are thin API consumers -- they never touch the chat file.

### Flow 1: User sends a prompt

1. User types in Claude Code
2. `UserPromptSubmit` hook fires -- calls `engram intent` (stage 1) or `engram post` (stage 2) targeting the API server (`from=<agent>:user`, `to=engram-agent`, `text=<full prompt>`)
3. CLI/MCP client calls `POST /message`. API server appends message to TOML chat file, returns cursor N.
4. API server's fsnotify watcher detects file change, broadcasts to all agent goroutines
5. Engram-agent goroutine reads from its cursor, sees the new message, advances cursor
6. Engram-agent invoked via `claude -p --resume` with the new message -- applies judgment: surface memories, extract learnings, both, or nothing
7. Engram-agent responds with structured JSON. Server validates (recovery protocol if malformed). On `surface` action -- posts response to chat file addressed to the lead
8. API server's fsnotify watcher detects file change, broadcasts to goroutines
9. Lead goroutine reads from its cursor, sees message addressed to it, advances cursor
10. Stage 1: CLI `engram intent` was blocking on `GET /wait-for-response?after-cursor=N`. The endpoint independently watches the file from cursor N, finds the matching response, returns it. Hook receives surfaced memories as additionalContext. Stage 2: MCP server's `GET /subscribe` loop picks up the message and pushes to the agent as context on its next turn.

### Flow 2: Agent completes a turn

1. `Stop` hook fires -- calls `engram intent` (stage 1) or `engram post` (stage 2) targeting the API server (`from=<agent>`, `to=engram-agent`, `text=<full agent output>`)
2. Same as flow 1 steps 3-10

### Flow 3: Agent explicitly announces intent (synchronous surfacing)

1. Agent calls `engram intent` (CLI) or `engram_intent` (MCP tool)
2. Client calls API server `POST /message` (intent message, `from=<agent>`, `to=engram-agent`), receives cursor N
3. API server appends to chat file, fsnotify broadcasts to goroutines
4. Engram-agent goroutine reads the intent, invokes `claude -p --resume`, returns structured response
5. Server validates, posts response to chat file addressed to requesting agent
6. Client is blocking on `GET /wait-for-response?after-cursor=N&from=engram-agent&to=<agent>`. The endpoint independently watches the file from cursor N, finds the matching response, returns it.
7. Surfaced memories returned to the agent

### Flow 4: Agent announces a learning

1. Agent calls `engram learn` (CLI) or `engram_learn` (MCP tool) with structured content fields
2. Client calls API server `POST /message`
3. Lead goroutine validates structured fields against memory format
4. If valid -- API accepts, posts to chat, returns success
5. If malformed -- API returns error with format guidance pointing to skill file. Client passes error to agent. Agent retries (up to 3 times).
6. If still malformed after 3 attempts -- API accepts raw content, posts to chat addressed to engram-agent as best-effort. Logs error.
7. Engram-agent goroutine reads the learning (it watches all messages), invokes `claude -p --resume`, does its best to interpret and either persists as memory file or discards

### Flow 5: Subagent output

1. `SubagentStop` hook fires -- calls `engram post` targeting the API server (`from=<agent>:subagent:<id>`, `to=engram-agent`, `text=<full output>`)
2. API server appends to chat file, fsnotify broadcasts to goroutines
3. Engram-agent goroutine reads the message (it watches all messages), applies judgment via `claude -p --resume`
4. Engram-agent responds with structured JSON. If it has something to surface -- server posts response to chat addressed to the lead that spawned the subagent (extracted from the `from` field prefix)
5. API server detects change, lead goroutine reads the response, advances cursor
6. Client receives the response via `GET /subscribe` (MCP push) or `engram subscribe` (CLI)

### Flow 6: Engram-agent error recovery

1. Server invokes `claude -p --resume` and gets an error (non-zero exit, timeout, no output, malformed structured output)
2. Follow unified error/recovery protocol (section 5): 3 retries on session, session reset with last 3 messages, 3 retries on fresh session, escalate
3. On escalation: post error to chat addressed to lead, log as critical error, stop invoking engram-agent

---

## 8. Skill Updates (Incremental)

Skills update incrementally as capabilities land. Each stage describes what changes to match what the system can now do. Note: the `engram-agent` skill is a **near-complete rewrite** in stage 1 (not an incremental edit). The current skill's self-directed model (cursor management, startup sequence, rate limiting, speech markers) is entirely replaced by the server-driven model. Treat it as writing a new skill from scratch.

### Stage 0: CLI client split

Add new CLI commands (`engram post`, `engram intent`, `engram learn`, `engram subscribe`, `engram status`) as HTTP API clients alongside the existing commands. No skill changes at this stage.

### Stage 1: API server + CLI client + hooks

**`engram-agent` rewritten:**

- Remove entirely: self-watch loop, cursor management, startup sequence, INTENT:/ACK:/WAIT: speech markers, rate limiting, resume-context parsing
- Agent is now invoked by the server via `claude -p --resume`
- Add structured JSON output contract: every response must be `{"action": "surface"|"log-only"|"learn", "to": "...", "text": "..."}`
- Preserve: memory judgment logic (what to surface, what to learn, what to ignore), failure correlation, memory quality evaluation
- Document that the server will re-prompt if output isn't structured, and will ask to reload this skill if issues persist

**`use-engram-chat-as` rewritten:**

- Remove entirely: subprocess-watching language (`engram chat watch`, cursor tracking, polling), INTENT:/ACK:/WAIT:/DONE: marker protocol, argument protocol (3-turn cap, ESCALATE), RESUME_REASON handling
- Replace with: the new interaction model -- hooks handle posting user/agent output, agents use CLI commands for explicit interaction
- Describe the new learn message format: structured fields matching memory TOML (feedback: situation/behavior/impact/action, fact: situation/subject/predicate/object)
- Agents interact with the API server via CLI (`engram post`, `engram learn`, `engram intent`)
- Note: the old ack/wait protocol is retired, not deferred. The server-mediated model replaces it.

**`engram-lead` updated:**

- Remove: dispatch management, hold patterns, compaction recovery via dispatch status
- Lead uses CLI commands (`engram intent`, `engram learn`, `engram post`) for all engram interaction
- Lead should call `engram intent` before significant actions
- Lead should call `engram learn` with properly structured content after learning something
- Lead spawns subagents via Claude Code's native subagent mechanism (not dispatch). Their output reaches engram via SubagentStop hooks.
- Note: holds, fan-in coordination, and worker lifecycle are retired. Subagent coordination uses Claude Code's built-in subagent features.

**`engram-up` rewritten (currently a ~5 line stub):**

- Startup sequence: `engram server up --chat-file <path> --log-file <path>`, then load skills via `/use-engram-chat-as` and `/engram-lead`
- If inside tmux: open a pane tailing the chat file, open a pane tailing the debug log
- Document the server flags and expected output

**`engram-down` updated:**

- Shutdown: `engram post --from <agent> --to engram-agent --text "shutdown"`, then `engram server` POST /shutdown
- If inside tmux: kill the chat tail and debug log tail panes
- Remove: dispatch drain, dispatch stop, LEARNED message scan (retired with marker protocol)

### Stage 1.5: Retire old CLI and dispatch

Delete old commands (`engram chat post`, `engram chat watch`, `engram chat cursor`, `engram chat ack-wait`, `engram dispatch start/assign/drain/stop/status`, `engram agent spawn/kill/list/wait-ready/run`, `engram hold acquire/release/list/check`) and their backing internal code. The new CLI commands are the only interface. No skill changes -- skills were already updated in stage 1 to use the new commands.

### Stage 2: MCP server

**`engram-lead` updated:**

- Replace CLI commands with MCP tools (`engram_intent`, `engram_learn`, `engram_post`)
- Document bidirectional MCP communication -- memories arrive as pushed context between turns. This is supplementary; for time-critical surfacing, use `engram_intent` explicitly.
- Document that skill reload reminders will arrive every 13 interactions -- comply by reloading `/use-engram-chat-as` and `/engram-lead` via the Skill tool.

**`engram-up` updated:**

- Startup sequence: start MCP server (auto-starts API server), then load skills
- Tmux panes unchanged

**`engram-down` updated:**

- Shutdown: post shutdown message via MCP instead of CLI
- Tmux pane cleanup unchanged

### Stage 3: Observability tuning

- All skills updated with note: "Debug logging is available at `<log-file>`. If engram isn't working as expected, check the server log."
- `engram-agent` updated if structured output contract needs adjustment based on real-world usage

### Skill Refresh Protocol

Skill refresh is entirely server-side for both agents:

**For the engram-agent:** every 13th invocation, the server prepends the skill reload instructions directly into the `claude -p` prompt.

**For the lead agent:** every 13 messages delivered to the lead (counted by the lead's goroutine), the server posts a skill refresh message to the chat file addressed to the lead: "Reload your engram skills: `/use-engram-chat-as` and `/engram-lead`." This reaches the lead like any other chat message -- via `GET /subscribe` (MCP push or CLI subscribe). The lead uses the Skill tool to reload. No client-side logic needed.

Both counters are per-agent and independent. This is documented in `use-engram-chat-as` and `engram-lead`.
