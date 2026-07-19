"""Recency Recall Value-Proof (#646): two-phase per-arm trial runner.

Each trial is two FRESH `claude -p` sessions sharing one per-trial-isolated vault + chunk index.
Phase 1 and phase 2 are commands in DIFFERENT tools (a `notes` tool vs an `orders` tool), but
phase 1 discovers an ORG-WIDE house convention that applies across every tool. Because phase-1's
transcript is about the unrelated `notes` tool, phase-2's recall angles ("prior work on the orders
report", "compute revenue") do NOT cosine-match it — so the RUNLOG convention can only reach phase
2 via the recency channel (the cross-context redesign that closed the shared-tool cosine leak,
where "prior work on hq" surfaced phase-1 `direct`):

  Phase 1 (self-capture): a fresh session implements `add` in the `notes` CLI and, forced by
  check.sh, discovers + narrates the idiosyncratic ORG-WIDE RUNLOG convention (EVERY command in
  EVERY tool appends `RUNLOG v1 :: <CMD> :: rows=<N> :: sig=QX7Z`). Its transcript is ingested as
  chunks (no planted note - note 288).

  Ingest: `engram ingest --transcript <phase-1 .jsonl> --chunks-dir <per-trial dir>` folds
  phase-1's narration into the chunk index as the newest chunks.

  Phase 2 (context-lost measurement): a BRAND-NEW session (no --resume, zero phase-1 context) in
  a strict-whitelist workspace containing ONLY {SPEC.md (orders tool), orders.db.json} — no
  check.sh, no RUNLOG, no phase-1 `notes` material, so the convention cannot be re-derived and
  must be RECALLED. A natural prompt ("implement report per SPEC.md"); the agent decides whether
  to /recall. Revenue is trivial (plain-dollar amt) so the RUNLOG convention is the SOLE
  differentiator. The only systematic variable between arms is the recency channel: ON = default;
  OFF = ENGRAM_RECENT_FILL=-1 (existing flag, resolveRecentFill: negative -> off).

Config isolation (Gate B FIX 3): phase 1 runs with NO engram skills so its own build task cannot
trip /learn; phase 2 gets the /recall skill ONLY. Workspace isolation (Task 1): the claude cwd is
created under a CLEAN temp dir NOT nested under any `.claude` (or VCS repo) — the recall skill's
`engram ingest --auto` walks the cwd's ancestors for `.claude` dirs, and a workspace under
`~/.claude/jobs/...` made it sweep the operator's global `~/.claude` into the per-trial recency
channel (~48 files, displacing phase-1's chunk). See _new_clean_workspace_dir / _assert_clean_cwd.

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

PHASE1_PROMPT = ("Implement `add` in the notes CLI per SPEC.md, then make "
                  "`./check.sh add` pass.")
PHASE2_PROMPT = "Implement `report` in the orders CLI per SPEC.md."

# Phase-2's mandated report invocation (per orders SPEC.md).
REPORT_CMD = ["./orders", "report", "orders.db.json"]

# Each workspace is a strict WHITELIST (Gate B FIX 1) mapping dst-name -> fixtures src-name, so
# nothing outside the map can leak. Phase 1 gets the notes tool + its check.sh; phase 2 gets ONLY
# the orders tool spec + data (no phase-1 `notes` material, no check.sh/RUNLOG, no answer key), so
# the org RUNLOG convention must be recalled, not read. The per-tool spec is copied in as SPEC.md.
WS1_FILES = {"SPEC.md": "notes_SPEC.md", "items.txt": "items.txt", "check.sh": "check.sh"}
WS2_FILES = {"SPEC.md": "orders_SPEC.md", "orders.db.json": "orders.db.json"}

# Surfacing is measured from the agent's ACTUAL in-band recall payloads in the phase-2 transcript
# (note-197 instrument-fidelity fix), NOT a post-hoc out-of-band re-query. See score.py's
# extract_query_payloads / surfaced_*_inband.


def build_recall_only_cfg(dst):
    """Phase-2 config: a clean cold cfg + the /recall skill ONLY (no /learn — Gate B FIX 3). The
    agent can recall naturally; a phase-2 learn-write cannot confound the surfacing measurement."""
    build_cold_cfg(dst)
    src = os.path.join(REPO, "skills", "recall")
    if os.path.isdir(src):
        shutil.copytree(src, os.path.join(dst, "skills", "recall"), dirs_exist_ok=True)


def _assert_clean_cwd(path):
    """Fail loud (note-197 instrument fidelity) if `path` is nested under any `.claude` directory
    or VCS repo. The recall skill runs `engram ingest --auto`, which (sweepspec.go) walks the cwd's
    ancestors collecting every `.claude` dir AND resolves the nearest VCS repo root for markdown —
    so a claude workspace under `~/.claude/jobs/...` or inside a repo would sweep operator-global /
    repo content into the per-trial recency channel, displacing phase-1's chunk."""
    directory = os.path.abspath(path)
    while True:
        if os.path.isdir(os.path.join(directory, ".claude")):
            raise RuntimeError(
                f"workspace {path} is under a .claude dir ({directory}/.claude) — engram ingest "
                "--auto would sweep operator-global memory into the per-trial recency channel")
        for marker in (".git", ".hg", ".jj"):
            if os.path.isdir(os.path.join(directory, marker)):
                raise RuntimeError(
                    f"workspace {path} is inside a VCS repo ({directory}) — engram ingest --auto's "
                    "repo-markdown sweep would ingest the repo's docs into the recency channel")

        parent = os.path.dirname(directory)
        if parent == directory:
            return

        directory = parent


def _new_clean_workspace_dir(arm, idx):
    """A per-trial base for the claude workspaces (cwd), created under the system temp dir so it is
    NOT nested under any `.claude` (Task 1 isolation). Prefers /tmp (stable, never under ~/.claude);
    the guard fails loud if the chosen base is somehow polluted rather than silently leaking."""
    base = "/tmp" if os.path.isdir("/tmp") and os.access("/tmp", os.W_OK) else tempfile.gettempdir()
    work_dir = tempfile.mkdtemp(prefix=f"rvwork-{arm}-{idx}-", dir=base)
    _assert_clean_cwd(work_dir)

    return work_dir


def _build_ws(ws, file_map):
    """Construct a workspace fresh as a strict WHITELIST (Gate B FIX 1): EXACTLY the mapped files,
    each copied from fixtures under its destination name (copy2 preserves check.sh's exec bit).
    Raises if the result set is not exactly the whitelist (defense-in-depth against a leak)."""
    os.makedirs(ws, exist_ok=True)
    for dst_name, src_name in file_map.items():
        shutil.copy2(os.path.join(FIXTURES, src_name), os.path.join(ws, dst_name))

    present = set(os.listdir(ws))
    if present != set(file_map):
        raise RuntimeError(
            f"workspace whitelist violated: expected {sorted(file_map)}, got {sorted(present)}")


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


def run_trial(arm, idx, model, recent_volume=0):
    """Run one full trial: phase 1 (self-capture) -> ingest -> phase 2 (context-lost measurement)
    -> score. Returns a dict with the keys consumed by recency_value_agg.aggregate."""
    os.makedirs(os.path.join(ROOT, "trials"), exist_ok=True)
    trial_dir = tempfile.mkdtemp(prefix=f"{arm}-{idx}-", dir=os.path.join(ROOT, "trials"))

    # The claude workspaces (cwd) live under a CLEAN temp dir, NOT under trial_dir — trial_dir may
    # be under ~/.claude/jobs/..., which would make `engram ingest --auto` sweep operator-global
    # memory into the recency channel (Task 1). Bookkeeping (cfg, chunks, transcripts) stays under
    # trial_dir for collection; only the cwd must be isolation-clean.
    work_dir = _new_clean_workspace_dir(arm, idx)

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
        "recall_fired": None, "correct": None, "runlog_ok": None, "revenue_ok": None,
        "surfaced_any": None, "surfaced_via_recency": None, "phase2_cost": None,
        "phase2_turns": None, "phase2_dur_ms": None, "phase1_cost": 0.0,
        "trial_dir": trial_dir, "work_dir": work_dir,
    }

    # Phase 1: self-capture. WS1 = the notes tool + check.sh only (its SPEC, items.txt, check.sh);
    # orders material + the scorer answer key are withheld.
    ws1 = os.path.join(work_dir, "ws1")
    _build_ws(ws1, WS1_FILES)

    env1 = dict(base_env)
    env1["CLAUDE_CONFIG_DIR"] = phase1_cfg
    env1["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(phase1_cfg, "projects", _slug(ws1))

    out1 = _run_claude(PHASE1_PROMPT, ws1, env1, model)
    result["phase1_cost"] = out1.get("total_cost_usd", 0) or 0

    # phase1_ok = did phase 1 discover + honor the org RUNLOG convention (a valid ADD line)?
    phase1_ok = score.notes_add_ok(ws1)
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

    # Phase 2: context-lost measurement in a brand-new session, strict-whitelist workspace (the
    # orders tool spec + data only — a DIFFERENT tool from phase 1's notes).
    ws2 = os.path.join(work_dir, "ws2")
    _build_ws(ws2, WS2_FILES)

    env2 = dict(base_env)
    env2["CLAUDE_CONFIG_DIR"] = phase2_cfg
    env2["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(phase2_cfg, "projects", _slug(ws2))

    out2 = _run_claude(PHASE2_PROMPT, ws2, env2, model)
    result["phase2_cost"] = out2.get("total_cost_usd", 0) or 0
    result["phase2_turns"] = out2.get("num_turns")
    result["phase2_dur_ms"] = out2.get("duration_ms")

    phase2_transcript = _find_transcript(phase2_cfg, out2.get("session_id"))
    result["recall_fired"] = bool(phase2_transcript) and score.recall_fired(phase2_transcript)

    # Exercise the agent's own `./orders report orders.db.json` per SPEC's mandated invocation.
    # This both checks revenue (stdout) AND drives the report command so it appends its RUNLOG line
    # if the agent built the convention in. revenue_ok is trivially easy (plain-dollar amt);
    # runlog_ok is the real lever (did the agent recall + apply the un-guessable ORG convention?).
    # correct requires BOTH — recording each separately shows convention-recall vs easy-revenue.
    report_result = subprocess.run(REPORT_CMD, cwd=ws2, capture_output=True, text=True)

    with open(os.path.join(FIXTURES, "expected.json")) as f:
        expected_total = json.load(f)["dollar_total"]

    revenue_ok = score.report_revenue_ok(report_result.stdout, expected_total)
    runlog_ok = score.runlog_report_ok(os.path.join(ws2, "RUNLOG"))
    result["revenue_ok"] = revenue_ok
    result["runlog_ok"] = runlog_ok
    result["correct"] = bool(revenue_ok and runlog_ok)

    # Surfacing from the agent's ACTUAL in-band recall payloads (note-197 instrument fix): parse
    # the phase-2 transcript for every engram query payload and match phase-1's chunk by its
    # session id (path-based — robust to the recall skill's --lazy-chunks zeroing content). No
    # out-of-band re-query (it read a post-ingest-auto-polluted index and used canned phrases).
    payloads = score.extract_query_payloads(phase2_transcript)
    with open(os.path.join(trial_dir, "recall_payload.yaml"), "w") as f:
        f.write("\n---\n".join(payloads))

    # surfaced_any = did the agent's recall surface phase-1's chunk at all (any provenance);
    # surfaced_via_recency = did the RECENCY CHANNEL specifically deliver it (note-83 diagnostic).
    result["surfaced_any"] = score.surfaced_any_inband(payloads, phase1_transcript)
    result["surfaced_via_recency"] = score.surfaced_via_recency_inband(payloads, phase1_transcript)

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
                  f"correct={r['correct']} (runlog={r['runlog_ok']} revenue={r['revenue_ok']}) "
                  f"surfaced_any={r['surfaced_any']} via_recency={r['surfaced_via_recency']} "
                  f"$p1={r['phase1_cost']:.2f} $p2={(r['phase2_cost'] or 0):.2f}")

    total_spend = sum((r["phase1_cost"] or 0) + (r["phase2_cost"] or 0) for r in results)
    print(f"\ntotal spend: ${total_spend:.2f}")

    with open(args.out, "w") as f:
        json.dump(results, f, indent=1)


if __name__ == "__main__":
    main()
