# Query Payload Renderer - Technical Spec

## 792b23a Orientation

- Ownership: Repository component; the YAML document goes to stdout.
- Responsibility: turns what the [query-engine](./query-engine.md) found into the payload the reader gets. It puts the
  trigger hits first and the newest chunks last, cuts the matched items to the limit, applies the chunk
  content policy (full text, snippets or none), previews an oversized red-flag list, labels each item,
  fills the budget block and the header fields, adds the stage timings and encodes the YAML document.
  It decides nothing about what matched.
- Current realization (informational): `renderQueryPayload`, `renderItems`, `kindFromContent`,
  `capItemsToLimit`, `splitTriggerItems`, `applyContentPolicy`, `capChunkContent`, `clearChunkContent`,
  `snippet`, `resolveContentBudget`, `renderTimings` and the types `queryPayload`, `queryItem`,
  `queryBudget` and `queryTimings` in `internal/cli/query.go`, `query_triggers.go` and
  `merged_query.go`; `capRedFlagsForPreview` in `internal/cli/redflags_truncation.go`. The keys that
  lead a payload with pending offers ([curate](../product/curate.md)) are added by that capability, and the tags of a
  merged payload are [792b23c9 The item tags of a merge](#792b23c9-the-item-tags-of-a-merge).
- Qualification: responsibility, shaping and cutting a payload is stated without naming any
  function; ownership, the [query-engine](./query-engine.md) decides what is in it, while this component owns its order,
  its cuts and its labels; independent verification, it is tested directly on item lists and notes,
  with a property test for the red-flag preview; stability, the contract survives any change to how
  matches are found; necessity, every query payload is made by it.
- Supported capabilities: [memory-query](../product/memory-query.md), [parent-sync](../product/parent-sync.md) ([792b23c9 The item tags of a merge](#792b23c9-the-item-tags-of-a-merge)), [curate](../product/curate.md) ([792b23c10 The lead of a payload with pending offers](#792b23c10-the-lead-of-a-payload-with-pending-offers)). Recall extends it.

Rationale: a recall pass should carry less to read without losing reach, so the payload is cut where
the reader gains least: chunk text is shortened or fetched on demand, the newest chunks have a count of
their own, and the matched items have a limit. A reader that opens the payload through a tool that truncates its output must still see the most
recent red flag, which a long runbook would otherwise leave past the truncation boundary.

## 792b23c Behavior Requirements

### 792b23c1 The document

Requirement: [792b23c1 The document](#792b23c1-the-document) The query-payload-renderer MUST encode one YAML document with the keys `version`
(always 1), `phrases`, `items`, `clusters`, `budget`, `refit_pending`, `model_id` and `timings`, in that
order, indented by two spaces. It MUST omit `refit_pending` unless the vault has a refit pending and
`timings` unless timing is on, MUST give `model_id` the embedding model of the run even when the vault
is empty, and MUST give `items` and `clusters` as empty lists, not omit them, when there are none.

Supports: [792a7b1a The payload has the documented shape](../product/memory-query.md#792a7b1-uat-scenarios), [792a7zc1a Every payload names its model](../product/memory-query.md#792a7zc1-uat-scenarios), [792a7zd1a A pending refit is reported](../product/memory-query.md#792a7zd1-uat-scenarios), [792a7zd1b Otherwise the field is absent](../product/memory-query.md#792a7zd1-uat-scenarios).

#### 792b23c1a Scenarios

```gherkin
Scenario: 792b23c1a1 A pending refit is reported
  Given a result whose refit is pending
  When the payload is rendered
  Then it contains refit_pending: true

Scenario: 792b23c1a2 Otherwise the key is absent
  Given a result whose refit is not pending
  When the payload is rendered
  Then it has no refit_pending

Scenario: 792b23c1a3 The model is named
  Given an empty vault and an embedder whose model is m@4
  When the user runs engram query with a phrase
  Then the payload's model_id is m@4

Scenario: 792b23c1a4 The keys come in order
  Given a result with items and clusters
  When the payload is rendered
  Then version, phrases, items, clusters, budget and model_id appear in that order
```

### 792b23c2 Item labels

Requirement: [792b23c2 Item labels](#792b23c2-item-labels) The query-payload-renderer MUST give each item its `path`, `kind`, `score` and
`provenances`, its `content` unless that is empty, and its `source_term` when it has one. The kind of a
chunk MUST be `chunk`; the kind of a note MUST be the value of the `type:` line found within the first
256 bytes of its content read as LF, and `unknown` when the content is shorter than the shortest
frontmatter that has a type, has no such line there, or has the line later.

Supports: [792a7b1b A note item is labelled by its type](../product/memory-query.md#792a7b1-uat-scenarios), [792a7m1a An explore note names its term](../product/memory-query.md#792a7m1-uat-scenarios), [792a7zg1f A CRLF note keeps its kind](../product/memory-query.md#792a7zg1-uat-scenarios).

#### 792b23c2a Scenarios

```gherkin
Scenario Outline: 792b23c2a1 The kind of a note
  Given the content <content>
  When its kind is read
  Then it is <kind>

  Examples:
    | case                     | content                                        | kind     |
    | empty content            | (empty)                                        | unknown  |
    | shorter than a type line | ---                                            | unknown  |
    | no type line             | ---\nsituation: x\n---\nbody                   | unknown  |
    | a fact                   | ---\ntype: fact\nluhmann: "1"\n---\nbody        | fact     |
    | a feedback note          | ---\ntype: feedback\n---\nbody                 | feedback |
    | the type is too far in   | 80 lines of "a: b" then type: fact             | unknown  |

Scenario: 792b23c2a2 A CRLF note has its kind
  Given a fact note whose lines end in \r\n
  When its kind is read
  Then it is fact

Scenario: 792b23c2a3 An explore item shows its term
  Given a resolved item with provenance explore and source term topic-a
  When the payload is rendered
  Then it contains "- explore" and "source_term: topic-a"
```

### 792b23c3 Order and limit

Requirement: [792b23c3 Order and limit](#792b23c3-order-and-limit) The query-payload-renderer MUST place the items that are trigger hits first, then the
other items that are not recent chunks, cut to the first `limit` of them in the order given, then the
recent chunks, which it MUST NOT cut. A limit of zero or less MUST cut nothing. It MUST NOT count a
trigger hit or a recent chunk against the limit.

Supports: [792a7q1a The default caps a large result](../product/memory-query.md#792a7q1-uat-scenarios), [792a7q1b An explicit limit caps the result](../product/memory-query.md#792a7q1-uat-scenarios), [792a7q1c A small result is returned whole](../product/memory-query.md#792a7q1-uat-scenarios), [792a7q1d Trigger hits do not use the limit](../product/memory-query.md#792a7q1-uat-scenarios), [792a7q1e The recent chunks survive a full set of matches](../product/memory-query.md#792a7q1-uat-scenarios).

Rationale: the recency channel has its own count, so cutting both under one limit would displace the
whole channel in any vault whose matches reach the limit, which is routine; trigger hits are exempt so
that a cue the author declared is never cut by an unrelated limit ([runbook-trigger-matching](./runbook-trigger-matching.md)).

#### 792b23c3a Scenarios

```gherkin
Scenario: 792b23c3a1 Only the first N are kept
  Given the paths a, b, c, d, e in rank order
  When they are cut to 3
  Then a, b and c remain

Scenario: 792b23c3a2 Fewer than the limit are all kept
  Given the paths a and b and a limit of 5
  When they are cut
  Then a and b remain

Scenario: 792b23c3a3 An explicit limit caps the payload
  Given ten matched notes
  When the user runs engram query --limit 3
  Then items holds three

Scenario: 792b23c3a4 A small result is whole
  Given one matched note
  When the user runs engram query
  Then items holds that note

Scenario: 792b23c3a5 The recent chunks survive a full set of matches
  Given 25 matched notes, a recent chunk and the default limit
  When the user runs engram query
  Then items holds the recent chunk, with provenance recent

Scenario: 792b23c3a6 A trigger hit does not use the limit
  Given a trigger hit, a similarity-matched note and one more note, and a limit of 1
  When the user runs engram query with the message and the phrases
  Then items holds the hit first and then one other item
```

### 792b23c4 Chunk content budget

Requirement: [792b23c4 Chunk content budget](#792b23c4-chunk-content-budget) The query-payload-renderer MUST keep the first `budget` chunk items of the payload,
in the order printed, with their full content and MUST replace the content of every later chunk item
with a snippet, counting the replaced items. The budget MUST be the default of 15 when given as zero, no
limit when negative, and otherwise as given. A snippet MUST be the content with each run of whitespace
collapsed to one space and the ends trimmed, and when it is longer than 160 characters it MUST be cut to
159 and end in `…`. Note items MUST NOT be changed.

Supports: [792a7r1a Later chunks become snippets](../product/memory-query.md#792a7r1-uat-scenarios), [792a7r1b A long chunk is cut with an ellipsis](../product/memory-query.md#792a7r1-uat-scenarios), [792a7r1c Notes are never reduced](../product/memory-query.md#792a7r1-uat-scenarios), [792a7r1d A negative budget means none](../product/memory-query.md#792a7r1-uat-scenarios).

#### 792b23c4a Scenarios

```gherkin
Scenario: 792b23c4a1 Later chunks become snippets
  Given a note and four long multi-line chunks, and a budget of 2
  When the content is cut
  Then the note and the first two chunks are unchanged, the others are single-line snippets ending in …, and 2 are counted

Scenario: 792b23c4a2 A budget of zero or less cuts nothing
  Given three long chunks and a budget of 0 passed straight to the cut
  When the content is cut
  Then every chunk is unchanged and none is counted

Scenario: 792b23c4a3 The budget defaults and overrides
  Given budgets of 0, 8 and -1
  When each is resolved
  Then they are the default of 15, 8, and no limit

Scenario: 792b23c4a4 A snippet collapses and truncates
  Given the text "  hello\n\n  world  " and a text of 500 characters
  When each is made a snippet
  Then the first is "hello world" and the second is at most 160 characters ending in …
```

### 792b23c5 Lazy chunks

Requirement: [792b23c5 Lazy chunks](#792b23c5-lazy-chunks) When lazy chunks are on, the query-payload-renderer MUST clear the content of every
chunk item, MUST leave every note item's content, MUST NOT snippet anything, and MUST put
`lazy_chunks: true` in the budget. Otherwise it MUST NOT put `lazy_chunks` in the budget.

Supports: [792a7s1a Chunks carry no content](../product/memory-query.md#792a7s1-uat-scenarios), [792a7s1b Notes keep their content](../product/memory-query.md#792a7s1-uat-scenarios), [792a7s1c The environment can turn it on](../product/memory-query.md#792a7s1-uat-scenarios), [792a7s1d Without the flag the field is absent](../product/memory-query.md#792a7s1-uat-scenarios).

#### 792b23c5a Scenarios

```gherkin
Scenario: 792b23c5a1 Chunks are cleared and notes kept
  Given a note and two chunks
  When the chunk content is cleared
  Then the note keeps its content and both chunks have none

Scenario: 792b23c5a2 Lazy mode is reported and snippets nothing
  Given a note and two long chunks, lazy chunks on and a budget of 1
  When the payload is rendered
  Then the budget has lazy_chunks: true and chunks_snippeted: 0

Scenario: 792b23c5a3 Without lazy mode the key is absent
  Given the same items, lazy chunks off and a budget of 1
  When the payload is rendered
  Then the budget has no lazy_chunks and chunks_snippeted: 1
```

### 792b23c6 Budget block

Requirement: [792b23c6 Budget block](#792b23c6-budget-block) The query-payload-renderer MUST fill the budget with the phrases queried, the notes
in the vault, the notes with embeddings, the clusters found, the limit and the explore allocation as
the [query-engine](./query-engine.md) gives them, `direct_hits_returned` as the number of items printed whose provenances include `direct`,
`items_with_full_content` as the number of items printed with content, less those reduced to snippets,
`content_budget` as the budget resolved by [792b23c4 Chunk content budget](#792b23c4-chunk-content-budget) and `chunks_snippeted` as the number reduced, and
MUST print `explore_allocated` in every payload, `{}` when empty.

Supports: [792a7b1a The payload has the documented shape](../product/memory-query.md#792a7b1-uat-scenarios), [792a7m1c The budget counts what was delivered](../product/memory-query.md#792a7m1-uat-scenarios), [792a7q1a The default caps a large result](../product/memory-query.md#792a7q1-uat-scenarios).

#### 792b23c6a Scenarios

```gherkin
Scenario: 792b23c6a1 The limit is reported as given
  Given a limit of 7 and one matched note
  When the user runs engram query --limit 7
  Then the budget's limit is 7

Scenario: 792b23c6a2 Counts follow what is printed
  Given a payload of two matched notes and one snippeted chunk
  When it is rendered
  Then direct_hits_returned counts the direct items, items_with_full_content leaves out the snippet and chunks_snippeted is 1

Scenario: 792b23c6a3 The allocation is always present
  Given a result whose explore allocation is empty
  When the payload is rendered
  Then the budget has explore_allocated: {}
```

### 792b23c7 Stage timings

Requirement: [792b23c7 Stage timings](#792b23c7-stage-timings) When timing is on, the query-payload-renderer MUST read the injected clock to end the
render stage and MUST add the `timings` block with `scan_ms`, `embed_ms`, `cluster_ms`, `nominate_ms`
and `render_ms`, each the whole milliseconds between the readings that the [query-engine](./query-engine.md) took at the
start of the query and at the end of each stage, the stages following one another. A run MUST read the
clock six times in all. When timing is off, or there is no clock, it MUST NOT read the clock for
timing and MUST NOT add the block.

Supports: [792a7ze1a Timings are reported on request](../product/memory-query.md#792a7ze1-uat-scenarios), [792a7ze1b Timings are absent by default](../product/memory-query.md#792a7ze1-uat-scenarios).

Rationale: a measurement-only diagnostic: the clock is injected so that tests can script it, and
the default payload stays unchanged so that consumers that ignore the block are unaffected.

#### 792b23c7a Scenarios

```gherkin
Scenario: 792b23c7a1 Each stage is the gap between readings
  Given a clock reading 0, 500, 1500, 1600, 1700 and 1720 milliseconds
  When the user runs engram query --timings with a phrase
  Then scan_ms is 500, embed_ms 1000, cluster_ms 100, nominate_ms 100 and render_ms 20

Scenario: 792b23c7a2 No timing, no block
  Given a query without --timings
  When the payload is rendered
  Then it contains no timings
```

### 792b23c8 Red-flag preview

Requirement: [792b23c8 Red-flag preview](#792b23c8-red-flag-preview) The query-payload-renderer MUST preview the content of a note read as its LF form
so that a `red_flags:` list taking more than 1200 bytes of rendered YAML keeps its newest entries,
those listed last, in a run that ends at the last entry and, with the marker, takes at most 1200
bytes, and replaces the earlier ones with the entry
`"[<n> EARLIER RED_FLAGS OMITTED — run engram show <basename> for all <total>]"` naming the number
omitted, the list's total and the note's real basename. Everything before and after the list MUST be
unchanged, and a note without a red-flag list, or whose list fits the budget, MUST be returned unchanged.
`engram show` MUST NOT be subject to the preview.

Supports: [792a7w1a Red flags are in the payload](../product/memory-query.md#792a7w1-uat-scenarios), [792a7w1b An oversized list keeps its newest entries](../product/memory-query.md#792a7w1-uat-scenarios), [792a7w1c A short list is unchanged](../product/memory-query.md#792a7w1-uat-scenarios), [792a7w1d A trigger hit is truncated the same way](../product/memory-query.md#792a7w1-uat-scenarios), [792a7zg1e A CRLF runbook's red flags are previewed](../product/memory-query.md#792a7zg1-uat-scenarios).

Rationale: callers append new entries, so the newest is last; dropping from the front keeps the entry
most likely to matter now. The marker names the real basename and counts so that the command it prints
resolves and the reader knows how much it has not seen. The budget is counted in rendered bytes, and
the marker's own length depends on how many entries it says were dropped, so the cut is refitted until
the kept set stops shrinking.

#### 792b23c8a Scenarios

```gherkin
Scenario: 792b23c8a1 An oversized list keeps its newest entry
  Given a runbook with 20 long red flags and a newest one after them
  When it is previewed
  Then the newest entry and the marker are present, the content before and after the list is unchanged, and the list is smaller

Scenario: 792b23c8a2 The marker names the count, the total and the basename
  Given 15 entries of exactly 100 bytes each and the basename 1.test-note
  When the note is previewed
  Then the list holds the marker "[4 EARLIER RED_FLAGS OMITTED — run engram show 1.test-note for all 15]" and the last 11 entries

Scenario: 792b23c8a3 The command in the marker returns the whole list
  Given a previewed oversized list and the vault's full note
  When the command in the marker is run
  Then engram show prints every entry, including the omitted ones

Scenario: 792b23c8a4 A note without red flags is unchanged
  Given a fact note
  When it is previewed
  Then it is returned unchanged

Scenario: 792b23c8a5 A list within the budget is unchanged
  Given a runbook with two short red flags
  When it is previewed
  Then it is returned byte for byte

Scenario: 792b23c8a6 Any list is cut to a contiguous newest run
  Given any red-flag list, of 0 to 25 entries of up to 120 characters
  When it is previewed
  Then a list within the budget is unchanged, and otherwise the result is the exact marker followed by a contiguous run of entries ending at the newest, within 1200 bytes

Scenario: 792b23c8a7 A CRLF note is previewed as its LF form
  Given a runbook whose lines end in \r\n and whose red flags take more than 1200 bytes
  When it is previewed
  Then the result equals the preview of the same note with LF line endings

Scenario: 792b23c8a8 A query previews a matched runbook
  Given a runbook with 21 red flags and a phrase that matches it
  When the user runs engram query
  Then its content holds the newest entry, the marker "EARLIER RED_FLAGS OMITTED" and "run engram show 1.2026-09-18.oversized-red-flags for all 21"

Scenario: 792b23c8a9 A query shows a runbook's red flags
  Given a runbook with two red flags and a phrase that matches it
  When the user runs engram query
  Then its content holds done_when, red_flags and both entries
```

### 792b23c9 The item tags of a merge

Requirement: [792b23c9 The item tags of a merge](#792b23c9-the-item-tags-of-a-merge) The query-payload-renderer MUST write an item's `from_parent` and `model_id` only when a merge tagged
the item, writing `from_parent: false` explicitly for a local item of a merged payload, so that a payload that no merge
produced holds no `from_parent` key at all.

Supports: [792a21g1a Parent-origin and local-origin items are distinguishable](../product/parent-sync.md#792a21g1-uat-scenarios), [792a21g1b A payload with no merge has no from_parent](../product/parent-sync.md#792a21g1-uat-scenarios).

Rationale: a boolean that is omitted when false would drop the local items' explicit `false`, and the merged payload could
not tell a local item from an item with no origin. The tag is a pointer for that reason, and each item gets its own, so
that a later write through one never aliases another.

#### 792b23c9a Scenarios

```gherkin
Scenario: 792b23c9a1 No merge, no from_parent
  Given resolved items and no merge step
  When the payload is rendered
  Then it contains no from_parent

Scenario: 792b23c9a2 A local item of a merged payload says false
  Given a merged payload with a local item and a parent item
  When it is marshaled to YAML
  Then the local item has from_parent: false and the parent item has from_parent: true
```

### 792b23c10 The lead of a payload with pending offers

Requirement: [792b23c10 The lead of a payload with pending offers](#792b23c10-the-lead-of-a-payload-with-pending-offers) The query-payload-renderer MUST write `pending_offers` and `pending_offers_hint` immediately after `version`, before
`items` and every other large field, so that a truncated preview of a large payload still shows them, and MUST write neither when
no offer is pending ([792b34c4 The curate instruction](./pending-offer-bookkeeping.md#792b34c4-the-curate-instruction)).

Supports: [792a22o1c The flag and hint lead the payload](../product/curate.md#792a22o1-uat-scenarios), [792a22o1e The payload has no hint without offers](../product/curate.md#792a22o1-uat-scenarios).

Rationale: an agent that reads only the first part of a large payload must still see that offers wait.

#### 792b23c10a Scenarios

```gherkin
Scenario: 792b23c10a1 The flag and hint lead a large payload
  Given a payload of 60 items of 2000 bytes each and pending offers
  When it is rendered
  Then pending_offers and pending_offers_hint precede items, and the first 1500 bytes contain the hint
```
