# Lead Auto-ACK Intents Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **REQUIRED for SKILL.md edits:** Use `superpowers:writing-skills` skill for every SKILL.md change (enforces TDD: baseline test → update → verify).

**Goal:** Fix the lead's chat processing so it automatically ACKs `intent` messages addressed to it, unblocking spawned agents that stall waiting for a lead ACK.

**Architecture:** Two pseudocode blocks in `skills/engram-tmux-lead/SKILL.md` need the same change — add an intent-ACK step to the post-drain sweep in both Section 1.6 and Section 6.1. The prose description in Section 1.6 step 2 also needs updating to document the expected behavior. No new sections or structural changes required.

**Tech Stack:** Markdown/skill prose. No Go code, no tests with `targ`. Skill behavioral tests use the `superpowers:writing-skills` pressure-test pattern.

---

## File Map

| File | Change |
|------|--------|
| `skills/engram-tmux-lead/SKILL.md` | Modify Section 1.6 (lines ~213–236) and Section 6.1 (lines ~853–868) |

---

### Task 1: Establish Baseline Behavior Test (RED)

**Files:**
- Read: `skills/engram-tmux-lead/SKILL.md:207-236` (Section 1.6)
- Read: `skills/engram-tmux-lead/SKILL.md:842-870` (Section 6.1)

This task documents what the skill currently directs the lead to do when a monitor fires, so we can verify the fix changes that behavior.

- [ ] **Step 1: Read current Section 1.6 step 2 text**

  ```bash
  sed -n '207,236p' skills/engram-tmux-lead/SKILL.md
  ```

  Expected: step 2 reads `"process the chat message — relay questions to the user, handle agent status updates, etc."` with no mention of ACKing intents.

- [ ] **Step 2: Read current Section 6.1 pseudocode**

  ```bash
  sed -n '842,870p' skills/engram-tmux-lead/SKILL.md
  ```

  Expected: `process_chat_messages(new_lines)` is called with no preceding intent-ACK step.

- [ ] **Step 3: Write baseline pressure test**

  Ask the skill: *"A spawned agent posts `type = \"intent\"`, `to = \"engram-agent, lead\"`, `thread = \"implementation\"`, `from = \"exec-1\"`. The monitor fires and returns `INTENT|exec-1|420|...`. What does the lead do next?"*

  Expected (current, broken) answer per skill: lead runs the post-drain sweep, calls `process_chat_messages`, spawns a new monitor. No ACK to `exec-1`. This confirms the baseline is RED.

---

### Task 2: Update Section 1.6 Step 2 Prose (GREEN)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md:213-214`

- [ ] **Step 1: Invoke writing-skills skill for the edit**

  ```
  Use superpowers:writing-skills to edit skills/engram-tmux-lead/SKILL.md
  ```

  Follow the skill's TDD protocol.

- [ ] **Step 2: Replace step 2 prose**

  Find this text (lines ~213–214):

  ```markdown
  1. After each user interaction, **replace** the chat monitor Agent (see drain-before-spawn pattern below)
  2. If the monitor Agent fires (agent posted something), process the chat message — relay questions to the user, handle agent status updates, etc.
  ```

  Replace with:

  ```markdown
  1. After each user interaction, **replace** the chat monitor Agent (see drain-before-spawn pattern below)
  2. If the monitor Agent fires (agent posted something):
     a. **ACK any `intent` messages** in the post-drain sweep `new_lines` that are addressed to `lead` or `all` — before any other processing. Extract `from` and `thread` from the TOML block; post `type = "ack"`, `to = <from-field>`, `thread = <thread-field>`, `text = "Received."`.
     b. Process the chat message — relay questions to the user, handle agent status updates, route tasks, etc.
  ```

- [ ] **Step 3: Verify the edit looks correct**

  ```bash
  sed -n '207,222p' skills/engram-tmux-lead/SKILL.md
  ```

  Expected: step 2 now has sub-steps a and b, with explicit ACK instruction before processing.

---

### Task 3: Update Section 1.6 Pseudocode (GREEN)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md:222-236`

- [ ] **Step 1: Replace the post-drain sweep pseudocode in Section 1.6**

  Find this block (lines ~229–232):

  ```python
  new_lines = run_bash(f'tail -n +{CURSOR + 1} "$CHAT_FILE"')
  CURSOR = run_bash('wc -l < "$CHAT_FILE"').strip()
  if new_lines.strip():
      process_chat_messages(new_lines)   # relay, route, or queue as normal
  ```

  Replace with:

  ```python
  new_lines = run_bash(f'tail -n +{CURSOR + 1} "$CHAT_FILE"')
  CURSOR = run_bash('wc -l < "$CHAT_FILE"').strip()
  if new_lines.strip():
      # ACK any intents addressed to lead or all — BEFORE routing/relay:
      for intent in toml_blocks(new_lines, type="intent", to_includes=["lead", "all"]):
          post_ack(to=intent.from, thread=intent.thread, text="Received.")
      process_chat_messages(new_lines)   # relay, route, or queue as normal
  ```

- [ ] **Step 2: Verify the edit**

  ```bash
  sed -n '220,240p' skills/engram-tmux-lead/SKILL.md
  ```

  Expected: `for intent in toml_blocks(...)` appears before `process_chat_messages`.

---

### Task 4: Update Section 6.1 Pseudocode (GREEN — second occurrence)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md:853-868`

Section 6.1 contains an identical replace-pattern pseudocode block. It needs the same intent-ACK addition.

- [ ] **Step 1: Replace the post-drain sweep pseudocode in Section 6.1**

  Find this block (lines ~860–863):

  ```python
  new_lines = run_bash(f'tail -n +{CURSOR + 1} "$CHAT_FILE"')
  CURSOR = run_bash('wc -l < "$CHAT_FILE"').strip()
  if new_lines.strip():
      process_chat_messages(new_lines)   # relay, route, or queue as normal
  ```

  Replace with:

  ```python
  new_lines = run_bash(f'tail -n +{CURSOR + 1} "$CHAT_FILE"')
  CURSOR = run_bash('wc -l < "$CHAT_FILE"').strip()
  if new_lines.strip():
      # ACK any intents addressed to lead or all — BEFORE routing/relay:
      for intent in toml_blocks(new_lines, type="intent", to_includes=["lead", "all"]):
          post_ack(to=intent.from, thread=intent.thread, text="Received.")
      process_chat_messages(new_lines)   # relay, route, or queue as normal
  ```

- [ ] **Step 2: Verify the edit**

  ```bash
  sed -n '850,875p' skills/engram-tmux-lead/SKILL.md
  ```

  Expected: same `toml_blocks(...)` loop appears before `process_chat_messages` in Section 6.1.

- [ ] **Step 3: Confirm no other occurrences of the post-drain pseudocode need updating**

  ```bash
  grep -n "process_chat_messages" skills/engram-tmux-lead/SKILL.md
  ```

  Expected: exactly 2 matches (Section 1.6 and Section 6.1). If more appear, apply the same intent-ACK addition to each.

---

### Task 5: Run Pressure Tests (VERIFY)

- [ ] **Step 1: Pressure test — monitor fires with INTENT**

  Ask the updated skill: *"A spawned agent `exec-1` posts `type = \"intent\"`, `to = \"engram-agent, lead\"`, `thread = \"implementation\"`, `from = \"exec-1\"`. The monitor fires. What does the lead do in the post-drain sweep before calling `process_chat_messages`?"*

  Expected answer: lead finds the intent in `new_lines` via `toml_blocks(new_lines, type="intent", to_includes=["lead","all"])`, calls `post_ack(to="exec-1", thread="implementation", text="Received.")`, THEN calls `process_chat_messages`.

- [ ] **Step 2: Pressure test — intent to "all"**

  Ask: *"An agent posts `type = \"intent\"`, `to = \"all\"`, `thread = \"lifecycle\"`, `from = \"planner-1\"`. Does the lead ACK?"*

  Expected: yes — `"all"` is in `to_includes`, so lead ACKs `to="planner-1"`, `thread="lifecycle"`.

- [ ] **Step 3: Pressure test — non-intent message (no spurious ACK)**

  Ask: *"An agent posts `type = \"info\"`, `to = \"lead\"`, `thread = \"status\"`. Does the lead post an ACK?"*

  Expected: no — `type != "intent"` so the `toml_blocks` filter produces nothing, no ACK is posted.

- [ ] **Step 4: Pressure test — existing routing unaffected**

  Ask: *"After ACKing an intent, what does the lead do next?"*

  Expected: calls `process_chat_messages(new_lines)` as before — routing/relay logic unchanged.

---

### Task 6: Commit

- [ ] **Step 1: Check diff**

  ```bash
  git diff skills/engram-tmux-lead/SKILL.md
  ```

  Expected: two pseudocode blocks modified (Sections 1.6 and 6.1), one prose step modified (Section 1.6 step 2).

- [ ] **Step 2: Commit**

  ```bash
  git add skills/engram-tmux-lead/SKILL.md
  git commit -m "$(cat <<'EOF'
  fix(engram-tmux-lead): auto-ACK intent messages addressed to lead (#503)

  The lead's post-drain sweep now checks new_lines for intent messages
  addressed to 'lead' or 'all' and posts an ack before any other
  processing. Fixes spawned agents stalling indefinitely waiting for
  a lead ACK that never came.

  AI-Used: [claude]
  EOF
  )"
  ```

---

## Acceptance Criteria Checklist

- [ ] Lead skill explicitly handles `intent` messages addressed to it by posting an `ack` before any other action (Section 1.6 step 2a, Section 6.1 pseudocode)
- [ ] Spawned agents that address `lead` in their intent `to` field no longer stall waiting for a lead ACK
- [ ] Non-intent messages (info, done, ack, etc.) do not trigger spurious ACKs
- [ ] Existing lead behaviors (routing, dispatch, escalation surfacing) are unaffected — ACK is posted first, then `process_chat_messages` runs unchanged
