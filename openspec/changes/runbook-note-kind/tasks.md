## 1. Capture: `engram learn runbook` + frontmatter schema (learn-runbook-capture)

- [ ] 1.1 Write RED tests for `type: runbook` frontmatter schema (fields: `type`, `date`, `situation`, `task_type`, `done_when`, `source`, optional `contributors`; filename `runbook.<YYYY-MM-DD>.<slug>.md`)
- [ ] 1.2 Implement `engram learn runbook` CLI command, wired via `cli.Primitives`/`cli.NewDeps` (DI, no direct I/O). Template: `internal/cli/qa.go` — or the `fact`/`feedback` capture path if #727 has already removed qa.go (see design.md Context sequencing note)
- [ ] 1.3 Wire embed-on-write for runbook notes: embedding (situation + body vectors, like fact/feedback) + vocab term assignment — NOT the `qa-question` exclusion pattern
- [ ] 1.4 Add `targets.go` CLI wiring + `--help` text for `engram learn runbook`
- [ ] 1.5 Get capture tests to GREEN; run `targ check-full`

## 2. write-memory + learn skill wiring

- [ ] 2.1 Using `superpowers:writing-skills` TDD: baseline behavior test (RED) for `write-memory` handling a runbook handoff
- [ ] 2.2 Update `write-memory` SKILL.md to compose/execute `engram learn runbook` from a `learn` handoff (slug, situation, task_type, done_when, body/steps, source, contributors)
- [ ] 2.3 Update `learn` SKILL.md: when to capture a runbook vs a feedback note (a reusable how-to keyed by task shape → runbook; a lesson from a correction/reversal → feedback), the three-questions schema (when to use it / the steps incl. `[[wikilinks]]` to facts+feedback worth reading / what's true when done), and the free-form `task_type` field with 2-3 concrete engram examples (`running-eval-harnesses`, `releasing-go-modules`, `recall-moment-discovery`)
- [ ] 2.4 Verify GREEN + pressure-test both skill edits per `superpowers:writing-skills`

## 3. Validation gate (blocks Section 4 — recall-runbook-surfacing gate)

- [ ] 3.1 Vault supply check: mine existing `feedback` notes for kind-4 (confirmed-approach) content already runbook-shaped; report count/examples
- [ ] 3.2 Design and run the A/B recall test (pre-filter-on vs pre-filter-off on a recurring task class; requires the Section 1 capture path to seed/author test runbook notes) to house harness standards — each rule below encodes a measured prior failure; none is optional:
  - headless `claude -p` per trial, fresh process (subagent controls inherit session context and contaminate the RED arm); no `--max-turns` — timeout-wrap and score what happened
  - real installed `engram` binary; per-trial `ENGRAM_VAULT_PATH`/`ENGRAM_CHUNKS_DIR` fixture vaults with real `.vec.json` sidecars — no stubbed retrieval (a stub once inverted rank order and invalidated a whole batch)
  - per-arm delivery gate verified from transcripts BEFORE scoring (the arm actually ran the query / actually received the runbook content); account for `~/.claude/CLAUDE.md` auto-loading into both arms
  - distractor runbooks with adjacent task_types planted in the fixture vault (a homogeneous pool cannot show discrimination)
  - noise floor sized from same-contrast repeats (A-vs-A), enough trials that a B−A gap below the floor reads "underpowered", never "no lift confirmed"
  - pass bars pre-registered before the first scored run; results reported as a labeled criteria table (units, arms side by side + Δ), and the projected spend estimated up front with the run then completing uninterrupted
- [ ] 3.3 Cross-check the task-type pre-filter approach against SPL's own eval methodology; note any material divergence
- [ ] 3.4 Document supply-check + A/B + SPL cross-check results in this change's PR/commit; confirm or revise the Decision 3 surfacing mechanism from design.md based on the A/B result

## 4. Recall/query surfacing (recall-runbook-surfacing) — gated on Section 3 passing

- [ ] 4.1 Write RED tests: runbook notes appear in `candidate_l2s` (no exclusion treatment)
- [ ] 4.2 Implement `engram query --task-type <slug>` (optional flag) + the task-type embedding-similarity pre-filter/boost, mirroring the `capWithNoteFloor` pattern from `recall-matched-note-floor`
- [ ] 4.3 Write RED tests for task-type match boosting ranking, and for the no-flag fallback to pure situation-similarity
- [ ] 4.4 Get ranking tests to GREEN; run `targ check-full`
- [ ] 4.5 Update `recall` SKILL.md: infer the session's current task type in Step 1 and pass `--task-type` on the Step 2 query; record the surfacing mechanism decision and rationale (per proposal acceptance criteria)

## 5. Close-out

- [ ] 5.1 Update `docs/architecture/adr.md` with a new ADR recording the `runbook` kind decision (name + three-questions schema, provenance: Joe 2026-08-23, vault note 789) and the surfacing mechanism choice
- [ ] 5.2 Run full `targ check-full`; confirm all four touch points (learn, write-memory, recall, query) covered; verify end-to-end via the installed binary with real args from a non-vault cwd
- [ ] 5.3 Run `/opsx:archive` to archive this change and sync `openspec/specs/`
