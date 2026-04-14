# Design: Fix ACK-wait Subagent Sleep-Polling and Cursor Timing (Issue #502)

**Date:** 2026-04-04
**Issue:** [#502](https://github.com/toejough/engram/issues/502)
**Scope:** Skill text changes to `skills/use-engram-chat-as/SKILL.md` only. No Go code, no binary changes.

---

## Problem

Two distinct bugs in how agents construct their internal ACK-wait subagents when implementing the Intent Protocol:

### Bug 1: sleep-polling instead of fswatch -1

Agents spawn ACK-wait subagents that use `sleep 30` polling loops instead of `fswatch -1`. The skill's "Background Monitor Pattern" section correctly specifies `fswatch -1`, but agents building ACK-wait subagents from the Intent Protocol step 2 don't connect to that section and reinvent polling.

**Impact:** Up to N × 30 seconds wasted per ACK wait. In observed case: 8+ minutes.

### Bug 2: Cursor initialized after ACK arrival

ACK-wait subagents re-derive `CURSOR=$(wc -l < "$CHAT_FILE")` at the moment they start, not at the moment the intent was posted. If the ACK was written between intent-post and subagent-startup, the cursor window is set past the ACK and the subagent never sees it.

**Impact:** ACK silently lost. Subagent polls indefinitely with no termination path.

---

## Root Cause

The Intent Protocol section (step 2) says "Spawn a background ACK-wait Agent" but does not show the subagent prompt template. Agents construct this from scratch and:
- Reinvent sleep-polling instead of reusing `fswatch -1`
- Capture cursor inside the subagent prompt (at subagent startup) instead of before posting intent

The relevant guidance exists in separate sections ("Background Monitor Pattern", "Reading New Content") but the connection is not made explicit where agents look when implementing the intent protocol.

---

## Design

### Approach: Template + HARD RULE + Common Mistakes (recommended)

Address both bugs at three levels — rule, example, self-check — so agents skimming, reading carefully, or reviewing their work all encounter the guidance.

### Change 1: Intent Protocol > The Flow — step 2 expansion

**Current text (step 2):**
```
2. Wait for explicit responses from ALL TO recipients:
   - Spawn a background ACK-wait Agent: watch CHAT_FILE from current cursor for ACK/WAIT
     from each expected recipient, applying the online/offline timing rules below
```

**New text:**

Replace step 2 with the following expanded version that:
- Adds explicit "Capture CURSOR before posting intent" note (fixes Bug 2)
- Adds a HARD RULE block prohibiting sleep-polling and late cursor capture (fixes both bugs)
- Adds a copy-pasteable ACK-wait subagent template with `fswatch -1` and literal cursor (fixes both bugs)

```
2. Wait for explicit responses from ALL TO recipients:
   - **BEFORE posting the intent message**, capture the current line count:
     `CURSOR=$(wc -l < "$CHAT_FILE")`
     This integer is embedded as a literal in the ACK-wait subagent prompt (see template below).
     Never let the subagent re-derive cursor at startup — the ACK may already be written by then.
   - Post intent message (with lock, fresh ts)
   - Spawn a background ACK-wait Agent using this template:

     ```
     ACK-wait monitor for [AGENT_NAME]'s intent.
     CHAT_FILE: /absolute/path/to/chat.toml   ← literal path, not a variable
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
   Sleep polling causes up to N × sleep-interval seconds of delay per ACK wait, and ACKs
   that arrive between polls are silently delayed — not missed, but the delay compounds.
   An ACK that arrives BEFORE the subagent initializes its cursor is silently lost entirely
   (the window is set past it). Both are prevented by: (a) fswatch for immediacy, and
   (b) cursor captured before intent for coverage.
```

### Change 2: Common Mistakes table — two new rows

Add at the end of the Common Mistakes table:

| Mistake | Fix |
|---------|-----|
| Let ACK-wait subagent re-derive cursor at startup | **Critical bug**: ACK posted between intent-post and subagent-init is silently lost. Capture `CURSOR=$(wc -l < "$CHAT_FILE")` BEFORE posting the intent message, then embed as an integer literal in the subagent prompt. |
| Use `sleep` polling in ACK-wait subagent | Same rule as the main monitor: use `fswatch -1` (or `inotifywait` on Linux). Sleep polling causes multi-minute delays and missed ACKs when the ACK arrives between poll intervals. |

---

## Scope

- **Files changed:** `skills/use-engram-chat-as/SKILL.md` only
- **No behavior changes** to Go binary, other skills, or tests
- **Tests:** Per `superpowers:writing-skills`, baseline behavior test before edit, verify behavioral change after. Test by reading the updated section and confirming the two bugs are addressed.

---

## Self-Review

- No TBDs or placeholders
- No contradictions — the new template is consistent with Background Monitor Pattern
- Scope is narrow: two text blocks in one file
- No ambiguity: the HARD RULE and template leave no room for alternative interpretations
