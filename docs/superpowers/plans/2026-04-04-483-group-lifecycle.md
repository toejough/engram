# Group Lifecycle Tracking for Lead Agent (#483) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix `skills/engram-tmux-lead/SKILL.md` so the lead tracks logical agent groups and never kills an agent whose group activity is still in progress.

**Architecture:** Add a `PENDING-GROUP-RELEASE` state to the agent lifecycle state machine. When an agent posts `done`, the lead checks its group registry; if the agent belongs to an active group, it enters PENDING-GROUP-RELEASE (pane kept alive) instead of DONE (pane killed). A cursor-based background wait task monitors for the group's release condition; when it fires, the lead kills all pending members. Three group types cover the existing pipeline: `plan-review` (planner + plan reviewer), `impl-review` (executor + impl reviewer), `plan-handoff` (planner + executor).

**Tech Stack:** Skill file only (`skills/engram-tmux-lead/SKILL.md`) — Markdown text edits. No Go code changes. Uses existing chat cursor pattern and background task machinery already documented in the skill.

**Spec:** `docs/superpowers/specs/2026-04-04-483-group-lifecycle-design.md`

---

## Files

- Modify: `skills/engram-tmux-lead/SKILL.md`
  - Section 3.1 (state diagram + state table)
  - New Section 3.5 (Group Lifecycle)
  - Section 4.2 (Plan-Execute-Review Pipeline)
  - Section 2.2 (Agent Role Templates)
  - Section 7.1 (What to Keep in Context)
  - Section 6.4 (Background Task Hygiene — drain note)

---

## Task 1: Invoke writing-skills skill and establish baseline behavioral tests

**Files:**
- Read: `skills/engram-tmux-lead/SKILL.md`

This is a skill file edit. Per CLAUDE.md: **ALWAYS use `superpowers:writing-skills` when editing any SKILL.md file. No exceptions.** Invoke it first.

- [ ] **Step 1: Invoke writing-skills skill**

```
Skill tool: superpowers:writing-skills
```

Follow the skill's instructions for TDD workflow. The steps below define the baseline tests and failure scenarios as required by that workflow.

- [ ] **Step 2: Write baseline behavioral test scenarios (RED)**

Create a file `docs/superpowers/plans/2026-04-04-483-baseline-tests.md` with the following content:

```markdown
# Baseline Behavioral Tests for engram-tmux-lead Issue #483

## Test A — Reviewer posts WAIT to planner after planner posts done

Scenario: Lead spawns planner-1 for plan phase. planner-1 completes its plan and posts:
  type = "done", from = "planner-1"
The current skill says (Section 3.1): "Any state ──(task done)──> DONE (window killed)"
Lead reads this done message and kills planner-1's pane immediately.

Later, reviewer-1 posts type = "wait" addressed to planner-1 to ask a question.
planner-1's pane is already dead. Reviewer-1 gets no response.

EXPECTED (post-fix): Lead does NOT kill planner-1's pane when it posts done.
planner-1 enters PENDING-GROUP-RELEASE. Reviewer-1's wait is received and answered.

CURRENT RESULT: FAIL — planner-1 is killed, reviewer-1 gets no response.

## Test B — Reviewer posts WAIT to executor after executor posts done

Scenario: Lead spawns exec-1 for execute phase. exec-1 completes its work and posts:
  type = "done", from = "exec-1"
Current skill kills exec-1's pane immediately.

reviewer-1 spawned in Phase 3 finds bugs and posts type = "wait" addressed to exec-1.
exec-1 is already dead. Changes cannot be implemented.

EXPECTED (post-fix): exec-1 stays in PENDING-GROUP-RELEASE during review.
If reviewer-1 posts wait, exec-1 is still alive to implement fixes.

CURRENT RESULT: FAIL — exec-1 is killed before reviewer-1 can request changes.

## Test C — Executor asks planner a question during execution

Scenario: planner-1 posts done. Lead kills it. exec-1 is later spawned.
exec-1 discovers an ambiguity in the plan and wants to ask planner-1.
planner-1's pane is dead. No answer available.

EXPECTED (post-fix): planner-1 stays in PENDING-GROUP-RELEASE until exec-1 signals
it no longer needs the planner (posts first intent or ack to planner-1).

CURRENT RESULT: FAIL — planner-1 is killed before exec-1 starts.

## Test D — Non-grouped agent is killed immediately (no regression)

Scenario: Lead spawns a standalone executor (not part of a pipeline) for a quick fix.
exec-1 posts type = "done". No group registered for exec-1.

EXPECTED (post-fix): Lead kills exec-1's pane immediately (same as current behavior).

CURRENT RESULT: PASS — this is the current behavior that must not regress.

## Test E — PENDING-GROUP-RELEASE agent that goes silent is still nudged

Scenario: planner-1 is in PENDING-GROUP-RELEASE (posted done, waiting for reviewer).
planner-1 goes silent for 3 minutes (silence_threshold for task agents).

EXPECTED (post-fix): Lead nudges planner-1 per Section 3.2, same as ACTIVE agents.

CURRENT RESULT: N/A — state doesn't exist yet.
```

- [ ] **Step 3: Commit baseline tests**

```bash
git add docs/superpowers/plans/2026-04-04-483-baseline-tests.md
git commit -m "test(skills): add baseline behavioral tests for #483 group lifecycle fix

AI-Used: [claude]"
```

---

## Task 2: Update Section 3.1 — State Machine Diagram and State Definitions Table

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` lines 285–302

- [ ] **Step 1: Replace the state machine diagram**

In `skills/engram-tmux-lead/SKILL.md`, replace the diagram block (lines 285–292):

```
STARTING ──(first chat message)──> ACTIVE
ACTIVE ──(no message for silence_threshold)──> SILENT
SILENT ──(nudge succeeds)──> ACTIVE
SILENT ──(nudge fails / tmux window gone)──> DEAD
DEAD ──(lead decides)──> RESPAWN or REPORT+DONE
Any state ──(task done)──> DONE (window killed)
```

With:

```
STARTING ──(first chat message)──> ACTIVE
ACTIVE ──(no message for silence_threshold)──> SILENT
SILENT ──(nudge succeeds)──> ACTIVE
SILENT ──(nudge fails / tmux pane gone)──> DEAD
DEAD ──(lead decides)──> RESPAWN or REPORT+DONE
ACTIVE ──(task done, NOT in active group)──> DONE (pane killed)
ACTIVE ──(task done, IS in active group)──> PENDING-GROUP-RELEASE
PENDING-GROUP-RELEASE ──(group release condition fires)──> DONE (pane killed)
PENDING-GROUP-RELEASE ──(no message for silence_threshold)──> SILENT
```

- [ ] **Step 2: Update DONE row in the state definitions table**

In the state definitions table (around line 302), replace the DONE row:

Old:
```
| **DONE** | Agent posted `done` for its assigned task | Kill pane by tracked ID: `tmux kill-pane -t <pane-id>`. Rebalance: `tmux select-layout main-vertical`. Remove from tracking. |
```

New:
```
| **DONE** | Agent posted `done` AND has no active group membership; OR group release condition fired | Kill pane by tracked ID: `tmux kill-pane -t <pane-id>`. Rebalance: `tmux select-layout main-vertical`. Remove from tracking. |
```

- [ ] **Step 3: Add PENDING-GROUP-RELEASE row after the DONE row**

After the DONE row, add:

```
| **PENDING-GROUP-RELEASE** | Agent posted `done` AND belongs to an active group whose release condition has not yet fired | Do NOT kill pane. Agent stays alive and responsive. Start a background wait task watching for the group's release condition (see Section 3.5). Silence threshold still applies — nudge per Section 3.2 if silent too long. When release fires: kill pane, rebalance, remove group record. |
```

- [ ] **Step 4: Verify the edit looks correct**

Run:
```bash
sed -n '281,315p' skills/engram-tmux-lead/SKILL.md
```

Expected: diagram shows 9 lines (not 6), state table has 6 rows (not 5), DONE row includes "NOT in active group" guard.

- [ ] **Step 5: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(skill/lead): add PENDING-GROUP-RELEASE state to lifecycle state machine

Addresses #483: agents that post done while in an active group now enter
PENDING-GROUP-RELEASE instead of DONE, keeping their pane alive until
the group's release condition fires.

AI-Used: [claude]"
```

---

## Task 3: Add Section 3.5 — Group Lifecycle

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (insert after Section 3.4)

- [ ] **Step 1: Insert new Section 3.5 after Section 3.4**

Find the line (around line 354):
```
Do not attempt to re-implement the shutdown sequence inline. The skill owns the procedure.
```

After that line (and its following blank line), before `## 4. Routing`, insert:

```markdown
### 3.5 Group Lifecycle

The lead tracks **logical agent groups** — sets of agents that need to interact to complete a shared activity. An agent posting `done` does NOT trigger a pane kill if it belongs to an active group; instead it enters PENDING-GROUP-RELEASE and stays alive until the group's release condition fires.

#### Group Types

| Type | Members | Created When | Release Condition |
|------|---------|-------------|-------------------|
| `plan-review` | [planner-N, reviewer-R] | Reviewer spawned for plan review | reviewer-R posts `type = "done"` |
| `impl-review` | [exec-N, reviewer-R] | Reviewer spawned for implementation review | reviewer-R posts `type = "done"` |
| `plan-handoff` | [planner-N, exec-N] | Planner spawned (pre-registered; exec slot filled when executor spawned after user approval) | exec-N posts `type = "ack"` addressed to planner-N OR exec-N posts its first `type = "intent"` |

#### Group Registry

The lead maintains a group registry alongside the agent registry (Section 7.1). Each group record:

```
{id: "impl-review-1", type: "impl-review",
 members: [exec-1, reviewer-1],
 pending: [exec-1],       # members currently in PENDING-GROUP-RELEASE
 release_task_id: <bkg>,  # background task ID watching for release condition
 release_cursor: 567}     # line count when background wait was started
```

#### Release Detection

Use a cursor-based background wait task (same pattern as Section 2.1). Embed the release_cursor as a **literal number** — background bash has no shell variables.

```bash
# Example: impl-review group, waiting for reviewer-1 done.
# Replace 567 with the literal value of release_cursor.
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

Timeout: 60 iterations × 2s = 2 minutes. If timed out: nudge the blocking agent (the one that hasn't completed), transition to SILENT per Section 3.2. Do NOT kill the PENDING-GROUP-RELEASE agent on release timeout.

#### Release Procedure

When a group's release condition fires:
1. Drain: `TaskOutput(task_id=release_task_id, block=False)` — prevent zombie task accumulation
2. Kill all `pending` members' panes by tracked pane ID: `tmux kill-pane -t <pane-id>`
3. Rebalance: `tmux select-layout main-vertical`
4. Remove group record from registry

**HARD RULE: track all release task IDs.** Include them in session shutdown drain (Section 3.4 / engram-down skill). A completed-but-unread release task is a zombie.
```

- [ ] **Step 2: Verify section was inserted correctly**

Run:
```bash
grep -n "^### 3\." skills/engram-tmux-lead/SKILL.md
```

Expected output includes: `3.1`, `3.2`, `3.3`, `3.4`, `3.5`

- [ ] **Step 3: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(skill/lead): add Section 3.5 Group Lifecycle

Defines group types (plan-review, impl-review, plan-handoff), the group
registry model, cursor-based release detection pattern, and release procedure.

AI-Used: [claude]"
```

---

## Task 4: Update Section 4.2 — Plan-Execute-Review Pipeline

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` Section 4.2 (around lines 374–418)

- [ ] **Step 1: Replace Phase 1 with group-aware version**

Find and replace the `**Phase 1: PLAN**` block:

Old:
```
**Phase 1: PLAN**
1. Capture per-spawn cursor (foreground bash): `wc -l < "$CHAT_FILE"` → note as `PLAN_START`
2. Spawn `planner-<N>` with issue context (per Section 2.1 Steps 1–2)
3. Send role prompt (Section 2.1)
4. Run background wait task (Section 2.1 Step 3) — embed `PLAN_START` as literal, filter `from = "planner-<N>"` and `type = "done"`
5. When planner done: read plan text from new lines (cursor-based), present to user
6. Update session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`
7. User approves (or modifies) -> Phase 2
```

New:
```
**Phase 1: PLAN**
1. Pre-register a `plan-handoff` group: `{id: "plan-handoff-<N>", type: "plan-handoff", members: [planner-<N>, exec-TBD], pending: [], release_cursor: TBD}` — exec slot filled later
2. Capture per-spawn cursor (foreground bash): `wc -l < "$CHAT_FILE"` → note as `PLAN_START`
3. Spawn `planner-<N>` with issue context (per Section 2.1 Steps 1–2), using group-aware role template (Section 2.2)
4. Run background wait task (Section 2.1 Step 3) — embed `PLAN_START` as literal, filter `from = "planner-<N>"` and `type = "done"`
5. When planner done: planner-<N> enters **PENDING-GROUP-RELEASE** (do NOT kill pane). Read plan text from new lines (cursor-based), present to user.
6. Update session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`
7. User approves (or modifies) -> Phase 2
```

- [ ] **Step 2: Replace Phase 2 with group-aware version**

Find and replace the `**Phase 2: EXECUTE**` block:

Old:
```
**Phase 2: EXECUTE**
1. Capture per-spawn cursor (foreground bash): `wc -l < "$CHAT_FILE"` → note as `EXEC_START`
2. Spawn `exec-<N>` with approved plan (per Section 2.1 Steps 1–2)
3. Send role prompt with the approved plan text (Section 2.1)
4. Run background wait task (Section 2.1 Step 3) — embed `EXEC_START` as literal, filter `from = "exec-<N>"` and `type = "done"`
5. When executor done: read result summary from new lines (cursor-based)
6. Update session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`
7. -> Phase 3
```

New:
```
**Phase 2: EXECUTE**
1. Capture per-spawn cursor (foreground bash): `wc -l < "$CHAT_FILE"` → note as `EXEC_START`
2. Fill plan-handoff group: update group record with exec-N name and `release_cursor = EXEC_START`
3. Spawn `exec-<N>` with approved plan (per Section 2.1 Steps 1–2), using group-aware role template (Section 2.2) that names planner-<N> as available for questions
4. Start plan-handoff release detection (background, Section 3.5): watch for exec-<N> posting `type = "ack"` to planner-<N> OR exec-<N>'s first `type = "intent"`. Store task ID in group record as `release_task_id`.
5. Run background wait task (Section 2.1 Step 3) — embed `EXEC_START` as literal, filter `from = "exec-<N>"` and `type = "done"`
6. When executor done: exec-<N> enters **PENDING-GROUP-RELEASE** (do NOT kill pane). Read result summary from new lines (cursor-based).
7. Drain plan-handoff release task (it should have fired by now — exec posted intent): `TaskOutput(task_id=<group.release_task_id>, block=False)`. Kill planner-<N> pane if still alive. Remove plan-handoff group record.
8. Update session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`
9. -> Phase 3
```

- [ ] **Step 3: Replace Phase 3 with group-aware version**

Find and replace the `**Phase 3: REVIEW**` block:

Old:
```
**Phase 3: REVIEW**
1. Capture per-spawn cursor (foreground bash): `wc -l < "$CHAT_FILE"` → note as `REVIEW_START`
2. Spawn `reviewer-<N>` with original plan + `git diff` output (per Section 2.1 Steps 1–2)
3. Send role prompt (Section 2.1)
4. Run background wait task (Section 2.1 Step 3) — embed the literal cursor value, filter `from = "reviewer-<N>"` and **either** `type = "wait"` (issues found) **or** `type = "done"` (approved):
   ```bash
   # Replace 412 with the literal value noted in step 1. NOT a variable — background bash has no shell vars.
   tail -n +"$((412 + 1))" "$CHAT_FILE" | awk '
     /^\[\[message\]\]/ { from=""; msgtype="" }
     /^from = "reviewer-1"/ { from=1 }
     /^type = "wait"/ { msgtype="WAIT" }
     /^type = "done"/ { msgtype="DONE" }
     from && msgtype { print msgtype; exit }
   '
   ```
5. When reviewer responds:
   - `WAIT`: relay issues to user. Decide: fix (re-enter Phase 2) or accept as-is.
   - `DONE`: report to user, clean up agents
6. Update session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`
```

New:
```
**Phase 3: REVIEW**
1. Capture per-spawn cursor (foreground bash): `wc -l < "$CHAT_FILE"` → note as `REVIEW_START`
2. Register an `impl-review` group: `{id: "impl-review-<N>", type: "impl-review", members: [exec-<N>, reviewer-<N>], pending: [exec-<N>], release_cursor: REVIEW_START}`
3. Spawn `reviewer-<N>` with original plan + `git diff` output (per Section 2.1 Steps 1–2), using group-aware role template (Section 2.2) that names exec-<N> as available for change requests
4. Start impl-review release detection (background, Section 3.5): watch for reviewer-<N> posting `type = "done"`. Store task ID in group record `release_task_id`.
5. Also run a background wait for `type = "wait"` from reviewer-<N> (embed `REVIEW_START` as literal):
   ```bash
   # Replace 412 with the literal REVIEW_START value. NOT a variable.
   tail -n +"$((412 + 1))" "$CHAT_FILE" | awk '
     /^\[\[message\]\]/ { from=""; msgtype="" }
     /^from = "reviewer-1"/ { from=1 }
     /^type = "wait"/ { msgtype="WAIT" }
     /^type = "done"/ { msgtype="DONE" }
     from && msgtype { print msgtype; exit }
   '
   ```
6. When reviewer responds:
   - `WAIT`: exec-<N> is still alive (PENDING-GROUP-RELEASE). Relay issues to user. If fixing: exec-<N> implements changes and posts `done` again (stays in PENDING-GROUP-RELEASE — group is still active). Re-run step 5 watch loop from new cursor.
   - `DONE`: group release fired. Execute release procedure (Section 3.5): drain release task, kill exec-<N> and reviewer-<N> panes, rebalance, remove group record.
7. Update session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`
```

- [ ] **Step 4: Verify section looks correct**

```bash
sed -n '374,435p' skills/engram-tmux-lead/SKILL.md
```

Expected: Phase 1 has 7 steps (not 7), Phase 2 has 9 steps, Phase 3 has 7 steps. All three phases reference PENDING-GROUP-RELEASE.

- [ ] **Step 5: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(skill/lead): add group lifecycle to Plan-Execute-Review pipeline

Phase 1: pre-register plan-handoff group before spawning planner.
Phase 2: activate plan-handoff group, planner enters PENDING-GROUP-RELEASE.
Phase 3: register impl-review group, executor enters PENDING-GROUP-RELEASE
  until reviewer posts done; executor stays alive if reviewer posts wait.

AI-Used: [claude]"
```

---

## Task 5: Update Section 2.2 — Agent Role Templates

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` Section 2.2 (around lines 231–264)

- [ ] **Step 1: Update Planner template**

Find and replace the Planner template:

Old:
```
**Planner:**
```
active planner named planner-<N>.
Your task: Analyze <issue/task> and produce a step-by-step implementation plan.
Do NOT implement -- only plan.
Post the plan as an info message when done.
```
```

New:
```
**Planner:**
```
active planner named planner-<N>.
Your task: Analyze <issue/task> and produce a step-by-step implementation plan.
Do NOT implement -- only plan.
Post the plan as a done message when complete.
After posting done, remain available — the executor assigned your plan may have questions. Stay alive and watch chat until you receive a shutdown message.
```
```

Note: also fix "info message" → "done message" (planners should post `done`, not `info`, so the lead's done-detection in Section 2.1 fires correctly).

- [ ] **Step 2: Update Executor template**

Find the Executor template and add a group-context variant. The base template stays; add a group-aware note:

After the Executor template block, add:

```
**Executor (when spawned from Plan-Execute-Review pipeline with planner still alive):**
```
active general-purpose executor named exec-<N>.
Your task: <task description from approved plan>.
Work in this directory: <pwd>.
Use relevant skills. Post intent before significant actions.
planner-<N> is still alive and available for questions about the plan. Address questions to planner-<N> in chat. When you no longer need the planner, post an ack or your first intent — this signals the lead to release the planner.
When done implementing, post done with a summary of what you changed.
```
```

- [ ] **Step 3: Update Reviewer template**

Find the Reviewer template and add a group-context variant:

After the existing Reviewer template block, add:

```
**Reviewer (impl-review — when executor is in PENDING-GROUP-RELEASE):**
```
active code reviewer named reviewer-<N>.
Your task: Review <what> for correctness, test coverage, and adherence to the plan.
exec-<N> is still alive. If you find issues that must be fixed, post a wait message addressed to exec-<N> with specific change requests — it can implement fixes directly.
Post done with your findings when the implementation is approved.
```
```

- [ ] **Step 4: Verify templates look correct**

```bash
sed -n '231,280p' skills/engram-tmux-lead/SKILL.md
```

Expected: four template blocks visible (executor base, executor pipeline variant, planner, reviewer base, reviewer impl-review variant) with correct content.

- [ ] **Step 5: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(skill/lead): add group-aware role templates for planner, executor, reviewer

Planner: stay alive after done for executor questions.
Executor (pipeline): names planner, signals release via first intent.
Reviewer (impl-review): names executor, uses wait for change requests.

AI-Used: [claude]"
```

---

## Task 6: Update Section 7.1 and Section 6.4 — Context retention and task hygiene

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` Sections 7.1 and 6.4

- [ ] **Step 1: Add group registry to Section 7.1 retention table**

Find the table in Section 7.1:

```
| Active agent registry (name, state, role, last-message-ts, task summary) | Always |
```

After that row, add:

```
| Group registry (id, type, members, pending, release_task_id, release_cursor) | Always (until group released) |
```

- [ ] **Step 2: Add release task drain to Section 6.4 Background Task Hygiene**

In Section 6.4, find the numbered rules list. After Rule 3 ("Drain on shutdown"), add:

```
4. **Drain release tasks on group completion.** When a group's release condition fires, drain `release_task_id` before killing panes (step 1 of release procedure, Section 3.5). Release tasks for active groups must also be drained at session shutdown.
```

Renumber the existing Rule 4 → Rule 5, Rule 5 → Rule 6, Rule 6 → Rule 7.

- [ ] **Step 3: Verify**

```bash
grep -n "release_task_id\|Group registry\|Drain release" skills/engram-tmux-lead/SKILL.md
```

Expected: at least 3 matches.

- [ ] **Step 4: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(skill/lead): track group registry in context; drain release tasks on completion

AI-Used: [claude]"
```

---

## Task 7: Run pressure tests and verify behavioral change (GREEN)

**Files:**
- Read: `skills/engram-tmux-lead/SKILL.md`
- Read: `docs/superpowers/plans/2026-04-04-483-baseline-tests.md`

- [ ] **Step 1: Re-read the updated skill**

Read `skills/engram-tmux-lead/SKILL.md` sections 2.2, 3.1, 3.5, 4.2, 6.4, 7.1 in full and verify each baseline test scenario now passes against the updated instructions:

- Test A (reviewer WAIT to planner): Section 3.1 PENDING-GROUP-RELEASE + Section 3.5 plan-review group → planner stays alive. PASS?
- Test B (reviewer WAIT to executor): Section 4.2 Phase 3 WAIT path + impl-review group → executor stays alive. PASS?
- Test C (executor asks planner): Section 4.2 Phase 2 plan-handoff group → planner stays alive during execution. PASS?
- Test D (no regression for standalone agent): Section 3.1 "NOT in active group → DONE" guard → standalone agents still killed immediately. PASS?
- Test E (PENDING-GROUP-RELEASE agent nudged when silent): Section 3.1 PENDING-GROUP-RELEASE → SILENT transition + Section 3.2 nudge behavior. PASS?

- [ ] **Step 2: Run adversarial pressure tests**

Verify these edge cases are handled by the updated skill:

**P1 — Release timeout:** reviewer-1 goes silent for 2+ minutes without posting done. The release background task times out. Does the skill say what to do? Expected: Section 3.5 says "nudge the blocking agent, transition to SILENT per Section 3.2. Do NOT kill the PENDING-GROUP-RELEASE agent."

**P2 — Executor in PENDING-GROUP-RELEASE implements fixes, posts done again:** reviewer posts wait → executor implements fix → executor posts done (second time). Does the group stay active? Expected: Yes — group record still has exec-<N> in pending; release only fires when reviewer posts done, not when executor re-posts done.

**P3 — Plan-handoff release doesn't fire (executor never posts intent or ack to planner):** Background wait times out. Does the planner stay alive forever? Expected: Section 4.2 Phase 2 step 7 says "Drain plan-handoff release task... Kill planner-<N> pane if still alive." So even if exec never explicitly releases, planner is released when executor posts its own `done`.

**P4 — Lead restart recovery with active groups:** Lead dies mid-pipeline. User restarts lead. Does Section 7.4 cover group recovery? Expected: Section 7.4 reconstructs agent registry from chat.toml. Group registry may be lost — document this as a known limitation (groups are not persisted; a restarted lead treats all alive agents as ungrouped).

If P4 has a gap, add a note to Section 7.4:

```
**Group registry note:** Group state is in-context only and is not persisted to chat.toml. After a lead restart, active groups are lost — the lead will treat surviving agents as ungrouped and kill them when they next post `done`. This is acceptable: the pipeline phase can be re-run from the last checkpoint.
```

- [ ] **Step 3: Commit note if added**

If you added the P4 recovery note:

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "docs(skill/lead): note group registry loss on lead restart (known limitation)

Groups are in-context only. On restart, surviving agents are ungrouped.
Re-run from last pipeline phase checkpoint.

AI-Used: [claude]"
```

- [ ] **Step 4: Final verification**

```bash
grep -n "PENDING-GROUP-RELEASE\|plan-handoff\|impl-review\|plan-review\|group registry\|release_task_id" skills/engram-tmux-lead/SKILL.md | wc -l
```

Expected: 15+ matches (the concept appears throughout the skill at every relevant touch point).

- [ ] **Step 5: Close the issue**

```bash
gh issue close 483 --comment "Fixed in skills/engram-tmux-lead/SKILL.md. Added PENDING-GROUP-RELEASE state, Section 3.5 Group Lifecycle, group-aware pipeline phases (4.2), and group-aware role templates (2.2)."
```

---

## Self-Review Against Spec

**Spec sections → Tasks:**

| Spec Requirement | Covered By |
|-----------------|-----------|
| PENDING-GROUP-RELEASE state in Section 3.1 | Task 2 |
| New Section 3.5 Group Lifecycle | Task 3 |
| Group types (plan-review, impl-review, plan-handoff) | Task 3 |
| Group registry model | Task 3 |
| Release detection pattern | Task 3 |
| Section 4.2 phase boundary group creation | Task 4 |
| Reviewer wait path — executor stays alive | Task 4 (Phase 3) |
| Section 2.2 role template updates | Task 5 |
| Section 7.1 group registry in context | Task 6 |
| Success criteria 1–5 | Task 7 |

**No gaps found.** All spec requirements are covered.
