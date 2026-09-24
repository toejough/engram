## 1. Eval fixture and skill baseline

- [x] 1.1 Design one representative learn eval task exercising: a correction-kind capture (kind=feedback), a save-request-kind capture (kind=fact), and the write-memory handoff (fetch-and-follow validity gate). Cheap enough for n=3 per arm; document rationale in TASK-RATIONALE.md
- [x] 1.2 Build the fixture (vault-only per curate/write-memory's pattern unless a real repo is genuinely needed — decide and justify) under `dev/eval/cumulative/runbook_vs_skill/phase2/fixtures/learn/`
- [x] 1.3 Hand-build an ideal end state by following the real skill literally with the real binary; confirm `done_when_checks.sh` passes it and at least 8 single-defect mutants each fail
- [ ] 1.4 Estimate eval cost, get Joe's confirmation, then run the skill-only baseline (n=3), record followed_all/end_state as the skill row

## 2. Conversion

- [x] 2.1 Enumerate `learn/SKILL.md`'s sections and tag each carry/sub-runbook/drop-as-shim-floor; write the conversion-fidelity report plan
- [x] 2.2 Write the top runbook (situation, Step 2's four-kind scan + Luhmann placement test, done_when, prioritized red_flags) in the fixture vault, wikilinking write-memory's real basename (`1053.2026-09-22.write-memory-compose-execute-verify`) directly — no fixture placeholder step (design D2)
- [x] 2.3 Write the three sub-runbooks (Step 1+1.5 sweep/vocab; Step 2.5 QA capture, carrying the self-referential duplicate-guard fix verbatim; batch mode `--reparent-luhmann`), wikilinked from the top runbook's body
- [x] 2.4 Prioritize the 8-data-row red_flags table (1,349 B) under the 1200-byte cap (design D3); verify the actual byte count against the built note (`engram show`/truncation code), don't assume; move overflow to the relevant sub-runbook
- [x] 2.5 Decide and set `triggers:` on the top runbook (design D4): score `/learn`, "remember this", "note for next time", "save that", "write this down" against real/synthetic phrases and over-fire probes in a scratch vault (no spend); keep only candidates that clear the bar
- [x] 2.6 Decide whether the batch-mode sub-runbook needs its own `triggers:` (design's Open Question) — default: no, reachable only via the top runbook's wikilink, since it's explicitly-invoked by an orchestrator, not user-triggered
- [x] 2.7 Complete the conversion-fidelity report: every carried/dropped/reworded item with reason, red_flags placement, wikilink targets
- [x] 2.8 Fresh-context reviewer checks the fidelity report against the real SKILL.md for lost content, including the QA duplicate-guard fidelity requirement's own check (spec `learn-adhoc-qa-capture`) — reviewed 2026-09-23, verdict PASS-WITH-FIXES, fixes applied including trigger add-back (remember this / note for next time / write this down restored as deliberate recorded over-fire, matching please/curate precedent); see `learn-conversion-fidelity-report.md`'s 2026-09-23 amendment notes and design.md's D3/D4 amendments

## 3. Validation (gates retirement)

- [x] 3.1 In a trial vault, confirm `engram show <basename>` resolves the top runbook and each sub-runbook, content matches the fidelity report (no-spend)
- [ ] 3.2 Retrieval check: harvest real agent phrases (from the skill-arm baseline transcripts, per vault note 1039 — never idealized situation-shaped phrases) and score against the top runbook's situation/triggers; include over-fire probes for each trigger candidate
- [ ] 3.3 **Before any paid run:** confirm any call-site wording fix reaches the LIVE `agent-instructions/skills/{recall,learn}/SKILL.md` (or the eval's frozen skill copy, whichever the harness actually deploys — verify with `--setup-only` + hash/diff check), not only a design doc or a stale copy — mirrors the write-memory conversion's own costly mixup
- [ ] 3.4 Run the shim-only arm, n=3; validity gate: the transcript shows the correct lesson kind chosen, correct placement decided, and write-memory's runbook actually fetched (`engram show`) before composing — a correct end state without these does not count as a pass
- [ ] 3.5 D8 bar: within one trial of the skill row (both followed_all and end_state) AND the validity gate holds in every passing trial
- [ ] 3.6 If D8 not met: record actuals, root-cause (wording, split boundary, trigger choice), fix, re-run — applying 3.3's hash-check discipline before every re-run; do not proceed to section 4 until met or Joe redirects

## 4. Promotion

- [ ] 4.1 Promote the sub-runbooks to the production vault via `engram learn runbook` first (so wikilinks resolve), then the top runbook; confirm `engram embed status` clean
- [ ] 4.2 Update the real `agent-instructions/skills/recall/SKILL.md`'s any learn-referencing text (if it exists) and any other live reference to name/wikilink the promoted learn runbook's real basename
- [ ] 4.3 Confirm `engram show <basename>` resolves against the real vault from a non-repo cwd, for the top runbook and each sub-runbook
- [ ] 4.4 Confirm `shim.md` is imported in the real `~/.claude/CLAUDE.md` (and `~/.pi/agent/AGENTS.md`) — re-confirm, don't assume

## 5. Retirement and references

- [ ] 5.1 Run the doc-surface enumeration grep for `learn` (as a skill reference) across docs, skills, guidance, specs, and code; write the per-file disposition list to `enumeration.md`; have a fresh-context reviewer independently verify it and run its own discovery pass
- [ ] 5.2 Update live references: `CLAUDE.md`, `README.md`, `docs/GLOSSARY.md`, `docs/architecture/{c1-system-context,c2-containers,c3-components}.md` (skill count drops to one: `recall`)
- [ ] 5.3 Delete `agent-instructions/skills/learn/`; run `engram update` and confirm the deployed Claude Code and Pi copies are removed
- [ ] 5.4 `targ check-full` passes; `openspec validate --all --strict` passes

## 6. Close-out

- [ ] 6.1 Record the eval outcome in `dev/eval/LEDGER.md`
- [ ] 6.2 Commit referencing #760 in prose only (no closing keyword — `recall` remains open scope)
- [ ] 6.3 Archive this change once merged
