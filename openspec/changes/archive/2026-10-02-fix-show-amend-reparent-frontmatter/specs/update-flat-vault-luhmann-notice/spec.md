## MODIFIED Requirements

### Requirement: Update surfaces a one-line notice offering a fresh Luhmann re-eval
When `engram update` detects an all-top-level vault, it SHALL print a one-line notice pointing
the user at a fresh Luhmann disposition pass. The notice SHALL name the learn skill's batch mode as the way the candidate
payload's answers are produced. It SHALL NOT modify any note's ID.

#### Scenario: Notice printed on detection
- **WHEN** the update Report records the vault as all-top-level
- **THEN** `engram update`'s output includes a notice naming `engram update --reparent-luhmann`
  as the remedy command, and no note file is modified by plain `engram update` as a result

#### Scenario: Silent when not all-top-level
- **WHEN** the update Report does NOT record the vault as all-top-level
- **THEN** `engram update`'s output includes no Luhmann-branching notice

#### Scenario: Notice names the answering procedure
- **WHEN** the update Report records the vault as all-top-level
- **THEN** the notice names both `engram update --reparent-luhmann` and the learn skill's batch mode
