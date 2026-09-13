## 1. Schema: runbook `red_flags` field

- [ ] 1.1 TDD: runbook note parse/render supports optional `red_flags` list in frontmatter (internal note kind, sidecar unaffected)
- [ ] 1.2 TDD: `engram learn runbook --red-flag <text>` (repeatable) writes the field; absent flag writes no field
- [ ] 1.3 TDD: `engram query` item content and `engram show` render `red_flags` for runbook notes
- [ ] 1.4 `write-memory` SKILL.md runbook compose block passes one `--red-flag` per handed-off entry (writing-skills TDD)
- [ ] 1.5 Settle the field name (`red_flags` vs `watch_for`) and record in design.md Open Questions

## 2. Shim: the follow frame

- [ ] 2.1 RED baseline: shim-only config (no engram skills; recall+learn as runbooks in the vault; current guidance in CLAUDE.md) on history-rewrite, runbook arm, n=3 — record found / restated / every-step / end / question-stops
- [ ] 2.2 Draft the four-part shim as a NEW file `agent-instructions/guidance/shim.md` (recall.md untouched): bootstrap `engram query`, treatment of each returned kind with the matched-runbook frame (announce, restate as plan/todos, done_when bar, stop-and-ask, red_flags, transitive wikilinks, `engram show` on truncation), general behavioral floor, four re-entry cues phrased as `engram query` actions — wording per vault notes 137/1030/1031
- [ ] 2.3 GREEN: same fixture and n as 2.1 with shim.md as the ONLY guidance in the trial CLAUDE.md (plus the probe marker); pressure tests against the rationalizations the skill-arm carrier named; iterate wording until the pre-registered bar holds or a question-stop finding amends the shim
- [ ] 2.4 Deploy: `engram update --with-guidance` syncs `shim.md` to `~/.claude/engram/guidance/shim.md` (update's owned-roots list gains the file; TDD in `internal/update`); confirm deployed copy matches source

## 3. Recall and learn as runbooks

- [ ] 3.1 Convert `agent-instructions/skills/recall/SKILL.md` into a top runbook plus wikilinked sub-runbooks (glance/deep procedure, coverage judgment, activation), bodies verbatim where possible; record fidelity deltas
- [ ] 3.2 Convert `agent-instructions/skills/learn/SKILL.md` the same way
- [ ] 3.3 Retrieval check: with only the shim, do the task-phrase and situation-phrase queries surface the recall runbook top-ranked on the eval fixtures (no spend: `engram query` against the trial vault)

## 4. Eval: shim-only comparison

- [ ] 4.1 Harness: `--shim-only` config option in `probe_phase2.py` (config dir without engram skills; recall/learn runbooks added to the trial vault; trial CLAUDE.md = `shim.md` + probe marker and nothing else, replacing the recall.md-plus-fifth-cue composition for that arm); scorer signal for "restated steps as plan before first mutating command"; question-stops reported as clarity findings, not failures
- [ ] 4.2 Move the history-rewrite skill carrier's red flags into the runbook carrier's `red_flags`; rebuild the fact carrier preserving `tags:`; re-verify zero-delta bodies
- [ ] 4.3 Stage 1: history-rewrite R and F, n=3, shim-only; validate (mid tier) against the skill row; amend shim on any question-stop finding
- [ ] 4.4 Stage 2: bisect-before-fix and route R and F, n=3, shim-only; validate
- [ ] 4.5 Decision: runbook vs fact vs skill, per the pre-registered bar; LEDGER row; ANALYSIS.md update

## 5. Close

- [ ] 5.1 `openspec validate --strict`; sync specs; archive
- [ ] 5.2 File follow-on change for converting route/please/curate if the frame holds; #751 stays open for capture-time red flags
