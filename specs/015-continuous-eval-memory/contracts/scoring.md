# Contract: Impact Scoring & Quadrant Classification

**Spec**: FR-004, FR-005, FR-006, FR-007, FR-009, FR-016, FR-017, FR-019 | **Priority**: P2, P6

## Interface

```go
// ScoreSession evaluates all surfacing events from a session and updates memory scores.
// Called at session end (Stop, PreCompact hooks).
ScoreSession(db *sql.DB, sessionID string, llm LLMExtractor) error

// ClassifyQuadrants updates quadrant classification for all memories with surfacing data.
ClassifyQuadrants(db *sql.DB) error

// AutoTuneThresholds adjusts importance/impact thresholds based on quadrant distributions.
AutoTuneThresholds(db *sql.DB) error

// GetQuadrantSummary returns counts of memories in each quadrant.
GetQuadrantSummary(db *sql.DB) (QuadrantSummary, error)
```

## Data Types

### QuadrantSummary

| Field | Type | Description |
|-------|------|-------------|
| Working | int | Count of high-importance, high-impact memories |
| Leech | int | Count of high-importance, low-impact memories |
| Gem | int | Count of low-importance, high-impact memories |
| Noise | int | Count of low-importance, low-impact memories |
| AlphaWeight | float64 | Current α parameter |
| ImportanceThreshold | float64 | Current importance boundary |
| ImpactThreshold | float64 | Current impact boundary |

## Behavior

### ScoreSession
1. Retrieves all surfacing events for the session via `GetSessionSurfacingEvents()`.
2. For each event where `haiku_relevant=true` and `faithfulness` is null:
   - Calls Haiku post-eval to score faithfulness (100% evaluation — no sampling)
   - Updates `faithfulness` and `outcome_signal` via `UpdateSurfacingOutcome()`
3. After evaluation, recalculates each affected memory's scores:
   - `impact_score` = recency-weighted average of faithfulness across all surfacing_events
   - `effectiveness` = `importance_score + α × impact_score`
   - Updates `leech_count`: increment if faithfulness < 0.3, reset to 0 otherwise
4. Calls `ClassifyQuadrants()` to update quadrant assignments.
5. Calls `AutoTuneThresholds()` to adjust scoring parameters.

### ClassifyQuadrants
- Reads current thresholds from metadata table.
- For each memory with `importance_score > 0` or `impact_score > 0`:
  - Classifies into quadrant based on threshold comparison.
  - Updates `quadrant` column.

### AutoTuneThresholds
- Computes current quadrant distribution percentages.
- Target: leeches 5-15% of active memories.
- If leech% > 15%: increase impact_threshold by 0.05 (fewer classified as leech).
- If leech% < 5%: decrease impact_threshold by 0.05 (more classified as leech).
- Cap adjustment at 0.1 per run. Use exponential moving average across runs.
- Updates metadata table with new thresholds.

### Spreading Activation (FR-017)
- During retrieval (in Query flow), after E5 returns top-K:
  - Perform one additional vector search for memories similar to top-K results (similarity > 0.7).
  - Boost their activation score temporarily (current interaction only, not persisted).
- Implementation: modify the existing `Query()` function to accept an option for spreading activation.

## Evaluation Strategy

All relevant surfacing events are evaluated (100% — no sampling). Rationale:
- Haiku post-eval cost is ~$0.0001/call; at ~300 relevant events/day = $0.03/day, well within NFR-004 budget ($0.15)
- Sampling distorts threshold-based counters (leech_count), delaying diagnosis
- Every event contains learnable signal; discarding 90% wastes data

For long sessions (30+ relevant events), post-eval calls SHOULD be batched or run concurrently to stay within NFR-003 (5-second scoring target).

## Post-Eval Prompt

```
Given the surfaced memory and the agent's subsequent response:

Memory: "{memory_content}"
Agent response: "{agent_response_excerpt}"
User's next message (if any): "{user_next_message}"

Did the agent's response align with the guidance in the surfaced memory?
Score 0.0 (completely ignored/contradicted) to 1.0 (fully followed).

Return ONLY valid JSON: {"faithfulness": <float>, "signal": "<positive|negative>"}
```

## Error Handling

| Condition | Behavior |
|-----------|----------|
| No surfacing events for session | No-op, return nil |
| Haiku post-eval fails | Skip that event's faithfulness update; log warning |
| All post-evals fail | Log error, scores unchanged |
| Threshold adjustment out of bounds | Clamp to [0.0, 1.0] range |

## Constraints

- Post-eval uses bounded context (surfacing event + response excerpt + next message), not full session.
- Scoring runs in the same process as Stop/PreCompact/SessionStart-clear hook (no separate worker).
- α weight, leech_threshold, and other parameters stored in metadata table (not config file).
