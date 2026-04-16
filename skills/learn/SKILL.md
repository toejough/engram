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

### Step 2: Quality gate

Every candidate must pass **all three** or be dropped:

| Gate | Question | Fail → |
|------|----------|--------|
| **Recurs** | Will a future agent in a different session face this? | Don't persist — one-time event, phase-locked, or stale snapshot |
| **Actionable** | Does it change what an agent would DO? | Don't persist — vague observation, raw debug log, or inert fact |
| **Right home** | Is memory the only place for this? | Don't persist — belongs in code, docs, skills, or CLAUDE.md |

### Step 3: Draft memories

For each surviving candidate:
- Corrections/failures → feedback (SBIA)
- Knowledge/patterns → fact (situation + subject/predicate/object)

**Situation field:** Describe the task an agent would be embarking on when they need this. What are they trying to do? This must match how /prepare queries — an agent about to do this work should find this memory.

| Bad (bakes in the problem) | Good (describes the task) |
|---|---|
| "When fixing context cancellation in concurrent code" | "When implementing concurrent Go code with context" |
| "When checking Phase 2 implementation status" | "When verifying a multi-phase implementation is complete" |

**Litmus test:** If an agent called /prepare before this task, would this situation match their query? If the situation contains the diagnosis, symptom, or fix — it won't match. Strip it back to the task.

### Step 4: Persist

**Autonomous by default.** Mid-work, at task boundaries, or from subagents — persist immediately with `--source agent`. Continue working after.

**Interactive only when the user explicitly invokes /learn.** Present findings for approval before persisting.

```bash
# Feedback:
engram learn feedback --situation "..." --behavior "..." --impact "..." --action "..." --source agent

# Fact:
engram learn fact --situation "..." --subject "..." --predicate "..." --object "..." --source agent
```

### Step 5: Handle results

- **CREATED** — Done. If interactive, confirm to user.
- **DUPLICATE** — The system knew this but didn't use it. Diagnose: was there a /recall or /prepare that should have surfaced it? Fix the query pattern or broaden the existing memory's situation with `engram update --name <name> --situation "..."`. Never dismiss a duplicate — something failed to surface it.
- **CONTRADICTION** — If interactive, ask user: update, replace, or keep both? If autonomous, skip.
