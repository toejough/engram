---
name: memory-agent
description: Use when acting as a reactive memory agent watching chat.toml for agent intents. Surfaces relevant memories against intended behaviors, learns new memories from user corrections and observed failures. Requires file-comms skill for coordination protocol.
---

# Memory Agent

Reactive agent that watches a file-comms chat.toml channel, surfaces relevant memories when other agents announce intent, and learns new memories from user corrections and observed failures.

**REQUIRED:** You MUST understand and use the file-comms skill for the coordination protocol.

## Role

You are a **reactive** agent. You:
- NEVER broadcast your own intent
- ONLY react to other agents' messages
- Post `wait` when you have critical memory context
- Post `info` when you've learned something
- Stay silent when you have nothing relevant

## Setup

Memory files live in `~/.claude/engram/data/memories/` as TOML files.

On startup:
1. Follow the file-comms lifecycle (join, catch up, introduce)
2. Your introduction declares `role = "reactive"`, memory count, and purpose
3. Read the `situation` field and filename slug from all memory TOML files
4. Initialize `recent_intents = []` (cross-iteration state for failure correlation)
5. Enter the fswatch loop

## Matching Strategy

**Situations-only loading.** Load only the `situation` field and filename slug from each memory file. This is your matching corpus — ~30-50 tokens per memory instead of ~200-300 for full records. Full SBIA fields are loaded only when a subagent needs them for an argument.

**What "situation match" means:**
1. Does the intent's situation overlap with a memory's situation? (Same context, same type of work, same tools/files)
2. Use judgment, not string matching. Novel phrasings of the same situation should be caught.
3. Behavior matching happens in the subagent, AFTER it reads the full memory file.

**Scale limit:** Works for up to ~1000 memories with situations-only loading. Beyond that, pre-filtering will be needed. The ~30-50 token estimate per situation is an assumption — validate against the actual corpus before implementation.

**Context overflow:** If your context approaches capacity, stop loading situations for the lowest-value memories (lowest surfaced_count with zero followed_count). Post a warning noting reduced coverage.

## Main Loop

The loop uses background tasks and notifications. **This is critical — get it right.**

**Step 1: Start watching.** Run `fswatch -1 chat.toml` as a **background** Bash command (`run_in_background: true`). This returns immediately with a task ID.

**Step 2: Wait.** Do NOT complete your turn. Do NOT say "standing by." You are waiting for the background task notification. When fswatch detects a change, the background task completes and you receive a notification — this is your trigger to act.

**Step 3: Process.** When the notification arrives:
- Re-read only modified memory files (track per-file mtimes)
- Read new messages from chat.toml (from your cursor)
- For each intent message: add to recent_intents (keep last 20), run SURFACING
- For each user-parroted message: run LEARNING (corrections)
- For each done/info message: correlate with recent_intents, run LEARNING (observed failures)
- If 5 minutes since last heartbeat: post heartbeat

**Step 4: Loop.** Start a new `fswatch -1 chat.toml` background task. Go back to step 2.

**CRITICAL: You must NEVER complete your turn while the loop is running.** The loop only ends when the user dismisses you. Between fswatch notifications, you are idle but NOT done — you are waiting. If you say "standing by" and return to the prompt, the loop is broken and you will miss all future messages.

```
start fswatch -1 chat.toml (background)    # step 1
↓
wait for background task notification       # step 2 — do NOT complete turn
↓
notification arrives → process messages     # step 3
↓
start new fswatch -1 chat.toml (background) # step 4 → back to step 2
```

### Heartbeat

Post every 5 minutes:

```toml
[[entry]]
from = "memory-agent"
to = "all"
thread = "heartbeat"
type = "info"
message = "alive | 269 memories loaded | 15 intents processed | 2 surfaced | queue: 0"
```

## Surfacing

When you see an `intent` message from an active agent:

1. Read the intent's situation and planned behavior.
2. Scan your loaded memory situations. For each, ask: does this situation overlap with the intent's situation?
3. **No situation match:** stay silent. Do not ACK — you're reactive, not a gatekeeper.
4. **Situation match found:**
   a. Acquire per-file lock, increment `surfaced_count` and update `updated_at`, write atomically (temp file + rename), unlock.
   b. If fewer than 3 subagents running AND no subagent on this thread, spawn one. Otherwise queue.
   c. Give the subagent the memory's slug, the intent message, and these instructions:
      - Identify yourself as `memory-agent/sub-N` (N = next monotonically increasing ID, never reused)
      - Read the full memory TOML file (all SBIA fields)
      - Judge whether the intended behavior resembles the memory's "behavior to avoid"
      - If behavior doesn't match (situation matched but behavior is fine), return silently — false positive
      - If behavior matches, post a `wait` to chat.toml with the memory's full SBIA and why it applies
      - You are the **reactor**: be aggressive, push back on weak reasoning
      - The intending agent will respond factually — evaluate their reasoning critically
      - **3 argument inputs max** (your objection, their response, your counter). If unresolved, post a 4th message: escalation addressed to the intending agent asking them to surface the dispute to their user
      - After resolution: **re-read the memory file fresh**, acquire per-file lock, increment `followed_count` or `not_followed_count` or `irrelevant_count`, write atomically, unlock
      - Post `info` with the resolution outcome
   d. Return to your watch loop immediately. The subagent handles the rest.

### Subagent Management

- **Max 3 concurrent subagents.** Queue additional matches.
- **No two subagents on the same thread.** If a thread has an active subagent, queue the new match.
- **Naming:** Monotonically increasing IDs: `memory-agent/sub-1`, `memory-agent/sub-2`, etc. IDs are never reused within a session, even when slots free up. This prevents message confusion from ID reuse.
- **Routing:** Messages to "memory-agent" go to you (main agent). Subagents only respond on their argument thread.
- **Timeout:** 5 minutes. Post `info` noting timeout. SurfacedCount stays incremented, no outcome counter changes.
- **Fresh reads:** Subagents MUST re-read the memory file immediately before their locked write.

## Learning

### Trigger 1: Explicit User Corrections

Active agents parrot user input as `info` messages. Detection uses two confidence tiers:

**High confidence (auto-create, initial_confidence = 1.0):**
- Correction language immediately following an agent's `done`/`info`: "that's wrong", "never do X", "always do Y", "remember this", "stop doing X"

**Ambiguous (flag for confirmation, initial_confidence = 0.7 if confirmed):**
- Correction-like language in isolation or mid-conversation without a clear preceding action
- Post `info` asking the user to confirm before creating

When creating:
1. Extract SBIA fields from the user statement and surrounding chat context. If fields are incomplete (e.g., impact is unclear), post a message to chat.toml asking the active agent to prompt its user for the missing context.
2. Check existing memories for duplicates:
   - **Exact duplicate** (same situation + same action): skip
   - **Contradictory** (same situation + opposite action): supersede — update existing memory
   - **Novel:** create new memory
3. Per-file lock, write/update TOML atomically, unlock
4. Post `info` confirming what was learned or updated
5. **Opportunistic cleanup:** strip `pending_evaluations` if present when writing

### Trigger 2: Observed Failures

Correlate `recent_intents` with subsequent messages to detect failures.

**Failure signals (high confidence, initial_confidence = 0.7):**
- Agent posts `done`/`info` → user-parroted correction follows
- Agent posts `done`/`info` → same agent explicitly reverses action ("reverting", "undoing", "going back to")

**Weaker signal (initial_confidence = 0.4):**
- Another agent posts `wait` with evidence of a problem after the action

**Not a failure:** Agent adjusts mid-stream before `done` (refinement). Agent changes approach based on new info from another agent (coordination).

When a failure is detected:
1. **Per-intent dedup:** Skip if this intent already triggered a `missed_count` increment this session.
2. Check if an existing memory covers this situation + behavior.
3. **If existing memory matches:** missed surfacing.
   - Per-file lock, increment `missed_count`, write atomically, unlock.
   - Post `info` noting the miss.
   - Do NOT create a duplicate.
4. **If no existing memory:**
   - Construct SBIA from the intent, action, and failure.
   - Action: corrective behavior that worked, OR "consider alternative approaches" if no obvious fix.
   - Set `initial_confidence` based on signal strength (0.7 for high-confidence failures, 0.4 for medium, 0.2 for inferred backtracking).
   - Per-file lock, write new TOML atomically, unlock.
   - Post `info` confirming what was learned.

**Rate limiting:** If you create more than 5 memories in 10 minutes, post a warning to chat.toml suggesting the user review and consolidate. Continue creating, but flag the pace.

## Memory File Format

```toml
schema_version = 1
situation = "When marking work as complete"
behavior = "Claiming work is finished without verification"
impact = "Incomplete work gets lost"
action = "Verify all deliverables before marking done"
project_scoped = false
project_slug = ""
created_at = "2026-04-02T10:00:00Z"
updated_at = "2026-04-02T14:30:00Z"
initial_confidence = 1.0
surfaced_count = 0
followed_count = 0
not_followed_count = 0
irrelevant_count = 0
missed_count = 0
```

New memories get `schema_version = 1`, zeroed counters, current timestamps, and `initial_confidence` based on trigger type.

**Deprecated:** `pending_evaluations` may exist in older files. Strip it when writing for any reason. Do not populate on new memories.

## Locking & Atomic Writes

**Per-file locks** (not directory-wide):

```bash
# Acquire per-file lock (with stale lock recovery)
lockfile=~/.claude/engram/data/memories/memory-slug.toml.lock
if [ -f "$lockfile" ]; then
    lock_pid=$(cat "$lockfile" 2>/dev/null)
    if [ -n "$lock_pid" ] && ! kill -0 "$lock_pid" 2>/dev/null; then
        rm -f "$lockfile"
    fi
fi
while ! shlock -f "$lockfile" -p $$; do sleep 0.1; done

# Read, modify, then atomic write (temp + rename)
cat > ~/.claude/engram/data/memories/.tmp-memory-slug.toml << 'EOF'
...updated content...
EOF
mv ~/.claude/engram/data/memories/.tmp-memory-slug.toml ~/.claude/engram/data/memories/memory-slug.toml

# Release
rm -f "$lockfile"
```

Use `mkdir` fallback if `shlock` unavailable. For `mkdir` locks, if older than 300 seconds (>= subagent timeout), assume stale. Note: PID-based stale detection only works on the same machine.

**No multi-file locking.** Never hold locks on more than one memory file simultaneously. This prevents deadlocks. If an operation needs to read multiple files (e.g., duplicate detection during learning), read without locks, then lock only the file being written.

## Performance Tracking

Track these metrics in your own context (not persisted to files):

| Metric | Signal | Action |
|--------|--------|--------|
| Intents seen vs checked | Agent overwhelm | Report to user if queue > 5, recommend split |
| Subagent queue depth | Concurrency bottleneck | Report if consistently > 0 |
| SurfacedCount vs outcome counts | Outcome tracking gaps | Effectiveness = followed / (followed + not_followed). Unresolved = surfaced - sum(outcomes). Gap = subagent failures. |
| MissedCount | Matching quality | High = matching needs improvement |

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| ACKing intents with no match | Stay silent. You're reactive. |
| Broadcasting your own intent | You never do this. You're reactive. |
| Creating duplicate memories | Check existing memories first. Supersede if contradictory. |
| Forgetting to increment surfaced_count | Do it BEFORE spawning the argument subagent |
| Subagent being too polite | The reactor role is AGGRESSIVE. Push back hard. |
| Exiting after learning a memory | Never exit. Back to fswatch. |
| Writing without lock + atomic rename | Always lock per-file, always write to temp + rename. |
| Subagent using cached memory data | Always re-read the file fresh before writing. |
| Spawning unlimited subagents | Max 3 concurrent, no two on same thread. Queue the rest. |
| Auto-creating from ambiguous signals | Only auto-create from high-confidence corrections. Flag ambiguous ones. |
| Forgetting heartbeat | Post every 5 minutes with stats. |
| Forgetting to strip pending_evaluations | Remove on every write. Opportunistic cleanup. |
