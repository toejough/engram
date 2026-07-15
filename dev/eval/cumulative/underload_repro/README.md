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
is the RED baseline for testing any fix to recall firing under load — no fix was applied this cycle.

The prior scorer (v2) under-counted the blind-endorse rate at 3/5 via two distinct failure modes, both
fixed by the tightened `unforced` mode: a literal `NOT proposed → RECONCILED` verdict loophole (beacon
t2), and a judge mis-rating that scored thematic caution ("measure it first") as acknowledgment
(driftwood t2).

**Files:** `run_underload_repro.py` (harness), `rescore_v2.py` (offline re-score of the stored v2
recommendations under the tightened unforced scorer), `results/red_baseline_v3.jsonl` (locked honest
baseline).
