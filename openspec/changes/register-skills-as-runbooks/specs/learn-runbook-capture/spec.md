## ADDED Requirements

### Requirement: Derived runbooks SHALL participate like captured runbooks but be owned by their skill
A runbook note derived by skill registration SHALL participate in the Luhmann hierarchy, embedding, vocab tagging, retrieval, triggers, and the shim's follow-frame exactly as a captured runbook does. Its situation, done_when, red_flags, triggers, and body SHALL be owned by the source skill file: registration overwrites them, and the note's body preamble SHALL say so.

#### Scenario: Derived runbook surfaces like a captured one
- **WHEN** `engram query --text` contains one of a derived runbook's triggers
- **THEN** the derived note surfaces first with `trigger` provenance, exactly as a captured runbook with the same triggers would

#### Scenario: Body preamble marks ownership
- **WHEN** `engram show` prints a derived runbook
- **THEN** the body begins with a line stating the note is derived from `<skill path>` and that edits belong in the skill file
