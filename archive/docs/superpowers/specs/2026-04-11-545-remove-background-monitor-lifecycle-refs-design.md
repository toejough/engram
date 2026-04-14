# Design: Remove Stale Background Monitor References from use-engram-chat-as Agent Lifecycle

**Issue:** #545
**Date:** 2026-04-11
**Status:** Approved

## Problem

Commit `83dbe68` removed the Background Monitor Pattern section from `use-engram-chat-as/SKILL.md`
because it caused non-lead agents to incorrectly spawn persistent internal monitor subagents (#539).
Non-lead agents are stateless — they process one task and exit; the lead or binary handles
re-invocation.

However, the Agent Lifecycle steps and several other sections still reference the removed pattern.
Agents following the current lifecycle would attempt to:

1. "Spawn background monitor Agent (Background Monitor Pattern, above)" — step 7 points to a
   section that no longer exists
2. "Wait for monitor Agent notification" — step 9 blocks on a monitor that was never spawned
3. "Go to step 9 -- ALWAYS" — step 13 loops back to the broken monitor wait indefinitely

This is a broken lifecycle contract.

## Scope of Changes

All locations in `skills/use-engram-chat-as/SKILL.md` that reference the removed monitor pattern:

| Location | Lines | Issue |
|----------|-------|-------|
| Agent Lifecycle step 3 comment | 489-490 | "...so the monitor captures..." |
| Agent Lifecycle step 7 | 496 | "Spawn background monitor Agent (Background Monitor Pattern, above)..." |
| Agent Lifecycle step 8 | 497 | Info message says "Monitor active." |
| Agent Lifecycle steps 9-10 | 498-499 | "Wait for monitor Agent notification" / "Monitor Agent returns..." |
| Agent Lifecycle step 13 | 511 | "Go to step 9 -- ALWAYS." |
| "The watch only ends when" section | 514-516 | Entire section describes non-existent loop |
| Ready Messages (3 places) | 423, 428, 430 | "spawning the monitor" |
| Shutdown Protocol step 4 | 465 | "Exit the monitor Agent loop." |
| Compaction Recovery Step 6 | 609-611 | "Re-enter the fswatch loop" / "Continue from step 9" |
| Compaction Recovery guard note | 536, 615 | "watch loop" / "waiting for fswatch" |
| Common Mistakes (5 rows) | 631-634, 642 | fswatch, background monitor Agent, stale loop behavior |

## Design

### Agent Lifecycle Rewrite (steps 7-13 → 7-11)

Replace the monitor-based persistent loop with a stateless task-and-exit model.

**Before:**
```
3. Initialize cursor: CURSOR=$(engram chat cursor) — BEFORE posting ready so the monitor
   captures any work routed by lead between your ready message and monitor startup
...
7. Spawn background monitor Agent (Background Monitor Pattern, above) using CURSOR from step 3
8. Post info: "Initialization complete. Monitor active." — signals lead that agent is operational
9. Wait for monitor Agent notification
10. Monitor Agent returns semantic event -> process event if addressed to you
11. If acting: [intent protocol]
12. Post response (with lock)
13. Go to step 9 -- ALWAYS. Even after completing a task.
```

**After:**
```
3. Initialize cursor: CURSOR=$(engram chat cursor) — BEFORE posting ready
...
7. Post info: "Initialization complete." — signals lead that agent is operational
8. Process assigned work (delivered at invocation time by lead or user)
9. If acting: [intent protocol — unchanged from old step 11]
10. Post result (info or done)
11. Exit — lead or binary handles re-invocation for subsequent tasks
```

Remove the "The watch only ends when:" block (only valid for a persistent watch loop).

### Ready Messages Section

Three references to "spawning the monitor" should be updated to reflect the stateless init sequence:
- Line 423: "...or spawning the monitor" → "...or doing other initialization"
- Line 428: "...before its monitor is watching" → "...before it is operational"
- Line 430: "...before spawning the monitor and posting the init-complete info" → "...before posting the init-complete info"

### Shutdown Protocol

Line 465: "Exit the monitor Agent loop. Do not spawn a new monitor Agent." →
"Complete in-flight work and exit. There is no persistent loop to exit."

### Compaction Recovery

- Step 6 heading: "Re-enter the fswatch loop." → "Resume task processing."
- Step 6 body: "Continue the lifecycle from step 9 of the Agent Lifecycle." →
  "Continue the lifecycle from step 8 of the Agent Lifecycle (process assigned work)."
- Guard note (line 536): "in your watch loop" → "before cursor-dependent operations"
- Guard note (line 615): "waiting for fswatch" → "processing a task"

### Common Mistakes Table

Remove or update 5 rows that reference the monitor/fswatch pattern:

| Old row | Disposition |
|---------|-------------|
| "Poll with `sleep 2` loop \| Use `fswatch -1` / `inotifywait`..." | Remove — non-lead agents don't block on fswatch |
| "Run fswatch/wc/grep directly in main agent context \| Use background monitor Agent..." | Remove — background monitor pattern is gone |
| "Post a message then stop \| Always re-enter the fswatch after posting" | Replace: "Exit before posting `done` \| Always post `done` when your assigned task is complete before exiting" |
| "Stop after task completion \| Completing a task != dismissed. Watch for next assignment" | Remove — stateless agents DO exit after task completion |
| "Ignore `shutdown` message \| Exit monitor Agent loop after completing in-flight work..." | Replace: "Ignore `shutdown` message \| Post `done` and exit when you receive a `shutdown` message addressed to you or `all`" |

## Testing

### Phase 1 — Baseline behavior test (RED)

Write a test script (`skills/use-engram-chat-as/test-lifecycle-refs.sh`) that:
1. Greps for `"Background Monitor Pattern"` in `SKILL.md` — expects 0 matches, currently finds 1 → RED
2. Greps for `"monitor Agent"` in the Agent Lifecycle section — expects 0 matches, currently finds several → RED
3. Greps for `"Go to step 9"` in `SKILL.md` — expects 0 matches, currently finds 1 → RED
4. Greps for `"Monitor active"` in `SKILL.md` — expects 0 matches, currently finds 1 → RED
5. Greps for `"fswatch loop"` in `SKILL.md` — expects 0 matches, currently finds several → RED

Run the test: all 5 checks should fail (RED).

### Phase 2 — Update skill (GREEN)

Apply all changes described in the Design section above. Run the test script: all 5 checks should pass (GREEN).

### Phase 3 — Verify and refine (pressure test)

Read the updated Agent Lifecycle section end-to-end and verify:
1. A non-lead agent following the lifecycle would initialize, post init-complete, process one task, and exit — no monitor spawn, no infinite loop
2. Step numbers in cross-references (Compaction Recovery, Common Mistakes) are consistent with the renumbered lifecycle
3. No new broken references were introduced
4. The skill as a whole reads coherently — no "phantom" sections referenced from Common Mistakes

## Implementation Notes

- Use `superpowers:writing-skills` skill for all edits — it enforces the TDD cycle
- The intent/ack-wait protocol in the middle of the lifecycle (old step 11, new step 9) is unchanged — only the monitor-related wrapper steps change
- Step 11f "Pre-done cursor-check: spawn background Agent..." is NOT a persistent monitor — it is a one-shot check and should be kept as-is
- Do NOT change the Cursor Tracking section — it was the Phase 5 replacement for the Background Monitor section and is correct
- After GREEN, run `targ check-full` to confirm no other issues
