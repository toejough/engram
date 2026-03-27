# Keyword Refinement Using Surfacing-Query Context

**Issue:** #372 (scoped to keyword refinement only; time decay and auto-removal removed from scope)

## Problem

When a memory gets irrelevant feedback, its `irrelevant_count` increments and the BM25 irrelevance penalty dampens future scoring, but the keywords that caused the false match remain unchanged. The memory keeps matching the same wrong contexts, just with a lower score. The maintain system proposes `refine_keywords` but has no evidence about *which* keywords caused the problem.

## Design

### Approach

Accumulate bad-match query contexts on irrelevant feedback. During maintain, include these contexts when generating keyword refinement proposals via LLM. Human reviews and applies during triage.

### Data model

Add `IrrelevantQueries` to both `MemoryRecord` (TOML serialization) and `Stored` (in-memory domain):

```go
// In memory/record.go (MemoryRecord):
IrrelevantQueries []string `toml:"irrelevant_queries,omitempty"`

// In memory/memory.go (Stored):
IrrelevantQueries []string
```

Capped at 20 entries (oldest dropped on overflow). Each entry is the surfacing query that triggered an irrelevant match.

Example TOML:

```toml
irrelevant_queries = [
  "how to test",
  "dependency injection patterns",
  "git rebase workflow"
]
```

### Feedback path

In `cli/feedback.go`, in `runFeedback` (not in `applyFeedbackCounters`, which only handles counters):

1. `applyFeedbackCounters` increments `irrelevant_count` (existing behavior, unchanged)
2. If `--irrelevant` and `--surfacing-query` are both non-empty, append `surfacingQuery` to `record.IrrelevantQueries`
3. If `len(record.IrrelevantQueries) > 20`, trim oldest entries (drop from front)
4. `writeFeedbackTOML` persists the updated record (existing behavior, unchanged)

No new CLI flags — `--surfacing-query` already exists and is parsed. The change is that feedback.go persists the query instead of only printing a confirmation message.

### Maintain path

In `maintain/maintain.go`, when `checkIrrelevance` generates a `refine_keywords` proposal:

1. Read the memory's `IrrelevantQueries` from the `Stored` record (available in the `memories map[string]*memory.Stored` passed to `Generate`)
2. If queries are non-empty, call `g.llmCaller` — the 4-arg function `func(ctx, model, systemPrompt, userPrompt string) (string, error)` — same injection point and calling convention as `handleLeech` and `handleHiddenGem`
3. Wrap the LLM response via `safeLLMDetails` (existing helper) and store in the proposal's `Details` field (`json.RawMessage`)

If `irrelevant_queries` is empty, fall back to current behavior (diagnosis string only, no LLM call).

#### LLM output schema

The LLM prompt requests JSON output matching:

```json
{
  "remove_keywords": ["generic-keyword-1", "generic-keyword-2"],
  "add_keywords": ["specific-replacement-1", "specific-replacement-2"],
  "rationale": "Why these keywords were problematic and what the replacements target"
}
```

This is stored in the proposal's `Details` field as `json.RawMessage` (same pattern as leech/hidden-gem proposals use `safeLLMDetails`).

#### Proposal → ApplyAction handoff

The CLI's triage flow reads `Proposal.Details` and populates `ApplyAction.Fields` with the parsed JSON keys. For `refine_keywords`, the `Details` JSON above maps directly to `Fields["remove_keywords"]` and `Fields["add_keywords"]`. This is the same pattern used by `rewrite` proposals where `Details` contains field updates that flow into `ApplyAction.Fields`.

### Apply path

In `signal/apply.go`, add a `refine_keywords` case to the `Apply` switch (alongside existing `remove`, `rewrite`, `broaden_keywords`, `escalate`):

1. Read the memory via `a.readMemory`
2. Type-assert `action.Fields["remove_keywords"]` to `[]any`, convert to `[]string`
3. Type-assert `action.Fields["add_keywords"]` to `[]any`, convert to `[]string`
4. Remove keywords listed in `remove_keywords` from `stored.Keywords`
5. Append `add_keywords` to `stored.Keywords` (normalized via `keyword.NormalizeAll`)
6. Clear `stored.IrrelevantQueries` (the evidence has been acted on)
7. Write the updated memory via `a.writeMem.Write`

### Files changed

| File | Change |
|------|--------|
| `internal/memory/record.go` | Add `IrrelevantQueries []string` field to `MemoryRecord` |
| `internal/memory/memory.go` | Add `IrrelevantQueries []string` field to `Stored` |
| `internal/retrieve/retrieve.go` | Map `IrrelevantQueries` in `parseMemoryFile` (`MemoryRecord` → `Stored`) |
| `internal/cli/signal.go` | Map `IrrelevantQueries` in `storedMemoryWriter.Write` (`Stored` → `MemoryRecord`) |
| `internal/cli/feedback.go` | Persist surfacing query to `IrrelevantQueries` in `runFeedback` |
| `internal/maintain/maintain.go` | LLM-backed keyword suggestions in `checkIrrelevance` |
| `internal/signal/apply.go` | Add `refine_keywords` case to `Apply` switch |

### What this does NOT cover

- **Time-based decay** — moved to #374 (frecency rebalancing)
- **Auto-removal** — existing Noise quadrant proposals + manual triage are sufficient
- **Immediate keyword correction** — rejected in favor of accumulate-and-review during maintain
- **Matched keyword identification** — the LLM infers which keywords are problematic from the evidence rather than us computing BM25 token overlap at feedback time
