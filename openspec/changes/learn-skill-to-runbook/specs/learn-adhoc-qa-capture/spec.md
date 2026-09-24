## MODIFIED Requirements

### Requirement: Learn SHALL treat a self-answered Step 2 note as already-captured, not as an additional QA-capture trigger

When a question is posed and answered within the learn runbook's own Step 2 processing this
session, and the answer's traceability is provided by a Step 2 note the agent wrote this same
turn, that note SHALL be treated as the capture for that question. The agent SHALL NOT additionally
write a separate QA pair (Step 2.5) for the same question, even though the answer would otherwise
satisfy the substantive-answer bar (a `[[wikilink]]` or "crystallized a new vault note" disjunct).

#### Scenario: Question answered by the agent's own just-written Step 2 note

- **WHEN** a question is posed and answered within this session, and the answer's traceability is
  provided by a Step 2 note the agent wrote this same turn (regardless of whether that note also
  cites a `[[wikilink]]`)
- **THEN** the agent following the learn runbook SHALL skip Step 2.5 QA capture for that question — the Step 2 note is the capture

## ADDED Requirements

### Requirement: The self-referential duplicate guard SHALL survive the runbook conversion verbatim

The self-referential QA duplicate-guard rule (above) SHALL be carried into the learn runbook's Step 2.5 sub-runbook without regression to its pre-fix behavior — the fix predates and is unaffected by the carrier change from skill to runbook.

#### Scenario: Conversion fidelity check

- **WHEN** the learn skill's Step 2.5 content is converted into the Step 2.5 sub-runbook
- **THEN** a fresh-context reviewer confirms the duplicate-guard wording and its scenario are present, unaltered in meaning, in the converted sub-runbook
