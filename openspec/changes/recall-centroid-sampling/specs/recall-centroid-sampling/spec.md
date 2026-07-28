# Spec: recall-centroid-sampling

## ADDED Requirements

### Requirement: Payload composition is half exploit, half explore
The recall note-channel payload SHALL be composed of two equal halves: the exploit half (the existing cosine-nearest notes to the query, with floors and caps unchanged) and the explore half (notes sampled from global vocab clusters). When the explore half cannot be filled (insufficient distinct notes or no centroids), the exploit half SHALL fill the remaining budget and the shortfall SHALL be visible in the query budget.

#### Scenario: Balanced payload on a healthy vault
- **WHEN** a query runs against a vault with a valid `vocab.centroids.json` and sufficient notes
- **THEN** the delivered note-channel payload contains an exploit half of query-nearest notes and an explore half of cluster-sampled notes of equal size

#### Scenario: Missing centroids degrade loudly to exploit-only
- **WHEN** `vocab.centroids.json` is absent or unreadable
- **THEN** the full note budget is filled by exploit matches and the query budget reports an empty explore allocation (`explore_allocated: {}`)

### Requirement: Explore allocation by softmax of query-centroid similarity
The explore budget SHALL be allocated across vocab terms proportionally to softmax(cosine(query vector, term centroid) / τ), with a fixed temperature τ. No radius or hard cutoff SHALL gate participation — distant clusters receive allocations approaching zero naturally. A term whose members appear in the exploit half MAY receive a bounded additive weight bonus (match-evidence boost).

#### Scenario: Nearer concepts sample more
- **WHEN** the explore allocation is computed for centroids A (higher query similarity) and B (lower)
- **THEN** A's allocated sample count is greater than or equal to B's, and totals across terms equal the explore budget

#### Scenario: Match evidence boosts a distant cluster
- **WHEN** a term's centroid is distant from the query but one of its member notes appears in the exploit half
- **THEN** that term's allocation weight includes the bounded match-evidence bonus

### Requirement: Within-cluster selection is centroid-proximal
Explore samples for a term SHALL be selected from the term's member notes (top-1 assignment; definition notes excluded) in descending order of cosine similarity to the term centroid, so each concept is represented by its core rather than its assignment-floor fringe.

#### Scenario: Core members outrank fringe members
- **WHEN** a term receives an allocation of k samples and has more than k members
- **THEN** the k members most similar to the term centroid are selected

### Requirement: Dedupe and backfill keep the explore half full
Explore selections that duplicate an exploit-half note, or duplicate a selection from another cluster, SHALL be dropped, and the freed slots SHALL be refilled following the remaining softmax allocation order until the explore budget is met or candidates are exhausted.

#### Scenario: Duplicate is replaced from the next-weighted cluster
- **WHEN** an explore selection is already present in the exploit half and other clusters retain unselected members
- **THEN** the duplicate is dropped and the slot is refilled from the highest-remaining-weight cluster

### Requirement: Explore provenance is reported
Every explore-half note SHALL carry `provenance: explore` and its source term, and the query budget SHALL report per-term allocation counts as `explore_allocated`.

#### Scenario: Budget reports allocations
- **WHEN** a query completes with a non-empty explore half
- **THEN** the query budget contains `explore_allocated` mapping each contributing term to its delivered sample count, and each explore note names its source term
