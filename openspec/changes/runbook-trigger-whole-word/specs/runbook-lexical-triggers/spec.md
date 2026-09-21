## MODIFIED Requirements

### Requirement: A trigger hit SHALL surface first in `items[]`, regardless of similarity
A runbook is a trigger hit when any of its `triggers` entries occurs in `--text` as a whole word, compared case-insensitively after collapsing whitespace runs to a single space on both sides. An occurrence is a whole-word match when, on each side where the trigger's own edge character is a letter or digit, the character immediately adjacent to the occurrence (if any) is not a letter or digit (Unicode-aware; underscore, hyphen, slash, and other punctuation are boundaries; the start or end of the text is a boundary). On a side where the trigger's own edge character is not a letter or digit, no boundary is required. Every occurrence SHALL be considered, so a later occurrence can match when an earlier one does not. Every trigger hit SHALL appear in the payload's top-level `items[]`, before all similarity-ranked items, with `trigger` listed among its `provenances`. Trigger hits SHALL be exempt from the relevance floor, the match-set cap, the matched-note floor, and `--limit`. A trigger hit that also matched by similarity SHALL keep its similarity score and its other provenance roles. Only `type: runbook` notes SHALL be trigger candidates.

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

#### Scenario: Bare word hits as a whole word
- **WHEN** `--text "please curate the offers"` or `--text "Curate!"` or `--text "(curate)"` is run against `triggers: ["curate"]`
- **THEN** the runbook is a trigger hit

#### Scenario: Bare word does not hit inside a longer word
- **WHEN** `--text "that is accurate"`, `"inaccurate"`, `"curated offers"`, or `"curates"` is run against `triggers: ["curate"]`
- **THEN** the runbook is not a trigger hit

#### Scenario: Slash form is unchanged
- **WHEN** `--text "/please fix it"` or `--text "do /please"` is run against `triggers: ["/please"]`
- **THEN** the runbook is a trigger hit, and `--text "/pleased"` is not

#### Scenario: Slash before a bare word is a boundary
- **WHEN** `--text "/curate"` is run against `triggers: ["curate"]`
- **THEN** the runbook is a trigger hit

#### Scenario: Boundary at the start and end of the text
- **WHEN** `--text "curate"` (the whole text) is run against `triggers: ["curate"]`
- **THEN** the runbook is a trigger hit

#### Scenario: A later occurrence can satisfy the boundary
- **WHEN** `--text "accurate, curate now"` is run against `triggers: ["curate"]`
- **THEN** the runbook is a trigger hit

#### Scenario: Unicode letters and digits are not boundaries
- **WHEN** `--text "écurate"`, `"curateé"`, or `"curate2 now"` is run against `triggers: ["curate"]`
- **THEN** the runbook is not a trigger hit

#### Scenario: Hyphen and underscore are boundaries
- **WHEN** `--text "re-curate-now"` or `--text "do_curate_it"` is run against `triggers: ["curate"]`
- **THEN** the runbook is a trigger hit
