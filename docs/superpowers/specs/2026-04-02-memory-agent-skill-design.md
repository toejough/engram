# Memory Agent Skill Design

Replaces the engram Go binary and hook infrastructure with a pure Claude Code skill that reads/writes memory TOML files directly. Coordinates with other agents via the file-comms protocol.

## Part 1: File-Comms Protocol Updates

### Agent Roles

Agents declare a role in their introduction message:

- **Active** — broadcasts intent before acting, waits for responses. Can also react to others.
- **Reactive** — never broadcasts intent. Only reacts to other agents' messages. The role is purely self-behavioral; it does not change how other agents treat this agent's responses. Active agents still wait for a reactive agent's WAIT/ACK like anyone else's.

### User Input Parroting

Active agents using file-comms must parrot user submissions into chat.toml as `info` messages. This gives reactive agents (like the memory agent) visibility into user corrections and feedback without requiring hook infrastructure.

### Argument Protocol

When a reactive agent posts a `wait` objecting to an active agent's intent, the two argue:

- **Initiator (the agent whose intent was challenged):** responds factually. States reasoning, evidence, and context without defensiveness.
- **Reactor (the agent that posted the WAIT):** responds aggressively. Pushes back hard on weak reasoning. Agents default to thinking well of their own work — the reactor's job is to counterbalance that.
- **3 inputs max.** Reactor objection, initiator response, reactor counter-response. If unresolved after 3, the reactor posts an escalation message addressed to the initiating agent. The initiating agent surfaces the dispute to its user through its own UX (question, warning, etc.). The dispute is never escalated via chat.toml alone — the user sees it in their agent's terminal.
- **Resolution recording.** After the argument resolves (agreement, user decision, or timeout), the reactor records the outcome.

## Part 2: Memory Agent Skill

### Identity

- **Role:** reactive
- **Responsibilities:** memory surfacing + memory learning (combined)
- **Splitting signal:** if `intents_seen >> intents_checked`, the agent is overwhelmed and should be split into separate surfacer and learner agents

### Main Loop

```
post introduction (role = reactive)
load all memory TOML file paths
loop:
    fswatch -1 chat.toml          # kernel block, zero CPU
    read new messages from chat.toml
    for each intent message:
        check memories for situation match
        if intended behavior resembles a memory's "behavior to avoid":
            increment SurfacedCount on the memory (with lock)
            spawn subagent to argue
    for each user-parroted message:
        check for explicit corrections (always/never/remember)
        if found: extract SBIA, create new memory (with lock)
    for each done/info/intent-result message:
        check for observed failures (user corrects agent, agent backtracks)
        if failure matches existing memory: increment MissedCount (with lock)
        if failure has no matching memory: create new memory (with lock)
    loop
```

### Surfacing Flow

1. Memory agent wakes on chat.toml change.
2. Reads new `intent` messages from active agents.
3. For each intent: reads memory TOML files from `~/.claude/engram/data/memories/`. Matches the intent's situation against memory situations. For matches, judges whether the intended behavior resembles a memory's "behavior to avoid."
4. If a match with negative impact is found:
   - Lock memory directory, increment `SurfacedCount` + update `UpdatedAt`, unlock.
   - Spawn a subagent to handle the argument. The subagent:
     - Posts `wait` to chat.toml with the memory's SBIA fields and reasoning
     - Follows the argument protocol (aggressive reactor, 3 inputs max, escalate via intending agent's UX)
     - After resolution: lock, increment `FollowedCount` or `NotFollowedCount` or `IrrelevantCount`, unlock
   - Memory agent returns to watch loop immediately (subagent handles the rest).
5. If no match: silent. No ACK.

### Learning Flow

Two triggers, both detected from chat.toml messages:

**Trigger 1: Explicit user corrections.** Active agents parrot user submissions. The memory agent scans for patterns: "always do X", "never do Y", "remember this", direct corrections ("no, that's wrong because..."). When detected:

- Extract SBIA fields from the user statement and surrounding chat context
- Check existing memories for near-duplicates (similar situation + behavior)
- If duplicate exists: skip creation, no action needed
- If new: lock, write new TOML file with zeroed counters, unlock
- Post `info` to chat.toml confirming what was learned

**Trigger 2: Observed failures.** An agent states intent, acts, and the result is bad — the user corrects the agent, tests fail, the agent backtracks. The memory agent sees this sequence in chat.toml. When detected:

- Check if an existing memory covers this situation
- If existing memory matches:
  - This is a missed surfacing — the memory was relevant but wasn't caught during the intent phase
  - Lock, increment `MissedCount` on that memory, unlock
  - Post `info` noting the miss
  - Do NOT create a duplicate
- If no existing memory matches:
  - Construct SBIA: situation from the intent, behavior from what the agent did, impact from the failure
  - Action: the corrective behavior that worked, OR "consider alternative approaches" if no obvious fix
  - Lock, write new memory TOML, unlock
  - Post `info` confirming what was learned

### Memory File Operations

The memory agent reads and writes TOML files directly. No engram binary dependency.

**Reading:** glob `~/.claude/engram/data/memories/*.toml`, parse each file.

**Writing:** lock `~/.claude/engram/data/memories/.lock` (via `shlock` on macOS, `mkdir` POSIX fallback), read file, edit fields, write file, unlock. One lockfile for the entire directory.

**New field on memory records:**

| Field | Type | Default | Purpose |
|-------|------|---------|---------|
| `missed_count` | int | 0 | Times this memory was relevant but not surfaced during the intent phase |

All existing fields remain unchanged.

### Subagent Spawning

The memory agent spawns subagents for:

- **Memory arguments** — one subagent per intent that triggers a memory match. Handles the full argument cycle and outcome recording.
- **Concurrent situations** — if multiple intents arrive simultaneously, multiple subagents can run in parallel.

The memory agent itself never gets pulled into long conversations. It stays on the fswatch loop.

### Performance Tracking

The memory agent tracks:

| Metric | Signal |
|--------|--------|
| `intents_seen` vs `intents_checked` | Agent overwhelm — split signal |
| `SurfacedCount` vs outcome counts (Followed + NotFollowed + Irrelevant) | Outcome tracking gaps |
| `MissedCount` | Surfacer matching quality |

## Part 3: Removal Scope

### Removed

- **All Go code** — `cmd/`, `internal/`, `go.mod`, `go.sum`, etc.
- **All hooks** — SessionStart, UserPromptSubmit, Stop hook configurations
- **engram binary** — no build step, no installation, no API token dependency
- **Migration commands** — `migrate-sbia`, `migrate-scores`, `migrate-slugs` (one-time, already applied)

### Preserved

- **Memory TOML files** — `~/.claude/engram/data/memories/*.toml` (the data store)
- **Skill files** — the new memory agent skill and updated file-comms skill
- **Repo structure** — engram repo continues to exist, now containing skills instead of Go code

### Deferred (tracked in #471)

Functionality from the Go binary not carried over, to be investigated for future skills:

- Maintenance diagnosis (decision tree for low-effectiveness memories)
- Consolidation (LLM-based duplicate detection and merge proposals)
- Adapt analysis (policy effectiveness and threshold tuning)
- Apply/reject proposal workflow
- Refine (re-extract SBIA from transcripts)
- Recall (cross-session context search)
- BM25 scoring (efficient pre-filter at scale)
- Cold-start budgeting (limit unproven memories)
- Transcript suppression (avoid resurfacing recently mentioned memories)

### Deferred (tracked in #470)

- Hook-based strengthening of the file-comms intent protocol (PreToolUse hooks that write intents automatically)
