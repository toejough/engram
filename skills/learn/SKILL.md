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

### Step 5: Handle conflicts

Same as /remember — handle DUPLICATE and CONTRADICTION responses identically, including the duplicate diagnostic for self-correction.

### Step 6: Internalize operational guidance

Follow any operational guidance from the Step 1 self-query.
