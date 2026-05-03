---
name: prepare
description: >
  REQUIRED before starting any new work, switching tasks, beginning a feature,
  changing direction, or tackling an issue. Call /prepare before implementation,
  debugging, or any significant new effort. Do NOT skip this step. Identifies
  the current situation and recalls relevant context and memories.
---

You are mentally preparing for upcoming work by loading relevant context.

**You MUST call this skill before starting any new work, switching tasks, or beginning implementation.** Do not skip this step even if you think you already know the context. Memories you haven't loaded may change how you approach the task.

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

Choose queries that describe the task you're about to do. Memory situations are written to match task descriptions, so query the same way.

**Query by task, not by fear.** What are you trying to do? Not what might go wrong.

Examples:
- About to write hooks → "implementing Claude Code hooks"
- About to write tests → "writing Go tests in [domain]"
- About to push → "git push workflow"
- DON'T query "common mistakes when writing hooks" — no memory is stored that way

### Step 3: Present briefing to user

Summarize the relevant context and memories from the domain queries for the user's awareness.
