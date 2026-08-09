## MODIFIED Requirements

### Requirement: Channel 1 — Clustered relevance with matched notes/chunks

The query SHALL emit a Channel 1 payload carrying matched notes and chunks clustered by vector similarity, with per-cluster `candidate_l2s` ranked from within-cluster members. Cross-cluster diversity is delivered separately in the note channel as explore-half items (see capability `recall-centroid-sampling`), not merged into `candidate_l2s`. Items in Channel 1 appear in `clusters[].members[]`.

The centroid used to rank `candidate_l2s` within a cluster is the unweighted vector mean of cluster members. A recency-weighted centroid (`centroid = Σ(recency_i · vec_i) / Σ recency_i`) was evaluated against real production vault data (16 eligible same-cluster supersedes pairs, clusters with >5 note members) and found to produce a zero-effect result — identical over-surfacing (8/16, 50.0%) and identical mean top-5 relevance (0.6480) to the unweighted centroid — so it did not clear the eval-gate and was not shipped. See `dev/eval/LEDGER.md` (`#652-recency-weighted-centroid-null`) for the full eval record.

#### Scenario: Clustered payload includes ranked candidates
- **WHEN** a query returns matched notes and chunks
- **THEN** the payload includes `clusters:` array with each cluster carrying `members: [{kind, path, cosine, ...}]` and `candidate_l2s: [{path, content, ...}]`

#### Scenario: Explore-half items are not merged into candidate_l2s
- **WHEN** a vocab term's centroid participates in the query's explore-half allocation (softmax over query→centroid cosine, owned by capability `recall-centroid-sampling`)
- **THEN** its sampled member notes are delivered as `provenance: explore` items in the note channel — not merged into any cluster's `candidate_l2s`

#### Scenario: Recency-weighted centroid was evaluated and did not ship
- **WHEN** the recency-eval harness ran comparing unweighted vs. recency-weighted centroid nomination on 16 real same-cluster supersedes pairs with more than 5 note members
- **THEN** the weighted formula showed zero reduction in superseded-note over-surfacing (50.0% vs. 50.0%) and identical relevance (0.6480 vs. 0.6480), so the unweighted centroid remains in place and `internal/cli/query.go` was not changed
