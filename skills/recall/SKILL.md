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

Run these commands and internalize the results. These guide YOUR behavior, not the user's:

```bash
engram recall --memories-only --query "when to call /prepare"
engram recall --memories-only --query "when to call /learn"
```

Follow any guidance from the results.

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

### Reinforce (agent-internal — do not show to user)

Internalize these reminders silently. Do not mention them to the user.
- Call /prepare before starting new work or switching focus.
- Call /learn after completing work or before switching focus.
