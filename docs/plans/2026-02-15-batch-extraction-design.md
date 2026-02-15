# Batch Session Learning: LLM-First Extraction Pipeline

## Problem

The current `extract-session` hook runs at session end with a 60s timeout, limiting extraction to a few items. Long sessions (100KB-161MB) contain many learning events that go unprocessed. We need batch extraction that can process full session transcripts offline.

## Design

### Pipeline

```
Session JSONL → Strip Noise → Chunk (25KB) → Haiku ID Events (parallel) → Sonnet Extract Principles → Deduplicate → Store
```

### Stage 1: Strip Noise

Mechanically reduce session JSONL to learning-relevant content:

- Remove `<system-reminder>` tags and content
- Keep `<teammate-message>` content (with sender ID)
- Keep raw user messages (system-reminders stripped)
- Collapse skill content to `(skill loaded)`
- Omit successful tool results (assistant narrates outcomes)
- Keep error tool results (up to 300 chars)
- Edit operations: show diff between old/new with ~60 chars context
- Truncate heredocs to first line + char count
- Keep assistant text and tool invocations (name + key args)

Typical reduction: 95-99% (368KB → 8KB, 161MB → 1555KB).

### Stage 2: Haiku Event Identification

Chunk stripped content into ~25KB pieces. Send each chunk to Haiku in parallel (4 workers) with assistant prefill `[` to force JSON output.

Each chunk produces events with:
- `line_range`: approximate location
- `event_type`: error-and-fix | user-correction | strategy-change | root-cause-discovery | environmental-issue | pattern-observed | coordination-issue
- `what_happened`: 1-2 sentences
- `why_it_matters`: 1 sentence reusable lesson

System prompt emphasizes: analyze transcript, never continue it. Look for both technical issues AND process/coordination patterns.

Expected: 92-100% chunk success rate. Failed chunks are logged and skipped.

### Stage 3: Sonnet Principle Extraction

Single Sonnet call with all events from a session. Produces 3-13 actionable principles per session.

Prompt rules:
- Merge duplicate events into single principles
- Frame as "When X, do Y" patterns
- Process/coordination lessons weighted equally with technical lessons
- Include concrete evidence from the session
- Categories: debugging, git-workflow, api-design, team-coordination, testing, code-quality, cli-design

### Stage 4: Deduplicate and Store

Compare extracted principles against existing memories via embedding similarity. Skip duplicates (similarity > 0.85). Store new principles via existing `memory.Learn()` path.

## Implementation

This pipeline runs as `projctl memory learn-sessions` (the existing CLI command). It processes sessions from the `processed_sessions` table that haven't been batch-extracted yet.

### What to build (Go)

1. **`strip_session.go`** — Go port of the Python stripping logic. Reads JSONL, outputs stripped text.
2. **`chunk.go`** — Split stripped text into ~25KB chunks by line boundaries.
3. **`haiku_events.go`** — Call Haiku API for each chunk in parallel (4 goroutines). Parse JSON events. Uses existing `LLMClient` infrastructure.
4. **`sonnet_principles.go`** — Call Sonnet API with all events. Parse JSON principles.
5. **`learn_sessions.go`** — Orchestrate: discover unprocessed sessions → strip → chunk → Haiku → Sonnet → deduplicate → store. Mark sessions as processed.

### What exists already

- `internal/memory/auth.go` — Keychain OAuth token retrieval
- `internal/memory/llm_api.go` — `LLMClient` interface, `NewLLMExtractor()`, direct API calls
- `internal/memory/session.go` — Session discovery, `processed_sessions` table
- `cmd/projctl/memory_learn_sessions.go` — CLI command skeleton

### API calls

- Haiku: `claude-haiku-4-5-20251001`, 4096 max tokens per chunk
- Sonnet: `claude-sonnet-4-5-20250929`, 4096 max tokens per session
- Auth: OAuth token from macOS keychain (`Claude Code-credentials`)
- Rate limiting: 4 parallel Haiku calls, sequential Sonnet calls

## Validation

Tested against 3 real sessions with 14 pre-approved lessons:
- **Recall**: 82% (11.5/14 approved lessons recovered)
- **Precision**: 100% (26/26 extracted principles were useful)
- **Quality**: Specific, actionable, with concrete examples

See `docs/plans/2026-02-15-extraction-validation.md` for full results.

## Known Limitations

- Brief process observations (2-3 lines) sometimes missed by Haiku
- Sonnet may merge related coordination lessons into a single principle
- ~8% chunk failure rate on very large sessions (Haiku returns non-JSON)
- No retry logic for failed chunks in initial implementation
