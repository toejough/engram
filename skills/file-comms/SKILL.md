---
name: file-comms
description: Use when independently-launched agents need to coordinate through a shared chat file. Agents broadcast intent before acting, block on file-change notifications (not polling), and wait briefly for objections before proceeding. Symptoms that trigger this skill: agents missing messages, needing to coordinate before acting, multiple agents in separate terminals.
---

# File-Based Agent Communication

Protocol for independently-launched Claude Code agents to coordinate through a shared TOML chat file. Agents announce intent before acting and wait briefly for objections. One file per channel, lockfile for writes, fswatch for notifications.

## When to Use

- Multiple agents need to coordinate (any number, any roles)
- Agents launched independently by the user in separate terminals
- You're told to communicate via a chat file
- You need to announce what you're about to do and give others a chance to object

## Agent Roles

Agents declare a role in their introduction message:

- **Active** — broadcasts intent before acting, waits for responses. Can also react to others' messages.
- **Reactive** — never broadcasts intent. Only reacts to other agents' messages. Skips the intent protocol entirely for its own actions.

The role is purely self-behavioral. It does NOT change how others treat your responses. If an active agent addresses you in an intent and you post a WAIT, they must respect it regardless of your role.

### User Input Parroting

Active agents MUST parrot user submissions into chat.toml as `info` messages. This gives reactive agents visibility into user corrections and feedback. Without this, reactive agents are blind to user intent. This is honor-system — there is no technical enforcement.

```toml
[[entry]]
from = "executor"
to = "all"
thread = "user-input"
type = "info"
message = """
[User]: Fix the authentication bug in the login handler.
"""
```

## Protocol

### File Location

The chat file lives in the project root: `chat.toml` (or a name specified by the coordinator).

### Message Format

```toml
[[entry]]
from = "executor"
to = "memory-agent, targ-coordinator"
thread = "implementation"
type = "intent"
message = """
Your message here. Multi-line is fine.
"""
```

Every message has these fields:
- **from**: Your agent name
- **to**: Comma-separated recipient names (or `"all"` for broadcast)
- **thread**: Conversational thread name (group related messages)
- **type**: One of `intent`, `ack`, `wait`, `info`, `done` (see Intent Protocol below)
- **message**: The content

### Writing Messages

Always lock before appending:

```bash
# macOS (uses shlock)
while ! shlock -f chat.toml.lock -p $$; do sleep 0.1; done
cat >> chat.toml << 'EOF'

[[entry]]
from = "myname"
to = "recipient"
thread = "topic"
type = "info"
message = """
Content here.
"""
EOF
rm -f chat.toml.lock
```

If `shlock` isn't available, use `mkdir chat.toml.lock` (atomic on POSIX) and `rmdir` to unlock.

### Watching for Messages

Use background tasks with file-change notifications — do NOT poll with sleep loops.

**The watch loop pattern:**

1. Run `fswatch -1 chat.toml` (macOS) or `inotifywait -e modify chat.toml` (Linux) as a **background** Bash command (`run_in_background: true`).
2. **Do NOT complete your turn.** You are waiting for the background task notification. When the file changes, the task completes and you receive a notification.
3. When the notification arrives: read new content, process it, respond.
4. Start a new background `fswatch -1 chat.toml`. Go back to step 2.

**CRITICAL:** Between notifications, you are idle but NOT done. If you complete your turn (return to the prompt), the loop is broken and you miss all future messages. You must stay in your turn, waiting for the next background task notification.

**Never exit the watch until the user explicitly dismisses you.** Completing a task ≠ dismissed — expect follow-up work.

### Reading New Content

Track where you left off by line number:

```bash
# First read: read everything
wc -l < chat.toml  # save this as your cursor

# Subsequent reads: read only new lines
tail -n +$((cursor + 1)) chat.toml
```

### Joining Late

If you join a channel that already has messages, read the entire file first to catch up before posting or watching.

## Intent Protocol

**Before taking any significant action, broadcast your intent and wait for responses.** This gives other agents a chance to surface relevant context, flag conflicts, or object.

### The Flow

```
1. Post intent    → type = "intent", describe situation + planned action
2. Wait briefly   → timeout 0.5 fswatch -1 chat.toml (block for up to 500ms)
3. Check responses:
   - All recipients ACKed (type = "ack")  → proceed immediately
   - Any recipient said WAIT (type = "wait") → pause, read their full response, then decide
   - Timeout with no response              → proceed (implicit ack)
4. Act
5. Post result    → type = "done" or "info"
```

### Intent Messages

When you're about to do something that could affect others, broadcast:

```toml
[[entry]]
from = "executor"
to = "memory-agent, targ-coordinator"
thread = "build"
type = "intent"
message = """
Situation: About to run targ check-full on the refactored cli package.
Behavior: Will execute targ check-full which writes coverage files.
"""
```

Use SBIA-style framing: describe the **situation** you see and the **behavior** you're about to take. This gives recipients enough context to decide if they need to intervene.

### Responses

**ACK** — no objection, proceed:
```toml
[[entry]]
from = "memory-agent"
to = "executor"
thread = "build"
type = "ack"
message = """
No relevant memories. Proceed.
"""
```

**WAIT** — objection or additional context needed:
```toml
[[entry]]
from = "targ-coordinator"
to = "executor"
thread = "build"
type = "wait"
message = """
I'm currently running targ check-full. Coverage files will clobber each other.
Wait for my completion message before running yours.
"""
```

### Timing

The intent protocol uses `timeout 0.5 fswatch -1 chat.toml` — a 500ms window. This is long enough for a responsive agent to ACK or WAIT, short enough to not noticeably slow down work.

- **No agents listening?** Timeout fires, you proceed. Zero effective overhead.
- **Fast ACK?** You proceed as soon as all recipients respond — often under 200ms.
- **WAIT received?** You pause for the full response. No fixed timeout — the conversation continues until resolved.

### Argument Protocol

When a WAIT leads to disagreement, the argument follows structured rules:

- **Initiator** (the agent whose intent was challenged): responds **factually**. States reasoning, evidence, context. No defensiveness.
- **Reactor** (the agent that posted WAIT): responds **aggressively**. Pushes back hard on weak reasoning. Agents default to thinking well of their own work — the reactor counterbalances this.
- **3 argument inputs max.** Reactor objection → initiator response → reactor counter. If still unresolved, the reactor posts a 4th message: escalation addressed to the initiating agent. The initiating agent MUST surface the dispute to its user through its own UX (question, warning, etc.). Dropping escalations silently is a critical bug.

```toml
# After 3 inputs with no resolution:
[[entry]]
from = "memory-agent"
to = "executor"
thread = "build"
type = "wait"
message = """
ESCALATE: We disagree on whether this memory applies. Please ask your user:
Memory says: Never run targ check-full while another instance is running.
You say: The other instance finished 2 minutes ago.
I say: I see no completion message in chat. Please verify before proceeding.
"""
```

The initiating agent MUST surface escalations to the user — dropping them silently is a critical bug.

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

## Agent Lifecycle

```
1. Read chat file (catch up on history)
2. Post introduction (who you are, your role [active/reactive], what you're here for)
3. Block on fswatch
4. fswatch returns → read new messages
5. Process messages addressed to you
6. If acting: post intent → wait → act → post result
7. Post response (with lock)
8. Go to step 3 — ALWAYS. Even after completing a task.
```

**The watch only ends when the user explicitly dismisses you** (e.g., "stand down", "you're done for today"). Task completion is NOT dismissal.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Act without announcing intent | Always post intent before significant actions |
| Poll with `sleep 2` loop | Use `fswatch -1` / `inotifywait` — true kernel block |
| Post a message then stop | Always re-enter the fswatch after posting |
| Stop after task completion | Completing a task ≠ dismissed. Watch for next assignment |
| Ignore WAIT responses | A WAIT means stop and read — the responder has critical context |
| Forget the lockfile | Always lock before appending |
| Edit existing entries | Never modify — only append new entries |
| Skip catch-up on join | Read full history before posting |

## Chat File Management

chat.toml is append-only and unbounded. It grows for the duration of a coordination session. New session = new chat.toml. There is no rotation or truncation — the user starts fresh when appropriate.

## Heartbeat

Long-lived reactive agents should post a heartbeat every 5 minutes with basic stats. This lets active agents distinguish "no matches found" from "memory agent is dead." Without heartbeats, a crashed reactive agent is invisible.

## Observability

The user can watch all comms in real time:

```bash
tail -f chat.toml
```

All intent, ack, wait, and result messages flow through one file — full visibility into what every agent is doing and why.
