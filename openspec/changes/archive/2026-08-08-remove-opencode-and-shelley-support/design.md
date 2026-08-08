## Context

Engram registers three harnesses in `internal/update/update.go`'s `supportedHarnesses()`: Claude Code, OpenCode, Pi. OpenCode is the sole consumer of the commands-deploy mechanism (`CommandsTargetRel` is empty for both other harnesses) — it's a whole feature axis (fields, planning functions, report rendering, `agent-instructions/commands/*.md` source files), not a single case in a switch. Shelley (exe.dev, #707) was researched but never implemented in code; its only footprint is `docs/ROADMAP.md`. Joe decided (2026-08-08) to narrow scope to Claude Code and Pi and remove both entirely rather than leave anything dormant, filed as issue #721.

Research during exploration (see conversation) found the removal's real blast radius is bigger than the issue's original wording ("code paths, docs, issues"):
- 3 hardcoded `HaveLen(3)` harness-count assertions across `sync_test.go`/`update_test.go` that must become `HaveLen(2)`.
- ~76 OpenCode-referencing test lines across 5 test files, plus the commands mechanism's own test coverage.
- README's opening description and multiple architecture docs (C1/C2, including mermaid diagram nodes) state OpenCode support as current, live behavior.
- `openspec/specs/update-deploy-sync/spec.md` bakes "one per command file" into a generic requirement and uses a command-deploy example in the manifest-mode scenario — both need delta specs, independent of code changes.
- ADR-0022 (status `Accepted`) lists OpenCode as a current harness in its Decision section — distinct from ADR-0009/ADR-0010, which already describe the OpenCode transcript-backend's prior removal in the past tense (`Superseded`) and need no change.
- `agent-instructions/skills/recall/tests/baseline-judgement-and-synthesis.md` uses OpenCode only as incidental flavor text for an example task, not a support claim.

## Goals / Non-Goals

**Goals:**
- Remove `HarnessOpencode` and every OpenCode-specific code path, test, and fixture.
- Remove the commands-deploy mechanism entirely (dead once OpenCode is gone) rather than keep it as unexercised infrastructure.
- Remove Shelley from `docs/ROADMAP.md`.
- Bring live-scope docs (README, CLAUDE.md, GLOSSARY, C1/C2 architecture docs) back in sync with the narrowed two-harness reality.
- Keep historical records (ADR-0009/0010, any git-recoverable code) intact and truthful to what happened when it happened.

**Non-Goals:**
- No rewrite of already-`Superseded` ADR text describing past removals — those remain accurate as written.
- No new harness-count abstraction or generalized N-harness design — just correcting the hardcoded 3 → 2.
- No re-litigating whether commands-as-a-concept might return for a future harness; per the "mine git history" decision, that's a from-scratch reintroduction if it ever happens, not a preserved-but-dormant feature.

## Decisions

- **Full deletion over deprecation shim.** No compat symlinks, no "removed" comments, no feature flag — matches this repo's `git_is_the_fallback` convention. If OpenCode or Shelley support is wanted later, it's reconstructed from git history, not toggled back on.
- **ADR-0022 gets an addendum, not a rewrite.** It's `Accepted` (not `Superseded`), and its Decision section is the actual historical record of *why* per-file vs per-directory symlink granularity was chosen — rewriting that prose to erase OpenCode would misrepresent what was actually decided and why. A short dated note ("Update 2026-08-08: harness scope narrowed to Claude Code + Pi, see #721") is appended instead, leaving the original decision text intact.
- **ADR-0009/ADR-0010 untouched.** Already `Superseded`, already past-tense, already correctly describe what was removed and when. No factual claim in them becomes false by this change.
- **Spec deltas over silent main-spec edits.** `update-deploy-sync`'s two affected requirements get proper MODIFIED-requirement deltas (this change's `specs/update-deploy-sync/spec.md`) so the spec history records why the command-file clause and manifest-mode example changed.
- **Test fixture reworded, not restructured.** `baseline-judgement-and-synthesis.md`'s OpenCode mention is incidental (an arbitrary realistic task for testing the recall skill's behavior) — swap the example task to a harness-neutral one, don't redesign the fixture.

Alternatives considered:
- *Keep commands-deploy mechanism as generic, currently-unused infrastructure*: rejected — explicitly the "dormant hanging around" outcome Joe ruled out.
- *Rewrite ADR-0022's Decision prose to remove OpenCode entirely*: rejected — would misstate the historical basis for the per-file-vs-per-directory symlink design decision, which was made when three harnesses were in scope.

## Risks / Trade-offs

- [Risk] Missing a call site during the commands-mechanism deletion causes a Go compile error. → Mitigation: static typing catches this immediately; `targ test`/`targ check-full` gate every task.
- [Risk] Docs enumeration is easy to under-scope (as the original issue's research showed). → Mitigation: this design's Context section already captures the full grep-verified list; tasks.md enumerates every known file. A final repo-wide grep for "opencode"/"shelley" (case-insensitive) closes any gap before declaring done.
- [Risk] Deleting `agent-instructions/commands/*.md` removes user-facing slash-command wrappers some external tooling might reference. → Mitigation: these wrappers only ever targeted OpenCode's discovery mechanism; Claude Code and Pi never used them. No known consumer outside OpenCode.

## Migration Plan

Single-PR removal, ordered to keep the repo compiling/testing green at each step:
1. Delete OpenCode + commands-mechanism code and update/fix all affected tests (RED→GREEN per task).
2. Delete `agent-instructions/commands/*.md`.
3. Update live-scope docs (README, CLAUDE.md, GLOSSARY, C1/C2).
4. Remove Shelley from ROADMAP.md.
5. Append ADR-0022 addendum.
6. Reword the recall-skill test fixture.
7. Apply the `update-deploy-sync` spec delta at archive time.
8. Final repo-wide grep sweep + `targ check-full`.

No rollback complexity — this is source-controlled deletion; `git revert` recovers everything if needed.

## Open Questions

None.
