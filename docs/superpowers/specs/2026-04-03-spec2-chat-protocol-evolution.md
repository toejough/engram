# Spec 2: Chat Protocol Evolution

Evolves the file-comms coordination protocol into `use-engram-chat-as`. Assumes Spec 1 (unified memory format) is shipped — memory files exist in the new format and layout. This spec defines the protocol that all agents run on, including new message types and lifecycle events.

**Dependency model:** Spec 1 defines the memory schema. This spec defines the chat protocol. Spec 1's engram-agent consumes this protocol (it must understand `learned` messages), but the engram-agent's internal handling of memories is defined in Spec 1. This spec defines the `learned` message format and semantics — Spec 1 defines what the engram-agent does with them.

## Scope

| In scope | Out of scope |
|----------|-------------|
| Rename file-comms → use-engram-chat-as | Memory format (Spec 1) |
| `learned` message type | engram-agent internals (Spec 1) |
| `ready` message type | engram-tmux-lead orchestration (Spec 3) |
| `shutdown` message type | Shutdown ordering policy (Spec 3) |
| `escalate` message type | Migration script (Spec 3) |
| Project-scoped chat files + lifecycle | Multi-channel chat (deferred) |
| Argument protocol: lead-mediated escalation | Hook-based intent strengthening (deferred) |
| Field renames (entry→message, message→text) | |
| Heartbeat (carried from file-comms) | |
| Skill file structure for use-engram-chat-as | |

## 1. Rename: file-comms → use-engram-chat-as

### Rationale

`file-comms` describes the mechanism. `use-engram-chat-as` describes the action: join engram's chat as a named agent with a role. The name becomes the invocation: `/use-engram-chat-as reactive memory agent named engram-agent`.

### Invocation

```
/use-engram-chat-as <role-description>
```

The role description is free-form text. Examples:

- `reactive memory agent named engram-agent`
- `reviewer named bob, who uses code review skills`
- `/engram-tmux-lead` — special case, loads lead orchestrator behavior (Spec 3)

The skill parses out:
- **Agent name**: extracted from "named X" in the role description. Required.
- **Role type**: `active` if not stated, `reactive` if the word "reactive" appears.
- **Behavioral context**: the full role description, included in the agent's system context.

### Chat File Location

**Before (file-comms):** `chat.toml` in project root. Location chosen ad-hoc by the coordinator.

**After (use-engram-chat-as):** `~/.claude/engram/data/chat/<project-slug>.toml`. Derived from `$PWD` using the same slug convention as `engram recall`.

**Slug derivation:** The project slug is the basename of the git root directory (resolved through symlinks), lowercased, with non-alphanumeric characters replaced by hyphens. If not in a git repo, use the basename of the resolved `$PWD`. Examples:

| `$PWD` | Git root | Slug | Chat file |
|--------|----------|------|-----------|
| `/Users/joe/repos/engram` | `/Users/joe/repos/engram` | `engram` | `~/.claude/engram/data/chat/engram.toml` |
| `/Users/joe/repos/My Project/src` | `/Users/joe/repos/My Project` | `my-project` | `~/.claude/engram/data/chat/my-project.toml` |
| `/tmp/scratch` | (none) | `scratch` | `~/.claude/engram/data/chat/scratch.toml` |

**Symlinks:** Always resolve symlinks before deriving the slug. `readlink -f` (Linux) or `realpath` (macOS) on the git root or `$PWD`. This ensures two agents in symlinked paths to the same repo use the same chat file.

### Chat File Lifecycle

Chat files are **per-session, not persistent**. The move to `~/.claude/engram/data/chat/` provides a deterministic location, not permanence.

- **Session start:** The first agent to join creates the chat file if it doesn't exist.
- **Session end:** The agent that initiates shutdown (lead or standalone active agent) truncates the chat file after all agents have posted their final `done` messages. Truncation = write an empty file, not delete — this avoids race conditions with agents that haven't fully exited yet.
- **Stale detection:** On join, if a chat file already exists, the agent checks the last message's timestamp. If older than 1 hour, treat the file as stale from a previous session: truncate it before posting. If within 1 hour, assume an active session and join normally (read full history, then participate).
- **Timestamp field:** All messages include a `ts` field (ISO 8601, required). This enables reliable stale detection by checking the last message's timestamp regardless of type.
- **No archival.** Old chat sessions are not preserved. The chat file is a coordination channel, not a record. The engram-agent extracts durable knowledge into memory files — that's the persistent layer.

**Migration:** The skill creates the chat directory and file on first use. No migration of old chat.toml files — they were ephemeral session artifacts.

### What Changes in the Protocol

The underlying protocol (locking, fswatch, intent protocol, timing) is **unchanged** from file-comms. Changes:

1. Skill name and invocation syntax
2. Chat file location (project-scoped, deterministic) + lifecycle management
3. New message types: `learned`, `ready`, `shutdown`, `escalate` (sections 2-5)
4. Argument escalation target (section 5)
5. Field renames: `[[entry]]` → `[[message]]`, `message` → `text` (section 6)
6. Heartbeat carried forward with explicit message type catalog entry (section 7)

## 2. Message Type: `learned`

### Purpose

Active agents announce knowledge extracted from their work. This gives the engram-agent a high-confidence signal for fact extraction — the originating agent has direct context about what it learned, making these signals more reliable than the engram-agent inferring facts from general conversation.

### Format

```toml
[[message]]
from = "executor"
to = "engram-agent"
thread = "implementation"
type = "learned"
ts = "2026-04-03T14:30:00Z"
text = """
engram → uses → targ for all build, test, and check operations.
Context: Discovered during build failure — running go test directly misses coverage thresholds.
"""
```

### Semantics

- **Addressed to engram-agent** (or `all`). Other agents may ignore `learned` messages unless they have reason to consume them.
- **Content structure:** Should include subject-predicate-object triples where possible, plus context. Free-form is acceptable — the consuming agent does the parsing.
- **Confidence signal:** `learned` messages are high-confidence signals (same tier as explicit user corrections). This is higher confidence than facts inferred from general conversation.
- **When to emit:** After completing a task that produced reusable knowledge. Not every `done` message needs a corresponding `learned`. Only emit when the agent discovered something that would be useful across sessions.

### Interface Contract with Spec 1

This spec defines the `learned` message: its format, when to emit, confidence level, and addressing convention. Spec 1 defines how the engram-agent processes `learned` messages (triple extraction, deduplication, storage). The contract:

- **Spec 2 guarantees:** `learned` messages have a `text` field containing knowledge, addressed to the engram-agent or `all`. The text SHOULD contain SPO triples but MAY be free-form.
- **Spec 1 is responsible for:** Defining the engram-agent's extraction, dedup, and storage behavior for `learned` messages. This spec does not dictate those internals.
- **No response expected:** The engram-agent does NOT ack or wait on `learned` messages. Silent processing. If extraction fails, the message is dropped (best-effort signal).

## 3. Message Type: `ready`

### Purpose

Agents announce they have completed initialization and are watching the chat. This solves a timing problem: without `ready`, an agent that posts an intent immediately after joining doesn't know if other agents are listening yet. The `ready` message provides a synchronization point.

### Format

```toml
[[message]]
from = "engram-agent"
to = "all"
thread = "lifecycle"
type = "ready"
ts = "2026-04-03T14:25:00Z"
text = "Loaded 47 feedback memories, 23 facts. Watching for intents."
```

### Semantics

- **Posted once**, after the agent has: (1) read the full chat history, (2) loaded any resources it needs (memories, configs), (3) started its fswatch loop.
- **Addressed to `all`**. Every agent posts `ready` regardless of role.
- **The `ts` field is required** on `ready` messages (used for stale detection and timeout calculation).
- **The text field** contains agent-specific initialization stats. No required format — it's informational.

### Who Waits for Whom

The `ready` message is a synchronization primitive. Who waits depends on the setup:

- **Lead setup (Spec 3):** The lead knows which agents it spawned. It waits for `ready` from each spawned agent before routing user work. The lead defines the expected agent set — this is orchestrator logic (Spec 3), not protocol logic.
- **Standalone setup (no lead):** Agents don't wait for each other's `ready` messages. They post `ready` and immediately enter the watch loop. The `ready` message is informational — it tells other agents "I'm here now" but doesn't gate anything.
- **Late joiners:** An agent that joins after intents have already been sent reads the full chat history on join (existing behavior). The `ready` message announces it's now watching — it doesn't replay missed intents.
- **Reactive agents:** Post `ready` but do not wait for anyone else. Reactive agents are listeners — they process what arrives, whenever it arrives.
- **Timeout:** If an agent hasn't posted `ready` within 30 seconds of being launched, the waiting agent (typically the lead) proceeds and logs a warning. A missing `ready` is not fatal — it means the agent may miss early intents.

### Lifecycle Position

The agent lifecycle (from file-comms) becomes:

```
1. Read chat file (catch up on history; check for stale session)
2. Load resources (memories, configs, etc.)
3. Start fswatch loop
4. Post ready message (with ts)                ← NEW
5. Block on fswatch
6. fswatch returns → read new messages
7. Process messages addressed to you
8. If acting: post intent → wait → act → post result
9. Post response (with lock)
10. Go to step 5 — ALWAYS
```

## 4. Message Type: `shutdown`

### Problem

file-comms has no shutdown protocol. "The watch only ends when the user explicitly dismisses you" — but there's no message type for dismissal, no cleanup, and no way for the lead to shut down agents programmatically.

### Format

```toml
[[message]]
from = "lead"
to = "engram-agent"
thread = "lifecycle"
type = "shutdown"
ts = "2026-04-03T16:00:00Z"
text = "Session complete."
```

`shutdown` is a first-class message type, not a magic prefix. It can be sent by the lead (or by any active agent in a non-lead setup) to specific agents or to `all`.

### Agent Shutdown Behavior

When an agent receives a `shutdown` message addressed to it (or to `all`):

1. **Stop accepting new work.** Do not process any further intents or messages after the shutdown.
2. **Complete in-flight work.** If currently executing an action, finish it and post the result.
3. **Post a final `done` message:**
   ```toml
   [[message]]
   from = "engram-agent"
   to = "all"
   thread = "lifecycle"
   type = "done"
   ts = "2026-04-03T16:00:05Z"
   text = "Shutting down. Session stats: surfaced 12 memories, learned 5 facts, 3 arguments (2 won, 1 escalated)."
   ```
4. **Exit the fswatch loop.** The agent's turn is complete.

### Shutdown Ordering

This spec defines the `shutdown` message type and per-agent behavior. **Shutdown ordering** — which agents to shut down first, timeout policies, fallback mechanisms for unresponsive agents — is orchestration policy defined by Spec 3 (engram-tmux-lead). In a standalone setup without a lead, the user dismisses agents directly and each agent handles its own shutdown.

### User-Initiated Shutdown

The user can dismiss agents directly with phrases like "stand down", "you're done", "shut down". In a lead setup, the user says this to the lead, and the lead issues `shutdown` messages. In a standalone setup, the agent recognizes the dismissal and exits its loop directly (posting a final `done` message before exiting).

## 5. Message Type: `escalate`

### Rationale

Escalation is a first-class protocol event with defined semantics, expected responses, and specific handling requirements. It deserves its own message type rather than being a magic prefix inside `wait`.

### Before (file-comms)

After 3 argument inputs with no resolution, the reactor posts an escalation as a `wait` with `ESCALATE:` prefix, addressed to the initiating agent. The initiating agent surfaces the dispute to its user.

### After (use-engram-chat-as)

After 3 argument inputs with no resolution, the reactor posts an `escalate` message **addressed to the lead**.

### Format

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

### Required Fields in Escalation Text

The `text` field of an `escalate` message MUST include:
- Summary of both positions
- Specific ask for the user (question to answer, decision to make)

### Rules

1. **Escalation target is always the lead.** Even if the lead is one of the arguing parties (e.g., lead vs. engram-agent), the escalation goes to the lead because only the lead has user access. The lead MUST surface lead-involved escalations fairly — presenting both sides without bias.
2. **The lead MUST surface the escalation to the user.** Dropping escalations silently is a critical bug.
3. **The lead posts the user's decision** as an `info` message on the same thread, so both parties see the resolution.
4. **Resolution recording unchanged:** After resolution, the reactor posts an `info` message with the outcome.
5. **Standalone agents (no lead):** If agents are running without a lead (direct user launch, no tmux orchestrator), the old behavior applies — escalation goes to the initiating agent's UX. The skill detects standalone mode by checking whether a `ready` message from an agent named `lead` exists in the chat history. No `lead` ready → standalone mode → escalate to initiating agent.

## 6. Field Renames

### Two Renames

This is a **clean break**, not a gradual migration. The new protocol uses both renames. Old chat files are ephemeral and not carried across sessions (see chat file lifecycle in section 1).

| Field | Before (file-comms) | After (use-engram-chat-as) |
|-------|--------------------|-----------------------------|
| TOML array key | `[[entry]]` | `[[message]]` |
| Content field | `message = """..."""` | `text = """..."""` |

### Rationale

- `[[entry]]` → `[[message]]`: better describes the content, aligns with the chat metaphor.
- `message` → `text`: the old content field name (`message`) collides with the new array key name (`[[message]]`). A `[[message]]` containing a field called `message` is confusing. `text` is unambiguous.

### Backward Compatibility

**None.** This is a clean break. The field renames coincide with the skill rename (file-comms → use-engram-chat-as) and the chat file relocation. Since:
- Chat files are ephemeral (truncated between sessions)
- The skill name changes (no agent will load the old protocol)
- The chat file location changes (old files are in project root, new files in `~/.claude/engram/data/chat/`)

...there is no scenario where a new agent reads an old-format file or vice versa. No backward-compat parser needed.

### New Message Format

```toml
[[message]]
from = "executor"
to = "engram-agent"
thread = "implementation"
type = "intent"
ts = "2026-04-03T14:30:00Z"
text = """
Situation: About to run targ check-full on the refactored cli package.
Behavior: Will execute targ check-full which writes coverage files.
"""
```

Fields per message:
- **from**: Agent name (required)
- **to**: Comma-separated recipient names or `"all"` (required)
- **thread**: Conversational thread name (required)
- **type**: One of `intent`, `ack`, `wait`, `info`, `done`, `learned`, `ready`, `shutdown`, `escalate` (required)
- **ts**: ISO 8601 timestamp (required on all message types)
- **text**: Message content (required)

## 7. Message Type Catalog

Complete catalog of all message types in use-engram-chat-as:

| Type | Purpose | Emitted by | Response expected |
|------|---------|-----------|-------------------|
| `intent` | Announce situation + planned action before acting | Active agents | ACK/WAIT within 500ms |
| `ack` | No objection, proceed; or early concession in argument | Any agent | No |
| `wait` | Objection, memory to surface, or request to pause | Any agent | Initiator response |
| `info` | Status updates, user-parroted input, resolution recording | Any agent | No |
| `done` | Task/action completed; final message before shutdown | Any agent | No |
| `learned` | Knowledge extracted from work (fact signal for engram-agent) | Active agents | No (silent processing) |
| `ready` | Agent initialization complete, watching chat | Any agent | No (but spawners may wait for it) |
| `shutdown` | Signal agent to exit after completing in-flight work | Lead or active agent | `done` message |
| `escalate` | Unresolved argument, needs user decision via lead | Reactor in argument | Lead surfaces to user |

### Heartbeat

Carried forward from file-comms unchanged. Long-lived reactive agents post a heartbeat every 5 minutes using `type = "info"` on the `heartbeat` thread:

```toml
[[message]]
from = "engram-agent"
to = "all"
thread = "heartbeat"
type = "info"
ts = "2026-04-03T14:35:00Z"
text = "alive | queue: 0 | feedback: 47 loaded | facts: 23 loaded"
```

Heartbeats use `type = "info"` because they are informational status updates, not a distinct protocol event. The `thread = "heartbeat"` convention makes them filterable. Without heartbeats, a dead reactive agent is invisible — active agents can't distinguish "no matches found" from "agent crashed."

## 8. Skill File Structure

The skill `skills/use-engram-chat-as/SKILL.md` replaces `skills/file-comms/SKILL.md`. It contains:

1. **Frontmatter:** Name (`use-engram-chat-as`), description, trigger conditions. Trigger: "when independently-launched agents need to coordinate, when joining a multi-agent session, when told to communicate via engram chat, when using /use-engram-chat-as."
2. **Protocol reference:** All rules from file-comms, updated with changes from this spec
3. **Message type catalog:** All 9 types from section 7
4. **Role descriptions:** Active vs. reactive, with behavioral expectations
5. **User input parroting:** Active agents parrot user submissions as `info` messages (honor-system)
6. **Argument protocol:** Including lead-mediated escalation via `escalate` type
7. **Shutdown protocol:** `shutdown` type and per-agent behavior
8. **Chat file location:** Project-scoped derivation with lifecycle management
9. **Common mistakes table:** Updated with new failure modes

### New Common Mistakes

| Mistake | Fix |
|---------|-----|
| Escalate to initiating agent instead of lead | Check for lead `ready` in chat history; escalate to lead if present |
| Skip `ready` message | Always post `ready` after initialization, before entering watch loop |
| Emit `learned` for trivial observations | Only emit when knowledge is reusable across sessions |
| Ignore `shutdown` message | Exit fswatch loop after completing in-flight work and posting `done` |
| Post intent before others are ready | In lead setup: wait for expected `ready` messages (30s timeout) |
| Use old field names (`[[entry]]`, `message =`) | Clean break: use `[[message]]` and `text =` |
| Assume chat file is persistent across sessions | Chat files are ephemeral — truncated between sessions |
| Skip heartbeat in long-lived reactive agent | Post heartbeat every 5 min so others know you're alive |

## 9. Summary of Protocol Changes from file-comms

| Aspect | file-comms (before) | use-engram-chat-as (after) |
|--------|--------------------|-----------------------------|
| Skill name | `file-comms` | `use-engram-chat-as` |
| Chat file | `chat.toml` in project root | `~/.claude/engram/data/chat/<slug>.toml` |
| Chat lifecycle | "New session = new file" (manual) | Stale detection (1h threshold) + truncation on shutdown |
| TOML array key | `[[entry]]` | `[[message]]` (clean break) |
| Content field | `message = """..."""` | `text = """..."""` (clean break) |
| Message types | intent, ack, wait, info, done | intent, ack, wait, info, done, **learned**, **ready**, **shutdown**, **escalate** |
| Heartbeat | info on heartbeat thread | Unchanged — info on heartbeat thread |
| Escalation target | Initiating agent's UX | Lead agent (falls back to initiating agent if no lead) |
| Escalation type | `wait` with ESCALATE: prefix | First-class `escalate` type |
| Shutdown | "User dismisses you" (informal) | First-class `shutdown` type with `done` response |
| Shutdown ordering | N/A | Defined by Spec 3 (orchestration policy) |
| Agent lifecycle | join → watch → process → loop | join → load → watch → **ready** → process → loop |
| Timestamps | Not in protocol | `ts` field: required on all message types |
