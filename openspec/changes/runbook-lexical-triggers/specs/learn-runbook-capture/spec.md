## ADDED Requirements

### Requirement: Runbook notes MAY carry a `triggers` field populated by `--trigger`
A runbook note SHALL support an optional frontmatter field `triggers` (a list of strings). `engram learn runbook` SHALL accept a repeatable `--trigger <text>` flag that populates it in order, wired through the same capture pipeline as `--red-flag` (Luhmann disposition, lock, embed, vocab). Full matching semantics are specified in capability `runbook-lexical-triggers`.

#### Scenario: Runbook captured with triggers
- **WHEN** `engram learn runbook … --trigger "<cue A>" --trigger "<cue B>"` is invoked
- **THEN** the written note's frontmatter contains `triggers:` with the two entries in order, after `red_flags` if present

#### Scenario: Runbook captured without triggers
- **WHEN** `engram learn runbook` is invoked with no `--trigger`
- **THEN** the note is written with no `triggers` field and no error
