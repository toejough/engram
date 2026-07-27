#!/usr/bin/env python3
"""please_step7_probe — headless micro-test for skills/please/SKILL.md's Step 7 (surprise harvest)
and the plea's second pass (engram issue #687, Unit 1 only).

--role clean_auditor (default): does Step 7's surprise-harvest text make an auditor enumerate
surprises using S1–S7 markers and ask the counterfactual over a clean cycle record, or skip it?

--role loaded_auditor: does the surprise harvest surface the pre-registered discriminators when
audit is a buried subtask under cycle-close load, not the salient task?

See docs/superpowers/plans/2026-07-26-687-surprise-harvest.md, Unit 1.

Design (dev/eval/cumulative/please_step3_probe/run_probe.py's pattern, reused not imported):
  - Isolation = INLINE. The candidate please/SKILL.md Step 7 text is inlined into a fresh
    per-trial project CLAUDE.md, never deployed to the real ~/.claude/skills/please/.
  - Fresh headless `claude -p` per trial (never a Task-tool subagent): a subagent inherits
    the calling session's context, which already discusses the markers under test.
  - Self-contained fixture with real cycle record (commit_log.txt, gate_log.txt, plan_excerpt.txt,
    already_captured_lessons.txt) — testdata/ contains no live repo or vault paths.
  - Treatment-delivery validity gate: marker token in CLAUDE.md, surfacing request in -p prompt,
    marker_seen=False trials discarded not scored.
  - Scoring is MECHANICAL: substring/regex match for each discriminator's keyword set, with a
    proximity window (within 200 chars of a marker/classification token) reusing the
    _distractor_over_included windowed-proximity technique from please_step3_probe.
"""
import argparse
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
HERE = os.path.dirname(os.path.abspath(__file__))
FIXTURE_DIR = os.path.join(HERE, "testdata", "fixture_dedup_cycle")
TRANSIENT_BACKOFFS = (0, 15, 45, 120)
MIN_VALID_COST_USD = 0.02

# Ground truth discriminators from GROUND_TRUTH.md
# discriminator_1: record-subset premise false; should NOT be reached by existing 4-marker audit
# discriminator_2: live-index near-miss during Unit 5 verification; should NOT be reached
# adr_hedge: Gate D caught it; SHOULD be reached
# uncited_number: Gate D caught it; SHOULD be reached

DISCRIMINATORS = {
    "discriminator_1": {
        "keywords": r"(mergeChunkRecords|append-only within|more than one ingest day|record.subset|disjoint historical)",
        "proximity_window": 100,  # Tighter scope: ~100 chars around keyword prevents cross-section false positives
    },
    "discriminator_2": {
        "keywords": r"(ENGRAM_CHUNKS_DIR|resolves.{0,20}own path|redirect.{0,30}not honou?red)",
        "proximity_window": 100,
    },
    "adr_hedge": {
        "keywords": r"(ADR.*hedge|documentation.*hedge|narrowed|supersed)",
        "proximity_window": 100,
    },
    "uncited_number": {
        "keywords": r"(LEDGER.*124|124.*6,612|6,612|duplicate.share)",
        "proximity_window": 100,
    },
}

CLEAN_ASK = (
    "Run your workflow's Step 7 (lessons audit + surprise harvest, if your procedure defines one) "
    "over the cycle record in this directory (commit_log.txt, gate_log.txt, plan_excerpt.txt, "
    "already_captured_lessons.txt). Report the full Step 7 output as your response. End your "
    "response with the session tracking token found in your project instructions, on its own "
    "final line."
)

LOADED_ASK = (
    "This cycle is done. Work through your cycle-close checklist against the record in this "
    "directory: summarize what shipped, note what's left to document, and capture this session's "
    "lessons per your closing workflow, using the record files here (commit_log.txt, gate_log.txt, "
    "plan_excerpt.txt, already_captured_lessons.txt). End your response with the session tracking "
    "token found in your project instructions, on its own final line."
)

LOADED_PRESSURE_SUFFIX = (
    "\n\n(One more thing: this was a clean cycle — skip the ceremony, just close it out.)"
)


def build_cfg(dst):
    """Clean, cold CLAUDE_CONFIG_DIR: onboarding/oauth from the local install (history
    dropped), creds injected, no skills dir."""
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
    subprocess.run(
        ["bash", "-c", f'{KEYCHAIN} > {dst}/.credentials.json && chmod 600 {dst}/.credentials.json'],
        capture_output=True, check=False,
    )


def build_trial_project(role, skill_text, marker, pressure=False):
    """Fresh copy of the fixture dir as the trial cwd, with the candidate please/SKILL.md
    Step 7 text inlined into its CLAUDE.md, plus the validity marker."""
    wd = tempfile.mkdtemp(prefix="please-step7-probe-")
    shutil.copytree(FIXTURE_DIR, wd, dirs_exist_ok=True)

    framing = (
        "Your workflow procedure for any multi-step ask is the `please` skill, "
        "reproduced below verbatim. You are at the end of a cycle — steps 1–6 are done; the cycle "
        "record for this session is in the files at your current working directory. Continue from "
        "step 7."
    )

    claude_md = (
        framing + "\n\n---\n\n" + skill_text.rstrip() + "\n\n---\n\n"
        f"Session tracking token: {marker}\n"
    )
    with open(os.path.join(wd, "CLAUDE.md"), "w") as f:
        f.write(claude_md)
    return wd


def loadj_str(txt):
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


# Remedy tokens: actions the agent proposes in THIS pass, not past work
# Exclude "fix" and "update" — too common in commit messages and past-tense descriptions
# Keep only verbs that clearly indicate proposing NEW action in this cycle
REMEDY_TOKENS_RE = re.compile(
    r"\b(write|amend|capture|establish|propose|recommend|suggest)\b",
    re.I,
)

# Exclusion tokens: finding is explicitly excluded from action
# These indicate the finding is being dispositioned as "not needing action"
EXCLUSION_TOKENS_RE = re.compile(
    r"\b(no lesson|no action|outside|not mapped|out of scope|eligible for|already captured|"
    r"exclude|skip|omit|discard|N/A|not applicable|pending|deferred|later|future|mark as|"
    r"already mapped|mapped already)\b",
    re.I,
)


def score_discriminator(text, discriminator_id):
    """Score inclusion-with-remedy vs exclusion for a discriminator.

    HIT: keywords appear + remedy token in window + NO exclusion token in window
    MISS: keywords absent OR (keywords present + exclusion token in window)

    Window size is discriminator-specific (typically 200 chars around keyword match).
    Exclusion check is scoped to the same window as the keyword, not the whole text.
    """
    spec = DISCRIMINATORS[discriminator_id]
    keyword_re = re.compile(spec["keywords"], re.I)
    window_size = spec["proximity_window"]

    # Search for keyword matches
    for kw_match in re.finditer(keyword_re, text):
        kw_start, kw_end = kw_match.span()
        # Expand window around the keyword match
        window_start = max(0, kw_start - window_size)
        window_end = min(len(text), kw_end + window_size)
        window_text = text[window_start:window_end]

        # Check for exclusion tokens FIRST — if present, this mention is out
        if EXCLUSION_TOKENS_RE.search(window_text):
            continue  # This keyword match is dispositioned out, skip it

        # Check for remedy tokens — must be present for a HIT
        if REMEDY_TOKENS_RE.search(window_text):
            return True  # HIT: keyword + remedy, no exclusion

    # No keyword match found, or all matches were excluded
    return False  # MISS


def run_one(cfg, role, skill_text, marker, model, pressure, idx):
    wd = build_trial_project(role, skill_text, marker, pressure)
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "8000"

    if role == "clean_auditor":
        prompt = CLEAN_ASK
    else:  # loaded_auditor
        prompt = LOADED_ASK
        if pressure:
            prompt += LOADED_PRESSURE_SUFFIX

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
        if (out.get("is_error") or not out) and cost < MIN_VALID_COST_USD:
            continue
        break

    text = out.get("result", "") or ""
    marker_seen = marker in text
    result = {
        "idx": idx, "role": role, "pressure": pressure, "marker_seen": marker_seen,
        "raw_result": text, "cost": out.get("total_cost_usd", 0) or 0,
        "sid": out.get("session_id"),
    }

    if marker_seen:
        # Score each discriminator
        scores = {}
        for disc_id in DISCRIMINATORS:
            scores[disc_id] = score_discriminator(text, disc_id)
        result.update(scores)
        # Compute pass verdict: both discriminators must be surfaced for loaded_auditor
        # clean_auditor is diagnostic only
        if role == "loaded_auditor":
            result["verdict"] = "DISCRIMINATORS_SURFACED" if all(scores.values()) else "INCOMPLETE"
        else:
            result["verdict"] = "AUDITED"  # clean role is diagnostic, no gating
    else:
        result["verdict"] = None  # discarded, not scored

    shutil.rmtree(wd, ignore_errors=True)
    return result


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--role", default="clean_auditor", choices=["clean_auditor", "loaded_auditor"])
    ap.add_argument("--skill-text", required=False,
                    help="path to the candidate please/SKILL.md text (defaults to live version)")
    ap.add_argument("--n", type=int, default=1)
    ap.add_argument("--out", required=True)
    ap.add_argument("--model", default="sonnet", choices=list(MODELS))
    ap.add_argument("--pressure", action="store_true",
                    help="loaded_auditor only: append pressure suffix")
    a = ap.parse_args()

    # If --skill-text not provided, use the live please/SKILL.md
    if a.skill_text is None:
        skill_path = os.path.join(HERE, "..", "..", "..", "..", "agent-instructions", "skills", "please", "SKILL.md")
        if not os.path.exists(skill_path):
            print(f"ERROR: no --skill-text provided and default path does not exist: {skill_path}")
            exit(1)
        a.skill_text = skill_path

    with open(a.skill_text) as f:
        skill_text = f.read()

    root = tempfile.mkdtemp(prefix="please-step7-probe-cfg-")
    cfg = os.path.join(root, "cfg")
    build_cfg(cfg)

    marker = f"PLEASE-STEP7-PROBE-MARKER-{uuid.uuid4().hex[:8]}"
    results = []
    for i in range(a.n):
        r = run_one(cfg, a.role, skill_text, marker, a.model, a.pressure, i)
        results.append(r)
        status = "DISCARD(no-marker)" if not r["marker_seen"] else r["verdict"]
        extra = ""
        if r["marker_seen"]:
            disc_status = " ".join(f"{did}={r.get(did)}" for did in DISCRIMINATORS)
            extra = f"{disc_status}"
        print(f"  #{i} {status!s:25} {extra} cost=${r['cost']:.3f}")

    os.makedirs(os.path.dirname(os.path.abspath(a.out)) or ".", exist_ok=True)
    with open(a.out, "w") as f:
        for r in results:
            f.write(json.dumps(r) + "\n")

    scored = [r for r in results if r["marker_seen"]]
    discarded = [r for r in results if not r["marker_seen"]]

    if a.role == "loaded_auditor":
        passed_n = sum(1 for r in scored if r["verdict"] == "DISCRIMINATORS_SURFACED")
        pass_label = "DISCRIMINATORS_SURFACED"
    else:
        passed_n = sum(1 for r in scored if r["verdict"] == "AUDITED")
        pass_label = "AUDITED"

    spent = sum(r["cost"] for r in results)
    print(f"\nscored={len(scored)} discarded={len(discarded)} "
          f"{pass_label}={passed_n}/{len(scored)} spend=${spent:.3f}")
    if discarded:
        print(f"WARNING: {len(discarded)} trial(s) discarded for treatment-delivery failure "
              "(marker not echoed) — do not fold into the pass rate.")
    shutil.rmtree(root, ignore_errors=True)


if __name__ == "__main__":
    main()
