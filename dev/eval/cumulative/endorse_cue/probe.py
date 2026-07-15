#!/usr/bin/env python3
"""Headless ambient-cue-firing probe (described-first-action, no-tools).

Measures whether a `guidance/recall.md` variant fires `/recall` at the moment an agent is
about to ENDORSE, RANK, or RECOMMEND a proposal — the decision moment that leaked in the
real session this eval was built to re-validate (see docs/superpowers/plans/
2026-07-14-recall-endorse-cue.md, Task 1).

Design (note 139 / 251 — recovered pattern, git show c0b00b1c:docs/design/
2026-06-29-recall-moments-revalidation-data/run_revalidation.sh):
  - Described-first-action, no-tools. Each trial is a fresh `claude -p` given a fictional
    endorse/rank/recommend scenario, constrained to output ONLY its single first action
    (`ACTION: <one line>`) and to call no tools. There is no free-form transcript to mine —
    the model must commit to one first move, so there is no vague-intent surface to inflate
    a `fired` score (ask-F2, resolved by construction).
  - Isolation = INLINE, not `@import`. The recall.md variant's text is written directly into
    the trial's project CLAUDE.md (a fresh temp cwd), never `@import`ed — a fresh project's
    first external `@import` triggers Claude Code's one-time approval dialog (note 151);
    headless with no TTY it would silently fail to load the guidance in BOTH arms, producing
    a false null indistinguishable from a real one.
  - Treatment-delivery validity gate. A unique marker line is appended to the inlined
    guidance for each run; the prompt requires the model to echo it back verbatim before
    describing its action. `marker_seen=False` -> the trial is DISCARDED, never scored 0.

Reuse note: only the low-level `claude -p` subprocess idiom and the cold CLAUDE_CONFIG_DIR
shape are reused (cf. dev/eval/cumulative/harness.py::claude, dev/eval/traps/run.py::
build_cold_cfg). The CLAUDE.md inlining/marker scaffolding here is new — this is NOT
dev/eval/traps/wrun.py::build_warm_cfg's RECALL_PREFIX (that prefix explicitly *tells* the
model to invoke /recall, a prompted-invocation design that would defeat an ambient-cue test).

Usage:
  probe.py --recall-md <variant-path> --n <k> --out <jsonl> [--model opus] \
      [--fixtures <dir>] [--workers 4]
"""
import argparse
import concurrent.futures as cf
import json
import os
import re
import shutil
import subprocess
import tempfile
import time
import uuid

MODELS = {"haiku": "claude-haiku-4-5-20251001", "sonnet": "claude-sonnet-4-6", "opus": "claude-opus-4-8"}
KEYCHAIN = 'security find-generic-password -s "Claude Code-credentials" -w'
ROOT = os.environ.get("ENDORSE_CUE_ROOT", "/tmp/endorse-cue-probe")
DEFAULT_FIXTURES_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "fixtures")
TRANSIENT_BACKOFFS = (0, 15, 45, 120)

PROMPT_PREFIX = (
    "You are a capable software engineering agent working in a fictional codebase, with "
    "Read, Edit, Bash, Grep, and (if your project instructions mention it) an engram "
    "/recall skill available. For this question you must NOT call any tool — describe only.\n\n"
)

PROMPT_SUFFIX = (
    "\n\nWhat is the SINGLE first action you would take? You do NOT need to complete the "
    "task and must NOT call any tools. Respond with EXACTLY two lines and nothing else:\n"
    "MARKER: <the exact token beginning with ENGRAM-PROBE-MARKER- found in your project "
    "instructions (CLAUDE.md); write NONE if there is no such token>\n"
    "ACTION: <the single first action or command you would take, one sentence>"
)

# fired = the described first action IS a recall: either names the skill directly, or
# describes checking recall/memory against prior refuted/parked work (plan Task 1 design;
# the wording rule is "name the action" — note 137 — so a bare "I'd check memory" without a
# recall verb does not count, and a bare "recall" mention without a prior-work referent
# does not count either).
RECALL_VERB_RE = re.compile(r"/recall|\bengram query\b|\brecall\b", re.I)
PRIOR_WORK_RE = re.compile(r"\b(prior|refuted|parked|past|previous|memory|note[s]?)\b", re.I)


def score_fired(action_line):
    """True if the described first action is a recall keyed to prior work."""
    if not action_line:
        return False
    if "/recall" in action_line.lower() or "engram query" in action_line.lower():
        return True
    return bool(RECALL_VERB_RE.search(action_line)) and bool(PRIOR_WORK_RE.search(action_line))


def load_fixtures(path):
    fixtures = {}
    for name in sorted(os.listdir(path)):
        if name.endswith(".txt"):
            with open(os.path.join(path, name)) as f:
                fixtures[name[:-4]] = f.read().strip()
    if not fixtures:
        raise SystemExit(f"no .txt fixtures found in {path}")
    return fixtures


def build_cfg(dst):
    """Clean CLAUDE_CONFIG_DIR: onboarding/oauth from the local install (history dropped),
    creds injected, no project history — the cold-cfg shape from dev/eval/traps/run.py::
    build_cold_cfg. This probe's own CLAUDE.md/guidance inlining happens per-trial in the
    project cwd (build_trial_project), not here."""
    shutil.rmtree(dst, ignore_errors=True)
    os.makedirs(dst, exist_ok=True)
    user_cfg = os.path.expanduser("~/.claude/.claude.json")
    base = {}
    if os.path.exists(user_cfg):
        try:
            base = json.load(open(user_cfg))
        except Exception:
            base = {}
    base["projects"] = {}
    json.dump(base, open(os.path.join(dst, ".claude.json"), "w"))
    subprocess.run(["bash", "-c", f'{KEYCHAIN} > {dst}/.credentials.json && chmod 600 {dst}/.credentials.json'],
                    capture_output=True, check=False)


def build_trial_project(recall_md_text, marker):
    """Isolated project cwd whose ONLY project instructions are the guidance variant text,
    INLINED verbatim into CLAUDE.md (never `@import`ed) plus the run's validity marker."""
    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    wd = tempfile.mkdtemp(prefix="endorse-cue-", dir=os.path.join(ROOT, "ws"))
    claude_md = (
        recall_md_text.rstrip()
        + "\n\nThis project's guidance-instance token (echo it back verbatim as your MARKER "
        + f"line when asked): {marker}\n"
    )
    with open(os.path.join(wd, "CLAUDE.md"), "w") as f:
        f.write(claude_md)
    return wd


def loadj_str(txt):
    """Fallback parse: --output-format json normally emits one JSON object, but if stdout
    is line-delimited, take the last object carrying total_cost_usd/type=result."""
    best = {}
    for line in txt.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            obj = json.loads(line)
        except Exception:
            continue
        if isinstance(obj, dict) and ("total_cost_usd" in obj or obj.get("type") == "result"):
            best = obj
    return best


def run_one(cfg, fixture_name, scenario_text, recall_md_text, marker, model, idx):
    wd = build_trial_project(recall_md_text, marker)
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "4000"
    prompt = PROMPT_PREFIX + scenario_text + PROMPT_SUFFIX
    args = ["claude", "-p", prompt, "--output-format", "json",
             "--model", MODELS[model], "--permission-mode", "bypassPermissions"]
    out = {}
    for backoff in TRANSIENT_BACKOFFS:
        if backoff:
            time.sleep(backoff)
        r = subprocess.run(args, cwd=wd, env=env, capture_output=True, text=True)
        try:
            out = json.loads(r.stdout)
        except Exception:
            out = loadj_str(r.stdout)
        cost = out.get("total_cost_usd", 0) or 0
        if (out.get("is_error") or not out) and cost < 0.02:
            continue
        break

    text = out.get("result", "") or ""
    marker_seen = marker in text
    action_match = re.search(r"ACTION:\s*(.+)", text)
    first_action = action_match.group(1).strip() if action_match else None
    fired = score_fired(first_action) if marker_seen else None  # None = discarded, not scored 0

    shutil.rmtree(wd, ignore_errors=True)
    return {
        "fixture": fixture_name,
        "idx": idx,
        "marker_seen": marker_seen,
        "fired": fired,
        "first_action": first_action,
        "raw_result": text,
        "cost": out.get("total_cost_usd", 0) or 0,
        "num_turns": out.get("num_turns"),
        "sid": out.get("session_id"),
    }


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--recall-md", required=True, help="path to the recall.md variant to inline (RED or GREEN)")
    ap.add_argument("--n", type=int, default=5, help="trials per fixture")
    ap.add_argument("--out", required=True, help="output JSONL path")
    ap.add_argument("--model", default="opus", choices=list(MODELS),
                     help="opus is the target model this guidance runs on; measuring on a "
                          "weaker model is not a valid stand-in")
    ap.add_argument("--fixtures", default=DEFAULT_FIXTURES_DIR)
    ap.add_argument("--workers", type=int, default=4)
    a = ap.parse_args()

    with open(a.recall_md) as f:
        recall_md_text = f.read()
    fixtures = load_fixtures(a.fixtures)

    cfg = os.path.join(ROOT, "cfg")
    build_cfg(cfg)

    marker = f"ENGRAM-PROBE-MARKER-{uuid.uuid4().hex[:8]}"
    jobs = [(name, text, i) for name, text in fixtures.items() for i in range(a.n)]
    print(f"running {len(fixtures)} fixtures x n={a.n} = {len(jobs)} {a.model} trials "
          f"(recall_md={a.recall_md}, marker={marker}, workers={a.workers})")

    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, cfg, name, text, recall_md_text, marker, a.model, i): (name, i)
                for name, text, i in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result()
            results.append(r)
            status = "DISCARD(no-marker)" if not r["marker_seen"] else ("FIRED" if r["fired"] else "no-fire")
            print(f"  [{r['fixture']:32} #{r['idx']}] {status:20} action={r['first_action']!r}")

    os.makedirs(os.path.dirname(os.path.abspath(a.out)) or ".", exist_ok=True)
    with open(a.out, "w") as f:
        for r in results:
            f.write(json.dumps(r) + "\n")

    scored = [r for r in results if r["marker_seen"]]
    discarded = [r for r in results if not r["marker_seen"]]
    fired_n = sum(1 for r in scored if r["fired"])
    spent = sum(r["cost"] for r in results)
    print(f"\nscored={len(scored)} discarded(marker_seen=false)={len(discarded)} "
          f"fired={fired_n}/{len(scored)} spend=${spent:.2f}")
    if discarded:
        print(f"WARNING: {len(discarded)} trial(s) discarded for treatment-delivery failure "
              "(guidance did not load) — do not fold these into the fire-rate.")


if __name__ == "__main__":
    main()
