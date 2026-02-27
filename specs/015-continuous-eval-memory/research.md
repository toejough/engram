# Research: Continuous Evaluation Memory Pipeline

**Branch**: `015-continuous-eval-memory` | **Date**: 2026-02-20

## R1: Haiku Filter Integration Point

**Decision**: Extend the existing `Curate()` method on `LLMExtractor` interface into a new `Filter()` method that returns structured tags (relevant/noise/should-be-hook) plus relevance scores, rather than the current free-form relevance strings.

**Rationale**: The existing `Curate()` already calls Haiku to select relevant results, but returns `CuratedResult` with just content+relevance+type. The new pipeline needs richer output (binary keep/drop, classification tags, relevance score 0-1, synthesis prediction). A new method avoids breaking existing callers while providing the structured data surfacing_events needs.

**Alternatives considered**:
- Modify `Curate()` in-place: Would break existing callers and test contracts.
- Separate filter service: Over-engineered; same Haiku call, different prompt.

## R2: Sonnet Synthesis Trigger

**Decision**: The Haiku filter call includes a `should_synthesize` boolean in its response. When true and 2+ memories pass, a follow-up Sonnet call synthesizes them. No separate trigger logic needed.

**Rationale**: Per design doc clarification, Haiku predicts synthesis value during the filter call itself. This avoids a mechanical "2+ memories" rule and makes the synthesis decision content-aware at zero additional cost (same API call).

**Alternatives considered**:
- Separate Haiku call to decide synthesis: Extra latency and cost for marginal improvement.
- Always synthesize when 2+: Mechanical rule that doesn't account for content independence.

## R3: Surfacing Events Storage Pattern

**Decision**: Use the existing `ALTER TABLE` migration pattern (fire-and-forget, ignore duplicate column errors) for `embeddings` columns. Create `surfacing_events` as a new table in `initEmbeddingsDB()`.

**Rationale**: The codebase already has 16+ `ALTER TABLE` statements that run on every DB open with errors silently ignored. This pattern is proven and avoids a separate migration system. The `surfacing_events` table is new, so `CREATE TABLE IF NOT EXISTS` suffices.

**Alternatives considered**:
- Formal migration system: Not worth the complexity for a single-database project.
- Separate database file: Adds operational complexity with no benefit.

## R4: End-of-Session Scoring Hook

**Decision**: Add scoring logic to the existing `Stop` and `PreCompact` hook commands. For `/clear`, investigate whether Claude Code fires a hookable event; if not, the scoring also runs at `Stop` time which captures most sessions.

**Rationale**: Per design doc resolved question #6 (Option C). Stop and PreCompact hooks already run `projctl memory extract-session`. Adding scoring to the same pipeline is natural. The pre-clear hook is tracked as an open question in the design doc.

**Alternatives considered**:
- In-hook scoring (Option A): Adds latency to every interaction; complex state management.
- Background worker (Option B): CLI sessions are ephemeral; long-running processes don't fit.

## R5: Impact-Weighted Activation Formula

**Decision**: Extend the existing `GetActivationStats()` function to include effectiveness in the activation score: `B_i = ln(Σ t_j^(-d)) + α × effectiveness_i`. α starts at 0.5.

**Rationale**: The existing ACT-R implementation in `GetActivationStats()` already computes base-level activation with type-specific decay. Adding the effectiveness term is a single arithmetic operation. Auto-tuning tracks quadrant distributions and adjusts α to maintain target ranges.

**Alternatives considered**:
- Multiplicative formula `B_i × (1 + effectiveness)`: Can zero out activation for negative effectiveness, which is too aggressive.
- Separate scoring function: Duplicates logic; the activation score is the natural home.

## R6: Existing Curation Flow vs. New Filter

**Decision**: The new `Filter()` replaces `Curate()` in the hook pipeline. `Curate()` remains available for backwards compatibility and CLI `--rich` usage.

**Rationale**: `Curate()` returns `[]CuratedResult` (content, relevance, memory_type) which is insufficient for surfacing_events. The new `Filter()` returns `[]FilterResult` (memory_id, haiku_relevant, haiku_tag, relevance_score, should_synthesize). The hook pipeline switches to `Filter()` while `TierCurated` formatting can still use `Curate()`.

**Alternatives considered**:
- Extend `CuratedResult` struct: Would add fields irrelevant to existing `TierCurated` callers.

## R7: Auto-Tune Mechanism

**Decision**: Use distribution-tracking with exponential moving averages. After each scoring run, compute the percentage of memories in each quadrant. If leech percentage exceeds 15% or drops below 5%, adjust the impact threshold by 0.05 in the appropriate direction. Same for α weight. Cap adjustments at 0.1 per scoring run to prevent oscillation.

**Rationale**: Simple, bounded, and interpretable. The exponential moving average smooths noise from individual sessions. The per-run cap prevents wild swings.

**Alternatives considered**:
- Bayesian optimization: Over-engineered for 2 parameters.
- Fixed grid search: Doesn't adapt to changing memory populations.

## R8: Leech Diagnosis Signal Sources

**Decision**: Four diagnosis categories, each detected from surfacing_events data:
1. **Content quality**: High surfacing count + low faithfulness + agent response doesn't reference memory content.
2. **Wrong tier**: Surfaced after correction already happened in same session (timestamp comparison).
3. **Enforcement gap**: Surfaced, agent references it, but user still corrects (agent understood but didn't comply).
4. **Retrieval mismatch**: Low Haiku relevance scores across diverse query contexts.

**Rationale**: These four categories cover the space of "why doesn't this memory work?" and each has a distinct data signature detectable from surfacing_events without additional API calls.

**Alternatives considered**:
- LLM diagnosis: Would require an additional Sonnet call per leech. Defer to a future enhancement if pattern matching proves insufficient.
