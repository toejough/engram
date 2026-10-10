# Controlled Vocabulary - Product Spec

The vault keeps a controlled vocabulary: a small set of terms, each defined by a definition note.
Every note that engram writes is tagged `vocab/<term>` with the terms closest to it, and
`engram vocab refit` re-derives the term set from the vault's own notes as it grows. `engram vocab`
also seeds, extends, reports on and repairs the vocabulary, and `engram update` flags vaults that
need a vocabulary migration. Which kinds of note are tagged (every kind but questions) is
note-writing's [792a3u Vocabulary tagging by kind](./note-writing.md#792a3u-vocabulary-tagging-by-kind). Using the vocabulary at query time, including the query payload's
`refit_pending` field, is the memory-query capability ([memory-query](./memory-query.md)); acting on a pending refit is the learn
skill's ([learn](./learn.md)). Tagging the notes that a pull-down or a served learn writes belongs to [parent-sync](./parent-sync.md) and
vault-serving ([792a20o The receiving vault tags a served offer](./vault-serving.md#792a20o-the-receiving-vault-tags-a-served-offer)).

## 792a6a Definition notes

Requirement: [792a6a Definition notes](#792a6a-definition-notes) Every term definition that `engram vocab bootstrap`, `engram vocab propose` or
`engram vocab refit` writes MUST be a top-level fact note with the slug `vocab-<term>-definition`,
`tags: [vocab, vocab/<term>]`, `tier: L2`, situation `recalling what the <term> vocab term covers, or
assigning vocab terms`, subject `the <term> vocab term`, predicate `covers`, the term's description as
its object, and source `vocab lifecycle (<command> v<version>)`. When the term comes with exemplars,
its body MUST end with an `Exemplars:` list holding each one. Every definition note MUST receive a
sidecar, or a warning naming it when embedding fails, and a definition note MUST NOT itself be
tagged with any term by assignment or refit.

Rationale: the definition's body is the term's embedding text, so exemplars from member notes pull
the term's vector toward its members; a description alone embeds far from them (recall at 5 of
45.5%, against 64.6% for the member centroid). The bare `vocab` tag marks a definition, so every
counting and tagging path can leave definitions out and a term's own description never skews its
centroid. The `vocab/<term>` self-tag was added so that a term's definition appears with its members
in Obsidian's tag pane and graph; it has no other effect.

### 792a6a1 UAT Scenarios

```gherkin
Scenario: 792a6a1a A definition note carries its self-tag
  Given a vault and a seed file naming the term cooking with the description "recipes and kitchens" and one exemplar
  When the user runs engram vocab bootstrap --seed with that file
  Then the vault holds a note <id>.<date>.vocab-cooking-definition.md with tags [vocab, vocab/cooking], object "recipes and kitchens", an Exemplars: list holding the exemplar, and a sidecar

Scenario: 792a6a1b A definition is never assigned a term
  Given a vault whose vocabulary holds the term cooking
  When the user runs engram vocab refit or engram vocab bootstrap
  Then the cooking definition note's tags are still exactly [vocab, vocab/cooking]
```

## 792a6b The family note

Requirement: [792a6b The family note](#792a6b-the-family-note) The vault's vocabulary MUST be recorded by one family note, slug
`vocab-definition`, tagged only `vocab`, whose frontmatter carries the vault-wide `vocab_version`
and whose text describes the tagging convention without naming any term. `engram vocab bootstrap`
and `engram update --regen-vocab` MUST create it with version `1.0` when it is absent and MUST leave
an existing one alone. `engram vocab propose` MUST raise the version's minor part by one and
`engram vocab refit` its major part by one, setting the minor part to `0`.

Rationale: the vocabulary is emergent: the terms are the definition notes that exist, found by their
tags. A note that listed the terms would be the stale `vocab.index.md` again, so the family note
holds only the convention and the version.

### 792a6b1 UAT Scenarios

```gherkin
Scenario: 792a6b1a Bootstrap creates the family note
  Given a vault with no vocabulary
  When the user runs engram vocab bootstrap with a seed file naming two terms
  Then the vault holds one note <id>.<date>.vocab-definition.md tagged only vocab, carrying vocab_version "1.0", and its text names neither term

Scenario: 792a6b1b The version follows propose and refit
  Given a vault whose family note carries vocab_version "2.0"
  When the user runs engram vocab propose and then an engram vocab refit that applies
  Then engram vocab stats reports version 2.1 after the propose and 3.0 after the refit
```

## 792a6c Categorical tags share one list

Requirement: [792a6c Categorical tags share one list](#792a6c-categorical-tags-share-one-list) A note's categorical classifications MUST all live in its `tags:` list:
vocabulary tagging, refit and `engram update --regen-vocab` MUST change only the `vocab/<term>`
entries, keeping every other tag and its order.

Rationale: route's dispatch records needed low-cardinality categories (work kind, tier, outcome) that
recall, `engram count` and Obsidian can all read, without a separate store. Frontmatter `tags:`, a
plain list written by `engram learn --tag <family>[/<value>]`, is the only categorical mechanism: no
attribute nodes, no categorical wikilinks, no tables. Three kinds of note ride on it: route's
evidence notes (one per dispatch, tagged `work-kind/<k>`, `tier/<t>` and `outcome/<o>`, recallable
like any fact), its aggregate notes (one per work kind, slug `route-evidence-<work-kind>`, untagged,
holding the tallies and a link to each evidence note), and family definition notes (a bare family
tag defines the family, a `family/value` tag is a member). Route reads the aggregates through plain
recall; `engram count --group-by tags --filter tags=...` recomputes the true tallies as an audit,
because tallies an LLM maintains will drift, and it is never on the read path. Many near-identical
evidence notes could outrank their aggregate on the read query; a scratch-vault gauge (20 evidence
notes and one aggregate) passed, so no remedy was built. If drowning is measured on a real vault,
the remedies on record are a "summarizes" ride-along that inserts the aggregate when its evidence
surfaces, or demoting evidence notes to the chunk ranking tier, chosen with the measured case in
hand, because a new edge type must first show retrieval value. The vocabulary moved onto the same
convention: definitions became bare-`vocab`-tagged fact notes and members `vocab/<term>` tags.

### 792a6c1 UAT Scenarios

```gherkin
Scenario: 792a6c1a A categorical tag survives tagging
  Given a vault whose vocabulary holds the term build-tooling
  When the user runs engram learn fact --tag work-kind/review with text close to build-tooling
  Then the note's tags are [work-kind/review, vocab/build-tooling], in that order

Scenario: 792a6c1b A categorical tag survives a refit
  Given a note tagged [work-kind/review, vocab/old-term]
  When the user runs an engram vocab refit that retires old-term
  Then the note still carries work-kind/review
```

## 792a6d Write-time vocabulary tagging

Requirement: [792a6d Write-time vocabulary tagging](#792a6d-write-time-vocabulary-tagging) After `engram learn`, `engram amend` or `engram resituate` writes a note, the
note's `vocab/<term>` tags MUST be set to the terms whose vector has a cosine similarity of at least
0.35 with the note's body vector, at most three, highest first with ties in term-name order,
replacing any earlier vocabulary tags. A note with no qualifying term MUST lose its vocabulary tags,
and its `tags:` field MUST be removed when nothing else is left in it. A term's vector MUST be the
centroid stored by the last bootstrap or refit when one exists for the vault's embedding model, and
its definition note's embedding otherwise.

Rationale: tagging is pure cosine, so it costs no LLM call on the write path. The 0.35 floor and the
cap of three were chosen by a sweep over 48 known retrieval misses: two passes at 0.35 with three
terms recovered 56.2% of them at a median candidate pool of 31, passing the gate of at least 54.2%
within a pool of 40. Tags replace rather than accumulate, so a revised note carries only the terms it
is close to now.

### 792a6d1 UAT Scenarios

```gherkin
Scenario: 792a6d1a A learned note is tagged with its closest terms
  Given a vault whose vocabulary holds the terms build-tooling and cooking
  When the user runs engram learn fact about running a linter before committing Go code
  Then the note carries vocab/build-tooling and not vocab/cooking

Scenario: 792a6d1b A note close to no term gets no tag
  Given a vault whose vocabulary holds only the term cooking
  When the user runs engram learn fact about configuring a TLS certificate
  Then the note has no vocab/ tag and no tags: field

Scenario: 792a6d1c At most three terms are kept
  Given a vault with four terms, each close to one note's text
  When the user writes that note with engram learn
  Then the note carries the three closest terms, closest first

Scenario: 792a6d1d An amend re-tags the note
  Given a note tagged vocab/cooking whose object is then rewritten to describe a linter
  When the user runs engram amend --object with the new text
  Then the note carries vocab/build-tooling and no longer vocab/cooking

Scenario: 792a6d1e A resituated note is re-tagged
  Given a fact note and a vault vocabulary
  When the user runs engram resituate on it
  Then its vocab/ tags are set from its rewritten text
```

## 792a6e Tagging is skipped without a vocabulary or an embedding

Requirement: [792a6e Tagging is skipped without a vocabulary or an embedding](#792a6e-tagging-is-skipped-without-a-vocabulary-or-an-embedding) A note write MUST succeed and write no vocabulary tag when the vault has no
definition note with a sidecar, or when the written note has no sidecar.

### 792a6e1 UAT Scenarios

```gherkin
Scenario: 792a6e1a A vault without a vocabulary writes plain notes
  Given a vault with no definition notes
  When the user runs engram learn fact
  Then the note is written with no vocab/ tag

Scenario: 792a6e1b A note whose embedding failed is not tagged
  Given a vault with a vocabulary, where embedding the new note fails
  When the user runs engram learn fact
  Then the note is written, stderr carries a warning naming it, and the note has no vocab/ tag
```

## 792a6f CRLF notes

Requirement: [792a6f CRLF notes](#792a6f-crlf-notes) Vocabulary tagging, term removal and refit MUST read a note with CRLF line
endings as its LF form. A tagging that changes a CRLF note's tags MUST write the note with LF
endings, and one that changes nothing MUST NOT write the note.

Rationale: a note's text must not change meaning with its line endings, and no reader rewrites a note
only to convert it: the conversion happens inside a write that is needed anyway.

### 792a6f1 UAT Scenarios

```gherkin
Scenario: 792a6f1a Vocab tagging reads a CRLF note
  Given a note whose lines end in \r\n
  When vocabulary tagging adds a new term to it
  Then the tag is written and the note is written as LF; a CRLF note whose tags do not change is not written
```

## 792a6g Bootstrapping a vocabulary

Requirement: [792a6g Bootstrapping a vocabulary](#792a6g-bootstrapping-a-vocabulary) `engram vocab bootstrap --seed <file>` MUST read a YAML list of entries with
`term`, `description` and optional `exemplars`, create the family note when absent ([792a6b The family note](#792a6b-the-family-note)), create a
definition note ([792a6a Definition notes](#792a6a-definition-notes)) for every term that has none, and leave terms that already have one
untouched. It MUST then tag every note that is neither a definition nor a question in two passes, first
against the definitions' embeddings and then against each term's centroid (the mean of the notes the
first pass gave it), MUST record those centroids and the vault's current note count and date as the
refit baseline, MUST clear any pending refit, and MUST print
`vocab bootstrap: <N> terms, <M> notes assigned`, where `<N>` is the number of seed entries and `<M>`
the number of term assignments the second pass made. `--floor` MAY replace the 0.35 similarity floor
of [792a6d Write-time vocabulary tagging](#792a6d-write-time-vocabulary-tagging) for this run.

Rationale: the second pass replaces each description embedding with the centroid of its members, so
terms are placed where their notes actually are; centroid assignment is what the 0.35, three-term
setting of [792a6d Write-time vocabulary tagging](#792a6d-write-time-vocabulary-tagging) was measured with.

### 792a6g1 UAT Scenarios

```gherkin
Scenario: 792a6g1a Bootstrap seeds terms and tags existing notes
  Given a vault holding notes about Go builds and about cooking, and a seed file naming build-tooling
  When the user runs engram vocab bootstrap --seed with that file
  Then it prints "vocab bootstrap: 1 terms, <M> notes assigned", the build notes carry vocab/build-tooling, and engram vocab stats reports verdict: OK

Scenario: 792a6g1b A second bootstrap creates nothing new
  Given a vault already bootstrapped with a seed file
  When the user runs engram vocab bootstrap with the same seed file again
  Then no new definition note or family note is created
```

## 792a6h Bootstrap failures

Requirement: [792a6h Bootstrap failures](#792a6h-bootstrap-failures) `engram vocab bootstrap` MUST be refused when `--seed` is absent, and MUST fail
with an error when the seed file cannot be read or is not a YAML list of entries, in each case
before writing anything.

### 792a6h1 UAT Scenarios

```gherkin
Scenario Outline: 792a6h1a A bad seed is refused
  Given a vault
  When the user runs engram vocab bootstrap <arguments>
  Then it fails with an error saying <problem>, and nothing is written

  Examples:
    | arguments                   | problem                              |
    | without --seed              | --seed is a missing required flag    |
    | --seed missing.yaml         | the seed file cannot be read         |
    | --seed with "term: [unclosed" | the seed YAML cannot be parsed     |
```

## 792a6i Proposing a term

Requirement: [792a6i Proposing a term](#792a6i-proposing-a-term) `engram vocab propose --term <term> --description <text>` MUST require both
flags, create the term's definition note ([792a6a Definition notes](#792a6a-definition-notes)), raise the vocabulary's minor version ([792a6b The family note](#792a6b-the-family-note)),
record the term as proposed ([792a6q Proposed terms survive derivation](#792a6q-proposed-terms-survive-derivation)), and print `vocab propose: created <term> (version → <v>)`. In a
vault without a family note it MUST still create the definition and warn that the family note was
not found.

Rationale: the terms the clustering cannot see, a concept scattered across the vault's geometry,
still need a way in. Proposing is that one judgment-based path; whether a term is worth adding
(no existing term covers it, and it would tag no more than a fifth of the vault) is decided by the
agent before it runs the command, which only writes.

### 792a6i1 UAT Scenarios

```gherkin
Scenario: 792a6i1a A proposed term is created
  Given a bootstrapped vault at vocab_version "1.0"
  When the user runs engram vocab propose --term gardening --description "growing plants"
  Then it prints "vocab propose: created gardening (version → 1.1)" and the vault holds a vocab-gardening-definition note

Scenario: 792a6i1b Propose in a vault without a family note
  Given a vault with no vocabulary
  When the user runs engram vocab propose --term test-term --description "a test"
  Then it prints "vocab propose: created test-term", and stderr warns that the family definition note (vocab-definition) was not found

Scenario: 792a6i1c Both flags are required
  Given a vault
  When the user runs engram vocab propose --term x without --description
  Then it fails saying --description is a missing required flag, and nothing is written
```

## 792a6j Deriving the vocabulary

Requirement: [792a6j Deriving the vocabulary](#792a6j-deriving-the-vocabulary) `engram vocab refit` MUST cluster the embeddings of every note that is not a
definition, choosing the number of clusters K between 2 and 40 that gives the highest silhouette
score, and keeping the previous derivation's K when its score is within 0.02 of the best. When no
clustering scores at least 0.02, or there are too few notes to cluster, it MUST print
`vocab refit: no structure found; vocabulary unchanged` and write nothing.

Rationale: the earlier refit edited the existing term list from an LLM-written plan, and could only
add terms: an untagged-rate trigger fired whenever new notes missed the 0.35 floor, and minting was
the only way to satisfy it, so one vault drifted to about 150 terms for 400 notes. Tagging is already
pure cosine geometry, so the vocabulary is now derived from that geometry directly, and the number of
terms is the number the vault's clusters support. The 0.02 floor only rejects input with no
structure: whole-vault MiniLM embeddings score between 0.05 and 0.15 at every K (the production vault
of 597 notes peaked at 0.0987 at K=9), so recall's 0.10 floor would have made every refit a no-op.
Keeping the previous K within 0.02 stops term names churning on near ties.

### 792a6j1 UAT Scenarios

```gherkin
Scenario: 792a6j1a The term count follows the vault's clusters
  Given a vault of notes in two clear topics
  When the user runs engram vocab refit --dry-run
  Then it prints "vocab refit (dry-run): K=2 silhouette=<s>"

Scenario: 792a6j1b A vault without structure keeps its vocabulary
  Given an empty vault, or one whose notes are too few to cluster
  When the user runs engram vocab refit
  Then it prints "vocab refit: no structure found; vocabulary unchanged" and no file changes
```

## 792a6k Term names stay stable

Requirement: [792a6k Term names stay stable](#792a6k-term-names-stay-stable) A derived cluster MUST keep the name and definition note of the existing term
whose vector has a cosine similarity of at least 0.80 with the cluster's centroid, pairing clusters
and terms best match first, each at most once. A cluster that matches no term is new and needs a
name; an existing term that no cluster matches is retired ([792a6p Retiring a term](#792a6p-retiring-a-term)) unless it was proposed ([792a6q Proposed terms survive derivation](#792a6q-proposed-terms-survive-derivation)).

Rationale: names users and notes already rely on survive re-derivation when the concept is still
there. Greedy best-first matching is auditable and stable at a few tens of terms.

### 792a6k1 UAT Scenarios

```gherkin
Scenario: 792a6k1a A cluster keeps its term's name
  Given a vault whose term build-tooling sits at the centre of one of its clusters
  When the user runs engram vocab refit --dry-run
  Then it prints "matched: build-tooling (cluster <n>, similarity <s>)" with s at least 0.800

Scenario: 792a6k1b Unmatched clusters and terms are reported
  Given a vault with an unmatched cluster and an unmatched derived term astronomy
  When the user runs engram vocab refit --dry-run
  Then it prints "new: cluster <n> (<m> members, needs naming)" and "retired: astronomy"
```

## 792a6l Dry run

Requirement: [792a6l Dry run](#792a6l-dry-run) `engram vocab refit --dry-run` MUST print `vocab refit (dry-run): K=<k>
silhouette=<s>`, one `matched:` line per kept term, one `new:` line per new cluster and one
`retired:` line per retired term, and MUST write nothing.

Rationale: the first derivation on a vault can retire most of its terms (K=9 against 29 existing
terms on the production vault), so reviewing the diff before applying it matters.

### 792a6l1 UAT Scenarios

```gherkin
Scenario: 792a6l1a Dry run reports the derivation diff without writing
  Given a vault whose derivation matches one term, finds one new cluster and retires one term
  When the user runs engram vocab refit --dry-run
  Then the K and silhouette line, the matched, new and retired lines are printed, and no vault file or centroids file is modified
```

## 792a6m Naming requests

Requirement: [792a6m Naming requests](#792a6m-naming-requests) When a refit finds new clusters and no `--names` file is given, `engram vocab
refit` MUST print a JSON document with a `naming_requests` list, one entry per new cluster holding its
`cluster` index, `member_count` and up to three `exemplars` (the members nearest its centroid, each
with its `note` basename, a `title` from its slug and a `snippet` of at most 200 bytes of its body),
an `instruction` for answering, and a `fingerprint` of the notes it clustered, and MUST write nothing.

Rationale: the agent's LLM only names clusters; it never decides the term set. Nearest members make
better naming evidence than random ones.

### 792a6m1 UAT Scenarios

```gherkin
Scenario: 792a6m1a New clusters are emitted as naming requests
  Given a vault whose derivation finds one cluster that matches no term
  When the user runs engram vocab refit
  Then it prints a naming_requests document for that cluster with its member count, three exemplars and a fingerprint, and no vault file changes
```

## 792a6n Naming answers

Requirement: [792a6n Naming answers](#792a6n-naming-answers) `engram vocab refit --names <file>` MUST apply the derivation only when the file
is JSON `{"names":[{"cluster":N,"term":"...","description":"..."}],"fingerprint":"..."}` whose
fingerprint equals the current vault's, whose entries each name a new cluster exactly once with a
non-empty term and description, and which names every new cluster. Otherwise it MUST fail, writing
nothing, with an error saying which check failed: the file cannot be read or parsed, the vault changed
since the requests were emitted (naming both fingerprints), a cluster is unknown, named twice or
unnamed, or an entry is empty.

Rationale: a cluster index means something only for the vault state that produced it; if notes
changed between the two runs, a name could land on the wrong cluster, so the stale answer is rejected
and the requests must be regenerated.

### 792a6n1 UAT Scenarios

```gherkin
Scenario Outline: 792a6n1a A bad naming answer is refused
  Given a vault whose refit emitted naming requests for cluster 1
  When the user runs engram vocab refit --names with <answer>
  Then it fails with an error saying <problem>, and nothing is written

  Examples:
    | answer                                      | problem                                                     |
    | a fingerprint that does not match           | the vault changed since the naming requests were emitted    |
    | no entry for cluster 1                      | --names must name every new cluster: cluster 1 unnamed      |
    | an entry for cluster 7                      | --names references an unknown cluster                       |
    | two entries for cluster 1                   | --names names the same cluster twice                        |
    | an entry with an empty description          | --names entry needs non-empty term and description          |
    | a file that is not JSON                     | cannot parse --names JSON                                   |
```

## 792a6o Applying a refit

Requirement: [792a6o Applying a refit](#792a6o-applying-a-refit) A refit that has no new clusters, or whose `--names` answer is accepted, MUST
raise the major version ([792a6b The family note](#792a6b-the-family-note)), retire the unmatched derived terms ([792a6p Retiring a term](#792a6p-retiring-a-term)), create a definition
note for each named cluster with its exemplars' snippets, re-tag every note that is neither a
definition nor a question against the derived centroids, record each term's centroid, member count and
origin with the derivation's K, silhouette and date and a new refit baseline, clear any pending refit,
and print `vocab refit applied: version → <v> (K=<k>, matched=<m>, new=<n>, retired=<r>)`.

### 792a6o1 UAT Scenarios

```gherkin
Scenario: 792a6o1a A named cluster becomes a term
  Given a vault at version 1.0 whose refit emitted a naming request for cluster 1
  When the user runs engram vocab refit --names with cooking for cluster 1 and the emitted fingerprint
  Then it prints "vocab refit applied: version → 2.0 (K=2, matched=1, new=1, retired=0)", a vocab-cooking-definition note exists, and the cluster's notes carry vocab/cooking

Scenario: 792a6o1b A refit with nothing to name applies at once
  Given a vault whose derivation matches every cluster to a term
  When the user runs engram vocab refit
  Then it applies without emitting naming requests, and engram vocab stats reports the new major version and verdict: OK
```

## 792a6p Retiring a term

Requirement: [792a6p Retiring a term](#792a6p-retiring-a-term) Retiring a term MUST delete its definition note and the note's sidecar, remove
its `vocab/<term>` tag from every note, and remove every reference to the deleted note from the other
notes' bodies and `supersedes:` lists: a `supersedes:` entry naming it is dropped (and the
`supersedes:` key with its last entry), a `Supersedes: [[...]]` line linking it is dropped, and any
other link to it becomes its plain basename. Retirement MUST NOT keep the definition in any form or
record a supersession for it.

Rationale: a dormant term keeps attracting members at tagging time and stops the vocabulary from
converging, so retirement is active. A definition note is the term's identity, not a fact worth
preserving, so it has no supersession story; an earlier refit that demoted retired definitions and
recorded them as superseded on the family note left ten demoted notes and their dangling links in the
vault.

### 792a6p1 UAT Scenarios

```gherkin
Scenario: 792a6p1a A retired term is deleted with its references
  Given a vault whose derived term astronomy matches no cluster, whose definition another note's body links to and names in a supersedes: entry
  When the user runs engram vocab refit
  Then the astronomy definition note and its sidecar are gone, no note carries vocab/astronomy, the supersedes: entry and its Supersedes: line are gone, and the body link reads as the plain basename

Scenario: 792a6p1b Retirement leaves no dangling body link
  Given a vault whose family note links to a term's definition that refit retires
  When the user runs engram vocab refit and then engram check
  Then engram check reports no dangling link to the deleted basename from a note body
```

## 792a6q Proposed terms survive derivation

Requirement: [792a6q Proposed terms survive derivation](#792a6q-proposed-terms-survive-derivation) A term created by `engram vocab propose` MUST NOT be retired by a refit that
finds no cluster for it: its definition note, its recorded origin `proposed` and its members' tags
MUST remain, and it MUST keep taking part in tagging.

Rationale: proposed terms exist precisely to name concepts the clustering cannot see, so the clustering
must not remove them.

### 792a6q1 UAT Scenarios

```gherkin
Scenario: 792a6q1a An unmatched proposed term is kept
  Given a vault where the term gardening was proposed and no derived cluster matches it
  When the user runs engram vocab refit
  Then the gardening definition note remains and engram vocab stats still lists gardening
```

## 792a6r Refit takes no term plan

Requirement: [792a6r Refit takes no term plan](#792a6r-refit-takes-no-term-plan) `engram vocab refit` MUST NOT accept an externally authored term plan: the
`--plan` and `--emit-request` flags MUST be refused as undefined.

Rationale: an authored term list cannot coexist with derivation; keeping the plan flow would keep the
additive path that [792a6j Deriving the vocabulary](#792a6j-deriving-the-vocabulary) removed.

### 792a6r1 UAT Scenarios

```gherkin
Scenario Outline: 792a6r1a The plan flow is gone
  Given a vault
  When the user runs engram vocab refit <flag> x
  Then it fails saying the flag provided is not defined, and nothing is written

  Examples:
    | flag           |
    | --plan         |
    | --emit-request |
```

## 792a6s Refit trigger

Requirement: [792a6s Refit trigger](#792a6s-refit-trigger) After every `engram learn`, `engram amend` and `engram resituate` write, the vault
MUST check whether a refit is due, without running one: when no refit baseline is recorded it MUST
record the current number of notes that are neither definitions nor questions, and today's date, as
the baseline; otherwise, when that number has grown by at least 40 since the baseline and at least 14
days have passed, it MUST mark a refit pending with the reason `growth: <N> notes, <D> days`. A
pending refit MUST stay pending until a bootstrap or an applied refit records a new baseline. The
untagged rate and term concentration MUST NOT mark a refit pending.

Rationale: the check is token-free and runs in the write path, so the vault notices its own drift and
the learn skill acts on the verdict. Growth is the only trigger because, under derivation, a high
untagged rate or a hub term no longer means terms should be minted; keeping them as triggers would
keep the old ratchet's cadence without its mechanism.

### 792a6s1 UAT Scenarios

```gherkin
Scenario: 792a6s1a Growth marks a refit pending
  Given a vault whose refit baseline is 100 notes 20 days ago, now holding 150 notes
  When the user runs engram learn fact
  Then engram vocab stats reports "verdict: REFIT_PENDING (growth: <N> notes, <D> days)"

Scenario: 792a6s1b Growth without enough time does not trigger
  Given a vault whose refit baseline is 100 notes 5 days ago, now holding 150 notes
  When the user runs engram learn fact
  Then engram vocab stats reports verdict: OK

Scenario: 792a6s1c A first write records the baseline
  Given a vault with no refit baseline
  When the user runs engram learn fact
  Then a baseline is recorded, and engram vocab stats reports verdict: OK

Scenario: 792a6s1d A high untagged rate alone does not trigger
  Given a vault where more than 8% of notes are untagged and growth is below 40 notes
  When the user writes a note
  Then engram vocab stats reports the untagged rate with [high] and verdict: OK
```

## 792a6t Vocabulary statistics

Requirement: [792a6t Vocabulary statistics](#792a6t-vocabulary-statistics) `engram vocab stats` MUST print, without writing anything:
`vocab stats (version: <v>)` (`1.0` when there is no family note); `terms: <T>  member-notes: <M>
untagged: <U>`; `untagged-rate: <r>%`, followed by ` [high]` when the rate is above 8%; for each term
in name order `  <term>: <n> members`, followed by ` [hub]` when the term tags more than a quarter
of the member notes and ` [orphan]` when it has fewer than two members; `verdict: REFIT_PENDING
(<reason>)` when a refit is pending ([792a6s Refit trigger](#792a6s-refit-trigger)) and `verdict: OK` otherwise; `qa pairs: <P>`, the number
of question notes whose answer note exists; and `qa round-2 gate: READY (<P>>=20)` when `<P>` is at
least 20, else `qa round-2 gate: accumulating (<P>/20)`. Member notes MUST exclude definition notes,
question notes and files without frontmatter; answer notes MUST count like any other note.

Rationale: the counts describe the notes that tagging applies to, so definitions (whose self-tag is
display-only) and questions (which carry no vocabulary, [792a3u Vocabulary tagging by kind](./note-writing.md#792a3u-vocabulary-tagging-by-kind)) are left out. The untagged rate and
hubs are diagnostics for a person reading the report. The QA lines track the question-and-answer
kind's second round, which waits for 20 real captured pairs.

### 792a6t1 UAT Scenarios

```gherkin
Scenario: 792a6t1a Stats on an empty vault
  Given an empty vault
  When the user runs engram vocab stats
  Then it prints "terms: 0  member-notes: 0  untagged: 0" and nothing on stderr

Scenario: 792a6t1b Hubs and orphans are flagged
  Given a vault with term a tagging 3 of 4 member notes and term b tagging none
  When the user runs engram vocab stats
  Then it prints "  a: 3 members [hub]" and "  b: 0 members [orphan]"

Scenario: 792a6t1c Questions and definitions are not member notes
  Given a vault with one definition note, one fact note and a question-and-answer pair
  When the user runs engram vocab stats
  Then member-notes counts the fact and the answer only, and it prints "qa pairs: 1" and "qa round-2 gate: accumulating (1/20)"

Scenario: 792a6t1d A pending refit is the verdict
  Given a vault with a refit pending for "growth: 45 notes, 20 days"
  When the user runs engram vocab stats
  Then it prints "verdict: REFIT_PENDING (growth: 45 notes, 20 days)"
```

## 792a6u Self-tag backfill

Requirement: [792a6u Self-tag backfill](#792a6u-self-tag-backfill) `engram vocab tag-definitions` MUST add the missing `vocab/<term>` self-tag to
every definition note, taking the term from its slug, and print `<basename>: added` for each, MUST
leave a definition that already has it unchanged and print `<basename>: already present`, MUST skip
the family note printing `<basename>: family note (skipped)`, and MUST skip a bare-`vocab` note whose
slug names no term, printing `<basename>: malformed definition slug (skipped)`. A rewritten note's
sidecar MUST stay current. The backfill MUST run only when this command is run, never as a side
effect of `engram update` or any other command.

Rationale: vaults whose definitions predate the self-tag need it added once. Tags are not part of a
note's content hash (the hash covers the situation and body), so a tags-only rewrite needs no
re-embedding. Mutating the vault stays an explicit command.

### 792a6u1 UAT Scenarios

```gherkin
Scenario: 792a6u1a Backfill adds missing self-tags
  Given a vault whose definition notes carry only the bare vocab tag
  When the user runs engram vocab tag-definitions
  Then each per-term definition's tags become [vocab, vocab/<term>], engram embed status reports none of them stale, and one "added" line is printed per note

Scenario: 792a6u1b Backfill is idempotent
  Given a vault on which engram vocab tag-definitions has run
  When the user runs it a second time
  Then no note changes and every definition is reported "already present", the family note "family note (skipped)"
```

## 792a6v Update notices

Requirement: [792a6v Update notices](#792a6v-update-notices) `engram update` MUST print the line
``old-format vocab files found — run `engram update --regen-vocab` to migrate them (preview with `engram update --regen-vocab --dry-run`)``
when the vault holds a file named `vocab.<name>.md`, and the line
``vocab definition notes missing their vocab/<term> self-tag found — run `engram vocab tag-definitions` to backfill them``
when a definition note lacks its self-tag. It MUST NOT change the vault to resolve either.

Rationale: update detects and notifies, and leaves rewriting the vault to an explicit command.

### 792a6v1 UAT Scenarios

```gherkin
Scenario: 792a6v1a An old-format vault is flagged
  Given a vault holding vocab.index.md
  When the user runs engram update
  Then it prints the old-format vocab notice and the file is still there

Scenario: 792a6v1b A missing self-tag is flagged
  Given a vault with a definition note tagged only vocab
  When the user runs engram update
  Then it prints the missing self-tag notice and the note is unchanged

Scenario: 792a6v1c A current vault gets no vocab notice
  Given a vault with no vocab.<name>.md file and fully self-tagged definitions
  When the user runs engram update
  Then neither vocab notice is printed
```

## 792a6w Migrating an old-format vocabulary

Requirement: [792a6w Migrating an old-format vocabulary](#792a6w-migrating-an-old-format-vocabulary) `engram update --regen-vocab` MUST migrate a vault that holds old-format
vocabulary files: it MUST create the family note when absent and a definition note for each
`vocab.<term>.md` term note from its `term`, `description` and `Exemplars:` list, remove the `vocab:`
frontmatter line and `Vocab:` body line from every note that has them, delete every `vocab.<name>.md`
file and its sidecar, and re-tag every note as bootstrap does ([792a6g Bootstrapping a vocabulary](#792a6g-bootstrapping-a-vocabulary)). It MUST then print
`update --regen-vocab: removed <F> old-format file(s), cleaned <C> member note(s), seeded <S> term(s),
reassigned <A> note(s)`, or `update --regen-vocab: nothing to regenerate` when there were no old files
and no legacy lines. With `--dry-run` it MUST change nothing and print the same line prefixed
`[dry-run] ` with `would remove` in place of `removed`. After a migration, the old-format notice of
[792a6v Update notices](#792a6v-update-notices) MUST NOT be printed.

Rationale: before the vocabulary moved onto tags, terms were `vocab.<term>.md` notes, a generated
`vocab.index.md` listed them, and members carried a `Vocab:` line and a `vocab:` key. The migration
re-scores members from the regenerated centroids rather than copying the old assignment.

### 792a6w1 UAT Scenarios

```gherkin
Scenario: 792a6w1a An old-format vocabulary is migrated
  Given a vault holding vocab.cooking.md, vocab.index.md and a note with a Vocab: line
  When the user runs engram update --regen-vocab
  Then a vocab-cooking-definition note exists, both old files are gone, the note has no Vocab: line, and the report says removed 2 old-format file(s) and cleaned 1 member note(s)

Scenario: 792a6w1b A dry run changes nothing
  Given the same vault
  When the user runs engram update --regen-vocab --dry-run
  Then it prints "[dry-run] update --regen-vocab: would remove 2 old-format file(s), ..." and every file is unchanged

Scenario: 792a6w1c Nothing to migrate
  Given a vault with no old-format vocabulary
  When the user runs engram update --regen-vocab
  Then it prints "update --regen-vocab: nothing to regenerate"
```
