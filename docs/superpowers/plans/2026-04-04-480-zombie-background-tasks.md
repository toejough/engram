# Fix Zombie Background Shell Tasks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent the engram-tmux-lead from accumulating orphaned background shell tasks (Claude Code "shells") by tracking background task IDs and draining them before replacement.

**Architecture:** All changes are to `skills/engram-tmux-lead/SKILL.md` only — no Go code. The fix introduces a `CHAT_FSWATCH_TASK_ID` variable tracking the active chat watcher background task. Before spawning a new fswatch, the lead drains the previous task via `TaskOutput(block:false)`. A new "Background Task Hygiene" callout documents the invariant. READY-check loops get a defensive drain instruction for the retry path.

**Tech Stack:** Markdown/shell skill documentation only.

---

## Root Cause Analysis

The lead's chat watch pattern:
1. Spawns `fswatch -1 "$CHAT_FILE"` as background task
2. Either (a) fswatch fires → process chat message → spawn new fswatch, OR (b) user types → process user message → spawn new fswatch

Zombie accumulation happens when the user types BEFORE the lead processes the fswatch notification. The lead processes user input first, spawns a new fswatch, and then the old completed task's notification arrives later — stale, unread, accumulating in Claude Code's background task queue.

After a session with N false-positive cycles: N orphaned completed tasks remain as "shells."

---

## Files

- Modify: `skills/engram-tmux-lead/SKILL.md`

---

### Task 1: Add `CHAT_FSWATCH_TASK_ID` to startup initialization (Section 1.3)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (around line 57)

- [ ] **Step 1: Locate the cursor initialization block**

The block ends with:
```bash
CURSOR=$(wc -l < "$CHAT_FILE")
```
This is in Section 1.3, around line 57.

- [ ] **Step 2: Add task registry variable after CURSOR**

Replace:
```bash
# Cursor: record the current end-of-file. ALL subsequent chat reads use
# tail -n +$((CURSOR + 1)) — NEVER grep the full file (matches old messages).
CURSOR=$(wc -l < "$CHAT_FILE")
```

With:
```bash
# Cursor: record the current end-of-file. ALL subsequent chat reads use
# tail -n +$((CURSOR + 1)) — NEVER grep the full file (matches old messages).
CURSOR=$(wc -l < "$CHAT_FILE")

# Background task registry: one active task per logical operation.
# Always drain (TaskOutput block:false) before replacing an entry.
CHAT_FSWATCH_TASK_ID=""  # task ID of the current chat watcher background task
```

- [ ] **Step 3: Verify edit**

```bash
grep -n "CHAT_FSWATCH_TASK_ID" skills/engram-tmux-lead/SKILL.md
```
Expected: at least 3 matches (initialization + Section 1.6 + Section 6.1 after later tasks).

- [ ] **Step 4: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(skills): initialize CHAT_FSWATCH_TASK_ID in lead startup"
```

---

### Task 2: Update Section 1.6 with drain-before-spawn pattern

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (around line 146–153)

Section 1.6 currently describes the chat watch loop in 4 bullet points without any task tracking. This is the primary zombie source.

- [ ] **Step 1: Locate Section 1.6 description**

```bash
grep -n "After each user interaction" skills/engram-tmux-lead/SKILL.md
```
Expected: one match around line 148.

- [ ] **Step 2: Replace the 4-step description with task-tracking version**

Replace:
```
**The lead does NOT enter the standard fswatch watch loop.** Unlike reactive agents that block on fswatch, the lead stays interactive — it must be available for user input at all times. Instead, the lead:

1. After each user interaction, starts a background `fswatch -1` on the chat file
2. If the fswatch fires (agent posted something), process the chat message — relay questions to the user, handle agent status updates, etc.
3. If the user types first, process the user message — parrot to chat, route to an agent
4. After processing either, start a new background fswatch

This means the lead processes chat messages opportunistically between user inputs, not as a blocking loop.
```

With:
```
**The lead does NOT enter the standard fswatch watch loop.** Unlike reactive agents that block on fswatch, the lead stays interactive — it must be available for user input at all times. Instead, the lead:

1. After each user interaction, **replace** the chat watcher background task (see drain-before-spawn pattern below)
2. If the fswatch fires (agent posted something), process the chat message — relay questions to the user, handle agent status updates, etc.
3. If the user types first, process the user message — parrot to chat, route to an agent
4. After processing either, replace the chat watcher (drain old → spawn new)

This means the lead processes chat messages opportunistically between user inputs, not as a blocking loop.

**HARD RULE: drain before spawn.** The lead must NEVER spawn a second fswatch while one is already running or has completed but not been drained. Unread completed tasks accumulate as zombie "shells" in Claude Code's background task queue. The replace pattern:

```python
# Before spawning a new chat watcher:
if CHAT_FSWATCH_TASK_ID:
    TaskOutput(task_id=CHAT_FSWATCH_TASK_ID, block=False)  # drain; discard output
# Spawn replacement
CHAT_FSWATCH_TASK_ID = <new background task id from run_in_background>
```
```

- [ ] **Step 3: Verify edit renders correctly**

```bash
grep -n "drain before spawn\|drain-before-spawn\|CHAT_FSWATCH_TASK_ID" skills/engram-tmux-lead/SKILL.md | head -20
```
Expected: matches in Section 1.6 area and the HARD RULE callout.

- [ ] **Step 4: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(skills): add drain-before-spawn pattern to chat watcher in Section 1.6"
```

---

### Task 3: Update Section 6.1 with code example

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (around line 442–449)

Section 6.1 currently says "Run the standard fswatch loop per `use-engram-chat-as` protocol" with no concrete code. Add the replace pattern here for discoverability.

- [ ] **Step 1: Locate Section 6.1**

```bash
grep -n "6.1 Chat Watch Loop" skills/engram-tmux-lead/SKILL.md
```
Expected: one match.

- [ ] **Step 2: Extend Section 6.1 with the replace pattern**

After the existing 3-item list in 6.1 (ending with "Handle pending agent messages before returning to user"), add:

```markdown
**Replace pattern for chat watcher (HARD RULE — prevents zombie tasks):**

```python
# Drain old watcher before spawning new one:
if CHAT_FSWATCH_TASK_ID:
    TaskOutput(task_id=CHAT_FSWATCH_TASK_ID, block=False)  # drain; discard output
# Spawn new watcher:
# run_in_background: true
# fswatch -1 "$CHAT_FILE"
CHAT_FSWATCH_TASK_ID = <task id from background task result>
```

Always do this — even if you processed user input rather than a chat notification. The previous watcher may have already fired and completed; draining it prevents it from queuing as a zombie.
```

- [ ] **Step 3: Verify**

```bash
grep -n "Drain old watcher" skills/engram-tmux-lead/SKILL.md
```
Expected: one match in Section 6.1.

- [ ] **Step 4: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(skills): add drain-before-spawn code example to Section 6.1"
```

---

### Task 4: Add defensive drain instruction for READY-check retries (Sections 1.4, 1.5, 2.1)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md`

The READY check loops in Sections 1.4, 1.5, and 2.1 run as background tasks. Normally, the lead awaits them before proceeding, so they are naturally drained. But if the loop times out (the agent never shows READY) and the lead decides to diagnose/retry, the old timed-out background task must be drained before spawning a new check.

- [ ] **Step 1: Locate Section 1.4's "When the background task completes" paragraph**

```bash
grep -n "When the background task completes, verify it printed" skills/engram-tmux-lead/SKILL.md
```
Expected: one match in Section 1.4 (around line 110).

- [ ] **Step 2: Add drain instruction after "verify it printed READY"**

After:
```
When the background task completes, verify it printed "READY". Then send the role prompt:
```

Add a new note:
```
**If READY was not printed (timeout):** Drain the task (it has already completed — the loop exited after 15 iterations). Do NOT spawn a second background READY check while the first is still tracked. Diagnose using `tmux capture-pane` before retrying. If you retry, the previous task is already complete and implicitly drained when you read its output.
```

- [ ] **Step 3: Locate Section 1.5's "When the background task completes" paragraph**

```bash
grep -n "When the background task completes, check its output" skills/engram-tmux-lead/SKILL.md
```
Expected: one match in Section 1.5 (around line 137).

- [ ] **Step 4: Add drain instruction after "check its output"**

After:
```
When the background task completes, check its output. If it did NOT print "ENGRAM-AGENT FOUND":
```

Add:
```
> **Note:** The background task is already complete at this point (whether FOUND or not). It is implicitly drained when you read its output. No zombie is created here. Only spawn a new READY check after fully reading the old task's output.
```

- [ ] **Step 5: Locate Section 2.1's READY check block**

```bash
grep -n "READY.*break" skills/engram-tmux-lead/SKILL.md
```
Expected: matches in Sections 1.4 and 2.1 (around lines 104 and 175).

- [ ] **Step 6: Add the same drain note after Section 2.1's "When the background task completes, verify it printed 'READY'"**

The text after the 2.1 loop reads:
```
When the background task completes, verify it printed "READY", then send the role prompt:
```

Add the same note as in Step 2:
```
**If READY was not printed (timeout):** The task has already completed. Read its output to drain it, then diagnose before retrying. Never spawn a parallel READY check while the previous one's output is unread.
```

- [ ] **Step 7: Verify all three sections have drain notes**

```bash
grep -n "drain\|Drain\|READY was not printed" skills/engram-tmux-lead/SKILL.md
```
Expected: at least 3 matches (one per section).

- [ ] **Step 8: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(skills): add drain-before-retry notes to READY check loops in 1.4, 1.5, 2.1"
```

---

### Task 5: Update shutdown sequence to drain tracked task IDs (Section 3.4)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (around line 324–340)

On shutdown, any in-flight background tasks must be drained to prevent them persisting into the next session's "shells" count.

- [ ] **Step 1: Locate Section 3.4 shutdown sequence**

```bash
grep -n "3.4 Shutdown" skills/engram-tmux-lead/SKILL.md
```
Expected: one match around line 323.

- [ ] **Step 2: Find the last numbered step in the sequence**

```bash
grep -n "Report session summary" skills/engram-tmux-lead/SKILL.md
```
Expected: one match (it's step 5 in the sequence).

- [ ] **Step 3: Add drain step after step 5**

Locate the line:
```
5. Report session summary to user (agents spawned, tasks completed, memories learned)
```

Add after it:
```
6. Drain all tracked background task IDs to prevent zombie "shells" persisting:
   - If `CHAT_FSWATCH_TASK_ID` is set: `TaskOutput(task_id=CHAT_FSWATCH_TASK_ID, block=False)`
   - Any other tracked background task IDs (READY checks in flight): drain them too
   - This ensures Claude Code's background task queue is empty when the session ends
```

- [ ] **Step 4: Verify**

```bash
grep -n "Drain all tracked background" skills/engram-tmux-lead/SKILL.md
```
Expected: one match in Section 3.4.

- [ ] **Step 5: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(skills): drain tracked background task IDs during shutdown"
```

---

### Task 6: Add Background Task Hygiene to Common Mistakes table

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md`

The skill doesn't have its own Common Mistakes table, but Section 6.3 has a cursor-based hard rule with a "WRONG/RIGHT" pattern. Add a similar reference in an appropriate place.

- [ ] **Step 1: Locate Section 6.3's HARD RULE**

```bash
grep -n "6.3 Cursor-Based Chat Reading" skills/engram-tmux-lead/SKILL.md
```
Expected: one match around line 458.

- [ ] **Step 2: Add a Section 6.5 for Background Task Hygiene**

After Section 6.4 (Unprompted Reporting), add:

```markdown
### 6.5 Background Task Hygiene (HARD RULE)

**NEVER let background tasks accumulate.** Each completed-but-unread background task appears as an open "shell" in Claude Code's status line. After a session with many false-positive wake cycles, this creates noise and confusion.

**Rules:**
1. **One chat watcher at a time.** `CHAT_FSWATCH_TASK_ID` holds the active watcher. Replace = drain old + spawn new.
2. **Drain on replace.** Before starting a new background task of the same logical type, always call `TaskOutput(task_id=old_id, block=False)` to drain the completed task.
3. **Drain on shutdown.** At session end (Section 3.4), drain all tracked task IDs.
4. **Read output before retrying.** If a background READY check times out, read its output (it has completed), then decide whether to retry.

```python
# WRONG — spawns new watcher without draining old one:
CHAT_FSWATCH_TASK_ID = run_background("fswatch -1 $CHAT_FILE")

# RIGHT — drain old, then spawn new:
if CHAT_FSWATCH_TASK_ID:
    TaskOutput(task_id=CHAT_FSWATCH_TASK_ID, block=False)
CHAT_FSWATCH_TASK_ID = run_background("fswatch -1 $CHAT_FILE")
```
```

- [ ] **Step 3: Verify**

```bash
grep -n "6.5 Background Task Hygiene" skills/engram-tmux-lead/SKILL.md
```
Expected: one match.

- [ ] **Step 4: Final verification — count all CHAT_FSWATCH_TASK_ID references**

```bash
grep -cn "CHAT_FSWATCH_TASK_ID" skills/engram-tmux-lead/SKILL.md
```
Expected: ≥5 (initialization in 1.3, drain reference in 1.6, code example in 6.1, hygiene section 6.5, shutdown in 3.4).

- [ ] **Step 5: Final commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(skills): add Background Task Hygiene section 6.5 with WRONG/RIGHT pattern"
```

---

## Self-Review

**Spec coverage:**
- ✅ Root cause addressed: drain-before-spawn in Section 1.6 and 6.1
- ✅ Startup initialization: CHAT_FSWATCH_TASK_ID in Section 1.3
- ✅ READY check defensive notes: Sections 1.4, 1.5, 2.1
- ✅ Shutdown drain: Section 3.4
- ✅ Discoverability: Section 6.5 with WRONG/RIGHT pattern

**Placeholder scan:** No TBDs. All replacement text is shown verbatim.

**Consistency:**
- `CHAT_FSWATCH_TASK_ID` — used consistently throughout
- `TaskOutput(block=False)` — consistent notation (pseudocode for Claude, not shell)
- All section references match actual section numbers in the current SKILL.md

**Scope:** Single file, no Go changes. All tasks are independent edits that compose cleanly.
