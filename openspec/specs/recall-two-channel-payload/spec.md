# Unified two-channel recall payload Specification

## Purpose

One retrieval call returns two kinds of context: Channel 1 (a clustered relevance channel with crystallized notes and raw transcript fragments ranked and grouped together) and Channel 2 (a separate, un-clustered recency channel containing the newest raw activity by ingest time). An agent re-orienting after context loss gets both "what's relevant" and "what just happened" in a single pass. Channel 1 implements relevance via k-means clustering and per-cluster candidate-note nomination. Channel 2 (provenanceRecent role, internal/cli/query.go) appends un-clustered chunks to the payload. Why: docs/architecture/adr.md ADR-0004. Validation: dev/eval/LEDGER.md#matched-note-floor (Channel 1 ranking), dev/eval/LEDGER.md#crowded-vault-capability-robustness (C4i re-rank and C5 recent-fill, both proven in same eval).

## Requirements

### Requirement: Channel 1 — Clustered relevance with matched notes/chunks

The query SHALL emit a Channel 1 payload carrying matched notes and chunks clustered by vector similarity, with per-cluster `candidate_l2s` ranked from within-cluster members. Cross-cluster diversity is delivered separately in the note channel as explore-half items (see capability `recall-centroid-sampling`), not merged into `candidate_l2s`. Items in Channel 1 appear in `clusters[].members[]`.

#### Scenario: Channel 1 clusters matched items
- **WHEN** running `engram query` with valid phrases against a vault with matched notes and chunks
- **THEN** the payload includes `clusters:` array with each cluster carrying `members: [{kind, path, cosine, ...}]` and `candidate_l2s: [{path, content, ...}]`

#### Scenario: Cross-cluster diversity via explore sampling
- **WHEN** a vocab term's centroid participates in the query's explore-half allocation (softmax over query→centroid cosine, owned by capability `recall-centroid-sampling`)
- **THEN** its sampled member notes are delivered as `provenance: explore` items in the note channel — not merged into any cluster's `candidate_l2s`

### Requirement: Channel 2 — Un-clustered recency channel

The query SHALL append a separate recency channel (Channel 2) carrying un-clustered newest-by-ingest chunks in `items[]` with provenanceRecent role, bounded by a configurable limit — the `--recent-fill` flag's semantics are owned by the recall-payload-cuts spec. These items never appear in `clusters[].members[]`.

#### Scenario: Channel 2 shows recent activity independent of relevance
- **WHEN** running `engram query` on a vault with both matched items and recent unrelated activity
- **THEN** the payload includes items with `provenance: recent` (Channel 2); these items do NOT appear in any cluster's `members[]`

### Requirement: Two-channel payload structure

The query SHALL render both channels in a single unified payload (items array + clusters), enabling the agent to process relevance and recency in one pass.

#### Scenario: Both channels in one query call
- **WHEN** running `engram query` against a populated vault
- **THEN** a single response includes both clustered relevance (clusters + candidates) and un-clustered recency (items with provenance: recent)
- **AND** no separate calls are needed to retrieve both views

#### Scenario: Channel 1 re-rank applies recency bias to matched items
- **WHEN** a matched note is newer than a competing note at similar cosine score
- **THEN** the newer note is ranked higher within its cluster (C4i re-rank proven 5/5)
