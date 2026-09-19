# production-guidance-activation Specification

## Purpose
TBD - created by archiving change activate-shim-guidance. Update Purpose after archive.
## Requirements
### Requirement: A production harness config SHALL import shim.md alongside the existing engram guidance imports
A harness's own config file (`CLAUDE.md` for Claude Code, `AGENTS.md` for Pi) that already imports
`recall.md`, `delegate.md`, and `learn.md` SHALL also import `shim.md`, additively — the three
existing imports MUST remain present. This activates `shim.md`'s runbook-follow-frame for real
sessions without removing the coverage `recall.md`/`delegate.md`/`learn.md` provide for content not
yet promoted to runbooks (per issue #760, separately scoped).

#### Scenario: A harness config with the three existing imports gains shim.md
- **WHEN** a harness's config file imports `recall.md`, `delegate.md`, and `learn.md` via engram's
  standard `@`-import convention, and that harness has not yet undergone the full recall-replacement
  effort (#760)
- **THEN** the config file also imports `shim.md`, and all three pre-existing imports remain
  unchanged

#### Scenario: Full replacement is not performed before recall/learn/write-memory are promoted
- **WHEN** `recall.md`, `delegate.md`, and `learn.md`'s corresponding skills (`recall`, `learn`,
  `write-memory`) have not yet been promoted to production runbooks
- **THEN** `recall.md`, `delegate.md`, and `learn.md` remain imported in the harness config; `shim.md`
  is added, not substituted for them

