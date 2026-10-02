## MODIFIED Requirements

### Requirement: `engram learn` SHALL support capturing runbook notes

The system SHALL accept a `runbook` capture path in `engram learn`, writing a single note with frontmatter `type: runbook`, distinct from `fact`, `feedback`, and `qa-question`/`qa-answer`.

#### Scenario: Successful runbook capture

- **WHEN** `engram learn runbook --slug <kebab> --situation="<when to use this runbook>" --done-when="<what should be true when done>" --body="<numbered steps>" --source="<provenance>" --position <top|continuation|sibling> [--target <luhmann-id>] [--contributors ...]` is invoked
- **THEN** one note is written: `<luhmann-id>.<YYYY-MM-DD>.<slug>.md`, under the vault root, with `type: runbook` in frontmatter — the same filename scheme `fact`/`feedback` use

### Requirement: Runbook notes MAY carry a `red_flags` field of task-specific failure modes

A runbook note SHALL support an optional frontmatter field `red_flags` (a list of strings), each naming a condition specific to this procedure that a general "follow the steps" rule would not catch (e.g. "filter-branch on all refs sweeps the backup branch"). `engram learn runbook` SHALL accept a repeatable `--red-flag <text>` flag that populates it. Entries SHALL be task-specific; the general behavioral floor (shim) is not restated here.

#### Scenario: Runbook captured with red flags
- **WHEN** `engram learn runbook … --red-flag="<text A>" --red-flag="<text B>"` is invoked
- **THEN** the written note's frontmatter contains `red_flags:` with the two entries in order, and the note otherwise matches the runbook schema

#### Scenario: Runbook captured without red flags
- **WHEN** `engram learn runbook` is invoked with no `--red-flag`
- **THEN** the note is written with no `red_flags` field and no error

### Requirement: Runbook notes MAY carry a `triggers` field populated by `--trigger`
A runbook note SHALL support an optional frontmatter field `triggers` (a list of strings). `engram learn runbook` SHALL accept a repeatable `--trigger <text>` flag that populates it in order, wired through the same capture pipeline as `--red-flag` (Luhmann disposition, lock, embed, vocab). Full matching semantics are specified in capability `runbook-lexical-triggers`.

#### Scenario: Runbook captured with triggers
- **WHEN** `engram learn runbook … --trigger="<cue A>" --trigger="<cue B>"` is invoked
- **THEN** the written note's frontmatter contains `triggers:` with the two entries in order, after `red_flags` if present

#### Scenario: Runbook captured without triggers
- **WHEN** `engram learn runbook` is invoked with no `--trigger`
- **THEN** the note is written with no `triggers` field and no error
