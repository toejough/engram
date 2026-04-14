# Engram Deterministic Coordination Design

**Session:** codesign-1 + codesign-2 (2026-04-05)
**Planners:** planner-1 through planner-8 (4 per session)
**Status:** Converged. Implementation unblocked.

---

## 1. Problem Statement

### Root Cause

The current engram skills implement coordination using LLM agents as event loops. This is the wrong abstraction. LLMs are called when there is work; event loops, cursor tracking, state machines, and timeout logic are deterministic algorithms. When LLMs implement these in natural language, the following failures occur:

- **Context compaction loses cursor state** → missed messages, silent protocol violations
- **Race conditions between monitor agents** → duplicate or lost ACKs
- **Protocol described in English is ambiguous** → violations, edge case misses
- **Background "monitor" agents become zombies** → no cleanup, increasing noise
- **Workers run persistent event loops** → context consumed for polling, not work
- **Chat writes fragile** → shlock (macOS-only), stale timestamps, lock races

### The Mismatch

Current skills: ~70% mechanical algorithms, ~30% semantic judgment. The 70% must move to Go.

The binary should own:
- All I/O to chat files (atomic, locked, correct TOML format)
- File watching (blocking, kernel-driven — not shell sleep loops)
- Online/offline detection (timestamp comparison)
- ACK-wait (composite: watch + response parsing + timeout logic)
- Agent lifecycle (spawn, resume, kill, state tracking)
- Hold registry (acquire, release, list)
- Speech-to-chat delivery (intercept JSONL stream, post to chat on agent's behalf)

Skills should own:
- Semantic judgment (when to post intent, what to write in it)
- Routing decisions (which agent gets which task)
- Argument content (how to argue, when to concede)
- Memory relevance (what memories to surface)

### Open Issues Addressed

| Issue | Description | Phase |
|-------|-------------|-------|
| #471 | Dropped Go binary functionality | 1 |
| #471/#502 | ACK-wait via bash sleep polling | 2 (depends on Phase 1) |
| #502 | ACK-wait subagent race conditions | 2 (depends on Phase 1) |
| #503 | Lead doesn't auto-ACK spawn intents | 2+3 |
| #505/#506 | Spawn window split bugs | 3 |
| #500 | engram-down can only kill agents from lead pane | 3 |
| #494 | Lead UX noise (raw JSONL in pane) | 4+5 |
| "forgot to post" class | Agents forgetting to call `engram chat post` | 4 — reformulated (must say INTENT: prefix instead; failure mode changes but is not eliminated) |

---

## 2. Architecture

### Binary/Skill Boundary

Three categories:

| Category | Meaning | Example |
|----------|---------|---------|
| **[B]** | Binary implements the mechanism; skill describes calling convention or behavior | `engram chat post`, hold acquire/release |
| **[S]** | Skill owns content and judgment; no binary equivalent | Routing decision, argument content |
| **[B+]** | Binary intercepts from agent speech; skill describes *intent*, not how to deliver | Speech-to-chat: agent says `INTENT: ...` in output; binary posts to chat |

[B+] is the correct architectural endpoint: skills are purely cognitive (what to say, when, why). Binary handles all delivery, timing, protocol mechanics. Skills contain **zero bash code** after Phase 6 + speech-to-chat.

### High-Level Split

```
Binary (new)                           Skill (simplified)
──────────────────────────────────────  ─────────────────────────────────
engram chat post/watch/ack-wait         Intent protocol: when, to whom
engram hold acquire/release/list        Argument content: how to argue
engram agent spawn/kill/wait-ready      Memory relevance judgment
engram agent resume (stream pipeline)   Routing table (request classification)
engram agent list                       Role prompt templates
JSONL display filter (pane output)      Hold pattern selection
Speech-to-chat relay (INTENT: etc.)     Escalation surfacing to user
State file (pane-id, session-id)
Online/offline detection
Heartbeat (on binary's behalf)
Health check timestamps
```

### Lead vs Worker Model

**Lead** = interactive terminal, has user conversation, never headless.
**Workers** = `-p` headless, driven by binary, no persistent event loop.

Workers receive a task via `-p --resume`, do the work, post result to chat (via [B+] speech-to-chat), and exit. Binary is the event loop. Workers never poll or run background watchers. This eliminates the zombie monitor agent problem entirely.

Tmux panes are **kept for observability** — users can watch agent work in real time. Workers are driven via `-p --resume` (not by sending keystrokes). The pane shows filtered output (no raw JSONL).

---

## 3. New Binary Commands (Full API)

All commands follow the existing DI/pure-Go/no-CGO principles. Stdout encoding for all outputs (exit 0 always, errors to stderr — ARCH-6 preserved).

### 3.1 Chat Commands

#### `engram chat post`

```
engram chat post \
  --from <name> \
  --to <name[,name...]> \
  --thread <thread> \
  --type <intent|ack|wait|info|done|learned|ready|shutdown|escalate> \
  --text <content>
```

- Derives chat file from CWD using `ProjectSlugFromPath` + `DataDirFromHome`
- Appends with lock (Go `os.OpenFile(O_CREATE|O_EXCL)` on `.lock` file, compatible with existing bash shlock convention)
- Generates `ts` fresh at write time (not from caller)
- Outputs new line count (the new cursor)

Binary is sole writer from Phase 1 onward. Skills stop writing directly and call `engram chat post` (or [B+] speech relay eliminates the call entirely).

#### `engram chat watch`

```
engram chat watch \
  --agent <name> \
  --cursor <line-number> \
  [--type <message-type>]
```

- Blocks until a message for `<name>` (or `"all"`) arrives after `<cursor>`
- Uses `fsnotify` (pure Go, kqueue on macOS, inotify on Linux — no external deps)
- Outputs: JSON object on stdout — `{"type":"WAIT","from":"engram-agent","cursor":1234,"text":"..."}`
- Exits when message found; caller re-invokes with new cursor

**Why JSON:** Chat text regularly contains pipe characters (code, commands, paths). A pipe-delimited output format (`TYPE|from|cursor|text`) is unparseable when text contains `|`. JSON avoids this because the text field is properly quoted and escaped. Callers parse with `jq` or any JSON library.

Replaces: `fswatch -1` + tail + grep pattern in all skills.

#### `engram chat ack-wait`

```
engram chat ack-wait \
  --agent <name> \
  --cursor <line-number> \
  --recipients <name[,name,...]> \
  [--max-wait <seconds>]
```

- Blocks until all recipients respond with `ack` or `wait`
- Handles online/offline detection internally (15-min window scan)
- **Offline recipient** (no message in last 15 min): 5s implicit ACK
- **Online + silent** (message in last 15 min, no response after 30s): escalate
- Outputs: JSON object on stdout:
  - `{"result":"ACK","cursor":1234}` — all recipients acknowledged
  - `{"result":"WAIT","from":"engram-agent","cursor":1234,"text":"..."}` — one recipient objected

Replaces: ACK-wait subagent pattern (~35 lines in use-engram-chat-as).

**Note on cursor:** Caller must capture cursor *before* posting the intent, then pass it to ack-wait. This ensures ACKs posted between intent-post and ack-wait-start are not missed.

### 3.2 Hold Commands

Hold state is chat-file-native: holds are TOML messages in the chat file. Binary scans for unmatched acquire/release pairs.

**Hold message types:** `engram hold acquire` posts a message with `type = "hold-acquire"` and `engram hold release` posts `type = "hold-release"`. These are new message types extending the catalog in section 2. The `Watcher` interface's `msgTypes []string` filter handles them transparently (unknown types pass through `slices.Contains` correctly). The `Message.To` field carries the `--target` agent name; `Message.From` carries the `--holder` agent name; `Message.Text` carries the hold-id and condition as JSON.

#### `engram hold acquire`

```
engram hold acquire \
  --holder <agent-name> \
  --target <agent-name> \
  [--condition <condition>]
```

- Appends a hold record to chat file
- Condition DSL: `done:<agent>`, `first-intent:<agent>`, `lead-release:<tag>`
- Outputs: `<hold-id>` (UUID)

#### `engram hold release`

```
engram hold release <hold-id>
```

- Posts matching release record to chat file
- Outputs: `OK`

#### `engram hold list`

```
engram hold list \
  [--holder <name>] \
  [--target <name>]
```

- Scans chat file for unmatched acquire/release pairs
- Outputs: one JSON object per line — `{"hold-id":"...","holder":"...","target":"...","condition":"...","acquired-ts":"..."}`

#### `engram hold check`

```
engram hold check
```

- Scans chat file for all active (unreleased) holds with conditions
- Evaluates each condition against the current chat state:
  - `done:<agent>` — auto-releases if a `done` message from `<agent>` exists after the hold's `acquired-ts`
  - `first-intent:<agent>` — auto-releases if any `intent` message from `<agent>` exists after the hold's `acquired-ts`
  - `lead-release:<tag>` — never auto-releases; requires explicit `engram hold release <hold-id>` from the lead
- Posts release records for any conditions that are met
- Outputs: one `{"hold-id":"...","result":"released"|"still-held","reason":"..."}` per evaluated hold

**When `engram hold check` is called automatically:**
- By `engram agent kill` after removing an agent (triggers `done:<agent>` evaluation for that agent)
- By `engram agent resume` at the end of each turn, after processing the JSONL output (Phase 5+)
- Manually by the lead at any time

**Known limitation:** `engram hold check` is O(n) in chat file size. For the current use case (moderate session lengths), this is acceptable. As chat files grow unboundedly across sessions, condition evaluation over old sessions' holds may produce false positives if agent names are reused. Phase 6 may address this with session-scoped hold evaluation.

### 3.3 Agent Commands

#### `engram agent spawn`

```
engram agent spawn \
  --name <agent-name> \
  --prompt <text> \
  [--model <model-id>] \
  [--resume <session-id>]
```

- Creates tmux pane (handles column split logic per Section 2.4 of engram-tmux-lead)
- Launches: `claude -p --verbose --output-format=stream-json "<prompt>"`
- Pipes stdout through display filter + speech relay (Phase 4 addition)
- Captures `session_id` from JSONL stream (top-level field on all events)
- Writes to state file: `~/.local/share/engram/state/<slug>.toml` → `{name, pane-id, session-id, state, spawned-at}`
- Auto-posts spawn intent to engram-agent (fixed template): "About to spawn `<name>` with task `<task-summary>`"
- Waits for ACK before spawning (5s implicit ACK if engram-agent offline)
- Outputs: `<pane-id>|<session-id>`

Replaces: SPAWN-PANE definition + bash column logic in engram-tmux-lead.

#### `engram agent wait-ready`

```
engram agent wait-ready \
  --name <agent-name> \
  --cursor <line-number> \
  [--max-wait <seconds>]
```

- Watches chat for `ready` or init-complete `info` message from `<agent-name>`
- Outputs: `READY|<cursor>` or `TIMEOUT`

(Alternatively implementable as: `engram chat watch --type info --agent <name> --cursor <n>`)

**watchDeadline contract:** `--max-wait` MUST compute `watchDeadline = time.Now().Add(maxWait)` and pass it via `context.WithDeadline(ctx, watchDeadline)` to the inner `Watch` call. It is NOT sufficient to check the deadline in the outer loop only — the inner `fsnotify` loop will block indefinitely if the deadline is not threaded all the way to the deepest blocking I/O. The #519 fix (b22dc0c) established this pattern for `ack-wait`; `wait-ready` must apply it by design, not rediscover it in production.

**watchDeadline invariant (applies to all `--max-wait` commands):**
> Any binary command that accepts a `--max-wait` duration flag MUST propagate the computed deadline via `context.WithDeadline` to ALL blocking I/O operations, including inner Watch loops. This is a correctness invariant.

#### `engram agent kill`

```
engram agent kill <agent-name>
```

- Reads state file to get pane-id
- Posts done intent to chat (fixed template), waits for engram-agent ACK
- Kills tmux pane
- Removes from state file
- Outputs: `OK`

Callable from any pane — fixes #500 (engram-down previously could only kill from lead pane).

#### `engram agent resume`

```
engram agent resume \
  --session-id <id> \
  --message <text>
```

- Runs: `claude -p --resume <id> "<message>"`
- Pipes stdout through display filter + speech relay (Phase 4+)
- Captures new session-id for next resume
- Outputs: `<new-session-id>` (or same if unchanged)

#### `engram agent list`

```
engram agent list
```

- Reads state file
- Outputs: one JSON object per line — `{"name":"exec-1","state":"running","session-id":"abc","pane-id":"main:1.2"}`

**Why NDJSON:** Skills parse binary output with `jq`. Pipe-delimited fields are fragile (field values may contain pipes; `cut`/`awk` is inconsistent across platforms). NDJSON matches `hold list` output format and uses standard escaping. Lesson from #516: ship NDJSON from day one.

---

## 4. New Go Packages

### 4.1 `internal/chat`

Pure domain package. No I/O.

```go
// Message is a single chat message.
type Message struct {
    From   string
    To     string
    Thread string
    Type   string
    TS     time.Time
    Text   string
}

// Poster appends messages to the chat file atomically.
type Poster interface {
    Post(msg Message) (newCursor int, err error)
}

// Watcher blocks until a matching message arrives after cursor.
type Watcher interface {
    Watch(ctx context.Context, agent string, cursor int, msgTypes []string) (Message, int, error)
}

// AckWaiter blocks until all recipients respond.
type AckWaiter interface {
    AckWait(ctx context.Context, agent string, cursor int, recipients []string) (AckResult, error)
}

type AckResult struct {
    AllAcked  bool
    Wait      *WaitResult
    NewCursor int
}

// WaitResult holds the content of a WAIT response.
// Fields correspond to the JSON output defined in section 3.1: {"result":"WAIT","from":"...","cursor":N,"text":"..."}.
type WaitResult struct {
    From   string // agent name that posted the wait
    Text   string // wait message content
    Cursor int    // new cursor after the wait message
}

// LockFile is injected for lock-file creation (O_CREATE|O_EXCL semantics).
// Compatible with bash shlock convention. Concrete os.OpenFile call lives at the CLI edge, not here.
type LockFile func(name string) (unlock func() error, err error)

// ParseMessages parses all messages from TOML data. Fast path — assumes well-formed TOML.
// Use for suffix data (bytes after cursor) where corruption risk is low.
func ParseMessages(data []byte) ([]Message, error)

// ParseMessagesSafe parses all messages from TOML data with per-message error recovery.
// Fast path: tries ParseMessages(data) first — no overhead for clean files.
// Fallback: on TOML error, splits on [[message]] block boundaries and parses each block
// independently. Returns all parseable messages; logs a warning per failed block via slog.
// Use for full-file reads (hold check, hold list, AckWaiter online/offline detection)
// where the caller cannot use a suffix and historical corruption must not crash the command.
func ParseMessagesSafe(data []byte) []Message
```

**ParseMessagesSafe rationale:** `hold check`, `hold list`, and `AckWaiter.buildRecipientStates` do full-file reads (no cursor, no suffix). If any historical `[[message]]` block has corrupt TOML — which occurs in practice, see #515 — a `ParseMessages` call returns an error and the command fails entirely. `ParseMessagesSafe` makes these callers resilient: they operate on incomplete but non-crashing history rather than failing. After Phase 3 ships the state file as hold-state authority, `ParseMessagesSafe` becomes the fallback for the state file repair path, not the primary path. Both uses are worth designing for now.

**File path derivation:** `internal/chat` receives the fully-resolved file path as a constructor parameter. Path derivation (`ProjectSlugFromPath` + `DataDirFromHome`) happens in the CLI wiring layer (`cli.go` or `targets.go`), not inside this package. This avoids a circular import: `internal/cli` imports `internal/chat` (for the new commands) while `internal/chat` importing `internal/cli` for path functions would create a cycle. Solution: caller derives the path and passes it in.

**Locking:** Lock-file creation is injected via `LockFile` func (defined above). The CLI wiring layer passes a concrete implementation using `os.OpenFile(O_CREATE|O_EXCL)`. `internal/chat` itself has zero `os.*` calls — consistent with "Pure domain package. No I/O." and with CLAUDE.md: no `os.*` in `internal/`.

**Dependencies:** `BurntSushi/toml` (already in go.mod). No new deps.

### 4.2 `internal/watch`

File watching abstraction. Wraps `fsnotify` (pure Go, kqueue/inotify, no CGO).

```go
// Watcher blocks until the file changes, then returns.
type Watcher interface {
    WaitForChange(ctx context.Context, path string) error
}
```

Single new dependency: `github.com/fsnotify/fsnotify`.

**TOML cursor parsing strategy:** TOML is not streamable — a partial file (content after line N) is not a valid TOML document and `BurntSushi/toml` will reject it. `Watch` uses full-file parse: parse the entire chat file with `BurntSushi/toml`, then skip messages by index. The cursor (line number) is converted to a message index by counting `[[message]]` block headers in the raw file up to that line, then only messages after that index are examined. Full-file parse is acceptable for current chat file sizes (sessions bounded in practice; even large sessions rarely exceed a few thousand lines). Optimization to byte-offset raw scanning is deferred — if needed, it can be swapped in behind the `Watcher` interface without changing callers.

### 4.3 `internal/tmux`

Thin tmux wrappers. All I/O through injected exec func.

```go
// ExecCmd runs a command and returns its combined output.
// Func injection consistent with codebase pattern (see internal/tokenresolver.execCmd).
// Separate name param matches the existing cli.go wiring (name string, args ...string).
type ExecCmd func(ctx context.Context, name string, args ...string) ([]byte, error)

// Pane operations receive ExecCmd as a parameter, not an interface.
func SpawnPane(exec ExecCmd, windowID, title string) (paneID string, err error)
func KillPane(exec ExecCmd, paneID string) error
func ListPanes(exec ExecCmd, sessionID string) ([]PaneInfo, error)
```

**Injection pattern:** Func fields, not interfaces. The codebase uses `execCmd func(ctx context.Context, name string, args ...string) ([]byte, error)` throughout (e.g., `internal/tokenresolver`). Using an interface here would be an inconsistency — single-method interfaces for I/O are idiomatic in standard library style but deviate from this project's established pattern.

Replaces: all inline `tmux` bash in engram-tmux-lead.

### 4.4 `internal/claude`

Claude CLI launcher and stream pipeline.

```go
// StreamCmd starts a command and returns a streaming stdout pipe.
// Distinct from tmux.ExecCmd (which buffers via cmd.Output()/cmd.CombinedOutput()).
// Streaming: cmd.StdoutPipe() is returned so the caller reads line-by-line via bufio.Scanner.
// Func injection consistent with codebase pattern.
//
// Returns: a ReadCloser for stdout, a wait func (call after reading is complete, like cmd.Wait()),
// and any startup error. The caller is responsible for closing stdout and calling wait.
type StreamCmd func(ctx context.Context, args ...string) (stdout io.ReadCloser, wait func() error, err error)

// Runner executes claude CLI and processes its stream.
type Runner struct {
    Stream    StreamCmd     // injected: starts claude binary, returns streaming pipe
    Pane      io.Writer     // pane display (filtered output)
    Chat      chat.Poster   // speech relay destination
    AgentName string        // used as "from" in posted messages
}

// Run executes: claude -p [--resume <id>] --output-format=stream-json "<prompt>"
// Processes JSONL stream line-by-line via bufio.Scanner on Stream's stdout pipe.
// Filters display output, relays speech acts to chat, handles INTENT ACK-wait.
// Returns new session-id.
func (r *Runner) Run(ctx context.Context, sessionID, prompt string) (newSessionID string, err error)
```

**Why StreamCmd is distinct from tmux.ExecCmd:** tmux commands terminate quickly and their full output is needed at once — `cmd.Output()` (buffered) is correct. The claude CLI runs for the full agent turn (minutes) and must emit filtered output in real time — `cmd.StdoutPipe()` (streaming) is required. Using a single interface for both would force the caller to choose between `Output()` and `StdoutPipe()` semantics at a level that belongs in the injection layer, not the domain code.

Three-mode stream pipeline:
1. **Display filter:** show only `type="assistant"` text blocks + `type="user"` turns; suppress `system`, `tool_use`, `tool_result`, `rate_limit_event`, `result`
2. **Speech relay:** scan assistant text blocks for prefix markers; post to chat via `chat.Poster.Post()`
3. **ACK-wait:** on `INTENT:` detection, block on `engram chat ack-wait`, resume agent with "Proceed." or "WAIT from X: [text]"

**Session-id capture:** All JSONL events have top-level `session_id` field. Runner captures it from the first event.

**Streaming:** `StreamCmd` returns `cmd.StdoutPipe()`. Runner reads with `bufio.Scanner`, one JSONL line per iteration. `cmd.Output()` semantics are explicitly NOT used here — the distinction is now enforced at the type level.

### 4.5 `internal/streamjson`

Stateless JSONL parser for `--output-format=stream-json` output. Pure function, trivially testable.

```go
// ParsedEvent is a display-relevant output event.
type ParsedEvent struct {
    Kind string // "assistant_text" | "user_message"
    Text string
}

// SpeechMarker is a detected prefix marker in assistant text.
type SpeechMarker struct {
    Kind string // "READY" | "INTENT" | "LEARNED" | "DONE" | "ACK" | "WAIT" | "INFO" | "ESCALATE"
    Text string // content from prefix marker through end of speech act
    From string // agent name injected by caller from spawn context
}

// Parse processes one JSONL line. Event-stateless — no state shared across JSONL events.
// Multi-line speech act detection happens within the assembled text string of a single event.
// Returns both display events and any speech markers found.
func Parse(line []byte) (events []ParsedEvent, markers []SpeechMarker, err error)
```

**Why event-stateless:** `--output-format=stream-json` delivers complete assembled messages (not streaming character deltas). Each JSONL line is one complete message — independently parseable with no state carried from prior lines. The `Parse` function receives a full JSON object; multi-line speech acts (e.g., INTENT: with Situation + Behavior on separate lines) are detected by scanning `\n`-split lines within the text string of that single event. "Stateless" refers to the boundary between JSONL events, not within a single event's text content.

**JSONL schema (verified from live run — no Claude CLI version pinned):**
- `type="assistant"` → `message.content[]` array → text blocks (`type="text"`) + tool_use blocks
- `type="user"` → incoming turn content (show in pane)
- `type="system"` → hook events, init metadata (filter out)
- `type="result"` → final summary with cost/duration (filter out — redundant)
- `type="rate_limit_event"` → noise (filter out)
- All events have top-level `session_id` field

**Schema resilience:** This schema was verified against a live run but no Claude CLI version is pinned. If the schema changes (field renames, new nesting, type value changes), speech relay silently drops all prefix markers — workers appear to run but coordination is dead. To surface schema drift early: the parser should emit an explicit warning to stderr when it encounters an `assistant` event with no recognized text content blocks. This makes breakage visible instead of invisible. The implementation must also record the Claude CLI version used during verification in a comment.

**Dependencies:** `encoding/json` (stdlib only). No regex library needed.

---

## 5. Speech-to-Chat Model

### 5.1 Overview

Instead of calling `engram chat post`, agents express coordination messages through natural language in their output. The binary intercepts the JSONL stream, detects prefix markers, and posts to chat on the agent's behalf.

This is the [B+] category: binary owns the entire delivery loop; skill governs content only.

**Old model:** agent decides to post → calls `engram chat post` → binary appends
**New model:** agent speaks in output → binary observes JSONL → binary posts on agent's behalf

### 5.2 Prefix Marker Catalog

Agents use these start-of-line prefixes in their text output:

```
READY: [initialization status — "Joining chat. Reading history." or stats if fast init]

INTENT: Situation: [describe what you see].
Behavior: [what you're about to do].

ACK: [brief acknowledgment or "No relevant memories. Proceed."]

WAIT: (to [agent-name]) [your concern — starts argument protocol]

DONE: [task completion summary]

LEARNED: [subject] -> [predicate] -> [object].
Context: [when this applies]

INFO: [status update, heartbeat, or user-parroted input]

ESCALATE: (to lead) [summary of both positions].
Request: [specific question or decision needed from user]
```

**READY: semantics:** Workers say `READY:` as their **first output** after launch. The binary intercepts this prefix and posts a `ready` message to chat on the worker's behalf (using the agent name from spawn context as `from`). This is the only mechanism by which headless workers can announce presence under speech-to-chat. `engram agent wait-ready` watches for the resulting chat `ready` message. The agent controls the init status message content.

**Note:** `ESCALATE:` fires only after 3 argument inputs with no resolution. Binary detects this prefix and routes to lead. The skill grammar must include it; implementers who drop it will silently swallow escalations.

### 5.3 Speech Act Recognition Grammar

Prefix detection fires when the prefix appears at **column 0** only. A sentence like "My INTENT: was to..." (not at line start) does not trigger.

Speech act scope:
- **Start:** line beginning with prefix + colon + space
- **Continuation:** subsequent non-empty lines up to blank line or next prefix
- **Terminator:** blank line, next prefix marker, or end of text block

```
speech_act := prefix ": " text (NL non_empty_line)*
prefix := "READY" | "INTENT" | "LEARNED" | "DONE" | "ACK" | "WAIT" | "INFO" | "ESCALATE"
non_empty_line := line not starting with another prefix
```

**Multi-line example** (both lines captured as INTENT body):
```
INTENT: Situation: About to run targ check-full.
Behavior: This will execute the full test suite and write coverage files.
```

**Why multi-line:** Meaningful intents need Situation + Behavior on separate lines (SBIA framing). Single-line-only would force agents to compress, reducing quality.

**Implementation:** `strings.HasPrefix` on individual lines after `strings.Split(text, "\n")` — no regex needed.

### 5.4 ACK-Wait Flow Under Speech-to-Chat (Option B)

Option A (mid-stream pause) is architecturally impossible: `--output-format=stream-json` delivers complete messages. By the time `INTENT:` is detected, the agent's turn is already done.

Option B (agent says INTENT, then waits):

1. Agent outputs `INTENT: ...` in its turn (ends turn)
2. Binary intercepts from JSONL stream, posts to chat
3. Binary calls `engram chat ack-wait` (blocks on ACK/WAIT from recipients)
4. **ACK received:** binary resumes agent: `claude -p --resume <id> "Proceed."`
5. **WAIT received:** binary resumes agent: `claude -p --resume <id> "WAIT from [agent]: [text]"`
6. Agent's next turn starts with "Proceed." or a WAIT in conversation history

**"Proceed." semantics:** An empty affirmation, not a re-instruction. The agent's context (from `--resume`) already contains what they were doing. "Proceed." signals "no objections" and the agent continues from their current reasoning state. Binary does not re-explain the task.

**Skill guidance (~3 lines):**
> "After saying INTENT:, end your turn. You will receive either 'Proceed.' (continue work) or 'WAIT from [agent-name]: [text]' (engage with the concern, then say DONE: when resolved)."

### 5.5 Argument Protocol Under Speech-to-Chat

Binary is the relay. Agents never detect chat responses themselves:

1. Initiator says `INTENT: ...` → binary posts to chat → waits for engram-agent response
2. Engram-agent's response arrives in chat as `WAIT:` (says it in its own pane output → binary relays to chat)
3. Binary relays WAIT to initiator via `claude -p --resume <initiator-id> "WAIT from engram-agent: ..."`
4. Initiator responds in their output — `ACK: ...` (concede) or counter-argument
5. Binary reads initiator's next turn output — `ACK:` prefix detected → argument resolved
6. Binary posts ACK to chat, resumes engram-agent with result

3-argument cap = binary counter. Still Option A from codesign-1's argument relay design (binary as relay, same session-id throughout).

### 5.6 Display Filtering

Pane output (what the user sees) is filtered to show only:
- `type="assistant"` text blocks (agent speech, minus `tool_use`)
- `type="user"` turns (incoming messages including "Proceed." and "WAIT from X: [text]")

Suppressed: `system`, `tool_use`, `tool_result`, `result`, `rate_limit_event`

This fixes #494 (raw JSONL noise in lead pane).

---

## 6. Worker Model

### 6.1 Interactive Lead

- Has user terminal, owns user conversation
- Loads engram-tmux-lead skill
- Makes routing decisions (skill judgment)
- Calls `engram agent spawn` / `engram agent resume` to drive workers
- Posts to chat by calling `engram chat post` **directly** — NOT via speech-to-chat

**Lead posting model:** The lead runs interactively (`claude` without `-p`). Its output goes to the terminal, NOT to a JSONL stream being parsed by the binary. Speech-to-chat [B+] only applies to headless `-p` workers whose JSONL stream the binary controls. The lead must call `engram chat post --from lead --type info --text "..."` explicitly from its skill instructions. This is the [B] category for the lead, not [B+].

**Parroting user input:** The lead calls `engram chat post --from lead --type info --thread user-input --text "[User]: [verbatim text]"` to parrot user submissions. The lead's skill describes WHEN and WHAT to parrot; the binary command handles the actual write.

### 6.2 -p --resume Workers

- Headless, no event loop, no persistent context
- Receive task via `claude -p --resume <id> "<prompt>"`
- Express coordination via prefix markers in text output (speech-to-chat)
- Post result by saying `DONE: [summary]` in output
- Exit when done

Workers are driven entirely by the binary. Between resumes, they are not running — no CPU, no context consumed for polling. The binary manages their state (session-id, pane-id) in the state file.

### 6.3 State File

Location: `~/.local/share/engram/state/<slug>.toml`

The state file is the **authority for all mutable binary-owned state**: agent registry AND hold registry. The chat file hold-acquire/hold-release messages are the audit trail; the state file is the query surface. `hold check` and `hold list` read from the state file, not the chat file — eliminating the full-TOML-unmarshal risk (same attack surface as #515) that would otherwise grow monotonically as the chat file expands across sessions.

Schema (per agent):
```toml
[[agent]]
name = "executor-1"
pane-id = "main:1.2"
session-id = "abc123def456"
state = "ACTIVE"              # STARTING | ACTIVE | SILENT | DEAD
spawned-at = "2026-04-05T04:00:00Z"
last-resumed-at = "2026-04-05T04:05:00Z"
argument-with = ""            # agent name if in active argument; empty string if none
argument-count = 0            # turns used in current argument (cap: 3; reset to 0 on resolution)
argument-thread = ""          # chat thread of active argument; empty string if none
```

Schema (per hold):
```toml
[[hold]]
hold-id = "c3151bfe-5ee4-4f3e-9055-0e6047c9acbf"
holder = "lead"
target = "planner-5"
condition = "lead-release:codesign-phase3"
tag = "codesign-phase3"
acquired-ts = "2026-04-06T16:15:02Z"
```

**Why hold state belongs in the state file:**
`hold check` and `hold list` originally scanned the full chat file using `chat.ParseMessages(entireFile)` — no cursor, no suffix. If any historical `[[message]]` block in the chat file has corrupt TOML (which occurs — see #515), hold check returns a parse error and the entire hold state becomes invisible to the binary. The chat file is append-only and grows forever, so this risk increases monotonically. The state file is bounded, binary-owned, and never contains user-generated TOML. `hold acquire` and `hold release` update the state file atomically in addition to appending to the chat file. This is the log (chat) vs. snapshot (state) pattern.

**Argument state fields** are required to enforce the 3-argument cap from SPEECH-2/SKILL-2. The binary cannot count turns without persisting the count across `engram agent resume` invocations. When an argument resolves (ACK: detected or escalation fires), the binary resets all three fields to empty/zero.

**Locking:** The state file is rewritten on every `engram agent spawn`, `engram agent kill`, `engram hold acquire`, and `engram hold release` (read-modify-write). Concurrent commands from different panes will corrupt the file without locking. Locking strategy: same `os.OpenFile(O_CREATE|O_EXCL)` on `.lock` file as the chat file, using path `~/.local/share/engram/state/<slug>.toml.lock`. Lock is held for the full read-modify-write cycle, then released. The binary is the sole writer of the state file — no skill writes it directly.

**Lock timeout:** The state file's read-modify-write critical section is wider than the chat file's append (milliseconds vs. microseconds). Use a 5s lock acquisition timeout with a clear error message rather than indefinite spin, to handle concurrent agent commands from different terminals gracefully.

**State file repair:** If the state file is missing or corrupted (e.g., process kill mid-write), it can be reconstructed by scanning the chat file for all hold-acquire/hold-release pairs and all lifecycle messages. The `engram agent list` command should detect a missing state file and attempt reconstruction before erroring. This is cheap to implement when Phase 3 ships; expensive to add after Phase 5 when full-file chat parsing paths may have been removed.

Chat file = append-only coordination log; audit trail for holds and agent lifecycle events.
State file = binary bookkeeping (agent registry, hold registry, argument state); query authority for `hold check`/`hold list`/`agent list`.
These are separate concerns and do not conflict.

---

## 7. Phased Delivery Plan

### Phase 1 — Chat I/O: post + watch (PR)

**Scope:**
- `internal/chat` package (Message struct, Poster + Watcher interfaces + impls, append+lock, path-as-param)
- `internal/watch` package (fsnotify wrapper)
- Commands: `engram chat post`, `engram chat watch`

**Skill deletions (use-engram-chat-as):**
- Writing Messages bash block (~25 lines) → replaced by `engram chat post` call
- Timestamp Freshness section (~20 lines) → binary owns timestamp
- All `shlock` blocks in engram-tmux-lead → replaced by `engram chat post`
- Heartbeat bash blocks → replaced by `engram chat post`

**Skill rewrites (not deletions — system must remain E2E-functional):**
- Background Monitor Pattern (~30 lines): rewrite to call `engram chat watch` instead of raw `fswatch -1` + tail + grep. Still a subagent; mechanism improves, pattern survives until Phase 2.
- ACK-wait subagent template (~35 lines): rewrite to call `engram chat watch --type ack` instead of `fswatch -1`. Simpler subagent, still a subagent.

**Replacement skill content (use-engram-chat-as, Writing Messages + Watching sections):**

```
## Writing Messages

Use the binary: `engram chat post --from <name> --to <names> --thread <thread> --type <type> --text "<content>"`

The binary handles locking, timestamp, and TOML formatting. Capture the returned line count as your new cursor.

## Watching for Messages

Use the binary: `RESULT=$(engram chat watch --agent <name> --cursor <n>)`
Returns JSON: `{"type":"WAIT","from":"engram-agent","cursor":1234,"text":"..."}`
```

**Issues fixed:** #471 (binary foundation), chat write reliability (atomic, correct TOML, fresh timestamps)

**Phase boundary rationale (E2E check):** Post + watch are independently useful. ~45 lines of fragile bash (shlock, timestamp freshness, heartbeat) deleted and replaced. Background Monitor and ACK-wait subagent survive in rewritten form — system is fully operational after Phase 1 ships. Phase 2 eliminates the ACK-wait subagent template (WAITER side); the Background Monitor Pattern (RESPONDER side) survives until Phase 5.

---

### Phase 2 — ACK-wait + Holds (PR, depends on Phase 1)

**Scope:**
- `internal/chat` additions: `AckWaiter` interface + impl, `AckResult`, `WaitResult` types
- Commands: `engram chat ack-wait`
- Commands: `engram hold acquire`, `engram hold release`, `engram hold list`

**Skill deletions (now safe — replaced by binary):**
- ACK-wait subagent template (~35 lines) → deleted; `engram chat ack-wait` replaces the WAITER side
- Cursor capture + Reading New Content section (~20 lines) → binary handles internally
- Timing section, online/offline detection bash snippets → binary handles internally
- Hold-based retention bash in engram-tmux-lead → replaced by `engram hold` commands

**NOT deleted in Phase 2 — deferred to Phase 5:**
- Background Monitor Pattern (~30 lines): survives until Phase 5. `engram chat ack-wait` replaces only the ACK-wait subagent template (the WAITER side). The Background Monitor Pattern is engram-agent's watch loop — the RESPONDER side. If deleted here, engram-agent has no watch loop, all intents get 5s implicit ACK, and the memory safety net is bypassed for the entire Phase 2→5 gap. Same rationale as Phase 3's deferred deletion (see below).

**Replacement skill content (use-engram-chat-as, ACK-Wait + Hold sections):**

```
## ACK-Wait

Use the binary: `RESULT=$(engram chat ack-wait --agent <name> --cursor <n> --recipients <names>)`
Returns JSON: `{"result":"ACK","cursor":1234}` or `{"result":"WAIT","from":"engram-agent","cursor":1234,"text":"..."}`

Capture cursor BEFORE posting the intent, then pass it to ack-wait.

## Hold Management

Acquire: `HOLD_ID=$(engram hold acquire --holder <name> --target <agent> [--condition <cond>])`
Release: `engram hold release $HOLD_ID`
List: `engram hold list [--holder <name>] [--target <name>]`
```

**Issues fixed:** #502 (ACK-wait sleep polling eliminated), partial #503

---

### Phase 3 Pre-Flight — engram-agent Main Loop Fix (skill-only PR, no binary changes)

**Context:**

engram-agent's main loop (Step 1) still references `fswatch -1 "$CHAT_FILE"` as a background Bash command — the pre-Phase-1 watch pattern. Phase 1 updated the Background Monitor Pattern in use-engram-chat-as to use `engram chat watch`, but engram-agent's own main loop was not updated in lockstep. The two watch mechanisms are architecturally different:

- **use-engram-chat-as:** Background Monitor Pattern = background Agent tool call running `engram chat watch`. Kernel-driven, correct timeout handling, no polling.
- **engram-agent main loop:** `fswatch -1 "$CHAT_FILE"` as a background Bash command (`run_in_background: true`). Pre-Phase-1 mechanism. Produces visible bash tool-call noise in the main agent context.

This mismatch is the primary source of engram-agent stalling in multi-agent sessions. Agents following engram-agent's main loop skill implement a different watch than the rest of the system.

**Change:**

Replace engram-agent Step 1:
```
Old: Run fswatch -1 "$CHAT_FILE" as a background Bash command (run_in_background: true)
New: Spawn background monitor Agent (Background Monitor Pattern from use-engram-chat-as):
     RESULT=$(engram chat watch --agent engram-agent --cursor CURSOR)
```

This is a skill-only change (~30 lines). No binary work. The Phase 1 binary (`engram chat watch`) is already in place and correct.

**Why pre-flight, not Phase 3:**

The fix requires no binary changes and can ship immediately. Deferring it to the Phase 3 PR means Phase 3 implementation begins with a broken engram-agent watch mechanism — any multi-agent testing during Phase 3 development will be unreliable. Fix now, then Phase 3 starts clean.

**Issue tracking:**

Issue #509 is titled "engram-agent skill still uses shlock/heredoc for chat writes." That title is misleading — chat writes in the skill are already correct (`engram chat post`). The actual open problem is the fswatch-1 main loop. Close #509 (chat writes are correct) and open a targeted successor issue for the main loop migration.

**Scope boundary:**

- IN scope: engram-agent main loop Steps 1 and 7 (spawn monitor, re-spawn after processing)
- OUT of scope: shlock for memory file writes (correct and appropriate until Phase 3+ adds binary memory commands); any binary changes; engram-tmux-lead skill updates (Phase 3 scope)

---

### Phase 3 — Agent Lifecycle (separate PR)

**Scope:**
- `internal/tmux` package
- `internal/agent` package (pure domain: `AgentRecord`, `StateFile` R-M-W, `ScanAgents`, `AddAgent`, `RemoveAgent`)
- Commands: `engram agent spawn`, `engram agent kill`, `engram agent list`, `engram agent wait-ready`
- State file (`~/.local/share/engram/state/<slug>.toml`) — includes **both** `[[agent]]` and `[[hold]]` sections (see §6.3)
- Binary auto-posts spawn intent (fixed template), waits for engram-agent ACK
- `ParseMessagesSafe` in `internal/chat` (see §4.1) — ships in this PR as the resilient fallback for state file repair; also protects `hold check`/`hold list` during the transition before those commands fully migrate to state-file-authority reads

**Design requirements (non-negotiable):**
- **watchDeadline contract:** `engram agent wait-ready --max-wait` MUST propagate the computed deadline via `context.WithDeadline` to the inner `Watch` call. This is a correctness invariant. See §3.3 for the full contract. _The same bug class as #519 will manifest silently if omitted._
- **NDJSON output:** All structured command output (spawn, list) uses NDJSON. No pipe-delimited fields. Skills use `jq` for parsing. _Lesson from #516: validate all outputs against the skill's actual parsing tools before merging._
- **State file lock timeout:** 5s (wider critical section than chat file; see §6.3).
- **State file authority:** `hold check` and `hold list` read from the state file for hold state. The chat file is the audit trail, not the query surface.

**Skill deletions:**
- SPAWN-PANE definition (engram-tmux-lead §1.3)
- Spawn template bash (§2.1)
- Concurrency limit tracking bash (§2.4)
- Pane registry instructions
- Shutdown kill-pane sequence

**NOT deleted in Phase 3 — deferred to Phase 4 or later:**
- Writing Messages bash (deferred to Phase 4 — workers still call `engram chat post` directly until speech relay ships)
- Heartbeat bash (deferred to Phase 4)
- Calling-convention text for `engram chat post` (deferred to Phase 4)
- Background Monitor Pattern references and Agent Lifecycle watch loop (use-engram-chat-as) — deferred to Phase 5

**E2E check (Phase 3):** After shipping Phase 3 alone, verify all five criteria:

1. **Binary in real session:** Run `engram agent spawn/kill/list/wait-ready` in a real multi-agent session using the engram-tmux-lead skill. All four commands execute without error against a live tmux session.
2. **jq-parseable output:** Parse all structured output with `jq` (the tool skills actually use): `engram agent list` output parses as NDJSON (one JSON object per line); `engram agent wait-ready` output parses as the same JSON object format as `engram chat watch`. No command requires `cut`/`awk` for structured fields. _(Lesson from #516: TSV→NDJSON was a production catch, not a CI catch. Validate against skill's actual parsing tools before merging.)_
3. **Historical chat file:** Run `engram agent wait-ready` and `engram agent spawn` against a chat file with 3+ sessions of historical data (multi-thousand-line file with TOML from multiple prior sessions). No parse errors — suffix-at-line pattern applied where needed.
4. **Flag validation paths:** Verify all required-flag validation paths match skill usage: `engram agent spawn` requires `--name` and `--prompt`; `engram agent wait-ready` requires `--name` and `--cursor`. Error messages are human-readable (same standard as `engram hold acquire` flag validation from #518).
5. **watchDeadline fires:** Run `engram agent wait-ready --name nonexistent --max-wait 5s` against a chat file where no ready message arrives. Verify the command exits within ~5s. This confirms the watchDeadline pattern is applied at the deepest blocking call — the same class of bug as #519 will manifest silently if omitted.

Old SPAWN-PANE bash, concurrency tracking bash, and pane registry bash are deleted from skills. Workers continue to call `engram chat post` directly — no speech relay yet.

**Replacement skill content (engram-tmux-lead, ~15 lines replacing ~50 deleted lines):**

```
## Agent Lifecycle

Spawn agent: `RESULT=$(engram agent spawn --name <n> --prompt "<p>")`
Returns NDJSON: `{"pane-id":"3","session-id":"abc123"}`

Wait for agent ready: `RESULT=$(engram agent wait-ready --name <n> --cursor <cursor>)`
Returns JSON: `{"from":"<n>","type":"ready","cursor":NNN,...}`
Parse with jq: `NEW_CURSOR=$(echo "$RESULT" | jq -r '.cursor')`

Kill agent: `engram agent kill --name <name>`

List agents: `engram agent list`
Returns NDJSON: one `{"name":"...","state":"running","session-id":"...","pane-id":"..."}` per line
Parse with jq: `engram agent list | jq -r '.name'`
```

**Issues fixed:** #505/#506 (spawn window bugs), #500 (engram-down any agent via `engram agent kill`), partial #503 (binary auto-ACKs spawn intents)

---

### Phase 4 — Speech-to-Chat (separate PR, depends on Phase 3)

**Scope:**
- `internal/streamjson` package (stateless JSONL parser)
- `internal/claude` package (stream pipeline: display filter + speech relay)
- Pipes stdout through display filter + speech relay for all spawned agents

**Skill deletions:**
- Writing Messages bash (workers express coordination via speech; binary posts on their behalf)
- Heartbeat bash (workers do not heartbeat — binary tracks state)
- Calling-convention text for `engram chat post` (replaced by speech-relay prefix-marker guidance for workers)

**NOT deleted in Phase 4 — deferred to Phase 5:**
- Background Monitor Pattern references and Agent Lifecycle watch loop (use-engram-chat-as)

**Phase boundary rationale:** engram-agent's watch loop must survive Phase 4. Workers now speak `INTENT:` in output; binary relays to chat. But engram-agent must still be watching to receive those intents and respond. Phase 5 builds binary auto-resume (the replacement for the watch loop) and deletes the old loop atomically at that point. If the watch loop is deleted in Phase 4, intents pile up unacknowledged, all workers time out to implicit ACK, and the memory safety net is bypassed for the entire Phase 4→5 gap.

**E2E check (Phase 4):** After shipping Phase 4, agents speak intents out loud using `INTENT:` prefix markers in their output. The binary detects these markers and relays them to chat. Old Writing Messages bash, heartbeat bash, and `engram chat post` calling-convention text are deleted from skills.

**"Forgot to post" reformulated, not eliminated (see §5.2 note):** Speech-to-chat changes the mechanism from "call `engram chat post`" to "say `INTENT:` in output." Agents that omit the prefix marker still fail to coordinate — the failure mode changes from "forgot to call command" to "forgot to say prefix." The skill must still teach WHEN and WHY to say each prefix marker.

**Issues fixed:** partial #494 (raw JSONL eliminated from panes), partial #503 (speech relay completes binary-mediated coordination)

---

### Phase 5 — Agent Resume + Auto-Resume (separate PR)

**Scope:**
- `engram agent resume` command
- engram-agent auto-resume: binary auto-resumes engram-agent on every `intent` message in chat
- Argument relay via prefix-marker detection + `-p --resume` turns
- Worker queue management (binary queues intents if 3 argument sessions in progress)

**Skill deletions:**
- Agent Lifecycle watch loop + Background Monitor Pattern references (use-engram-chat-as) — **now safe to delete** because binary auto-resume replaces them in Phase 5
- Compaction recovery section (partial — cursor, CHAT_FILE, active-intent tracking are binary-owned; ~5 lines survive for lead state reconstruction — see notes)
- Heartbeat timer sections
- `tmux send-keys` routing pattern
- Remaining calling-convention text

**engram-agent architectural conversion (interactive → stateless worker):**

Phase 5 converts engram-agent from a persistent interactive agent into a stateless `-p` worker invoked per intent. Required design decisions before implementing:

1. **`recent_intents` correlation (Trigger 2):** Failure correlation requires cross-invocation memory. Binary should pass the last 5 intent summaries from the chat file in the resume prompt. The engram-agent skill must describe this startup context — not rely on in-context memory.

2. **Subagent management (max 3 concurrent):** Enforcement moves to the binary's worker queue. Extend the existing intent-queuing mechanism to cover concurrent subagent limits.

3. **Rate limiting (>5 new memories in 10 min):** Stateless invocations must scan recent memory file timestamps on startup. This rule survives in the Phase 6 ~72 line skill — behavioral judgment, cannot be deleted.

4. **Performance tracking (surfaced/followed counts):** Already persisted to TOML files — survives stateless invocations without change.

5. **Tiered loading on startup:** Each invocation cold-loads from disk. The Phase 6 ~72 line skill must preserve the tiered loading description (core memories always, recents on startup) — behavioral content, not bash.

6. **Cursor passing in resume prompt:** When binary auto-resumes engram-agent for an intent, the resume prompt must include the current chat cursor (e.g., "Current chat cursor: <N>."). This allows engram-agent to call `engram chat ack-wait` correctly if it needs to post an intent itself (nested intent). Without this, engram-agent has no safe cursor baseline for its own ack-wait calls.

**Compaction recovery (revised scope):**

The "cursor is binary-owned, no compaction problem" claim is partially correct. Binary-owned state eliminated after Phase 5: CURSOR, CHAT_FILE path, active-intent tracking. State still requiring skill coverage:

- **Lead in-context state** (agent registry, task IDs): reconstructable via `engram agent list` and `engram hold list`. The Phase 6 engram-tmux-lead skill must include a lead compaction recovery procedure (~5 lines calling these commands).
- **Protocol rules**: "re-invoke skill" survives — trivial (~45 lines).
- **engram-agent state**: resolved by stateless model — each invocation starts fresh.

The compaction section reduces to ~5 lines describing lead state reconstruction commands, not fully deleted.

**Issues fixed:** #494 remainder (binary handles logistics silently, lead pane shows only routing decisions and user-visible events)

---

### Phase 6 — Full Binary Dispatcher (Aspirational)

**Scope:**
- `engram dispatch` command: watches chat, maintains agent session registry, resumes workers as needed
- Binary becomes the complete event loop

**End state:**
- Skills: ~330 lines total, zero bash code
- All coordination is [B+] — skills are pure cognitive guides

---

## 8. Skill Simplification

After Phase 6 + speech-to-chat, total skill content reduces from ~2,525 lines (today) to ~330 lines (Phase 6 outcome — aspirational; see §7 Phase 6).

Actual baseline counts (verified 2026-04-05):

| Skill | Today (actual) | Phase 6 | + Speech-to-Chat |
|-------|---------------|---------|-----------------|
| use-engram-chat-as | 760 | ~60 | ~45 |
| engram-tmux-lead | 1,225 | ~130 | ~100 |
| engram-agent | 370 | ~80 | ~72 |
| engram-up | 13 | ~13 | ~13 |
| engram-down | 110 | ~50 | ~45 |
| recall | 47 | ~47 | ~47 |
| **Total** | **~2,525** | **~380** | **~322** |

The two largest skills (use-engram-chat-as and engram-tmux-lead) are 2–2.4x larger than previously claimed. The deletion scope is correspondingly larger. Phase 6 targets are achievable but require significant content reduction beyond what individual phases accomplish.

### What Remains in Each Skill

**use-engram-chat-as (~45 lines):**
- Message type catalog (condensed — semantics only, no bash)
- Prefix marker catalog (~15 lines): `INTENT: / ACK: / WAIT: / DONE: / LEARNED: / INFO:` with format examples
- Intent protocol behavioral rules (when intent is warranted, who must be in TO field)
- Argument protocol: how to argue, when to concede, escalation
- Agent roles (active vs reactive)
- Shutdown protocol
- Chat file lifecycle (2 lines — no bash)

**engram-tmux-lead (~100 lines):**
- Routing decision table (request classification — pure LLM judgment)
- Role prompt templates (what to put in executor/planner/reviewer prompts)
- Task-type → skill mapping table
- Hold PATTERN selection guidance (pair/handoff/fan-in/barrier/merge-queue — the judgment call)
- Escalation surfacing to user (how to present choices)
- Context pressure management guidance

**engram-agent (~72 lines):**
- Situation matching guidance (examples of what counts as a match)
- Argument reactor behavior (aggressive — examples)
- Feedback learning signal detection (correction vs noise)
- Fact extraction patterns (triple examples, negative examples)
- Conflict resolution judgment (when semantic conflict overrides)
- Memory file locking & atomic write section **SURVIVES** (this is for memory files, not chat files)

**engram-down (~45 lines):**
- Session summary format guidance
- Minimal behavioral guidance for shutdown sequence

**engram-up (~20 lines):**
- Unchanged (already minimal)

**recall (~48 lines):**
- Summary presentation guidance (decisions > discussions)
- Already at minimal size

**Zero bash code** remains in any skill after Phase 6 + speech-to-chat. Every remaining line is behavioral prose or a prefix marker catalog.

---

## 9. Existing Binary Alignment

### Reused Without Change

| Symbol | Description |
|--------|-------------|
| `ProjectSlugFromPath(path string) string` | Converts filesystem path to slug (path separators → `-`). Already matches skill's bash `tr '/' '-'`. |
| `DataDirFromHome(home string) string` | XDG_DATA_HOME + fallback to `~/.local/share/engram`. Chat files live at `DataDirFromHome/chat/<slug>.toml`. |
| `BurntSushi/toml` | Already in go.mod. TOML chat parsing costs zero new dependencies. |
| `exec.CommandContext` pattern | Already used for keychain calls. Tmux and claude CLI calls follow the same wiring. |
| `signalContext()` | Already in cli/signal.go. Long-running watchers use this for clean shutdown. |
| `applyDataDirDefault` / `applyProjectSlugDefault` | Pattern for deriving defaults from env/CWD. New commands follow same flag + default pattern. |

### New Dependencies (Minimal)

| Dependency | Purpose |
|------------|---------|
| `github.com/fsnotify/fsnotify` | File watching (kqueue/inotify — pure Go, no CGO) |

No other new external dependencies. JSONL parsing uses `encoding/json` (stdlib).

### New Packages (5 total)

`internal/chat`, `internal/watch`, `internal/tmux`, `internal/claude`, `internal/streamjson`

### New Subcommands (12 total)

`engram chat post`, `engram chat watch`, `engram chat ack-wait` (3),
`engram hold acquire`, `engram hold release`, `engram hold list`, `engram hold check` (4),
`engram agent spawn`, `engram agent kill`, `engram agent list`,
`engram agent wait-ready`, `engram agent resume` (5)

Dispatch requires changes to **both** `targets.go` (targ.Group registration) **and** `cli.go` (nested switch dispatch). Without targ registration, new subcommands are invisible to targ — they won't appear in `--help` output and cannot be invoked. See "Targ Registration Layer" below.

### Targ Registration Layer (targets.go)

`cli.go` routes parsed commands after targ resolves them. `targets.go` registers commands with targ. Hierarchical subcommands require `targ.Group`:

```go
// targets.go additions — extend BuildTargets to include new command groups
targ.Group("chat",
    targ.Targ(func(a ChatPostArgs) { run("chat post", ChatPostFlags(a)) }).
        Name("post").Description("Post a message to chat"),
    targ.Targ(func(a ChatWatchArgs) { run("chat watch", ChatWatchFlags(a)) }).
        Name("watch").Description("Watch for messages"),
    targ.Targ(func(a ChatAckWaitArgs) { run("chat ack-wait", ChatAckWaitFlags(a)) }).
        Name("ack-wait").Description("Wait for ACK/WAIT responses"),
),
targ.Group("hold",
    targ.Targ(func(a HoldAcquireArgs) { run("hold acquire", HoldAcquireFlags(a)) }).
        Name("acquire").Description("Acquire a hold"),
    targ.Targ(func(a HoldReleaseArgs) { run("hold release", HoldReleaseFlags(a)) }).
        Name("release").Description("Release a hold"),
    targ.Targ(func(a HoldListArgs) { run("hold list", HoldListFlags(a)) }).
        Name("list").Description("List active holds"),
),
targ.Group("agent",
    targ.Targ(func(a AgentSpawnArgs) { run("agent spawn", AgentSpawnFlags(a)) }).
        Name("spawn").Description("Spawn an agent"),
    targ.Targ(func(a AgentKillArgs) { run("agent kill", AgentKillFlags(a)) }).
        Name("kill").Description("Kill an agent"),
    targ.Targ(func(a AgentListArgs) { run("agent list", AgentListFlags(a)) }).
        Name("list").Description("List agents"),
    targ.Targ(func(a AgentWaitReadyArgs) { run("agent wait-ready", AgentWaitReadyFlags(a)) }).
        Name("wait-ready").Description("Wait for agent ready"),
    targ.Targ(func(a AgentResumeArgs) { run("agent resume", AgentResumeFlags(a)) }).
        Name("resume").Description("Resume an agent"),
),
```

`cli.go` switch dispatch is extended to handle the two-level `cmd + subCmd` routing (e.g., `case "chat": switch subCmd { case "post": ... }`). No architectural changes to existing `recall` and `show` commands.

---

## 10. Design Decisions Log

All questions settled during codesign-1 and codesign-2. No open items.

### ARCH-1: Binary/Skill Boundary
**Q:** What does the binary own vs the skill?
**A:** Binary owns ~70% — all deterministic algorithms (file I/O, watching, online/offline detection, timers, state machines). Skills own ~30% — semantic judgment only (when to post intent, what to write, routing, argument content).

### ARCH-2: Worker Model
**Q:** Should workers run persistent event loops or be invoked on demand?
**A:** Workers are invoked on demand via `-p --resume`. No persistent event loops. Binary is the event loop. Workers receive task, do work, post result (via speech-to-chat), exit.

### ARCH-3: Panes vs Pure -p
**Q:** Should workers have tmux panes or be pure headless processes?
**A:** Keep panes for observability — users can watch agent work in real time. Workers are driven via `-p --resume` (not persistent event loops inside panes). No UX regression.

### ARCH-4: Storage Split
**Q:** Chat file vs state file — what goes where?
**A:** Chat file = online/offline truth (agent presence via message timestamps). State file (`~/.local/share/engram/state/<slug>.toml`) = binary bookkeeping (pane-id, session-id, lifecycle state). Separate stores, no conflict.

### ARCH-5: Hold State
**Q:** Where is hold state stored?
**A:** Chat-file-native. Holds are TOML messages in the chat file. Binary scans for unmatched acquire/release pairs. No separate hold state file.

### ARCH-6: Exit Codes and Stdout Encoding
**Q:** How do binary commands communicate results to callers?
**A:** Always exit 0, errors to stderr. Simple outputs without text fields use pipe-delimited strings: `ONLINE`, `OFFLINE`, `pane-id|session-id`. Commands whose output includes a freeform `text` field use JSON to avoid pipe-character ambiguity: `engram chat watch` and `engram chat ack-wait` output JSON objects. `engram hold list` outputs one JSON object per line. JSON fields are the same logical fields previously described as pipe-delimited — only the encoding changed.

### ARCH-7: Locking Strategy (sole-writer migration)
**Q:** How do we handle the transition from bash shlock to Go locking?
**A:** Option B — binary becomes sole writer immediately. Go uses `os.OpenFile(O_CREATE|O_EXCL)` on `.lock` file, compatible with existing bash shlock convention during the brief migration window. Skills updated atomically per phase (no dual-writer period within a single skill). Once a skill is updated, only `engram chat post` (or speech-to-chat) writes to the chat file.

**Shlock interop note:** `shlock` (bash) and `os.OpenFile(O_CREATE|O_EXCL)` (Go) are compatible at the file system level because both use the same lock-file-existence convention: lock is held if the `.lock` file exists, released by deleting it. `shlock` creates the file and writes its PID; `O_CREATE|O_EXCL` creates the file atomically and fails if it exists. Neither checks the file content — they only check existence. A bash shlock holder and a Go holder will correctly see each other's locks. The migration window is brief: once a skill is updated for a phase, the binary is the sole writer for that skill's chat operations. No concurrent bash-and-Go writers within the same skill version.

### ARCH-8: File Watching
**Q:** `fswatch -1` (shell dependency) or pure Go?
**A:** `fsnotify` (pure Go, kqueue on macOS, inotify on Linux). No external process dependency. One new dep.

### ARCH-9: Command Naming
**Q:** Flat (`engram chat-post`) or hierarchical (`engram chat post`)?
**A:** Hierarchical sub-subcommands. `engram chat post`, `engram chat watch`, `engram agent spawn`. Dispatch requires extending **both** `targets.go` (targ.Group registrations — makes subcommands visible in help and invocable) **and** `cli.go` (nested switch dispatch for cmd + subCmd routing). Omitting either layer produces a broken implementation.

### ARCH-10: Routing Judgment
**Q:** Should the binary classify requests and spawn agents, or should the skill do it?
**A:** Option A — skill classifies (LLM judgment, cannot be made deterministic), then calls `engram agent spawn` directly. Binary executes mechanically. This is correct for Phases 1-5. Phase 6 dispatcher may absorb some routing patterns.

### ARCH-11: wait-ready Command Shape
**Q:** Should `engram agent spawn` block until ready, or should that be a separate command?
**A:** Separate `engram agent wait-ready` command. Spawn returns `pane-id|session-id` immediately. Caller (or binary) calls `engram agent wait-ready --name X --cursor N --max-wait 30` explicitly. Keeps commands composable.

### ARCH-12: Phase Packaging
**Q:** How many PRs for phases 1-4?
**A:** Phases 1+2 as single PR (shared chat I/O foundation). Phase 3 separate PR (agent lifecycle). Phase 4 separate PR (speech-to-chat, depends on Phase 3). Phase 5 separate PR.

### ARCH-13: Heartbeats
**Q:** Do workers need to heartbeat?
**A:** No. Workers are not running between resumes. Binary tracks last-resume timestamp and last-posted-chat timestamp. Heartbeat concept disappears from skills.

### ARCH-14: Targ Integration
**Q:** When to wire new commands into targ build system?
**A:** Defer until after `Run()` dispatch works. Practical implementation note — don't block on targ integration during early phases.

### SPEECH-1: ACK-Wait Option A vs B
**Q:** Binary pauses agent execution mid-turn (Option A) vs agent says INTENT and explicitly ends turn (Option B)?
**A:** Option B is the only architecturally viable choice. `--output-format=stream-json` delivers complete assembled messages — the agent's turn is already done when INTENT is detected. Mid-stream interception is impossible. Option B also makes the protocol legible: "say INTENT, then say nothing more until you receive Proceed. or WAIT from [agent]."

### SPEECH-2: "Proceed." Semantics
**Q:** What does the binary send to resume the agent after ACK?
**A:** "Proceed." — an empty affirmation, not a re-instruction. The agent's context (from `--resume`) already contains what they were doing. Binary does not re-explain the task on the resume turn.

### SPEECH-3: Speech Relay Implementation
**Q:** How does the binary relay speech acts to chat — subprocess call to `engram chat post` or direct Go function call?
**A:** Direct call via `chat.Poster.Post()` DI injection in `internal/claude.Runner`. No subprocess, no self-call. Clean dependency injection.

### SPEECH-4: Multi-line Speech Acts
**Q:** Should INTENT be single-line only, or support multi-line continuation?
**A:** Multi-line, paragraph-level scoping. Prefix on first line, continuation through blank line or next prefix. SBIA framing (Situation + Behavior) naturally spans two lines. Single-line-only would force compression.

### SPEECH-5: Memory File Locking Survival
**Q:** Does the engram-agent's locking & atomic write section get deleted when chat writes are eliminated?
**A:** NO. Memory file locking (for `~/.local/share/engram/memory/` files) is unrelated to chat writes. Only chat write bash disappears. Memory file locking survives.

### SKILL-1: "Calling convention" residue
**Q:** After binary handles delivery, do skills still need to describe how to call `engram chat post`?
**A:** No. Under [B+] speech-to-chat, even the instruction "use `engram chat post`" is gone. The skill simply says "say INTENT: in your response." Skills NEVER describe delivery mechanics after Phase 6 + speech-to-chat.

### SKILL-2: Argument relay
**Q:** How does the argument protocol work under speech-to-chat?
**A:** Binary is the relay. Initiator says INTENT in output → binary posts to chat → waits for responses. Engram-agent responds in its pane output with WAIT: → binary relays back to initiator via -p --resume. Initiator says ACK: → binary marks argument resolved. 3-argument cap enforced by binary counter persisted in state file (see Section 6.3 argument state fields).

### ARCH-15: READY: Prefix and Worker Presence Announcement
**Q:** How do headless workers announce presence under speech-to-chat when the binary controls the JSONL stream?
**A:** Workers say `READY:` as their first output. Binary detects this prefix in the JSONL stream and posts a `ready` message to chat on the worker's behalf. `engram agent wait-ready` then waits for that chat message. Binary auto-posting ready before launch was rejected: the agent controls the init status message content (e.g., memory counts, initialization stats).

### ARCH-16: Output Format for Text-Bearing Commands
**Q:** Should commands that output freeform text use pipe-delimited or JSON encoding?
**A:** JSON for commands where the output includes a `text` field (`engram chat watch`, `engram chat ack-wait`, `engram hold list`). Pipe-delimited is correct for outputs that contain only identifiers and fixed-format values (`pane-id|session-id`, agent list, etc.). Chat text regularly contains pipe characters (code, commands, file paths) — pipe-delimited output for these fields is unparseable. JSON avoids this with standard quoting/escaping.

### ARCH-17: Hold Condition Auto-Evaluation Mechanism
**Q:** The condition DSL (`done:<agent>`, `first-intent:<agent>`, `lead-release:<tag>`) defines trigger conditions. What mechanism actually evaluates and fires them?
**A:** `engram hold check` — a new subcommand that scans active holds and evaluates conditions against current chat state. `done:<agent>` fires when a `done` message from that agent exists after the hold's acquire timestamp. `first-intent:<agent>` fires on first `intent` message from that agent after acquire. `lead-release:<tag>` never auto-fires; requires explicit `engram hold release`. The command is called automatically by `engram agent kill` (for `done:` conditions on the killed agent) and by `engram agent resume` after each turn (Phase 5+). The lead can also call it manually.
