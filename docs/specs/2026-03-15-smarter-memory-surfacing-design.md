# Smarter Memory Surfacing: Reduce Noise from Unproven Memories

**Issue:** #307
**Date:** 2026-03-15

## Problem

Unproven memories (0 surfacings) dominate surfacing slots because:
1. `effectivenessScoreFor()` gives them a generous 50% default — higher than many proven memories
2. `filterByEffectivenessGate()` only gates memories with >=5 surfacings, so unproven always pass
3. `minRelevanceScore = 0.05` is trivially low, letting almost any BM25 match through
4. No limit on how many unproven memories can surface per invocation

## Design

Two complementary mechanisms applied to all three surfacing modes:

### 1. Stricter Relevance Thresholds for Unproven Memories

Unproven memories must clear a *higher* BM25 relevance bar than proven ones. The intuition: if a memory has never demonstrated value, it should need stronger textual evidence of relevance to earn context space.

| Mode | Proven BM25 floor | Unproven BM25 floor |
|------|-------------------|---------------------|
| Tool | 0.05 | 0.30 |
| Prompt | 0.05 | 0.20 |
| Session-start | N/A (no BM25) | N/A — uses effectiveness ranking |

For session-start mode (effectiveness-ranked, no BM25): lower the default effectiveness score for unproven memories from 50% to 30%, so proven memories with real track records rank ahead.

**Definition of "unproven":** `SurfacedCount == 0` in effectiveness data, or no effectiveness entry at all.

### 2. Cold-Start Budget

After ranking and filtering, limit how many unproven memories can surface per invocation to 1 across all modes. This ensures unproven memories get *some* exposure (needed for bootstrapping) without dominating.

Applied after all other filtering (BM25, effectiveness gate, suppression) but before token budget enforcement.

## Changes

### `internal/surface/surface.go`

**New constants:**
- `unprovenBM25FloorTool = 0.30`
- `unprovenBM25FloorPrompt = 0.20`
- `unprovenDefaultEffectiveness = 30.0`
- `coldStartBudget = 1`

**`matchPromptMemories(message, memories, effectiveness)`** — add effectiveness parameter. After BM25 scoring, apply `unprovenBM25FloorPrompt` for memories where `isUnproven(path, effectiveness)` is true. Keep `minRelevanceScore` for proven memories.

**`matchToolMemories(toolName, toolInput, memories, effectiveness)`** — same pattern with `unprovenBM25FloorTool`.

**`effectivenessScoreFor(path, effectiveness)`** — when memory is unproven, return `unprovenDefaultEffectiveness` (30%) instead of `sessionStartDefaultEffectiveness` (50%).

**New `applyColdStartBudget[T](matches []T, effectiveness, getPath func(T) string) []T`** — generic helper (or two typed variants for promptMatch/toolMatch) that keeps all proven matches plus at most `coldStartBudget` unproven matches, preserving rank order.

**`runPrompt`** — call `applyColdStartBudget` after frecency re-rank, before suppression passes.

**`runTool`** — call `applyColdStartBudget` after frecency re-rank, before token budget.

**`runSessionStart`** — call cold-start budget variant on `[]*memory.Stored` after effectiveness ranking, before top-7 limit.

**New `isUnproven(path, effectiveness) bool`** — returns true when effectiveness map has no entry or entry has `SurfacedCount == 0`.

### No changes to

- `budget.go` — per-invocation token budgets unchanged
- `suppress_p4f.go` — suppression passes unchanged
- `frecency/frecency.go` — frecency scoring unchanged

## Testing

- Unit tests for `isUnproven` with: no data, 0 surfacings, 1+ surfacings
- Unit tests for cold-start budget: all unproven (keeps 1), all proven (keeps all), mixed (keeps all proven + 1 unproven)
- Unit tests for higher BM25 floor: unproven memory below `unprovenBM25FloorPrompt` gets filtered, proven memory at same score passes
- Integration-level tests for `runPrompt`, `runTool`, `runSessionStart` confirming unproven memories don't dominate output

## Acceptance Criteria (from #307)

- [x] Unproven memories don't dominate surfacing slots over proven, effective memories
- [x] Intelligence-based solution (better ranking/filtering), not a blunt hard cap
- [x] Existing effectiveness tracking and suppression infrastructure leveraged
- [ ] Token budget per session respected (out of scope — separate concern, no cumulative budget exists today)
