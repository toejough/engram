## MODIFIED Requirements

### Requirement: Write-memory handoff carries placement fields

The write-memory runbook's handoff contract — carried by a vault note of type `runbook`, reached by basename/wikilink from `learn`'s own instructions rather than by `engram query`, since write-memory has no trigger of its own — SHALL accept `position` (`top`, `continuation`, or `sibling`) and `target` (a Luhmann note ID, required when position is not `top`) from the calling skill, and SHALL pass them through to the `engram learn <kind>` command it composes.

#### Scenario: Continuation handoff composes the correct command
- **WHEN** learn hands off a note with `position=continuation` and `target=1a`
- **THEN** write-memory composes `engram learn <kind> ... --position continuation --target 1a`

#### Scenario: Top-level handoff omits target
- **WHEN** learn hands off a note with `position=top`
- **THEN** write-memory composes `engram learn <kind> ... --position top` with no `--target` flag
