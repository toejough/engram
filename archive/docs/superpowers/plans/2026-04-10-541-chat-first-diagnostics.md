# Issue #541: Chat-First Diagnostics in engram-tmux-lead Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent the lead from reaching for `tmux capture-pane` before reading the chat file when an agent appears stuck or silent.

**Architecture:** Pure skill edit — four text changes to `skills/engram-tmux-lead/SKILL.md` plus behavioral test assertions, governed by the `superpowers:writing-skills` TDD discipline (baseline test → RED → GREEN → verification). No Go code changes.

**Tech Stack:** Markdown skill file, `superpowers:writing-skills` skill, `engram chat` binary for verification commands.

---

## Background: What the Issue Is

The skill currently sends the lead to `tmux capture-pane` as the **first** diagnostic step in two timeout scenarios:

| Location | Trigger | Current (broken) behavior |
|----------|---------|--------------------------|
| Section 1.5, line after `wait-ready` fails | `engram agent wait-ready` times out | `1. Check pane exists: tmux capture-pane -t "$PANE_ID" -p \| tail -20` |
| Section 2.1, TIMEOUT branch | Background watch for `done` times out | `TIMEOUT: agent may be stuck. Check via tmux capture-pane -t "$PANE_ID" -p -S -20` |

The chat file is the authoritative record of agent state. An agent that is stuck typically posted a WAIT, a long intent thread, a `conversation` message, or went silent in a recognizable pattern — all visible in chat. `tmux` output is noisy (tool call JSON, status bar updates, compaction banners) and harder to interpret than structured TOML messages. Reading chat first is cheaper and more reliable.

A third location — the preamble's list of allowed lead commands — describes `capture-pane` without any ordering constraint, implicitly blessing it as a peer tool alongside chat.

---

## File Map

| File | Action | What changes |
|------|--------|-------------|
| `skills/engram-tmux-lead/SKILL.md` | Modify (4 sites) | Preamble (ordering constraint), new Chat-First Diagnostics section, Section 1.5 timeout, Section 2.1 TIMEOUT |
| `skills/engram-tmux-lead/tests/behavioral_test.sh` | Modify | Add Group 10 RED assertions for chat-first rule |

---

## Task 1: Baseline Behavior Test (RED)

Establish and document what the skill currently instructs so the GREEN edit has a concrete target.

**Files:**
- Read: `skills/engram-tmux-lead/SKILL.md`

- [ ] **Step 1: Invoke `superpowers:writing-skills` skill**

This is a mandatory gate before editing any SKILL.md file. It enforces the TDD cycle. Invoke it now, before touching any text.

- [ ] **Step 2: Read Section 1.5 timeout block and confirm baseline**

```bash
grep -n "capture-pane\|tmux capture" skills/engram-tmux-lead/SKILL.md
```

Expected output (approximate line numbers):
```
24:- `tmux` send-keys, capture-pane (for nudging and health checks only)
154:1. Check pane exists: `tmux capture-pane -t "$PANE_ID" -p | tail -20`
243:- TIMEOUT: agent may be stuck. Check via `tmux capture-pane -t "$PANE_ID" -p -S -20`.
```

This confirms the three sites. If line numbers differ, identify them — the plan refers to them by their surrounding context, not by line number.

- [ ] **Step 3: Add failing Group 10 assertions to `behavioral_test.sh` (RED)**

Open `skills/engram-tmux-lead/tests/behavioral_test.sh`. Locate the Group 9 block and the Summary section. Insert the following Group 10 block between them (before the `# Summary` separator):

```bash
# ---------------------------------------------------------------------------
# Group 10: Chat-first diagnostics (#541)
# ---------------------------------------------------------------------------

echo "--- Group 10: Chat-first diagnostics ---"

assert_contains "Chat-First Diagnostics hard rule exists" "HARD RULE: The chat file is the primary diagnostic source"
assert_contains "Preamble capture-pane is last resort" "last resort"
assert_contains "Section 1.5 timeout reads chat first" "Read chat from cursor"
assert_contains "Section 2.1 TIMEOUT reads chat first" "Read chat from cursor"

echo ""
```

- [ ] **Step 4: Run the test to confirm it fails (RED)**

```bash
bash skills/engram-tmux-lead/tests/behavioral_test.sh 2>&1 | tail -20
```

Expected: 4 failures in Group 10 — `FAIL: Chat-First Diagnostics hard rule exists`, `FAIL: Preamble capture-pane is last resort`, `FAIL: Section 1.5 timeout reads chat first`, `FAIL: Section 2.1 TIMEOUT reads chat first`.

- [ ] **Step 5: Commit the RED state**

```bash
git add skills/engram-tmux-lead/tests/behavioral_test.sh
git commit -m "test(skills): add RED assertions for chat-first diagnostics (#541)

Group 10 assertions fail against current SKILL.md. GREEN edit follows.

AI-Used: [claude]"
```

---

## Task 2: GREEN Edit — Update the Skill

Four targeted changes to `skills/engram-tmux-lead/SKILL.md`. Make them in order.

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md`

### Change 1: Preamble — Clarify ordering constraint on `capture-pane`

- [ ] **Step 1: Find the preamble line**

It currently reads:
```
- `tmux` send-keys, capture-pane (for nudging and health checks only)
```

- [ ] **Step 2: Replace it**

Change it to:
```
- `tmux` send-keys, capture-pane (**last resort** — always read the chat file first; see Chat-First Diagnostics below)
```

### Change 2: Add HARD RULE section — "Chat-First Diagnostics"

This goes immediately after the preamble block (after the line `- grep on the chat file to check agent status`) and before the `HARD GATE — parrot FIRST` paragraph.

- [ ] **Step 3: Insert the following block**

```markdown
### Chat-First Diagnostics

**HARD RULE: The chat file is the primary diagnostic source. `tmux capture-pane` is a last resort.**

When an agent appears stuck, silent, or has not responded in the expected time:

1. **Read the chat file from cursor first.**
   ```bash
   tail -n +$((CURSOR + 1)) "$CHAT_FILE" | tail -40
   ```
   Look for: `type = "wait"` (agent is blocked waiting for a response), `type = "conversation"` (agent reasoning aloud — may be mid-tool-call), `type = "intent"` (agent announced an action but you haven't ACKed), or any message with a recent timestamp.

2. **If chat explains the silence** — the agent posted a WAIT, is in a long tool call, or is mid-intent cycle — engage with the protocol, not the terminal.

3. **Only if chat has no messages from the agent since your cursor** — use `tmux capture-pane` to inspect the raw pane for crash output or stuck prompts:
   ```bash
   tmux capture-pane -t "$PANE_ID" -p -S -20
   ```

Skipping step 1 means you may interrupt an agent mid-task or misdiagnose a protocol event as a crash.
```

### Change 3: Section 1.5 — `wait-ready` timeout, replace step 1

- [ ] **Step 4: Find the timeout block in Section 1.5**

It currently reads:
```markdown
If the command times out (exit non-zero):
1. Check pane exists: `tmux capture-pane -t "$PANE_ID" -p | tail -20`
2. Report to user with diagnostic info. Do NOT silently proceed without memory.
```

- [ ] **Step 5: Replace it with the chat-first sequence**

```markdown
If the command times out (exit non-zero):
1. **Read chat from cursor** — check whether the agent posted anything since spawn:
   ```bash
   tail -n +$((PRE_SPAWN_CURSOR + 1)) "$CHAT_FILE" | tail -40
   ```
   If you see a `ready` message, `info`, `wait`, or `conversation` from `engram-agent`, engage with that before going further.
2. **If chat shows nothing from the agent:** inspect the raw pane for crash output:
   ```bash
   tmux capture-pane -t "$PANE_ID" -p | tail -20
   ```
3. Report to user with diagnostic info (chat excerpt + pane output). Do NOT silently proceed without memory.
```

### Change 4: Section 2.1 — TIMEOUT branch, replace immediate `capture-pane`

- [ ] **Step 6: Find the TIMEOUT line in Section 2.1**

It currently reads:
```markdown
- TIMEOUT: agent may be stuck. Check via `tmux capture-pane -t "$PANE_ID" -p -S -20`. Transition to SILENT per Section 3.2.
```

- [ ] **Step 7: Replace it with the chat-first sequence**

```markdown
- TIMEOUT: agent may be stuck.
  1. **Read chat from cursor** — check whether the agent posted a WAIT, conversation message, or long-running intent since the task was assigned:
     ```bash
     tail -n +$((CURSOR + 1)) "$CHAT_FILE" | tail -40
     ```
     If the agent posted a WAIT or is mid-intent cycle, engage with the protocol before treating this as SILENT.
  2. **If chat shows no messages from the agent:** inspect raw pane output:
     ```bash
     tmux capture-pane -t "$PANE_ID" -p -S -20
     ```
  3. Transition to SILENT per Section 3.2.
```

- [ ] **Step 8: Run quality checks**

```bash
targ check-full
```

Expected: no errors. This is a Markdown-only skill edit so Go tests are unaffected; confirm no accidental whitespace/syntax issues.

- [ ] **Step 9: Run behavioral tests to confirm all 4 Group 10 assertions now pass**

```bash
bash skills/engram-tmux-lead/tests/behavioral_test.sh 2>&1 | tail -20
```

Expected: Group 10 shows 4 passes. Overall `FAIL: 0`.

- [ ] **Step 10: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(skills): read chat before tmux capture-pane in lead diagnostics

Closes #541. The lead was reaching for tmux capture-pane as step 1
whenever wait-ready or wait-for-done timed out. The chat file is the
authoritative coordination source — an agent that is blocked will have
posted a WAIT, intent, or conversation message. Adds a Chat-First
Diagnostics hard rule and updates both timeout paths to read chat before
falling back to tmux.

AI-Used: [claude]"
```

---

## Task 3: Verification Test (GREEN confirmed)

Verify that the edited skill no longer instructs `tmux capture-pane` as a first step in any timeout scenario.

**Files:**
- Read: `skills/engram-tmux-lead/SKILL.md`
- Run: `skills/engram-tmux-lead/tests/behavioral_test.sh`

- [ ] **Step 1: Run the full behavioral test suite**

```bash
bash skills/engram-tmux-lead/tests/behavioral_test.sh 2>&1
```

Expected: `FAIL: 0`, `All tests passed.` — all Groups 1–10 pass.

- [ ] **Step 2: Re-run the grep to confirm the preamble change**

```bash
grep -n "capture-pane\|send-keys, capture" skills/engram-tmux-lead/SKILL.md
```

Expected: the preamble line now contains "last resort", not the original wording.

- [ ] **Step 3: Verify Section 1.5 timeout no longer starts with `tmux capture-pane`**

```bash
grep -A 10 "If the command times out" skills/engram-tmux-lead/SKILL.md
```

Expected output begins with:
```
1. **Read chat from cursor** — check whether the agent posted anything since spawn:
```

Not:
```
1. Check pane exists: `tmux capture-pane`
```

- [ ] **Step 4: Verify Section 2.1 TIMEOUT no longer immediately calls `capture-pane`**

```bash
grep -A 10 "TIMEOUT: agent may be stuck" skills/engram-tmux-lead/SKILL.md
```

Expected: first sub-step is "Read chat from cursor", not the tmux command.

- [ ] **Step 5: Verify the new HARD RULE section exists**

```bash
grep -n "Chat-First Diagnostics\|HARD RULE.*chat.*primary" skills/engram-tmux-lead/SKILL.md
```

Expected: at least one match.

- [ ] **Step 6: Pressure-test scenario A (mental walkthrough)**

Re-read against updated skill: `wait-ready` times out. Lead follows Section 1.5. Step 1 is now "read chat from cursor." Lead reads chat, sees `conversation` message from engram-agent (it's in a long tool call), waits. Correct behavior. ✓

- [ ] **Step 7: Pressure-test scenario B (mental walkthrough)**

Re-read against updated skill: Background watch for `done` returns TIMEOUT. Lead follows updated Section 2.1. Step 1 is "read chat from cursor." Lead reads chat, sees `type = "wait"` from executor to engram-agent (engram-agent hasn't ACKed yet). Lead engages with the WAIT instead of treating the agent as crashed. Correct behavior. ✓

- [ ] **Step 8: Post done with findings to chat**

(Agent-specific — handled outside the plan.)

---

## Self-Review Checklist

**Spec coverage:**
- ✓ Preamble ordering constraint updated (Change 1)
- ✓ HARD RULE section added (Change 2)
- ✓ Section 1.5 timeout path fixed (Change 3)
- ✓ Section 2.1 TIMEOUT path fixed (Change 4)
- ✓ TDD cycle: Group 10 RED assertions added to behavioral_test.sh → RED commit → GREEN SKILL.md edit → verification run
- ✓ File Map lists all 4 SKILL.md sites + behavioral_test.sh

**Placeholder scan:** No TBD or TODO items. All grep commands include expected output. All replacement text is fully written out. Group 10 assertions use exact strings present in the new SKILL.md text.

**Type consistency:** N/A — no code types. Variable names (`PANE_ID`, `CURSOR`, `PRE_SPAWN_CURSOR`, `CHAT_FILE`) match the existing skill's conventions exactly.
