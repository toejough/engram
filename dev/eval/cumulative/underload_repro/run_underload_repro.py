#!/usr/bin/env python3
"""underload_repro — headless multi-turn repro harness for the under-load endorse-moment
recall miss (see /Users/joe/.claude/jobs/ac0c61e1/tmp/underload-repro-spec.md, RED-only).

Goal: reproduce, not fix, this real failure — an agent under multi-turn LOAD endorses/ranks/
recommends a proposal WITHOUT recalling a refuted lever that memory would have surfaced, so it
re-proposes a killed lever blind. Two prior harnesses each capture only part of this:
  * dev/eval/cumulative/lever_recheck (C7) FORCES recall (RECALL_PREFIX) — measures in-recall
    re-check fidelity, not the decision to recall at all.
  * dev/eval/cumulative/endorse_cue is single-shot — guidance is maximally salient with no
    competing turns, and over-fires (15/15 — a null; see endorse_cue/README.md "Status").

This harness adds the missing ingredient: real multi-turn LOAD via `claude --resume`, built on
the SAME primitives as both priors (never reinvented):
  * harness.claude() — the `claude -p` / `--resume` subprocess primitive (dev/eval/cumulative/
    harness.py). ALSO used here for its `chunks=` param, which sets ENGRAM_CHUNKS_DIR — load-
    bearing: a smoke test proved `engram query`'s chunk-index component defaults to
    $XDG_DATA_HOME/engram/chunks (Joe's REAL production chunk store) when unset, and `/recall
    glance` ad hoc `engram ingest`s the trial's CLAUDE.md + its own session transcript INTO
    whatever chunks dir is active. Every call below passes an isolated per-trial chunks dir —
    never omit `chunks=`, or a live trial will write fictional-fixture content into Joe's real
    vault chunk index (this happened once during harness development; cleaned up by hand).
  * matrix.build_cfg_template(warm=True) / matrix.refresh_creds — the isolated CLAUDE_CONFIG_DIR
    with the repo's real recall+learn skills wired in and creds injected (dev/eval/cumulative/
    matrix.py).
  * lever_recheck_scorer.score_fixture — C7's AMNESIA/RECONCILED adversarial judge, reused
    as-is (default-AMNESIA, majority-vote over JUDGE_RUNS, meaning-based not vocabulary-based).
  * The seeded-vault + closed_levers.json shape (canonical_action/closure_reason/lever_terms)
    from lever_recheck's fixtures, rebuilt fresh (build_fixture_vaults.py) via REAL `engram
    learn` + `engram embed apply` — never hand-authored markdown/sidecars.
  * endorse_cue's UN-forced framing: NO RECALL_PREFIX anywhere in this file. The agent decides
    whether to recall; guidance/recall.md is inlined verbatim into the trial's project CLAUDE.md
    (never @import — a fresh project's first @import triggers Claude Code's one-time approval
    dialog, which a headless run with no TTY silently fails in BOTH arms — note 151), so the
    ambient cue is present exactly as it is in a real global CLAUDE.md, not force-invoked.

The NEW ingredient — 4-turn LOAD, one resumed session per trial:
  Turn 1: adjacent substantive work (review/sanity-check a doc) — builds context, no decision
          about the refuted lever.
  Turn 2: the agent HANDLES the refutation itself in a NON-DECISION framing — asked to write a
          decisions-archive wiki entry from raw notes, one of which recounts the closed lever's
          rollback. Mirrors the real failure ("I filed the refuting note myself, then didn't
          connect it") — the agent sees the refutation but files it away.
  Turn 3: more adjacent load — triage two OTHER backlog items, neither the refuted lever.
  Turn 4 (the endorse moment): CASUAL continuation framing — "someone proposed X [the refuted
          mechanism] — worth doing?" No RECOMMENDATION: directive, no forced format; the whole
          reply is scored as-is (recheck.extract_recommendation falls back to full text when no
          explicit line is found — exactly this case, by design).

Validity marker (endorse_cue-derived, adapted for multi-turn): a per-trial marker is appended to
the inlined CLAUDE.md; turn 1's prompt (only) asks the agent to echo a `PROJECT-TOKEN: <token>`
line for "internal tracking" — a neutral aside unrelated to recall, asked once at the FAR end of
the session from the endorse moment so it cannot prime turn 4. `marker_seen=False` trials are
discarded (never scored) — verified-treatment-delivery, not assumed.

Measure: at turn 4, the C7 judge decides AMNESIA (advocated the lever, did not reconcile the
prior rollback) vs RECONCILED (declined it, or advocated it while engaging the closure reason —
whether from the agent's own turn-2 memory of writing the archive, or from an actual recall
firing). `recall_fired` per turn (does `"skill":"recall"` AND `engram query` newly appear in the
session transcript since the last turn) is recorded as a DIAGNOSTIC alongside the verdict —
never folded into the pass/fail — so a synthesis miss (recalled, still amnesic) is distinguishable
from a pure decision-to-recall miss (never recalled, endorsed blind) and from a genuine
in-context save (never recalled, still reconciled from turn-2 memory).

Usage:
  python3 run_underload_repro.py --n 3 --model opus --out results/red_baseline.jsonl \
      [--fixtures fixture1_beacon_relay,fixture2_driftwood_index] [--workers 3] [--resume] \
      [--guidance path/to/guidance.md]
"""
import argparse
import concurrent.futures as cf
import glob
import json
import os
import shutil
import sys
import tempfile
import time
import uuid

HERE = os.path.dirname(os.path.abspath(__file__))
CUM = os.path.dirname(HERE)  # dev/eval/cumulative
sys.path.insert(0, CUM)
import harness  # noqa: E402  (claude(), MODELS)
import lever_recheck_scorer as scorer  # noqa: E402  (AMNESIA/RECONCILED judge, cheap import)
import matrix  # noqa: E402  (build_cfg_template, refresh_creds, REPO)
import recheck  # noqa: E402  (agent_text, extract_recommendation, rec_line_found)

FIXTURES_DIR = os.path.join(HERE, "fixtures")
DEFAULT_OUT = os.path.join(HERE, "results", "red_baseline.jsonl")
GUIDANCE_PATH = os.path.join(matrix.REPO, "guidance", "recall.md")

MIN_VALID_COST_USD = 0.02  # degraded-call floor, per dev/eval/traps/wrun.py's heuristic (reused
                            # by C7's run_recheck.py; a near-zero-cost "success" is a degraded call)
TRANSIENT_BACKOFFS = (0, 15, 45, 120)
RETRY_CAP_MULTIPLIER = 2

TURN_FILES = ["turn1.txt", "turn2.txt", "turn3.txt", "turn4.txt"]

MARKER_INSTRUCTION = (
    "\n\n(Separately, for our internal session tracking — unrelated to the above: end your "
    "reply with one extra line, `PROJECT-TOKEN: <token>`, where <token> is the project "
    "instructions token given in your CLAUDE.md, if your project instructions contain one. If "
    "not, write `PROJECT-TOKEN: NONE`.)"
)


def discover_fixtures():
    return sorted(
        name for name in os.listdir(FIXTURES_DIR)
        if name.startswith("fixture") and os.path.isdir(os.path.join(FIXTURES_DIR, name))
    )


def resolve_fixtures(arg):
    if not arg or arg.strip().lower() == "all":
        return discover_fixtures()
    return [f.strip() for f in arg.split(",") if f.strip()]


def read_turn_texts(fixture_dir):
    texts = []
    for fn in TURN_FILES:
        path = os.path.join(fixture_dir, fn)
        if not os.path.isfile(path):
            raise FileNotFoundError(f"fixture turn file missing: {path!r}")
        with open(path) as fh:
            texts.append(fh.read().strip())
    return texts


# ----- validity-marker CLAUDE.md scaffolding (endorse_cue-derived) -----

def build_trial_cwd(guidance_text, marker):
    wd = tempfile.mkdtemp(prefix="underload-repro-wd-")
    claude_md = (
        guidance_text.rstrip()
        + "\n\nThis project's guidance-instance token (if ever asked for a project token, it is "
          f"this — echo it back verbatim when asked): {marker}\n"
    )
    with open(os.path.join(wd, "CLAUDE.md"), "w") as f:
        f.write(claude_md)
    return wd


def _guidance_probe(guidance_text):
    """The note-194 treatment-delivery probe: guidance_text's first non-blank line (of ANY
    kind — markdown heading, HTML comment, plain bullet), stripped. Unlike a heading-only probe,
    this always yields a value for non-empty guidance, so the delivery gate can never silently
    skip itself for heading-less treatment text. Returns None only for empty/whitespace-only
    input."""
    return next((line.strip() for line in guidance_text.splitlines() if line.strip()), None)


# ----- transcript scanning: cumulative marker counts across the WHOLE isolated cfg (a fresh
# per-trial CLAUDE_CONFIG_DIR holds exactly one project, so no sid/filename matching is needed —
# just sum both needles across every jsonl under cfg/projects, main session + any subagents). -----

def _count_markers(cfg):
    proj = os.path.join(cfg, "projects")
    skill_n, query_n = 0, 0
    if not os.path.isdir(proj):
        return 0, 0
    for root, _, files in os.walk(proj):
        for fn in files:
            if not fn.endswith(".jsonl"):
                continue
            try:
                txt = open(os.path.join(root, fn), errors="replace").read()
            except OSError:
                continue
            skill_n += txt.count('"skill":"recall"')
            query_n += txt.count("engram query")
    return skill_n, query_n


def _call_cost(out):
    return round(float((out.get("total_cost_usd") if isinstance(out, dict) else 0.0) or 0.0), 4)


def _degraded(out):
    cost = _call_cost(out)
    return bool(out.get("is_error")) and cost < MIN_VALID_COST_USD


def call_turn(cfg, model, vault, wd, chunks, prompt, resume_sid=None):
    """One turn, with the family's transient-rate-limit backoff/retry (harness.py's do_build /
    endorse_cue probe.py's TRANSIENT_BACKOFFS pattern): only retries the degraded-call signature
    (is_error AND cost below floor), never a genuine substantive failure."""
    out = {}
    for backoff in TRANSIENT_BACKOFFS:
        if backoff:
            time.sleep(backoff)
        out = harness.claude(cfg, model, vault, wd, prompt, resume_sid=resume_sid, chunks=chunks)
        if not _degraded(out):
            break
    return out


# ----- one trial: 4 sequential resumed turns -----

def run_one_trial(fixture_name, fixture_dir, model, judge_model, guidance_text):
    marker = f"UNDERLOAD-REPRO-MARKER-{uuid.uuid4().hex[:8]}"
    turn_texts = read_turn_texts(fixture_dir)

    scratch = tempfile.mkdtemp(prefix="underload-repro-")
    t0 = time.time()
    try:
        cfg = os.path.join(scratch, "cfg")
        matrix.build_cfg_template(cfg, warm=True)
        matrix.refresh_creds(cfg)
        chunks = os.path.join(scratch, "chunks")
        os.makedirs(chunks, exist_ok=True)  # ENGRAM_CHUNKS_DIR: isolated, never the real default
        vault = os.path.join(scratch, "vault")
        shutil.copytree(os.path.join(fixture_dir, "vault_seed"), vault)
        wd = build_trial_cwd(guidance_text, marker)

        # note 194 treatment-delivery gate: build_trial_cwd inlines guidance_text into the
        # trial's CLAUDE.md; verify it actually landed rather than trusting the seam silently.
        # The probe is guidance_text's first non-blank line of ANY kind (heading, HTML comment,
        # plain bullet) so this ALWAYS runs — never silently skips for heading-less guidance,
        # the exact failure note 194 exists to prevent.
        guidance_probe = _guidance_probe(guidance_text)
        if guidance_probe is None:
            raise RuntimeError(
                f"treatment-delivery gate failed for fixture {fixture_name!r}: guidance_text is "
                "empty/whitespace-only — cannot deliver an empty treatment"
            )
        with open(os.path.join(wd, "CLAUDE.md")) as probe_check_fh:
            inlined_claude_md = probe_check_fh.read()
        if guidance_probe not in inlined_claude_md:
            raise RuntimeError(
                f"treatment-delivery gate failed for fixture {fixture_name!r}: guidance probe "
                f"{guidance_probe!r} not found in inlined CLAUDE.md at {wd!r}"
            )

        record = {"fixture": fixture_name, "model": model, "judge_model": judge_model,
                  "marker": marker, "session_id": None}

        turn_costs = []
        turn_recall_fired = []
        prev_skill_n, prev_query_n = 0, 0
        sid = None
        agent_texts = []

        for i, base_text in enumerate(turn_texts, start=1):
            prompt = base_text + (MARKER_INSTRUCTION if i == 1 else "")
            out = call_turn(cfg, model, vault, wd, chunks, prompt, resume_sid=sid)
            cost = _call_cost(out)
            turn_costs.append(cost)
            text = recheck.agent_text(out)
            agent_texts.append(text)
            sid = sid or out.get("session_id")
            record["session_id"] = sid

            if i == 1:
                marker_seen = marker in text
                record["marker_seen"] = marker_seen
                if not marker_seen or _degraded(out):
                    record.update({
                        "status": "invalid",
                        "invalid_reason": "no_marker" if not marker_seen else "degraded_turn1",
                        "turn_costs": turn_costs, "total_cost_usd": round(sum(turn_costs), 4),
                        "turn_recall_fired": turn_recall_fired, "agent_texts": agent_texts,
                        "wall_s": round(time.time() - t0, 1),
                    })
                    return record  # short-circuit — never pay for turns 2-4 on an invalid trial
            elif _degraded(out):
                record.update({
                    "status": "invalid", "invalid_reason": f"degraded_turn{i}",
                    "turn_costs": turn_costs, "total_cost_usd": round(sum(turn_costs), 4),
                    "turn_recall_fired": turn_recall_fired, "agent_texts": agent_texts,
                    "wall_s": round(time.time() - t0, 1),
                })
                return record

            skill_n, query_n = _count_markers(cfg)
            fired = (skill_n > prev_skill_n) and (query_n > prev_query_n)
            turn_recall_fired.append(fired)
            prev_skill_n, prev_query_n = skill_n, query_n

        recommendation = agent_texts[-1]
        if not recommendation.strip():
            record.update({
                "status": "invalid", "invalid_reason": "empty_turn4_text",
                "turn_costs": turn_costs, "total_cost_usd": round(sum(turn_costs), 4),
                "turn_recall_fired": turn_recall_fired, "agent_texts": agent_texts,
                "wall_s": round(time.time() - t0, 1),
            })
            return record

        recall_fired_any = any(turn_recall_fired)
        scored = scorer.score_fixture(recommendation, fixture_dir, note_surfaced=recall_fired_any,
                                      stub=False, judge_model=judge_model, unforced=True)

        record.update({
            "status": "valid",
            "turn_costs": turn_costs, "total_cost_usd": round(sum(turn_costs), 4),
            "turn_recall_fired": turn_recall_fired,
            "recall_fired_any": recall_fired_any,
            "recall_fired_turn4": turn_recall_fired[3] if len(turn_recall_fired) > 3 else False,
            "recommendation": recommendation,
            "rec_line_found": recheck.rec_line_found(recommendation),
            "cell_verdict": scored["cell_verdict"],
            "per_lever": scored["per_lever"],
            "agent_texts": agent_texts,
            "wall_s": round(time.time() - t0, 1),
        })
        return record
    finally:
        shutil.rmtree(scratch, ignore_errors=True)


# ----- checkpointed batch driver (append_jsonl/load_completed pattern from run_recheck.py) -----

def append_jsonl(out_path, record):
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
    completed = {}
    for row in read_jsonl(out_path):
        if row.get("kind"):
            continue
        completed[(row["fixture"], row["trial_idx"])] = row.get("status")
    return completed


def run_fixture(fixture_name, fixture_dir, n, retry_cap, out_path, model, judge_model,
                guidance_text, already_done):
    valid_count = sum(1 for s in already_done.values() if s == "valid")
    attempts = len(already_done)
    next_idx = (max(already_done) + 1) if already_done else 0
    new_records = []
    while valid_count < n and attempts < retry_cap:
        record = run_one_trial(fixture_name, fixture_dir, model, judge_model, guidance_text)
        record["trial_idx"] = next_idx
        append_jsonl(out_path, record)
        new_records.append(record)
        attempts += 1
        if record.get("status") == "valid":
            valid_count += 1
        next_idx += 1
    if valid_count < n and attempts >= retry_cap and new_records:
        append_jsonl(out_path, {"kind": "cap_exhausted", "fixture": fixture_name,
                                "attempts": attempts, "valid": valid_count, "target_valid": n})
    return new_records


def summarize(records):
    per_fixture = {}
    total_cost = 0.0
    marker_seen_n = 0
    invalid_n = 0
    for r in records:
        if r.get("kind"):
            continue
        total_cost += r.get("total_cost_usd") or 0.0
        if r.get("marker_seen"):
            marker_seen_n += 1
        agg = per_fixture.setdefault(r["fixture"], {"valid": 0, "invalid": 0, "amnesia": 0,
                                                      "reconciled": 0, "marker_seen": 0})
        if r.get("marker_seen"):
            agg["marker_seen"] += 1
        if r.get("status") != "valid":
            agg["invalid"] += 1
            invalid_n += 1
            continue
        agg["valid"] += 1
        if r.get("cell_verdict") == "AMNESIA":
            agg["amnesia"] += 1
        else:
            agg["reconciled"] += 1
    overall_valid = sum(a["valid"] for a in per_fixture.values())
    overall_amnesia = sum(a["amnesia"] for a in per_fixture.values())
    return {"kind": "summary", "per_fixture": per_fixture, "overall_valid": overall_valid,
            "overall_amnesia": overall_amnesia,
            "overall_amnesia_rate": round(overall_amnesia / overall_valid, 3) if overall_valid else None,
            "marker_seen_total": marker_seen_n, "invalid_total": invalid_n,
            "total_cost_usd": round(total_cost, 4)}


def build_argparser():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--fixtures", default="all")
    ap.add_argument("--n", type=int, default=3, help="target VALID trials per fixture (default 3)")
    ap.add_argument("--model", default="opus", choices=list(harness.MODELS),
                     help="opus is the target model per spec; a weaker model is invalid")
    ap.add_argument("--judge-model", default=scorer.DEFAULT_JUDGE_MODEL)
    ap.add_argument("--out", default=DEFAULT_OUT)
    ap.add_argument("--guidance", default=GUIDANCE_PATH,
                     help="path to the recall-firing guidance inlined into each trial CLAUDE.md "
                          "(default: repo guidance/recall.md — the current-wording control arm)")
    ap.add_argument("--workers", type=int, default=3)
    ap.add_argument("--resume", action="store_true", help="skip (fixture, trial_idx) rows already in --out")
    return ap


def main(argv=None):
    args = build_argparser().parse_args(argv)
    fixtures = resolve_fixtures(args.fixtures)
    if not fixtures:
        raise SystemExit(f"no fixtures matched --fixtures={args.fixtures!r} under {FIXTURES_DIR}")

    if os.path.isfile(args.out) and not args.resume:
        raise SystemExit(f"--out {args.out!r} exists; pass --resume or remove it first")

    completed = load_completed(args.out) if args.resume else {}
    retry_cap = args.n * RETRY_CAP_MULTIPLIER
    guidance_text = open(args.guidance).read()

    with cf.ThreadPoolExecutor(max_workers=args.workers) as ex:
        futs = {}
        for name in fixtures:
            fixture_dir = os.path.join(FIXTURES_DIR, name)
            already = {idx: status for (f, idx), status in completed.items() if f == name}
            fut = ex.submit(run_fixture, name, fixture_dir, args.n, retry_cap, args.out,
                            args.model, args.judge_model, guidance_text, already)
            futs[fut] = name
        for fut in cf.as_completed(futs):
            name = futs[fut]
            recs = fut.result()
            print(f"=== {name}: {len(recs)} new attempt(s) ===")
            for r in recs:
                verdict = r.get("cell_verdict") or r.get("invalid_reason") or r.get("status")
                print(f"  trial {r['trial_idx']}: status={r['status']} verdict={verdict} "
                      f"marker_seen={r.get('marker_seen')} cost=${r.get('total_cost_usd', 0):.2f}")

    summary = summarize(read_jsonl(args.out))
    append_jsonl(args.out, summary)
    print(json.dumps(summary, indent=2))


if __name__ == "__main__":
    main()
