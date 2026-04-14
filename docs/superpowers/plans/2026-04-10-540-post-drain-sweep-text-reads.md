# Issue #540: post-drain sweep reads full message text Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> **REQUIRED SUB-SKILL: superpowers:writing-skills** — CLAUDE.md mandates this skill for every SKILL.md edit. No exceptions.

**Goal:** Fix `skills/engram-tmux-lead/SKILL.md` so the lead explicitly reads the full `text` field — including `conversation` type messages — during post-drain sweeps, not just message headers.

**Architecture:** Writing-skills TDD cycle: add failing behavioral tests (RED), update skill text (GREEN), verify all tests pass (VERIFY). Two files change: the test script gets new assertions, the skill gets an expanded step 2b and pseudocode block in Section 1.6.

**Tech Stack:** Bash behavioral tests (`grep -qF` pattern matching), SKILL.md prose editing.

---

### Task 1: Add failing RED behavioral tests (Group 10)

**Files:**
- Modify: `skills/engram-tmux-lead/tests/behavioral_test.sh`

- [ ] **Step 1: Run the existing tests to establish a clean baseline**

```bash
bash skills/engram-tmux-lead/tests/behavioral_test.sh
```

Expected: All existing tests pass. Record the PASS count — it should stay the same or go up after this fix; it must never go down.

- [ ] **Step 2: Add Group 10 assertions to the test file**

Open `skills/engram-tmux-lead/tests/behavioral_test.sh`. After the last `echo ""` before the `=== Results ===` summary block (after Group 9), add:

```bash
# ---------------------------------------------------------------------------
# Group 10: Post-drain sweep reads full message text (#540)
# ---------------------------------------------------------------------------

echo "--- Group 10: Post-drain sweep reads full message text ---"

assert_contains "Lead reads full text in post-drain sweep" "Read the full"
assert_contains "Conversation messages handled in sweep" "natural-prose signals"
assert_contains "Natural-prose coordination signals mentioned" "natural-prose coordination signals"
assert_contains "Wait handled immediately in sweep" "engage immediately"

echo ""
```

- [ ] **Step 3: Run the tests — confirm all 4 new assertions FAIL**

```bash
bash skills/engram-tmux-lead/tests/behavioral_test.sh
```

Expected output includes:
```
--- Group 10: Post-drain sweep reads full message text ---
FAIL: Lead reads full text in post-drain sweep [pattern not found: 'Read the full']
FAIL: Conversation messages handled in sweep [pattern not found: 'natural-prose signals']
FAIL: Natural-prose coordination signals mentioned [pattern not found: 'natural-prose coordination signals']
FAIL: Wait handled immediately in sweep [pattern not found: 'engage immediately']
```

All prior tests still pass. Exit code 1 (expected — RED phase).

- [ ] **Step 4: Commit the RED tests**

```bash
git add skills/engram-tmux-lead/tests/behavioral_test.sh
git commit -m "test(engram-tmux-lead): add RED tests for post-drain sweep full text reads (#540)

AI-Used: [claude]"
```

---

### Task 2: Update SKILL.md Section 1.6 — step 2b prose (GREEN)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md`

- [ ] **Step 1: Read Section 1.6 to locate the exact text to replace**

Search for the current step 2b text:

```bash
grep -n "Process the chat message" skills/engram-tmux-lead/SKILL.md
```

Expected: one match near line 166 containing:
```
   b. Process the chat message — relay questions to the user, handle agent status updates, route tasks, etc.
```

- [ ] **Step 2: Replace step 2b with the per-type processing guide**

In `skills/engram-tmux-lead/SKILL.md`, replace:

```
   b. Process the chat message — relay questions to the user, handle agent status updates, route tasks, etc.
```

With:

```
   b. Read the full `text` field of each message — not just headers — before routing or relaying. Per type:
      - **`conversation`**: headless worker natural prose. The primary vehicle for natural-prose coordination signals from headless agents. Scan for questions, blockers, and decisions. **Never skip these.**
      - **`wait`**: engage immediately. Read `text` before anything else — this is an active blocker.
      - **`done` / `info` / `learned`**: read `text` for status, facts, and outcomes. Relay to user when significant.
      - **`intent`**: already ACKed in step 2a. Read `text` to understand the planned action for routing context.
```

- [ ] **Step 3: Run Group 10 tests — confirm 3 of the 4 assertions now PASS**

```bash
bash skills/engram-tmux-lead/tests/behavioral_test.sh 2>&1 | grep -A1 "Group 10"
```

Expected (assertion 2 remains RED — `natural-prose signals` is only added in Task 3's pseudocode):
```
--- Group 10: Post-drain sweep reads full message text ---
PASS: Lead reads full text in post-drain sweep
FAIL: Conversation messages handled in sweep [pattern not found: 'natural-prose signals']
PASS: Natural-prose coordination signals mentioned
PASS: Wait handled immediately in sweep
```

If assertions 1, 3, or 4 fail, check the exact string in the skill (use `grep -n "Read the full" skills/engram-tmux-lead/SKILL.md`) and adjust phrasing until they pass. Assertion 2 will pass after Task 3.

---

### Task 3: Update SKILL.md Section 1.6 — expand pseudocode (GREEN continued)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md`

- [ ] **Step 1: Locate the pseudocode one-liner to replace**

```bash
grep -n "process_chat_messages" skills/engram-tmux-lead/SKILL.md
```

Expected: **two matches** — one near line 187 (Section 1.6 post-drain loop) and one near line 892 (Section 6.1 drain-before-spawn pattern). Both show the same one-liner:
```
    process_chat_messages(new_lines)   # relay, route, or queue as normal
```

Both need to be replaced — they are the same instructional pseudocode with the same bug.

- [ ] **Step 2: Replace both one-liners with the per-type loop**

In `skills/engram-tmux-lead/SKILL.md`, use `replace_all: true` (or two sequential edits) to replace every occurrence of:

```
    process_chat_messages(new_lines)   # relay, route, or queue as normal
```

With:

```
    # Read full text of each message — not just headers:
    for msg in toml_blocks(new_lines):
        if msg.type == "conversation":
            relay_or_flag_if_significant(msg.from_, msg.text)  # natural-prose signals
        elif msg.type == "wait":
            handle_wait_immediately(msg.from_, msg.thread, msg.text)
        elif msg.type in ("done", "info", "learned"):
            relay_or_route_if_significant(msg.from_, msg.text)
        elif msg.type == "intent":
            pass  # already ACKed above; text informs routing context
```

Both occurrences (lines 187 and 892) must be replaced.

- [ ] **Step 3: Verify both pseudocode replacements are clean**

```bash
grep -n "process_chat_messages" skills/engram-tmux-lead/SKILL.md
```

Expected: **zero matches** (both one-liners are gone).

```bash
grep -n "relay_or_flag_if_significant" skills/engram-tmux-lead/SKILL.md
```

Expected: exactly **two matches** — one in the Section 1.6 pseudocode block and one in the Section 6.1 block.

---

### Task 4: Full test suite verification (VERIFY)

**Files:** None changed — verification only.

- [ ] **Step 1: Run the complete behavioral test suite**

```bash
bash skills/engram-tmux-lead/tests/behavioral_test.sh
```

Expected: `All tests passed.` Exit code 0. FAIL count = 0.

If any prior test broke, read the failure message carefully. The only changes were to step 2b and the pseudocode — prior tests match different phrases and should be unaffected. If a prior test fails, the skill edit accidentally changed other text; undo and redo the edit more carefully.

- [ ] **Step 2: Pressure test — read the updated skill and answer**

Read `skills/engram-tmux-lead/SKILL.md` Section 1.6. Answer:

> "What does the lead do when it drains the monitor and finds a `conversation` message in new_lines?"

Expected answer: calls `relay_or_flag_if_significant(msg.from_, msg.text)` — reads the full `text` field, treats it as natural-prose coordination signal from a headless agent, relays or flags if significant. Must not say "skip" or reference only headers.

- [ ] **Step 3: Commit the GREEN skill update**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(engram-tmux-lead): read full message text in post-drain sweep (#540)

Expands Section 1.6 step 2b from a vague one-liner into explicit per-type
guidance. conversation messages (headless worker natural prose) are now
explicitly named and their text required to be read. Pseudocode one-liner
replaced with per-type loop showing msg.text access.

AI-Used: [claude]"
```

---

## Self-Review

**Spec coverage:**
- ✓ Step 2b expanded with per-type guide — Task 2
- ✓ `conversation` type explicitly named — Task 2 + Task 3
- ✓ `text` field reading required — Task 2 + Task 3
- ✓ RED tests added before GREEN — Task 1
- ✓ Full test suite VERIFY — Task 4
- ✓ Pressure test — Task 4 Step 2
- ✓ Commit after RED — Task 1 Step 4
- ✓ Commit after GREEN — Task 4 Step 3

**Placeholder scan:** No TBDs. All steps show exact strings, exact commands, exact expected output.

**Type consistency:** `toml_blocks`, `relay_or_flag_if_significant`, `handle_wait_immediately`, `relay_or_route_if_significant` are pseudocode stubs — consistent across Task 3 and the spec. Not real function definitions, just illustrative.

**Occurrence count:** `process_chat_messages` appears twice in SKILL.md (lines 187 and 892). Task 3 replaces both. Verification expects zero matches after replacement and two matches for `relay_or_flag_if_significant`.

**Intermediate test state:** After Task 2 only, 3 of 4 Group 10 assertions pass. Assertion 2 (`natural-prose signals`) passes only after Task 3 adds the pseudocode comment. Task 2 Step 3 expected output reflects this (3 pass / 1 fail).
