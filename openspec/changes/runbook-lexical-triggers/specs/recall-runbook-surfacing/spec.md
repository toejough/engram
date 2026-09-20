## MODIFIED Requirements

### Requirement: Runbook notes SHALL rank purely by situation-similarity

`runbook` notes SHALL receive no task-type-based ranking treatment — no query flag, no pre-filter, no boost. They rank against other matched notes exactly as `fact`/`feedback` notes do, on situation-similarity alone. (A `task_type` pre-filter mechanism was considered and rejected — design.md Decision 3, Non-Goals.) The single exception is an author-declared literal trigger hit (capability `runbook-lexical-triggers`): when `engram query` is given `--text` and a runbook's `triggers` entry is a substring of it, that runbook is placed ahead of all similarity-ranked items. No other kind-specific treatment exists.

#### Scenario: Runbook ranks like fact/feedback

- **WHEN** a query is executed with a `runbook` note and a `fact`/`feedback` note both matching a phrase with comparable situation-similarity scores, and no trigger hit applies
- **THEN** their relative ranking is determined by situation-similarity alone, with no kind-specific boost applied to either

#### Scenario: Trigger hit is the only exception

- **WHEN** a query is executed with `--text` and a runbook is a trigger hit
- **THEN** it precedes all similarity-ranked items; runbooks that are not trigger hits are ranked by similarity alone as before
