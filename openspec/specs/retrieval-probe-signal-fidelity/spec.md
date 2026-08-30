# Retrieval Probe Signal Fidelity Specification

## Purpose

A surfacing probe — a harness or script that measures whether a target note "surfaced" from an `engram query` — reads the correct signal from the query payload: the top-level `items[]` list rank, not the within-cluster `candidate_l2s` membership (which is a capped top-5 centroid-based subset used for synthesis candidate selection, not a general "did this match" signal per ADR-0025). This specification ensures probe implementations measure genuine retrieval success, not a different ranking mechanism.

## Requirements

### Requirement: A surfacing probe SHALL read `items[]` rank as the surfacing signal

Any harness or script that determines whether a target note "surfaced" for an `engram query` SHALL do so by checking the query payload's top-level `items[]` list (matching on path/basename) and reporting the note's 1-based rank there. It SHALL NOT use `clusters[].candidate_l2s` membership as the surfacing signal — `candidate_l2s` is a capped top-5-by-centroid-cosine, within-cluster subset used for L2-synthesis-candidate selection (ADR-0025), not a general "did this note match" signal, and a note can be correctly matched and present in `items[]` while absent from every cluster's `candidate_l2s` simply because its cluster has more than 5 members closer to the centroid.

#### Scenario: Target present in items[] counts as surfaced

- **WHEN** a probe runs a query and the target note's path appears in the payload's top-level `items[]` list
- **THEN** the probe reports the note as surfaced, at its 1-based rank within `items[]`

#### Scenario: candidate_l2s absence does not count as a miss

- **WHEN** a target note is present in `items[]` but is not among any cluster's `candidate_l2s` entries
- **THEN** the probe still reports the note as surfaced — `candidate_l2s` absence is never treated as a retrieval failure by the probe
