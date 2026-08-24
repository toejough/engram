## 1. Capture: `engram learn runbook` + frontmatter schema (learn-runbook-capture)

- [ ] 1.1 Write RED tests for `type: runbook` frontmatter schema (fields: `type`, `date`, `luhmann`, `situation`, `done_when`, `source`, optional `contributors`; filename `<luhmann-id>.<YYYY-MM-DD>.<slug>.md`, identical scheme to `fact`/`feedback`)
- [ ] 1.2 Implement `engram learn runbook` as a third subcommand in the `learn` targ.Group (`internal/cli/targets.go`), reusing the shared `fact`/`feedback` pipeline: `LearnRunbookArgs` embedding `CommonLearnArgs` (gets `--target`/`--position` disposition flags for free) plus `--situation`/`--done-when`/`--body`; a `learnArgsFromRunbook` mapper into the existing `LearnArgs`; a `runbook` case in `assembleLearnContent` (new `runbookFields`/`runbookFrontmatterDoc` + render functions in `internal/cli/learn.go`). `RunLearn`/`writeLearnUnderLock`/`nextLuhmannID`/`learnPath` are reused unchanged. Template: `internal/cli/learn.go`'s existing `fact`/`feedback` handling — NOT `qa.go` (design.md Context)
- [ ] 1.3 Confirm embed-on-write and vocab term assignment apply to runbook notes automatically via the shared `autoEmbedNote`/`applyVocabAssignmentAfterLearn` pipeline — no new wiring needed, same path `fact`/`feedback` already use
- [ ] 1.4 Add `--help` text for `engram learn runbook`
- [ ] 1.5 Get capture tests to GREEN; run `targ check-full`

## 2. write-memory + learn skill wiring

- [ ] 2.1 Using `superpowers:writing-skills` TDD: baseline behavior test (RED) for `write-memory` handling a runbook handoff
- [ ] 2.2 Update `write-memory` SKILL.md to compose/execute `engram learn runbook` from a `learn` handoff (slug, situation, done_when, body/steps, source, target/position disposition, contributors) — the same disposition-judgment handoff shape write-memory already composes for fact/feedback
- [ ] 2.3 Update `learn` SKILL.md: when to capture a runbook vs a feedback note (a reusable how-to for a recurring task → runbook; a lesson from a correction/reversal → feedback), the three-questions schema (when to use it / the steps incl. `[[wikilinks]]` to facts+feedback worth reading / what's true when done), and that runbook capture judges Luhmann disposition (target/position) exactly like fact/feedback — a runbook extracted from existing kind-4 feedback content will often disposition as continuation/sibling of that feedback note, not top
- [ ] 2.4 Verify GREEN + pressure-test both skill edits per `superpowers:writing-skills`

## 3. Recall/query surfacing (recall-runbook-surfacing)

- [ ] 3.1 Write RED tests: runbook notes appear in `candidate_l2s` (no exclusion treatment), retrieval ranking is pure situation-similarity like fact/feedback
- [ ] 3.2 Get tests to GREEN; run `targ check-full`
- [ ] 3.3 Update `recall` SKILL.md: `runbook` notes surface and rank exactly like `fact`/`feedback` in Step 2's clustering/coverage judgment — no new Step-1 phrase, no new query flag

## 4. Close-out

- [ ] 4.1 Update `docs/architecture/adr.md` with a new ADR recording the `runbook` kind decision (name + three-questions schema, provenance: Joe 2026-08-23, vault note 789), the retrieval-treatment decision (situation-similarity only; a `task_type` mechanism was considered and rejected — design.md Decision 3, Non-Goals), and the Luhmann-participation decision (runbooks get a Luhmann ID via the same disposition judgment as fact/feedback, not a kind-prefixed flat name — design.md Decision 1)
- [ ] 4.2 Run full `targ check-full`; confirm all four touch points (learn, write-memory, recall, query) covered; verify end-to-end via the installed binary with real args from a non-vault cwd
- [ ] 4.3 Run `/opsx:archive` to archive this change and sync `openspec/specs/`
