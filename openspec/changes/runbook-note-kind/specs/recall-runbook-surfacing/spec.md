## ADDED Requirements

### Requirement: Runbook notes SHALL compete in the query pipeline's main matched set

`type: runbook` notes SHALL receive no retrieval-exclusion treatment (no `isQueryExcludedKind` membership while that mechanism exists — if #727 has removed it, no exclusion mechanism is added); they SHALL appear in `candidate_l2s` alongside `fact` and `feedback` notes.

#### Scenario: Runbook note inclusion in query results

- **WHEN** a query is executed and a `runbook` note is relevance-ranked above the match floor
- **THEN** it appears in the query's `candidate_l2s` list like any other retrieval-competitive note

### Requirement: Runbook notes SHALL rank purely by situation-similarity

`runbook` notes SHALL receive no task-type-based ranking treatment — no query flag, no pre-filter, no boost. They rank against other matched notes exactly as `fact`/`feedback` notes do, on situation-similarity alone. (A `task_type` pre-filter mechanism was considered and rejected — design.md Decision 3, Non-Goals.)

#### Scenario: Runbook ranks like fact/feedback

- **WHEN** a query is executed with a `runbook` note and a `fact`/`feedback` note both matching a phrase with comparable situation-similarity scores
- **THEN** their relative ranking is determined by situation-similarity alone, with no kind-specific boost applied to either
