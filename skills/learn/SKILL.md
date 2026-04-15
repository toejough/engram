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

### Step 1: Self-query (agent-internal — do not show to user)

```bash
engram recall --memories-only --query "how to review sessions for learnable moments, and when to call /prepare or /learn"
```

Internalize any guidance.

### Step 2: Load session context

```bash
engram recall
```

Review the output for learnable moments:
- **User corrections** — the user told you to do something differently
- **Failed approaches** — something was tried and didn't work
- **Discovered facts** — new knowledge about the codebase, tools, or domain
- **Patterns** — recurring behaviors that should be codified

### Step 3: Draft findings

For each learnable moment, draft a memory:
- Corrections/failures → feedback (SBIA)
- Knowledge/patterns → fact (situation + subject/predicate/object)

Present all findings to the user for approval. Each should have all fields filled.

### Step 4: Save approved memories

For each approved memory:

```bash
# Feedback:
engram learn feedback --situation "..." --behavior "..." --impact "..." --action "..." --source agent

# Fact:
engram learn fact --situation "..." --subject "..." --predicate "..." --object "..." --source agent
```

Note: source is agent because these are agent-identified learnings, not explicit user instructions.

### Step 5: Handle results

For each `engram learn` response:

- **CREATED: \<name\>** — Confirm to user. Done.
- **DUPLICATE: \<name\>** — The system already knew this but failed to use it. Run the duplicate diagnostic (Step 6). Do NOT skip this.
- **CONTRADICTION: \<name\>** — Present the conflict. Ask user: update existing, replace it, or keep both (use --no-dup-check)?

### Step 6: Duplicate diagnostic (REQUIRED on every DUPLICATE)

A duplicate means the memory existed but didn't fire when it mattered. Diagnose WHY:

1. **Was there a /recall or /prepare call this session that should have surfaced this?**
   - **Yes, but queries missed it:** Suggest additional queries that would have found it. Draft these as behavioral feedback memories with situations starting with `"When deciding when to call /prepare during..."` or `"When deciding when to call /learn during..."` so future self-queries find them. Present to user for approval.
   - **Yes, but memory situation wording too narrow:** The memory existed but its situation field didn't match the actual scenario. Suggest a rewrite of the existing memory's situation field. Use `engram update --name <name> --situation "broader situation"` after user approval.

2. **No relevant /recall or /prepare call:**
   - Suggest a behavioral memory. Use situation wording that matches self-query language: `"When deciding when to call /prepare during <context>"` or `"When deciding when to call /learn during <context>"`. This ensures the memory surfaces in future self-queries. Present to user for approval.

**Never dismiss a duplicate as "correct — no update needed."** If it was truly known, something failed to surface it. Find what.

### Step 7: Internalize operational guidance

Follow any operational guidance from the Step 1 self-query.

### Reinforce (agent-internal — do not show to user)

Internalize these reminders silently. Do not mention them to the user.
- Call /prepare before starting new work.
- Call /learn again after your next commit.
