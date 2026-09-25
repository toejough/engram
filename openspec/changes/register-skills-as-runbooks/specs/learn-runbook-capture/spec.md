## ADDED Requirements

### Requirement: Skill runbook notes SHALL participate like captured runbooks, with the body owned by the skill
A runbook note mirroring a skill SHALL participate in the Luhmann hierarchy, embedding, vocab tagging, retrieval, triggers, and the shim's follow-frame exactly as a captured runbook does. Its `situation`, `done_when`, `red_flags`, and `triggers` SHALL be authored on the note and edited with `engram amend` like any runbook's. Its body SHALL mirror the skill's `SKILL.md`: an accepted refresh replaces it, and the body SHALL begin with a one-line preamble stating it mirrors `<skill path>` and that procedure edits belong in the skill file.

#### Scenario: Skill note surfaces like a captured one
- **WHEN** `engram query --text` contains one of a (non-pending) skill note's triggers
- **THEN** the note surfaces first with `trigger` provenance, exactly as a captured runbook with the same triggers would

#### Scenario: Body preamble marks the source
- **WHEN** `engram show` prints a skill runbook note
- **THEN** the body begins with a line naming the skill file it mirrors

#### Scenario: Authored fields survive a refresh
- **WHEN** `engram amend` changes a skill note's `red_flags` and a later refresh is accepted
- **THEN** the amended `red_flags` are still present
