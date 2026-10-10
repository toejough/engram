# Write-Memory Worker + Capture Guards Specification

## Purpose

Residue awaiting later #792 nodes. The write-memory worker (handoff, composition, flags, verification) is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a13-added-write-memory`. What remains here waits for the nodes named below: the four kinds of explicit lessons for learn (792c7n1), and the Step 7 lessons audit for please (792c7q1). A decision-support addition in Step 7's lessons audit asks which existing artifact should have surfaced each lesson and suggests rewording stale notes before writing duplicates. This addition was originally committed in `agent-instructions/skills/please/SKILL.md` (662e50ba; still carried by that skill's Step 7 lessons audit); deployed as of 2026-07-27; unvalidated — no measurement supports it (`dev/eval/LEDGER.md#687-surprise-harvest`).
## Requirements
### Requirement: Learn skill SHALL capture four kinds of explicit lessons

Learn SHALL identify and capture corrections (user-corrected approaches), explicit save-requests, reversals (presented conclusions later overturned), and confirmed approaches (user-praised or self-validated behaviors). Each kind generates a distinct vault note via handoff to write-memory.

#### Scenario: Capture corrections

- **WHEN** the user corrects an approach mid-task
- **THEN** learn SHALL invoke write-memory with kind=feedback, capturing the situation, what was done wrong (behavior), why it was costly (impact), and what to do instead (action)

#### Scenario: Capture explicit save-requests

- **WHEN** the user says "remember this", "save that", or "note for next time"
- **THEN** learn SHALL invoke write-memory immediately with kind=fact, capturing the situation, subject, predicate, and object

#### Scenario: Capture self-discovered reversals

- **WHEN** a presented conclusion or design is later overturned (by the agent itself, a reviewer, or an instrument)
- **THEN** learn SHALL invoke write-memory with kind=feedback, capturing the situation (when this failure mode applies), what the original reasoning did wrong (behavior), what the reversal cost (impact), and the root cause guard (action)

#### Scenario: Capture confirmed approaches

- **WHEN** a specific, generalizable approach is validated (by user praise or observable outcome)
- **THEN** learn SHALL invoke write-memory with kind=feedback, capturing the situation (when this approach would apply again), what worked (behavior), confirming evidence (impact), and keep-doing trigger (action)

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

