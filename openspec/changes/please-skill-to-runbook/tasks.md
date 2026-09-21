## Status: UNPARKED 2026-09-20 (R1 chosen by Joe: fully retire `please/SKILL.md` now)

**Unblock condition met.** `runbook-lexical-triggers` shipped (commits 00efc1c6, 00c9299c, 8468823a, ec5f3735).
The real-vault please runbook (note 1045.2026-09-20.please-drive-ask-end-to-end) carries
`triggers: ["/please", "take this end-to-end", "please"]`. The shim-only rerun with triggers (task 6.2 of
that change; `dev/eval/cumulative/runbook_vs_skill/phase2/results/6.2_please_triggers_sonnet5.md`): runbook
surfaced with `trigger` provenance 3/3 (was 2/3), end_state 3/3, `--text` in the first query 3/3.
followed_all was scored 0/3 but that is a scorer artifact (step-10 python-heredoc signal plus the `after`
chain; 17-18 of 19 with `after` removed), not a behavior regression. Route regression 3/3.

**Done:** section 1 (fixture, baselines), section 2 (conversion; 2.7 review still open), section 3/3a
(D8 bar met on 3.3d, shim re-check deployed), section 4 (promotion, notes 1042-1045, retrieval and shim
import confirmed).

**Remaining:** 2.7 fresh-context fidelity review; section 5 (enumeration verified by a fresh reviewer,
live-reference updates, delete `please/`, `engram update`, real-query check, `targ check-full`); 6.x
close-out. Enumeration draft: `enumeration.md` (this directory).

**History (why it was parked):** semantic retrieval of the agent's paraphrase missed the please runbook
(0/3 before the shim fix, 2/3 after, misses whenever phrase two names the deliverable); the literal
third-phrase rule (D11, section 3b) lifted rank/score in 0/95 cells of the scratch-vault probe and was
dropped. Lexical triggers replaced it.

## 1. Eval fixture and skill-arm baseline

- [x] 1.1 Pick one representative multi-step please ask (exercises the seven-step spine, at least one review gate, and the Step 7 lessons-audit close) and record why it is representative
- [x] 1.2 Build the please eval task under `dev/eval/cumulative/runbook_vs_skill/phase2/` (fixture repo, task prompt, done_when checks) following the route task's layout
- [x] 1.3 Estimate eval cost, get Joe's confirmation, then run the skill-arm baseline (n=3) and record `followed_all`/`end_state` as the skill row

## 2. Conversion

- [x] 2.1 Enumerate `please/SKILL.md` sections and tag each: carry / sub-runbook / drop-as-shim-floor; write the plan into the conversion-fidelity report
- [x] 2.2 Write the three sub-runbooks in the fixture vault (adversarial review gates, lessons audit, doc-surface enumeration grep), body verbatim where possible
- [x] 2.3 Write the top runbook: `situation` (with `/please` trigger discrimination, D5), seven-step spine, `[[basename]]` links to the sub-runbooks in the body only, `done_when`
- [x] 2.4 Prioritize the ~22-row Red Flags table under the 1200-byte `red_flags` cap (D3); include the plain-text-instead-of-`[[wikilink]]` entry; move overflow to sub-runbook `red_flags`/body
- [x] 2.5 State the `[[wikilink]]` syntax requirement emphatically at each place the agent writes a vault-note reference
- [x] 2.6 Complete the conversion-fidelity report: every dropped/moved/reworded item with reason, including each shim-floor drop mapped to the shim.md rule that covers it
- [ ] 2.7 Fresh-context reviewer checks the fidelity report against the SKILL.md for lost content

## 3. Validation (gates retirement)

- [x] 3.1 Retrieval check in a trial vault: task-phrase and situation-phrase `engram query` surface the top runbook top-ranked; a casual-"please" phrase does not (no LLM spend). The probe must use the exact excluded phrases ("casual please", "please read this file", "please rename this var") because embeddings do not encode negation and the situation's trailing exclusion may attract them; reword the situation if it does
- [x] 3.2 Confirm validity gate: treatment arms actually loaded the runbook (marker-in-transcript + skill-shadowing scan) before scoring. (First run INVALID: never retrieved. Re-run 2026-09-20 after 3a: treatment loaded 2/3, marker 3/3, no shadowing; see results/3.3c_please_shim_only_after_shim_fix_sonnet5.md.)
- [x] 3.3 Run `probe_phase2.py --task please --shim-only` (n=3) and record `followed_all`/`end_state` against the skill row; D8 bar = within one trial. (First run INVALID: never retrieved 0/3. Re-run 2026-09-20: retrieved 2/3, followed_all 1/3, end_state 1/3 vs skill row 0/3, 2/3 -> D8 bar met on the letter (end_state >= 1/3), fragile; see results/3.3c_please_shim_only_after_shim_fix_sonnet5.md. Re-run 2026-09-20 with situation P2h + step-1 runbook fix: retrieved 2/3, followed_all 2/3, end_state 2/3, D8 met; see results/3.3d_please_shim_only_p2h_sonnet5.md.)
- [x] 3.4 (N/A, satisfied vacuously: D8 met on 3.3d, `results/3.3d_please_shim_only_p2h_sonnet5.md`) If D8 not met, record actuals, iterate on sub-runbook boundaries / `red_flags` selection, re-run; do not proceed to section 4 until met or Joe redirects

## 3a. Retrieval fix (shim re-check + situation rewrite, D10)

- [x] 3a.1 Draft the phrase-two re-check and one non-dispatch example in `agent-instructions/guidance/shim.md`
- [x] 3a.2 (evidence: `openspec/changes/runbook-lexical-triggers/tasks.md` 5.3 DONE 2026-09-20, `engram update --with-guidance` ran; re-verified 2026-09-20 `cmp` byte-identical between `agent-instructions/guidance/shim.md` (contains the phrase-two re-check) and `~/.claude/engram/guidance/shim.md`, `~/.pi/agent/engram/guidance/shim.md`, `~/.pi/agent/guidance/shim.md`) Deploy via `engram update --with-guidance` and confirm the deployed shim carries the re-check
- [x] 3a.3 Headless RED/GREEN validation of phrase-two behavior: fresh `claude -p` per arm, shim old vs new the only variable, fictional/varied tasks, verify the treatment loaded (marker in transcript), score whether the second `--phrase` is free of ticket nouns (see vault notes on headless guidance evals)
- [x] 3a.4 Route retrieval regression check: route's shim-only task still retrieves the route runbook at the pre-change rate with the updated shim
- [x] 3a.5 Pick a process-shaped `situation` for the top please runbook from the scored candidates; apply to the fixture; re-run the 3.1 retrieval and over-fire probes
- [x] 3a.6 Then re-run 3.2/3.3

## 3b. Literal-phrase rule (D11) — DROPPED 2026-09-20

Section removed: D11 (a third `--phrase` quoting the user's message verbatim) was drafted in the shim
and scored in a scratch vault, and changed the top runbook's rank/score in 0/95 cells. The shim draft
is reverted; the tasks that were here (headless RED/GREEN, route regression, please re-probe, redeploy,
re-promotion) do not apply. Triggering moves to the follow-on change `runbook-lexical-triggers`.

## 4. Promotion

- [x] 4.1 Create the sub-runbooks in the production vault via `engram learn runbook` (never a hand-edit or file copy), sub-runbooks first so wikilinks resolve
- [x] 4.2 Create the top runbook via `engram learn runbook`; confirm `engram embed status` is clean
- [x] 4.2a Ensure the promoted please runbook wikilinks route's production runbook `[[1036.2026-09-18.route-dispatch-tier-selection]]` in its body. The fixture references route by plain name because trial vaults exclude luhmann >= 955; production must use the wikilink so the route rubric and LESSONS contract are reachable by transitive follow
- [x] 4.3 Re-run the retrieval check against the production vault
- [x] 4.4 Confirm `shim.md` is imported in the real `~/.claude/CLAUDE.md` (and any `~/.pi/agent/` equivalent)

## 5. Retirement and references

- [x] 5.1 (DONE 2026-09-20; evidence: `enumeration.md` verified by an independent fresh-context reviewer, whose corrections (README prune line, dev/eval live harness code, skill lists, glossary runbook entry, extra spec rows) were applied and every update/rewrite row performed) Run the enumeration grep for `please` across docs, skills, guidance, specs and code; write the per-file disposition list and have a fresh-context reviewer verify it and run an independent discovery pass
- [x] 5.2 (DONE 2026-09-20; evidence: CLAUDE.md, README.md, delegate.md, learn/SKILL.md:89, GLOSSARY (skill + new runbook entries), c1, c2, adr.md, memory-invariants.md, probe/traps docs edited; ROADMAP rows are dated history, kept; `targ test` and `targ check-full` green except check-uncommitted; `openspec validate --strict` valid; phase2 pytest 353 passed, step7 scorer cases 17/17) Update live references: `CLAUDE.md`, `README.md`, `agent-instructions/guidance/delegate.md`, `agent-instructions/skills/learn/SKILL.md`, `docs/GLOSSARY.md`, `docs/architecture/{c1-system-context,c2-containers,adr}.md`, `docs/ROADMAP.md`
- [x] 5.3 (DONE 2026-09-20; evidence: `git rm -r` of `agent-instructions/skills/please/`; `engram update --with-guidance` run from this clone deleted `skills/please` from both `~/.claude/engram` and `~/.pi/agent/engram` roots and removed dangling links `~/.claude/skills/please` and `~/.pi/agent/skills/please`; `ls` confirms all four gone; curate/learn/recall/write-memory and guidance files byte-identical to the repo) Delete `agent-instructions/skills/please/`; run `engram update` and confirm the deployed Claude Code and Pi copies are removed
- [x] 5.4 (DONE 2026-09-20; evidence: from cwd `~`, `engram query --lazy-chunks --text "/please rename the tally CLI flag"` and `--text "please take this end-to-end: X"` both return 1045.2026-09-20.please-drive-ask-end-to-end as items[0] (kind runbook, score 0.706, provenances direct+trigger); without `--text` 1045 is also items[0] via direct only) Run the real `engram query` with a `/please`-style ask from a non-repo cwd and confirm the top runbook surfaces
- [x] 5.5 (DONE 2026-09-20; evidence: `targ check-full` all checks PASS except check-uncommitted, which passes once this change is committed; `targ test` green; `openspec validate --strict` valid for both changes) `targ check-full` passes
- [ ] 5.6 On archive/sync, edit the main spec `openspec/specs/please-doc-enumeration-gate/spec.md` title and Purpose (deltas cannot modify them): replace "the please skill's Step 3" with the please runbook / its doc-surface-enumeration sub-runbook
- [ ] 5.7 (archive time) Edit `openspec/specs/write-memory-worker/spec.md` line 5 (Purpose): it says the Step-7 addition "is committed in agent-instructions/skills/please/SKILL.md (commit 662e50ba)"; reword to "originally committed in agent-instructions/skills/please/SKILL.md (662e50ba, since retired; now carried by the please lessons-audit sub-runbook)" (a delta cannot edit Purpose)

## 6. Close-out

- [x] 6.1 (DONE 2026-09-20; ledger rows recorded in `dev/eval/LEDGER.md`) Record the eval outcome in `dev/eval/LEDGER.md`
- [x] 6.2 (DONE 2026-09-20; the retirement commit carries `Fixes #758`) Commit using `Fixes #758` so the issue autocloses
