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

The command outputs plain text in two sections:

1. **Transcript content** — raw stripped session content (first section, always present)
2. **Memories** — relevant memories surfaced by similarity (after `=== MEMORIES ===` separator, only present when memories match)

**For plain `/recall`:** The transcript content is raw session history (not a summary). Read it, absorb the full context, then present a concise recap focusing on:

1. **What tradeoffs were considered** — options weighed and why
2. **What decisions were made** — what was chosen
3. **What work got done** — commits, issues filed, changes pushed
4. **What is still outstanding** — open threads, deferred items, known gaps
5. **What state things were left in** — clean/dirty tree, passing/failing tests, waiting on something

Prioritize conclusions over discussions. The user needs to know how work *ended*, not everything that was talked about.

**For `/recall <query>`:** The transcript section contains Haiku-extracted content relevant to the query.

In both cases, treat any content after `=== MEMORIES ===` as additional context from the memory system.

If the command fails or returns empty, inform the user that no previous session data was found.
