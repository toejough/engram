# Lead Skill 6-Bug Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:writing-skills` for ALL edits to SKILL.md (TDD: baseline test → edit → pressure test). Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement task-by-task.

**Goal:** Fix 6 confirmed bugs in `skills/engram-tmux-lead/SKILL.md` — blocking ready-wait (#490), monitor relay latency (#498), missing select-layout (#489), no skill injection (#485), agents linger after task (#497), and missing health-check mechanism (#493).

**Architecture:** All changes are to a single file (`skills/engram-tmux-lead/SKILL.md`). Changes are grouped by section so related edits land together. No Go code changes. No new files.

**Tech Stack:** Skill Markdown, bash, tmux, Claude Code Agent/Task tools.

---

## File Structure

**Modified:**
- `skills/engram-tmux-lead/SKILL.md` — five targeted section edits (Groups A–E)

---

## Task 1 (Group A): Fix #490 — Section 1.5: Non-Blocking engram-agent Ready Wait

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (Section 1.5, ~lines 165–189)

### Background

Section 1.5 documents a background Agent that waits for engram-agent's first chat message, but does not specify that `TaskOutput` must be called with `block=False`. The lead has been calling `TaskOutput(block=True)`, freezing the lead's entire turn until the 30-second timeout (or engram-agent posts). The user cannot interact with the lead during this window.

### Baseline Test (RED)

Read Section 1.5. Verify: the section does NOT contain an explicit instruction to call `TaskOutput(..., block=False)` for the ready wait, and does NOT show a polling loop with a 30-second timeout ceiling.

Expected: current text says "When it completes, check whether the engram-agent posted" — implies waiting for completion, no explicit `block=False`.

### Change

Replace the closing paragraph of Section 1.5 (after the background Agent task block), from:

```
When it completes, check whether the engram-agent posted. If not after 30 seconds:
1. Check pane exists: `tmux list-panes -F '#{pane_id} #{pane_pid}' | grep <tracked-pane-id>`
2. Check window output: `tmux capture-pane -t "${PROJECT_PREFIX}:engram-agent" -p | tail -20`
3. Report to user with diagnostic info. Do NOT silently proceed without memory.
```

...with:

```
**Use `block=False` — do NOT block.** After spawning the background Agent, poll it non-blocking between interactions. Never call `TaskOutput(task_id=..., block=True)` for this wait — that freezes the lead and blocks all user input.

```python
# After spawning ENGRAM_READY_TASK_ID:
import time
deadline = time.time() + 30
while time.time() < deadline:
    result = TaskOutput(task_id=ENGRAM_READY_TASK_ID, block=False)
    if result is not None:  # task completed
        break
    # Check for user input; handle it if present, then re-poll
    time.sleep(0.5)
```

If result is "ENGRAM-AGENT FOUND": proceed.
If result is "TIMEOUT" or deadline passed with no result:
1. Drain: `TaskOutput(task_id=ENGRAM_READY_TASK_ID, block=False)` (discard)
2. Check pane exists: `tmux list-panes -F '#{pane_id} #{pane_pid}' | grep <tracked-pane-id>`
3. Check window output: `tmux capture-pane -t "$PANE_ID" -p | tail -20`
4. Report to user with diagnostic info. Do NOT silently proceed without memory.

> **Background task drain note:** The polling loop background task completes as soon as the loop exits (found or timed out). Reading its output drains it. If you need to retry after a timeout, the previous task is already drained when you read its output. Only spawn a new check after fully reading the old task's result — never run two concurrent READY check loops.
```

### Pressure Test (GREEN)

Re-read Section 1.5. Verify:
- `block=False` appears explicitly
- A polling loop with a deadline is shown
- The note about "drain note" is preserved (or merged into the new text)
- The diagnostic steps (check pane, capture-pane) are preserved

---

## Task 2 (Group B): Fix #498 — Sections 1.6 + 6.1: Close Monitor Relay Race Window

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (Section 1.6 drain-before-spawn, ~lines 204–213; Section 6.1 replace pattern, ~lines 805–816)

### Background

The drain-before-spawn pattern has a race window: between draining the old monitor and spawning the new one, agent messages can arrive in the chat file and be missed. The lead's monitor won't see them until the next fswatch event. This caused `done` messages to sit unseen for 1–2+ minutes.

The fix: after draining the old monitor (and after processing any message it returned), do one additional sweep of the chat file from `CURSOR` to EOF. Process any messages found there before spawning the new monitor.

### Baseline Test (RED)

Read the drain-before-spawn pattern in Sections 1.6 and 6.1. Verify: neither contains a post-drain sweep of `CURSOR`→EOF for missed messages.

Expected: current pattern is drain → spawn, with no intermediate sweep.

### Change to Section 1.6 (drain-before-spawn pattern block)

Replace:
```python
# Before spawning a new chat monitor Agent:
if CHAT_MONITOR_TASK_ID:
    TaskOutput(task_id=CHAT_MONITOR_TASK_ID, block=False)  # drain; discard output
# Spawn replacement (Agent tool, run_in_background: true)
CHAT_MONITOR_TASK_ID = <new background Agent task id>
```

With:
```python
# Before spawning a new chat monitor Agent:
if CHAT_MONITOR_TASK_ID:
    TaskOutput(task_id=CHAT_MONITOR_TASK_ID, block=False)  # drain; discard output

# Post-drain sweep: catch any messages that arrived in the race window
# (foreground bash — embed CURSOR as literal integer):
new_lines = run_bash(f'tail -n +{CURSOR + 1} "$CHAT_FILE"')
CURSOR = run_bash('wc -l < "$CHAT_FILE"').strip()
if new_lines.strip():
    process_chat_messages(new_lines)   # relay, route, or queue as normal

# Spawn replacement (Agent tool, run_in_background: true)
CHAT_MONITOR_TASK_ID = <new background Agent task id>
```

### Change to Section 6.1 (replace pattern block)

Replace:
```python
# Drain old monitor Agent before spawning new one:
if CHAT_MONITOR_TASK_ID:
    TaskOutput(task_id=CHAT_MONITOR_TASK_ID, block=False)
# Spawn replacement monitor Agent (Agent tool, run_in_background: true):
# Task: Background Monitor Pattern from use-engram-chat-as, with current cursor
CHAT_MONITOR_TASK_ID = <task id from Agent tool result>
```

With:
```python
# Drain old monitor Agent before spawning new one:
if CHAT_MONITOR_TASK_ID:
    TaskOutput(task_id=CHAT_MONITOR_TASK_ID, block=False)

# Post-drain sweep: catch any messages that arrived in the race window.
# Run foreground bash before spawning new monitor:
new_lines = run_bash(f'tail -n +{CURSOR + 1} "$CHAT_FILE"')
CURSOR = run_bash('wc -l < "$CHAT_FILE"').strip()
if new_lines.strip():
    process_chat_messages(new_lines)   # relay, route, or queue as normal

# Spawn replacement monitor Agent (Agent tool, run_in_background: true):
# Task: Background Monitor Pattern from use-engram-chat-as, with current cursor
CHAT_MONITOR_TASK_ID = <task id from Agent tool result>
```

### Pressure Test (GREEN)

Re-read Sections 1.6 and 6.1. Verify:
- Both contain the post-drain sweep pattern
- Sweep uses `tail -n +$((CURSOR + 1))` (cursor-based, not full-file grep)
- CURSOR is updated after the sweep
- New monitor is spawned AFTER the sweep (not before)

---

## Task 3 (Group C): Fix #485 + #497 (template side) — Section 2.2: Skill Injection + Shutdown Watch Instruction

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (Section 2.2, ~lines 270–338)

### Background

**#485:** Role templates say "Use relevant skills" generically. The lead knows the task type at spawn time and should inject specific skill references so agents don't miss or underuse them.

**#497 (template side):** Templates say "continue watching chat for further instructions." After posting done, agents stay in their watch loop and can act on `all`-addressed messages meant for other agents. The fix: agents only respond to messages explicitly addressed to them by name after completing their primary task, and only exit on `shutdown` from lead.

### Baseline Test (RED)

Read Section 2.2. Verify:
- There is NO task-type → skill mapping table before the templates
- Templates say "Use relevant skills" (generic), not specific skill names
- Every template ends with "continue watching chat for further instructions"

Expected: all three conditions hold in the current file.

### Change A: Add Task-Type → Skill Mapping Table

Insert BEFORE the `**Executor:**` block (after the `### 2.2 Agent Role Templates` heading):

```markdown
**Task-Type → Skill Mapping**

Include these specific skills in the agent's role prompt based on task type:

| Task type | Skills to inject |
|-----------|-----------------|
| Planning / design | `superpowers:brainstorming`, `superpowers:writing-plans` |
| Implementation (feature/bug) | `superpowers:test-driven-development`, `feature-dev:feature-dev` |
| Code review | `superpowers:receiving-code-review` |
| Skill editing | `superpowers:writing-skills` |
| GitHub issues / PR filing | No specific skill — generic executor |
| Research | No specific skill — generic researcher |

Reference the mapping when writing role prompts. Be explicit: "Use `superpowers:brainstorming` then `superpowers:writing-plans` to produce the plan."
```

### Change B: Update Role Templates with Skill References and Shutdown Watch Instruction

Replace ALL occurrences of:
```
After posting done, continue watching chat for further instructions. You may receive follow-up questions or requests while held in PENDING-RELEASE.
```
and:
```
After posting done, continue watching chat for further instructions.
```
and:
```
After posting done, continue watching chat — a reviewer and/or executor may have questions while you are held in PENDING-RELEASE.
```
and:
```
After posting done, continue watching chat — a synthesizer may have follow-up questions while you are held in PENDING-RELEASE.
```

...with (per-template versions, see below).

**Updated Executor template:**
```
active general-purpose executor named exec-<N>.
Your task: <task description>.
Work in this directory: <pwd>.
Use <skill-refs per mapping table above>. Post intent before significant actions.
When done, post done with a summary of what you changed.
After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
```

**Updated Planner template:**
```
active planner named planner-<N>.
Your task: Analyze <issue/task> and produce a step-by-step implementation plan.
Do NOT implement -- only plan.
Use superpowers:brainstorming to explore requirements, then superpowers:writing-plans to produce the plan.
Post the plan as an info message when done.
After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
```

**Updated Reviewer template:**
```
active code reviewer named reviewer-<N>.
Your task: Review <what> for <criteria>.
Use superpowers:receiving-code-review.
<subject-agent> is alive and can respond to your feedback.
Post wait addressed to <subject-agent> if you find issues that must be fixed.
Post done with findings when review is complete.
After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
```

**Updated Researcher template:**
```
active researcher named researcher-<N>.
Your task: Research <topic> and report findings.
Do NOT modify code.
Post done with findings when research is complete.
After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
```

**Updated Synthesizer template:**
```
active synthesizer named synthesizer-<N>.
Your task: Wait for all researchers to post done. Read their findings from chat.
Ask follow-up questions to any researcher if findings are unclear or incomplete.
Synthesize findings into a unified report.
Post done with synthesis when complete.
After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
```

**Updated Co-Designer template:**
```
active co-designer named planner-<N>.
Your task: Contribute the <perspective> perspective to the design of <artifact>.
Post to thread "codesign-<M>". Read other planners' contributions and respond.
Collaborate until the design converges.
Post done with your final contribution when the lead signals completion.
After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
```

**Updated Plan Reviewer template:**
```
active code reviewer named reviewer-<N>.
Your task: Review the plan for <issue> for completeness, correctness, and feasibility.
Check: are edge cases addressed? Is the design overcomplicated? Does it align with CLAUDE.md?
Use superpowers:receiving-code-review.
planner-<N> is alive — post wait addressed to planner-<N> if issues need revision.
Post done with findings when plan is ready for user review.
After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
```

### Pressure Test (GREEN)

Re-read Section 2.2. Verify:
- Task-type → skill mapping table exists before templates
- Executor template references specific skills (or instructs lead to inject per mapping)
- Planner template explicitly names `superpowers:brainstorming` and `superpowers:writing-plans`
- Reviewer template explicitly names `superpowers:receiving-code-review`
- Every template's closing watch instruction says "watch for a shutdown message from lead" and "only respond to messages explicitly addressed to you"
- NO template says "continue watching chat for further instructions" without the shutdown/scoping qualifier

---

## Task 4 (Group D): Fix #489 + #497 (lead side) — Sections 3.1 + 3.3: select-layout + Shutdown on DONE

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (Section 3.1 state table, ~line 389; Section 3.3 respawn procedure, ~lines 441–444)

### Background

**#489:** After killing a pane on the DONE transition, the lead omits `tmux select-layout main-vertical`. The pane layout is not rebalanced. Section 3.1 mentions it in prose but it's easy to miss.

**#497 (lead side):** The lead transitions agents to DONE and kills their pane without first sending a `shutdown` message to the agent via chat. The agent's final protocol state is undefined — it may have already exited due to the pane kill, or it may post more messages. Sending `shutdown` before killing aligns the protocol state and is required by `use-engram-chat-as` shutdown protocol.

### Baseline Test (RED)

1. Read Section 3.1 state table, DONE row. Verify: the Lead Behavior column does NOT mention sending a `shutdown` chat message before killing the pane.
2. Read Section 3.3 respawn procedure step 1. Verify: `tmux select-layout main-vertical` is NOT present after the `kill-pane` call.

### Change A: Section 3.1 — DONE State Lead Behavior

Replace the DONE row in the state table:

```
| **DONE** | Agent posted `done` AND no incoming holds remain (or last hold just dissolved) | Kill pane by tracked ID: `tmux kill-pane -t <pane-id>`. Rebalance: `tmux select-layout main-vertical`. Remove from tracking. |
```

With:
```
| **DONE** | Agent posted `done` AND no incoming holds remain (or last hold just dissolved) | 1. Post `shutdown` to agent via chat (`type = "shutdown"`, `to = "<agent-name>"`). 2. Kill pane by tracked ID: `tmux kill-pane -t <pane-id>`. 3. Rebalance: `tmux select-layout main-vertical` (single-column mode only — see Section 2.4). 4. Remove from tracking. |
```

Also update the hard rule below the state table (line ~393):
```
**ALWAYS kill panes by tracked pane ID.** Never by window index or name. After killing, run `tmux select-layout main-vertical` to rebalance remaining panes.
```

Prepend to make it two hard rules:
```
**ALWAYS send `shutdown` to the agent via chat before killing its pane.** This aligns the agent's protocol state so it doesn't post stale messages after pane death.

**ALWAYS kill panes by tracked pane ID.** Never by window index or name. After killing, run `tmux select-layout main-vertical` to rebalance remaining panes (single-column mode only — see Section 2.4 for two-column mode).
```

### Change B: Section 3.5 Hold Release — Add select-layout and shutdown

In Section 3.5 "When a hold fires" steps 4–6, replace:
```
4. If no remaining holds → kill target pane → DONE
5. If remaining holds → target stays in PENDING-RELEASE
6. `tmux select-layout main-vertical` (if pane was killed)
```

With:
```
4. If no remaining holds → post `shutdown` to target via chat → kill target pane → DONE
5. If remaining holds → target stays in PENDING-RELEASE
6. `tmux select-layout main-vertical` after kill (single-column mode only — see Section 2.4)
```

### Change C: Section 3.3 Respawn — Add select-layout after kill

Replace:
```
1. Kill existing pane: `tmux kill-pane -t <pane-id> 2>/dev/null`
```

With:
```
1. Kill existing pane: `tmux kill-pane -t <pane-id> 2>/dev/null` then `tmux select-layout main-vertical` (single-column mode only — see Section 2.4).
```

### Pressure Test (GREEN)

Re-read Sections 3.1, 3.3, and 3.5. Verify:
- DONE transition sends `shutdown` before `kill-pane`
- `tmux select-layout main-vertical` appears in DONE (3.1), respawn (3.3), and hold release (3.5)
- All references include the "(single-column mode only)" qualifier to avoid confusing two-column mode
- Hard rule at bottom of 3.1 mentions both shutdown and select-layout

---

## Task 5 (Group E): Fix #493 — Section 6.2: Health Check Implementation Pattern

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (Section 6.2, ~lines 818–824)

### Background

Section 6.2 lists what to check every 2 minutes but provides no mechanism to trigger the check. The lead is not a daemon — it only runs code when woken by user input or a chat monitor Agent. Without a trigger, the health check never fires.

The fix: introduce a health-check trigger Agent (a persistent background loop that posts a trigger message to the chat file every 120 seconds). Because it writes to the chat file, it wakes the lead's chat monitor. The lead reads the trigger message in its normal chat processing loop and runs the health check.

Track `HEALTH_CHECK_TASK_ID` alongside `CHAT_MONITOR_TASK_ID`. Drain on shutdown.

### Baseline Test (RED)

Read Section 6.2. Verify: it contains NO implementation pattern (no background task, no trigger file, no timing mechanism). It describes what to check, not how to schedule it.

Expected: section ends with step 4 ("If engram-agent missed heartbeat...") and nothing more.

### Change: Add Implementation subsection to Section 6.2

Append after the existing Section 6.2 checklist:

```markdown
#### Implementation

The health check must fire even when no agent is posting. Use a **health-check trigger loop** — a persistent background Agent that posts a trigger message to the chat file every 120 seconds, waking the lead's chat monitor naturally.

**Start at session end (after posting `ready` in Section 1.6):**

Spawn a background Agent (`Agent` tool, `run_in_background: true`) with this task:
```
Health-check trigger loop.
CHAT_FILE: [full path — literal string]

Loop forever:
1. sleep 120
2. Derive current timestamp (ISO 8601)
3. Append to CHAT_FILE (with shlock):
   [[message]]
   from = "health-checker"
   to = "lead"
   thread = "health-check"
   type = "info"
   ts = "<timestamp>"
   text = "HEALTH_CHECK_TRIGGER"
4. Go back to step 1

Exit only when explicitly killed (pane closed or process signal).
```

Store the returned task ID as `HEALTH_CHECK_TASK_ID`.

**In the main chat processing loop**, after reading new lines from the chat file:

```python
# Check for health-check trigger in new messages:
if 'from = "health-checker"' in new_lines and "HEALTH_CHECK_TRIGGER" in new_lines:
    run_health_checks()   # execute Section 6.2 checklist items 1–4
```

**On shutdown (Section 3.4):** drain the health check task:
```python
if HEALTH_CHECK_TASK_ID:
    TaskOutput(task_id=HEALTH_CHECK_TASK_ID, block=False)  # drain
```

**Add `HEALTH_CHECK_TASK_ID` to Section 6.4 Rule 3 (drain on shutdown):**

In the list of tracked task IDs that must be drained on shutdown, add `HEALTH_CHECK_TASK_ID`.
```

### Pressure Test (GREEN)

Re-read Section 6.2. Verify:
- An "Implementation" subsection exists after the checklist
- It provides a background Agent task description for the trigger loop
- It shows how to detect the trigger in the main loop
- It instructs draining `HEALTH_CHECK_TASK_ID` on shutdown
- Section 6.4 Rule 3 (drain on shutdown) references `HEALTH_CHECK_TASK_ID`

---

## Self-Review

### Spec Coverage

| Issue | Fix | Task |
|-------|-----|------|
| #490 — blocking TaskOutput on engram-agent ready | `block=False` + polling loop | Task 1 |
| #498 — monitor relay latency / race window | Post-drain sweep in Sections 1.6 + 6.1 | Task 2 |
| #485 — no specific skill references in templates | Mapping table + updated templates | Task 3 |
| #497 (template) — agents act on `all` after done | "shutdown from lead" + "addressed to you" instruction | Task 3 |
| #497 (lead) — no shutdown sent before kill | Shutdown before kill-pane in DONE transition | Task 4 |
| #489 — missing select-layout after kill | select-layout in 3.1, 3.3, 3.5 | Task 4 |
| #493 — no health check mechanism | Trigger loop + detection in main loop | Task 5 |

All 6 issues (7 sub-fixes) accounted for.

### Placeholder Check

No TBD, TODO, or "similar to Task N" references. All code blocks are complete. All section/line references verified against current file content.

### Type/Name Consistency

- `HEALTH_CHECK_TASK_ID` introduced in Task 5; referenced in Task 5 drain instruction. Consistent.
- `block=False` used consistently in Task 1.
- Section numbers (1.5, 1.6, 2.2, 3.1, 3.3, 3.5, 6.1, 6.2, 6.4) all verified against current SKILL.md.
