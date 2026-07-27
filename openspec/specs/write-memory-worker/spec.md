# Write-Memory Worker + Capture Guards Specification

## Purpose

The write-memory skill is a dedicated worker that executes vault-write commands handed off by parent skills (recall and learn), keeping the judgment of what to capture separate from the mechanics of writing it. Learn also captures self-discovered reversals and confirmed approaches as explicit lesson kinds, and `please` audits each cycle's mechanical corpus (failed gates, corrections, escalations) to catch lessons that went uncaptured. A decision-support addition in Step 7's lessons audit asks which existing artifact should have surfaced each lesson and suggests rewording stale notes before writing duplicates. This addition is committed in `agent-instructions/skills/please/SKILL.md` (commit 662e50ba); deployed as of 2026-07-27; unvalidated — no measurement supports it. Why: `docs/architecture/adr.md` ADR-0001. Validation: `dev/eval/LEDGER.md#write-memory-worker-fire-rates` (worker + G1/G2/G6); `dev/eval/LEDGER.md#687-surprise-harvest` (decision-support addition — committed 662e50ba; deployed 2026-07-27; unvalidated).

## Requirements

### Requirement: Write-memory worker SHALL accept handoff and execute vault writes

The worker is invoked by a parent skill that has already made the judgment (what to write and why). The worker SHALL receive a structured handoff containing kind (fact/feedback/qa), content fields, source, and optional chunk-sources, tags, and supersedes. The worker SHALL NOT re-judge the parent's decision and SHALL NOT decide whether to write.

#### Scenario: Worker receives handoff from parent skill

- **WHEN** a parent skill (recall, learn) invokes write-memory with a complete handoff (kind, required content fields, source)
- **THEN** the worker SHALL compose the corresponding `engram learn` command from the provided fields

#### Scenario: Worker rejects incomplete handoff

- **WHEN** required handoff fields are missing
- **THEN** the worker SHALL ask the parent skill (via in-session context) to provide the missing fields
- **AND** the worker SHALL NOT invent content on behalf of the parent

### Requirement: Write-memory worker SHALL handle three note kinds with distinct field sets

The worker SHALL support fact (subject/predicate/object), feedback (situation/behavior/impact/action), and qa (question/answer/contributors/certainty) kinds, each with their own field structure and engram command form.

#### Scenario: Fact note composition

- **WHEN** kind=fact is provided with situation, subject, predicate, object, and source
- **THEN** the worker SHALL compose: `engram learn fact --slug <kebab-slug> --position top --source "<source>" --situation "<situation>" --subject "<subject>" --predicate "<predicate>" --object "<object>" [--tag ...]`

#### Scenario: Feedback note composition

- **WHEN** kind=feedback is provided with situation, behavior, impact, action, and source
- **THEN** the worker SHALL compose: `engram learn feedback --slug <kebab-slug> --position top --source "<source>" --situation "<situation>" --behavior "<behavior>" --impact "<impact>" --action "<action>" [--tag ...]`

#### Scenario: QA note composition

- **WHEN** kind=qa is provided with slug, question, answer, contributors, certainty, and source
- **THEN** the worker SHALL compose: `engram learn qa --slug "<kebab-slug>" --question "<question>" --answer "<answer>" --source "<source>" --certainty <high|medium|low> [--contributors <basename> ...]`

### Requirement: Write-memory worker SHALL handle chunk-sources, tags, and supersedes

The worker SHALL append chunk-source flags for provenance tracking, tag flags for categorical tagging (fact/feedback only), and supersedes flags when the write corrects an existing note.

#### Scenario: Chunk-source provenance

- **WHEN** chunk-sources are provided (source#anchor format)
- **THEN** one `--chunk-source <source#anchor>` flag SHALL be appended per provided chunk ID

#### Scenario: Tag handling for fact and feedback

- **WHEN** tags are provided with fact or feedback kind
- **THEN** one `--tag <family>/<value>` flag SHALL be appended per tag
- **AND** tags SHALL be passed through exactly as provided; the worker SHALL NOT invent tags or write the `vocab/` namespace (vocab terms are auto-assigned by the binary)

#### Scenario: Tag rejection for QA

- **WHEN** tags are provided with kind=qa
- **THEN** the worker SHALL drop the tags and report: `tags dropped: qa takes no tag flags`

#### Scenario: Supersedes handling

- **WHEN** supersedes is provided (format: `<basename>|<type>|<claim>` with type one of updates/narrows/refutes)
- **THEN** `--supersedes "<basename>|<type>|<claim>"` flag SHALL be appended (repeatable)

### Requirement: Write-memory worker SHALL verify command execution and report results

The worker SHALL run the composed command, verify the result, and report the written note path(s) to the parent flow.

#### Scenario: Successful write

- **WHEN** the engram command executes successfully
- **THEN** the CLI prints the written note path(s)
- **AND** the worker SHALL report the path(s) to the parent in one line

#### Scenario: CLI error handling

- **WHEN** the command fails with a CLI error
- **THEN** the worker SHALL read the error, identify the named problem (missing flag, bad value, typo)
- **AND** the worker SHALL retry (max 2 retries total) with the corrected command

#### Scenario: Persistent failure

- **WHEN** the command fails after max retries
- **THEN** the worker SHALL report the exact command and the CLI error verbatim to the parent
- **AND** the worker SHALL NOT silently skip a handed-off write

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

The closing `/learn` in please Step 7 SHALL audit the cycle's mechanical corpus: every pre-registered STOP, every gate failure, every CORRECTION-class commit, and every mid-cycle escalation. Each item SHALL be mapped to an existing vault note or marked "no lesson: <why>". Unmapped items become reversal handoffs to learn's Step 2 kind 3.

#### Scenario: Lessons audit enumeration

- **WHEN** the closing learn runs after a `please` cycle
- **THEN** it SHALL enumerate every pre-registered STOP, gate FAIL verdict, CORRECTION-class commit, and user escalation from the cycle

#### Scenario: Vault mapping for lessons

- **WHEN** an enumerated mechanical item is about to be captured
- **THEN** the audit SHALL ask which existing artifact should have surfaced it first and map it to an existing vault note if one covers the situation
- **AND** if a note existed but did not surface at the moment it was needed, the audit SHALL check whether its `situation:` line matches how the moment actually presented
- **AND** if it does not match, the audit SHALL suggest rewording the note before writing a duplicate

