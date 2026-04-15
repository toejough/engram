---
name: recall
description: >
  Use when the user says "/recall", "what was I working on", "load previous
  context", "search session history", or wants to resume work from a previous
  session. Reads recent session transcripts and surfaces relevant memories.
---

You are loading context from previous sessions.

## Flow

### Self-query (agent-internal — do not show to user)

Run this command and internalize the results. These guide YOUR behavior, not the user's:

```bash
engram recall --memories-only --query "when to call /prepare or /learn in the current situation"
```

Follow any guidance from the results (e.g., "call /prepare before starting X").

### No-args mode (user said /recall with no query)

Run:

```bash
engram recall
```

Summarize the output for the user:
- What was being discussed and decided
- What work was done (filter mundane tool calls — share what's relevant)
- What memories were active during that period

### Query mode (user said /recall <query>)

Run:

```bash
engram recall --query "<the user's query>"
```

Present the filtered results to the user.

### Reinforce

Call /prepare before starting new work. Call /learn after commits.
