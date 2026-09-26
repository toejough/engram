## MODIFIED Requirements

### Requirement: Write-memory handoff SHALL carry placement fields

The write-memory skill's handoff contract (`agent-instructions/skills/write-memory/SKILL.md`; its vault runbook is the registered, derived mirror of that skill) — the field set `learn` passes when invoking write-memory — SHALL specify **kind** (`fact|feedback|qa|runbook`), the kind's content fields, **source** (provenance string), optional **chunk-sources**, optional **supersedes** (`basename|type|claim`), optional **target** (for positioning within a topic/MOC), and optional **tags** (`vocab/<term>` or `project:<slug>` or user-supplied `<family>/<value>`).

#### Scenario: Continuation handoff composes the correct command
- **WHEN** learn hands off a note with `position=continuation` and `target=1a`
- **THEN** write-memory composes `engram learn <kind> ... --position continuation --target 1a`

#### Scenario: Top-level handoff omits target
- **WHEN** learn hands off a note with `position=top`
- **THEN** write-memory composes `engram learn <kind> ... --position top` with no `--target` flag
