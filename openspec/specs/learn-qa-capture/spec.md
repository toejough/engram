# Q&A Memory Round-1 Specification

## Purpose

`engram learn qa` captures a question and its answer as a linked pair, making the answer retrieval-competitive while keeping the question out of the main search space. The answer competes like any other vault note, while the question remains reachable only through its paired answer note. This asymmetric design prevents question wording from hurting retrieval quality while maintaining full traceability. Why: `docs/architecture/adr.md` ADR-0012. Validation: `dev/eval/LEDGER.md#qa-arm-v-borderline` (round-3 premise check, round-1 capture itself shipped without dedicated eval row).

## Requirements

### Requirement: Question and answer notes SHALL be written as a linked pair

The system SHALL create both notes in a single operation with atomic-ish semantics: the question note is written first, then the answer note. On answer-write failure, the orphaned question note SHALL be removed (best-effort cleanup). Each pair SHALL share a date-stamped slug prefix in their filenames.

#### Scenario: Successful pair capture

- **WHEN** `engram learn qa --slug <kebab> --question "<text>" --answer "<body>" --source "<provenance>" --certainty <level> [--contributors ...]` is invoked
- **THEN** two notes are written atomically: `qa.<YYYY-MM-DD>.<slug>.q.md` (question) and `qa.<YYYY-MM-DD>.<slug>.a.md` (answer), both under the vault root

#### Scenario: Partial failure recovery

- **WHEN** the answer note write fails after the question note is successfully written
- **THEN** the question note is removed (best-effort), and an error is returned naming both the A-write failure and the Q-removal attempt

### Requirement: Answer notes SHALL compete in retrieval; question notes MUST be excluded

Answer notes SHALL be ordinary vault notes that compete in the query pipeline's main matched set as synthesis notes. Question notes MUST be excluded from the query's main matched set via the `isQueryExcludedKind` filter and SHALL remain reachable only through their paired answer note's `answered_by` wikilink.

#### Scenario: Answer note inclusion

- **WHEN** a query is executed
- **THEN** answer notes (type: `qa-answer`) appear in the query's `candidate_l2s` list alongside other retrieval-competitive notes

#### Scenario: Question node exclusion

- **WHEN** a query is executed
- **THEN** question notes (type: `qa-question`) do NOT appear in the query's main matched set; they are excluded by `isQueryExcludedKind` check

### Requirement: Question and answer notes SHALL carry mutual linkages

Both note types SHALL use YAML frontmatter to reference each other and SHALL include hand-authored body markers to make the linkage visible.

#### Scenario: Question note structure

- **WHEN** a question note is rendered
- **THEN** its frontmatter includes `type: qa-question`, `date: <YYYY-MM-DD>`, `answered_by: <answer-basename>` (full basename without .md), and `source: <provenance>`
- **AND** its body ends with an `Answered by: [[<answer-basename>]]` wikilink marker

#### Scenario: Answer note structure

- **WHEN** an answer note is rendered
- **THEN** its frontmatter includes `type: qa-answer`, `date: <YYYY-MM-DD>`, `answers: <question-basename>` (full basename without .md), `certainty: <high|medium|low>`, `source: <provenance>`, and optional `contributors: [<basename>, ...]`
- **AND** its body ends with an `Answers: [[<question-basename>]]` wikilink marker
- **AND** if contributors were provided, a `Contributors: [[<contrib1>]], [[<contrib2>]], ...` marker follows the answers marker

### Requirement: QA embedding asymmetry — question note SHALL get embed only, answer note SHALL get embed + vocab

The question note SHALL be embedded to generate a dual-vector sidecar but receive NO vocab assignment. The answer note SHALL be embedded AND receive vocab term assignment. This asymmetry ensures the answer competes in retrieval while the question remains reachable only through its paired answer. Embed-on-write mechanics are owned by the vault-embed-on-write spec.

#### Scenario: Question note embedding without vocab

- **WHEN** a question note is written
- **THEN** it is passed to `autoEmbedNote` to generate a dual-vector sidecar (situation and body vectors)
- **AND** no vocab assignment pipeline is run on the question note

#### Scenario: Answer note embedding and vocab assignment

- **WHEN** an answer note is written
- **THEN** it is passed to `autoEmbedNote` to generate a dual-vector sidecar
- **AND** `applyVocabAssignmentAfterLearn` is invoked to assign vocabulary tags from the answer note's body vector

### Requirement: Contributors SHALL be validated before write; certainty MUST default to medium

All arguments SHALL be validated before acquiring the vault lock. Contributors SHALL be checked against existing vault members. Certainty level MUST default to `medium` when not provided and SHALL be one of `high`, `medium`, or `low`.

#### Scenario: Contributor validation

- **WHEN** `--contributors <basename>` flags are provided
- **THEN** each basename is validated against the full list of .md filenames in the vault before any write occurs
- **AND** if any contributor is not found, the command returns an error and no notes are written

#### Scenario: Certainty validation and default

- **WHEN** the `--certainty` flag is omitted or set to empty string
- **THEN** the certainty defaults to `medium`
- **WHEN** `--certainty` is provided
- **THEN** it must be one of `high`, `medium`, or `low`, or an error is returned

