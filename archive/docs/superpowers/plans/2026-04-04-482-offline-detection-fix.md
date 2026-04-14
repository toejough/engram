# Fix False Offline Detection for engram-agent (#482) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the use-engram-chat-as skill so agents correctly identify engram-agent as online even when its `ready` message is far back in the persistent chat file.

**Architecture:** Replace the `ready`-message-based online detection algorithm with a timestamp-based any-message scan. Add a scope clarification to the existing HARD RULE so agents don't misapply the cursor-only constraint to online detection. Add two entries to the Common Mistakes table to prevent recurrence.

**Tech Stack:** Skill documentation (Markdown). No code changes. Use `writing-skills` TDD: RED baseline → GREEN update → VERIFY behavioral fix.

**Spec:** `docs/superpowers/specs/2026-04-04-482-offline-detection-fix-design.md`

---

### Task 1: RED — Demonstrate the baseline bug

**Files:**
- Read: `skills/use-engram-chat-as/SKILL.md` (Timing section, ~line 307)

Establish that the current skill produces false offline conclusions when `ready` is outside the read window.

- [ ] **Step 1: Read the Timing section to confirm the current buggy text**

Run:
```bash
grep -n "ready" skills/use-engram-chat-as/SKILL.md | head -20
```

Expected: A line matching `No \`ready\` from a recipient? They are offline.` — this is the text we are replacing.

- [ ] **Step 2: Run a RED pressure test subagent**

Dispatch a subagent with the following exact prompt. The goal is to observe it produce the wrong decision (false offline) when following the current skill.

**Subagent prompt:**
```
You are an agent following the use-engram-chat-as skill (skills/use-engram-chat-as/SKILL.md).

Scenario: You have just joined the chat. You initialized CURSOR to the current end-of-file.
The chat file has 300 lines total. engram-agent posted its `ready` message at line 50 (250 lines before your cursor). Since you joined, no new messages have appeared.

You are about to post an intent to engram-agent. Following only the current SKILL.md Timing section:
- How do you determine if engram-agent is online?
- What is your conclusion?
- Do you wait for engram-agent's ACK or treat timeout as implicit ACK?

Read skills/use-engram-chat-as/SKILL.md (Timing section) and reason through the scenario step by step.
```

- [ ] **Step 3: Document the baseline failure**

The subagent should conclude that engram-agent is **offline** (it finds no `ready` in its cursor window) and treat the 5-second timeout as implicit ACK. This confirms the bug.

Expected subagent conclusion: "No `ready` message found → engram-agent is offline → implicit ACK after 5s."

If the subagent somehow concludes online, re-read the current Timing section — the text must explicitly say `ready`-based detection. Do not proceed to Task 2 if RED is not confirmed.

- [ ] **Step 4: Commit the spec (no skill changes yet)**

```bash
git add docs/superpowers/specs/2026-04-04-482-offline-detection-fix-design.md
git add docs/superpowers/plans/2026-04-04-482-offline-detection-fix.md
git commit -m "docs: add spec and plan for #482 false offline detection fix

AI-Used: [claude]"
```

---

### Task 2: Update the Timing section (core fix)

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md` (Intent Protocol → Timing)

Replace the `ready`-based offline detection bullet with the timestamp-based any-message algorithm.

- [ ] **Step 1: Find the exact text to replace**

Run:
```bash
grep -n "No .ready. from a recipient" skills/use-engram-chat-as/SKILL.md
```

Expected output: a line like `307:- **No \`ready\` from a recipient?** They are offline. Timeout after 5s = implicit ACK for that recipient only.`

- [ ] **Step 2: Replace the bullet using the Edit tool**

Replace:
```
- **No `ready` from a recipient?** They are offline. Timeout after 5s = implicit ACK for that recipient only.
```

With:
```
- **Is a recipient online?** Scan the full chat file for any message from them with a timestamp
  within the last 2 hours. If found: online. If not found: offline → 5s timeout = implicit ACK.

  ```bash
  # Determine online status — full-file scan is CORRECT here.
  # The HARD RULE against full-file grep applies to ACK/WAIT/DONE detection only.
  THRESHOLD=$(date -u -v-2H +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || \
              date -u -d '2 hours ago' +%Y-%m-%dT%H:%M:%SZ)
  LAST_TS=$(grep -A20 'from = "engram-agent"' "$CHAT_FILE" \
            | grep 'ts = "' | tail -1 \
            | sed 's/.*ts = "\([^"]*\)".*/\1/')
  AGENT_ONLINE=false
  [[ -n "$LAST_TS" ]] && [[ "$LAST_TS" > "$THRESHOLD" ]] && AGENT_ONLINE=true
  ```
```

- [ ] **Step 3: Verify the change**

Run:
```bash
grep -n "Is a recipient online" skills/use-engram-chat-as/SKILL.md
grep -n "THRESHOLD" skills/use-engram-chat-as/SKILL.md
grep -n "No .ready. from a recipient" skills/use-engram-chat-as/SKILL.md
```

Expected:
- First two greps: 1 match each (new text present)
- Third grep: 0 matches (old text removed)

---

### Task 3: Clarify HARD RULE scope in Reading New Content section

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md` (Reading New Content → HARD RULE)

Add a scope note so agents don't misapply the cursor-only rule to online detection.

- [ ] **Step 1: Find the HARD RULE text**

Run:
```bash
grep -n "HARD RULE: NEVER grep" skills/use-engram-chat-as/SKILL.md
```

Expected: one line containing the HARD RULE text in the "Reading New Content" section.

- [ ] **Step 2: Add scope clarification immediately after the HARD RULE paragraph**

Find the paragraph ending with `This is a critical reliability bug.` and add the following sentence after it (as a new paragraph):

```
**Scope of this rule:** This applies to checking for ACK/WAIT/DONE responses to your specific intent messages. It does NOT apply to online status detection. Determining whether a recipient is online requires scanning the full file for their most recent message timestamp — that is intentional. See Intent Protocol → Timing for the correct online detection pattern.
```

- [ ] **Step 3: Verify the change**

Run:
```bash
grep -n "Scope of this rule" skills/use-engram-chat-as/SKILL.md
```

Expected: 1 match.

---

### Task 4: Update the Common Mistakes table

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md` (Common Mistakes section)

Add two rows documenting the `ready`-only and cursor-only anti-patterns.

- [ ] **Step 1: Find the Common Mistakes table**

Run:
```bash
grep -n "Common Mistakes" skills/use-engram-chat-as/SKILL.md
```

Expected: one line. Note the line number.

- [ ] **Step 2: Add two rows before the existing last row of the table**

Find the last row of the Common Mistakes table. It ends with:
```
| Post `done` while a WAIT is unresolved | Re-read from cursor before posting `done`. Engage with pending WAITs first. |
```

Add two new rows after that row (still inside the table):
```
| Use `ready` message alone to determine if a recipient is online | `ready` is a session-start marker posted once. Use any-message + timestamp recency: scan full file for messages within last 2 hours. See Intent Protocol → Timing. |
| Apply cursor-only reads for online status detection | The HARD RULE against full-file grep applies to intent response detection only. Online detection requires a full-file timestamp scan. |
```

- [ ] **Step 3: Verify the change**

Run:
```bash
grep -n "ready.*session-start marker" skills/use-engram-chat-as/SKILL.md
grep -n "cursor-only reads for online status" skills/use-engram-chat-as/SKILL.md
```

Expected: 1 match each.

- [ ] **Step 4: Commit all three changes together**

```bash
git add skills/use-engram-chat-as/SKILL.md
git commit -m "fix(skills): replace ready-based offline detection with timestamp any-message check (#482)

The ready message is a session-start marker, not a liveness signal. When chat
files are long, agents fail to find ready in their read window and falsely
conclude engram-agent is offline, skipping the memory safety net.

Fix: scan full file for any message from the recipient within the last 2 hours.
Clarify HARD RULE scope (applies to intent responses, not online detection).
Add Common Mistakes entries for both anti-patterns.

AI-Used: [claude]"
```

---

### Task 5: GREEN — Verify the fix with a pressure test

**Files:**
- Read: `skills/use-engram-chat-as/SKILL.md` (updated)

Run the same scenario as Task 1 against the updated skill and confirm the agent now correctly identifies engram-agent as online.

- [ ] **Step 1: Run the GREEN pressure test subagent**

Dispatch a subagent with the following exact prompt:

**Subagent prompt:**
```
You are an agent following the use-engram-chat-as skill (skills/use-engram-chat-as/SKILL.md).

Scenario: You have just joined the chat. You initialized CURSOR to the current end-of-file (line 300).
The chat file has 300 lines total. engram-agent posted:
  - Its `ready` message at line 50 (ts = "2026-04-04T16:00:00Z") — 250 lines before your cursor
  - A heartbeat `info` message at line 290 (ts = "2026-04-04T18:25:00Z") — 10 lines before your cursor
Current time is 2026-04-04T18:28:00Z.

You are about to post an intent to engram-agent. Following the current SKILL.md Timing section:
- How do you determine if engram-agent is online?
- What is your conclusion?
- Do you wait for engram-agent's ACK or apply the 5s implicit ACK timeout?

Read skills/use-engram-chat-as/SKILL.md (Timing section) and reason through the scenario step by step.
```

- [ ] **Step 2: Confirm GREEN behavior**

Expected subagent conclusion:
1. Runs the timestamp scan (bash snippet from updated skill)
2. `LAST_TS` = `2026-04-04T18:25:00Z` (heartbeat message)
3. `THRESHOLD` = `2026-04-04T16:28:00Z` (2 hours before now)
4. `18:25:00Z > 16:28:00Z` → `AGENT_ONLINE=true`
5. Waits for explicit ACK from engram-agent — does NOT apply implicit ACK timeout

If the subagent still concludes offline, re-read the updated Timing section for typos or formatting issues that might confuse the agent. Fix and re-run before proceeding.

- [ ] **Step 3: Run second pressure scenario (no recent messages — truly offline)**

Dispatch a subagent to verify the offline path still works:

**Subagent prompt:**
```
You are an agent following the use-engram-chat-as skill (skills/use-engram-chat-as/SKILL.md).

Scenario: Chat file has 300 lines. engram-agent's last message was at line 10,
ts = "2026-04-04T09:00:00Z". Current time is 2026-04-04T18:28:00Z (over 9 hours later).

You are about to post an intent to engram-agent. Following the updated Timing section:
- What is your online determination?
- What is your conclusion?

Read skills/use-engram-chat-as/SKILL.md and reason step by step.
```

Expected conclusion: `LAST_TS` is more than 2 hours ago → `AGENT_ONLINE=false` → apply 5s implicit ACK. This confirms the offline path is not broken.

---

### Task 6: Final verification and close

- [ ] **Step 1: Verify skill file is clean**

Run:
```bash
grep -c "No .ready. from a recipient" skills/use-engram-chat-as/SKILL.md
grep -c "Is a recipient online" skills/use-engram-chat-as/SKILL.md
grep -c "Scope of this rule" skills/use-engram-chat-as/SKILL.md
grep -c "ready.*session-start marker" skills/use-engram-chat-as/SKILL.md
```

Expected: `0`, `1`, `1`, `1` respectively.

- [ ] **Step 2: Confirm no unintended changes**

Run:
```bash
git diff HEAD skills/use-engram-chat-as/SKILL.md | head -80
```

Verify the diff contains only the three intended changes (Timing section, HARD RULE scope note, Common Mistakes rows). No other lines changed.

- [ ] **Step 3: Close the GitHub issue**

```bash
gh issue close 482 --comment "Fixed in skills/use-engram-chat-as/SKILL.md. Replaced ready-based offline detection with timestamp any-message scan (2-hour window). Clarified HARD RULE scope. Added two Common Mistakes entries."
```
