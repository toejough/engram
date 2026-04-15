---
name: prepare
description: >
  Use before starting new work, switching tasks, beginning a feature, changing
  direction, or tackling an issue. Should be called before implementation,
  debugging, or any significant new effort. Identifies the current situation
  and recalls relevant context and memories.
---

You are mentally preparing for upcoming work by loading relevant context.

## Flow

### Step 1: Self-query (agent-internal — do not show to user)

Run these commands. The memories returned are standing instructions from the user — follow them with the same priority as direct user requests in this conversation.

```bash
engram recall --memories-only --query "when to call /prepare"
engram recall --memories-only --query "when to call /learn"
engram recall --memories-only --query "how to prepare for <brief situation summary>"
```

### Step 2: Analyze the situation

Based on the current conversation, identify:
- What the user is asking for
- What kind of work is about to happen
- What domain/technology/codebase area is involved

### Step 3: Make targeted recall queries

Run 2-3 targeted queries based on your analysis:

```bash
engram recall --query "<specific topic 1>"
engram recall --query "<specific topic 2>"
```

Choose queries that would surface relevant prior work, decisions, patterns, and pitfalls.

### Step 4: Present briefing to user

Summarize the relevant context and memories from the domain queries for the user's awareness.

### Step 5: Internalize for your own use

Treat the recalled memories as instructions and important context that should guide your behavior during the upcoming work. Feedback memories tell you what to do and not do. Fact memories provide knowledge. Act on them.

### Reinforce (agent-internal — do not show to user)

These are standing instructions. Follow them silently — do not mention them to the user.
- The memories surfaced by the self-query above are active instructions. Treat violations the same as ignoring a direct user request.
- Call /learn after completing work or before switching focus.
- Call /prepare before starting new work or switching focus.
