## Context

`route-skill-to-runbook`, `please-skill-to-runbook`, `curate-skill-to-runbook`, and `recall-learn-writememory-to-runbook` (all archived) established the pattern: promote a validated runbook body via `engram learn runbook`, gate retirement on a shim-only eval within one trial of the skill row (with an explicit validity gate, not just an end-state match), verify retrieval/discovery in the real vault, confirm `shim.md` is imported, delete the `SKILL.md`, update every live reference. `write-memory`'s conversion (just archived) proved the "worker reached by name" variant; `learn` is the please/curate variant — directly triggered by user words and the shim's re-entry cues, not just named by another skill.

`agent-instructions/skills/learn/SKILL.md` measured today: 307 lines, 21,233 bytes of body (post-frontmatter). Section byte sizes: Step 1 Sweep 1,250 B; Step 1.5 Vocab liveness 2,067 B; Step 2 Crystallize (four lesson kinds) 10,167 B; Step 2.5 Ad-hoc QA capture 2,014 B (carries this session's self-referential duplicate-guard fix, spec `learn-adhoc-qa-capture`); Batch mode `--reparent-luhmann` 2,283 B; Red flags 8 data rows / 1,349 B (table incl. header+separator; re-verified 2026-09-23 fresh-context review — an earlier pass had miscounted the 2 markdown header/separator lines as data rows, giving 10/1,372). `learn` already hands off six write sites to write-memory's real promoted runbook (`1053.2026-09-22.write-memory-compose-execute-verify`), repointed in the prior change — no fixture-placeholder step needed here.

Three specs currently name "the learn skill" as a carrier in requirement bodies (not just Purpose lines, so deltas apply now, not at archive — per the precedent `curate`'s conversion set and the write-memory conversion's independent review confirmed): `learn-branching-disposition` (title and body: "Learn skill decides note placement"), `learn-adhoc-qa-capture` ("within the learn skill's own Step 2 processing"), and `write-memory-worker` ("invoked by a parent skill (`recall`, `learn`)" — `learn` specifically needs updating once it is a runbook; `recall` stays a skill until its own future conversion, so that half of the sentence is unchanged).

`guidance-runbook-follow-frame`'s existing requirement (line 118, added for `write-memory`) already covers "a runbook named by a skill's own instructions" — that requirement is for a runbook reached by *another skill's* text (e.g. `recall` naming `learn`'s runbook, if that ever happens). `learn` itself becoming a runbook is the standard query-matched/triggered case, already covered by the follow-frame's core requirements (announce, restate, `done_when`, `red_flags`, wikilinked sub-runbook fetch). No new follow-frame requirement is expected; confirmed during spec-writing, not assumed.

No `learn` eval exists in `dev/eval/cumulative/runbook_vs_skill/phase2/`. `write-memory`'s conversion cost $4.73 across three attempts because a call-site wording fix was applied to a frozen fixture copy instead of the live skill file the eval harness actually deploys — a hash-check step now exists in that change's tasks and must be reused here before any paid re-run.

## Goals / Non-Goals

**Goals:**
- Promote `learn`'s validated runbook body (top runbook + sub-runbooks) to the production vault, retiring `agent-instructions/skills/learn/SKILL.md`.
- Build a `learn` eval task and skill-arm baseline from scratch, using the same D8 bar and validity-gate discipline every prior conversion used.
- Carry forward the Step 2.5 self-referential QA duplicate-guard fix into the runbook body verbatim — no regression.
- Update the three specs that name "the learn skill" as a carrier in requirement bodies.

**Non-Goals:**
- Converting `recall`. Its glance/deep split (two runbook notes) is decided but not built here; `recall`-to-runbook is its own future change.
- Retiring `agent-instructions/guidance/learn.md` or resolving its redundancy with `shim.md`'s re-entry moments. Waits on `recall`'s own conversion (per the write-memory change's own deferral reasoning — retiring the guidance files is sound only once both `recall` and `learn` are real runbooks).
- Changing `learn`'s actual behavior (the four-kind scan, the placement test, the sweep, vocab liveness, QA capture, batch mode). Only the carrier changes.
- A new schema field or query mechanism. `triggers`/`--text`/whole-word matching already ship.

## Decisions

**D1. Split into a top runbook plus three sub-runbooks, along the skill's own section seams.** Top runbook: `situation`, the Step 2 four-kind scan and Luhmann placement test (the largest section, 10,167 B, and the one every session actually exercises), `done_when`, prioritized `red_flags`. Sub-runbooks, wikilinked from the top runbook's body: (a) Step 1 + Step 1.5 (sweep and vocab liveness — a fixed opening sequence, naturally paired, 3,317 B combined), (b) Step 2.5 (QA capture, 2,014 B, conditionally needed only "when a new substantive Q&A occurred"), (c) Batch mode (`--reparent-luhmann`, 2,283 B, explicitly the skill's own text calls this "NOT part of the normal Step 1→2.5 sequence... runs standalone, once, over a batch" — a genuinely separate entry point, not a step in the main flow). This mirrors `please`'s three-sub-runbook split (gates, lessons audit, doc-surface grep) along conditional/standalone seams rather than an arbitrary byte-count cut. Alternative considered: one runbook, no splits (write-memory's shape) — rejected, Step 2 alone plus a usable `red_flags` set is already close to `please`'s pre-split single-note size, and the batch mode is unconditionally the wrong thing to load on every normal Step 1→2.5 pass.

**D2. Body wikilinks write-memory's real basename directly; no fixture-placeholder stage.** Unlike `write-memory`'s own conversion (which had to invent a temporary fixture basename because write-memory wasn't yet promoted when its call sites were first written), write-memory is already live in production as `1053.2026-09-22.write-memory-compose-execute-verify`. Every write site in the new `learn` runbook (and its sub-runbooks) references this real basename from the start, in both the fixture-vault version used for eval and the eventual production promotion. This removes one whole class of the write-memory conversion's own failure mode (the two wasted paid attempts that re-tested a fix applied to the wrong copy) — there is no "temporary basename" to forget to repoint.

**D3. `red_flags` prioritization under the 1200-byte cap, mirroring `please`'s D3.** The skill's 8-data-row table (1,349 B) exceeds the cap on its own. Keep on the top runbook: the two highest-value, most session-frequent rows (skipping the sweep; over-capturing routine successes/bare "thanks") and the just-shipped QA self-referential duplicate-guard row (a live, previously-shipped-and-fixed defect — the highest-cost row to lose). Move sub-runbook-specific rows (episode-workflow reconstruction → Step 1 sub-runbook; QA-specific rows beyond the duplicate guard → Step 2.5 sub-runbook; `--tier`/L3 writing, a batch-mode-adjacent historical footgun → wherever it fits) to their relevant sub-runbook's own `red_flags`/body. Exact row-by-row placement is a task-level activity (verify actual byte counts against the built note; don't assume, per write-memory's own D1 caution that byte assumptions must be checked, not guessed).

**D4. `triggers:` set, verified against over-fire before acceptance.** Candidates from `learn`'s firing language and `guidance/learn.md`: `/learn`, "remember this", "note for next time", "save that", "write this down". Each candidate is scored against real/synthetic phrases and over-fire probes (ordinary prose containing "remember"/"note"/"save" in non-capture contexts) before being kept, exactly as `please`/`curate` scored their candidates — whole-word matching (shipped) prevents substring collisions but not semantic over-fire from a common word used in its ordinary sense. If a candidate over-fires unacceptably, drop it rather than accept it by default; `/learn` alone is a safe floor if the others don't clear the bar.

**D4 amendment (2026-09-23, task-2.8 fresh-context review).** The task-2.4/2.5 execution dropped all four
multi-word candidates against tailored over-fire probes, keeping only `/learn`. A fresh-context review found
this over-strict relative to this repo's own shipped precedent for deliberate single-word/short-phrase
triggers: `runbook-lexical-triggers` D9(c) ("Joe added the plain word `please` as a third trigger ... This is
a deliberate exception to the write-memory authoring rule against generic single-word triggers. Over-firing
is accepted: a hit is only a candidate, and the agent decides relevance from the runbook's own applicability
guard.") and `curate-skill-to-runbook` D3 ("`curate` fires on 'curate a playlist'/'curate the docs' ... a
trigger hit is a candidate only, the agent judges it against the situation; documented ... as the accepted
trade for a deliberate invocation word."). The reviewer's reproduction confirmed the concrete cost of the
over-strict cut: with only `["/learn"]`, `engram query --text "Remember this: always tag sanitation notes."`
— the skill's own canonical Kind-2 example phrasing — returned `items: []`. A missed capture is learn's
highest-cost failure mode. Re-scored per D4's own asymmetry (a trigger hit is only a candidate the agent
still judges): "remember this" (the skill's own documented example phrase — excluding it was indefensible),
"note for next time", and "write this down" are added back as a deliberate, recorded over-fire acceptance;
"save that" stays dropped, since it is a generic verb+demonstrative with no capture-specific signal, unlike
the other three which each name a distinctive capture-intent action. Final `triggers:` = `["/learn",
"remember this", "note for next time", "write this down"]`. Full re-verification (over-fire probes accepted,
the "Remember this" fix confirmed, `red_flags` budget unaffected) is recorded in
`learn-conversion-fidelity-report.md`'s "Trigger scoring" section.

**D5. Eval built from scratch, mirroring please/curate's decision, with the write-memory cost lesson applied.** No `learn` eval exists. Build an explicit-ask task exercising at least one of each: a correction-kind capture, a save-request-kind capture, and the write-memory handoff (fetch-and-follow validity gate, per D4 of the write-memory change). Skill-arm baseline first (paid, cost confirmed), then shim-only. **Before any paid re-run following a wording fix, verify via hash-check that the fix reached the live skill file the harness actually deploys, not only a frozen fixture copy** — the write-memory conversion burned $2.24 re-testing the same broken wording because this check was skipped once.

**D6. Retirement gate is the same three-part bar** every prior conversion used: (a) D5's eval bar met, (b) retrieval verified against the real vault, (c) `shim.md` confirmed imported (already true, re-confirm don't assume).

**D7. Spec deltas fix carrier language now, not at archive.** `learn-branching-disposition` and `learn-adhoc-qa-capture`'s "the learn skill" wording lives in requirement bodies, not Purpose lines, so a delta applies now (curate's and write-memory's own conversions set this precedent — a fresh-context review on write-memory's change specifically flagged that Purpose-only deferral is wrong for requirement-body claims). `write-memory-worker`'s "parent skill (`recall`, `learn`)" line needs `learn` updated to "runbook" while `recall` stays "skill" until its own conversion — a partial-sentence carrier fix, verified precisely, not a blanket find-replace.

## Risks / Trade-offs

- **[Risk] Sub-runbook split boundaries chosen by section seam, not measurement, could still leave the top runbook too large or a sub-runbook awkwardly thin.** → Task list requires measuring the actual built note's byte size against the real truncation/preview rules before declaring the split final, per write-memory's D1/D3 caution.
- **[Risk] `triggers:` candidates over-fire on common words ("remember", "note", "save").** → D4's scoring step; drop rather than force-accept a candidate that fails.
- **[Risk] Repeating write-memory's fixture-vs-live-file mixup.** → D5's explicit hash-check requirement before any paid re-run; call out in tasks.md as its own checklist item, not assumed knowledge.
- **[Trade-off] New eval spend on a task with no existing baseline.** → Estimate and confirm cost before running, per project standard.
- **[Risk] Missed doc references after retirement.** → Doc-surface enumeration plus an independent fresh-context reviewer before deletion, per every prior conversion's established pattern in this session.

## Migration Plan

1. Build the eval task and skill-arm baseline; confirm it's a usable D8 reference before building anything else.
2. Write the top runbook + three sub-runbooks in the fixture vault, from the real `SKILL.md`, wikilinking write-memory's real basename directly (D2); write the conversion-fidelity report.
3. Fresh-context reviewer checks the fidelity report against the real `SKILL.md` for lost content, before any paid validation spend.
4. Run the shim-only eval arm against the baseline (D5); apply the hash-check discipline before any wording-fix re-run.
5. If D8 met: promote to the production vault via `engram learn runbook` (sub-runbooks first so wikilinks resolve); update the real `recall`/`learn`-adjacent references.
6. Doc-surface enumeration; independent fresh-context review of the enumeration; perform every update/rewrite row.
7. Delete `agent-instructions/skills/learn/`; run `engram update`; `targ check-full`; commit, referencing #760 in prose (not closing it — `recall` remains open).
8. Archive the change; sync specs.

Rollback: revert the retirement commit restores the skill; the promoted runbook notes can be removed via `engram` note deletion.

## Open Questions

- Exact final `red_flags` row placement (D3) — resolved at task time against real byte counts, not decided here.
- Whether the batch-mode sub-runbook needs its own `triggers:` (e.g. a distinct cue for `--reparent-luhmann` invocation) or stays reachable only via the top runbook's wikilink — decide during task 2.x once the split is built; batch mode's own text says it's "explicitly-invoked," which argues against a lexical trigger of its own (nobody types a trigger phrase for it — it's invoked by an orchestrator handing over a specific payload).
