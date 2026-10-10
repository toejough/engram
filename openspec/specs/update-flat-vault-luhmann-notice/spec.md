## Purpose

The notice `engram update` prints is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a10a-the-flat-vault-notice` (792a10a). The detection below waits for note-reparenting (792c7i2).


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
