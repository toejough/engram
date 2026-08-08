## 1. Red: update test expectations ahead of removal

- [x] 1.1 In `internal/update/sync_test.go` and `internal/update/update_test.go`, change the 3 hardcoded `HaveLen(3)` harness-count assertions to `HaveLen(2)`. Run `targ test` and confirm these fail (RED) against current code.
- [x] 1.2 Remove OpenCode-specific test cases/fixtures from `internal/update/{runner_test.go,symlink_test.go,migration_test.go,export_test.go}` and `internal/cli/update_test.go`. Remove command-mechanism test cases from the same files. Confirm compile/test failures as expected against current code (RED — either compile errors from now-undefined helpers, or failing assertions).

## 2. Green: remove OpenCode harness and the commands-deploy mechanism

- [x] 2.1 In `internal/update/update.go`, remove `HarnessOpencode` and its `supportedHarnesses()` entry, and any detection/probe logic branching on it.
- [x] 2.2 In `internal/update/update.go`, remove `CommandsRoot`, `CommandsTargetRel`, `CommandFiles` fields, and the command-planning functions (`cmdRootFor`, `intendedCommandBasenames`, `planCommandCopies`, and any other command-specific helpers found via grep).
- [x] 2.3 In `internal/cli/update.go`, remove `writeCommandRows` and the "OpenCode) render nothing" comment/branch at ~line 605.
- [x] 2.4 Delete `agent-instructions/commands/learn.md`, `please.md`, `recall.md`, `route.md`.
- [x] 2.5 Run `targ test` and confirm everything from step 1 now passes (GREEN), with no leftover references (`grep -rniE "opencode" --include="*.go" internal/ cmd/` returns nothing).

## 3. Shelley removal

- [x] 3.1 Remove Shelley references from `docs/ROADMAP.md`.

## 4. Live-scope docs sync

- [x] 4.1 Update `README.md`: opening description, harness-support paragraph, engram-owned-root path examples, commands-source note — remove OpenCode and commands-mechanism mentions.
- [x] 4.2 Update `CLAUDE.md`: directory-structure comment referencing "Source for OpenCode slash commands."
- [x] 4.3 Update `docs/GLOSSARY.md`: harness enumeration (drop OpenCode), "command" term definition (was "OpenCode-specific" — reword or remove the term if it no longer applies to any harness).
- [x] 4.4 Update `docs/architecture/c1-system-context.md`: prose, mermaid diagram nodes, and sequence-diagram notes naming OpenCode or `~/.config/opencode/...` paths.
- [x] 4.5 Update `docs/architecture/c2-containers.md`: prose naming OpenCode/commands deploy targets.

## 5. Historical-record addendum

- [x] 5.1 Append a short dated note to ADR-0022 in `docs/architecture/adr.md` ("Update 2026-08-08: harness scope narrowed to Claude Code + Pi, see #721") without rewriting the original Decision prose. Leave ADR-0009/ADR-0010 untouched (already accurate, past-tense, `Superseded`).

## 6. Test fixture reword

- [x] 6.1 In `agent-instructions/skills/recall/tests/baseline-judgement-and-synthesis.md`, reword the scenario prompt's example task away from "wire OpenCode session transcripts into the ingest pipeline" to a harness-neutral realistic task, preserving everything else about the fixture unchanged.

## 7. Spec sync and final verification

- [x] 7.1 Apply the `update-deploy-sync` delta spec (this change's `specs/update-deploy-sync/spec.md`) into `openspec/specs/update-deploy-sync/spec.md` at archive time.
- [x] 7.2 Run a repo-wide case-insensitive grep for `opencode` and `shelley` (`grep -rniE "opencode|shelley" .` excluding `.git`) and confirm the only remaining hits are inside already-archived openspec changes and ADR-0009/ADR-0010's historical (Superseded) text.
- [x] 7.3 Run `targ check-full` and confirm all checks pass.
