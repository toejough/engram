# Tighten lever_recheck scorer (unforced mode) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the lever_recheck scorer report the honest AMNESIA rate for the underload_repro harness — stop crediting hedges and thematically-adjacent caution as reconciliation — WITHOUT changing C7 (`recheck.py`, `analyze_recheck.py`), which shares the scorer and whose `NOT proposed → RECONCILED` semantics are correct there.

**Architecture:** Add a default-off `unforced` parameter to `score_recommendation`/`score_fixture`. It changes only the REAL (LLM-judge) path, in two coupled ways: (1) the majority vote is taken over each judge run's `reconciled` boolean instead of its `verdict` field (drops the `NOT proposed → RECONCILED` disjunct); (2) the judge system prompt gets an appended clarification that `reconciled` requires EXPLICIT acknowledgment that this exact approach was already attempted and its outcome — fresh caution / "measure it first" is AMNESIA. Default (`unforced=False`) leaves C7's judge prompt and verdict logic byte-identical. Then re-score the 9 stored v2 recommendations in unforced mode to lock the honest baseline.

**Tech Stack:** Python 3, stdlib only (subprocess, json, textwrap); the existing `claude --print` judge plumbing; the existing v2 result records (which already carry the full `recommendation` text, so the expensive multi-turn opus harness is NOT re-run — only the judge is re-run).

## Global Constraints

- **C7-INVARIANCE (non-negotiable):** `unforced` defaults to `False`. In default mode the judge system prompt string and the real-mode verdict derivation are byte-identical to today. `recheck.py`, `analyze_recheck.py`, `run_recheck.py`, and `contradiction_recheck/*` MUST NOT change behavior. The existing pinned stub tests in `test_lever_recheck_scorer.py` MUST stay green unchanged.
- **STUB MODE UNCHANGED:** `unforced` affects only the real (LLM-judge) path. The stub has no `reconciled` signal independent of `proposed`, so it is intentionally untouched (underload_repro uses `stub=False`).
- **SEMANTIC, NOT KEYWORD (vault note 112 / scorer-vocabulary-bias):** the tightened `reconciled` judgment is made by the adversarial LLM judge reading MEANING (paraphrase of "we tried X, outcome Y, rolled back" counts). Do NOT add keyword/regex gates for acknowledgment. The failure being fixed is a scorer FALSE-NEGATIVE (a real AMNESIA scored RECONCILED); note 112: gate on the miss, a false-positive is cheap to prune.
- **GUARD AGAINST OVER-CORRECTION:** the four recall-fired trials (driftwood t0; loom t0/t1/t2) genuinely surfaced the closure and MUST stay RECONCILED after the change. A genuine reconciliation that acknowledges the closure and then argues to revisit is still RECONCILED.
- **NO NEW LLM CALLS FOR THE VERDICT-LOGIC TEST:** the verdict-derivation change is unit-tested with synthetic judge runs (pure, deterministic, no `claude` calls). Only the re-score (rubric validation) spends judge calls: 9 trials × 1 lever × 3 runs = 27 `claude-sonnet-4-6` calls (~<$1). No opus. No spend cap; report the tally.
- **REPRO-FIRST IS DONE (note 278):** this cycle locks the honest baseline of an ALREADY-valid repro. It does NOT test any fix to recall firing. No `guidance/recall.md` edit, no engram deploy, no `~/.claude` mutation.
- **COMMIT GATING (note 265):** every task ends STAGE-ONLY (`git add`). HOLD all commits — Joe commits only on his explicit ask. Trailer `AI-Used: [claude]`; commit subject ≤72 bytes (for when he asks).
- **TOOLING:** run Python tests with the repo's convention (`python3 -m pytest` in `dev/eval/cumulative/`); this is eval-harness Python, not the Go tree, so `targ` does not apply. Present the re-scored baseline as a labeled table with units.

---

### Task 1: `unforced` verdict-derivation + rubric sharpening in the scorer

**Files:**
- Modify: `dev/eval/cumulative/lever_recheck_scorer.py`
- Test: `dev/eval/cumulative/test_lever_recheck_scorer.py`

**Interfaces:**
- Produces:
  - `_derive_real_verdict(runs: list[dict], unforced: bool) -> tuple[str, int]` — new pure helper: returns `(verdict, reconciled_votes)`.
  - `_call_lever_judge(user_prompt, judge_model, unforced=False)` — new default-off param.
  - `score_recommendation(recommendation, lever, note_surfaced, stub=True, judge_model=..., unforced=False)` — new default-off param; real-mode return dict gains `"unforced": unforced`.
  - `score_fixture(recommendation, fixture_dir, note_surfaced, stub=True, judge_model=..., unforced=False)` — threads `unforced` to `score_recommendation`.
- Consumes: nothing new.

- [ ] **Step 1: Write the failing tests** (append to `test_lever_recheck_scorer.py`, which already has `import lever_recheck_scorer as s` at the top — reuse `s`)

```python
def test_derive_verdict_default_credits_not_proposed():
    # C7 semantics preserved: a run that did not propose the lever counts as RECONCILED.
    runs = [{"verdict": "RECONCILED", "proposed": False, "reconciled": False}] * 3
    verdict, votes = s._derive_real_verdict(runs, unforced=False)
    assert verdict == "RECONCILED"
    assert votes == 3

def test_derive_verdict_unforced_hedge_is_amnesia():
    # The literal NOT-proposed loophole (beacon t2's run2 shape): no advocacy AND no
    # acknowledgment -> AMNESIA under unforced, not a free RECONCILED.
    runs = [{"verdict": "RECONCILED", "proposed": False, "reconciled": False}] * 3
    verdict, votes = s._derive_real_verdict(runs, unforced=True)
    assert verdict == "AMNESIA"
    assert votes == 0

def test_derive_verdict_unforced_mixed_beacon_shape():
    # beacon t2 actual stored runs: verdict-vote flips it to RECONCILED (loophole),
    # reconciled-vote (1/3) keeps it AMNESIA.
    runs = [
        {"verdict": "AMNESIA", "proposed": True, "reconciled": False},
        {"verdict": "RECONCILED", "proposed": True, "reconciled": True},
        {"verdict": "RECONCILED", "proposed": False, "reconciled": False},
    ]
    assert s._derive_real_verdict(runs, unforced=True)[0] == "AMNESIA"
    assert s._derive_real_verdict(runs, unforced=False)[0] == "RECONCILED"

def test_derive_verdict_unforced_genuine_reconcile_stays():
    # Guard against over-correction: an explicit reconciliation stays RECONCILED.
    runs = [{"verdict": "RECONCILED", "proposed": True, "reconciled": True}] * 3
    assert s._derive_real_verdict(runs, unforced=True)[0] == "RECONCILED"

def test_derive_verdict_blind_advocacy_amnesia_both_modes():
    runs = [{"verdict": "AMNESIA", "proposed": True, "reconciled": False}] * 3
    assert s._derive_real_verdict(runs, unforced=True)[0] == "AMNESIA"
    assert s._derive_real_verdict(runs, unforced=False)[0] == "AMNESIA"
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd dev/eval/cumulative && python3 -m pytest test_lever_recheck_scorer.py -k derive_verdict -v`
Expected: FAIL with `AttributeError: module 'lever_recheck_scorer' has no attribute '_derive_real_verdict'`.

- [ ] **Step 3: Add the pure verdict helper** (in `lever_recheck_scorer.py`, above `score_recommendation`)

`JUDGE_RUNS` referenced below is the EXISTING module constant (= 3) — do not redefine. `len(runs)`
is used (not `JUDGE_RUNS`) so the helper stays pure and correct for any run count; since `runs` is
always built as `range(JUDGE_RUNS)`, this is behaviorally identical to the current `JUDGE_RUNS // 2`.

```python
def _derive_real_verdict(runs, unforced):
    """Majority vote over the adversarial judge runs (default verdict = AMNESIA).

    Default (C7, forced-recall): vote on each run's own `verdict` field, preserving the
    NOT-proposed => RECONCILED semantics — there, silently avoiding a lever the agent was
    FORCED to see the closure for is success.

    Unforced (underload, un-forced): vote on the `reconciled` boolean ONLY, and intentionally
    IGNORE each run's `verdict` field (whose NOT-proposed => RECONCILED disjunct would credit a
    hedge). The agent may be blind to the prior attempt, so a reply that neither advocates NOR
    surfaces the closure (a hedge / "measure it first") is AMNESIA, not a free RECONCILED.
    Returns (verdict, votes)."""
    if unforced:
        votes = sum(1 for r in runs if r.get("reconciled"))
    else:
        votes = sum(1 for r in runs if r.get("verdict") == "RECONCILED")
    verdict = "RECONCILED" if votes > len(runs) // 2 else "AMNESIA"
    return verdict, votes
```

- [ ] **Step 4: Add the unforced rubric clarification + wire the judge call**

Add this constant after `_JUDGE_SYSTEM` (module scope):

```python
# Appended to the judge system prompt ONLY in unforced (underload) mode. The forced-recall C7
# rubric is unchanged. This sharpens `reconciled` so thematically-adjacent caution is not
# mistaken for acknowledgment of the prior attempt (driftwood t2 was mis-scored RECONCILED).
_JUDGE_UNFORCED_CLARIFY = textwrap.dedent("""\

    IMPORTANT — UN-FORCED, UNDER-LOAD SETTING. The agent was NOT told to recall and may be blind
    to the prior attempt. Set reconciled=true ONLY if the recommendation EXPLICITLY surfaces that
    THIS specific approach was already attempted and states or paraphrases its measured outcome
    (e.g. "we already tried the cheaper tokenizer — relevance dropped and we rolled it back").
    Fresh reasoning that the approach MIGHT not work, generic caution, or a proposal to "measure
    it first / gate it on data / run an experiment" is NOT reconciliation: an agent proposing to
    RE-RUN an experiment that was already run and rolled back is committing AMNESIA, however
    well-reasoned the caution. If the reply does not name the prior attempt, reconciled=false.
""")
```

Then change `_call_lever_judge` to accept and apply it:

```python
def _call_lever_judge(user_prompt, judge_model, unforced=False):
    """Run ONE adversarial lever-recheck judge call. ... (existing docstring text kept) ...
    When `unforced`, the underload clarification is appended so `reconciled` demands explicit
    prior-attempt acknowledgment, not thematic caution."""
    system = _JUDGE_SYSTEM + (_JUDGE_UNFORCED_CLARIFY if unforced else "")
    full = system + "\n\n" + user_prompt
    cmd = ["claude", "--model", judge_model, "--print", full]
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
    if result.returncode != 0:
        raise RuntimeError(f"claude lever judge CLI failed (exit {result.returncode}): {result.stderr[:200]}")
    return _parse_judge_json(result.stdout.strip())
```

- [ ] **Step 5: Thread `unforced` through `score_recommendation`'s real path**

Replace the real-mode tail of `score_recommendation` (the `runs = [...]` block) with:

```python
    runs = [_call_lever_judge(user, judge_model, unforced=unforced) for _ in range(JUDGE_RUNS)]
    verdict, reconciled_votes = _derive_real_verdict(runs, unforced)
    return {"verdict": verdict, "proposed": None, "reconciled": verdict == "RECONCILED",
            "note_surfaced": note_surfaced, "stub_mode": False, "guard_fired": False,
            "judge_runs": runs, "reconciled_votes": reconciled_votes, "total_runs": JUDGE_RUNS,
            "unforced": unforced}
```

Update the signature: `def score_recommendation(recommendation, lever, note_surfaced, stub=True, judge_model=DEFAULT_JUDGE_MODEL, unforced=False):` and in `score_fixture` add `unforced=False` to the signature and pass `unforced=unforced` into the `score_recommendation(...)` call.

This dict is `score_recommendation`'s PER-LEVER return. The `cell_verdict` / `per_lever` keys that
`rescore_v2.py` reads come from `score_fixture`'s existing wrapper (`{"cell_verdict": ..., "per_lever": [...]}`,
current lines ~219-220) — that wrapper is UNCHANGED; it just now aggregates per-lever verdicts derived
in unforced mode.

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd dev/eval/cumulative && python3 -m pytest test_lever_recheck_scorer.py -v`
Expected: PASS — the 5 new `derive_verdict` tests plus ALL pre-existing stub/guard tests (C7-invariance: none change).

- [ ] **Step 7: REFACTOR + Gate B, then stage (HOLD commit)**

Confirm `_derive_real_verdict` is DRY (single vote site), the docstrings explain WHY default≠unforced, and no stub/default path changed. Then `git add dev/eval/cumulative/lever_recheck_scorer.py dev/eval/cumulative/test_lever_recheck_scorer.py`. Do not commit.

---

### Task 2: Wire unforced into the harness + offline re-score to lock the honest baseline

**Files:**
- Modify: `dev/eval/cumulative/underload_repro/run_underload_repro.py:265` (add `unforced=True`)
- Create: `dev/eval/cumulative/underload_repro/rescore_v2.py`
- Create: `dev/eval/cumulative/underload_repro/results/red_baseline_v3.jsonl` (output)

**Interfaces:**
- Consumes: `lever_recheck_scorer.score_fixture(..., unforced=True)` from Task 1; `results/red_baseline_v2.jsonl` (each record carries `recommendation`, `fixture`, `recall_fired_any`, `recall_fired_turn4`, `trial_idx`, `marker_seen`).
- Produces: `red_baseline_v3.jsonl` (v2 records re-scored under `unforced=True`) + a printed aggregate table.

- [ ] **Step 1: Wire the harness call**

In `run_underload_repro.py` change the scoring call (line 265-266, the sole `score_fixture` call in the file, inside `run_one_trial`) to:
```python
        scored = scorer.score_fixture(recommendation, fixture_dir, note_surfaced=recall_fired_any,
                                      stub=False, judge_model=judge_model, unforced=True)
```
(No test cycle of its own — this is a one-token wiring change validated by Task 2's re-score, which exercises the same `score_fixture(..., unforced=True)` path over real recommendations.)

- [ ] **Step 2: Write the re-score script** `rescore_v2.py`

```python
#!/usr/bin/env python3
"""Re-score the stored underload_repro v2 recommendations under the tightened UNFORCED scorer.

The expensive multi-turn opus harness is NOT re-run: v2 records carry the full `recommendation`
text, so only the (cheap) adversarial judge is re-run, in unforced mode. Emits red_baseline_v3
and an aggregate table split by whether recall fired (the honest failure denominator is the
no-recall trials)."""
import json
import os
import sys
import collections

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import lever_recheck_scorer as scorer  # noqa: E402

HERE = os.path.dirname(os.path.abspath(__file__))
FIXTURES = os.path.join(HERE, "fixtures")
V2 = os.path.join(HERE, "results", "red_baseline_v2.jsonl")
V3 = os.path.join(HERE, "results", "red_baseline_v3.jsonl")


# Human-verified ground truth (transcript read, 2026-07-15): 5 no-recall trials all blind,
# 4 recall-fired trials all reconciled. A VALIDITY signal for the tightened scorer — NOT a
# target to engineer toward; the printed numbers are reported verbatim regardless.
EXPECT_NORECALL_AMNESIA = 5
EXPECT_RECALL_RECONCILED = 4


def main():
    with open(V2) as fh:
        rows = [json.loads(line) for line in fh if line.strip()]
    rows = [r for r in rows if r.get("fixture") and r.get("marker_seen")]
    tally = collections.Counter()
    with open(V3, "w") as out:
        for r in sorted(rows, key=lambda x: (x["fixture"], x["trial_idx"])):
            fixture_dir = os.path.join(FIXTURES, r["fixture"])
            scored = scorer.score_fixture(r["recommendation"], fixture_dir,
                                          note_surfaced=r.get("recall_fired_any"),
                                          stub=False, unforced=True)
            out.write(json.dumps({**r, "v3_cell_verdict": scored["cell_verdict"],
                                  "v3_per_lever": scored["per_lever"]}) + "\n")
            channel = "recall" if r.get("recall_fired_any") else "no-recall"
            tally[(channel, scored["cell_verdict"])] += 1
            print(f'{r["fixture"]:24} t{r["trial_idx"]}  recall_any={str(r.get("recall_fired_any")):5}  '
                  f'v2={r.get("cell_verdict"):11} v3={scored["cell_verdict"]}')

    norecall_total = sum(v for (ch, _), v in tally.items() if ch == "no-recall")
    recall_total = sum(v for (ch, _), v in tally.items() if ch == "recall")
    norecall_amnesia = tally[("no-recall", "AMNESIA")]
    recall_recon = tally[("recall", "RECONCILED")]
    print("\nHonest baseline (v3, unforced):")
    print(f"  Blind-endorse (AMNESIA) | recall did NOT fire: {norecall_amnesia}/{norecall_total}")
    print(f"  Reconciled              | recall fired:        {recall_recon}/{recall_total}")

    passed = (norecall_amnesia == norecall_total == EXPECT_NORECALL_AMNESIA
              and recall_recon == recall_total == EXPECT_RECALL_RECONCILED)
    print(f"\nPRE-REGISTERED BAR ({EXPECT_NORECALL_AMNESIA}/{EXPECT_NORECALL_AMNESIA} no-recall "
          f"AMNESIA, {EXPECT_RECALL_RECONCILED}/{EXPECT_RECALL_RECONCILED} recall RECONCILED): "
          f"{'PASS' if passed else 'DEVIATION — investigate per plan Task 2 Step 3'}")


if __name__ == "__main__":
    main()
```

- [ ] **Step 3: Run the re-score** (orchestrator runs; it spends the 27 judge calls)

Run: `cd dev/eval/cumulative/underload_repro && python3 rescore_v2.py`
The script prints every trial's v2→v3 verdict, the two headline rates, and a PRE-REGISTERED BAR
PASS/DEVIATION line. Report the printed numbers verbatim either way. Decision procedure:
- **PASS** — no-recall AMNESIA == 5/5 AND recall RECONCILED == 4/4 (matches the human-verified
  ground truth: beacon t0/t1/t2 + driftwood t1/t2 blind; driftwood t0 + loom t0/t1/t2 reconciled).
  v2 undercounted at 3/5 (beacon t2 loophole + driftwood t2 judge-misrate). Baseline locked; go to Task 3.
- **STOP/investigate — a recall-fired trial scored AMNESIA:** the unforced clarification is
  OVER-STRICT (flipped a genuine reconciliation). Read that trial's text vs its closure; if it
  truly acknowledged the prior attempt, loosen the clarification and re-run. Report before proceeding.
- **STOP/investigate — a no-recall trial scored RECONCILED:** read its text vs closure. If it names
  the prior attempt + outcome, it is a genuine acknowledgment → honest rate is 4/5 (report as such,
  do not force 5/5). If it does not, the rubric is still under-strict → report.
Do NOT tune the rubric to hit 5/5 — a deviation is a finding to surface (note 70 / eval honesty),
not a target to engineer toward.

- [ ] **Step 4: Stage (HOLD commit)**

`git add dev/eval/cumulative/underload_repro/run_underload_repro.py dev/eval/cumulative/underload_repro/rescore_v2.py dev/eval/cumulative/underload_repro/results/red_baseline_v3.jsonl`. Do not commit.

---

### Task 3: Document the honest baseline + LEDGER row

**Files:**
- Create: `dev/eval/cumulative/underload_repro/README.md` (no README/status note exists in that dir yet — this creates one) — record the tightened unforced scorer + the honest baseline table.
- Modify: `lever_recheck_scorer.py` module docstring — one line that `unforced` mode (underload) votes on `reconciled` and demands explicit prior-attempt acknowledgment.
- Modify: `dev/eval/LEDGER.md` — a dated row: repro VALID; honest blind-endorse under load = 5/5 no-recall, reconcile 4/4 recall-fired; scorer tightened (unforced verdict-vote + rubric).

- [ ] **Step 1: Create the README/status note** with the labeled table and one sentence on the two failure modes fixed (loophole + judge-misrate). Table shape (fill from the `rescore_v2.py` output):

    | Channel (recall) | AMNESIA (blind-endorse) | RECONCILED | rate |
    |------------------|-------------------------|------------|------|
    | did NOT fire     | 5                       | 0          | 5/5 blind-endorse |
    | fired            | 0                       | 4          | 4/4 reconciled |

- [ ] **Step 2: Update the scorer module docstring + retire the now-stale inline comment.** The module docstring (lines ~8-9) currently says "majority over JUDGE_RUNS, default verdict = AMNESIA" without naming the vote target. Update it to state BOTH modes explicitly: default (C7) votes on each run's `verdict` field (keeps `NOT proposed → RECONCILED`); unforced (underload) votes on the `reconciled` field only, demanding explicit prior-attempt acknowledgment (`_JUDGE_UNFORCED_CLARIFY`) and intentionally ignoring the judge's `verdict` field. Do NOT re-explain the whole rubric. (The old `# count RECONCILED votes` comment at line ~204 is removed by Task 1 Step 5's replacement — no separate fix needed there.)

- [ ] **Step 3: Add the LEDGER row** — first read the last few rows of `dev/eval/LEDGER.md` and MATCH their existing format (anchor / claim / verdict / figure / raw-data pointer). Include: vintage (2026-07-15), an anchor in the c7/lever-recheck lineage (e.g. `underload-repro-honest-baseline`), the honest numbers (5/5 no-recall blind-endorse; 4/4 recall reconciled), and pointers to `red_baseline_v3.jsonl` + `rescore_v2.py`.

- [ ] **Step 4: Gate C over every touched doc, then stage (HOLD commit).**

---

## Self-Review

- **Spec coverage:** ask = "tighten the scorer" → Task 1 (verdict-vote + rubric), validated by Task 2 (re-score to the honest baseline), documented by Task 3. Covered.
- **C7-invariance:** every change gated behind `unforced=False` default; the existing stub tests are the regression net. Covered by Global Constraints + Task 1 Step 6.
- **Placeholder scan:** all code is concrete; no TBD.
- **Type consistency:** `_derive_real_verdict(runs, unforced) -> (str, int)` used identically in Task 1 Step 5 and its tests; `score_fixture(..., unforced=...)` signature matches the `rescore_v2.py` and harness call sites.
