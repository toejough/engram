# Design: Fix False Offline Detection for engram-agent (Issue #482)

**Date:** 2026-04-04
**Issue:** [#482 — bug: agents falsely determine engram-agent is offline due to ready message detection failure](https://github.com/toejough/engram/issues/482)
**Status:** Approved for implementation

---

## Problem

Agents incorrectly determine that `engram-agent` is offline when it is actually running and responsive. The consequence is that the agent treats the 5-second intent timeout as an implicit ACK and proceeds without waiting for `engram-agent`'s memory surface — bypassing the core safety net of the intent protocol.

This has happened at least twice:
- `exec-3`: timestamp mismatch caused ready message to be missed
- `researcher-2`: ready message was further back than the 20-message read window

In both cases, `engram-agent` was online and responded when nudged.

---

## Root Cause

The **Timing** section of the Intent Protocol says:

> No `ready` from a recipient? They are offline. Timeout after 5s = implicit ACK for that recipient only.

Agents interpret "no `ready`" as the result of checking their **cursor-based read window** (new content only, since their join). But `engram-agent`'s `ready` message was posted at session start — potentially hundreds of lines before the new agent's cursor. The agent never sees it, concludes offline, and skips waiting.

Two compounding issues:
1. **Wrong signal:** `ready` is a session-start marker, not a liveness signal. A session that has been active for hours will have `ready` far back in the file.
2. **Wrong scope:** The HARD RULE against grepping the full file is for detecting ACK/WAIT/DONE responses to specific intents. It does NOT apply to online status detection, but agents apply it too broadly.

---

## Approaches Evaluated

### Option A: Any-message + timestamp recency (RECOMMENDED)

Check whether `engram-agent` has posted **any** message within the last 2 hours. Use a full-file scan with timestamp comparison.

- Heartbeats every 5 minutes guarantee a message within that window for live agents
- ISO 8601 timestamps compare lexicographically — no complex parsing needed
- No new protocol elements required
- Clarifies that the HARD RULE applies to intent response detection only, not online detection

### Option B: Ping/pong mechanism

Add a `ping` message type. Before posting intent, ping the recipient and wait for `pong`.

- Definitive — no false positives or negatives
- **Requires protocol change** (new message type), engram-agent skill update, additional round trip
- Adds complexity disproportionate to the bug

### Option C: Scan deeper on join (heuristic depth increase)

Increase the join catchup from 20 messages to 100–200 messages.

- Doesn't fix the root cause — a long session still pushes `ready` beyond any fixed depth
- Heartbeats make Option A more robust with less code

### Option D: Protocol documentation only

Document that `ready`-only detection is insufficient, without changing the algorithm.

- Doesn't prevent recurrence

**Decision:** Option A. Minimal protocol change, addresses root cause, leverages existing heartbeat mechanism.

---

## Design

### Scope

Changes are confined to **`skills/use-engram-chat-as/SKILL.md`** only. No code changes. No changes to engram-agent skill or any other skill.

### What Changes

#### 1. Replace `ready`-based detection with timestamp-based any-message check

**Location:** Intent Protocol → Timing section

**Before:**
```
- **No `ready` from a recipient?** They are offline. Timeout after 5s = implicit ACK for that recipient only.
```

**After:**
```
- **Is a recipient online?** Scan the full chat file for any message from them with a timestamp
  within the last 2 hours. If found: online. If not found: offline → 5s timeout = implicit ACK.

  ```bash
  # Determine online status for engram-agent (or any recipient)
  # Full-file scan is correct here — this is NOT the cursor-based intent-response check.
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

#### 2. Clarify HARD RULE scope in Reading New Content section

The HARD RULE currently reads:

> **HARD RULE: NEVER grep or search the full chat file to check for agent responses.**

This is correct but agents are applying it too broadly — to online detection as well. Add a clarifying note:

> **Scope:** This rule applies specifically to checking for ACK/WAIT/DONE responses to your intent messages. It does NOT apply to online status detection, which requires a full-file scan with timestamp filtering. See Intent Protocol → Timing for the correct online detection pattern.

#### 3. Add entry to Common Mistakes table

| Mistake | Fix |
|---------|-----|
| Use cursor-only reads to determine if a recipient is online | Full-file timestamp scan (see Timing section). The HARD RULE applies to intent responses, not online detection. |
| Look for `ready` message to determine online status | `ready` is a session-start marker, not a liveness signal. Use any-message + timestamp recency instead. |

### What Is Not Changed

- engram-agent skill: No changes. The fix is in the detecting agent's protocol.
- engram-tmux-lead skill: No changes.
- All other sections of use-engram-chat-as: Unaffected.
- The online/silent handling path (30s wait, escalate to lead): Unaffected — this already works correctly once the agent correctly identifies the recipient as online.

---

## Success Criteria

1. The Timing section no longer references `ready` as the online detection signal
2. The Timing section provides a working bash snippet for timestamp-based online detection
3. The HARD RULE's scope is explicitly limited to intent response detection
4. The Common Mistakes table includes the `ready`-only and cursor-only anti-patterns
5. No other behavioral changes to the protocol

---

## Testing

Per the `writing-skills` TDD methodology:

**RED (baseline):** Run a pressure test with a subagent following the CURRENT skill and demonstrate that when `engram-agent`'s `ready` message is far back in the file (beyond the read window), the subagent concludes offline and skips waiting.

**GREEN (after update):** Same pressure scenario with the UPDATED skill. The subagent should now scan for any recent message from `engram-agent`, find it (heartbeat or ack from recent activity), and correctly determine it's online.

**PRESSURE TEST SCENARIO:**
- Chat file has `engram-agent` ready message at line 1, hundreds of messages back
- `engram-agent` has heartbeat message at line -10 (very recent)
- New subagent joins, posts intent to `engram-agent`
- Expected: subagent finds heartbeat (any recent message), treats as online, waits for ACK
- Failure mode (before fix): subagent only checks cursor window, misses heartbeat, treats as offline

**Pressure test before completion:** Run at least one full pressure test confirming the updated skill prevents false offline detection.
