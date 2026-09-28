## MODIFIED Requirements

### Requirement: Runbook notes SHALL compete in the query pipeline's main matched set

`type: runbook` notes SHALL receive no retrieval-exclusion treatment (no `isQueryExcludedKind` membership while that mechanism exists — if #727 has removed it, no exclusion mechanism is added); they SHALL appear in the query payload's top-level `items[]` matched set alongside `fact` and `feedback` notes, on the same terms (matched-note floor, phrase limit, dedup) as any other kind.

#### Scenario: Runbook note inclusion in query results

- **WHEN** a query is executed and a `runbook` note is relevance-ranked above the match floor
- **THEN** it appears in the query's top-level `items[]` list like any other retrieval-competitive note

#### Scenario: candidate_l2s is not the surfacing signal

- **WHEN** a `runbook` note is present in `items[]` but its cluster's `candidate_l2s` (capped at the top 5 by centroid cosine, per ADR-0025) does not include it
- **THEN** the note is still considered to have surfaced — `candidate_l2s` membership is not required and is not a signal of surfacing; only `items[]` presence is
