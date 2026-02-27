# Data Model: Continuous Evaluation Memory Pipeline

**Branch**: `015-continuous-eval-memory` | **Date**: 2026-02-20

## New Table: surfacing_events

Tracks every memory surfacing with filter results and post-interaction evaluation.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | INTEGER | PK AUTOINCREMENT | Unique event identifier |
| memory_id | INTEGER | NOT NULL, FK→embeddings(id) | Which memory was surfaced |
| query_text | TEXT | NOT NULL | The query/prompt that triggered retrieval |
| hook_event | TEXT | NOT NULL | Hook type: "UserPromptSubmit", "PreToolUse", etc. |
| timestamp | TEXT | NOT NULL | RFC3339 timestamp of surfacing |
| session_id | TEXT | | Claude Code session ID |
| haiku_relevant | BOOLEAN | | Did the filter keep this memory? |
| haiku_tag | TEXT | | Classification: "relevant", "noise", "should-be-hook", "should-be-earlier" |
| haiku_relevance_score | REAL | | Filter confidence 0.0-1.0 |
| should_synthesize | BOOLEAN | | Did filter recommend synthesis? |
| faithfulness | REAL | | Post-eval score 0.0-1.0, filled async |
| outcome_signal | TEXT | | "positive", "negative", or NULL |
| user_feedback | TEXT | | "helpful", "wrong", "unclear", or NULL |
| e5_similarity | REAL | | Original E5 cosine similarity score |
| context_precision | REAL | | Kept/total ratio for this query batch |

**Indexes**:
- `idx_surfacing_memory(memory_id)` — fast lookup by memory
- `idx_surfacing_timestamp(timestamp)` — time-range queries
- `idx_surfacing_session(session_id)` — session-scoped queries

**Relationships**: Many surfacing_events → one embedding (memory_id FK).

## Modified: embeddings (new columns)

| Column | Type | Default | Description |
|--------|------|---------|-------------|
| importance_score | REAL | 0.0 | ACT-R base-level activation |
| impact_score | REAL | 0.0 | Avg faithfulness from surfacing_events |
| effectiveness | REAL | 0.0 | importance + α × impact |
| quadrant | TEXT | 'noise' | Classification: 'working', 'leech', 'gem', 'noise' |
| leech_count | INTEGER | 0 | Consecutive low-impact surfacings |

Added via existing `ALTER TABLE` migration pattern (fire-and-forget).

## Modified: metadata (new keys)

| Key | Value | Description |
|-----|-------|-------------|
| alpha_weight | "0.5" | Current α parameter for effectiveness formula |
| leech_threshold | "5" | Consecutive low-impact surfacings before diagnosis |
| importance_threshold | "0.0" | Boundary between low/high importance |
| impact_threshold | "0.3" | Boundary between low/high impact |
| last_autotune_at | RFC3339 | Last time thresholds were auto-adjusted |

## State Transitions

### Memory Quadrant Lifecycle

```
                    ┌──────────┐
                    │  noise   │ (default for new memories)
                    └────┬─────┘
                         │ importance increases
                         ▼
             ┌──────────────────────┐
             │  gem  OR  leech      │ (depends on impact)
             └──────┬───────┬───────┘
        impact high │       │ impact low
                    ▼       ▼
              ┌─────────┐ ┌──────────┐
              │ working │ │  leech   │
              └─────────┘ │ (diagnose)│
                          └──────────┘
                               │ after diagnosis + user action
                               ▼
                    ┌──────────────────┐
                    │ rewritten / moved│
                    │ / converted hook │
                    └──────────────────┘
```

### Surfacing Event Lifecycle

```
1. Hook fires → E5 retrieves candidates
2. Haiku Filter → haiku_relevant, haiku_tag, haiku_relevance_score populated
3. Memory surfaced (or suppressed) → event logged
4. Session ends → Post-eval populates faithfulness, outcome_signal
5. User feedback (optional) → user_feedback populated
6. Scoring run → memory's importance/impact/quadrant updated
```

## Computed Scores (at scoring time)

```
importance_score = existing ACT-R base-level activation

impact_score = weighted_avg(faithfulness)
    FROM surfacing_events
    WHERE memory_id = this AND faithfulness IS NOT NULL
    -- Recent events weighted higher (recency weight)

effectiveness = importance_score + α × impact_score

quadrant = CASE
    WHEN importance >= threshold AND impact >= threshold → 'working'
    WHEN importance >= threshold AND impact < threshold  → 'leech'
    WHEN importance < threshold  AND impact >= threshold → 'gem'
    ELSE → 'noise'
END

leech_count = consecutive surfacings where faithfulness < 0.3
    -- Reset to 0 when faithfulness >= 0.3
    -- Trigger diagnosis at leech_count >= leech_threshold
```

## Entity: FilterResult

Returned by the new `Filter()` method on `LLMExtractor`.

| Field | Type | Description |
|-------|------|-------------|
| MemoryID | int64 | ID of the evaluated memory |
| Content | string | Memory content text |
| Relevant | bool | Should this be surfaced? |
| Tag | string | Classification tag |
| RelevanceScore | float64 | Confidence 0.0-1.0 |
| ShouldSynthesize | bool | Would combining with others add value? |
| MemoryType | string | Original memory type |

## Entity: LeechDiagnosis

Produced by the leech diagnosis engine.

| Field | Type | Description |
|-------|------|-------------|
| MemoryID | int64 | ID of the leech memory |
| Content | string | Memory content |
| DiagnosisType | string | "content_quality", "wrong_tier", "enforcement_gap", "retrieval_mismatch" |
| Signal | string | Evidence description |
| ProposedAction | string | "rewrite", "promote_to_claude_md", "convert_to_hook", "narrow_scope" |
| SuggestedContent | string | Rewritten content (for "rewrite" action) |
| Recommendation | *Recommendation | Non-nil for non-memory actions (promote, convert_to_hook) |

## Entity: CLAUDEMDProposal

Produced by the quality gate for CLAUDE.md changes.

| Field | Type | Description |
|-------|------|-------------|
| Action | string | "add", "update", "remove" |
| Section | string | Target section name |
| Content | string | Proposed content |
| SourceMemoryID | int64 | Memory that triggered the proposal |
| Reason | string | Why this change is proposed |
| QualityChecks | map[string]bool | Results of each quality gate check |
| Recommendation | Recommendation | Describes what to do (no tool names) |

## Entity: Recommendation

Produced by any non-memory operation. Tool-agnostic — describes desired outcome without naming specific external tools.

| Field | Type | Description |
|-------|------|-------------|
| ID | string | Stable ID for dedup (e.g., "R001") |
| Category | string | Type of change (see categories below) |
| Priority | string | "high", "medium", "low" |
| Summary | string | One-line summary for stdout display |
| Description | string | Full details: what to do, why, suggested content |
| SourceMemoryIDs | []int64 | Memory/entry IDs that triggered this |
| Evidence | string | Data supporting the recommendation (scores, counts, patterns) |

**Categories** (open-ended, not an enum — new categories can be added):
- `claude-md-promotion`: Add content to CLAUDE.md
- `claude-md-demotion-to-hook`: Remove from CLAUDE.md, create deterministic hook
- `claude-md-demotion-to-skill`: Remove from CLAUDE.md, create skill
- `hook-conversion`: Convert memory to deterministic hook
- `skill-merge`: Consolidate related memories/entries into a single skill
- `skill-split`: Break an overly broad skill into focused pieces
