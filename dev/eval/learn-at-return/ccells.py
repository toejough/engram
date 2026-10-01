#!/usr/bin/env python3
"""Confined curate-skill-wording validation arm launcher.

Re-run of openspec change fix-show-amend-reparent-frontmatter's tasks.md task 1.4, per the U1 review
(ruling V1): the first pass (unconfined `--dangerously-skip-permissions`, no delivery check, and the
pre-#772-fix `engram` binary still on PATH) is void. This reuses lar.py's confinement primitives
as-is, unmodified:

  - env -i + the exact D5 variable allowlist, HOME under /private/tmp, the sandbox + permissions.deny
    settings (lar.settings), per-batch write probes (lar.run_probe) that must pass before any arm
    runs, before/after real-vault and ~/.claude isolation checks (lar.isolation_snapshot/_record),
    the OAuth token handed to the arm through an inherited pipe fd, never argv or an env var the
    launcher's own process carries (lar.run_claude / lar.read_token).
  - score.py's stream-json parsing helpers (_events, _text_of) for the delivery-gate check.

What's curate-specific (lar.py's build_arm/score_arm are learn-at-return's own fixture/cell/report
machinery, not reusable here):

  - the arm's $ARM/home/.claude holds ONLY the one curate SKILL.md under test (RED: the pre-#772-fix
    text at commit RED_PIN; GREEN/PRESSURE: the current worktree text) plus a unique per-cell marker
    line, no CLAUDE.md, no other skill directory -- so nothing else could satisfy a curate-shaped
    request and nothing can shadow the treatment;
  - the `engram` binary under test is built from THIS worktree (`go build`, never `go install`) into
    a scratch dir and copied to $ARM/bin, which lar.run_claude's PATH already puts first;
  - the fixture is a pending skill-registration note whose red_flags render over the 1200-byte query
    preview budget, all still valid against its body (RED/GREEN: 22 entries/~1264 B; PRESSURE: 40
    entries/~2347 B);
  - the pass condition is task 1.4's own text: RED records whether the arm drops a still-valid entry
    (it is observational, not gated); GREEN and PRESSURE must keep every entry.

    python3 ccells.py batch --name v1-red --cell RED --n 3
    python3 ccells.py batch --name v1-green --cell GREEN --n 3
    python3 ccells.py batch --name v1-pressure --cell PRESSURE --n 3
"""
import argparse
import datetime as dt
import hashlib
import json
import os
import shutil
import subprocess
import sys
import threading
import time
from typing import Any, Dict, List, Tuple

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)

import lar  # noqa: E402  (reused: confinement primitives)
import score  # noqa: E402  (reused: stream-json parsing helpers)

REPO = lar.REPO
RESULTS = os.path.join(HERE, "results")
RED_PIN = "bbad31f6"  # the commit before #772's curate/SKILL.md wording fix (U1-review finding 3)
CURATE_SKILL_REL = "agent-instructions/skills/curate/SKILL.md"
MARKER_PREFIX = "Curate-cell-marker: "
MARKERS = {"RED": "CCELLS-RED-K2F8", "GREEN": "CCELLS-GREEN-R6V1", "PRESSURE": "CCELLS-PRESSURE-T9N4"}
DEFAULT_TIMEOUT_S = 600
COST_CAP_USD = 8.0


class CcellsError(RuntimeError):
    """Refuse to proceed: bad input, probe/isolation/confinement failure, or the cost cap."""


# ---------------------------------------------------------------------------
# engram binary (THIS worktree, `go build`, never `go install`)
# ---------------------------------------------------------------------------


def build_engram_binary(scratch: str) -> str:
    out = os.path.join(scratch, "engram")
    subprocess.run(["go", "build", "-o", out, "./cmd/engram"], cwd=REPO, check=True,
                   capture_output=True, text=True)
    return out


# ---------------------------------------------------------------------------
# curate SKILL.md text sources
# ---------------------------------------------------------------------------


def curate_text(cell: str) -> str:
    """RED reads the pre-#772-fix text at RED_PIN; GREEN/PRESSURE read the current worktree file
    (so a fix-round edit to the wording is picked up automatically on the next run)."""
    if cell == "RED":
        return subprocess.run(["git", "-C", REPO, "show", f"{RED_PIN}:{CURATE_SKILL_REL}"],
                              check=True, capture_output=True, text=True).stdout
    with open(os.path.join(REPO, CURATE_SKILL_REL)) as f:
        return f.read()


def marked_curate_text(cell: str) -> str:
    return curate_text(cell) + "\n" + MARKER_PREFIX + MARKERS[cell] + "\n"


# ---------------------------------------------------------------------------
# fixture: a pending skill-registration note with an oversized, all-valid red_flags list
# ---------------------------------------------------------------------------


FIXTURE_FLAGS = [
    "ran the migration without a backup branch first", "force-pushed main instead of the feature branch",
    "skipped the dry-run preview before applying", "forgot to re-run tests after the schema change",
    "merged without waiting for CI to go green", "assumed the staging config matched production",
    "deleted the lock file instead of regenerating it", "rebased a shared branch other people had already pulled",
    "ignored a failing linter warning as noise", "copied credentials into a committed file",
    "ran the destructive command against the wrong environment variable",
    "trusted a cached dependency instead of re-resolving it", "skipped the changelog entry for a breaking change",
    "assumed empty output meant success instead of checking the exit code",
    "left a debug flag enabled in the final commit", "renamed a public API without a deprecation shim",
    "closed the issue before the fix actually shipped", "edited the generated file instead of its source template",
    "ran two migrations out of order", "skipped the manual verification step before declaring done",
    "reused a branch name from a previous, unrelated change", "left a TODO in place of the actual fix",
]
PRESSURE_EXTRA_FLAGS = [
    "assumed the feature flag defaulted off in every environment", "trusted stale documentation instead of the current schema",
    "skipped code review because the change looked trivial", "assumed the test suite covered the new edge case",
    "left commented-out code in the final diff", "assumed the cache would invalidate automatically",
    "skipped updating the API version header", "trusted the CI cache instead of a clean build",
    "assumed downstream consumers would handle the new field gracefully",
    "left a hardcoded path that only works on one machine", "skipped notifying the on-call rotation about the deploy",
    "assumed the rollback plan had been tested", "trusted an unverified third-party dependency update",
    "left a race condition unaddressed because it was rare", "assumed the feature was backward compatible without checking",
    "skipped load-testing the new endpoint", "trusted a green dashboard without checking the underlying alert",
    "left the deprecated flag enabled past its sunset date",
]
FIXTURE_BASENAME = "1.ccells-fixture-skill"


def fixture_flags(cell: str) -> List[str]:
    return FIXTURE_FLAGS + PRESSURE_EXTRA_FLAGS if cell == "PRESSURE" else list(FIXTURE_FLAGS)


def fixture_note(cell: str) -> str:
    flags = fixture_flags(cell)
    fm_flags = "\n".join(f'    - "{f}"' for f in flags)
    body_lines = "\n".join(f"{i + 1}. Watch for: {f}." for i, f in enumerate(flags))
    return (
        "---\n"
        "type: runbook\n"
        "situation: refreshing a registered skill note during curation\n"
        "done_when: the note's runbook fields match its current body and the pending marker is cleared\n"
        f'skill_hash: "deadbeefcafefeed{"0" * (64 - 16)}"\n'
        'skill_key: "claude:ccells-fixture-skill"\n'
        'skill_source: "claude"\n'
        "triggers:\n    - fixture skill trigger phrase\n"
        f"red_flags:\n{fm_flags}\n"
        'luhmann: "1"\n'
        'created: "2026-01-01"\n'
        "pending: true\n"
        "---\n\n"
        "Fixture skill body. Every numbered item below corresponds to one still-valid red flag in "
        "the frontmatter above -- none of them are stale.\n\n"
        f"{body_lines}\n"
    )


# ---------------------------------------------------------------------------
# arm construction (curate-specific; reuses lar.make_dirs / lar.settings / lar._write / lar._sha)
# ---------------------------------------------------------------------------


CURATE_PROMPT = f"""You are curating a pending engram vault note.

Environment: the `engram` CLI binary is on PATH; ENGRAM_VAULT_PATH and XDG_DATA_HOME are already \
set, so `engram` subcommands need no --vault flag. Only `engram ...` Bash commands are permitted \
(the Bash tool is scoped to them) -- use the Skill tool, Read and `engram` only.

The pending note's basename is `{FIXTURE_BASENAME}`. Read it with `engram show {FIXTURE_BASENAME}`, \
and curate it using the curate skill (it is a pending skill runbook note whose runbook fields are \
already present -- this is the "refreshed notes" case in that skill). When you are done, print a \
one-line summary of exactly what you changed (or did not change) and why, including how many \
red_flags entries the note ends up with.
"""


def build_ccell_arm(arm: str, cell: str, engram_bin: str) -> None:
    lar.make_dirs(arm)
    cfg = os.path.join(arm, "home", ".claude")
    lar._write(os.path.join(cfg, "skills", "curate", "SKILL.md"), marked_curate_text(cell))
    # Deliberately no CLAUDE.md and no other skill directory: the arm's HOME holds only the one
    # curate SKILL.md under test, so nothing else can satisfy "curate this" and nothing can shadow
    # the treatment (U1-review finding 2).
    lar._write(os.path.join(arm, "vault", f"{FIXTURE_BASENAME}.md"), fixture_note(cell))
    shutil.copy2(engram_bin, os.path.join(arm, "bin", "engram"))


# ---------------------------------------------------------------------------
# scoring: red_flags entries kept/dropped + delivery gate
# ---------------------------------------------------------------------------


def read_written_note(arm_vault_copy: str) -> str:
    path = os.path.join(arm_vault_copy, f"{FIXTURE_BASENAME}.md")
    if not os.path.exists(path):
        return ""
    with open(path) as f:
        return f.read()


def entries_kept(written: str, cell: str) -> Tuple[List[str], List[str]]:
    flags = fixture_flags(cell)
    kept = [f for f in flags if f in written]
    dropped = [f for f in flags if f not in written]
    return kept, dropped


def delivery_gate(all_text: str, cell: str) -> Dict[str, Any]:
    own = MARKERS[cell]
    foreign = [tok for name, tok in MARKERS.items() if name != cell and tok in all_text]
    return {"own_marker_present": own in all_text, "foreign_markers_present": foreign,
           "ok": own in all_text and not foreign}


def score_arm(out_dir: str, cell: str) -> Dict[str, Any]:
    with open(os.path.join(out_dir, "stream.jsonl")) as f:
        stream_lines = f.readlines()
    events = score._events(stream_lines)
    result = next((e for e in reversed(events) if e.get("type") == "result"), None)
    cost = (result or {}).get("total_cost_usd")
    final_text = score._text_of((result or {}).get("result"))

    session_texts = []
    session_dir = os.path.join(out_dir, "session")
    if os.path.isdir(session_dir):
        for dirpath, _, files in os.walk(session_dir):
            for fn in files:
                if fn.endswith(".jsonl"):
                    with open(os.path.join(dirpath, fn), errors="replace") as f:
                        session_texts.append(f.read())

    all_text = "\n".join(stream_lines) + "\n" + "\n".join(session_texts)
    gate = delivery_gate(all_text, cell)

    written = read_written_note(os.path.join(out_dir, "vault"))
    kept, dropped = entries_kept(written, cell)

    rec: Dict[str, Any] = {
        "cell": cell, "cost_usd": cost, "final_message": final_text,
        "delivery_gate": gate, "entries_total": len(fixture_flags(cell)),
        "entries_kept": len(kept), "entries_dropped": len(dropped), "dropped_entries": dropped,
        "written_note_present": bool(written),
    }
    if cell == "RED":
        # Observational (task 1.4): record whether it drops an entry or measures plain bytes; not
        # gated pass/fail the way GREEN/PRESSURE are.
        rec["drop_observed"] = len(dropped) > 0
        rec["label"] = "drop-observed" if rec["drop_observed"] else "no-drop"
    else:
        rec["label"] = "PASS" if (gate["ok"] and written and not dropped) else "FAIL"
    return rec


# ---------------------------------------------------------------------------
# batch
# ---------------------------------------------------------------------------


def _append(path: str, rec: Dict[str, Any], lock: threading.Lock) -> None:
    with lock:
        with open(path, "a") as f:
            f.write(json.dumps(rec) + "\n")
            f.flush()


def cumulative_cost() -> float:
    cost_log = os.path.join(RESULTS, "ccells-cost-log.jsonl")
    if not os.path.exists(cost_log):
        return 0.0
    total = 0.0
    with open(cost_log) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            total += json.loads(line).get("cost_usd") or 0.0
    return total


def run_batch(name: str, cell: str, n: int, model: str, timeout: int, claude_bin: str,
             engram_bin: str, cost_cap: float) -> int:
    if cell not in MARKERS:
        raise CcellsError(f"unknown cell {cell!r}: want RED, GREEN or PRESSURE")
    batch_dir = os.path.join(RESULTS, f"ccells-{name}")
    if os.path.exists(batch_dir):
        raise CcellsError(f"{batch_dir} exists; pick a new batch name")
    os.makedirs(batch_dir)
    home, vault, claude_dir = lar.real_paths()
    start = time.time()
    before = lar.isolation_snapshot(vault, claude_dir, start)
    lock = threading.Lock()
    cost_log = os.path.join(RESULTS, "ccells-cost-log.jsonl")
    manifest: Dict[str, Any] = {"batch": name, "cell": cell, "n": n,
                                "started": dt.datetime.fromtimestamp(start, dt.timezone.utc).isoformat(),
                                "arms": []}
    status = 0
    try:
        spent = cumulative_cost()
        manifest["cost_before_batch_usd"] = spent
        if spent >= cost_cap:
            raise CcellsError(f"cost cap ${cost_cap} already reached (${spent:.4f} spent) -- aborting before any arm")
        with open(engram_bin, "rb") as fh:
            engram_sha = hashlib.sha256(fh.read()).hexdigest()
        manifest.update({
            "claude_bin": claude_bin, "claude_version": lar.claude_version(claude_bin),
            "engram_bin": engram_bin, "engram_bin_sha256": engram_sha,
            "model_requested": model, "repo_head": subprocess.run(
                ["git", "-C", REPO, "rev-parse", "HEAD"], check=True, capture_output=True, text=True).stdout.strip(),
            "markers": MARKERS, "cost_cap_usd": cost_cap,
        })
        token = lar.read_token()
        deny = lar.deny_entries(lar.claude_tmp_root())
        manifest["deny_write"] = deny
        probe = lar.run_probe(batch_dir, token, deny, home, claude_bin)
        _append(cost_log, {"batch": name, "arm_id": "probe", "kind": "probe", "cost_usd": probe["cost_usd"]}, lock)
        manifest["probe"] = {k: probe[k] for k in ("ok", "reason", "cost_usd")}
        if not probe["ok"]:
            manifest["aborted"] = "probe failed: " + probe["reason"]
            status = 2
        else:
            for i in range(n):
                _run_one(i, cell, batch_dir, engram_bin, deny, home, token, model, timeout, claude_bin,
                        name, cost_log, lock, manifest, cost_cap)
    except BaseException as e:  # record, then re-raise: the after-check and manifest must still land
        manifest["error"] = f"{type(e).__name__}: {e}"
        status = 4
        raise
    finally:
        after = lar.isolation_snapshot(vault, claude_dir, start)
        iso = lar.isolation_record(start, vault, claude_dir, before, after)
        lar._write(os.path.join(batch_dir, "isolation.json"), json.dumps(iso, indent=2) + "\n")
        manifest["isolation_ok"] = iso["ok"]
        if not iso["ok"]:
            manifest["failed"] = "real-vault entry newer than batch start"
            status = status or 3
        manifest["finished"] = dt.datetime.now(dt.timezone.utc).isoformat()
        lar._write(os.path.join(batch_dir, "run-manifest.json"), json.dumps(manifest, indent=2) + "\n")
        print(f"probe={manifest.get('probe', {}).get('ok')} isolation={iso['ok']} status={status}", flush=True)
    return status


def _run_one(i: int, cell: str, batch_dir: str, engram_bin: str, deny: List[str], home: str, token: str,
            model: str, timeout: int, claude_bin: str, name: str, cost_log: str, lock: threading.Lock,
            manifest: Dict[str, Any], cost_cap: float) -> None:
    spent = cumulative_cost()
    if spent >= cost_cap:
        raise CcellsError(f"cost cap ${cost_cap} reached (${spent:.4f} spent) before arm {i} -- aborting batch")
    arm_id = f"{i:02d}-{cell}"
    out_dir = os.path.join(batch_dir, arm_id)
    arm = lar.new_arm_dir()
    try:
        build_ccell_arm(arm, cell, engram_bin)
        lar._write(os.path.join(arm, "home", ".claude", "settings.json"),
                  json.dumps(lar.settings(arm, deny, home), indent=2) + "\n")
        launch = lar.run_claude(arm, CURATE_PROMPT, token, model, out_dir, timeout, claude_bin, extra_tools=())
        main_texts, other_texts = lar.collect_sessions(arm, os.path.join(out_dir, "session"))
        shutil.copytree(os.path.join(arm, "vault"), os.path.join(out_dir, "vault"), dirs_exist_ok=True)
    finally:
        lar.remove_arm(arm)
    rec = {**score_arm(out_dir, cell), "arm_id": arm_id, "timed_out": launch["timed_out"],
          "wall_s": launch["wall_s"]}
    lar._write(os.path.join(out_dir, "score.json"), json.dumps(rec, indent=2) + "\n")
    _append(os.path.join(batch_dir, "arms.jsonl"), rec, lock)
    _append(cost_log, {"batch": name, "arm_id": arm_id, "kind": "arm", "cell": cell,
                       "cost_usd": rec["cost_usd"], "label": rec["label"]}, lock)
    manifest["arms"].append({"arm_id": arm_id, "cell": cell, "label": rec["label"], "cost_usd": rec["cost_usd"]})
    print(f"{arm_id}: {rec['label']} dropped={rec['entries_dropped']}/{rec['entries_total']} "
         f"gate={rec['delivery_gate']['ok']} cost=${rec['cost_usd']}", flush=True)


def main(argv=None) -> int:
    ap = argparse.ArgumentParser(description="confined curate-skill-wording validation launcher")
    sub = ap.add_subparsers(dest="cmd", required=True)
    b = sub.add_parser("batch")
    b.add_argument("--name", required=True)
    b.add_argument("--cell", required=True, choices=sorted(MARKERS))
    b.add_argument("--n", type=int, default=3)
    b.add_argument("--model", default="sonnet")
    b.add_argument("--timeout", type=int, default=DEFAULT_TIMEOUT_S)
    b.add_argument("--claude-bin", default=os.path.realpath(os.path.expanduser("~/.local/bin/claude")))
    b.add_argument("--engram-bin", required=True, help="built from THIS worktree, see build_engram_binary")
    b.add_argument("--cost-cap", type=float, default=COST_CAP_USD)
    a = ap.parse_args(argv)
    return run_batch(a.name, a.cell, a.n, a.model, a.timeout, a.claude_bin, a.engram_bin, a.cost_cap)


if __name__ == "__main__":
    sys.exit(main())
