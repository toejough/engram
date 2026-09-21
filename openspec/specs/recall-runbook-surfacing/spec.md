# Runbook Note Retrieval Specification

## Purpose

`runbook` notes retrieve and rank in `recall`/`query` symmetrically with `fact`/`feedback` — no
exclusion treatment (unlike `qa-question`), no dedicated query flag, no dedicated ranking
mechanism beyond one author-declared literal-trigger exception (`--text` / `triggers`, capability `runbook-lexical-triggers`). A `task_type` classification field and pre-filter were considered and rejected during
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

`runbook` notes SHALL receive no task-type-based ranking treatment — no query flag, no pre-filter, no boost. They rank against other matched notes exactly as `fact`/`feedback` notes do, on situation-similarity alone. (A `task_type` pre-filter mechanism was considered and rejected — design.md Decision 3, Non-Goals.) The single exception is an author-declared literal trigger hit (capability `runbook-lexical-triggers`): when `engram query` is given `--text` and a runbook's `triggers` entry is a substring of it, that runbook is placed ahead of all similarity-ranked items. No other kind-specific treatment exists.

#### Scenario: Runbook ranks like fact/feedback

- **WHEN** a query is executed with a `runbook` note and a `fact`/`feedback` note both matching a phrase with comparable situation-similarity scores, and no trigger hit applies
- **THEN** their relative ranking is determined by situation-similarity alone, with no kind-specific boost applied to either

#### Scenario: Trigger hit is the only exception

- **WHEN** a query is executed with `--text` and a runbook is a trigger hit
- **THEN** it precedes all similarity-ranked items; runbooks that are not trigger hits are ranked by similarity alone as before

### Requirement: A surfaced runbook SHALL render its `red_flags` and be retrievable in full

When a `runbook` note appears in a query payload (`items[]` or `candidate_l2s`), its rendered content SHALL include the `red_flags` field when present. `engram show <basename>` SHALL return the full note (frontmatter and body) so an agent can restate every step when the payload's inline content is truncated. When a runbook's `red_flags` list is large enough that the calling harness's own output truncation would otherwise drop entries silently, `engram query` and `engram show` SHALL both keep the most-recently-added entries and SHALL replace any dropped earlier entries with an explicit in-band marker naming the omission and how to retrieve the full list — `engram show <basename>` is not exempt from this guarantee merely because it is the prescribed fallback for a truncated preview.

#### Scenario: Red flags visible in the query payload
- **WHEN** a runbook note with `red_flags` is returned by `engram query`
- **THEN** the item's content includes the `red_flags` entries

#### Scenario: Full runbook via show
- **WHEN** an agent runs `engram show <runbook basename>`
- **THEN** the output contains the complete frontmatter (situation, done_when, red_flags if any) and the full step body

#### Scenario: Oversized red_flags list keeps its newest entry
- **WHEN** a runbook's `red_flags` list is large enough that rendering it in full would exceed the external output-truncation boundary the calling harness applies
- **THEN** both `engram query`'s item content and `engram show`'s output keep the most-recently-added `red_flags` entries and include an explicit marker naming how many earlier entries were omitted and how to retrieve them

