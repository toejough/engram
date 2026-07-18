"""Recency Recall Value-Proof (#646): two-phase per-arm trial runner.

Each trial is two FRESH `claude -p` sessions sharing one per-trial-isolated vault + chunk index:

  Phase 1 (self-capture): a fresh session builds `import` in the orders-cli sandbox and, forced
  by validate_import.py, discovers + narrates the idiosyncratic milli-dollar convention. Its
  transcript is ingested as chunks (no planted note - note 288).

  Ingest: `engram ingest --transcript <phase-1 .jsonl> --chunks-dir <per-trial dir>` folds
  phase-1's narration into the chunk index as the newest chunks.

  Phase 2 (context-lost measurement): a BRAND-NEW session (no --resume, zero phase-1 context) in
  a strict-whitelist workspace containing ONLY {SPEC.md, orders.db.json} (Gate B FIX 1 - no
  phase-1-authored importer source, which would leak the round(dollars*1000) formula to a phase-2
  agent extending the same CLI, making the contrast vacuous in both arms). A natural prompt
  ("implement report per SPEC.md"); the agent decides on its own whether to /recall. The only
  systematic variable between arms is the recency channel: ON = default; OFF =
  ENGRAM_RECENT_FILL=-1 (existing flag, resolveRecentFill: negative -> channel off).

Config isolation (Gate B FIX 3): phase 1 runs with NO engram skills so its own build task (fail
validator -> discover -> fix) cannot trip /learn and write a confounding vault note; its
transcript is captured by the explicit `engram ingest` regardless of skills. Phase 2 gets the
/recall skill ONLY (no /learn) so it can naturally recall without a learn-write confound.

See docs/superpowers/plans/2026-07-18-646-recency-value-proof.md for the full design.

Usage:
  python3 recency_value.py --arm on|off --trials N --model opus [--recent-volume K] --out <path>
"""
import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time

import concurrent.futures as cf

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "recency_value"))
import score
from run import MODELS, build_cold_cfg
from wrun import _slug

REPO = "/Users/joe/repos/personal/engram"
ROOT = os.environ.get("TRAPS_ROOT", "/tmp/recency-value")
FIXTURES = os.path.join(os.path.dirname(os.path.abspath(__file__)), "recency_value", "fixtures")

PHASE1_PROMPT = ("Implement `import` in orders-cli per SPEC.md, then make "
                  "`./validate_import.py orders.db.json` pass.")
PHASE2_PROMPT = "Implement `report` in orders-cli per SPEC.md."

# Out-of-band surfacing probe (run_trial step 6): topically distant from the phase-1 build task
# so a hit is attributable to the recency channel, not plain cosine on task vocabulary.
SURFACING_PHRASES = ["implement the revenue report", "orders-cli report total"]

# The out-of-band probe must render FULL chunk content (the milli-dollar narration lives in a
# chunk's content) so the scorer can match markers. --content-budget=-1 = unlimited full content;
# NOT --lazy-chunks, which zeroes all chunk content (clearChunkContent) and would make surfacing
# undetectable. Note: the `=` form is required — a bare `-1` is parsed as another flag.
PROBE_CONTENT_BUDGET = "--content-budget=-1"

# WS2 is a strict WHITELIST (Gate B FIX 1): EXACTLY these files, constructed fresh (not copied
# from WS1 and subtracted). SPEC.md comes from fixtures; orders.db.json is phase-1's own output.
# No phase-1-authored importer source (leaks round(dollars*1000)), no orders.csv, no validator.
WS2_WHITELIST = ("SPEC.md", "orders.db.json")


def build_recall_only_cfg(dst):
    """Phase-2 config: a clean cold cfg + the /recall skill ONLY (no /learn — Gate B FIX 3). The
    agent can recall naturally; a phase-2 learn-write cannot confound the surfacing measurement."""
    build_cold_cfg(dst)
    src = os.path.join(REPO, "skills", "recall")
    if os.path.isdir(src):
        shutil.copytree(src, os.path.join(dst, "skills", "recall"), dirs_exist_ok=True)


def build_trial_env(arm, trial_dir):
    """Base per-trial engram env: empty vault, per-trial chunks dir (mandatory isolation - LEDGER
    harder-regime-op-cost-unmeasurable, the contamination bug), and the arm's recency-channel
    toggle. ENGRAM_RECENT_FILL is present ("-1", channel off) iff arm=="off", and ABSENT iff
    arm=="on" (default channel ON). Pure / no I/O - callers layer the per-phase CLAUDE_CONFIG_DIR
    (phase-1 cold cfg / phase-2 recall-only cfg) and ENGRAM_TRANSCRIPT_DIR on top.
    """
    env = dict(os.environ)
    env.pop("ENGRAM_RECENT_FILL", None)
    env["ENGRAM_VAULT_PATH"] = os.path.join(trial_dir, "vault")
    env["ENGRAM_CHUNKS_DIR"] = os.path.join(trial_dir, "chunks")
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "32000"

    if arm == "off":
        env["ENGRAM_RECENT_FILL"] = "-1"

    return env


def _run_claude(prompt, cwd, env, model):
    """One claude -p call with the C5-family degraded-build backoff (c5.py:62-68)."""
    args = ["claude", "-p", prompt, "--output-format", "json",
            "--model", MODELS[model], "--permission-mode", "bypassPermissions"]
    out = {}
    for backoff in (0, 15, 45, 120):
        if backoff:
            time.sleep(backoff)

        result = subprocess.run(args, cwd=cwd, env=env, capture_output=True, text=True)

        try:
            out = json.loads(result.stdout)
        except (json.JSONDecodeError, ValueError):
            out = {}

        cost = out.get("total_cost_usd", 0) or 0
        if (out.get("is_error") or not out) and cost < 0.02:
            continue

        break

    return out


def _find_transcript(cfg, session_id):
    """Locate a session's own transcript file under this trial's isolated CLAUDE_CONFIG_DIR
    (wrun.py:65-73's approach - grep the transcript for whether recall fired)."""
    if not session_id:
        return None

    projects_dir = os.path.join(cfg, "projects")
    for root, _, files in os.walk(projects_dir):
        name = f"{session_id}.jsonl"
        if name in files:
            return os.path.join(root, name)

    return None


def _ingest_padding(chunks_dir, env, count):
    """Optional volume padding (§2.5): ingest `count` extra unrelated recent chunks AFTER
    phase-1's transcript so they compete for the recency channel's limited slots, stress-testing
    whether the milli-dollar narration still survives realistic recent-session volume."""
    pad_dir = tempfile.mkdtemp(prefix="pad-", dir=os.path.dirname(chunks_dir.rstrip("/")))
    paths = []
    for i in range(count):
        path = os.path.join(pad_dir, f"pad-{i}.md")
        with open(path, "w") as f:
            f.write(f"# Session note {i}\n\nRefactored an unrelated helper in module {i}; "
                     "no change to money handling or order data.\n")
        paths.append(path)

    cmd = ["engram", "ingest", "--chunks-dir", chunks_dir]
    for path in paths:
        cmd += ["--markdown", path]

    subprocess.run(cmd, env=env, capture_output=True, text=True)


def _build_ws2(ws1, ws2):
    """Construct phase-2's workspace fresh as a strict WHITELIST (Gate B FIX 1): EXACTLY SPEC.md
    (from fixtures) + orders.db.json (phase-1's output copied from WS1). Never copytree-WS1-and-
    subtract: phase-1-authored importer source contains the literal round(dollars*1000) math, and
    a phase-2 agent extending the same CLI would recover the unit by reading it — ZERO recall,
    both arms — making the contrast vacuous. Raises if the result set is not exactly the whitelist
    (defense-in-depth against an accidental leak)."""
    os.makedirs(ws2, exist_ok=True)
    shutil.copy(os.path.join(FIXTURES, "SPEC.md"), os.path.join(ws2, "SPEC.md"))
    shutil.copy(os.path.join(ws1, "orders.db.json"), os.path.join(ws2, "orders.db.json"))

    present = set(os.listdir(ws2))
    if present != set(WS2_WHITELIST):
        raise RuntimeError(
            f"WS2 whitelist violated: expected {sorted(WS2_WHITELIST)}, got {sorted(present)}")


def run_trial(arm, idx, model, recent_volume=0):
    """Run one full trial: phase 1 (self-capture) -> ingest -> phase 2 (context-lost measurement)
    -> score. Returns a dict with the keys consumed by recency_value_agg.aggregate."""
    os.makedirs(os.path.join(ROOT, "trials"), exist_ok=True)
    trial_dir = tempfile.mkdtemp(prefix=f"{arm}-{idx}-", dir=os.path.join(ROOT, "trials"))

    # base_env carries ONLY the engram env (vault, per-trial chunks dir, arm's recency toggle);
    # CLAUDE_CONFIG_DIR is layered per-phase so the two phases get DIFFERENT configs (FIX 3).
    base_env = build_trial_env(arm, trial_dir)
    chunks_dir = base_env["ENGRAM_CHUNKS_DIR"]

    # Phase 1 = NO engram skills (can't fire /learn); phase 2 = /recall skill ONLY (no /learn).
    phase1_cfg = os.path.join(trial_dir, "cfg-phase1")
    build_cold_cfg(phase1_cfg)
    phase2_cfg = os.path.join(trial_dir, "cfg-phase2")
    build_recall_only_cfg(phase2_cfg)

    result = {
        "arm": arm, "idx": idx, "phase1_ok": False, "phase1_learned": None,
        "recall_fired": None, "correct": None, "surfaced_any": None,
        "surfaced_via_recency": None, "phase2_cost": None, "phase2_turns": None,
        "phase2_dur_ms": None, "phase1_cost": 0.0, "trial_dir": trial_dir,
    }

    # Phase 1: self-capture. expected.json is the scorer's answer key — it must NEVER enter an
    # agent workspace (§2.1: "used only by the scorer, never in an agent workspace"), so it is
    # excluded here (WS2 is built fresh from a whitelist, so it can never leak there).
    ws1 = os.path.join(trial_dir, "ws1")
    shutil.copytree(FIXTURES, ws1, ignore=shutil.ignore_patterns("expected.json"))

    env1 = dict(base_env)
    env1["CLAUDE_CONFIG_DIR"] = phase1_cfg
    env1["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(phase1_cfg, "projects", _slug(ws1))

    out1 = _run_claude(PHASE1_PROMPT, ws1, env1, model)
    result["phase1_cost"] = out1.get("total_cost_usd", 0) or 0

    db_path = os.path.join(ws1, "orders.db.json")
    csv_path = os.path.join(ws1, "orders.csv")
    phase1_ok = os.path.exists(db_path) and score.import_ok(db_path, csv_path)
    result["phase1_ok"] = phase1_ok

    phase1_transcript = _find_transcript(phase1_cfg, out1.get("session_id"))
    if phase1_transcript:
        # P1 defense-in-depth (FIX 3): flag if phase 1 wrote a note via `engram learn` despite
        # having no skills — such a note would surface via cosine in BOTH arms and confound.
        result["phase1_learned"] = score.phase1_used_learn(phase1_transcript)

    if not phase1_ok:
        # Excluded and counted (not silently dropped) - phase 1 must have captured the
        # lesson for the phase-2 measurement to be meaningful.
        return result

    # Ingest: fold phase-1's narration into the chunk index as the newest chunks.
    if phase1_transcript:
        ingest_cmd = ["engram", "ingest", "--transcript", phase1_transcript,
                      "--chunks-dir", chunks_dir]
        subprocess.run(ingest_cmd, env=base_env, capture_output=True, text=True)

    if recent_volume > 0:
        _ingest_padding(chunks_dir, base_env, recent_volume)

    # Phase 2: context-lost measurement in a brand-new session, strict-whitelist workspace.
    ws2 = os.path.join(trial_dir, "ws2")
    _build_ws2(ws1, ws2)

    env2 = dict(base_env)
    env2["CLAUDE_CONFIG_DIR"] = phase2_cfg
    env2["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(phase2_cfg, "projects", _slug(ws2))

    out2 = _run_claude(PHASE2_PROMPT, ws2, env2, model)
    result["phase2_cost"] = out2.get("total_cost_usd", 0) or 0
    result["phase2_turns"] = out2.get("num_turns")
    result["phase2_dur_ms"] = out2.get("duration_ms")

    phase2_transcript = _find_transcript(phase2_cfg, out2.get("session_id"))
    result["recall_fired"] = bool(phase2_transcript) and score.recall_fired(phase2_transcript)

    # Independently verify the artifact (the built CLI's own stdout), not the agent's narration.
    report_cmd = ["./orders-cli", "report"]
    report_result = subprocess.run(report_cmd, cwd=ws2, capture_output=True, text=True)

    with open(os.path.join(FIXTURES, "expected.json")) as f:
        expected_total = json.load(f)["dollar_total"]

    result["correct"] = score.report_revenue_ok(report_result.stdout, expected_total)

    # Step 6: out-of-band surfacing capture. Does NOT feed the agent (phase 2's own recall already
    # ran) - it only scores what the arm's ranking would surface, teed to disk. Uses the SAME arm
    # env (base_env carries ENGRAM_RECENT_FILL). --content-budget=-1 (NOT --lazy-chunks) so full
    # chunk content is rendered and the milli-dollar markers are matchable.
    query_cmd = ["engram", "query", PROBE_CONTENT_BUDGET]
    for phrase in SURFACING_PHRASES:
        query_cmd += ["--phrase", phrase]

    query_result = subprocess.run(query_cmd, env=base_env, capture_output=True, text=True)

    with open(os.path.join(trial_dir, "recall_payload.yaml"), "w") as f:
        f.write(query_result.stdout)

    # Dual surfacing metric (FIX 2): surfaced_any = P2 vacuous-contrast gate (any provenance,
    # catches the re-rank leak); surfaced_via_recency = note-83 diagnostic (recency channel only).
    result["surfaced_any"] = score.surfaced_any(query_result.stdout)
    result["surfaced_via_recency"] = score.surfaced_via_recency(query_result.stdout)

    return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--arm", choices=["on", "off"], required=True)
    parser.add_argument("--trials", type=int, default=1)
    parser.add_argument("--model", default="opus")
    parser.add_argument("--recent-volume", type=int, default=0)
    parser.add_argument("--workers", type=int, default=1)
    parser.add_argument("--out", required=True)
    args = parser.parse_args()

    print(f"recency-value: arm={args.arm} trials={args.trials} model={args.model} "
          f"recent_volume={args.recent_volume}")

    results = []
    with cf.ThreadPoolExecutor(max_workers=args.workers) as executor:
        futures = [executor.submit(run_trial, args.arm, i, args.model, args.recent_volume)
                   for i in range(args.trials)]
        for future in cf.as_completed(futures):
            r = future.result()
            results.append(r)
            print(f"  [{r['arm']:3} #{r['idx']}] phase1_ok={r['phase1_ok']} "
                  f"learned={r['phase1_learned']} recall_fired={r['recall_fired']} "
                  f"correct={r['correct']} surfaced_any={r['surfaced_any']} "
                  f"via_recency={r['surfaced_via_recency']} $p1={r['phase1_cost']:.2f} "
                  f"$p2={(r['phase2_cost'] or 0):.2f}")

    total_spend = sum((r["phase1_cost"] or 0) + (r["phase2_cost"] or 0) for r in results)
    print(f"\ntotal spend: ${total_spend:.2f}")

    with open(args.out, "w") as f:
        json.dump(results, f, indent=1)


if __name__ == "__main__":
    main()
