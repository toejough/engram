## Purpose

`engram update` detects when a vault holds only top-level Luhmann IDs — no
`continuation`/`sibling` (child) note anywhere — and surfaces a notice naming the remedy
command, `engram update --reparent-luhmann` (see `update-reparent-luhmann-batch`). A notice
that only points at a mechanism affecting future notes (the `learn` skill's per-capture
disposition step, restored by `restore-luhmann-branching-disposition`) is impotent for a vault
that has been flat since f620bfaf — this gives the notice a real remedy to point at.

## Requirements

### Requirement: Update detects an all-top-level Luhmann ID vault
`engram update` SHALL detect, on each run, whether every note in the vault has a top-level
(depth-1) Luhmann ID — i.e. no `continuation` or `sibling` (child) note exists anywhere in the
vault.

#### Scenario: Vault has only top-level notes
- **WHEN** every note in the vault parses to a Luhmann ID of depth 1
- **THEN** the update Report records that the vault is all-top-level

#### Scenario: Vault has at least one branched note
- **WHEN** at least one note in the vault parses to a Luhmann ID of depth greater than 1
- **THEN** the update Report does NOT record the vault as all-top-level

#### Scenario: Empty vault
- **WHEN** the vault contains no notes
- **THEN** the update Report does NOT record the vault as all-top-level (nothing to
  re-evaluate)

### Requirement: Update surfaces a one-line notice offering a fresh Luhmann re-eval
When `engram update` detects an all-top-level vault, it SHALL print a one-line notice pointing
the user at a fresh Luhmann disposition pass. It SHALL NOT modify any note's ID.

#### Scenario: Notice printed on detection
- **WHEN** the update Report records the vault as all-top-level
- **THEN** `engram update`'s output includes a notice naming `engram update --reparent-luhmann`
  as the remedy command, and no note file is modified by plain `engram update` as a result

#### Scenario: Silent when not all-top-level
- **WHEN** the update Report does NOT record the vault as all-top-level
- **THEN** `engram update`'s output includes no Luhmann-branching notice
