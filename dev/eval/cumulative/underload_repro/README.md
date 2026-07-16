# underload_repro

Headless multi-turn repro harness for the under-load endorse-moment recall miss: does an agent, under
real multi-turn LOAD (`claude --resume`, no forced recall), endorse a vault-refuted lever at an
un-forced endorse moment without recalling the closure that would have stopped it? Target model opus,
3 fixtures (`beacon_relay` = batch-writes, `driftwood_index` = cheap stemmer tokenizer, `loom_scheduler`
= round-robin scheduler), 9 valid (`marker_seen`) trials.

**Honest baseline** (`results/red_baseline_v3.jsonl`, produced by `rescore_v2.py` under the tightened
unforced scorer):

| Channel (recall) | AMNESIA (blind-endorse) | RECONCILED | rate |
|---|---|---|---|
| did NOT fire | 5 | 0 | 5/5 blind-endorse |
| fired | 0 | 4 | 4/4 reconciled |

Recall is the clean difference-maker: fires → reconciles 4/4; doesn't fire → blind-endorses 5/5. This
is the RED baseline for testing any fix to recall firing under load.

The prior scorer (v2) under-counted the blind-endorse rate at 3/5 via two distinct failure modes, both
fixed by the tightened `unforced` mode: a literal `NOT proposed → RECONCILED` verdict loophole (beacon
t2), and a judge mis-rating that scored thematic caution ("measure it first") as acknowledgment
(driftwood t2).

## Fix result — engagement-led wording cue (2026-07-15, DEPLOYED)

A one-bullet `guidance/recall.md` cue targeting the under-load **engagement/conversational-context**
gap (not verb-coverage) was tested via `--guidance` as a paired fresh RED (current wording) vs GREEN
(treatment = `treatment_recall.md`) batch, fixture-balanced n=6/fixture, 18 valid/arm, unforced scorer:

| Arm | n | recall firing | blind-endorse AMNESIA | cost |
|---|---|---|---|---|
| RED (current wording) | 18 | 9/18 = **50%** | 8/18 = **44%** | $8.65 |
| GREEN (treatment cue) | 18 | 18/18 = **100%** | 0/18 = **0%** | $17.46 |

Δ firing **+50.0 pp**, 95% CI **[+26.9, +73.1] pp**, z=3.46 → CI excludes 0 (pre-registered
two-proportion bar PASSES). The firing→verdict coupling holds near-deterministically (as in the
baseline): in the fresh RED, 8/9 not-fired trials blind-endorsed (the 1 exception an in-context save)
and all 9 fired trials reconciled — GREEN's 100% firing is precisely why its AMNESIA is 0%. Adversarially
verified: reconciliations genuine, firing real, instrument stable (fresh RED 50% ≈ locked baseline 44%). **Overturns/narrows** the prior that prose can't move
under-load firing (notes 145/198/277): verb-coverage was already handled, but the engagement cue moves
it. **Deployed** to `guidance/recall.md` via `engram update --with-guidance`.

Caveats (→ GH #695): (1) 5-note fixture vaults make "fires" ≈ "surfaces the needle" — retrieval
precision at real vault scale is untested; (2) the lift is mostly turn-1 firing on the "review a draft"
framing (endorse-moment turn-4 firing barely moved, 5/18→6/18) — mechanism is early-fire +
context-persistence; (3) no placebo control, so "engagement framing specifically" is not isolated from
"any leading cue."

**Files:** `run_underload_repro.py` (harness; `--guidance <path>` selects the arm),
`test_run_underload_repro.py` (harness unit tests), `treatment_recall.md` (the GREEN treatment guidance
= current `recall.md` + the one cue), `rescore_v2.py` (offline re-score of stored v2 recommendations),
`results/red_baseline_v3.jsonl` (locked honest baseline), `results/{red_current,green_treatment}_v3.jsonl`
(the paired fix run).
