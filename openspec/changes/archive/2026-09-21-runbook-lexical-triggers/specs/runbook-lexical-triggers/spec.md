## Purpose

Lets a runbook author declare literal cue strings (`triggers`) that, when they appear verbatim in the user's raw message passed as `engram query --text`, surface that runbook first in `items[]` regardless of semantic similarity — a single lexical exception to purely semantic ranking, added because paraphrased query phrases measurably dropped the cue. Why: `docs/architecture/adr.md` ADR-0026 amendment.

## ADDED Requirements

### Requirement: Runbook notes MAY carry a `triggers` field of literal cue strings
A runbook note SHALL support an optional frontmatter field `triggers` (a list of strings), each a literal cue that, when present in the user's raw message, identifies this runbook as applicable. `engram learn runbook` SHALL accept a repeatable `--trigger <text>` flag that populates it in order; `engram amend` SHALL accept the same flag with replace-whole semantics. A trigger SHALL be rejected at write time if it is empty, whitespace-only, or shorter than 3 characters after trimming.

#### Scenario: Runbook captured with triggers
- **WHEN** `engram learn runbook … --trigger "/please" --trigger "take this end-to-end"` is invoked
- **THEN** the written note's frontmatter contains `triggers:` with the two entries in order, and the note otherwise matches the runbook schema

#### Scenario: Runbook captured without triggers
- **WHEN** `engram learn runbook` is invoked with no `--trigger`
- **THEN** the note is written with no `triggers` field and no error

#### Scenario: Invalid trigger rejected
- **WHEN** `engram learn runbook … --trigger "  "` or `--trigger "ab"` is invoked
- **THEN** the command fails with an error naming the invalid trigger and writes nothing

#### Scenario: Amend replaces the whole list
- **WHEN** `engram amend <basename> --trigger "/please"` is invoked on a runbook that had two triggers
- **THEN** the note's `triggers:` contains exactly `/please`; the note is re-stamped; the sidecar is not stale (content hash excludes `triggers`)

### Requirement: `engram query` SHALL accept the user's raw message via `--text`
`engram query` SHALL accept an optional `--text <string>` argument carrying the user's message verbatim. `--text` SHALL NOT be embedded or used for similarity matching. When `--text` is absent, trigger matching SHALL NOT run and the payload SHALL be identical to today's. `--text` without any `--phrase` SHALL be accepted (trigger-only lookup). The served query path SHALL round-trip `text` byte-identically up to 2 KB; longer text SHALL be silently truncated to 2048 bytes on a UTF-8 rune boundary (capability `vault-serve-api`).

#### Scenario: Query without --text is unchanged
- **WHEN** `engram query --phrase A --phrase B` is run against a vault whose runbooks carry triggers
- **THEN** the payload is byte-identical to the payload produced before this change for the same vault and phrases

#### Scenario: Served round-trip
- **WHEN** a client builds a served query with `--text` (2 KB or less) containing quotes, newlines, `&`, and non-ASCII characters
- **THEN** the server receives the identical string and produces the same payload as the local path

### Requirement: A trigger hit SHALL surface first in `items[]`, regardless of similarity
A runbook is a trigger hit when any of its `triggers` entries, compared case-insensitively after collapsing whitespace runs to a single space on both sides, is a substring of `--text`. Every trigger hit SHALL appear in the payload's top-level `items[]`, before all similarity-ranked items, with `trigger` listed among its `provenances`. Trigger hits SHALL be exempt from the relevance floor, the match-set cap, the matched-note floor, and `--limit`. A trigger hit that also matched by similarity SHALL keep its similarity score and its other provenance roles. Only `type: runbook` notes SHALL be trigger candidates.

#### Scenario: Hit with no similarity match
- **WHEN** `engram query --text "/please fix the flaky login test" --phrase "fixing a flaky test" --phrase "diagnosing intermittent test failures"` is run and a runbook with `triggers: ["/please"]` has no similarity match above the floor
- **THEN** that runbook is `items[0]` with `provenances: [trigger]` and `kind: runbook`

#### Scenario: Hit ranks above a stronger similarity match
- **WHEN** a `feedback` note scores 0.9 by similarity and a runbook scores 0.3 but is a trigger hit
- **THEN** the runbook precedes the feedback note in `items[]`

#### Scenario: Case and whitespace insensitivity
- **WHEN** `--text "PLEASE   take this end-to-end: rename X"` is run against `triggers: ["take this end-to-end"]`
- **THEN** the runbook is a trigger hit

#### Scenario: Non-runbook kinds never trigger
- **WHEN** a `fact` note's frontmatter contains a `triggers:` list matching `--text`
- **THEN** it receives no `trigger` provenance and ranks by similarity only

#### Scenario: --limit does not drop a hit
- **WHEN** `engram query --text … --limit 1` produces one trigger hit and several similarity items
- **THEN** `items[]` contains the trigger hit plus the top similarity item (hits do not consume the limit)

### Requirement: The shim SHALL pass the user's verbatim message as `--text`
The shim's first-action `engram query` block SHALL include `--text "<the user's message, word for word>"` alongside the two `--phrase` values, with the instruction that the text is pasted verbatim (first ~300 characters if long) and never rewritten.

#### Scenario: Shim-only agent passes the raw message
- **WHEN** an agent with the updated shim receives the message "/please rename the tally CLI's --out flag to --output"
- **THEN** its first `engram query` carries `--text` whose value contains that message verbatim up to whitespace differences, plus the two phrases

### Requirement: Trigger cues SHALL be specific, author-chosen strings
Guidance for authoring triggers (write-memory template) SHALL state that a trigger is a distinctive cue — a slash form or a multi-word phrase — never a lone common word, and that a trigger hit is a candidate the agent still judges against the runbook's own applicability text.

#### Scenario: Write-memory composes a runbook with triggers
- **WHEN** a parent skill hands off kind=runbook with `triggers: ["/please", "take this end-to-end"]`
- **THEN** write-memory appends `--trigger "/please" --trigger "take this end-to-end"` to the `engram learn runbook` command
