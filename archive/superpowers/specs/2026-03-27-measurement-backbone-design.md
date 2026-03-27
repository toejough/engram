# Measurement Backbone Design

Closes #398, #399, #400, #401. Adds the measurement loop that makes adaptive policies self-correcting.

## Decision: Snapshot-Based Attribution

Corpus-wide metric snapshots at policy activation and evaluation. No per-event attribution. Sufficient for single-user system where policies are approved one at a time.

## Decision: Session-Based Measurement Window

Count sessions, not calendar time. Default 10 sessions post-activation. Directly tied to usage — no measurement during idle periods.

## Section 1: MaintenanceHistory Population (#398)

When `maintain/apply.go` executes an action (rewrite, remove, broaden_keywords, review_staleness):

1. Read current `EffectivenessScore` from memory record → `EffectivenessBefore`
2. Read current surfaced count → `SurfacedCountBefore`
3. Apply the action
4. Write `MaintenanceAction{Action, Timestamp, EffectivenessBefore, SurfacedCountBefore, EffectivenessAfter: 0, SurfacedCountAfter: 0, Measured: false}` to memory's `MaintenanceHistory`
5. Persist the memory record

**Deferred measurement:** A `MeasureOutcomes` function scans memories with `Measured: false` entries. For each, checks if sufficient feedback has accumulated since the action timestamp (5+ new feedback events). If so, fills in `EffectivenessAfter`, `SurfacedCountAfter`, sets `Measured: true`.

`MeasureOutcomes` runs in the analysis pipeline alongside the pattern analyzers.

## Section 2: Surfacing Pattern Analysis (#399)

**At policy activation:** Snapshot corpus-wide surfacing metrics → `Policy.Effectiveness.Before`:
- Aggregate follow rate
- Aggregate irrelevance ratio
- Mean effectiveness across all memories

Uses existing `EffectivenessComputer` aggregate. No new telemetry.

**`AnalyzeSurfacingPatterns` function** in `internal/adapt`:
- Scans active surfacing policies past their measurement window
- Computes current corpus-wide metrics
- Compares to stored snapshot
- Policies that degraded metrics → auto-propose retirement
- Policies that improved metrics → mark as validated, populate `Policy.Effectiveness.After`

Runs alongside `AnalyzeContentPatterns` and `AnalyzeStructuralPatterns`.

## Section 3: Maintenance Outcome Analysis (#400)

**`AnalyzeMaintenanceOutcomes` function** in `internal/adapt`:
- Scans all memories with `Measured: true` entries in `MaintenanceHistory`
- Groups outcomes by action type (rewrite, remove, broaden_keywords, review_staleness)
- Computes success rate per type: percentage of actions that improved effectiveness
- If an action type has <40% success rate with 10+ measured outcomes → generate `DimensionMaintenance` policy proposal suggesting alternative actions
- Surfaced through triage alongside other proposals

Depends on #398 `MeasureOutcomes` having populated the "after" scores.

## Section 4: Policy Effectiveness Measurement & Auto-Retirement (#401)

**Session counter:** `Policy.Effectiveness.MeasuredSessions` starts at 0 on activation. Incremented each session that involves the policy's dimension.

**Measurement window:** Default 10 sessions. When `MeasuredSessions >= measurementWindow`, policy is eligible for evaluation.

**Snapshot mechanics:**
- At activation: `Policy.Effectiveness.Before` = `CorpusSnapshot{FollowRate, IrrelevanceRatio, MeanEffectiveness}`
- At evaluation: `Policy.Effectiveness.After` = same metrics

**`EvaluateActivePolicies` function** in `internal/adapt`:
- Scans active policies past measurement window
- Computes current corpus snapshot
- Compares Before vs After
- `After` worse or equal → auto-propose retirement (surfaced via triage, user approves via `/adapt`)
- `After` better → mark policy as validated (skip future measurement windows)

**`CorpusSnapshot` utility:** Shared function taking `EffectivenessComputer` + memory list, returns `CorpusSnapshot` struct. Used at activation time and evaluation time.

**`EvaluateActivePolicies` runs in the analysis pipeline** alongside all other analyzers.

## Shared Types

```go
// CorpusSnapshot captures corpus-wide metrics at a point in time.
type CorpusSnapshot struct {
    FollowRate       float64
    IrrelevanceRatio float64
    MeanEffectiveness float64
    SessionCount     int // total sessions at snapshot time
}
```

`Policy.Effectiveness` already has `Before` and `After` as `float64`. Replace them with `CorpusSnapshot` structs (or expand `Effectiveness` to hold `BeforeFollowRate`, `BeforeIrrelevanceRatio`, `BeforeMeanEffectiveness` and the same for After). The latter avoids a nested struct in TOML. Prefer the flat approach for TOML simplicity.

## Integration Points

- **Analysis pipeline** (`cli.go` adapt wiring): Add `MeasureOutcomes`, `AnalyzeSurfacingPatterns`, `AnalyzeMaintenanceOutcomes`, `EvaluateActivePolicies` to the pipeline
- **Policy activation** (`policy.go` or `cli/adapt.go`): When approving a policy, snapshot Before metrics
- **Session counting** (`cli.go` surface/extract paths): Increment `MeasuredSessions` on active policies each session
- **Triage output**: Retirement proposals surface alongside existing adaptation proposals

## What "Done" Means

- Every maintenance action records before-state on the memory
- Deferred measurement fills in after-state when sufficient feedback exists
- Active surfacing policies get corpus snapshots at activation
- All active policies are evaluated after their measurement window
- Ineffective policies get retirement proposals surfaced to the user
- Effective policies get validated and stop being re-measured
- No deferred items. Everything listed here ships.
