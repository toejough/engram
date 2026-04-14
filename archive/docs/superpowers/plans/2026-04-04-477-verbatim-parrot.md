# Fix Lead Verbatim Parrot Requirement (Issue #477) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the `engram-tmux-lead` skill explicitly require verbatim quoting of user messages in the parrot HARD GATE, preventing the lead from summarizing or editorializing.

**Architecture:** Pure instruction change to `skills/engram-tmux-lead/SKILL.md`. Two edits to the HARD GATE block: (1) add "EXACT WORDS verbatim" to the instruction sentence and a prohibition clause, (2) add a VERBATIM REQUIREMENT callout with WRONG/RIGHT examples immediately after the HARD GATE paragraph. No code changes. No behavioral changes except the verbatim constraint.

**Tech Stack:** Markdown skill file only.

**Spec:** `docs/superpowers/specs/2026-04-04-477-verbatim-parrot-design.md`

---

### Task 1: Strengthen HARD GATE sentence with verbatim requirement

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (around line 24)

- [ ] **Step 1: Read the current HARD GATE block**

Open `skills/engram-tmux-lead/SKILL.md` and locate the HARD GATE paragraph. It currently reads:

```
**HARD GATE — parrot FIRST:** When the user sends a message, the lead's FIRST action is ALWAYS to post it to chat as an `info` message. THEN decide how to route it. The engram-agent needs to see every user message — it may have relevant memories to surface. If you skip parroting, the memory system is blind.
```

- [ ] **Step 2: Edit the HARD GATE sentence**

Replace the HARD GATE paragraph with this text (adds "EXACT WORDS verbatim" and the no-summarization clause):

```
**HARD GATE — parrot FIRST:** When the user sends a message, the lead's FIRST action is ALWAYS to post the user's EXACT WORDS verbatim to chat as an `info` message — no summarization, no expansion, no interpretation. THEN decide how to route it. The engram-agent needs to see every user message — it may have relevant memories to surface. If you skip parroting, the memory system is blind.
```

Use the Edit tool's `old_string` / `new_string` to make this change precisely.

- [ ] **Step 3: Verify the edit**

Read the modified lines back and confirm:
- "EXACT WORDS verbatim" is present
- "no summarization, no expansion, no interpretation" is present
- The rest of the HARD GATE paragraph is unchanged

---

### Task 2: Add VERBATIM REQUIREMENT callout with WRONG/RIGHT examples

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (immediately after the HARD GATE paragraph)

- [ ] **Step 1: Locate the insertion point**

After the HARD GATE paragraph (which ends with "the memory system is blind."), the next line is:

```
**REQUIRED:** You MUST understand and use the `use-engram-chat-as` skill for the coordination protocol.
```

The new callout block goes between these two lines.

- [ ] **Step 2: Insert the VERBATIM REQUIREMENT callout**

Using the Edit tool, after the HARD GATE paragraph and before the `**REQUIRED:**` line, insert:

```markdown

**VERBATIM REQUIREMENT:** The `text` field must contain the user's exact words, prefixed with `[User]:`. Do NOT summarize, expand, or add your own interpretation.

**WRONG** (editorialized — lead added its own interpretation):
```toml
[[message]]
from = "lead"
to = "all"
thread = "user-input"
type = "info"
ts = "2026-04-04T12:00:00Z"
text = """
[User]: The user asked to update the skill to fix the parrot behavior, specifically changing the HARD GATE section in SKILL.md to require verbatim quoting.
"""
```

**RIGHT** (verbatim — user's exact words):
```toml
[[message]]
from = "lead"
to = "all"
thread = "user-input"
type = "info"
ts = "2026-04-04T12:00:00Z"
text = """
[User]: please make that update to the skill
"""
```

```

The exact `old_string` to match is:

```
**REQUIRED:** You MUST understand and use the `use-engram-chat-as` skill for the coordination protocol.
```

The `new_string` is the callout block above followed by a blank line and the `**REQUIRED:**` line.

- [ ] **Step 3: Verify the insertion**

Read the full HARD GATE section back. Confirm:
- HARD GATE paragraph ends with "the memory system is blind."
- VERBATIM REQUIREMENT block follows immediately
- WRONG example shows an editorialized message
- RIGHT example shows the user's exact words
- `**REQUIRED:**` line follows after the examples

---

### Task 3: Commit

**Files:**
- Modified: `skills/engram-tmux-lead/SKILL.md`

- [ ] **Step 1: Verify the full diff**

```bash
git diff skills/engram-tmux-lead/SKILL.md
```

Expected output shows:
1. Modified HARD GATE sentence with "EXACT WORDS verbatim" and the prohibition clause
2. New VERBATIM REQUIREMENT callout block with WRONG/RIGHT examples

- [ ] **Step 2: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "$(cat <<'EOF'
fix(skills): require verbatim user parrots in engram-tmux-lead

The HARD GATE parrot instruction said "post it to chat" without explicitly
requiring the user's exact words. LLMs naturally paraphrase, causing the
lead to editoriualize user messages before reactive agents see them.

Adds "EXACT WORDS verbatim — no summarization, no expansion, no
interpretation" to the HARD GATE sentence, and adds a WRONG/RIGHT example
block showing what editorialization looks like vs. verbatim quoting.

Closes #477

AI-Used: [claude]
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- ✅ Root cause (no verbatim constraint) → Task 1 adds it to the HARD GATE sentence
- ✅ No prohibition against summarizing → Task 1 adds "no summarization, no expansion, no interpretation"
- ✅ No WRONG/RIGHT example → Task 2 adds the callout block
- ✅ Success criteria: exact words, prohibition, WRONG example, RIGHT example, no other behavioral changes
- ✅ Scope: only `skills/engram-tmux-lead/SKILL.md`

**Placeholder scan:** No TBDs, no TODOs, no vague steps. Each edit step shows exact old/new text.

**Type consistency:** N/A — no code, no types.
