#!/usr/bin/env python3
"""C7 lever-recheck checkpointing CLI runner (#654 plan, Runner section).

Drives the REAL /recall+/learn skill (via `harness.claude()`, never stubbed — the stub is only the
`engram` binary on PATH, per note 142/86) against the lever_recheck fixtures, across three arms:

  A  vault_with_closed, diagnostic framing  — the RED cell (closed lever should stay unproposed OR
     get reconciled; current skill is expected to commit AMNESIA — no lever-keyed query fired).
  B  vault_open,        diagnostic framing  — degenerate-scorer control: proves the task's data
     legitimately tempts the agent toward the lever, in a world where NO closure info exists at all.
  C  vault_with_closed, consult-memory framing (fixture1 task.txt), SINGLE-CALL —
     regression/positive-class cell: the lever is conceivable at recall time (the scratch data is
     in the cwd from the start + the task says consult memory), so the upfront recall surfaces the
     disproving note and the agent is expected to RECONCILE (pilot 3 validated exactly this form:
     surfaced → RECONCILED). Arm C deliberately does NOT use the two-turn split: with no data in
     turn 1 the consult framing cannot phrase the lever either, so a two-turn C is a mislabeled
     duplicate of arm A. Arm C is OPT-IN (never a default arm): it is only meaningful where the
     fixture has a DISTINCT consult-memory task (task_diagnostic.txt present and different from
     task.txt — today: fixture1). Requesting C on a fixture without one appends an explicit
     `status: skipped` record — never a silent duplicate of arm A.

Two-TURN trial structure — arms A/B (plan amendment 3, from pilot-3 evidence): the phase split —
recall first, analysis data after — is enforced MECHANICALLY with two `claude` calls per trial,
because instructional ordering cannot be trusted (note 145): pilot 3 measured the agent reading
the cwd scratch file before phrasing its recall (lever_query_issued=True in both arms), and
pilot 2 measured the same with inline context.

- Turn 1 (fresh session): RECALL_PREFIX + the arm's task text + TURN1_SUFFIX (a neutral note that
  the team's supporting data is still being gathered and will follow; reply READY). NO scratch
  file in the cwd, NO pointer, NO format directive — the recall runs data-blind and can only
  phrase the diagnostic ask (fixture gate: no lever_terms group is satisfiable from the task text
  alone). RECALL_PREFIX forces the real /recall skill to fire (a bare headless `claude -p` answers
  directly, never touching engram — pilot 1: 100% empty stub log; family precedent
  dev/eval/traps/wrun.py).
- Turn 2 (`--resume <turn-1 sid>`; harness.claude threads resume_sid and still passes the same
  isolated cwd): the scratch data is delivered inline in the message AND written to the trial cwd
  as scratch-notes.md (belt and braces — resume cwd handling verified but not load-bearing),
  together with the format directive: end the reply with one `RECOMMENDATION: <...>` line. The
  lever is conceived HERE, after turn 1's recall. ONE stub log spans both turns (turn 2 does not
  truncate), so the mechanism metric is session-wide; turn-scoped variants are recorded alongside.
- The trial cwd is a bare temp dir (empty at turn 1; scratch-notes.md only at turn 2) — the
  fixture's ground truth (closed_levers.json, vaults) is never inside or reachable from it.
- The format directive pins extract_recommendation to the LAST RECOMMENDATION: line, and every row
  records `rec_line_found` so a whole-text-fallback verdict is always distinguishable (pilot 2's
  fallback let the deterministic guard fire on recall *narration*). Prefix/suffixes are identical
  for ALL arms and content-NEUTRAL per note 138 — they force the invocation, defer the data, and
  fix the reply shape, never hinting at lever-checking or prior attempts.

Expected RED SIGNATURE (arm A, current skill) — a description of the anticipated failure shape,
reported alongside the verdict, never a definitional pass/fail conjunction: turn-1 recall fires on
the diagnostic ask → buried note not returned → the agent meets the data in turn 2 → conceives the
lever mid-analysis → no lever-keyed re-query in either turn → the RECOMMENDATION line re-proposes
the lever → AMNESIA. The GREEN shape is a turn-2 lever-keyed re-query that surfaces the note and
yields RECONCILED (#655's gate — or the shipped criterion-3 reconcile-rule already firing today,
an acceptable honest outcome; `lever_query_issued_turn2` exists to measure exactly that).

Each trial is a fully isolated live run: a throwaway CLAUDE_CONFIG_DIR (`matrix.build_cfg_template`,
warm=True — both /recall and /learn skills; creds injected via `matrix.refresh_creds`), then two
`recheck.live_recall()` calls for arms A/B (turn 1 fresh + truncating the stub log; turn 2
resume_sid-threaded + log-preserving; each returns the FULL harness result dict — text via
`recheck.agent_text`) or ONE call for arm C (scratch present from the start, ARM_C_SUFFIX pointer +
directive, single-turn fields: turn1 == the session, turn2_cost = None) → per-trial validity gate →
scoring (valid trials only). Every completed attempt is appended to the --out JSONL immediately
(flushed) so a killed/resumed batch never loses or re-pays for finished work; --resume skips
(fixture, arm, trial_idx) keys already recorded.

Validity gate (per note 168): INVALID iff the TURN-1 stub log is empty (the recall must have run
against the stub before the data arrived — checked right after turn 1, short-circuiting so an
already-invalid trial never pays for turn 2) OR the summed two-call cost is under $0.02 OR turn 2
ITSELF cost under $0.02 (`degraded_turn2` — a healthy turn 1 must not mask a degraded turn 2; the
verdict text comes from turn 2) OR the turn-2 agent text is empty. turn1_cost/turn2_cost are
recorded separately; `cost_usd` is their sum. The $0.02 floor derives from the harness family's degraded-call heuristic
(`dev/eval/traps/wrun.py`: `(out.get("is_error") or not out) and cost < 0.02` flags a
rate-limited/degraded call); here it is deliberately TIGHTENED to cost-alone — a near-zero-cost
"success" is still a degraded trial that must not be pooled (detect-degraded-builds lesson).
INVALID trials are recorded (excluded from verdicts) and retried, capped at n ×
RETRY_CAP_MULTIPLIER total attempts per (fixture, arm) — the plan's "≤6 total attempts" at the
default n=3. A (fixture, arm) that exhausts the cap with < n valid trials gets an explicit
`kind: cap_exhausted` record; for arm A its classification is "NOT-RED" (the pre-registered
decision procedure: a fixture that cannot reach 3 valid trials fails toward the bar, never
rescues it).

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
# engram — pilot 1: 100% empty_or_missing_stub_log INVALID). Adapted from dev/eval/traps/wrun.py's
# RECALL_PREFIX for a recommend-task. Identical for ALL arms; content-NEUTRAL per note 138 (forces
# the invocation + generic apply-what-surfaces only — never hints at levers or prior attempts).
RECALL_PREFIX = (
    "Before answering, consult your memory: actually INVOKE YOUR /recall skill (do not skip it, do "
    "not hand-run engram yourself in its place). Read what it surfaces and apply anything relevant "
    "as a hard requirement. Then complete this task:\n\n")

# The domain-plausible name the fixture's context.md lands under in the trial cwd (turn 2 only).
SCRATCH_NOTES_NAME = "scratch-notes.md"

# Turn-1 tail (identical for ALL arms, content-neutral): defers the data so the recall runs
# data-blind — the MECHANICAL phase split (amendment 3). No pointer, no format directive here.
TURN1_SUFFIX = (
    "\n\nThe team's supporting data is still being gathered and will be sent to you next — after "
    "your recall pass, reply READY.")

# Turn-2 delivery (resumed session): the scratch data arrives inline AND as a cwd file (belt and
# braces), with the format directive so extract_recommendation gets a real RECOMMENDATION line
# (pilot 2's whole-text fallback let the guard fire on recall narration and left arm B with no
# recommendation at all).
TURN2_TEMPLATE = (
    "Here is the team's scratch log (also written to " + SCRATCH_NOTES_NAME + " in your working "
    "directory):\n\n{context}\n\nNow complete the task. End your reply with exactly one line: "
    "RECOMMENDATION: <the single recommended change>.")

# Arm-C single-call tail (the pilot-3-validated form that produced the clean surfaced→RECONCILED):
# scratch pointer + format directive in the one and only call. Arm C's premise is the lever being
# conceivable AT recall time, so the data is available from the start — under the two-turn split a
# data-blind arm C could not phrase the lever either and would collapse into a mislabeled duplicate
# of arm A.
ARM_C_SUFFIX = (
    f"\n\nThe team's scratch log is in {SCRATCH_NOTES_NAME} in your working directory — read it "
    "before deciding.\n\nEnd your reply with exactly one line: RECOMMENDATION: <the single "
    "recommended change>.")


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


# ----- two-turn prompt construction + trial cwd (amendment 3) -----

def read_fixture_prompt(fixture_dir, task_file):
    """Build the TURN-1 prompt: RECALL_PREFIX (identical for all arms — forces the /recall skill to
    actually fire) + the arm's task ask + TURN1_SUFFIX (data-will-follow note). No context body, no
    scratch pointer, no format directive — turn 1 is data-blind by construction; the data arrives
    in turn 2 (build_turn2_message). Fails LOUD when the task file is missing."""
    task_path = os.path.join(fixture_dir, task_file)
    if not os.path.isfile(task_path):
        raise FileNotFoundError(f"fixture prompt input missing: {task_path!r}")
    with open(task_path) as fh:
        task = fh.read().strip()
    return f"{RECALL_PREFIX}{task}{TURN1_SUFFIX}"


def read_fixture_context(fixture_dir):
    """The fixture's context.md body (the analysis data). Fails LOUD when missing — a fixture
    without its context is a broken cell, not a leaner trial."""
    ctx_path = os.path.join(fixture_dir, "context.md")
    if not os.path.isfile(ctx_path):
        raise FileNotFoundError(f"fixture context missing: {ctx_path!r}")
    with open(ctx_path) as fh:
        return fh.read().strip()


def build_turn2_message(fixture_dir):
    """The TURN-2 resumed-session message: the scratch data inline (belt) + the pointer to its cwd
    copy (braces) + the RECOMMENDATION format directive. Fails LOUD when context.md is missing."""
    return TURN2_TEMPLATE.format(context=read_fixture_context(fixture_dir))


def build_arm_c_prompt(fixture_dir):
    """Arm C's SINGLE-CALL prompt (pilot-3-validated form): RECALL_PREFIX + the consult-memory task
    (always task.txt) + ARM_C_SUFFIX (scratch pointer + format directive). Arm C measures the
    positive class — lever conceivable at recall → note surfaces → RECONCILED — so the data is
    available from the start; no TURN1_SUFFIX deferral, no second turn. Fails LOUD when task.txt is
    missing."""
    task_path = os.path.join(fixture_dir, "task.txt")
    if not os.path.isfile(task_path):
        raise FileNotFoundError(f"fixture prompt input missing: {task_path!r}")
    with open(task_path) as fh:
        task = fh.read().strip()
    return f"{RECALL_PREFIX}{task}{ARM_C_SUFFIX}"


def prepare_trial_cwd(fixture_dir, dst_dir):
    """Write the fixture's context.md into the trial cwd as scratch-notes.md — called BETWEEN turn 1
    and turn 2, so the file does not exist while the recall runs (the mechanical phase split). The
    fixture dir itself — closed_levers.json ground truth, both vaults — is never inside or reachable
    from the cwd (closes the leak where cwd=fixture_dir exposed the ground truth to the agent).
    Fails LOUD when context.md is missing. Returns dst_dir."""
    content = read_fixture_context(fixture_dir)
    os.makedirs(dst_dir, exist_ok=True)
    with open(os.path.join(dst_dir, SCRATCH_NOTES_NAME), "w") as fh:
        fh.write(content + "\n")
    return dst_dir


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


# ----- per-trial validity gate (two-turn) -----

def count_stub_queries(log_path):
    """Number of query rows in the stub log right now (0 when missing/empty). Read after turn 1 it
    is both the treatment-delivery check AND the turn boundary for turn_scoped_mechanism."""
    if not os.path.isfile(log_path):
        return 0
    with open(log_path) as fh:
        return sum(1 for line in fh if line.strip())


def trial_validity(n_turn1_queries, total_cost_usd, turn2_text, turn2_cost=None):
    """INVALID iff turn 1 issued no stub queries (the recall must run BEFORE the data arrives —
    treatment delivery), OR the summed two-call cost is below the degraded-call floor, OR turn 2
    itself cost under the floor (turn2_cost, when turn 2 ran — a healthy turn 1 must not mask a
    degraded/rate-limited turn 2, since the verdict text comes from turn 2), OR turn 2 produced no
    text. Returns (is_valid, reason); reason is None when valid."""
    if n_turn1_queries <= 0:
        return False, "empty_turn1_stub_log"
    if (total_cost_usd or 0) < MIN_VALID_COST_USD:
        return False, "cost_below_floor"
    if turn2_cost is not None and turn2_cost < MIN_VALID_COST_USD:
        return False, "degraded_turn2"
    if not (turn2_text or "").strip():
        return False, "empty_agent_text"
    return True, None


# ----- turn-scoped mechanism (the turn-2 re-query is the criterion-3 signal — a measured output) ---

def turn_scoped_mechanism(log_rows, n_turn1):
    """Split the session-wide stub-log rows at the turn boundary (n_turn1 = rows present when turn 1
    finished) and report the mechanism signals per turn. The session-wide fields keep their meaning
    (recheck.read_stub_log over the whole log); these variants answer WHICH turn did it — a turn-2
    lever-keyed re-query is the shipped criterion-3 rule firing, the plan's named measured output."""
    turn1, turn2 = log_rows[:n_turn1], log_rows[n_turn1:]
    return {
        "lever_query_issued_turn1": any(q.get("lever_keyed") for q in turn1),
        "lever_query_issued_turn2": any(q.get("lever_keyed") for q in turn2),
        "note_surfaced_turn1": any(q.get("returned_buried") for q in turn1),
        "note_surfaced_turn2": any(q.get("returned_buried") for q in turn2),
        "n_queries_turn1": len(turn1),
        "n_queries_turn2": len(turn2),
    }


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

def _call_cost(out):
    return round(float((out.get("total_cost_usd") if isinstance(out, dict) else 0.0) or 0.0), 4)


def _run_arm_c_trial(cell, model, judge, cfg, bin_dir, log_path, trial_cwd,
                     buried_basename, lever_terms):
    """Arm C's SINGLE-CALL flow (pilot-3-validated): scratch-notes.md is in the cwd from the start,
    one fresh claude call with build_arm_c_prompt, single-turn fields (turn1 == the session;
    turn2_cost = None; no turn-scoped turn-2 fields), validity = session stub log non-empty + cost
    floor + non-empty text, verdict from the extracted RECOMMENDATION line via recheck_result."""
    fixture_dir = cell["fixture_dir"]
    prepare_trial_cwd(fixture_dir, trial_cwd)  # data available AT recall time — arm C's premise
    out = recheck.live_recall(fixture_dir, cfg, model, build_arm_c_prompt(fixture_dir), bin_dir,
                              log_path, buried_basename, lever_terms,
                              vault_subdir=cell["vault_subdir"], cwd=trial_cwd)
    cost = _call_cost(out)
    text = recheck.agent_text(out)
    n_queries = count_stub_queries(log_path)

    record = {
        "vault_subdir": cell["vault_subdir"], "task_file": cell["task_file"],
        "model": model, "judge": judge, "session_id": out.get("session_id"),
        "turn1_cost": cost, "turn2_cost": None, "cost_usd": cost,
        "n_queries_turn1": n_queries, "agent_text": text,
        "rec_line_found": recheck.rec_line_found(text),
    }
    is_valid, invalid_reason = trial_validity(n_queries, cost, text)  # turn2_cost N/A (one call)
    record["status"] = "valid" if is_valid else "invalid"
    if not is_valid:
        record["invalid_reason"] = invalid_reason
        return record  # excluded from scoring/verdicts

    scored = recheck.recheck_result(fixture_dir, text, log_path, stub=(judge == "stub"))
    record.update({"cell_verdict": scored["cell_verdict"], "per_lever": scored["per_lever"],
                   "recommendation": scored["recommendation"],
                   "rec_line_found": scored["rec_line_found"],
                   "note_surfaced": scored["note_surfaced"],
                   "lever_query_issued": scored["lever_query_issued"],
                   "n_queries": scored["n_queries"]})
    return record


def run_one_live_trial(cell, model, judge):
    """Execute ONE live attempt for a planned (fixture, arm) cell — TWO claude calls for arms A/B
    (amendment 3), ONE for arm C (_run_arm_c_trial; a data-blind two-turn C would collapse into a
    mislabeled duplicate of arm A).

    Arms A/B — Turn 1: fresh session, data-blind prompt (read_fixture_prompt), empty isolated cwd;
    the recall runs against the stub. The turn-1 stub-log count is the treatment-delivery gate — an
    empty log short-circuits the trial as INVALID before paying for turn 2 — and the turn boundary
    for the turn-scoped mechanism fields. Turn 2: scratch-notes.md written into the SAME cwd
    (prepare_trial_cwd), then recheck.live_recall with resume_sid=<turn-1 sid> and
    truncate_log=False (one log spans both turns), delivering the data inline + the format
    directive (build_turn2_message). Scoring (valid trials only): recheck.recheck_result for arms
    A/C; scorer.score_arm_b on the extracted RECOMMENDATION line for arm B. Verdicts come from the
    final (turn-2 / only-turn) text. Returns a result dict WITHOUT fixture/arm/trial_idx
    (run_fixture_arm stamps those)."""
    import shutil
    import tempfile

    import matrix  # lazy: imports harness at ITS top level; not needed by the pure test suite

    fixture_dir = cell["fixture_dir"]
    buried_basename, lever_terms = stub_config(fixture_dir)

    scratch = tempfile.mkdtemp(prefix="c7-recheck-")
    try:
        cfg = os.path.join(scratch, "cfg")
        matrix.build_cfg_template(cfg, warm=True)
        matrix.refresh_creds(cfg)  # the cfg template carries no creds; every harness cell injects them
        bin_dir = os.path.join(scratch, "bin")
        log_path = os.path.join(scratch, "stub_log.jsonl")
        trial_cwd = os.path.join(scratch, "cwd")
        os.makedirs(trial_cwd)  # EMPTY at turn 1 (A/B) — arm C fills it before its only call

        if cell["arm"] == "C":
            return _run_arm_c_trial(cell, model, judge, cfg, bin_dir, log_path, trial_cwd,
                                    buried_basename, lever_terms)

        turn1_prompt = read_fixture_prompt(fixture_dir, cell["task_file"])
        turn2_message = build_turn2_message(fixture_dir)

        # --- turn 1: data-blind recall ---
        out1 = recheck.live_recall(fixture_dir, cfg, model, turn1_prompt, bin_dir, log_path,
                                   buried_basename, lever_terms, vault_subdir=cell["vault_subdir"],
                                   cwd=trial_cwd)
        turn1_cost = _call_cost(out1)
        session_id = out1.get("session_id")
        n_turn1 = count_stub_queries(log_path)

        record = {
            "vault_subdir": cell["vault_subdir"], "task_file": cell["task_file"],
            "model": model, "judge": judge, "session_id": session_id,
            "turn1_cost": turn1_cost, "turn2_cost": 0.0, "cost_usd": turn1_cost,
            "n_queries_turn1": n_turn1,
        }

        if n_turn1 <= 0:
            # recall never touched the stub — invalid; never pay for turn 2 (short-circuit)
            record.update({"status": "invalid", "invalid_reason": "empty_turn1_stub_log",
                           "agent_text": recheck.agent_text(out1), "rec_line_found": False})
            return record

        # --- turn 2: deliver the data (cwd file + inline) on the resumed session ---
        prepare_trial_cwd(fixture_dir, trial_cwd)
        out2 = recheck.live_recall(fixture_dir, cfg, model, turn2_message, bin_dir, log_path,
                                   buried_basename, lever_terms, vault_subdir=cell["vault_subdir"],
                                   cwd=trial_cwd, resume_sid=session_id, truncate_log=False)
        turn2_cost = _call_cost(out2)
        agent_text = recheck.agent_text(out2)
        total_cost = round(turn1_cost + turn2_cost, 4)
        record.update({"turn2_cost": turn2_cost, "cost_usd": total_cost,
                       "session_id_turn2": out2.get("session_id"), "agent_text": agent_text,
                       "rec_line_found": recheck.rec_line_found(agent_text)})

        is_valid, invalid_reason = trial_validity(n_turn1, total_cost, agent_text,
                                                  turn2_cost=turn2_cost)
        record["status"] = "valid" if is_valid else "invalid"
        if not is_valid:
            record["invalid_reason"] = invalid_reason
            return record  # excluded from scoring/verdicts

        mech = recheck.read_stub_log(log_path)  # session-wide (one log spans both turns)
        record.update(turn_scoped_mechanism(mech["queries"], n_turn1))
        if cell["arm"] == "B":
            # advocacy on the extracted RECOMMENDATION line, not the whole text — narration echoes
            # surfaced distractor notes and could false-positive the check (pilot-2 (b)/(c)).
            rec = recheck.extract_recommendation(agent_text)
            record["recommendation"] = rec
            record.update(scorer.score_arm_b(rec, fixture_dir))
            record.update({"note_surfaced": mech["note_surfaced"],
                           "lever_query_issued": mech["lever_query_issued"],
                           "n_queries": mech["n_queries"]})
        else:
            scored = recheck.recheck_result(fixture_dir, agent_text, log_path, stub=(judge == "stub"))
            record.update({"cell_verdict": scored["cell_verdict"], "per_lever": scored["per_lever"],
                           "recommendation": scored["recommendation"],
                           "rec_line_found": scored["rec_line_found"],
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
