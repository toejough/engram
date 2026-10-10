# update-local-install-safety

## Purpose

The product behavior (the local source's revision in the report, the refusal of a provable downgrade, `--allow-downgrade`, and the fail-open cases) is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a11-added-install-and-update` (792a11c, 792a11d). The requirements below keep the mechanism for self-installer (792c7j2).

## Requirements

### Requirement: Provable local-mode downgrades are refused without `--allow-downgrade`
Before running `go install` in local mode, if a binary already exists at the resolved install path, `engram update` SHALL capture that binary's embedded VCS revision (via its Go build info) and SHALL compare it against the module root's resolved revision using git ancestry. When the module root's revision is provably NOT a descendant of the installed binary's revision (both revisions are known to the module root's git history and the ancestry check fails), `engram update` SHALL refuse the install and report the detected downgrade, UNLESS `--allow-downgrade` was passed, in which case the install proceeds as if no check occurred.

#### Scenario: Provable downgrade is refused by default
- **WHEN** `engram update` runs in local mode, a binary is already installed at a revision the module root's git history contains as a proper ancestor of a later commit, and the module root's current HEAD is an earlier commit than the installed revision
- **THEN** the install is refused, the report states the provable downgrade, and the command exits nonzero

#### Scenario: `--allow-downgrade` bypasses the gate
- **WHEN** `engram update --allow-downgrade` runs under the same provable-downgrade conditions
- **THEN** the install proceeds and completes as it would with no gate present

### Requirement: Unprovable ancestry fails open
When the installed binary's revision cannot be determined (no prior binary at the install path, the binary predates Go's automatic VCS-embedding, or its embedded revision is unrecognized), or when the module root's git history cannot establish ancestry either way for a known installed revision (the revision is unknown to this clone, e.g. a shallow clone, a different repository, or divergent unmerged history), `engram update` SHALL proceed with the install exactly as it would with no downgrade check, while still reporting the resolved revision for visibility.

#### Scenario: First-ever install proceeds without a gate check
- **WHEN** `engram update` runs in local mode and no binary exists yet at the resolved install path
- **THEN** the install proceeds with no downgrade check performed

#### Scenario: Unknown installed revision fails open
- **WHEN** `engram update` runs in local mode and the installed binary's embedded revision is not a commit known to the module root's git history
- **THEN** the install proceeds, and the report shows the resolved revision without claiming a downgrade
