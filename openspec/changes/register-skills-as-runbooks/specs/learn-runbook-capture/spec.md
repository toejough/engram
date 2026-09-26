## ADDED Requirements

### Requirement: Skill runbook notes SHALL participate like captured runbooks, with the body owned by the skill
A runbook note mirroring a skill SHALL participate in the Luhmann hierarchy, embedding, vocab tagging, retrieval, triggers, and the shim's follow-frame exactly as a captured runbook does. Its `situation`, `done_when`, `red_flags`, and `triggers` SHALL be authored on the note and edited with `engram amend` like any runbook's. Its body SHALL mirror the skill's `SKILL.md`: an accepted refresh replaces it, and the body SHALL begin with a one-line preamble stating it mirrors `<skill path>` and that procedure edits belong in the skill file.

#### Scenario: Skill note surfaces like a captured one
- **WHEN** `engram query --text` contains one of a (non-pending) skill note's triggers
- **THEN** the note surfaces first with `trigger` provenance, exactly as a captured runbook with the same triggers would

#### Scenario: Body preamble marks the source
- **WHEN** `engram show` prints a skill runbook note
- **THEN** the body begins with a line naming the skill file it mirrors

## MODIFIED Requirements

### Requirement: `write-memory` SHALL compose and execute runbook-note writes

The write-memory skill (`agent-instructions/skills/write-memory/SKILL.md`; its vault runbook is the registered, derived mirror of that skill per capability `skill-runbook-registration`) SHALL compose the `engram learn runbook` command from fields handed off by `learn` (slug, situation, done_when, body/steps, source, target/position disposition, optional contributors, optional red_flags list), execute it, verify the result, and report the written note path — consistent with how it handles `fact` and `feedback` handoffs. Each handed-off red flag becomes one `--red-flag` argument.

#### Scenario: write-memory executes a runbook handoff

- **WHEN** `learn` hands off a confirmed runbook (slug, situation, done_when, body, source, position, optional target, optional red_flags) to the write-memory skill
- **THEN** the write-memory skill's steps compose and run the corresponding `engram learn runbook` command, with one `--red-flag` per handed-off entry, and report the written note path

#### Scenario: Authored fields survive a refresh
- **WHEN** `engram amend` changes a skill note's `red_flags` and a later refresh is accepted
- **THEN** the amended `red_flags` are still present
