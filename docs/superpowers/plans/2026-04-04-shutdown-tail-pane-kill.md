# Shutdown Tail Pane Kill Fix (Issue #479) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the shutdown sequence in `skills/engram-tmux-lead/SKILL.md` to kill the chat tail pane by its tracked pane ID instead of unreliably grepping for the `tail` command.

**Architecture:** Single one-line edit to Section 3.4 Step 4 of the skill file. Replace the `tmux list-panes | grep tail | ...` pipeline with `tmux kill-pane -t "$TAIL_PANE_ID"`. The `TAIL_PANE_ID` variable is already captured in Section 1.3 and used consistently for all other pane kills in the skill.

**Tech Stack:** Markdown (skill file), tmux, bash

---

### Task 1: Fix shutdown tail pane kill in SKILL.md

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md:336-338`

- [ ] **Step 1: Verify the broken pattern exists**

Run:
```bash
grep -n 'pane_current_command.*grep.*tail\|grep tail.*awk' skills/engram-tmux-lead/SKILL.md
```

Expected output (line number may vary by a few lines):
```
338:     tmux list-panes -F '#{pane_id} #{pane_current_command}' | grep tail | awk '{print $1}' | xargs -I{} tmux kill-pane -t {}
```

If this grep returns no results, the bug has already been fixed — stop here.

- [ ] **Step 2: Apply the fix**

In `skills/engram-tmux-lead/SKILL.md`, replace the entire step 4 block within the shutdown sequence (inside the fenced code block in Section 3.4). The exact old text to replace is:

```
4. Kill the chat tail pane (the split pane created during startup):
   - Find and kill any pane running tail on the chat file:
     tmux list-panes -F '#{pane_id} #{pane_current_command}' | grep tail | awk '{print $1}' | xargs -I{} tmux kill-pane -t {}
```

Replace with:

```
4. Kill the chat tail pane (the split pane created during startup):
   - Kill by tracked pane ID captured in Section 1.3:
     tmux kill-pane -t "$TAIL_PANE_ID" 2>/dev/null || true
```

- [ ] **Step 3: Verify the old pattern is gone**

Run:
```bash
grep -n 'pane_current_command\|grep tail\|xargs.*kill-pane' skills/engram-tmux-lead/SKILL.md
```

Expected: no output (the old grep pipeline is fully removed).

- [ ] **Step 4: Verify the new pattern is present**

Run:
```bash
grep -n 'TAIL_PANE_ID' skills/engram-tmux-lead/SKILL.md
```

Expected: two matches — one in Section 1.3 (capture) and one in Section 3.4 (kill):
```
61:TAIL_PANE_ID=$(tmux split-window -h -d -P -F '#{pane_id}' "tail -F $CHAT_FILE")
338:     tmux kill-pane -t "$TAIL_PANE_ID" 2>/dev/null || true
```

Line numbers may vary slightly; what matters is exactly two occurrences.

- [ ] **Step 5: Verify the diff looks correct**

Run:
```bash
git diff skills/engram-tmux-lead/SKILL.md
```

The diff should show exactly three lines changed in Section 3.4 — the old three-line step 4 block replaced by the new two-line block. Nothing else should change.

- [ ] **Step 6: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(skills): kill chat tail pane by TAIL_PANE_ID in shutdown sequence

The previous approach used pane_current_command to grep for 'tail',
which fails when the shell (e.g. fish) is reported as the current
command instead of the child tail process.

TAIL_PANE_ID is already captured in Section 1.3 — use it directly.
Adds 2>/dev/null || true to handle already-dead panes gracefully.

Fixes #479

AI-Used: [claude]"
```
