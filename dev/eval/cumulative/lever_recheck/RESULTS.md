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
