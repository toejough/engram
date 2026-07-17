#!/usr/bin/env python3
"""Payload-prune smoke test — validates the isolation premise (design: 2026-06-30).

Two arms share ONE recall call per app. Arms differ only in what the build session carries:
  Arm A (carried): resume the recall session → full query payload in context for all build rounds.
  Arm B (pruned):  fresh session, only Step-3 synthesis injected → payload dropped.

Validates: does Arm B save build_cost without capability (rounds/success) regression?
Verdict gate (design doc):
  - Net win IFF build_cost(B) < build_cost(A) by a margin ABOVE the noise floor
               AND capability(B) ≈ capability(A): rounds within ±1 AND success not worse.
  - If B saves $ but costs rounds → NOT a win (under-capture).
  - If $ delta below noise floor → UNDERPOWERED, "can't distinguish at n=3."

Reuses from harness.py: claude, build_prompt, recall_only_prompt, split_costs, _round_rec,
converged, convergence_score, split_failed, feedback_prompt, refresh_creds_path, MODELS.
Do NOT import do_build (local closure in run_build) — re-implemented below as do_build_call().
"""
import json
import os
import shutil
import subprocess
import sys
import time
from collections import defaultdict

CUM = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(os.path.dirname(os.path.dirname(CUM)))

sys.path.insert(0, CUM)
import harness
import score as scoremod

# All 3 harness apps — ordered hardest-first by check count.
# feeds=8 checks (hardest), links=7, notes=4 (easiest).
# These are the ONLY harness apps; opus may one-shot all three (oracle-saturation risk, per
# EXPERIMENT-LOG run #8). If so, the rounds-regression signal is absent — note this in verdict.
APP_SPEC = {
    "feeds": "feeds_spec.json",
    "links": "links_spec.json",
    "notes": "notes_spec.json",
}
APPS_HARDEST_FIRST = ["feeds", "links", "notes"]

# Default: real vault (no override → engram uses ~/.local/share/engram/vault)
REAL_VAULT = os.path.expanduser("~/.local/share/engram/vault")

SMOKE_ROOT = "/tmp/smoke_prune"
DATA_DIR = SMOKE_ROOT + "/data"

MAX_ROUNDS = 8
STALL_PATIENCE = 3


# ---------------------------------------------------------------------------
# recall_synthesis_prompt — the key new function (not in harness.py)
# ---------------------------------------------------------------------------

def recall_synthesis_prompt(app: str) -> str:
    """Like recall_only_prompt but instructs the agent to emit the FULL Step-3 synthesis
    block as its final message — the text Arm B will inject into a fresh session.

    The block must include:
      1. The opening count line (Query surfaced N items...)
      2. Per-action bullet for every numbered plan step (confirmed/adjusted/contradicted/silent)
      3. The 'Apply these as requirements:' list with every convention surfaced

    This is distinct from recall_only_prompt (which asks only for a one-line impact summary).
    The full block is what Arm B needs — it must carry the same convention payload the
    recall session context would carry in Arm A.
    """
    return (
        "Consult your memory by INVOKING YOUR /recall SKILL — actually run the skill "
        "(it prints its Step 0 plan, queries the vault, and synthesizes impact). "
        "Do NOT hand-run `engram query` yourself in place of the skill. "
        f"Frame the recall around building a command-line {app} in Go and its "
        "architecture/conventions. Read every note the skill surfaces.\n\n"
        "IMPORTANT — after the /recall skill finishes, emit its COMPLETE Step-3 synthesis "
        "block verbatim as your FINAL message. The block must include ALL of:\n"
        "  1. The opening count line "
        "(e.g. 'Query surfaced N items (K chunks, M notes); crystallized J lessons.')\n"
        "  2. A per-action bullet for EVERY numbered plan step "
        "(confirmed / adjusted / contradicted / silent)\n"
        "  3. The 'Apply these as requirements:' list with EVERY convention the skill "
        "surfaced (drawn from both fact/feedback notes and chunk evidence)\n\n"
        "Emit the full synthesis block verbatim, then STOP: do not write any code, "
        "do not initialize a Go module, do not create files — only complete the recall "
        "and print the synthesis block."
    )


# ---------------------------------------------------------------------------
# Infrastructure helpers
# ---------------------------------------------------------------------------

def setup_cfg(warm: bool) -> str:
    """Create an isolated CLAUDE_CONFIG_DIR with skills wired (warm) or not."""
    kind = "warm" if warm else "cold"
    dst = os.path.join(SMOKE_ROOT, f"cfg_{kind}")
    os.makedirs(dst, exist_ok=True)

    user_cfg = os.path.expanduser("~/.claude/.claude.json")
    base: dict = {}
    if os.path.exists(user_cfg):
        try:
            base = json.load(open(user_cfg))
        except Exception:
            base = {}
    base["projects"] = {}  # drop per-project history; keep auth/onboarding flags
    json.dump(base, open(os.path.join(dst, ".claude.json"), "w"))

    if warm:
        for skill in ("recall", "learn"):
            src = os.path.join(REPO, "skills", skill)
            tgt = os.path.join(dst, "skills", skill)
            if os.path.isdir(src) and not os.path.isdir(tgt):
                shutil.copytree(src, tgt)

    # Inject credentials
    subprocess.run(
        ["bash", "-c",
         f'security find-generic-password -s "Claude Code-credentials" -w '
         f'> {dst}/.credentials.json && chmod 600 {dst}/.credentials.json'],
        capture_output=True,
    )
    return dst


def seed_vault(workdir: str, vault_src: str) -> str:
    """Create a per-cell isolated vault copy (mirrors harness._seed_build_vault)."""
    build_vault = workdir + ".vault"
    shutil.rmtree(build_vault, ignore_errors=True)
    if vault_src != "none" and os.path.isdir(vault_src):
        shutil.copytree(vault_src, build_vault)
    else:
        os.makedirs(os.path.join(build_vault, "Permanent"), exist_ok=True)
    return build_vault


# ---------------------------------------------------------------------------
# Build call wrapper (re-implements harness.run_build's do_build closure)
# ---------------------------------------------------------------------------

def do_build_call(
    cfg: str,
    model_key: str,
    vault: str,
    cwd: str,
    prompt: str,
    resume_sid: str | None = None,
) -> dict:
    """Wraps harness.claude() with rate-limit retry (mirrors harness.run_build:do_build)."""
    res = harness.claude(cfg, model_key, vault, cwd, prompt, resume_sid=resume_sid)
    for backoff in (15, 45, 120):
        is_err = bool(res.get("is_error"))
        cost = (res.get("total_cost_usd") or 0)
        if not (is_err and cost < 0.02):
            break
        print(f"  rate-limit hit; retry after {backoff}s...", file=sys.stderr, flush=True)
        harness.refresh_creds_path(cfg)
        time.sleep(backoff)
        res = harness.claude(cfg, model_key, vault, cwd, prompt, resume_sid=resume_sid)
    return res


# ---------------------------------------------------------------------------
# Build loop (shared by both arms)
# ---------------------------------------------------------------------------

def run_build_loop(
    cfg: str,
    model_key: str,
    vault: str,
    cwd: str,
    spec_path: str,
    first_prompt: str,
    arm_label: str,
    resume_sid: str | None = None,
    max_rounds: int = MAX_ROUNDS,
) -> tuple[list, bool, int | None, str | None]:
    """Full build → score → feedback loop. Returns (rounds, completed, rounds_to_converge, sid)."""
    os.makedirs(cwd, exist_ok=True)
    res = do_build_call(cfg, model_key, vault, cwd, first_prompt, resume_sid=resume_sid)
    sid = resume_sid or res.get("session_id")

    is_err = bool(res.get("is_error"))
    cost = (res.get("total_cost_usd") or 0)
    if is_err and cost < 0.02:
        raise RuntimeError(f"{arm_label}: build call failed at round 1 (likely rate-limit)")

    spec = scoremod.load_spec(spec_path)  # single spec-load path (merges house checks when named)
    sc = scoremod.score(cwd, spec_path)
    conv, feat = harness.split_failed(sc.get("failed", []))
    rounds = [harness._round_rec(1, sc, res, conv, feat)]
    print(
        f"  [{arm_label}] round 1: score={sc.get('total')} "
        f"arch={sc.get('arch_pass',0)}/10 "
        f"conv_fails={len(conv)} feat_fails={len(feat)} "
        f"cost=${cost:.4f}",
        flush=True,
    )

    stated_counts: dict = defaultdict(int)
    rnd = 1
    no_improve = 0
    conv_score = harness.convergence_score(sc)

    while not harness.converged(sc) and rnd < max_rounds and sc.get("build") == "ok":
        rnd += 1
        fb = harness.feedback_prompt(sc["failed"], stated_counts, spec)
        for lbl, _ in sc["failed"]:
            stated_counts[lbl] += 1
        res = do_build_call(cfg, model_key, vault, cwd, fb, resume_sid=sid)
        if bool(res.get("is_error")):
            print(f"  [{arm_label}] round {rnd}: error — stopping loop", file=sys.stderr, flush=True)
            break
        sc = scoremod.score(cwd, spec_path)
        conv, feat = harness.split_failed(sc.get("failed", []))
        rounds.append(harness._round_rec(rnd, sc, res, conv, feat))
        new_score = harness.convergence_score(sc)
        print(
            f"  [{arm_label}] round {rnd}: score={sc.get('total')} "
            f"arch={sc.get('arch_pass',0)}/10 "
            f"conv_fails={len(conv)} feat_fails={len(feat)} "
            f"cost=${(res.get('total_cost_usd') or 0):.4f}",
            flush=True,
        )
        if new_score > conv_score:
            conv_score = new_score
            no_improve = 0
        else:
            no_improve += 1
        if no_improve >= STALL_PATIENCE and not harness.converged(sc):
            print(
                f"  [{arm_label}] STALL at round {rnd} "
                f"(score flat for {STALL_PATIENCE} rounds) — halting",
                file=sys.stderr,
                flush=True,
            )
            break

    completed = harness.converged(sc)
    rounds_to_conv = rnd if completed else None
    return rounds, completed, rounds_to_conv, sid


# ---------------------------------------------------------------------------
# Per-app smoke run
# ---------------------------------------------------------------------------

def run_smoke_app(
    app: str,
    cfg_warm: str,
    model_key: str,
    vault_src: str,
    data_dir: str,
    max_rounds: int = MAX_ROUNDS,
) -> dict:
    """Run one app: recall → Arm A (carried) → Arm B (pruned). Returns per-app result dict."""
    spec_path = os.path.join(CUM, APP_SPEC[app])
    interface = scoremod.load_spec(spec_path)["interface"]  # single spec-load path

    # Arm A shares the recall cwd (recall runs first, then build resumes in same session).
    # Arm B gets its own cwd (fresh session, no prior context).
    wd_a = os.path.join(data_dir, f"{app}_arm_a")
    wd_b = os.path.join(data_dir, f"{app}_arm_b")
    shutil.rmtree(wd_a, ignore_errors=True)
    shutil.rmtree(wd_b, ignore_errors=True)
    os.makedirs(wd_a, exist_ok=True)
    os.makedirs(wd_b, exist_ok=True)

    # Isolated vault copies: Arm A and Arm B start from the same real vault state.
    vault_a = seed_vault(wd_a, vault_src)
    vault_b = seed_vault(wd_b, vault_src)

    # --- Recall (shared by both arms) ---
    # Uses wd_a / vault_a because Arm A resumes this session in that directory.
    print(f"\n{'='*60}", flush=True)
    print(f"RECALL: {app}", flush=True)
    print(f"{'='*60}", flush=True)
    t_recall_start = time.time()
    recall_res = do_build_call(
        cfg_warm, model_key, vault_a, wd_a, recall_synthesis_prompt(app)
    )
    t_recall_end = time.time()

    recall_sid = recall_res.get("session_id")
    synthesis_text = recall_res.get("result", "")
    recall_cost = round((recall_res.get("total_cost_usd") or 0), 4)

    if not recall_sid:
        raise RuntimeError(f"Recall for {app} returned no session_id")
    # A real Step-3 synthesis block is routinely 500-2000+ chars; below that the agent
    # likely failed to echo it into its final message, which would make Arm B a strawman
    # (no conventions) and the carried-vs-pruned comparison uninterpretable. Hard-abort
    # this app rather than producing a result from a strawman.
    if len(synthesis_text) < 500:
        print(
            f"  ABORT {app}: synthesis_text only {len(synthesis_text)} chars — recall did "
            f"not emit the full Step-3 block. Got:\n{synthesis_text!r}",
            file=sys.stderr,
            flush=True,
        )
        raise RuntimeError(
            f"{app}: synthesis_text too short ({len(synthesis_text)} chars) — Arm B would "
            "run on a strawman; aborting rather than producing an uninterpretable result"
        )

    print(
        f"  recall_sid={recall_sid}  cost=${recall_cost:.4f}  "
        f"synthesis_len={len(synthesis_text)} chars  "
        f"time={t_recall_end - t_recall_start:.0f}s",
        flush=True,
    )
    print(f"  synthesis preview: {synthesis_text[:300]!r}", flush=True)

    # --- Arm A: Carried — resume recall session, full payload in context ---
    print(f"\n{'='*60}", flush=True)
    print(f"ARM A (carried): {app}", flush=True)
    print(f"{'='*60}", flush=True)
    # build_prompt with include_recall=False: suppress re-invoking /recall in the resumed session.
    build_msg_a = harness.build_prompt(app, interface, "skill", include_recall=False)
    t_a_start = time.time()
    rounds_a, completed_a, rtc_a, _ = run_build_loop(
        cfg_warm, model_key, vault_a, wd_a, spec_path,
        first_prompt=build_msg_a,
        arm_label=f"A-{app}",
        resume_sid=recall_sid,
        max_rounds=max_rounds,
    )
    t_a_end = time.time()
    _, build_cost_a = harness.split_costs(recall_res, rounds_a)
    print(
        f"  ARM A done: build_cost=${build_cost_a:.4f}  "
        f"rounds={len(rounds_a)}  completed={completed_a}  "
        f"time={t_a_end - t_a_start:.0f}s",
        flush=True,
    )

    # --- Arm B: Pruned — fresh session, only synthesis injected ---
    print(f"\n{'='*60}", flush=True)
    print(f"ARM B (pruned): {app}", flush=True)
    print(f"{'='*60}", flush=True)
    # Arm B's first message = synthesis text + build prompt (no recall, read_mode="none").
    # The synthesis carries the "Apply these as requirements:" conventions; the build prompt
    # carries the app spec and build instructions without a /recall directive.
    build_msg_b = (
        synthesis_text
        + "\n\n"
        + harness.build_prompt(app, interface, "none", include_recall=False)
    )
    t_b_start = time.time()
    rounds_b, completed_b, rtc_b, _ = run_build_loop(
        cfg_warm, model_key, vault_b, wd_b, spec_path,
        first_prompt=build_msg_b,
        arm_label=f"B-{app}",
        resume_sid=None,  # fresh session — no payload carried
        max_rounds=max_rounds,
    )
    t_b_end = time.time()
    _, build_cost_b = harness.split_costs(None, rounds_b)
    print(
        f"  ARM B done: build_cost=${build_cost_b:.4f}  "
        f"rounds={len(rounds_b)}  completed={completed_b}  "
        f"time={t_b_end - t_b_start:.0f}s",
        flush=True,
    )

    return {
        "app": app,
        "model_key": model_key,
        "model_id": harness.MODELS[model_key],
        "recall": {
            "cost_usd": recall_cost,
            "sid": recall_sid,
            "synthesis_len": len(synthesis_text),
            "synthesis_preview": synthesis_text[:500],
            "time_s": round(t_recall_end - t_recall_start, 1),
        },
        "arm_a": {
            "name": "carried",
            "build_cost_usd": build_cost_a,
            "rounds": len(rounds_a),
            "rounds_to_converge": rtc_a,
            "completed": completed_a,
            "time_s": round(t_a_end - t_a_start, 1),
            "rounds_detail": rounds_a,
        },
        "arm_b": {
            "name": "pruned",
            "build_cost_usd": build_cost_b,
            "rounds": len(rounds_b),
            "rounds_to_converge": rtc_b,
            "completed": completed_b,
            "time_s": round(t_b_end - t_b_start, 1),
            "rounds_detail": rounds_b,
        },
    }


# ---------------------------------------------------------------------------
# Verdict logic
# ---------------------------------------------------------------------------

def compute_verdict(results: list[dict]) -> dict:
    """Compute per-metric aggregates and the verdict-gate outcome.

    Note on noise floor: n=3 is too small to size a noise floor from a same-arm A-vs-A
    contrast (the design specifies this — note 96). If the $ delta is below what the design
    says is meaningful (~$1 per op, note 95), the verdict is UNDERPOWERED.
    """
    total_a_cost = sum(r["arm_a"]["build_cost_usd"] for r in results)
    total_b_cost = sum(r["arm_b"]["build_cost_usd"] for r in results)
    delta_cost = total_b_cost - total_a_cost  # negative = B cheaper

    # rounds_to_converge: use total rounds (converged or not) for a fair comparison
    total_a_rounds = sum(r["arm_a"]["rounds"] for r in results)
    total_b_rounds = sum(r["arm_b"]["rounds"] for r in results)
    delta_rounds = total_b_rounds - total_a_rounds  # positive = B needs more rounds

    succ_a = sum(1 for r in results if r["arm_a"]["completed"])
    succ_b = sum(1 for r in results if r["arm_b"]["completed"])

    # Per-app deltas for the labeled table
    per_app = []
    for r in results:
        per_app.append({
            "app": r["app"],
            "cost_a": r["arm_a"]["build_cost_usd"],
            "cost_b": r["arm_b"]["build_cost_usd"],
            "delta_cost": r["arm_b"]["build_cost_usd"] - r["arm_a"]["build_cost_usd"],
            "rounds_a": r["arm_a"]["rounds"],
            "rounds_b": r["arm_b"]["rounds"],
            "delta_rounds": r["arm_b"]["rounds"] - r["arm_a"]["rounds"],
            "completed_a": r["arm_a"]["completed"],
            "completed_b": r["arm_b"]["completed"],
        })

    # Verdict gate (design doc):
    # Net win IFF: (1) B cheaper by >noise floor AND (2) rounds within ±1 AND (3) success not worse.
    # Noise floor: at n=3 we cannot size this empirically. Design says the ~$1/op premium is the
    # expected saving (note 95/100). Treat |delta| < $0.50 across all 3 apps as UNDERPOWERED.
    # (The design says "above noise floor" — at n=3 we can't compute it; we name the threshold.)
    NOISE_THRESHOLD_USD = 0.50  # total across 3 apps; below this → underpowered
    cost_saves = delta_cost < 0  # B is cheaper
    cost_material = abs(delta_cost) > NOISE_THRESHOLD_USD
    capability_ok = abs(delta_rounds) <= 1 and succ_b >= succ_a

    if not cost_saves:
        verdict = "not-a-win"
        reason = f"Arm B is NOT cheaper (Δ=${delta_cost:+.4f})"
    elif not cost_material:
        verdict = "underpowered"
        reason = (
            f"Arm B saves ${-delta_cost:.4f} but below the n=3 noise threshold "
            f"(${NOISE_THRESHOLD_USD:.2f} total). Cannot distinguish from noise at this n."
        )
    elif not capability_ok:
        verdict = "not-a-win"
        if succ_b < succ_a:
            reason = f"Cost saves but success regressed ({succ_a}/3 → {succ_b}/3) — under-capture"
        else:
            reason = (
                f"Cost saves but rounds regressed (Δ={delta_rounds:+d} rounds > ±1 tolerance) "
                "— synthesis may under-capture; not a win"
            )
    else:
        verdict = "win"
        reason = (
            f"Arm B saves ${-delta_cost:.4f} (>{NOISE_THRESHOLD_USD:.2f} threshold) "
            f"with rounds within ±1 (Δ={delta_rounds:+d}) and success {succ_b}/{len(results)} "
            f"vs {succ_a}/{len(results)} — isolation premise holds"
        )

    return {
        "total_a_cost": round(total_a_cost, 4),
        "total_b_cost": round(total_b_cost, 4),
        "delta_cost": round(delta_cost, 4),
        "total_a_rounds": total_a_rounds,
        "total_b_rounds": total_b_rounds,
        "delta_rounds": delta_rounds,
        "succ_a": succ_a,
        "succ_b": succ_b,
        "n": len(results),
        "verdict": verdict,
        "reason": reason,
        "noise_threshold_usd": NOISE_THRESHOLD_USD,
        "per_app": per_app,
    }


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    import argparse

    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "--model", default="opus", choices=list(harness.MODELS.keys()),
        help="Model key (default: opus → claude-opus-4-8)",
    )
    ap.add_argument(
        "--vault", default=REAL_VAULT,
        help="Path to the real engram vault (default: ~/.local/share/engram/vault)",
    )
    ap.add_argument(
        "--apps", default=",".join(APPS_HARDEST_FIRST),
        help="Comma-separated app list (default: feeds,links,notes — hardest first)",
    )
    ap.add_argument("--max-rounds", type=int, default=MAX_ROUNDS)
    ap.add_argument(
        "--data-dir", default=DATA_DIR,
        help="Directory for result JSONs and build workdirs",
    )
    args = ap.parse_args()

    apps = [a.strip() for a in args.apps.split(",")]
    model_key = args.model
    vault_src = args.vault
    data_dir = args.data_dir

    os.makedirs(data_dir, exist_ok=True)

    model_id = harness.MODELS[model_key]
    print(f"\nSmoke-prune: model={model_key} ({model_id})", flush=True)
    print(f"vault={vault_src}", flush=True)
    print(f"apps={apps}", flush=True)
    print(f"data_dir={data_dir}", flush=True)
    print(
        f"\nSpend estimate (~$7/warm-cell, prior runs): "
        f"1 recall + 2 builds per app = ~$6.50/app × {len(apps)} apps ≈ "
        f"~${6.5 * len(apps):.0f} total (estimate; report actual)",
        flush=True,
    )

    cfg_warm = setup_cfg(warm=True)

    results = []
    t_total_start = time.time()

    for app in apps:
        result = run_smoke_app(
            app=app,
            cfg_warm=cfg_warm,
            model_key=model_key,
            vault_src=vault_src,
            data_dir=data_dir,
            max_rounds=args.max_rounds,
        )
        results.append(result)

        # Save per-app result incrementally
        per_app_path = os.path.join(data_dir, f"smoke_prune_{app}.json")
        json.dump(result, open(per_app_path, "w"), indent=2)
        print(f"\n  Saved per-app result: {per_app_path}", flush=True)

    t_total_end = time.time()

    # Aggregate + verdict
    v = compute_verdict(results)
    total_recall_cost = sum(r["recall"]["cost_usd"] for r in results)
    total_build_cost_a = v["total_a_cost"]
    total_build_cost_b = v["total_b_cost"]
    actual_spend = round(total_recall_cost + total_build_cost_a + total_build_cost_b, 4)

    agg = {
        "apps": apps,
        "model_key": model_key,
        "model_id": model_id,
        "vault": vault_src,
        "results": results,
        "verdict": v,
        "total_recall_cost_usd": round(total_recall_cost, 4),
        "total_build_cost_a_usd": total_build_cost_a,
        "total_build_cost_b_usd": total_build_cost_b,
        "actual_spend_usd": actual_spend,
        "wall_min": round((t_total_end - t_total_start) / 60.0, 1),
    }
    agg_path = os.path.join(data_dir, "smoke_prune_results.json")
    json.dump(agg, open(agg_path, "w"), indent=2)

    # --- Print labeled results table (Joe's standing requirement) ---
    n = len(results)
    print("\n\n" + "=" * 80, flush=True)
    print("RESULTS TABLE", flush=True)
    print("=" * 80, flush=True)
    print(
        f"{'Metric':<25} {'Unit':<10} {'Arm A (carried)':>18} {'Arm B (pruned)':>18} "
        f"{'Δ':>10} {'vs noise':>14} {'sub-verdict':>14}",
        flush=True,
    )
    print("-" * 110, flush=True)

    # build_cost row
    cost_delta_str = f"{v['delta_cost']:+.4f}"
    cost_noise = "above" if abs(v["delta_cost"]) > v["noise_threshold_usd"] else "below"
    cost_sv = "✓" if (v["delta_cost"] < 0 and cost_noise == "above") else "✗"
    print(
        f"{'build_cost':<25} {'USD':<10} {total_build_cost_a:>18.4f} "
        f"{total_build_cost_b:>18.4f} {cost_delta_str:>10} "
        f"{'>' + str(v['noise_threshold_usd']) + ' threshold':>14} {cost_sv:>14}",
        flush=True,
    )

    # rounds_to_converge row (total rounds used — converged or not)
    rounds_delta_str = f"{v['delta_rounds']:+d}"
    rounds_noise = "within ±1" if abs(v["delta_rounds"]) <= 1 else "outside ±1"
    rounds_sv = "✓" if abs(v["delta_rounds"]) <= 1 else "✗"
    print(
        f"{'rounds (total used)':<25} {'rounds':<10} {v['total_a_rounds']:>18} "
        f"{v['total_b_rounds']:>18} {rounds_delta_str:>10} "
        f"{rounds_noise:>14} {rounds_sv:>14}",
        flush=True,
    )

    # success row
    succ_noise = "within noise" if v["succ_b"] >= v["succ_a"] else "regressed"
    succ_sv = "✓" if v["succ_b"] >= v["succ_a"] else "✗"
    print(
        f"{'success':<25} {'n/N':<10} "
        f"{v['succ_a']}/{n!s:>16} {v['succ_b']}/{n!s:>16} "
        f"{'':>10} {succ_noise:>14} {succ_sv:>14}",
        flush=True,
    )

    print("-" * 110, flush=True)
    print(
        f"{'Net win':<25} {'—':<10} {'—':>18} {'—':>18} {'—':>10} "
        f"{'—':>14} {v['verdict'].upper():>14}",
        flush=True,
    )
    print("=" * 80, flush=True)

    print(f"\nVERDICT: {v['verdict'].upper()}", flush=True)
    print(f"WHY: {v['reason']}", flush=True)

    print("\nPer-app breakdown:", flush=True)
    print(
        f"  {'App':<10} {'cost_A':>10} {'cost_B':>10} {'Δcost':>10} "
        f"{'rds_A':>7} {'rds_B':>7} {'Δrds':>7} {'ok_A':>6} {'ok_B':>6}",
        flush=True,
    )
    for p in v["per_app"]:
        print(
            f"  {p['app']:<10} {p['cost_a']:>10.4f} {p['cost_b']:>10.4f} "
            f"{p['delta_cost']:>+10.4f} {p['rounds_a']:>7} {p['rounds_b']:>7} "
            f"{p['delta_rounds']:>+7} {str(p['completed_a']):>6} {str(p['completed_b']):>6}",
            flush=True,
        )

    print(f"\nActual spend (recall + both build arms): ${actual_spend:.4f}", flush=True)
    print(f"  recall subtotal: ${total_recall_cost:.4f}", flush=True)
    print(f"  Arm A builds:    ${total_build_cost_a:.4f}", flush=True)
    print(f"  Arm B builds:    ${total_build_cost_b:.4f}", flush=True)
    print(f"\nWall time: {agg['wall_min']:.1f} min", flush=True)
    print(f"Raw results JSON: {agg_path}", flush=True)

    print(
        "\nApps chosen: feeds (8 checks), links (7 checks), notes (4 checks) — "
        "all 3 harness apps, hardest-first. Opus may one-shot all three "
        "(oracle-saturation observed in run #8 of EXPERIMENT-LOG). "
        "If all apps converge in 1 round, the rounds-regression signal is absent "
        "but the $ delta is still measurable (each round re-sends the payload).",
        flush=True,
    )


if __name__ == "__main__":
    main()
