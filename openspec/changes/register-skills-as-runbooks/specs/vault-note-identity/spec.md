## ADDED Requirements

### Requirement: Skill notes SHALL carry a `skill_hash` field
A runbook note mirroring a skill SHALL carry `skill_hash: <sha256 of the SKILL.md bytes its body was copied from>` in its frontmatter. The field SHALL be preserved by `engram amend` (amend re-stamps repo/user/vault but SHALL NOT drop or alter `skill_hash`); only registration SHALL change it.

#### Scenario: Amend preserves the hash
- **WHEN** `engram amend` rewrites a note carrying `skill_hash`
- **THEN** the field is unchanged in the written note

#### Scenario: Non-skill notes have no such field
- **WHEN** a note is captured by `engram learn` outside registration
- **THEN** it carries no `skill_hash` field
