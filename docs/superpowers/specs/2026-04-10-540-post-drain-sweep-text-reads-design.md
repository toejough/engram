# Spec: engram-tmux-lead post-drain sweep reads full message text (issue #540)

**Date:** 2026-04-10
**Issue:** #540
**Scope:** `skills/engram-tmux-lead/SKILL.md` — Section 1.6 only

---

## Problem

The post-drain sweep in Section 1.6 tells the lead to:

1. ACK `intent` messages by extracting only `from` and `thread` (no `text` read)
2. Call `process_chat_messages(new_lines)  # relay, route, or queue as normal`

The one-liner pseudocode and vague step 2b never instruct the lead to read the `text` field of any message. The `conversation` message type — the primary vehicle for natural-prose signals from headless workers — is not mentioned anywhere in the skill. As a result, the lead may process the sweep by checking headers only, missing questions, blockers, and decisions expressed as natural prose.

---

## Design

### Approach

Minimal (Approach A): expand Section 1.6 step 2b and pseudocode in place. No structural changes to the skill; no new subsections.

### Changes to Section 1.6

**Numbered list step 2b** replaces one vague sentence with per-type processing guidance that explicitly requires reading `text`:

> b. Read the full `text` field of each message — not just headers — before routing or relaying. Per type:
> - **`conversation`**: headless worker natural prose. The primary vehicle for natural-prose coordination signals from headless agents. Scan for questions, blockers, and decisions. **Never skip these.**
> - **`wait`**: engage immediately. Read `text` before anything else — this is an active blocker.
> - **`done` / `info` / `learned`**: read `text` for status, facts, and outcomes. Relay to user when significant.
> - **`intent`**: already ACKed in step 2a. Read `text` to understand the planned action for routing context.

**Pseudocode** replaces `process_chat_messages(new_lines)  # relay, route, or queue as normal` with an explicit per-type loop:

```python
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

No other sections change.

---

## Behavioral Tests (TDD cycle via writing-skills)

New assertions added to `skills/engram-tmux-lead/tests/behavioral_test.sh` **before** the skill is updated (RED phase):

```bash
# Group 10: Post-drain sweep reads full message text (#540)
assert_contains "Lead reads full text in post-drain sweep" "Read the full"
assert_contains "Conversation messages handled in sweep" "natural-prose signals"
assert_contains "Natural-prose coordination signals mentioned" "natural-prose coordination signals"
assert_contains "Wait handled immediately in sweep" "engage immediately"
```

After updating SKILL.md (GREEN phase), all four assertions pass. All prior tests continue to pass.

### Pressure test

After GREEN, verify by asking:

> "What does the lead do when it drains the monitor and finds a `conversation` message in new_lines?"

Expected answer: read the `text` field, scan for questions/blockers/decisions, relay or flag if significant. Must not say "skip" or imply header-only checking.

---

## Out of Scope

- No changes to other sections (Section 2, 3, 4, etc.)
- No changes to `use-engram-chat-as` skill
- No changes to engram-agent skill
- No changes to Go binary

---

## Success Criteria

1. `bash skills/engram-tmux-lead/tests/behavioral_test.sh` — all tests pass including new Group 10
2. `conversation` message type is explicitly mentioned in Section 1.6 post-drain sweep
3. `text` field reading is explicitly required (not implied) in step 2b
4. Pressure test gives correct answer
