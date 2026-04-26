# Recall Query Pipeline Redesign

## Problem

`engram recall --query` accumulates raw session content newest-first into a 50KB buffer, then sends it to Haiku for summarization. This produces irrelevant output when the newest session has nothing to do with the query — one large irrelevant session fills the buffer and crowds out older relevant sessions.

## Design

Three-phase pipeline: memories first (cheap, local + 1 Haiku call), then per-session verbatim extraction (1 Haiku call each, newest-first, budget-bounded), then a final structured summary (1 Haiku call).

### Phase 1: Memory Search

1. List all memories via `MemoryLister`
2. Build index, ask Haiku which names are relevant (existing `listAndMatchMemories`)
3. Format matched memories, append to output buffer
4. stderr: `found N relevant memories`

### Phase 2: Per-Session Extraction

For each session (newest first), until output buffer reaches `DefaultExtractCap` (10KB):

1. Read + strip session via `Reader`
2. Call `ExtractRelevant(ctx, content, query)` — returns verbatim quotes
3. Skip session if extraction is empty
4. Append snippets to output buffer
5. stderr: `found N bytes from session <filename>`

### Phase 3: Structured Summary

1. Call `SummarizeFindings(ctx, buffer, query)` — new method on `SummarizerI`
2. Return result as `Result.Summary`
3. stderr: `summarizing N bytes of findings`

### Interface Changes

**SummarizerI** gains one method:

```go
type SummarizerI interface {
    ExtractRelevant(ctx context.Context, content, query string) (string, error)
    SummarizeFindings(ctx context.Context, content, query string) (string, error)
}
```

**ExtractRelevant system prompt** updated from:

> Extract only content relevant to the following query. Return relevant excerpts verbatim or tightly paraphrased. Return nothing if irrelevant.

To:

> Extract only content relevant to the following query. Return relevant excerpts verbatim or very lightly paraphrased in service of grammatical correctness and consistency. Return nothing if irrelevant.

**Orchestrator** gains optional status writer:

```go
func WithStatusWriter(w io.Writer) OrchestratorOption
```

### Budget Accounting

- `DefaultExtractCap` (10KB) is the shared budget for phases 1 + 2
- Memories count against the budget
- Per-session extraction stops when budget is reached
- Sessions that return empty extraction are skipped (no budget consumed)

### Error Handling

- Memory search failure: log to status, continue to phase 2
- Per-session read failure: skip session, continue
- Per-session extraction failure: skip session, continue
- Final summary failure: return error (this is the only fatal error)
- Empty buffer after phases 1+2: return empty result (no summary call)

### Files Modified

- `internal/recall/orchestrate.go` — rewrite `recallModeB`, add status output, add `OrchestratorOption`
- `internal/recall/orchestrate_test.go` — update mode B tests
- `internal/recall/summarize.go` — add `SummarizeFindings`, update `ExtractRelevant` prompt
- `internal/recall/summarize_test.go` — add tests for new method
- `internal/cli/cli.go` — wire `WithStatusWriter(os.Stderr)`
