## ADDED Requirements

### Requirement: `engram learn` SHALL support capturing runbook notes

The system SHALL accept a `runbook` capture path in `engram learn`, writing a single note with frontmatter `type: runbook`, distinct from `fact`, `feedback`, and `qa-question`/`qa-answer`.

#### Scenario: Successful runbook capture

- **WHEN** `engram learn runbook --slug <kebab> --situation "<when to use this runbook>" --task-type <free-form-slug> --done-when "<what should be true when done>" --body "<numbered steps>" --source "<provenance>" [--contributors ...]` is invoked
- **THEN** one note is written: `runbook.<YYYY-MM-DD>.<slug>.md`, under the vault root, with `type: runbook` in frontmatter

### Requirement: Runbook notes SHALL answer the three schema questions

A runbook note's schema answers exactly: when should you use it, what are the steps, and what should be true when you're done. Accordingly: `situation` (the when-to-use retrieval handle, phrased the way a future task would present) SHALL be required and SHALL be embedded as the note's situation vector exactly as for `fact`/`feedback` notes; `done_when` (ending expectations) SHALL be required; the body SHALL carry the steps and MAY include `[[wikilinks]]` to fact/feedback notes to read and consider. There SHALL be no `inputs`/argument-signature field.

#### Scenario: Runbook note frontmatter structure

- **WHEN** a runbook note is rendered
- **THEN** its frontmatter includes `type: runbook`, `date: <YYYY-MM-DD>`, `situation: <when-to-use>`, `task_type: <free-form-slug>`, `done_when: <ending expectations>`, and `source: <provenance>`
- **AND** optional `contributors: [<basename>, ...]` is included if provided

#### Scenario: Missing done_when or situation is rejected

- **WHEN** `engram learn runbook` is invoked without `--situation` or without `--done-when`
- **THEN** the command errors naming the missing flag and no note is written

### Requirement: Runbook notes SHALL carry a free-form `task_type` field

The `task_type` field SHALL be a free-form kebab-case slug derived from the captured runbook's own subject at capture time, not selected from a fixed enum. Skill docs SHALL anchor this field with at least 2-3 concrete engram-specific examples (e.g. `running-eval-harnesses`, `releasing-go-modules`, `recall-moment-discovery`).

#### Scenario: Free-form task type accepted

- **WHEN** `engram learn runbook` is invoked with any kebab-case `--task-type` value
- **THEN** the value is stored verbatim in `task_type` with no enum validation

### Requirement: `write-memory` SHALL compose and execute runbook-note writes

The `write-memory` skill SHALL compose the `engram learn runbook` command from fields handed off by `learn` (slug, situation, task_type, done_when, body/steps, source, optional contributors), execute it, verify the result, and report the written note path — consistent with how it handles `fact` and `feedback` handoffs.

#### Scenario: write-memory executes a runbook handoff

- **WHEN** `learn` hands off a confirmed runbook (slug, situation, task_type, done_when, body, source) to `write-memory`
- **THEN** `write-memory` composes and runs the corresponding `engram learn runbook` command and reports the written note path

### Requirement: Runbook notes SHALL receive embed + vocab like fact/feedback notes

A runbook note SHALL receive both embedding (dual-vector sidecar: situation vector + body vector) and vocab term assignment on write, since the full note content is the retrieval target (no excluded half, unlike the asymmetric `qa-question`/`qa-answer` split).

#### Scenario: Runbook note embed-on-write

- **WHEN** a runbook note is written via `engram learn runbook`
- **THEN** it receives a `.vec.json` sidecar and vocab term assignment, per the `vault-embed-on-write` spec's embed-on-write mechanics
