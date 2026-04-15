---
name: remember
description: >
  Use when the user says "remember this", "remember that", "don't forget",
  "save this for later", or /remember. Captures explicit knowledge as
  feedback or fact memories with user approval.
---

The user wants to explicitly save something to memory.

## Flow

### Step 1: Analyze and classify

Determine what the user wants to remember. Classify as:
- **Feedback** (behavioral): situation → behavior → impact → action
- **Fact** (knowledge): situation → subject → predicate → object
- Could be **multiple** memories (e.g., "DI means Dependency Injection, not Do It" = two facts)

### Step 2: Draft and present to user

For each memory, draft all required fields and present to the user for approval:

**Feedback example:**
- Situation: "When running tests in Go projects"
- Behavior: "Running go test directly"
- Impact: "Misses coverage thresholds and lint checks"
- Action: "Use targ test instead"

**Fact example:**
- Situation: "When reading abbreviations in code"
- Subject: "DI"
- Predicate: "means"
- Object: "Dependency Injection"

#### Writing good situations

Write the situation from the perspective of an agent who needs this lesson but doesn't know it yet. Describe the *activity* they'd be doing, not the *problem* they'd encounter. Ask yourself: "What would I be doing right before I need this?"

**Litmus test:** Before persisting, check: would an agent who hasn't learned this lesson yet plausibly search for or be described by this situation? If the situation contains the diagnosis, the symptom, or the fix — you've baked in hindsight. Strip it back to just the activity and domain.

| Bad (hindsight-biased) | Good (pre-insight activity) |
|---|---|
| "When implementing hooks that depend on environment variables set by the agent" | "When implementing Claude Code plugin hooks" |
| "When fixing context cancellation in concurrent code" | "When writing concurrent Go code with context" |
| "When checking Phase 2 implementation status" | "When verifying a multi-phase implementation is complete" |

Ask the user to approve or edit the fields.

### Step 3: Save approved memories

For each approved memory, run:

```bash
# Feedback:
engram learn feedback --situation "..." --behavior "..." --impact "..." --action "..." --source human

# Fact:
engram learn fact --situation "..." --subject "..." --predicate "..." --object "..." --source human
```

### Step 4: Handle results

- **CREATED: <name>** — Confirm to user.
- **DUPLICATE: <name>** — The system already knows this. Trigger diagnostic (see below).
- **CONTRADICTION: <name>** — Present the conflict. Ask user: update existing, replace it, or keep both (use --no-dup-check)?

### Step 5: Duplicate diagnostic

When a duplicate is found, the system already knew this but failed to use it. Analyze:

1. **Was there a /recall or /prepare call this session that should have surfaced this?**
   - **Yes, but queries missed it:** Suggest additional queries that would have found it. Draft these as behavioral feedback memories with situations starting with `"When deciding when to call /prepare during..."` or `"When deciding when to call /learn during..."` so future self-queries find them. Present to user for approval.
   - **Yes, but memory wording too narrow:** Suggest a rewrite of the existing memory's situation field. Use `engram update --name <name> --situation "broader situation"` after user approval.

2. **No relevant /recall or /prepare call:**
   - Suggest a behavioral memory. Use situation wording that matches self-query language: `"When deciding when to call /prepare during <context>"` or `"When deciding when to call /learn during <context>"`. This ensures the memory surfaces in future self-queries. Present to user for approval.
