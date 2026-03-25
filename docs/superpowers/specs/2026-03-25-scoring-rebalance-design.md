# Scoring Rebalance Design (#374)

## Problem

The frecency scoring formula is dead code. `toFrecencyInput` doesn't wire `SurfacedCount` or `LastSurfacedAt` from `memory.Stored`, so activation is always zero. `CombinedScore = BM25 × (1 + 0) = BM25`. Frecency re-ranking is a no-op in prompt and tool modes.

Additionally:
- The recency term `1/(1+hours)` crushes to ~0 after 24h, making it useless even if wired
- The spread factor (`SurfacingContexts`) has only 1-3 possible values — no signal
- The insufficient-data gate (`totalEvals >= 5`) means only 11 of 2553 memories are classifiable
- Spreading activation exists but only in the dead session-start code path (#375)

## Design: Approach B — Two-Stage Scoring with BM25-Seeded Spreading

### Stage 1: Candidate Generation

BM25 matches candidates against the query (unchanged from today):
- **Prompt mode**: matches user message against title + content + principle + keywords + concepts
- **Tool mode**: matches tool name + input against anti-pattern memories only

New: For each BM25 match above the existing BM25 floor (same threshold used for filtering), retrieve graph neighbors via `LinkReader`. Add neighbors to the candidate set. This lets topically related memories surface even when BM25 misses them by keyword.

### Stage 2: Quality-Weighted Ranking

For each candidate:

```
relevance  = BM25 score (0 if memory came only from graph neighbors)
spreading  = sum(BM25[linker] × link_weight) / link_count
             (0 if no BM25-matched neighbors link to this memory)
eff        = followed / (followed + contradicted + ignored), default 0.5
recency    = 1 / (1 + days_since_last_surfaced / 7)   # 7-day half-life
freq       = ln(1 + surfaced_count) / ln(1 + max_surfaced_in_corpus)  # 0-1, corpus-wide max

quality    = 1.5×eff + 0.5×recency + 1.0×freq
score      = (relevance + 1.0×spreading) × (1 + quality)
```

**`max_surfaced_in_corpus`** is computed once at scorer initialization across all loaded memories (not per-query). This ensures `freq` is stable across queries.

**Division-by-zero guard:** When `link_count = 0` (no BM25-matched neighbors link to a memory), `spreading = 0`.

**Weight rationale:**
- `eff=1.5`: Effectiveness is the strongest quality signal — proven memories should dominate
- `freq=1.0`: Broadly-referenced memories are more likely useful
- `recency=0.5`: Gentle nudge, not a dominator. 7-day half-life means a memory loses half its recency after a week, not after an hour
- `alpha=1.0` (spreading): Graph neighbors get equal consideration to direct BM25 matches

### Spreading Activation: BM25-Seeded, Not Global

The current P3 implementation computes spreading globally (every memory gets boosted by ALL its links). This gives the 24 super-linked memories ~500x scores over unlinked ones.

The new design seeds spreading FROM BM25-matched memories only:
1. Run BM25, get matches above threshold
2. For each match, look up its graph neighbors
3. Each neighbor gets: `boost = BM25_score_of_matcher × link_weight`
4. Normalize by link count to prevent link-dense memories from dominating
5. Neighbors join the candidate pool with their spreading score

If no BM25 matches, no spreading happens. Spreading amplifies relevance, not replaces it.

## Changes

### Wire `toFrecencyInput` (fix #376)

Connect `SurfacedCount` and `LastSurfacedAt` from `memory.Stored` into the scoring input.

Add `LastSurfacedAt time.Time` to `memory.Stored` struct. Update `internal/retrieve/retrieve.go` to parse `LastSurfacedAt` from `MemoryRecord` (string) into `Stored` (time.Time).

Note: `SurfacedCount` already flows through the retriever. `LastSurfacedAt` is on `MemoryRecord` as a string but not on `Stored` — needs parsing and a new field.

The `WithSurfacingRecorder` in `cli.go:1519-1524` correctly increments both fields via `ReadModifyWrite` on every surfacing event, so the data is current.

### Replace the frecency formula

Replace the 4-factor product formula (`freq × recency × spread × eff`) with the two-stage Approach B scoring. The `frecency.Scorer` and `frecency.Input` types will be updated to implement the new formula.

### Add spreading activation to prompt/tool modes

Wire `LinkReader` into the surfacing pipeline for prompt and tool modes (currently only wired for the dead session-start path). Add BM25-seeded neighbor discovery.

### Drop `SurfacingContexts`

Remove the spread factor from `frecency.Input`. `SurfacingContexts` can only be 1-3 values (session-start/prompt/tool) — not enough cardinality to be useful. Remove from `track.ComputeUpdate` as well.

### Remove insufficient-data gate from surfacing

Remove the `insufficientDataThreshold` constant (line 994 in surface.go) from surfacing code paths. The gating logic in `effectivenessScoreFor` (line 1201) and the filter functions that use it (`filterSessionStartByEffectiveness`, `filterToolMatchesByEffectivenessGate`) no longer block memories with <5 evaluations from surfacing. Memories with no evaluations get `eff = 0.5` (neutral default), and the scoring formula handles it naturally.

The maintenance system (review.go) keeps its own ≥5 evaluation gate for quadrant classification — that's unchanged.

### Keep existing mechanisms

These are unchanged:
- BM25 matching and floor filtering (0.05 proven, 0.20/0.30 unproven)
- Irrelevance penalty: `5/(5 + irrelevant_count)` applied to BM25 score before floor comparison
- Cold-start budget: max 1 unproven memory per invocation
- Top-2 limit for prompt/tool modes
- Contradiction suppression, P4f cluster dedup, cross-ref, transcript suppression

## Files

| File | Change |
|------|--------|
| `internal/frecency/frecency.go` | Replace Activation/CombinedScore with new two-stage formula |
| `internal/frecency/frecency_test.go` | Update tests for new formula |
| `internal/surface/surface.go` | Wire scoring input properly; add spreading to prompt/tool; remove insufficient-data gate; add `bm25Score` field to `promptMatch`/`toolMatch` structs to thread scores through pipeline |
| `internal/surface/surface_test.go` | Update scoring tests |
| `internal/memory/memory.go` | Add `LastSurfacedAt time.Time` to `Stored` struct |
| `internal/retrieve/retrieve.go` | Parse `LastSurfacedAt` string from `MemoryRecord` into `Stored` as `time.Time` |
| `internal/track/track.go` | Remove `SurfacingContexts` from `Update` |
| `internal/track/track_test.go` | Update tests |

## Out of Scope

- Session-start surfacing (dead code, #375)
- PreCompact surfacing (dead code, #378)
- Maintenance quadrant system changes
- Link graph rebalancing (the 24 super-linked memories are a data quality issue, not a formula issue)
- Stale spec updates (#377)

## Validation

- Playground: `scoring-playground.html` with Approach B, alpha=1, wEff=1.5, wRec=0.5, wFreq=1, half-life=7
- Test queries: "targ build test", "git commit", "memory surfacing", "nil check"
- Verify that high-effectiveness memories rank above low-effectiveness ones for the same BM25 score
- Verify that spreading activation surfaces relevant neighbors without overwhelming direct matches
