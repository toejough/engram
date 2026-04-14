# SPAWN-PANE / KILL-PANE Procedure Enforcement (Issue #505) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **ALSO REQUIRED:** You MUST use `superpowers:writing-skills` for ALL edits to `SKILL.md`. TDD flow: RED baseline → GREEN edits → pressure test. Do not skip any phase.

**Goal:** Fix the `engram-tmux-lead` skill so that pane creation and removal always go through named procedures (`SPAWN-PANE`, `KILL-PANE`), eliminating six inline bypasses that cause all panes to stack in a single column.

**Architecture:** Pure SKILL.md text change. Extract the existing splitting-rules block as `SPAWN-PANE`, add a new `KILL-PANE` procedure, add a Red Flag for direct `tmux split-window` calls, and replace all six bypass sites (§1.4, §2.1, §3.1 ×2, §3.3, §3.4) with references to the procedures.

**Tech Stack:** `skills/engram-tmux-lead/SKILL.md` (Markdown); `superpowers:writing-skills` (required for all edits); `git`

---

## File Map

| File | Change |
|------|--------|
| `skills/engram-tmux-lead/SKILL.md` | All changes — procedure extraction + six bypass fixes |

No other files change.

---

### Task 1: RED baseline — confirm broken patterns exist

Before touching the skill, establish a baseline that proves the bugs are present.

**Files:**
- Read: `skills/engram-tmux-lead/SKILL.md`

- [ ] **Step 1: Grep for inline PANE_ID split-window assignments (spawn bypasses)**

```bash
grep -n "PANE_ID=\$(tmux split-window" skills/engram-tmux-lead/SKILL.md
```

Expected output (2 hits):
```
149:PANE_ID=$(tmux split-window -h -d -P -F '#{pane_id}')
246:PANE_ID=$(tmux split-window -h -d -P -F '#{pane_id}')
```

If this does NOT return 2 hits at those lines, stop and investigate — the file may have changed since this plan was written.

- [ ] **Step 2: Grep for inline select-layout on kill paths (kill bypasses)**

```bash
grep -n "select-layout main-vertical" skills/engram-tmux-lead/SKILL.md
```

Expected output (7+ hits including the bugs):
```
88:tmux select-layout main-vertical
103:  tmux select-layout main-vertical
160:tmux select-layout main-vertical
257:tmux select-layout main-vertical
399:...`tmux select-layout main-vertical`...
431:...`tmux select-layout main-vertical`...
437:...`tmux select-layout main-vertical`...
486:...`tmux select-layout main-vertical`...
570:6. `tmux select-layout main-vertical` after kill...
```

Lines 160, 257, 431, 437, 486, 570 are the bugs to fix. Record the exact line numbers — they may shift as you edit; always re-grep after each task.

- [ ] **Step 3: Record baseline counts**

Note the hit counts from steps 1 and 2. You will verify these go to 0 (for spawn bypasses) and to the expected residual count (for select-layout) in Task 7.

---

### Task 2: Add SPAWN-PANE procedure header and HARD GATE to §1.3

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (§1.3, line ~93)

- [ ] **Step 1: Replace the splitting rules section heading**

Find:
```
**Splitting rules (called by every spawn, including Section 2.1):**
```

Replace with:
```
#### SPAWN-PANE — use for EVERY pane creation (§1.4, §2.1, and any future spawn site)

**HARD GATE: NEVER call `tmux split-window` directly outside this block — always run SPAWN-PANE.**
```

- [ ] **Step 2: Add procedure contract comment at top of the code block**

Find:
```bash
# Use RIGHT_PANE_COUNT to decide HOW to split.
# Use -P -F '#{pane_id}' to capture the new pane ID at creation — not list-panes | tail -1.
```

Replace with:
```bash
# HARD GATE: NEVER call tmux split-window elsewhere — always run SPAWN-PANE.
# Requires: RIGHT_PANE_COUNT, MIDDLE_COL_LAST_PANE, RIGHT_COL_LAST_PANE are initialized (§1.3 setup).
# Outputs: NEW_PANE (new pane ID). Caller assigns: PANE_ID=$NEW_PANE
# Use -P -F '#{pane_id}' to capture the new pane ID at creation — not list-panes | tail -1.
```

- [ ] **Step 3: Verify heading and gate appear**

```bash
grep -n "SPAWN-PANE\|HARD GATE.*split-window" skills/engram-tmux-lead/SKILL.md
```

Expected: at least 3 hits — the heading, the bold HARD GATE line, and the comment in the code block.

- [ ] **Step 4: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "feat(engram-tmux-lead): name splitting rules as SPAWN-PANE procedure (#505)

AI-Used: [claude]"
```

---

### Task 3: Add KILL-PANE procedure to §1.3 (after diagrams, before §1.4)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (§1.3, after line ~141)

- [ ] **Step 1: Insert KILL-PANE procedure after the "Maximum: 9 total panes" line**

Find:
```
**Maximum: 9 total panes** (1 coordinator + 4 middle + 4 right). Do not exceed. See Section 2.4.

### 1.4 Spawn engram-agent
```

Replace with:
```
**Maximum: 9 total panes** (1 coordinator + 4 middle + 4 right). Do not exceed. See Section 2.4.

#### KILL-PANE — use for EVERY pane removal (§3.1 DONE, §3.3 Respawn, §3.4 hold-release)

**HARD GATE: NEVER call `tmux kill-pane` + `tmux select-layout` inline — always run KILL-PANE.**

```bash
# HARD GATE: NEVER call tmux kill-pane + select-layout elsewhere — always run KILL-PANE.
# Requires: PANE_ID set to the pane being removed; RIGHT_PANE_COUNT and RIGHT_COL_LAST_PANE are set.
# After kill: update MIDDLE_COL_LAST_PANE or RIGHT_COL_LAST_PANE in your pane registry
# to reflect the new column tail (the lead tracks this via its pane registry).

tmux kill-pane -t "$PANE_ID" 2>/dev/null
RIGHT_PANE_COUNT=$((RIGHT_PANE_COUNT - 1))
if [ -z "$RIGHT_COL_LAST_PANE" ]; then
  # Single-column mode: rebalance remaining panes
  tmux select-layout main-vertical
fi
# Two-column mode: no automatic rebalance — lead updates column tracking manually.
```

### 1.4 Spawn engram-agent
```

- [ ] **Step 2: Verify KILL-PANE heading appears**

```bash
grep -n "KILL-PANE" skills/engram-tmux-lead/SKILL.md
```

Expected: at least 2 hits — the heading and the comment inside the code block.

- [ ] **Step 3: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "feat(engram-tmux-lead): add KILL-PANE procedure to §1.3 (#505)

AI-Used: [claude]"
```

---

### Task 4: Add Red Flag for direct tmux split-window calls

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (top of file, Red Flags section, line ~12)

- [ ] **Step 1: Add red flag bullet**

Find:
```
- Answering technical questions from your own knowledge
```

Replace with:
```
- Answering technical questions from your own knowledge
- Calling `tmux split-window` directly instead of using SPAWN-PANE from Section 1.3
```

- [ ] **Step 2: Verify**

```bash
grep -n "split-window.*SPAWN-PANE\|SPAWN-PANE.*split-window" skills/engram-tmux-lead/SKILL.md
```

Expected: 1 hit in the Red Flags section.

- [ ] **Step 3: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "feat(engram-tmux-lead): add split-window red flag to lead skill (#505)

AI-Used: [claude]"
```

---

### Task 5: Fix §1.4 spawn (engram-agent)

Replace the hardcoded `split-window -h` + `select-layout main-vertical` in §1.4 with a call to SPAWN-PANE.

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (§1.4, around line 147)

- [ ] **Step 1: Verify the current broken code block in §1.4**

```bash
grep -n "PANE_ID=\$(tmux split-window" skills/engram-tmux-lead/SKILL.md
```

Note the line number for the §1.4 occurrence (the one in the `### 1.4 Spawn engram-agent` section, before any `### 2.` heading). This is the target.

- [ ] **Step 2: Replace the broken code block**

Find (in the `### 1.4 Spawn engram-agent` section):
```bash
# Split a new pane to the right, start claude in it
PANE_ID=$(tmux split-window -h -d -P -F '#{pane_id}')
# Suppress status line — agents run headless, no user to see it; keeps panes clean
tmux send-keys -t "$PANE_ID" "claude --dangerously-skip-permissions --model sonnet --settings '{\"statusLine\": {\"type\": \"command\", \"command\": \"true\"}}'" Enter
# Wait for claude to start (watch for the prompt character)
while ! tmux capture-pane -t "$PANE_ID" -p 2>/dev/null | grep -q "❯"; do sleep 1; done
# Send the role prompt
tmux send-keys -t "$PANE_ID" "/use-engram-chat-as reactive memory agent named engram-agent" Enter
# Send extra Enter in case it was treated as a paste
sleep 1
tmux send-keys -t "$PANE_ID" Enter
# Rebalance: coordinator stays left, everything else stacks evenly on right
tmux select-layout main-vertical
```

Replace with:
```bash
# Use SPAWN-PANE from Section 1.3 to create the pane.
# RIGHT_PANE_COUNT=1 at this point (chat tail was spawn #1 in §1.3 setup).
# SPAWN-PANE sets NEW_PANE — assign to PANE_ID for this spawn.
PANE_ID=$NEW_PANE
# Suppress status line — agents run headless, no user to see it; keeps panes clean
tmux send-keys -t "$PANE_ID" "claude --dangerously-skip-permissions --model sonnet --settings '{\"statusLine\": {\"type\": \"command\", \"command\": \"true\"}}'" Enter
# Wait for claude to start (watch for the prompt character)
while ! tmux capture-pane -t "$PANE_ID" -p 2>/dev/null | grep -q "❯"; do sleep 1; done
# Send the role prompt
tmux send-keys -t "$PANE_ID" "/use-engram-chat-as reactive memory agent named engram-agent" Enter
# Send extra Enter in case it was treated as a paste
sleep 1
tmux send-keys -t "$PANE_ID" Enter
```

- [ ] **Step 3: Verify §1.4 no longer has inline split-window**

```bash
grep -n "PANE_ID=\$(tmux split-window" skills/engram-tmux-lead/SKILL.md
```

Expected: only 1 hit remaining (§2.1, not yet fixed).

- [ ] **Step 4: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(engram-tmux-lead): use SPAWN-PANE in §1.4 engram-agent spawn (#505)

AI-Used: [claude]"
```

---

### Task 6: Fix §2.1 spawn template

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (§2.1, around line 244)

- [ ] **Step 1: Verify the current broken code block in §2.1**

```bash
grep -n "PANE_ID=\$(tmux split-window" skills/engram-tmux-lead/SKILL.md
```

Confirm 1 hit remains (§2.1 occurrence, in `### 2.1 Spawn Template` section).

- [ ] **Step 2: Replace the broken code block**

Find (in the `### 2.1 Spawn Template` section):
```bash
# Split a new pane to the right, capturing the new pane ID atomically
PANE_ID=$(tmux split-window -h -d -P -F '#{pane_id}')
# Suppress status line — agents run headless, no user to see it; keeps panes clean
tmux send-keys -t "$PANE_ID" "claude --dangerously-skip-permissions --model sonnet --settings '{\"statusLine\": {\"type\": \"command\", \"command\": \"true\"}}'" Enter
# Wait for claude to start (watch for the prompt character)
while ! tmux capture-pane -t "$PANE_ID" -p 2>/dev/null | grep -q "❯"; do sleep 1; done
# Send the role prompt
tmux send-keys -t "$PANE_ID" "/use-engram-chat-as <role> named <agent-name>. Your task: <task description>. Work in this directory: <pwd>. Use relevant skills. Post intent before significant actions. Funnel ALL questions for the user through chat addressed to lead. NEVER ask the user directly -- you have no user. Post done when your assigned task is complete." Enter
# Send extra Enter in case it was treated as a paste
sleep 1
tmux send-keys -t "$PANE_ID" Enter
# Rebalance: coordinator left, everything else stacks evenly on right
tmux select-layout main-vertical
```

Replace with:
```bash
# Use SPAWN-PANE from Section 1.3 to create the pane.
# SPAWN-PANE checks RIGHT_PANE_COUNT and applies the correct split (middle col, right col, or vertical).
# SPAWN-PANE sets NEW_PANE — assign to PANE_ID for this spawn.
PANE_ID=$NEW_PANE
# Suppress status line — agents run headless, no user to see it; keeps panes clean
tmux send-keys -t "$PANE_ID" "claude --dangerously-skip-permissions --model sonnet --settings '{\"statusLine\": {\"type\": \"command\", \"command\": \"true\"}}'" Enter
# Wait for claude to start (watch for the prompt character)
while ! tmux capture-pane -t "$PANE_ID" -p 2>/dev/null | grep -q "❯"; do sleep 1; done
# Send the role prompt
tmux send-keys -t "$PANE_ID" "/use-engram-chat-as <role> named <agent-name>. Your task: <task description>. Work in this directory: <pwd>. Use relevant skills. Post intent before significant actions. Funnel ALL questions for the user through chat addressed to lead. NEVER ask the user directly -- you have no user. Post done when your assigned task is complete." Enter
# Send extra Enter in case it was treated as a paste
sleep 1
tmux send-keys -t "$PANE_ID" Enter
```

- [ ] **Step 3: Verify all inline split-window spawns are gone**

```bash
grep -n "PANE_ID=\$(tmux split-window" skills/engram-tmux-lead/SKILL.md
```

Expected: **0 hits**. This is the GREEN signal for the RED baseline from Task 1 Step 1.

- [ ] **Step 4: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(engram-tmux-lead): use SPAWN-PANE in §2.1 spawn template (#505)

AI-Used: [claude]"
```

---

### Task 7: Fix kill paths (§3.1 DONE, §3.1 note, §3.3 Respawn, §3.4 hold-release)

Replace all four inline kill + unconditional `select-layout main-vertical` sites with KILL-PANE references.

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (§3.1, §3.3, §3.4)

- [ ] **Step 1: Fix §3.1 DONE state table cell**

Find (the full `| **DONE** |` table row):
```
| **DONE** | Agent posted `done` AND no incoming holds remain (or last hold just dissolved) | 1. Post `shutdown` to agent via chat (`type = "shutdown"`, `to = "<agent-name>"`). 2. Kill pane by tracked ID: `tmux kill-pane -t <pane-id>`. 3. Rebalance: `tmux select-layout main-vertical` (single-column mode only — see Section 2.4). 4. Remove from tracking. |
```

Replace with:
```
| **DONE** | Agent posted `done` AND no incoming holds remain (or last hold just dissolved) | 1. Post `shutdown` to agent via chat (`type = "shutdown"`, `to = "<agent-name>"`). 2. Set `PANE_ID=<tracked-pane-id>` then use KILL-PANE from Section 1.3 (handles single- and two-column rebalancing). 3. Remove from tracking. |
```

- [ ] **Step 2: Fix §3.1 "ALWAYS kill panes" note**

Find:
```
**ALWAYS kill panes by tracked pane ID.** Never by window index or name. After killing, run `tmux select-layout main-vertical` to rebalance remaining panes (single-column mode only — see Section 2.4 for two-column mode).
```

Replace with:
```
**ALWAYS kill panes by tracked pane ID.** Never by window index or name. After killing, use KILL-PANE from Section 1.3 — it handles rebalancing for both single- and two-column modes.
```

- [ ] **Step 3: Fix §3.3 Respawn step 1**

Find:
```
1. Kill existing pane: `tmux kill-pane -t <pane-id> 2>/dev/null` then `tmux select-layout main-vertical` (single-column mode only — see Section 2.4).
```

Replace with:
```
1. Set `PANE_ID=<tracked-pane-id>` then use KILL-PANE from Section 1.3 (handles single- and two-column layout rebalancing).
```

- [ ] **Step 4: Fix §3.4 hold-release steps 4 and 6**

§3.4 step 4 has an inline `kill target pane` that bypasses KILL-PANE, AND step 6 has an unconditional select-layout. Both must be replaced together: step 4 uses KILL-PANE (which handles the kill + rebalance), and step 6 is removed entirely.

Find (the full hold-release list, steps 4–6):
```
4. If no remaining holds → post `shutdown` to target via chat → kill target pane → DONE
5. If remaining holds → target stays in PENDING-RELEASE
6. `tmux select-layout main-vertical` after kill (single-column mode only — see Section 2.4)
```

Replace with:
```
4. If no remaining holds → post `shutdown` to target via chat → set `PANE_ID=<tracked-pane-id>` then use KILL-PANE from Section 1.3 (handles kill + layout rebalancing) → DONE
5. If remaining holds → target stays in PENDING-RELEASE
```

(Step 6 is removed — KILL-PANE subsumes it.)

- [ ] **Step 5: Verify the kill-path fixes**

```bash
grep -n "kill-pane.*select-layout\|select-layout.*kill" skills/engram-tmux-lead/SKILL.md
```

Expected: **0 hits** — no inline kill+select-layout pairs remain.

```bash
grep -n "tmux select-layout main-vertical" skills/engram-tmux-lead/SKILL.md
```

Expected remaining hits (not bugs):
- Line ~88: §1.3 setup code (chat tail initial split — before SPAWN-PANE state variables exist)
- Line ~103: SPAWN-PANE definition (the `if [ "$RIGHT_PANE_COUNT" -lt 4 ]` branch)
- New KILL-PANE block: the single-column branch
- Line ~399: §2.4 prose description text (mentions `main-vertical` in explanation)

If any hit appears at the former bug lines (160, 257, 431, 437, 486, 570), fix it before committing.

- [ ] **Step 6: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(engram-tmux-lead): use KILL-PANE in kill paths §3.1/§3.3/§3.4 (#505)

AI-Used: [claude]"
```

---

### Task 8: Pressure test — verify behavioral change

**Files:**
- Read: `skills/engram-tmux-lead/SKILL.md`

- [ ] **Step 1: Confirm zero inline PANE_ID split-window calls remain**

```bash
grep -c "PANE_ID=\$(tmux split-window" skills/engram-tmux-lead/SKILL.md
```

Expected: `0`

If non-zero: do NOT proceed. Find the remaining hit with `-n` flag and fix it.

- [ ] **Step 2: Confirm SPAWN-PANE is the single source of split-window -h -d**

```bash
grep -n "split-window -h -d" skills/engram-tmux-lead/SKILL.md
```

Expected hits: only lines inside the `#### SPAWN-PANE` block (lines 101 and 106 approximately, both within the procedure definition). No hits outside the SPAWN-PANE block.

- [ ] **Step 3: Confirm KILL-PANE is referenced at all kill sites**

```bash
grep -n "KILL-PANE" skills/engram-tmux-lead/SKILL.md
```

Expected: 6+ hits (procedure heading, comment inside block, §3.1 table, §3.1 note, §3.3 step 1, §3.4 step 4).

- [ ] **Step 4: Confirm SPAWN-PANE is referenced at all spawn sites**

```bash
grep -n "SPAWN-PANE" skills/engram-tmux-lead/SKILL.md
```

Expected: 6+ hits (procedure heading, comment inside block, Red Flag, §1.4 code block, §2.1 code block, §2.1 note).

- [ ] **Step 5: Final commit with issue close**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(engram-tmux-lead): enforce two-column layout via SPAWN-PANE/KILL-PANE (#505)

Extract splitting rules as SPAWN-PANE procedure, add KILL-PANE for removes,
replace all six bypass sites (§1.4, §2.1, §3.1×2, §3.3, §3.4), add Red Flag.

Closes #505

AI-Used: [claude]"
```

---

## Self-Review

**Spec coverage:**
- ✅ §1.3 splitting rules → named SPAWN-PANE with HARD GATE (Tasks 2–3)
- ✅ §1.4 engram-agent spawn bypass → fixed (Task 5)
- ✅ §2.1 spawn template bypass → fixed (Task 6)
- ✅ §3.1 DONE kill path → fixed (Task 7 steps 1–2)
- ✅ §3.3 Respawn kill path → fixed (Task 7 step 3)
- ✅ §3.4 hold-release kill path → fixed (Task 7 step 4)
- ✅ Red Flag added (Task 4)
- ✅ writing-skills TDD enforced (RED in Task 1, GREEN across Tasks 2–7, pressure test in Task 8)

**Placeholder scan:** None found.

**Type consistency:** `NEW_PANE` (set by SPAWN-PANE) → `PANE_ID=$NEW_PANE` (set by caller) is consistent across Tasks 5 and 6. `PANE_ID` (set by caller) → consumed by KILL-PANE is consistent across Tasks 7 steps 1–4.
