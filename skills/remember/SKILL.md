---
name: remember
description: >
  Use when the user says "remember this", "remember that", "don't forget",
  "save this for later", or /remember. Captures explicit knowledge as
  feedback or fact memories with user approval.
---

The user wants to explicitly save something to memory.

## Flow

### Step 1: Self-query (agent-internal — do not show to user)

```bash
engram recall --memories-only --query "when to call /prepare or /learn in the current situation"
```

Internalize any guidance.

### Step 2: Analyze and classify

Determine what the user wants to remember. Classify as:
- **Feedback** (behavioral): situation → behavior → impact → action
- **Fact** (knowledge): situation → subject → predicate → object
- Could be **multiple** memories (e.g., "DI means Dependency Injection, not Do It" = two facts)

### Step 3: Draft and present to user

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

Ask the user to approve or edit the fields.

### Step 4: Save approved memories

For each approved memory, run:

```bash
# Feedback:
engram learn feedback --situation "..." --behavior "..." --impact "..." --action "..." --source human

# Fact:
engram learn fact --situation "..." --subject "..." --predicate "..." --object "..." --source human
```

### Step 5: Handle results

- **CREATED: <name>** — Confirm to user.
- **DUPLICATE: <name>** — The system already knows this. Trigger diagnostic (see below).
- **CONTRADICTION: <name>** — Present the conflict. Ask user: update existing, replace it, or keep both (use --no-dup-check)?

### Step 6: Duplicate diagnostic

When a duplicate is found, the system already knew this but failed to use it. Analyze:

1. **Was there a /recall or /prepare call this session that should have surfaced this?**
   - **Yes, but queries missed it:** Suggest additional queries that would have found it. Draft these as behavioral feedback memories with situations matching the self-query format (e.g., "how to prepare for <topic>") so future self-queries will find them. Present to user for approval.
   - **Yes, but memory wording too narrow:** Suggest a rewrite of the existing memory's situation field. Use `engram update --name <name> --situation "broader situation"` after user approval.

2. **No relevant /recall or /prepare call:**
   - Suggest a behavioral memory: "When <situation>, call /prepare before proceeding." Draft with a situation field matching the self-query format. Present to user for approval.
