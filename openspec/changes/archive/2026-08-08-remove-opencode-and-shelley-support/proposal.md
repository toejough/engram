## Why

Engram is narrowing its supported-harness scope to Claude Code and Pi (Joe, 2026-08-08). OpenCode is currently a first-class registered harness (`HarnessOpencode` in `supportedHarnesses()`, one of three) with live deploy code, ~30+ test references, and prose across README/CLAUDE.md/docs/GLOSSARY/architecture docs claiming current support. Shelley (exe.dev) was proposed as a fourth harness (#707) but never implemented — its footprint is limited to `docs/ROADMAP.md`. Both need removing so the codebase and docs stop claiming support engram no longer intends to provide. See issue #721.

Full removal, not a docs-only scrub: keeping OpenCode's harness registration and its sole-consumer commands-deploy mechanism (`CommandsRoot`/`CommandsTargetRel`/`agent-instructions/commands/`) alive with zero remaining consumers would be exactly the kind of dormant, unexercised machinery this repo's convention rejects (`git_is_the_fallback` — delete dead code promptly, mine git history if it's ever wanted back).

## What Changes

- Remove the OpenCode harness entirely from `internal/update`: `HarnessOpencode` constant, its `supportedHarnesses()` entry, detection/probe logic, and every OpenCode-specific test case and fixture across `internal/update` and `internal/cli`. Harness-count assumptions (3 hardcoded `HaveLen(3)`) become `HaveLen(2)`.
- Remove the commands-deploy mechanism entirely, since OpenCode was its sole consumer (Claude Code and Pi both have empty `CommandsTargetRel`): `CommandsRoot`/`CommandsTargetRel`/`CommandFiles` fields, `cmdRootFor`, `intendedCommandBasenames`, `planCommandCopies` and related planning/reporting code (`writeCommandRows`), and the `agent-instructions/commands/*.md` source files (4 files: learn, please, recall, route).
- Remove Shelley references from `docs/ROADMAP.md` (never implemented in code — no code-side removal needed).
- Update live-scope docs that currently claim OpenCode/commands support as current behavior: `README.md`, `CLAUDE.md`, `docs/GLOSSARY.md`, `docs/architecture/c1-system-context.md` (prose + mermaid diagram nodes/sequences), `docs/architecture/c2-containers.md`.
- Add a short dated addendum to ADR-0022 (status `Accepted`, still describes OpenCode as a live harness in its Decision section) noting the harness-scope narrowing, without rewriting the original decision prose — ADR-0009/ADR-0010 already describe the OpenCode transcript-backend removal in the past tense (`Superseded`) and need no change.
- Reword the OpenCode-flavored example task in `agent-instructions/skills/recall/tests/baseline-judgement-and-synthesis.md` to a harness-neutral example — the fixture uses OpenCode incidentally as a stand-in realistic task, not as a support claim.
- Update `openspec/specs/update-deploy-sync/spec.md`: drop "one per command file" from the symlink-materialization requirement, and rewrite/remove the manifest-mode scenario that uses a deployed command as its concrete example.
- Close/leave-closed related issues: #707 (Shelley), #667 (OpenCode guidance deploy), #644 (OpenCode SQLite ingest) — already closed as superseded by #721.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `update-deploy-sync`: the "Harness-visible paths are symlinks into the engram-owned root" requirement no longer covers command files (commands-deploy mechanism removed); the "First sync migrates previously-copied deployments" and manifest-mode requirement's example scenario is reworded off "a deployed command."

## Impact

- `internal/update/update.go`: remove `HarnessOpencode`, its `supportedHarnesses()` entry, `CommandsRoot`/`CommandsTargetRel`/`CommandFiles` fields and all command-planning functions (`cmdRootFor`, `intendedCommandBasenames`, `planCommandCopies`), detection/probe references to OpenCode.
- `internal/cli/update.go`: remove `writeCommandRows` and the "OpenCode) render nothing" comment/branch.
- Tests: `internal/update/{runner_test.go,symlink_test.go,migration_test.go,export_test.go,sync_test.go,update_test.go}`, `internal/cli/update_test.go` — remove OpenCode/command test cases, fix 3 hardcoded `HaveLen(3)` → `HaveLen(2)`.
- `agent-instructions/commands/*.md` — delete all 4 files (learn, please, recall, route command wrappers).
- Docs: `README.md`, `CLAUDE.md`, `docs/GLOSSARY.md`, `docs/architecture/{c1-system-context.md,c2-containers.md}`, `docs/ROADMAP.md`, `docs/architecture/adr.md` (ADR-0022 addendum only).
- `agent-instructions/skills/recall/tests/baseline-judgement-and-synthesis.md` — reword example task.
- `openspec/specs/update-deploy-sync/spec.md` — delta spec (this change).
- No production code paths outside `internal/update`/`internal/cli` reference OpenCode or the commands mechanism (verified by repo-wide grep during research).
