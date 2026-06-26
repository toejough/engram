# Cost-meter + Trap Regression Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a directly-billed `recall_cost` to the cumulative eval harness and a tiered C3/C4-idio/C5/C6 trap regression gate, so engram cost/usage optimization can proceed measurably and without silently eroding the verified capability wins.

**Architecture:** Component 1 splits the harness's fused round-1 (recall+build) into a recall-only `claude` call plus a `--resume`'d build call, making `recall_cost` a billed figure and `build_cost` recall-free; consumers are swept so op-cost totals stay whole. Component 2 adds the missing C3 seed and a thin `gate.py` that runs the existing warm trap harnesses, normalizes their results to a common trial shape, and emits a GREEN/RED/INCONCLUSIVE verdict.

**Tech Stack:** Python 3 (eval harness in `dev/eval/cumulative/` + `dev/eval/traps/`), pytest, the `engram` and `claude` CLIs.

## Global Constraints

- **Tests run via `pytest`, not `targ`.** `targ` governs the Go module; the eval harness is Python and **already ships pytest files** in `dev/eval/cumulative/` (`test_build_prompt.py`, `test_recheck.py`, `test_lever_recheck_scorer.py`) — this is the established pre-existing pattern, not a new deviation. CLAUDE.md's "use targ" applies to the Go code, not this Python harness.
- **Match the existing `cumulative/` pytest house style:** plain `def test_*` functions with `assert`; `import os`/`import sys` as needed; module docstring; **no `import pytest`** unless `pytest.raises` is actually used. Templates: `test_build_prompt.py`, `test_recheck.py`.
- **Fail loud, never silent-fallback:** a missing fixture/seed must `raise`/`sys.exit(non-zero)`, never silently strawman to a pass (MEMORY.md: eval-fail-loud, don't-bypass-component-under-test).
- **`recall_cost` must be added to EVERY op-cost summation** post-split — partial sweep makes warm silently cheaper (note 66 + the meter's whole purpose). The enumerated sites (verified against the working tree) are in `aggregate.py` (243, 269, 342/344, 363, 521, 574, 685), `matrix.py:op_cost` (178), and **`harness.py`'s own `token_audit` call (805)** — the last is an in-file consumer, easy to miss.
- **Verify with the real binary before "done":** Component 1 needs a real *warm* cumulative run (not just `--stub`, which forces `warm=False` and makes `recall_cost=0` trivially); Component 2 needs a real `--tier smoke` gate run (MEMORY.md: verify-cli-real-binary).
- **No spend cap:** smoke ~$3, full ~$18, one real warm cumulative cell ~$0.20–0.40 estimated up front; let runs finish.
- Reuse existing warm harnesses (`wrun.py`, `c4_idio.py`, `seed_c5.py`+`c5.py`, `c6_clean.py --arm warm`) — do not reimplement trial logic.
- **`engram learn fact` CLI contract (verified this session):** flags `--slug --position {top|continuation|sibling} --source <str> --situation <str> --subject/--predicate/--object <str> --relation "<target>|<rationale>" (repeatable)`; prints the written note path; non-zero exit on error. `seed_c3.py` builds on this exact signature.

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
- [ ] **Step 5: Commit** — `git add dev/eval/cumulative/harness.py dev/eval/cumulative/test_recall_split.py && git commit -m "feat(eval): build_prompt include_recall flag + recall_only_prompt for \$METER split"`

### Task 2: Pure cost-split helper

**Files:**
- Modify: `dev/eval/cumulative/harness.py` (add helper near `_round_rec`)
- Test: `dev/eval/cumulative/test_recall_split.py` (extend)

**Interfaces:**
- Produces: `split_costs(recall_res, rounds) -> (recall_cost: float, build_cost: float)` where `recall_res` is the recall call's result dict (or `None` for cold) and `rounds` is the list of `_round_rec` dicts (each has a `"cost"` key — confirmed harness.py:634).

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
- Modify: `dev/eval/cumulative/harness.py` (`run_build`: prompt construction ~654-655, recall/build brackets 673-781, cost assembly 803-805, out dict 837/848)

**Interfaces:**
- Consumes: `recall_only_prompt`, `build_prompt(..., include_recall=False)`, `split_costs`.
- Produces: `out` dict gains `recall_cost` + `axis_c2_recall_cost`; `recall_s` = recall-only; `build_s` = all build rounds incl. round-1.

This task is I/O orchestration; its logic is covered by Tasks 1-2 unit tests, and it is verified by the real warm run (Step 5) + `validate.py` green (Task 4), not a new unit test (per "test logic, not I/O").

- [ ] **Step 1 (review-only): read the current flow.** Read `run_build` lines 643-810 of `harness.py` to hold the current structure in context: prompt build (654-655), `do_build` closure (657-671), round-1 (673-679), the build!=ok error-exit (685-688), `rate_limited` (690), the feedback `while` loop (709-750), the recall-fired check (760-764), the in-session learn (770-781), cost assembly (803-805). No edits this step.

- [ ] **Step 2: Restructure into two phases.** Replace the prompt construction (654-655) and the round-1 bracket (`t_recall_start`/`do_build(prompt)`/`t_recall_end`, 673-675) so the structure becomes:

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
```

  Then **preserve, unchanged and in order**, the existing blocks that currently follow `do_build(prompt)`: round-1 scoring (`sc = scoremod.score(...)`, `conv, feat = split_failed(...)`, `rounds = [_round_rec(1, ...)]`, 677-679); the build!=ok error-exit (685-688); `rate_limited` and the round-1 arch snapshot (690-696); the feedback `while` loop (709-750). **Delete the now-duplicate inner `t_build_start = time.time()` at line 712** (Phase 2 above already opened the build bracket), so `t_build_start` brackets round-1 **and** the loop. Keep `t_build_end = time.time()` after the loop (750). The recall-fired check (760-764) is unchanged and still uses `sid` (the recall call's session, which the build resumed).

- [ ] **Step 3: Assemble costs + fix the in-file `token_audit` consumer.** Replace `build_cost = round(sum(r["cost"] for r in rounds), 4)` (803) with:

```python
    recall_cost, build_cost = split_costs(recall_res, rounds)
    op_cost_total = round(recall_cost + build_cost, 4)   # full billed op cost (recall + all build rounds)
    audit = ({"tokens": dict(EMPTY_TOKENS), "recomputed_cost": 0.0, "cost_ratio": None} if args.stub
             else token_audit(args.cfg, sid, MODELS[args.model], op_cost_total))
```

  (The existing `audit = ... token_audit(..., build_cost)` at 804-805 is REPLACED by the line above — `token_audit` recomputes cost over the whole `sid` session, which now includes the recall turn, so it must compare against `recall_cost + build_cost`, not the recall-free `build_cost`, or `cost_ratio` is systematically inflated for warm.) Then add to the `out` dict next to `"build_cost": build_cost,` (837): `"recall_cost": recall_cost,`, and in the axis block next to `"axis_c2_cost_usd"` (848): `"axis_c2_recall_cost": recall_cost,`.

- [ ] **Step 4: Verify brackets + stub self-check.** `grep -n "t_build_start\|t_recall_start\|token_audit" harness.py` shows each `t_*_start` defined once. Then run the stub consumer self-check: `cd dev/eval/cumulative && python3 validate.py` → it runs the `--stub` matrix (build→score→loop→schema) and must stay green (recall_cost present as 0.0 on stub).
- [ ] **Step 5: REAL warm verification (no-bypass check).** Announce ~$0.20–0.40. Run one real warm cell: `cd dev/eval/cumulative && python3 matrix.py --models opus --trials 1 --regimes real.full --max-rounds 2 --workers 4`. When the first warm build result JSON is written (under the matrix's run dir), read it and assert: `recall_cost > 0.02` AND `recall_s` is a few-seconds-to-minutes recall-only figure (not the whole round-1) AND `build_cost` excludes it. If `recall_cost == 0`, Phase 1 silently failed — fix before proceeding. (Stub alone cannot catch this — stub forces `warm=False`.)
- [ ] **Step 6: Commit** — `git commit -am "feat(eval): split round-1 into recall-only + resumed build; billed recall_cost; token_audit over full op"`

### Task 4: Schema v5 + consumer sweep (note 66)

**Files:**
- Modify: `dev/eval/cumulative/harness.py:31` (`SCHEMA_VERSION = 5`)
- Modify: `dev/eval/cumulative/aggregate.py` (op-cost sums + the two table helpers)
- Modify: `dev/eval/cumulative/matrix.py:178` (`op_cost` rollup)
- Modify: `dev/eval/cumulative/validate.py:141` (`axis_fields`)

**Interfaces:** Consumes the v5 `out` dict with `recall_cost`/`axis_c2_recall_cost`.

- [ ] **Step 1: Bump** `SCHEMA_VERSION = 4` → `5` (harness.py:31).
- [ ] **Step 2: Sweep aggregate.py op-cost sums.** At EACH site that sums an op's dollars, add `recall_cost`:
  - Lines 243, 269, 363, 521, 574, 685: wherever a row's op dollars are summed as `build_cost (+ learn_cost)`, add `+ (x.get("recall_cost", 0) or 0)`.
  - **`per_regime_cost_table` (342-344):** after `ln, bd = mean([r["learn_cost"]...]), mean([r["build_cost"]...])` (342), add `rc = mean([r.get("recall_cost", 0) or 0 for r in rows])`; then change the display at 344 from `{ln+bd:.2f}` (the "total$" column) to `{ln+bd+rc:.2f}`. (Without the line-344 change, "total$" silently excludes recall.)
  - **`amortized_economics_table.axes` (the 4-tuple list at 291-293):** add `("recall cost", "USD", "axis_c2_recall_cost", False)`.
  - **`axis_ci_table.metrics` (the 2-tuple list at ~81):** add `("recall_cost", "axis_c2_recall_cost")` — this also puts the string `axis_c2_recall_cost` into the CI-table header that `validate.py` greps for.
- [ ] **Step 3: Sweep matrix.py `op_cost` (178) — ADD, do not replace.** The current return is `(build_cost) + (learn_cost) + (total_cost) + learn_nested` (4 terms; `learn_nested` is the in-session `/learn` cost). Add a fifth term so it reads:

```python
        return ((d.get("build_cost") or 0) + (d.get("learn_cost") or 0)
                + (d.get("total_cost") or 0) + learn_nested + (d.get("recall_cost") or 0))
```

- [ ] **Step 4: Update validate.py axis_fields** (141): add `"axis_c2_recall_cost"` to the list so the stub dry-pass asserts the new field is emitted.
- [ ] **Step 5: Run validate.py to GREEN** — `cd dev/eval/cumulative && python3 validate.py` → all checks pass (the note-66 consumer self-check; do not declare done until green).
- [ ] **Step 6: Commit** — `git commit -am "feat(eval): schema v5 — recall_cost into aggregate/matrix op-cost sums + validate; green"`

---

# Component 2 — Trap Regression Gate

### Task 5: `seed_c3.py` — the missing C3 warm-vault fixture

**Files:**
- Create: `dev/eval/traps/seed_c3.py`
- Test: `dev/eval/traps/test_gate.py` (new — pure list assertion only)

**Interfaces:**
- Produces: `C3_NOTES: list[dict]` (each `{slug, situation, subject, predicate, object}`) and `seed(vault_path)` that runs `engram learn fact` per note into `vault_path` (using the verified CLI contract in Global Constraints) and **raises** on any non-zero exit (fail loud).

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
- [ ] **Step 3: Implement** mirroring `c4_idio.py`'s `seed_vaults()` subprocess pattern (read it first for the exact `subprocess.run(["engram","learn","fact",...])` env/flag usage). `C3_NOTES` = the 5 conventions, each `object` naming the exact code form: `req-with-context`→"use http.NewRequestWithContext, never http.Get"; `nocolor`→"gate color output behind a NO_COLOR env check"; `t-parallel`→"call t.Parallel() in every test and subtest"; `nil-guard-split`→"guard with a nil/len( check before indexing a split result"; `wrapped-error`→"wrap errors with fmt.Errorf(\"...: %w\", err)". `seed(vault)` runs `engram learn fact` per note with `ENGRAM_VAULT_PATH=vault` and `check=True` (or explicit returncode check) so a non-zero exit raises `RuntimeError` (fail loud).
- [ ] **Step 4: Run to verify PASS** — pytest → passed.
- [ ] **Step 5: Real seed check** (real binary) — `rm -rf /tmp/c3-seed-test && python3 -c "import seed_c3; seed_c3.seed('/tmp/c3-seed-test')" && ENGRAM_VAULT_PATH=/tmp/c3-seed-test engram query --phrase "making an HTTP request in Go"` surfaces the req-with-context note; then `rm -rf /tmp/c3-seed-test`.
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
    # 1 contaminated of 5 (20%, not over), remaining 4 all pass
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
- Produces: `normalize(axis: str, rows: list[dict]) -> list[{"pass","contaminated"}]` via a **dict dispatch** (`_ADAPTERS = {"C3": _norm_c3, "C4i": _norm_c4i, "C5": _norm_c5, "C6": _norm_c6}; return _ADAPTERS[axis](rows)`), raising `ValueError` on unknown axis. Each adapter encodes its harness's real keys:
  - C3 (`warm-results.json`): pass = `r["verdict"]=="applied"`; contaminated = `r["verdict"]=="nobuild"`.
  - C4i (`c4-idio-results.json`): **filter to `r["arm"]=="warm-XXp"`**; contaminated = `not r["built"]`; pass = `bool((r.get("score") or {}).get("supersession_correct"))` (guard the nested `score`, which is `None` when not built).
  - C5 (`c5-results.json`, warm arm): contaminated = `not r["built"]`; pass = `bool(r.get("honored"))`.
  - C6 (`c6-warm.json`): contaminated = `not (r.get("answer") or "").strip()` (empty answer = degraded build, no `built` field); pass = `bool(r.get("hit"))`.

- [ ] **Step 1: Write failing tests** (note the two C4i cases — nested-dict + arm filter):

```python
def test_normalize_c3_applied_pass_nobuild_contaminated():
    rows = [{"verdict": "applied"}, {"verdict": "trap"}, {"verdict": "nobuild"}]
    out = gv.normalize("C3", rows)
    assert out[0] == {"pass": True, "contaminated": False}
    assert out[1] == {"pass": False, "contaminated": False}
    assert out[2]["contaminated"] is True

def test_normalize_c4i_warmxxp_built_supersession_passes():
    rows = [{"arm": "cold", "built": True, "score": {"supersession_correct": False}},
            {"arm": "warm-XXp", "built": True, "score": {"supersession_correct": True}}]
    out = gv.normalize("C4i", rows)               # cold filtered out
    assert out == [{"pass": True, "contaminated": False}]

def test_normalize_c4i_unbuilt_is_contaminated_no_crash():
    rows = [{"arm": "warm-XXp", "built": False, "score": None}]
    out = gv.normalize("C4i", rows)
    assert out[0]["contaminated"] is True and out[0]["pass"] is False

def test_normalize_c5_unbuilt_contaminated():
    rows = [{"built": True, "honored": True}, {"built": False, "honored": None}]
    out = gv.normalize("C5", rows)
    assert out[0] == {"pass": True, "contaminated": False}
    assert out[1]["contaminated"] is True

def test_normalize_c6_empty_answer_is_contaminated():
    rows = [{"hit": True, "answer": "HIT because..."}, {"hit": False, "answer": ""}]
    out = gv.normalize("C6", rows)
    assert out[0] == {"pass": True, "contaminated": False}
    assert out[1]["contaminated"] is True

def test_normalize_unknown_axis_raises():
    import pytest
    with pytest.raises(ValueError):
        gv.normalize("C9", [])
```

  (This is the one test file that legitimately needs `import pytest` — for `pytest.raises`.)

- [ ] **Step 2: Run to verify FAIL.**
- [ ] **Step 3: Implement** the four `_norm_*` helpers + the `_ADAPTERS` dict dispatch + `normalize` exactly per the Interfaces block above; unknown axis → `raise ValueError(f"unknown axis {axis!r}")`.
- [ ] **Step 4: Run to verify PASS.**
- [ ] **Step 5: Commit** — `git commit -m "feat(eval): per-axis trap-result adapters normalizing to (pass, contaminated)"`

### Task 8: `gate.py` orchestrator

**Files:**
- Create: `dev/eval/traps/gate.py`

**Interfaces:** Consumes `seed_c3`, `gate_verdict` (`normalize`/`axis_verdict`/`gate_verdict`). CLI: `python3 gate.py --tier smoke|full [--workers N]`.

This is orchestration (shells out to the warm harnesses); verified by the real smoke run in Task 9, not a unit test.

- [ ] **Step 1: Implement.** Tier config (reps per axis; C3 runs all 5 traps × reps, C6 runs 2 cases × reps):

```python
SMOKE = {"C3": 1, "C4i": 1, "C5": 1, "C6": 1}
FULL  = {"C3": 5, "C4i": 5, "C5": 5, "C6": 4}
```

  For each axis, create a temp `TRAPS_ROOT` (via `tempfile.mkdtemp`), set it in the env, run the warm harness via `subprocess`, **fail loud** on a non-zero return or a missing output JSON, then read + normalize + score. Skeleton (exact commands per axis):

```python
import json, os, subprocess, sys, tempfile
import seed_c3, gate_verdict as gv
TRAPS = os.path.dirname(os.path.abspath(__file__))

def run_axis(axis, reps, workers):
    root = tempfile.mkdtemp(prefix=f"gate-{axis}-")
    env = {**os.environ, "TRAPS_ROOT": root}
    if axis == "C3":
        vault = os.path.join(root, "vault"); seed_c3.seed(vault)
        cmd = ["python3", "wrun.py", "--vault", vault, "--n", str(reps), "--workers", str(workers)]
        out_file = "warm-results.json"
    elif axis == "C4i":
        cmd = ["python3", "c4_idio.py", "--n", str(reps), "--workers", str(workers)]
        out_file = "c4-idio-results.json"
    elif axis == "C5":
        subprocess.run(["python3", "seed_c5.py"], cwd=TRAPS, env=env, check=True)
        cmd = ["python3", "c5.py", "--n", str(reps), "--workers", str(workers)]
        out_file = "c5-results.json"
    elif axis == "C6":
        cmd = ["python3", "c6_clean.py", "--arm", "warm", "--n", str(reps), "--workers", str(workers)]
        out_file = "c6-warm.json"
    else:
        raise ValueError(axis)
    r = subprocess.run(cmd, cwd=TRAPS, env=env)
    if r.returncode != 0:
        sys.exit(f"GATE ABORT: {axis} harness exited {r.returncode}")
    path = os.path.join(root, out_file)
    if not os.path.exists(path):
        sys.exit(f"GATE ABORT: {axis} produced no {out_file} (no silent pass)")
    rows = json.load(open(path))
    trials = gv.normalize(axis, rows)
    return gv.axis_verdict(trials, bar=len([t for t in trials if not t['contaminated']]))
```

  `main()` parses `--tier`, picks `SMOKE`/`FULL`, runs each axis, assembles `axes={ax: run_axis(...)}`, calls `gv.gate_verdict(axes)`, prints a labeled table (`axis | valid | contaminated | passed | status`) + the overall verdict, writes `gate-verdict.json`, and `sys.exit(0)` only on GREEN (non-zero on RED/INCONCLUSIVE so it works as a pre-merge check). **Confirm the `TRAPS_ROOT` env override is honored by each harness** (the map said it is) during the real run; if any harness ignores it, the per-axis temp isolation must be revisited.
- [ ] **Step 2: Parse check** — `python3 -c "import ast; ast.parse(open('dev/eval/traps/gate.py').read())"` → no error.
- [ ] **Step 3: Commit** — `git commit -m "feat(eval): gate.py — tiered C3/C4i/C5/C6 trap regression gate"`

### Task 9: Real smoke verification (the no-bypass check)

- [ ] **Step 1:** Announce cost (~$3). Run `cd dev/eval/traps && python3 gate.py --tier smoke`.
- [ ] **Step 2:** Confirm it invoked the real harnesses (non-zero per-trial costs printed), emitted the per-axis table, and produced a GREEN/RED/INCONCLUSIVE verdict + `gate-verdict.json`. A first run on the current (unchanged) skills should be GREEN (or INCONCLUSIVE on transient trouble — re-run), never a silent pass with zero trials. If a harness ignored `TRAPS_ROOT` (output not isolated), fix the env wiring.
- [ ] **Step 3:** Verification only (no commit); if a real bug surfaced, fix and re-run.

---

## Self-Review

- **Spec coverage:** $METER session-split (T1-T3) ✓; token_audit consumer fix (T3 Step 3) ✓; real warm verify (T3 Step 5) ✓; schema v5 + complete consumer sweep incl. matrix 5-term + aggregate 344 (T4) ✓; seed_c3 with verified CLI contract (T5) ✓; tiered gate, exact bars, contamination→INCONCLUSIVE (T6) ✓; per-axis adapters incl. C4i nested/arm + C6 empty-answer (T7) ✓; gate.py reuse + fail-loud + subprocess skeleton (T8) ✓; real smoke verify (T9) ✓; build order $METER→gate ✓.
- **Placeholder scan:** none — every code/test step has real content; no bare `...`; verification commands are exact (`validate.py`, `matrix.py --regimes real.full`, `gate.py --tier smoke`).
- **Type consistency:** trial shape `{"pass","contaminated"}` consistent across `normalize`/`axis_verdict`; `axes` dict-of-status consistent across `axis_verdict`/`gate_verdict`; `split_costs`/`recall_only_prompt`/`recall_cost` names consistent across tasks.
- **Gate A findings resolved:** clarity (placeholders/commands/dispatch/subprocess) ✓; docs (pytest carve-out cite, engram-learn contract) ✓; ask-alignment (A1 real warm verify, A2 token_audit) ✓; code-alignment (F1 matrix 5-term ADD, F2 helper names, F3 line-344 display, F4 C4i tests) ✓.
- **Accepted trade-offs (documented, no fix):** smoke n=1 means any single contaminated trial → INCONCLUSIVE for that axis (re-run); `cost_ratio>1.0` for warm is expected if token_audit were left on build_cost — fixed in T3 Step 3.
