# Cost-meter + Trap Regression Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a directly-billed `recall_cost` to the cumulative eval harness and a tiered C3/C4-idio/C5/C6 trap regression gate, so engram cost/usage optimization can proceed measurably and without silently eroding the verified capability wins.

**Architecture:** Component 1 splits the harness's fused round-1 (recall+build) into a recall-only `claude` call plus a `--resume`'d build call, making `recall_cost` a billed figure and `build_cost` recall-free; consumers are swept so op-cost totals stay whole. Component 2 adds the missing C3 seed and a thin `gate.py` that runs the existing warm trap harnesses, normalizes their results to a common trial shape, and emits a GREEN/RED/INCONCLUSIVE verdict.

**Tech Stack:** Python 3 (eval harness in `dev/eval/cumulative/` + `dev/eval/traps/`), pytest, the `engram` and `claude` CLIs.

## Global Constraints

- **Tests run via `pytest`, not `targ`.** `targ` governs the Go module; the eval harness is Python and already ships pytest files in `dev/eval/cumulative/`. (CLAUDE.md's "use targ" applies to the Go code, not this Python harness.)
- **Match the existing `cumulative/` pytest house style:** plain `def test_*` functions with `assert`; `import os`/`import sys` as needed; module docstring; **no `import pytest`** unless `pytest.raises` is actually used. Templates: `test_build_prompt.py`, `test_recheck.py`.
- **Fail loud, never silent-fallback:** a missing fixture/seed must `raise`/`sys.exit(non-zero)`, never silently strawman to a pass (MEMORY.md: eval-fail-loud, don't-bypass-component-under-test).
- **`recall_cost` must be added to EVERY op-cost summation** post-split — partial sweep makes warm silently cheaper (note 66 + the meter's whole purpose).
- **Verify with the real binary before "done":** a real `--tier smoke` gate run and a real warm harness run, not only unit tests (MEMORY.md: verify-cli-real-binary).
- **No spend cap:** smoke ~$3, full ~$18 estimated up front; let runs finish.
- Reuse existing warm harnesses (`wrun.py`, `c4_idio.py`, `seed_c5.py`+`c5.py`, `c6_clean.py --arm warm`) — do not reimplement trial logic.

---

# Component 1 — $METER (recall_cost)

### Task 1: `build_prompt` gains `include_recall`; extract `recall_only_prompt`

**Files:**
- Modify: `dev/eval/cumulative/harness.py:133-164` (`build_prompt`)
- Create (same file, near `build_prompt`): `recall_only_prompt(app)`
- Test: `dev/eval/cumulative/test_recall_split.py` (new)

**Interfaces:**
- Produces: `build_prompt(app, interface, read_mode, checklist=False, include_recall=True) -> str`; `recall_only_prompt(app) -> str`

- [ ] **Step 1: Write failing tests**

```python
"""TDD for the $METER session-split: build_prompt can omit the recall directive while keeping
the checklist gating, and a recall-only prompt exists for the separate recall call."""
import os, sys
sys.path.insert(0, os.path.dirname(__file__))
import harness

def test_build_prompt_include_recall_false_omits_recall_directive():
    p = harness.build_prompt("todo", "add/list", "skill", include_recall=False)
    assert "/recall" not in p
    assert "go mod init todo" in p  # still a real build prompt

def test_build_prompt_include_recall_false_keeps_checklist_gating():
    p = harness.build_prompt("todo", "add/list", "skill", checklist=True, include_recall=False)
    assert "checklist" in p.lower()           # gating block survives
    assert "/recall" not in p                 # but no recall directive

def test_recall_only_prompt_invokes_recall_and_stops_without_building():
    p = harness.recall_only_prompt("todo")
    assert "/recall" in p
    assert "go mod init" not in p             # must NOT build
    assert "STOP" in p.upper() or "do not write" in p.lower()
```

- [ ] **Step 2: Run to verify FAIL** — `cd dev/eval/cumulative && python3 -m pytest test_recall_split.py -v` → FAIL (`include_recall` unexpected kw / `recall_only_prompt` missing).

- [ ] **Step 3: Implement.** In `build_prompt`, add `include_recall=True` to the signature; change the recall guard to `if read_mode == "none" or not include_recall: recall = ""`. Leave the `gating` block keyed on `checklist and read_mode == "skill"` unchanged. Add:

```python
def recall_only_prompt(app):
    """Recall-only message for the split round-1: run /recall, print impact, then STOP — no build.
    Used so recall_cost/recall_s can be measured on their own billed claude call."""
    return (
        "Consult your memory by INVOKING YOUR /recall SKILL — actually run the skill (it prints its "
        "Step 0 plan, queries the vault, and synthesizes impact). Do NOT hand-run `engram query` "
        f"yourself. Frame the recall around building a command-line {app} in Go and its "
        "architecture/conventions. Read every note the skill surfaces. Then STOP: do not write any "
        "code, do not run `go mod init`, do not create files — only complete the recall and print its "
        "one-line impact summary.")
```

- [ ] **Step 4: Run to verify PASS** — same pytest command → 3 passed.
- [ ] **Step 5: Commit** — `git add dev/eval/cumulative/harness.py dev/eval/cumulative/test_recall_split.py && git commit -m "feat(eval): build_prompt include_recall flag + recall_only_prompt for $METER split"`

### Task 2: Pure cost-split helper

**Files:**
- Modify: `dev/eval/cumulative/harness.py` (add helper near `_round_rec`)
- Test: `dev/eval/cumulative/test_recall_split.py` (extend)

**Interfaces:**
- Produces: `split_costs(recall_res, rounds) -> (recall_cost: float, build_cost: float)` where `recall_res` is the recall call's result dict (or `None` for cold) and `rounds` is the list of `_round_rec` dicts.

- [ ] **Step 1: Write failing tests**

```python
def test_split_costs_warm_separates_recall_from_build():
    recall_res = {"total_cost_usd": 0.5}
    rounds = [{"cost": 1.0}, {"cost": 0.5}]
    rc, bc = harness.split_costs(recall_res, rounds)
    assert rc == 0.5
    assert bc == 1.5          # build = sum(rounds), recall excluded

def test_split_costs_cold_recall_is_zero():
    rounds = [{"cost": 2.0}]
    rc, bc = harness.split_costs(None, rounds)
    assert rc == 0.0
    assert bc == 2.0
```

- [ ] **Step 2: Run to verify FAIL** — `python3 -m pytest test_recall_split.py -v` → FAIL (`split_costs` missing).
- [ ] **Step 3: Implement**

```python
def split_costs(recall_res, rounds):
    """Separate the billed recall cost from the build cost. recall_res is the recall-only call's
    result (None for cold). build_cost is the sum of build rounds and never includes recall."""
    recall_cost = round((recall_res or {}).get("total_cost_usd", 0) or 0, 4) if recall_res else 0.0
    build_cost = round(sum(r["cost"] for r in rounds), 4)
    return recall_cost, build_cost
```

- [ ] **Step 4: Run to verify PASS** — pytest → all passed.
- [ ] **Step 5: Commit** — `git commit -am "feat(eval): split_costs pure helper separating recall_cost from build_cost"`

### Task 3: Wire the two-call split into `run_build`

**Files:**
- Modify: `dev/eval/cumulative/harness.py:673-803` (`run_build` recall/build brackets + cost assembly)

**Interfaces:**
- Consumes: `recall_only_prompt`, `build_prompt(..., include_recall=False)`, `split_costs`.
- Produces: `out` dict gains `recall_cost`; `recall_s` = recall-only; `build_s` = all build rounds incl. round-1.

This task is I/O orchestration; it is verified by Task 4's `validate.py` green + the real smoke run in Task 9, not a unit test (per "test logic, not I/O").

- [ ] **Step 1: Restructure the phases.** Replace the round-1 block (currently `t_recall_start`/`do_build(prompt)`/`t_recall_end` at 673-675 and the build prompt construction at 654-655) with:

```python
    interface = json.load(open(args.spec))["interface"]
    warm = regime["read_mode"] == "skill" and not args.stub

    # Phase 1 — recall only (warm). Billed on its own claude call so recall_cost/recall_s are clean.
    recall_res, sid = None, None
    t_recall_start = time.time()
    if warm:
        recall_res = do_build(recall_only_prompt(args.app))
        sid = recall_res.get("session_id")
        if recall_res.get("is_error") and (recall_res.get("total_cost_usd", 0) or 0) < 0.02:
            print(f"recall call FAILED ({args.app} {args.regime}) — likely rate_limit; no result "
                  f"written so resume re-runs it.", file=sys.stderr)
            sys.exit(1)
    t_recall_end = time.time()

    # Phase 2 — build (round-1 + feedback rounds), resumed in the same session for warm.
    build_msg = build_prompt(args.app, interface, regime["read_mode"],
                             checklist=regime.get("checklist", False), include_recall=False)
    t_build_start = time.time()
    res = do_build(build_msg, resume_sid=sid)
    # ... existing round-1 scoring, the build!=ok error-exit (685-688), rounds=[...], the while loop ...
    t_build_end = time.time()
```

Move the existing `t_build_start`/`t_build_end` so they bracket round-1 **and** the feedback loop (delete the old inner `t_build_start` at 712). The recall-fired check (760-764) stays, using `sid` (the recall call's session, which the build resumed).

- [ ] **Step 2: Assemble costs via the helper.** Replace `build_cost = round(sum(...))` (803) with `recall_cost, build_cost = split_costs(recall_res, rounds)`. Add to the `out` dict (next to `build_cost`, ~837): `"recall_cost": recall_cost,` and in the axis block (~848): `"axis_c2_recall_cost": recall_cost,`.
- [ ] **Step 3: Verify nothing references the removed inner bracket** — `grep -n "t_build_start\|t_recall" harness.py` shows each defined once, bracketing the right phase.
- [ ] **Step 4: Smoke-run the stub path** (no spend) — `python3 harness.py build --stub ...` (use an existing stub invocation from `validate.py`/`matrix.py`) → emits a result JSON containing `recall_cost` (0.0 for stub/cold). Read the file; confirm the field is present.
- [ ] **Step 5: Commit** — `git commit -am "feat(eval): split round-1 into recall-only + resumed build; emit billed recall_cost (schema-affecting)"`

### Task 4: Schema v5 + consumer sweep (note 66)

**Files:**
- Modify: `dev/eval/cumulative/harness.py:31` (`SCHEMA_VERSION = 5`)
- Modify: `dev/eval/cumulative/aggregate.py` (every op-cost summation + axis CI table + field map)
- Modify: `dev/eval/cumulative/matrix.py` (chain `$` rollup)
- Modify: `dev/eval/cumulative/validate.py:141` (`axis_fields`)

**Interfaces:** Consumes the v5 `out` dict with `recall_cost`/`axis_c2_recall_cost`.

- [ ] **Step 1: Bump** `SCHEMA_VERSION = 4` → `5` (harness.py:31).
- [ ] **Step 2: Sweep aggregate.py op-cost sums.** At EACH site that sums an op's dollars (240-243, 269, 342, 363, 521, 574, 685 per grep), add `recall_cost`. Concretely the op-cost of a build row becomes `(b.get("build_cost",0) or 0) + (b.get("recall_cost",0) or 0) + ((ln or {}).get("learn_cost",0) or 0)`. Add `("recall cost", "USD", "axis_c2_recall_cost", False)` to the axis CI table (near 291-293) and `("recall_cost", "axis_c2_recall_cost")` to the field map (near 78-81).
- [ ] **Step 3: Sweep matrix.py chain $** (line ~178): chain dollars = `build_cost + learn_cost + recall_cost`.
- [ ] **Step 4: Update validate.py axis_fields** (141): add `"axis_c2_recall_cost"` to the list so the stub dry-pass asserts the new field is emitted.
- [ ] **Step 5: Run validate.py to GREEN** — `cd dev/eval/cumulative && python3 validate.py` → all checks pass (this is the note-66 consumer self-check; do not declare done until green).
- [ ] **Step 6: Commit** — `git commit -am "feat(eval): schema v5 — recall_cost into aggregate/matrix op-cost sums + validate; green"`

---

# Component 2 — Trap Regression Gate

### Task 5: `seed_c3.py` — the missing C3 warm-vault fixture

**Files:**
- Create: `dev/eval/traps/seed_c3.py`
- Test: `dev/eval/traps/test_gate.py` (new — pure list assertion only)

**Interfaces:**
- Produces: `C3_NOTES: list[dict]` (each `{slug, situation, subject, predicate, object}`) and `seed(vault_path)` that runs `engram learn fact` per note into `vault_path`.

- [ ] **Step 1: Write failing test** (pure — the 5 conventions are present and map to their code forms):

```python
"""TDD for the trap regression gate: C3 seed covers all 5 conventions; verdict logic is correct."""
import os, sys
sys.path.insert(0, os.path.dirname(__file__))
import seed_c3

def test_c3_seed_covers_all_five_conventions():
    objs = " ".join(n["object"] + n["subject"] for n in seed_c3.C3_NOTES)
    for form in ["NewRequestWithContext", "NO_COLOR", "t.Parallel(", "%w", "len("]:
        assert form in objs, f"C3 seed missing convention form {form}"
    assert len(seed_c3.C3_NOTES) == 5
```

- [ ] **Step 2: Run to verify FAIL** — `cd dev/eval/traps && python3 -m pytest test_gate.py -v` → FAIL (no `seed_c3`).
- [ ] **Step 3: Implement** mirroring `c4_idio.seed_vaults` (use its `engram learn fact` subprocess pattern). `C3_NOTES` = the 5 conventions with objects naming the exact code forms (`NewRequestWithContext`, `NO_COLOR` gate, `t.Parallel()`, nil/`len(` guard before index, `fmt.Errorf("...%w", err)`). `seed(vault)` raises if `engram learn` returns non-zero (fail loud).
- [ ] **Step 4: Run to verify PASS** — pytest → passed.
- [ ] **Step 5: Real seed check** (uses the real binary) — `python3 seed_c3.py /tmp/c3-seed-test && engram query --phrase "http request in Go" ...` (set `ENGRAM_VAULT_PATH=/tmp/c3-seed-test`) surfaces the req-with-context note. Confirm, then `rm -rf /tmp/c3-seed-test`.
- [ ] **Step 6: Commit** — `git commit -m "feat(eval): seed_c3.py — committed C3 warm-vault fixture (5 convention notes)"`

### Task 6: Pure gate verdict core

**Files:**
- Create: `dev/eval/traps/gate_verdict.py`
- Test: `dev/eval/traps/test_gate.py` (extend)

**Interfaces:**
- Produces:
  - `axis_verdict(trials, bar, contam_threshold=0.2) -> dict` where each trial is `{"pass": bool, "contaminated": bool}`; returns `{"valid": int, "contaminated": int, "passed": int, "bar": int, "status": "GREEN"|"RED"|"INCONCLUSIVE"}`.
  - `gate_verdict(axes: dict[str, dict]) -> dict` returning `{"verdict": "GREEN"|"RED"|"INCONCLUSIVE", "axes": axes}`.

- [ ] **Step 1: Write failing tests**

```python
import gate_verdict as gv

def test_axis_all_pass_is_green():
    t = [{"pass": True, "contaminated": False}] * 5
    assert gv.axis_verdict(t, bar=5)["status"] == "GREEN"

def test_axis_one_valid_miss_is_red():
    t = [{"pass": True, "contaminated": False}] * 4 + [{"pass": False, "contaminated": False}]
    assert gv.axis_verdict(t, bar=5)["status"] == "RED"

def test_axis_high_contamination_is_inconclusive():
    t = [{"pass": True, "contaminated": True}] * 3 + [{"pass": True, "contaminated": False}] * 2
    assert gv.axis_verdict(t, bar=5)["status"] == "INCONCLUSIVE"   # 3/5 = 60% > 20%

def test_axis_contaminated_excluded_rest_pass_green():
    # 1 contaminated of 5 (20%, not over), remaining 4 all pass, bar adjusts to valid count
    t = [{"pass": True, "contaminated": True}] + [{"pass": True, "contaminated": False}] * 4
    v = gv.axis_verdict(t, bar=5)
    assert v["valid"] == 4 and v["passed"] == 4 and v["status"] == "GREEN"

def test_gate_red_if_any_axis_red():
    axes = {"C3": {"status": "GREEN"}, "C6": {"status": "RED"}}
    assert gv.gate_verdict(axes)["verdict"] == "RED"

def test_gate_inconclusive_if_any_inconclusive_and_none_red():
    axes = {"C3": {"status": "GREEN"}, "C5": {"status": "INCONCLUSIVE"}}
    assert gv.gate_verdict(axes)["verdict"] == "INCONCLUSIVE"
```

- [ ] **Step 2: Run to verify FAIL** — pytest → FAIL (no `gate_verdict`).
- [ ] **Step 3: Implement**

```python
"""Pure verdict logic for the trap regression gate — no I/O, unit-tested."""

def axis_verdict(trials, bar, contam_threshold=0.2):
    n = len(trials)
    contaminated = sum(1 for t in trials if t["contaminated"])
    valid = [t for t in trials if not t["contaminated"]]
    passed = sum(1 for t in valid if t["pass"])
    if n and contaminated / n > contam_threshold:
        status = "INCONCLUSIVE"
    elif passed == len(valid) and len(valid) > 0:
        status = "GREEN"          # exact bar: every VALID trial passes
    else:
        status = "RED"
    return {"valid": len(valid), "contaminated": contaminated, "passed": passed,
            "bar": bar, "status": status}

def gate_verdict(axes):
    statuses = [a["status"] for a in axes.values()]
    if "RED" in statuses:
        verdict = "RED"
    elif "INCONCLUSIVE" in statuses:
        verdict = "INCONCLUSIVE"
    else:
        verdict = "GREEN"
    return {"verdict": verdict, "axes": axes}
```

- [ ] **Step 4: Run to verify PASS** — pytest → all passed.
- [ ] **Step 5: Commit** — `git commit -m "feat(eval): gate_verdict — pure RED/GREEN/INCONCLUSIVE trap-gate logic"`

### Task 7: Per-axis result adapters (normalize JSON → trials)

**Files:**
- Modify: `dev/eval/traps/gate_verdict.py` (add adapters)
- Test: `dev/eval/traps/test_gate.py` (extend)

**Interfaces:**
- Produces: `normalize(axis: str, rows: list[dict]) -> list[{"pass","contaminated"}]` for axis in `{"C3","C4i","C5","C6"}`, encoding each harness's real keys:
  - C3 (`warm-results.json`): pass = `verdict=="applied"`; contaminated = `verdict=="nobuild"`.
  - C4i (`c4-idio-results.json`, warm-XXp arm only): contaminated = `not built`; pass = `score["supersession_correct"]`.
  - C5 (`c5-results.json`, warm arm): contaminated = `not built`; pass = `honored`.
  - C6 (`c6-warm.json`): contaminated = empty `answer` (degraded build, no `built` field); pass = `hit`.

- [ ] **Step 1: Write failing tests**

```python
def test_normalize_c3_applied_pass_nobuild_contaminated():
    rows = [{"verdict": "applied"}, {"verdict": "trap"}, {"verdict": "nobuild"}]
    out = gv.normalize("C3", rows)
    assert out[0] == {"pass": True, "contaminated": False}
    assert out[1] == {"pass": False, "contaminated": False}
    assert out[2]["contaminated"] is True

def test_normalize_c6_empty_answer_is_contaminated():
    rows = [{"hit": True, "answer": "HIT because..."}, {"hit": False, "answer": ""}]
    out = gv.normalize("C6", rows)
    assert out[0] == {"pass": True, "contaminated": False}
    assert out[1]["contaminated"] is True

def test_normalize_c5_unbuilt_contaminated():
    rows = [{"built": True, "honored": True}, {"built": False, "honored": False}]
    out = gv.normalize("C5", rows)
    assert out[0] == {"pass": True, "contaminated": False}
    assert out[1]["contaminated"] is True
```

- [ ] **Step 2: Run to verify FAIL.**
- [ ] **Step 3: Implement** `normalize` with a per-axis branch using the exact keys above; C4i filters to the `warm-XXp` arm; unknown axis `raise ValueError` (fail loud).
- [ ] **Step 4: Run to verify PASS.**
- [ ] **Step 5: Commit** — `git commit -m "feat(eval): per-axis trap-result adapters normalizing to (pass, contaminated)"`

### Task 8: `gate.py` orchestrator

**Files:**
- Create: `dev/eval/traps/gate.py`

**Interfaces:** Consumes `seed_c3`, `gate_verdict` (`normalize`/`axis_verdict`/`gate_verdict`). CLI: `python3 gate.py --tier smoke|full [--workers N]`.

This is orchestration (shells out to the warm harnesses); verified by the real smoke run in Task 9, not a unit test.

- [ ] **Step 1: Implement.** Tier config:
  `SMOKE = {"C3": 1, "C4i": 1, "C5": 1, "C6": 1}` (C3 runs all 5 traps × this rep count; C6 runs 2 cases × this), `FULL = {"C3": 5, "C4i": 5, "C5": 5, "C6": 4}`. Bars derived from valid counts. For each axis: create a temp `TRAPS_ROOT`, run its warm harness via `subprocess` (`seed_c3`+`wrun.py --vault <V> --n <r>`; `c4_idio.py --n <r>`; `seed_c5.py`+`c5.py --n <r>`; `c6_clean.py --arm warm --n <r>`), **`sys.exit(non-zero)` if any harness errors or its JSON is missing** (fail loud — do not score a missing axis as pass). Read each axis JSON, `normalize`, `axis_verdict`, then `gate_verdict`. Print a labeled table (axis | valid | contaminated | passed | bar | status) and the overall verdict; write `gate-verdict.json`. Exit 0 only on GREEN; non-zero on RED/INCONCLUSIVE (so it works as a pre-merge check).
- [ ] **Step 2: Lint/parse check** — `python3 -c "import ast; ast.parse(open('dev/eval/traps/gate.py').read())"` → no error.
- [ ] **Step 3: Commit** — `git commit -m "feat(eval): gate.py — tiered C3/C4i/C5/C6 trap regression gate"`

### Task 9: Real smoke verification (the no-bypass check)

- [ ] **Step 1:** Estimate + announce cost (~$3). Run `cd dev/eval/traps && python3 gate.py --tier smoke`.
- [ ] **Step 2:** Confirm it actually invoked the real harnesses (real `claude` trials ran, costs are non-zero), emitted the per-axis table, and produced a GREEN/RED/INCONCLUSIVE verdict + `gate-verdict.json`. A first run on the current (unchanged) skills should be GREEN (or INCONCLUSIVE on transient trouble — re-run), never a silent pass with zero trials.
- [ ] **Step 3:** No commit (verification only); if the run surfaced a real bug, fix and re-run.

---

## Self-Review

- **Spec coverage:** $METER session-split (T1-T3) ✓; schema v5 + consumer sweep with validate green (T4) ✓; seed_c3 (T5) ✓; tiered gate with exact bars + contamination→INCONCLUSIVE (T6) ✓; per-axis adapters incl. C6 empty-answer (T7) ✓; gate.py reuse of warm harnesses + fail-loud (T8) ✓; real smoke verify (T9) ✓; build order $METER→gate ✓.
- **Placeholder scan:** none — every code/test step has real content; the one cross-file reference (`split_costs`, `recall_only_prompt`, `normalize`) is defined in an earlier task.
- **Type consistency:** trial shape `{"pass","contaminated"}` is consistent across `normalize`/`axis_verdict`; `axes` dict-of-status consistent across `axis_verdict`/`gate_verdict`.
- **Note on `bar`:** with exact bars over valid trials, `bar` is informational (GREEN requires `passed == len(valid)`), not a separate threshold — kept in the output for the report only.
