## Context

`route-skill-to-runbook` (archived 2026-09-19) is the only skill retired into a runbook so far: its
validated eval fixture (`Route-R`) was promoted to the production vault via `engram learn runbook`,
gated on a shim-only D8 probe and on `shim.md` being confirmed imported in the real CLAUDE.md, then
`SKILL.md` was deleted (commit 87e9a006) and `engram update` sync removed the deployed copies.
`guidance-runbook-follow-frame` is the generic contract for matched runbooks (announce, restate steps,
`done_when` bar, `red_flags` stop, follow wikilinked sub-runbooks). `red_flags` output is capped at 1200
bytes by `internal/cli/redflags_truncation.go` (newest entries kept, omission marker added); no per-item
cap exists on runbook `body`.

`agent-instructions/skills/please/SKILL.md` today: 156 lines, ~3,400 words. Sections: anti-sycophantic
lean, adversarial review gates (table A-D, seven angle charges, six-point reviewer protocol), escalation
provenance, required argument, task tracking, seven-step workflow (Step 3 carries the non-waivable
doc-surface enumeration grep; Step 7 the lessons audit), stop conditions, and a ~22-row "Red flags —
STOP" table. It references `/recall`, `/learn`, route (already a runbook), capability-named skills
(brainstorming, plans, TDD, verification, commit), `AskUserQuestion`, `TaskCreate`, and route's
`LESSONS:` completion-report contract.

No `please` eval task or skill-arm baseline exists in `dev/eval/cumulative/runbook_vs_skill/phase2/`.

## Goals / Non-Goals

**Goals:**
- Replace `please/SKILL.md` with a top runbook plus wikilinked sub-runbooks that carry its
  task-specific procedure, in the production vault.
- Validate the conversion with a shim-only eval against a skill-arm baseline, using the same D8 bar
  route used, before retiring the skill.
- Keep every live doc/spec reference accurate after retirement.

**Non-Goals:**
- Changing please's behavior (steps, gates, lessons-audit rules, enumeration grep). Only the carrier
  changes; the delta specs update carrier language only.
- Converting curate (#759), recall/learn/write-memory (#760), or the openspec skills (#761).
- Re-deriving general behavioral-floor rules the shim already states uniformly.
- Changing `guidance-runbook-follow-frame`.

## Decisions

**D1. Split into a top runbook plus sub-runbooks along existing seams.** Top runbook: `situation`,
seven-step spine, `done_when`, prioritized `red_flags`. Sub-runbooks (linked from the top runbook
*body* with `[[basename]]`): (a) adversarial review gates (gate table, angle charges, reviewer
protocol), (b) Step 7 lessons audit, (c) Step 3 doc-surface enumeration grep. These are the sections
that already have their own normative specs and are only conditionally needed (the grep fires only
when a plan alters a repeated invariant), so retrieving them lazily via wikilink is a natural fit.
Alternative considered: a single runbook body. Rejected — ~3.4k words in one retrieved item risks
truncation and buries the conditional procedures; the recall/learn conversion set the precedent for
splitting.

**D2. Wikilinks only in bodies; state the syntax requirement emphatically.** Route's eval found agents
writing a structured-field reference as plain text instead of `[[wikilink]]`. Please's runbook writes
wikilinks in the body (sub-runbook links) and asks agents to write them in the lessons-audit output
(citing vault notes). Body links get the shim's transitive-follow; `red_flags`/`done_when` links would
not. The top runbook carries one `red_flags` entry naming the plain-text-instead-of-`[[...]]` condition,
plus emphatic statement of the syntax at each place the agent writes a reference.

**D3. Prioritize `red_flags` under the 1200-byte cap.** The ~22-row table cannot fit. Rank rows by
(failure severity × how specific to please, not the generic floor): keep the please-specific
skip-a-gate / skip-the-grep / self-review / phase-continuation-as-ambiguity rows in `red_flags`; move
the remainder into the relevant sub-runbook's own `red_flags` or body, so a failure mode is reachable
where it applies. Record which row went where in the fidelity report. Alternative: raise the cap.
Rejected — the cap exists to keep Bash output under Claude Code's ~2KB truncation (f5b44504);
weakening it reintroduces silent truncation.

**D4. Drop what the shim already says; record each drop.** Anti-sycophantic lean, escalation
provenance, "N/A is a high bar", and "user can't waive steps" duplicate `shim.md`'s floor. They are
removed from the runbook and listed as fidelity deltas so a reviewer can confirm each is genuinely
covered rather than lost.

*Amendment (post-review, as actually implemented).* Kept: escalation provenance (full in the gates
sub-runbook, one-sentence version in the top runbook, since it applies to any mid-cycle escalation);
the "N/A is a high bar" rule (condensed section, because the "name the missing mechanism in the task
description" instruction has no shim analogue). Restored after the fresh-context review: the resolution
rule (record the dissent in the plan) and the declarative-challenge / concrete-stakes / no-praise clause,
both in Step 2; rows 8 (user cannot waive steps) and 11 (settled task, deviate only on a NEW fact stated
as a reversal) survive as top red_flags because the shim floor covers them only partially. Dropped only
where the shim genuinely covers it (one todo per step; generic reread-the-step rules).

**D5. Situation line carries trigger discrimination.** The SKILL.md `description` fires on `/please
<ask>` and end-to-end phrasings and explicitly not on casual "please" attached to one trivial action.
With no skill description, the runbook's `situation:` is the only discriminator retrieval sees. Its
wording follows the runbook-`situation` shape ("<verb>-ing <object> in <context>") and is validated by
the retrieval check (task+situation phrases surface it top-ranked; a casual-"please" phrase does not).
Outcome (3.1): the original trailing exclusion clause ("never for a casual please ...") was removed. Embeddings do not encode negation, so the clause attracted the excluded phrases (top runbook surfaced, rank 1 for "casual please"). The situation is positive-only; the single-action guard lives in the body's Required-argument section.
Outcome (shim-only run 3.3): the ceremony-vocabulary situation (seven-step workflow, /learn, /recall, "when the user says /please") was never retrieved -- real agents phrase `engram query` as the concrete work ("landing a breaking rename of a CLI flag end-to-end ..."), never the ceremony. Reframed per Joe by WHAT SITUATIONS QUALIFY: "landing a change end-to-end in a repository that touches code, tests, and documentation and must be verified, reviewed, and committed properly, such as a breaking rename of a CLI flag, a field rename across docs, or a feature that needs tests and docs". Validated against the real agent phrase pairs (results/3.1_please_retrieval_check.md, section "Qualifying-situation reframe").
Outcome (3a.5, after the D10 shim re-check): the work-class-only situation was tuned to the OLD deliverable-laden phrase two; with the fixed shim, agents write phrase two as a process decision ("carrying out a mechanical rename across a repo and deciding how to verify it is complete before calling it done"), which a process-only situation matches better on synthetic phrasings but which still misses some real please-rename phrasings. The chosen situation is a hybrid: "landing a change end-to-end in a repository that touches code, tests, and documentation and deciding how to verify, review, and commit it before calling it done, such as a breaking rename of a CLI flag, a field rename across docs, a bug fix with tests and docs, or a feature that needs tests and docs". Chosen over 10 other candidates by scoring 7 real please-rename phrase twos (as written by agents in the RED/GREEN/3.3c runs), 5 real GREEN phrase twos for other tasks, 6 synthetic process-shaped phrase twos and 7 over-fire probes (results/3.1, section "Process-shaped situation (3a.5)"): 7/7 real please-rename, 2/5 other-task, 3/6 synthetic, 0/7 over-fire. A process-only wording (P2h) had 4/7, 2/5, 6/6, 0/7. Known weakness: bug-fix, refactor and docs-overhaul phrase twos mostly still miss; the shim-only please run is the deciding test.
Outcome (2026-09-20 revision): the situation is now the process-only P2h, "implementing, renaming, fixing, restructuring, or rewriting documentation for a codebase and deciding how to verify, review, and land the change before calling it done". P2h was chosen over P2j and C2 because P2j and C2 carry the eval task's own example (a breaking rename of a CLI flag), which is test-fitting; P2h is process-only and names nothing from the eval task. Caveat: P2h misses 3 of the 7 real please-rename phrase twos (the ones that still name the flag), scoring 4/7 real, 2/5 other-task, 6/6 synthetic, 0/7 over-fire. Also fixed in the same pass: the top runbook's step 1 and closing /learn now say how to proceed when learn is a vault runbook rather than an installed skill (fidelity report W9).

**D6. Build a please eval task + skill-arm baseline (in scope).** Reuse the phase2 harness
(`probe_phase2.py`, `fixtures/`, `encodings/`). The task must be a representative multi-step please
ask that exercises the spine, at least one gate, and the lessons-audit close. Baseline: run the task
with the skill installed (skill arm), n≥3. Treatment: `--shim-only` with the runbook set in a trial
vault and no skills. D8 bar: `followed_all`/`end_state` within one trial of the skill row. (Decision note: the strict bar was kept per Joe's call. Bar = within one trial of the skill row; the skill row is followed_all 0/3, end_state 2/3 (`results/1.3_please_baseline_sonnet5.md`), so the effective binding constraint is end_state >= 1/3; graded step counts are reported as information only.) Cost
estimated and confirmed before any paid run; per project rule the run is not capped once approved. A
validity gate (marker-in-transcript, shadowing scan) proves treatment text was actually loaded before
any zero-rate is reported.

**D7. Promote via `engram learn runbook`, never by hand.** Content comes from the validated fixture;
each note is created through the CLI (situation, body, done_when, red_flags as real fields) so the
sidecar embeds correctly. Sub-runbooks are created first so the top runbook's wikilinks resolve to real
basenames.

**D8. Retire in the same change, behind a three-part gate.** `SKILL.md` is deleted only when (a) the D8
bar is met, (b) retrieval is verified against the production vault (task and situation phrases surface
the top runbook; a casual-"please" phrase does not), and (c) `shim.md` is confirmed imported in the
real `~/.claude/CLAUDE.md` (per vault note 1031a — a D8 pass inside the eval's shim-only config does
not prove real sessions discover the runbook). If any part fails, retirement does not proceed and the
change reports the finding.

**D9. Delta specs change carrier language only.** `please-doc-enumeration-gate` and the Step 7
requirement in `write-memory-worker` name "the please skill's Step N"; they are re-pointed at the
please runbook / its sub-runbooks with behavior unchanged. The gate-fidelity claim is tested by the
existing `dev/eval/cumulative/please_step3_probe/` harness where feasible.

**D10. Add a generic phrase-two re-check to the shim, alongside a situation rewrite.** The shim-only run
(3.3) never retrieved the please runbook: `shim.md` already says phrase two must name "your own process,
not the ticket", yet 3/3 agents kept the deliverable in it (e.g. "landing a breaking rename of a
user-facing CLI flag end-to-end"). The existing instruction is prose plus a single dispatch-tier
example, so agents pattern-match on that example and skip the rule. Fix has two necessary halves:
(a) shim: a short re-check placed adjacent to the phrase-shape instruction (not in the floor) telling
the agent to test its draft phrase two for concrete ticket nouns or request paraphrase and rewrite it as
a work-handling decision, plus one non-dispatch example; wording stays generic (no runbook, rename, or
eval terms). Cost ~559 bytes / ~97 words. (b) situation: the please runbook's `situation` is rewritten
process-shaped so a well-formed phrase two lands on it. **Caveat:** the shim change alters second-phrase
composition for ALL runbooks, not only please. It could shift retrieval of route and any future
runbook, in either direction, so it must be validated headless (RED/GREEN on phrase-two shape, per the
headless guidance-eval practice: fresh `claude -p`, CLAUDE.md the only variable) and regression-checked
against route's shim-only retrieval before promotion. Alternative rejected: situation-only fix; it leaves
every future runbook exposed to the same deliverable-laden phrase two.

**D11. REJECTED 2026-09-20 — Add a third `--phrase`: the user's message, verbatim; and give the please
`situation` a user-side trigger.** Rejected on measurement: in the scratch-vault probe (5 situation texts x
19 probes, no LLM spend) adding the literal phrase changed the top runbook's rank/score in 0 of 95 cells,
so it buys nothing under semantic retrieval. The shim draft was reverted to the two-phrase wording; the
need it targeted (the user's own trigger words reaching the runbook) moves to the follow-on change
`runbook-lexical-triggers` (a `triggers:` field matched literally against the user's raw text). Original
reasoning kept below for the record. (Joe's direction 2026-09-20.) The failed evals showed agents strip "Please" and "/please" when
they paraphrase into phrase one, and phrase two (after D10) is deliberately process-shaped, so nothing
in the query carries the user's own trigger words. Two coupled halves: (a) shim: a third phrase, the
user's message copied word for word (if long, its first sentence or two, about 200 characters at most),
placed next to the two-phrase rule; the existing "Add more phrases" permission already covers extra
phrases, so this makes one of them mandatory. (b) situation: the top please runbook's `situation` keeps
its process-shaped content AND names the user-side trigger ("when the user says /please or asks, with
"please", to take a multi-step piece of work end-to-end"), so the literal phrase behaves like a direct
`/please` command. The agent still decides whether to honor the surfaced runbook; the runbook body's
"Required argument" section says a single edit or tool call does not use the workflow (the explicit
`/please` form is an opt-in and always applies). **Caveats:** the shim is shared, so the literal phrase
changes the query for EVERY request and every runbook (blast radius as D10): it can attract runbooks
whose situations mention words the user typed ("please", a command name), and it must not crowd out
phrase one/two matches. Route's shim-only retrieval must be regression-checked. D10 showed agents copy
the shim's example wording verbatim; a rule to quote the user must be tested to confirm agents
really quote the message and do not paraphrase it into a "verbatim" phrase or copy the placeholder.
**Measured (scratch vault, 5 situation texts x 19 probes, no LLM spend):** adding the literal phrase changed the
top runbook's rank/score in 0 of 95 cells versus the same query without it, and the literal alone almost never
surfaces the runbook (only short canonical wording such as "please take this end-to-end" does). The retrieval
lever is still phrase two; D11 is cheap insurance for canonical wording, not a proven fix, so headless
validation (tasks 3b.4-3b.6) gates adoption. The wording is generic: no runbook, please, rename, or eval terms. Alternative rejected: keying only
on phrase one (agent's paraphrase) with "keep please in it"; that relies on the agent choosing to
preserve a word it has already shown it drops.

## Risks / Trade-offs

- **[Risk] D8 not met — a 3.4k-word procedure may lose fidelity when chunked.** → Retirement gated on
  the eval; report actuals either way; iterate on sub-runbook boundaries or `red_flags` selection
  rather than forcing the retirement.
- **[Risk] Retrieval does not surface the top runbook for real `/please`-style asks, or over-fires on
  casual "please".** → Retrieval check against the production vault in D8(b); reword `situation:` per
  D5 until top-ranked / correctly discriminating.
- **[Risk] `red_flags` cap forces cutting failure modes that matter.** → D3's ranking plus overflow
  into sub-runbooks; fidelity report lists placement of every row so nothing is silently lost.
- **[Risk] Retiring the skill removes `/please` invocability; discovery becomes retrieval-dependent.**
  → D8(c) shim-activation precondition; explicit BREAKING note in the proposal.
- **[Trade-off] New eval fixture and baseline cost real spend.** → Scoped to n=3 per arm for the one
  representative task; cost estimated and confirmed up front.
- **[Risk] Missed doc references after retirement.** → Run the enumeration grep on `please` across
  docs/skills/guidance/specs before deleting (dogfooding the gate's own rule) and review the
  disposition list with a fresh-context reviewer.

## Migration Plan

1. Build the eval task and skill-arm baseline; run the baseline.
2. Convert to fixture runbooks (top + sub-runbooks) with the conversion-fidelity report.
3. Retrieval check in a trial vault; run the shim-only probe; compare to baseline (D8).
4. If D8 met: promote to the production vault via `engram learn runbook`; re-embed; confirm
   `engram embed status` clean; re-run the retrieval check against production.
5. Confirm `shim.md` imported in the real CLAUDE.md.
6. Update live references; delete `agent-instructions/skills/please/`; run `engram update` sync and
   confirm deployed copies are gone.
7. `targ check-full`; close #758.

Rollback: `git revert` the retirement commit restores the skill; the vault runbooks can be removed
with `engram` note deletion (git is the fallback for the vault store).

## Open Questions

- Which real please task is representative enough for the eval (spine + a gate + lessons audit)
  without being expensive? Pick during task 1.
- Do the three sub-runbooks need their own D8 coverage, or is the top-level end-to-end task enough?
  Default: end-to-end only; add targeted probes if the end-to-end run shows a sub-runbook was skipped.

**Parked (2026-09-20).** Retirement (section 5) is blocked on reliable runbook triggering. Semantic
retrieval of the agent's paraphrase missed the please runbook in 1/3 trials (run 3.3d) and misses
whenever phrase two names the deliverable; the literal-phrase rule (D11) measured no lift. Triggering
moves to the follow-on change `runbook-lexical-triggers` (lexical `triggers:` field on runbooks, matched
literally against the user's raw text). When that ships: rerun a 3.3-style shim-only eval with a
`/please` ask, then section 5. See the Status block in `tasks.md`.
