## MODIFIED Requirements

### Requirement: Amend re-stamps identity fields on every write
`engram amend` SHALL re-detect and overwrite a note's `repo:`, `user:`, and `vault:` frontmatter fields on every call that changes content or relations, using the same detection as `engram learn` (see the three requirements above), regardless of what the note previously held. One exception: when `user:` detection resolves to an empty string (both `git config user.email` and the OS username lookup fail), a re-stamping amend SHALL keep the note's existing non-empty `user:` value and SHALL print a warning to stderr naming the note and the failed detection, rather than writing `user: ""`. When the note also has no prior `user:`, the field is written as detected. `vault:` SHALL be resolved through the same flag, then `ENGRAM_VAULT_NAME`, then `"personal"` order as `engram learn`, so a re-stamping amend never writes `vault: ""`. The calls that change content or relations are any content flag, `--supersedes`, and `--chunk-source`. This is unlike `source:`, `project:`, `issue:`, and `tier:`, which continue to be preserved verbatim through amend. Bookkeeping amends SHALL NOT re-stamp these fields, so an accepted offer keeps its declared author. Those amends are:
- `--activate` alone;
- `--clear-pending`;
- `--discard --into` on the target note;
- the frontmatter-only link write that records an offer receipt.

#### Scenario: Amend from a different environment than the note's origin
- **WHEN** `engram amend` with a content, `--supersedes`, or `--chunk-source` flag rewrites a note whose `repo:`, `user:`, or `vault:` values differ from the environment `amend` is currently running in
- **THEN** the amended note's `repo:`, `user:`, and `vault:` values are overwritten with the current environment's freshly detected values

#### Scenario: Amend backfills missing identity fields
- **WHEN** `engram amend` with a content, `--supersedes`, or `--chunk-source` flag rewrites a note written before this capability existed (no `repo:`, `user:`, or `vault:` frontmatter fields present)
- **THEN** the amended note gains freshly detected `repo:`, `user:`, and `vault:` fields, same as any other amend

#### Scenario: Amend does not trigger re-embed for identity-only changes
- **WHEN** `engram amend` is invoked with only `--supersedes` or `--chunk-source` (no content-changing flag), and `repo:`/`user:`/`vault:` are the only identity fields that change
- **THEN** the note's vector sidecar is not re-embedded — identity re-stamping is a provenance-only change

#### Scenario: Accepting an offer keeps its author
- **WHEN** `engram amend --target O --clear-pending` runs on a pending offer whose `user:` is alice, from an environment whose detected user is bob
- **THEN** O's `user:` is still alice

#### Scenario: Empty user detection keeps the prior user
- **WHEN** a re-stamping `engram amend` runs on a note whose `user:` is alice, and user detection resolves to an empty string
- **THEN** the amended note's `user:` is still alice, a warning naming the note is printed to stderr, and the amend otherwise succeeds

#### Scenario: Amend with no vault name configured stamps the default
- **WHEN** a re-stamping `engram amend` runs with neither `--vault-name` nor `ENGRAM_VAULT_NAME` set
- **THEN** the amended note's `vault:` is `personal`, never an empty string
