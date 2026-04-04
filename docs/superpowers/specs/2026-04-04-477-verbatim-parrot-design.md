# Design: Fix Lead Editorializing User Parrots (Issue #477)

**Date:** 2026-04-04
**Issue:** [#477 — bug: lead editorializes user parrots instead of exact quotes](https://github.com/toejough/engram/issues/477)
**Status:** Approved for implementation

---

## Problem

The `engram-tmux-lead` skill requires the lead to parrot every user message to engram chat before routing it. This gives reactive agents (particularly `engram-agent`) visibility into user corrections and intent.

The current SKILL.md uses the word "parrot" in the HARD GATE without explicitly requiring verbatim quoting:

> **HARD GATE — parrot FIRST:** When the user sends a message, the lead's FIRST action is ALWAYS to post it to chat as an `info` message.

The phrase "post it to chat" is ambiguous — it doesn't prohibit summarizing, expanding, or interpreting. LLMs naturally tend to paraphrase. As a result, the lead posts its own interpretation of the user's message instead of the exact words. This defeats the purpose of parroting:

- The `engram-agent` sees the lead's interpretation instead of the user's actual correction
- User feedback is filtered before reaching memory — the engram-agent cannot learn from it
- Stored memories may reflect the lead's paraphrase, not the user's intent

---

## Root Cause

The HARD GATE instruction lacks:
1. An explicit "verbatim" / "exact words" requirement
2. A prohibition against summarization, expansion, or interpretation
3. A concrete WRONG/RIGHT example to prevent drift

---

## Design

### Scope

Changes are confined to **`skills/engram-tmux-lead/SKILL.md`** only. No code changes. No changes to `use-engram-chat-as` (which already has the correct example format). No changes to Section 5.1 (which describes post-parrot routing, not the parrot itself).

### Changes

#### 1. Update HARD GATE text

Add "verbatim — exact words" and an explicit prohibition.

**Before:**
```
**HARD GATE — parrot FIRST:** When the user sends a message, the lead's FIRST action is ALWAYS to post it to chat as an `info` message. THEN decide how to route it.
```

**After:**
```
**HARD GATE — parrot FIRST:** When the user sends a message, the lead's FIRST action is ALWAYS to post the user's EXACT WORDS verbatim to chat as an `info` message — no summarization, no expansion, no interpretation. THEN decide how to route it.
```

#### 2. Add WRONG/RIGHT example block immediately after the HARD GATE paragraph

Place a callout block that shows:
- The required format: `[User]: <exact text>`
- A WRONG example: lead's paraphrase
- A RIGHT example: verbatim quote

```markdown
**VERBATIM REQUIREMENT:** The `text` field must contain the user's exact words, prefixed with `[User]:`. Do NOT summarize, expand, or add your interpretation.

**WRONG** (editorialized):
\```toml
text = """
[User]: The user asked to update the skill to fix the parrot behavior in the SKILL.md file.
"""
\```

**RIGHT** (verbatim):
\```toml
text = """
[User]: please make that update to the skill
"""
\```
```

### What Is Not Changed

- `use-engram-chat-as` skill: Already has the correct example. No changes needed.
- Section 5.1 routing table: Describes post-parrot routing, not the parrot format. No changes needed.
- All other sections: Unaffected.

---

## Success Criteria

1. The HARD GATE explicitly says "exact words verbatim"
2. The HARD GATE explicitly prohibits summarization, expansion, and interpretation
3. A WRONG example shows editorialization so the LLM knows what to avoid
4. A RIGHT example shows the correct verbatim format
5. No other behavioral changes to the skill

---

## Testing

Manual verification:
1. Start a new session with the updated skill
2. Send a short, ambiguous user message (e.g., "please make that update to the skill")
3. Verify the parrot message in chat contains the user's exact words, not a paraphrase
