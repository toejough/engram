## ADDED Requirements

### Requirement: Skill notes SHALL carry `skill_key` and `skill_source` preserved like `skill_hash`
A runbook note created or adopted by skill registration SHALL carry `skill_key` (its source-qualified skill key, capability `skill-runbook-registration`) and `skill_source` (the home-relative resolved `SKILL.md` path it was last copied from). `engram amend` SHALL preserve both fields unchanged; only registration SHALL change them. A note captured by `engram learn` outside registration SHALL carry neither field.

#### Scenario: Amend preserves skill_key and skill_source
- **WHEN** `engram amend` rewrites a note carrying `skill_key` and `skill_source`
- **THEN** both fields are unchanged in the written note

#### Scenario: Ordinary capture carries neither
- **WHEN** a note is captured by `engram learn` outside registration
- **THEN** it carries no `skill_key` and no `skill_source` field
