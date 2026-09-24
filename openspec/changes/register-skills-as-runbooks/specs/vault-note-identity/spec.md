## ADDED Requirements

### Requirement: Derived notes SHALL carry a `registered_from` identity field
A runbook note derived from a registered skill SHALL carry `registered_from: <skill-name>/<runbook-slug>` in its frontmatter. The field SHALL be preserved by `engram amend` (amend re-stamps repo/user/vault but SHALL NOT drop or alter `registered_from`) and SHALL be the sole key registration uses to locate the note.

#### Scenario: Amend preserves the identity field
- **WHEN** `engram amend` rewrites a note carrying `registered_from`
- **THEN** the field is unchanged in the written note

#### Scenario: Non-derived notes have no such field
- **WHEN** a note is captured by `engram learn` outside registration
- **THEN** it carries no `registered_from` field
