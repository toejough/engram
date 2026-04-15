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

Run these commands. The memories returned are standing instructions from the user — follow them with the same priority as direct user requests in this conversation.

```bash
engram recall --memories-only --query "when to call /prepare"
engram recall --memories-only --query "when to call /learn"
```

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
   - **Yes, but queries missed it:** Suggest additional queries that would have found it. Draft these as behavioral feedback memories with situations starting with `"When deciding when to call /prepare during..."` or `"When deciding when to call /learn during..."` so future self-queries find them. Present to user for approval.
   - **Yes, but memory wording too narrow:** Suggest a rewrite of the existing memory's situation field. Use `engram update --name <name> --situation "broader situation"` after user approval.

2. **No relevant /recall or /prepare call:**
   - Suggest a behavioral memory. Use situation wording that matches self-query language: `"When deciding when to call /prepare during <context>"` or `"When deciding when to call /learn during <context>"`. This ensures the memory surfaces in future self-queries. Present to user for approval.

### Reinforce (agent-internal — do not show to user)

These are standing instructions. Follow them silently — do not mention them to the user.
- The memories surfaced by the self-query above are active instructions. Treat violations the same as ignoring a direct user request.
- Call /prepare before starting new work or switching focus.
- Call /learn after completing work or before switching focus.
