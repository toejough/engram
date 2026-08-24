# Runbook Note Capture Specification

## Purpose

`engram learn runbook` captures a task-shaped "how to approach X" strategy as a fourth first-class
vault note kind, distinct from `fact`, `feedback`, and `qa`. Its schema answers exactly three
questions — when should you use it, what are the steps, what should be true when you're done —
and it is wired through the same capture pipeline as `fact`/`feedback` (Luhmann disposition, lock,
embed, vocab), not a bespoke standalone implementation. Why: `docs/architecture/adr.md` ADR-0026.

## Requirements

### Requirement: `engram learn` SHALL support capturing runbook notes

The system SHALL accept a `runbook` capture path in `engram learn`, writing a single note with frontmatter `type: runbook`, distinct from `fact`, `feedback`, and `qa-question`/`qa-answer`.

#### Scenario: Successful runbook capture

- **WHEN** `engram learn runbook --slug <kebab> --situation "<when to use this runbook>" --done-when "<what should be true when done>" --body "<numbered steps>" --source "<provenance>" --position <top|continuation|sibling> [--target <luhmann-id>] [--contributors ...]` is invoked
- **THEN** one note is written: `<luhmann-id>.<YYYY-MM-DD>.<slug>.md`, under the vault root, with `type: runbook` in frontmatter — the same filename scheme `fact`/`feedback` use

### Requirement: Runbook notes SHALL answer the three schema questions

A runbook note's schema answers exactly: when should you use it, what are the steps, and what should be true when you're done. Accordingly: `situation` (the when-to-use retrieval handle, phrased the way a future task would present) SHALL be required and SHALL be embedded as the note's situation vector exactly as for `fact`/`feedback` notes; `done_when` (ending expectations) SHALL be required; the body SHALL carry the steps and MAY include `[[wikilinks]]` to fact/feedback notes to read and consider. There SHALL be no `inputs`/argument-signature field and no `task_type` field.

#### Scenario: Runbook note frontmatter structure

- **WHEN** a runbook note is rendered
- **THEN** its frontmatter includes `type: runbook`, `date: <YYYY-MM-DD>`, `luhmann: <ID>`, `situation: <when-to-use>`, `done_when: <ending expectations>`, and `source: <provenance>`
- **AND** optional `contributors: [<basename>, ...]` is included if provided

#### Scenario: Missing done_when or situation is rejected

- **WHEN** `engram learn runbook` is invoked without `--situation` or without `--done-when`
- **THEN** the command errors naming the missing flag and no note is written

### Requirement: Runbook notes SHALL participate in the Luhmann hierarchy like fact/feedback

`engram learn runbook` SHALL require the same `--position <top|continuation|sibling>` disposition judgment (with `--target <luhmann-id>` for `continuation`/`sibling`) that `fact`/`feedback` capture already requires, and SHALL assign the resulting Luhmann ID via the same `nextLuhmannID` machinery. There SHALL be no kind-specific opt-out from Luhmann participation.

#### Scenario: Disposition judgment assigns a Luhmann ID

- **WHEN** `engram learn runbook` is invoked with `--position continuation --target <luhmann-id>`
- **THEN** the new note's `luhmann:` field is a child ID of `<luhmann-id>`, via the same `nextLuhmannID` logic `fact`/`feedback` capture uses

#### Scenario: Top-level disposition needs no target

- **WHEN** `engram learn runbook` is invoked with `--position top` and no `--target`
- **THEN** the new note receives a fresh top-level Luhmann ID and the command does not require a target

### Requirement: `write-memory` SHALL compose and execute runbook-note writes

The `write-memory` skill SHALL compose the `engram learn runbook` command from fields handed off by `learn` (slug, situation, done_when, body/steps, source, target/position disposition, optional contributors), execute it, verify the result, and report the written note path — consistent with how it handles `fact` and `feedback` handoffs.

#### Scenario: write-memory executes a runbook handoff

- **WHEN** `learn` hands off a confirmed runbook (slug, situation, done_when, body, source, position, optional target) to `write-memory`
- **THEN** `write-memory` composes and runs the corresponding `engram learn runbook` command and reports the written note path

### Requirement: Runbook notes SHALL receive embed + vocab like fact/feedback notes

A runbook note SHALL receive both embedding (dual-vector sidecar: situation vector + body vector) and vocab term assignment on write, since the full note content is the retrieval target (no excluded half, unlike the asymmetric `qa-question`/`qa-answer` split).

#### Scenario: Runbook note embed-on-write

- **WHEN** a runbook note is written via `engram learn runbook`
- **THEN** it receives a `.vec.json` sidecar and vocab term assignment, per the `vault-embed-on-write` spec's embed-on-write mechanics
