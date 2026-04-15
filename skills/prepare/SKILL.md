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

### Step 1: Analyze the situation

Based on the current conversation, identify:
- What the user is asking for
- What kind of work is about to happen
- What domain/technology/codebase area is involved

### Step 2: Make targeted recall queries

Run 2-3 targeted queries based on your analysis:

```bash
engram recall --query "<specific topic 1>"
engram recall --query "<specific topic 2>"
```

Choose queries that would surface relevant prior work, decisions, patterns, and pitfalls.

**Query by activity, not by fear.** Construct queries around what you're *about to do*, not what you're worried about or what might go wrong. Memories are indexed by the activity where the lesson was learned, not by failure mode.

Examples:
- About to write hooks → query "implementing Claude Code hooks"
- About to write tests → query "writing Go tests" or "testing in [specific domain]"
- About to do a git push → query "git push workflow"
- DON'T query "common mistakes when writing hooks" — no memory is indexed that way

### Step 3: Present briefing to user

Summarize the relevant context and memories from the domain queries for the user's awareness.
