# Runbook Note Retrieval Specification

## Purpose

`runbook` notes retrieve and rank in `recall`/`query` symmetrically with `fact`/`feedback` — no
exclusion treatment (unlike `qa-question`), no dedicated query flag, no dedicated ranking
mechanism. A `task_type` classification field and pre-filter were considered and rejected during
design review (the cited SPL benchmark evidence measured a whole bundled system, not
type-classification in isolation — insufficient grounds for a new ranking mechanism and its
regression risk to existing retrieval). Why: `docs/architecture/adr.md` ADR-0026.

## Requirements

### Requirement: Runbook notes SHALL compete in the query pipeline's main matched set

`type: runbook` notes SHALL receive no retrieval-exclusion treatment (no `isQueryExcludedKind` membership while that mechanism exists — if #727 has removed it, no exclusion mechanism is added); they SHALL appear in the query payload's top-level `items[]` matched set alongside `fact` and `feedback` notes, on the same terms (matched-note floor, phrase limit, dedup) as any other kind.

#### Scenario: Runbook note inclusion in query results

- **WHEN** a query is executed and a `runbook` note is relevance-ranked above the match floor
- **THEN** it appears in the query's top-level `items[]` list like any other retrieval-competitive note

#### Scenario: candidate_l2s is not the surfacing signal

- **WHEN** a `runbook` note is present in `items[]` but its cluster's `candidate_l2s` (capped at the top 5 by centroid cosine, per ADR-0025) does not include it
- **THEN** the note is still considered to have surfaced — `candidate_l2s` membership is not required and is not a signal of surfacing; only `items[]` presence is

### Requirement: Runbook notes SHALL rank purely by situation-similarity

`runbook` notes SHALL receive no task-type-based ranking treatment — no query flag, no pre-filter, no boost. They rank against other matched notes exactly as `fact`/`feedback` notes do, on situation-similarity alone. (A `task_type` pre-filter mechanism was considered and rejected — design.md Decision 3, Non-Goals.)

#### Scenario: Runbook ranks like fact/feedback

- **WHEN** a query is executed with a `runbook` note and a `fact`/`feedback` note both matching a phrase with comparable situation-similarity scores
- **THEN** their relative ranking is determined by situation-similarity alone, with no kind-specific boost applied to either
