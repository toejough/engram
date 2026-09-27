## ADDED Requirements

### Requirement: Skill-note identity fields SHALL survive every frontmatter rewrite
A runbook note created, adopted, or refreshed by skill registration SHALL carry these fields:

- `skill_key`, its source-qualified key (capability `skill-runbook-registration`);
- `skill_source`, the home-relative resolved path it was last copied from.

`skill_hash`, `skill_key`, and `skill_source` SHALL be preserved unchanged by every path that rewrites a note's frontmatter without being registration: `engram amend` (including `--clear-pending`), `engram resituate`, Luhmann reparenting (`engram update --reparent-luhmann`), rename-and-rewrite of wikilinks, identity backfill, and vocab tag rewrites. Only registration SHALL change them.

A note captured by `engram learn` outside registration SHALL carry neither `skill_key` nor `skill_source`.

#### Scenario: Amend preserves skill_key and skill_source
- **WHEN** `engram amend` rewrites a note carrying `skill_key` and `skill_source`
- **THEN** both fields are unchanged in the written note

#### Scenario: Reparenting preserves the identity fields
- **WHEN** Luhmann reparenting renames a skill note carrying `skill_hash`, `skill_key`, and `skill_source`
- **THEN** all three fields are unchanged in the renamed note

#### Scenario: Wikilink rewrite preserves the identity fields
- **WHEN** a rename-and-rewrite pass rewrites a wikilink inside a skill note's frontmatter
- **THEN** its `skill_hash`, `skill_key`, and `skill_source` are unchanged

#### Scenario: Ordinary capture carries neither
- **WHEN** a note is captured by `engram learn` outside registration
- **THEN** it carries no `skill_key` and no `skill_source` field
