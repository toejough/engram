# Fix PENDING-RELEASE Stale Language (Issue #517) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Phase 1 hold-monitoring language in the PENDING-RELEASE state table row with accurate Phase 2 language.

**Architecture:** Single-phrase edit in one Markdown file. No Go code, no binary changes. The existing Section 3.5 of the same file already documents the Phase 2 `engram hold check` procedure in full — this fix aligns the state table summary with that procedure. Per CLAUDE.md, all SKILL.md edits must go through the `superpowers:writing-skills` TDD workflow.

**Tech Stack:** Markdown, git, `superpowers:writing-skills` skill.

---

### Task 1: Invoke writing-skills and run RED baseline test

**Files:**
- Read-only: `skills/engram-tmux-lead/SKILL.md:477`

- [ ] **Step 1: Invoke the `superpowers:writing-skills` skill**

Before touching any SKILL.md file, invoke:
```
Skill: superpowers:writing-skills
```
Follow its instructions for the full TDD workflow. The steps below implement the RED/GREEN/verify/pressure-test cycle it prescribes.

- [ ] **Step 2: Run the RED baseline test — confirm stale text is present**

```bash
grep -n "Monitor holds via background tasks" skills/engram-tmux-lead/SKILL.md
```

Expected output (line number may vary slightly):
```
477:| **PENDING-RELEASE** | ... | Do NOT kill pane. Agent remains alive and responsive. Monitor holds via background tasks. Silence threshold still applies ...
```

If this grep returns no output, the stale text is already gone — stop and investigate before proceeding.

- [ ] **Step 3: Confirm the replacement text is NOT yet present (RED)**

```bash
grep -n "Run engram hold check after each agent done event" skills/engram-tmux-lead/SKILL.md
```

Expected output: **no matches**. If this grep returns a hit, the fix was already applied — stop.

---

### Task 2: Apply the edit (GREEN)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md:477`

- [ ] **Step 1: Apply the replacement**

In `skills/engram-tmux-lead/SKILL.md` line 477, replace the action cell phrase:

Old text (exact):
```
Monitor holds via background tasks.
```

New text (exact):
```
Run engram hold check after each agent done event.
```

The full line after the edit should read:
```
| **PENDING-RELEASE** | Agent posted `done` AND lead's hold registry contains at least one hold targeting this agent | Do NOT kill pane. Agent remains alive and responsive. Run engram hold check after each agent done event. Silence threshold still applies — use PENDING-RELEASE-specific nudge text (see 3.2). |
```

- [ ] **Step 2: Verify the edit applied (GREEN)**

```bash
grep -n "Run engram hold check after each agent done event" skills/engram-tmux-lead/SKILL.md
```

Expected: exactly one match at (or near) line 477.

- [ ] **Step 3: Confirm stale text is gone**

```bash
grep -n "Monitor holds via background tasks" skills/engram-tmux-lead/SKILL.md
```

Expected: **no output**. If any line still contains this phrase, the edit did not apply — fix before continuing.

- [ ] **Step 4: Scan for other stale Phase 1 hold-monitoring references**

```bash
grep -n "background tasks" skills/engram-tmux-lead/SKILL.md
```

All remaining hits should be Phase 2-valid references (chat monitor hygiene at line ~78, drain rules at ~232, READY-check loops at ~339, merge-queue wait at ~855). None should describe hold monitoring. If any hit describes monitoring holds via background tasks, fix it before committing.

---

### Task 3: Run pressure tests and commit

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (already edited above)

- [ ] **Step 1: Run pressure tests per writing-skills skill guidance**

Follow the pressure test procedure defined by `superpowers:writing-skills`. At minimum, verify the skill reads coherently around the changed row by inspecting context:

```bash
sed -n '470,482p' skills/engram-tmux-lead/SKILL.md
```

Confirm:
- The PENDING-RELEASE row reads naturally with the new phrase.
- The DONE row and surrounding rows are unchanged.
- No formatting is broken (pipe-delimited table structure intact).

- [ ] **Step 2: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(skills): update PENDING-RELEASE row to reference engram hold check

Phase 2 replaced background hold monitoring with manual engram hold check
triggered after each agent done event. The state table still said 'Monitor
holds via background tasks', causing lead agents to attempt non-existent
Phase 1 behavior.

Closes #517

AI-Used: [claude]"
```
