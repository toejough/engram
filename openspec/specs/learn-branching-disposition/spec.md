## Purpose

Restores canonical Luhmann/zettelkasten branching disposition (sibling vs. child vs. fresh
top-level) to the `learn` skill's note-crystallization flow, so notes placed by `learn` are not
always flat top-level IDs.

## Requirements

### Requirement: Learn skill decides note placement before write-memory handoff
When the `learn` skill crystallizes a new vault note (Step 2), it SHALL decide the note's Luhmann
placement — `top`, `continuation`, or `sibling`, plus a `target` note ID when not `top` — before
invoking write-memory, using the ordered disposition test below.

#### Scenario: Note develops a sub-point of a note from this session
- **WHEN** the note being crystallized develops one specific sub-point raised inside a note that
  was written or recalled earlier in the same session
- **THEN** the learn skill selects `--position continuation --target <that note's ID>`

#### Scenario: Note continues the same line of thought at the same level
- **WHEN** the note being crystallized continues or extends the same overall thought as a note
  from this session, without being a sub-point of it
- **THEN** the learn skill selects `--position sibling --target <that note's ID>`

#### Scenario: Note starts an unrelated thread
- **WHEN** the note being crystallized has no continuation or sibling relationship to any note
  written or recalled earlier in this session
- **THEN** the learn skill selects `--position top`

#### Scenario: Disposition test order
- **WHEN** a note could plausibly be read as either developing a sub-point or continuing the same
  thought as an existing in-session note
- **THEN** the learn skill checks continuation (sub-point) before sibling (same-level continuation)
  and applies the first test that matches

### Requirement: Disposition scope is limited to the current session
The learn skill's disposition decision SHALL consider only notes written or recalled earlier in
the current session as candidate continuation/sibling targets. It SHALL NOT perform a full-vault
search for a placement target.

#### Scenario: No in-session candidate exists
- **WHEN** no note has been written or recalled earlier in the current session
- **THEN** the learn skill selects `--position top` without searching the rest of the vault

### Requirement: Write-memory handoff carries placement fields
The write-memory skill's handoff contract SHALL accept `position` (`top`, `continuation`, or
`sibling`) and `target` (a Luhmann note ID, required when position is not `top`) from the calling
skill, and SHALL pass them through to the `engram learn <kind>` command it composes.

#### Scenario: Continuation handoff composes the correct command
- **WHEN** learn hands off a note with `position=continuation` and `target=1a`
- **THEN** write-memory composes `engram learn <kind> ... --position continuation --target 1a`

#### Scenario: Top-level handoff omits target
- **WHEN** learn hands off a note with `position=top`
- **THEN** write-memory composes `engram learn <kind> ... --position top` with no `--target` flag
