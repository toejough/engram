# Memory Query - Product Spec

`engram query` searches the vault's notes and the chunk index for what is relevant to a set of
phrases and prints one YAML payload: the matched notes and chunks, the clusters they fall into, the
newest chunks, and a budget of counts. `engram show` prints one note in full, `engram show-chunk`
prints one chunk's text, and `engram query-chunks` searches the chunk index alone. All four only
read. The `recall` skill is the main caller and decides what to do with the payload. A query on a
vault that does not exist creates it first and returns an empty payload ([792a2e First-use vault creation](./vault-identity.md#792a2e-first-use-vault-creation)). The payload's
`pending_offers` flag and the exclusion of pending offers from results belong to [curate](./curate.md), results
merged in from a parent vault belong to [parent-sync](./parent-sync.md), and the served form of a query belongs to
vault-serving ([792a20m A served query takes the message and can return dedupe keys](./vault-serving.md#792a20m-a-served-query-takes-the-message-and-can-return-dedupe-keys)). Which kinds of note a query returns, and that it never returns a question note, is
note-writing's [792a3t Question-and-answer retrieval](./note-writing.md#792a3t-question-and-answer-retrieval).

## 792a7a Query input

Requirement: [792a7a Query input](#792a7a-query-input) `engram query` MUST accept one or more `--phrase` values and an optional
`--text`, and MUST fail with `query: empty query string` and a non-zero exit when it is given no
phrase and a `--text` that is empty or only whitespace.

### 792a7a1 UAT Scenarios

```gherkin
Scenario: 792a7a1a A query with nothing to search for fails
  Given any vault
  When the user runs engram query with no --phrase and no --text, or with --text "   "
  Then it fails with "query: empty query string" and exits non-zero

Scenario: 792a7a1b Several phrases are searched together
  Given a vault holding a note about Go tests and a chunk about baking
  When the user runs engram query --phrase "writing Go tests" --phrase "how to bake sourdough bread"
  Then the payload's phrases lists both phrases in order, and its items hold the note and the chunk
```

## 792a7b The payload

Requirement: [792a7b The payload](#792a7b-the-payload) `engram query` MUST print one YAML document on stdout with `version: 1`, the
`phrases` it was given, `items`, `clusters`, a `budget` and the `model_id` ([792a7zc The embedding model](#792a7zc-the-embedding-model)), and MUST give
each item a `path`, a `kind`, a `score`, its `provenances` and, unless the item was reduced to path
and score ([792a7s Lazy chunks](#792a7s-lazy-chunks)), its `content`. A note item's `path` MUST be the note's file name and its `kind`
the note's `type`, or `unknown` when it has none; a chunk item's `path` MUST be its chunk ID
`<source>#<anchor>` ([chunk-indexing](./chunk-indexing.md)) and its `kind` `chunk`. A note's content MUST show each `[[target]]` as
`target` and each `[[target|display]]` as `display`. The `budget` MUST report `phrases_queried`,
`total_notes`, `with_embeddings`, `clusters_found`, `direct_hits_returned` (the items whose
provenances include `direct`), `items_with_full_content`, `limit`, `content_budget`,
`chunks_snippeted` and `explore_allocated`.

Rationale: one call returns both "what is relevant" and "what just happened", so an agent
re-orienting after losing its context reads one document instead of making separate calls. Inline wikilink syntax is noise in the payload, because the matched set is already
returned in `items`.

### 792a7b1 UAT Scenarios

```gherkin
Scenario: 792a7b1a The payload has the documented shape
  Given a vault of six notes, each with a current sidecar, and no chunk index
  When the user runs engram query --phrase "writing Go tests"
  Then stdout is a YAML document with version 1, phrases ["writing Go tests"], items, clusters and a budget whose total_notes and with_embeddings are both 6, and a model_id

Scenario: 792a7b1b A note item is labelled by its type
  Given a vault holding a fact note
  When the user runs engram query with a phrase that matches it
  Then its item has kind fact, provenances [direct] and its full file text as content

Scenario: 792a7b1c Wikilinks are flattened in the content
  Given a matched note whose body contains [[1.target]] and [[2.other|the other note]]
  When the user runs engram query with a phrase that matches it
  Then its content contains 1.target and the other note, and no [[
```

## 792a7c Embeddings of the current model

Requirement: [792a7c Embeddings of the current model](#792a7c-embeddings-of-the-current-model) `engram query` MUST ignore a note whose sidecar was made by another embedding
model, and MUST print one warning on stderr naming the number dropped, their model IDs and
`engram embed apply --all`. It MUST ignore a note whose sidecar is on an old schema and print one
warning naming the number skipped and `engram embed apply --force`. When the vault has notes but none
has a usable sidecar it MUST fail with `query: vault has notes but no current-model embeddings; run
`engram embed apply --all``.

Rationale: a silent model swap would otherwise empty the results without saying why, so the
warning names the cause and the remedy. The all-mismatch case fails because an empty payload would
read as "nothing relevant".

### 792a7c1 UAT Scenarios

```gherkin
Scenario: 792a7c1a Notes of another model are dropped with a warning
  Given a vault of ten notes of which nine have sidecars made by another model
  When the user runs engram query --phrase pasta
  Then stderr says "dropped 9 sidecar(s)" and names the other model and the active one, and the one note with a current sidecar can still be returned

Scenario: 792a7c1b A vault with no usable embeddings is an error
  Given a vault of notes whose sidecars are all from another model, or missing
  When the user runs engram query --phrase pasta
  Then it fails with "query: vault has notes but no current-model embeddings; run `engram embed apply --all`"

Scenario: 792a7c1c Old-schema sidecars are skipped with their own warning
  Given a vault in which a sidecar is on an old schema version
  When the user runs engram query --phrase pasta
  Then stderr says "1 sidecar(s) on an old schema were skipped" and names engram embed apply --force
```

## 792a7d Matched notes and chunks

Requirement: [792a7d Matched notes and chunks](#792a7d-matched-notes-and-chunks) For each phrase `engram query` MUST score every note by the higher of its
situation and body cosine to the phrase, and every indexed chunk by its cosine, rank them together
by recency-scaled score ([792a7f Recency scaling](#792a7f-recency-scaling)) with ties by path, and keep the 30 best. It MUST then merge the
phrases keeping each item's best score, drop every item whose unscaled cosine is below 0.25, and
keep at most the 300 highest-scored. Every item kept MUST carry the provenance `direct`. The
matched notes MUST come first in `items`, highest score first, and the matched chunks, highest score
first, MUST follow the notes and the other note items ([792a7m Explore picks are reported](#792a7m-explore-picks-are-reported), [792a7n Superseding notes ride along](#792a7n-superseding-notes-ride-along)).
Only question notes ([792a3t Question-and-answer retrieval](./note-writing.md#792a3t-question-and-answer-retrieval)) MUST be left out of the matched set: a vocabulary definition note
([792a6a Definition notes](./controlled-vocabulary.md#792a6a-definition-notes)) MUST be matched like any other note.

Rationale: the floor is applied to the unscaled cosine, so a topically perfect but old item still
qualifies (vault note 53). The cap bounds the cost of clustering the set, whose silhouette scoring
is quadratic in its size, at a few hundred items however large the vault and the chunk index grow.
Notes and chunks are ranked together so that a transcript fragment and the lesson distilled from it
compete on one scale.

### 792a7d1 UAT Scenarios

```gherkin
Scenario: 792a7d1a Notes are ranked by their better axis
  Given a note whose situation is close to the phrase and whose body is not, and a note whose body is close and whose situation is not
  When the user runs engram query with that phrase
  Then each note's score is its higher cosine, and the notes are ordered by it, highest first

Scenario: 792a7d1b An item below the floor is dropped
  Given a note and a chunk whose cosine to the phrase is below 0.25
  When the user runs engram query with that phrase
  Then neither appears among the direct matches

Scenario: 792a7d1c Only the best 30 of a phrase are kept
  Given a phrase that 40 chunks match above the floor
  When the user runs engram query with it
  Then 30 of them are direct matches, the 30 with the highest scores

Scenario: 792a7d1d An item keeps its best score across phrases
  Given a note that scores 0.4 for one phrase and 0.7 for another
  When the user runs engram query with both phrases
  Then the note appears once, scored 0.7

Scenario: 792a7d1e The matched set is capped at 300
  Given twelve phrases that each match 30 different chunks above the floor
  When the user runs engram query with them
  Then 300 matched items are clustered, the 300 with the highest scores

Scenario: 792a7d1f Notes and chunks compete in one list
  Given a vault note and an indexed chunk that match the phrase
  When the user runs engram query with that phrase
  Then both appear, the note first

Scenario: 792a7d1g A definition note is an ordinary match
  Given a vault whose only note close to the phrase is a vocabulary definition note
  When the user runs engram query with that phrase
  Then the definition note is returned

Scenario: 792a7d1h A note is found by its own situation
  Given several notes whose body text is their situation
  When the user runs engram query with one note's situation as the phrase
  Then that note is among the items, with cosine 1
```

## 792a7e Matched-note floor

Requirement: [792a7e Matched-note floor](#792a7e-matched-note-floor) Before keeping a phrase's 30 best, `engram query` MUST reserve up to 5 of the
slots for notes whose unscaled cosine is at least 0.25, promoting the best such notes ranked below
the cut and evicting the lowest-scoring chunks to make room. A note whose unscaled cosine is below
0.25 MUST NOT be promoted.

Rationale: without the reservation, a flood of transcript chunks with higher cosine pushed a
crystallized lesson out of the matched set entirely: a note the embedder ranks in its top five when
notes are searched alone reached the top five of the combined ranking only 22% of the time, against
83% once the reservation existed (measured 2026-06-28, `dev/eval/LEDGER.md#matched-note-floor`). The
floor means a promoted note is relevant in its own right, so the reservation never surfaces an
irrelevant one.

### 792a7e1 UAT Scenarios

```gherkin
Scenario: 792a7e1a A relevant note survives a flood of chunks
  Given a phrase that matches 30 or more chunks above a note that scores 0.4 for it
  When the user runs engram query with that phrase
  Then the note appears among the results, in place of the lowest-scoring chunk

Scenario: 792a7e1b A note below the floor is not promoted
  Given a phrase that matches 30 or more chunks, and a note that scores 0.2 for it
  When the user runs engram query with that phrase
  Then the note is not among the results

Scenario: 792a7e1c Several notes share the reserved slots
  Given a phrase that 40 chunks match above seven notes that each score above 0.25
  When the user runs engram query with that phrase
  Then five of the notes appear, the five with the highest scores

Scenario: 792a7e1d A definition note is promoted like any other
  Given a phrase that 30 chunks match above a vocabulary definition note that scores 0.45
  When the user runs engram query with that phrase
  Then the definition note appears among the results
```

## 792a7f Recency scaling

Requirement: [792a7f Recency scaling](#792a7f-recency-scaling) `engram query` MUST multiply an item's cosine by a recency factor of
2 to the power of minus its age in days over 60 before ranking, so that an item loses half its score
every 60 days. A chunk's age MUST be the time since it was first indexed ([792a7p When a chunk was first indexed](#792a7p-when-a-chunk-was-first-indexed)), and a chunk from
a transcript turn MUST also gain up to 20% in proportion to how late in its session the turn
falls. A note's age MUST be the time since the date in its sidecar's `last_used`, or, when it has
none, since its `created` date when that is written as a bare date; a note with neither MUST NOT be
scaled. An item with an age of zero, or with an unreadable date, MUST NOT be scaled.

Rationale: operators work in monthly cycles, so the decay is a soft tilt that favours recent
content without drowning relevance (2 weeks 0.85, 1 month 0.71, 2 months 0.50). The newest chunks
are guaranteed a place separately by the recency channel ([792a7o The newest chunks](#792a7o-the-newest-chunks)), so the scaling need not carry
that guarantee. A recency-weighted cluster centroid was tried and did not ship ([792a7h Candidate notes](#792a7h-candidate-notes)). Notes that
`engram learn` writes quote their `created` date, which cannot be read as a bare date, so such a
note is not scaled until it has been used (#774).

### 792a7f1 UAT Scenarios

```gherkin
Scenario: 792a7f1a An old chunk is scaled down
  Given an indexed chunk first indexed 120 days ago, and the same chunk indexed today
  When the user runs engram query with a phrase that matches it
  Then the old chunk's score is a quarter of the fresh chunk's

Scenario: 792a7f1b A long-unused note is scaled down
  Given two notes with the same text, one whose sidecar says last_used today and one 120 days ago
  When the user runs engram query with a phrase that matches them
  Then the first scores four times the second and is listed first

Scenario: 792a7f1c The floor ignores the scaling
  Given a chunk first indexed 120 days ago whose cosine to the phrase is 0.53
  When the user runs engram query with that phrase
  Then the chunk is a direct match, scored about 0.13

Scenario: 792a7f1d A note that was never used is not scaled
  Given a note engram learn wrote 120 days ago, with no last_used in its sidecar
  When the user runs engram query with a phrase that matches it
  Then its score is its cosine

Scenario: 792a7f1e A late turn is lifted
  Given two chunks of one transcript with the same cosine to the phrase, one at its first turn and one at its last, both first indexed today
  When the user runs engram query with a phrase that matches them
  Then the last turn's score is 1.2 times the first turn's
```

## 792a7g Clusters

Requirement: [792a7g Clusters](#792a7g-clusters) `engram query` MUST cluster the matched notes and chunks together, once for
all the phrases and not once per phrase, into `clusters`. Each cluster MUST have an `id`, an empty
`phrase`, a `size`, a `silhouette` (the mean silhouette score of its members, 0 for the one-cluster
case below), and `members` listing each member's `path` and `score`, highest
score first, with exactly one member marked `is_representative`: the member whose vector is closest
to the cluster's centroid, ties going to the higher score and then to the lower path. The number of
clusters MUST be between 2 and 7 when a split scores a silhouette of at least 0.10, and otherwise
the whole matched set MUST be one cluster with silhouette 0. Nothing matched MUST give no clusters.
The same vault and phrases MUST give the same clusters.

Rationale: the clusters group what was found by topic so that the reader judges each topic once.
Clustering the union once keeps the number of clusters small, and the seed is derived from the
phrases, so the result does not vary between runs.

### 792a7g1 UAT Scenarios

```gherkin
Scenario: 792a7g1a Matches form clusters with a representative
  Given a vault whose matches fall into two topics
  When the user runs engram query with phrases for both
  Then clusters holds two clusters, each listing its members by score and marking exactly one representative

Scenario: 792a7g1b Matches with no structure form one cluster
  Given a vault whose matches are too alike to split
  When the user runs engram query with a phrase that matches them
  Then clusters holds one cluster holding every match, with silhouette 0

Scenario: 792a7g1c Nothing matched means no clusters
  Given a vault with no note above the floor
  When the user runs engram query with a phrase
  Then clusters is empty

Scenario: 792a7g1d The same query gives the same clusters
  Given a vault and a set of phrases
  When the user runs engram query with them twice
  Then both payloads hold the same clusters
```

## 792a7h Candidate notes

Requirement: [792a7h Candidate notes](#792a7h-candidate-notes) `engram query` MUST give each cluster that has note members a `candidate_l2s`
list of its own notes, at most five, ranked by the higher of each note's situation and body cosine
to the cluster's centroid, highest first with ties by path, each with its `path`, `cosine` and
`content`. A cluster with no note members MUST have no `candidate_l2s`. The list MUST hold nothing
but the cluster's own notes: not notes sampled for other topics ([792a7i Half of the notes are explored](#792a7i-half-of-the-notes-are-explored)) and not superseding notes
([792a7n Superseding notes ride along](#792a7n-superseding-notes-ride-along)), which travel in `items`.

Rationale: the cosine only nominates; whether a candidate covers the situation is the reader's
judgement. The centroid is the unweighted mean of the cluster's vectors: a centroid weighted by each
member's recency was measured on the production vault, on the 16 same-cluster pairs where one note
supersedes another and the cluster has more than five notes, and it changed nothing, with 8 of 16
(50.0%) superseded notes over-surfaced either way and a mean top-5 relevance of 0.6480 for both
(`dev/eval/LEDGER.md#652-recency-weighted-centroid-null`), so it did not ship.
Recency weighting was eval-gated because it is a third recency lever on the same axis and over-rotating toward
recency is a real risk; a null result closes the question and ships no production change.

### 792a7h1 UAT Scenarios

```gherkin
Scenario: 792a7h1a Candidates are the cluster's nearest notes
  Given a cluster of seven notes and three chunks
  When the user runs engram query with a phrase that matches them
  Then candidate_l2s lists five of the notes, ordered by cosine to the centroid, and no chunk

Scenario: 792a7h1b A cluster of chunks has no candidates
  Given a cluster whose members are all chunks
  When the user runs engram query with a phrase that matches them
  Then that cluster has no candidate_l2s

Scenario: 792a7h1c A small cluster lists all its notes
  Given a cluster of three notes
  When the user runs engram query with a phrase that matches them
  Then candidate_l2s lists all three
```

## 792a7i Half of the notes are explored

Requirement: [792a7i Half of the notes are explored](#792a7i-half-of-the-notes-are-explored) The notes `engram query` delivers MUST be an exploit half, the notes matched
above ([792a7d Matched notes and chunks](#792a7d-matched-notes-and-chunks)), and an explore half sampled from the vocabulary's terms ([controlled-vocabulary](./controlled-vocabulary.md)) by their
closeness to the query. The explore half's budget MUST be the number of distinct notes in the
exploit half, so the halves are equal in notes, and chunks MUST count towards neither. When the
exploit half holds no note the query MUST skip explore sampling. When the explore half cannot be
filled, because the vault has too few distinct notes or `vocab.centroids.json` is absent or
unreadable, it MUST be smaller, and the budget's `explore_allocated` MUST show the shortfall, as `{}`
when it is empty.

Rationale: matches alone only find what resembles the query, so global reach comes from sampling
near concepts. Tag nomination did this by joining any note that shared a term with the top three
notes of a cluster, whatever its cosine, which tied candidates to an accident of rank, gave every
term-sharing note equal footing, and needed a per-cluster cap; it was replaced on 2026-07-29.
The missing-centroids case fails loudly, as an empty allocation, rather than silently.

### 792a7i1 UAT Scenarios

```gherkin
Scenario: 792a7i1a The halves are equal on a healthy vault
  Given a bootstrapped vault with enough notes
  When the user runs engram query with a phrase that matches two notes
  Then the payload holds those two notes and two explore notes

Scenario: 792a7i1b Missing centroids leave only the matches
  Given a vault whose vocab.centroids.json is absent or unreadable
  When the user runs engram query with a phrase that matches notes
  Then no item has provenance explore and the budget's explore_allocated is {}

Scenario: 792a7i1c No matched note means no explore half
  Given a vault whose only matches are chunks
  When the user runs engram query with a phrase
  Then no item has provenance explore and explore_allocated is {}

Scenario: 792a7i1d A small vault shortens the explore half
  Given a vault whose notes are all in the exploit half
  When the user runs engram query with a phrase
  Then the explore half is smaller than the exploit half and explore_allocated sums to the number delivered
```

## 792a7j Explore allocation follows the query

Requirement: [792a7j Explore allocation follows the query](#792a7j-explore-allocation-follows-the-query) `engram query` MUST divide the explore budget among the vocabulary's terms in
proportion to the softmax of the cosine between the mean of the phrases' vectors and the term's
centroid, divided by a temperature of 0.106, so that every term with a centroid takes part, nearer
terms are allocated at least as much as farther ones, and the allocations total the budget. A term
that has at least one member in the exploit half MUST have 0.05 added to that cosine first, once, no
matter how many members it has.

Rationale: a hard radius is brittle in cosine space, whereas a softmax needs one temperature and
degrades continuously, so far terms approach an allocation of zero without a cutoff. The temperature
was calibrated on ten queries against the production vault (best centroid cosine 0.550 and median
0.307, giving (0.550 - 0.307) / ln 10 = 0.106; `dev/eval/LEDGER.md#recall-explore-tau-calibration`).
The bonus keeps the one thing tag nomination did well, reach to a term whose notes already matched,
and is a constant so it cannot compound.

### 792a7j1 UAT Scenarios

```gherkin
Scenario: 792a7j1a Nearer terms sample more
  Given two terms, one with a centroid nearer the query
  When the explore budget is divided
  Then the nearer term's allocation is at least the other's, and together they equal the budget

Scenario: 792a7j1b Evidence lifts a distant term
  Given a distant term with one member in the exploit half and an equally distant term with none
  When the explore budget is divided
  Then the term with a member is allocated more

Scenario: 792a7j1c The bonus does not stack
  Given two terms of equal distance, one with one member in the exploit half and one with three
  When the explore budget is divided
  Then the two are allocated the same
```

## 792a7k Explore picks the core of a term

Requirement: [792a7k Explore picks the core of a term](#792a7k-explore-picks-the-core-of-a-term) `engram query` MUST pick a term's explore notes from the notes tagged
`vocab/<term>`, leaving out definition notes ([792a6a Definition notes](./controlled-vocabulary.md#792a6a-definition-notes)), question notes and notes without a vector,
in descending order of the cosine between each note's body vector and the term's centroid, ties by
path. A note with several terms MUST be a candidate under each.

Rationale: members near the query are already in the exploit half, so the explore half shows each
concept by its core, and the core excludes the fringe of notes that only just cleared the
assignment floor of 0.35.

### 792a7k1 UAT Scenarios

```gherkin
Scenario: 792a7k1a The core outranks the fringe
  Given a term with five member notes and an allocation of two
  When the explore notes are picked
  Then they are the two members closest to the term's centroid

Scenario: 792a7k1b Definition notes are never picked
  Given a term whose definition note is the closest note to its centroid
  When the explore notes are picked
  Then the definition note is not among them
```

## 792a7l Explore picks are not repeated

Requirement: [792a7l Explore picks are not repeated](#792a7l-explore-picks-are-not-repeated) `engram query` MUST drop an explore pick that is already in the exploit half or
that another term already picked, and MUST refill each freed slot from the terms in order of
allocation, until the explore budget is met or no candidates remain.

### 792a7l1 UAT Scenarios

```gherkin
Scenario: 792a7l1a A duplicate is replaced
  Given an explore pick that is already in the exploit half, and other terms with unpicked members
  When the explore notes are picked
  Then the duplicate is dropped and its slot is filled from the next term in allocation order

Scenario: 792a7l1b A note under two terms is picked once
  Given a note that is the nearest member of two allocated terms
  When the explore notes are picked
  Then it appears once

Scenario: 792a7l1c Exhausted candidates leave the budget unmet
  Given fewer eligible notes than the explore budget
  When the explore notes are picked
  Then every eligible note is picked and the half is smaller than its budget
```

## 792a7m Explore picks are reported

Requirement: [792a7m Explore picks are reported](#792a7m-explore-picks-are-reported) `engram query` MUST give every explore note the provenance `explore` and a
`source_term` naming its term, the cosine to that term's centroid as its `score`, and its full
content, and MUST place the explore notes in `items` after the matched notes and before the matched
chunks. They MUST NOT be members of any cluster or appear in any `candidate_l2s`. The budget's
`explore_allocated` MUST map each contributing term to the number of notes delivered for it, after
duplicates are dropped, and MUST be present in every payload.

### 792a7m1 UAT Scenarios

```gherkin
Scenario: 792a7m1a An explore note names its term
  Given a bootstrapped vault with a term go-tooling
  When the user runs engram query with a phrase about cooking
  Then an item has provenance explore, source_term go-tooling and a score equal to its cosine to that term's centroid

Scenario: 792a7m1b Explore notes are outside the clusters
  Given a payload with an explore note
  When its clusters are read
  Then the explore note is not a member of any cluster and not in any candidate_l2s

Scenario: 792a7m1c The budget counts what was delivered
  Given a payload whose explore half holds two notes of one term
  When its budget is read
  Then explore_allocated maps that term to 2
```

## 792a7n Superseding notes ride along

Requirement: [792a7n Superseding notes ride along](#792a7n-superseding-notes-ride-along) When a delivered matched note has been superseded, because a newer note's
`supersedes:` list names it with `updates`, `narrows` or `refutes`, `engram query` MUST insert the
newer note in `items` directly after it, with the provenance `ride_along` and a score of 0, whether
or not the newer note matched. When several notes supersede one note it MUST insert them in order,
and it MUST NOT insert a note that is already among the results, or one that is not in the vault
with a usable sidecar. Only matched notes MUST be examined, not chunks, recent chunks or notes
already inserted.

Rationale: a note that corrects another should reach the reader with it. The provenance is distinct
so that an analysis of rank shifts can tell an insertion from a match.

### 792a7n1 UAT Scenarios

```gherkin
Scenario: 792a7n1a The newer note follows the older one
  Given a note P matched by the phrase, and a note N that supersedes P but does not match
  When the user runs engram query with that phrase
  Then N is the item directly after P, with provenance ride_along and score 0

Scenario: 792a7n1b A note already present is not inserted again
  Given both P and N matched
  When the user runs engram query with a phrase that matches them
  Then N appears once

Scenario: 792a7n1c Several superseders are all inserted
  Given two notes N1 and N2 that both supersede P
  When the user runs engram query with a phrase that matches P
  Then N1 and N2 follow P in order

Scenario: 792a7n1d A missing superseder is skipped
  Given a note P whose superseder has no usable sidecar
  When the user runs engram query with a phrase that matches P
  Then no item is inserted and the query succeeds
```

## 792a7o The newest chunks

Requirement: [792a7o The newest chunks](#792a7o-the-newest-chunks) `engram query` MUST append to `items`, after the matched items, the newest
chunks of the chunk index by the time they were first indexed ([792a7p When a chunk was first indexed](#792a7p-when-a-chunk-was-first-indexed)), newest first and with ties
going to the later turn, skipping chunks that are already matched, up to the `--recent-fill` count.
That count MUST be 25 by default, `ENGRAM_RECENT_FILL` MUST set it from the environment, a positive
`--recent-fill` MUST replace it, and a negative one MUST turn the channel off. Each chunk MUST carry
the provenance `recent` and a score of 0, MUST NOT be a member of any cluster, and MUST NOT count
towards `--limit` ([792a7q Limit](#792a7q-limit)). Fewer chunks than the count MUST give all of them. A chunk with no
first-indexed time MUST sort after every chunk that has one.

Rationale: the agent that has lost its context re-encounters its own recent narration and recovers
what it was doing from recency, with no separate provenance mechanism. The channel is
non-load-bearing for recall, so a small count shrinks the payload: a count of 25 cut a real-vault
payload by 28% (230 to 165 KB, `dev/eval/LEDGER.md#payload-cut-recent-fill`). A zero value stands for
the default, because an unset flag and an unset environment variable both read as zero; to turn the
channel off the value must be negative.

### 792a7o1 UAT Scenarios

```gherkin
Scenario: 792a7o1a Recent chunks follow the matches
  Given a chunk index of three chunks, one of which matches the phrase
  When the user runs engram query with that phrase
  Then items ends with the two other chunks, newest first, each with provenance recent and score 0, and they are in no cluster

Scenario: 792a7o1b A matched chunk is not repeated
  Given a chunk that is both matched and among the newest
  When the user runs engram query with a phrase that matches it
  Then it appears once, with provenance direct

Scenario: 792a7o1c The count can be set
  Given a chunk index of three chunks and none matched
  When the user runs engram query --recent-fill 1, or sets ENGRAM_RECENT_FILL=1
  Then exactly one recent chunk is returned, the newest

Scenario: 792a7o1d A negative count turns the channel off
  Given a chunk index of three chunks
  When the user runs engram query --recent-fill=-1
  Then no item has provenance recent

Scenario: 792a7o1e Zero means the default
  Given a chunk index of three chunks
  When the user runs engram query --recent-fill 0
  Then all three chunks are returned as recent, as with no flag

Scenario: 792a7o1f A chunk without a time sorts last
  Given a chunk index holding a chunk with no first-indexed time and a chunk with one
  When the user runs engram query with a count of 1
  Then the chunk with a time is the one returned
```

## 792a7p When a chunk was first indexed

Requirement: [792a7p When a chunk was first indexed](#792a7p-when-a-chunk-was-first-indexed) Every chunk `engram ingest` indexes MUST record the time it was first indexed:
for a Markdown source the time of the ingest, and for a transcript the timestamp of the last entry
the reader returned. Ingesting a source again MUST keep the recorded time of every chunk it already
holds. A chunk in an index written before times were recorded MUST be given its source's recorded
modification time the next time the source is indexed again, or no time when the manifest has none.

Rationale: recency used to be approximated from the source file's modification time, so one edit
made every chunk in a source look new. A per-chunk time orders chunks across and within sessions;
all chunks of one transcript share the session's last timestamp, which is a sufficient
approximation because each session is its own source.

### 792a7p1 UAT Scenarios

```gherkin
Scenario: 792a7p1a A Markdown chunk takes the ingest time
  Given a Markdown document ingested at 2026-06-15 12:00
  When its index file is read
  Then each record's ingested_at is 2026-06-15 12:00

Scenario: 792a7p1b A transcript chunk takes the session's last timestamp
  Given a transcript whose last entry is stamped 2026-05-10 14:00, ingested on 2026-06-01
  When its index file is read
  Then each record's ingested_at is 2026-05-10 14:00

Scenario: 792a7p1c Ingesting again keeps the first time
  Given an ingested transcript whose chunk is unchanged on a second ingest
  When the second ingest runs
  Then the chunk's ingested_at is still the first one

Scenario: 792a7p1d An old chunk gets its source's modification time
  Given an index record with no ingested_at whose source the manifest records as modified 2026-01-20
  When the source is ingested again
  Then the record's ingested_at is 2026-01-20
```

## 792a7q Limit

Requirement: [792a7q Limit](#792a7q-limit) `engram query` MUST keep at most `--limit` of the items that are neither trigger
hits ([792a7za A trigger hit comes first](#792a7za-a-trigger-hit-comes-first)) nor recent chunks ([792a7o The newest chunks](#792a7o-the-newest-chunks)), the highest-ranked first, applied after the recency
channel is assembled and before chunk content is reduced ([792a7r Chunk content budget](#792a7r-chunk-content-budget)). The limit MUST be 20 when the
flag is absent, MUST come from the flag alone, and MUST be reported as the budget's `limit`. Trigger
hits MUST NOT be counted against it, so `items` can hold more than `--limit` items of the first
kind by the number of hits, and recent chunks MUST NOT be counted against it, so a full set of
matches never displaces them. It MUST apply in the same way to a query merged with a parent vault.

Rationale: before this limit `--limit` only reported a number, and the item count was whatever
clustering and nomination produced, with no ceiling. Counting the recent chunks under the same limit
would displace the whole channel whenever the matches alone reached it, which in a vault of any size
is routine, and so defeat `--recent-fill`.

### 792a7q1 UAT Scenarios

```gherkin
Scenario: 792a7q1a The default caps a large result
  Given a vault whose matches exceed 20 items
  When the user runs engram query with no --limit
  Then items holds exactly 20 of them, the highest ranked, and the budget's limit is 20

Scenario: 792a7q1b An explicit limit caps the result
  Given a vault whose matches exceed 5 items
  When the user runs engram query --limit 5
  Then items holds at most 5 of them

Scenario: 792a7q1c A small result is returned whole
  Given a vault whose matches total fewer than the limit
  When the user runs engram query
  Then every match is returned

Scenario: 792a7q1d Trigger hits do not use the limit
  Given a message that hits one runbook, and several matched notes
  When the user runs engram query --text "<message>" --limit 1 with a phrase
  Then items lists the hit first, then one matched item, and the hit is not counted against the limit

Scenario: 792a7q1e The recent chunks survive a full set of matches
  Given matches that alone reach the limit, and a chunk index with recent chunks
  When the user runs engram query --limit 2
  Then items holds two matched items followed by the recent chunks
```

## 792a7r Chunk content budget

Requirement: [792a7r Chunk content budget](#792a7r-chunk-content-budget) `engram query` MUST give the first `--content-budget` chunk items of `items`, in
the order printed, their full content, and MUST reduce the content of every later chunk item to a
snippet: its whitespace collapsed to single spaces, trimmed, and cut to at most 160 characters,
the last being `…` when it was cut. The budget MUST be 15 by default, `ENGRAM_CONTENT_BUDGET` MUST set
it from the environment, a negative value MUST mean no limit, and zero MUST stand for the default. Note items MUST never be
reduced.
The budget MUST report the budget used as `content_budget`, the number of items still carrying full
content as `items_with_full_content`, and the number reduced as `chunks_snippeted`.

Rationale: the budget sits at the knee of the cost and quality curve with headroom for long-chunk
recalls that the eval harnesses do not probe (2026-06-24).

### 792a7r1 UAT Scenarios

```gherkin
Scenario: 792a7r1a Later chunks become snippets
  Given a payload with three chunk items
  When the user runs engram query with a content budget of 1
  Then the first chunk keeps its content, the other two carry snippets, and chunks_snippeted is 2

Scenario: 792a7r1b A long chunk is cut with an ellipsis
  Given a payload whose second chunk item is 300 characters long
  When the user runs engram query with a content budget of 1
  Then the second chunk's content is 160 characters ending in …

Scenario: 792a7r1c Notes are never reduced
  Given a payload with a note and chunks beyond the budget
  When the user runs engram query with a content budget of 1
  Then the note keeps its full content

Scenario: 792a7r1d A negative budget means none
  Given a payload with 20 chunk items
  When the user runs engram query --content-budget=-1
  Then every chunk keeps its full content
```

## 792a7s Lazy chunks

Requirement: [792a7s Lazy chunks](#792a7s-lazy-chunks) When `--lazy-chunks` is set, or `ENGRAM_LAZY_CHUNKS` is true, `engram query`
MUST print each chunk item with its `path` and `score` but no content, MUST leave note items
with their full content, and MUST report `lazy_chunks: true` in the budget. A payload without the
flag MUST NOT contain `lazy_chunks`.

Rationale: a recall pass should carry less to read without losing reach, so the evidence is fetched
on demand with `engram show-chunk` ([792a7t Reading one chunk](#792a7t-reading-one-chunk)) and not delivered up front. On the real vault the
payload shrank 33.7% (146 to 97 KB), and across 13 realistic recalls that were not told to fetch,
none fetched a chunk, while a chunk that was the only source of a needed fact was fetched when it
mattered (`dev/eval/LEDGER.md#payload-cut-lazy-chunks`).

### 792a7s1 UAT Scenarios

```gherkin
Scenario: 792a7s1a Chunks carry no content
  Given a vault with a matching chunk
  When the user runs engram query --lazy-chunks
  Then the chunk item has a path and a score and no content, and the budget says lazy_chunks: true

Scenario: 792a7s1b Notes keep their content
  Given a vault with a matching note
  When the user runs engram query --lazy-chunks
  Then the note item keeps its full content

Scenario: 792a7s1c The environment can turn it on
  Given ENGRAM_LAZY_CHUNKS=true
  When the user runs engram query with a phrase that matches a chunk
  Then the chunk item has no content

Scenario: 792a7s1d Without the flag the field is absent
  Given a vault with a matching chunk
  When the user runs engram query with no --lazy-chunks
  Then the chunk item has its content and the budget has no lazy_chunks
```

## 792a7t Reading one chunk

Requirement: [792a7t Reading one chunk](#792a7t-reading-one-chunk) `engram show-chunk <source>#<anchor>` MUST print the text of the chunk with
that chunk ID, followed by a newline when the text lacks one, and MUST NOT write anything. It MUST
fail with `chunk not found: <id>` when no chunk has that ID, and with `show-chunk: empty chunk ref`
when the ID is empty or blank.

### 792a7t1 UAT Scenarios

```gherkin
Scenario: 792a7t1a A chunk's text is printed
  Given an ingested document with a "Baking" section
  When the user runs engram show-chunk "<document>#Baking"
  Then stdout is that section's text

Scenario: 792a7t1b An unknown chunk is an error
  Given a chunk index that holds no chunk "nosuch#x"
  When the user runs engram show-chunk "nosuch#x"
  Then it fails with "chunk not found: nosuch#x" and exits non-zero

Scenario: 792a7t1c An empty ID is an error
  Given any chunk index
  When the user runs engram show-chunk with a blank ID
  Then it fails with "show-chunk: empty chunk ref"
```

## 792a7u Project filter

Requirement: [792a7u Project filter](#792a7u-project-filter) `engram query --project <slug>` MUST restrict the result to notes of that project: it MUST drop every note
item, whether matched, superseding, explore or ride-along, whose frontmatter `project:` line is not the slug or
that has no readable frontmatter, and MUST NOT take a mention of the slug in a note's body as a match. A
dropped note MUST appear nowhere in the output, clusters included. Chunks, which carry no project, are not filtered,
and without `--project` it MUST drop nothing.

Rationale: the flag restricts items to notes with a matching `project:` field. The filter was written for the matched notes before the explore half and the superseding notes existed and was never extended to them; that is a bug (#804), and the requirement states the intent.

### 792a7u1 UAT Scenarios

```gherkin
Scenario: 792a7u1a Notes of other projects are dropped
  Given matched notes with project: alpha and project: beta
  When the user runs engram query --project alpha
  Then only the alpha note remains among the matched notes

Scenario: 792a7u1b A body mention does not match
  Given a matched note whose body says "project: alpha" and whose frontmatter has no project
  When the user runs engram query --project alpha
  Then the note is dropped

Scenario: 792a7u1c No filter drops nothing
  Given matched notes of several projects
  When the user runs engram query with no --project
  Then all of them are returned

Scenario: 792a7u1d Explore and ride-along notes of other projects are dropped
  Given an explore-half note and a ride-along note with project: beta, and matched notes with project: alpha
  When the user runs engram query --project alpha
  Then the output holds no note whose project is not alpha, in the items or in the clusters
```

## 792a7v Runbooks rank like other notes

Requirement: [792a7v Runbooks rank like other notes](#792a7v-runbooks-rank-like-other-notes) A `runbook` note MUST be a candidate for `engram query` on the same terms as a
`fact` or `feedback` note, with the same floor, phrase limit and deduplication, and MUST appear in
`items` when it ranks above the floor. It MUST rank by the similarity of its situation alone, with
no query flag, pre-filter or boost for its kind; a trigger hit ([792a7za A trigger hit comes first](#792a7za-a-trigger-hit-comes-first)) MUST be the only exception.
Its presence in `items` MUST be what counts as surfacing: a runbook missing from its cluster's
`candidate_l2s` has still surfaced.

Rationale: a `task_type` classification field with a pre-filter was considered and rejected: the
benchmark evidence for it measured a whole bundled system, not type classification on its own, which
was too little to justify a new ranking mechanism and its risk to existing retrieval. Only
`qa-question` notes are withheld from retrieval, so a runbook needs no exclusion. A literal cue the author declares is the one exception, because the agent
paraphrases the user's request into its phrases and the paraphrase drops the cue: on a real `/please`
ask "please" and "end-to-end" vanished and the deliverable was named instead, and adding the user's
words as a third phrase changed the runbook's rank or score in none of 95 scratch-vault cells.

### 792a7v1 UAT Scenarios

```gherkin
Scenario: 792a7v1a A runbook is returned like a fact
  Given a runbook whose situation is close to the phrase
  When the user runs engram query with that phrase
  Then it appears in items with kind runbook

Scenario: 792a7v1b Surfacing does not need a candidate slot
  Given a runbook in items whose cluster's candidate_l2s does not list it
  When the payload is read
  Then the runbook has surfaced

Scenario: 792a7v1c A runbook has no kind boost
  Given a runbook and a feedback note whose situations are equally close to the phrase, and no trigger hit
  When the user runs engram query with that phrase
  Then their order follows their scores alone
```

## 792a7w Red flags in the query preview

Requirement: [792a7w Red flags in the query preview](#792a7w-red-flags-in-the-query-preview) When a runbook is a direct match, a candidate note or a trigger hit, `engram query`
MUST render its `red_flags` in its content. When the list takes more than 1200 bytes of rendered
YAML, `engram query` MUST keep the newest entries, those listed last, that fit and MUST replace the
earlier ones with one marker entry
`[<n> EARLIER RED_FLAGS OMITTED — run engram show <basename> for all <total>]`, giving the number
omitted, the list's total and the note's own basename. A list
within 1200 bytes MUST be rendered unchanged, and a note without red flags MUST be unchanged.

Rationale: `engram learn runbook --red-flag` and `engram amend --red-flag` append, so the newest entry
is last. In #763 a runbook's fifteenth and newest entry, added after fourteen, sat past a truncation
boundary of about 2 KB applied outside engram and never reached the agent. Dropping from the front
keeps the entry most likely to matter now, and the marker tells the reader that it has not seen the
whole list and how to get it.

### 792a7w1 UAT Scenarios

```gherkin
Scenario: 792a7w1a Red flags are in the payload
  Given a runbook with two short red flags
  When the user runs engram query with a phrase that matches it
  Then its item's content includes both entries

Scenario: 792a7w1b An oversized list keeps its newest entries
  Given a runbook with 30 red flags of about 100 characters each
  When the user runs engram query with a phrase that matches it
  Then its content holds a marker saying "20 EARLIER RED_FLAGS OMITTED" with the note's basename and "all 30", followed by the last ten entries

Scenario: 792a7w1c A short list is unchanged
  Given a runbook whose red flags take under 1200 bytes
  When the user runs engram query with a phrase that matches it
  Then its red_flags block is identical to the note file's

Scenario: 792a7w1d A trigger hit is truncated the same way
  Given the same oversized runbook and a message that hits one of its triggers
  When the user runs engram query --text "<message>"
  Then its content holds the same marker
```

## 792a7x Reading one note

Requirement: [792a7x Reading one note](#792a7x-reading-one-note) `engram show <ref>` MUST print the content of the note the ref names,
frontmatter and body exactly as the file holds them, with every `red_flags` entry and no omission marker, then a line
`# outbound links (fetch with: engram show <basename>)` followed by the target of each wikilink in
the note's text, each once in the order it first appears, with a trailing `.md` dropped (unless the link has a
`|display` part, which is kept as written, so `[[2.alpha.md|disp]]` is listed as `2.alpha.md|disp`), one per line, or `(none)`. A link inside a fenced code block MUST NOT be
listed. The ref
MUST be a full basename, a basename in `[[ ]]` with or without `|display`, a basename with a
trailing `.md`, or a bare Luhmann ID, and MUST fail with `show: note not found: "<ref>"` when no
note matches and with `show: empty note reference` when blank. It MUST NOT change any note. A note that
has been exchanged is preceded by one `# exchange_hash:` line, which belongs to curate.

Rationale: `engram show` is the full-fidelity source that the guidance shim names, so an agent can
restate every step and check every red flag when the query payload truncated them ([792a7w Red flags in the query preview](#792a7w-red-flags-in-the-query-preview)). It
therefore never applies the preview budget.
`show` never truncates red flags and `query`'s marker names the basename. A bounded `show` with a `--full`
flag was rejected because one skill's red flags alone render to about 4 KB, which recreates the truncation
loop. Truncating inside engram is the only reliable fix, because a harness that persists large output keeps
the full text.

### 792a7x1 UAT Scenarios

```gherkin
Scenario: 792a7x1a A note is printed with its links
  Given a note 1.hub that links [[2.alpha]] and [[3.beta]]
  When the user runs engram show 1.hub
  Then stdout is the note, then the outbound links line and 2.alpha and 3.beta

Scenario: 792a7x1b A note with no links says so
  Given a note that links to nothing
  When the user runs engram show <basename>
  Then stdout ends with the outbound links line and (none)

Scenario: 792a7x1c Every reference form resolves
  Given the note 1.2026-10-09.go-testing
  When the user runs engram show with 1.2026-10-09.go-testing, [[1.2026-10-09.go-testing]], 1.2026-10-09.go-testing.md or 1
  Then each prints the same note

Scenario: 792a7x1d Show never truncates red flags
  Given a runbook whose red flags take more than 1200 bytes
  When the user runs engram show for it
  Then stdout holds every entry in file order and no omission marker

Scenario: 792a7x1e A fenced link is not listed
  Given a note whose body has [[9.fenced]] inside a code fence and [[2.alpha]] outside it
  When the user runs engram show for it
  Then only 2.alpha is listed

Scenario: 792a7x1f An unknown note is an error
  Given a vault with no note nosuch
  When the user runs engram show nosuch
  Then it fails with show: note not found: "nosuch" and exits non-zero

Scenario: 792a7x1g A blank reference is an error
  Given any vault
  When the user runs engram show with a blank reference
  Then it fails with show: empty note reference

Scenario: 792a7x1h Each link is listed once
  Given a note whose body links [[2.alpha]], [[3.beta.md]], [[2.alpha]] and [[4.gamma|the gamma note]]
  When the user runs engram show for it
  Then the outbound links are 2.alpha, 3.beta and 4.gamma|the gamma note, in that order
```

## 792a7y Trigger text

Requirement: [792a7y Trigger text](#792a7y-trigger-text) `engram query` MUST accept the user's message verbatim as `--text`, and MUST
use it only to look for runbook triggers: it MUST NOT embed it or match it by similarity. With
`--text` and no `--phrase` it MUST answer a trigger lookup alone, embedding nothing, with `phrases`
empty, and no clusters. Without `--text` it MUST NOT look for triggers, and its payload MUST be the
same as it was before `--text` existed.

Rationale: matching the agent's paraphrase of a request against a runbook misses a literal cue like
`/please` too often for cue-triggered skills to rest on similarity alone, so the user's own words are
passed unchanged. The served form of `--text` is [792a20m A served query takes the message and can return dedupe keys](./vault-serving.md#792a20m-a-served-query-takes-the-message-and-can-return-dedupe-keys).

### 792a7y1 UAT Scenarios

```gherkin
Scenario: 792a7y1a Text alone is a trigger lookup
  Given a runbook with triggers ["/release"]
  When the user runs engram query --text "please /Release it now"
  Then items holds only that runbook, with provenances [trigger] and score 0, and phrases and clusters are empty

Scenario: 792a7y1b Without text nothing changes
  Given a vault whose runbooks carry triggers
  When the user runs engram query with phrases and no --text
  Then the payload is the same as before --text existed and no item has provenance trigger

Scenario: 792a7y1c Text is not matched by similarity
  Given a runbook whose situation is close to the text but whose triggers are not in it
  When the user runs engram query --text "<that text>"
  Then the runbook is not returned
```

## 792a7za A trigger hit comes first

Requirement: [792a7za A trigger hit comes first](#792a7za-a-trigger-hit-comes-first) When any entry of a runbook's `triggers` occurs in `--text` ([792a7zb Triggers match whole words](#792a7zb-triggers-match-whole-words)), the
runbook MUST be a trigger hit: it MUST appear in `items` before every other item, with `trigger`
among its provenances, exempt from the relevance floor, the matched-set cap, the matched-note floor
and `--limit`. A hit that also matched by similarity MUST keep its score and its other provenances,
and a hit that did not MUST have score 0. Only `runbook` notes with a usable sidecar MUST be candidates: a `triggers:`
list on any other kind MUST be ignored. With `--project`, a hit outside the project MUST NOT be surfaced.

Rationale: a literal cue the author declared is evidence that the runbook applies, whatever its
similarity to the agent's paraphrase. A hit is still only a candidate: the agent judges it against
the runbook's own applicability text.
Hits are items with a `trigger` provenance in `items[]` and not a separate key, because the shim's follow
frame keys on `items[]`. A probe that checks surfacing must read the `items[]` rank;
`clusters[].candidate_l2s` is only a capped top-5 for synthesis and is not a surfacing signal.

### 792a7za1 UAT Scenarios

```gherkin
Scenario: 792a7za1a A hit with no similarity match is first
  Given a runbook with triggers ["/please"] and no similarity match above the floor
  When the user runs engram query --text "/please fix the flaky login test" --phrase "fixing a flaky test"
  Then the runbook is items[0] with provenances [trigger] and kind runbook

Scenario: 792a7za1b A hit outranks a stronger match
  Given a feedback note that scores 0.9 and a runbook that scores 0.3 and is a trigger hit
  When the user runs engram query with the message and a phrase
  Then the runbook precedes the feedback note

Scenario: 792a7za1c Other kinds never hit
  Given a fact note whose frontmatter has a triggers list matching the text
  When the user runs engram query --text "<the text>"
  Then it has no trigger provenance and ranks by similarity only

Scenario: 792a7za1d A hit that matched keeps its score
  Given a runbook that both matches a phrase and is a trigger hit
  When the user runs engram query with the phrase and the message
  Then its provenances include direct and trigger and its score is its similarity score

Scenario: 792a7za1e A hit outside the project is not surfaced
  Given a runbook trigger hit whose project is beta
  When the user runs engram query --project alpha with the message
  Then the runbook is not returned
```

## 792a7zb Triggers match whole words

Requirement: [792a7zb Triggers match whole words](#792a7zb-triggers-match-whole-words) A trigger MUST occur in `--text` as a whole word, compared without regard to
case after each run of whitespace in both is collapsed to one space. An occurrence is a whole word
when, on each side where the trigger's own edge character is a letter or a digit, the character next
to the occurrence, if there is one, is not a letter or a digit, where letters and digits are those of
Unicode, so an underscore, a hyphen, a slash and any other punctuation are boundaries, as are the
start and end of the text. On a side where the trigger's edge character is not a letter or a digit
there MUST be no boundary requirement. Every occurrence MUST be tried, so a later one can match when
an earlier one does not.

Rationale: a plain substring let a bare-word trigger such as `curate` fire inside `accurate`. The
bare words `curate` and `please` make such false hits certain, so the boundary is enforced on the
sides that need it. A trigger written with punctuation at its edge, such as `/please`, already
carries its own delimiter, so `do/please` still hits while `/pleased` does not. The rule is two rune
lookups and a hand-written scan, with no pattern compiled from author text. Inflected forms no
longer hit, because the author lists the forms that should: `curate` does not match `curated`.

### 792a7zb1 UAT Scenarios

```gherkin
Scenario: 792a7zb1a Case and whitespace are ignored
  Given triggers ["take this end-to-end"]
  When --text is "PLEASE   take this end-to-end: rename X"
  Then the runbook is a trigger hit

Scenario: 792a7zb1b A bare word hits as a whole word
  Given triggers ["curate"]
  When --text is "please curate the offers", "Curate!" or "(curate)"
  Then the runbook is a trigger hit

Scenario: 792a7zb1c A bare word does not hit inside a longer word
  Given triggers ["curate"]
  When --text is "that is accurate", "inaccurate", "curated offers" or "curates"
  Then the runbook is not a trigger hit

Scenario: 792a7zb1d A slash form needs only a right boundary
  Given triggers ["/please"]
  When --text is "/please fix it" or "do /please", and when it is "/pleased"
  Then the first two hit and the last does not

Scenario: 792a7zb1e A slash before a bare word is a boundary
  Given triggers ["curate"]
  When --text is "/curate"
  Then the runbook is a trigger hit

Scenario: 792a7zb1f The edges of the text are boundaries
  Given triggers ["curate"]
  When --text is "curate"
  Then the runbook is a trigger hit

Scenario: 792a7zb1g A later occurrence can match
  Given triggers ["curate"]
  When --text is "accurate, curate now"
  Then the runbook is a trigger hit

Scenario: 792a7zb1h Unicode letters and digits are not boundaries
  Given triggers ["curate"]
  When --text is "écurate", "curateé" or "curate2 now"
  Then the runbook is not a trigger hit

Scenario: 792a7zb1i Hyphen and underscore are boundaries
  Given triggers ["curate"]
  When --text is "re-curate-now" or "do_curate_it"
  Then the runbook is a trigger hit
```

## 792a7zc The embedding model

Requirement: [792a7zc The embedding model](#792a7zc-the-embedding-model) `engram query` MUST put in its payload the `model_id` of the embedding model that
produced its results.

Rationale: a consumer comparing or fusing results from more than one vault can then tell whether
their models match before treating the scores as comparable.

### 792a7zc1 UAT Scenarios

```gherkin
Scenario: 792a7zc1a Every payload names its model
  Given any vault
  When the user runs engram query with a phrase
  Then the payload's model_id is minilm-l6-v2@384
```

## 792a7zd A refit is pending

Requirement: [792a7zd A refit is pending](#792a7zd-a-refit-is-pending) `engram query` MUST print `refit_pending: true` when the vault's
`vocab.centroids.json` records a pending vocabulary refit ([792a6s Refit trigger](./controlled-vocabulary.md#792a6s-refit-trigger)), and MUST NOT print the field
otherwise, including when the file is absent or unreadable.

### 792a7zd1 UAT Scenarios

```gherkin
Scenario: 792a7zd1a A pending refit is reported
  Given a vault whose vocab.centroids.json has refit_pending true
  When the user runs engram query with a phrase
  Then the payload has refit_pending: true

Scenario: 792a7zd1b Otherwise the field is absent
  Given a vault whose vocab.centroids.json has no pending refit, or no such file
  When the user runs engram query with a phrase
  Then the payload has no refit_pending
```

## 792a7ze Stage timings

Requirement: [792a7ze Stage timings](#792a7ze-stage-timings) When `--timings` is set, `engram query` MUST add a `timings` block to the
payload with `scan_ms`, `embed_ms`, `cluster_ms`, `nominate_ms` and `render_ms`, the wall-clock
milliseconds of the stages that read the vault and the chunk index, embed and match the phrases,
cluster, assemble the explore half, ride-alongs and recent chunks, and prepare the payload for
printing. Without the flag the payload MUST NOT contain `timings`, and the flag MUST NOT change
anything else in it.

Rationale: a measurement-only diagnostic (#691), so that query speed can be decomposed before it is
optimized. On a vault with 41,367 chunk-index files (364 MB) the scan dominated at about 67% of the
binary's wall time (4,787 ms median against embed 2,009, cluster 212, nominate 60 and render 0
ms; `dev/eval/LEDGER.md#query-inflight-split`). The stage key `nominate` remains from the era of tag
nomination and now measures explore sampling and ride-along assembly.

### 792a7ze1 UAT Scenarios

```gherkin
Scenario: 792a7ze1a Timings are reported on request
  Given a vault with notes
  When the user runs engram query --timings with a phrase
  Then the payload has a timings block with five keys: scan_ms, embed_ms, cluster_ms, nominate_ms and render_ms

Scenario: 792a7ze1b Timings are absent by default
  Given the same vault
  When the user runs engram query with a phrase and no --timings
  Then the payload has no timings block
```

## 792a7zf Searching chunks alone

Requirement: [792a7zf Searching chunks alone](#792a7zf-searching-chunks-alone) `engram query-chunks --phrase <p>` MUST score every chunk in the chunk index by
its best cosine over the phrases, without recency scaling, and print the best `--limit` (20 by
default) as `items` with `source`, `anchor`, `score` and `content`, highest score first, under
`version: 1` and `space: chunks`. When it returns at least six chunks it MUST also print `clusters`,
groups of the returned chunks found by the clustering of [792a7g Clusters](#792a7g-clusters), each with an `id`, a `size` and its
members' `source`, `anchor` and `score`. It MUST end with a `budget` of `total_chunks` and
`returned`. It MUST fail with `query chunks: at least one --phrase is required` without a phrase. An
empty or absent chunk index MUST give an empty payload without embedding the phrase.

### 792a7zf1 UAT Scenarios

```gherkin
Scenario: 792a7zf1a Chunks are ranked by cosine
  Given a chunk index holding a baking chunk and a networking chunk
  When the user runs engram query-chunks --phrase "sourdough"
  Then items lists the baking chunk first, each with source, anchor, score and content, and the budget says total_chunks: 2 and returned: 2

Scenario: 792a7zf1b The limit is honoured
  Given a chunk index of five chunks
  When the user runs engram query-chunks --phrase p --limit 2
  Then two items are returned and the budget says returned: 2

Scenario: 792a7zf1c Many chunks are clustered
  Given a chunk index of twelve chunks about two topics
  When the user runs engram query-chunks with a phrase
  Then the payload has clusters, each with an id, a size and members

Scenario: 792a7zf1d A phrase is required
  Given any chunk index
  When the user runs engram query-chunks with no phrase
  Then it fails with "query chunks: at least one --phrase is required"

Scenario: 792a7zf1e An empty index is an empty payload
  Given a chunk index directory that does not exist
  When the user runs engram query-chunks --phrase p
  Then the payload has no items, and the budget says total_chunks: 0
```

## 792a7zg CRLF notes

Requirement: [792a7zg CRLF notes](#792a7zg-crlf-notes) `engram query` MUST read a note whose lines end in `\r\n` as its LF form (each
`\r\n` read as `\n`), in its project filter, its recency date, the tags, supersessions and triggers it
reads from frontmatter, the note's kind label, and the preview of its red flags, and MUST NOT
write a note only to convert it.

### 792a7zg1 UAT Scenarios

```gherkin
Scenario: 792a7zg1a The project filter reads a CRLF note
  Given a note with CRLF line endings and project: alpha
  When the user runs engram query --project alpha with a phrase that matches it
  Then the note is returned

Scenario: 792a7zg1b A CRLF runbook's trigger matches
  Given a runbook whose lines end in \r\n and whose triggers include "curate"
  When the user runs engram query --text "curate"
  Then it is returned as a trigger hit with kind runbook

Scenario: 792a7zg1c A CRLF note is not rewritten
  Given a note with CRLF line endings
  When the user runs engram query with a phrase that matches it
  Then the note's file is byte-identical afterwards

Scenario: 792a7zg1d The creation date of a CRLF note is read
  Given a CRLF note whose created line is the bare date 120 days ago and whose sidecar has no last_used
  When the user runs engram query with a phrase that matches it
  Then its score is scaled by that age, as for the same note with LF line endings

Scenario: 792a7zg1e A CRLF runbook's red flags are previewed
  Given a runbook whose lines end in \r\n and whose red flags take more than 1200 bytes
  When the user runs engram query with a phrase that matches it
  Then its content holds the omission marker and the newest entries, as for the same note with LF line endings

Scenario: 792a7zg1f A CRLF note keeps its kind
  Given a fact note whose lines end in \r\n
  When the user runs engram query with a phrase that matches it
  Then its item has kind fact

Scenario: 792a7zg1g The lists in a CRLF note's frontmatter are read
  Given a CRLF note tagged vocab/cooking, and a CRLF note that supersedes another
  When the user runs engram query with phrases that match the term's members, and the superseded note
  Then the tagged note can be an explore note, and the superseding note rides along
```

## 792a7zh A query changes nothing

Requirement: [792a7zh A query changes nothing](#792a7zh-a-query-changes-nothing) `engram query` MUST NOT write any note, sidecar or chunk-index file, and MUST NOT
mark a note as used: it MUST NOT change a sidecar's `last_used`, and its payload MUST NOT carry an
`activated` field. A note is marked as used only by the agent, through `engram activate` or `engram amend --activate`.

Rationale: activation is agent-driven. A note that was returned but not used goes stale and fades in
rank, which is the mechanism by which unhelpful memory loses its place; if every returned note were
bumped, nothing would ever fade.

### 792a7zh1 UAT Scenarios

```gherkin
Scenario: 792a7zh1a A query leaves the vault as it was
  Given a vault with notes and sidecars
  When the user runs engram query with a phrase that matches them
  Then every note, sidecar and chunk-index file is byte-identical afterwards, and the payload has no activated field
```
