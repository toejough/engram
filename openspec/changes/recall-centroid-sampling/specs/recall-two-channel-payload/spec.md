# Delta: recall-two-channel-payload — explore sampling replaces tag nomination

## MODIFIED Requirements

### Requirement: Channel 1 — Clustered relevance with matched notes/chunks

The query SHALL emit a Channel 1 payload carrying matched notes and chunks clustered by vector similarity, with per-cluster `candidate_l2s` ranked from within-cluster members. Cross-cluster diversity is delivered separately in the note channel as explore-half items (see capability `recall-centroid-sampling`), not merged into `candidate_l2s`. Items in Channel 1 appear in `clusters[].members[]`.

#### Scenario: Channel 1 clusters matched items
- **WHEN** running `engram query` with valid phrases against a vault with matched notes and chunks
- **THEN** the payload includes `clusters:` array with each cluster carrying `members: [{kind, path, cosine, ...}]` and `candidate_l2s: [{path, content, ...}]`

#### Scenario: Cross-cluster diversity via explore sampling
- **WHEN** a vocab term's centroid participates in the query's explore-half allocation (softmax over query→centroid cosine, owned by capability `recall-centroid-sampling`)
- **THEN** its sampled member notes are delivered as `provenance: explore` items in the note channel — not merged into any cluster's `candidate_l2s`
