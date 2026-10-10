# Query Engine - Technical Spec

## 792b24a Orientation

- Ownership: Repository component (the caller of the [text-embedder](./text-embedder.md), the [embedding-sidecar](./embedding-sidecar.md), the
  [chunk-index-store](./chunk-index-store.md), the [similarity-clustering](./similarity-clustering.md), the [vocabulary-engine](./vocabulary-engine.md), the [recency-ranking](./recency-ranking.md), the
  [runbook-trigger-matching](./runbook-trigger-matching.md) and the [query-payload-renderer](./query-payload-renderer.md)).
- Responsibility: answers a query. It reads the vault's notes and their sidecars and the chunk index,
  embeds each phrase, scores notes and chunks against it, keeps the best of each phrase, merges and
  bounds the matches, clusters them once, nominates each cluster's candidate notes, samples the
  vocabulary's terms for the explore half, inserts the notes that supersede a match, adds the newest
  chunks and the trigger hits, and hands the result to the renderer. It also answers a search of the
  chunk index alone. It writes nothing.
- Current realization (informational): `RunQuery` and its stages in `internal/cli/query.go`;
  `internal/cli/query_chunks.go` (`RunChunkQuery`, chunk scoring, `listJSONLIndexes`);
  `internal/cli/query_vault_meta.go` (the one pass over every note's metadata, the supersession
  ride-along); `internal/cli/query_explore.go` (the explore half). The pending-offer exclusion in
  `internal/cli/offer.go` and the merge with a parent vault in `internal/cli/merged_query.go` are
  added by [curate](../product/curate.md) and [parent-sync](../product/parent-sync.md).
- Qualification: responsibility, finding and bounding what answers a query is stated without naming
  any function; ownership, the renderer shapes the payload and the callers choose the vault and the
  flags, while this component owns what is matched, clustered and sampled; independent verification,
  it is tested through `RunQuery` and its helpers over an in-memory vault and a fixed embedder, and by
  property tests; stability, the contract survives any change to the embedder, the index layout or
  the payload's shape; necessity, every recall passes through it.
- Supported capabilities: [memory-query](../product/memory-query.md) and note-writing's [792a3t Question-and-answer retrieval](../product/note-writing.md#792a3t-question-and-answer-retrieval). Recall ([recall](../product/recall.md)), [ambient-guidance](../product/ambient-guidance.md),
  [vault-serving](../product/vault-serving.md) and [parent-sync](../product/parent-sync.md) extend it.

Rationale: the engine is one deterministic pass with no LLM, so the same vault and phrases give the
same payload, and the judgment about coverage stays with the reader. A single query process loads the
vault once and computes everything from it; every command is its own process, so nothing is cached
between queries. The matched set is bounded before it is clustered because the cost of scoring a
silhouette grows with the square of its size: the cap of 300 is the 10 phrases a deep recall uses times
the 30 kept for each, and the chunks the recency channel adds never enter the clustering. The explore
half exists because matching only finds what resembles the query: tag nomination reached further by
joining any note that shared a term with the top three notes of a cluster, which tied the reach to an
accident of rank, treated every term-sharing note alike and needed a per-cluster cap, so it was
replaced on 2026-07-29 with direct sampling by closeness of the query to each term's centroid.
Superseding notes ride along in `items`, not in a cluster's candidates, so the candidates stay the
cluster's own nearest notes.

## 792b24c Behavior Requirements

### 792b24c1 Input and defaults

Requirement: [792b24c1 Input and defaults](#792b24c1-input-and-defaults) The query-engine MUST fail with `query: empty query string` when given no phrase and
a message that is empty after trimming, MUST accept any number of phrases, and MUST use a limit of 20
when the limit is zero. It MUST fail, wrapping the cause as `query: embed`, when the embedder fails on a
phrase. It MUST give an empty list of items for a vault with no notes, and MUST report the number of
phrases it was given.

Supports: [792a7a1a A query with nothing to search for fails](../product/memory-query.md#792a7a1-uat-scenarios), [792a7a1b Several phrases are searched together](../product/memory-query.md#792a7a1-uat-scenarios), [792a7b1a The payload has the documented shape](../product/memory-query.md#792a7b1-uat-scenarios), [792a7q1a The default caps a large result](../product/memory-query.md#792a7q1-uat-scenarios).

#### 792b24c1a Scenarios

```gherkin
Scenario: 792b24c1a1 Nothing to search for is an error
  Given a vault, and no phrase and no message, or no phrase and the message "  "
  When a query runs
  Then it fails

Scenario: 792b24c1a2 Several phrases are accepted
  Given the phrases "a" and "b"
  When a query runs
  Then it succeeds and the budget counts 2 phrases queried

Scenario: 792b24c1a3 The default limit is 20
  Given 25 matched notes and no limit
  When a query runs
  Then items holds 20

Scenario: 792b24c1a4 An embedding failure is an error
  Given an embedder that fails
  When a query runs
  Then it fails with an error that names the embedder's failure

Scenario: 792b24c1a5 An empty vault has no items
  Given a vault with no notes
  When a query runs
  Then items is empty and the query succeeds
```

### 792b24c2 Usable embeddings

Requirement: [792b24c2 Usable embeddings](#792b24c2-usable-embeddings) The query-engine MUST use only the notes whose sidecar reads, is of the current
schema and was made by the embedding model of the run. It MUST warn once for the sidecars of another
model, naming how many, their model IDs, the active one and `engram embed apply --all`, and once for
sidecars of an old schema, naming `engram embed apply --force`. It MUST fail with
`query: vault has notes but no current-model embeddings; run `engram embed apply --all`` when the vault
has notes and none is usable. A note with a missing or unparseable sidecar MUST be left out silently.

Supports: [792a7c1a Notes of another model are dropped with a warning](../product/memory-query.md#792a7c1-uat-scenarios), [792a7c1b A vault with no usable embeddings is an error](../product/memory-query.md#792a7c1-uat-scenarios), [792a7c1c Old-schema sidecars are skipped with their own warning](../product/memory-query.md#792a7c1-uat-scenarios).

Rationale: a silent model swap would empty the results with no explanation, so the warning names the
cause and the remedy; the all-mismatch case fails because an empty payload would read as "nothing is
relevant".

#### 792b24c2a Scenarios

```gherkin
Scenario: 792b24c2a1 Notes without sidecars are an error
  Given a vault of notes with no sidecars
  When a query runs
  Then it fails with an error naming engram embed apply --all

Scenario: 792b24c2a2 Notes with incompatible sidecars are an error
  Given a vault of notes whose sidecars were made by another model
  When a query runs
  Then it fails with an error naming engram embed apply --all

Scenario: 792b24c2a3 Old-schema sidecars are skipped with an advisory
  Given sidecars on an old schema
  When a query runs
  Then a warning names the schema and the skipped notes are left out

Scenario: 792b24c2a4 Another model's sidecars are warned about
  Given sidecars of another model alongside current ones
  When a query runs
  Then one warning names the number dropped, their model and the active model
```

### 792b24c3 Scoring notes

Requirement: [792b24c3 Scoring notes](#792b24c3-scoring-notes) For each phrase the query-engine MUST score each usable note whose file reads by the
higher of the cosines between the phrase's vector and the note's situation and body vectors, keep the
vector of the higher as the note's coordinate, multiply the score by the recency factor of its age
([792b21c1 Recency factor](./recency-ranking.md#792b21c1-recency-factor), [792b21c2 Note age](./recency-ranking.md#792b21c2-note-age)) using the time at the end of the scan stage when timing is on and otherwise the
clock, and not multiply it when no clock was supplied. The unscaled cosine MUST be kept for the floor.
A note's content MUST show each wikilink as its display text or target, and be previewed for red flags
([792b23c8 Red-flag preview](./query-payload-renderer.md#792b23c8-red-flag-preview)). Notes MUST be ordered by score, highest first.

Supports: [792a7b1c Wikilinks are flattened in the content](../product/memory-query.md#792a7b1-uat-scenarios), [792a7d1a Notes are ranked by their better axis](../product/memory-query.md#792a7d1-uat-scenarios), [792a7f1b A long-unused note is scaled down](../product/memory-query.md#792a7f1-uat-scenarios), [792a7v1c A runbook has no kind boost](../product/memory-query.md#792a7v1-uat-scenarios).

#### 792b24c3a Scenarios

```gherkin
Scenario: 792b24c3a1 Notes are ranked by descending cosine
  Given notes whose cosines to the phrase are 0.9, 0.5 and 0.7
  When a query runs
  Then they are returned as 0.9, 0.7, 0.5

Scenario: 792b24c3a2 The situation axis can win
  Given a note whose situation vector matches the phrase and whose body vector does not
  When a query runs
  Then its score is the situation cosine

Scenario: 792b24c3a3 The body axis can win
  Given a note whose body vector matches the phrase and whose situation vector does not
  When a query runs
  Then its score is the body cosine

Scenario: 792b24c3a4 Wikilinks are flattened
  Given a note whose body contains [[1.target]] and [[2.other|the other note]]
  When a query runs
  Then the item's content has 1.target and the other note, and no [[

Scenario: 792b24c3a5 A recently used note outranks a stale one
  Given two notes with equal cosine, last used 2 days and 120 days before the clock
  When a query runs with a clock
  Then the fresher note is first with the higher score

Scenario: 792b24c3a6 A runbook competes like a fact
  Given a runbook and a fact with the same situation text, in a vault with the chunk index
  When a query runs
  Then both appear with the same score, and the runbook is among the candidate notes
```

### 792b24c4 Scoring chunks

Requirement: [792b24c4 Scoring chunks](#792b24c4-scoring-chunks) For each phrase the query-engine MUST score each chunk by the cosine between the
phrase's vector and the chunk's vector, multiplied by the recency factor ([792b21c1 Recency factor](./recency-ranking.md#792b21c1-recency-factor)) of the days since
the chunk's `ingested_at`, counting zero when it has none or the clock is absent, and of the chunk's
turn number over the highest turn number of its source ([792b21c4 Turn positions](./recency-ranking.md#792b21c4-turn-positions)), counting zero when its anchor has no
turn or its source has none above zero. The unscaled cosine MUST be kept for the floor.

Supports: [792a7f1a An old chunk is scaled down](../product/memory-query.md#792a7f1-uat-scenarios), [792a7f1c The floor ignores the scaling](../product/memory-query.md#792a7f1-uat-scenarios), [792a7f1e A late turn is lifted](../product/memory-query.md#792a7f1-uat-scenarios).

#### 792b24c4a Scenarios

```gherkin
Scenario: 792b24c4a1 A recent low-cosine chunk surfaces among stale ones
  Given a chunk ingested an hour ago with cosine 0.32 and five chunks ingested 120 days ago with cosine 0.71
  When a query runs with a clock
  Then the recent chunk is among the items

Scenario: 792b24c4a2 A newer chunk outranks an older one with the same vector
  Given two chunks with the same vector, ingested a day and 100 days before the clock
  When a query runs
  Then the newer chunk scores higher

Scenario: 792b24c4a3 The floor uses the unscaled cosine
  Given a chunk with cosine 0.53 ingested 120 days ago
  When a query runs
  Then it is a direct match, scored about 0.13
```

### 792b24c5 The matched set

Requirement: [792b24c5 The matched set](#792b24c5-the-matched-set) For each phrase the query-engine MUST rank the notes and chunks together by score,
ties by key, and keep the first 30 after the reservation of [792b24c6 The matched-note floor](#792b24c6-the-matched-note-floor). It MUST then merge the phrases
keeping, for each key, the item with the highest score, drop each item whose unscaled cosine is below
0.25, and keep at most the 300 highest-scored, in key order. It MUST give each kept item the provenance
`direct`.

Supports: [792a7d1b An item below the floor is dropped](../product/memory-query.md#792a7d1-uat-scenarios), [792a7d1c Only the best 30 of a phrase are kept](../product/memory-query.md#792a7d1-uat-scenarios), [792a7d1d An item keeps its best score across phrases](../product/memory-query.md#792a7d1-uat-scenarios), [792a7d1e The matched set is capped at 300](../product/memory-query.md#792a7d1-uat-scenarios).

#### 792b24c5a Scenarios

```gherkin
Scenario: 792b24c5a1 Items below the floor are dropped
  Given a note and a chunk whose cosines are 0.2
  When a query runs
  Then neither is a direct match

Scenario: 792b24c5a2 Only the best 30 of a phrase enter the union
  Given a phrase that 40 chunks match above the floor
  When a query runs
  Then 30 are matched, the 30 with the highest scores

Scenario: 792b24c5a3 The union is capped at 300
  Given twelve phrases that each match 30 different chunks above the floor, and generated chunk sets of any size
  When a query runs
  Then 300 chunks are clustered at most

Scenario: 792b24c5a4 An item keeps its best score across phrases
  Given a note scoring 0.4 for one phrase and 0.7 for another
  When a query runs with both
  Then it is scored 0.7

Scenario: 792b24c5a5 An item appears once
  Given a note matched by every phrase
  When a query runs
  Then items holds it once
```

### 792b24c6 The matched-note floor

Requirement: [792b24c6 The matched-note floor](#792b24c6-the-matched-note-floor) When a phrase's ranked list is longer than 30, the query-engine MUST reserve up to
5 of the 30 kept for notes whose unscaled cosine is at least 0.25, promoting the best such notes from
below the cut and evicting the lowest-scoring chunks to make room, and MUST keep the result ordered by
score. It MUST NOT promote a note below the floor or a question note, and MUST treat a vocabulary
definition note as any other note.

Supports: [792a7e1a A relevant note survives a flood of chunks](../product/memory-query.md#792a7e1-uat-scenarios), [792a7e1b A note below the floor is not promoted](../product/memory-query.md#792a7e1-uat-scenarios), [792a7e1c Several notes share the reserved slots](../product/memory-query.md#792a7e1-uat-scenarios), [792a7e1d A definition note is promoted like any other](../product/memory-query.md#792a7e1-uat-scenarios), [792a7d1g A definition note is an ordinary match](../product/memory-query.md#792a7d1-uat-scenarios).

#### 792b24c6a Scenarios

```gherkin
Scenario: 792b24c6a1 A relevant note survives a flood of chunks
  Given 30 chunks scoring above a note whose cosine is 0.4
  When the phrase's list is cut
  Then the note is kept and the lowest-scoring chunk is evicted

Scenario: 792b24c6a2 A definition note is promoted like any other
  Given 30 chunks scoring 0.50 above a definition note scoring 0.45 and a normal note scoring 0.30
  When the phrase's list is cut
  Then the definition note is kept

Scenario: 792b24c6a3 A definition note alone is returned
  Given a definition note as the only matched item
  When the phrase's list is cut
  Then the definition note is kept
```

### 792b24c7 Question notes

Requirement: [792b24c7 Question notes](#792b24c7-question-notes) The query-engine MUST leave a question note out of the matched set at each of the
four places that build it: when promoting under the matched-note floor, when merging a phrase into the
union, when dropping below the floor, and when indexing vocabulary members for the explore half. An
answer note MUST compete as any other note.

Supports: [792a3t1a The answer is found and the question is not](../product/note-writing.md#792a3t1-uat-scenarios), [792a7k1a The core outranks the fringe](../product/memory-query.md#792a7k1-uat-scenarios).

#### 792b24c7a Scenarios

```gherkin
Scenario Outline: 792b24c7a1 Which kinds are left out
  Given the content <content>
  When it is tested for exclusion from queries
  Then the result is <excluded>

  Examples:
    | case                  | content                           | excluded |
    | a vocabulary note     | ---\ntype: vocab\n---\n            | no       |
    | a vocabulary index    | ---\ntype: vocab-index\n---\n      | no       |
    | a question note       | ---\ntype: qa-question\n---\n      | yes      |
    | an answer note        | ---\ntype: qa-answer\n---\n        | no       |
    | a fact                | ---\ntype: fact\n---\n             | no       |
    | a feedback note       | ---\ntype: feedback\n---\n         | no       |
    | a runbook             | ---\ntype: runbook\n---\n          | no       |
```

### 792b24c8 Chunks and notes together

Requirement: [792b24c8 Chunks and notes together](#792b24c8-chunks-and-notes-together) The query-engine MUST rank the chunks of the chunk index with the notes when a chunk
index directory and a lister are supplied, cluster them together ([792b24c9 Clusters](#792b24c9-clusters)), and MUST behave as a
query over the vault alone when none is. A missing chunk index directory MUST be an empty index.

Supports: [792a7d1f Notes and chunks compete in one list](../product/memory-query.md#792a7d1-uat-scenarios).

#### 792b24c8a Scenarios

```gherkin
Scenario: 792b24c8a1 Chunks and notes are ranked together
  Given a note and a chunk that both match
  When a query runs with the chunk index
  Then both appear, the chunk with kind chunk and the note with kind fact

Scenario: 792b24c8a2 Without a chunk index only notes are returned
  Given no chunk index directory
  When a query runs
  Then the payload is that of the vault alone
```

### 792b24c9 Clusters

Requirement: [792b24c9 Clusters](#792b24c9-clusters) The query-engine MUST cluster the matched set once with the [similarity-clustering](./similarity-clustering.md)
([792b19c4 Choosing the number of clusters](./similarity-clustering.md#792b19c4-choosing-the-number-of-clusters)) for 2 to 7 clusters and a floor of 0.10, seeded from the FNV-1a hash of the phrases joined
by line breaks, so that the same vault and phrases give the same clusters. When no clustering is found,
or it fails, the whole set MUST be one cluster, its centroid the mean of the members' vectors and its
silhouette 0. It MUST report for each cluster the mean per-point silhouette ([792b19c5 Per-point silhouette](./similarity-clustering.md#792b19c5-per-point-silhouette)) of its members,
list its members by score, highest first, with ties kept in order, and mark as representative the
member whose vector is closest to the centroid, ties going to the higher score and then to the lower
path. An empty matched set MUST give no clusters.

Supports: [792a7g1a Matches form clusters with a representative](../product/memory-query.md#792a7g1-uat-scenarios), [792a7g1b Matches with no structure form one cluster](../product/memory-query.md#792a7g1-uat-scenarios), [792a7g1c Nothing matched means no clusters](../product/memory-query.md#792a7g1-uat-scenarios), [792a7g1d The same query gives the same clusters](../product/memory-query.md#792a7g1-uat-scenarios).

Rationale: clustering the union once, not per phrase, keeps the number of clusters small, and the seed
from the phrases removes run-to-run variation (the former invariant C1).

#### 792b24c9a Scenarios

```gherkin
Scenario: 792b24c9a1 The number of clusters stays in range
  Given any vault of one to 30 notes and a phrase
  When a query runs
  Then clusters_found is between 0 and 7

Scenario: 792b24c9a2 The same query gives the same payload
  Given a generated vault with at least six matches and a phrase
  When a query runs twice
  Then both payloads are byte for byte identical and have at least one cluster

Scenario: 792b24c9a3 Ties for the representative are broken
  Given two members at the same distance from the centroid
  When the representative is chosen
  Then the higher-scoring member wins, and on equal scores the lower path
```

### 792b24c10 Candidate notes

Requirement: [792b24c10 Candidate notes](#792b24c10-candidate-notes) The query-engine MUST give each cluster that has note members a `candidate_l2s`
list of at most five of those notes, ordered by the higher of the cosines between the cluster's
centroid and each note's situation and body vectors, highest first, ties by path, each with its
`path`, `cosine` and `content`. It MUST list no chunk, no note from another cluster and no explore or
superseding note, and a cluster with no note has none.

Supports: [792a7h1a Candidates are the cluster's nearest notes](../product/memory-query.md#792a7h1-uat-scenarios), [792a7h1b A cluster of chunks has no candidates](../product/memory-query.md#792a7h1-uat-scenarios), [792a7h1c A small cluster lists all its notes](../product/memory-query.md#792a7h1-uat-scenarios).

#### 792b24c10a Scenarios

```gherkin
Scenario: 792b24c10a1 Chunk clusters carry candidates with inline content
  Given a note about linting and eight chunks in two neighbourhoods
  When a query runs for "linter conventions"
  Then at least one cluster has candidate_l2s, and one candidate carries the note's content

Scenario: 792b24c10a2 A near-duplicate of the centroid has a high cosine
  Given 1 to 6 notes exactly at the query vector and one note within 0.3 of it on another axis
  When a query runs
  Then every cluster's first candidate has cosine at least 0.95
```

### 792b24c11 Item order

Requirement: [792b24c11 Item order](#792b24c11-item-order) The query-engine MUST order the direct matches that are notes by the number of
their provenances, then by the highest rank among them, `trigger` 4, `direct` 3, `cluster_rep` 2 and any
other 0 (no path of a query gives an item `cluster_rep`), then by score, and MUST add a provenance to an item at most once. In `items` the matched notes
MUST be followed by the notes that ride along ([792b24c17 Superseding notes](#792b24c17-superseding-notes)), the explore notes ([792b24c16 Delivering the explore half](#792b24c16-delivering-the-explore-half)), the matched
chunks by score, the recent chunks ([792b24c19 The newest chunks](#792b24c19-the-newest-chunks)), with the trigger hits moved to the front ([792b22c4 Placing the hits](./runbook-trigger-matching.md#792b22c4-placing-the-hits)).

Supports: [792a7d1f Notes and chunks compete in one list](../product/memory-query.md#792a7d1-uat-scenarios), [792a7m1a An explore note names its term](../product/memory-query.md#792a7m1-uat-scenarios), [792a7n1a The newer note follows the older one](../product/memory-query.md#792a7n1-uat-scenarios), [792a7o1a Recent chunks follow the matches](../product/memory-query.md#792a7o1-uat-scenarios).

#### 792b24c11a Scenarios

```gherkin
Scenario Outline: 792b24c11a1 The rank of a role
  Given the role <role>
  When its rank is read
  Then it is <rank>

  Examples:
    | role         | rank |
    | trigger      | 4    |
    | direct       | 3    |
    | cluster_rep  | 2    |
    | anything else | 0   |

Scenario: 792b24c11a2 More provenances sort first, then rank, then score
  Given items with two roles and one role, with equal counts and ranks, and with equal both
  When they are compared
  Then the item with more provenances precedes, then the one with the higher rank, then the higher score

Scenario: 792b24c11a3 A provenance is added once
  Given an item that already has the provenance direct
  When direct is added again, and when cluster_rep is added
  Then it keeps direct once and gains cluster_rep
```

### 792b24c12 Project filter

Requirement: [792b24c12 Project filter](#792b24c12-project-filter) When a project is requested the query-engine MUST drop each matched note whose
content, read as LF, has no frontmatter or has no `project:` line equal to it, the line being at the
start of a line inside the frontmatter and its slug made of lowercase letters, digits and hyphens, and
MUST NOT drop anything when no project is requested. The filter MUST apply to every note item, including the superseding,
explore and ride-along notes, so that a dropped note appears nowhere in the output, clusters included, and MUST NOT apply to chunk and recent-chunk items, which carry no project.

Supports: [792a7u1a Notes of other projects are dropped](../product/memory-query.md#792a7u1-uat-scenarios), [792a7u1b A body mention does not match](../product/memory-query.md#792a7u1-uat-scenarios), [792a7u1c No filter drops nothing](../product/memory-query.md#792a7u1-uat-scenarios), [792a7u1d Explore and ride-along notes of other projects are dropped](../product/memory-query.md#792a7u1-uat-scenarios).

#### 792b24c12a Scenarios

```gherkin
Scenario: 792b24c12a1 A body mention does not match
  Given a note with project: alpha only in its body
  When the project alpha is requested
  Then the note is dropped

Scenario: 792b24c12a2 Notes of other projects are dropped
  Given notes with project: alpha and project: beta
  When the project alpha is requested
  Then only the alpha note remains

Scenario: 792b24c12a3 No project keeps everything
  Given matched notes of several projects
  When no project is requested
  Then all remain

Scenario: 792b24c12a4 Only the frontmatter is read
  Given a note whose body repeats a project line after the closing fence
  When the project is tested
  Then only the frontmatter line counts

Scenario: 792b24c12a5 A full query is restricted
  Given notes of two projects
  When the user runs engram query --project alpha
  Then the items hold only the alpha notes

Scenario: 792b24c12a6 A CRLF note's project is read
  Given a note whose lines end in \r\n and whose project is widgets
  When the project widgets is tested
  Then the note matches

Scenario: 792b24c12a7 Explore and ride-along notes are filtered too
  Given an explore-half note and a ride-along note of project beta, and matched notes of project alpha
  When the user runs engram query --project alpha
  Then the items and clusters hold no beta note, and chunks are unchanged
```

### 792b24c13 Explore budget and allocation

Requirement: [792b24c13 Explore budget and allocation](#792b24c13-explore-budget-and-allocation) The query-engine MUST size the explore half as the number of distinct notes among
the matched notes after the project filter, and MUST skip it when that is zero or when
`vocab.centroids.json` is absent, unreadable or has no terms, returning an empty allocation. Otherwise it
MUST divide the budget among the file's terms by the softmax of the cosine between the mean of the
phrases' vectors and each term's centroid, divided by 0.106, after adding 0.05 once to the cosine of
each term that has a member among the matched notes, giving each term the floor of its share and the
remainder, one each, to the terms with the largest fractional parts, ties by higher cosine and then by
name, so that the allocation always sums to the budget.

Supports: [792a7i1a The halves are equal on a healthy vault](../product/memory-query.md#792a7i1-uat-scenarios), [792a7i1b Missing centroids leave only the matches](../product/memory-query.md#792a7i1-uat-scenarios), [792a7i1c No matched note means no explore half](../product/memory-query.md#792a7i1-uat-scenarios), [792a7j1a Nearer terms sample more](../product/memory-query.md#792a7j1-uat-scenarios), [792a7j1b Evidence lifts a distant term](../product/memory-query.md#792a7j1-uat-scenarios), [792a7j1c The bonus does not stack](../product/memory-query.md#792a7j1-uat-scenarios).

#### 792b24c13a Scenarios

```gherkin
Scenario: 792b24c13a1 The allocation adds up
  Given any set of terms, temperature and budget
  When the budget is allocated
  Then the allocations sum to the budget

Scenario: 792b24c13a2 Nearer terms get at least as much
  Given terms with cosines 0.95 and 0.40
  When the budget is allocated
  Then the nearer term's allocation is at least the other's

Scenario: 792b24c13a3 Nearness never reverses
  Given any two terms whose cosines differ
  When the budget is allocated
  Then the nearer term never receives less

Scenario: 792b24c13a4 No terms or no budget give nothing
  Given no terms, or a budget of zero
  When the budget is allocated
  Then the allocation is empty

Scenario: 792b24c13a5 Evidence adds a flat bonus
  Given a term with one member among the matched notes and a term with three
  When the bonus is applied
  Then each has the bonus once, and the other terms none

Scenario: 792b24c13a6 Evidence can lift a distant term
  Given a term at the vault's median cosine with a member among the matches and a term half a bonus above it with none
  When the bonus is applied
  Then the first ranks above the second

Scenario: 792b24c13a7 The bonus never reorders unevidenced terms
  Given any two terms with no member among the matches
  When the bonus is applied
  Then their order is unchanged

Scenario: 792b24c13a8 Members among the matches are found
  Given terms whose members include a matched note
  When the evidence is read
  Then only those terms are marked
```

### 792b24c14 Picking a term's core

Requirement: [792b24c14 Picking a term's core](#792b24c14-picking-a-terms-core) The query-engine MUST rank a term's member notes by the cosine between each
note's body vector and the term's centroid, highest first, ties by path, and pick the first as many as
the term was allocated, all of them when there are fewer, none for an allocation of zero or no members.
A member MUST be a note tagged `vocab/<term>` that is not a definition note or a question note and
has a vector, and a note with several terms MUST be a member of each.

Supports: [792a7k1a The core outranks the fringe](../product/memory-query.md#792a7k1-uat-scenarios), [792a7k1b Definition notes are never picked](../product/memory-query.md#792a7k1-uat-scenarios).

#### 792b24c14a Scenarios

```gherkin
Scenario: 792b24c14a1 The core comes first
  Given members at cosines 0.99, 0.9 and 0.3 to the centroid and an allocation of 2
  When the picks are made
  Then they are the 0.99 and 0.9 members

Scenario: 792b24c14a2 An allocation above the members returns them all
  Given one member and an allocation of 5
  When the picks are made
  Then the one member is returned

Scenario: 792b24c14a3 An allocation of zero picks nothing
  Given members and an allocation of 0
  When the picks are made
  Then the result is empty

Scenario: 792b24c14a4 Definitions and vectorless members are left out
  Given a definition note and a note with no vector among a term's members
  When the explore half is computed
  Then neither is picked
```

### 792b24c15 Dedupe and backfill

Requirement: [792b24c15 Dedupe and backfill](#792b24c15-dedupe-and-backfill) The query-engine MUST fill each term's allocation from its own ranked members in
order of the cosines, skipping a note that is among the matched notes or that another term already
picked, and MUST then sweep the terms again in that order, taking one note at a time from whichever
still has candidates, until the budget is met or none remain.

Supports: [792a7l1a A duplicate is replaced](../product/memory-query.md#792a7l1-uat-scenarios), [792a7l1b A note under two terms is picked once](../product/memory-query.md#792a7l1-uat-scenarios), [792a7l1c Exhausted candidates leave the budget unmet](../product/memory-query.md#792a7l1-uat-scenarios).

#### 792b24c15a Scenarios

```gherkin
Scenario: 792b24c15a1 A note under two terms is picked once
  Given a note that is the nearest member of two allocated terms
  When the picks are made
  Then it is picked under one term only, and the other term takes its next member

Scenario: 792b24c15a2 A matched note is replaced
  Given a pick that is among the matched notes and another term with an unpicked member
  When the picks are made
  Then the pick is dropped and the slot is filled from the other term

Scenario: 792b24c15a3 Too few candidates leave the budget unmet
  Given fewer eligible notes than the budget
  When the picks are made
  Then every eligible note is picked and the result is shorter than the budget
```

### 792b24c16 Delivering the explore half

Requirement: [792b24c16 Delivering the explore half](#792b24c16-delivering-the-explore-half) The query-engine MUST deliver each explore pick as an item with provenance
`explore`, its term as `source_term`, its cosine to the term's centroid as score and content, after the
matched notes and the notes that ride along and before the matched chunks, and MUST report the
allocation as the number delivered for each term.

Supports: [792a7i1a The halves are equal on a healthy vault](../product/memory-query.md#792a7i1-uat-scenarios), [792a7i1b Missing centroids leave only the matches](../product/memory-query.md#792a7i1-uat-scenarios), [792a7m1a An explore note names its term](../product/memory-query.md#792a7m1-uat-scenarios), [792a7m1b Explore notes are outside the clusters](../product/memory-query.md#792a7m1-uat-scenarios), [792a7m1c The budget counts what was delivered](../product/memory-query.md#792a7m1-uat-scenarios).

#### 792b24c16a Scenarios

```gherkin
Scenario: 792b24c16a1 A member outside the matches is delivered
  Given a term whose member is not among the matched notes
  When the explore half is computed
  Then the member is delivered under that term with its cosine

Scenario: 792b24c16a2 A zero budget delivers nothing
  Given a budget of 0
  When the explore half is computed
  Then there are no picks and the allocation is {}

Scenario: 792b24c16a3 A full query delivers an explore note
  Given a vault with a term and a member orthogonal to the query
  When a query runs
  Then the member is an item with provenance explore and its source term, and the budget reports it

Scenario: 792b24c16a4 Missing centroids leave only the matches
  Given no vocab.centroids.json
  When a query runs
  Then no item is an explore item and explore_allocated is {}
```

### 792b24c17 Superseding notes

Requirement: [792b24c17 Superseding notes](#792b24c17-superseding-notes) The query-engine MUST read, in one pass over every usable note, the notes that
supersede each other from the `supersedes:` lists of their frontmatter, and MUST insert each note that
supersedes a matched note directly after it with provenance `ride_along` and score 0 and its content, in
the order listed, unless it is already among the items or is not a usable note. It MUST examine only
the matched notes, not chunks, recent chunks or notes it inserted, and MUST change nothing when no note
supersedes another.

Supports: [792a7n1a The newer note follows the older one](../product/memory-query.md#792a7n1-uat-scenarios), [792a7n1b A note already present is not inserted again](../product/memory-query.md#792a7n1-uat-scenarios), [792a7n1c Several superseders are all inserted](../product/memory-query.md#792a7n1-uat-scenarios), [792a7n1d A missing superseder is skipped](../product/memory-query.md#792a7n1-uat-scenarios).

#### 792b24c17a Scenarios

```gherkin
Scenario: 792b24c17a1 No superseding notes changes nothing
  Given a vault with no supersedes lists
  When the ride-along is applied
  Then the items are unchanged

Scenario: 792b24c17a2 The newer note follows the older one
  Given a matched note P and a note N that supersedes it
  When the ride-along is applied
  Then N follows P with provenance ride_along

Scenario: 792b24c17a3 A note already present is not inserted again
  Given N already among the items
  When the ride-along is applied
  Then N appears once

Scenario: 792b24c17a4 A missing superseder is skipped
  Given a superseder that is not a usable note
  When the ride-along is applied
  Then nothing is inserted

Scenario: 792b24c17a5 Several superseders are all inserted
  Given two notes that supersede P
  When the ride-along is applied
  Then both follow P in order

Scenario: 792b24c17a6 The inverse of the supersedes lists is built
  Given notes whose supersedes lists name older notes
  When the inverse is built
  Then each older note maps to its superseders
```

### 792b24c18 The metadata pass

Requirement: [792b24c18 The metadata pass](#792b24c18-the-metadata-pass) The query-engine MUST read, from each usable note's frontmatter read as its LF
form, the `vocab/<term>` entries of `tags:`, in block or inline form, ignoring the bare `vocab` entry and
every other tag, the `supersedes:` entries, and the `triggers:` ([792b22c1 Trigger index](./runbook-trigger-matching.md#792b22c1-trigger-index)), and MUST index the notes by
term for the explore half, leaving out question notes. A frontmatter that is absent or does not parse
MUST give nothing. A vault with no terms and no supersedes lists MUST give empty indexes.

Supports: [792a7zg1g The lists in a CRLF note's frontmatter are read](../product/memory-query.md#792a7zg1-uat-scenarios), [792a7k1a The core outranks the fringe](../product/memory-query.md#792a7k1-uat-scenarios).

#### 792b24c18a Scenarios

```gherkin
Scenario: 792b24c18a1 The bare vocab tag contributes no term
  Given a definition note tagged [vocab]
  When the metadata is read
  Then it contributes no term

Scenario: 792b24c18a2 The term index and the inverse are built
  Given notes tagged with terms and a note that supersedes another
  When the vault is read
  Then the term index lists each term's members with their vectors, and the inverse maps the older note to its superseder

Scenario: 792b24c18a3 Only vocab terms are indexed
  Given a note tagged [work-kind/review, vocab/a, vocab/b]
  When the metadata is read
  Then the terms are a and b

Scenario: 792b24c18a4 A vault without vocabulary has empty indexes
  Given notes with no vocab tags and no supersedes
  When the vault is read
  Then the indexes are empty

Scenario: 792b24c18a5 A block-style tags list is read
  Given a note whose tags are written one per line
  When the frontmatter is parsed
  Then the vocab terms are found

Scenario: 792b24c18a6 A note without frontmatter gives nothing
  Given a text with no frontmatter
  When the frontmatter is parsed
  Then the result is empty

Scenario: 792b24c18a7 Supersedes entries are parsed
  Given a note with a supersedes list
  When the frontmatter is parsed
  Then each entry has its note, type and claim

Scenario: 792b24c18a8 A CRLF note is parsed as LF
  Given a note whose lines end in \r\n and which has tags and supersedes lists
  When the frontmatter is parsed
  Then the lists are read as for the LF note
```

### 792b24c19 The newest chunks

Requirement: [792b24c19 The newest chunks](#792b24c19-the-newest-chunks) The query-engine MUST append the newest chunks ([792b21c5 Newest chunks](./recency-ranking.md#792b21c5-newest-chunks)) up to the recent-fill
count, skipping chunks that are matched, with provenance `recent` and score 0, outside every cluster.
The count MUST be 25 when the flag is zero, the flag's value when positive and none when negative. All
chunks MUST be returned when there are fewer than the count.

Supports: [792a7o1a Recent chunks follow the matches](../product/memory-query.md#792a7o1-uat-scenarios), [792a7o1b A matched chunk is not repeated](../product/memory-query.md#792a7o1-uat-scenarios), [792a7o1c The count can be set](../product/memory-query.md#792a7o1-uat-scenarios), [792a7o1d A negative count turns the channel off](../product/memory-query.md#792a7o1-uat-scenarios), [792a7o1e Zero means the default](../product/memory-query.md#792a7o1-uat-scenarios).

#### 792b24c19a Scenarios

```gherkin
Scenario: 792b24c19a1 Recent chunks follow the matches with provenance recent
  Given a chunk index of chunks of different ages
  When a query runs
  Then the newest unmatched chunks follow with provenance recent

Scenario: 792b24c19a2 Recent chunks are in no cluster
  Given a payload with recent chunks
  When its clusters are read
  Then no recent chunk is a member

Scenario: 792b24c19a3 A matched chunk is not repeated
  Given a chunk that is both matched and among the newest
  When a query runs
  Then it appears once

Scenario: 792b24c19a4 Fewer chunks than the count are all returned
  Given three chunks and the default count
  When a query runs
  Then all three are returned

Scenario Outline: 792b24c19a5 The count resolves
  Given the flag value <flag>
  When the count is resolved
  Then it is <count>

  Examples:
    | flag | count |
    | 0    | 25    |
    | 5    | 5     |
    | -1   | 0     |
```

### 792b24c20 A query writes nothing

Requirement: [792b24c20 A query writes nothing](#792b24c20-a-query-writes-nothing) The query-engine MUST NOT write or change any file and MUST NOT put an `activated`
field on any item.

Supports: [792a7zh1a A query leaves the vault as it was](../product/memory-query.md#792a7zh1-uat-scenarios).

#### 792b24c20a Scenarios

```gherkin
Scenario: 792b24c20a1 No item is activated and no sidecar changes
  Given a high-cosine note and a low-cosine note with sidecars
  When a query runs
  Then no item has an activated field and the sidecar files are unchanged
```

### 792b24c21 The refit flag

Requirement: [792b24c21 The refit flag](#792b24c21-the-refit-flag) The query-engine MUST report a pending refit when `vocab.centroids.json` records one
and MUST report none when the file is absent or does not parse.

Supports: [792a7zd1a A pending refit is reported](../product/memory-query.md#792a7zd1-uat-scenarios), [792a7zd1b Otherwise the field is absent](../product/memory-query.md#792a7zd1-uat-scenarios).

#### 792b24c21a Scenarios

```gherkin
Scenario: 792b24c21a1 A pending refit is read from the centroids file
  Given a centroids file with refit_pending true
  When a query runs
  Then the payload has refit_pending: true
```

### 792b24c22 A note is found by its situation

Requirement: [792b24c22 A note is found by its situation](#792b24c22-a-note-is-found-by-its-situation) A note whose embedded text is S MUST be returned by a query for the phrase S, with
a cosine of 1.

Supports: [792a7d1h A note is found by its own situation](../product/memory-query.md#792a7d1-uat-scenarios).

Rationale: learn and recall are inverse over the situation field, so what was written for a situation
is found by asking for it (the former invariant R1).

#### 792b24c22a Scenarios

```gherkin
Scenario: 792b24c22a1 A note is returned for its own situation
  Given several notes whose body is their situation
  When a query runs with one situation as the phrase
  Then that note is among the items with cosine 1
```

### 792b24c23 Searching chunks alone

Requirement: [792b24c23 Searching chunks alone](#792b24c23-searching-chunks-alone) For `query-chunks` the query-engine MUST fail with
`query chunks: at least one --phrase is required` when given no phrase, MUST read every index file
([792b18c12 Listing index files](./chunk-index-store.md#792b18c12-listing-index-files)) and fail, wrapping the cause, when one cannot be listed, read or decoded, and MUST return an
empty payload with `total_chunks: 0` and without embedding when the index holds no chunk. Otherwise it
MUST score each chunk by its highest cosine over the phrases, rank by score, keep the first `limit`
(20 when zero), and print `version: 1`, `space: chunks`, the phrases, the items, the clusters of the
returned chunks, found by the [similarity-clustering](./similarity-clustering.md) for 2 to 7 clusters, a floor of 0.10 and the seed 1,
when there are at least six, and the budget `total_chunks` and `returned`.

Supports: [792a7zf1a Chunks are ranked by cosine](../product/memory-query.md#792a7zf1-uat-scenarios), [792a7zf1b The limit is honoured](../product/memory-query.md#792a7zf1-uat-scenarios), [792a7zf1c Many chunks are clustered](../product/memory-query.md#792a7zf1-uat-scenarios), [792a7zf1d A phrase is required](../product/memory-query.md#792a7zf1-uat-scenarios), [792a7zf1e An empty index is an empty payload](../product/memory-query.md#792a7zf1-uat-scenarios).

#### 792b24c23a Scenarios

```gherkin
Scenario: 792b24c23a1 Chunks are ranked by cosine
  Given a chunk index of chunks with different cosines to the phrase
  When a chunk query runs
  Then the items are in descending score order

Scenario: 792b24c23a2 The limit is honoured
  Given ten chunks and a limit of 3
  When a chunk query runs
  Then three items are returned

Scenario: 792b24c23a3 Many chunks are clustered
  Given enough chunks in two neighbourhoods
  When a chunk query runs
  Then the output has clusters

Scenario: 792b24c23a4 A phrase is required
  Given no phrase
  When a chunk query runs
  Then it fails

Scenario: 792b24c23a5 An empty index is an empty payload
  Given an index with no chunks
  When a chunk query runs
  Then total_chunks is 0

Scenario: 792b24c23a6 A missing chunk directory is an empty index
  Given a chunk directory that does not exist
  When a chunk query runs
  Then total_chunks is 0

Scenario: 792b24c23a7 A malformed index is an error
  Given an index file that does not decode
  When a chunk query runs
  Then it fails
```

### 792b24c24 Stage readings

Requirement: [792b24c24 Stage readings](#792b24c24-stage-readings) When timing is on, the query-engine MUST read the injected clock at the start of the
query and at the end of its scan, embed, cluster and nominate stages, the scan stage ending when the
notes, sidecars and chunk index have been read, the embed stage when the phrases are embedded and the
matches found, the cluster stage when the matched set is clustered, and the nominate stage when the
explore half, the ride-alongs, the chunks and the recent chunks have been assembled, and MUST use the
reading that ends the scan stage as the time for recency. It MUST NOT read the clock for timing
otherwise.

Supports: [792a7ze1a Timings are reported on request](../product/memory-query.md#792a7ze1-uat-scenarios).

#### 792b24c24a Scenarios

```gherkin
Scenario: 792b24c24a1 Each stage ends with a reading
  Given timing on and a clock that advances between readings
  When a query runs
  Then each of scan_ms, embed_ms, cluster_ms and nominate_ms is the gap between its stage's readings
```
