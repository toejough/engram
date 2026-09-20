## Status: PARKED 2026-09-20

**Done:** please eval fixture + skill-arm and bare baselines (section 1); conversion to a top runbook +
three sub-runbooks with the fidelity report (section 2); D8 bar met on run 3.3d (retrieved 2/3,
followed_all 2/3, end_state 2/3 vs skill row 0/3, 2/3); shim phrase-two re-check drafted and validated
headless (27/30 vs 0/30 clean phrase two); runbooks promoted to the real vault via `engram learn runbook`
as notes 1042-1045 with the P2h situation, retrieval re-checked against production (section 4).

**Not done:** 3a.2 shim deploy (blocked on the `engram update --with-guidance` binary-overwrite
decision: hand-copy vs update); section 5 retirement of `please/SKILL.md` (blocked on reliable
triggering); 2.7 fresh-context fidelity review; 6.x close-out.

**Why parked:** semantic retrieval of the agent's paraphrase missed the please runbook in 1/3 trials
even after the situation rewrite, and misses whenever phrase two names the deliverable. The literal
third-phrase rule (D11, section 3b) was drafted as a fix and measured 0/95 cells of rank/score lift in
the scratch-vault probe; it is dropped and the shim draft reverted to two phrases.

**Unblock:** the new change `runbook-lexical-triggers` ships (a lexical `triggers:` field on runbooks,
matched literally against the user's raw text). Then rerun a 3.3-style shim-only eval with a `/please`
ask, then section 5.

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
- [ ] 3.4 If D8 not met, record actuals, iterate on sub-runbook boundaries / `red_flags` selection, re-run; do not proceed to section 4 until met or Joe redirects

## 3a. Retrieval fix (shim re-check + situation rewrite, D10)

- [x] 3a.1 Draft the phrase-two re-check and one non-dispatch example in `agent-instructions/guidance/shim.md`
- [ ] 3a.2 Deploy via `engram update --with-guidance` and confirm the deployed shim carries the re-check
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

- [ ] 5.1 Run the enumeration grep for `please` across docs, skills, guidance, specs and code; write the per-file disposition list and have a fresh-context reviewer verify it and run an independent discovery pass
- [ ] 5.2 Update live references: `CLAUDE.md`, `README.md`, `agent-instructions/guidance/delegate.md`, `agent-instructions/skills/learn/SKILL.md`, `docs/GLOSSARY.md`, `docs/architecture/{c1-system-context,c2-containers,adr}.md`, `docs/ROADMAP.md`
- [ ] 5.3 Delete `agent-instructions/skills/please/`; run `engram update` and confirm the deployed Claude Code and Pi copies are removed
- [ ] 5.4 Run the real `engram query` with a `/please`-style ask from a non-repo cwd and confirm the top runbook surfaces
- [ ] 5.5 `targ check-full` passes

## 6. Close-out

- [ ] 6.1 Record the eval outcome in `dev/eval/LEDGER.md`
- [ ] 6.2 Commit using `Fixes #758` so the issue autocloses
