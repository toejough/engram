# Similarity Clustering - Technical Spec

## 792b19a Orientation

- Ownership: Repository component.
- Responsibility: partitions a set of embedding vectors into clusters by cosine distance, scores a
  partition by its silhouette, and chooses the number of clusters with the best score within a range,
  returning each vector's cluster and each cluster's mean. Given the same vectors, range and seed it
  always returns the same result. It does no input or output.
- Current realization (informational): `CosineDistance`, `KMeans`, `Silhouette` and `AutoK` in
  `internal/cluster`; called by `deriveAtKRange` in `internal/cli/vocab_derive.go` (K from 2 to 40,
  floor 0.02, seed 1), and by the query path.
- Qualification: responsibility, clustering vectors is stated without naming any function;
  ownership, callers decide which vectors to cluster, the range, the floor and the seed, and what a
  cluster means, while this component owns the partition and its score; independent verification,
  it is tested directly on vectors; stability, the contract survives any change to its callers or to
  how vectors are stored; necessity, the vocabulary's terms are its clusters.
- Supported capabilities: [controlled-vocabulary](../product/controlled-vocabulary.md), [memory-query](../product/memory-query.md). Vault-migration ([vault-migration](../product/vault-migration.md)) adds its
  mappings and requirements.

Rationale: the vocabulary is derived from the vault's own geometry, so it reuses the clustering that
recall already relied on rather than a second algorithm. A fixed seed makes clustering
deterministic, so the same vault always yields the same vocabulary and naming requests stay valid
between two runs.

## 792b19c Behavior Requirements

### 792b19c1 Cosine distance

Requirement: [792b19c1 Cosine distance](#792b19c1-cosine-distance) The similarity-clustering MUST measure the distance between two vectors as one
minus their cosine similarity, and MUST return 1 for vectors of different lengths, empty vectors or a
zero vector.

Supports: [792a6j1a The term count follows the vault's clusters](../product/controlled-vocabulary.md#792a6j1-uat-scenarios).

#### 792b19c1a Scenarios

```gherkin
Scenario Outline: 792b19c1a1 Distances between vectors
  Given the vectors <a> and <b>
  When their distance is measured
  Then it is <distance>

  Examples:
    | a          | b           | distance |
    | [1, 0]     | [1, 0]      | 0        |
    | [1, 0]     | [0, 1]      | 1        |
    | [1, 0]     | [-1, 0]     | 2        |
    | [0, 0]     | [1, 0]      | 1        |
```

### 792b19c2 K-means partition

Requirement: [792b19c2 K-means partition](#792b19c2-k-means-partition) The similarity-clustering MUST, given vectors, a cluster count k and a seed,
pick k starting centroids by distance-weighted random draws from a generator seeded by the seed,
then repeatedly assign each vector to its nearest centroid ([792b19c1 Cosine distance](#792b19c1-cosine-distance)) and move each centroid to the
mean of its vectors, stopping when no assignment changes or after 100 rounds, and return every
vector's cluster and every centroid. The same vectors, k and seed MUST give the same result. It MUST
fail when there are no vectors, when k is less than 2, or when k exceeds the number of vectors.

Supports: [792a6j1a The term count follows the vault's clusters](../product/controlled-vocabulary.md#792a6j1-uat-scenarios).

Rationale: the earlier invariant C1 asked that clustering return the same result for a fixed vault
and phrase; the query path meets it by seeding from the phrase, and refit by a fixed seed, both
relying on this determinism.

#### 792b19c2a Scenarios

```gherkin
Scenario: 792b19c2a1 The same seed gives the same partition
  Given a set of vectors, a cluster count and a seed
  When they are clustered twice
  Then both runs return the same assignments and centroids

Scenario: 792b19c2a2 Separated groups are found
  Given two well-separated groups of three vectors and a count of 2
  When they are clustered
  Then each group falls in its own cluster

Scenario Outline: 792b19c2a3 Impossible counts are refused
  Given <vectors>
  When they are clustered with k = <k>
  Then it fails

  Examples:
    | vectors          | k |
    | no vectors       | 2 |
    | two vectors      | 5 |
    | two vectors      | 1 |
```

### 792b19c3 Silhouette score

Requirement: [792b19c3 Silhouette score](#792b19c3-silhouette-score) The similarity-clustering MUST score a partition as the mean, over every
vector, of (b − a) / max(a, b), where a is the vector's mean distance to the other members of its
cluster and b its smallest mean distance to another cluster's members, counting 0 for a vector that
is alone in its cluster or whose a and b are both 0, and MUST score an empty set 0.

Supports: [792a6j1a The term count follows the vault's clusters](../product/controlled-vocabulary.md#792a6j1-uat-scenarios), [792a6l1a Dry run reports the derivation diff without writing](../product/controlled-vocabulary.md#792a6l1-uat-scenarios).

#### 792b19c3a Scenarios

```gherkin
Scenario: 792b19c3a1 Well-separated clusters score high
  Given two tight, well-separated clusters
  When the partition is scored
  Then the score is above 0.5

Scenario: 792b19c3a2 A poor partition scores low
  Given vectors assigned to clusters at random
  When the partition is scored
  Then the score is below 0.1

Scenario: 792b19c3a3 Singleton clusters score zero
  Given a partition in which every vector is alone in its cluster
  When the partition is scored
  Then the score is 0
```

### 792b19c4 Choosing the number of clusters

Requirement: [792b19c4 Choosing the number of clusters](#792b19c4-choosing-the-number-of-clusters) The similarity-clustering MUST, given vectors, a range of cluster counts, a
floor and a seed, partition the vectors ([792b19c2 K-means partition](#792b19c2-k-means-partition)) at every count from the range's minimum to the
smaller of its maximum and the number of vectors; move each vector left alone in its cluster to the
nearest cluster with more than one member; skip a count that leaves fewer than two clusters; renumber
the clusters from 0; and keep the partition with the highest silhouette score ([792b19c3 Silhouette score](#792b19c3-silhouette-score)) above 0,
the earliest count on a tie. It MUST return that partition's cluster count, score, assignments and
member means, and MUST return no clusters (K of 0) when there are fewer vectors than the range's
minimum, when the range is empty, or when the best score is below the floor.

Supports: [792a6j1a The term count follows the vault's clusters](../product/controlled-vocabulary.md#792a6j1-uat-scenarios), [792a6j1b A vault without structure keeps its vocabulary](../product/controlled-vocabulary.md#792a6j1-uat-scenarios), [792a6k1a A cluster keeps its term's name](../product/controlled-vocabulary.md#792a6k1-uat-scenarios), [792a6l1a Dry run reports the derivation diff without writing](../product/controlled-vocabulary.md#792a6l1-uat-scenarios).

Rationale: a cluster of one vector would make a term of one note, so singletons are folded into
their nearest real cluster before the partition is scored. The floor lets each caller reject input
without structure rather than force clusters onto it.

#### 792b19c4a Scenarios

```gherkin
Scenario: 792b19c4a1 Three groups give three clusters
  Given three well-separated groups of four vectors and a range of 2 to 7
  When the number of clusters is chosen with a floor of 0.10
  Then K is 3 and the score is above 0.5

Scenario: 792b19c4a2 No structure gives no clusters
  Given eight identical vectors
  When the number of clusters is chosen with a floor of 0.10
  Then K is 0 and no assignments are returned

Scenario: 792b19c4a3 The range is capped by the number of vectors
  Given three vectors and a range of 2 to 7
  When the number of clusters is chosen
  Then K is at most 3

Scenario: 792b19c4a4 Singletons are folded into a cluster
  Given five close vectors and one outlier, and a range of 2 to 2
  When the number of clusters is chosen
  Then every returned cluster has at least two members
```

### 792b19c5 Per-point silhouette

Requirement: [792b19c5 Per-point silhouette](#792b19c5-per-point-silhouette) The similarity-clustering MUST score one point, given every vector, the members of
each cluster, the point's cluster and its own index, as (b - a) / max(a, b) with a and b as in [792b19c3 Silhouette score](#792b19c3-silhouette-score),
and MUST score 0 when the point's cluster is out of range or holds only that point, when no other
non-empty cluster exists, and when a and b are both 0.

Supports: [792a7g1a Matches form clusters with a representative](../product/memory-query.md#792a7g1-uat-scenarios).

Rationale: the query reports each cluster's silhouette as the mean of its members' scores, which the
whole-partition mean of [792b19c3 Silhouette score](#792b19c3-silhouette-score) cannot give.

#### 792b19c5a Scenarios

```gherkin
Scenario: 792b19c5a1 A point in a separated cluster scores high
  Given two clusters of two well-separated vectors each
  When the first vector is scored
  Then the score is above 0.5

Scenario: 792b19c5a2 A point alone in its cluster scores zero
  Given two clusters of one vector each
  When the first vector is scored
  Then the score is 0

Scenario: 792b19c5a3 An out-of-range cluster scores zero
  Given a point whose cluster number is -1 or beyond the clusters
  When it is scored
  Then the score is 0

Scenario: 792b19c5a4 Identical vectors score zero
  Given four identical vectors in two clusters of two
  When the first vector is scored
  Then the score is 0
```
