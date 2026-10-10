# Recency Ranking - Technical Spec

## 792b21a Orientation

- Ownership: Repository component (the wall clock is supplied by the caller).
- Responsibility: turns time into ranking weight and orders chunks by recency. Given an age in days and
  a position within a session it returns the factor a score is multiplied by; given a note's dates it
  returns the note's age; given a note's text it reads the creation date; given a chunk's anchor it
  reads the turn number; and given chunks it picks the newest. It does no input or output and reads
  no clock.
- Current realization (informational): `recencyMultiplier`, `defaultRecencyParams`, `noteAgeDays`,
  `parseCreatedFromNote`, `parseTurnN`, `maxTurnBySource` and `newestChunkItems` in
  `internal/cli/recency.go`; the query engine applies the factor to note and chunk scores. The band
  helpers `fillRecencyBand`, `spliceRecent` and `mostRecentlyUsedNoteItems` have no production caller
  and are not specified.
- Qualification: responsibility, mapping ages and positions to a factor and ordering chunks is stated
  without naming any function; ownership, the query engine decides which scores to scale and where the
  clock comes from, while this component owns the curve, the age rules and the ordering; independent
  verification, it is tested directly on numbers, note texts and chunk lists; stability, the contract
  survives any change to what is scored; necessity, recency scaling ([792a7f Recency scaling](../product/memory-query.md#792a7f-recency-scaling)) and the newest chunks
  ([792a7o The newest chunks](../product/memory-query.md#792a7o-the-newest-chunks)) both rest on it.
- Supported capabilities: [memory-query](../product/memory-query.md).

Rationale: operators work in monthly cycles, so a short half-life would suppress legitimate older
content. A 60-day half-life is a soft tilt (2 weeks 0.85, 1 month 0.71, 2 months 0.50, 4 months 0.25)
that slightly favours recent content without drowning relevance; the in-suite experiment
`recency_eval_test` swept half-lives of 1, 7, 30 and 60 days over a pool of chunks of every age and
chose 60. The re-rank is only a tilt: the guarantee that fresh content reaches the reader comes from
the newest chunks ([792b21c5 Newest chunks](#792b21c5-newest-chunks)), picked by age whatever their cosine. A note's age counts from when it
last proved useful, not from when it was written, so a note that keeps being used stays fresh and one
that is never used fades.

## 792b21c Behavior Requirements

### 792b21c1 Recency factor

Requirement: [792b21c1 Recency factor](#792b21c1-recency-factor) The recency-ranking MUST return, for an age in days of zero or more, a half-life
in days, a tail weight and a position in a session from 0 to 1, the factor
2^(-age / half-life) × (1 + tail weight × position), so that the factor is 1 at age 0 and position 0,
halves with each half-life of age and gains the tail weight at position 1. The weights used to rank
MUST be a half-life of 60 days and a tail weight of 0.2.

Supports: [792a7f1a An old chunk is scaled down](../product/memory-query.md#792a7f1-uat-scenarios), [792a7f1b A long-unused note is scaled down](../product/memory-query.md#792a7f1-uat-scenarios), [792a7f1e A late turn is lifted](../product/memory-query.md#792a7f1-uat-scenarios).

#### 792b21c1a Scenarios

```gherkin
Scenario Outline: 792b21c1a1 Factors for an age and a position
  Given a half-life of 3 days and a tail weight of 0.2
  When the factor is taken for age <age> and position <position>
  Then it is <factor>

  Examples:
    | age | position | factor           |
    | 0   | 0        | 1.0              |
    | 3   | 0        | 0.5              |
    | 0   | 1        | 1.2              |
    | 6   | 0        | less than at 3   |

Scenario: 792b21c1a2 Production ranking uses a 60-day half-life
  Given the weights the query ranks with
  When they are read
  Then the half-life is 60 days and the tail weight is 0.2
```

### 792b21c2 Note age

Requirement: [792b21c2 Note age](#792b21c2-note-age) The recency-ranking MUST give a note's age in days at a given time as the time
since the date in its sidecar's `last_used`, or, when that is empty, since the date in its `created`
field, reading a `YYYY-MM-DD` date as midnight UTC, and MUST give an age of 0 when both are empty, when
the date does not read, and when it is in the future.

Supports: [792a7f1b A long-unused note is scaled down](../product/memory-query.md#792a7f1-uat-scenarios), [792a7f1d A note that was never used is not scaled](../product/memory-query.md#792a7f1-uat-scenarios).

#### 792b21c2a Scenarios

```gherkin
Scenario Outline: 792b21c2a1 Ages from the two dates
  Given the time 2026-06-17 12:00 UTC
  When the age is taken for last_used <last_used> and created <created>
  Then it is <age> days

  Examples:
    | last_used  | created    | age |
    | 2026-06-15 | 2026-01-01 | 2.5 |
    |            | 2026-06-10 | 7.5 |
    |            |            | 0   |

Scenario: 792b21c2a2 An unreadable or future date is no age
  Given a created value of "2026-07-01" with its quotes, or a date after the time
  When the age is taken
  Then it is 0
```

### 792b21c3 Creation date of a note

Requirement: [792b21c3 Creation date of a note](#792b21c3-creation-date-of-a-note) The recency-ranking MUST read a note's creation date as the value of the
`created:` key at the start of a line in the note's frontmatter, trimmed and otherwise exactly as
written, quotes included, from the note read as its LF form, and MUST read none when the note has no
frontmatter. It MUST NOT take a `created:` line that follows the closing fence, that is indented, or
that falls inside a block scalar, and an indented `---` inside a block scalar MUST NOT close the
frontmatter.

Supports: [792a7f1d A note that was never used is not scaled](../product/memory-query.md#792a7f1-uat-scenarios), [792a7zg1d The creation date of a CRLF note is read](../product/memory-query.md#792a7zg1-uat-scenarios).

Rationale: `engram learn` writes `created` quoted, and the quoted value cannot be read as a date, so
such a note reads as age 0 until it is used. Reading it without its quotes would put about 972 of the
1,006 notes of the real vault under decay at once, a ranking change that needs the measured check of
vault note 655 first (#774, controller ruling R33).

#### 792b21c3a Scenarios

```gherkin
Scenario: 792b21c3a1 The frontmatter date is read
  Given a note whose frontmatter has created: 2026-06-10, and a text without frontmatter
  When the creation date is read
  Then the first gives 2026-06-10 and the second gives nothing

Scenario: 792b21c3a2 A body line is not the date
  Given a note with created: 2026-06-10 in its frontmatter and created: 2020-01-01 after the closing fence, and a text of only a created: line
  When the creation date is read
  Then the first gives 2026-06-10 and the second gives nothing

Scenario: 792b21c3a3 An indented line is not the date
  Given a note whose situation is a block scalar holding the line "    created: 2020-01-01", then created: 2026-06-10
  When the creation date is read
  Then it is 2026-06-10

Scenario: 792b21c3a4 An indented fence does not end the frontmatter
  Given a note whose block scalar holds an indented --- line before created: 2026-06-10
  When the creation date is read
  Then it is 2026-06-10

Scenario: 792b21c3a5 A quoted date keeps its quotes
  Given a note with created: "2026-07-01"
  When the creation date is read
  Then it is "2026-07-01" with its quotes

Scenario: 792b21c3a6 A CRLF note is read as its LF form
  Given a note whose lines end in \r\n and whose created line is 2026-06-10
  When the creation date is read
  Then it is 2026-06-10
```

### 792b21c4 Turn positions

Requirement: [792b21c4 Turn positions](#792b21c4-turn-positions) The recency-ranking MUST read the turn number of an anchor of the form `turn-N`, with
N a non-negative whole number, and MUST read none from any other anchor. It MUST give for a list of
chunk records the highest turn number of each source, leaving out a source that has none.

Supports: [792a7f1e A late turn is lifted](../product/memory-query.md#792a7f1-uat-scenarios).

#### 792b21c4a Scenarios

```gherkin
Scenario Outline: 792b21c4a1 Turn numbers
  Given the anchor <anchor>
  When its turn number is read
  Then it is <turn>

  Examples:
    | anchor       | turn |
    | turn-0       | 0    |
    | turn-42      | 42   |
    | preamble     | none |
    | Some Heading | none |
    | turn-        | none |
    | turn-x       | none |

Scenario: 792b21c4a2 The highest turn of each source
  Given records at turn-0, turn-5 and preamble of a.jsonl, turn-2 of b.jsonl, and a heading of c.md
  When the highest turns are taken
  Then a.jsonl is 5, b.jsonl is 2 and c.md is absent
```

### 792b21c5 Newest chunks

Requirement: [792b21c5 Newest chunks](#792b21c5-newest-chunks) The recency-ranking MUST pick, from a list of scored chunks and a count, the chunks
with the latest `ingested_at`, newest first, ignoring their scores, with ties going to the later turn
number, a chunk without a time after every chunk that has one, and ties among those too going to the
later turn number. It MUST return nothing when the count is zero or less, and MUST give each chunk
picked as an item with its chunk ID as path, its text as content, its score, the given provenance and
the kind `chunk`.

Supports: [792a7o1a Recent chunks follow the matches](../product/memory-query.md#792a7o1-uat-scenarios), [792a7o1c The count can be set](../product/memory-query.md#792a7o1-uat-scenarios), [792a7o1f A chunk without a time sorts last](../product/memory-query.md#792a7o1-uat-scenarios).

#### 792b21c5a Scenarios

```gherkin
Scenario: 792b21c5a1 A count of zero picks nothing
  Given one chunk and a count of 0
  When the newest chunks are picked
  Then nothing is returned

Scenario: 792b21c5a2 Newest first, whatever the score
  Given chunks ingested 60 days, 14 days and 12 hours ago, with scores 0.9, 0.5 and 0.2
  When the newest two are picked
  Then they are the 12-hour chunk and then the 14-day chunk

Scenario: 792b21c5a3 A tie goes to the later turn
  Given three chunks of one source with the same ingested_at, at turn-2, turn-9 and turn-5
  When the newest two are picked
  Then they are turn-9 and then turn-5

Scenario: 792b21c5a4 A chunk without a time comes last
  Given a chunk with no ingested_at and a chunk with one
  When the newest one is picked
  Then it is the chunk with a time
```
