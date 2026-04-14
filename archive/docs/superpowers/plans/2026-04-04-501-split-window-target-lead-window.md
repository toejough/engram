# Issue #501: split-window Target Lead's Window Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all `tmux split-window` calls in `engram-tmux-lead` to always target the lead's window so panes open correctly regardless of which tmux window the user is currently viewing.

**Architecture:** Capture `LEAD_WINDOW=$(tmux display-message -p '#{window_id}')` once at startup in Section 1.3 (same pattern as the existing `TAIL_PANE_ID` capture), then add `-t "$LEAD_WINDOW"` to the 4 `split-window` calls that have no `-t` flag. The 2 calls that already target tracked pane IDs (`$MIDDLE_COL_LAST_PANE`, `$RIGHT_COL_LAST_PANE`) are unaffected — those pane IDs are always in the lead's window.

**Tech Stack:** Bash embedded in `skills/engram-tmux-lead/SKILL.md`. All edits MUST use the `superpowers:writing-skills` skill (required by CLAUDE.md — enforces TDD for skill edits).

---

## Files

- Modify: `skills/engram-tmux-lead/SKILL.md` — 1 new line + 4 modified lines

---

## Calls That Need Fixing

| Location | Current | Fix |
|----------|---------|-----|
| Section 1.3 ~line 84 — chat tail | `tmux split-window -h -d -P -F '#{pane_id}' "tail -F $CHAT_FILE"` | add `-t "$LEAD_WINDOW"` |
| Section 1.3 ~line 101 — splitting rules first branch | `tmux split-window -h -d -P -F '#{pane_id}'` | add `-t "$LEAD_WINDOW"` |
| Section 1.4 ~line 149 — engram-agent spawn | `tmux split-window -h -d -P -F '#{pane_id}'` | add `-t "$LEAD_WINDOW"` |
| Section 2.1 ~line 246 — spawn template | `tmux split-window -h -d -P -F '#{pane_id}'` | add `-t "$LEAD_WINDOW"` |

## Calls Already Correct (do NOT change)

| Location | Why OK |
|----------|--------|
| Section 1.3 ~line 106 — splitting rules second branch | `-t "$MIDDLE_COL_LAST_PANE"` — already targets a tracked pane in lead's window |
| Section 1.3 ~line 111 — splitting rules third branch | `-t "$RIGHT_COL_LAST_PANE"` — already targets a tracked pane in lead's window |

---

### Task 1: Invoke writing-skills and add LEAD_WINDOW capture

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (~line 73, inside Section 1.3 bash block)

- [ ] **Step 1: Invoke superpowers:writing-skills skill**

  This is mandatory before any SKILL.md edit per CLAUDE.md. The skill enforces the TDD cycle.

- [ ] **Step 2: Establish baseline behavior (RED)**

  Confirm the current Section 1.3 bash block (around line 66–88) has NO `LEAD_WINDOW` variable. Run:
  ```bash
  grep -n 'LEAD_WINDOW' skills/engram-tmux-lead/SKILL.md
  ```
  Expected: no output.

- [ ] **Step 3: Add LEAD_WINDOW capture**

  In Section 1.3's bash block, add this line immediately after `touch "$CHAT_FILE"` (around line 72) and before the `# Background task registry` comment:

  Old (context around insertion point):
  ```bash
  touch "$CHAT_FILE"

  # Background task registry: one active task per logical operation.
  ```

  New:
  ```bash
  touch "$CHAT_FILE"

  # Capture lead's window ID — passed to every split-window so panes stay in this window
  LEAD_WINDOW=$(tmux display-message -p '#{window_id}')

  # Background task registry: one active task per logical operation.
  ```

- [ ] **Step 4: Verify line was added (GREEN)**
  ```bash
  grep -n 'LEAD_WINDOW' skills/engram-tmux-lead/SKILL.md
  ```
  Expected: at least one hit showing the new `LEAD_WINDOW=$(tmux display-message ...)` line.

---

### Task 2: Fix chat tail split-window (Section 1.3)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (~line 84)

- [ ] **Step 1: Confirm current state (RED)**
  ```bash
  grep -n 'TAIL_PANE_ID' skills/engram-tmux-lead/SKILL.md
  ```
  Expected: a line like `TAIL_PANE_ID=$(tmux split-window -h -d -P -F '#{pane_id}' "tail -F $CHAT_FILE")` with no `-t` flag.

- [ ] **Step 2: Apply fix**

  Old:
  ```bash
  TAIL_PANE_ID=$(tmux split-window -h -d -P -F '#{pane_id}' "tail -F $CHAT_FILE")
  ```

  New:
  ```bash
  TAIL_PANE_ID=$(tmux split-window -h -d -t "$LEAD_WINDOW" -P -F '#{pane_id}' "tail -F $CHAT_FILE")
  ```

- [ ] **Step 3: Verify (GREEN)**
  ```bash
  grep -n 'TAIL_PANE_ID' skills/engram-tmux-lead/SKILL.md
  ```
  Expected: the line now contains `-t "$LEAD_WINDOW"`.

---

### Task 3: Fix splitting rules first branch (Section 1.3)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (~line 101)

- [ ] **Step 1: Identify the first branch (RED)**

  The splitting rules block has three branches. The first branch (RIGHT_PANE_COUNT < 4) is the only one without a `-t` flag:
  ```bash
  if [ "$RIGHT_PANE_COUNT" -lt 4 ]; then
    # Middle column not full: split right from coordinator, rebalance into single right column
    NEW_PANE=$(tmux split-window -h -d -P -F '#{pane_id}')
  ```

  Confirm:
  ```bash
  grep -n 'NEW_PANE' skills/engram-tmux-lead/SKILL.md
  ```
  Expected: 3 hits — one without `-t`, one with `-t "$MIDDLE_COL_LAST_PANE"`, one with `-t "$RIGHT_COL_LAST_PANE"`.

- [ ] **Step 2: Apply fix to first branch only**

  Old (first branch only — do NOT change the other two):
  ```bash
    NEW_PANE=$(tmux split-window -h -d -P -F '#{pane_id}')
  ```

  New:
  ```bash
    NEW_PANE=$(tmux split-window -h -d -t "$LEAD_WINDOW" -P -F '#{pane_id}')
  ```

- [ ] **Step 3: Verify all three branches (GREEN)**
  ```bash
  grep -n 'NEW_PANE' skills/engram-tmux-lead/SKILL.md
  ```
  Expected:
  - Line with first branch: contains `-t "$LEAD_WINDOW"`
  - Line with second branch: contains `-t "$MIDDLE_COL_LAST_PANE"` (unchanged)
  - Line with third branch: contains `-t "$RIGHT_COL_LAST_PANE"` (unchanged)

---

### Task 4: Fix engram-agent spawn (Section 1.4)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (~line 149)

- [ ] **Step 1: Identify the call (RED)**
  ```bash
  grep -n 'split-window' skills/engram-tmux-lead/SKILL.md
  ```
  Find the line in Section 1.4 under the comment `# Split a new pane to the right, start claude in it`. It should read:
  ```bash
  PANE_ID=$(tmux split-window -h -d -P -F '#{pane_id}')
  ```

- [ ] **Step 2: Apply fix**

  Old (Section 1.4):
  ```bash
  # Split a new pane to the right, start claude in it
  PANE_ID=$(tmux split-window -h -d -P -F '#{pane_id}')
  ```

  New:
  ```bash
  # Split a new pane to the right, start claude in it
  PANE_ID=$(tmux split-window -h -d -t "$LEAD_WINDOW" -P -F '#{pane_id}')
  ```

- [ ] **Step 3: Verify (GREEN)**

  Check both Section 1.4 and Section 2.1 still have distinct occurrences (both need fixing; fixing one does not fix the other):
  ```bash
  grep -n 'PANE_ID=$(tmux split-window' skills/engram-tmux-lead/SKILL.md
  ```
  After this step: the Section 1.4 line should contain `-t "$LEAD_WINDOW"`. The Section 2.1 line may or may not yet (it gets fixed in Task 5).

---

### Task 5: Fix spawn template (Section 2.1)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (~line 246)

- [ ] **Step 1: Identify the call (RED)**

  Section 2.1 "Spawn Template" has its own `split-window` call under the comment `# Split a new pane to the right, capturing the new pane ID atomically`:
  ```bash
  PANE_ID=$(tmux split-window -h -d -P -F '#{pane_id}')
  ```

- [ ] **Step 2: Apply fix**

  Old (Section 2.1):
  ```bash
  # Split a new pane to the right, capturing the new pane ID atomically
  PANE_ID=$(tmux split-window -h -d -P -F '#{pane_id}')
  ```

  New:
  ```bash
  # Split a new pane to the right, capturing the new pane ID atomically
  PANE_ID=$(tmux split-window -h -d -t "$LEAD_WINDOW" -P -F '#{pane_id}')
  ```

- [ ] **Step 3: Verify all split-window calls (GREEN)**

  ```bash
  grep -n 'split-window' skills/engram-tmux-lead/SKILL.md
  ```
  Expected: every `split-window` call now has a `-t` flag — either `-t "$LEAD_WINDOW"`, `-t "$MIDDLE_COL_LAST_PANE"`, or `-t "$RIGHT_COL_LAST_PANE"`. No bare `split-window` without `-t`.

---

### Task 6: Commit and close issue

- [ ] **Step 1: Final verification**
  ```bash
  grep -n 'split-window' skills/engram-tmux-lead/SKILL.md
  ```
  Confirm all 6 `split-window` calls have a `-t` flag. Count: 4 with `-t "$LEAD_WINDOW"`, 1 with `-t "$MIDDLE_COL_LAST_PANE"`, 1 with `-t "$RIGHT_COL_LAST_PANE"`.

  Also confirm LEAD_WINDOW capture is present:
  ```bash
  grep -n 'LEAD_WINDOW' skills/engram-tmux-lead/SKILL.md
  ```
  Expected: 5 hits — 1 assignment (`LEAD_WINDOW=$(tmux display-message...)`), 4 usages (`-t "$LEAD_WINDOW"`).

- [ ] **Step 2: Commit**
  ```bash
  git add skills/engram-tmux-lead/SKILL.md
  git commit -m "$(cat <<'EOF'
fix(engram-tmux-lead): target lead's window in all split-window calls (#501)

Capture LEAD_WINDOW at startup and pass -t flag to every split-window
call that previously had no target. Panes now always open in the lead's
window regardless of which window the user has navigated to.

AI-Used: [claude]
EOF
  )"
  ```

- [ ] **Step 3: Close issue**
  ```bash
  gh issue close 501 --repo toejough/engram --comment "Fixed: LEAD_WINDOW captured at startup (Section 1.3), -t flag added to all 4 untargeted split-window calls (Sections 1.3, 1.3 splitting rules, 1.4, 2.1)."
  ```
