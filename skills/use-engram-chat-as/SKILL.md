---
name: use-engram-chat-as
description: Use when independently-launched agents need to coordinate, when joining a multi-agent session, when told to communicate via engram chat, when using /use-engram-chat-as. Agents broadcast intent before acting, block on file-change notifications (not polling), and wait briefly for objections before proceeding. Symptoms that trigger this skill: agents missing messages, needing to coordinate before acting, multiple agents in separate terminals.
---

# Engram Chat Protocol

Protocol for independently-launched Claude Code agents to coordinate through a shared TOML chat file. Agents announce intent before acting and wait briefly for objections. One file per project, lockfile for writes, fswatch for notifications.

## Invocation

```
/use-engram-chat-as <role-description>
```

The role description is free-form text. Examples:

- `reactive memory agent named engram-agent`
- `reviewer named bob, who uses code review skills`
- `/engram-tmux-lead` -- special case, loads lead orchestrator behavior
- `/engram-agent` -- special case, loads engram-agent behavior skill (reactive memory agent named engram-agent)

### Special-case skill references

When the role description is a slash command (starts with `/`), it refers to another skill that defines the agent's behavior. After joining chat with this protocol, **immediately invoke the referenced skill**. This loads both the coordination protocol AND the agent-specific behavior in a single user command, avoiding queued-message race conditions.

| Role description | Parsed as | Skill to invoke |
|-----------------|-----------|----------------|
| `/engram-tmux-lead` | active agent named `lead` | `engram:engram-tmux-lead` |
| `/engram-agent` | reactive agent named `engram-agent` | `engram:engram-agent` |

For non-slash role descriptions, parse normally:

The skill parses out:
- **Agent name**: extracted from "named X" in the role description. Required.
- **Role type**: `active` if not stated, `reactive` if the word "reactive" appears.
- **Behavioral context**: the full role description, included in the agent's system context.

## When to Use

- Multiple agents need to coordinate (any number, any roles)
- Agents launched independently by the user in separate terminals
- You're told to communicate via engram chat
- You need to announce what you're about to do and give others a chance to object

## Agent Roles

Agents declare a role in their introduction message:

- **Active** -- broadcasts intent before acting, waits for responses. Can also react to others' messages.
- **Reactive** -- never broadcasts intent. Only reacts to other agents' messages. Skips the intent protocol entirely for its own actions.

The role is purely self-behavioral. It does NOT change how others treat your responses. If an active agent addresses you in an intent and you post a WAIT, they must respect it regardless of your role.

### User Input Parroting

Active agents MUST parrot user submissions into the chat file as `info` messages. This gives reactive agents visibility into user corrections and feedback. Without this, reactive agents are blind to user intent. This is honor-system -- there is no technical enforcement.

```toml
[[message]]
from = "executor"
to = "all"
thread = "user-input"
type = "info"
ts = "2026-04-03T14:30:00Z"
text = """
[User]: Fix the authentication bug in the login handler.
"""
```

## Chat File Location

**Location:** `~/.local/share/engram/chat/<project-slug>.toml`

Derived from `$PWD` using the project slug convention.

**Slug derivation:** The full resolved path of the git common dir's parent (the main repo root, or `$PWD` if not in a git repo), with path separators replaced by dashes. This matches the Go binary's `ProjectSlugFromPath` convention. Using `--git-common-dir` instead of `--show-toplevel` ensures worktree agents use the same chat file as the main repo.

```bash
PROJECT_SLUG=$(realpath "$(dirname "$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null)" 2>/dev/null || pwd)" | tr '/' '-')
```

| `$PWD` | Resolved root | Slug | Chat file |
|--------|--------------|------|-----------|
| `/Users/joe/repos/engram` | `/Users/joe/repos/engram` | `-Users-joe-repos-engram` | `~/.local/share/engram/chat/-Users-joe-repos-engram.toml` |
| `/Users/joe/repos/traced/src` | `/Users/joe/repos/traced` | `-Users-joe-repos-traced` | `~/.local/share/engram/chat/-Users-joe-repos-traced.toml` |
| `/Users/joe/repos/engram-wt1` (worktree) | `/Users/joe/repos/engram` | `-Users-joe-repos-engram` | `~/.local/share/engram/chat/-Users-joe-repos-engram.toml` |
| `/tmp/scratch` | (none) | `-tmp-scratch` | `~/.local/share/engram/chat/-tmp-scratch.toml` |

**Symlinks:** `realpath` resolves symlinks before slug derivation. Two agents in symlinked paths to the same repo use the same chat file.

**Directory creation:** The skill creates `~/.local/share/engram/chat/` on first use if it doesn't exist.

## Chat File Lifecycle

Chat files are **persistent and append-only**. They grow forever. Prior sessions contain valuable dialog — decisions, context, learned facts.

- **Session start:** The first agent to join creates the chat file if it doesn't exist.
- **Joining an existing chat:** Read the last 20 messages to get an idea of what's been going on. Read further back if you need more context. Do NOT truncate, delete, or overwrite the file — prior conversations may contain important context.
- **Session end:** Agents post their final `done` messages. The chat file is NOT truncated — the next session picks up where this one left off.
- **Never truncate.** The chat file is a persistent record of agent coordination. The engram-agent also extracts durable knowledge into memory files, but the raw chat is preserved for context.

## Message Format

```toml
[[message]]
from = "executor"
to = "engram-agent, reviewer"
thread = "implementation"
type = "intent"
ts = "2026-04-03T14:30:00Z"
text = """
Your message here. Multi-line is fine.
"""
```

Every message has these fields:
- **from**: Your agent name (required)
- **to**: Comma-separated recipient names or `"all"` for broadcast (required)
- **thread**: Conversational thread name (required)
- **type**: One of `intent`, `ack`, `wait`, `info`, `done`, `learned`, `ready`, `shutdown`, `escalate` (required)
- **ts**: ISO 8601 timestamp (required on all message types)
- **text**: The content (required)

## Message Type Catalog

| Type | Purpose | Emitted by | Response expected |
|------|---------|-----------|-------------------|
| `intent` | Announce situation + planned action before acting | Active agents | Explicit ACK or WAIT from all TO recipients |
| `ack` | No objection, proceed; or early concession in argument | Any agent | No |
| `wait` | Objection, memory to surface, or request to pause | Any agent | Initiator response |
| `info` | Status updates, user-parroted input, resolution recording | Any agent | No |
| `done` | Task/action completed; final message before shutdown | Any agent | No |
| `learned` | Knowledge extracted from work (fact signal for engram-agent) | Active agents | No (silent processing) |
| `ready` | Agent initialization complete, watching chat | Any agent | No (but spawners may wait for it) |
| `shutdown` | Signal agent to exit after completing in-flight work | Lead or active agent | `done` message |
| `escalate` | Unresolved argument, needs user decision via lead | Reactor in argument | Lead surfaces to user |

## Writing Messages

Always lock before appending:

```bash
# Derive chat file path (use --git-common-dir so worktree agents share the main repo's chat file)
PROJECT_SLUG=$(realpath "$(dirname "$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null)" 2>/dev/null || pwd)" | tr '/' '-')
CHAT_FILE="$HOME/.local/share/engram/chat/$PROJECT_SLUG.toml"
mkdir -p "$(dirname "$CHAT_FILE")"

# Lock, append, unlock (macOS shlock)
while ! shlock -f "$CHAT_FILE.lock" -p $$; do sleep 0.1; done
cat >> "$CHAT_FILE" << 'EOF'

[[message]]
from = "myname"
to = "recipient"
thread = "topic"
type = "info"
ts = "2026-04-03T14:30:00Z"
text = """
Content here.
"""
EOF
rm -f "$CHAT_FILE.lock"
```

If `shlock` isn't available, use `mkdir "$CHAT_FILE.lock"` (atomic on POSIX) and `rmdir` to unlock.

## Watching for Messages

Delegate all chat monitoring to a background Agent — do **not** run `fswatch`, cursor tracking, or grep operations as direct Bash tool calls in the main agent context. These produce visible bash tool-call noise in the agent pane. All monitoring belongs inside a background subagent where it is invisible.

**Background Monitor Pattern:**

Spawn a monitoring Agent (`Agent` tool, `run_in_background: true`) with this task:

```
Monitor engram chat file for semantic events.
CHAT_FILE: [full path — embed as literal string, not a variable]
CURSOR: [current line count — embed as integer literal, not a shell variable]
AGENT_NAME: [the agent name to filter messages for]

1. Run foreground bash: fswatch -1 "$CHAT_FILE"
   (Blocks until one file change. Linux: inotifywait -e modify "$CHAT_FILE")
2. Read new lines: tail -n +$((CURSOR + 1)) "$CHAT_FILE"
3. Parse TOML blocks. Find the FIRST block where `to` contains AGENT_NAME or "all".
4. Advance cursor: CURSOR=$(wc -l < "$CHAT_FILE")
5. Return: EVENT_TYPE|from|CURSOR|text
   EVENT_TYPE is one of: INTENT, ACK, WAIT, SHUTDOWN, DONE, INFO
6. If no relevant message found: go back to step 1 (with advanced cursor)
```

**Main agent loop:**
1. Spawn background monitor Agent (embed current cursor as integer literal)
2. **Do NOT complete your turn** — wait for the Agent notification
3. When notified: parse the `EVENT_TYPE|from|cursor|text` result, process, act
4. Spawn new monitor Agent with the returned cursor value
5. Repeat — **ALWAYS**, even after completing a task

**CRITICAL:** Between notifications you are idle but NOT done. Completing your turn breaks the loop and drops all future messages.

**Never exit the watch until the user explicitly dismisses you or you receive a `shutdown` message.**

### Reading New Content

> **Note:** The cursor pattern below is what background monitoring Agents use internally. The main agent does not run these bash commands directly — it embeds cursor values as integer literals when spawning monitoring Agents.

**HARD RULE: NEVER grep or search the full chat file to check for agent responses.** The chat file is persistent and grows across sessions. Grepping the full file matches messages from old sessions, causing false positives — you'll see agents as "done" when they haven't started, or relay stale content as if it were new output. This is a critical reliability bug.

**Scope note:** Online/offline presence detection is the one exception — it scans the full file by design to find the recipient's most recent timestamp across all sessions. This is not checking for responses to your intent; it is determining whether the recipient is active. All other response checking (ACK, WAIT, done) must use the cursor.

Track where you left off by line number:

```bash
# Initialize cursor: record current end-of-file BEFORE starting work or posting ready.
CURSOR=$(wc -l < "$CHAT_FILE")

# Read new content: ONLY lines after cursor
tail -n +$((CURSOR + 1)) "$CHAT_FILE"

# Update cursor after processing each batch
CURSOR=$(wc -l < "$CHAT_FILE")
```

**Wrong:**
```bash
grep -q 'type = "done"' "$CHAT_FILE"          # BUG: matches old messages from prior sessions
grep 'from = "agent-1"' "$CHAT_FILE"           # BUG: matches old messages from prior sessions
```

**Right:**
```bash
tail -n +$((CURSOR + 1)) "$CHAT_FILE" | grep -q 'type = "done"'   # only new messages
tail -n +$((CURSOR + 1)) "$CHAT_FILE" | grep 'from = "agent-1"'   # only new messages
```

### Joining Late

If you join a channel that already has messages, read the entire file first to catch up before posting or watching.

## Intent Protocol

**Before taking any significant action, broadcast your intent and wait for responses.** This gives other agents a chance to surface relevant context, flag conflicts, or object.

### The Flow

```
1. Post intent    -> type = "intent", describe situation + planned action
2. Wait for explicit responses from ALL TO recipients:
   - Spawn a background ACK-wait Agent: watch CHAT_FILE from current cursor for ACK/WAIT
     from each expected recipient, applying the online/offline timing rules below
   - ONLY proceed when every TO recipient has responded (ACK or WAIT)
   - Offline exception: if a recipient has NOT posted any message in the last 15 min
     (scan full file), treat timeout as implicit ACK for that recipient only, after 5s
   - Online + silent: if a recipient has posted a message within the last 15 min but
     is silent after 5s, post info noting no response; wait up to 30s, then escalate to lead
3. Check responses:
   - All recipients ACKed (type = "ack")  -> proceed immediately
   - Any recipient said WAIT (type = "wait") -> pause, read their full response, then decide
4. Act
5. Pre-done check: re-read from cursor before posting done.
   If any WAIT addressed to you is unresolved, engage before posting done.
6. Post result    -> type = "done" or "info"
```

**HARD RULE: ALL intent messages MUST include `engram-agent` in the `to` field.**

The memory agent must have the opportunity to surface relevant memories before every significant action. An intent that excludes engram-agent bypasses the memory safety net entirely.

Acceptable:
```
to = "engram-agent"
to = "engram-agent, reviewer"
to = "all"
```

Not acceptable:
```
to = "reviewer"    <- missing engram-agent
to = "lead"        <- missing engram-agent (unless intent is purely coordinative)
```

### Intent Messages

When you're about to do something that could affect others, broadcast:

```toml
[[message]]
from = "executor"
to = "engram-agent"            # always present; add other recipients as needed
thread = "build"
type = "intent"
ts = "2026-04-03T14:30:00Z"
text = """
Situation: About to run targ check-full on the refactored cli package.
Behavior: Will execute targ check-full which writes coverage files.
"""
```

Use SBIA-style framing: describe the **situation** you see and the **behavior** you're about to take.

### Responses

**ACK** -- no objection, proceed:
```toml
[[message]]
from = "engram-agent"
to = "executor"
thread = "build"
type = "ack"
ts = "2026-04-03T14:30:01Z"
text = """
No relevant memories. Proceed.
"""
```

**WAIT** -- objection or additional context needed:
```toml
[[message]]
from = "reviewer"
to = "executor"
thread = "build"
type = "wait"
ts = "2026-04-03T14:30:01Z"
text = """
I'm currently running targ check-full. Coverage files will clobber each other.
Wait for my completion message before running yours.
"""
```

### Timing

The intent protocol waits for explicit ACK or WAIT from **all** TO recipients. Timeout is NOT implicit permission to proceed for online agents.

**Detecting online/offline:** A recipient is online if they posted any message within the last 15 minutes. Scan the **full** chat file — heartbeats and prior-session messages count. This is the one exception to the cursor rule: presence detection uses the whole file, not just new messages.

```bash
RECIPIENT="engram-agent"   # replace with the actual recipient name
FIFTEEN_MIN_AGO=$(date -u -v-15M +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null \
  || date -u -d "15 minutes ago" +"%Y-%m-%dT%H:%M:%SZ")
LAST_TS=$(grep -A6 "^from = \"$RECIPIENT\"" "$CHAT_FILE" \
  | grep "^ts = " | sed 's/ts = "//;s/"//' | sort | tail -1)
if [ -n "$LAST_TS" ] && [ "$LAST_TS" \> "$FIFTEEN_MIN_AGO" ]; then
  echo "online"
else
  echo "offline"
fi
```

- **Recipient offline (no message in last 15 min, or no messages at all)?** Timeout after 5s = implicit ACK for that recipient only.
- **Recipient online (message within last 15 min) but silent?** Do NOT proceed on timeout. Post info noting no response after 5s; wait up to 30s, then escalate to lead.
- **Fast ACK?** You proceed as soon as all recipients respond -- often under 200ms.
- **WAIT received?** You pause for the full response. No fixed timeout -- the conversation continues until resolved.

### HARD RULE: WAIT Is Unconditional

**A WAIT received after you have started executing is still valid.**

If engram-agent or any recipient posts WAIT after you've already started executing (e.g., they ACKed but then found a relevant memory mid-task), stop at the next safe point and engage. The argument protocol applies regardless of when the WAIT arrives.

When you receive a WAIT mid-execution:
1. Stop at the next safe stopping point (finish the current atomic operation; do NOT leave files half-written).
2. Post an info message acknowledging the pause: `type = "info"`, `text = "Pausing execution to respond to WAIT from <agent>."`
3. Engage with the WAIT per the Argument Protocol.
4. Only resume (or concede) after the argument resolves.

**Posting `done` while a WAIT is unresolved is a protocol violation.** Do not complete the task and then retroactively respond — by then the action is done and the challenge is moot.

### When to Use Intent

Use the intent protocol before:
- Running build/test/coverage tools (resource conflicts)
- Modifying shared files (merge conflicts)
- Making architectural decisions (context others might have)
- Committing or pushing (coordination with other branches)
- Any action another agent asked you to check in about

Skip the intent protocol for:
- Reading files (no side effects)
- Searching/grepping (no side effects)
- Posting informational messages to chat

## Argument Protocol

When a WAIT leads to disagreement, the argument follows structured rules:

- **Initiator** (the agent whose intent was challenged): responds **factually**. States reasoning, evidence, context. No defensiveness.
- **Reactor** (the agent that posted WAIT): responds **aggressively**. Pushes back hard on weak reasoning. Agents default to thinking well of their own work -- the reactor counterbalances this.
- **3 argument inputs max.** Reactor objection -> initiator response -> reactor counter. If still unresolved after 3 inputs, the reactor posts an `escalate` message addressed to the lead.
- **Early concession.** If the initiator agrees with the reactor after the first objection, the initiator posts an `ack` to end the argument early.
- **Resolution recording.** After the argument resolves (agreement, concession, user decision, or timeout), the reactor posts an `info` message with the resolution outcome for observability.

### Lead-Mediated Escalation

After 3 argument inputs with no resolution, the reactor posts an `escalate` message **addressed to the lead**:

```toml
[[message]]
from = "engram-agent"
to = "lead"
thread = "build"
type = "escalate"
ts = "2026-04-03T15:45:00Z"
text = """
Unresolved disagreement with executor.
Memory says: Never run targ check-full while another instance is running.
executor says: The other instance finished 2 minutes ago.
I say: I see no completion message in chat. Please verify before proceeding.
Request: Ask the user whether to proceed or wait.
"""
```

**Required fields in escalation text:**
- Summary of both positions
- Specific ask for the user (question to answer, decision to make)

**Rules:**
1. **Escalation target is always the lead.** Even if the lead is one of the arguing parties, escalation goes to the lead because only the lead has user access. The lead MUST surface lead-involved escalations fairly.
2. **The lead MUST surface the escalation to the user.** Dropping escalations silently is a critical bug.
3. **The lead posts the user's decision** as an `info` message on the same thread.
4. **Resolution recording unchanged:** After resolution, the reactor posts an `info` message with the outcome.
5. **Standalone agents (no lead):** If no `ready` message from an agent named `lead` exists in chat history, use standalone mode -- escalate to the initiating agent's UX instead. The initiating agent MUST surface the dispute to its user.

## Learned Messages

Active agents announce knowledge extracted from their work. This gives the engram-agent high-confidence signals for fact extraction.

```toml
[[message]]
from = "executor"
to = "engram-agent"
thread = "implementation"
type = "learned"
ts = "2026-04-03T14:30:00Z"
text = """
engram -> uses -> targ for all build, test, and check operations.
Context: Discovered during build failure -- running go test directly misses coverage thresholds.
"""
```

**Semantics:**
- Addressed to `engram-agent` (or `all`). Other agents may ignore `learned` messages.
- Content should include subject-predicate-object triples where possible, plus context. Free-form is acceptable.
- `learned` messages are high-confidence signals (same tier as explicit user corrections).
- Only emit when the agent discovered something reusable across sessions. Not every `done` needs a `learned`.
- **No response expected.** The engram-agent silently processes these. If extraction fails, the message is dropped (best-effort).

## Ready Messages

Agents announce they have completed initialization and are watching the chat. This provides a synchronization point.

```toml
[[message]]
from = "engram-agent"
to = "all"
thread = "lifecycle"
type = "ready"
ts = "2026-04-03T14:25:00Z"
text = "Loaded 47 feedback memories, 23 facts. Watching for intents."
```

**Semantics:**
- Posted **once**, after the agent has: (1) read full chat history, (2) loaded resources, (3) spawned its background monitor Agent.
- Addressed to `all`. Every agent posts `ready` regardless of role.
- The `text` field contains agent-specific initialization stats. No required format.

**Who waits for whom:**
- **Lead setup:** The lead waits for `ready` from spawned agents before routing work (30s timeout).
- **Standalone setup:** Agents don't wait for each other. `ready` is informational.
- **Late joiners:** Read full history on join. `ready` announces presence but doesn't replay missed intents.
- **Reactive agents:** Post `ready` but do not wait for anyone else.

## Shutdown Protocol

### Shutdown Message

```toml
[[message]]
from = "lead"
to = "engram-agent"
thread = "lifecycle"
type = "shutdown"
ts = "2026-04-03T16:00:00Z"
text = "Session complete."
```

`shutdown` can be sent by the lead (or any active agent in a non-lead setup) to specific agents or to `all`.

### Agent Shutdown Behavior

When an agent receives a `shutdown` message addressed to it (or to `all`):

1. **Stop accepting new work.** Do not process further intents or messages after the shutdown.
2. **Complete in-flight work.** If currently executing, finish and post the result.
3. **Post a final `done` message:**
   ```toml
   [[message]]
   from = "engram-agent"
   to = "all"
   thread = "lifecycle"
   type = "done"
   ts = "2026-04-03T16:00:05Z"
   text = "Shutting down. Session stats: surfaced 12 memories, learned 5 facts."
   ```
4. **Exit the monitor Agent loop.** Do not spawn a new monitor Agent. The agent's turn is complete.

### User-Initiated Shutdown

The user can dismiss agents with phrases like "stand down", "you're done", "shut down". In a lead setup, the user says this to the lead, and the lead issues `shutdown` messages. In standalone, the agent recognizes dismissal and exits directly (posting `done` first).

## Agent Lifecycle

```
1. Derive chat file path from $PWD
2. Create chat directory if needed
3. Read last 20 messages to catch up (read further back if needed)
4. Read chat file (catch up on history)
5. Load resources (memories, configs, etc.)
6. Spawn background monitor Agent (Background Monitor Pattern, above)
7. Post ready message (with ts)
8. Wait for monitor Agent notification
9. Monitor Agent returns semantic event -> process event if addressed to you
10. If acting:
    a. Post intent to (engram-agent + any other relevant recipients)
    b. Wait for explicit ACK from all TO recipients (see Intent Protocol)
    c. Act
    d. Pre-done cursor-check: spawn background Agent to tail CHAT_FILE from cursor, grep for unresolved WAITs
       If any WAIT addressed to you and unresolved: engage before posting done
    e. Post result
11. Post response (with lock)
12. Go to step 8 -- ALWAYS. Even after completing a task.
```

**The watch only ends when:**
- You receive a `shutdown` message addressed to you (or `all`)
- The user explicitly dismisses you

## Heartbeat

Long-lived reactive agents post a heartbeat every 5 minutes using `type = "info"` on the `heartbeat` thread:

```toml
[[message]]
from = "engram-agent"
to = "all"
thread = "heartbeat"
type = "info"
ts = "2026-04-03T14:35:00Z"
text = "alive | queue: 0 | feedback: 47 loaded | facts: 23 loaded"
```

Heartbeats use `type = "info"` because they are informational status updates, not a distinct protocol event. The `thread = "heartbeat"` convention makes them filterable. Without heartbeats, a dead reactive agent is invisible.

## Compaction Recovery

Context compaction occurs when Claude Code compresses prior conversation history to manage context limits. After compaction, bash variable state is lost — including `CURSOR` — and the agent's recollection of recent protocol events may be incomplete or absent.

**What gets lost:**

| State | Lost? | Recovery |
|-------|-------|---------|
| `CURSOR` position | Yes | Re-derive from last posted message |
| `CHAT_FILE` path | Yes | Re-derive from `$PWD` |
| Agent name | Partial | Re-read skill; name is in role description |
| Active intent threads | Yes | Scan post-cursor messages for pending WAITs |
| Protocol rules | Partial | Re-invoke `engram:use-engram-chat-as` skill |

### Detecting Compaction

**Signal**: `CURSOR` is undefined or zero at a point where you know you have sent messages.

Add this guard before every `tail -n +$((CURSOR + 1))` call in your watch loop:

```bash
if [ -z "$CURSOR" ]; then
  # Context was compacted — run recovery before proceeding
  run_compaction_recovery  # see procedure below
fi
```

Claude Code may also insert a compaction notice in a `<system-reminder>`. Treat any such notice as a compaction signal.

### Recovery Procedure

Run this procedure whenever compaction is detected:

**Step 1: Re-invoke the skill.**

Invoke `engram:use-engram-chat-as` with your role description to restore protocol knowledge. This is the most important step — the protocol rules must be reloaded before acting.

**Step 2: Re-derive environment variables.**

```bash
# Re-derive chat file path (env may be lost)
CHAT_FILE="$HOME/.local/share/engram/chat/$(realpath "$(git rev-parse --show-toplevel 2>/dev/null || pwd)" | tr '/' '-').toml"
AGENT_NAME="my-agent-name"   # your name as declared in the ready message
```

**Step 3: Re-derive CURSOR from your last posted message.**

```bash
# Find the line number of the last message you sent
LAST_OUR_LINE=$(grep -n "^from = \"$AGENT_NAME\"" "$CHAT_FILE" | tail -1 | cut -d: -f1)

# Set CURSOR to just after that line so we can see what arrived after it
CURSOR=${LAST_OUR_LINE:-$(wc -l < "$CHAT_FILE")}
```

If you have never posted (fresh join), CURSOR defaults to end-of-file and you start watching from now.

**Step 4: Scan for missed messages.**

```bash
# Read everything after CURSOR (messages that arrived while context was compacted)
MISSED=$(tail -n +$((CURSOR + 1)) "$CHAT_FILE")

# Check for critical message types addressed to you
echo "$MISSED" | grep -q "type = \"shutdown\""  && echo "SHUTDOWN pending"
echo "$MISSED" | grep -q "type = \"wait\""      && echo "WAIT pending — engage before resuming"
echo "$MISSED" | grep -q "type = \"intent\""    && echo "INTENT pending — check if response needed"

# Advance cursor to end of file
CURSOR=$(wc -l < "$CHAT_FILE")
```

Engage with any pending `wait` per the Argument Protocol before proceeding.

**Step 5: Announce re-initialization.**

```toml
[[message]]
from = "my-agent"
to = "all"
thread = "lifecycle"
type = "info"
ts = "2026-04-04T19:00:00Z"
text = """
Context compaction detected. Re-initialized from chat history.
Cursor re-derived from last posted message (line <N>).
Scanned for missed messages: <count> new lines found.
Resuming watch loop.
"""
```

**Step 6: Re-enter the fswatch loop.**

Continue the lifecycle from step 8 of the Agent Lifecycle. Do not re-post a `ready` message — `info` is sufficient.

### Critical: Guard Every Cursor Use

The compaction check must run **before every tail call**, not just at startup. A compaction can occur mid-task while the agent is waiting for fswatch.

```bash
# ❌ BAD: no guard — silent misbehavior if CURSOR is lost
tail -n +$((CURSOR + 1)) "$CHAT_FILE"

# ✅ GOOD: check before use
[ -z "$CURSOR" ] && run_compaction_recovery
tail -n +$((CURSOR + 1)) "$CHAT_FILE"
```

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Act without announcing intent | Always post intent before significant actions |
| Poll with `sleep 2` loop | Use `fswatch -1` / `inotifywait` -- true kernel block (inside monitoring Agent) |
| Run fswatch/wc/grep directly in main agent context | Use background monitor Agent — bash monitoring in main context produces visible tool-call noise |
| Post a message then stop | Always re-enter the fswatch after posting |
| Stop after task completion | Completing a task != dismissed. Watch for next assignment |
| Ignore WAIT responses | A WAIT means stop and read -- the responder has critical context |
| Forget the lockfile | Always lock before appending |
| Edit existing messages | Never modify -- only append new messages |
| Skip catch-up on join | Read full history before posting |
| Escalate to initiating agent instead of lead | Check for lead `ready` in chat history; escalate to lead if present |
| Skip `ready` message | Always post `ready` after initialization, before entering watch loop |
| Emit `learned` for trivial observations | Only emit when knowledge is reusable across sessions |
| Ignore `shutdown` message | Exit monitor Agent loop after completing in-flight work and posting `done` |
| Post intent before others are ready | In lead setup: wait for expected `ready` messages (30s timeout) |
| Use old field names (`[[entry]]`, `message =`) | Clean break: use `[[message]]` and `text =` |
| Truncate or delete the chat file | Chat files are persistent -- prior sessions contain valuable context |
| Skip heartbeat in long-lived reactive agent | Post heartbeat every 5 min so others know you're alive |
| Grep full file to detect agent responses | **Critical bug**: full-file grep matches old messages. Always use cursor: `tail -n +$((CURSOR + 1)) "$CHAT_FILE" \| grep ...` |
| Fabricate or invent agent output when relaying | Always read the actual `text` field from new lines first. Summarize accurately — never predict or invent what the agent said. |
| Omit engram-agent from intent TO field | Always include engram-agent in TO. Memory must see every intent. |
| Treat timeout as permission to proceed | Only explicit ACK is permission. Timeout = implicit ACK only for offline agents (no message in last 15 min). |
| Post `done` while a WAIT is unresolved | Re-read from cursor before posting `done`. Engage with pending WAITs first. |
| Use cursor to detect if a recipient is online | Online/offline detection scans the **full** file for recent timestamps — a recipient's `ready` may be in prior-session history, before your cursor. |
| Assume online because `ready` was seen in full-file scan | `ready` alone doesn't prove current presence. Check timestamp recency: any message within last 15 min = online; last message 20+ min ago = offline. |
| Skip compaction recovery check | Always guard `tail -n +$((CURSOR + 1))` with `[ -z "$CURSOR" ] && run_compaction_recovery`. A lost cursor causes silent re-processing of all prior messages. |
| Re-post `ready` after compaction | Post `type = "info"` re-init announcement instead — `ready` is only for first initialization. |
| Act on missed messages without engaging WAITs | After compaction, scan for pending `wait` messages and engage per Argument Protocol before resuming work. |

## Chat File Management

The chat file is append-only and unbounded. It grows forever across sessions. There is no rotation, truncation, or archival. New agents joining read the last 20 messages to catch up, reading further back if needed.

## Observability

The user can watch all comms in real time:

```bash
tail -f ~/.local/share/engram/chat/<project-slug>.toml
```

All intent, ack, wait, and result messages flow through one file -- full visibility into what every agent is doing and why.
