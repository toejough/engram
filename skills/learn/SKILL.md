---
name: learn
description: >
  Use after completing a task, finishing work, changing direction, or when the
  user says "review what we learned" or /learn. Should be called after
  implementation, after resolving a bug, after completing a plan step.
  Reviews the recent session for learnable moments.
---

You are reviewing the recent session for things worth remembering.

## Flow

### Step 1: Identify learnable moments

Review your current session for:
- **User corrections** — the user told you to do something differently
- **Failed approaches** — something was tried and didn't work
- **Discovered facts** — new knowledge about the codebase, tools, or domain
- **Patterns** — recurring behaviors that should be codified

### Step 2: Draft and save memories

For each learnable moment, draft a memory:
- Corrections/failures → feedback (SBIA)
- Knowledge/patterns → fact (situation + subject/predicate/object)

**Autonomous by default.** When called mid-work, at a task boundary, or by a subagent, draft and persist memories immediately — do not stop to ask the user. The `--source agent` flag marks these as agent-identified, which is the correct provenance. Continue your work after persisting.

**Interactive review only when the user explicitly invokes /learn** (e.g., "review what we learned", end-of-session reflection). In that case, present findings for approval before persisting.

### Step 3: Persist memories

For each memory (approved or autonomously drafted):

```bash
# Feedback:
engram learn feedback --situation "..." --behavior "..." --impact "..." --action "..." --source agent

# Fact:
engram learn fact --situation "..." --subject "..." --predicate "..." --object "..." --source agent
```

Note: source is agent because these are agent-identified learnings, not explicit user instructions.

### Step 4: Handle results

For each `engram learn` response:

- **CREATED: \<name\>** — Done. If interactive, confirm to user.
- **DUPLICATE: \<name\>** — The system already knew this but failed to use it. Run the duplicate diagnostic (Step 5). Do NOT skip this.
- **CONTRADICTION: \<name\>** — If interactive, present the conflict and ask user: update existing, replace it, or keep both? If autonomous, skip the contradicting memory and move on.

### Step 5: Duplicate diagnostic (REQUIRED on every DUPLICATE)

A duplicate means the memory existed but didn't fire when it mattered. Diagnose WHY:

1. **Was there a /recall or /prepare call this session that should have surfaced this?**
   - **Yes, but queries missed it:** Draft behavioral feedback memories with situations starting with `"When deciding when to call /prepare during..."` or `"When deciding when to call /learn during..."` so future self-queries find them. If interactive, present to user for approval. If autonomous, persist directly.
   - **Yes, but memory situation wording too narrow:** The memory existed but its situation field didn't match the actual scenario. Suggest a rewrite of the existing memory's situation field. Use `engram update --name <name> --situation "broader situation"`. If interactive, get user approval first.

2. **No relevant /recall or /prepare call:**
   - Draft a behavioral memory. Use situation wording that matches self-query language: `"When deciding when to call /prepare during <context>"` or `"When deciding when to call /learn during <context>"`. If interactive, present to user for approval. If autonomous, persist directly.

**Never dismiss a duplicate as "correct — no update needed."** If it was truly known, something failed to surface it. Find what.
