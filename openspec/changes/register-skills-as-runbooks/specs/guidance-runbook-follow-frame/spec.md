## ADDED Requirements

### Requirement: The follow-frame SHALL work on runbooks received from a parent vault
When a `kind: runbook` item tagged `from_parent` is matched, the shim's follow-frame obligations (announce, restate, `done_when`, `red_flags`, transitive wikilink fetch) SHALL apply unchanged, and the bare `engram show <basename>` the frame instructs SHALL resolve the note (via the `show` parent fallback in capability `vault-merged-recall`) without the agent needing to know the note's origin.

#### Scenario: Following a shared runbook
- **WHEN** a client session with `ENGRAM_PARENT` set surfaces a runbook that exists only in the parent vault and runs `engram show <basename>`
- **THEN** the full note is returned and the agent restates its steps as its plan

### Requirement: Registered skills SHALL be the runbooks the shim finds, not replacements for skills
The shim's bootstrap query and follow-frame SHALL operate on runbook notes derived from registered skills (capability `skill-runbook-registration`) in addition to captured runbooks; a skill remaining installed as a skill SHALL NOT be treated as a reason to skip the follow-frame when its derived note is matched.

#### Scenario: Skill installed and registered
- **WHEN** `curate` is deployed as a skill AND its derived runbook is matched by the first-action query
- **THEN** the agent applies the follow-frame to the matched runbook (announce, restate, red_flags, done_when) rather than deferring to the skill's own description-based firing
