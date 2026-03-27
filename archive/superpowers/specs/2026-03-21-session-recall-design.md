# Session Resume Redesign

## Goal

Replace the per-turn rolling summary (session-context.md) with an opt-in `/recall` skill that reads session transcripts on demand. Two modes: no-args summarizes recent sessions ("where was I?"); with-args searches session history for content relevant to a query ("what did we decide about X?"). Session start becomes lightweight — no memory surfacing, no context injection.

## Problem

The current session-context feature writes a Haiku-generated rolling summary every turn but only reads it at session start. Three issues:

1. **Hard truncation** — 1024-byte cap cuts summaries mid-word
2. **Low-signal content** — Haiku generates emoji-heavy status reports; useful content (next steps, blockers) gets truncated
3. **Always-on overhead** — Haiku API call every turn, memory surfacing every session start, even when unwanted

The deeper issue: most of the value is redundant with git log + memories. The unique value (ephemeral working state) is poorly captured by a rolling summary.

## Design

### Architecture change

| Component | Before | After |
|---|---|---|
| Stop hook (every turn) | `flush: learn + context-update` | `flush: learn` only |
| Session start | Surface memories + inject session context | Show `/recall` notification + run `maintain` |
| `/recall` skill (new) | N/A | Read transcripts, summarize/search, surface memories |
| PreToolUse/PostToolUse | Surface memories (unchanged) | Surface memories (unchanged) |

### 1. `/recall` skill

A new engram skill invoked explicitly by the user. Two modes:

#### Mode A: No arguments — `/recall`

"Where was I?" — summarize recent session history.

1. **Find recent sessions** — Glob `~/.claude/projects/{project-slug}/*.jsonl`, sort by mtime descending. No staleness cutoff (a project revisited after 6 months still benefits from context).
2. **Read and strip** — Read sessions newest-first, apply existing `Strip()` logic (remove toolResult blocks, base64, truncate long lines). Stop when 50KB of stripped content is accumulated or all sessions are read.
3. **Summarize via Haiku** — Send stripped content with a focused prompt (see below). Output budget: ~1500 bytes.
4. **Surface memories** — Pass the summary as the query to `engram surface`.
5. **Inject both** — Return summary + surfaced memories as `additionalContext`.

#### Mode B: With arguments — `/recall <query>`

"What did we discuss about X?" — search session history for relevant content.

1. **Find sessions** — Same glob, sorted by mtime descending.
2. **Iterative extract loop** — For each session, newest-first:
   a. Read and strip the session transcript.
   b. Send to Haiku with the query: "Extract only content relevant to: `<query>`. Return relevant excerpts verbatim or tightly paraphrased. Return nothing if irrelevant."
   c. Append Haiku's output to an accumulator.
   d. If accumulator >= 1500 bytes, stop. Otherwise, continue to next session.
3. **Surface memories** — Pass `<query>` as the query to `engram surface`.
4. **Inject both** — Return accumulated relevant content + surfaced memories as `additionalContext`.

### 2. Summarization prompt

Replace the current vague prompt:

```
Update this task-focused working summary. Focus on what's being worked on,
decisions made, progress, and open questions.
```

With a structured, budget-aware prompt:

```
Summarize these session transcripts for someone resuming work on this project.
Prioritize in this order:
1. What was being worked on and current status
2. Open questions and blockers
3. Key decisions made and why
4. What was attempted but didn't work

Use plain text. No emoji. No markdown headers or formatting. Keep it under
1500 bytes — concise sentences, not a report.
```

### 3. Session start hook changes

Remove from `session-start.sh`:
- `engram surface --mode session-start` call
- Session context file reading and injection

Keep:
- `engram maintain` (triage signals — cheap, useful for housekeeping)
- Build-if-stale logic
- Symlink creation

Add:
- Notification: `[engram] Say /recall to load context from previous sessions, or /recall <query> to search session history.`

### 4. New `recall` CLI subcommand

Add `engram recall` subcommand to the CLI:
- New `case "recall":` in the dispatch switch in `cli.go`
- Accepts `--data-dir`, `--project-slug`, `--query` flags
- Reads the OAuth token from keychain (same pattern as other Haiku callers)
- Orchestrates: find sessions → read+strip → summarize → surface → output JSON
- Output format: `{"summary": "...", "memories": "..."}` for the skill to inject

New skill file: `skills/recall/SKILL.md` — the engram plugin skill that invokes the subcommand and formats the output as `additionalContext`. Follows the existing directory pattern (`skills/memory-triage/SKILL.md`).

### 5. Flush pipeline simplification

Remove `context-update` from flush. Inline `FlushRunner` — with only one step (`learn`), the two-step abstraction adds no value. `runFlush` calls `RunLearn` directly.

Code deletions:
- `internal/context/orchestrate.go` — delete entirely
- `internal/context/delta.go` — delete entirely
- `internal/context/file.go` — delete entirely
- `internal/context/summarize.go` — delete entirely
- `internal/context/context.go` — remove `HaikuClient` interface and session-context types; keep `Strip()` or move it to `internal/transcript/`
- `internal/cli/flush.go` — inline `FlushRunner`; delete `FlushRunner` struct and `NewFlushRunner`
- `internal/cli/targets.go` — delete `ContextUpdateArgs` struct entirely; remove `ContextPath` field from `FlushArgs` and its wiring at lines ~183 and ~213
- `internal/cli/cli.go` — remove `runContextUpdate` function, `contextSummarizationPrompt` constant, `MaxSummaryBytes` constant, `haikuClientAdapter` (repurpose for `recall` subcommand)

Test deletions:
- `internal/context/context_test.go` — delete tests for orchestrator, delta reader, session file, summarizer (all of: TestOrchestrator_*, TestT134-T157, TestRead_*, TestWrite_*, TestDeltaReader_*). Keep `Strip()` tests if `Strip()` is retained.
- `internal/cli/cli_test.go` — delete context-update tests (T-160 and any haikuClientAdapter coverage)
- `internal/cli/flush_test.go` — delete/rewrite tests for `--context-path` flag path (lines ~136, ~182-205)

### 6. Cleanup

- Delete all `session-context.md` files from `~/.claude/engram/data/projects/*/`
- Remove `--context-path` flag from `hooks/stop.sh`
- Update README session lifecycle documentation

## Data flow

```
User types /recall [optional query]
    |
    v
Skill invokes: engram recall --data-dir ... --project-slug ... [--query "..."]
    |
    v
Find sessions: glob ~/.claude/projects/{slug}/*.jsonl, sort by mtime
    |
    +--- No query (mode A) --------+--- With query (mode B) ------+
    |                               |                              |
    v                               v                              |
Read + strip: newest-first,     For each session:                  |
accumulate up to 50KB stripped    Read + strip                     |
    |                               |                              |
    v                               v                              |
Haiku summarize:                Haiku extract: "content relevant   |
focused prompt, ~1500 bytes     to <query>?" Append to accumulator |
    |                               |                              |
    |                             Loop until 1500 bytes or no more |
    |                               |                              |
    +-------------------------------+                              |
    |                                                              |
    v                                                              |
Surface memories: use summary/query as input                       |
    |                                                              |
    v                                                              |
Return to skill: context + memories as additionalContext            |
```

## What stays the same

- Memory surfacing in PreToolUse/PostToolUse hooks (conversation-context-driven)
- `engram learn` in flush pipeline (memory extraction from transcripts)
- `engram maintain` at session start (triage signals)
- Memory feedback tracking
- All memory CRUD operations

## Risks and mitigations

| Risk | Mitigation |
|---|---|
| User forgets `/recall` exists | Session-start notification reminds them |
| 50KB budget too small for long projects | Budget is configurable; 50KB is starting point |
| Haiku summarization adds latency to `/recall` | Acceptable — explicit opt-in, user expects it |
| Losing session-context.md breaks something | Audit all consumers before deleting |
| `Strip()` function needs new home | Move to `internal/transcript/` or keep in `internal/context/` |

## Out of scope

- Changing memory surfacing in PreToolUse/PostToolUse hooks
- Changing the memory data model
- Changing `engram learn` behavior
- Archival/rotation of old session transcripts (Claude Code manages these)
