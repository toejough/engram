#!/usr/bin/env python3
"""C7 lever-recheck analysis: pre-registered verdicts + post-hoc gates over the runner's results
JSONLs (#654 plan — Pre-registered bars + the Gate-B analysis advisories).

What it computes, offline (no LLM):
- Per-fixture classification, derived ONLY from arm-A TRIAL ROWS — never from `kind:
  cap_exhausted` bookkeeping (advisory: the cap record can be lost/stale; the rows are the ground
  truth). Decision procedure (pre-registered): a fixture is RED iff it has >= N_TARGET_VALID (3)
  valid arm-A trials AND 0 of them scored RECONCILED; NOT-RED when any valid trial reconciled OR
  when fewer than 3 valid trials exist (a fixture that cannot reach the target fails toward the
  bar, never rescues it). Pre-registered discard: a "valid" row whose turn2_cost < $0.02 (produced
  by a runner version predating the per-turn degraded gate) is EXCLUDED from classification and
  reported in `excluded_degraded`.
- Bar verdict: the C7 RED baseline is ESTABLISHED iff >= RED_BAR_FIXTURES (4) fixtures are RED.
- Summary per fixture/arm: attempts, valid/invalid, AMNESIA/RECONCILED counts, guard-fired count,
  re-query RATES per turn (lever_query_issued_turn1/turn2 — the turn-2 rate is the criterion-3
  signal the plan names as a measured output), arm-B advocacy rate, costs. Mechanism rates are
  reported alongside verdicts, never folded into them.
- Cost tally: trial spend total + per arm, plus an order-of-magnitude ESTIMATE of the analysis's
  own judge spend (per-call estimate, labeled as such — the plan's honest-tally requirement).

Live post-hoc gates (opt-in flags; the judge/paraphrase calls are injectable so the offline test
suite mocks them):
- --revote (judge-variance measurement): sample >= MIN_REVOTE_SAMPLE scored rows spanning fixtures
  and both verdict classes (ALL RECONCILED rows when they are the rare class), re-run the FULL
  3-judge vote REVOTE_ROUNDS (3) times per row, report per-row and overall verdict flip-rates.
  This number goes in the LEDGER row; any future GREEN claim (#655) must exceed it.
- --paraphrase (paraphrase hard gate): for 1 seeded-random AMNESIA row per RED fixture, generate a
  meaning-preserving vocabulary-shifted paraphrase of the recommendation (one sonnet call),
  re-judge it with the full 3-vote, and report stable/flipped. The plan's mechanical checks ride
  along on each row: note_surfaced / lever_query_issued from the original row, plus
  `guard_fired_original` — a guard-fired AMNESIA is flagged specially (negated-advocacy advisory:
  the deterministic guard, not the judge, produced that verdict).

Both gates append their rows (kind=revote / kind=paraphrase) to the FIRST --in JSONL so the gate
evidence lives with the trial rows it judged; the machine-readable analysis goes to --out (JSON)
and a human table to stdout.

CLI:
  python3 analyze_recheck.py --in results_A.jsonl,results_BC.jsonl \
      [--revote] [--paraphrase] [--seed 0] [--out lever_recheck/analysis.json]
"""
import argparse
import json
import os
import random
import subprocess
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import lever_recheck_scorer as scorer  # noqa: E402
import run_recheck as rr  # noqa: E402  (read_jsonl/append_jsonl/MIN_VALID_COST_USD/LEVER_RECHECK_DIR)

N_TARGET_VALID = 3           # pre-registered: >=3 valid arm-A trials per fixture
RED_BAR_FIXTURES = 4         # pre-registered: baseline ESTABLISHED iff >=4 of 5 fixtures RED
MIN_REVOTE_SAMPLE = 5
REVOTE_ROUNDS = 3            # re-run the full 3-judge vote this many times per sampled row
EST_JUDGE_CALL_USD = 0.02    # order-of-magnitude sonnet judge-call estimate (labeled estimate)
PARAPHRASE_MODEL = "claude-sonnet-4-6"
DEFAULT_ANALYSIS_OUT = os.path.join(rr.LEVER_RECHECK_DIR, "analysis.json")


# ----- input loading -----

def load_rows(paths):
    rows = []
    for path in paths:
        rows.extend(rr.read_jsonl(path))
    return rows


def trial_rows(rows):
    """Trial rows only: drop bookkeeping (`kind`: summary/cap_exhausted/revote/paraphrase) and
    arm-C gate `status: skipped` records."""
    return [r for r in rows if not r.get("kind") and r.get("status") != "skipped"]


def _degraded_turn2(row):
    return row.get("turn2_cost") is not None and row["turn2_cost"] < rr.MIN_VALID_COST_USD


def _guard_fired(row):
    return any(pl.get("guard_fired") for pl in row.get("per_lever", []))


# ----- per-fixture classification (from trial rows, never bookkeeping) -----

def classify_fixture(fixture, rows, n_target=N_TARGET_VALID):
    """Classify one fixture from its arm-A TRIAL ROWS per the pre-registered decision procedure.
    Returns fixture, valid_n, amnesia, reconciled, excluded_degraded, attempts, verdict, reason."""
    arm_a = [r for r in rows if r.get("fixture") == fixture and r.get("arm") == "A"]
    valid_raw = [r for r in arm_a if r.get("status") == "valid"]
    excluded = [r for r in valid_raw if _degraded_turn2(r)]
    valid = [r for r in valid_raw if not _degraded_turn2(r)]
    reconciled = sum(1 for r in valid if r.get("cell_verdict") == "RECONCILED")
    amnesia = sum(1 for r in valid if r.get("cell_verdict") == "AMNESIA")
    if len(valid) < n_target:
        verdict = "NOT-RED"
        reason = f"insufficient valid arm-A trials ({len(valid)}/{n_target})"
    elif reconciled == 0:
        verdict = "RED"
        reason = f"0 of {len(valid)} valid arm-A trials RECONCILED"
    else:
        verdict = "NOT-RED"
        reason = f"{reconciled} of {len(valid)} valid arm-A trial(s) RECONCILED"
    return {"fixture": fixture, "attempts": len(arm_a), "valid_n": len(valid),
            "amnesia": amnesia, "reconciled": reconciled,
            "excluded_degraded": len(excluded), "verdict": verdict, "reason": reason}


def bar_verdict(classifications):
    red = sum(1 for c in classifications if c["verdict"] == "RED")
    return {"red_fixtures": red, "fixtures_seen": len(classifications),
            "established": red >= RED_BAR_FIXTURES}


# ----- summary rates -----

def _rate(num, den):
    return {"num": num, "den": den, "rate": round(num / den, 3) if den else None}


def _arm_summary(arm, arm_rows):
    valid = [r for r in arm_rows if r.get("status") == "valid"]
    summary = {
        "attempts": len(arm_rows), "valid": len(valid),
        "invalid": sum(1 for r in arm_rows if r.get("status") == "invalid"),
        "cost_usd": round(sum(r.get("cost_usd") or 0.0 for r in arm_rows), 4),
    }
    if arm in ("A", "C"):
        summary["amnesia"] = sum(1 for r in valid if r.get("cell_verdict") == "AMNESIA")
        summary["reconciled"] = sum(1 for r in valid if r.get("cell_verdict") == "RECONCILED")
        summary["guard_fired"] = sum(1 for r in valid if _guard_fired(r))
    if arm == "B":
        summary["advocacy"] = _rate(sum(1 for r in valid if r.get("advocates")), len(valid))
    # re-query rates (mechanism, reported as rates per the ask-alignment requirement; the turn-2
    # rate is the criterion-3 signal). Arm C is single-call: only turn1 exists.
    summary["requery_turn1"] = _rate(
        sum(1 for r in valid if r.get("lever_query_issued_turn1")), len(valid))
    summary["requery_turn2"] = _rate(
        sum(1 for r in valid if r.get("lever_query_issued_turn2")), len(valid))
    summary["note_surfaced"] = _rate(
        sum(1 for r in valid if r.get("note_surfaced")), len(valid))
    return summary


def analyze(rows, n_target=N_TARGET_VALID):
    """The offline analysis: classifications + bar + per-fixture/arm summaries + cost tally."""
    trials = trial_rows(rows)
    fixtures = sorted({r["fixture"] for r in trials})
    classifications = [classify_fixture(fx, trials, n_target=n_target)
                       for fx in fixtures if any(r.get("arm") == "A" for r in trials
                                                 if r["fixture"] == fx)]
    per_fixture = {}
    for fx in fixtures:
        fx_rows = [r for r in trials if r["fixture"] == fx]
        per_fixture[fx] = {arm: _arm_summary(arm, [r for r in fx_rows if r.get("arm") == arm])
                           for arm in sorted({r.get("arm") for r in fx_rows})}
    return {"classifications": classifications, "bar": bar_verdict(classifications),
            "fixtures": per_fixture, "cost": cost_tally(trials)}


# ----- cost tally -----

def cost_tally(trials, n_revote_rows=0, n_paraphrase_rows=0):
    per_arm = {}
    for row in trials:
        arm = row.get("arm")
        per_arm[arm] = round(per_arm.get(arm, 0.0) + (row.get("cost_usd") or 0.0), 4)
    # revote: rows x rounds x JUDGE_RUNS votes; paraphrase: JUDGE_RUNS votes + 1 paraphrase call.
    judge_calls = (n_revote_rows * REVOTE_ROUNDS * scorer.JUDGE_RUNS
                   + n_paraphrase_rows * (scorer.JUDGE_RUNS + 1))
    return {"total_trials_usd": round(sum(per_arm.values()), 4), "per_arm_usd": per_arm,
            "analysis_judge_calls": judge_calls,
            "analysis_spend_estimate_usd": round(judge_calls * EST_JUDGE_CALL_USD, 2)}


# ----- judge-variance protocol (--revote) -----

def sample_for_revote(scored_rows, min_n=MIN_REVOTE_SAMPLE, seed=0):
    """Deterministic (seeded) sample of scored rows spanning fixtures and BOTH verdict classes:
    ALL RECONCILED rows (the rare class) + AMNESIA rows drawn round-robin across fixtures until
    >= min_n. Returns fewer only when fewer scored rows exist."""
    rng = random.Random(seed)
    reconciled = [r for r in scored_rows if r.get("cell_verdict") == "RECONCILED"]
    amnesia_by_fx = {}
    for row in scored_rows:
        if row.get("cell_verdict") == "AMNESIA":
            amnesia_by_fx.setdefault(row["fixture"], []).append(row)
    for rows_ in amnesia_by_fx.values():
        rng.shuffle(rows_)
    fixture_order = sorted(amnesia_by_fx)
    rng.shuffle(fixture_order)

    selected = list(reconciled)
    amnesia_added = 0
    # fill to min_n, and always include some AMNESIA so both classes are present
    while (len(selected) < min_n or amnesia_added == 0) and any(amnesia_by_fx.values()):
        for fx in fixture_order:
            if amnesia_by_fx[fx] and (len(selected) < min_n or amnesia_added == 0):
                selected.append(amnesia_by_fx[fx].pop())
                amnesia_added += 1
    return selected


def default_judge_fn(recommendation, fixture_dir, note_surfaced):
    """ONE full majority-of-3 adversarial vote (live sonnet judges) -> cell verdict."""
    return scorer.score_fixture(recommendation, fixture_dir, note_surfaced=note_surfaced,
                                stub=False)["cell_verdict"]


def run_revote(sampled, judge_fn, rounds=REVOTE_ROUNDS, fixtures_root=None):
    """Re-run the FULL 3-judge vote `rounds` times per sampled row. Returns (kind=revote rows,
    overall flip-rate). A flip = a re-vote verdict differing from the row's original verdict."""
    root = fixtures_root or rr.LEVER_RECHECK_DIR
    out_rows = []
    total_flips = 0
    for row in sampled:
        fixture_dir = os.path.join(root, row["fixture"])
        verdicts = [judge_fn(row.get("recommendation") or "", fixture_dir, row.get("note_surfaced"))
                    for _ in range(rounds)]
        flips = sum(1 for v in verdicts if v != row.get("cell_verdict"))
        total_flips += flips
        out_rows.append({"kind": "revote", "fixture": row["fixture"], "arm": row.get("arm"),
                         "trial_idx": row.get("trial_idx"),
                         "original_verdict": row.get("cell_verdict"),
                         "revotes": verdicts, "flips": flips,
                         "flip_rate": round(flips / rounds, 3)})
    overall = round(total_flips / (rounds * len(sampled)), 3) if sampled else None
    return out_rows, overall


# ----- paraphrase hard gate (--paraphrase) -----

def sample_paraphrase_targets(trials, classifications, seed=0):
    """1 seeded-random valid AMNESIA arm-A row per RED fixture (degraded rows never sampled)."""
    rng = random.Random(seed)
    red_fixtures = sorted(c["fixture"] for c in classifications if c["verdict"] == "RED")
    targets = []
    for fx in red_fixtures:
        candidates = [r for r in trials
                      if r.get("fixture") == fx and r.get("arm") == "A"
                      and r.get("status") == "valid" and not _degraded_turn2(r)
                      and r.get("cell_verdict") == "AMNESIA"]
        if candidates:
            targets.append(rng.choice(candidates))
    return targets


def default_paraphrase_fn(text):
    """One sonnet call: meaning-preserving, vocabulary-shifted paraphrase of a recommendation."""
    prompt = ("Rewrite the following recommendation so it keeps EXACTLY the same meaning and "
              "claims but uses different vocabulary and sentence structure (synonyms, reordered "
              "phrasing; add nothing, drop nothing). Reply with ONLY the rewritten "
              "recommendation.\n\n" + text)
    result = subprocess.run(["claude", "--model", PARAPHRASE_MODEL, "--print", prompt],
                            capture_output=True, text=True, timeout=120)
    if result.returncode != 0:
        raise RuntimeError(f"paraphrase call failed (exit {result.returncode}): {result.stderr[:200]}")
    return result.stdout.strip()


def run_paraphrase_gate(targets, judge_fn, paraphrase_fn, fixtures_root=None):
    """Per target row: paraphrase the recommendation, re-judge with one full 3-vote, report
    stable/flipped. The plan's mechanical checks ride along (note_surfaced / lever_query_issued
    from the original row), plus guard_fired_original — a guard-fired AMNESIA is flagged so the
    reader knows the deterministic guard, not the judge, produced the original verdict."""
    root = fixtures_root or rr.LEVER_RECHECK_DIR
    out_rows = []
    for row in targets:
        paraphrase = paraphrase_fn(row.get("recommendation") or "")
        fixture_dir = os.path.join(root, row["fixture"])
        verdict = judge_fn(paraphrase, fixture_dir, row.get("note_surfaced"))
        out_rows.append({"kind": "paraphrase", "fixture": row["fixture"], "arm": row.get("arm"),
                         "trial_idx": row.get("trial_idx"),
                         "original_verdict": row.get("cell_verdict"),
                         "paraphrase": paraphrase, "paraphrase_verdict": verdict,
                         "stable": verdict == row.get("cell_verdict"),
                         # mechanical checks (plan's paraphrase-gate protocol), surfaced alongside:
                         "note_surfaced": row.get("note_surfaced"),
                         "lever_query_issued": row.get("lever_query_issued"),
                         "guard_fired_original": _guard_fired(row)})
    return out_rows


# ----- rendering -----

def _fmt_rate(rate_dict):
    if not rate_dict or rate_dict.get("den") in (0, None):
        return "-"
    return f"{rate_dict['num']}/{rate_dict['den']}"


def render_table(analysis):
    """Human-readable summary: one line per fixture (arm-A verdict + counts + mechanism rates +
    arm-B advocacy + cost), then the bar line and the honest cost tally."""
    lines = []
    header = (f"{'fixture':10} {'verdict':8} {'valid':>5} {'AMN':>4} {'REC':>4} {'guard':>5} "
              f"{'reQ.t1':>7} {'reQ.t2':>7} {'B.adv':>6} {'excl':>4} {'cost$':>7}  reason")
    lines.append(header)
    lines.append("-" * len(header))
    by_fx = {c["fixture"]: c for c in analysis["classifications"]}
    for fx in sorted(analysis["fixtures"]):
        arms = analysis["fixtures"][fx]
        c = by_fx.get(fx)
        a = arms.get("A", {})
        b = arms.get("B", {})
        cost = round(sum(s.get("cost_usd") or 0.0 for s in arms.values()), 2)
        # verdict-grade columns (valid/AMN/REC/excl) come from the CLASSIFICATION (degraded rows
        # excluded); the mechanism/cost columns are raw-summary diagnostics.
        lines.append(
            f"{fx:10} {(c['verdict'] if c else '-'):8} {(c['valid_n'] if c else 0):>5} "
            f"{(c['amnesia'] if c else 0):>4} {(c['reconciled'] if c else 0):>4} "
            f"{a.get('guard_fired', 0):>5} "
            f"{_fmt_rate(a.get('requery_turn1')):>7} {_fmt_rate(a.get('requery_turn2')):>7} "
            f"{_fmt_rate(b.get('advocacy')):>6} {(c['excluded_degraded'] if c else 0):>4} "
            f"{cost:>7.2f}  {c['reason'] if c else ''}")
    bar = analysis["bar"]
    lines.append("")
    lines.append(f"C7 RED baseline: {'ESTABLISHED' if bar['established'] else 'NOT ESTABLISHED'} "
                 f"({bar['red_fixtures']}/{max(bar['fixtures_seen'], RED_BAR_FIXTURES + 1)} fixtures RED; "
                 f"bar: >={RED_BAR_FIXTURES})")
    cost = analysis["cost"]
    per_arm = " ".join(f"{arm}=${usd:.2f}" for arm, usd in sorted(cost["per_arm_usd"].items()))
    lines.append(f"spent (trials): ${cost['total_trials_usd']:.2f} ({per_arm})"
                 + (f" | analysis judge spend est: ~${cost['analysis_spend_estimate_usd']:.2f} "
                    f"({cost['analysis_judge_calls']} judge calls)"
                    if cost["analysis_judge_calls"] else ""))
    if analysis.get("revote"):
        lines.append(f"judge variance (revote): overall flip-rate "
                     f"{analysis['revote']['overall_flip_rate']} over "
                     f"{len(analysis['revote']['rows'])} rows x {REVOTE_ROUNDS} re-votes")
    if analysis.get("paraphrase"):
        stable = sum(1 for r in analysis["paraphrase"]["rows"] if r["stable"])
        total = len(analysis["paraphrase"]["rows"])
        guard_flagged = sum(1 for r in analysis["paraphrase"]["rows"] if r["guard_fired_original"])
        lines.append(f"paraphrase hard gate: {stable}/{total} stable"
                     + (f" ({guard_flagged} original verdict(s) were GUARD-fired, not judge-ruled)"
                        if guard_flagged else ""))
    return "\n".join(lines)


# ----- CLI -----

def build_argparser():
    ap = argparse.ArgumentParser(description=(
        "C7 lever-recheck analysis: pre-registered per-fixture verdicts + the >=4-of-5 bar from "
        "trial rows, plus opt-in live gates (judge-variance revote, paraphrase hard gate)."))
    ap.add_argument("--in", dest="inputs", required=True,
                    help="comma-separated results JSONL paths (runner output)")
    ap.add_argument("--revote", action="store_true",
                    help="LIVE: judge-variance protocol (3x full 3-vote on a seeded sample)")
    ap.add_argument("--paraphrase", action="store_true",
                    help="LIVE: paraphrase hard gate (1 AMNESIA row per RED fixture)")
    ap.add_argument("--seed", type=int, default=0, help="sampling seed (default 0, deterministic)")
    ap.add_argument("--out", default=DEFAULT_ANALYSIS_OUT,
                    help="machine-readable analysis JSON path")
    return ap


def main(argv=None):
    args = build_argparser().parse_args(argv)
    paths = [p.strip() for p in args.inputs.split(",") if p.strip()]
    for path in paths:
        if not os.path.isfile(path):
            raise SystemExit(f"input JSONL missing: {path!r}")
    rows = load_rows(paths)
    trials = trial_rows(rows)
    analysis = analyze(rows)

    n_revote = n_para = 0
    if args.revote:
        scored = [r for r in trials if r.get("status") == "valid" and r.get("cell_verdict")
                  and not _degraded_turn2(r)]
        sampled = sample_for_revote(scored, seed=args.seed)
        revote_rows, overall = run_revote(sampled, default_judge_fn)
        for row in revote_rows:
            rr.append_jsonl(paths[0], row)  # gate evidence lives with the trial rows it judged
        analysis["revote"] = {"rows": revote_rows, "overall_flip_rate": overall, "seed": args.seed}
        n_revote = len(revote_rows)
    if args.paraphrase:
        targets = sample_paraphrase_targets(trials, analysis["classifications"], seed=args.seed)
        para_rows = run_paraphrase_gate(targets, default_judge_fn, default_paraphrase_fn)
        for row in para_rows:
            rr.append_jsonl(paths[0], row)
        analysis["paraphrase"] = {"rows": para_rows, "seed": args.seed}
        n_para = len(para_rows)

    analysis["cost"] = cost_tally(trials, n_revote_rows=n_revote, n_paraphrase_rows=n_para)
    out_dir = os.path.dirname(os.path.abspath(args.out))
    if out_dir:
        os.makedirs(out_dir, exist_ok=True)
    with open(args.out, "w") as fh:
        json.dump(analysis, fh, indent=2)
    print(render_table(analysis))
    print(f"\nanalysis written: {args.out}")


if __name__ == "__main__":
    main()
