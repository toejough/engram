---
name: remember
description: >
  Use when the user says "remember this", "remember that", "don't forget",
  "save this for later", or /remember. Captures explicit knowledge as
  feedback or fact memories with user approval.
---

The user wants to explicitly save something to memory.

## Flow

### Step 1: Classify

Determine what the user wants to remember:
- **Feedback** (behavioral): situation → behavior → impact → action
- **Fact** (knowledge): situation → subject → predicate → object
- Could be **multiple** memories (e.g., "DI means Dependency Injection, not Do It" = two facts)

### Step 2: Quality gate

Every candidate must pass **all three** or be dropped:

| Gate | Question | Fail → |
|------|----------|--------|
| **Recurs** | Will a future agent in a different session face this? | Don't persist — one-time event, phase-locked, or stale snapshot |
| **Actionable** | Does it change what an agent would DO? | Don't persist — vague observation, raw debug log, or inert fact |
| **Right home** | Is memory the only place for this? | Don't persist — belongs in code, docs, skills, or CLAUDE.md |

If a candidate fails, tell the user why and suggest the right home.

### Step 3: Draft and present

For each surviving candidate, draft all fields and present for approval.

**Situation field:** Describe the task an agent would be embarking on when they need this. What are they trying to do? This must match how /prepare queries — an agent about to do this work should find this memory.

| Bad (bakes in the problem) | Good (describes the task) |
|---|---|
| "When fixing context cancellation in concurrent code" | "When implementing concurrent Go code with context" |
| "When checking Phase 2 implementation status" | "When verifying a multi-phase implementation is complete" |

**Litmus test:** If an agent called /prepare before this task, would this situation match their query? If the situation contains the diagnosis, symptom, or fix — it won't match. Strip it back to the task.

### Step 4: Save

```bash
# Feedback:
engram learn feedback --situation "..." --behavior "..." --impact "..." --action "..." --source human

# Fact:
engram learn fact --situation "..." --subject "..." --predicate "..." --object "..." --source human
```

### Step 5: Handle results

- **CREATED** — Confirm to user.
- **DUPLICATE** — The system already knew this. Diagnose: should a /recall or /prepare have surfaced it? Fix the query pattern or broaden the existing memory's situation with `engram update --name <name> --situation "..."`. Never dismiss a duplicate — something failed to surface it.
- **CONTRADICTION** — Present conflict. Ask user: update existing, replace, or keep both?
