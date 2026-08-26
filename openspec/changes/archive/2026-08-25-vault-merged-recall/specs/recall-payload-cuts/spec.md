## ADDED Requirements

### Requirement: Limit caps the relevance-ranked item count
The query SHALL cap the number of Channel 1 (relevance-ranked) items in the
returned `items[]` array at the resolved `--limit` value (flag only, no
environment override; default 20 via `defaultQueryLimit`), applied after
recency-channel assembly and before content-budget capping. Before this
requirement, `--limit` was report-only metadata (`Budget.Limit`); item
count was governed entirely by clustering/candidate-nomination sizing
(`matchPhraseLimit`, `matchSetCap`, `candidateNoteK`), with no hard
ceiling. This applies identically to local, `ENGRAM_SERVER`-exclusive, and
`ENGRAM_PARENT`-merged query modes.

`--limit` SHALL NOT cap Channel 2 (the recency channel, `provenance:
recent`) — that channel has its own dedicated budget (`--recent-fill`,
capability `recall-payload-cuts`) and SHALL survive intact even when
Channel 1 alone already reaches `--limit`. Capping the combined channels
under one budget would silently displace the entire recency channel in
any vault whose Channel 1 result count meets or exceeds `--limit` — not a
rare edge case; a routine one in a non-trivial vault.

#### Scenario: Default limit caps a large Channel 1 result set
- **WHEN** `engram query` runs against a vault whose Channel 1 (matched)
  items exceed 20, with no `--limit` flag given
- **THEN** the returned `items[]` array contains exactly 20 Channel 1
  items, the highest-ranked ones in existing rank order

#### Scenario: Explicit --limit caps Channel 1 to the requested count
- **WHEN** `engram query` runs with `--limit N`
- **THEN** the returned `items[]` array contains at most N Channel 1 items

#### Scenario: Fewer Channel 1 items than the limit are all returned
- **WHEN** `engram query` runs against a vault whose Channel 1 items total
  fewer than the resolved `--limit`
- **THEN** every Channel 1 item is returned, unchanged from before this
  requirement

#### Scenario: The recency channel survives a full Channel 1
- **WHEN** `engram query` runs against a vault whose Channel 1 (matched)
  items alone already meet or exceed the resolved `--limit`, and the
  recency channel (governed by `--recent-fill`) has items to contribute
- **THEN** the returned `items[]` array still includes the recency
  channel's items, in addition to the `--limit`-capped Channel 1 items —
  `--limit` never displaces Channel 2
