Conventions: TDD for every Go change (RED via `targ test`, then GREEN, refactor); gomega + rapid + `t.Parallel()`; no direct I/O in `internal/` (DI fakes); `targ check-full` before ticking section 8. Skill edits go through `superpowers:writing-skills`. No paid eval is planned (design D9); if one becomes necessary, estimate and get Joe's confirmation first.

## 1. Registration mechanism (Go)

- [ ] 1.1 RED→GREEN: a SKILL.md/runbook-file frontmatter parser (`name`, `description`, `situation`, `done_when`, `triggers`, `red_flags`; body) with a "registered iff situation+done_when present" predicate; property test: any YAML the runbook-note renderer emits round-trips; unknown keys tolerated
- [ ] 1.2 RED→GREEN: discovery of a skill directory's runbooks (`SKILL.md` = slug `SKILL`; `runbooks/<slug>.md`); slug validation; a test that a skill with only `name`/`description` yields zero registrations
- [ ] 1.3 RED→GREEN: `registered_from` frontmatter field on runbook notes (`runbookFrontmatterDoc`, render, parse) preserved by `amend`; scan-by-field lookup over the vault via `QueryDeps`/`LearnDeps` (no new I/O)
- [ ] 1.4 RED→GREEN: upsert — found → amend all derived fields (replace-whole), preserve `created`; not found → create via the runbook capture path; idempotence test (second run writes nothing); adoption test (a pre-existing note whose fields equal the skill's is stamped, not duplicated — design D3)
- [ ] 1.5 RED→GREEN: slug cross-reference rewriting `[[<skill>/<slug>]]` → real basenames for the same skill's notes; unresolved slug fails that skill loudly; property test: non-slug wikilinks untouched, rewrite is a no-op on already-rewritten bodies
- [ ] 1.6 RED→GREEN: sync removal — derived notes with no matching registered runbook are removed (with sidecars); a test that captured (non-derived) runbooks are never touched
- [ ] 1.7 RED→GREEN: derived-note body preamble; `engram amend` warns when the target has `registered_from`
- [ ] 1.8 RED→GREEN: `engram register-skills [--dry-run]` command (targets wiring; `cmd/engram/main.go` stays wiring-only) and the `runPostUpdateChecks` hook in `internal/cli/update.go`; `--dry-run` for both prints creates/updates/removes and writes nothing; registration errors are reported and do not roll back the deploy
- [ ] 1.9 `targ test` green; `targ check-full` green apart from `check-uncommitted`

## 2. Restore the four skills with authored metadata

- [ ] 2.1 Restore `route/` from `f5b44504` (SKILL.md, price-table.md, tests/), `please/SKILL.md` from `ec5f3735`, `curate/SKILL.md` from `14049280`, `write-memory/SKILL.md` from `cbda7f6e` (`git checkout <commit> -- <path>`); confirm byte-identical to the last intact versions
- [ ] 2.2 Lift frontmatter from the promoted notes: route ← 1036 (situation, done_when, 15 red_flags, no triggers); curate ← 1049 (7 red_flags; triggers curate,/curate,pending offers,pending offer); write-memory ← 1053 (situation, done_when, red_flags minus the "no longer exists as a skill" entry, no triggers); verify red_flags render ≤ 1200 bytes each with the real truncation rule
- [ ] 2.3 Split `please` into `SKILL.md` (from 1045: situation, done_when, 9 red_flags, triggers /please, take this end-to-end, please) + `runbooks/adversarial-review-gates.md` (1042), `runbooks/step7-lessons-audit.md` (1043), `runbooks/step3-doc-surface-enumeration-grep.md` (1044), each with its note's frontmatter; rewrite the body wikilinks between them as `[[please/<slug>]]`; route's reference stays a real basename (different skill)
- [ ] 2.4 Revert the nine `recall`/`learn` write-memory call sites to native skill invocation ("invoke the **write-memory** skill with this handoff") via `superpowers:writing-skills` (RED: the current wording tells agents the skill does not exist); keep `learn`'s Step 2.5 duplicate-guard fix and the `REQUIRED NEXT ACTION` labels only where they are still accurate
- [ ] 2.5 Fresh-context reviewer diffs each restored skill against (a) its last intact commit and (b) its promoted note's fields, and confirms nothing was lost in either direction

## 3. First registration and adoption

- [ ] 3.1 `engram register-skills --dry-run` on the real vault shows exactly: adopt 1036, 1045, 1042, 1043, 1044, 1049, 1053 (stamp `registered_from`), rewrite please's cross-links to those basenames, create nothing, remove nothing
- [ ] 3.2 Run it for real; `engram embed status` clean; `engram show` on each note shows `registered_from`, the preamble, and resolving outbound links for please's subs
- [ ] 3.3 No-spend retrieval checks: for each restored skill, the same real-agent phrases used in its conversion's retrieval check (results files under `dev/eval/cumulative/runbook_vs_skill/phase2/results/`) surface the derived note exactly as before; over-fire probes unchanged
- [ ] 3.4 Idempotence on the real vault: a second run writes nothing

## 4. Register recall and learn

- [ ] 4.1 Author `learn`'s frontmatter from `encodings/taskLearn/Learn-R/vault/` (top 1057 → `SKILL.md`; 1054/1055/1056 → `runbooks/`), with the review-fixed triggers (`/learn`, remember this, note for next time, write this down) and red_flags placements; slug cross-links
- [ ] 4.2 Author `recall`'s files per design D5 (`runbooks/glance.md`, `runbooks/deep.md`, `runbooks/core.md`, `runbooks/write-extension.md` from `encodings/taskRecall/Recall-R/vault/`; `SKILL.md` carries mode selection only — resolve the Open Question on whether it is itself a router skill); no triggers on recall's notes until the D5-style similarity check says otherwise
- [ ] 4.3 Register both; retrieval checks with the real-agent phrases harvested in those changes' fixture work; write-memory reached natively as a skill from both
- [ ] 4.4 Fresh-context reviewer confirms recall/learn's authored frontmatter matches the reviewed fixture runbooks field-for-field

## 5. Abandon the two delete-conversion changes

- [ ] 5.1 Remove `openspec/changes/learn-skill-to-runbook/` and `openspec/changes/recall-glance-skill-to-runbook/` in a commit whose message records the reversal (vault note 1054), what was carried forward (fixtures, fidelity reports, harness fixes stay under `dev/eval/`), and that no paid run was made; `openspec validate --all --strict` passes afterward

## 6. `show` parent fallback (design D7)

- [ ] 6.1 RED→GREEN in `internal/cli/show.go` + `serve_client.go`: local miss + `ENGRAM_PARENT` set + no `ENGRAM_SERVER` → resolve against the parent, label output parent-sourced; local hit never contacts the parent; no parent configured → unchanged not-found error; explicit `--parent` and `ENGRAM_SERVER` precedence unchanged (tests for each scenario in the delta spec)
- [ ] 6.2 Merged-vault smoke (no spend): a runbook present only in a parent vault is surfaced by a client's `engram query` and then fetched by bare `engram show <basename>`

## 7. Docs, specs, enumeration

- [ ] 7.1 Doc-surface enumeration for "retired"/"no longer a skill"/skill-count language and the reversed end goal: `CLAUDE.md`, `README.md`, `docs/GLOSSARY.md` (`skill`, `runbook`, write-memory entries), `docs/architecture/{adr (ADR-0026 amendment, ADR-0015 note), c1, c2, c3}.md`, `docs/ROADMAP.md` (#758/#759/write-memory "Shipped" rows amended, #760 re-scoped), `agent-instructions/guidance/shim.md` if it asserts skills are gone; write `enumeration.md`; independent fresh-context review; perform every row
- [ ] 7.2 Archive-time task: replace `openspec/specs/guidance-runbook-follow-frame/spec.md`'s Purpose sentence "every procedure, including recall and learn, is a runbook note it finds and follows" with wording that registered skills are mirrored as runbook notes and skills remain the shipping form (a delta cannot edit Purpose)
- [ ] 7.3 Archive-time task: amend the "retired" Purpose wording in `openspec/specs/write-memory-worker/spec.md` and any other main-spec Purpose line the enumeration finds
- [ ] 7.4 Comment on GitHub #757, #758, #759, #760 recording the regression, the pivot, and this change; do not close #760

## 8. Close-out

- [ ] 8.1 `engram update --with-guidance` on this machine; verify all six skills deployed in `~/.claude/engram/skills/` and the Pi root, all derived notes present, `engram embed status` clean
- [ ] 8.2 `targ check-full` fully green; `openspec validate --all --strict`
- [ ] 8.3 Commit(s) with `AI-Used: [claude]`; record the outcome in `dev/eval/LEDGER.md` (mechanism unit-tested; retrieval re-checked at no spend)
- [ ] 8.4 Archive this change; sync specs; perform 7.2/7.3
