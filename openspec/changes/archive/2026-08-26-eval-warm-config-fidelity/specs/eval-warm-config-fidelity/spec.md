## ADDED Requirements

### Requirement: A WARM trial config SHALL be verified to carry the real skill sources it claims to grant

`build_warm_cfg` (or any function that constructs a "warm" / memory-equipped eval trial's Claude Code config) SHALL confirm, for each skill it installs, that the skill's source directory exists and contains a `SKILL.md` before treating the installation as successful. The function SHALL NOT rely on the copy operation's own success/failure alone — a copy that silently does nothing because its source path is wrong is indistinguishable from a successful no-op copy of an empty directory unless the source and destination content are independently checked.

#### Scenario: Successful skill installation is verified

- **WHEN** `build_warm_cfg` is called and the `recall` and `learn` skill sources exist with their `SKILL.md` files present
- **THEN** both skills' `SKILL.md` files exist under the destination config's skills directory after the call returns

#### Scenario: Post-copy destination content is confirmed

- **WHEN** the skill copy step completes
- **THEN** `build_warm_cfg` confirms `SKILL.md` exists at the destination path for each installed skill before returning successfully

### Requirement: A missing or empty skill source SHALL raise, never silently proceed

If a named skill's source directory does not exist, or exists but contains no `SKILL.md`, config construction SHALL raise an error identifying the specific missing path and skill name. It SHALL NOT log a warning and continue, and it SHALL NOT silently produce a config with fewer skills than requested.

#### Scenario: Missing skill source raises

- **WHEN** `build_warm_cfg` is called and a named skill's source directory does not exist at the expected path
- **THEN** the call raises an error, the error message names the specific missing skill and the path that was checked, and no partial config is left in a state that claims success

#### Scenario: Empty skill source raises

- **WHEN** `build_warm_cfg` is called and a named skill's source directory exists but contains no `SKILL.md`
- **THEN** the call raises an error identifying the skill and the empty/invalid source path

### Requirement: The skill source path SHALL be derived relative to the eval harness's own file location, not hardcoded to an absolute path

The repo root used to locate skill sources SHALL be computed from the calling module's own file location (e.g. `os.path.dirname(__file__)` plus the fixed relative offset to the repo root), never hardcoded as a machine- or session-specific absolute path. A hardcoded absolute path is exactly the fragility class that allowed this requirement's violation to go undetected for a month.

#### Scenario: Path resolves correctly independent of working directory or hardcoded overrides

- **WHEN** the eval harness computes the repo root used to locate skill sources
- **THEN** the computation is derived from the calling file's own location on disk, with no hardcoded absolute path override present in the source
