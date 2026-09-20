## MODIFIED Requirements

### Requirement: Content, recency, and item-count budgets apply to the merged set, not per-source
When `ENGRAM_PARENT` is set, `engram query` SHALL apply `--content-budget`,
`--recent-fill`, and `--limit` as a single final pass over the merged
local+parent item set, not independently to each source before merging.
`--content-budget` applies across both channels combined; `--recent-fill`
governs Channel 2 (recency) only; `--limit` governs Channel 1 (relevance)
only and never displaces Channel 2 — the two budgets are independent, not
stacked into one combined cap. Trigger hits (items whose `provenances` include `trigger`,
capability `runbook-lexical-triggers`) SHALL be split out of each source
before the `--limit` cap and placed first in the merged `items[]`: local
trigger hits first, then parent trigger hits, each in the order its source
returned them, ahead of the score-ranked Channel 1 items. They are exempt
from `--limit` (it caps only the non-trigger Channel 1 items) and are not
re-sorted by score across sources. `--content-budget` still applies to them
as part of the final pass. (`recall-payload-cuts` owns `--limit`'s base
enforcement; this requirement governs only where in the merge pipeline
these budgets apply).

#### Scenario: content-budget caps full-content chunk items across the merged set
- **WHEN** `engram query` runs with `ENGRAM_PARENT` set and
  `--content-budget N`
- **THEN** at most N chunk items in the merged payload, by rank across both
  sources, render full content; lower-ranked chunk items from either source
  render as a snippet

#### Scenario: recent-fill caps the recency channel across the merged set
- **WHEN** `engram query` runs with `ENGRAM_PARENT` set and
  `--recent-fill N`
- **THEN** the merged payload's recency channel contains at most N
  newest-by-ingest items drawn from both sources combined, not N from each
  source independently

#### Scenario: limit caps the merged Channel 1, not each source
- **WHEN** `engram query` runs with `ENGRAM_PARENT` set and `--limit N`
- **THEN** the returned payload's Channel 1 (relevance-ranked) items total
  at most N, drawn from the combined, score-ranked local+parent set — not
  N from each source independently before merging, and not counting the
  merged recency channel (which `--limit` never displaces — see
  `recall-payload-cuts`)

#### Scenario: trigger hits lead the merged payload, local before parent
- **WHEN** `engram query --text "<msg>" --limit 1` runs with `ENGRAM_PARENT`
  set, the local vault returns one trigger hit and a score-0.9 item, and
  the parent returns one trigger hit and a score-0.8 item
- **THEN** the merged `items[]` order is: local trigger hit, parent
  trigger hit, local score-0.9 item — both trigger hits survive `--limit 1`
  and the parent's score-0.8 item is dropped by the limit
