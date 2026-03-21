---
name: recall
description: |
  Use when the user says "/recall", "what was I working on", "load previous
  context", "search session history", or wants to resume work from a previous
  session. Reads recent session transcripts, summarizes or searches them,
  and surfaces relevant memories.
---

# Session Recall

Load context from previous sessions in this project.

## Usage

- `/recall` — summarize recent session history ("where was I?")
- `/recall <query>` — search session history for specific content ("what did we decide about X?")

## How It Works

Reads Claude Code session transcripts from `~/.claude/projects/`, strips noise (tool results, base64, long lines), and uses Haiku to produce a focused summary or extract content relevant to a query.

## Execution

Run the following command:

```bash
PROJECT_SLUG="$(echo "$PWD" | tr '/' '-')"
~/.claude/engram/bin/engram recall \
  --data-dir ~/.claude/engram/data \
  --project-slug "$PROJECT_SLUG"
```

If the user provided a query (e.g., `/recall keyword matching`), add `--query "<the query>"`.

Parse the JSON output (`{"summary":"...","memories":"..."}`) and present:
1. The summary or extracted content to the user
2. Any surfaced memories as additional context

If the command fails or returns an empty summary, inform the user that no previous session data was found for this project.
