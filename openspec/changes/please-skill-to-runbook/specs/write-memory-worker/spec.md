## MODIFIED Requirements

### Requirement: Please Step 7 lessons audit SHALL map mechanical corpus findings to vault notes

The closing `/learn` in the please runbook's Step 7 (carried by the lessons-audit sub-runbook, reached by `[[basename]]` wikilink from the top runbook body) SHALL audit the cycle's mechanical corpus: every pre-registered STOP, every gate failure, every CORRECTION-class commit, and every mid-cycle escalation. Each item SHALL be mapped to an existing vault note or marked "no lesson: <why>". Unmapped items become reversal handoffs to learn's Step 2 kind 3.

#### Scenario: Lessons audit enumeration

- **WHEN** the closing learn runs after a `please` cycle
- **THEN** it SHALL enumerate every pre-registered STOP, gate FAIL verdict, CORRECTION-class commit, and user escalation from the cycle

#### Scenario: Vault mapping for lessons

- **WHEN** an enumerated mechanical item is about to be captured
- **THEN** the audit SHALL ask which existing artifact should have surfaced it first and map it to an existing vault note if one covers the situation
- **AND** if a note existed but did not surface at the moment it was needed, the audit SHALL check whether its `situation:` line matches how the moment actually presented
- **AND** if it does not match, the audit SHALL suggest rewording the note before writing a duplicate

#### Scenario: Vault note citations use wikilink syntax

- **WHEN** the lessons audit cites an existing vault note in a structured field
- **THEN** the citation SHALL be written as `[[note-basename]]` wikilink syntax, not plain text
