# Fix ACK-wait Subagent Sleep-Polling and Cursor Timing (#502) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> **REQUIRED:** Use `superpowers:writing-skills` skill for all SKILL.md edits. No exceptions.

**Goal:** Fix two bugs in `use-engram-chat-as` skill text that cause ACK-wait subagents to use sleep-polling instead of fswatch and to initialize their cursor after ACK arrival.

**Architecture:** Pure skill text change — two edits to `skills/use-engram-chat-as/SKILL.md`. No Go code, no binary changes. Change 1: expand Intent Protocol step 2 with a HARD RULE and ACK-wait subagent template. Change 2: add two Common Mistakes rows.

**Tech Stack:** Markdown, `superpowers:writing-skills` skill for TDD on skill edits.

---

### Task 1: Establish baseline behavior (RED phase)

**Files:**
- Read: `skills/use-engram-chat-as/SKILL.md` lines 268–286 (Intent Protocol > The Flow)
- Read: `skills/use-engram-chat-as/SKILL.md` lines 688–718 (Common Mistakes table)

- [ ] **Step 1: Read and record current Intent Protocol step 2 text**

```bash
sed -n '268,286p' skills/use-engram-chat-as/SKILL.md
```

Expected output — current step 2 reads:
```
2. Wait for explicit responses from ALL TO recipients:
   - Spawn a background ACK-wait Agent: watch CHAT_FILE from current cursor for ACK/WAIT
     from each expected recipient, applying the online/offline timing rules below
   - ONLY proceed when every TO recipient has responded (ACK or WAIT)
   ...
```

Note: No cursor-capture-before-intent instruction. No fswatch requirement. No subagent template.

- [ ] **Step 2: Confirm Common Mistakes table lacks both new rows**

```bash
grep -n "ACK-wait" skills/use-engram-chat-as/SKILL.md
grep -n "re-derive cursor" skills/use-engram-chat-as/SKILL.md
```

Expected: Both commands return no matches. This confirms the baseline state — neither bug is explicitly called out in the mistakes table.

- [ ] **Step 3: Commit baseline checkpoint**

```bash
git add --intent-to-add skills/use-engram-chat-as/SKILL.md
# (no changes yet — this is just a checkpoint note; skip commit if no staged changes)
```

---

### Task 2: Update Intent Protocol step 2 (GREEN phase — Bug 1 + Bug 2)

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md` — Intent Protocol > The Flow, step 2

- [ ] **Step 1: Invoke `superpowers:writing-skills` skill**

This skill enforces TDD for skill edits. Follow its guidance throughout this task.

- [ ] **Step 2: Locate the exact text to replace**

```bash
grep -n "Spawn a background ACK-wait Agent" skills/use-engram-chat-as/SKILL.md
```

Expected: matches a line near line 272. Note the line number.

- [ ] **Step 3: Replace step 2 in the Intent Protocol flow**

In `skills/use-engram-chat-as/SKILL.md`, find this block (inside "### The Flow"):

```
2. Wait for explicit responses from ALL TO recipients:
   - Spawn a background ACK-wait Agent: watch CHAT_FILE from current cursor for ACK/WAIT
     from each expected recipient, applying the online/offline timing rules below
   - ONLY proceed when every TO recipient has responded (ACK or WAIT)
   - Offline exception: if a recipient has NOT posted any message in the last 15 min
     (scan full file), treat timeout as implicit ACK for that recipient only, after 5s
   - Online + silent: if a recipient has posted a message within the last 15 min but
     is silent after 5s, post info noting no response; wait up to 30s, then escalate to lead
```

Replace with:

```
2. Wait for explicit responses from ALL TO recipients:
   - **BEFORE posting the intent message**, capture the current line count:
     `CURSOR=$(wc -l < "$CHAT_FILE")`
     Embed this integer as a literal in the ACK-wait subagent prompt (see template below).
     Never let the subagent re-derive cursor at startup — the ACK may already be written by then.
   - Post intent message (with lock, fresh ts)
   - Spawn a background ACK-wait Agent using this template:

     ```
     ACK-wait monitor for [AGENT_NAME]'s intent.
     CHAT_FILE: /absolute/path/to/chat.toml   ← literal path, not a shell variable
     CURSOR: 12345                              ← literal integer captured BEFORE intent was posted
     RECIPIENTS: engram-agent, reviewer        ← exact names from the TO field

     1. Run foreground bash: fswatch -1 "$CHAT_FILE"
        (Linux: inotifywait -e modify "$CHAT_FILE")
        ← MUST use fswatch. NEVER use sleep polling.
     2. Read new lines: tail -n +$((CURSOR + 1)) "$CHAT_FILE"
     3. Find blocks where `from` is one of RECIPIENTS and `type` is "ack" or "wait"
     4. Advance cursor: CURSOR=$(wc -l < "$CHAT_FILE")
     5. If all recipients found: return ACK|CURSOR or WAIT|from|CURSOR|text
     6. If partial (some but not all): go back to step 1 with advanced cursor
     ```

   **HARD RULE: ACK-wait subagents MUST use `fswatch -1`, never `sleep` loops.**
   Sleep polling causes up to N × sleep-interval seconds of delay per ACK wait.
   An ACK that arrives BEFORE the subagent initializes its cursor is silently lost —
   the tail window is set past it. Both are prevented by: (a) fswatch for immediacy,
   and (b) cursor captured before intent for full coverage.

   - ONLY proceed when every TO recipient has responded (ACK or WAIT)
   - Offline exception: if a recipient has NOT posted any message in the last 15 min
     (scan full file), treat timeout as implicit ACK for that recipient only, after 5s
   - Online + silent: if a recipient has posted a message within the last 15 min but
     is silent after 5s, post info noting no response; wait up to 30s, then escalate to lead
```

- [ ] **Step 4: Verify the edit looks correct**

```bash
grep -n -A 35 "BEFORE posting the intent message" skills/use-engram-chat-as/SKILL.md
```

Expected: Shows the full expanded step 2 block with HARD RULE and template. Confirm:
- "BEFORE posting the intent message" appears
- `fswatch -1` appears in the template
- "MUST use fswatch. NEVER use sleep polling." appears
- "HARD RULE" block appears after the template
- Offline/online timing rules still present below

---

### Task 3: Add Common Mistakes rows (GREEN phase — observability)

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md` — Common Mistakes table

- [ ] **Step 1: Locate the end of the Common Mistakes table**

```bash
grep -n "Reusing a cached TS variable" skills/use-engram-chat-as/SKILL.md
```

Note the line number of that row (the current last row).

- [ ] **Step 2: Add two new rows after the last existing row**

After the line containing `"Reusing a cached TS variable across messages"`, add:

```markdown
| Let ACK-wait subagent re-derive cursor at startup | **Critical bug**: ACK posted between intent-post and subagent-init is silently lost. Capture `CURSOR=$(wc -l < "$CHAT_FILE")` BEFORE posting the intent message, then embed as an integer literal in the subagent prompt. |
| Use `sleep` polling in ACK-wait subagent | Same rule as the main monitor: use `fswatch -1` (or `inotifywait` on Linux). Sleep polling causes multi-minute delays; an ACK between poll intervals is delayed, and an ACK before cursor init is permanently lost. |
```

- [ ] **Step 3: Verify the two new rows are present**

```bash
grep -n "ACK-wait subagent re-derive" skills/use-engram-chat-as/SKILL.md
grep -n "sleep.*polling.*ACK-wait" skills/use-engram-chat-as/SKILL.md
```

Expected: Both commands return one match each, in the Common Mistakes section.

---

### Task 4: Behavior verification (verify GREEN)

**Files:**
- Read: `skills/use-engram-chat-as/SKILL.md`

- [ ] **Step 1: Verify Bug 1 (sleep-polling) is addressed**

```bash
grep -n "NEVER use sleep" skills/use-engram-chat-as/SKILL.md
grep -n "HARD RULE.*ACK-wait" skills/use-engram-chat-as/SKILL.md
```

Expected: Both return one match each, in the Intent Protocol section.

- [ ] **Step 2: Verify Bug 2 (cursor timing) is addressed**

```bash
grep -n "BEFORE posting the intent message" skills/use-engram-chat-as/SKILL.md
grep -n "literal integer captured BEFORE" skills/use-engram-chat-as/SKILL.md
```

Expected: Both return one match each, in the Intent Protocol section.

- [ ] **Step 3: Verify Common Mistakes table additions**

```bash
grep -c "ACK-wait" skills/use-engram-chat-as/SKILL.md
```

Expected: 3 or more matches (one in the template/HARD RULE, two in Common Mistakes).

- [ ] **Step 4: Verify no regressions — check surrounding structure is intact**

```bash
grep -n "### The Flow\|HARD RULE\|Offline exception\|Online + silent\|Common Mistakes" skills/use-engram-chat-as/SKILL.md
```

Expected: "The Flow" heading, all HARD RULE callouts, offline/online timing rules, and "Common Mistakes" heading all present. No duplicated section headers.

---

### Task 5: Commit

**Files:**
- Modified: `skills/use-engram-chat-as/SKILL.md`

- [ ] **Step 1: Stage changes**

```bash
git diff --stat skills/use-engram-chat-as/SKILL.md
git add skills/use-engram-chat-as/SKILL.md
```

Expected diff stat: ~40–60 lines added, 7 lines removed (the step 2 replacement + 2 table rows).

- [ ] **Step 2: Commit**

```bash
git commit -m "$(cat <<'EOF'
fix(use-engram-chat-as): prevent sleep-polling and cursor-timing bugs in ACK-wait subagents (#502)

- Add HARD RULE and fswatch subagent template to Intent Protocol step 2
- Require cursor capture before posting intent (not inside subagent at startup)
- Add two Common Mistakes rows for both failure modes

AI-Used: [claude]
EOF
)"
```

Expected: commit succeeds, no hook failures.

---

## Self-Review

**Spec coverage:**
- Bug 1 (sleep-polling): addressed in Task 2 (HARD RULE + template) and Task 3 (Common Mistakes row) ✓
- Bug 2 (cursor timing): addressed in Task 2 (cursor-before-intent note) and Task 3 (Common Mistakes row) ✓
- Both bugs have verification steps in Task 4 ✓

**Placeholder scan:** No TBDs, no "fill in later", all code blocks have actual content. ✓

**Type consistency:** N/A — no code types. All grep patterns match the exact text being added. ✓
