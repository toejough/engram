## MODIFIED Requirements

### Requirement: Learn runbook decides note placement before write-memory handoff
When the `learn` runbook crystallizes a new vault note (Step 2), it SHALL decide the note's Luhmann
placement — `top`, `continuation`, or `sibling`, plus a `target` note ID when not `top` — before
invoking write-memory, using the ordered disposition test below.

#### Scenario: Note develops a sub-point of a note from this session
- **WHEN** the note being crystallized develops one specific sub-point raised inside a note that
  was written or recalled earlier in the same session
- **THEN** the agent following the learn runbook selects `--position continuation --target <that note's ID>`

#### Scenario: Note continues the same line of thought at the same level
- **WHEN** the note being crystallized continues or extends the same overall thought as a note
  written or recalled earlier in the same session, at the same level (not a sub-point of it)
- **THEN** the agent following the learn runbook selects `--position sibling --target <that note's ID>`

#### Scenario: No in-session candidate exists
- **WHEN** no note written or recalled earlier in this session is a plausible placement target
- **THEN** the agent following the learn runbook selects `--position top` with no target, and does not search the rest of the vault for a placement target

### Requirement: Disposition scope is limited to the current session

The placement test SHALL consider only notes written or recalled earlier in the current session — never a full-vault search for a placement target.

#### Scenario: Placement search is session-scoped

- **WHEN** deciding placement for a newly crystallized note
- **THEN** only in-session notes are considered as continuation/sibling candidates

### Requirement: Write-memory handoff carries placement fields

The handoff to write-memory SHALL include the decided `position` and, when not `top`, the `target` note ID.

#### Scenario: Handoff includes placement

- **WHEN** the learn runbook hands off a crystallized note to write-memory
- **THEN** the handoff includes `position` and, if `continuation` or `sibling`, `target`
