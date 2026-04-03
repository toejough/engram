# Memory Agent Skill Design

Replaces the engram Go binary and hook infrastructure with a pure Claude Code skill that reads/writes memory TOML files directly. Coordinates with other agents via the file-comms protocol.

## Part 1: File-Comms Protocol Updates

### Agent Roles

Agents declare a role in their introduction message:

- **Active** — broadcasts intent before acting, waits for responses. Can also react to others.
- **Reactive** — never broadcasts intent. Only reacts to other agents' messages. The role is purely self-behavioral; it does not change how other agents treat this agent's responses. Active agents still wait for a reactive agent's WAIT/ACK like anyone else's.

### User Input Parroting

Active agents using file-comms must parrot user submissions into chat.toml as `info` messages. This gives reactive agents (like the memory agent) visibility into user corrections and feedback without requiring hook infrastructure.

**Enforcement:** Honor-system. There is no technical enforcement mechanism. If an active agent doesn't parrot, the memory agent is blind to user corrections from that agent. A reactive agent cannot detect the absence of messages it never sees. This is a known limitation of V1.

### Argument Protocol

When any agent posts a `wait` objecting to another agent's intent, the two argue:

- **Initiator (the agent whose intent was challenged):** responds factually. States reasoning, evidence, and context without defensiveness.
- **Reactor (the agent that posted the WAIT):** responds aggressively. Pushes back hard on weak reasoning. Agents default to thinking well of their own work — the reactor's job is to counterbalance that.
- **3 argument inputs max.** Reactor objection, initiator response, reactor counter-response. If still unresolved, the reactor posts a 4th message: an escalation addressed to the initiating agent. The initiating agent surfaces the dispute to its user through its own UX (question, warning, etc.). The dispute is never escalated via chat.toml alone — the user sees it in their agent's terminal.
- **Early concession.** If the initiator agrees with the reactor after the first objection, the initiator posts an `ack` to end the argument early. No need to use all 3 inputs.
- **Resolution recording.** After the argument resolves (agreement, concession, user decision, or timeout), the reactor updates the memory's outcome counters (per the Surfacing Flow) and posts an `info` with the resolution.

### Chat File Management

chat.toml is append-only and unbounded. It grows for the duration of a coordination session. New session = new chat.toml file. There is no rotation or truncation mechanism — the user starts fresh when appropriate.

### Heartbeat

Reactive agents should post a heartbeat every 5 minutes:

```toml
[[entry]]
from = "memory-agent"
to = "all"
thread = "heartbeat"
type = "info"
message = "alive | 269 memories loaded | 15 intents processed | 2 surfaced | queue: 0"
```

This gives active agents a way to check if the memory agent is still running. Without it, a crashed memory agent is indistinguishable from one that found no matches.

## Part 2: Memory Agent Skill

### Identity

- **Role:** reactive
- **Responsibilities:** memory surfacing + memory learning (combined)
- **Splitting signal:** if the unprocessed intent queue exceeds 5, or if the agent is routinely still processing when the next intent arrives, post a warning to chat.toml and recommend splitting. The user decides.

### Matching Strategy

**V1: Situations-only LLM judgment.** The memory agent loads only the `situation` field from each memory into its context, along with the memory's filename (slug). It uses its own reasoning to match situations against intents. Full SBIA fields are only loaded when a match is found — the subagent reads the complete memory file for the argument.

**Why situations-only:** The situation field is the matching key (~30-50 tokens each). The behavior/impact/action fields (~150-250 tokens) are only needed after a match, when arguing about whether it applies. Loading just situations gives ~5x capacity: ~1000 memories before hitting context limits, vs ~200 with full records.

**Scale limits:** Works for up to ~1000 memories (~40k tokens of situation content). The current corpus of 269 memories is well within this limit. Beyond ~1000, a pre-filtering mechanism (BM25 or similar) becomes necessary — tracked in #471. The ~30-50 token estimate per situation is an assumption that should be validated against the actual corpus before implementation.

**What "match" means concretely:**
1. Does the intent's situation overlap with a memory's situation? (Same context, same type of work, same tools/files involved)
2. The agent uses judgment, not string matching. Novel phrasings of the same situation should be caught.
3. When a situation matches, the subagent reads the full memory file and judges whether the intended behavior resembles the memory's "behavior to avoid."

**Token economics:** With 269 memories at ~40 tokens each, the agent consumes ~11k tokens of situation content. At 100 intents per session, that's ~1.1M input tokens for situation matching. Full SBIA is only loaded for matched memories (typically 0-3 per intent).

**Context window overflow:** With situations-only loading at ~40 tokens each, the 1M context window supports ~25,000 memories. This is a non-issue for the foreseeable future. If it ever becomes relevant, the agent should stop loading the lowest-value memories (high surfaced_count with zero followed_count) and post a warning.

### Main Loop

```
post introduction (role = reactive, memory count)
read situation field + slug from all memory TOML files into context
initialize cross-iteration state: recent_intents = []
post heartbeat
start fswatch -1 chat.toml as background task
loop:
    wait for background task notification  # do NOT complete turn
    re-read only modified memory files (track per-file mtimes)
    read new messages from chat.toml (from cursor)
    for each intent message:
        add to recent_intents (keep last 20)
        check memories for situation match
        if match found:
            increment SurfacedCount on the memory (with lock)
            spawn subagent to argue (max 3 concurrent, no two on same thread)
    for each user-parroted message:
        check for explicit corrections (always/never/remember)
        if found: extract SBIA, create new memory (with lock)
    for each done/info message:
        correlate with recent_intents to detect failures
        dedup: skip if this intent already triggered a missed_count increment
        if failure matches existing memory: increment MissedCount (with lock)
        if failure has no matching memory: create new memory (with lock)
    if 5+ minutes since last heartbeat: post heartbeat (with queue depth)
    start new fswatch -1 chat.toml as background task
    loop — go back to waiting for notification
```

**Background task loop pattern:** The agent runs `fswatch -1 chat.toml` as a background Bash command. When the file changes, the background task completes and the agent receives a notification — this is the trigger to process. After processing, start a new background fswatch. The agent must NEVER complete its turn while the loop is running.

**Cross-iteration state:** The agent maintains `recent_intents` — the last 20 intent messages with their threads and originating agents. This allows correlating outcomes (done/info messages) with the intents that preceded them across multiple fswatch wakeups. This is best-effort — failures discovered beyond the 20-intent window cannot be correlated.

### Surfacing Flow

1. Memory agent wakes on chat.toml change.
2. Reads new `intent` messages from active agents.
3. For each intent: matches the intent's situation against loaded memory situations using LLM judgment (see Matching Strategy).
4. If a match with negative impact is found:
   - Acquire per-file lock for the matched memory, increment `surfaced_count` + update `updated_at`, write atomically, unlock.
   - If fewer than 3 subagents are running AND no subagent is already active on this thread, spawn a subagent. Otherwise queue.
   - The subagent:
     - Reads the full memory TOML file (all SBIA fields) for the matched memory
     - Identifies itself in chat.toml as `memory-agent/sub-N` (where N is 1, 2, or 3)
     - Judges whether the intended behavior resembles the memory's "behavior to avoid"
     - If it does, posts `wait` to chat.toml with the memory's SBIA fields and reasoning
     - If it doesn't (situation matched but behavior is fine), posts nothing — false positive, return silently
     - Follows the argument protocol (aggressive reactor, 3 argument inputs max, 4th escalation if unresolved)
     - After resolution: **re-reads the memory file fresh** (not from cached data), acquires per-file lock, increments the appropriate counter, writes atomically, unlocks
     - Posts `info` with the resolution outcome for observability
   - Memory agent returns to watch loop immediately (subagent handles the rest).
5. If no match: silent. No ACK.

### Learning Flow

Two triggers, both detected from chat.toml messages:

**Trigger 1: Explicit user corrections.** Active agents parrot user submissions. The memory agent scans for correction patterns. Detection uses two confidence tiers:

- **High confidence** (auto-create): correction language immediately following an agent's `done` or `info` message. Clear signals: "that's wrong", "never do X", "always do Y", "remember this", "stop doing X". Creates memory with `initial_confidence = 1.0`.
- **Ambiguous** (flag, don't auto-create): correction-like language in isolation or mid-conversation without a clear preceding action. Signals: bare "no", "wrong" in a longer sentence, "revert" used metaphorically. Post `info` to chat.toml asking the user to confirm if this should be a memory. Creates with `initial_confidence = 0.7` if confirmed.

When creating:
- Extract SBIA fields from the user statement and surrounding chat context. If fields are incomplete (e.g., impact is unclear), post a message to chat.toml asking the active agent to prompt its user for the missing context
- Check existing memories for duplicates using LLM judgment: does any existing memory cover the same situation AND recommend the same (or contradictory) action? If contradictory, the new memory supersedes — update the existing memory's fields rather than creating a duplicate.
- If genuine duplicate: skip creation, no action needed
- If new or superseding: lock, write/update TOML file atomically with appropriate counters, unlock
- Post `info` to chat.toml confirming what was learned (or what was updated)

**Trigger 2: Observed failures.** The memory agent detects failures by correlating `recent_intents` with subsequent messages.

**Concrete failure signals (same agent, same thread):**
- Agent posts `done`/`info` → user-parroted correction follows. High confidence.
- Agent posts `done`/`info` → same agent posts a new intent explicitly reversing the action ("reverting", "undoing", "going back to"). High confidence.
- Another agent posts `wait` with evidence of a problem caused by the action. Medium confidence.

**Not a failure:** Agent adjusts approach mid-stream before posting `done` (that's refinement). Agent changes approach based on new information from another agent (that's coordination).

When a failure is detected:

- Check if an existing memory covers this situation
- If existing memory matches:
  - This is a missed surfacing — the memory was relevant but wasn't caught during the intent phase
  - Lock per-file, increment `missed_count`, write atomically, unlock
  - Post `info` noting the miss
  - Do NOT create a duplicate
- If no existing memory matches:
  - Construct SBIA: situation from the intent, behavior from what the agent did, impact from the failure
  - Action: the corrective behavior that worked, OR "consider alternative approaches" if no obvious fix
  - Set `initial_confidence` based on signal strength: 0.7 for high-confidence failures, 0.2 for inferred backtracking
  - Lock, write new TOML atomically, unlock
  - Post `info` confirming what was learned

**Rate limiting:** If the memory agent creates more than 5 memories in 10 minutes, post a warning to chat.toml suggesting the user review and consolidate. Continue creating, but flag the pace.

### Memory File Operations

The memory agent reads and writes TOML files directly. No engram binary dependency.

**Reading:** glob `~/.claude/engram/data/memories/*.toml`, parse each file. Track per-file mtimes — only re-read files that changed since last loop iteration.

**Atomic writes:** Always write to a temp file in the same directory, then rename:

```bash
# Write to temp
cat > ~/.claude/engram/data/memories/.tmp-memory-slug.toml << 'EOF'
...updated content...
EOF
# Atomic rename
mv ~/.claude/engram/data/memories/.tmp-memory-slug.toml ~/.claude/engram/data/memories/memory-slug.toml
```

This prevents corruption if the process crashes mid-write.

**Per-file locking:** Lock individual memory files, not the whole directory:

```bash
# Acquire per-file lock (with stale lock recovery)
lockfile=~/.claude/engram/data/memories/memory-slug.toml.lock
if [ -f "$lockfile" ]; then
    lock_pid=$(cat "$lockfile" 2>/dev/null)
    if [ -n "$lock_pid" ] && ! kill -0 "$lock_pid" 2>/dev/null; then
        rm -f "$lockfile"  # stale lock, owner is dead
    fi
fi
while ! shlock -f "$lockfile" -p $$; do sleep 0.1; done

# ... read, modify, atomic write ...

# Release
rm -f "$lockfile"
```

Use `mkdir` fallback if `shlock` unavailable. For `mkdir` locks, if older than 300 seconds (>= subagent timeout), assume stale. Note: PID-based stale detection only works on the same machine.

**No multi-file locking.** An agent must never hold locks on more than one memory file simultaneously. This prevents deadlocks. If an operation needs to read multiple files (e.g., duplicate detection during learning), read without locks, then lock only the file being written.

**Opportunistic cleanup:** When writing a memory file for any reason, strip the `pending_evaluations` field if present. This gradually cleans up deprecated data without a dedicated migration.

**Subagents always re-read before writing.** A subagent must read the memory file fresh immediately before its locked write, not use cached data from when the argument started.

**Field changes:**

| Field | Type | Default | Purpose |
|-------|------|---------|---------|
| `schema_version` | int | 1 | NEW: Memory file schema version |
| `missed_count` | int | 0 | NEW: Times this memory was relevant but not surfaced during the intent phase |
| `initial_confidence` | float | 1.0 | NEW: Confidence at creation. 1.0 = explicit user correction, 0.7 = clear failure, 0.2 = inferred |
| `pending_evaluations` | - | - | DEPRECATED: Strip on write. Was used by the removed Go evaluate pipeline. |

All other existing fields remain unchanged.

### Subagent Management

**Concurrency limit:** Maximum 3 concurrent subagents. If a 4th match arrives, it is queued and processed when a slot opens.

**Thread exclusivity:** No two subagents on the same thread. If a thread already has an active subagent, queue the new match.

**Naming convention:** Subagents use monotonically increasing IDs: `memory-agent/sub-1`, `memory-agent/sub-2`, etc. IDs are never reused within a session, even when slots free up. This prevents message confusion from ID reuse.

**Message routing:** Messages addressed to `memory-agent` go to the main agent only. Subagents only receive and respond to messages on their specific argument thread.

**Failure handling:** If a subagent fails (crashes, times out after 5 minutes, or otherwise doesn't post a resolution):
- SurfacedCount was already incremented (correct — the memory was surfaced)
- No outcome counter gets incremented (correct — no outcome was observed)
- The gap between SurfacedCount and outcome counts reflects real uncertainty, not a bug
- The memory agent posts an `info` message noting the subagent timeout

### Performance Tracking

The memory agent tracks in its own context (not persisted):

| Metric | Signal | Action |
|--------|--------|--------|
| intents seen vs checked | Agent overwhelm | Report to user if queue > 5, recommend split |
| Subagent queue depth | Concurrency bottleneck | Report if consistently > 0 |
| SurfacedCount vs outcome counts | Outcome tracking gaps | Gap = subagent failures. Effectiveness = followed / (followed + not_followed). Unresolved = surfaced - sum(outcomes). |
| MissedCount | Surfacer matching quality | High = matching needs improvement |

## Part 3: Removal Scope

### Removed

- **Go code for surfacing, evaluation, correction** — replaced by the memory agent skill
- **All hooks** — SessionStart, UserPromptSubmit, Stop hook configurations
- **Migration commands** — `migrate-sbia`, `migrate-scores`, `migrate-slugs` (one-time, already applied)
- **memory-triage skill** — references removed engram CLI commands. Maintenance functionality deferred to #471.

### Retained from binary

- **`engram recall`** — cross-session context search. Fast, cheap, doesn't consume agent context. Useful as a utility called from hooks or subagents.
- **`engram show`** — inspect memory details from the terminal.
- **Supporting Go code** — `internal/recall/`, `internal/memory/` (record types), `cmd/engram/` (CLI wiring for retained commands). Dead code for removed commands should be cleaned up.

### Preserved

- **Memory TOML files** — `~/.claude/engram/data/memories/*.toml` (the data store)
- **Skill files** — the new memory agent skill and updated file-comms skill
- **engram binary** — stripped down to `recall` and `show` commands only
- **Repo structure** — engram repo contains skills + a slim Go binary

### Migration Path

**User experience:** After this change, memory surfacing requires running the memory agent in a separate terminal alongside other agents using file-comms. Start interactively:

```bash
# In a separate terminal, in the same project directory:
claude
# Then tell it: "You are the memory agent. Use the memory-agent and file-comms skills. Chat file: ./chat.toml."
```

**Graceful degradation:** When no memory agent is running, intents time out after 500ms and proceed normally. There is no error, no missing dependency — the system works exactly as it would without memory, just without the memory safety net. This is by design.

**Rollback path:** Git. If the skill-based approach doesn't work, `git revert` restores the Go binary and hooks.

**Skill audit:** Any skill in `~/.claude/skills/` that references removed `engram` CLI commands must be audited and updated or removed. Known affected: `memory-triage` (removed). The `recall` skill can remain since the binary retains the `recall` command.

### Deferred (tracked in #471)

Functionality from the Go binary not carried over, to be investigated for future skills:

- Maintenance diagnosis (decision tree for low-effectiveness memories)
- Consolidation (LLM-based duplicate detection and merge proposals)
- Adapt analysis (policy effectiveness and threshold tuning)
- Apply/reject proposal workflow
- Refine (re-extract SBIA from transcripts)
- BM25 scoring (efficient pre-filter at scale — needed when memories exceed ~1000 with situations-only loading)
- Cold-start budgeting (limit unproven memories)
- Transcript suppression (avoid resurfacing recently mentioned memories)

### Deferred (tracked in #470)

- Hook-based strengthening of the file-comms intent protocol (PreToolUse hooks that write intents automatically)
