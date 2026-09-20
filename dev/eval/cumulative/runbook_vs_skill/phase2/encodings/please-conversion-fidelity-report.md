# Conversion-Fidelity Report: `please/SKILL.md` -> Please-R runbook set

OpenSpec change `please-skill-to-runbook`, tasks 2.1 (plan) and 2.6 (final). Format mirrors `conversion-fidelity-report.md`
(route/commit conversions). Revised after the fresh-context review (task 2.7): LESSONS-in-report reverted, resolution rule / declarative-challenge clause restored, rows 8 and 11 kept as flags, escalation provenance one-liner in top.

- **Source**: `agent-instructions/skills/please/SKILL.md` (156 lines; frozen byte-identical copy at `encodings/taskPlease/Please-S/skills/please/SKILL.md`).
- **Encoding** (`encodings/taskPlease/Please-R/vault/`, all `type: runbook`, each with a `.vec.json` sidecar):

| Note | Role | Bytes | `red_flags` bytes (cap 1200) |
|---|---|---|---|
| `6.2026-09-19.please-drive-ask-end-to-end` | top runbook (situation, spine, done_when) | 12866 | 1163 |
| `3.2026-09-19.please-adversarial-review-gates` | sub-runbook (gates A-D) | 6354 | 894 |
| `4.2026-09-19.please-step7-lessons-audit` | sub-runbook (audit + LESSONS handoff) | 4932 | 889 |
| `5.2026-09-19.please-step3-doc-surface-enumeration-grep` | sub-runbook (Step 3 grep) | 2265 | 547 |

`red_flags` bytes = the exact `    - ...\n` block that `capRedFlagsForPreview` measures (`internal/cli/redflags_truncation.go`,
budget 1200). Verified two ways: byte count of the raw block, and `engram show --vault <fixture>` on the top note prints no
`EARLIER RED_FLAGS OMITTED` marker. Pinned by `test_please_r_red_flags_fit_the_redflags_preview_budget_in_every_note`.

**Numbering note.** Luhmann prefixes are 3/4/5 (subs) and 6 (top), not 1/1a/1b/1c. `add_carrier` reports the LAST `.md` in
sorted order as `carrier_basename`, and the harness scores "found" on it; the top runbook must therefore sort last. (`1.` would
sort before `1a`.) Production promotion (task 4.x) assigns real ids via `engram learn runbook`, so this is fixture-only.

## Section plan and disposition (2.1)

Tags: **carry** = in the top runbook; **sub** = moved into a sub-runbook; **drop** = removed as shim floor (see shim map).

| # | SKILL.md section | Tag | Where / what changed |
|---|---|---|---|
| S1 | Frontmatter `description` (trigger discrimination) | carry, reframed | Becomes the top `situation:` (D5), REFRAMED by qualifying situations: the ceremony-vocabulary situation (seven-step workflow, /learn, /recall, "when the user says /please") failed retrieval in the shim-only run (never retrieved; agents phrase queries as the concrete work). The `situation:` is now PROCESS-only (D10 / 3a.5, revised 2026-09-20 to P2h): 'implementing, renaming, fixing, restructuring, or rewriting documentation for a codebase and deciding how to verify, review, and land the change before calling it done'. P2h replaced the hybrid P2j and the earlier work-class C2 because P2j and C2 carry the eval task's own example (a breaking CLI-flag rename), which is test-fitting; P2h names no example from the eval task. Caveat: P2h misses 3 of the 7 real please-rename phrase twos, the ones that still name the flag. See results/3.1 'Process-shaped situation (3a.5)'; no ceremony vocabulary, no exclusion clause (embeddings do not encode negation). The `/please <ask>` and single-edit exclusion live only in the body's Required-argument section. "An `<ask>` is required" also stays there. |
| S2 | Intro paragraph (seven steps, `/recall` `/learn`, capability-named skills, meta-orchestration, delegate per route) | carry, reworded | Verbatim except "This skill" -> "This runbook" and "per the `route` skill" -> "per the route runbook" (route is already a runbook; NOT wikilinked, because the production route note is luhmann 1036 and the trial vault excludes luhmann >= 955, so a link would dangle). Adds the three-link index to the sub-runbooks (body only, D2). |
| S3 | Anti-sycophantic lean (5 bullets) | carry, condensed into Step 2 | Step 2's assessment bullet now also carries the "declarative sentences with concrete stakes, no leading question, no reflexive praise" clause and the resolution rule ("challenge once; if reaffirmed, proceed and record the dissent in the plan: considered X; user chose Y because Z"), because shim.md has no analogue. Anti-displacement kept as a top red_flag (see rows 8/11 below). |
| S4 | Adversarial review gates: intro + non-waivable/routed paragraph | sub (gates) | Verbatim; "the `route` rubric" -> "the route runbook's rubric". |
| S5 | Gate table A-D | sub (gates) | Verbatim. |
| S6 | Seven angle charges | sub (gates) | Verbatim (docs/diagrams-alignment keeps the "verify AND still run own independent discovery" clause required by spec `please-doc-enumeration-gate`). |
| S7 | Reviewer protocol (6 points) | sub (gates) | Verbatim, except point 1: "the `route` skill's recall-first rule" -> explicit "its FIRST action is `/recall` (or `engram query`)... Say so explicitly in every reviewer dispatch prompt." (clarification; behavior identical, the route skill it cited is retired). |
| S8 | Escalation provenance | sub (gates) full + one-liner in top | D4 said drop. NOT dropped: shim.md has no rule about evidence pointers for measured claims. Detail in the gates sub-runbook (paragraph + red_flag); a one-sentence version sits in the top runbook's intro because it applies to any mid-cycle escalation, not only gate ones. |
| S9 | Required argument | carry | Verbatim in a `## Required argument` section (one `AskUserQuestion`, no other action). |
| S10 | Task tracking | carry, condensed | One sentence in the workflow preamble ("push all seven steps... in a single call, mark in_progress / completed"). The "one todo per step" content is also shim step 2. |
| S11 | Workflow steps 1-7 | carry | Seven-step spine kept verbatim; changes listed below (W1-W5). |
| S12 | Step 3 doc-surface enumeration grep | sub (grep) | Verbatim paragraph moved to sub-runbook 5; top Step 3 keeps a compressed pointer with the `[[link]]` and the "paste the list into the plan" requirement. |
| S13 | Step 4 `LESSONS:` running-list paragraph | carry | Kept in the top Step 4 as a bold "The LESSONS: contract" block (W4). |
| S14 | Step 7 lessons audit (corpus, mapping, kind-3/kind-4 remarks) | sub (audit) | Verbatim in sub-runbook 4; top Step 7 compressed to a pointer plus the wikilink requirement. |
| S15 | Step 7 "which artifact should have surfaced it" paragraph | sub (audit) | Verbatim. |
| S16 | Step 7 "Then: run learn again, hand it LESSONS + audit" | sub (audit) + carry | Sub-runbook 4 `The LESSONS: handoff` block (W5); top Step 7 and `done_when` restate it, as SKILL.md does (list goes to `/learn`; only the audit list goes in the report). |
| S17 | Stop conditions: "N/A is a high bar" | carry, condensed | Kept as a short `## When a step seems not applicable` section rather than dropped (shim floor covers the reread/substitution part but not the "name the missing mechanism in the task description" rule). Flagged. |
| S18 | Stop conditions: sequence is part of the workflow | carry | Verbatim into the workflow preamble; step-1-before-step-2 sentence strengthened (W1). |
| S19 | Stop conditions: multi-step test for ambiguous triggers | carry | In `## Required argument` (single edit/tool call -> handle directly; unclear -> use the workflow; `/please` slash form always applies). Not duplicated in `situation:` (positive trigger phrasing only; see S1). |
| S20 | Stop conditions: user interrupts/redirects | carry | Last sentence of the workflow preamble, verbatim. |
| S21 | Red Flags table (21 rows) | split | See placement table below. |

### Reworded or added items in the spine (behavior unchanged)

| ID | Item | Reason |
|---|---|---|
| W1 | Step 1 now states "This is the FIRST action of the workflow, before ANY `/recall` or `engram query`... Step 1 must be `completed` before step 2's `/recall` begins." | Baseline (`results/1.3_please_baseline_sonnet5.md`): skill-arm agents ran `/recall` before the opening `/learn`, breaking the `after` chain for steps 1->2->3. The ordering was already normative (SKILL.md "Sequence is part of the workflow"); this makes it hard to miss. Also a top red_flag and a `done_when` clause. |
| W2 | Step 2 keeps the literal-`/recall` block verbatim; the "per the anti-sycophantic lean" phrase is cut | Lean section is dropped (S3). |
| W3 | Step 3 restates grep inline in one sentence and links sub-runbook 5 | Body-only wikilinks (D2); grep stays non-waivable. |
| W4 | Step 4 `LESSONS:` block adds "write that requirement into every unit's dispatch prompt" | Baseline: step 19 (dispatch handoffs carry the `LESSONS:` contract) failed 3/3 skill trials. The contract already says "every dispatched unit's completion report ends with a `LESSONS:` line"; the added clause names the mechanism that makes that true. |
| W5 | REVERTED to SKILL.md behavior. Earlier draft made the closing report END with the full `LESSONS:` list; the reviewer confirmed SKILL.md hands the list to `/learn` and puts only the audit list in the report, and the extra requirement is unspecced. Now: audit list in report; running LESSONS list handed to `/learn`. `LESSONS:` in dispatch prompts (W4) stays, spec-neutral per Route-R. | Review fix 1. |
| W6 | Wikilink syntax stated emphatically in sub-runbook 4 (bold paragraph), top Step 7, top `done_when`, and one red_flag in the top and sub 4 (red_flags describe it in words; the literal `[[note-basename]]` placeholder is body-only) | Design D2 / task 2.5; route eval found agents writing structured references as plain text. New content by design (SKILL.md never asked for wikilinks); spec `write-memory-worker` delta requires it. |
| W7 | Added top-level index list of the three sub-runbook wikilinks | Wikilinks only in the body, so the shim's transitive-follow (frame item 6) reaches them. |
| W8 | Gate-dispatch prompts must say "recall first" (S7) and gate FAILs/escalations noted as they happen (last line of sub 3) | Small clarifications so the audit corpus is collected before step 7; both already implied. |
| W9 | DELIBERATE FIDELITY DELTA (carrier-shape adaptation): Step 1 and the closing `/learn` in Step 7 say what to do when `learn` is not an installed skill but a vault runbook: retrieve it with `engram query` (retrieving it counts as part of step 1, so it is exempt from the learn-first ordering rule) and follow it, or its wikilink if given; if no learn runbook exists, run the equivalent `engram` commands (`engram ingest --auto`, the sweep SKILL.md's learn Step 1 runs) or mark N/A naming the missing mechanism; do not stop to ask. Red_flag 1 is narrowed from 'any /recall or engram query' to 'any /recall or engram query about the ask' to stay consistent. No wikilink literal was added. | SKILL.md assumes `learn` is an installed skill; the shim-only config has it only as a runbook, and one 3.3c trial (R-1) read the runbook and then stopped to ask at step 1. This delta exists only because the carrier changed from skill to runbook; the skill's behavior is unchanged. Nothing was added that SKILL.md does not already do (the sweep command is the learn skill's own step 1). |

## Shim-floor drops (D4), each mapped to the shim.md rule that covers it

| Dropped content | shim.md rule | Coverage verdict |
|---|---|---|
| Anti-sycophantic lean: "evaluate the ask on its merits", "challenge plainly and directly" | Frame item 5 (stop and ask when uncertain; a question stop is a clarity signal) | **Partial.** Item 5 covers asking on genuine uncertainty, not "state a flaw declaratively". Step 2's own bullet still says state your assessment and raise challenges NOW. |
| Resolution rule (challenge once, then commit; record dissent) | none | **UNCOVERED, so RESTORED** in Step 2 (review fix 2). |
| Anti-displacement / settled decision (row 11) | Floor rule 2 (substituting a related action is a skip) | **Partial.** The floor does not say the asked task is settled or that deviation needs a NEW fact stated as a reversal. Kept as top red_flag #6. |
| Praise-before-analysis opener; softening a challenge into a hinting question | none | **UNCOVERED, so RESTORED** as the "declarative sentences, concrete stakes, no reflexive praise" clause in Step 2. |
| Escalation provenance paragraph | none | Kept (S8): full in gates sub-runbook, one-liner in top. |
| "The user cannot waive steps" (row 8) | Floor rule 1 + frame item 2 (steps become the plan) | **Partial.** The floor covers self-shortcutting, not an explicit user "skip it" instruction. Kept as top red_flag #5. |
| "N/A is a high bar" | Floor rules 1-2; frame item 5 ("a step names a target that plainly isn't present" = the legitimate N/A) | Good fit for the rule; the "write the mechanism into the task description" instruction is kept (S17). |
| Task tracking: one todo per step | Frame item 2 | Exact match. |
| "Do not skip / don't do it from memory" rows: `/recall` substituted by file reads | Floor rule 2 (substitution is a skip) | Covered in principle, but the recall-vs-file-reads distinction is please-specific, so it stays as a top red_flag and in Step 2 body. |

## Red Flags table: placement of all 21 rows (D3)

| # | SKILL.md row (short) | Placement | Reason |
|---|---|---|---|
| 1 | Start working without step 1 `/learn` | TOP red_flag #1 (merged with ordering, W1) | Baseline failure; highest severity. |
| 2 | Skipped `/recall` (already know, diff is here, grep) | TOP #2 | please-specific substitution; frequent. |
| 3 | Writing code before plan committed | TOP #3 (+ gate A closure) | Gate-order violation. |
| 4 | Skipping RED ("too simple") | body only (Step 4 RED bullet, now with the "too simple to test is not an exemption" counter) | Generic TDD rule; budget. |
| 5 | Declared unit done without verifier | body (Step 4 "verify before declaring any unit done") + `done_when` clause "every unit was verified by running the actual commands and reading their output" (added in review) | Shim frame item 4 makes `done_when` the completion bar. |
| 6 | Ending without step-7 closing `/learn` | TOP #7 (merged with row 19) | `done_when` also requires it. |
| 7 | No ask, started anyway | body only (`## Required argument`) | Situation-level; the single-question rule is first in the body. |
| 8 | "No ceremony / skip the plan / hurry" | TOP #5 (Partial coverage) | Review fix 3; see shim map. |
| 9 | Marked N/A because "wouldn't produce anything" | body (`## When a step seems not applicable`) | Kept in body, not a flag (shim rules 1, 2, 5). |
| 10 | Noticed a problem with the ask, planned around silently | body (Step 2 assessment bullet + resolution rule) | |
| 11 | Recommending prerequisite/"the real blocker" instead of the task | TOP #6 (Partial coverage) | Review fix 3; see shim map. |
| 12 | Reply opens with praise | body (Step 2 clause) | Restored. |
| 13 | Softened challenge into hinting question | body (Step 2 clause) | Restored. |
| 14 | Skipped a review gate ("obviously fine") | TOP #4 (merged with row 15 and grep skip) + gates sub-runbook flag 1 | please-specific, severe. |
| 15 | Batched angles into one reviewer | TOP #4 (merged) + gates sub flag 2 | |
| 16 | Reviewer dispatched without `/recall` first | gates sub-runbook flag 3 (+ protocol point 1) | Conditional to the gate step. |
| 17 | Resolved a finding by silently dropping it | gates sub flag 4 | |
| 18 | Argued past ~2 rounds without escalating | gates sub flag 5 | |
| 19 | Closing without step-7 lessons audit | TOP #7 (merged with row 6) + audit sub flag 1 | |
| 20 | Step 7 without a running LESSONS list | TOP #6 + audit sub flag 2 | Baseline failure (step 19). |
| 21 | Measured claim without evidence pointer | gates sub flag 6 | Escalations happen in the gate protocol. |
| new | Doc-surface grep skipped because surface is small | TOP #4 (merged) + grep sub flags 1-3 | Not a row in SKILL.md (the rule is body text); added because design D3 lists it. |
| new | Stopping to ask "should I proceed?" between steps | TOP #5 | Not a SKILL.md row; design D3 lists phase-continuation-as-ambiguity; mirrors shim frame item 5 clause. |
| new | Wikilink reference written as plain text | TOP #8, audit sub flag 3 | Design D2 / task 2.4. |
| new | Pre-filtering LESSONS lines before handoff; writing a duplicate note instead of rewording | audit sub flags 4-5 | Restate body rules that were prose in SKILL.md. |

Top `red_flags` = 9 entries, 1163 bytes (1149 before the W9 edit) (<=1200; re-measured after the review edit, `engram show` prints no omitted marker). To fit
rows 8 and 11 (which the shim floor covers only partially), re-prioritized: shortened the recall and gate-skip flags, merged
the no-LESSONS-list flag with the no-audit / no-closing-learn flag (rows 6, 19, 20). Step-1 ordering flag still leads, wikilink flag is last.
No `[[...]]` appears in any `red_flags` or `done_when` (the bracketed placeholder would make a shim-frame-6 agent `engram show note-basename`).

## Deliberately NOT changed

- `steps.json` (19 steps) and the skill/bare scoring: unchanged.
- No fixture-specific hints in the runbooks (no mention of flags, CHANGELOG, completion scripts, or `--out`).
- The note bodies (only) contain the literal `[[note-basename]]` as the syntax placeholder; `engram show` lists it as an unresolved outbound link
  (`note-basename`). Route-R does the same. Never in `red_flags`/`done_when`.

## Wikilink-citation scoring (harness limitation)

`evaluate_steps` has no per-arm step filter (no `arms`/`only_arms` key in `steps.json`; every step is scored in every arm). A
`[[wikilink]]` audit-citation step would fail the skill row by construction, so it was NOT added. Options for the caller: (a) add an
`arms` key to `_evaluate_signal`/`evaluate_steps` (small harness change, then score it on R only); (b) score wikilinks by an ad hoc
regex over the R-arm results; (c) leave it unscored and rely on the validity gate. Left for a decision.
