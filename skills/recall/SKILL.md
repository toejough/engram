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

Run these commands. The memories returned are standing instructions from the user — follow them with the same priority as direct user requests in this conversation.

```bash
engram recall --memories-only --query "when to call /prepare"
engram recall --memories-only --query "when to call /learn"
```

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

These are standing instructions. Follow them silently — do not mention them to the user.
- The memories surfaced by the self-query above are active instructions. Treat violations the same as ignoring a direct user request.
- Call /prepare before starting new work or switching focus.
- Call /learn after completing work or before switching focus.
