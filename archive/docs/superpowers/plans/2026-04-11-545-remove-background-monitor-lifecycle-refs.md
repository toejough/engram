# Remove Stale Background Monitor References — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove all stale Background Monitor Pattern references from `skills/use-engram-chat-as/SKILL.md` to match the stateless agent model established in #539.

**Architecture:** Three-phase TDD cycle using `superpowers:writing-skills` for all edits. Phase 1 writes a behavior baseline test that greps for the forbidden patterns (RED). Phase 2 applies the skill edits (GREEN). Phase 3 pressure-tests the updated lifecycle for internal consistency.

**Tech Stack:** Bash (test script), Markdown (skill doc), `superpowers:writing-skills` skill

**Spec:** `docs/superpowers/specs/2026-04-11-545-remove-background-monitor-lifecycle-refs-design.md`

---

## Phase 1 — Baseline Behavior Test (RED)

### Task 1: Write the behavior baseline test

**Files:**
- Create: `skills/use-engram-chat-as/test-lifecycle-refs.sh`

- [ ] **Step 1: Invoke superpowers:writing-skills skill**

Before editing any skill file, invoke `superpowers:writing-skills`. Follow it for all edits in this plan.

- [ ] **Step 2: Write the test script**

Create `skills/use-engram-chat-as/test-lifecycle-refs.sh` with these contents:

```bash
#!/usr/bin/env bash
# Behavior baseline test for issue #545
# Each check greps for a pattern that MUST NOT appear in the final skill.
# Returns 0 (pass) when all patterns are absent, 1 (fail) when any are found.

set -euo pipefail
SKILL="$(dirname "$0")/SKILL.md"
FAILED=0

check_absent() {
  local description="$1"
  local pattern="$2"
  if grep -qF "$pattern" "$SKILL"; then
    echo "FAIL: Found forbidden pattern: $description"
    echo "      Pattern: $pattern"
    FAILED=1
  else
    echo "PASS: $description"
  fi
}

check_absent \
  'Step 7 references Background Monitor Pattern' \
  'Background Monitor Pattern, above'

check_absent \
  'Step 7 says spawn background monitor Agent' \
  'Spawn background monitor Agent'

check_absent \
  'Step 8 info message says Monitor active' \
  'Initialization complete. Monitor active.'

check_absent \
  'Step 9 says wait for monitor Agent notification' \
  'Wait for monitor Agent notification'

check_absent \
  'Step 10 says Monitor Agent returns' \
  'Monitor Agent returns semantic event'

check_absent \
  'Step 13 says Go to step 9' \
  'Go to step 9 -- ALWAYS'

check_absent \
  'The watch only ends when section present' \
  'The watch only ends when:'

check_absent \
  'Ready Messages says spawning the monitor (line 423)' \
  'or spawning the monitor. Announcing'

check_absent \
  'Ready Messages says before its monitor is watching (line 428)' \
  'before its monitor is watching'

check_absent \
  'Ready Messages says before spawning the monitor (line 430)' \
  'before spawning the monitor and posting'

check_absent \
  'Shutdown Protocol says Exit the monitor Agent loop' \
  'Exit the monitor Agent loop.'

check_absent \
  'Compaction Recovery Step 6 says Re-enter the fswatch loop' \
  'Re-enter the fswatch loop.'

check_absent \
  'Compaction Recovery says Continue from step 9 of the Agent Lifecycle' \
  'Continue the lifecycle from step 9 of the Agent Lifecycle'

check_absent \
  'Compaction Recovery guard note says in your watch loop' \
  'in your watch loop:'

check_absent \
  'Compaction Recovery guard note says waiting for fswatch' \
  'while the agent is waiting for fswatch.'

check_absent \
  'Common Mistakes row: Poll with sleep 2 loop / fswatch' \
  '| Poll with `sleep 2` loop | Use `fswatch -1`'

check_absent \
  'Common Mistakes row: Run fswatch/wc/grep / background monitor Agent' \
  '| Run fswatch/wc/grep directly in main agent context | Use background monitor Agent'

check_absent \
  'Common Mistakes row: Always re-enter the fswatch after posting' \
  'Always re-enter the fswatch after posting'

check_absent \
  'Common Mistakes row: Watch for next assignment (conflicts with stateless)' \
  'Completing a task != dismissed. Watch for next assignment'

check_absent \
  'Common Mistakes row: Exit monitor Agent loop' \
  'Exit monitor Agent loop after completing in-flight work'

if [ "$FAILED" -eq 0 ]; then
  echo ""
  echo "All checks passed — no stale monitor references found."
  exit 0
else
  echo ""
  echo "Some checks failed — stale monitor references remain."
  exit 1
fi
```

- [ ] **Step 3: Make the script executable**

```bash
chmod +x skills/use-engram-chat-as/test-lifecycle-refs.sh
```

- [ ] **Step 4: Run the test and confirm RED**

```bash
bash skills/use-engram-chat-as/test-lifecycle-refs.sh
```

Expected output: Multiple `FAIL:` lines, followed by:
```
Some checks failed — stale monitor references remain.
```
Exit code: 1

If any checks unexpectedly PASS, re-read the grep pattern against the actual file content (the patterns must match exactly). Do NOT proceed until the test fails on at least the step-7 check.

- [ ] **Step 5: Commit the RED test**

```bash
git add skills/use-engram-chat-as/test-lifecycle-refs.sh
git commit -m "test(skills): add RED lifecycle-refs behavior test for #545

Checks for 20 forbidden patterns in use-engram-chat-as/SKILL.md
that reference the removed Background Monitor Pattern. All checks
currently fail (RED).

AI-Used: [claude]"
```

---

## Phase 2 — Update the Skill (GREEN)

Use `superpowers:writing-skills` for all edits in this phase. Invoke it at the start of the phase if not already active.

### Task 2: Rewrite Agent Lifecycle steps 3, 7-13

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md` (Agent Lifecycle section, lines ~486-516)

- [ ] **Step 1: Read the current Agent Lifecycle section**

Read lines 486-520 of `skills/use-engram-chat-as/SKILL.md` to confirm current content before editing.

- [ ] **Step 2: Replace the lifecycle numbered list**

Replace this exact block:

```
1. Derive chat file path from $PWD
2. Create chat directory if needed
3. Initialize cursor: CURSOR=$(engram chat cursor) — BEFORE posting ready so the monitor
   captures any work routed by lead between your ready message and monitor startup
4. Post ready message and capture cursor:
   CURSOR=$(engram chat post --from <name> --to all --thread lifecycle --type ready --text "...")
   Initial cursor = the integer returned by this command. No separate wc -l needed.
5. Read last 20 messages to catch up (read further back if needed)
6. Load resources (memories, configs, etc.)
7. Spawn background monitor Agent (Background Monitor Pattern, above) using CURSOR from step 3
8. Post info: "Initialization complete. Monitor active." — signals lead that agent is operational
9. Wait for monitor Agent notification
10. Monitor Agent returns semantic event -> process event if addressed to you
11. If acting:
    a. PRE_CURSOR=$(engram chat cursor)   # BEFORE posting intent
    b. Post intent to (engram-agent + any other relevant recipients)
    c. RESULT=$(engram chat ack-wait --agent <name> --cursor $PRE_CURSOR --recipients <names> --max-wait 30)
       Parse: RESULT_TYPE / FROM / CURSOR / TEXT (see ACK-Wait Protocol)
    d. If ACK: proceed. If WAIT: engage argument protocol. If TIMEOUT: escalate to lead.
    e. Act
    f. Pre-done cursor-check: spawn background Agent to tail CHAT_FILE from cursor, grep for unresolved WAITs
       If any WAIT addressed to you and unresolved: engage before posting done
    g. Post result
12. Post response (with lock)
13. Go to step 9 -- ALWAYS. Even after completing a task.
```

With this replacement:

```
1. Derive chat file path from $PWD
2. Create chat directory if needed
3. Initialize cursor: CURSOR=$(engram chat cursor) — BEFORE posting ready
4. Post ready message and capture cursor:
   CURSOR=$(engram chat post --from <name> --to all --thread lifecycle --type ready --text "...")
   Initial cursor = the integer returned by this command. No separate wc -l needed.
5. Read last 20 messages to catch up (read further back if needed)
6. Load resources (memories, configs, etc.)
7. Post info: "Initialization complete." — signals lead that agent is operational
8. Process assigned work (delivered at invocation time by lead or user)
9. If acting:
    a. PRE_CURSOR=$(engram chat cursor)   # BEFORE posting intent
    b. Post intent to (engram-agent + any other relevant recipients)
    c. RESULT=$(engram chat ack-wait --agent <name> --cursor $PRE_CURSOR --recipients <names> --max-wait 30)
       Parse: RESULT_TYPE / FROM / CURSOR / TEXT (see ACK-Wait Protocol)
    d. If ACK: proceed. If WAIT: engage argument protocol. If TIMEOUT: escalate to lead.
    e. Act
    f. Pre-done cursor-check: spawn background Agent to tail CHAT_FILE from cursor, grep for unresolved WAITs
       If any WAIT addressed to you and unresolved: engage before posting done
    g. Post result
10. Post result (info or done)
11. Exit — lead or binary handles re-invocation for subsequent tasks
```

- [ ] **Step 3: Remove the "The watch only ends when" section**

Locate and delete this block (appears immediately after the lifecycle numbered list):

```markdown
**The watch only ends when:**
- You receive a `shutdown` message addressed to you (or `all`)
- The user explicitly dismisses you
```

- [ ] **Step 4: Run the test, confirm partial progress**

```bash
bash skills/use-engram-chat-as/test-lifecycle-refs.sh
```

Expected: checks 1-7 now PASS (the lifecycle-related ones). Checks 8-20 still FAIL.
If any of 1-7 still fail, re-read the file to confirm the edit was applied correctly.

---

### Task 3: Update Ready Messages section

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md` (Ready Messages section, ~lines 422-430)

- [ ] **Step 1: Read the Ready Messages semantics block**

Read lines 420-432 of `skills/use-engram-chat-as/SKILL.md`.

- [ ] **Step 2: Update line 423 — "spawning the monitor"**

Replace:
```
- Posted **once**, as the agent's **first action** after deriving the chat file path — before reading history, loading resources, or spawning the monitor. Announcing presence early prevents observers from mistaking initialization silence for a dead agent.
```

With:
```
- Posted **once**, as the agent's **first action** after deriving the chat file path — before reading history, loading resources, or doing other initialization. Announcing presence early prevents observers from mistaking initialization silence for a dead agent.
```

- [ ] **Step 3: Update line 428 — "before its monitor is watching"**

Replace:
```
- **Lead setup:** The lead waits for the agent's "initialization complete" `info` message before routing work (30s timeout from that message, not from `ready`). The initial `ready` message only announces presence — the agent may still be reading history. Routing work before the init-complete signal risks the agent processing the assignment before its monitor is watching.
```

With:
```
- **Lead setup:** The lead waits for the agent's "initialization complete" `info` message before routing work (30s timeout from that message, not from `ready`). The initial `ready` message only announces presence — the agent may still be reading history. Routing work before the init-complete signal risks the agent processing the assignment before it is fully operational.
```

- [ ] **Step 4: Update line 430 — "before spawning the monitor and posting"**

Replace:
```
- **Late joiners:** Post `ready` first to announce presence, then read full history before spawning the monitor and posting the init-complete `info`.
```

With:
```
- **Late joiners:** Post `ready` first to announce presence, then read full history before posting the init-complete `info`.
```

- [ ] **Step 5: Run the test, confirm partial progress**

```bash
bash skills/use-engram-chat-as/test-lifecycle-refs.sh
```

Expected: checks 1-10 now PASS. Checks 11-20 still FAIL.

---

### Task 4: Update Shutdown Protocol

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md` (Shutdown Protocol, ~line 465)

- [ ] **Step 1: Read the Agent Shutdown Behavior block**

Read lines 450-470 of `skills/use-engram-chat-as/SKILL.md`.

- [ ] **Step 2: Update step 4 of the shutdown behavior**

Replace:
```
4. **Exit the monitor Agent loop.** Do not spawn a new monitor Agent. The agent's turn is complete.
```

With:
```
4. **Exit.** Complete in-flight work and exit. There is no persistent loop to maintain. The agent's turn is complete.
```

- [ ] **Step 3: Run the test, confirm partial progress**

```bash
bash skills/use-engram-chat-as/test-lifecycle-refs.sh
```

Expected: checks 1-11 now PASS. Checks 12-20 still FAIL.

---

### Task 5: Update Compaction Recovery section

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md` (Compaction Recovery section, ~lines 518-625)

- [ ] **Step 1: Read the Compaction Recovery section**

Read lines 518-625 of `skills/use-engram-chat-as/SKILL.md`.

- [ ] **Step 2: Update the cursor guard note (~line 536)**

Replace:
```
Add this guard before every `tail -n +$((CURSOR + 1))` call in your watch loop:
```

With:
```
Add this guard before every `tail -n +$((CURSOR + 1))` call in your cursor-dependent operations:
```

- [ ] **Step 3: Update Compaction Recovery Step 6 heading and body (~lines 609-611)**

Replace:
```
**Step 6: Re-enter the fswatch loop.**

Continue the lifecycle from step 9 of the Agent Lifecycle. Do not re-post a `ready` message — `info` is sufficient.
```

With:
```
**Step 6: Resume task processing.**

Continue the lifecycle from step 8 of the Agent Lifecycle (process assigned work). Do not re-post a `ready` message — `info` is sufficient.
```

- [ ] **Step 4: Update the "Critical: Guard Every Cursor Use" note (~line 615)**

Replace:
```
The compaction check must run **before every tail call**, not just at startup. A compaction can occur mid-task while the agent is waiting for fswatch.
```

With:
```
The compaction check must run **before every tail call**, not just at startup. A compaction can occur mid-task while the agent is processing a task.
```

- [ ] **Step 5: Run the test, confirm partial progress**

```bash
bash skills/use-engram-chat-as/test-lifecycle-refs.sh
```

Expected: checks 1-15 now PASS. Checks 16-20 still FAIL.

---

### Task 6: Update Common Mistakes table

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md` (Common Mistakes table, ~lines 628-660)

- [ ] **Step 1: Read the Common Mistakes table**

Read lines 626-665 of `skills/use-engram-chat-as/SKILL.md`.

- [ ] **Step 2: Remove the "Poll with sleep 2 loop" row**

Remove this entire row from the table:
```
| Poll with `sleep 2` loop | Use `fswatch -1` / `inotifywait` -- true kernel block (inside monitoring Agent) |
```

- [ ] **Step 3: Remove the "Run fswatch/wc/grep" row**

Remove this entire row from the table:
```
| Run fswatch/wc/grep directly in main agent context | Use background monitor Agent — bash monitoring in main context produces visible tool-call noise |
```

- [ ] **Step 4: Update the "Post a message then stop" row**

Replace:
```
| Post a message then stop | Always re-enter the fswatch after posting |
```

With:
```
| Exit before posting `done` | Always post `done` when your assigned task is complete before exiting |
```

- [ ] **Step 5: Remove the "Stop after task completion" row**

Remove this entire row from the table (it conflicts with the stateless model where agents DO exit after task completion):
```
| Stop after task completion | Completing a task != dismissed. Watch for next assignment |
```

- [ ] **Step 6: Update the "Ignore shutdown message" row**

Replace:
```
| Ignore `shutdown` message | Exit monitor Agent loop after completing in-flight work and posting `done` |
```

With:
```
| Ignore `shutdown` message | Post `done` and exit when you receive a `shutdown` message addressed to you or `all` |
```

- [ ] **Step 7: Run the test, confirm full GREEN**

```bash
bash skills/use-engram-chat-as/test-lifecycle-refs.sh
```

Expected output:
```
PASS: Step 7 references Background Monitor Pattern
PASS: Step 7 says spawn background monitor Agent
PASS: Step 8 info message says Monitor active
PASS: Step 9 says wait for monitor Agent notification
PASS: Step 10 says Monitor Agent returns
PASS: Step 13 says Go to step 9
PASS: The watch only ends when section present
PASS: Ready Messages says spawning the monitor (line 423)
PASS: Ready Messages says before its monitor is watching (line 428)
PASS: Ready Messages says before spawning the monitor (line 430)
PASS: Shutdown Protocol says Exit the monitor Agent loop
PASS: Compaction Recovery Step 6 says Re-enter the fswatch loop
PASS: Compaction Recovery says Continue from step 9 of the Agent Lifecycle
PASS: Compaction Recovery guard note says in your watch loop
PASS: Compaction Recovery guard note says waiting for fswatch
PASS: Common Mistakes row: Poll with sleep 2 loop / fswatch
PASS: Common Mistakes row: Run fswatch/wc/grep / background monitor Agent
PASS: Common Mistakes row: Always re-enter the fswatch after posting
PASS: Common Mistakes row: Watch for next assignment (conflicts with stateless)
PASS: Common Mistakes row: Exit monitor Agent loop

All checks passed — no stale monitor references found.
```
Exit code: 0

- [ ] **Step 8: Commit the GREEN skill edits**

```bash
git add skills/use-engram-chat-as/SKILL.md
git commit -m "fix(skills): remove stale Background Monitor refs from use-engram-chat-as (#545)

Rewrite Agent Lifecycle steps 7-13 to reflect the stateless agent
model: initialize, post init-complete, process one task, exit.
Non-lead agents do not run persistent watch loops.

Also update Ready Messages, Shutdown Protocol, Compaction Recovery,
and Common Mistakes table to remove all references to the removed
Background Monitor Pattern and fswatch loop.

Finishes cleanup started in 83dbe68 (#539).

AI-Used: [claude]"
```

---

## Phase 3 — Verify and Refine (Pressure Test)

### Task 7: End-to-end lifecycle consistency check

**Files:**
- Read: `skills/use-engram-chat-as/SKILL.md`

- [ ] **Step 1: Read the full Agent Lifecycle section**

Read lines 471-520 of `skills/use-engram-chat-as/SKILL.md`.

Verify:
- Steps are numbered 1-11 (old 13 steps → new 11 steps)
- Step 7 says "Post info: 'Initialization complete.'" — no "Monitor active."
- Step 8 says "Process assigned work (delivered at invocation time...)"
- Step 9 contains the intact intent protocol (steps a-g) including the one-shot cursor-check in step 9f
- Step 10 says "Post result (info or done)"
- Step 11 says "Exit — lead or binary handles re-invocation..."
- "The watch only ends when:" block is gone

- [ ] **Step 2: Verify Compaction Recovery step reference**

Read lines 608-615 of `skills/use-engram-chat-as/SKILL.md`.

Verify:
- Step 6 says "Resume task processing."
- Body says "step 8" (not "step 9") — matches the new step numbering

- [ ] **Step 3: Verify Common Mistakes table is self-consistent**

Read lines 626-665 of `skills/use-engram-chat-as/SKILL.md`.

Verify:
- No row says "re-enter the fswatch"
- No row says "Watch for next assignment"
- "Ignore shutdown message" row says "Post `done` and exit" (no mention of monitor loop)
- "Exit before posting `done`" row is present (new row from step 6.4)

- [ ] **Step 4: Verify no new broken cross-references**

Run:
```bash
grep -n "step 9\b\|step 11\b\|step 13\b" skills/use-engram-chat-as/SKILL.md
```

Any matches should be checked manually — the old step 9 (monitor wait) and step 13 (loop-back) were renumbered. If any reference still uses the old numbers in a way that points to wrong behavior, fix inline.

- [ ] **Step 5: Run targ check-full**

```bash
targ check-full
```

Expected: no failures. If any failures appear, fix them before proceeding.

- [ ] **Step 6: Commit the test script as permanent fixture**

The test script is a permanent guard against future regressions. Keep it in the repo:

```bash
git add skills/use-engram-chat-as/test-lifecycle-refs.sh
git commit -m "test(skills): retain lifecycle-refs behavior test as regression guard (#545)

AI-Used: [claude]"
```

Note: If the test script was already committed in Phase 1 Task 1, skip this step.

---

## Done

After Task 7 completes:
1. All 20 behavior checks pass (GREEN)
2. The Agent Lifecycle reads as a coherent stateless model
3. Cross-references (Compaction Recovery step number, Common Mistakes) are consistent
4. No stale Background Monitor Pattern references remain
5. `targ check-full` passes
