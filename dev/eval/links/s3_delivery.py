"""S3 delivery eval — Stage S3 of the link-value exploration.

Claim under test (delivery, not retrieval): when a traversal variant recovers a needed note,
does an agent GIVEN that variant's output actually APPLY the needed note's knowledge to the
query's task, above the baseline-payload control?

Arms:
  A (control): baseline delivered notes (top-10 fact/feedback, rendered compactly)
  B (variant): A plus the variant's recovered additions
    - L6×TAG: baseline notes + tag-nominated candidate pool (needed note + noise; realistic form)
    - L5×T5: baseline with superseder inserted at its T5 position (rendered as top-10)

JUDGE: blind, name-agnostic, opus — HIT only if the plan APPLIES the needed note's principle
to the task domain (tracks reasoning, not vocabulary or note names).

L6×TAG: all 26 recovered cases, reps=2 per case per arm (26×2×2 = 104 runs + 104 judges).
L5×T5: 2 recovered@10 cases, reps=3 per case per arm (12 runs + 12 judges).
         Explicitly UNDERPOWERED (n=2 cases); it is a mechanism smoke, not a verdict.

Usage:
  python3 dev/eval/links/s3_delivery.py [--workers 8] [--dry-run]
"""
from __future__ import annotations

import argparse
import concurrent.futures as cf
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import threading
import time

# ---------------------------------------------------------------------------
# Path setup
# ---------------------------------------------------------------------------
HERE = os.path.dirname(os.path.abspath(__file__))
TRAPS = os.path.join(os.path.dirname(HERE), "traps")
VAULT = os.environ.get("ENGRAM_VAULT_PATH", os.path.expanduser("~/.local/share/engram/vault"))
OUT_PATH = os.path.join(HERE, "s3_results.json")
ROOT = os.environ.get("S3_ROOT", "/tmp/s3-delivery-eval")
CHECKPOINT_PATH = os.path.join(HERE, "s3_trials.jsonl")
_CHECKPOINT_LOCK = threading.Lock()

sys.path.insert(0, HERE)
sys.path.insert(0, TRAPS)

from s3_score import split_tally, tally, verdict
from traversal import (
    _add_md,
    _strip_md,
    supersession_ride_along,
    tag_filter_candidates,
)
from probe import run_engram_query
from run import MODELS, build_cold_cfg

# ---------------------------------------------------------------------------
# Data loading
# ---------------------------------------------------------------------------

def _load(path: str):
    with open(path) as fh:
        return json.load(fh)


def load_all_data() -> dict:
    replays = _load(os.path.join(HERE, "replays.json"))
    return {
        "replay_index": {(r["query_id"], r["n"]): r["ranked_notes"] for r in replays},
        "queries": {q["id"]: q for q in _load(os.path.join(HERE, "queries.json"))},
        "misses_p1": _load(os.path.join(HERE, "misses_p1.json")),
        "bridges_p2": _load(os.path.join(HERE, "bridges_p2.json")),
        "supersession_p3": _load(os.path.join(HERE, "supersession_p3.json")),
        "l5_fabric": _load(os.path.join(HERE, "fabrics", "l5.json")),
        "l6_fabric": _load(os.path.join(HERE, "fabrics", "l6.json")),
    }

# ---------------------------------------------------------------------------
# Case building
# ---------------------------------------------------------------------------

def _population(case_id: str, n: int | None) -> str:
    if case_id.startswith("P1-"):
        return "P1-n3" if n == 3 else "P1-n10"
    if case_id.startswith("P2-"):
        return "P2"
    return "P3"


def build_all_cases(data: dict, dry_run: bool = False) -> list[dict]:
    """Build all 48 miss cases (same structure as probe.py).

    Each case: {case_id, kind, n, baseline, needed_note, query_phrases, population}
    """
    replay_index = data["replay_index"]
    queries = data["queries"]
    cases: list[dict] = []

    # P1
    for miss in data["misses_p1"]:
        qid = miss["query_id"]
        n = miss["n"]
        baseline = replay_index.get((qid, n), [])
        phrases = queries.get(qid, {}).get("phrases", [qid])
        case_id = f"P1-{qid}-n{n}"
        cases.append({
            "case_id": case_id,
            "kind": "P1",
            "n": n,
            "baseline": baseline,
            "needed_note": miss["missed_note"],
            "query_phrases": phrases,
            "population": _population(case_id, n),
        })

    # P2
    for bridge in data["bridges_p2"]:
        top10 = bridge["delivered_top10"]
        total = max(len(top10), 1)
        baseline = [
            {"basename": bn, "score": 1.0 - i / total, "kind": "fact", "rank": i + 1}
            for i, bn in enumerate(top10)
        ]
        cases.append({
            "case_id": f"P2-{bridge['case_id']}",
            "kind": "P2",
            "n": None,
            "baseline": baseline,
            "needed_note": bridge["needed_note"],
            "query_phrases": bridge.get("phrases", [bridge["case_id"]]),
            "population": "P2",
        })

    # P3 (supersession-miss cases only)
    for p3 in data["supersession_p3"]:
        if not p3.get("supersession_miss"):
            continue
        if dry_run:
            baseline: list[dict] = []
        else:
            print(f"  Querying P3-{p3['pair_id']}…", flush=True)
            baseline = run_engram_query(p3["phrases"])
        cases.append({
            "case_id": f"P3-{p3['pair_id']}",
            "kind": "P3",
            "n": None,
            "baseline": baseline,
            "needed_note": p3["new_note"],
            "query_phrases": p3["phrases"],
            "population": "P3",
        })

    return cases


def build_s3_cases(all_cases: list[dict], data: dict) -> dict[str, list[dict]]:
    """Identify recovered cases for L6×TAG and L5×T5; build S3 case dicts.

    Returns: {'L6×TAG': [...], 'L5×T5': [...]}
    Each case dict adds: variant_notes, baseline_notes_for_arm_a
    """
    l6 = data["l6_fabric"]
    l5 = data["l5_fabric"]
    recovered: dict[str, list[dict]] = {"L6×TAG": [], "L5×T5": []}

    for case in all_cases:
        baseline = case["baseline"]
        needed = case["needed_note"]
        target = _strip_md(needed)

        # L6×TAG: tag-nominated pool contains needed note?
        if baseline:
            pool, pool_size = tag_filter_candidates(baseline, l6, top_m=3)
            if any(_strip_md(b) == target for b in pool):
                # Variant B = baseline notes + pool additions
                baseline_set = {_strip_md(r["basename"]) for r in baseline}
                pool_extras = [b for b in pool if _strip_md(b) not in baseline_set]
                recovered["L6×TAG"].append({**case, "pool": pool, "pool_extras": pool_extras})

        # L5×T5: superseder appears at rank ≤ 10 in T5 result?
        if baseline:
            t5_result = supersession_ride_along(baseline, l5)
            t5_rank = next(
                (i + 1 for i, n in enumerate(t5_result) if _strip_md(n["basename"]) == target),
                None,
            )
            if t5_rank is not None and t5_rank <= 10:
                recovered["L5×T5"].append({**case, "t5_result": t5_result, "t5_rank": t5_rank})

    return recovered

# ---------------------------------------------------------------------------
# Note content reading
# ---------------------------------------------------------------------------

def read_note(basename: str) -> dict | None:
    """Read a vault note; return {situation, lesson_text, required_principle} or None."""
    path = os.path.join(VAULT, basename)
    if not os.path.exists(path):
        return None
    content = open(path).read()
    fm_m = re.match(r"---\n(.*?)\n---", content, re.DOTALL)
    if not fm_m:
        return None
    fm: dict[str, str] = {}
    for line in fm_m.group(1).splitlines():
        kv = re.match(r"^(\w+):\s*(.*)", line)
        if kv:
            fm[kv.group(1)] = kv.group(2).strip().strip("'\"")
    situation = fm.get("situation", "")
    # Extract the full lesson paragraph after frontmatter
    body = content[fm_m.end():].strip()
    lesson_m = re.search(
        r"(?:Information learned|Lesson learned):\s*(.*?)(?:\n\nRelated to:|$)",
        body,
        re.DOTALL,
    )
    if lesson_m:
        lesson_text = lesson_m.group(1).strip()
    else:
        # Fallback: combine frontmatter fields
        parts = [fm.get("subject", ""), fm.get("predicate", ""), fm.get("object", "")]
        parts += [fm.get("behavior", ""), fm.get("impact", ""), fm.get("action", "")]
        lesson_text = " — ".join(p for p in parts if p)
    return {
        "situation": situation,
        "lesson_text": lesson_text,
        "required_principle": f"When {situation}:\n{lesson_text}" if situation else lesson_text,
    }


def render_notes_block(basenames: list[str]) -> str:
    """Render a compact 'notes from memory' block for the given list of basenames."""
    lines: list[str] = []
    for bn in basenames:
        info = read_note(bn)
        if info is None:
            continue  # note not in current vault; skip
        slug = re.sub(r"^\d+\.\d{4}-\d{2}-\d{2}\.", "", _strip_md(bn))[:60]
        lines.append(f"[{slug}]")
        if info["situation"]:
            lines.append(f"  Situation: {info['situation'][:200]}")
        lines.append(f"  Lesson: {info['lesson_text'][:400]}")
        lines.append("")
    return "\n".join(lines).strip()


def top10_note_basenames(ranked: list[dict], limit: int = 10) -> list[str]:
    """Return up to `limit` fact/feedback basenames from a ranked list."""
    result: list[str] = []
    for item in ranked:
        if item.get("kind") in ("fact", "feedback"):
            result.append(item["basename"])
            if len(result) >= limit:
                break
    return result

# ---------------------------------------------------------------------------
# Task description
# ---------------------------------------------------------------------------

def task_desc_for(case: dict) -> str:
    phrases = case.get("query_phrases", [])
    if not phrases:
        return case["case_id"]
    # Join first 3 phrases into a readable task description
    selected = phrases[:3]
    return "; ".join(p.rstrip(".") for p in selected)

# ---------------------------------------------------------------------------
# Agent + judge prompts
# ---------------------------------------------------------------------------

_AGENT_PROMPT = (
    "You are about to work on this task: {task_desc}.\n\n"
    "Notes surfaced from memory are below. "
    "Plan your approach in ~10 sentences, applying whatever from memory genuinely bears on it.\n\n"
    "Notes surfaced from memory:\n\n{notes_block}"
)

_JUDGE_PROMPT = (
    "Grade whether a plan applies a specific required principle to the task's domain.\n\n"
    "TASK:\n{task_desc}\n\n"
    "REQUIRED PRINCIPLE (what a correct plan must demonstrate):\n{required_principle}\n\n"
    "PLAN:\n{plan}\n\n"
    "HIT only if the plan applies the required principle's reasoning to THIS task's domain — "
    "showing the described reasoning or its logical consequence, even in different words. "
    "Track the REASONING, not vocabulary or note names: an answer that cites a concept without "
    "applying it to this task is a MISS; an answer that applies the right reasoning in different "
    "words is a HIT. Do NOT be influenced by Luhmann numbers, basenames, or source labels.\n\n"
    "Reply 'HIT' or 'MISS' on line 1, then one sentence."
)

# ---------------------------------------------------------------------------
# Claude subprocess (from qanchor_eval pattern)
# ---------------------------------------------------------------------------

def _claude(prompt: str, cfg: str, model: str = "opus", max_tokens: int = 6000) -> dict:
    """One isolated `claude -p` call with degraded-build retry."""
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = str(max_tokens)
    wd = tempfile.mkdtemp(dir=os.path.join(ROOT, "ws"))
    args = [
        "claude", "-p", prompt, "--output-format", "json",
        "--model", MODELS[model],
        "--permission-mode", "bypassPermissions",
    ]
    out: dict = {}
    for backoff in (0, 15, 45, 120):
        if backoff:
            time.sleep(backoff)
        result = subprocess.run(args, cwd=wd, env=env, capture_output=True, text=True)
        try:
            out = json.loads(result.stdout)
        except Exception:
            out = {}
        if (out.get("is_error") or not out) and (out.get("total_cost_usd", 0) or 0) < 0.01:
            continue
        break
    return out


def _judge(task_desc: str, required_principle: str, plan: str, judge_cfg: str) -> tuple[bool, float]:
    prompt = _JUDGE_PROMPT.format(
        task_desc=task_desc,
        required_principle=required_principle,
        plan=plan or "(no plan produced)",
    )
    out = _claude(prompt, judge_cfg, "opus", max_tokens=1500)
    hit = (out.get("result") or "").strip().upper().startswith("HIT")
    cost = (out.get("total_cost_usd", 0) or 0)
    return hit, cost

# ---------------------------------------------------------------------------
# Trial runner
# ---------------------------------------------------------------------------

def _arm_notes_block(case: dict, arm: str) -> str:
    """Build the notes block for a given arm (A=control, B=variant)."""
    baseline = case.get("baseline", [])
    base_bns = top10_note_basenames(baseline, limit=10)

    if arm == "A":
        return render_notes_block(base_bns)

    # Arm B — survivor-specific extras
    survivor = case.get("survivor", "")
    if survivor == "L6×TAG":
        # Base notes + tag-pool additions (all, not just top 10)
        pool_extras = case.get("pool_extras", [])
        all_bns = base_bns + pool_extras
        return render_notes_block(all_bns)
    elif survivor == "L5×T5":
        # T5 result: take top-10 notes (includes superseder at its inserted position)
        t5_result = case.get("t5_result", baseline)
        t5_bns = top10_note_basenames(t5_result, limit=10)
        return render_notes_block(t5_bns)
    return render_notes_block(base_bns)


def trial_uid(case: dict, arm: str, idx: int) -> str:
    """Unique id for a trial — disambiguates same-case_id rows with different needed notes."""
    return f"{case['survivor']}|{case['case_id']}|{case['needed_note']}|{arm}|{idx}"


def load_checkpoint() -> dict[str, dict]:
    """Load completed trials from the JSONL checkpoint; returns {uid: row}."""
    done: dict[str, dict] = {}
    if not os.path.exists(CHECKPOINT_PATH):
        return done
    with open(CHECKPOINT_PATH) as fh:
        for line in fh:
            line = line.strip()
            if not line:
                continue
            try:
                row = json.loads(line)
            except Exception:
                continue
            if "uid" in row:
                done[row["uid"]] = row
    return done


def append_checkpoint(row: dict) -> None:
    """Thread-safe append of one completed trial to the JSONL checkpoint."""
    with _CHECKPOINT_LOCK:
        with open(CHECKPOINT_PATH, "a") as fh:
            fh.write(json.dumps(row) + "\n")


def run_trial(case: dict, arm: str, cfg: str, judge_cfg: str, idx: int) -> dict:
    """One delivery trial: prompt agent with notes, judge principle-application blind."""
    task_desc = case["task_desc"]
    required_principle = case["required_principle"]
    notes_block = _arm_notes_block(case, arm)

    agent_prompt = _AGENT_PROMPT.format(task_desc=task_desc, notes_block=notes_block)
    out = _claude(agent_prompt, cfg, "opus", max_tokens=6000)
    plan = out.get("result") or ""
    agent_cost = out.get("total_cost_usd", 0) or 0

    hit, judge_cost = _judge(task_desc, required_principle, plan, judge_cfg)
    total_cost = agent_cost + judge_cost

    row = {
        "uid": trial_uid(case, arm, idx),
        "case_id": case["case_id"],
        "needed_note": case["needed_note"],
        "survivor": case["survivor"],
        "population": case["population"],
        "arm": arm,
        "idx": idx,
        "hit": hit,
        "cost": total_cost,
    }
    append_checkpoint(row)
    return row

# ---------------------------------------------------------------------------
# Verdict printing
# ---------------------------------------------------------------------------

def _print_arm_table(label: str, t: dict, v: dict) -> None:
    print(f"\n=== {label} — DELIVERY RATE ===")
    print(f"  {'arm':5} {'hits':>5} {'n':>5} {'rate%':>7} {'±1σ%':>7}")
    for arm in ("A", "B"):
        if arm in t:
            a = t[arm]
            print(f"  {arm:5} {a['hits']:>5} {a['n']:>5} {a['rate']*100:>6.0f}%  {a['sigma']*100:>5.1f}%")
    print(f"\n  Verdict: {v.get('status', '?')}")
    diff = v.get("diff")
    thr = v.get("threshold_2sigma")
    if diff is not None and thr is not None:
        print(f"  B−A = {diff*100:+.0f}pp, 2σ threshold = {thr*100:.0f}pp")


def _print_pop_table(pop_tallies: dict) -> None:
    print("\n=== L6×TAG — PER-POPULATION BREAKDOWN ===")
    print(f"  {'population':12} {'arm':4} {'hits':>5} {'n':>5} {'rate%':>7}")
    for pop in ("P1-n3", "P1-n10", "P2", "P3"):
        pt = pop_tallies.get(pop, {})
        for arm in ("A", "B"):
            if arm in pt:
                a = pt[arm]
                print(f"  {pop:12} {arm:4} {a['hits']:>5} {a['n']:>5} {a['rate']*100:>6.0f}%")

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--workers", type=int, default=8)
    ap.add_argument("--dry-run", action="store_true", help="build cases only, no LLM calls")
    ap.add_argument(
        "--sunk-usd", type=float, default=0.0,
        help="spend from a prior killed run with no checkpoint (honest-tally accounting)",
    )
    args = ap.parse_args()

    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    cfg = os.path.join(ROOT, "cfg")
    build_cold_cfg(cfg)
    judge_cfg = os.path.join(ROOT, "judge-cfg")
    build_cold_cfg(judge_cfg)

    print("=== S3 Delivery Eval — Link-Value Exploration ===\n", flush=True)
    print("Loading data…", flush=True)
    data = load_all_data()

    print("Building cases…", flush=True)
    all_cases = build_all_cases(data, dry_run=args.dry_run)

    print("Identifying recovered cases…", flush=True)
    recovered = build_s3_cases(all_cases, data)

    # Annotate with survivor label, task_desc, required_principle
    for survivor, cases in recovered.items():
        for case in cases:
            case["survivor"] = survivor
            case["task_desc"] = task_desc_for(case)
            note_info = read_note(case["needed_note"])
            case["required_principle"] = (
                note_info["required_principle"] if note_info else case["needed_note"]
            )

    n_l6 = len(recovered["L6×TAG"])
    n_l5 = len(recovered["L5×T5"])
    print(f"\nL6×TAG recovered cases: {n_l6}")
    print(f"L5×T5 recovered@10 cases: {n_l5}")

    if n_l6 == 0 and n_l5 == 0:
        print("No recovered cases found — check data.")
        sys.exit(1)

    # Build job list
    l6_reps = 2
    l5_reps = 3

    jobs: list[dict] = []
    for case in recovered["L6×TAG"]:
        for arm in ("A", "B"):
            for idx in range(l6_reps):
                jobs.append({"case": case, "arm": arm, "idx": idx})

    for case in recovered["L5×T5"]:
        for arm in ("A", "B"):
            for idx in range(l5_reps):
                jobs.append({"case": case, "arm": arm, "idx": idx})

    n_l6_trials = n_l6 * 2 * l6_reps
    n_l5_trials = n_l5 * 2 * l5_reps
    total_trials = n_l6_trials + n_l5_trials
    judge_calls = total_trials  # one judge per trial

    print(
        f"\nL6×TAG: {n_l6} cases × {l6_reps} reps × 2 arms = {n_l6_trials} agent calls"
        f" + {n_l6_trials} judge calls"
    )
    print(
        f"L5×T5: {n_l5} cases × {l5_reps} reps × 2 arms = {n_l5_trials} agent calls"
        f" + {n_l5_trials} judge calls"
    )
    print(
        f"Total: {total_trials} agent calls + {judge_calls} judge calls "
        f"= {total_trials + judge_calls} LLM calls"
    )

    if args.dry_run:
        print("\n[DRY RUN] — stopping before LLM calls.")
        return

    # Resume from checkpoint: skip trials already completed in a prior (partial) run
    done = load_checkpoint()
    results: list[dict] = list(done.values())
    spent = sum(r.get("cost", 0.0) for r in results)
    pending = [j for j in jobs if trial_uid(j["case"], j["arm"], j["idx"]) not in done]
    if done:
        print(
            f"\nResuming: {len(done)} trials from checkpoint "
            f"(${spent:.2f} already spent); {len(pending)} remaining."
        )

    print(f"\nRunning {len(pending)} trials (workers={args.workers})…", flush=True)
    with cf.ThreadPoolExecutor(max_workers=args.workers) as ex:
        futs = {
            ex.submit(run_trial, j["case"], j["arm"], cfg, judge_cfg, j["idx"]): j
            for j in pending
        }
        for fut in cf.as_completed(futs):
            r = fut.result()
            results.append(r)
            spent += r["cost"]
            print(
                f"  [{r['case_id'][:28]:28} {r['survivor'][:8]:8} {r['arm']:1} #{r['idx']}]"
                f" hit={int(r['hit'])}  ${r['cost']:.3f}  (cumulative ${spent:.2f})",
                flush=True,
            )

    # -----------------------------------------------------------------------
    # Tally and verdict — L6×TAG
    # -----------------------------------------------------------------------
    l6_results = [r for r in results if r["survivor"] == "L6×TAG"]
    l6_t = tally(l6_results)
    l6_v = verdict(l6_t, none_ceiling=1.0)  # no none-arm; use very high ceiling
    l6_pop = split_tally(l6_results)

    _print_arm_table("L6×TAG", l6_t, l6_v)
    _print_pop_table(l6_pop)

    # -----------------------------------------------------------------------
    # Tally and verdict — L5×T5 (underpowered smoke)
    # -----------------------------------------------------------------------
    l5_results = [r for r in results if r["survivor"] == "L5×T5"]
    l5_t = tally(l5_results)
    l5_v = verdict(l5_t, none_ceiling=1.0)

    _print_arm_table("L5×T5 [UNDERPOWERED — n=2 cases, mechanism smoke only]", l5_t, l5_v)
    print(
        "  NOTE: L5×T5 is EXPLICITLY UNDERPOWERED (n=2 cases). "
        "Result is a mechanism smoke, not a delivery verdict."
    )

    total_spend = spent + args.sunk_usd
    print(f"\nRun spend: ${spent:.2f}")
    if args.sunk_usd:
        print(f"Sunk spend from prior killed run (no checkpoint): ${args.sunk_usd:.2f}")
    print(f"Total spend (honest tally): ${total_spend:.2f}")

    # -----------------------------------------------------------------------
    # Write s3_results.json
    # -----------------------------------------------------------------------
    import datetime
    out = {
        "run_date": datetime.date.today().isoformat(),
        "survivors_tested": ["L6×TAG", "L5×T5"],
        "model": MODELS["opus"],
        "l6_tag": {
            "n_cases": n_l6,
            "reps_per_case_per_arm": l6_reps,
            "total_trials": n_l6_trials,
            "tally": l6_t,
            "verdict": l6_v,
            "per_population": l6_pop,
            "per_pop_verdict": {
                pop: verdict(pt, none_ceiling=1.0)
                for pop, pt in l6_pop.items()
            },
        },
        "l5_t5": {
            "n_cases": n_l5,
            "reps_per_case_per_arm": l5_reps,
            "total_trials": n_l5_trials,
            "tally": l5_t,
            "verdict": l5_v,
            "underpowered": True,
            "note": "n=2 cases — mechanism smoke only, not a delivery verdict",
        },
        "spend_usd": round(spent, 4),
        "sunk_prior_run_usd": round(args.sunk_usd, 4),
        "total_spend_usd": round(spent + args.sunk_usd, 4),
        "trials": results,
    }
    with open(OUT_PATH, "w") as fh:
        json.dump(out, fh, indent=2)
    print(f"\nWrote {OUT_PATH}", flush=True)


if __name__ == "__main__":
    main()
