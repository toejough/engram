# Contract: Memory Filter

**Spec**: FR-001, FR-002, FR-003 | **Priority**: P1

## Interface

```go
// Filter evaluates retrieved memories for relevance and returns structured results.
// Added to LLMExtractor interface.
Filter(ctx context.Context, query string, candidates []QueryResult) ([]FilterResult, error)
```

## Input

| Parameter | Type | Description |
|-----------|------|-------------|
| ctx | context.Context | Cancellation/timeout context |
| query | string | The user's prompt or tool query |
| candidates | []QueryResult | E5-retrieved memory candidates (pre-scored) |

## Output: FilterResult

| Field | Type | Description |
|-------|------|-------------|
| MemoryID | int64 | ID from QueryResult |
| Content | string | Memory content |
| Relevant | bool | Should surface to agent? |
| Tag | string | "relevant", "noise", "should-be-hook", "should-be-earlier" |
| RelevanceScore | float64 | 0.0-1.0 confidence |
| ShouldSynthesize | bool | Would combining with other relevant results add value? |
| MemoryType | string | Original memory type |

## Behavior

1. Sends all candidates to Haiku in a single API call with structured JSON response format.
2. Each candidate gets a binary relevant/not-relevant decision plus classification tag.
3. The `ShouldSynthesize` flag is set per-candidate; synthesis triggers when 2+ candidates have it set.
4. Returns all candidates (both kept and filtered) so surfacing_events can be logged for both.
5. On LLM failure (API error, timeout), falls back to passing all candidates through unfiltered (degradation, not failure).

## Synthesis Follow-Up

When `Filter()` returns 2+ results with `ShouldSynthesize=true`:

```go
// Existing Synthesize method on LLMExtractor. The caller must use a Sonnet-configured
// extractor instance (the current Synthesize signature has no model override parameter).
Synthesize(ctx context.Context, memories []string) (string, error)
```

The caller (hook pipeline) is responsible for:
1. Checking if synthesis should trigger
2. Calling `Synthesize()` on a Sonnet-configured LLMExtractor instance
3. Presenting synthesized output instead of individual memories
4. Scoring attribution: all source memories with ShouldSynthesize=true share the synthesized output's faithfulness score at post-eval time

## Error Handling

| Condition | Behavior |
|-----------|----------|
| Haiku API unavailable | Return all candidates with `Relevant=true` but `RelevanceScore=-1.0` (degradation sentinel). Caller logs surfacing events with `haiku_relevant=NULL`, `haiku_tag=NULL`, `haiku_relevance_score=NULL`. These events are excluded from post-eval scoring (contribute to importance only, same as unevaluated surfacings). |
| Haiku returns malformed JSON | Same degradation behavior as API unavailable + log warning |
| Empty candidates | Return empty slice (no API call) |
| Context cancelled | Return context error immediately |

## Constraints

- Latency: ~200ms per call (Haiku speed)
- Cost: ~$0.0001 per call
- No side effects: surfacing_event logging is the caller's responsibility
