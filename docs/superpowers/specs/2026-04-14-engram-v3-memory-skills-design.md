# Engram v3: Memory Skills Redesign

## Overview

Engram becomes a focused memory system with four skills and a lean binary. The multi-agent coordination layer (chat protocol, agent spawning, tmux orchestration) is dropped entirely. What remains: skills that help an agent prepare for work, recall context, learn from experience, and remember things explicitly — backed by a Go binary that handles search, filtering, and storage.

### Core Loop

```
/prepare (before work) → do work → /learn (after work)
                ↑                         |
                └── memories improve ─────┘
```

### Architecture

```
Skills (agent-facing)     Binary (computation)        Storage (filesystem)
─────────────────────     ────────────────────        ────────────────────
/recall                   engram recall               session transcripts
/prepare                  engram learn feedback       memory files
/learn                    engram learn fact             ├── feedback/*.toml
/remember                 engram update                └── facts/*.toml
                          engram show
                          engram list
```

Skills call whichever binary commands they need — the mapping is many-to-many. Binary reads session transcripts (read-only) and reads/writes memory files.

## Binary Commands

### `engram recall`

Search session transcripts and/or memories.

**Flags:**

- `--query <string>` — filter results for relevance (triggers Haiku). Omit for chronological summary.
- `--memories-only` — search only memory files, skip transcripts.
- `--limit <int>` — max memories to return (default 10).
- `--data-dir <string>` — memory storage root (defaults to XDG engram path).

**No-query mode:** Returns recent session context chronologically:

- Stripped transcript content (as today).
- Tool call summaries from those sessions (see Transcript Parsing below).
- Memories created or updated within the time window of the included session transcripts.

Memories are included per-session-log: for each transcript file included, memories with `created_at` or `updated_at` within that session's time range are surfaced alongside it. This uses multiple narrow windows rather than one spanning window, so a 6-month hiatus doesn't flood context with months of unrelated memories. Transcript inclusion uses the existing byte budget from the current implementation.

**Query mode:** Same candidate set, filtered by Haiku for relevance to the query. Returns matching results. No special treatment of any memory type — a memory is a memory, whether it's domain knowledge or operational guidance.

**`--memories-only` mode:** Searches only memory files against the query. Used by skills to self-query for operational guidance.

**Memory ranking:** Human-sourced first, then agent-sourced. Within each group, most recent first (`updated_at`).

### `engram list`

Returns all memories as compact index entries for efficient Haiku scanning.

**Output format:**

```
feedback | use-targ-for-tests | When running build commands in the engram project
fact     | engram-uses-targ   | engram uses targ for all build operations
```

One line per memory: type, name, situation. Used internally by `recall` and `learn` for Haiku's two-phase search (scan situations first, load full content of matches only).

### `engram learn feedback`

Create a feedback memory with duplicate/contradiction detection.

**Required flags:** `--situation`, `--behavior`, `--impact`, `--action`, `--source` (human|agent)
**Optional flags:** `--no-dup-check`

**Default behavior:** Before writing, sends the new memory's situation and content to Haiku along with existing memory situations (via `engram list`). Haiku checks for:

- **Duplicates** — same situation, same lesson. Does NOT write. Returns the existing memory.
- **Contradictions** — same/overlapping situation, conflicting behavior/action. Does NOT write. Returns the conflicting memory.

Output format for conflicts:

```
DUPLICATE: <name>
situation: ...
behavior: ...
impact: ...
action: ...

CONTRADICTION: <name>
situation: ...
behavior: ...
impact: ...
action: ...
```

The type prefix (`DUPLICATE` or `CONTRADICTION`) tells the calling skill how to handle it.

**With `--no-dup-check`:** Writes immediately, returns the new memory path.

### `engram learn fact`

Create a fact memory with duplicate/contradiction detection.

**Required flags:** `--situation`, `--subject`, `--predicate`, `--object`, `--source` (human|agent)
**Optional flags:** `--no-dup-check`

Same dedup/contradiction behavior as `learn feedback`.

### `engram update`

Update an existing memory.

**Required flags:** `--name <string>` — memory name (slug).
**Optional flags:** `--data-dir <string>` (defaults to XDG engram path), plus field flags matching the memory type (situation, behavior, subject, etc.). Updates only the fields provided, preserves the rest. Updates `updated_at` timestamp.

### `engram show`

Display a memory. Unchanged from current implementation.

**Required flags:** `--name <string>` — memory name (slug).
**Optional flags:** `--data-dir <string>` (defaults to XDG engram path).

## Memory Format

```toml
schema_version = 2
type = "feedback"
situation = "When running tests in Go projects"
source = "human"

[content]
behavior = "Running go test directly"
impact = "Misses coverage and lint checks"
action = "Use targ test instead"

created_at = "2026-04-14T10:00:00Z"
updated_at = "2026-04-14T10:00:00Z"
```

```toml
schema_version = 2
type = "fact"
situation = "When building or testing the engram project"
source = "agent"

[content]
subject = "DI"
predicate = "means"
object = "Dependency Injection"

created_at = "2026-04-14T10:00:00Z"
updated_at = "2026-04-14T10:00:00Z"
```

**Fields:** type, situation, source (human|agent), content (type-specific), created_at, updated_at. Nothing else.

## Skills

### `/recall`

**Frontmatter trigger:** "what was I working on", "load previous context", "search session history", "resume work", `/recall`

**No-args flow:**

1. Self-query: `engram recall --memories-only --query "when to call /prepare or /learn in the current situation"` — these results are for the agent's internal use only, not shown to the user.
2. Call `engram recall` (no query).
3. Summarize for the user: what was discussed, what was done (filtering mundane tool calls — the agent decides what's relevant to share), and memories from that time window.
4. Internally follow any operational guidance from the self-query.

**Query flow:**

1. Self-query: `engram recall --memories-only --query "when to call /prepare or /learn in the current situation"` — agent-internal, not shown to user.
2. Call `engram recall --query "<user's query>"`.
3. Present the query results to the user.
4. Internally follow any operational guidance from the self-query.

### `/prepare`

**Frontmatter trigger:** Before starting new work, switching tasks, beginning a feature, changing direction, tackling an issue. Should be called before implementation, debugging, or any significant new effort.

**Flow:**

1. Self-query: `engram recall --memories-only --query "how to prepare for <situation summary>, and when to call /prepare or /learn"` — agent-internal, not shown to user.
2. Internally follow any operational guidance from the self-query.
3. Analyze current context (what the user asked for, what's about to happen).
4. Make 2-3 targeted `engram recall --query "..."` calls based on the situation.
5. Present a compact briefing to the user: relevant context + memories from the domain queries.
6. Internally treat recalled memories as instructions/important context that should guide the agent's behavior during the upcoming work.

### `/remember`

**Frontmatter trigger:** "remember this", "remember that", "don't forget", "save this for later", `/remember`

**Flow:**

1. Self-query: `engram recall --memories-only --query "when to call /prepare or /learn in the current situation"`
2. Analyze what the user wants to remember.
3. Classify as feedback (SBIA) or fact(s) — could be multiple memories.
4. Draft fields, present to user for approval/editing.
5. For each approved memory: call `engram learn feedback|fact --source human ...`
6. Handle results:
   - **Written successfully:** Confirm to user.
   - **Contradiction returned:** Present the conflict. Offer to update existing, replace it, or force-write both.
   - **Duplicate returned:** Present the existing memory. Trigger diagnostic:
     - Was there a `/recall` or `/prepare` call during this session that should have surfaced this memory?
       - **Yes, queries were too narrow:** Suggest specific additional queries that would have found it. Present as concrete behavioral memories for user approval.
       - **Yes, memory wording too narrow:** Suggest a specific rewrite to the memory's situation field for user approval.
     - **No relevant `/recall` or `/prepare` call:** Suggest a behavioral memory about when to call `/prepare` for user approval.
   - Suggested behavioral memories must use situations that match the self-query format the skills use (e.g., "how to prepare for <topic>", "when to call /prepare or /learn for <topic>") so they'll actually be found by future self-queries. Do not depend on the user to know this format — draft the memory with the correct situation.
7. Internally follow any operational guidance from the self-query.

### `/learn`

**Frontmatter trigger:** After completing a task, after finishing work, when changing direction, "review what we learned", `/learn`. Should be called after implementation, after resolving a bug, after completing a plan step.

**Flow:**

1. Self-query: `engram recall --memories-only --query "how to review sessions for learnable moments, and when to call /prepare or /learn"`
2. Call `engram recall` (no query) to get recent session context.
3. Analyze for learnable moments: user corrections, failed approaches, discovered facts, patterns.
4. Present findings to user for approval (each as a drafted feedback or fact with all fields filled).
5. For each approved: call `engram learn feedback|fact --source agent ...`
6. Same conflict/duplicate/diagnostic handling as `/remember`.
7. Internally follow any operational guidance from the self-query.

## Transcript Parsing: Tool Call Extraction

Currently `context.Strip()` removes tool calls from transcripts. The new behavior preserves them in compact summarized form.

**For each tool call in the JSONL transcript, emit:**

```
[tool] Read(file_path="/src/main.go", offset=0, limit=100) → exit 0 | first line of output here
[tool] Bash(command="targ test") → exit 1 | Error: test failed
[tool] Edit(file_path="/src/foo.go", old_string="...", new_string="...") → exit 0 | Applied edit
```

**Mechanical rules:**

- Tool name from the JSONL event.
- All args included, key=value format, entire arg string truncated at 120 chars.
- Exit code preserved as-is (exit 0, exit 1, exit 2, etc.).
- First non-empty line of output, truncated at 120 chars.

**What gets stripped (as today):**

- Full tool call parameters beyond the summary.
- Full tool results beyond the first line.
- System messages, heartbeats, other noise.

## What Gets Dropped

### Skills (6 deleted)

- `engram-agent` — reactive memory agent
- `engram-lead` — non-tmux orchestrator
- `engram-tmux-lead` — tmux orchestrator
- `engram-up` — multi-agent entry point
- `engram-down` — multi-agent shutdown
- `use-engram-chat-as` — chat coordination protocol

### Binary packages

- `internal/bm25/` — keyword matching (replaced by Haiku judgment)
- `internal/tokenize/` — tokenizer for BM25
- `internal/surface/` — memory surfacing orchestration (replaced by Haiku over `engram list`)
- `internal/policy/` — policy.toml config and thresholds

### Binary commands

- `engram chat` and subcommands
- `engram agent` and subcommands

### Storage

- Chat files (`~/.local/share/engram/chat/`)
- `policy.toml`
- `creation-log.jsonl`

### Memory fields (stripped in migration)

- `initial_confidence`, `core`, `project_scoped`, `project_slug`
- All tracking counters (`surfaced_count`, `followed_count`, `not_followed_count`, `irrelevant_count`, `missed_count`)
- `pending_evaluations`

## Migration

### Memory files

One-time pass over all `~/.local/share/engram/memory/feedback/*.toml` and `facts/*.toml`:

1. Strip removed fields: `initial_confidence`, `core`, `project_scoped`, `project_slug`, `pending_evaluations`, all tracking counters.
2. Bump `schema_version` to 2.
3. Preserve content fields, situation, timestamps.
4. Normalize `source`: map freetext to `human` or `agent` (e.g., "user correction" → `human`, anything else → `agent`).
5. Add missing `situation` fields:
   - For feedback: derive from behavior/impact/action context (e.g., behavior "running go test directly" → situation "When running tests in Go projects").
   - For facts: derive from subject/predicate/object (e.g., subject "engram", predicate "uses", object "targ" → situation "When building or testing the engram project").
   - If content alone is ambiguous, check session transcripts around the memory's `created_at` timestamp for surrounding context.
   - Flag any memories where the inferred situation is low-confidence for manual review.

### Plugin manifest

Update `.claude-plugin/` to remove references to dropped skills, add new ones (`/prepare`, `/learn`, `/remember`). Update `/recall` description.

### Hooks

Update `hooks.json` — remove any chat/agent-related hooks. Session-start hook stays (binary rebuild).

### Docs

Existing architecture/design docs will be stale. Update or archive after implementation. Update the README.
