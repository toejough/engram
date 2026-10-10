# Runbook Trigger Matching - Technical Spec

## 792b22a Orientation

- Ownership: Repository component.
- Responsibility: decides which runbooks a user's message triggers and puts them first. It indexes the
  literal triggers of the runbook notes, matches them against the message as whole words without regard
  to case or runs of whitespace, and places each hit ahead of every other result with the provenance
  `trigger`. It reads no files, embeds nothing and never ranks by similarity.
- Current realization (informational): `matchTriggers`, `containsWholeWord`, `applyTriggerHits` and
  `runTriggerOnlyQuery` in `internal/cli/query_triggers.go`; the trigger index filled by
  `loadAllVaultNotesMeta` in `internal/cli/query_vault_meta.go`; the ordering rule `resolvedItemLess`
  in `internal/cli/query.go`.
- Qualification: responsibility, finding and placing triggered runbooks is stated without naming any
  function; ownership, the query engine decides what to pass in and what the payload looks like, while
  this component owns what counts as a hit and where hits go; independent verification, matching is
  tested directly on texts and trigger lists, with a property test against a regular-expression
  oracle; stability, the contract survives any change to how notes are scanned or the payload is
  rendered; necessity, a cue-triggered runbook can be surfaced reliably only through it.
- Supported capabilities: [memory-query](../product/memory-query.md), [ambient-guidance](../product/ambient-guidance.md) (the shim passes the
  message).

Rationale: a runbook is normally found by the similarity of its situation to the agent's phrases, but
the agent paraphrases the user's request into those phrases and the paraphrase drops the cue. Embedding a paraphrase cannot reproduce a literal cue. So the author declares literal cues
(`triggers:`, written by `engram learn runbook --trigger` and replaced by `engram amend --trigger`),
and the message is matched against them lexically and never embedded. `triggers:` is not part of the
content hash, so declaring one re-embeds nothing, and a runbook without a hit, and a query without a
message, rank as they always did. A hit is only a candidate: the agent still judges it against the
runbook's own applicability text. The match is whole-word, because bare-word triggers such as `curate` and `please` would
otherwise hit `accurate` and `pleased`.

## 792b22c Behavior Requirements

### 792b22c1 Trigger index

Requirement: [792b22c1 Trigger index](#792b22c1-trigger-index) The runbook-trigger-matching MUST index, from each note read as its LF form, the
non-empty entries of the `triggers:` list of every note whose `type` is `runbook`, under the note's
basename and in the order written, and MUST NOT index a runbook that has none or a `triggers:` list on a
note of any other kind. Only a note with a current sidecar MUST be indexed.

Supports: [792a7za1c Other kinds never hit](../product/memory-query.md#792a7za1-uat-scenarios), [792a7zg1b A CRLF runbook's trigger matches](../product/memory-query.md#792a7zg1-uat-scenarios).

Rationale: triggers are read in the same pass over the vault that finds vocabulary tags and
supersessions, so a query needs no second scan, and only `runbook` notes carry a contract for them.

#### 792b22c1a Scenarios

```gherkin
Scenario: 792b22c1a1 A runbook's triggers are indexed
  Given a runbook with a sidecar and triggers ["/please", "take this end-to-end"]
  When the index is built
  Then it maps the runbook's basename to both triggers in order

Scenario: 792b22c1a2 Other kinds are not indexed
  Given a fact note with a sidecar whose frontmatter lists triggers ["/please"]
  When a message containing /please is matched
  Then the note is not a hit and its item has no trigger provenance

Scenario: 792b22c1a3 A CRLF runbook is indexed and matched
  Given a runbook whose lines end in \r\n and whose triggers include "rotate the keys"
  When the index is built and the message "please rotate the keys today" is matched
  Then the index maps the basename to ["rotate the keys"] and the runbook is the one hit
```

### 792b22c2 Whole-word occurrence

Requirement: [792b22c2 Whole-word occurrence](#792b22c2-whole-word-occurrence) The runbook-trigger-matching MUST treat a trigger as occurring in a message when,
after the message and the trigger are lowercased and each run of whitespace in them is collapsed to one
space, the trigger is found in the message with, on each side where the trigger's own edge character is
a letter or a digit, no letter or digit next to it, letters and digits being those of Unicode, so
punctuation, an underscore and a hyphen are boundaries, as are the start and end of the message. On a side
where the trigger's edge character is neither, there MUST be no boundary requirement. Every occurrence
MUST be tried. A trigger that is empty after normalization MUST never occur.

Supports: [792a7zb1a Case and whitespace are ignored](../product/memory-query.md#792a7zb1-uat-scenarios), [792a7zb1b A bare word hits as a whole word](../product/memory-query.md#792a7zb1-uat-scenarios), [792a7zb1c A bare word does not hit inside a longer word](../product/memory-query.md#792a7zb1-uat-scenarios), [792a7zb1d A slash form needs only a right boundary](../product/memory-query.md#792a7zb1-uat-scenarios), [792a7zb1e A slash before a bare word is a boundary](../product/memory-query.md#792a7zb1-uat-scenarios), [792a7zb1f The edges of the text are boundaries](../product/memory-query.md#792a7zb1-uat-scenarios), [792a7zb1g A later occurrence can match](../product/memory-query.md#792a7zb1-uat-scenarios), [792a7zb1h Unicode letters and digits are not boundaries](../product/memory-query.md#792a7zb1-uat-scenarios),
[792a7zb1i Hyphen and underscore are boundaries](../product/memory-query.md#792a7zb1-uat-scenarios).

Rationale: the boundary is enforced only where the trigger itself ends in a letter or a digit, because
an author who wrote punctuation at the edge supplied the delimiter: `/please` hits in `do/please` and
not in `/pleased`. The scan is two rune lookups over `strings.Index`, not a regular expression built
from author text, which needs escaping and could be abused.

#### 792b22c2a Scenarios

```gherkin
Scenario Outline: 792b22c2a1 Occurrence in a message
  Given the trigger <trigger>
  When the message is <message>
  Then the result is <result>

  Examples:
    | case | message | trigger | result |
    | bare word mid sentence | please curate the offers | curate | hit |
    | bare word at start | curate the offers | curate | hit |
    | bare word at end | time to curate | curate | hit |
    | bare word is whole text | curate | curate | hit |
    | trailing punctuation | Curate! | curate | hit |
    | parentheses | (curate) | curate | hit |
    | slash before is a boundary | /curate | curate | hit |
    | hyphen is a boundary | re-curate-now | curate | hit |
    | underscore is a boundary | do_curate_it | curate | hit |
    | inside longer word (prefix) | that is accurate | curate | no hit |
    | inside longer word (in-) | inaccurate | curate | no hit |
    | suffix letters | curated offers | curate | no hit |
    | plural suffix | curates | curate | no hit |
    | pleased is not please | I am pleased | please | no hit |
    | displease is not please | do not displease | please | no hit |
    | first occurrence fails, later matches | accurate, curate now | curate | hit |
    | all occurrences fail | accurate and curated | curate | no hit |
    | digit after blocks | curate2 now | curate | no hit |
    | digit before blocks | 2curate now | curate | no hit |
    | unicode letter before blocks | écurate | curate | no hit |
    | unicode letter after blocks | curateé | curate | no hit |
    | unicode punctuation neighbor allows | «curate» | curate | hit |
    | case insensitive | CURATE the offers | Curate | hit |
    | slash trigger at start | /please fix it | /please | hit |
    | slash trigger at end | do /please | /please | hit |
    | slash trigger needs no left boundary | x/please fix | /please | hit |
    | slash trigger still needs right boundary | /pleased | /please | no hit |
    | phrase after prefix word | Please take this end-to-end: X | take this end-to-end | hit |
    | phrase left boundary enforced | retake this end-to-end | take this end-to-end | no hit |
    | trigger ending in punctuation needs no right boundary | ok e.g.x | e.g. | hit |
    | whitespace collapse | PLEASE   take\tthis<newline>end-to-end | take this end-to-end | hit |
    | empty text | (empty) | curate | no hit |

Scenario: 792b22c2a2 A runbook is a hit exactly when a trigger occurs as a whole word
  Given any message and any trigger lists drawn from letters, digits, é, underscore, slash, dot and hyphen
  When the runbooks are matched
  Then a runbook is a hit if and only if one of its triggers occurs, by the rule of 792b22c2, in the normalized message

Scenario: 792b22c2a3 Case and whitespace are ignored in a full query
  Given a runbook with the trigger "take this end-to-end"
  When the user runs engram query with the message "PLEASE   TAKE   this\tend-to-end: rename X"
  Then the runbook is the first item, with provenance trigger
```

### 792b22c3 The hits

Requirement: [792b22c3 The hits](#792b22c3-the-hits) The runbook-trigger-matching MUST return the basenames of the runbooks with at
least one occurring trigger, each once, in ascending order, and MUST return none for a message that is
empty after normalization.

Supports: [792a7y1a Text alone is a trigger lookup](../product/memory-query.md#792a7y1-uat-scenarios), [792a7za1a A hit with no similarity match is first](../product/memory-query.md#792a7za1-uat-scenarios).

#### 792b22c3a Scenarios

```gherkin
Scenario: 792b22c3a1 Hits are listed once, in order
  Given three runbooks whose triggers occur in the message, one of them by two triggers
  When the runbooks are matched
  Then their basenames are listed in ascending order, each once

Scenario: 792b22c3a2 An empty message has no hits
  Given the message "" and a runbook with the trigger "curate"
  When the runbooks are matched
  Then there are no hits
```

### 792b22c4 Placing the hits

Requirement: [792b22c4 Placing the hits](#792b22c4-placing-the-hits) The runbook-trigger-matching MUST, given the ranked items and the hits, put every
hit ahead of all other items with `trigger` added to its provenances. A hit that is already ranked MUST
keep its score, content and other provenances. A hit that is not ranked MUST be added with a score of 0,
the provenance `trigger`, and its note's content, and MUST NOT be added when a project is requested and
the note's frontmatter does not name it. A chunk MUST NOT be turned into a hit. The hits MUST be ordered
among themselves by the ordinary rule, more provenances first, then the higher provenance rank, then the
higher score. With no hits the items MUST be returned unchanged.

Supports: [792a7za1a A hit with no similarity match is first](../product/memory-query.md#792a7za1-uat-scenarios), [792a7za1b A hit outranks a stronger match](../product/memory-query.md#792a7za1-uat-scenarios), [792a7za1d A hit that matched keeps its score](../product/memory-query.md#792a7za1-uat-scenarios), [792a7za1e A hit outside the project is not surfaced](../product/memory-query.md#792a7za1-uat-scenarios).

#### 792b22c4a Scenarios

```gherkin
Scenario: 792b22c4a1 A hit with no similarity match leads the payload
  Given a runbook whose situation is unrelated to the phrases but whose trigger "/please" is in the message, and a similarity-matched feedback note
  When the user runs engram query with the phrases and the message "/please fix the flaky login test"
  Then the first item is the runbook, kind runbook, with provenance trigger and its content, and the feedback note follows

Scenario: 792b22c4a2 A ranked hit keeps its score
  Given a runbook that is both similarity-matched and a hit
  When the hits are placed
  Then the runbook is first with provenances including direct and trigger and its own score

Scenario: 792b22c4a3 A chunk is not a hit
  Given a chunk item whose path equals a hit's path
  When the hits are placed
  Then the chunk keeps its place after the hits and gains no trigger provenance

Scenario: 792b22c4a4 A hit outside the requested project is not added
  Given a hit that is not ranked and whose note names another project
  When the hits are placed with a project requested
  Then the note is not added

Scenario: 792b22c4a5 No hits changes nothing
  Given no hits
  When the hits are placed
  Then the items are returned as they were
```

### 792b22c5 Trigger rank

Requirement: [792b22c5 Trigger rank](#792b22c5-trigger-rank) The runbook-trigger-matching MUST order any item whose provenances include
`trigger` before every item whose provenances do not, whatever their numbers of provenances, their
ranks and their scores.

Supports: [792a7za1b A hit outranks a stronger match](../product/memory-query.md#792a7za1-uat-scenarios).

Rationale: a hit may have only the one role, and it must still beat an item that matched by similarity
under two.

#### 792b22c5a Scenarios

```gherkin
Scenario: 792b22c5a1 A trigger always sorts first
  Given any provenance sets and scores for two items, one of which includes trigger
  When they are compared
  Then the one with trigger precedes the other, and not the reverse
```

### 792b22c6 Lookup by message alone

Requirement: [792b22c6 Lookup by message alone](#792b22c6-lookup-by-message-alone) When a query has a message and no phrase, the runbook-trigger-matching MUST answer
from the hits alone: it MUST embed nothing, match no notes by similarity, cluster nothing, sample no
explore notes and add no recent chunks, and MUST return the hits as the items with no clusters and an
empty `explore_allocated`.

Supports: [792a7y1a Text alone is a trigger lookup](../product/memory-query.md#792a7y1-uat-scenarios).

#### 792b22c6a Scenarios

```gherkin
Scenario: 792b22c6a1 The message alone returns the hit
  Given a runbook with the trigger "/please" and a similarity-matched feedback note
  When the user runs engram query with only the message "/please anything"
  Then the payload has one item, the runbook, with provenances [trigger]
```

### 792b22c7 No message, no matching

Requirement: [792b22c7 No message, no matching](#792b22c7-no-message-no-matching) When a query has no message, the runbook-trigger-matching MUST NOT match any
trigger, and a message that hits nothing MUST leave the payload equal to the payload without a message.

Supports: [792a7y1b Without text nothing changes](../product/memory-query.md#792a7y1-uat-scenarios).

#### 792b22c7a Scenarios

```gherkin
Scenario: 792b22c7a1 A query without a hit is unchanged
  Given a vault whose runbooks carry triggers
  When the user runs engram query with phrases and no message, and then with a message that hits nothing
  Then the two payloads are equal and no item has provenance trigger
```
