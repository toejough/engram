---
name: recall
description: |
  Use when the user says "/recall", "what was I working on", "load previous
  context", "search session history", or wants to resume work from a previous
  session. Reads recent session transcripts and surfaces relevant memories.
---

# Session Recall

Load context from previous sessions in this project.

## Two Modes

- `/recall` (no query) — returns raw stripped transcript content. No LLM calls. You absorb it as context and present a concise summary to the user.
- `/recall <query>` — uses Haiku to extract content relevant to the query from each session.

## Execution

```bash
PROJECT_SLUG="$(echo "$PWD" | tr '/' '-')"
~/.claude/engram/bin/engram recall \
  --data-dir ~/.claude/engram/data \
  --project-slug="$PROJECT_SLUG"
```

If the user provided a query (e.g., `/recall keyword matching`), add `--query "<the query>"`.

## Handling Output

The command returns JSON: `{"summary":"...","memories":"..."}`.

**For plain `/recall`:** The `summary` field contains raw transcript content (not a summary despite the field name). Read it, absorb the full context, then present a concise recap focusing on:

1. **What tradeoffs were considered** — options weighed and why
2. **What decisions were made** — what was chosen
3. **What work got done** — commits, issues filed, changes pushed
4. **What is still outstanding** — open threads, deferred items, known gaps
5. **What state things were left in** — clean/dirty tree, passing/failing tests, waiting on something

Prioritize conclusions over discussions. The user needs to know how work *ended*, not everything that was talked about.

**For `/recall <query>`:** The `summary` field contains Haiku-extracted content relevant to the query.

In both cases, present any `memories` content as additional context.

If the command fails or returns empty, inform the user that no previous session data was found.
