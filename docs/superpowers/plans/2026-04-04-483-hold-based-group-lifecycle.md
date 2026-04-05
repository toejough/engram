# Hold-Based Group Lifecycle (#483) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**CRITICAL: This is a skill file edit.** Per CLAUDE.md: **ALWAYS use `superpowers:writing-skills` when editing any SKILL.md file. No exceptions.** Invoke it first — it requires baseline behavioral tests (RED), skill update (GREEN), behavioral verification.

**Goal:** Replace the lead's unconditional `done → kill` transition with a hold-based retention model that handles all coordination patterns (review pairs, handoffs, research synthesis, merge queues, co-design) without requiring new mechanism for new patterns.

**Architecture:** Add a `PENDING-RELEASE` state to the agent lifecycle. When an agent posts `done`, the lead checks its hold registry; if any hold targets that agent, it enters PENDING-RELEASE instead of DONE. Holds are directed keep-alive edges between agents, dissolved by observable release conditions. Patterns (pair, fan-in, barrier, etc.) are documented recipes, not enforced types.

**Tech Stack:** Skill file only (`skills/engram-tmux-lead/SKILL.md`) — Markdown text edits. No Go code changes. Uses existing chat cursor pattern and background task machinery.

**Spec:** `docs/superpowers/specs/2026-04-04-483-hold-based-group-lifecycle-design.md`

---

## Files

- Modify: `skills/engram-tmux-lead/SKILL.md`
  - Section 2.2 (Agent Role Templates) — lines 231–264
  - Section 3 (Agent Lifecycle State Machine) — lines 281–302
  - Section 3.1 (State Definitions) — lines 294–302
  - New Section 3.5 (Hold-Based Agent Retention) — insert after Section 3.4 (~line 354)
  - Section 4.1 (Routing Decision Table) — lines 358–372
  - Section 4.2 (Plan-Execute-Review Pipeline) — lines 374–418
  - Section 4.3 (Parallel Executor Isolation) — lines 420–435
  - New Section 4.5 (Research Synthesis) — insert after Section 4.4
  - New Section 4.6 (Co-Design) — insert after new 4.5
  - Section 6.4 (Background Task Hygiene) — lines 517–555
  - Section 7.1 (What to Keep in Context) — lines 559–566

---

## Task 1: Invoke writing-skills skill and establish baseline behavioral tests

**Files:**
- Read: `skills/engram-tmux-lead/SKILL.md`

- [ ] **Step 1: Invoke writing-skills skill**

This is a skill file edit. Invoke `superpowers:writing-skills` to establish baseline behavioral tests before making any changes. The writing-skills skill will guide the TDD process for skill modifications. Follow its instructions to:
1. Define behavioral test cases that exercise the CURRENT skill behavior
2. Run them to establish the RED baseline (they should pass against current skill)
3. Return here for the actual edits

The key behaviors to test against the CURRENT skill:
- When an agent posts `done`, lead kills its pane (unconditional `DONE` transition)
- State diagram has 6 transitions (will become 12)
- State table has 5 states (STARTING, ACTIVE, SILENT, DEAD, DONE) (will gain PENDING-RELEASE)
- Section 4.3 merge strategy says "merge each one at a time" with no hold awareness
- No Section 3.5 exists
- No Section 4.5 or 4.6 exists

- [ ] **Step 2: Commit baseline tests**

```bash
git add -A
git commit -m "test(lead-skill): baseline behavioral tests for #483 hold-based lifecycle

AI-Used: [claude]"
```

---

## Task 2: Update state diagram and state definitions (Section 3)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` — lines 281–302

- [ ] **Step 1: Replace the state diagram**

Replace the state diagram at lines 285–292 with:

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

12 transitions (up from 6).

- [ ] **Step 2: Update the state table**

Replace the DONE row in the state table (line 302) with two rows:

| **PENDING-RELEASE** | Agent posted `done` AND lead's hold registry contains at least one hold targeting this agent | Do NOT kill pane. Agent remains alive and responsive. Monitor holds via background tasks. Silence threshold still applies — use PENDING-RELEASE-specific nudge text (see Step 2b). |
| **DONE** | Agent posted `done` AND no incoming holds remain (or last hold just dissolved) | Kill pane by tracked ID: `tmux kill-pane -t <pane-id>`. Rebalance: `tmux select-layout main-vertical`. Remove from tracking. |

- [ ] **Step 2b: Update Section 3.2 nudge text for PENDING-RELEASE agents**

In Section 3.2 (Nudging), add a conditional to Step 1 (Chat nudge):

```markdown
**If agent is in PENDING-RELEASE**, use this nudge text instead:

```toml
[[message]]
from = "lead"
to = "<agent-name>"
thread = "nudge"
type = "info"
ts = "<now>"
text = "You are held in PENDING-RELEASE and may receive further instructions. If idle, post a brief heartbeat."
```
```

This prevents PENDING-RELEASE agents from interpreting "post a status update" as needing to post `done` again.

- [ ] **Step 3: Run behavioral tests**

Run the writing-skills behavioral tests to verify the state diagram changes are reflected. Expected: tests that checked for 6 transitions now fail (RED), confirming the change was detected.

- [ ] **Step 4: Update behavioral tests for new state machine**

Update tests to expect 12 transitions and the PENDING-RELEASE state (including the PENDING-RELEASE → PENDING-RELEASE no-op on repeated done). Run again — should PASS (GREEN).

- [ ] **Step 5: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "feat(lead-skill): add PENDING-RELEASE state to lifecycle (#483)

State diagram updated from 6 to 12 transitions. PENDING-RELEASE
entered when agent posts done but has incoming holds. Includes
no-op self-transition on repeated done and PENDING-RELEASE-specific
nudge text.

AI-Used: [claude]"
```

---

## Task 3: Add Section 3.5 — Hold-Based Agent Retention

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` — insert after Section 3.4 (after ~line 354)

This is the core of the design. Insert the entire hold mechanism as a new section.

- [ ] **Step 1: Insert Section 3.5 header and hold primitive definition**

Insert after Section 3.4 (Shutdown), before Section 4 (Routing):

```markdown
### 3.5 Hold-Based Agent Retention

A **hold** is a directed keep-alive relationship: "Agent A is holding Agent B alive because A might still need B." An agent stays in PENDING-RELEASE as long as any hold targets it.

Holds are the ONLY retention primitive. Coordination patterns (review, handoff, synthesis, merge, co-design) are documented recipes that configure holds — not enforced types. New patterns require only a new recipe section, not a mechanism change.

#### Hold Definition

```
Hold {
  id:      string        # unique identifier (e.g., "h1")
  holder:  string        # agent (or virtual process) keeping target alive
  target:  string        # agent being kept alive
  release: Condition     # when this hold dissolves
  tag:     string        # workflow label for bookkeeping (e.g., "plan-review-1")
}
```

#### Release Conditions

| Condition | Syntax | Fires When |
|-----------|--------|------------|
| Agent done | `done(agent)` | `agent` posts `type = "done"` |
| Agent signal | `signal(agent, msg_type)` | `agent` posts `type = msg_type` |
| Targeted signal | `signal(agent, msg_type, to)` | `agent` posts `type = msg_type` addressed to `to` |
| First intent | `first_intent(agent)` | `agent` posts its first `type = "intent"` |
| Lead release | `lead_release(tag)` | Lead explicitly dissolves all holds with this tag |

`lead_release` is the lead's manual intervention for coordinator-controlled patterns (merge queue, co-design). The lead posts an `info` message to chat documenting the release, then removes the holds from its registry.

#### Hold Registry (In-Context State)

Maintain alongside the agent registry:

```
Holds:
- {id: "h1", holder: "reviewer-1", target: "planner-1", release: done("reviewer-1"), tag: "plan-review-1", task_id: <bg task id>, cursor: 567}
- {id: "h2", holder: "exec-1", target: "planner-1", release: first_intent("exec-1"), tag: "plan-handoff-1", task_id: <bg task id>, cursor: 580}
```

#### Hold Detection (Background Tasks)

One background task per hold. Cursor-based polling (same pattern as Section 2.1 / 6.4):

```bash
# Example: hold h1, watching for reviewer-1 to post done
# Replace 567 with literal cursor value captured when hold was created.
# PERSISTENT watcher — restarts on timeout until the release fires.
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

The watcher is persistent: it restarts after each 2-minute window, advancing the cursor. This prevents stuck holds where the holder's event arrives after a one-shot timeout. The background task only exits when the release event fires.

**When a hold fires:**
1. Drain its background task (TaskOutput block:false)
2. Remove hold from registry
3. Check if target has remaining incoming holds
4. If no remaining holds → kill target pane → DONE
5. If remaining holds → target stays in PENDING-RELEASE
6. `tmux select-layout main-vertical` (if pane was killed)

#### When to Create Holds

Create holds at spawn time or phase transitions — BEFORE the agent whose work triggers the hold has posted done. The lead decides which pattern to use based on the workflow.

**NEVER create holds retroactively.** If an agent posts `done` before a hold is created, the agent is already DONE and the pane is killed. Holds are preventive, not corrective.

**Hold watchers replace standard agent wait tasks.** In hold-aware phases (e.g., Phase 3 of the pipeline), the hold watcher watches for the same event as the standard agent wait task (Section 2.1 Step 3). Do NOT run both. The lead uses the hold watcher's output as the signal to both dissolve the hold AND advance the phase.
```

- [ ] **Step 2: Insert documented patterns subsection**

Append after the hold mechanism definition:

```markdown
#### Documented Patterns

These are RECIPES for common workflows. The lead references them when deciding what holds to create. New patterns are added here without any mechanism change.

**Pattern: Pair (Review)**

When spawning a reviewer for an agent's work:
- Hold: `{holder: reviewer, target: subject, release: done(reviewer)}`
- Subject enters PENDING-RELEASE when it posts done. Reviewer can post `wait` and subject responds. When reviewer posts done, hold dissolves.

**Pattern: Handoff**

When one agent passes work to another:
- Hold: `{holder: receiver, target: sender, release: first_intent(receiver)}`
- Sender enters PENDING-RELEASE when it posts done. Receiver can ask questions. When receiver begins independent work (first intent), hold dissolves.
- Alternative release: `signal(receiver, "ack", sender)` for explicit handshake.

**Pattern: Fan-In (Research Synthesis)**

When multiple producers report to a single consumer:
- Holds: for each producer, `{holder: consumer, target: producer, release: done(consumer)}`
- All producers enter PENDING-RELEASE when they post done. Consumer reads findings, asks follow-ups. When consumer posts done, ALL holds dissolve.

**Pattern: Merge Queue**

When parallel worktree executors need sequential lead-coordinated merging:
- Holds: for each executor, `{holder: "merge-process", target: exec-K, release: lead_release("merge-N-exec-K")}`
- `"merge-process"` is a virtual holder — the lead itself has the need.
- All executors enter PENDING-RELEASE when done. Lead merges one at a time:
  1. Tell exec-K to rebase on main and re-test
  2. If rebase conflicts → exec-K resolves (alive in PENDING-RELEASE)
  3. After green tests → `git merge --ff-only` → `lead_release` for that executor
  4. Next executor rebases on updated main, repeat

**Pattern: Barrier (Co-Design)**

When multiple agents collaborate equally on a shared artifact:
- Holds: for each member, `{holder: "codesign-coordinator", target: member-K, release: lead_release("codesign-N")}`
- All members stay alive. When lead signals design complete, ALL holds dissolve.
- Uses virtual holder for simplicity (N holds vs N*(N-1) for bidirectional).

**Pattern: Expert Consultation**

When an executor needs a specialist answer:
- Hold: `{holder: exec-N, target: researcher-K, release: done(exec-N)}`
- Researcher stays alive until executor finishes (in case of follow-up questions).
- For one-shot consultations: `release: done(researcher-K)` instead.
```

- [ ] **Step 3: Run behavioral tests**

Verify new Section 3.5 is detected and patterns are present. Should PASS.

- [ ] **Step 4: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "feat(lead-skill): add Section 3.5 hold-based agent retention (#483)

Hold primitive, release conditions, registry, detection pattern,
and 6 documented patterns: pair, handoff, fan-in, merge queue,
barrier, expert consultation.

AI-Used: [claude]"
```

---

## Task 4: Update Section 4.2 — Plan-Execute-Review Pipeline with holds

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` — lines 374–418

- [ ] **Step 1: Update Phase 1 (PLAN) for PENDING-RELEASE**

Replace step 5 (when planner done) with:

```
5. When planner done:
   a. Do NOT kill planner yet — Phase 1b will create a plan-review hold, and Phase 2 will create a plan-handoff hold. For now, simply note that planner posted done.
   b. Read plan text from new lines (cursor-based)
   c. Spawn reviewer for mandatory plan review (Phase 1b)
```

Note: plan-handoff hold is created in Phase 2 at executor spawn time (not pre-registered with a placeholder). The planner is not at risk of premature killing because Phase 1b immediately creates a plan-review hold targeting it.

- [ ] **Step 2: Add Phase 1b (mandatory plan review)**

Insert between Phase 1 and Phase 2:

```markdown
**Phase 1b: PLAN REVIEW (mandatory)**
1. Capture per-spawn cursor: `wc -l < "$CHAT_FILE"` → `PLAN_REVIEW_START`
2. Create plan-review hold: {id: "h-planrev-N", holder: "reviewer-R", target: "planner-N", release: done("reviewer-R"), tag: "plan-review-N"}
3. Spawn `reviewer-<R>` with plan-review role (Section 2.2):
   ```
   active code reviewer named reviewer-<R>.
   Your task: Review the plan for <issue> for completeness, correctness, and feasibility.
   Check: are edge cases addressed? Is the design overcomplicated? Does it align with CLAUDE.md?
   planner-<N> is alive — post wait addressed to planner-<N> if issues need revision.
   Post done with findings when plan is ready for user review.
   ```
4. Create hold detection background task for h-planrev-N
5. When reviewer-R posts done:
   a. Dissolve plan-review hold → reviewer-R has no other holds → kill reviewer-R
   b. Planner-N stays alive (still has plan-handoff hold)
   c. Present reviewed plan to user for approval
6. User approves → Phase 2
```

- [ ] **Step 3: Update Phase 2 (EXECUTE) with hold creation**

After spawning exec-N, add:

```
2b. Create plan-handoff hold: {id: "h-handoff-N", holder: "exec-N", target: "planner-N", release: first_intent("exec-N"), tag: "plan-handoff-N"}. Capture cursor and start hold detection background task.
2c. Planner-N now has an incoming hold → enters PENDING-RELEASE (if not already from plan-review).
2d. Include in executor role prompt: "planner-<N> is still alive and can answer questions about the plan. Address questions to planner-<N> in chat. After posting done, continue watching chat for further instructions."
```

Note: the hold is created HERE at executor spawn time, not pre-registered in Phase 1 with a placeholder. This eliminates the fragile exec-TBD activation dance.

- [ ] **Step 4: Update Phase 3 (REVIEW) with hold creation**

When executor posts done (transition from Phase 2 to Phase 3):

```
7. Executor enters PENDING-RELEASE (wait for impl-review)
8. Capture REVIEW_START cursor
9. Create impl-review hold: {id: "h-implrev-N", holder: "reviewer-R", target: "exec-N", release: done("reviewer-R"), tag: "impl-review-N"}
10. Spawn reviewer-R with impl-review role:
    ```
    active code reviewer named reviewer-<R>.
    Your task: Review the implementation of <plan> against the spec.
    exec-<N> is alive — post wait addressed to exec-<N> if changes are needed. It can implement fixes.
    Post done with findings when review is complete.
    ```
11. Create hold detection background task for h-implrev-N
12. When reviewer-R posts done:
    a. Dissolve impl-review hold → exec-N has no other holds → kill both exec-N and reviewer-R
    b. Report to user
```

Replace the existing WAIT handling:

```
If reviewer posts wait (requesting changes):
- Executor is alive (PENDING-RELEASE) — receives wait directly
- Executor implements fixes, posts done again — stays in PENDING-RELEASE (hold still active)
- Reviewer continues, eventually posts done → hold dissolves → both released
```

- [ ] **Step 5: Run behavioral tests**

Verify pipeline now includes hold creation at all phase boundaries. Phase 1b mandatory.

- [ ] **Step 6: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "feat(lead-skill): add holds to plan-execute-review pipeline (#483)

Plan-review hold, plan-handoff hold, impl-review hold created at
phase boundaries. Mandatory Phase 1b for plan review. Executor
and planner stay alive via PENDING-RELEASE for reviewer/executor
interactions.

AI-Used: [claude]"
```

---

## Task 5: Update Section 4.3 — Merge Queue for Parallel Executors

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` — lines 420–435

- [ ] **Step 1: Replace merge strategy with merge queue pattern**

Replace the "Merge strategy after all complete" section (lines 430–433) with:

```markdown
**Merge strategy — lead-coordinated merge queue:**

When all executors post done, each enters PENDING-RELEASE (held by merge-process holds — see Section 3.5 Merge Queue pattern).

Create holds at executor spawn time:
```
For each exec-K:
  {holder: "merge-process", target: exec-K, release: lead_release("merge-N-exec-K"), tag: "merge-queue-N"}
```

Sequential merge procedure (lead coordinates — does NOT delegate):
1. Pick exec-1 (first executor to merge)
2. Tell exec-1 via chat: "Rebase your branch on main, run `targ check-full`, and post done when green."
3. Wait for exec-1 done (background task, cursor-based)
4. If exec-1 reports rebase conflicts: exec-1 resolves (it's alive in PENDING-RELEASE), re-test, post done again
5. After green tests: lead runs `git merge --ff-only /tmp/engram-worktree-exec-1-branch` (lead delegates this to a short-lived executor if needed per coordinator rules)
6. `lead_release("merge-N-exec-1")` → exec-1 released → pane killed
7. Move to exec-2: tell it to rebase on updated main, repeat from step 3
8. After all merged: clean up worktrees

**Why this beats the race:** No executor independently rebases or retests. The lead controls merge order. Each executor resolves only its own conflicts when asked. Rebase onto updated main happens once per executor, in sequence.
```

- [ ] **Step 2: Run behavioral tests**

Verify merge queue pattern replaces old merge strategy.

- [ ] **Step 3: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "feat(lead-skill): replace merge race with lead-coordinated merge queue (#483)

Parallel executors held alive via merge-process holds. Lead merges
one at a time in sequence — no independent rebase/retest races.

AI-Used: [claude]"
```

---

## Task 6: Add new routing sections — Research Synthesis (4.5) and Co-Design (4.6)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` — insert after Section 4.4

- [ ] **Step 1: Add Section 4.5 — Research Synthesis**

Insert after Section 4.4:

```markdown
### 4.5 Research Synthesis

When a task requires gathering information from multiple angles before producing a unified output.

**When to use:** User says "research X from angles A, B, C and synthesize" or the lead determines a question needs multiple independent investigations combined.

**Procedure:**
1. Spawn N researchers in parallel, each with a distinct research angle
2. Spawn 1 synthesizer agent with role:
   ```
   active synthesizer named synthesizer-<N>.
   Your task: Wait for all researchers to post done. Read their findings from chat.
   Ask follow-up questions to any researcher if findings are unclear or incomplete.
   Synthesize findings into a unified report and post done.
   ```
3. Create fan-in holds (Section 3.5 Fan-In pattern):
   ```
   For each researcher-K:
     {holder: "synthesizer-N", target: "researcher-K", release: done("synthesizer-N"), tag: "synthesis-N"}
   ```
4. Lead's wait task watches for synthesizer's done (not researchers' done)
5. When synthesizer posts done: all holds dissolve, all researchers released, synthesizer killed normally
6. Lead reads synthesizer's report from chat, presents to user
```

- [ ] **Step 2: Add Section 4.6 — Co-Design**

Insert after new Section 4.5:

```markdown
### 4.6 Co-Design

When a task requires multiple perspectives collaborating on a shared design artifact.

**When to use:** User says "design X considering architecture, UX, and use cases" or the lead determines a design needs multiple specialist planners.

**Procedure:**
1. Spawn N planners in parallel, each with a specialist perspective:
   - Architecture planner: system structure, data flow, integration points
   - UX planner: user experience, interaction patterns, accessibility
   - Use-case planner: functional requirements, edge cases, acceptance criteria
   (Perspectives vary by task — these are examples.)
2. All planners post to a shared chat thread (e.g., `thread = "codesign-N"`)
3. Create barrier holds (Section 3.5 Barrier pattern):
   ```
   For each planner-K:
     {holder: "codesign-coordinator", target: "planner-K", release: lead_release("codesign-N"), tag: "codesign-N"}
   ```
4. Lead monitors the shared thread. Planners read each other's contributions and respond naturally via the chat protocol.
5. When the lead determines the design is converging (or the user approves):
   a. Post info to chat summarizing the consensus
   b. `lead_release("codesign-N")` → all holds dissolve → all planners released
6. Lead presents unified design to user for final approval
```

- [ ] **Step 3: Update routing decision table (Section 4.1)**

Add two new rows to the table at lines 362–372:

| "Research X from multiple angles" / "Investigate X and Y and synthesize" | **Research Synthesis** (Section 4.5) | N researchers + 1 synthesizer |
| "Design X with architecture + UX + use cases" / multi-perspective design | **Co-Design** (Section 4.6) | N specialist planners |

- [ ] **Step 4: Run behavioral tests**

Verify new sections exist and routing table has new entries.

- [ ] **Step 5: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "feat(lead-skill): add research synthesis and co-design routing (#483)

Section 4.5 (research synthesis with fan-in holds) and Section 4.6
(co-design with barrier holds). Routing table updated with new patterns.

AI-Used: [claude]"
```

---

## Task 7: Update role templates (Section 2.2) and context retention (Section 7.1)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` — lines 231–264, 559–566

- [ ] **Step 1: Update existing role templates with hold-awareness**

**CRITICAL (review finding #1):** ALL role templates must include this line:
> "After posting done, continue watching chat for further instructions. You may receive follow-up questions or requests while held in PENDING-RELEASE."

This is a prerequisite for PENDING-RELEASE to work — without it, held agents are deaf.

Update the executor template (lines 233–240):

```
active general-purpose executor named exec-<N>.
Your task: <task description>.
Work in this directory: <pwd>.
Use relevant skills. Post intent before significant actions.
When done, post done with a summary of what you changed.
After posting done, continue watching chat for further instructions. You may receive follow-up questions or requests while held in PENDING-RELEASE.
```

Update the planner template (lines 243–248):

```
active planner named planner-<N>.
Your task: Analyze <issue/task> and produce a step-by-step implementation plan.
Do NOT implement -- only plan.
Post the plan as an info message when done.
After posting done, continue watching chat — a reviewer and/or executor may have questions while you are held in PENDING-RELEASE.
```

Update the reviewer template (lines 250–256):

```
active code reviewer named reviewer-<N>.
Your task: Review <what> for <criteria>.
<subject-agent> is alive and can respond to your feedback.
Post wait addressed to <subject-agent> if you find issues that must be fixed.
Post done with findings when review is complete.
After posting done, continue watching chat for further instructions.
```

Update the researcher template (lines 258–264):

```
active researcher named researcher-<N>.
Your task: Research <topic> and report findings.
Do NOT modify code.
Post done with findings when research is complete.
After posting done, continue watching chat — a synthesizer may have follow-up questions while you are held in PENDING-RELEASE.
```

- [ ] **Step 2: Add new role templates**

Add after the Researcher template (after line 264):

```markdown
**Synthesizer:**
```
active synthesizer named synthesizer-<N>.
Your task: Wait for all researchers to post done. Read their findings from chat.
Ask follow-up questions to any researcher if findings are unclear or incomplete.
Synthesize findings into a unified report.
Post done with synthesis when complete.
After posting done, continue watching chat for further instructions.
```

**Co-Designer:**
```
active co-designer named planner-<N>.
Your task: Contribute the <perspective> perspective to the design of <artifact>.
Post to thread "codesign-<M>". Read other planners' contributions and respond.
Collaborate until the design converges.
Post done with your final contribution when the lead signals completion.
After posting done, continue watching chat for further instructions.
```

**Plan Reviewer:**
```
active code reviewer named reviewer-<N>.
Your task: Review the plan for <issue> for completeness, correctness, and feasibility.
Check: are edge cases addressed? Is the design overcomplicated? Does it align with CLAUDE.md?
planner-<N> is alive — post wait addressed to planner-<N> if issues need revision.
Post done with findings when plan is ready for user review.
After posting done, continue watching chat for further instructions.
```
```

- [ ] **Step 3: Update Section 7.1 — add hold registry to context retention**

Add a new row to the table at line 563:

| Hold registry (id, holder, target, release, tag, task_id, cursor) | Always (alongside agent registry) |

- [ ] **Step 4: Run behavioral tests**

Verify new templates and context retention changes.

- [ ] **Step 5: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "feat(lead-skill): update role templates and context retention for holds (#483)

ALL role templates updated with PENDING-RELEASE "keep watching" line.
Hold-aware planner/reviewer templates. New synthesizer, co-designer,
plan-reviewer templates. Hold registry added to context retention.

AI-Used: [claude]"
```

---

## Task 8: Update Section 6.4 — Background Task Hygiene for holds

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` — lines 517–555

- [ ] **Step 1: Add hold-specific hygiene rules**

Add after Rule 6 (line 527):

```markdown
7. **One persistent background task per hold.** Each hold in the registry has a `task_id` tracking its detection task. Hold watchers are persistent — they restart on timeout (see Section 3.5 detection pattern). When a hold dissolves (release condition fires), drain its task immediately.
8. **Drain on lead_release.** When `lead_release(tag)` fires, drain ALL background tasks for holds with that tag. Do not wait for them to detect the release — the lead already knows.
9. **Hold detection tasks do not replace each other.** Unlike the single chat watcher (Rule 1), multiple hold detection tasks run concurrently — one per active hold. This is bounded by the concurrency limit (max 5 agents → practical max ~10 holds).
10. **Hold watchers replace standard agent wait tasks in hold-aware phases.** When a hold watcher watches for the same event as the standard Section 2.1 wait task (e.g., reviewer done), do NOT run both. Use the hold watcher's output for both hold dissolution and phase advancement.
```

- [ ] **Step 2: Run behavioral tests**

Verify new hygiene rules are present.

- [ ] **Step 3: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "feat(lead-skill): add hold-specific background task hygiene rules (#483)

Rules 7-9: one task per hold, drain on lead_release, concurrent
hold watchers bounded by agent concurrency limit.

AI-Used: [claude]"
```

---

## Task 9: Final verification and cleanup

- [ ] **Step 1: Run full behavioral test suite**

Run all writing-skills behavioral tests. Everything should PASS against the updated skill.

- [ ] **Step 2: Verify spec coverage**

Cross-check against spec success criteria:

| Criterion | Covered By |
|-----------|------------|
| 1. Reviewer posts wait to planner, planner responds | Task 4 Phase 1b (plan-review hold) |
| 2. Reviewer posts wait to executor, executor fixes | Task 4 Phase 3 (impl-review hold) |
| 3. Executor asks planner questions during handoff | Task 4 Phase 2 (plan-handoff hold) |
| 4. Synthesizer asks follow-up to researchers | Task 6 Section 4.5 (fan-in holds) |
| 5. Sequential lead-coordinated merge | Task 5 (merge queue holds) |
| 6. Co-design planners stay alive | Task 6 Section 4.6 (barrier holds) |
| 7. No regression for holdless agents | Task 2 (state diagram: no holds → DONE) |
| 8. PENDING-RELEASE silence nudging | Task 2 (state diagram: PENDING-RELEASE → SILENT) |
| 9. New patterns need only new docs | Task 3 (patterns are recipes, not types) |

- [ ] **Step 3: Run `targ check-full`**

Not applicable — this is a Markdown-only change. No Go code modified.

- [ ] **Step 4: Final commit (if any cleanup needed)**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(lead-skill): final cleanup for hold-based lifecycle (#483)

AI-Used: [claude]"
```

---

## Self-Review Matrix

| Spec Section | Plan Task(s) |
|-------------|-------------|
| Problem / Root Cause | Context only (no task needed) |
| Hold Primitive definition | Task 3 Step 1 |
| Release Conditions | Task 3 Step 1 |
| Hold Lifecycle | Task 3 Step 1 |
| Agent State Machine (updated) | Task 2 Steps 1-2 |
| PENDING-RELEASE state | Task 2 Step 2 |
| Hold Registry | Task 3 Step 1 |
| Hold Detection | Task 3 Step 1 |
| Pattern: Pair (Review) | Task 3 Step 2, Task 4 Steps 2/4 |
| Pattern: Handoff | Task 3 Step 2, Task 4 Steps 1/3 |
| Pattern: Fan-In | Task 3 Step 2, Task 6 Step 1 |
| Pattern: Merge Queue | Task 3 Step 2, Task 5 Step 1 |
| Pattern: Barrier (Co-Design) | Task 3 Step 2, Task 6 Step 2 |
| Pattern: Expert Consultation | Task 3 Step 2 |
| PENDING-RELEASE responsiveness prerequisite | Task 7 Step 1 (all templates) |
| Changes: Section 3.1 | Task 2 |
| Changes: Section 3.2 (nudge text) | Task 2 Step 2b |
| Changes: New 3.5 | Task 3 |
| Changes: Section 4.2 | Task 4 |
| Changes: Section 4.3 | Task 5 |
| Changes: New 4.5/4.6 | Task 6 |
| Changes: Section 2.2 | Task 7 Steps 1-2 |
| Changes: Section 7.1 | Task 7 Step 3 |
| Changes: Section 6.4 | Task 8 |
| Success Criteria 1-9 | Task 9 Step 2 |
