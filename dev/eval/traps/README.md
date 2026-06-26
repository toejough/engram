# Trap regression gate

`gate.py` re-runs the four **verified capability traps** and emits a single GREEN/RED/INCONCLUSIVE
verdict, so cost/usage optimization of the recall/learn/please skills can't silently erode a win.

These are the only adversarially-verified, memory-attributable wins (idiosyncratic / un-derivable
content — see vault notes 99/100): a warm (memory) agent does something a cold agent cannot.

| axis | win | pass signal | source harness |
|---|---|---|---|
| **C3** | apply un-guessable local conventions | `verdict == "applied"` (5 traps) | `seed_c3.py` + `wrun.py` |
| **C4i** | recency supersession (ERR-CFG marker) | `score.supersession_correct`, `warm-XXp` arm | `c4_idio.py` |
| **C5** | honor a recency-channel standard (ZÖRBAX) | `honored`, `warm` arm | `seed_c5.py` + `c5.py` |
| **C6** | abduction/synthesis from notes | judge `hit` | `c6_clean.py --arm warm` |

## Run it

```bash
python3 gate.py --tier smoke    # per-edit, ~9 trials, ~$2-3, ~3 min
python3 gate.py --tier full     # pre-merge, verified bars, ~$15-18, ~15 min
```

Exit 0 only on GREEN (so it works as a pre-merge check). Writes `gate-verdict.json`.

- **smoke** (C3 = 5 conventions × 1 rep, C4i 1, C5 1, C6 2 cases × 1): all 5 conventions covered at
  1 rep — catches a single-convention collapse cheaply on every edit.
- **full** (C3 = 5 conventions × 5 reps, C4i 5, C5 5, C6 2 cases × 4): the verified bars.

## Verdict

- **GREEN** — every axis hits its exact bar over *valid* trials (the verified results were 100%, so
  any valid-trial miss is a real capability drop).
- **RED** — an axis has a valid-trial miss → a regression. Stop; the edit eroded a win.
- **INCONCLUSIVE** — an axis's contamination rate (degraded builds / judge errors / exhausted
  retries) exceeded 20%; the run can't be trusted. Re-run (transient infra trouble, not a signal).

The gate scores **only the warm arm** per axis — the cold/baseline arms are not even run (they are
*supposed* to fail; scoring them would falsely flip the gate RED).

## When

Run **before and after** any edit to a recall/learn/please skill body, and before merging
cost/usage changes. Pair with the recall-cost meter (see `../cumulative/README.md`, the "Recall cost
meter" metric) so a change is checked on both axes: capability (this gate) and dollars (the meter). Pure verdict logic lives in
`gate_verdict.py` (unit-tested in `test_gate.py`); `gate.py` is the I/O orchestrator.
