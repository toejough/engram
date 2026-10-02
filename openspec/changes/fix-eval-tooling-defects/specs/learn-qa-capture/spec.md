## MODIFIED Requirements

### Requirement: Question and answer notes SHALL be written as a linked pair

The system SHALL create both notes in a single operation with atomic-ish semantics: the question note is written first, then the answer note. On answer-write failure, the orphaned question note SHALL be removed (best-effort cleanup). Each pair SHALL share a date-stamped slug prefix in their filenames.

#### Scenario: Successful pair capture

- **WHEN** `engram learn qa --slug <kebab> --question="<text>" --answer="<body>" --source="<provenance>" --certainty <level> [--contributors ...]` is invoked
- **THEN** two notes are written atomically: `qa.<YYYY-MM-DD>.<slug>.q.md` (question) and `qa.<YYYY-MM-DD>.<slug>.a.md` (answer), both under the vault root

