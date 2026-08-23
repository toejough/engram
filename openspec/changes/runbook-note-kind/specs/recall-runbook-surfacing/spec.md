## ADDED Requirements

### Requirement: Runbook notes SHALL compete in the query pipeline's main matched set

`type: runbook` notes SHALL receive no retrieval-exclusion treatment (no `isQueryExcludedKind` membership while that mechanism exists — if #727 has removed it, no exclusion mechanism is added); they SHALL appear in `candidate_l2s` alongside `fact` and `feedback` notes.

#### Scenario: Runbook note inclusion in query results

- **WHEN** a query is executed and a `runbook` note is relevance-ranked above the match floor
- **THEN** it appears in the query's `candidate_l2s` list like any other retrieval-competitive note

### Requirement: `engram query` SHALL accept an optional `--task-type` flag

The task-type signal reaches the query pipeline through one interface: an optional `--task-type <free-form-slug>` flag on `engram query`. The recall skill SHALL infer the session's current task type and pass it; other callers MAY pass it directly. The flag being absent SHALL be valid and SHALL change no behavior for non-runbook notes.

#### Scenario: Task type provided via flag

- **WHEN** `engram query --task-type <slug> --phrase ...` is invoked
- **THEN** the task-type pre-filter (below) applies to `runbook` notes, and `fact`/`feedback`/chunk ranking is unchanged

#### Scenario: Flag absent

- **WHEN** `engram query` is invoked without `--task-type`
- **THEN** the query behaves exactly as before this capability existed, aside from `runbook` notes competing on situation-similarity

### Requirement: Runbook notes SHALL be pre-filtered/boosted by matching `task_type`

When a task type is provided via `--task-type`, `runbook` notes whose `task_type` matches by embedding similarity SHALL be boosted ahead of pure situation-similarity ranking, mirroring the matched-note-floor pattern (`recall-matched-note-floor` spec).

#### Scenario: Task-type match boosts a runbook note

- **WHEN** a recall/query call carries a task type that embedding-matches a `runbook` note's `task_type`
- **THEN** that runbook note is ranked ahead of notes it would otherwise tie or lose to on situation-similarity alone

#### Scenario: No task-type signal falls back to situation-similarity only

- **WHEN** a recall/query call carries no task type (or none of the vault's `runbook` notes match by embedding similarity)
- **THEN** `runbook` notes are ranked purely by situation-similarity, same as `fact`/`feedback` notes

### Requirement: Shipping this capability SHALL pass a pre-ship validation gate

Before the task-type pre-filter (surfacing mechanism) ships, the following SHALL be completed and their results documented in the shipping commit/PR: (1) a vault supply check quantifying existing kind-4 feedback content that is already runbook-shaped, (2) an A/B recall test on a recurring task class comparing pre-filter-on vs pre-filter-off, run to the harness standards in tasks.md 3.2 (headless fresh-process trials, real binary with real fixture sidecars, per-arm delivery gate, planted distractor runbooks, noise floor from same-contrast repeats) against bars pre-registered before the first scored run, (3) a cross-check of the approach against SPL's own eval methodology.

#### Scenario: Gate blocks ranking implementation until evidence is in

- **WHEN** the supply check or A/B recall test is not yet complete
- **THEN** the task-type pre-filter/boost logic SHALL NOT be shipped; capture (learn-runbook-capture) may ship independently since it does not depend on the gate

#### Scenario: A/B test shows no lift or a regression

- **WHEN** the A/B recall test shows no lift above its pre-registered bar (a gap below the measured noise floor reads "underpowered", not "no lift"), or a regression
- **THEN** the surfacing mechanism decision SHALL be revisited before the ranking logic ships, and the outcome documented in the PR/commit rather than shipped as originally designed
