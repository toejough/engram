## ADDED Requirements

### Requirement: Limit caps returned item count
The query SHALL cap the number of items in the returned `items[]` array at
the resolved `--limit` value (flag only, no environment override; default
20 via `defaultQueryLimit`), applied as the final step after
content-budget capping and recency-channel assembly.
Before this requirement, `--limit` was report-only metadata
(`Budget.Limit`); item count was governed entirely by
clustering/candidate-nomination sizing (`matchPhraseLimit`, `matchSetCap`,
`candidateNoteK`), with no hard ceiling. This applies identically to
local, `ENGRAM_SERVER`-exclusive, and `ENGRAM_PARENT`-merged query modes.

#### Scenario: Default limit caps a large result set
- **WHEN** `engram query` runs against a vault whose matched-plus-recency
  items exceed 20 total, with no `--limit` flag given
- **THEN** the returned `items[]` array contains exactly 20 items, the
  highest-ranked ones in existing rank order

#### Scenario: Explicit --limit caps to the requested count
- **WHEN** `engram query` runs with `--limit N`
- **THEN** the returned `items[]` array contains at most N items

#### Scenario: Fewer items than the limit are all returned
- **WHEN** `engram query` runs against a vault whose matched-plus-recency
  items total fewer than the resolved `--limit`
- **THEN** every item is returned, unchanged from before this requirement
