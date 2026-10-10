# update-self-reexec Specification

## Purpose

The product behavior of the self-update lifecycle (the hand-off to the freshly installed binary, the dry-run exemption, the in-process fallback and the single coherent report) is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a11-added-install-and-update` (792a11e, 792a11f). The requirements below keep the mechanism (the installed path, the loop-guard sentinel) for self-installer (792c7j2). Why: docs/architecture/adr.md ADR-0023.

## Requirements

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

