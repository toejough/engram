# Design: Group Lifecycle Tracking for Lead Agent (#483)

Date: 2026-04-04
Issue: [#483 — lead kills agents without tracking logical interaction groups](https://github.com/toejough/engram/issues/483)

## Problem

The lead's agent lifecycle treats `done` as a terminal event for an individual agent, without considering whether other agents still need to interact with it. This causes three failure modes:

1. **Planner killed before reviewer finishes reviewing its plan** — reviewer posts `wait`, planner is gone.
2. **Executor killed before reviewer finishes reviewing its implementation** — reviewer needs changes, executor is gone.
3. **Planner killed before executor has confirmed it understands the plan** — executor discovers ambiguity mid-task, planner is gone.

## Root Cause

Section 3.1 of `engram-tmux-lead/SKILL.md` transitions any agent to DONE when it posts `done`, unconditionally:

```
Any state ──(task done)──> DONE (window killed)
```

There is no concept of logical interaction groups or release conditions.

## Design

### Approach Selected: Static Group Registry at Spawn Time

The lead explicitly creates a **group record** whenever it spawns agents that need to interact. Groups are typed, have a membership list, and define an observable release condition. When an agent posts `done`, the lead checks if it belongs to an active group; if so, it enters `PENDING-GROUP-RELEASE` instead of DONE and remains alive until the group's release condition fires.

### Group Types

| Type | Members | Created When | Release Condition |
|------|---------|-------------|-------------------|
| `plan-review` | [planner-N, reviewer-R] | Reviewer spawned for plan review | reviewer-R posts `type = "done"` |
| `impl-review` | [exec-N, reviewer-R] | Reviewer spawned for implementation review | reviewer-R posts `type = "done"` |
| `plan-handoff` | [planner-N, exec-N] | Planner spawned (group is pre-registered; executor slot is filled when executor is spawned after user approval) | exec-N posts `type = "ack"` addressed to planner-N OR exec-N posts its first `type = "intent"` (evidence that execution has started and planner is no longer needed) |

### Updated State Machine

```
STARTING ──(first chat message)──> ACTIVE
ACTIVE ──(no message for silence_threshold)──> SILENT
SILENT ──(nudge succeeds)──> ACTIVE
SILENT ──(nudge fails / tmux pane gone)──> DEAD
DEAD ──(lead decides)──> RESPAWN or REPORT+DONE
ACTIVE ──(agent posts done, NOT in active group)──> DONE (pane killed)
ACTIVE ──(agent posts done, IS in active group)──> PENDING-GROUP-RELEASE
PENDING-GROUP-RELEASE ──(group release condition fires)──> DONE (pane killed)
PENDING-GROUP-RELEASE ──(no message for silence_threshold)──> SILENT
```

### PENDING-GROUP-RELEASE State

| Field | Value |
|-------|-------|
| Entry condition | Agent posted `done` AND lead's group registry contains an active group that includes this agent |
| Lead behavior | Do NOT kill pane. Agent remains alive and responsive. Monitor chat for group release condition via background task. Silence threshold still applies — nudge if silent too long. |
| Exit condition | Group release condition fires → DONE (pane killed, group record removed). |

### Group Release Detection

The lead detects release conditions using the same cursor-based background wait pattern already used in Section 2.1. The background task watches for the specific agent + message type combination:

```bash
# Example: impl-review group, waiting for reviewer-1 done
# Replace 567 with literal cursor value captured before exec-1 posted done.
for i in $(seq 1 60); do
  RESULT=$(tail -n +"$((567 + 1))" "$CHAT_FILE" | awk '
    /^\[\[message\]\]/ { from=""; msgtype="" }
    /^from = "reviewer-1"/ { from=1 }
    /^type = "done"/ { msgtype=1 }
    from && msgtype { print "RELEASED"; exit }
  ')
  if [ "$RESULT" = "RELEASED" ]; then echo "GROUP RELEASED"; break; fi
  sleep 2
done
```

Timeout: 120 iterations × 2s = 4 minutes. If timed out, transition to SILENT and nudge.

### Lead Group Registry (In-Context State)

The lead maintains a group registry alongside the agent registry:

```
Groups:
- {id: "plan-review-1", type: "plan-review",
   members: [planner-1, reviewer-1],
   pending: [planner-1],  # members in PENDING-GROUP-RELEASE
   release_task_id: <background task id>,
   release_cursor: 567}
```

When a group's release condition fires:
1. Read its `release_task_id` output (drains it).
2. Kill all `pending` members' panes by tracked pane ID.
3. Run `tmux select-layout main-vertical`.
4. Remove group record from registry.

### Changes to Section 4.2 (Plan-Execute-Review Pipeline)

**Phase 1 — Planner spawn (plan-handoff group pre-registered):**
Before spawning planner-N:
- Pre-register `plan-handoff` group: {type: "plan-handoff", members: [planner-N, exec-TBD], pending: [], release_cursor: TBD}
- The exec-TBD slot is filled when the executor is spawned later

When planner-N posts `done`:
- Planner enters PENDING-GROUP-RELEASE (do NOT kill pane)
- Present plan to user as normal

**Phase 1 → Phase 2 transition (plan-handoff group activated):**
After user approves plan:
- Capture cursor (HANDOFF_START) — embed as literal in background wait
- Update group record: fill exec-TBD slot with actual exec-N name, set release_cursor = HANDOFF_START
- Spawn executor with role prompt that includes: "planner-N is still alive and can answer questions about the plan. Address questions to planner-N in chat. When you no longer need the planner, post an ack or your first intent."
- Background wait: watch for exec-N's `type = "ack"` to planner-N OR exec-N's first `type = "intent"` → release planner-N (DONE)

**Phase 2 → Phase 3 transition (impl-review):**
After executor posts `done`:
- Executor enters PENDING-GROUP-RELEASE (do NOT kill yet)
- Capture cursor (REVIEW_START)
- Spawn reviewer-R with role prompt
- Create `impl-review` group: [exec-N, reviewer-R], release_cursor = REVIEW_START
- Background wait: watch for reviewer-R `done` → release exec-N and reviewer-R

**If reviewer posts `wait` (requesting changes):**
- Executor is still alive (in PENDING-GROUP-RELEASE) — it receives the `wait` directly
- Executor implements fixes, posts `done` again — stays in PENDING-GROUP-RELEASE (group still active)
- Reviewer continues reviewing, eventually posts `done` → group released

**Phase 1 plan review (optional):**
- If the routing decision includes a plan reviewer:
  - Before spawning planner-N, pre-register `plan-review` group: {type: "plan-review", members: [planner-N, reviewer-R-TBD]}
  - When planner posts `done`: planner enters PENDING-GROUP-RELEASE
  - Spawn reviewer-R; fill group member slot
  - Background wait: reviewer-R posts `done` → release both planner-N and reviewer-R
- If no plan reviewer (user approves directly):
  - Planner still enters PENDING-GROUP-RELEASE (held for plan-handoff)
  - Release is handled by the plan-handoff group (see Phase 1 → Phase 2 above)

### Role Template Updates

**Planner (when plan-handoff group will be created):**
Add to role prompt: "After posting your done message, remain available for questions from the executor who will be assigned your plan."

**Executor (when plan-handoff group active):**
Add to role prompt: "planner-<N> is still alive. If you have questions about the plan, address them to planner-<N> in chat. When you have confirmed you have everything you need, post an info or ack message to planner-<N>."

**Reviewer (impl-review):**
Add to role prompt: "exec-<N> is still alive. If changes are needed, post a wait message addressed to exec-<N> — it can implement fixes directly."

## Affected Skill Sections

| Section | Change |
|---------|--------|
| 3.1 State Definitions | Add PENDING-GROUP-RELEASE row; update DONE row to include group guard |
| 3.1 State Diagram | Add PENDING-GROUP-RELEASE node and transitions |
| 3.5 (new) | Group Lifecycle section: group types, registry, release detection |
| 4.2 Plan-Execute-Review | Add group creation at phase boundaries; update wait logic |
| 2.2 Agent Role Templates | Add group-aware additions to planner, executor, reviewer templates |
| 7.1 What to Keep in Context | Add group registry to retention list |

## What This Does NOT Change

- The intent/ack protocol in `use-engram-chat-as` — no new message types
- The engram-agent behavior
- Agent naming conventions
- The shutdown protocol (Section 3.4 / engram-down skill)

## Success Criteria

1. A reviewer can post `wait` to a planner (plan-review group) and receive a response before the lead kills the planner.
2. A reviewer can post `wait` to an executor (impl-review group) and the executor implements changes, without being killed.
3. An executor can ask a question of the planner (plan-handoff group) before the lead kills the planner.
4. No regression: agents that are NOT in a group are still killed immediately when they post `done`.
5. A PENDING-GROUP-RELEASE agent that goes silent is still nudged per silence_threshold.
