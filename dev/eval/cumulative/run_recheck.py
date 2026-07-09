#!/usr/bin/env python3
"""C7 lever-recheck checkpointing CLI runner (#654 plan, Runner section).

Drives the REAL /recall+/learn skill (via `harness.claude()`, never stubbed — the stub is only the
`engram` binary on PATH, per note 142/86) against the lever_recheck fixtures, across three arms:

  A  vault_with_closed, diagnostic framing  — the RED cell (closed lever should stay unproposed OR
     get reconciled; current skill is expected to commit AMNESIA — no lever-keyed query fired).
  B  vault_open,        diagnostic framing  — degenerate-scorer control: proves the task's data
     legitimately tempts the agent toward the lever, in a world where NO closure info exists at all.
  C  vault_with_closed, consult-memory framing (fixture1 task.txt style) — regression/positive-class
     cell: the lever is conceivable at recall time, so the skill's upfront recall surfaces the
     disproving note and the agent is expected to RECONCILE. Arm C is OPT-IN (never a default arm):
     it is only meaningful where the fixture has a DISTINCT consult-memory task (task_diagnostic.txt
     present and different from task.txt — today: fixture1). Requesting C on a fixture without one
     appends an explicit `status: skipped` record — never a silent duplicate of arm A.

The live prompt per trial = RECALL_PREFIX + the fixture's `context.md` (the analysis data that
makes the closed lever emerge mid-analysis — load-bearing, fail-loud if missing) + the arm's task
file. RECALL_PREFIX forces the real /recall skill to fire: a bare headless `claude -p` answers
directly and never touches engram (the fixture2 pilot measured 100% `empty_or_missing_stub_log`
INVALID without it); family precedent is dev/eval/traps/wrun.py's RECALL_PREFIX. The prefix is
identical for ALL arms and content-NEUTRAL per note 138 — it forces the skill invocation and
generic apply-what-surfaces, never hinting at lever-checking or prior attempts.

Each trial is a fully isolated live run: a throwaway CLAUDE_CONFIG_DIR (`matrix.build_cfg_template`,
warm=True — both /recall and /learn skills; creds injected via `matrix.refresh_creds`), then the
plan's prescribed chain — `recheck.live_recall()` (writes the stub_engram shim, truncates the query
log, and calls `harness.claude()` once with the stub on PATH; returns the FULL harness result dict,
whose `total_cost_usd`/`session_id` the validity gate and cost tally need — text via
`recheck.agent_text`) → per-trial validity gate → scoring (valid trials only). Every completed
attempt is appended to the --out JSONL immediately (flushed) so a killed/resumed batch never loses
or re-pays for finished work; --resume skips (fixture, arm, trial_idx) keys already recorded.

Validity gate (per note 168): INVALID iff the stub log is empty/missing OR the call cost is under
$0.02 OR the agent text is empty. The $0.02 floor derives from the harness family's degraded-call
heuristic (`dev/eval/traps/wrun.py`: `(out.get("is_error") or not out) and cost < 0.02` flags a
rate-limited/degraded call); here it is deliberately TIGHTENED to cost-alone — a near-zero-cost
"success" is still a degraded trial that must not be pooled (detect-degraded-builds lesson). INVALID
trials are recorded (excluded from verdicts) and retried, capped at n × RETRY_CAP_MULTIPLIER total
attempts per (fixture, arm) — the plan's "≤6 total attempts" at the default n=3. A (fixture, arm)
that exhausts the cap with < n valid trials gets an explicit `kind: cap_exhausted` record; for arm A
its classification is "NOT-RED" (the pre-registered decision procedure: a fixture that cannot reach
3 valid trials fails toward the bar, never rescues it).

Arm-B scoring semantics (resolved by reading lever_recheck_scorer.py; Gate-B reviewed):

`score_recommendation`/`score_fixture` accept `note_surfaced` as a passthrough diagnostic ONLY — it
is recorded, never folded into the AMNESIA/RECONCILED verdict. Neither the deterministic guard nor
the judge prompt has any "evidence unavailable, don't penalize" branch: the judge always defaults to
AMNESIA unless the recommendation itself acknowledges the prior attempt. Arm B's vault carries NO
note recording the closure — so a legitimate recommendation that arrives at the lever (which is, by
fixture design, the natural answer to the task's data) can never engage a closure it was never told
about, and running the amnesia judge would false-flag every working arm-B trial (forbidden: the
control bar pre-registers false-AMNESIA = 0). Decision: arm B is scored by the advocacy check only —
`lever_recheck_scorer.score_arm_b()` (which wraps the same `_advocates` lexical check the scorer
uses internally), never the judge; every arm-B record carries `per_lever_advocacy` + the aggregate
`advocates` boolean; no cell_verdict is ever produced for arm B. `advocates=False` is not a scored
failure — it is a fixture-tuning signal (the diagnostic framing didn't sufficiently point at the
lever in a closure-free world) for the plan's capped tuning rule to act on.

CLI:
  python3 run_recheck.py --fixtures all --arm A,B --n 3 --model opus --judge stub \
      --out lever_recheck/results.jsonl [--resume]
"""
import argparse
import json
import os

import lever_recheck_scorer as scorer  # cheap, pure import (no harness/matrix deps)
import recheck  # cheap: recheck.py itself imports harness lazily, only inside live_recall

HERE = os.path.dirname(os.path.abspath(__file__))
LEVER_RECHECK_DIR = os.path.join(HERE, "lever_recheck")
DEFAULT_OUT = os.path.join(LEVER_RECHECK_DIR, "results.jsonl")

# Arm -> vault subdir (design table, plan's "Arms (cells) per fixture").
ARM_VAULT = {"A": "vault_with_closed", "B": "vault_open", "C": "vault_with_closed"}
# Arm C is opt-in: only meaningful where a fixture has a DISTINCT consult-memory task (see
# arm_c_skip_reason); running it by default would silently duplicate arm A on diagnostic-only fixtures.
DEFAULT_ARMS = ("A", "B")

# Validity floor, derived from the family's degraded-call heuristic (dev/eval/traps/wrun.py checks
# `(out.get("is_error") or not out) and cost < 0.02`); tightened here to cost-alone per the plan's gate.
MIN_VALID_COST_USD = 0.02

# Pre-registered bar (arm A, n=3) caps retries at 6 TOTAL attempts per fixture-arm — a 2x multiple
# of the target valid count. Generalized to any --n; n=3 reproduces the literal "<=6" ceiling.
RETRY_CAP_MULTIPLIER = 2

# Classification stamped on a cap_exhausted record: the plan's NOT-RED term is arm-A-specific (the
# RED-bar decision procedure: <3 valid at the cap fails toward the bar); other arms get a neutral label.
CAP_CLASSIFICATION = {"A": "NOT-RED"}
CAP_CLASSIFICATION_DEFAULT = "insufficient_valid_trials"

# Forces the real /recall skill to fire (a bare headless `claude -p` answers directly, never touching
# engram — pilot: 100% empty_or_missing_stub_log INVALID). Adapted from dev/eval/traps/wrun.py's
# RECALL_PREFIX for a recommend-task. Identical for ALL arms; content-NEUTRAL per note 138 (forces
# the invocation + generic apply-what-surfaces only — never hints at levers or prior attempts).
RECALL_PREFIX = (
    "Before answering, consult your memory: actually INVOKE YOUR /recall skill (do not skip it, do "
    "not hand-run engram yourself in its place). Read what it surfaces and apply anything relevant "
    "as a hard requirement. Then complete this task:\n\n")


# ----- arm-matrix expansion -----

def arm_task_file(fixture_dir, arm):
    """A/B: task_diagnostic.txt if the fixture has one, else task.txt. C always reads task.txt (the
    consult-memory framing lives there — fixture1-style)."""
    if arm == "C":
        return "task.txt"
    diagnostic = os.path.join(fixture_dir, "task_diagnostic.txt")
    return "task_diagnostic.txt" if os.path.isfile(diagnostic) else "task.txt"


def expand_matrix(fixtures, arms, n):
    """fixtures: ordered [(name, fixture_dir), ...]. arms: ordered arm letters. n: trials per
    (fixture, arm) pair. Returns the ordered pre-retry plan: one dict per (fixture, arm, trial_idx)
    carrying the arm's vault_subdir/task_file. This is the baseline matrix; live retries (invalid
    trials) extend trial_idx past n-1 at run time — see run_fixture_arm."""
    plan = []
    for name, fixture_dir in fixtures:
        for arm in arms:
            for trial_idx in range(n):
                plan.append({
                    "fixture": name, "fixture_dir": fixture_dir, "arm": arm,
                    "vault_subdir": ARM_VAULT[arm], "task_file": arm_task_file(fixture_dir, arm),
                    "trial_idx": trial_idx,
                })
    return plan


def parse_arms(raw_values):
    """raw_values: the argparse --arm list (append action; each item may itself be comma-separated),
    or None. Returns a deduped, order-preserving, validated list of arm letters. Defaults to
    DEFAULT_ARMS (A, B) when raw_values is falsy — arm C must be requested explicitly."""
    if not raw_values:
        return list(DEFAULT_ARMS)
    arms = []
    for value in raw_values:
        for part in value.split(","):
            part = part.strip().upper()
            if not part:
                continue
            if part not in ARM_VAULT:
                raise ValueError(f"unknown --arm {part!r}; must be one of {sorted(ARM_VAULT)}")
            if part not in arms:
                arms.append(part)
    return arms


def discover_fixtures(lever_recheck_root):
    """Fixture dir names directly under lever_recheck_root matching fixture*, sorted."""
    return sorted(
        name for name in os.listdir(lever_recheck_root)
        if name.startswith("fixture") and os.path.isdir(os.path.join(lever_recheck_root, name))
    )


def resolve_fixtures(fixtures_arg, lever_recheck_root):
    """--fixtures 'all' discovers every fixture* dir; otherwise a comma-separated name list (order
    preserved as given). Returns [(name, fixture_dir), ...]."""
    if fixtures_arg.strip().lower() == "all":
        names = discover_fixtures(lever_recheck_root)
    else:
        names = [f.strip() for f in fixtures_arg.split(",") if f.strip()]
    return [(name, os.path.join(lever_recheck_root, name)) for name in names]


# ----- fixture prompt (context.md + task file; both load-bearing, both fail-loud) -----

def read_fixture_prompt(fixture_dir, task_file):
    """Build the live prompt: RECALL_PREFIX (identical for all arms — forces the /recall skill to
    actually fire), then context.md (the analysis data that makes the closed lever emerge
    mid-analysis), then the arm's task ask — the order the materials read naturally in (fixture1:
    the scratch cost log, then "recommend the highest-leverage change"). Mirrors
    contradiction_recheck's read_cell_prompt. Fails LOUD when either file is missing — a fixture
    without its context is a broken cell, not a leaner prompt."""
    ctx_path = os.path.join(fixture_dir, "context.md")
    task_path = os.path.join(fixture_dir, task_file)
    for path in (ctx_path, task_path):
        if not os.path.isfile(path):
            raise FileNotFoundError(f"fixture prompt input missing: {path!r}")
    with open(ctx_path) as fh:
        context = fh.read().strip()
    with open(task_path) as fh:
        task = fh.read().strip()
    return f"{RECALL_PREFIX}{context}\n\n{task}"


# ----- arm-C gate: only where a DISTINCT consult-memory task exists -----

def arm_c_skip_reason(fixture_dir):
    """Return the reason arm C is NOT runnable on this fixture, or None when it is. Arm C reads
    task.txt (consult-memory framing); it is only a distinct cell when the diagnostic arms read a
    DIFFERENT file — i.e. task_diagnostic.txt exists and differs from task.txt. Otherwise arm C
    would silently duplicate arm A's exact run."""
    diagnostic_path = os.path.join(fixture_dir, "task_diagnostic.txt")
    if not os.path.isfile(diagnostic_path):
        return ("no distinct consult-memory task: task_diagnostic.txt absent, so arms A and C would "
                "both run task.txt — a duplicate, not a control")
    with open(diagnostic_path) as fh:
        diagnostic = fh.read().strip()
    with open(os.path.join(fixture_dir, "task.txt")) as fh:
        task = fh.read().strip()
    if diagnostic == task:
        return "no distinct consult-memory task: task_diagnostic.txt is identical to task.txt"
    return None


# ----- checkpoint / resume (JSONL append, flushed immediately) -----

def append_jsonl(out_path, record):
    """Append one record as a line, flushed + fsynced before returning — so a killed process never
    loses an already-completed trial."""
    out_dir = os.path.dirname(os.path.abspath(out_path))
    if out_dir:
        os.makedirs(out_dir, exist_ok=True)
    with open(out_path, "a") as fh:
        fh.write(json.dumps(record) + "\n")
        fh.flush()
        os.fsync(fh.fileno())


def read_jsonl(path):
    if not os.path.isfile(path):
        return []
    rows = []
    with open(path) as fh:
        for line in fh:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return rows


def load_completed(out_path):
    """(fixture, arm, trial_idx) -> status, for every trial row already checkpointed. Rows carrying
    a `kind` (summary, cap_exhausted) are bookkeeping, not trials — skipped."""
    completed = {}
    for row in read_jsonl(out_path):
        if row.get("kind"):
            continue
        completed[(row["fixture"], row["arm"], row["trial_idx"])] = row.get("status")
    return completed


# ----- per-trial validity gate -----

def trial_validity(stub_log_path, cost_usd, agent_text):
    """INVALID iff the stub log is empty/missing, OR cost is below the degraded-call floor, OR the
    agent produced no text. Returns (is_valid, reason); reason is None when valid."""
    if not os.path.isfile(stub_log_path) or os.path.getsize(stub_log_path) == 0:
        return False, "empty_or_missing_stub_log"
    if (cost_usd or 0) < MIN_VALID_COST_USD:
        return False, "cost_below_floor"
    if not (agent_text or "").strip():
        return False, "empty_agent_text"
    return True, None


# ----- retry-capped driver for one (fixture, arm) pair -----

def run_fixture_arm(fixture, arm, n, retry_cap, out_path, attempt_fn, already_done=None):
    """Drive one (fixture, arm) pair to >= n VALID attempts, capped at retry_cap TOTAL attempts
    (valid + invalid combined). already_done: {trial_idx: status} for rows already checkpointed for
    THIS (fixture, arm) pair (from a --resume scan) — those trial_idx slots are never re-run; new
    attempts continue numbering after the highest one seen.

    attempt_fn(trial_idx) executes ONE new attempt and returns a result dict that MUST include
    "status": "valid"|"invalid". This driver stamps fixture/arm/trial_idx onto the result and
    appends+flushes it to out_path IMMEDIATELY — before deciding whether another attempt is needed —
    so progress is never lost.

    When the cap is exhausted with < n valid trials, an explicit `kind: cap_exhausted` record is
    appended, classified per the pre-registered decision procedure (arm A: "NOT-RED" — a fixture
    that cannot reach n valid trials fails toward the bar, never rescues it). The record is only
    written when this invocation actually ran attempts, so a pure resume of an already-exhausted
    pair never duplicates it.

    Returns the list of newly-produced trial records (excludes already_done + the cap record)."""
    already_done = already_done or {}
    valid_count = sum(1 for status in already_done.values() if status == "valid")
    attempts = len(already_done)
    next_idx = (max(already_done) + 1) if already_done else 0
    new_records = []
    while valid_count < n and attempts < retry_cap:
        record = dict(attempt_fn(next_idx))
        record["fixture"] = fixture
        record["arm"] = arm
        record["trial_idx"] = next_idx
        append_jsonl(out_path, record)
        new_records.append(record)
        attempts += 1
        if record.get("status") == "valid":
            valid_count += 1
        next_idx += 1
    if valid_count < n and attempts >= retry_cap and new_records:
        append_jsonl(out_path, {
            "kind": "cap_exhausted", "fixture": fixture, "arm": arm,
            "attempts": attempts, "valid": valid_count, "target_valid": n,
            "classification": CAP_CLASSIFICATION.get(arm, CAP_CLASSIFICATION_DEFAULT),
        })
    return new_records


# ----- batch driver (pure: attempt_maker supplies the per-cell attempt function) -----

def run_batch(fixtures, arms, n, retry_cap, out_path, attempt_maker, completed=None):
    """Run every (fixture, arm) pair of the matrix through run_fixture_arm. attempt_maker(cell) ->
    attempt_fn(trial_idx); cell carries fixture/fixture_dir/arm/vault_subdir/task_file. completed is
    load_completed()'s map for --resume. Arm-C pairs on fixtures without a distinct consult-memory
    task are skipped with an explicit `status: skipped` record (written once — a resume never
    duplicates it), never run as a silent duplicate of arm A."""
    completed = completed or {}
    seen_pairs = set()
    for cell in expand_matrix(fixtures, arms, n):
        pair = (cell["fixture"], cell["arm"])
        if pair in seen_pairs:
            continue
        seen_pairs.add(pair)
        already = {idx: status for (f, a, idx), status in completed.items() if (f, a) == pair}
        if cell["arm"] == "C":
            reason = arm_c_skip_reason(cell["fixture_dir"])
            if reason:
                if not any(status == "skipped" for status in already.values()):
                    append_jsonl(out_path, {"fixture": cell["fixture"], "arm": "C", "trial_idx": -1,
                                            "status": "skipped", "skip_reason": reason})
                continue
        already = {idx: status for idx, status in already.items() if status != "skipped"}
        run_fixture_arm(cell["fixture"], cell["arm"], n, retry_cap, out_path,
                        attempt_maker(cell), already)


# ----- cost tally -----

def summarize(records):
    """Aggregate cost + valid/invalid attempt counts per (fixture, arm) and overall. Ignores
    bookkeeping rows (`kind`: summary/cap_exhausted) so re-summarizing an out-file is safe; skipped
    arm-C rows are counted separately (they are gate decisions, not attempts)."""
    per_fixture_arm = {}
    total_cost = 0.0
    skipped = 0
    for record in records:
        if record.get("kind"):
            continue
        if record.get("status") == "skipped":
            skipped += 1
            continue
        key = f"{record['fixture']}/{record['arm']}"
        agg = per_fixture_arm.setdefault(key, {"attempts": 0, "valid": 0, "invalid": 0, "cost_usd": 0.0})
        agg["attempts"] += 1
        agg["valid" if record.get("status") == "valid" else "invalid"] += 1
        cost = record.get("cost_usd") or 0.0
        agg["cost_usd"] = round(agg["cost_usd"] + cost, 4)
        total_cost += cost
    return {"kind": "summary", "per_fixture_arm": per_fixture_arm, "skipped": skipped,
            "total_cost_usd": round(total_cost, 4)}


# ----- fail-loud stub config, derived from closed_levers.json -----

def stub_config(fixture_dir):
    """STUB_ENGRAM_BURIED = closed_levers[0].note_basename; STUB_ENGRAM_LEVER_TERMS = closed_levers[0]
    .lever_terms. Fails LOUD (KeyError) when either is missing/empty — no silent default (per the
    fixture's own load_closed_levers/read_stub_log fail-loud convention)."""
    levers = scorer.load_closed_levers(fixture_dir)
    lever = levers[0]
    if "note_basename" not in lever:
        raise KeyError(f"closed_levers.json[0] in {fixture_dir!r} is missing required 'note_basename'")
    lever_terms = lever.get("lever_terms")
    if not lever_terms:
        raise KeyError(
            f"closed_levers.json[0] in {fixture_dir!r} (id={lever.get('id')!r}) is missing required "
            f"'lever_terms' — STUB_ENGRAM_LEVER_TERMS has no safe default; fail loud, no silent fallback")
    return lever["note_basename"], lever_terms


# ----- live execution (I/O + paid LLM; never unit-tested, exercised only by the CLI) -----

def run_one_live_trial(cell, model, judge):
    """Execute ONE live attempt for a planned (fixture, arm) cell. Builds a throwaway scratch dir
    (fresh CLAUDE_CONFIG_DIR via matrix.build_cfg_template + matrix.refresh_creds), then runs the
    plan's prescribed chain — recheck.live_recall (writes the stub_engram shim, truncates the query
    log, calls harness.claude once with the stub on PATH, returns the full result dict) → validity
    gate → scoring (recheck.recheck_result for arms A/C; scorer.score_arm_b's advocacy-only check
    for arm B). Returns a result dict WITHOUT fixture/arm/trial_idx (run_fixture_arm stamps those)."""
    import shutil
    import tempfile

    import matrix  # lazy: imports harness at ITS top level; not needed by the pure test suite

    fixture_dir = cell["fixture_dir"]
    prompt = read_fixture_prompt(fixture_dir, cell["task_file"])
    buried_basename, lever_terms = stub_config(fixture_dir)

    scratch = tempfile.mkdtemp(prefix="c7-recheck-")
    try:
        cfg = os.path.join(scratch, "cfg")
        matrix.build_cfg_template(cfg, warm=True)
        matrix.refresh_creds(cfg)  # the cfg template carries no creds; every harness cell injects them
        bin_dir = os.path.join(scratch, "bin")
        log_path = os.path.join(scratch, "stub_log.jsonl")

        out = recheck.live_recall(fixture_dir, cfg, model, prompt, bin_dir, log_path,
                                  buried_basename, lever_terms, vault_subdir=cell["vault_subdir"])
        agent_text = recheck.agent_text(out)
        cost_usd = round(float(out.get("total_cost_usd") or 0.0), 4)

        is_valid, invalid_reason = trial_validity(log_path, cost_usd, agent_text)
        record = {
            "vault_subdir": cell["vault_subdir"], "task_file": cell["task_file"],
            "model": model, "judge": judge, "cost_usd": cost_usd,
            "session_id": out.get("session_id"), "agent_text": agent_text,
            "status": "valid" if is_valid else "invalid",
        }
        if not is_valid:
            record["invalid_reason"] = invalid_reason
            return record  # excluded from scoring/verdicts

        if cell["arm"] == "B":
            record.update(scorer.score_arm_b(agent_text, fixture_dir))
            mech = recheck.read_stub_log(log_path)
            record.update({"note_surfaced": mech["note_surfaced"],
                           "lever_query_issued": mech["lever_query_issued"],
                           "n_queries": mech["n_queries"]})
        else:
            scored = recheck.recheck_result(fixture_dir, agent_text, log_path, stub=(judge == "stub"))
            record.update({"cell_verdict": scored["cell_verdict"], "per_lever": scored["per_lever"],
                           "recommendation": scored["recommendation"],
                           "note_surfaced": scored["note_surfaced"],
                           "lever_query_issued": scored["lever_query_issued"],
                           "n_queries": scored["n_queries"]})
        return record
    finally:
        shutil.rmtree(scratch, ignore_errors=True)


# ----- CLI -----

def build_argparser():
    ap = argparse.ArgumentParser(description=(
        "C7 lever-recheck checkpointing runner: arm A (with_closed/diagnostic, the RED cell), arm B "
        "(open/diagnostic, degenerate-scorer control), arm C (with_closed/consult-memory, regression "
        "— opt-in, only on fixtures with a distinct consult task)."))
    ap.add_argument("--fixtures", default="all",
                    help="comma-separated fixture dir names under lever_recheck/, or 'all' to discover "
                         "every fixture* dir (default: all)")
    ap.add_argument("--arm", action="append", default=None,
                    help="A|B|C, repeatable (--arm A --arm B) or comma-separated (--arm A,B); "
                         "default: A,B (arm C is opt-in)")
    ap.add_argument("--n", type=int, default=3, help="target VALID trials per (fixture, arm) (default: 3)")
    ap.add_argument("--model", default="opus", help="harness.MODELS key (opus|sonnet|haiku|fable)")
    ap.add_argument("--judge", choices=["stub", "live"], default="stub",
                    help="stub = zero-cost deterministic judge; live = real adversarial LLM judge")
    ap.add_argument("--out", default=DEFAULT_OUT, help="checkpoint JSONL path")
    ap.add_argument("--resume", action="store_true",
                    help="skip (fixture, arm, trial_idx) rows already present in --out")
    return ap


def main(argv=None):
    args = build_argparser().parse_args(argv)
    fixtures = resolve_fixtures(args.fixtures, LEVER_RECHECK_DIR)
    if not fixtures:
        raise SystemExit(f"no fixtures matched --fixtures={args.fixtures!r} under {LEVER_RECHECK_DIR}")
    arms = parse_arms(args.arm)

    if os.path.isfile(args.out) and not args.resume:
        raise SystemExit(
            f"--out {args.out!r} already exists; pass --resume to continue it or remove it first "
            f"(checkpoint semantics require an explicit choice, never a silent overwrite/merge)")

    completed = load_completed(args.out) if args.resume else {}
    retry_cap = args.n * RETRY_CAP_MULTIPLIER

    def attempt_maker(cell):
        def attempt(trial_idx):
            return run_one_live_trial(cell, args.model, args.judge)
        return attempt

    run_batch(fixtures, arms, args.n, retry_cap, args.out, attempt_maker, completed)

    append_jsonl(args.out, summarize(read_jsonl(args.out)))


if __name__ == "__main__":
    main()
