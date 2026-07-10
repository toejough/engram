# C7 lever-recheck (anti-amnesia) — RED baseline results (#654)

Plan: `docs/superpowers/plans/2026-07-08-654-lever-recheck-harness.md`. Analysis: `analysis.json`
(machine-readable; this doc narrates it). Ledger row: `dev/eval/LEDGER.md#c7-lever-recheck-red-baseline`.

## Verdict

**C7 RED baseline ESTABLISHED.** Bar (pre-registered): ≥4 of 5 fixtures RED, each on ≥3 valid arm-A
trials, 0 of which score RECONCILED. **4/5 fixtures RED.**

Terms: a **fixture** is a fictional-domain test case (task + scratch data + seeded vault + a
ground-truth *closed lever* — an approach that was tried and killed, recorded only in a buried vault
note). Per-trial verdicts: **AMNESIA** = the recommendation re-proposes the closed lever as fresh,
without acknowledging the prior attempt and outcome; **RECONCILED** = it acknowledges the prior
attempt and drops the lever (or justifies revisiting) — or does not propose the lever at all. Arms:
**A** = the RED cell (closure note buried; data arrives only after recall); **B** = temptation
control (no closure note anywhere; measures whether the task genuinely elicits the lever); **C** =
positive-class cell (consult framing; the note is reachable at recall time). Arm C is absent from
the table below because only arm A feeds the RED bar.

| fixture | domain (real isomorph) | valid n | AMNESIA | RECONCILED | verdict |
|---|---|---|---|---|---|
| fixture1 | Orchestra doc pipeline (note 80, cheap-retrieval-model) | 3 | 3 | 0 | **RED** |
| fixture2 | log-analytics ingest (notes 77/79, size-cap null) | 3 | 3 | 0 | **RED** |
| fixture3 | CI build farm (note 108, async-learn) | 3 | 3 | 0 | **RED** |
| fixture4 | support-KB search (note 153, qanchor park) | 3 | 3 | 0 | **RED** |
| fixture5 | data-sync daemon (note 191, payload-prune build) | 3 | 2 | 1 | NOT-RED |

fixture5 is an honest non-reproduction, not a defect (see Known limitations). The bar (≥4/5) is met.

## Mechanism findings (the load-bearing sub-metric)

| metric | value | across |
|---|---|---|
| re-query rate, turn 1 | 0/15 | all official arm-A trials (5 fixtures × 3) |
| re-query rate, turn 2 | 0/15 | all official arm-A trials (5 fixtures × 3) |
| note surfaced (arm A) | 0/15 | all official arm-A trials |

The current recall skill **never** issues a lever-keyed query after the lever is conceived (turn 2,
once the diagnostic data arrives) — in either turn, across every official arm-A trial. The buried
disproving note never surfaces. This confirms the C7 gap is real, not an artifact of the harness.

**Scoping consequence for #655:** the shipped **criterion-3** reconcile-proposals rule (already live
in `skills/recall/SKILL.md`) does **not** reliably fire a vault re-query in this two-turn structure —
0/15. #655's remaining, load-bearing scope is **criterion 1**, the re-entry query itself (a second,
lever-keyed `engram query` mid-synthesis before shipping a recommendation). Criteria 2/3 are
defense-in-depth but do not close this gap alone.

Control (arm B, degenerate-scorer check — proves the task's data legitimately tempts the lever with
no closure info in the vault at all): advocacy rates were f1 2/2, f2 2/2, f3 2/2, f4 1/2, f5 1/2 — the
lever is a plausible answer on the merits in every fixture, not a scorer artifact.

Arm C (fixture1, single-call consult-memory form — the positive-class/regression cell, where the
lever *is* conceivable at recall time): note surfaced 2/2. t1 scored a clean RECONCILED
("do NOT switch retrieval to the cheap small model") — the judge's positive-class demonstration. t0
scored AMNESIA, but is a measured false-positive from the deterministic guard (see Known limitations).

## Design lineage — 3 pilot-falsified amendments

**Amendment 1 (original, inline-context design).** `context.md` embedded directly in the prompt. The
lever was conceivable **at recall time**: the Step-0 recall phrased a lever-keyed query, the buried
note surfaced, and the model reconciled — every time. This reproduced note 85's own finding (the miss
requires the lever to be conceived *after* the single recall) rather than the miss itself: a small
vault can't bury a note from a recall that can already phrase it. Not RED-capable as designed.

**Amendment 2 (cwd scratch-file, post-pilot-2).** Moved `context.md` out of the prompt into a cwd file
(`scratch-notes.md`) the agent is told to read *after* recall fires, plus a `RECOMMENDATION:` format
directive (fixing two side defects: without a delimited line, the scorer fell back to scanning the
agent's whole reply — including its recall narration, which quotes note terms and closure cues — so
the deterministic guard could fire on narration rather than the recommendation; and arm B's final
reply had contained no recommendation at all). Pilot 3 measured this still doesn't
work: the agent reads the scratch file *before or while phrasing* its own recall (instructional
ordering isn't a real constraint — note 145: mechanisms, not prose), so the lever stays conceivable at
recall time in both arms. Not RED-capable.

**Amendment 3 (two-turn `--resume` structure, shipped).** Enforces the phase split **mechanically**
with two `claude` calls: turn 1 is RECALL_PREFIX + the diagnostic task only (no scratch file
reachable, data explicitly "still being gathered"); turn 2 is `claude --resume <turn-1 sid>`,
delivering the scratch data plus the `RECOMMENDATION:` directive — after turn 1's recall has already
run and returned. One stub log spans both turns, so any re-query anywhere in the session is visible.
This is the structure that reproduces the RED signature (see Mechanism findings above). Addendum:
under two-turn, arm C collapses into a duplicate of arm A (no data in turn 1 means the consult framing
can't phrase the lever either) — arm C instead runs the single-call form pilot 3 validated (scratch
file present from the start + consult framing + directive), which is the surfaced→RECONCILED
positive-class cell.

## Fixture tuning history

Tuning rule: ≤2 rounds/fixture, each round = one n=1 re-pilot, from the approved-moves list: reword
`task.txt` (keeping the neutral diagnose-and-recommend shape); reshape `context.md`'s data (dilute or
reposition the lines that suggest the lever, neutralize competing alternatives as neutral history);
add distractor notes (up to the AC's cap of 10); tighten `STUB_ENGRAM_LEVER_TERMS` (the AND-group
keyword matcher shared by the stub's query-keying and the advocacy check).

| fixture | rounds used | what changed | post-tune outcome |
|---|---|---|---|
| fixture1 | 1 | round-1 batch measured vacuous-RECONCILED (agent found a better visible alternative); closure rewritten to an idiosyncratic, cold-model-underivable outcome (re-generation-loop cost, +9%) | RED (3/3) |
| fixture2 | 1 (separate, pre-batch) | pilot-3 found the lever pointed the wrong *direction* — the data tempts toward fewer/larger page reads, the lever said shrink; flipped to increase-archive-page-size (isomorph preserved: size/request knobs don't move a flat-committed-rate bill), vault note swapped + re-embedded, `lever_terms` matched to the naturally-produced phrasing | RED (3/3), fixture untouched since |
| fixture3 | 2 | round 1: vacuous-RECONCILED (agents derived the closure a priori) → idiosyncratic closure. Round 2 (final): agents still escaped via a selective-retry/quarantine alternative → all visible outs closed as neutral facts (all-or-nothing vendored runner, per-merge full-pass compliance policy), leaving the preemptible-lane lever as the only option with visible supporting math | RED (3/3) |
| fixture4 | 1 | round-1 batch measured a `lever_terms` **false-negative** (matcher missed the natural phrasing of the recommendation, not a vacuous-RECONCILED problem) → terms widened to the batch's natural phrasings | RED (3/3) |
| fixture5 | 1 (2nd round available, deliberately unspent) | round-1 batch measured vacuous-RECONCILED → idiosyncratic closure (coordinator-hotspot IPC cost, 3x); the post-tune official batch still measured 1/3 RECONCILED — the agent invented an *uncovered* alternative (checkpoint-offset truncation), not memory-driven. Round 2 was available but deliberately left unspent: the ≥4/5 bar was already met by the other four fixtures, and a second round risked overfitting the fixture to this harness rather than measuring a real gap | **NOT-RED** (2/3 AMNESIA, 1/3 RECONCILED) — honest, not a defect |

## Validity + gates

- **Treatment-delivery gate (per trial):** turn-1 stub log non-empty (recall actually ran against the
  stub) AND `total_cost_usd ≥ $0.02` per turn (degraded/rate-limited-call floor, `dev/eval/traps/wrun.py`
  precedent) AND non-empty agent text. All **27 official trials valid** (0 excluded as degraded).
- **Judge-variance measurement:** full 3-judge vote re-run 3× on 5 transcripts spanning fixtures and
  both verdict classes (2 RECONCILED, 3 AMNESIA) — **overall flip-rate 0.0** (0 flips / 5 rows × 3
  re-votes each). Any future GREEN claim (#655) must exceed this bar to be distinguishable from noise.
- **Paraphrase hard gate:** 1 randomly-sampled AMNESIA transcript per RED fixture (4 rows, seed 0) —
  transcript grep confirmed the buried note's content never appeared in recall output, stub log
  confirmed no lever-keyed query, and a meaning-preserving vocabulary-shifted paraphrase of the
  recommendation was re-judged. **4/4 stable** (verdict unchanged under paraphrase). One row
  (fixture1) carries `guard_fired_original: true` — the guard fired on the original AND the paraphrase
  agreed AMNESIA, so this gate passed clean; it is a separate instance from the arm-C false-positive
  below (that one never entered the paraphrase sample — it's an arm-C RECONCILED-class row, and the
  sample here is AMNESIA-only per fixture).
- **Judge determinism cross-check:** stub-judge pass over all live transcripts; no unresolved
  disagreement with the live judge before verdicts shipped.

## Known limitations

- **Guard negation-blindness (arm C, fixture1, t0).** The deterministic reconciliation-by-vocabulary
  guard keys on lever-terms + closure-cue vocabulary appearing in the recommendation text. In this
  trial the agent explicitly **rejected** the lever (negated advocacy), but the guard fired on the
  lever-terms present in that rejection and the trial scored AMNESIA — a measured false-AMNESIA, not a
  real miss. This is a known, accepted limitation: it biases the measurement toward RED, which is the
  safe direction for *establishing* a RED baseline (it can only make a fixture look worse than it is,
  never manufacture a false GREEN). **Watch-item for #655's GREEN judging:** a GREEN claim run through
  this scorer must account for this bias — a naive re-run would need the guard's negation-blindness
  fixed first, or GREEN verdicts under a guard-fire need a corroborating manual read.
- **Single-line `RECOMMENDATION:` extraction.** Scoring reads the last `RECOMMENDATION: <...>` line;
  this pins extraction cleanly today but is a rigid contract — any future prompt/fixture change that
  drops or reformats that line silently falls back to whole-text extraction (the amendment-2 failure
  mode). Watch-item, not a defect in this baseline.
- **fixture5 honest NOT-RED.** See the tuning table above — the residual RECONCILED is a genuinely
  uncovered alternative (checkpoint-offset truncation) the fixture doesn't neutralize, not a
  memory-driven reconciliation. Recorded as a non-reproducing fixture per the amended tuning rule,
  not forced to RED.
- **Stub payload not byte-faithful.** The stub `engram` returns a functionally sufficient payload but
  `candidate_l2s` is always empty and there is no `budget` object — this is inert for what C7 measures
  (Step 3 reads `items[]` content, which the stub populates fully) and was already validated against
  live `/recall` in the merged fixture1 control runs. Accepted, not re-litigated here.

## Provenance / curation

`results_official.jsonl` (36 lines: 27 trial rows + 5 revote rows + 4 paraphrase rows) is curated
per-fixture from the round-appropriate raw file, not a single flat batch:

| fixture | source | why |
|---|---|---|
| fixture2 | round-1 file | its fixture was untouched after the pre-batch direction-flip tune — round-1 data is the official data |
| fixture1, fixture4, fixture5 | round-2 (post-tune) files | round-1 tuning applied; round-2 is the official post-tune batch |
| fixture3 | round-2-final (post-second-tune) file | needed the additional round-2 tune to close the selective-retry escape |

Two mixed-provenance rows were excluded as straddles — trials that ran while a fixture tune landed
mid-batch, so the trial read a different fixture state than its file's round label implies
(attributable by recommendation content, not just filename): a fixture5 round-1 row that read the
tuned context, and a fixture3 round-2_A pilot row (pre-final-tune). All raw files
(`results_pilot*.jsonl`, `results_round2_*.jsonl`, `results_batch_*.jsonl`, `results_fable.jsonl`) are
retained on disk for audit.

**Secondary (no bar attached):** fable (`claude-fable-5`) fixture2 arm A, n=2 — 2/2 AMNESIA, the same
blindness signature as opus (~$1.37/trial vs opus ~$0.66/trial). The miss is not model-specific.

## Spend

| item | cost |
|---|---|
| official trials (27, opus, per analysis.json) | $17.29 |
| pilots (4 rounds, incl. 2 falsified designs — amendments 1 and 2) | ~$5.6 |
| analysis judges (revote + paraphrase + per-trial scoring) | ~$1.2 |
| **total** | **~$24** |

No mid-run spend cap was imposed (per standing practice — estimate up front, let the run finish).

## How to re-run

Runner (checkpointed; `--resume` skips completed (fixture, arm, trial) rows):

```
python3 dev/eval/cumulative/run_recheck.py \
  --fixtures fixture1,fixture2,fixture3,fixture4,fixture5 \
  --arm A --arm B --n 3 --model opus --judge live \
  --out dev/eval/cumulative/lever_recheck/results_batch_A.jsonl --resume
```

Arm C is opt-in, and only meaningful on fixture1 (the only fixture with a distinct consult-memory
task):

```
python3 dev/eval/cumulative/run_recheck.py \
  --fixtures fixture1 --arm C --n 2 --model opus --judge live \
  --out dev/eval/cumulative/lever_recheck/results_round2_C.jsonl --resume
```

Fable secondary (no bar attached):

```
python3 dev/eval/cumulative/run_recheck.py \
  --fixtures fixture2 --arm A --n 2 --model fable --judge live \
  --out dev/eval/cumulative/lever_recheck/results_fable.jsonl --resume
```

Analysis (bar verdict + mechanism rates + cost tally; `--revote`/`--paraphrase` are live LLM gates,
each opt-in):

```
python3 dev/eval/cumulative/analyze_recheck.py \
  --in dev/eval/cumulative/lever_recheck/results_official.jsonl \
  --revote --paraphrase --seed 0 \
  --out dev/eval/cumulative/lever_recheck/analysis.json
```

## GREEN validation (#655)

Plan: `docs/superpowers/plans/2026-07-09-655-reentry-query.md` (incl. two named amendments + addendum).
Ledger row: `dev/eval/LEDGER.md#c7-reentry-query-green`. Scope: criterion 1 (the re-entry query itself)
— a second, lever-keyed `engram query` mid-synthesis, issued before an emergent recommendation ships.
Criteria 2 and 3 are dispositioned/shipped, not re-built (see below).

### Iteration arc

Same harness, same 5 fixtures, arm A only (the RED cell — closure note buried; data arrives only after
recall), n=3 valid trials/fixture (15/batch), opus, live judge. Bar (pre-registered): fixtures 1–4 only
(fixture 5 informational, never counted toward or against the bar) — **GREEN iff all 4 fixtures are 3/3
RECONCILED.**

| batch | edit | fire-rate (turn-2 re-query) | honored-when-fired | RECONCILED | bar verdict |
|---|---|---|---|---|---|
| baseline (pre-#655) | none — shipped criterion-3 reconcile-rule only | 0/15 | n/a (0 fired) | 1/15 (0 bar-relevant; the 1 is a vacuous-RECONCILED on informational fixture5) | NOT MET — 0/4 fixtures GREEN (RED, see `c7-lever-recheck-red-baseline`) |
| v1 (worded honor-rule) | Step 3.5 added: re-entry query + a worded "acknowledge and drop/justify" rule | 7/15 | ~1/7 | 3/15 | NOT MET — 0/4 fixtures GREEN |
| v2 (forced output contract) | worded rule replaced by a forced per-recommendation `Re-entry:` line (note 145's concrete-referent mechanism) | 10/15 | 10/10 | 14/15 | NOT MET — fixture2 t0 the sole non-fired trial, scored AMNESIA |
| **v3 (recommendation-line adjacency)** | Re-entry line(s) required directly above the `RECOMMENDATION:` line — coupled to the one output act measured 15/15 across all batches | **14/15 (93%)** | **14/14** | 14/15 | NOT MET — fixture4 t1 the sole non-fired trial, scored AMNESIA (fixtures 1/2/3, the bar's remaining population, each 3/3; fixture5, informational-only, also 3/3) |

Raw data: `results_655_pilot{,2}.jsonl` (v1 pilots), `results_655_green_A12.jsonl` + `results_655_green_A345.jsonl`
(v1 batch); `results_655v2_pilot{,2}.jsonl` (v2 pilots, first STOPped on the instrument bug below),
`results_655v2_green_A12.jsonl` + `results_655v2_green_A345.jsonl` (v2 batch); `results_655v3_pilot.jsonl`
(v3 pilot), `results_655v3_green_A12.jsonl` + `results_655v3_green_A345.jsonl` (v3 batch, the definitive
result).

### Instrument-invalid episode + stub fix

The v2 pilot re-run STOPped: the note surfaced but the trial still scored AMNESIA. Diagnosis
(`ca782a1d`) found the bug was in the harness, not the skill: `stub_engram.py` emitted the buried
closure note **last and lowest-scored** (filename-order iteration, ~0.38 after seven distractors) on
lever-keyed queries — the opposite of the shipped matched-note floor's real, measured behavior
(lever-keyed closure notes rank #1; the stub's own docstring cites this). The v2-pilot agent's "clean"
verdict honestly described the *top* of the payload it was actually given — not a synthesis failure.

Re-attribution: the v1 batch's surfaced-but-ignored verdicts and the v2 pilot's verdict are
**instrument-invalid** (excluded from the pooled result, per the degraded-instrument rule — do not
pool). The fire-rate finding (re-query fired 7/15 in v1 vs 0/15 baseline) is order-independent and
**stands** regardless of the ranking bug — it measures whether the query was issued, not what it
returned. `note_surfaced` was shown to mean "present in payload," not "prominently readable"; the two
are conflated names for different facts.

Fix (`9353e8da`, TDD — 2 RED tests reproduced the bug first): the stub now ranks the buried note
first/top-score (0.92) specifically on lever-keyed queries; the distractor ladder, non-keyed exclusion,
and `returned_buried` logging are unchanged. 149 tests green. The v2 batch above is the **post-fix,
honest** result — every "note surfaced" row in v2/v3 reflects the note actually being prominent, not
merely present.

### Guard negation-fix

Prerequisite (`48153a3f`, Unit 0, ahead of any GREEN trial): the deterministic scorer guard
(`lever_recheck_scorer.py`) fired AMNESIA on any recommendation containing advocacy + closure-cue
vocabulary, with no negation check — so a recommendation that **rejects** the lever by name (e.g. "…not
downgrading the retrieval model, which was already tried and rolled back…") could still auto-score
AMNESIA. RED fixture: the verbatim arm-C fixture1 t0 text from `results_round2_C.jsonl` (the RED
baseline's known false-AMNESIA). Fix: split the recommendation into sentences; if an advocacy-satisfying
sentence also carries a negation marker from a pinned set (`not `, `n't `, `no longer`, `rather than`,
`instead of`, `avoid`, `don't`, `do not`), the guard defers (`return None`) to the LLM judge instead of
auto-scoring; un-negated advocacy + closure cues still auto-AMNESIA (regression-tested). 4 new stub
tests + 3 guard tests; suite green throughout (145 after Unit 0, 149 after the stub fix). Known residual,
recorded and accepted: cross-sentence negation still guard-fires — the paraphrase hard gate (RED
baseline) covers this class.

### Zero-stub live smoke

One real end-to-end trial with the stub removed entirely: the real `skills/recall/SKILL.md`, a real
`engram` binary, and a real 212-note vault copy (not the fixture harness); result archived: `smoke_655_e2e.json` + `smoke_655_e2e_ask.txt`, session cost $0.67. Full chain PASS:
the agent conceived the payload-prune lever on its own (unprompted by the fixture apparatus), Step 3.5
fired, `engram query` returned note 191 (the buried closure note) top-ranked, and the agent wrote both
Re-entry contract lines verbatim — `CLOSED(191): <prior outcome> → drop` with the reopen condition for
the payload-prune lever, and a clean-with-referent line for its second, non-emergent recommendation —
then recommended the already-shipped tier-routing lever instead. This is the one trial in the whole
cycle not mediated by `stub_engram.py`; it corroborates the v3 batch result against the real production
path, not just the harness.

### Criterion-2 disposition — SUPERSEDED

Premise (verbatim, #655 body, dated 2026-06-24): "a note that carries negation ('X was rolled back') is
outranked by raw chunks and gets no priority." This predates the matched-note floor (shipped
2026-06-28, `dev/eval/LEDGER.md#matched-note-floor` — per-phrase note slots reserved so notes are not
drowned by chunks) by four days. Per the plan's Unit 2 decision rule: the live test IS the GREEN run
itself — across 24 fired trials with the honest (post-stub-fix) instrument (10 in v2 + 14 in v3), **0**
show `note_surfaced=True` with verdict AMNESIA (a surfaced-but-out-argued-by-chunks case). The earlier
apparent live cases (arm-C fixture1, pre-fix) evaporate with the instrument bug — they were the buried
note ranking last, not chunks outranking a surfaced note. Disposition: **criterion 2's premise is
superseded**, recorded on #655 per the plan's close condition; no override unit is warranted on current
evidence.

### Honest bounds

- **Strict bar: unmet.** The pre-registered GREEN bar (fixtures 1–4 each 3/3 RECONCILED) was not
  achieved by any batch, including the final v3. Report the fired-path result (93% fire, 14/14 honor)
  as what it is — not a claim that Step 3.5 fires every time.
- **Asymptote, not a bug to chase.** At a stable ~93% measured per-trial fire-rate, the probability of
  12 consecutive fires (roughly one "session" of emergent-recommendation moments) is
  0.93^12 ≈ 42% — well under even-odds. A single-mechanism, prose-plus-structure step (however tightly
  coupled to an output act) has a ceiling; reaching a reliable every-trial guarantee needs an
  out-of-text enforcement layer (harness-level assert, lint, or hook) rather than another wording
  iteration. Filed as **#677** (mechanical enforcement layer for the Step 3.5 Re-entry contract,
  93%→100%) rather than a fourth prose iteration.
- **Judge flip-rate context.** The RED baseline's judge-variance measurement (flip-rate 0.0, 5 rows × 3
  re-votes) is the noise floor this result is compared against. 93% (14/15) is far outside that noise
  band — the one non-fire (fixture4 t1) is a real, stochastic miss, not judge jitter.
- **Guard bias direction unchanged.** The negation-fix only widens which cases the guard defers to the
  judge; it never auto-reconciles. Any residual guard-fire bias still pushes toward AMNESIA (conservative),
  the same safe direction documented in the RED baseline's Known limitations.
- **Scope.** This validates criterion 1 only (the retrieval-side re-entry query). Criterion 3 was
  already shipped and validated separately (100%→0%, per the #655 issue body); criterion 2 is
  dispositioned above, not built.

### Spend (#655 cycle; raw sums from the JSONL summary rows)

| item | cost |
|---|---|
| v1 GREEN batch (instrument-invalid for surfaced-case verdicts) | $12.22 |
| v2 GREEN batch | $15.18 |
| v3 GREEN batch (the shipped result) | $15.04 |
| pilots (v1 ×2, v2 ×2, v3 ×1) | $4.10 |
| zero-stub live smoke (`smoke_655_e2e.json`) | $0.67 |
| trap gate smoke ×2 (before/after) | ~$6 (estimate from per-axis trial costs) |
| live judge calls (sonnet ×3/lever/trial, unmetered) | ~$3–5 (estimate) |
| **total** | **~$56–58** |

Raw agent-turn costs are exact (summed from the batch summary rows); judge and gate figures are
labeled estimates (those calls are not per-trial metered).

### How to re-run

```
# v3 batch (the shipped, definitive edit)
python3 dev/eval/cumulative/run_recheck.py \
  --fixtures fixture1,fixture2,fixture3,fixture4,fixture5 \
  --arm A --n 3 --model opus --judge live \
  --out dev/eval/cumulative/lever_recheck/results_655v3_green_A12.jsonl --resume

python3 dev/eval/cumulative/analyze_recheck.py \
  --in dev/eval/cumulative/lever_recheck/results_655v3_green_A12.jsonl,dev/eval/cumulative/lever_recheck/results_655v3_green_A345.jsonl \
  --out dev/eval/cumulative/lever_recheck/analysis_655v3.json
```
