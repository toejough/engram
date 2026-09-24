## 1. Eval fixture and skill baseline

- [x] 1.1 Re-validate the four existing eval-fixture runbook notes (`dev/eval/cumulative/runbook_vs_skill/phase2/encodings/shim/recall-learn/vault/`: recall-glance-vs-deep-mode, recall-note-synthesis-and-coverage, recall-activation-and-closing-synthesis, recall-from-unified-memory) against the CURRENT real `recall/SKILL.md` for drift since 2026-09-14; do not trust them as-is
- [x] 1.2 Design one representative recall eval task exercising: a glance-only pass (cheap, read side only), and a scenario requiring escalation to deep (a recent-channel/Channel 2 standard the decision turns on, per C5). Decide whether these need separate task variants or one task with a branch point (design's Open Question). Cheap enough for n=3 per arm per variant. Document rationale in TASK-RATIONALE.md
- [x] 1.3 Build the fixture(s) under `dev/eval/cumulative/runbook_vs_skill/phase2/fixtures/recall/`
- [x] 1.4 Hand-build ideal end states for both the glance-only and escalation scenarios via the real binary in a scratch vault; confirm `done_when_checks.sh` passes each and at least 8 single-defect mutants per scenario fail
- [ ] 1.5 Estimate eval cost, get Joe's confirmation, then run the skill-only baseline(s), n=3, record followed_all/end_state as the skill row(s)

## 2. Conversion

- [x] 2.1 Enumerate `recall/SKILL.md`'s sections (Overview, Modes, the 9-step procedure, Red flags) and tag each carry/sub-runbook/drop-as-shim-floor; write the conversion-fidelity report plan
- [x] 2.2 Decide the final sub-runbook split (design D1) against measured byte counts: shared core procedure sub-runbook (Steps 0–3.5), write-extension sub-runbook (Step 2.5C + Step 4, wikilinking write-memory's real basename `1053.2026-09-22.write-memory-compose-execute-verify` directly), and whether Step 2.5 (4,783 B, the largest single step) needs its own further split — decided NOT to split Step 2.5 further (please's precedent shows ~13-18KB single notes are fine; the 1200-byte cap is on `red_flags` only, not body); Step 2.5's criterion half stayed in core, its action half moved to write-extension (see recall-conversion-fidelity-report.md's "Coverage-table split")
- [x] 2.3 Write `recall-glance` (situation, done_when, red_flags, body wikilinking the core sub-runbook + the C5 escalation rule + a wikilink to `recall-deep`)
- [x] 2.4 Write `recall-deep` (situation, done_when, red_flags, body wikilinking the core sub-runbook + the write-extension sub-runbook + an explanation of when glance should have escalated)
- [x] 2.5 Write the shared sub-runbook(s) from task 2.2's decision
- [x] 2.6 Prioritize the 26-row (measured; design's "28" was a byte-count-correct but row-count-off estimate) red_flags table under the 1200-byte cap PER NOTE (design D3); verified actual byte counts against each built note via `capRedFlagsForPreview`'s exact block and `engram show` (no omission marker) — all four notes well under budget (357-1028/1200)
- [x] 2.7 Complete the conversion-fidelity report: every carried/dropped/reworded item with reason, red_flags placement, wikilink targets, sub-runbook boundary rationale — `dev/eval/cumulative/runbook_vs_skill/phase2/encodings/recall-conversion-fidelity-report.md`

**Prerequisite harness fix (not in the original task list):** `probe_phase2.discover_tasks` doesn't
recurse into nested `fixtures/<name>/<scenario>/` layouts (immediate-children-of-`fixtures/`-only,
confirmed by reading it). Fixed by moving `fixtures/recall/{glance,escalation}/` to top-level
`fixtures/recall-glance/` and `fixtures/recall-escalation/`, mirroring `fixtures/gitignore`/
`fixtures/gitignore-nested` — zero harness code changes. TDD: RED test added first
(`test_recall_glance_and_escalation_are_discovered_as_top_level_tasks`), then the directory move, now
GREEN. Full `test_probe_phase2.py` suite: 455 passed, 2 skipped, no regressions. See
recall-conversion-fidelity-report.md's "Prerequisite" section for detail.
- [x] 2.8 Fresh-context reviewer checks the fidelity report against the real SKILL.md for lost content — reviewed 2026-09-23, verdict PASS-WITH-FIXES, fixes applied: harness `carrier_entry_basename` fix (blocker, TDD RED/GREEN, 460 passed/2 skipped); report self-contradictions on rows 6/15 corrected and the actual content restored into `recall-core` via `engram amend`; two missing SKILL.md:231-232 Step-3 sentences restored; two shipped-but-omitted red_flags rows added to the placement table; `1053...` both-fixtures claim corrected. See recall-conversion-fidelity-report.md's "Fresh review findings" section.

**Prerequisite harness fix, round 2 (task 2.8 fresh-context review, not in the original task
list):** `add_carrier` always returned the last-sorted `.md` basename in a carrier directory as
"the carrier" — correct for every single-entry-point carrier, but wrong once `recall-glance` and
`recall-escalation` began sharing ONE carrier directory (Recall-R's vault): both scored against
`4.2026-09-23.recall-deep`, even though `recall-glance`'s own entry point is
`3.2026-09-23.recall-glance`. Fixed by adding an optional `carrier_entry_basename` field to
`task.json`, used by `add_carrier` when present and falling back to the old last-sorted rule
otherwise (backward compatible with please/curate/write-memory/learn). TDD: RED test
(`test_recall_glance_found_scoring_uses_its_own_entry_point_not_recall_deep`) failed against the
old code, GREEN after the fix; full `test_probe_phase2.py` suite: 460 passed, 2 skipped, no
regressions. See recall-conversion-fidelity-report.md's "Fixture-vault wiring" section for detail.

## 3. Validation (gates retirement)

- [ ] 3.1 Retrieval check: score `recall-glance` and `recall-deep` against real agent first-query phrases (harvested from the skill-arm baseline transcripts, per vault note 1039) with NO `triggers:` field first (design D5)
- [ ] 3.2 Decide whether `triggers:` are needed based on 3.1's result; if similarity-only surfacing measurably fails against the shim's actual re-entry phrasing, add and re-score `/recall`, "recall glance", "recall deep" as candidates, with over-fire probes
- [ ] 3.3 **Before any paid run:** confirm any call-site wording fix reaches the LIVE `agent-instructions/skills/recall/SKILL.md` (or the eval's frozen skill copy, whichever the harness deploys) — hash-check per the write-memory conversion's own lesson
- [ ] 3.4 Run the shim-only arm(s) — glance and deep may need separate runs — n=3 each; validity gate: the transcript shows the wikilink chain actually followed (core sub-runbook fetched; write-extension fetched only when deep), and for the escalation scenario, that glance genuinely escalates to deep rather than guessing the right answer without following the rule
- [ ] 3.5 D7 bar: within one trial of the skill row(s) (followed_all and end_state, for each entry point/scenario) AND the validity gate holds in every passing trial
- [ ] 3.6 If D7 not met: record actuals, root-cause (split confusion → consider D1's fallback of two independent copies; wording; trigger absence), fix, re-run — applying 3.3's hash-check discipline before every re-run; do not proceed to section 4 until met or Joe redirects

## 4. Promotion

- [ ] 4.1 Promote the shared sub-runbook(s) to the production vault first via `engram learn runbook` (so wikilinks resolve), then `recall-glance` and `recall-deep`; confirm `engram embed status` clean
- [ ] 4.2 Update any live reference naming `recall` as a skill that needs a basename (should be minimal — recall itself has no incoming "fetch and follow" call sites from other skills, unlike write-memory)
- [ ] 4.3 Confirm `engram show <basename>` resolves against the real vault from a non-repo cwd, for each entry point and sub-runbook
- [ ] 4.4 Confirm `shim.md` is imported in the real `~/.claude/CLAUDE.md` (and `~/.pi/agent/AGENTS.md`) — re-confirm, don't assume

## 5. Retirement and references

- [ ] 5.1 Run the doc-surface enumeration grep for `recall` (as a skill reference) across docs, skills, guidance, specs, and code — note `recall` is referenced more widely than any prior conversion (it's cited by other skills/runbooks as the memory-lookup mechanism); write the per-file disposition list to `enumeration.md`; have a fresh-context reviewer independently verify it and run its own discovery pass
- [ ] 5.2 Fix `openspec/specs/recall-payload-cuts/spec.md:90`'s dead file-path reference (design D4) and update live references: `CLAUDE.md`, `README.md`, `docs/GLOSSARY.md`, `docs/architecture/{c1-system-context,c2-containers,c3-components}.md` (skill count in `agent-instructions/skills/` drops to zero once this and the sibling `learn` change both land)
- [ ] 5.3 Delete `agent-instructions/skills/recall/`; run `engram update` and confirm the deployed Claude Code and Pi copies are removed
- [ ] 5.4 `targ check-full` passes; `openspec validate --all --strict` passes

## 6. Close-out

- [ ] 6.1 Record the eval outcome(s) in `dev/eval/LEDGER.md`
- [ ] 6.2 Commit referencing #760 in prose (close #760 with a closing keyword only if the `learn` sibling change has already landed and this is genuinely the last piece — otherwise reference in prose only)
- [ ] 6.3 File the `guidance/recall.md` retirement as its own new issue (proposal.md's Deferred section)
- [ ] 6.4 Archive this change once merged
