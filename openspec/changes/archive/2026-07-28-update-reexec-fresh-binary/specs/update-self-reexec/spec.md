## ADDED Requirements

### Requirement: Sync phase runs in the freshly installed binary
When `engram update`'s binary install step completes without error, it SHALL re-exec the installed binary (at the resolved installed path, not the running process's path) to perform all subsequent phases — planning, harness sync, and vault/index checks — and the parent process SHALL exit with the re-execed process's exit code without performing those phases itself.

#### Scenario: Post-install phases execute in the new binary
- **WHEN** `engram update` runs, the install step succeeds, and re-exec succeeds
- **THEN** the parent performs no planning, sync, or vault checks after the install, the re-execed process performs them, and the command's exit code is the re-execed process's exit code

#### Scenario: Re-exec targets the installed path
- **WHEN** the running `engram` process's own path differs from the resolved installed binary path
- **THEN** the re-exec invokes the resolved installed binary path

### Requirement: Loop-guard sentinel prevents repeated install
The re-exec invocation SHALL carry a sentinel in its environment (`ENGRAM_UPDATE_REEXEC=1`) meaning "install already completed — sync only." A `engram update` run that observes the sentinel MUST NOT perform source resolution or binary install and MUST NOT re-exec again; it SHALL proceed directly to planning, sync, and checks.

#### Scenario: Sentinel run skips install and does not re-exec
- **WHEN** `engram update` runs with the loop-guard sentinel set in its environment
- **THEN** no `go install` is invoked, no further process is spawned, and sync and checks complete in-process

#### Scenario: Sentinel is set on the re-exec invocation
- **WHEN** the parent re-execs after a successful install
- **THEN** the child invocation's environment contains the loop-guard sentinel and its arguments preserve the original update arguments

### Requirement: Dry-run never installs or re-execs
`engram update --dry-run` SHALL skip binary install and MUST NOT re-exec; the dry-run preview completes in the invoking process.

#### Scenario: Dry-run stays in-process
- **WHEN** `engram update --dry-run` runs
- **THEN** no binary is installed, no process is spawned, and the dry-run report is produced by the invoking process

### Requirement: In-process fallback on re-exec failure
If spawning the installed binary fails, `engram update` SHALL complete planning, sync, and checks in-process with its current logic and SHALL record in the update report that re-exec failed and pre-update logic completed the run, including the failure reason.

#### Scenario: Missing installed binary falls back
- **WHEN** the install step reports success but the re-exec spawn fails
- **THEN** the update completes in-process, exits by the in-process result, and the report states that re-exec failed with the underlying error

### Requirement: Single coherent report across the re-exec boundary
The user SHALL see one coherent update run: install output attributed to the parent, and exactly one sync/check report produced by the re-execed process. The sync report MUST NOT be duplicated by the parent.

#### Scenario: No duplicated sync report
- **WHEN** an update installs and re-execs successfully
- **THEN** the combined output contains the install result once and the sync/check report once, produced by the re-execed process
