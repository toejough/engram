# Design: Hold-Based Group Lifecycle for Lead Agent (#483)

Date: 2026-04-04
Issue: [#483 — lead kills agents without tracking logical interaction groups](https://github.com/toejough/engram/issues/483)
Supersedes: `docs/superpowers/specs/2026-04-04-483-group-lifecycle-design.md` (rejected as too narrow)

## Problem

The lead's agent lifecycle treats `done` as a terminal event for an individual agent, without considering whether other agents still need to interact with it. The previous design attempted to solve this with a fixed enum of three group types (`plan-review`, `impl-review`, `plan-handoff`), but this only covered the plan-execute-review pipeline. The user needs a general model that handles ALL coordination patterns without requiring a new type for each new workflow.

## Root Cause

Section 3.1 of `engram-tmux-lead/SKILL.md` transitions any agent to DONE when it posts `done`, unconditionally:

```
Any state ──(task done)──> DONE (window killed)
```

There is no mechanism to express "Agent X still needs Agent Y alive."

## Design Principle: Holds, Not Groups

The previous design modeled **groups** — sets of agents with typed release conditions. This required a new type for each coordination pattern.

The redesign models **holds** — directed keep-alive relationships between individual agents. A hold says: "Agent A is holding Agent B alive because A might still need B." An agent stays alive as long as any hold targets it.

A "group" is just the set of holds created for a particular workflow — an organizational label (tag), not a structural entity. New coordination patterns = new hold configurations, not new mechanism.

## The Hold Primitive

### Definition

```
Hold {
  id:      string        # unique identifier (e.g., "h1")
  holder:  string        # agent keeping target alive
  target:  string        # agent being kept alive
  release: Condition     # when this hold dissolves
  tag:     string        # workflow label for bookkeeping (e.g., "plan-review-1")
}
```

A hold is **directed**: holder → target. The holder has a need; the target is held alive to satisfy it.

### Release Conditions

Release conditions are observable chat events. The lead monitors them via cursor-based background tasks (same pattern as existing Section 2.1).

| Condition | Syntax | Fires When |
|-----------|--------|------------|
| Agent done | `done(agent)` | `agent` posts `type = "done"` |
| Agent signal | `signal(agent, msg_type)` | `agent` posts `type = msg_type` |
| Targeted signal | `signal(agent, msg_type, to_agent)` | `agent` posts `type = msg_type` addressed to `to_agent` |
| First intent | `first_intent(agent)` | `agent` posts its first `type = "intent"` (evidence of independent work) |
| Lead release | `lead_release(tag)` | Lead explicitly dissolves all holds with this tag |

`lead_release` is the escape hatch for coordinator-controlled patterns (merge queue, co-design completion). It is the lead's manual intervention, not an observed chat event. The lead posts an `info` message to chat documenting the release for observability, then removes the holds from its registry.

### Hold Lifecycle

```
CREATED ──(release condition fires)──> DISSOLVED
```

When a hold is dissolved:
1. Remove from the hold registry
2. Check if target has remaining incoming holds
3. If no remaining holds → target transitions to DONE (pane killed)
4. If remaining holds → target stays in PENDING-RELEASE

### Agent State Machine (Updated)

```
STARTING ──(first chat message)──> ACTIVE
ACTIVE ──(no message for silence_threshold)──> SILENT
ACTIVE ──(agent posts done, NO incoming holds)──> DONE (pane killed)
ACTIVE ──(agent posts done, HAS incoming holds)──> PENDING-RELEASE
PENDING-RELEASE ──(last incoming hold dissolved)──> DONE (pane killed)
PENDING-RELEASE ──(agent posts done again, HAS incoming holds)──> PENDING-RELEASE (no-op)
PENDING-RELEASE ──(no message for silence_threshold)──> SILENT
SILENT ──(nudge succeeds, has incoming holds)──> PENDING-RELEASE
SILENT ──(nudge succeeds, no incoming holds)──> ACTIVE
SILENT ──(nudge fails / tmux pane gone)──> DEAD
DEAD ──(lead decides)──> RESPAWN or REPORT+DONE
```

Total: 12 transitions (up from 6).

### PENDING-RELEASE State

| Field | Value |
|-------|-------|
| Entry condition | Agent posted `done` AND lead's hold registry contains at least one hold targeting this agent |
| Lead behavior | Do NOT kill pane. Agent remains alive and responsive. Monitor holds via background tasks. Silence threshold still applies — nudge if silent too long (use PENDING-RELEASE-specific nudge text: "You are held in PENDING-RELEASE and may receive further instructions. If idle, post a brief heartbeat."). |
| Exit condition | All incoming holds dissolved → DONE (pane killed) |

### Hold Registry (Lead In-Context State)

The lead maintains a hold registry alongside the agent registry:

```
Holds:
- {id: "h1", holder: "reviewer-1", target: "planner-1", release: lead_release("plan-review-1"), tag: "plan-review-1"}
- {id: "h2", holder: "exec-1", target: "planner-1", release: first_intent("exec-1"), tag: "plan-handoff-1"}
```

When creating holds, the lead assigns a tag that groups related holds for a workflow. Tags are for human readability and `lead_release` targeting — the mechanism operates on individual holds.

### Hold Detection (Background Tasks)

One background task per hold. Uses the existing cursor-based polling pattern (Section 6.4):

```bash
# Example: hold h1, watching for reviewer-1 to post done
# Replace 567 with literal cursor value captured when hold was created.
# This is a PERSISTENT watcher — restarts on timeout until the release fires.
HOLD_CURSOR=567
while true; do
  for i in $(seq 1 60); do
    RESULT=$(tail -n +"$((HOLD_CURSOR + 1))" "$CHAT_FILE" | awk '
      /^\[\[message\]\]/ { from=""; msgtype="" }
      /^from = "reviewer-1"/ { from=1 }
      /^type = "done"/ { msgtype=1 }
      from && msgtype { print "RELEASED"; exit }
    ')
    if [ "$RESULT" = "RELEASED" ]; then echo "HOLD RELEASED h1"; exit 0; fi
    sleep 2
  done
  # Timeout — advance cursor and restart. Hold is still active.
  HOLD_CURSOR=$(wc -l < "$CHAT_FILE")
done
```

The watcher is persistent: it restarts after each 2-minute timeout window, advancing the cursor. The hold stays in the registry until the release event fires. This prevents stuck holds where the holder's event arrives after the watcher's one-shot timeout.

**Background task hygiene:** Each hold watcher is tracked by task ID in the hold registry. Drain before replacing (per Section 6.4). When a hold dissolves, drain its task. When a `lead_release(tag)` fires, drain all tasks for holds with that tag.

**Hold watchers replace standard agent wait tasks.** In hold-aware phases (e.g., Phase 3 of the pipeline), the hold watcher watches for the same event as the standard agent wait task (Section 2.1 Step 3). Do NOT run both. The lead uses the hold watcher's output as the signal to both dissolve the hold AND advance the phase.

## Documented Patterns

These are RECIPES — advisory documentation for common workflows. They are NOT enforced types. The lead references them when deciding what holds to create at spawn time. New patterns are added as new documentation sections without any mechanism change.

### Pattern 1: Pair (Review)

**When:** Spawning a reviewer for an agent's work (plan review, impl review, code review).

**Holds:**
```
{holder: reviewer, target: subject, release: done(reviewer), tag: "<review-type>-N"}
```

**Behavior:** Subject enters PENDING-RELEASE when it posts done. Reviewer can post `wait` and the subject responds. When reviewer posts `done`, hold dissolves, subject is released.

**Applied to plan-review:** `{holder: reviewer-R, target: planner-N, release: lead_release("plan-review-N"), tag: "plan-review-1"}`
Note: uses `lead_release` (not `done(reviewer-R)`) to prevent a lifecycle gap. Planner-N must stay alive through the user-approval window (Phase 1b end → Phase 2 exec spawn). In Phase 2, the lead creates the plan-handoff hold first, then calls `lead_release("plan-review-N")` — ensuring planner-N is never momentarily holdless.

**Applied to impl-review:** `{holder: reviewer-R, target: exec-N, release: done(reviewer-R), tag: "impl-review-1"}`

### Pattern 2: Handoff

**When:** One agent passes work to another and should stay available for questions.

**Holds:**
```
{holder: receiver, target: sender, release: first_intent(receiver), tag: "handoff-N"}
```

Alternative release: `signal(receiver, "ack", sender)` if you want an explicit handshake rather than inferring from first intent.

**Behavior:** Sender enters PENDING-RELEASE when it posts done. Receiver can ask questions of sender via chat. When receiver begins independent work (first intent) or explicitly acks, hold dissolves, sender is released.

**Applied to plan-handoff:** `{holder: exec-N, target: planner-N, release: first_intent(exec-N), tag: "plan-handoff-1"}`

### Pattern 3: Fan-In (Research Synthesis)

**When:** Multiple producers report findings to a single consumer who synthesizes them.

**Holds:** One hold per producer:
```
For each researcher-K:
  {holder: synthesizer, target: researcher-K, release: done(synthesizer), tag: "synthesis-N"}
```

**Behavior:** Each researcher enters PENDING-RELEASE when it posts done. Synthesizer reads all findings, may ask follow-up questions to any researcher. When synthesizer posts done, ALL holds dissolve simultaneously, all researchers are released.

**Spawning sequence:**
1. Spawn researchers in parallel (each in their own worktree if needed)
2. Spawn synthesizer with role: "Wait for all researchers to post done. Read their findings. Ask follow-up questions if needed. Synthesize and post done."
3. Create holds: synthesizer → each researcher
4. Lead's wait task watches for synthesizer's done (not researchers' done)

### Pattern 4: Merge Queue (Sequential Coordination)

**When:** Multiple worktree executors complete independently and need sequential, lead-coordinated merging.

**Holds:** One hold per executor:
```
For each exec-K:
  {holder: "merge-process", target: exec-K, release: lead_release("merge-N-exec-K"), tag: "merge-queue-N"}
```

Note: `holder` is `"merge-process"` — a virtual holder representing the lead's merge coordination, not an actual agent. This is valid because holds are data in the lead's registry; the holder field identifies who has the need, and the lead itself can have needs.

**Behavior:**
1. All executors work in parallel in worktrees
2. Each enters PENDING-RELEASE when it posts done
3. Lead merges branches ONE AT A TIME in order (Section 4.3 updated):
   a. Rebase exec-1's branch onto main: lead tells exec-1 to rebase and re-test
   b. If rebase conflicts → exec-1 resolves (it's alive in PENDING-RELEASE)
   c. After rebase + green tests → `git merge --ff-only exec-1-branch`
   d. `lead_release("merge-N-exec-1")` → exec-1 released
   e. Rebase exec-2's branch onto updated main, repeat from (b)

**Why this beats the race:** No executor independently rebases or retests. The lead controls the merge order. Each executor only resolves its OWN conflicts, and only when the lead asks. Rebasing onto updated main happens once per executor, in order, not in a retry loop.

### Pattern 5: Barrier (Co-Design)

**When:** Multiple agents collaborate equally on a shared artifact. All need each other alive for back-and-forth.

**Holds:** Bidirectional holds between all pairs, all released by coordinator signal:
```
For each pair (A, B) where A != B:
  {holder: A, target: B, release: lead_release("codesign-N"), tag: "codesign-N"}
  {holder: B, target: A, release: lead_release("codesign-N"), tag: "codesign-N"}
```

For 3 agents: 6 holds (3 pairs x 2 directions). For N agents: N*(N-1) holds.

**Behavior:**
1. All planners work simultaneously, posting to a shared thread
2. Each reads others' contributions and responds
3. When the lead determines the design is complete (consensus reached, or user approves), it issues `lead_release("codesign-N")` → all holds dissolve → all planners released

**Alternative (simpler):** Instead of N*(N-1) bidirectional holds, use a single virtual holder:
```
For each member-K:
  {holder: "codesign-coordinator", target: member-K, release: lead_release("codesign-N"), tag: "codesign-N"}
```
This is equivalent when ALL releases fire at the same time (which they do for `lead_release`). The bidirectional model is only needed if individual releases are possible (which they aren't in co-design — the group ends atomically).

**Recommended: Use the simpler virtual-holder variant.** N holds instead of N*(N-1).

### Pattern 6: Expert Consultation

**When:** An executor needs an answer from a specialist. Lead spawns a researcher to answer.

**Holds:**
```
{holder: exec-N, target: researcher-K, release: done(exec-N), tag: "consult-N"}
```

Note: the hold direction is exec → researcher (exec might need more answers). Researcher is held alive until exec finishes (not just until researcher posts its answer).

Alternative: `{release: done(researcher-K)}` if the consultation is one-shot and exec doesn't need follow-ups.

## PENDING-RELEASE Agent Responsiveness Prerequisite

For PENDING-RELEASE to work, held agents MUST continue watching chat after posting `done`. Without this, held agents are deaf to follow-up instructions (reviewer WAITs, merge rebase requests, follow-up questions).

**ALL role templates** in Section 2.2 must include this line for any agent that might be held:

> "After posting done, continue watching chat for further instructions. You may receive follow-up questions or requests while held in PENDING-RELEASE."

This applies to: planner, executor, reviewer, researcher, synthesizer, co-designer — every role template. Even roles not currently expected to be held should include it, since new hold patterns may target any agent.

## Changes to Lead Skill Sections

| Section | Change |
|---------|--------|
| 3.1 State Definitions | Add PENDING-RELEASE row. Update DONE row to include hold guard. Update state diagram to 12 transitions. |
| New 3.5 | "Hold-Based Agent Retention" — hold primitive, release conditions, hold registry, detection pattern, documented patterns |
| 4.2 Plan-Execute-Review | Add hold creation at each phase boundary. Plan-review hold (Phase 1b, uses `lead_release` not `done(reviewer-R)` — see plan-review pattern note), plan-handoff hold (Phase 2 at executor spawn), impl-review hold (Phase 3). Phase 2 calls `lead_release` for plan-review hold after creating plan-handoff hold — atomic handoff, planner never holdless. |
| 4.3 Parallel Executor Isolation | Replace "merge each one at a time" with merge queue pattern using holds. Lead-controlled sequential merge with conflict resolution. |
| New 4.5 | "Research Synthesis" routing pattern — when to use, spawning, holds |
| New 4.6 | "Co-Design" routing pattern — when to use, spawning, holds |
| 2.2 Agent Role Templates | Add: synthesizer, co-designer, plan-reviewer. Update ALL templates with PENDING-RELEASE "keep watching" line. Update: planner (mention PENDING-RELEASE availability), executor (mention planner availability during handoff), reviewer (mention subject availability for changes) |
| 3.2 Nudging | Add conditional nudge text for PENDING-RELEASE agents |
| 7.1 What to Keep in Context | Add hold registry to retention list |
| 6.4 Background Task Hygiene | Add hold watcher drain rules |

## What This Does NOT Change

- The chat protocol (`use-engram-chat-as`) — no new message types
- The intent/ack/wait protocol
- The engram-agent behavior
- Agent naming conventions
- The shutdown protocol (Section 3.4 / engram-down skill)
- Concurrency limit (Section 2.4)

## Success Criteria

1. A reviewer can post `wait` to a planner (pair pattern) and receive a response — planner is in PENDING-RELEASE, not killed.
2. A reviewer can post `wait` to an executor (pair pattern) and the executor implements changes.
3. An executor can ask a planner questions during handoff — planner stays alive until executor's first intent.
4. A synthesizer can ask follow-up questions to any researcher — all researchers stay in PENDING-RELEASE until synthesizer posts done.
5. Parallel worktree executors are merged sequentially by the lead — no racing, no independent rebase loops. Each executor stays alive until its merge completes.
6. Co-design planners all stay alive until the lead signals design completion.
7. No regression: agents with NO incoming holds are still killed immediately when they post done.
8. A PENDING-RELEASE agent that goes silent is still nudged per silence_threshold.
9. Adding a new coordination pattern (e.g., "competing approaches") requires ONLY a new documentation section in Section 3.5, not a mechanism change.
