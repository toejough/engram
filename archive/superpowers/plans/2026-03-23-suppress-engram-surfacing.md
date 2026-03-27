# Suppress Memory Surfacing for Engram Operations — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop engram from surfacing irrelevant memories when its own skills or CLI commands are invoked.

**Architecture:** Two filtering changes in hook shell scripts: (1) early exit in user-prompt-submit.sh when the user message is an engram skill invocation, (2) widen the Bash command filter in pre/post-tool-use hooks to match all engram binary invocations.

**Tech Stack:** Bash (hook scripts), no Go changes.

**Spec:** `docs/superpowers/specs/2026-03-23-suppress-engram-surfacing-design.md`

---

## File Structure

### Modified Files

| File | Change |
|------|--------|
| `hooks/user-prompt-submit.sh` | Add skill-directory check to skip surfacing for `/recall`, `/memory-triage`, etc. |
| `hooks/pre-tool-use.sh` | Replace narrow `engram feedback`/`engram correct` filter with `$ENGRAM_BIN` path match |
| `hooks/post-tool-use.sh` | Same `$ENGRAM_BIN` path match replacement |

---

## Task 1: Add Skill Invocation Guard to User-Prompt-Submit Hook

**Files:**
- Modify: `hooks/user-prompt-submit.sh`

- [ ] **Step 1: Add the skill directory check**

Insert after the `TRANSCRIPT_PATH=` line (line 30) and before the `# UC-3:` comment (line 32), add:

```bash
# Skip surfacing for engram skill invocations (#369)
SKILL_CMD="${USER_MESSAGE%% *}"
if [[ "$SKILL_CMD" == /* ]]; then
    SKILL_NAME="${SKILL_CMD#/}"
    if [[ -d "$PLUGIN_ROOT/skills/$SKILL_NAME" ]]; then
        exit 0
    fi
fi
```

- [ ] **Step 2: Manual verification**

In a Claude Code session with the engram plugin active:
1. Type `/recall` — confirm no engram-internal memories appear in the system reminder.
2. Type `/recall some query` — confirm also suppressed.
3. Type `/memory-triage` — confirm also suppressed.
4. Type a normal message like "what's next?" — confirm memories still surface.
5. Type `/something-not-a-skill` — confirm memories still surface.

- [ ] **Step 3: Commit**

```bash
git add hooks/user-prompt-submit.sh
git commit -m "fix(hooks): skip surfacing for engram skill invocations (#369)"
```

---

## Task 2: Widen Bash Filter in Pre-Tool-Use Hook

**Files:**
- Modify: `hooks/pre-tool-use.sh`

- [ ] **Step 1: Replace the narrow filter with $ENGRAM_BIN match**

In `hooks/pre-tool-use.sh`, replace:

```bash
    if [[ "$BASH_CMD" == *"engram feedback"* || "$BASH_CMD" == *"engram correct"* ]]; then
```

With:

```bash
    if [[ "$BASH_CMD" == *"$ENGRAM_BIN"* ]]; then
```

Also update the comment above it from:

```bash
# Don't surface memories for engram plumbing calls — they create a feedback loop (#352)
```

To:

```bash
# Don't surface memories for any engram CLI calls (#352, #369)
```

- [ ] **Step 2: Manual verification**

In a Claude Code session with the engram plugin active:
1. Run a Bash command like `engram show --help` — confirm no memories surface in the pre-tool-use system reminder.
2. Run `grep "engram" hooks/pre-tool-use.sh` — confirm memories DO still surface (not an engram binary invocation).
3. Run `targ test` — confirm memories still surface normally.

- [ ] **Step 3: Commit**

```bash
git add hooks/pre-tool-use.sh
git commit -m "fix(hooks): widen pre-tool-use filter to all engram commands (#369)"
```

---

## Task 3: Widen Bash Filter in Post-Tool-Use Hook

**Files:**
- Modify: `hooks/post-tool-use.sh`

- [ ] **Step 1: Replace the narrow filter with $ENGRAM_BIN match**

In `hooks/post-tool-use.sh`, replace:

```bash
    if [[ "$BASH_CMD" == *"engram feedback"* || "$BASH_CMD" == *"engram correct"* ]]; then
```

With:

```bash
    if [[ "$BASH_CMD" == *"$ENGRAM_BIN"* ]]; then
```

Also update the comment above it from:

```bash
# Don't surface memories for engram plumbing calls — they create a feedback loop (#352)
```

To:

```bash
# Don't surface memories for any engram CLI calls (#352, #369)
```

- [ ] **Step 2: Commit**

```bash
git add hooks/post-tool-use.sh
git commit -m "fix(hooks): widen post-tool-use filter to all engram commands (#369)"
```

---

## Task 4: End-to-End Verification

Verify all three changes together. Session-start surfacing should be unaffected since `session-start.sh` was not modified.

- [ ] **Step 1: Start a fresh Claude Code session with the engram plugin active**

Confirm session-start memories still appear normally.

- [ ] **Step 2: Verify skill suppression**

1. Type `/recall` — no engram-internal memories in system reminder.
2. Type `/memory-triage` — no engram-internal memories in system reminder.

- [ ] **Step 3: Verify Bash command suppression**

1. Run `engram show --help` — no memories in pre/post-tool-use system reminders.
2. Run `grep "engram" hooks/pre-tool-use.sh` — memories DO surface (false positive avoided).

- [ ] **Step 4: Verify normal surfacing still works**

1. Type a normal message — memories surface in user-prompt-submit reminder.
2. Run `targ test` — memories surface in pre/post-tool-use reminders.
