Conventions: TDD for every Go change (RED via `targ test`, then GREEN, refactor); gomega + rapid + `t.Parallel()`; no direct I/O in `internal/` (DI fakes); `targ check-full` before ticking section 8. Skill edits go through `superpowers:writing-skills`. No paid eval is planned (design D10); if one becomes necessary, estimate and get Joe's confirmation first.

## 1. Registration mechanism (Go)

- [x] 1.1 RED→GREEN: `skill_hash` frontmatter field on runbook notes (`runbookFrontmatterDoc`, render, parse), preserved by `amend`; a test that `engram learn` outside registration never writes it
- [x] 1.2 RED→GREEN: pending-offer marker recognized on `runbook` notes (`noteHasPendingMarker`, `offer.go:84`): excluded from normal query results, raises `pending_offers`, cleared by `engram amend --clear-pending`; existing fact/feedback behavior unchanged
- [ ] 1.3 RED→GREEN: skill-note lookup by basename suffix `.skill-<name>.md` on runbook notes carrying `skill_hash` (via existing list/read deps, no new I/O); duplicate matches error naming both
- [ ] 1.4 RED→GREEN: offer comparison — per shipped skill: no note → register offer; hash differs → refresh offer; note with no shipped skill → removal offer; hash match or declined hash → nothing. Property test: a run after answering every offer makes no offers
- [ ] 1.5 RED→GREEN: `skill-registrations.json` decline state at the vault root (read, record per skill name, tolerate absent file); decline records the current hash and changes nothing else; a new hash re-offers once
- [ ] 1.6 RED→GREEN: accept actions — register creates a pending note (capture path, slug `skill-<name>`, preamble + `SKILL.md` body, `skill_hash`, no runbook fields, sidecar); refresh replaces body + hash, preserves fields/`created`/basename, sets pending, rebuilds sidecar; removal deletes note + sidecar
- [ ] 1.7 RED→GREEN: `--adopt <name>=<note-ref>` via `RenameAndRewriteReferences` (id and date kept, inbound links rewritten, sidecar moved), body replaced, `skill_hash` stamped, fields kept, not pending
- [ ] 1.8 RED→GREEN: prompting — interactive prompt per offer when stdin is a terminal; non-interactive path writes nothing, records nothing, and prints one line naming waiting skills and the answering command; `--accept`/`--decline <name>` answer without prompting
- [ ] 1.9 RED→GREEN: `engram register-skills [--dry-run] [--accept <name>]... [--decline <name>]... [--adopt <name>=<ref>]...` (targets wiring; `cmd/engram/main.go` stays wiring-only) and the `runPostUpdateChecks` hook in `internal/cli/update.go`; `--dry-run` for both lists offers and writes nothing; registration errors are reported and do not roll back the deploy
- [ ] 1.10 `targ test` green; `targ check-full` green apart from `check-uncommitted`

## 2. Restore the four skills

- [x] 2.1 Restore `route/` from `f5b44504` (SKILL.md, price-table.md, tests/), `please/SKILL.md` from `ec5f3735`, `curate/SKILL.md` from `14049280`, `write-memory/SKILL.md` from `cbda7f6e` (`git checkout <commit> -- <path>`); confirm byte-identical to the last intact versions and that no frontmatter beyond `name`/`description` exists
- [x] 2.2 Revert the nine `recall`/`learn` write-memory call sites to native skill invocation ("invoke the **write-memory** skill with this handoff") via `superpowers:writing-skills` (RED: the current wording tells agents the skill does not exist); keep `learn`'s Step 2.5 duplicate-guard fix and the `REQUIRED NEXT ACTION` labels only where they are still accurate
- [x] 2.3 Add curate's branch for pending skill notes (author, or after a refresh re-check, `situation`/`done_when`/`triggers`/`red_flags` from the body within the 1200-byte red_flags cap, then `engram amend --clear-pending`) via `superpowers:writing-skills`
- [x] 2.4 Fresh-context reviewer diffs each restored skill against its last intact commit and confirms nothing was lost

## 3. Adopt the existing notes

- [ ] 3.1 `engram register-skills --dry-run` on the real vault before adoption shows register offers for route, please, curate, write-memory (no notes carry the `skill-` slug yet); answer nothing
- [ ] 3.2 Adopt: `--adopt route=1036 --adopt please=1045 --adopt curate=1049 --adopt write-memory=1053`; drop write-memory's "no longer exists as a skill" red_flag from 1053 with `engram amend`
- [ ] 3.3 please: review sub-notes 1042/1043/1044's red_flags into the please note within the 1200-byte cap (record anything cut in this task's notes); repoint every inbound link to 1042–1044 at the please note; remove 1042–1044 and their sidecars after Joe confirms the list
- [ ] 3.4 `engram embed status` clean; `engram show` on each adopted note shows the `skill-` basename, `skill_hash`, the preamble, and no pending marker; `engram register-skills --dry-run` shows no offers for the four
- [ ] 3.5 No-spend retrieval checks: for each adopted skill, the same real-agent phrases used in its conversion's retrieval check (results files under `dev/eval/cumulative/runbook_vs_skill/phase2/results/`) surface the note as before; over-fire probes unchanged; for please, flag any regression from the one-note shape (design D10) to Joe before considering a paid re-run

## 4. Register recall and learn

- [ ] 4.1 Accept registration offers for `recall` and `learn` (`engram register-skills --accept recall --accept learn`); confirm both notes are pending with no runbook fields
- [ ] 4.2 Curate `learn`'s note: author fields from `encodings/taskLearn/Learn-R/vault/` top note 1057 with the review-fixed triggers (`/learn`, remember this, note for next time, write this down), folding sub-note red_flags within the cap; clear pending
- [ ] 4.3 Curate `recall`'s note: author one situation covering both modes from `encodings/taskRecall/Recall-R/vault/` glance/deep entry notes; no triggers until a similarity check says otherwise; clear pending
- [ ] 4.4 Retrieval checks with the real-agent phrases harvested in those changes' fixture work; write-memory reached natively as a skill from both
- [ ] 4.5 Fresh-context reviewer confirms recall/learn's authored fields match the reviewed fixture runbooks, noting anything consolidated from sub-notes

## 5. Abandon the two delete-conversion changes

- [x] 5.1 Remove `openspec/changes/learn-skill-to-runbook/` and `openspec/changes/recall-glance-skill-to-runbook/` in a commit whose message records the reversal (vault note 1054), what was carried forward (fixtures, fidelity reports, harness fixes stay under `dev/eval/`), and that no paid run was made; `openspec validate --all --strict` passes afterward

## 6. `show` parent fallback (design D8)

- [x] 6.1 RED→GREEN in `internal/cli/show.go` + `serve_client.go`: local miss + `ENGRAM_PARENT` set + no `ENGRAM_SERVER` → resolve against the parent, label output parent-sourced; local hit never contacts the parent; no parent configured → unchanged not-found error; explicit `--parent` and `ENGRAM_SERVER` precedence unchanged (tests for each scenario in the delta spec)
- [x] 6.2 Merged-vault smoke (no spend): a runbook present only in a parent vault is surfaced by a client's `engram query` and then fetched by bare `engram show <basename>`

## 7. Docs, specs, enumeration

- [ ] 7.1 Doc-surface enumeration for "retired"/"no longer a skill"/skill-count language and the reversed end goal: `CLAUDE.md`, `README.md`, `docs/GLOSSARY.md` (`skill`, `runbook`, write-memory entries), `docs/architecture/{adr (ADR-0026 amendment, ADR-0015 note), c1, c2, c3}.md`, `docs/ROADMAP.md` (#758/#759/write-memory "Shipped" rows amended, #760 re-scoped), `agent-instructions/guidance/shim.md` if it asserts skills are gone; write `enumeration.md`; independent fresh-context review; perform every row
- [ ] 7.2 Archive-time task: replace `openspec/specs/guidance-runbook-follow-frame/spec.md`'s Purpose sentence "every procedure, including recall and learn, is a runbook note it finds and follows" with wording that registered skills are mirrored as runbook notes and skills remain the shipping form (a delta cannot edit Purpose)
- [ ] 7.3 Archive-time task: amend the "retired" Purpose wording in `openspec/specs/write-memory-worker/spec.md` and any other main-spec Purpose line the enumeration finds
- [ ] 7.4 Comment on GitHub #757, #758, #759, #760 recording the regression, the pivot, and this change; do not close #760

## 8. Close-out

- [ ] 8.1 `engram update --with-guidance` on this machine; verify all six skills deployed in `~/.claude/engram/skills/` and the Pi root, every shipped skill has a note with a matching `skill_hash`, no registration offers outstanding, `engram embed status` clean
- [ ] 8.2 `targ check-full` fully green; `openspec validate --all --strict`
- [ ] 8.3 Commit(s) with `AI-Used: [claude]`; record the outcome in `dev/eval/LEDGER.md` (mechanism unit-tested; retrieval re-checked at no spend)
- [ ] 8.4 Archive this change; sync specs; perform 7.2/7.3
