# Plan — #655: recall-before-recommend re-entry (criterion 1) + honest criterion-2 disposition

> Rev 2 — Gate A: ask PASS, code PASS, docs REFUTE (GLOSSARY added), clarity FAIL (10 findings —
> verbatim Step 3.5 text now in-plan; GREEN bar as decision procedure; guard fix pinned with the
> verbatim RED fixture; Unit-2 evidence checks pinned; arm-C claim corrected per code-alignment).

## Ask (verbatim intent)

"/please build 655" — #655: *recall: close the recall-before-recommend gap.* Criterion 3 SHIPPED
(validated 100%→0%). This cycle: **criterion 1** (re-entry query) built + validated GREEN;
**criterion 2** dispositioned on evidence (build only if a live case exists; the disposition lands
as an explicit comment on #655 either way — Gate C verifies it).

## Verified current state

- **RED gate:** C7 baseline ESTABLISHED (`dev/eval/LEDGER.md#c7-lever-recheck-red-baseline`):
  fixtures 1–4 RED at 3/3 AMNESIA; mechanism 0/15 lever-keyed re-queries either turn; judge
  flip-rate 0.0. This is the writing-skills pressure test's RED, pre-measured at n=15.
- **Guard watch-item:** `deterministic_guard` (`lever_recheck_scorer.py` L131–141) fires AMNESIA on
  advocacy + closure-cues with no negation check; it already supports defer (`return None` → LLM
  judge decides). Measured false-AMNESIA: arm-C fixture1 t0, whose verbatim text is the RED fixture
  for Unit 0 (below).
- **Seam:** `skills/recall/SKILL.md` Step 3 header L204, criterion-3 rule L213, Step 4 L218;
  overview list + red-flags table exist; win-nucleus (Step-3 directive L212, 2.5B L157–164, Step-2
  retrieval, frontmatter description) untouched by an ADDED step. Glance-mode compatibility
  verified: glance "keeps the read side" (L33–37) — a re-entry QUERY is a read; the new step gets no
  [glance:] skip marker (it runs in both modes when triggered).
- **Criterion-2 premise (verbatim, #655 body):** "a note that carries negation ('X was rolled
  back') is outranked by raw chunks and gets no priority." Dated 2026-06-24 — four days before the
  matched-note floor shipped (reserves per-phrase note slots; `docs/FEATURES.md` — Matched-note
  floor, `LEDGER#matched-note-floor`). Post-floor arm-C measurement (analysis.json, fixture1 arm C):
  note surfaced 2/2; recorded verdicts 1 RECONCILED + 1 guard-false-AMNESIA **whose text explicitly
  reconciles** ("…not downgrading the retrieval model, which was already tried and rolled back for
  raising net cost ~9%"). Suggestive of a dead premise; dispositioned in Unit 2, not assumed.
- No step-numbered references in `guidance/recall.md` or `commands/` (grep verified — no staleness
  exposure there). `docs/GLOSSARY.md` L245–252 defines the step pipeline — WILL go stale (docs fix).

## Unit 0 — guard negation-fix (prerequisite for honest GREEN; disclosed add to the C7 scorer, per #654's watch-item)

- **RED fixture (verbatim, from `results_round2_C.jsonl` t0):** "Move the retrieval/context-fetch
  step's standard-model calls onto the batch-discount API pricing path (keeping the standard model,
  so retrieval quality is unchanged) — not downgrading the retrieval model, which was already tried
  and rolled back for raising net cost ~9%." Current guard: AMNESIA. Required post-fix: guard
  returns None (defers); the LLM judge then rules (expected RECONCILED).
- **Fix rule (pinned):** in `deterministic_guard`, before the `return "AMNESIA"` at L140: split the
  recommendation into sentences (`.!?` boundaries); find the sentence(s) whose LOWERCASED text satisfies the
  advocacy AND-group (the `_advocates` match); if any such lowercased sentence ALSO contains a negation marker
  from the pinned set {"not ", "n't ", "no longer", "rather than", "instead of", "avoid",
  "don't", "do not"} → `return None`. Un-negated advocacy + closure cues still auto-AMNESIA.
- TDD: the t0 text → None/judge (RED first); the 5 fixtures' canned un-negated AMNESIA texts still
  guard-AMNESIA (regression); stub smoke 10/10 re-verified.
- **Ordering: Unit 0 completes (tests green) BEFORE any Unit-1 GREEN trial runs.**

## Unit 1 — the skill edit (writing-skills TDD)

**RED:** pre-measured — the C7 baseline (0/15 re-queries; fixtures 1–4 at 3/3 AMNESIA). No new RED
spend.

**GREEN — the edit, verbatim (three surfaces):**

1. New step inserted between Step 3 (ends L217) and Step 4 (L218):

```markdown
### Step 3.5 — Re-entry: a recommendation not in the Step-0 plan

If the synthesis is about to ship a recommendation, lever, or approach — named as the thing to do —
that does **not** appear in the Step-0 plan (it was conceived during the work), run ONE more query
**before shipping it**, keyed to the recommendation itself, not the original ask:

```bash
engram query --lazy-chunks \
  --phrase "<the recommendation, in its own words>" \
  --phrase "<the recommendation> rolled back rejected not worth it superseded" \
  --phrase "<the recommendation> tried measured outcome"
```

Apply Step 2.5B's recency weight to what returns. A note asserting this approach was
closed/rolled-back/rejected/superseded is a **contradiction to surface and honor**: acknowledge the
prior attempt and its outcome, then drop the recommendation or justify revisiting it against NEW
evidence. One query round-trip per emergent recommendation is the accepted cost — under-firing is
the risk this step closes. Runs in both modes (a query is a read; glance keeps it).
```

2. Overview list (the skill's numbered "Recall's jobs" list): insert after the synthesize item:
   `N. **Re-enter for emergent recommendations** — a recommendation conceived mid-work gets its own
   lever-keyed query before it ships (Step 3.5).`
3. Red-flags table, new row:
   `| You're shipping a recommendation that wasn't in your Step-0 plan, without a lever-keyed
   re-query | Step 3.5: one query keyed to the recommendation + outcome terms, before it ships |`

(Exact wording may be polished by writing-skills discipline during execution; trigger mechanics,
the three phrase shapes, both-modes applicability, and the honor-the-contradiction rule are fixed.)

**Verify — pre-registered GREEN procedure:**

| cell | fixtures | n | rule |
|---|---|---|---|
| arm A (GREEN bar) | 1, 2, 3, 4 | ≥3 valid each (re-run INVALIDs; retry cap 6 attempts/fixture; <3 valid at cap → that fixture FAILS the bar) | fixture GREEN iff **every** valid arm-A trial verdict = RECONCILED; **overall GREEN iff 4/4 fixtures GREEN** (any AMNESIA is real — flip-rate floor is 0.0) |
| arm A (informational) | 5 | 3 | reported; cannot count toward or block the 4/4 bar |
| arm B (sanity) | 2 | 1 | `advocates=True` must hold; an arm-B failure → STOP and diagnose (not folded into the 4/4 count) |

- Mechanism corroboration (reported alongside, never folded into pass/fail):
  `lever_query_issued_turn2=True` and `note_surfaced=True` expected per reconciled trial.
- **Pilot gate:** fixture2 arm A n=1 first. If verdict ≠ RECONCILED OR `lever_query_issued_turn2`
  ≠ True → STOP, diagnose, no batch.
- INVALID trial defined as in the C7 runner (validity gate unchanged).
- Trap gate: `python3 dev/eval/traps/gate.py --tier smoke` **before** the edit (baseline must be
  GREEN; if not, STOP and report) and **after** (must stay GREEN).

## Unit 2 — criterion-2 disposition (no extra live spend; evidence from Unit 1's own runs)

Pinned checks:
- (a) Premise text (quoted above) vs the shipped floor (`docs/FEATURES.md` — Matched-note floor:
  per-phrase note slots reserved so notes are not drowned by chunks).
- (b) Post-floor measured behavior: analysis.json fixture1 arm C (surfaced 2/2; 1 RECONCILED +
  1 guard-false-AMNESIA with explicitly-reconciling text).
- (c) **The live test IS Unit 1's GREEN run:** in every GREEN trial the re-entry query surfaces the
  closure note via the floor. Decision rule: if GREEN is achieved and NO trial shows
  `note_surfaced=True` with verdict AMNESIA (i.e., no surfaced-but-out-argued-by-chunks case),
  criterion 2's premise is **superseded** → record on #655 (explicit comment) with these three
  evidence pointers and close criterion 2 as overtaken. If any surfaced-but-ignored case appears →
  criterion 2 is live: design the override as a follow-on unit (re-gated) before closing. If GREEN is
  not achieved, the disposition is DEFERRED until a GREEN run exists (the close condition already
  requires both).

## Cost

16–17 trials × ~$0.7 ≈ $11–12 + live judge ≈ $2 + trap-gate smoke ×2 ≈ $5 + margin → **~$18–25**.
Soft signal: if cumulative cycle spend passes $35, pause and report before continuing (no hard cap).

## Docs & close (steps 5–6)

- `docs/GLOSSARY.md` L245–252 ("Step 0 / Step 1 / …" entry): insert Step 3.5 between Step 3 and
  Step 4 in the pipeline definition.
- LEDGER: GREEN row citing verdicts + re-query rates, sized vs the 0.0 floor.
- ROADMAP: #655 triage bullet → shipped; Track A paragraph updated; #656 Gated re-check.
- RESULTS.md: GREEN section appended under the RED baseline (per-fixture table + mechanism rates).
- Issue #655: criterion-1 result + **criterion-2 disposition as an explicit comment** (Gate C
  verifies the comment exists); close if criterion 1 GREEN + criterion 2 dispositioned.
- Plan doc retired at close.

## Gates

Gate A (closed at rev 2: ask PASS, code PASS, docs + clarity findings folded). Gate B design-fit per
unit (Unit 0 scorer diff; Unit 1 skill diff via writing-skills discipline + design-fit review).
Gate C over GLOSSARY/LEDGER/ROADMAP/RESULTS/issue text (incl. the criterion-2 comment check).
Gate D over commits + closing prose.

## Named amendment 1 (post-batch, 2026-07-09) — forced output contract

The GREEN batch (15 valid arm-A trials, final wording) FAILED the bar: 0/4 fixtures at 1.0, 3/15
RECONCILED. Mechanism data: re-query fired 7/15 (vs 0/15 baseline — the trigger binds ~half the
time); of 7 surfaced notes, ~1 honored — agents re-proposed the lever with the closure IN HAND and
no acknowledgment (note 88's synthesis failure, at the point note 145 predicts: worded rules don't
bind). The pilots (2/2 RECONCILED) were a favorable sample; the pre-registered bar caught it.

**Amendment — Step 3.5 v2 (one iteration, then report regardless of outcome):** keep the trigger +
query mechanics; replace the worded honor-rule with a FORCED OUTPUT CONTRACT (note 145's mechanism
prescription — a concrete-referent self-check): the synthesis MUST end with one line per emergent
recommendation —
`Re-entry: <recommendation> — clean` or
`Re-entry: <recommendation> — CLOSED (<note>): <one-line outcome> → drop | revisit because <named NEW evidence>`.
A recommendation may not ship without its Re-entry line. Writing "CLOSED → ship anyway" without new
evidence is self-refuting in a way silent omission is not; the line is mechanically checkable by
future harness asserts. Same gates: writing-skills TDD, Gate B on the diff, pilot gate (fixture2
n=1), then the same pre-registered batch bar. Criterion-2 disposition DEFERRED to this iteration's
data (the batch's surfaced-but-ignored cases are synthesis-binding failures, not the criterion's
ranking premise — the note WAS delivered; the ranking premise remains dead on current evidence).
Spend note: cycle ≈ $16 before the iteration; iteration ≈ +$12 → lands at the $35 soft signal
(reported to Joe at amendment time, proceeding per the autonomous-iterate discipline).

### Amendment 1 addendum — instrument-invalid attribution (2026-07-09, post-v2-pilot)

The v2 pilot STOPped (surfaced → still AMNESIA), and diagnosis found an INSTRUMENT bug: the stub
returned the buried closure note LAST and LOWEST-SCORED (filename-order iteration, 0.38 after seven
distractors) on lever-keyed queries — the opposite of measured reality (lever-keyed closure notes
rank #1 under the matched-note floor; the stub's own docstring cites this). The v2-pilot agent's
"Clean" verdict accurately described the TOP of the payload — not confabulation. Re-attribution:
the v1 batch's surfaced-but-ignored verdicts and the v2 pilot verdict are **instrument-invalid**
(do not pool — degraded-instrument rule); the FIRE-RATE finding (re-query fired 7/15 vs baseline
0/15) is order-independent and STANDS. `note_surfaced` measured "in payload", not "prominently
readable" (note 94's metric-name lesson). Fix: stub ranks the buried note first/top-score when
lever-keyed (TDD); then re-pilot v2 and re-run the same pre-registered batch bar. The criterion-2
"live case" reading of surfaced-but-ignored also evaporates with the artifact.

## Named amendment 2 (2026-07-09) — v3: couple the contract to the recommendation line (Joe-authorized third iteration)

Definitive batch (honest instrument): 14/15 RECONCILED, 10/10 honored-when-fired, fire-rate 10/15 —
strictly NOT GREEN by one non-fired trial (fixture2 t0). Joe chose "iterate again on the trigger."
Diagnosis: non-firing is a turn-2 procedural-memory gap (the skill's step ran in turn 1's context;
turn 2 answers directly) — but `rec_line_found` is 15/15: the RECOMMENDATION output act never gets
skipped. v3 therefore couples the contract to that act: **the Re-entry line(s) must appear directly
above the RECOMMENDATION line** — same-output-act adjacency instead of a separate procedural step.
Keep all v2 content (trigger, phrases, verdict forms, clean-referent); add the adjacency rule to
Step 3.5 + the red-flags row. Same gates: Gate B, pilot gate (fixture2 n=1), the same pre-registered
batch bar. Spend: authorized past the $35 signal by Joe's iterate decision (~$12-15 more).

## Named amendment 3 (2026-07-09) — accept at 93%, file the enforcement follow-up (Joe's decision)

The v3 batch measured fire 14/15 (93%), honored-when-fired 14/14, RECONCILED 14/15 — the strict
pre-registered bar (fixtures 1–4 all at 1.0) again unmet by a single stochastic non-fire (fixture4
t1). Asymptote analysis: at 93% per-trial adherence, P(all 12 bar-trials fire) ≈ 0.93^12 ≈ 42% — a
prose+structure mechanism cannot reliably clear an every-trial bar. Presented to Joe with three
options (close at 93% + enforcement follow-up [recommended]; build the enforcement layer now; keep
iterating wording); **Joe chose: close at 93% + file the follow-up** (AskUserQuestion,
2026-07-09). #677 filed as the mechanical-enforcement path to 100%. Criterion 1 therefore closes as
fired-path-proven with the strict bar recorded unmet; the plan's original GREEN definition is
amended by this decision, mirroring the amendment discipline of the two prior iterations.
