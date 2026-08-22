## Purpose

Surfaces which embedding model produced a query's results, so a downstream consumer comparing or fusing results from more than one vault node (a future need, not built here) can tell whether their embedding model versions actually match before treating their scores as comparable.

## ADDED Requirements

### Requirement: Query payload includes the embedding model_id
`engram query`'s payload SHALL include the embedding `model_id` used to produce that response's results.

#### Scenario: model_id present on every query response
- **WHEN** `engram query` runs, locally or served
- **THEN** the payload includes a `model_id` field naming the embedding model that produced the results
