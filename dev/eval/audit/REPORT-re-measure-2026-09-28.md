# Learn-rate re-measure (learn-rate-skill-only, tasks 3.2–3.3)

The pre-registered re-measure of W2 after the v1 skill edits shipped on 2026-09-07 (`re-measure-plan.md`, design D-E). The run went from 2026-09-28 23:38 UTC to 2026-09-29 06:35 UTC, after the pre-registered window of 2026-09-21..28.

All figures come from `compute_remeasure.py` over `results-re-measure-2026-09-28/moments.jsonl`. It uses `build_scorecard.py`'s own bootstrap, rate and percentage functions (n=2000, seed=739) and the baseline's definitions. Run on the frozen `results/moments.jsonl`, the same script reproduces the baseline exactly.

**Scope:** moments dated 2026-09-07T19:19 to 2026-09-24T21:33 UTC, from 47 transcripts started on or after 2026-09-07, one seeded window each.

## 1. Result

Rates are per moment. Baseline moments are dated 2026-07-22..09-01; re-measure moments are dated 2026-09-07..09-24.

| Metric | Baseline | Re-measure | Change (95% interval) |
|---|---|---|---|
| **W2: learn fired, of worth-learning moments** | 4/71 = 5.6% [1.4, 11.3]% | 3/66 = 4.5% [0.0, 10.6]% | −1.1 pp [−8.5, +6.3] pp |
| Lost lessons | 67 of 150 moments (44.7%) | 63 of 119 moments (52.9%) | +8.2 pp of moments (no interval) |
| Share of lost lessons judged fixable | 40/67 = 59.7% [47.8, 71.6]% | 50/63 = 79.4% [68.3, 88.9]% | +19.7 pp (samples not comparable) |
| Fixable lessons by role | workflow 14 / fresh 11 / fork 8 / main 7 | fresh 50 | not comparable |
| W1: worth-learning share (for context) | 71/150 = 47.3% | 66/119 = 55.5% | +8.2 pp |

W2 did not rise detectably. That is "can't distinguish from baseline", not "confirmed unchanged": a gain of up to about 6 pp is still inside the interval. All 3 learn fires are in one transcript, `agent-a61fed4470591bb25` (#29, #36, #49). The per-moment bootstrap ignores clustering within a window, as the baseline's did, so the intervals are somewhat too narrow.

## 2. Deviations

1. **Run date:** the run finished after the pre-registered window.
2. **Date filter:** transcripts whose first timestamp's UTC date is 2026-09-07 or later; 312 started earlier and 29 had no timestamp.
3. **Exclusions:** named, tested rules in `run_audit.py`.

   | Step | Transcripts |
   |---|---|
   | Enumerated after the `-private-tmp` and `-dev-eval-` directory rules | 892 |
   | Dropped: started before 2026-09-07 | 312 |
   | Dropped: no timestamp | 29 |
   | Excluded: the orchestrating session (f6dfe139) | 118 |
   | Excluded: test-subject dispatch description | 14 |
   | Excluded: role-play prompt | 0 (the same agents were already caught by the rule above) |
   | Excluded: headless eval arm | 7 |
   | Excluded: automated security review | 124 |
   | **Eligible** | **288** |

4. **One window per transcript, a deviation and not a match:**
   - the baseline audited every window of each sampled transcript;
   - here 47 of the sample's 292 windows were audited;
   - windows were chosen uniformly by one `random.Random(739)` stream in sample order;
   - each chosen window's line range is stored, and the driver stops if it changes.
5. **Two rate-limit halts:** at 27/47 and 32/47. Each halted item had no moments and no done-marker saved, and was redone from scratch on resume. Checks afterwards: 119 unique moment ids and 47 unique done-markers.
6. **Uniform instead of stratified sampling:** runbook-vs-skill 41, phone-llm 4, local-llm-optimization 2 (the baseline covered 7 repos). All windows are subagents'.
7. **Output directory:** `results-re-measure-2026-09-28/`, because the tool refuses anything under `results/`.
8. **How it was run and costed:**
   - the instrument ran unchanged via `run_window_sample.py`, with only its window split limited to the chosen window;
   - costs are the sum of each `claude -p` call's reported `total_cost_usd` (323 calls);
   - `--estimate` wasn't usable because it scales per-transcript yield by the number of transcripts.
9. **Cost:** $107.56 against the plan's ~$35, under the $150 cap Joe set. That total includes $17.65 of smoke runs whose windows were thrown away when the sample was redrawn.

Command: `python3 dev/eval/audit/run_window_sample.py --output-dir dev/eval/audit/results-re-measure-2026-09-28 --since 2026-09-07 --prior-spend-usd 17.65 --cost-cap-usd 150` (sample 47, seed 739; instrument version 2, haiku detects, sonnet judges; resumed twice with the same command).

## 3. Validity caveats

- **a. The instrument can't see v1's capture path.** It counts learn only inside the audited window. All sampled windows are subagents', while v1 captures through the parent's closing `/learn`, which writes the subagents' LESSONS lines. A lesson v1 captured correctly still scores as "learn didn't fire". The baseline had the same blind spot, but its sample included main and workflow windows. As a rough, unmeasured sign the contract is being followed, 223 subagent transcripts contain `LESSONS:`. A parent-side measure of the curation step would answer the question, but it wasn't pre-registered.
- **b. engram query rc 1:** 2 failures in `agent-a6a6a1c5e30232429`, leaving 2 empty "memory existed" fields; 4 more are empty by design. The cause is a search phrase starting with `--`, which engram reads as a flag (`flag needs an argument: --phrase`), printing the error to stdout. W2 is unaffected. The instrument bug was not fixed.
- **c. Secondary metrics:**
  - 9 of the 63 lost lessons have no fixable/not-fixable category; each is marked `judged_not_applicable` with a rationale.
  - The rise in the fixable share rests on an all-subagent sample, so it doesn't show a behaviour change.
  - Moment mix: baseline success 72 / failure 27 / rework 17 / dispatch 28 / correction 6; re-measure 53 / 33 / 24 / 3 / 6.
- **d. Version stamps:** every re-measure record is stamped v2. The baseline records carry no stamp; they are v1 records patched to v2 during the gate week.
- **e. Safety:** vault writes were only the instrument's temporary worktrees, which Joe allowed. One worktree left behind by a killed smoke was removed. The final vault check matched the pre-run state.

## 4. Escalation (task 3.4): options only

No numeric bar is pre-registered. The plan says: "If W2 moves below Joe's bar… escalate per design D-F… If W2 hits or exceeds Joe's bar, park the escalations further." The options are:

1. a watcher/metacognition layer;
2. a mechanical validator on the LESSONS contract;
3. a hooks/harness push (Pi-era);
4. park all three further.

None is recommended here. Caveat 3a is relevant input to that call.

## 5. Artifacts

- `results-re-measure-2026-09-28/`: `sample.json`, `moments.jsonl` (119), `transcript-events.jsonl` (47), `cost-log.jsonl` (323 calls), `engram-query-failures.jsonl` (2), `run-manifest.json`, `scorecard-remeasure.json`.
- `run_window_sample.py` and `test_run_window_sample.py`: the one-window driver.
- `compute_remeasure.py`: metric computation.
- `run_audit.py` and `test_run_audit.py`: the date filter and exclusion rules.
- `dev/eval/LEDGER.md`: row `learn-rate-remeasure-w2`.
